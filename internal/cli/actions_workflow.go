// Copyright 2026 cathrynlavery. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newActionsCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "actions",
		Short: "Connector action lifecycle commands through The Close approval boundary",
		Long:  "Create, dry-run, inspect, approve, execute, and audit connector action proposals through The Close. External providers remain downstream connectors.",
		RunE:  parentNoSubcommandRunE(flags),
	}
	cmd.AddCommand(newActionsProposeCmd(flags))
	cmd.AddCommand(newActionsDryRunCmd(flags))
	cmd.AddCommand(newActionsStatusCmd(flags))
	cmd.AddCommand(newActionsApproveCmd(flags))
	cmd.AddCommand(newActionsRejectCmd(flags))
	cmd.AddCommand(newActionsExecuteCmd(flags))
	cmd.AddCommand(newActionsRunsCmd(flags))
	cmd.AddCommand(newActionsAuditCmd(flags))
	return cmd
}

func newActionsProposeCmd(flags *rootFlags) *cobra.Command {
	var dealID, taskID, connectorID, capabilityID, purpose, inputJSON, idempotencyKey string
	cmd := &cobra.Command{
		Use:   "propose",
		Short: "Create a connector action proposal in The Close",
		RunE: func(cmd *cobra.Command, args []string) error {
			if dealID == "" && !flags.dryRun {
				return usageErr(fmt.Errorf("--deal-id is required"))
			}
			if connectorID == "" && !flags.dryRun {
				return usageErr(fmt.Errorf("--connector-id is required"))
			}
			if capabilityID == "" && !flags.dryRun {
				return usageErr(fmt.Errorf("--capability-id is required"))
			}
			if purpose == "" && !flags.dryRun {
				return usageErr(fmt.Errorf("--purpose is required"))
			}
			body, err := buildConnectorActionProposalBody(dealID, taskID, connectorID, capabilityID, purpose, inputJSON, idempotencyKey)
			if err != nil {
				return usageErr(err)
			}
			c, err := flags.newClient()
			if err != nil {
				return err
			}
			data, _, err := c.Post("/api/connector-actions/proposals", body)
			if err != nil {
				return classifyAPIError(err, flags)
			}
			return printOutputWithFlags(cmd.OutOrStdout(), extractData(data), flags)
		},
	}
	cmd.Flags().StringVar(&dealID, "deal-id", "", "The Close deal/transaction ID")
	cmd.Flags().StringVar(&taskID, "task-id", "", "Optional task ID the proposal is satisfying")
	cmd.Flags().StringVar(&connectorID, "connector-id", "", "The Close connector ID, such as follow_up_boss")
	cmd.Flags().StringVar(&capabilityID, "capability-id", "", "Connector capability ID")
	cmd.Flags().StringVar(&purpose, "purpose", "", "Human-readable purpose for TC review")
	cmd.Flags().StringVar(&inputJSON, "input-json", "{}", "Connector action input as a JSON object")
	cmd.Flags().StringVar(&idempotencyKey, "idempotency-key", "", "Idempotency key; generated when omitted")
	return cmd
}

func newActionsDryRunCmd(flags *rootFlags) *cobra.Command {
	var idempotencyKey string
	cmd := &cobra.Command{
		Use:   "dry-run <proposal-id>",
		Short: "Run a connector action proposal dry-run without provider writes",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body := actionDryRunBody(idempotencyKey)
			c, err := flags.newClient()
			if err != nil {
				return err
			}
			data, _, err := c.Post("/api/connector-actions/proposals/"+args[0]+"/dry-run", body)
			if err != nil {
				return classifyAPIError(err, flags)
			}
			return printOutputWithFlags(cmd.OutOrStdout(), extractData(data), flags)
		},
	}
	cmd.Flags().StringVar(&idempotencyKey, "idempotency-key", "", "Idempotency key; generated when omitted")
	return cmd
}

func newActionsStatusCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "status <proposal-id>",
		Aliases: []string{"get"},
		Short:   "Read connector action proposal status",
		Args:    cobra.ExactArgs(1),
		Annotations: map[string]string{
			"mcp:read-only": "true",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := flags.newClient()
			if err != nil {
				return err
			}
			data, err := c.Get("/api/connector-actions/proposals/"+args[0], nil)
			if err != nil {
				return classifyAPIError(err, flags)
			}
			return printOutputWithFlags(cmd.OutOrStdout(), extractData(data), flags)
		},
	}
	return cmd
}

func newActionsApproveCmd(flags *rootFlags) *cobra.Command {
	var version int
	var approvedByID, idempotencyKey string
	cmd := &cobra.Command{
		Use:     "approve <proposal-id> --version <version>",
		Short:   "Approve a connector action proposal through the TC/session-only API path",
		Example: "  theclose-pp-cli actions approve 550e8400-e29b-41d4-a716-446655440000 --version 2 --idempotency-key approve-0001",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if version == 0 && !flags.dryRun {
				return usageErr(fmt.Errorf("--version is required"))
			}
			body := actionVersionedBody("approve", version, idempotencyKey)
			if approvedByID != "" {
				body["approvedById"] = approvedByID
			}
			c, err := flags.newClient()
			if err != nil {
				return err
			}
			data, _, err := c.Post("/api/connector-actions/proposals/"+args[0]+"/approve", body)
			if err != nil {
				return classifyActionBoundaryError(err, flags, "approval")
			}
			return printOutputWithFlags(cmd.OutOrStdout(), extractData(data), flags)
		},
	}
	cmd.Flags().IntVar(&version, "version", 0, "Expected proposal version")
	cmd.Flags().StringVar(&approvedByID, "approved-by-id", "", "Approving TC/user ID when the session API accepts it")
	cmd.Flags().StringVar(&idempotencyKey, "idempotency-key", "", "Idempotency key; generated when omitted")
	return cmd
}

func newActionsRejectCmd(flags *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "reject <proposal-id>",
		Short: "Report that connector action rejection is not exposed by the API yet",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			err := fmt.Errorf("connector action rejection is not exposed by The Close API yet; use the The Close approval queue/UI and verify audit events")
			if flags != nil && flags.asJSON {
				_ = json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{
					"error":       "unsupported",
					"code":        "connector_action_reject_not_exposed",
					"proposal_id": args[0],
					"guidance":    "The Close CLI will not bypass The Close; use the approval queue/UI until the API exposes rejection.",
				})
			}
			return apiErr(err)
		},
	}
}

func newActionsExecuteCmd(flags *rootFlags) *cobra.Command {
	var version int
	var idempotencyKey string
	cmd := &cobra.Command{
		Use:   "execute <proposal-id> --version <version>",
		Short: "Execute an approved connector action through The Close",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if version == 0 && !flags.dryRun {
				return usageErr(fmt.Errorf("--version is required"))
			}
			body := actionVersionedBody("execute", version, idempotencyKey)
			c, err := flags.newClient()
			if err != nil {
				return err
			}
			data, _, err := c.Post("/api/connector-actions/proposals/"+args[0]+"/execute", body)
			if err != nil {
				return classifyActionBoundaryError(err, flags, "execution")
			}
			return printOutputWithFlags(cmd.OutOrStdout(), extractData(data), flags)
		},
	}
	cmd.Flags().IntVar(&version, "version", 0, "Expected proposal version")
	cmd.Flags().StringVar(&idempotencyKey, "idempotency-key", "", "Idempotency key; generated when omitted")
	return cmd
}

func newActionsRunsCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "runs <proposal-id>",
		Aliases: []string{"list-runs"},
		Short:   "List execution runs included in connector action proposal status",
		Args:    cobra.ExactArgs(1),
		Annotations: map[string]string{
			"mcp:read-only": "true",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := flags.newClient()
			if err != nil {
				return err
			}
			data, err := c.Get("/api/connector-actions/proposals/"+args[0], nil)
			if err != nil {
				return classifyAPIError(err, flags)
			}
			out := proposalRuns(extractData(data))
			return printOutputWithFlags(cmd.OutOrStdout(), out, flags)
		},
	}
	return cmd
}

func newActionsAuditCmd(flags *rootFlags) *cobra.Command {
	var dealID string
	var limit int
	cmd := &cobra.Command{
		Use:     "audit <proposal-id>",
		Aliases: []string{"events"},
		Short:   "Show The Close audit events related to a connector action proposal",
		Args:    cobra.ExactArgs(1),
		Annotations: map[string]string{
			"mcp:read-only": "true",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			proposalID := args[0]
			c, err := flags.newClient()
			if err != nil {
				return err
			}
			if dealID == "" {
				data, err := c.Get("/api/connector-actions/proposals/"+proposalID, nil)
				if err != nil {
					return classifyAPIError(err, flags)
				}
				dealID = proposalTransactionID(extractData(data))
				if dealID == "" {
					return apiErr(fmt.Errorf("proposal %s did not include transactionId; pass --deal-id to read audit events", proposalID))
				}
			}
			data, err := c.Get("/api/transactions/"+dealID+"/events", map[string]string{"limit": fmt.Sprintf("%d", limit)})
			if err != nil {
				return classifyAPIError(err, flags)
			}
			out := filterEventsByProposal(extractDataArray(data), proposalID)
			return printOutputWithFlags(cmd.OutOrStdout(), out, flags)
		},
	}
	cmd.Flags().StringVar(&dealID, "deal-id", "", "Deal/transaction ID; discovered from proposal status when omitted")
	cmd.Flags().IntVar(&limit, "limit", 100, "Maximum events to inspect")
	return cmd
}

func buildConnectorActionProposalBody(dealID, taskID, connectorID, capabilityID, purpose, inputJSON, idempotencyKey string) (map[string]any, error) {
	var input any = map[string]any{}
	if strings.TrimSpace(inputJSON) != "" {
		if err := json.Unmarshal([]byte(inputJSON), &input); err != nil {
			return nil, fmt.Errorf("parsing --input-json: %w", err)
		}
		if _, ok := input.(map[string]any); !ok {
			return nil, fmt.Errorf("--input-json must be a JSON object")
		}
	}
	body := map[string]any{
		"transactionId":  dealID,
		"connectorId":    connectorID,
		"capabilityId":   capabilityID,
		"purpose":        purpose,
		"idempotencyKey": actionIdempotencyKey("proposal", idempotencyKey),
		"actionInput":    input,
	}
	if taskID != "" {
		body["taskId"] = taskID
	}
	return body, nil
}

func actionIdempotencyKey(phase, provided string) string {
	if provided != "" {
		return provided
	}
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "theclose:" + phase + ":fallback"
	}
	return "theclose:" + phase + ":" + hex.EncodeToString(b[:])
}

func actionDryRunBody(idempotencyKey string) map[string]any {
	return map[string]any{"idempotencyKey": actionIdempotencyKey("dry-run", idempotencyKey)}
}

func actionVersionedBody(phase string, version int, idempotencyKey string) map[string]any {
	return map[string]any{
		"version":        version,
		"idempotencyKey": actionIdempotencyKey(phase, idempotencyKey),
	}
}

func classifyActionBoundaryError(err error, flags *rootFlags, operation string) error {
	msg := err.Error()
	if strings.Contains(msg, "HTTP 401") || strings.Contains(msg, "HTTP 403") {
		return authErr(fmt.Errorf("%w\nhint: connector action %s is protected by The Close's TC/session approval boundary. Agent bearer tokens may propose and dry-run, then wait for TC approval rather than writing directly to the downstream provider", err, operation))
	}
	return classifyAPIError(err, flags)
}

func proposalRuns(data json.RawMessage) json.RawMessage {
	var obj map[string]json.RawMessage
	if json.Unmarshal(data, &obj) == nil {
		for _, key := range []string{"runs", "executions", "executionRuns"} {
			if v := obj[key]; len(v) > 0 {
				return v
			}
		}
	}
	return json.RawMessage(`[]`)
}

func proposalTransactionID(data json.RawMessage) string {
	var obj map[string]any
	if json.Unmarshal(data, &obj) != nil {
		return ""
	}
	for _, key := range []string{"transactionId", "transaction_id", "dealId", "deal_id"} {
		if v, ok := obj[key].(string); ok && v != "" {
			return v
		}
	}
	return ""
}

func filterEventsByProposal(data json.RawMessage, proposalID string) json.RawMessage {
	var events []json.RawMessage
	if json.Unmarshal(data, &events) != nil {
		return data
	}
	filtered := make([]json.RawMessage, 0, len(events))
	for _, event := range events {
		if strings.Contains(string(event), proposalID) {
			filtered = append(filtered, event)
		}
	}
	out, _ := json.Marshal(filtered)
	return out
}
