// Copyright 2026 cathrynlavery. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"encoding/json"
	"testing"

	"theclose-pp-cli/internal/store"
)

func TestWorkQueueBlockedAndMissingFields(t *testing.T) {
	t.Parallel()
	db, err := store.OpenWithContext(t.Context(), t.TempDir()+"/data.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if _, _, err := db.UpsertBatch("tasks", []json.RawMessage{
		json.RawMessage(`{"id":"task-blocked","status":"blocked","title":"Waiting on title"}`),
		json.RawMessage(`{"id":"task-open","status":"open","title":"Ready"}`),
	}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := db.UpsertBatch("transactions_fields", []json.RawMessage{
		json.RawMessage(`{"id":"field-missing","fieldKey":"closing_date","value":""}`),
		json.RawMessage(`{"id":"field-present","fieldKey":"sales_price","value":500000}`),
	}); err != nil {
		t.Fatal(err)
	}

	blocked, err := workQueueBlocked(db, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(blocked) != 1 || !jsonContainsID(blocked[0], "task-blocked") {
		t.Fatalf("blocked = %s", blocked)
	}

	missing, err := workQueueMissingFields(db, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(missing) != 1 || !jsonContainsID(missing[0], "field-missing") {
		t.Fatalf("missing = %s", missing)
	}
}

func TestWorkQueueClosingSoonAndNeedsApproval(t *testing.T) {
	t.Parallel()
	db, err := store.OpenWithContext(t.Context(), t.TempDir()+"/data.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	soon := startOfToday().AddDate(0, 0, 3).Format("2006-01-02")
	later := startOfToday().AddDate(0, 0, 30).Format("2006-01-02")
	if _, _, err := db.UpsertBatch("transactions", []json.RawMessage{
		json.RawMessage(`{"id":"deal-soon","address":{"closingDate":"` + soon + `"}}`),
		json.RawMessage(`{"id":"deal-later","address":{"closingDate":"` + later + `"}}`),
	}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := db.UpsertBatch("agent-actions", []json.RawMessage{
		json.RawMessage(`{"id":"action-pending","status":"pending_approval"}`),
		json.RawMessage(`{"id":"action-done","status":"executed"}`),
	}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := db.UpsertBatch("transactions_events", []json.RawMessage{
		json.RawMessage(`{"id":"event-proposal","transactions_id":"deal-soon","type":"connector.proposal_created","payload":{"proposalId":"proposal-1"}}`),
		json.RawMessage(`{"id":"event-other","transactions_id":"deal-soon","type":"task.completed"}`),
	}); err != nil {
		t.Fatal(err)
	}

	closing, err := workQueueClosingSoon(db, 10, 14)
	if err != nil {
		t.Fatal(err)
	}
	if len(closing) != 1 || !jsonContainsID(closing[0], "deal-soon") {
		t.Fatalf("closing = %s", closing)
	}

	approval, err := workQueueNeedsApproval(db, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(approval) != 2 || !jsonContainsID(approval[0], "action-pending") || !jsonContainsID(approval[1], "event-proposal") {
		t.Fatalf("approval = %s", approval)
	}
}

func jsonContainsID(raw json.RawMessage, id string) bool {
	var obj map[string]any
	if json.Unmarshal(raw, &obj) != nil {
		return false
	}
	return obj["id"] == id
}
