// Copyright 2026 cathrynlavery. Licensed under Apache-2.0. See LICENSE.

package store

import (
	"encoding/json"
	"testing"
)

func TestSearchResourceFiltersResourceType(t *testing.T) {
	t.Parallel()
	s, err := OpenWithContext(t.Context(), t.TempDir()+"/data.db")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	if _, _, err := s.UpsertBatch("transactions", []json.RawMessage{
		json.RawMessage(`{"id":"deal-1","address":{"street":"Shared Search Term"}}`),
	}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.UpsertBatch("tasks", []json.RawMessage{
		json.RawMessage(`{"id":"task-1","title":"Shared Search Term"}`),
	}); err != nil {
		t.Fatal(err)
	}

	got, err := s.SearchResource("tasks", "Shared", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("SearchResource returned %d rows, want 1: %s", len(got), got)
	}
	var obj map[string]any
	if err := json.Unmarshal(got[0], &obj); err != nil {
		t.Fatal(err)
	}
	if obj["id"] != "task-1" {
		t.Fatalf("SearchResource returned id %v, want task-1", obj["id"])
	}
}
