package launcher_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/daaa1k/ma/internal/convert"
	"github.com/daaa1k/ma/internal/launcher"
	"github.com/daaa1k/ma/internal/model"
)

// TestLauncherBuildArgs verifies that each Launcher produces the expected argv
// and environment without actually exec'ing anything.
func TestLauncherBuildArgs(t *testing.T) {
	t.Parallel()

	extra := []string{"--foo", "bar"}

	// Write a minimal Copilot JSON config for the copilot test case.
	copilotJSON := `{"mcpServers":{"srv":{"command":"node","args":["s.js"]}}}`
	copilotFile := filepath.Join(t.TempDir(), "copilot.json")
	if err := os.WriteFile(copilotFile, []byte(copilotJSON), 0o600); err != nil {
		t.Fatal(err)
	}

	// Write a minimal Codex TOML config for the codex test case.
	codexTOML := "[mcp_servers.srv]\ncommand = \"node\"\nargs = [\"s.js\"]\n"
	codexFile := filepath.Join(t.TempDir(), "codex.toml")
	if err := os.WriteFile(codexFile, []byte(codexTOML), 0o600); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name       string
		l          launcher.Launcher
		configFile string
		checkArgs  func(t *testing.T, got []string)
		wantEnvKey string // non-empty: expect this key prefix in EnvVars
	}{
		{
			name:       "copilot",
			l:          launcher.CopilotLauncher{},
			configFile: copilotFile,
			checkArgs: func(t *testing.T, got []string) {
				t.Helper()
				if len(got) < 2 || got[0] != "--additional-mcp-config" {
					t.Fatalf("expected --additional-mcp-config as first arg, got %v", got)
				}
				if !strings.Contains(got[1], "mcpServers") {
					t.Errorf("expected JSON content in arg[1], got %q", got[1])
				}
				if got[len(got)-2] != "--foo" || got[len(got)-1] != "bar" {
					t.Errorf("extra args not appended: %v", got)
				}
			},
		},
		{
			name:       "opencode",
			l:          launcher.OpenCodeLauncher{},
			configFile: "/tmp/fake-opencode.json",
			checkArgs: func(t *testing.T, got []string) {
				t.Helper()
				if len(got) != 2 || got[0] != "--foo" || got[1] != "bar" {
					t.Errorf("expected only extra args, got %v", got)
				}
			},
			wantEnvKey: "OPENCODE_CONFIG=",
		},
		{
			name:       "codex",
			l:          launcher.CodexLauncher{},
			configFile: codexFile,
			checkArgs: func(t *testing.T, got []string) {
				t.Helper()
				// Expect: -c mcp_servers.srv={...} --foo bar
				if len(got) < 4 {
					t.Fatalf("expected at least 4 args, got %d: %v", len(got), got)
				}
				if got[0] != "-c" {
					t.Errorf("arg[0]: got %q, want \"-c\"", got[0])
				}
				if !strings.HasPrefix(got[1], "mcp_servers.srv=") {
					t.Errorf("arg[1]: got %q, want mcp_servers.srv=...", got[1])
				}
				if got[len(got)-2] != "--foo" || got[len(got)-1] != "bar" {
					t.Errorf("extra args not appended: %v", got)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := tc.l.BuildArgs(tc.configFile, extra)
			tc.checkArgs(t, got)

			if tc.wantEnvKey != "" {
				envVars := tc.l.EnvVars(tc.configFile)
				found := false
				for _, e := range envVars {
					if strings.HasPrefix(e, tc.wantEnvKey) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("EnvVars: expected entry with prefix %q, got %v", tc.wantEnvKey, envVars)
				}
			}
		})
	}
}

// TestLauncherTargetFormats ensures each launcher produces valid output for a
// sample config containing stdio, http, and sse servers (SSE for Codex is
// expected to be skipped with a warning).
func TestLauncherTargetFormats(t *testing.T) {
	t.Parallel()

	cfg := &model.Config{
		Servers: map[string]model.Server{
			"stdio-srv": {
				Type:    model.TypeStdio,
				Command: "npx",
				Args:    []string{"-y", "my-mcp"},
			},
			"http-srv": {
				Type:    model.TypeHTTP,
				URL:     "https://api.example.com/mcp",
				Headers: map[string]string{"Authorization": "Bearer ${TOKEN}"},
			},
			"sse-srv": {
				Type: model.TypeSSE,
				URL:  "https://api.example.com/sse",
			},
		},
	}

	launchers := []launcher.Launcher{
		launcher.CopilotLauncher{},
		launcher.OpenCodeLauncher{},
		launcher.CodexLauncher{},
	}

	for _, l := range launchers {
		t.Run(l.Name(), func(t *testing.T) {
			t.Parallel()
			result, err := convert.Encode(l.TargetFormat(), cfg)
			if err != nil {
				t.Fatalf("Encode: %v", err)
			}
			if len(result.Data) == 0 {
				t.Error("Encode produced empty output")
			}
			// Codex must skip SSE with exactly 1 warning.
			if l.TargetFormat() == convert.FormatCodex {
				if len(result.Warnings) != 1 {
					t.Errorf("Codex: expected 1 SSE warning, got %d: %v", len(result.Warnings), result.Warnings)
				}
			}
		})
	}
}

// TestWriteTempConfig verifies that WriteTempConfig writes a temp file with the
// correct extension and valid content without executing any external binary.
func TestWriteTempConfig(t *testing.T) {
	t.Parallel()

	cfg := &model.Config{
		Servers: map[string]model.Server{
			"srv": {Type: model.TypeStdio, Command: "node", Args: []string{"server.js"}},
		},
	}

	launchers := []launcher.Launcher{
		launcher.CopilotLauncher{},
		launcher.OpenCodeLauncher{},
		launcher.CodexLauncher{},
	}

	for _, l := range launchers {
		t.Run(l.Name(), func(t *testing.T) {
			t.Parallel()

			path, err := launcher.WriteTempConfig(l, cfg, os.Stderr)
			if err != nil {
				t.Fatalf("WriteTempConfig: %v", err)
			}

			ext := ".json"
			if l.TargetFormat() == convert.FormatCodex {
				ext = ".toml"
			}
			if !strings.HasSuffix(path, ext) {
				t.Errorf("expected temp file with extension %s, got %s", ext, path)
			}
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				t.Fatalf("temp file not readable: %v", readErr)
			}
			if len(data) == 0 {
				t.Error("temp file is empty")
			}
		})
	}
}
