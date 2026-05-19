// Copyright 2026 cathrynlavery. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newTasksReadyWorkCmd(flags *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "ready-work",
		Short: "List open tasks ready for agent work",
		Annotations: map[string]string{
			"mcp:read-only": "true",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := flags.newClient()
			if err != nil {
				return err
			}
			data, err := c.Get("/api/tasks", map[string]string{"status": "open"})
			if err != nil {
				return classifyAPIError(err, flags)
			}
			return printOutputWithFlags(cmd.OutOrStdout(), extractDataArray(data), flags)
		},
	}
}

func newTasksByDealCmd(flags *rootFlags) *cobra.Command {
	var status string
	cmd := &cobra.Command{
		Use:   "by-deal <deal-id>",
		Short: "List tasks for one deal",
		Args:  cobra.ExactArgs(1),
		Annotations: map[string]string{
			"mcp:read-only": "true",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := flags.newClient()
			if err != nil {
				return err
			}
			params := map[string]string{}
			if status != "" {
				params["status"] = status
			}
			data, err := c.Get("/api/transactions/"+args[0]+"/tasks", params)
			if err != nil {
				return classifyAPIError(err, flags)
			}
			return printOutputWithFlags(cmd.OutOrStdout(), extractDataArray(data), flags)
		},
	}
	cmd.Flags().StringVar(&status, "status", "", "Filter by task status")
	return cmd
}

func newTaskStatusCmd(flags *rootFlags, verb, status string) *cobra.Command {
	return &cobra.Command{
		Use:   verb + " <task-id>",
		Short: fmt.Sprintf("Set task status to %s", status),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := flags.newClient()
			if err != nil {
				return err
			}
			data, _, err := c.Patch("/api/tasks/"+args[0], map[string]any{"status": status})
			if err != nil {
				return classifyAPIError(err, flags)
			}
			return printOutputWithFlags(cmd.OutOrStdout(), extractData(data), flags)
		},
	}
}
