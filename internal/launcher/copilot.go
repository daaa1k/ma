package launcher

import "github.com/daaa1k/ma/internal/convert"

// CopilotLauncher launches GitHub Copilot CLI with an injected MCP config via
// the --additional-mcp-config flag.
//
// Resulting command:
//
//	copilot --additional-mcp-config <tmpfile> [extra-args...]
type CopilotLauncher struct{}

func (CopilotLauncher) Name() string                 { return "copilot" }
func (CopilotLauncher) Binary() string               { return "copilot" }
func (CopilotLauncher) TargetFormat() convert.Format { return convert.FormatCopilot }

func (CopilotLauncher) BuildArgs(configFile string, extraArgs []string) []string {
	args := []string{"--additional-mcp-config", configFile}
	return append(args, extraArgs...)
}

func (CopilotLauncher) EnvVars(_ string) []string { return nil }
