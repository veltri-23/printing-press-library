// Copyright 2026 cathrynlavery. Licensed under Apache-2.0. See LICENSE.

package cli

import "testing"

func TestBuildFUBProposalBody(t *testing.T) {
	t.Parallel()
	body := buildFUBProposalBody(
		fubCommonOptions{
			dealID:         "deal-1",
			taskID:         "task-1",
			idempotencyKey: "caller-key",
		},
		fubContactNoteCreateID,
		"Create note",
		map[string]any{"body": "hello"},
	)

	if body["transactionId"] != "deal-1" {
		t.Fatalf("transactionId = %v", body["transactionId"])
	}
	if body["taskId"] != "task-1" {
		t.Fatalf("taskId = %v", body["taskId"])
	}
	if body["connectorId"] != fubConnectorID {
		t.Fatalf("connectorId = %v", body["connectorId"])
	}
	if body["capabilityId"] != fubContactNoteCreateID {
		t.Fatalf("capabilityId = %v", body["capabilityId"])
	}
	if body["purpose"] != "Create note" {
		t.Fatalf("purpose = %v", body["purpose"])
	}
	if body["idempotencyKey"] != "caller-key" {
		t.Fatalf("idempotencyKey = %v", body["idempotencyKey"])
	}
}

func TestParseFUBMapping(t *testing.T) {
	t.Parallel()
	mapping, status, err := parseFUBMapping(`{"accountId":"acct","pipelines":{}}`, "seller", "pending")
	if err != nil {
		t.Fatal(err)
	}
	if mapping["accountId"] != "acct" {
		t.Fatalf("mapping accountId = %v", mapping["accountId"])
	}
	if status["status"] != "provided" {
		t.Fatalf("status = %v", status["status"])
	}

	_, status, err = parseFUBMapping("", "", "pending")
	if err != nil {
		t.Fatal(err)
	}
	if status["status"] != "missing_mapping" {
		t.Fatalf("status = %v", status["status"])
	}
	missing, ok := status["missing"].([]string)
	if !ok || len(missing) != 2 {
		t.Fatalf("missing = %#v", status["missing"])
	}
}
