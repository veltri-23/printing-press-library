// Copyright 2026 cathrynlavery. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"strings"
	"testing"
)

func TestActionIdempotencyKey(t *testing.T) {
	t.Parallel()
	got := actionIdempotencyKey("proposal", "")
	if !strings.HasPrefix(got, "theclose:proposal:") {
		t.Fatalf("generated key %q should include phase prefix", got)
	}
	if len(got) <= len("theclose:proposal:") {
		t.Fatalf("generated key %q should include entropy", got)
	}
	if got == actionIdempotencyKey("proposal", "") {
		t.Fatalf("generated keys should be unique")
	}
	if provided := actionIdempotencyKey("execute", "caller-key"); provided != "caller-key" {
		t.Fatalf("provided key = %q, want caller-key", provided)
	}
}

func TestBuildConnectorActionProposalBodyRequiresObjectInput(t *testing.T) {
	t.Parallel()
	if _, err := buildConnectorActionProposalBody("deal", "", "follow_up_boss", "fub.note.create", "test", `["not-object"]`, "key"); err == nil {
		t.Fatalf("array input should be rejected")
	}
}

func TestActionIdempotencyBodies(t *testing.T) {
	t.Parallel()
	proposal, err := buildConnectorActionProposalBody("deal", "task", "follow_up_boss.actions", "fub.contact.note.create", "purpose", `{}`, "proposal-key")
	if err != nil {
		t.Fatal(err)
	}
	if proposal["idempotencyKey"] != "proposal-key" {
		t.Fatalf("proposal idempotencyKey = %v", proposal["idempotencyKey"])
	}
	if proposal["taskId"] != "task" {
		t.Fatalf("proposal taskId = %v", proposal["taskId"])
	}
	if dryRun := actionDryRunBody("dry-key"); dryRun["idempotencyKey"] != "dry-key" {
		t.Fatalf("dry-run idempotencyKey = %v", dryRun["idempotencyKey"])
	}
	execute := actionVersionedBody("execute", 3, "execute-key")
	if execute["version"] != 3 {
		t.Fatalf("execute version = %v", execute["version"])
	}
	if execute["idempotencyKey"] != "execute-key" {
		t.Fatalf("execute idempotencyKey = %v", execute["idempotencyKey"])
	}
}

func TestProposalHelpers(t *testing.T) {
	t.Parallel()
	proposal := []byte(`{"transactionId":"tx-1","runs":[{"id":"run-1"}]}`)
	if got := proposalTransactionID(proposal); got != "tx-1" {
		t.Fatalf("proposalTransactionID = %q, want tx-1", got)
	}
	if got := string(proposalRuns(proposal)); got != `[{"id":"run-1"}]` {
		t.Fatalf("proposalRuns = %s", got)
	}
	events := []byte(`[{"id":"e1","metadata":{"proposalId":"prop-1"}},{"id":"e2"}]`)
	if got := string(filterEventsByProposal(events, "prop-1")); got != `[{"id":"e1","metadata":{"proposalId":"prop-1"}}]` {
		t.Fatalf("filterEventsByProposal = %s", got)
	}
}
