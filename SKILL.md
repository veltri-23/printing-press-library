---
name: pp-theclose
description: "Printing Press CLI for Theclose. Agent-facing API for The Close. Use scoped bearer tokens for transaction, field, document, task, event, and connector proposal workflows."
author: "cathrynlavery"
license: "Apache-2.0"
argument-hint: "<command> [args] | install cli|mcp"
allowed-tools: "Read Bash"
metadata:
  openclaw:
    requires:
      bins:
        - theclose-pp-cli
---

# Theclose — Printing Press CLI

## Prerequisites: Install the CLI

This skill drives the `theclose-pp-cli` binary. **You must verify the CLI is installed before invoking any command from this skill.** If it is missing, install it first:

1. Install via the Printing Press installer:
   ```bash
   npx -y @mvanhorn/printing-press install theclose --cli-only
   ```
2. Verify: `theclose-pp-cli --version`
3. Ensure `$GOPATH/bin` (or `$HOME/go/bin`) is on `$PATH`.

If the `npx` install fails before this CLI has a public-library category, install Node or use the category-specific Go fallback after publish.

If `--version` reports "command not found" after install, the install step did not put the binary on `$PATH`. Do not proceed with skill commands until verification succeeds.

Agent-facing API for The Close. Use scoped bearer tokens for transaction, field, document, task, event, and connector proposal workflows.

## Control Plane Rules

The Close is the control plane. Use `theclose-pp-cli` to discover work, inspect deal context, update The Close state, create connector-action proposals, dry-run proposals, wait for TC approval, execute only when The Close allows execution, and verify audit events.

Do not use provider CLIs or APIs to bypass The Close for transaction work. Follow Up Boss, Google Workspace, dotloop, MLS, Webflow, SkySlope, Zillow, and similar systems are downstream connectors. If the user asks for provider work tied to a deal, create a The Close connector proposal first and keep approval/audit inside The Close.

Provider-specific CLIs are acceptable only for exploration, read-only catalog research, or building a future connector adapter. They are not acceptable for live transaction writes unless the write is represented as a The Close connector action proposal and approved/executed through The Close.

## Core Agent Loop

Use these commands before raw endpoint wrappers:

```bash
theclose-pp-cli tasks ready-work --agent
theclose-pp-cli deals get <deal-id> --workspace --agent
theclose-pp-cli fields list <deal-id> --agent
theclose-pp-cli transactions documents list <deal-id> --agent
theclose-pp-cli transactions contacts list-transaction <deal-id> --agent
theclose-pp-cli transactions emails list <deal-id> --agent
theclose-pp-cli transactions events list-transaction <deal-id> --agent
```

For connector work:

```bash
theclose-pp-cli actions propose --deal-id <deal-id> --task-id <task-id> --connector-id follow_up_boss.actions --capability-id fub.contact.note.create --purpose "Add coordination note" --input-json '{"contactId":"...","body":"..."}' --agent
theclose-pp-cli actions dry-run <proposal-id> --agent
theclose-pp-cli actions status <proposal-id> --agent
theclose-pp-cli actions execute <proposal-id> --version <version> --agent
theclose-pp-cli actions audit <proposal-id> --agent
```

For Follow Up Boss shortcuts, prefer:

```bash
theclose-pp-cli fub propose-contact-upsert --deal-id <deal-id> --task-id <task-id> --first-name <first-name> --last-name <last-name> --email <contact-email> --agent
theclose-pp-cli fub propose-note-create --deal-id <deal-id> --task-id <task-id> --contact-id <fub-contact-id> --body "TC update..." --agent
theclose-pp-cli fub propose-deal-create --deal-id <deal-id> --pipeline-key seller --stage-key pending --deal-json '{"name":"<deal-name>"}' --agent
theclose-pp-cli fub propose-stage-update --deal-id <deal-id> --fub-deal-id <fub-deal-id> --pipeline-key seller --stage-key closed --agent
theclose-pp-cli fub propose-deal-delete --deal-id <deal-id> --fub-deal-id <fub-deal-id> --confirm-destructive --agent
theclose-pp-cli fub taxonomy-check --deal-id <deal-id> --pipeline-key seller --stage-key pending --agent
```

Approval and execution may return `401` or `403` for agent bearer tokens. Treat that as a successful boundary check: report that TC/session approval is required, do not call the downstream provider yourself, and continue by monitoring `actions status` and `actions audit`.

## Command Reference

**address** — Manage address

- `theclose-pp-cli address enrich` — Enrich a partial address with missing structured fields
- `theclose-pp-cli address suggest-addresses` — Suggest address matches for autocomplete

**admin** — Operator metrics and dead-letter queue tools.

- `theclose-pp-cli admin dismiss-dlqentry` — Removes a failed job from the DLQ without retrying.
- `theclose-pp-cli admin get-approval-rate` — TC approval rate (trust test)
- `theclose-pp-cli admin get-field-accuracy` — Most-corrected fields across all transactions. TC-owner only.
- `theclose-pp-cli admin get-portal-adoption` — Portal adoption — unique external views
- `theclose-pp-cli admin get-prompt-version-accuracy` — Accuracy stats per prompt version for before/after comparison.
- `theclose-pp-cli admin get-regression-set` — Top 10 most-corrected fields to test before prompt updates.
- `theclose-pp-cli admin get-weekly-summary` — VA replacement proxy — weekly action summary
- `theclose-pp-cli admin list-corrections` — Full correction log with original AI values and TC corrections.
- `theclose-pp-cli admin list-dlq` — Returns failed jobs across all queues. TC-owner only.
- `theclose-pp-cli admin retry-dlqentry` — Re-enqueues a failed job for processing.

**agent-actions** — Approval queue proposals and review endpoints.

- `theclose-pp-cli agent-actions create` — Create an agent action
- `theclose-pp-cli agent-actions invalidate-stale-actions` — Checks all pending actions for a transaction and marks matching ones as stale based on staleness rules.
- `theclose-pp-cli agent-actions list` — Returns agent actions, optionally filtered by deal or status.

**agent-tokens** — Manage agent tokens

- `theclose-pp-cli agent-tokens create` — Create a scoped API token
- `theclose-pp-cli agent-tokens list` — List API tokens for the signed-in TC

**billing** — Manage billing

- `theclose-pp-cli billing create-checkout-session` — Create a Stripe checkout session for plan changes
- `theclose-pp-cli billing create-portal-session` — Create a Stripe customer portal session
- `theclose-pp-cli billing get-summary` — Get current billing summary
- `theclose-pp-cli billing handle-stripe-webhook` — Stripe webhook for subscription lifecycle events

**callouts** — Manage callouts

- `theclose-pp-cli callouts check-seen` — Check whether a specific callout has been seen
- `theclose-pp-cli callouts list-seen` — List dismissed callout keys for the authenticated user
- `theclose-pp-cli callouts mark-seen` — Mark a callout key as seen (dismissed) for the authenticated user

**close-developer-health** — Manage close developer health

- `theclose-pp-cli close-developer-health` — Returns the process health and confirms the API runtime is responsive.

**close-developer-profile** — Manage close developer profile

- `theclose-pp-cli close-developer-profile get` — Get the signed-in TC profile
- `theclose-pp-cli close-developer-profile patch` — Update the signed-in TC profile

**connector-actions** — Connector-backed task proposal, dry-run, approval, and audited execution records.

- `theclose-pp-cli connector-actions approve-proposal` — Approve a connector action proposal
- `theclose-pp-cli connector-actions create-proposal` — Create a connector action proposal
- `theclose-pp-cli connector-actions dry-run` — Run a connector action dry-run
- `theclose-pp-cli connector-actions execute` — Execute an approved connector action
- `theclose-pp-cli connector-actions get-proposal` — Read connector action proposal status

**contacts** — Party and vendor contact records.

- `theclose-pp-cli contacts create` — Create a contact
- `theclose-pp-cli contacts delete` — Delete a contact
- `theclose-pp-cli contacts get` — Get contact by ID
- `theclose-pp-cli contacts list` — List contacts
- `theclose-pp-cli contacts update` — Update a contact

**documents** — Upload, version, and status management for documents.

- `theclose-pp-cli documents get` — Get document by ID
- `theclose-pp-cli documents soft-delete` — Marks document as deleted. S3 object is retained.
- `theclose-pp-cli documents update` — Update document metadata

**emails** — Manage emails

- `theclose-pp-cli emails <emailId>` — Update an email (edit draft, approve, reject)

**events** — Activity timeline feeds and streaming updates.

- `theclose-pp-cli events` — Returns activity events across all transactions, newest first. Used by dashboard and NL context injection.

**forms** — Intake form creation and submissions.

- `theclose-pp-cli forms create` — Create an intake form
- `theclose-pp-cli forms delete` — Delete a form
- `theclose-pp-cli forms get` — Get a form by ID
- `theclose-pp-cli forms update` — Update a form

**intake** — Contract upload intake, extraction polling, and cancellation.

- `theclose-pp-cli intake get-session` — Get intake extraction status
- `theclose-pp-cli intake undo-auto-created-transaction` — Undo an auto-created transaction from a forwarded contract
- `theclose-pp-cli intake upload-contract-for` — Accepts server-side multipart PDF upload, stores the file, creates an intake session, and enqueues extraction.

**notifications** — Notification feeds and read-state updates.

- `theclose-pp-cli notifications list` — Returns notifications ordered with unread first, then by recency. Supports filtering by type.
- `theclose-pp-cli notifications mark-all-read` — Mark all notifications as read for a TC

**portal** — Shared document portal invitations and access.

- `theclose-pp-cli portal` — Validates the magic link JWT and returns a short-lived session token for portal API access.

**schema** — Manage schema

- `theclose-pp-cli schema` — Returns every FieldDefinition. Agents call this to self-document available fields before acting on a transaction.

**tasks** — Task queue and transaction coordination actions.

- `theclose-pp-cli tasks delete` — Delete a task
- `theclose-pp-cli tasks list-all` — Cross-deal task list for pipeline UI. Filter by assignee type or unassigned.
- `theclose-pp-cli tasks update` — Update a task

**templates** — Template CRUD and application workflows.

- `theclose-pp-cli templates create` — Create a custom template
- `theclose-pp-cli templates delete` — Built-in templates cannot be deleted.
- `theclose-pp-cli templates get` — Get template by ID
- `theclose-pp-cli templates list` — Browse template library, optionally filtered by type.
- `theclose-pp-cli templates update` — Update a template

**transactions** — Transaction lifecycle and search endpoints.

- `theclose-pp-cli transactions confirm-csv-import` — Creates transactions in bulk from CSV rows. Import is additive only and each imported field is tagged with...
- `theclose-pp-cli transactions create` — Create a new deal
- `theclose-pp-cli transactions delete` — Delete a deal
- `theclose-pp-cli transactions get` — Get deal by ID
- `theclose-pp-cli transactions list` — Returns deals ordered by health. Supports search (?q=), status filter, health filter, and closing date range.
- `theclose-pp-cli transactions preview-csv-import` — Parses uploaded CSV content and returns auto-mapped columns against canonical field keys. Unmapped columns are...
- `theclose-pp-cli transactions update` — Status changes are validated against the deal state machine. Invalid transitions return 422.

**webhooks** — Webhook subscriptions and delivery history.

- `theclose-pp-cli webhooks create-subscription` — Create a webhook subscription
- `theclose-pp-cli webhooks delete-subscription` — Delete a webhook subscription
- `theclose-pp-cli webhooks handle-inbound-email` — Receives inbound emails, associates with transaction via reply address, stores as received email.
- `theclose-pp-cli webhooks list-deliveries` — List delivery attempts for a webhook subscription
- `theclose-pp-cli webhooks list-subscriptions` — List webhook subscriptions for the signed-in TC
- `theclose-pp-cli webhooks update-subscription` — Update a webhook subscription


### Finding the right command

When you know what you want to do but not which command does it, ask the CLI directly:

```bash
theclose-pp-cli which "<capability in your own words>"
```

`which` resolves a natural-language capability query to the best matching command from this CLI's curated feature index. Exit code `0` means at least one match; exit code `2` means no confident match — fall back to `--help` or use a narrower query.

## Auth Setup

Run `theclose-pp-cli auth setup` for the URL and steps to obtain a token (add `--launch` to open the URL). Then store it:

```bash
theclose-pp-cli auth set-token YOUR_TOKEN_HERE
```

Or set `THECLOSE_API_TOKEN` as an environment variable. Use `THECLOSE_BASE_URL` when targeting a non-default origin such as local development.

Run `theclose-pp-cli doctor` to verify setup.

## Agent Mode

Add `--agent` to any command. Expands to: `--json --compact --no-input --no-color --yes`.

- **Pipeable** — JSON on stdout, errors on stderr
- **Filterable** — `--select` keeps a subset of fields. Dotted paths descend into nested structures; arrays traverse element-wise. Critical for keeping context small on verbose APIs:

  ```bash
  theclose-pp-cli agent-actions list --agent --select id,name,status
  ```
- **Previewable** — `--dry-run` shows the request without sending
- **Offline-friendly** — sync/search commands can use the local SQLite store when available
- **Non-interactive** — never prompts, every input is a flag
- **Explicit retries** — use `--idempotent` only when an already-existing create should count as success, and `--ignore-missing` only when a missing delete target should count as success

### Response envelope

Commands that read from the local store or the API wrap output in a provenance envelope:

```json
{
  "meta": {"source": "live" | "local", "synced_at": "...", "reason": "..."},
  "results": <data>
}
```

Parse `.results` for data and `.meta.source` to know whether it's live or local. A human-readable `N results (live)` summary is printed to stderr only when stdout is a terminal AND no machine-format flag (`--json`, `--csv`, `--compact`, `--quiet`, `--plain`, `--select`) is set — piped/agent consumers and explicit-format runs get pure JSON on stdout.

## Agent Feedback

When you (or the agent) notice something off about this CLI, record it:

```
theclose-pp-cli feedback "the --since flag is inclusive but docs say exclusive"
theclose-pp-cli feedback --stdin < notes.txt
theclose-pp-cli feedback list --json --limit 10
```

Entries are stored locally at `~/.theclose-pp-cli/feedback.jsonl`. They are never POSTed unless `THECLOSE_FEEDBACK_ENDPOINT` is set AND either `--send` is passed or `THECLOSE_FEEDBACK_AUTO_SEND=true`. Default behavior is local-only.

Write what *surprised* you, not a bug report. Short, specific, one line: that is the part that compounds.

## Output Delivery

Every command accepts `--deliver <sink>`. The output goes to the named sink in addition to (or instead of) stdout, so agents can route command results without hand-piping. Three sinks are supported:

| Sink | Effect |
|------|--------|
| `stdout` | Default; write to stdout only |
| `file:<path>` | Atomically write output to `<path>` (tmp + rename) |
| `webhook:<url>` | POST the output body to the URL (`application/json` or `application/x-ndjson` when `--compact`) |

Unknown schemes are refused with a structured error naming the supported set. Webhook failures return non-zero and log the URL + HTTP status on stderr.

## Named Profiles

A profile is a saved set of flag values, reused across invocations. Use it when a scheduled agent calls the same command every run with the same configuration - HeyGen's "Beacon" pattern.

```
theclose-pp-cli profile save briefing --json
theclose-pp-cli --profile briefing agent-actions list
theclose-pp-cli profile list --json
theclose-pp-cli profile show briefing
theclose-pp-cli profile delete briefing --yes
```

Explicit flags always win over profile values; profile values win over defaults. `agent-context` lists all available profiles under `available_profiles` so introspecting agents discover them at runtime.

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 2 | Usage error (wrong arguments) |
| 3 | Resource not found |
| 4 | Authentication required |
| 5 | API error (upstream issue) |
| 7 | Rate limited (wait and retry) |
| 10 | Config error |

## Argument Parsing

Parse `$ARGUMENTS`:

1. **Empty, `help`, or `--help`** → show `theclose-pp-cli --help` output
2. **Starts with `install`** → ends with `mcp` → MCP installation; otherwise → see Prerequisites above
3. **Anything else** → Direct Use (execute as CLI command with `--agent`)

## MCP Server Installation

Install the MCP binary from this CLI's published public-library entry or pre-built release, then register it:

```bash
claude mcp add theclose-pp-mcp -- theclose-pp-mcp
```

Verify: `claude mcp list`

## Direct Use

1. Check if installed: `which theclose-pp-cli`
   If not found, offer to install (see Prerequisites at the top of this skill).
2. Match the user query to the best command from the Unique Capabilities and Command Reference above.
3. Execute with the `--agent` flag:
   ```bash
   theclose-pp-cli <command> [subcommand] [args] --agent
   ```
4. If ambiguous, drill into subcommand help: `theclose-pp-cli <command> --help`.
