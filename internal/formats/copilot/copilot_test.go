package copilot_test

import (
	"encoding/json"
	"testing"

	"github.com/daaa1k/ma/internal/formats/copilot"
	"github.com/daaa1k/ma/internal/model"
)

func TestDecode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		input       string
		wantServers map[string]model.Server
		wantErr     bool
	}{
		{
			name: "local_stdio_with_tools",
			input: `{
				"mcpServers": {
					"sentry": {
						"type": "local",
						"command": "npx",
						"args": ["@sentry/mcp-server@latest"],
						"env": {"SENTRY_HOST": "COPILOT_MCP_SENTRY_HOST"},
						"tools": ["get_issue_details", "get_issue_summary"]
					}
				}
			}`,
			wantServers: map[string]model.Server{
				"sentry": {
					Type:    model.TypeStdio,
					Command: "npx",
					Args:    []string{"@sentry/mcp-server@latest"},
					Env:     map[string]string{"SENTRY_HOST": "COPILOT_MCP_SENTRY_HOST"},
					Tools:   []string{"get_issue_details", "get_issue_summary"},
				},
			},
		},
		{
			name: "http_with_tools",
			input: `{
				"mcpServers": {
					"github-mcp-server": {
						"type": "http",
						"url": "https://api.githubcopilot.com/mcp/readonly",
						"tools": ["*"],
						"headers": {"X-MCP-Toolsets": "repos,issues"}
					}
				}
			}`,
			wantServers: map[string]model.Server{
				"github-mcp-server": {
					Type:    model.TypeHTTP,
					URL:     "https://api.githubcopilot.com/mcp/readonly",
					Headers: map[string]string{"X-MCP-Toolsets": "repos,issues"},
					Tools:   []string{"*"},
				},
			},
		},
		{
			name: "sse",
			input: `{
				"mcpServers": {
					"cloudflare": {
						"type": "sse",
						"url": "https://docs.mcp.cloudflare.com/sse",
						"tools": ["*"]
					}
				}
			}`,
			wantServers: map[string]model.Server{
				"cloudflare": {
					Type:  model.TypeSSE,
					URL:   "https://docs.mcp.cloudflare.com/sse",
					Tools: []string{"*"},
				},
			},
		},
		{
			name:    "unknown_type",
			input:   `{"mcpServers":{"s":{"type":"grpc"}}}`,
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cfg, _, err := copilot.Decode([]byte(tc.input))
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
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
				if len(got.Tools) != len(want.Tools) {
					t.Errorf("[%s] Tools length: got %d, want %d", name, len(got.Tools), len(want.Tools))
				}
			}
		})
	}
}

func TestEncode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		input       *model.Config
		checkOutput func(t *testing.T, data []byte)
	}{
		{
			name: "stdio_uses_local_type",
			input: &model.Config{Servers: map[string]model.Server{
				"srv": {Type: model.TypeStdio, Command: "npx", Args: []string{"-y", "pkg"}},
			}},
			checkOutput: func(t *testing.T, data []byte) {
				t.Helper()
				var raw map[string]interface{}
				if err := json.Unmarshal(data, &raw); err != nil {
					t.Fatalf("unmarshal: %v", err)
				}
				servers := raw["mcpServers"].(map[string]interface{})
				srv := servers["srv"].(map[string]interface{})
				if srv["type"] != "local" {
					t.Errorf("type: got %v, want local", srv["type"])
				}
			},
		},
		{
			name: "sse_roundtrip",
			input: &model.Config{Servers: map[string]model.Server{
				"sse": {Type: model.TypeSSE, URL: "https://example.com/sse", Tools: []string{"*"}},
			}},
			checkOutput: func(t *testing.T, data []byte) {
				t.Helper()
				cfg, _, err := copilot.Decode(data)
				if err != nil {
					t.Fatalf("roundtrip decode: %v", err)
				}
				s := cfg.Servers["sse"]
				if s.Type != model.TypeSSE {
					t.Errorf("Type: got %v, want sse", s.Type)
				}
				if len(s.Tools) != 1 || s.Tools[0] != "*" {
					t.Errorf("Tools: got %v, want [*]", s.Tools)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			data, _, err := copilot.Encode(tc.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.checkOutput != nil {
				tc.checkOutput(t, data)
			}
		})
	}
}
