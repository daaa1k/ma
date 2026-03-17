package launcher

import (
	"os"

	"github.com/daaa1k/ma/internal/convert"
	"github.com/daaa1k/ma/internal/formats/codex"
)

// CodexLauncher launches OpenAI Codex CLI with injected MCP config via
// -c mcp_servers.NAME=<inline-table> overrides.
//
// The codex CLI's -c flag accepts key=value overrides (not a config file path),
// so each MCP server is injected as an individual -c override.
//
// Resulting command:
//
//	codex -c mcp_servers.NAME={...} [...] [extra-args...]
type CodexLauncher struct{}

func (CodexLauncher) Name() string                 { return "codex" }
func (CodexLauncher) Binary() string               { return "codex" }
func (CodexLauncher) TargetFormat() convert.Format { return convert.FormatCodex }

func (CodexLauncher) BuildArgs(configFile string, extraArgs []string) []string {
	data, err := os.ReadFile(configFile)
	if err != nil {
		return extraArgs
	}
	overrides, err := codex.BuildConfigOverrides(data)
	if err != nil {
		return extraArgs
	}
	return append(overrides, extraArgs...)
}

func (CodexLauncher) EnvVars(_ string) []string { return nil }
