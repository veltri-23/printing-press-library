# tc-wghg.11 Dogfood Transcript

Date: 2026-05-14

This transcript uses synthetic local/dev data only. The local token value is not recorded.

## Environment

- `THECLOSE_BASE_URL=http://localhost:3000`
- `THECLOSE_API_TOKEN=<redacted>`
- CLI: `<printing-press-library>/theclose/build/theclose-pp-cli`
- Local cache DB: `/tmp/theclose-tc-wghg-11.db`

## Setup

- Migrated the local development database through `0016_connector_action_approval_idempotency`.
- Seeded synthetic TC-owned data:
  - Deal: `11111111-1111-4111-8111-111111111111`
  - Open task: `22222222-2222-4222-8222-222222222222`
  - Contact, document, received email, and seed activity event for workspace inspection.

## CLI Loop

### Find Open Task

Command:

```sh
theclose-pp-cli tasks ready-work --agent --data-source live
```

Result:

- Returned the synthetic open task `22222222-2222-4222-8222-222222222222`.
- Task belonged to deal `11111111-1111-4111-8111-111111111111`.

### Claim/Start Task

Command:

```sh
theclose-pp-cli tasks start 22222222-2222-4222-8222-222222222222 --agent --data-source live
```

Result:

- Updated task status to `in_progress` through The Close API.

### Inspect Deal Workspace

Command:

```sh
theclose-pp-cli deals get 11111111-1111-4111-8111-111111111111 --workspace --agent --data-source live
```

Result:

- Returned deal context, contacts, documents, emails, events, fields, and tasks.
- All visible data was synthetic placeholder data.

### Field Write Approval Boundary

Command:

```sh
theclose-pp-cli fields set 11111111-1111-4111-8111-111111111111 closing_date --value '"2026-06-01"' --agent --data-source live
```

Result:

- Rejected with `BEARER_APPROVAL_REQUIRED`.
- Confirms bearer agents cannot silently write protected transaction fields.

### Create FUB Connector Proposal Through The Close

Command:

```sh
theclose-pp-cli fub propose-note-create \
  --deal-id 11111111-1111-4111-8111-111111111111 \
  --task-id 22222222-2222-4222-8222-222222222222 \
  --contact-id fub-contact-dogfood \
  --body 'Synthetic follow-up note from The Close CLI dogfood.' \
  --idempotency-key tc-wghg-11-fub-note-001 \
  --agent --data-source live
```

Result:

- Created proposal `0af1f4de-2fa8-4d25-b723-6d541d6ed1a2`.
- Connector: `follow_up_boss.actions`.
- Capability: `fub.contact.note.create`.
- Status: `proposed`.
- Audit event: `connector.proposal_created`.
- No direct Follow Up Boss write occurred.

### Dry Run

Command:

```sh
theclose-pp-cli actions dry-run 0af1f4de-2fa8-4d25-b723-6d541d6ed1a2 \
  --idempotency-key tc-wghg-11-dry-run-001 \
  --agent --data-source live
```

Result:

- Proposal advanced to `dry_run_completed`.
- Dry-run result reported `externalCalls: false` and `outcome: approval_required`.
- Audit event: `connector.dry_run_completed`.

### Inspect Status

Command:

```sh
theclose-pp-cli actions status 0af1f4de-2fa8-4d25-b723-6d541d6ed1a2 --agent --data-source live
```

Result:

- Status remained `dry_run_completed`.
- Version: `2`.
- Proposal included dry-run evidence and audit event IDs.

### Approval And Execution Boundary

Commands:

```sh
theclose-pp-cli actions approve 0af1f4de-2fa8-4d25-b723-6d541d6ed1a2 --version 2 \
  --idempotency-key tc-wghg-11-approve-boundary-001 --agent --data-source live

theclose-pp-cli actions execute 0af1f4de-2fa8-4d25-b723-6d541d6ed1a2 --version 2 \
  --idempotency-key tc-wghg-11-execute-boundary-001 --agent --data-source live
```

Result:

- Both commands returned `BEARER_SESSION_ONLY`.
- CLI hints told the agent to wait for TC/session approval rather than writing directly to the downstream provider.
- This is the expected agent boundary. TC-session happy-path approval/execution is covered by `tests/api/connector-actions.test.ts`.

### Audit Trail

Command:

```sh
theclose-pp-cli actions audit 0af1f4de-2fa8-4d25-b723-6d541d6ed1a2 \
  --deal-id 11111111-1111-4111-8111-111111111111 \
  --agent --data-source live
```

Result:

- Returned:
  - `connector.proposal_created`
  - `connector.dry_run_completed`

### Local Cache And Work Queue

Commands:

```sh
theclose-pp-cli sync --db /tmp/theclose-tc-wghg-11.db \
  --resources transactions,tasks --latest-only --agent

theclose-pp-cli search Dogfood --db /tmp/theclose-tc-wghg-11.db \
  --data-source local --agent --limit 5

theclose-pp-cli work-queue needs-approval --db /tmp/theclose-tc-wghg-11.db \
  --agent --limit 5
```

Result:

- Synced transactions, tasks, and transaction-dependent resources.
- Local search returned the deal, task, proposal-created event, and dry-run event.
- Local work queue surfaced the connector proposal/dry-run events as pending approval work.

## Accepted Local Gap

The local dogfood run did not execute the connector action because the generated CLI was operating as an external bearer agent. The Close intentionally keeps connector approval and execution behind TC/session-only routes. The app-level happy path is verified by `tests/api/connector-actions.test.ts`; the CLI loop verifies the agent-facing proposal, dry-run, approval boundary, and audit trail.
