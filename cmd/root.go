// Package cmd contains the cobra command definitions for the ma CLI.
package cmd

import (
	"errors"
	"fmt"
	"io/fs"
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

Supported tools: copilot, opencode, codex, cursor`,
		SilenceUsage: true,
	}

	root.PersistentFlags().StringVarP(&configFlag, "config", "c", "", "MCP config file path (Claude Code JSON format)")

	root.AddCommand(
		newToolCmd("copilot", "Launch GitHub Copilot CLI with MCP config injected via --additional-mcp-config", launcher.CopilotLauncher{}, &configFlag),
		newToolCmd("opencode", "Launch OpenCode with MCP config injected via OPENCODE_CONFIG", launcher.OpenCodeLauncher{}, &configFlag),
		newToolCmd("codex", "Launch Codex CLI with MCP config injected via --config", launcher.CodexLauncher{}, &configFlag),
		newCursorCmd(&configFlag),
	)

	return root
}

func newCursorCmd(configFlag *string) *cobra.Command {
	return &cobra.Command{
		Use:   "cursor",
		Short: "Symlink shared .mcp.json to .cursor/mcp.json and launch Cursor CLI",
		Long: `Resolves the same MCP config as other ma commands, symlinks it to
<workspace>/.cursor/mcp.json (Cursor uses the same JSON schema), then runs
cursor-agent, or agent if cursor-agent is not in PATH.

Workspace is the current directory, or the path given by --workspace in the
arguments after -- (forwarded to the Cursor CLI).`,
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			src, err := resolveConfigPath(*configFlag)
			if err != nil {
				return err
			}
			return launcher.RunCursor(src, args, cmd.ErrOrStderr())
		},
		Example: "  ma cursor\n  ma cursor -- --workspace /path/to/project",
	}
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

// resolveConfigPath returns the path to the first existing MCP config among:
// --config flag, ./.mcp.json, ~/.mcp.json.
func resolveConfigPath(flagPath string) (string, error) {
	candidates := resolveCandidates(flagPath)

	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		} else if !errors.Is(err, fs.ErrNotExist) {
			return "", fmt.Errorf("stat config %s: %w", path, err)
		}
	}

	return "", fmt.Errorf("no MCP config found; tried: %v\nCreate .mcp.json or use --config", candidates)
}

// loadConfig reads and decodes the MCP source config from resolveConfigPath.
func loadConfig(flagPath string) (*model.Config, error) {
	path, err := resolveConfigPath(flagPath)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}
	cfg, _, err := convert.Decode(convert.FormatClaude, data)
	if err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}
	return cfg, nil
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
