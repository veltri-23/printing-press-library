// Copyright 2026 cathrynlavery. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"theclose-pp-cli/internal/store"
)

func newWorkQueueCmd(flags *rootFlags) *cobra.Command {
	var dbPath string
	cmd := &cobra.Command{
		Use:     "work-queue",
		Aliases: []string{"queue"},
		Short:   "Answer local-first cross-deal work queue questions",
		Long:    "Read synced The Close data from SQLite to answer compound work-queue questions. The API remains authoritative; run sync to refresh local data.",
		RunE:    parentNoSubcommandRunE(flags),
		Annotations: map[string]string{
			"mcp:read-only": "true",
		},
	}
	cmd.PersistentFlags().StringVar(&dbPath, "db", "", "Database path (default: ~/.local/share/theclose-pp-cli/data.db)")
	cmd.AddCommand(newWorkQueueListCmd(flags, &dbPath, "overdue", "List overdue open tasks", workQueueOverdue))
	cmd.AddCommand(newWorkQueueListCmd(flags, &dbPath, "blocked", "List blocked tasks", workQueueBlocked))
	cmd.AddCommand(newWorkQueueListCmd(flags, &dbPath, "needs-approval", "List pending approval actions", workQueueNeedsApproval))
	cmd.AddCommand(newWorkQueueClosingSoonCmd(flags, &dbPath))
	cmd.AddCommand(newWorkQueueListCmd(flags, &dbPath, "missing-fields", "List missing or empty deal fields", workQueueMissingFields))
	cmd.AddCommand(newWorkQueueListCmd(flags, &dbPath, "stale-actions", "List stale agent actions", workQueueStaleActions))
	return cmd
}

type workQueueFilter func(*store.Store, int) ([]json.RawMessage, error)

func newWorkQueueListCmd(flags *rootFlags, dbPath *string, name, short string, filter workQueueFilter) *cobra.Command {
	var limit int
	cmd := &cobra.Command{
		Use:   name,
		Short: short,
		Args:  cobra.NoArgs,
		Annotations: map[string]string{
			"mcp:read-only": "true",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := openWorkQueueStore(cmd, dbPath)
			if err != nil {
				return err
			}
			defer db.Close()
			results, err := filter(db, limit)
			if err != nil {
				return err
			}
			return outputWorkQueueResults(cmd, flags, db, name, results)
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 50, "Maximum results to return")
	return cmd
}

func newWorkQueueClosingSoonCmd(flags *rootFlags, dbPath *string) *cobra.Command {
	var limit, days int
	cmd := &cobra.Command{
		Use:   "closing-soon",
		Short: "List deals closing soon from local synced data",
		Args:  cobra.NoArgs,
		Annotations: map[string]string{
			"mcp:read-only": "true",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := openWorkQueueStore(cmd, dbPath)
			if err != nil {
				return err
			}
			defer db.Close()
			results, err := workQueueClosingSoon(db, limit, days)
			if err != nil {
				return err
			}
			return outputWorkQueueResults(cmd, flags, db, "closing-soon", results)
		},
	}
	cmd.Flags().IntVar(&days, "days", 14, "Closing window in days")
	cmd.Flags().IntVar(&limit, "limit", 50, "Maximum results to return")
	return cmd
}

func openWorkQueueStore(cmd *cobra.Command, dbPath *string) (*store.Store, error) {
	path := ""
	if dbPath != nil {
		path = *dbPath
	}
	if path == "" {
		path = defaultDBPath("theclose-pp-cli")
	}
	db, err := store.OpenWithContext(cmd.Context(), path)
	if err != nil {
		return nil, fmt.Errorf("opening local database: %w\nRun 'theclose-pp-cli sync' first to populate the local database.", err)
	}
	return db, nil
}

func outputWorkQueueResults(cmd *cobra.Command, flags *rootFlags, db *store.Store, queue string, results []json.RawMessage) error {
	data, _ := json.Marshal(results)
	prov := localProvenance(db, queue, "work_queue")
	if flags.asJSON || (!isTerminal(cmd.OutOrStdout()) && !flags.csv && !flags.quiet && !flags.plain) {
		wrapped, err := wrapWithProvenance(data, prov)
		if err != nil {
			return err
		}
		return printOutput(cmd.OutOrStdout(), wrapped, true)
	}
	if wantsHumanTable(cmd.OutOrStdout(), flags) {
		printProvenance(cmd, len(results), prov)
	}
	return printOutputWithFlags(cmd.OutOrStdout(), data, flags)
}

func workQueueOverdue(db *store.Store, limit int) ([]json.RawMessage, error) {
	return filterLocalResources(db, []string{"tasks", "transactions_tasks"}, limit, func(obj map[string]any) bool {
		status := strings.ToLower(stringAtAny(obj, "status"))
		if status == "completed" || status == "skipped" || status == "done" {
			return false
		}
		due, ok := dateAtAny(obj, "dueDate", "due_date", "deadline", "date")
		return ok && due.Before(startOfToday())
	})
}

func workQueueBlocked(db *store.Store, limit int) ([]json.RawMessage, error) {
	return filterLocalResources(db, []string{"tasks", "transactions_tasks"}, limit, func(obj map[string]any) bool {
		return strings.ToLower(stringAtAny(obj, "status")) == "blocked"
	})
}

func workQueueNeedsApproval(db *store.Store, limit int) ([]json.RawMessage, error) {
	return filterLocalResources(db, []string{"agent-actions", "activity_events", "transactions_events"}, limit, func(obj map[string]any) bool {
		status := strings.ToLower(stringAtAny(obj, "status"))
		if status == "pending_approval" || status == "proposed" || status == "pending" {
			return true
		}
		eventType := strings.ToLower(stringAtAny(obj, "type"))
		return eventType == "connector.proposal_created" || eventType == "connector.dry_run_completed"
	})
}

func workQueueClosingSoon(db *store.Store, limit int, days int) ([]json.RawMessage, error) {
	if days <= 0 {
		days = 14
	}
	cutoff := startOfToday().AddDate(0, 0, days+1)
	return filterLocalResources(db, []string{"transactions"}, limit, func(obj map[string]any) bool {
		closing, ok := dateAtAny(obj, "closingDate", "closing_date", "address.closingDate", "dateFields.closing_date")
		return ok && !closing.Before(startOfToday()) && closing.Before(cutoff)
	})
}

func workQueueMissingFields(db *store.Store, limit int) ([]json.RawMessage, error) {
	return filterLocalResources(db, []string{"transactions_fields"}, limit, func(obj map[string]any) bool {
		value, ok := valueAtAny(obj, "value", "tcValue", "tc_value", "aiValue", "ai_value")
		if !ok || value == nil {
			return true
		}
		if s, ok := value.(string); ok {
			return strings.TrimSpace(s) == ""
		}
		return false
	})
}

func workQueueStaleActions(db *store.Store, limit int) ([]json.RawMessage, error) {
	return filterLocalResources(db, []string{"agent-actions"}, limit, func(obj map[string]any) bool {
		return strings.ToLower(stringAtAny(obj, "status")) == "stale" || stringAtAny(obj, "staleReason", "stale_reason") != ""
	})
}

func filterLocalResources(db *store.Store, resourceTypes []string, limit int, keep func(map[string]any) bool) ([]json.RawMessage, error) {
	if limit <= 0 {
		limit = 50
	}
	out := make([]json.RawMessage, 0, limit)
	for _, resourceType := range resourceTypes {
		rows, err := db.List(resourceType, 1000)
		if err != nil {
			return nil, fmt.Errorf("list %s: %w", resourceType, err)
		}
		for _, raw := range rows {
			var obj map[string]any
			if json.Unmarshal(raw, &obj) != nil {
				continue
			}
			if keep(obj) {
				out = append(out, raw)
				if len(out) >= limit {
					return out, nil
				}
			}
		}
	}
	return out, nil
}

func stringAtAny(obj map[string]any, paths ...string) string {
	for _, path := range paths {
		if value, ok := valueAtAny(obj, path); ok {
			return fmt.Sprintf("%v", value)
		}
	}
	return ""
}

func dateAtAny(obj map[string]any, paths ...string) (time.Time, bool) {
	for _, path := range paths {
		value, ok := valueAtAny(obj, path)
		if !ok {
			continue
		}
		if parsed, ok := parseWorkQueueDate(fmt.Sprintf("%v", value)); ok {
			return parsed, true
		}
	}
	return time.Time{}, false
}

func valueAtAny(obj map[string]any, path string, more ...string) (any, bool) {
	paths := append([]string{path}, more...)
	for _, p := range paths {
		current := any(obj)
		for _, part := range strings.Split(p, ".") {
			m, ok := current.(map[string]any)
			if !ok {
				current = nil
				break
			}
			current, ok = m[part]
			if !ok {
				current = nil
				break
			}
		}
		if current != nil {
			return current, true
		}
	}
	return nil, false
}

func parseWorkQueueDate(value string) (time.Time, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, false
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02"} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed, true
		}
	}
	return time.Time{}, false
}

func startOfToday() time.Time {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
}
