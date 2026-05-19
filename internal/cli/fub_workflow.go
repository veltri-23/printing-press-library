// Copyright 2026 cathrynlavery. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"theclose-pp-cli/internal/client"
)

const (
	fubConnectorID           = "follow_up_boss.actions"
	fubContactIngestID       = "fub.contact.ingest"
	fubContactNoteCreateID   = "fub.contact.note.create"
	fubDealCreateID          = "fub.deal.create"
	fubDealUpdateStageID     = "fub.deal.update_stage"
	fubDealDeleteID          = "fub.deal.delete"
	fubTaxonomyReadID        = "fub.taxonomy.read"
	defaultFUBDestructiveMsg = "Delete or archive the mapped Follow Up Boss deal through The Close approval flow"
)

func newFUBCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fub",
		Short: "Follow Up Boss proposal helpers that route through The Close",
		Long:  "Create Follow Up Boss connector action proposals through The Close. These commands never call Follow Up Boss directly.",
		RunE:  parentNoSubcommandRunE(flags),
	}
	cmd.AddCommand(newFUBContactUpsertCmd(flags))
	cmd.AddCommand(newFUBNoteCreateCmd(flags))
	cmd.AddCommand(newFUBDealCreateCmd(flags))
	cmd.AddCommand(newFUBStageUpdateCmd(flags))
	cmd.AddCommand(newFUBDealDeleteCmd(flags))
	cmd.AddCommand(newFUBTaxonomyCmd(flags))
	return cmd
}

func newFUBContactUpsertCmd(flags *rootFlags) *cobra.Command {
	var opts fubCommonOptions
	var contactJSON, firstName, lastName, email, phone, role string
	cmd := &cobra.Command{
		Use:   "propose-contact-upsert",
		Short: "Propose creating or updating a Follow Up Boss contact",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireFUBDeal(opts.dealID, flags); err != nil {
				return err
			}
			contact, err := parseJSONObjectFlag("contact-json", contactJSON)
			if err != nil {
				return usageErr(err)
			}
			for k, v := range map[string]string{"firstName": firstName, "lastName": lastName, "email": email, "phone": phone, "role": role} {
				if v != "" {
					contact[k] = v
				}
			}
			input := map[string]any{
				"contact": contact,
				"summary": map[string]any{
					"operation": "contact.upsert",
					"role":      role,
					"email":     email,
				},
			}
			return runFUBProposal(cmd, flags, opts, fubContactIngestID, defaultPurpose(opts.purpose, "Create or update Follow Up Boss contact"), input)
		},
	}
	addFUBCommonFlags(cmd, &opts)
	cmd.Flags().StringVar(&contactJSON, "contact-json", "{}", "Contact fields as a JSON object")
	cmd.Flags().StringVar(&firstName, "first-name", "", "Contact first name")
	cmd.Flags().StringVar(&lastName, "last-name", "", "Contact last name")
	cmd.Flags().StringVar(&email, "email", "", "Contact email")
	cmd.Flags().StringVar(&phone, "phone", "", "Contact phone")
	cmd.Flags().StringVar(&role, "role", "", "The Close participant role")
	return cmd
}

func newFUBNoteCreateCmd(flags *rootFlags) *cobra.Command {
	var opts fubCommonOptions
	var contactID, body string
	cmd := &cobra.Command{
		Use:   "propose-note-create",
		Short: "Propose creating a note on a Follow Up Boss contact",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireFUBDeal(opts.dealID, flags); err != nil {
				return err
			}
			if contactID == "" && !flags.dryRun {
				return usageErr(fmt.Errorf("--contact-id is required"))
			}
			if body == "" && !flags.dryRun {
				return usageErr(fmt.Errorf("--body is required"))
			}
			input := map[string]any{
				"contactId": contactID,
				"body":      body,
				"summary": map[string]any{
					"operation": "contact.note.create",
					"contactId": contactID,
				},
			}
			return runFUBProposal(cmd, flags, opts, fubContactNoteCreateID, defaultPurpose(opts.purpose, "Create Follow Up Boss contact note"), input)
		},
	}
	addFUBCommonFlags(cmd, &opts)
	cmd.Flags().StringVar(&contactID, "contact-id", "", "Follow Up Boss contact/person ID")
	cmd.Flags().StringVar(&body, "body", "", "Note body")
	return cmd
}

func newFUBDealCreateCmd(flags *rootFlags) *cobra.Command {
	var opts fubCommonOptions
	var dealJSON, pipelineKey, stageKey, accountMappingJSON string
	cmd := &cobra.Command{
		Use:   "propose-deal-create",
		Short: "Propose creating a Follow Up Boss deal",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireFUBDeal(opts.dealID, flags); err != nil {
				return err
			}
			deal, err := parseJSONObjectFlag("deal-json", dealJSON)
			if err != nil {
				return usageErr(err)
			}
			accountMapping, mappingStatus, err := parseFUBMapping(accountMappingJSON, pipelineKey, stageKey)
			if err != nil {
				return usageErr(err)
			}
			input := map[string]any{
				"deal":          deal,
				"pipelineKey":   pipelineKey,
				"stageKey":      stageKey,
				"mappingStatus": mappingStatus,
				"summary": map[string]any{
					"operation":   "deal.create",
					"pipelineKey": pipelineKey,
					"stageKey":    stageKey,
					"mapping":     mappingStatus["status"],
				},
			}
			if accountMapping != nil {
				input["accountMapping"] = accountMapping
			}
			return runFUBProposal(cmd, flags, opts, fubDealCreateID, defaultPurpose(opts.purpose, "Create Follow Up Boss deal"), input)
		},
	}
	addFUBCommonFlags(cmd, &opts)
	cmd.Flags().StringVar(&dealJSON, "deal-json", "{}", "FUB deal payload summary as a JSON object")
	cmd.Flags().StringVar(&pipelineKey, "pipeline-key", "", "The Close mapping key for the FUB pipeline")
	cmd.Flags().StringVar(&stageKey, "stage-key", "", "The Close mapping key for the FUB stage")
	cmd.Flags().StringVar(&accountMappingJSON, "account-mapping-json", "", "Optional account mapping JSON object")
	return cmd
}

func newFUBStageUpdateCmd(flags *rootFlags) *cobra.Command {
	var opts fubCommonOptions
	var fubDealID, pipelineKey, stageKey, accountMappingJSON string
	cmd := &cobra.Command{
		Use:   "propose-stage-update",
		Short: "Propose moving a Follow Up Boss deal to another stage",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireFUBDeal(opts.dealID, flags); err != nil {
				return err
			}
			if fubDealID == "" && !flags.dryRun {
				return usageErr(fmt.Errorf("--fub-deal-id is required"))
			}
			accountMapping, mappingStatus, err := parseFUBMapping(accountMappingJSON, pipelineKey, stageKey)
			if err != nil {
				return usageErr(err)
			}
			input := map[string]any{
				"fubDealId":     fubDealID,
				"pipelineKey":   pipelineKey,
				"stageKey":      stageKey,
				"mappingStatus": mappingStatus,
				"summary": map[string]any{
					"operation":   "deal.update_stage",
					"fubDealId":   fubDealID,
					"pipelineKey": pipelineKey,
					"stageKey":    stageKey,
					"mapping":     mappingStatus["status"],
				},
			}
			if accountMapping != nil {
				input["accountMapping"] = accountMapping
			}
			return runFUBProposal(cmd, flags, opts, fubDealUpdateStageID, defaultPurpose(opts.purpose, "Update Follow Up Boss deal stage"), input)
		},
	}
	addFUBCommonFlags(cmd, &opts)
	cmd.Flags().StringVar(&fubDealID, "fub-deal-id", "", "Follow Up Boss deal ID")
	cmd.Flags().StringVar(&pipelineKey, "pipeline-key", "", "The Close mapping key for the FUB pipeline")
	cmd.Flags().StringVar(&stageKey, "stage-key", "", "The Close mapping key for the FUB stage")
	cmd.Flags().StringVar(&accountMappingJSON, "account-mapping-json", "", "Optional account mapping JSON object")
	return cmd
}

func newFUBDealDeleteCmd(flags *rootFlags) *cobra.Command {
	var opts fubCommonOptions
	var fubDealID, reason string
	var confirmDestructive bool
	cmd := &cobra.Command{
		Use:     "propose-deal-delete",
		Aliases: []string{"propose-deal-archive"},
		Short:   "Propose deleting or archiving a Follow Up Boss deal",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireFUBDeal(opts.dealID, flags); err != nil {
				return err
			}
			if fubDealID == "" && !flags.dryRun {
				return usageErr(fmt.Errorf("--fub-deal-id is required"))
			}
			if !confirmDestructive && !flags.dryRun {
				return usageErr(fmt.Errorf("--confirm-destructive is required for FUB delete/archive proposals"))
			}
			input := map[string]any{
				"fubDealId": fubDealID,
				"reason":    reason,
				"destructiveAction": map[string]any{
					"confirmed": confirmDestructive,
					"warning":   "Follow Up Boss deal delete/archive is always approval-sensitive.",
				},
				"summary": map[string]any{
					"operation":   "deal.delete",
					"fubDealId":   fubDealID,
					"destructive": true,
				},
			}
			return runFUBProposal(cmd, flags, opts, fubDealDeleteID, defaultPurpose(opts.purpose, defaultFUBDestructiveMsg), input)
		},
	}
	addFUBCommonFlags(cmd, &opts)
	cmd.Flags().StringVar(&fubDealID, "fub-deal-id", "", "Follow Up Boss deal ID")
	cmd.Flags().StringVar(&reason, "reason", "", "Reason for destructive FUB proposal")
	cmd.Flags().BoolVar(&confirmDestructive, "confirm-destructive", false, "Required acknowledgement for delete/archive proposals")
	return cmd
}

func newFUBTaxonomyCmd(flags *rootFlags) *cobra.Command {
	var opts fubCommonOptions
	var pipelineKey, stageKey, accountMappingJSON string
	cmd := &cobra.Command{
		Use:   "taxonomy-check",
		Short: "Propose a Follow Up Boss taxonomy read or mapping check",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireFUBDeal(opts.dealID, flags); err != nil {
				return err
			}
			accountMapping, mappingStatus, err := parseFUBMapping(accountMappingJSON, pipelineKey, stageKey)
			if err != nil {
				return usageErr(err)
			}
			input := map[string]any{
				"pipelineKey":   pipelineKey,
				"stageKey":      stageKey,
				"mappingStatus": mappingStatus,
				"summary": map[string]any{
					"operation": "taxonomy.read",
					"mapping":   mappingStatus["status"],
				},
			}
			if accountMapping != nil {
				input["accountMapping"] = accountMapping
			}
			return runFUBProposal(cmd, flags, opts, fubTaxonomyReadID, defaultPurpose(opts.purpose, "Read Follow Up Boss taxonomy for mapping validation"), input)
		},
	}
	addFUBCommonFlags(cmd, &opts)
	cmd.Flags().StringVar(&pipelineKey, "pipeline-key", "", "Optional pipeline mapping key to check")
	cmd.Flags().StringVar(&stageKey, "stage-key", "", "Optional stage mapping key to check")
	cmd.Flags().StringVar(&accountMappingJSON, "account-mapping-json", "", "Optional account mapping JSON object")
	return cmd
}

type fubCommonOptions struct {
	dealID         string
	taskID         string
	purpose        string
	idempotencyKey string
}

func addFUBCommonFlags(cmd *cobra.Command, opts *fubCommonOptions) {
	cmd.Flags().StringVar(&opts.dealID, "deal-id", "", "The Close deal/transaction ID")
	cmd.Flags().StringVar(&opts.taskID, "task-id", "", "Optional The Close task ID")
	cmd.Flags().StringVar(&opts.purpose, "purpose", "", "Override TC-facing proposal purpose")
	cmd.Flags().StringVar(&opts.idempotencyKey, "idempotency-key", "", "Idempotency key; generated when omitted")
}

func requireFUBDeal(dealID string, flags *rootFlags) error {
	if dealID == "" && !flags.dryRun {
		return usageErr(fmt.Errorf("--deal-id is required"))
	}
	return nil
}

func runFUBProposal(cmd *cobra.Command, flags *rootFlags, opts fubCommonOptions, capabilityID, purpose string, input map[string]any) error {
	c, err := flags.newClient()
	if err != nil {
		return err
	}
	if !flags.dryRun {
		enrichFUBInputWithContext(c, opts, input)
	}
	body := buildFUBProposalBody(opts, capabilityID, purpose, input)
	data, _, err := c.Post("/api/connector-actions/proposals", body)
	if err != nil {
		return classifyAPIError(err, flags)
	}
	return printOutputWithFlags(cmd.OutOrStdout(), extractData(data), flags)
}

func buildFUBProposalBody(opts fubCommonOptions, capabilityID, purpose string, input map[string]any) map[string]any {
	body := map[string]any{
		"transactionId":  opts.dealID,
		"connectorId":    fubConnectorID,
		"capabilityId":   capabilityID,
		"purpose":        purpose,
		"idempotencyKey": actionIdempotencyKey("fub-proposal", opts.idempotencyKey),
		"actionInput":    input,
	}
	if opts.taskID != "" {
		body["taskId"] = opts.taskID
	}
	return body
}

func enrichFUBInputWithContext(c *client.Client, opts fubCommonOptions, input map[string]any) {
	context := map[string]any{}
	if opts.dealID != "" {
		if data, err := c.Get("/api/transactions/"+opts.dealID, nil); err == nil {
			context["deal"] = rawJSONAny(extractData(data))
		} else {
			context["dealError"] = err.Error()
		}
	}
	if opts.dealID != "" && opts.taskID != "" {
		if data, err := c.Get("/api/transactions/"+opts.dealID+"/tasks", nil); err == nil {
			context["task"] = findRawObjectByID(extractDataArray(data), opts.taskID)
		} else {
			context["taskError"] = err.Error()
		}
	}
	if len(context) > 0 {
		input["theCloseContext"] = context
	}
}

func parseJSONObjectFlag(name, value string) (map[string]any, error) {
	if value == "" {
		return map[string]any{}, nil
	}
	var parsed map[string]any
	if err := json.Unmarshal([]byte(value), &parsed); err != nil {
		return nil, fmt.Errorf("parsing --%s: %w", name, err)
	}
	if parsed == nil {
		parsed = map[string]any{}
	}
	return parsed, nil
}

func parseFUBMapping(mappingJSON, pipelineKey, stageKey string) (map[string]any, map[string]any, error) {
	var mapping map[string]any
	if mappingJSON != "" {
		parsed, err := parseJSONObjectFlag("account-mapping-json", mappingJSON)
		if err != nil {
			return nil, nil, err
		}
		mapping = parsed
	}
	status := map[string]any{
		"status":      "missing_mapping",
		"pipelineKey": pipelineKey,
		"stageKey":    stageKey,
	}
	missing := []string{}
	if mapping == nil {
		missing = append(missing, "accountMapping")
	}
	if pipelineKey == "" {
		missing = append(missing, "pipelineKey")
	}
	if stageKey == "" {
		missing = append(missing, "stageKey")
	}
	if len(missing) == 0 {
		status["status"] = "provided"
	} else {
		status["missing"] = missing
	}
	return mapping, status, nil
}

func defaultPurpose(value, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}

func rawJSONAny(data json.RawMessage) any {
	var out any
	if json.Unmarshal(data, &out) == nil {
		return out
	}
	return string(data)
}

func findRawObjectByID(data json.RawMessage, id string) any {
	var items []map[string]any
	if json.Unmarshal(data, &items) != nil {
		return nil
	}
	for _, item := range items {
		if itemID, _ := item["id"].(string); itemID == id {
			return item
		}
	}
	return map[string]any{"id": id, "status": "not_found_in_deal_tasks"}
}
