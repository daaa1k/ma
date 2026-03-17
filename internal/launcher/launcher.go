// Package launcher provides the tool-launch abstraction used by each ma
// subcommand. A Launcher knows how to:
//
//  1. Convert a canonical MCP config into the format expected by a specific tool.
//  2. Build the argv list that the tool should be started with (including any
//     flag that injects the converted config file).
//  3. Optionally supply additional environment variables.
//
// The typical execution flow is:
//
//	source file  ──decode──►  canonical  ──encode──►  tmp file
//	                                                       │
//	                                                       ▼
//	                                      tool binary + argv ──exec──►
package launcher

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/daaa1k/ma/internal/convert"
	"github.com/daaa1k/ma/internal/model"
)

// Launcher describes how to launch a specific AI tool with an injected MCP
// configuration.
type Launcher interface {
	// Name returns the tool's user-facing name (e.g. "copilot").
	Name() string

	// Binary returns the executable that will be exec'd.
	Binary() string

	// TargetFormat returns the convert.Format the config must be converted to.
	TargetFormat() convert.Format

	// BuildArgs returns the complete argv (excluding binary) to pass to the
	// tool. configFile is the path of the already-written converted config.
	// extraArgs are the arguments the user appended after "--" on the ma CLI.
	BuildArgs(configFile string, extraArgs []string) []string

	// EnvVars returns additional environment variables to set before exec.
	// Returns nil if none are required.
	EnvVars(configFile string) []string
}

// Run is the shared entry-point used by every tool subcommand.
// It:
//  1. Encodes the canonical config to the launcher's target format.
//  2. Writes the result to a deterministic temp file (overwritten on each run).
//  3. Replaces the current process with the target tool via execTool.
//
// Warnings are written to errW (typically os.Stderr).
func Run(l Launcher, cfg *model.Config, extraArgs []string, errW io.Writer) error {
	result, err := convert.Encode(l.TargetFormat(), cfg)
	if err != nil {
		return fmt.Errorf("encode config for %s: %w", l.Name(), err)
	}
	convert.WriteWarnings(errW, result.Warnings)

	tmpFile := filepath.Join(os.TempDir(), fmt.Sprintf("ma-%s-mcp-config%s", l.Name(), configExt(l.TargetFormat())))
	if err := os.WriteFile(tmpFile, result.Data, 0o600); err != nil {
		return fmt.Errorf("write temp config: %w", err)
	}

	argv := l.BuildArgs(tmpFile, extraArgs)
	env := buildEnv(l.EnvVars(tmpFile))

	return execTool(l.Binary(), argv, env)
}

// configExt returns the file extension appropriate for the target format.
func configExt(f convert.Format) string {
	if f == convert.FormatCodex {
		return ".toml"
	}
	return ".json"
}

// buildEnv merges the current process environment with extra variables.
func buildEnv(extra []string) []string {
	if len(extra) == 0 {
		return os.Environ()
	}
	return append(os.Environ(), extra...)
}
