package launcher

import (
	"os"

	"github.com/daaa1k/ma/internal/convert"
)

// CopilotLauncher launches GitHub Copilot CLI with an injected MCP config via
// the --additional-mcp-config flag.
//
// The --additional-mcp-config flag accepts JSON content directly (not a file
// path), so BuildArgs reads the temp file and inlines the JSON string.
//
// Resulting command:
//
//	copilot --additional-mcp-config <json-content> [extra-args...]
type CopilotLauncher struct{}

func (CopilotLauncher) Name() string                 { return "copilot" }
func (CopilotLauncher) Binary() string               { return "copilot" }
func (CopilotLauncher) TargetFormat() convert.Format { return convert.FormatCopilot }

func (CopilotLauncher) BuildArgs(configFile string, extraArgs []string) []string {
	data, err := os.ReadFile(configFile)
	var jsonContent string
	if err == nil {
		jsonContent = string(data)
	}
	args := []string{"--additional-mcp-config", jsonContent}
	return append(args, extraArgs...)
}

func (CopilotLauncher) EnvVars(_ string) []string { return nil }
