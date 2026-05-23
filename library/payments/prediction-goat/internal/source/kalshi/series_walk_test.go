// Copyright 2026 mvanhorn. Licensed under Apache-2.0. See LICENSE.

package kalshi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/mvanhorn/printing-press-library/library/payments/prediction-goat/internal/cliutil"
)

// memoryStore is the in-memory SeriesWalkStore used by the series-walk
// tests. It captures upserts per resource type, seeds ListIDs for
// kalshi_series, and tracks sync_state keyed per series so resume can
// be exercised without spinning up a real SQLite database.
type memoryStore struct {
	mu       sync.Mutex
	rows     map[string]map[string]json.RawMessage // resource_type -> id -> raw
	seriesID []string                              // seed for ListIDs("kalshi_series")
	state    map[string]memoryState
}

type memoryState struct {
	cursor     string
	lastSynced time.Time
	count      int
}

func newMemoryStore(seriesIDs []string) *memoryStore {
	return &memoryStore{
		rows:     map[string]map[string]json.RawMessage{},
		seriesID: append([]string(nil), seriesIDs...),
		state:    map[string]memoryState{},
	}
}

func (m *memoryStore) Upsert(resourceType, id string, data json.RawMessage) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.rows[resourceType]; !ok {
		m.rows[resourceType] = map[string]json.RawMessage{}
	}
	m.rows[resourceType][id] = append(json.RawMessage(nil), data...)
	return nil
}

func (m *memoryStore) ListIDs(resourceType string) ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if resourceType == "kalshi_series" {
		return append([]string(nil), m.seriesID...), nil
	}
	rows, ok := m.rows[resourceType]
	if !ok {
		return nil, nil
	}
	ids := make([]string, 0, len(rows))
	for id := range rows {
		ids = append(ids, id)
	}
	return ids, nil
}

func (m *memoryStore) GetSyncState(resourceType string) (string, time.Time, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.state[resourceType]
	if !ok {
		return "", time.Time{}, 0, nil
	}
	return s.cursor, s.lastSynced, s.count, nil
}

func (m *memoryStore) SaveSyncState(resourceType, cursor string, count int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.state[resourceType] = memoryState{cursor: cursor, lastSynced: time.Now(), count: count}
	return nil
}

func (m *memoryStore) countByType(resourceType string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.rows[resourceType])
}

func (m *memoryStore) hasMarket(ticker string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.rows["kalshi_markets"][ticker]
	return ok
}

// fixtureServer maps each series ticker to its list of markets and serves
// /markets?series_ticker=<X>. It also supports a hook that kills a worker
// after N requests so the resume test can simulate an interruption.
type fixtureServer struct {
	t            *testing.T
	markets      map[string][]map[string]any
	callsBySeries map[string]int
	mu            sync.Mutex
}

func newFixtureServer(t *testing.T, markets map[string][]map[string]any) *httptest.Server {
	fs := &fixtureServer{
		t:             t,
		markets:       markets,
		callsBySeries: map[string]int{},
	}
	return httptest.NewServer(http.HandlerFunc(fs.serve))
}

func (fs *fixtureServer) serve(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, "/markets") {
		http.NotFound(w, r)
		return
	}
	series := r.URL.Query().Get("series_ticker")
	fs.mu.Lock()
	fs.callsBySeries[series]++
	fs.mu.Unlock()

	mkts, ok := fs.markets[series]
	if !ok {
		// Zero open markets for this series — return empty list, not an
		// error. Test expects the walk to log and continue.
		fs.writeJSON(w, map[string]any{"markets": []any{}})
		return
	}

	fs.writeJSON(w, map[string]any{"markets": mkts})
}

func (fs *fixtureServer) writeJSON(w http.ResponseWriter, body any) {
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	if err := enc.Encode(body); err != nil {
		fs.t.Fatalf("encode response: %v", err)
	}
}

// newTestClient wires the kalshi Client at a fixture server's URL with
// the AdaptiveLimiter disabled so the tests don't waste seconds waiting
// for rate-limit windows. Rate-limit verification re-enables it
// explicitly with a low rate.
func newTestClient(serverURL string) *Client {
	c := New()
	c.BaseURL = serverURL
	c.HTTPClient = &http.Client{Timeout: 5 * time.Second}
	c.limiter = nil
	return c
}

func TestSyncMarketsBySeries_CoversAllSeries(t *testing.T) {
	seriesIDs := []string{"KXOSCAR", "KXBTC", "KXMENWORLDCUP"}
	markets := map[string][]map[string]any{
		"KXOSCAR": {
			{"ticker": "KXOSCAR-26-BP", "series_ticker": "KXOSCAR"},
			{"ticker": "KXOSCAR-26-BD", "series_ticker": "KXOSCAR"},
		},
		"KXBTC": {
			{"ticker": "KXBTC-26-100K", "series_ticker": "KXBTC"},
			{"ticker": "KXBTC-26-50K", "series_ticker": "KXBTC"},
		},
		"KXMENWORLDCUP": {
			{"ticker": "KXMENWORLDCUP-26-PT", "series_ticker": "KXMENWORLDCUP"},
			{"ticker": "KXMENWORLDCUP-26-BR", "series_ticker": "KXMENWORLDCUP"},
		},
	}
	srv := newFixtureServer(t, markets)
	defer srv.Close()

	st := newMemoryStore(seriesIDs)
	c := newTestClient(srv.URL)

	n, err := SyncMarketsBySeries(context.Background(), c, st, 0)
	if err != nil {
		t.Fatalf("SyncMarketsBySeries: %v", err)
	}
	if want := 6; n != want {
		t.Fatalf("upserted=%d want=%d", n, want)
	}
	if got := st.countByType("kalshi_markets"); got != 6 {
		t.Fatalf("store has %d markets, want 6", got)
	}
}

func TestSyncMarketsBySeries_NamedEventFamiliesLand(t *testing.T) {
	seriesIDs := []string{"KXMENWORLDCUP", "KXBTC", "KXOSCAR"}
	markets := map[string][]map[string]any{
		"KXMENWORLDCUP": {
			{"ticker": "KXMENWORLDCUP-26-PT", "series_ticker": "KXMENWORLDCUP", "title": "Portugal wins"},
		},
		"KXBTC": {
			{"ticker": "KXBTC-MAX100", "series_ticker": "KXBTC"},
		},
		"KXOSCAR": {
			{"ticker": "KXOSCAR-BP-26", "series_ticker": "KXOSCAR"},
		},
	}
	srv := newFixtureServer(t, markets)
	defer srv.Close()

	st := newMemoryStore(seriesIDs)
	c := newTestClient(srv.URL)

	if _, err := SyncMarketsBySeries(context.Background(), c, st, 0); err != nil {
		t.Fatalf("walk: %v", err)
	}

	for _, ticker := range []string{"KXMENWORLDCUP-26-PT", "KXBTC-MAX100", "KXOSCAR-BP-26"} {
		if !st.hasMarket(ticker) {
			t.Errorf("named-event ticker %s missing from store", ticker)
		}
	}
}

func TestSyncMarketsBySeries_ResumeAfterPartialRun(t *testing.T) {
	seriesIDs := []string{"KXOSCAR", "KXBTC", "KXMENWORLDCUP"}
	markets := map[string][]map[string]any{
		"KXOSCAR":       {{"ticker": "KXOSCAR-1", "series_ticker": "KXOSCAR"}},
		"KXBTC":         {{"ticker": "KXBTC-1", "series_ticker": "KXBTC"}},
		"KXMENWORLDCUP": {{"ticker": "KXMENWORLDCUP-1", "series_ticker": "KXMENWORLDCUP"}},
	}
	srv := newFixtureServer(t, markets)
	defer srv.Close()
	c := newTestClient(srv.URL)

	st := newMemoryStore(seriesIDs)

	// Simulate "first run processed KXOSCAR" by manually flagging its
	// state as done. The next walk should skip it and only fetch
	// KXBTC + KXMENWORLDCUP.
	if err := st.SaveSyncState(seriesWalkStateKey("KXOSCAR"), seriesWalkDoneSentinel, 1); err != nil {
		t.Fatal(err)
	}
	// Pre-seed the upsert for the already-done series so the count
	// reflects "freshly walked this run" only.
	if err := st.Upsert("kalshi_markets", "KXOSCAR-1", json.RawMessage(`{"ticker":"KXOSCAR-1"}`)); err != nil {
		t.Fatal(err)
	}

	n, err := SyncMarketsBySeries(context.Background(), c, st, 0)
	if err != nil {
		t.Fatalf("resume walk: %v", err)
	}
	if want := 2; n != want {
		t.Fatalf("resumed walk upserted %d want %d", n, want)
	}
	if !st.hasMarket("KXBTC-1") || !st.hasMarket("KXMENWORLDCUP-1") {
		t.Fatalf("post-resume store missing expected markets: %+v", st.rows["kalshi_markets"])
	}

	// All three series should now be marked done.
	for _, s := range seriesIDs {
		cursor, _, _, _ := st.GetSyncState(seriesWalkStateKey(s))
		if cursor != seriesWalkDoneSentinel {
			t.Errorf("series %s not marked done after walk; cursor=%q", s, cursor)
		}
	}
}

func TestSyncMarketsBySeries_RateLimitHonored(t *testing.T) {
	// Pace 4 series through a limiter pinned at 4 req/s with the walk
	// forced to workers=1 (via IsVerifyEnv). The AdaptiveLimiter
	// serializes lastRequest only once a request has fired; with a
	// single worker each successive Wait() honors the 250ms gap, so
	// the wall-clock floor is ~750ms (3 inter-call gaps at 250ms).
	t.Setenv("PRINTING_PRESS_VERIFY", "1")

	seriesIDs := []string{"S1", "S2", "S3", "S4"}
	markets := map[string][]map[string]any{
		"S1": {{"ticker": "S1-A"}},
		"S2": {{"ticker": "S2-A"}},
		"S3": {{"ticker": "S3-A"}},
		"S4": {{"ticker": "S4-A"}},
	}
	srv := newFixtureServer(t, markets)
	defer srv.Close()

	c := New()
	c.BaseURL = srv.URL
	c.HTTPClient = &http.Client{Timeout: 5 * time.Second}
	c.limiter = cliutil.NewAdaptiveLimiter(4.0)

	st := newMemoryStore(seriesIDs)
	start := time.Now()
	if _, err := SyncMarketsBySeries(context.Background(), c, st, 0); err != nil {
		t.Fatalf("walk: %v", err)
	}
	elapsed := time.Since(start)
	if elapsed < 600*time.Millisecond {
		t.Fatalf("walk completed in %s; expected ≥ 600ms with 4 req/s limiter and serialized workers (rate limiting not honored)", elapsed)
	}
}

func TestSyncMarketsBySeries_EmptySeriesNoError(t *testing.T) {
	seriesIDs := []string{"KXEMPTY", "KXPOPULATED"}
	markets := map[string][]map[string]any{
		// "KXEMPTY" intentionally missing -> server returns empty list.
		"KXPOPULATED": {{"ticker": "KXPOP-1"}},
	}
	srv := newFixtureServer(t, markets)
	defer srv.Close()

	st := newMemoryStore(seriesIDs)
	c := newTestClient(srv.URL)

	n, err := SyncMarketsBySeries(context.Background(), c, st, 0)
	if err != nil {
		t.Fatalf("walk with empty series errored: %v", err)
	}
	if want := 1; n != want {
		t.Fatalf("upserted %d markets, want %d", n, want)
	}
	if !st.hasMarket("KXPOP-1") {
		t.Errorf("populated series's market missing from store")
	}
}

func TestSyncMarketsBySeries_NoSeriesIsNoOp(t *testing.T) {
	srv := newFixtureServer(t, nil)
	defer srv.Close()
	c := newTestClient(srv.URL)
	st := newMemoryStore(nil)
	n, err := SyncMarketsBySeries(context.Background(), c, st, 0)
	if err != nil {
		t.Fatalf("empty-corpus walk errored: %v", err)
	}
	if n != 0 {
		t.Fatalf("empty-corpus walk upserted %d markets, want 0", n)
	}
}

func TestSyncMarketsBySeries_CapMaxSeries(t *testing.T) {
	seriesIDs := []string{"S1", "S2", "S3", "S4", "S5"}
	markets := map[string][]map[string]any{
		"S1": {{"ticker": "S1-A"}},
		"S2": {{"ticker": "S2-A"}},
		"S3": {{"ticker": "S3-A"}},
		"S4": {{"ticker": "S4-A"}},
		"S5": {{"ticker": "S5-A"}},
	}
	srv := newFixtureServer(t, markets)
	defer srv.Close()

	st := newMemoryStore(seriesIDs)
	c := newTestClient(srv.URL)
	n, err := SyncMarketsBySeries(context.Background(), c, st, 2)
	if err != nil {
		t.Fatalf("capped walk: %v", err)
	}
	if n != 2 {
		t.Fatalf("maxSeries=2 walked %d markets, want 2", n)
	}
}

func TestSyncMarketsBySeries_MidPageCursor(t *testing.T) {
	// One series, two pages. Walk should follow the cursor and persist
	// both pages' markets.
	seriesIDs := []string{"KXMULTI"}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cursor := r.URL.Query().Get("cursor")
		if cursor == "" {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintln(w, `{"markets":[{"ticker":"KXMULTI-1"}],"cursor":"page2"}`)
			return
		}
		if cursor == "page2" {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintln(w, `{"markets":[{"ticker":"KXMULTI-2"}],"cursor":""}`)
			return
		}
		http.Error(w, "unknown cursor", http.StatusBadRequest)
	}))
	defer srv.Close()
	c := newTestClient(srv.URL)
	st := newMemoryStore(seriesIDs)
	n, err := SyncMarketsBySeries(context.Background(), c, st, 0)
	if err != nil {
		t.Fatalf("multi-page walk: %v", err)
	}
	if n != 2 {
		t.Fatalf("multi-page walk landed %d markets, want 2", n)
	}
	if !st.hasMarket("KXMULTI-1") || !st.hasMarket("KXMULTI-2") {
		t.Fatalf("missing markets after multi-page walk: %+v", st.rows["kalshi_markets"])
	}
	cursor, _, _, _ := st.GetSyncState(seriesWalkStateKey("KXMULTI"))
	if cursor != seriesWalkDoneSentinel {
		t.Errorf("series not marked done after full walk; cursor=%q", cursor)
	}
}
