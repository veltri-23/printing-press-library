// Copyright 2026 cathrynlavery. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	appconfig "theclose-pp-cli/internal/config"
)

func newConfigCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage The Close CLI configuration",
		RunE:  parentNoSubcommandRunE(flags),
	}

	cmd.AddCommand(newConfigGetCmd(flags))
	cmd.AddCommand(newConfigSetCmd(flags))

	return cmd
}

func newConfigGetCmd(flags *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "get",
		Short: "Show effective configuration without secrets",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := appconfig.Load(flags.configPath)
			if err != nil {
				return configErr(err)
			}
			_ = cfg.AuthHeader()
			out := map[string]any{
				"base_url":          cfg.BaseURL,
				"config_path":       cfg.Path,
				"api_token_set":     cfg.AuthHeader() != "",
				"api_token_source":  cfg.AuthSource,
				"canonical_env":     []string{"THECLOSE_BASE_URL", "THECLOSE_API_TOKEN"},
				"legacy_token_env":  "CLOSE_DEVELOPER_BEARER_AUTH",
				"token_redaction":   "enabled",
				"session_auth_note": "TC session headers are browser/internal only; agents should use bearer tokens.",
			}
			if flags.asJSON {
				return printJSONFiltered(cmd.OutOrStdout(), out, flags)
			}
			w := cmd.OutOrStdout()
			fmt.Fprintf(w, "base_url: %s\n", out["base_url"])
			fmt.Fprintf(w, "config_path: %s\n", out["config_path"])
			fmt.Fprintf(w, "api_token_set: %v\n", out["api_token_set"])
			fmt.Fprintf(w, "api_token_source: %s\n", out["api_token_source"])
			fmt.Fprintln(w, "canonical_env: THECLOSE_BASE_URL, THECLOSE_API_TOKEN")
			return nil
		},
	}
}

func newConfigSetCmd(flags *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "set <base-url|api-token> <value>",
		Short: "Persist base URL or API token",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := strings.ToLower(strings.TrimSpace(args[0]))
			value := strings.TrimSpace(args[1])
			if value == "" {
				return usageErr(fmt.Errorf("value cannot be empty"))
			}
			cfg, err := appconfig.Load(flags.configPath)
			if err != nil {
				return configErr(err)
			}

			switch key {
			case "base-url", "base_url":
				cfg.BaseURL = strings.TrimRight(value, "/")
			case "api-token", "api_token", "token":
				cfg.AuthHeaderVal = ""
				cfg.AccessToken = value
			default:
				return usageErr(fmt.Errorf("unknown config key %q (valid: base-url, api-token)", args[0]))
			}

			if err := cfg.Save(); err != nil {
				return configErr(err)
			}
			if flags.asJSON {
				return printJSONFiltered(cmd.OutOrStdout(), map[string]any{
					"saved":       true,
					"key":         key,
					"config_path": cfg.Path,
				}, flags)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Saved %s to %s\n", key, cfg.Path)
			return nil
		},
	}
}
