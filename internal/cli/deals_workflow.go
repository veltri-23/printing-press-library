// Copyright 2026 cathrynlavery. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

func newDealsCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deals",
		Short: "High-level deal workspace commands for transaction coordination",
		RunE:  parentNoSubcommandRunE(flags),
		Annotations: map[string]string{
			"mcp:read-only": "true",
		},
	}
	cmd.AddCommand(newDealsListCmd(flags))
	cmd.AddCommand(newDealsGetCmd(flags))
	cmd.AddCommand(newDealsCreateCmd(flags))
	cmd.AddCommand(newDealsUpdateStatusCmd(flags))
	cmd.AddCommand(newDealsArchiveCmd(flags))
	return cmd
}

func newDealsListCmd(flags *rootFlags) *cobra.Command {
	var q, status, health, closingBefore, closingAfter string
	var archive bool
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"search"},
		Short:   "List or search deals by address, status, health, and closing filters",
		Annotations: map[string]string{
			"mcp:read-only": "true",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := flags.newClient()
			if err != nil {
				return err
			}
			params := map[string]string{}
			for k, v := range map[string]string{"q": q, "status": status, "health": health, "closing_before": closingBefore, "closing_after": closingAfter} {
				if v != "" {
					params[k] = v
				}
			}
			if archive {
				params["archive"] = "true"
			}
			data, err := c.Get("/api/transactions", params)
			if err != nil {
				return classifyAPIError(err, flags)
			}
			return printOutputWithFlags(cmd.OutOrStdout(), extractDataArray(data), flags)
		},
	}
	cmd.Flags().StringVar(&q, "q", "", "Search address text")
	cmd.Flags().StringVar(&status, "status", "", "Deal lifecycle status")
	cmd.Flags().StringVar(&health, "health", "", "Health badge")
	cmd.Flags().StringVar(&closingBefore, "closing-before", "", "Closing date before YYYY-MM-DD")
	cmd.Flags().StringVar(&closingAfter, "closing-after", "", "Closing date after YYYY-MM-DD")
	cmd.Flags().BoolVar(&archive, "archive", false, "Include archived deals")
	return cmd
}

func newDealsGetCmd(flags *rootFlags) *cobra.Command {
	var workspace bool
	cmd := &cobra.Command{
		Use:   "get <deal-id>",
		Short: "Inspect a deal, optionally with its workspace context",
		Args:  cobra.ExactArgs(1),
		Annotations: map[string]string{
			"mcp:read-only": "true",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := flags.newClient()
			if err != nil {
				return err
			}
			id := args[0]
			deal, err := c.Get("/api/transactions/"+id, nil)
			if err != nil {
				return classifyAPIError(err, flags)
			}
			if !workspace {
				return printOutputWithFlags(cmd.OutOrStdout(), extractData(deal), flags)
			}
			read := func(path string, params map[string]string) json.RawMessage {
				data, err := c.Get(path, params)
				if err != nil {
					b, _ := json.Marshal(map[string]string{"error": err.Error()})
					return b
				}
				return extractDataArray(data)
			}
			out := map[string]json.RawMessage{
				"deal":      extractData(deal),
				"tasks":     read("/api/transactions/"+id+"/tasks", nil),
				"fields":    read("/api/transactions/"+id+"/fields", nil),
				"documents": read("/api/transactions/"+id+"/documents", nil),
				"contacts":  read("/api/transactions/"+id+"/contacts", nil),
				"emails":    read("/api/transactions/"+id+"/emails", nil),
				"events":    read("/api/transactions/"+id+"/events", map[string]string{"limit": "50"}),
			}
			data, _ := json.Marshal(out)
			return printOutputWithFlags(cmd.OutOrStdout(), data, flags)
		},
	}
	cmd.Flags().BoolVar(&workspace, "workspace", false, "Include tasks, fields, documents, contacts, emails, and events")
	return cmd
}

func newDealsCreateCmd(flags *rootFlags) *cobra.Command {
	var street, city, state, zip, dealType string
	cmd := &cobra.Command{
		Use:   "create --street <street>",
		Short: "Create a deal in The Close",
		RunE: func(cmd *cobra.Command, args []string) error {
			if street == "" && !flags.dryRun {
				return usageErr(fmt.Errorf("--street is required"))
			}
			c, err := flags.newClient()
			if err != nil {
				return err
			}
			body := map[string]any{"address": map[string]any{"street": street, "city": city, "state": state, "zip": zip}}
			if dealType != "" {
				body["type"] = dealType
			}
			data, _, err := c.Post("/api/transactions", body)
			if err != nil {
				return classifyAPIError(err, flags)
			}
			return printOutputWithFlags(cmd.OutOrStdout(), extractData(data), flags)
		},
	}
	cmd.Flags().StringVar(&street, "street", "", "Street address")
	cmd.Flags().StringVar(&city, "city", "", "City")
	cmd.Flags().StringVar(&state, "state", "", "State")
	cmd.Flags().StringVar(&zip, "zip", "", "ZIP code")
	cmd.Flags().StringVar(&dealType, "type", "", "Deal type")
	return cmd
}

func newDealsUpdateStatusCmd(flags *rootFlags) *cobra.Command {
	var status string
	cmd := &cobra.Command{
		Use:   "update-status <deal-id> --status <status>",
		Short: "Update a deal lifecycle status through The Close",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if status == "" && !flags.dryRun {
				return usageErr(fmt.Errorf("--status is required"))
			}
			c, err := flags.newClient()
			if err != nil {
				return err
			}
			data, _, err := c.Patch("/api/transactions/"+args[0], map[string]any{"status": status})
			if err != nil {
				return classifyAPIError(err, flags)
			}
			return printOutputWithFlags(cmd.OutOrStdout(), extractData(data), flags)
		},
	}
	cmd.Flags().StringVar(&status, "status", "", "New lifecycle status")
	return cmd
}

func newDealsArchiveCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "archive <deal-id>",
		Short: "Archive or soft-delete a deal where the API allows it",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := flags.newClient()
			if err != nil {
				return err
			}
			data, _, err := c.Delete("/api/transactions/" + args[0])
			if err != nil {
				return classifyDeleteError(err, flags)
			}
			return printOutputWithFlags(cmd.OutOrStdout(), extractData(data), flags)
		},
	}
	return cmd
}

func extractData(data json.RawMessage) json.RawMessage {
	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if json.Unmarshal(data, &envelope) == nil && envelope.Data != nil {
		return envelope.Data
	}
	return data
}

func extractDataArray(data json.RawMessage) json.RawMessage {
	return extractData(data)
}
