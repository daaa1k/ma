package codex_test

import (
	"strings"
	"testing"

	"github.com/daaa1k/ma/internal/formats/codex"
	"github.com/daaa1k/ma/internal/model"
)

func TestDecode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		input        string
		wantServers  map[string]model.Server
		wantWarnings int
		wantErr      bool
	}{
		{
			name: "stdio",
			input: `
[mcp_servers.context7]
command = "npx"
args = ["-y", "@upstash/context7-mcp"]
`,
			wantServers: map[string]model.Server{
				"context7": {
					Type:    model.TypeStdio,
					Command: "npx",
					Args:    []string{"-y", "@upstash/context7-mcp"},
				},
			},
		},
		{
			name: "http_with_bearer",
			input: `
[mcp_servers.figma]
url = "https://mcp.figma.com/mcp"
bearer_token_env_var = "FIGMA_OAUTH_TOKEN"
`,
			wantServers: map[string]model.Server{
				"figma": {
					Type: model.TypeHTTP,
					URL:  "https://mcp.figma.com/mcp",
					Headers: map[string]string{
						"Authorization": "Bearer ${FIGMA_OAUTH_TOKEN}",
					},
				},
			},
			wantWarnings: 1,
		},
		{
			name: "http_with_extra_headers",
			input: `
[mcp_servers.figma]
url = "https://mcp.figma.com/mcp"
bearer_token_env_var = "TOKEN"
[mcp_servers.figma.http_headers]
"X-Region" = "us-east-1"
`,
			wantServers: map[string]model.Server{
				"figma": {
					Type: model.TypeHTTP,
					URL:  "https://mcp.figma.com/mcp",
					Headers: map[string]string{
						"Authorization": "Bearer ${TOKEN}",
						"X-Region":      "us-east-1",
					},
				},
			},
			wantWarnings: 1,
		},
		{
			name:    "invalid_toml",
			input:   `[bad toml`,
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cfg, warnings, err := codex.Decode([]byte(tc.input))
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(warnings) != tc.wantWarnings {
				t.Errorf("warnings count: got %d, want %d: %v", len(warnings), tc.wantWarnings, warnings)
			}
			for name, want := range tc.wantServers {
				got, ok := cfg.Servers[name]
				if !ok {
					t.Fatalf("server %q not found", name)
				}
				if got.Type != want.Type {
					t.Errorf("[%s] Type: got %q, want %q", name, got.Type, want.Type)
				}
				if got.URL != want.URL {
					t.Errorf("[%s] URL: got %q, want %q", name, got.URL, want.URL)
				}
				for k, v := range want.Headers {
					if got.Headers[k] != v {
						t.Errorf("[%s] Headers[%q]: got %q, want %q", name, k, got.Headers[k], v)
					}
				}
			}
		})
	}
}

func TestEncode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		input        *model.Config
		wantWarnings int
		wantErr      bool
		checkOutput  func(t *testing.T, output string)
	}{
		{
			name: "stdio_roundtrip",
			input: &model.Config{Servers: map[string]model.Server{
				"srv": {Type: model.TypeStdio, Command: "npx", Args: []string{"-y", "pkg"}},
			}},
			checkOutput: func(t *testing.T, output string) {
				t.Helper()
				if !strings.Contains(output, `command = "npx"`) {
					t.Errorf("output missing command: %s", output)
				}
			},
		},
		{
			name: "http_bearer_roundtrip",
			input: &model.Config{Servers: map[string]model.Server{
				"srv": {
					Type: model.TypeHTTP,
					URL:  "https://example.com/mcp",
					Headers: map[string]string{
						"Authorization": "Bearer ${MY_TOKEN}",
					},
				},
			}},
			checkOutput: func(t *testing.T, output string) {
				t.Helper()
				if !strings.Contains(output, `bearer_token_env_var = "MY_TOKEN"`) {
					t.Errorf("bearer_token_env_var not found in output: %s", output)
				}
			},
		},
		{
			name: "sse_skipped_with_warning",
			input: &model.Config{Servers: map[string]model.Server{
				"sse-srv": {Type: model.TypeSSE, URL: "https://example.com/sse"},
			}},
			wantWarnings: 1,
			checkOutput: func(t *testing.T, output string) {
				t.Helper()
				if strings.Contains(output, "sse-srv") {
					t.Errorf("SSE server should be skipped but appears in output: %s", output)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			data, warnings, err := codex.Encode(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(warnings) != tc.wantWarnings {
				t.Errorf("warnings count: got %d, want %d: %v", len(warnings), tc.wantWarnings, warnings)
			}
			if tc.checkOutput != nil {
				tc.checkOutput(t, string(data))
			}
		})
	}
}
