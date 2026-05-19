// Copyright 2026 cathrynlavery. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

func newFieldsCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fields",
		Short: "High-level field review and mutation commands",
		RunE:  parentNoSubcommandRunE(flags),
	}
	cmd.AddCommand(newFieldsListCmd(flags))
	cmd.AddCommand(newFieldsSetCmd(flags))
	cmd.AddCommand(newFieldActionCmd(flags, "review", "review"))
	cmd.AddCommand(newFieldActionCmd(flags, "dispute", "dispute"))
	cmd.AddCommand(newFieldActionCmd(flags, "stale", "mark-stale"))
	cmd.AddCommand(newFieldsSummaryCmd(flags))
	return cmd
}

func newFieldsListCmd(flags *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "list <deal-id>",
		Short: "List field values for a deal",
		Args:  cobra.ExactArgs(1),
		Annotations: map[string]string{
			"mcp:read-only": "true",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := flags.newClient()
			if err != nil {
				return err
			}
			data, err := c.Get("/api/transactions/"+args[0]+"/fields", nil)
			if err != nil {
				return classifyAPIError(err, flags)
			}
			return printOutputWithFlags(cmd.OutOrStdout(), extractDataArray(data), flags)
		},
	}
}

func newFieldsSetCmd(flags *rootFlags) *cobra.Command {
	var value string
	cmd := &cobra.Command{
		Use:   "set <deal-id> <field-key> --value <json-or-string>",
		Short: "Set a field value through The Close",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if value == "" && !flags.dryRun {
				return usageErr(fmt.Errorf("--value is required"))
			}
			c, err := flags.newClient()
			if err != nil {
				return err
			}
			var parsed any = value
			_ = json.Unmarshal([]byte(value), &parsed)
			data, _, err := c.Patch("/api/transactions/"+args[0]+"/fields/"+args[1], map[string]any{"value": parsed})
			if err != nil {
				return classifyAPIError(err, flags)
			}
			return printOutputWithFlags(cmd.OutOrStdout(), extractData(data), flags)
		},
	}
	cmd.Flags().StringVar(&value, "value", "", "Field value as JSON or string")
	return cmd
}

func newFieldActionCmd(flags *rootFlags, verb, pathSuffix string) *cobra.Command {
	var reason, reviewedBy string
	cmd := &cobra.Command{
		Use:   verb + " <deal-id> <field-key>",
		Short: "Mark a field " + verb,
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := flags.newClient()
			if err != nil {
				return err
			}
			body := map[string]any{}
			if reason != "" {
				body["reason"] = reason
			}
			if reviewedBy != "" {
				body["reviewedBy"] = reviewedBy
			}
			data, _, err := c.Post("/api/transactions/"+args[0]+"/fields/"+args[1]+"/"+pathSuffix, body)
			if err != nil {
				return classifyAPIError(err, flags)
			}
			return printOutputWithFlags(cmd.OutOrStdout(), extractData(data), flags)
		},
	}
	cmd.Flags().StringVar(&reason, "reason", "", "Reason for dispute/stale action")
	cmd.Flags().StringVar(&reviewedBy, "reviewed-by", "agent", "Reviewer identifier")
	return cmd
}

func newFieldsSummaryCmd(flags *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "summary <deal-id>",
		Short: "Summarize field review state for a deal",
		Args:  cobra.ExactArgs(1),
		Annotations: map[string]string{
			"mcp:read-only": "true",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := flags.newClient()
			if err != nil {
				return err
			}
			data, err := c.Get("/api/transactions/"+args[0]+"/fields/review-summary", nil)
			if err != nil {
				return classifyAPIError(err, flags)
			}
			return printOutputWithFlags(cmd.OutOrStdout(), extractData(data), flags)
		},
	}
}
