package launcher

import "github.com/daaa1k/ma/internal/convert"

// CodexLauncher launches OpenAI Codex CLI with an injected MCP config via the
// --config flag.
//
// Resulting command:
//
//	codex --config <tmpfile.toml> [extra-args...]
type CodexLauncher struct{}

func (CodexLauncher) Name() string                { return "codex" }
func (CodexLauncher) Binary() string              { return "codex" }
func (CodexLauncher) TargetFormat() convert.Format { return convert.FormatCodex }

func (CodexLauncher) BuildArgs(configFile string, extraArgs []string) []string {
	args := []string{"--config", configFile}
	return append(args, extraArgs...)
}

func (CodexLauncher) EnvVars(_ string) []string { return nil }
