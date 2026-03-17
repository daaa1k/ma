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

	const fakeConfig = "/tmp/fake-config.json"
	extra := []string{"--foo", "bar"}

	tests := []struct {
		name       string
		l          launcher.Launcher
		wantArgs   []string // prefix of expected argv
		wantEnvKey string   // non-empty: expect this key in EnvVars
	}{
		{
			name:     "copilot",
			l:        launcher.CopilotLauncher{},
			wantArgs: []string{"--additional-mcp-config", fakeConfig, "--foo", "bar"},
		},
		{
			name:       "opencode",
			l:          launcher.OpenCodeLauncher{},
			wantArgs:   []string{"--foo", "bar"},
			wantEnvKey: "OPENCODE_CONFIG=",
		},
		{
			name:     "codex",
			l:        launcher.CodexLauncher{},
			wantArgs: []string{"--config", fakeConfig, "--foo", "bar"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := tc.l.BuildArgs(fakeConfig, extra)
			if len(got) != len(tc.wantArgs) {
				t.Fatalf("BuildArgs length: got %d %v, want %d %v", len(got), got, len(tc.wantArgs), tc.wantArgs)
			}
			for i, w := range tc.wantArgs {
				if got[i] != w {
					t.Errorf("BuildArgs[%d]: got %q, want %q", i, got[i], w)
				}
			}

			if tc.wantEnvKey != "" {
				envVars := tc.l.EnvVars(fakeConfig)
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

// TestRunWritesTempFile verifies that launcher.Run writes a temp file with the
// correct extension without actually exec'ing (we use a fake binary name that
// does not exist, so we expect a "not found" error, not a conversion error).
func TestRunWritesTempFile(t *testing.T) {
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

			err := launcher.Run(l, cfg, nil, os.Stderr)
			// We expect an error because the tool binary is not in PATH in tests.
			// The important thing is that the error is about PATH lookup, not
			// about config encoding.
			if err == nil {
				t.Fatal("expected error (binary not in PATH), got nil")
			}
			if strings.Contains(err.Error(), "encode config") {
				t.Errorf("unexpected encode error: %v", err)
			}

			// Verify the temp file was created.
			ext := ".json"
			if l.TargetFormat() == convert.FormatCodex {
				ext = ".toml"
			}
			tmpFile := filepath.Join(os.TempDir(), "ma-"+l.Name()+"-mcp-config"+ext)
			if _, statErr := os.Stat(tmpFile); os.IsNotExist(statErr) {
				t.Errorf("temp file %s was not created", tmpFile)
			}
		})
	}
}
