# Theclose CLI

Agent-facing API for The Close. Use scoped bearer tokens for transaction, field, document, task, and event workflows.

Learn more at [Theclose](/developers).

## Install

The recommended path installs both the `theclose-pp-cli` binary and the `pp-theclose` agent skill in one shot:

```bash
npx -y @mvanhorn/printing-press install theclose
```

For CLI only (no skill):

```bash
npx -y @mvanhorn/printing-press install theclose --cli-only
```


### Without Node

The initial distribution target is the Printing Press library. If `npx` is not available before publish, install from this generated tree:

```bash
cd <printing-press-library>/theclose
go install ./cmd/theclose-pp-cli
go install ./cmd/theclose-pp-mcp
```

### Pre-built binary

Download a pre-built binary for your platform from the [latest release](https://github.com/mvanhorn/printing-press-library/releases/tag/theclose-current). On macOS, clear the Gatekeeper quarantine: `xattr -d com.apple.quarantine <binary>`. On Unix, mark it executable: `chmod +x <binary>`.

<!-- pp-hermes-install-anchor -->
## Install for Hermes

From the Hermes CLI:

```bash
hermes skills install mvanhorn/printing-press-library/cli-skills/pp-theclose --force
```

Inside a Hermes chat session:

```bash
/skills install mvanhorn/printing-press-library/cli-skills/pp-theclose --force
```

## Install for OpenClaw

Tell your OpenClaw agent (copy this):

```
Install the pp-theclose skill from https://github.com/mvanhorn/printing-press-library/tree/main/cli-skills/pp-theclose. The skill defines how its required CLI can be installed.
```

## Quick Start

### 1. Install

See [Install](#install) above.

### Local Development From This Generated Tree

This baseline was generated outside the app repo at:

```bash
<printing-press-library>/theclose
```

Run The Close locally before using live commands:

```bash
cd <theclose-app-repo>
bun run dev
```

The source OpenAPI document is:

```bash
curl -fsS http://localhost:3000/openapi.json
```

Reprint from the current app spec with:

```bash
cd <cli-printing-press-repo>
go run ./cmd/printing-press generate \
  --refresh \
  --spec http://localhost:3000/openapi.json \
  --spec-source official \
  --spec-url http://localhost:3000/openapi.json \
  --name theclose \
  --owner cathrynlavery \
  --output <printing-press-library>/theclose \
  --force
```

### Release And Maintenance

This CLI is not vendored into The Close and does not have a separate product repo yet. The Close app repo owns the OpenAPI contract and CI check; the Printing Press library owns this generated source, MCP bundle, and agent skill.

Versioning follows the Developer API version:

- OpenAPI `info.version: 0.1.0`
- CLI release line: `theclose-0.1.x`
- MCP bundle line: `theclose-0.1.x`

Patch releases may include Printing Press polish, docs, skills, and non-breaking workflow helpers. Breaking OpenAPI, auth-policy, path, or response-shape changes require a new CLI release line tied to the updated OpenAPI version.

Before publishing or handing this CLI to production agents, run:

```bash
cd <printing-press-library>/theclose
go test ./...
go build ./...
go build -o ./build/theclose-pp-cli ./cmd/theclose-pp-cli
go build -o ./build/theclose-pp-mcp ./cmd/theclose-pp-mcp

cd <cli-printing-press-repo>
go run ./cmd/printing-press dogfood --dir <printing-press-library>/theclose --spec http://localhost:3000/openapi.json --json
go run ./cmd/printing-press scorecard --dir <printing-press-library>/theclose --spec http://localhost:3000/openapi.json --json
go run ./cmd/printing-press verify-skill --dir <printing-press-library>/theclose --json
go run ./cmd/printing-press pii-audit <printing-press-library>/theclose --strict --json
go run ./cmd/printing-press bundle <printing-press-library>/theclose
```

The Close app CI runs `bun run cli:check-openapi` so route/tag/auth changes that would break the generated CLI contract are caught before reprint.

### 2. Set Up Credentials

Get a scoped agent token from The Close Settings > API, then store it:

```bash
theclose-pp-cli auth set-token YOUR_TOKEN_HERE
```

Or set it via the generated baseline environment variable:

```bash
export THECLOSE_API_TOKEN="your-token-here"
```

Set the API origin explicitly when you are not targeting the local default:

```bash
export THECLOSE_BASE_URL="http://localhost:3000"
```

`CLOSE_DEVELOPER_BEARER_AUTH` remains as a generated legacy fallback, but `THECLOSE_API_TOKEN` is the canonical agent token variable.

### 3. Verify Setup

```bash
theclose-pp-cli doctor
```

This checks your configuration and credentials.

### 4. Try Your First Command

```bash
theclose-pp-cli agent-actions list
```

## Usage

Run `theclose-pp-cli --help` for the full command reference and flag list.

## Agent Workflow Commands

Use these workflow-shaped commands before falling back to raw endpoint wrappers:

```bash
theclose-pp-cli tasks ready-work --agent
theclose-pp-cli deals get <deal-id> --workspace --agent
theclose-pp-cli fields list <deal-id> --agent
theclose-pp-cli transactions documents list <deal-id> --agent
theclose-pp-cli transactions contacts list-transaction <deal-id> --agent
theclose-pp-cli transactions emails list <deal-id> --agent
theclose-pp-cli transactions events list-transaction <deal-id> --agent
```

Risky writes support `--dry-run` through the shared client:

```bash
theclose-pp-cli deals create --street "<street-address>" --dry-run --agent
theclose-pp-cli fields set <deal-id> closing_date --value '"2026-06-15"' --dry-run --agent
theclose-pp-cli tasks complete <task-id> --dry-run --agent
```

## Connector Action Lifecycle

Agents must route downstream provider work through The Close connector action proposals. Do not write directly to Follow Up Boss, Google Workspace, dotloop, MLS, Webflow, SkySlope, Zillow, or other providers from an agent workflow.

```bash
theclose-pp-cli actions propose \
  --deal-id <deal-id> \
  --task-id <task-id> \
  --connector-id follow_up_boss \
  --capability-id fub.contact.note.create \
  --purpose "Add coordination note from completed task" \
  --input-json '{"contactId":"...","body":"..."}' \
  --agent

theclose-pp-cli actions dry-run <proposal-id> --agent
theclose-pp-cli actions status <proposal-id> --agent
theclose-pp-cli actions approve <proposal-id> --version <version> --agent
theclose-pp-cli actions execute <proposal-id> --version <version> --agent
theclose-pp-cli actions audit <proposal-id> --agent
```

`actions propose`, `actions dry-run`, `actions approve`, and `actions execute` generate idempotency keys automatically unless `--idempotency-key` is provided. Stale version conflicts remain `409`, duplicate or invalid connector references remain `422`, and missing proposals remain `404` from the API.

The approval and execute endpoints are TC/session-gated by The Close. Agent bearer tokens can create proposals and run dry-runs, but if the API returns `401` or `403` for approval or execution the CLI surfaces that boundary instead of attempting any provider bypass. `actions reject` currently returns a structured unsupported error because the OpenAPI does not expose connector-action rejection yet. `actions runs` reads execution runs from proposal status when present; there is no standalone runs endpoint today.

## Follow Up Boss Helpers

The `fub` helpers are curated shortcuts over `actions propose`. They gather The Close deal/task context on live proposal creation, use the static connector catalog IDs, and submit proposals to The Close only:

```bash
theclose-pp-cli fub propose-contact-upsert --deal-id <deal-id> --task-id <task-id> --first-name <first-name> --last-name <last-name> --email <contact-email> --agent
theclose-pp-cli fub propose-note-create --deal-id <deal-id> --task-id <task-id> --contact-id <fub-contact-id> --body "TC update..." --agent
theclose-pp-cli fub propose-deal-create --deal-id <deal-id> --pipeline-key seller --stage-key pending --deal-json '{"name":"<deal-name>"}' --agent
theclose-pp-cli fub propose-stage-update --deal-id <deal-id> --fub-deal-id <fub-deal-id> --pipeline-key seller --stage-key closed --agent
theclose-pp-cli fub propose-deal-delete --deal-id <deal-id> --fub-deal-id <fub-deal-id> --confirm-destructive --agent
theclose-pp-cli fub taxonomy-check --deal-id <deal-id> --pipeline-key seller --stage-key pending --agent
```

FUB helper proposals always use `connectorId: follow_up_boss.actions` and one of `fub.contact.ingest`, `fub.contact.note.create`, `fub.deal.create`, `fub.deal.update_stage`, `fub.deal.delete`, or `fub.taxonomy.read`. Stage/deal helpers include a mapping status in `actionInput` so dry-runs expose missing `accountMapping`, `pipelineKey`, or `stageKey` before TC approval. Delete/archive proposals require `--confirm-destructive` outside dry-run and remain approval-sensitive.

## Local Cache And Work Queue

The Close API remains authoritative. Local SQLite is an agent read accelerator for repeated inspection and cross-resource questions.

```bash
theclose-pp-cli sync --latest-only --agent
theclose-pp-cli search "<query>" --data-source local --agent
theclose-pp-cli search "<query>" --type transactions_documents --data-source local --agent
theclose-pp-cli work-queue overdue --agent
theclose-pp-cli work-queue blocked --agent
theclose-pp-cli work-queue needs-approval --agent
theclose-pp-cli work-queue closing-soon --days 14 --agent
theclose-pp-cli work-queue missing-fields --agent
theclose-pp-cli work-queue stale-actions --agent
```

`sync` stores deals, top-level tasks, contacts, events, and transaction-scoped contacts, documents, emails, events, fields, and tasks where the token has access. `search` uses the synced full-text index; `--type` narrows to a synced resource such as `transactions`, `transactions_tasks`, `transactions_documents`, `transactions_contacts`, or `transactions_events`.

Use `--data-source live` when freshness matters more than speed, `--data-source local` when intentionally working from the cache, and `--data-source auto` for the CLI default. Local work-queue output includes provenance metadata so agents can decide when to run `sync` again.

## Mutation Safety Matrix

| Command family | Risk | Required guardrail |
| --- | --- | --- |
| `deals create`, `deals update-status`, `tasks claim/start/complete/skip/block/unblock`, `fields set/review/dispute/stale` | The Close state changes | Use `--dry-run` before live writes when an agent is unsure; API remains the source of truth. |
| `actions propose`, `actions dry-run`, `actions approve`, `actions execute` | Connector lifecycle | Idempotency keys are generated unless provided. Approval/execution keep API-side version checks and TC/session boundaries. |
| `fub propose-*` | Downstream CRM proposal | Creates The Close connector proposals only. No direct FUB writes. Destructive delete/archive requires `--confirm-destructive`. |
| Generated raw endpoint wrappers | Varies by endpoint | Prefer workflow commands first. Use `--dry-run`, `--idempotent`, and `--ignore-missing` only when the API semantics match the intended no-op. |

Dry-run output masks auth headers and redacts JSON fields with secret-shaped keys such as `apiKey`, `accessToken`, `systemKey`, `secret`, `password`, and `authorization`. Public examples use placeholders rather than real names, email addresses, provider credentials, or machine-local paths.

## Commands

### address

Manage address

- **`theclose-pp-cli address enrich`** - Enrich a partial address with missing structured fields
- **`theclose-pp-cli address suggest-addresses`** - Suggest address matches for autocomplete

### admin

Operator metrics and dead-letter queue tools.

- **`theclose-pp-cli admin dismiss-dlqentry`** - Removes a failed job from the DLQ without retrying.
- **`theclose-pp-cli admin get-approval-rate`** - TC approval rate (trust test)
- **`theclose-pp-cli admin get-field-accuracy`** - Most-corrected fields across all transactions. TC-owner only.
- **`theclose-pp-cli admin get-portal-adoption`** - Portal adoption — unique external views
- **`theclose-pp-cli admin get-prompt-version-accuracy`** - Accuracy stats per prompt version for before/after comparison.
- **`theclose-pp-cli admin get-regression-set`** - Top 10 most-corrected fields to test before prompt updates.
- **`theclose-pp-cli admin get-weekly-summary`** - VA replacement proxy — weekly action summary
- **`theclose-pp-cli admin list-corrections`** - Full correction log with original AI values and TC corrections.
- **`theclose-pp-cli admin list-dlq`** - Returns failed jobs across all queues. TC-owner only.
- **`theclose-pp-cli admin retry-dlqentry`** - Re-enqueues a failed job for processing.

### agent-actions

Approval queue proposals and review endpoints.

- **`theclose-pp-cli agent-actions create`** - Create an agent action
- **`theclose-pp-cli agent-actions invalidate-stale-actions`** - Checks all pending actions for a transaction and marks matching ones as stale based on staleness rules.
- **`theclose-pp-cli agent-actions list`** - Returns agent actions, optionally filtered by deal or status.

### agent-tokens

Manage agent tokens

- **`theclose-pp-cli agent-tokens create`** - Create a scoped API token
- **`theclose-pp-cli agent-tokens list`** - List API tokens for the signed-in TC

### billing

Manage billing

- **`theclose-pp-cli billing create-checkout-session`** - Create a Stripe checkout session for plan changes
- **`theclose-pp-cli billing create-portal-session`** - Create a Stripe customer portal session
- **`theclose-pp-cli billing get-summary`** - Get current billing summary
- **`theclose-pp-cli billing handle-stripe-webhook`** - Stripe webhook for subscription lifecycle events

### callouts

Manage callouts

- **`theclose-pp-cli callouts check-seen`** - Check whether a specific callout has been seen
- **`theclose-pp-cli callouts list-seen`** - List dismissed callout keys for the authenticated user
- **`theclose-pp-cli callouts mark-seen`** - Mark a callout key as seen (dismissed) for the authenticated user

### close-developer-health

Manage close developer health

- **`theclose-pp-cli close-developer-health get`** - Returns the process health and confirms the API runtime is responsive.

### close-developer-profile

Manage close developer profile

- **`theclose-pp-cli close-developer-profile get`** - Get the signed-in TC profile
- **`theclose-pp-cli close-developer-profile patch`** - Update the signed-in TC profile

### connector-actions

Connector-backed task proposal, dry-run, approval, and audited execution records.

- **`theclose-pp-cli connector-actions approve-proposal`** - Approve a connector action proposal
- **`theclose-pp-cli connector-actions create-proposal`** - Create a connector action proposal
- **`theclose-pp-cli connector-actions dry-run`** - Run a connector action dry-run
- **`theclose-pp-cli connector-actions execute`** - Execute an approved connector action
- **`theclose-pp-cli connector-actions get-proposal`** - Read connector action proposal status

### contacts

Party and vendor contact records.

- **`theclose-pp-cli contacts create`** - Create a contact
- **`theclose-pp-cli contacts delete`** - Delete a contact
- **`theclose-pp-cli contacts get`** - Get contact by ID
- **`theclose-pp-cli contacts list`** - List contacts
- **`theclose-pp-cli contacts update`** - Update a contact

### documents

Upload, version, and status management for documents.

- **`theclose-pp-cli documents get`** - Get document by ID
- **`theclose-pp-cli documents soft-delete`** - Marks document as deleted. S3 object is retained.
- **`theclose-pp-cli documents update`** - Update document metadata

### emails

Manage emails

- **`theclose-pp-cli emails update`** - Update an email (edit draft, approve, reject)

### events

Activity timeline feeds and streaming updates.

- **`theclose-pp-cli events list-global`** - Returns activity events across all transactions, newest first. Used by dashboard and NL context injection.

### forms

Intake form creation and submissions.

- **`theclose-pp-cli forms create`** - Create an intake form
- **`theclose-pp-cli forms delete`** - Delete a form
- **`theclose-pp-cli forms get`** - Get a form by ID
- **`theclose-pp-cli forms update`** - Update a form

### intake

Contract upload intake, extraction polling, and cancellation.

- **`theclose-pp-cli intake get-session`** - Get intake extraction status
- **`theclose-pp-cli intake undo-auto-created-transaction`** - Undo an auto-created transaction from a forwarded contract
- **`theclose-pp-cli intake upload-contract-for`** - Accepts server-side multipart PDF upload, stores the file, creates an intake session, and enqueues extraction.

### notifications

Notification feeds and read-state updates.

- **`theclose-pp-cli notifications list`** - Returns notifications ordered with unread first, then by recency. Supports filtering by type.
- **`theclose-pp-cli notifications mark-all-read`** - Mark all notifications as read for a TC

### portal

Shared document portal invitations and access.

- **`theclose-pp-cli portal exchange-token`** - Validates the magic link JWT and returns a short-lived session token for portal API access.

### schema

Manage schema

- **`theclose-pp-cli schema list-field-definitions`** - Returns every FieldDefinition. Agents call this to self-document available fields before acting on a transaction.

### tasks

Task queue and transaction coordination actions.

- **`theclose-pp-cli tasks delete`** - Delete a task
- **`theclose-pp-cli tasks list-all`** - Cross-deal task list for pipeline UI. Filter by assignee type or unassigned.
- **`theclose-pp-cli tasks update`** - Update a task

### templates

Template CRUD and application workflows.

- **`theclose-pp-cli templates create`** - Create a custom template
- **`theclose-pp-cli templates delete`** - Built-in templates cannot be deleted.
- **`theclose-pp-cli templates get`** - Get template by ID
- **`theclose-pp-cli templates list`** - Browse template library, optionally filtered by type.
- **`theclose-pp-cli templates update`** - Update a template

### transactions

Transaction lifecycle and search endpoints.

- **`theclose-pp-cli transactions confirm-csv-import`** - Creates transactions in bulk from CSV rows. Import is additive only and each imported field is tagged with source=csv_import provenance.
- **`theclose-pp-cli transactions create`** - Create a new deal
- **`theclose-pp-cli transactions delete`** - Delete a deal
- **`theclose-pp-cli transactions get`** - Get deal by ID
- **`theclose-pp-cli transactions list`** - Returns deals ordered by health. Supports search (?q=), status filter, health filter, and closing date range.
- **`theclose-pp-cli transactions preview-csv-import`** - Parses uploaded CSV content and returns auto-mapped columns against canonical field keys. Unmapped columns are flagged for manual review.
- **`theclose-pp-cli transactions update`** - Status changes are validated against the deal state machine. Invalid transitions return 422.

### webhooks

Webhook subscriptions and delivery history.

- **`theclose-pp-cli webhooks create-subscription`** - Create a webhook subscription
- **`theclose-pp-cli webhooks delete-subscription`** - Delete a webhook subscription
- **`theclose-pp-cli webhooks handle-inbound-email`** - Receives inbound emails, associates with transaction via reply address, stores as received email.
- **`theclose-pp-cli webhooks list-deliveries`** - List delivery attempts for a webhook subscription
- **`theclose-pp-cli webhooks list-subscriptions`** - List webhook subscriptions for the signed-in TC
- **`theclose-pp-cli webhooks update-subscription`** - Update a webhook subscription


## Output Formats

```bash
# Human-readable table (default in terminal, JSON when piped)
theclose-pp-cli agent-actions list

# JSON for scripting and agents
theclose-pp-cli agent-actions list --json

# Filter to specific fields
theclose-pp-cli agent-actions list --json --select id,name,status

# Dry run — show the request without sending
theclose-pp-cli agent-actions list --dry-run

# Agent mode — JSON + compact + no prompts in one flag
theclose-pp-cli agent-actions list --agent
```

## Agent Usage

This CLI is designed for AI agent consumption:

- **Non-interactive** - never prompts, every input is a flag
- **Pipeable** - `--json` output to stdout, errors to stderr
- **Filterable** - `--select id,name` returns only fields you need
- **Previewable** - `--dry-run` shows the request without sending
- **Explicit retries** - add `--idempotent` to create retries and `--ignore-missing` to delete retries when a no-op success is acceptable
- **Confirmable** - `--yes` for explicit confirmation of destructive actions
- **Piped input** - write commands can accept structured input when their help lists `--stdin`
- **Offline-friendly** - sync/search commands can use the local SQLite store when available
- **Agent-safe by default** - no colors or formatting unless `--human-friendly` is set

Exit codes: `0` success, `2` usage error, `3` not found, `4` auth error, `5` API error, `7` rate limited, `10` config error.

## Use with Claude Code

Install the focused skill — it auto-installs the CLI on first invocation:

```bash
npx skills add mvanhorn/printing-press-library/cli-skills/pp-theclose -g
```

Then invoke `/pp-theclose <query>` in Claude Code. The skill is the most efficient path — Claude Code drives the CLI directly without an MCP server in the middle.

<details>
<summary>Use as an MCP server in Claude Code (advanced)</summary>

If you'd rather register this CLI as an MCP server in Claude Code, install the MCP binary first:


Install the MCP binary from this CLI's published public-library entry or pre-built release.

Then register it:

```bash
claude mcp add theclose theclose-pp-mcp -e THECLOSE_API_TOKEN=<your-token>
```

</details>

## Use with Claude Desktop

This CLI ships an [MCPB](https://github.com/modelcontextprotocol/mcpb) bundle — Claude Desktop's standard format for one-click MCP extension installs (no JSON config required).

To install:

1. Download the `.mcpb` for your platform from the [latest release](https://github.com/mvanhorn/printing-press-library/releases/tag/theclose-current).
2. Double-click the `.mcpb` file. Claude Desktop opens and walks you through the install.
3. Fill in `THECLOSE_API_TOKEN` when Claude Desktop prompts you.

Requires Claude Desktop 1.0.0 or later. Pre-built bundles ship for macOS Apple Silicon (`darwin-arm64`) and Windows (`amd64`, `arm64`); for other platforms, use the manual config below.

<details>
<summary>Manual JSON config (advanced)</summary>

If you can't use the MCPB bundle (older Claude Desktop, unsupported platform), install the MCP binary and configure it manually.


Install the MCP binary from this CLI's published public-library entry or pre-built release.

Add to your Claude Desktop config (`~/Library/Application Support/Claude/claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "theclose": {
      "command": "theclose-pp-mcp",
      "env": {
        "THECLOSE_API_TOKEN": "<your-key>"
      }
    }
  }
}
```

</details>

## Health Check

```bash
theclose-pp-cli doctor
```

Verifies configuration, credentials, and connectivity to the API.

## Configuration

Config file: `~/.config/close-developer-pp-cli/config.toml`

Static request headers can be configured under `headers`; per-command header overrides take precedence.

Environment variables:

| Name | Kind | Required | Description |
| --- | --- | --- | --- |
| `THECLOSE_BASE_URL` | origin | No | The Close app origin. Defaults to `http://localhost:3000`. |
| `THECLOSE_API_TOKEN` | bearer | Yes | Scoped agent token from The Close Settings > API. |
| `CLOSE_DEVELOPER_BEARER_AUTH` | bearer | No | Legacy generated fallback; prefer `THECLOSE_API_TOKEN`. |

## Troubleshooting
**Authentication errors (exit code 4)**
- Run `theclose-pp-cli doctor` to check credentials
- Verify the environment variable is set without printing its value into logs: `test -n "$THECLOSE_API_TOKEN"`
**Not found errors (exit code 3)**
- Check the resource ID is correct
- Run the `list` command to see available items

---

Generated by [CLI Printing Press](https://github.com/mvanhorn/cli-printing-press)
