// Package cmd contains the cobra command definitions for the ma CLI.
package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/daaa1k/ma/internal/convert"
	"github.com/daaa1k/ma/internal/launcher"
	"github.com/daaa1k/ma/internal/model"
	"github.com/spf13/cobra"
)

// NewRootCmd returns the root cobra command for the ma CLI.
func NewRootCmd(version string) *cobra.Command {
	var configFlag string

	root := &cobra.Command{
		Use:     "ma",
		Short:   "ma — MCP config adapter and tool launcher",
		Version: version,
		Long: `ma adapts a shared MCP server config and launches AI coding tools with it.

Source config (Claude Code JSON format, searched in order):
  1. --config flag
  2. ./.mcp.json  (current directory)
  3. ~/.mcp.json (home directory)

Supported tools: copilot, opencode, codex`,
		SilenceUsage: true,
	}

	root.PersistentFlags().StringVarP(&configFlag, "config", "c", "", "MCP config file path (Claude Code JSON format)")

	root.AddCommand(
		newToolCmd("copilot", "Launch GitHub Copilot CLI with MCP config injected via --additional-mcp-config", launcher.CopilotLauncher{}, &configFlag),
		newToolCmd("opencode", "Launch OpenCode with MCP config injected via OPENCODE_CONFIG", launcher.OpenCodeLauncher{}, &configFlag),
		newToolCmd("codex", "Launch Codex CLI with MCP config injected via --config", launcher.CodexLauncher{}, &configFlag),
	)

	return root
}

// newToolCmd builds a subcommand that launches a specific AI tool.
func newToolCmd(use, short string, l launcher.Launcher, configFlag *string) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		// Accept any number of positional args so that extra tool args after "--"
		// are collected by cobra automatically.
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig(*configFlag)
			if err != nil {
				return err
			}
			return launcher.Run(l, cfg, args, cmd.ErrOrStderr())
		},
		Example: fmt.Sprintf("  ma %s\n  ma %s -- --some-tool-flag", use, use),
	}
}

// loadConfig reads and decodes the MCP source config from the first location
// that exists among: --config flag, ./mcp.json, ~/.mcp.json.
func loadConfig(flagPath string) (*model.Config, error) {
	candidates := resolveCandidates(flagPath)

	for _, path := range candidates {
		data, err := os.ReadFile(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("read config %s: %w", path, err)
		}
		cfg, _, err := convert.Decode(convert.FormatClaude, data)
		if err != nil {
			return nil, fmt.Errorf("parse config %s: %w", path, err)
		}
		return cfg, nil
	}

	return nil, fmt.Errorf("no MCP config found; tried: %v\nCreate .mcp.json or use --config", candidates)
}

func resolveCandidates(flagPath string) []string {
	if flagPath != "" {
		return []string{flagPath}
	}
	candidates := []string{".mcp.json"}
	if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates, filepath.Join(home, ".mcp.json"))
	}
	return candidates
}
