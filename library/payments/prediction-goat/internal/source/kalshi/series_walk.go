// Copyright 2026 mvanhorn. Licensed under Apache-2.0. See LICENSE.

package kalshi

import (
	"context"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mvanhorn/printing-press-library/library/payments/prediction-goat/internal/cliutil"
)

// SeriesWalkStore is the storage surface SyncMarketsBySeries needs. It is a
// superset of KalshiStore (Upsert) because the walk also needs (a) the list
// of synced series tickers as the iteration seed and (b) sync_state I/O
// keyed per series so a re-run resumes mid-walk.
//
// PATCH(series-driven-kalshi-markets-sync-walk): introduced so the walk can
// be exercised in tests against an in-memory store without dragging in the
// full *store.Store implementation.
type SeriesWalkStore interface {
	KalshiStore
	ListIDs(resourceType string) ([]string, error)
	GetSyncState(resourceType string) (cursor string, lastSynced time.Time, count int, err error)
	SaveSyncState(resourceType, cursor string, count int) error
}

// defaultSeriesWalkWorkers paces the per-series fan-out. Kalshi's /markets
// endpoint with series_ticker is cheap (one round trip per call) but the
// client's AdaptiveLimiter still gates each request, so multiple workers
// amortize wait time across in-flight calls without violating rate limits.
const defaultSeriesWalkWorkers = 8

// seriesWalkStateKey returns the sync_state row identifier for the per-series
// walk progress so each series has its own resume cursor.
func seriesWalkStateKey(ticker string) string {
	return "kalshi_markets_walk:" + ticker
}

// seriesWalkDoneSentinel marks a series as fully walked. The presence of a
// sync_state row with this cursor and a non-zero count signals "skip this
// series on the next resume" so a re-run picks up where the previous one
// left off without re-fetching already-covered prefixes.
const seriesWalkDoneSentinel = "__done__"

// SyncMarketsBySeries iterates every series ticker already present in the
// kalshi_series table and fetches /markets?series_ticker=<X> for each.
// This is the second-pass walk that covers named-event families
// (KXMENWORLDCUP-*, KXBTC*, KXOSCAR*) that the natural `/markets?status=open`
// pagination starves on high-volume factory-market prefixes.
//
// status defaults to "open" so the walk only fills the index with currently
// tradable markets. maxSeries=0 means "iterate every series" — pass a small
// positive value in tests / dogfood to cap the wall-clock.
//
// Progress is persisted per-series to sync_state under
// "kalshi_markets_walk:<ticker>" so an interrupted run resumes mid-walk
// rather than re-fetching from scratch.
//
// Returns the number of market rows newly upserted (across all series) plus
// the first non-context-canceled error if any worker hit one. Per-series
// errors are logged to stderr and counted but do not abort the walk.
func SyncMarketsBySeries(ctx context.Context, c *Client, st SeriesWalkStore, maxSeries int) (count int, err error) {
	if c == nil {
		c = New()
	}
	if st == nil {
		return 0, fmt.Errorf("nil store")
	}

	seriesTickers, err := st.ListIDs("kalshi_series")
	if err != nil {
		return 0, fmt.Errorf("list kalshi_series: %w", err)
	}
	if len(seriesTickers) == 0 {
		return 0, nil
	}

	limit := 1000
	if cliutil.IsDogfoodEnv() {
		limit = 50
	}

	workers := defaultSeriesWalkWorkers
	if cliutil.IsVerifyEnv() {
		// Single-flight under verify so the store writer doesn't trip
		// SQLITE_BUSY without natural network jitter.
		workers = 1
	}
	if workers > len(seriesTickers) {
		workers = len(seriesTickers)
	}

	work := make(chan string, len(seriesTickers))
	var (
		mu       sync.Mutex
		firstErr error
		total    int64
		enqueued int
	)

	enqueueLimit := len(seriesTickers)
	if maxSeries > 0 && maxSeries < enqueueLimit {
		enqueueLimit = maxSeries
	}

	for _, ticker := range seriesTickers {
		if enqueued >= enqueueLimit {
			break
		}
		// Resume gate: skip series flagged done in sync_state. Any
		// series with a non-sentinel cursor is mid-walk and re-enters
		// the queue so the worker can resume from its cursor.
		cursor, _, _, _ := st.GetSyncState(seriesWalkStateKey(ticker))
		if cursor == seriesWalkDoneSentinel {
			continue
		}
		work <- ticker
		enqueued++
	}
	close(work)

	if enqueued == 0 {
		return 0, nil
	}

	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for ticker := range work {
				if err := ctx.Err(); err != nil {
					mu.Lock()
					if firstErr == nil {
						firstErr = err
					}
					mu.Unlock()
					return
				}
				n, walkErr := walkOneSeries(ctx, c, st, ticker, limit)
				atomic.AddInt64(&total, int64(n))
				if walkErr != nil {
					fmt.Fprintf(os.Stderr, "kalshi: series %s walk error: %v\n", ticker, walkErr)
					mu.Lock()
					if firstErr == nil && ctx.Err() == nil {
						firstErr = walkErr
					}
					mu.Unlock()
				}
			}
		}()
	}
	wg.Wait()

	return int(atomic.LoadInt64(&total)), firstErr
}

// walkOneSeries pages /markets?series_ticker=<ticker> until the cursor
// terminates and upserts every market into the store. Saves the cursor to
// sync_state after each page so a re-run can resume mid-series. On
// successful completion, replaces the cursor with seriesWalkDoneSentinel so
// the next walk skips this series until --full clears it.
func walkOneSeries(ctx context.Context, c *Client, st SeriesWalkStore, ticker string, limit int) (int, error) {
	stateKey := seriesWalkStateKey(ticker)
	cursor, _, prior, _ := st.GetSyncState(stateKey)
	if cursor == seriesWalkDoneSentinel {
		return 0, nil
	}

	total := 0
	for {
		if err := ctx.Err(); err != nil {
			return total, err
		}

		body, err := c.GetMarketsBySeries(ctx, ticker, "open", cursor, limit)
		if err != nil {
			return total, fmt.Errorf("get markets for series %s: %w", ticker, err)
		}

		items, nextCursor, err := decodeMarkets(body)
		if err != nil {
			return total, err
		}

		for _, raw := range items {
			id, err := extractTicker(raw, "ticker")
			if err != nil {
				return total, err
			}
			if id == "" {
				fmt.Fprintf(os.Stderr, "kalshi: skipping kalshi_markets item with empty ticker (series %s)\n", ticker)
				continue
			}
			if err := st.Upsert("kalshi_markets", id, raw); err != nil {
				return total, fmt.Errorf("upsert market %s (series %s): %w", id, ticker, err)
			}
			total++
		}

		if nextCursor == "" {
			// Mark this series done so the next resume skips it.
			if saveErr := st.SaveSyncState(stateKey, seriesWalkDoneSentinel, prior+total); saveErr != nil {
				fmt.Fprintf(os.Stderr, "kalshi: save sync_state done for series %s: %v\n", ticker, saveErr)
			}
			return total, nil
		}

		// Persist progress so an interruption resumes from the next cursor.
		if saveErr := st.SaveSyncState(stateKey, nextCursor, prior+total); saveErr != nil {
			fmt.Fprintf(os.Stderr, "kalshi: save sync_state cursor for series %s: %v\n", ticker, saveErr)
		}
		cursor = nextCursor
	}
}

