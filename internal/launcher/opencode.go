package launcher

import "github.com/daaa1k/ma/internal/convert"

// OpenCodeLauncher launches OpenCode with an injected MCP config.
// OpenCode reads its configuration from the path given by the OPENCODE_CONFIG
// environment variable when set, falling back to the default XDG config
// directory. We set OPENCODE_CONFIG to the converted temp file so that the
// existing user config is superseded for this session.
//
// Resulting invocation:
//
//	OPENCODE_CONFIG=<tmpfile> opencode [extra-args...]
type OpenCodeLauncher struct{}

func (OpenCodeLauncher) Name() string                { return "opencode" }
func (OpenCodeLauncher) Binary() string              { return "opencode" }
func (OpenCodeLauncher) TargetFormat() convert.Format { return convert.FormatOpenCode }

func (OpenCodeLauncher) BuildArgs(_ string, extraArgs []string) []string {
	return extraArgs
}

func (OpenCodeLauncher) EnvVars(configFile string) []string {
	return []string{"OPENCODE_CONFIG=" + configFile}
}
