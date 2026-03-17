package claude_test

import (
	"encoding/json"
	"testing"

	"github.com/daaa1k/ma/internal/formats/claude"
	"github.com/daaa1k/ma/internal/model"
)

func TestDecode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    map[string]model.Server
		wantErr bool
	}{
		{
			name: "stdio",
			input: `{
				"mcpServers": {
					"my-server": {
						"type": "stdio",
						"command": "claude",
						"args": ["mcp", "serve"],
						"env": {"FOO": "bar"}
					}
				}
			}`,
			want: map[string]model.Server{
				"my-server": {
					Type:    model.TypeStdio,
					Command: "claude",
					Args:    []string{"mcp", "serve"},
					Env:     map[string]string{"FOO": "bar"},
				},
			},
		},
		{
			name: "http",
			input: `{
				"mcpServers": {
					"api-server": {
						"type": "http",
						"url": "https://api.example.com/mcp",
						"headers": {"Authorization": "Bearer ${API_KEY}"}
					}
				}
			}`,
			want: map[string]model.Server{
				"api-server": {
					Type: model.TypeHTTP,
					URL:  "https://api.example.com/mcp",
					Headers: map[string]string{
						"Authorization": "Bearer ${API_KEY}",
					},
				},
			},
		},
		{
			name: "sse",
			input: `{
				"mcpServers": {
					"sse-server": {
						"type": "sse",
						"url": "https://api.example.com/sse"
					}
				}
			}`,
			want: map[string]model.Server{
				"sse-server": {
					Type: model.TypeSSE,
					URL:  "https://api.example.com/sse",
				},
			},
		},
		{
			name:    "unknown_type",
			input:   `{"mcpServers":{"s":{"type":"grpc"}}}`,
			wantErr: true,
		},
		{
			name:    "invalid_json",
			input:   `{bad json}`,
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cfg, err := claude.Decode([]byte(tc.input))
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(cfg.Servers) != len(tc.want) {
				t.Fatalf("server count: got %d, want %d", len(cfg.Servers), len(tc.want))
			}
			for name, want := range tc.want {
				got, ok := cfg.Servers[name]
				if !ok {
					t.Fatalf("server %q not found", name)
				}
				assertServerEqual(t, got, want)
			}
		})
	}
}

func TestEncode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   *model.Config
		wantErr bool
		check   func(t *testing.T, data []byte)
	}{
		{
			name: "roundtrip_stdio",
			input: &model.Config{Servers: map[string]model.Server{
				"s": {Type: model.TypeStdio, Command: "npx", Args: []string{"-y", "pkg"}},
			}},
			check: func(t *testing.T, data []byte) {
				t.Helper()
				cfg, err := claude.Decode(data)
				if err != nil {
					t.Fatalf("roundtrip decode: %v", err)
				}
				s := cfg.Servers["s"]
				if s.Command != "npx" {
					t.Errorf("command: got %q, want %q", s.Command, "npx")
				}
			},
		},
		{
			name: "roundtrip_http",
			input: &model.Config{Servers: map[string]model.Server{
				"h": {Type: model.TypeHTTP, URL: "https://example.com/mcp", Headers: map[string]string{"X-Key": "val"}},
			}},
			check: func(t *testing.T, data []byte) {
				t.Helper()
				var raw map[string]interface{}
				if err := json.Unmarshal(data, &raw); err != nil {
					t.Fatalf("unmarshal: %v", err)
				}
				servers := raw["mcpServers"].(map[string]interface{})
				s := servers["h"].(map[string]interface{})
				if s["type"] != "http" {
					t.Errorf("type: got %v, want http", s["type"])
				}
			},
		},
		{
			name: "roundtrip_sse",
			input: &model.Config{Servers: map[string]model.Server{
				"ss": {Type: model.TypeSSE, URL: "https://example.com/sse"},
			}},
			check: func(t *testing.T, data []byte) {
				t.Helper()
				cfg, err := claude.Decode(data)
				if err != nil {
					t.Fatalf("roundtrip decode: %v", err)
				}
				if cfg.Servers["ss"].Type != model.TypeSSE {
					t.Errorf("type: got %v, want sse", cfg.Servers["ss"].Type)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			data, err := claude.Encode(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.check != nil {
				tc.check(t, data)
			}
		})
	}
}

func assertServerEqual(t *testing.T, got, want model.Server) {
	t.Helper()
	if got.Type != want.Type {
		t.Errorf("Type: got %q, want %q", got.Type, want.Type)
	}
	if got.Command != want.Command {
		t.Errorf("Command: got %q, want %q", got.Command, want.Command)
	}
	if got.URL != want.URL {
		t.Errorf("URL: got %q, want %q", got.URL, want.URL)
	}
	for k, v := range want.Env {
		if got.Env[k] != v {
			t.Errorf("Env[%q]: got %q, want %q", k, got.Env[k], v)
		}
	}
	for k, v := range want.Headers {
		if got.Headers[k] != v {
			t.Errorf("Headers[%q]: got %q, want %q", k, got.Headers[k], v)
		}
	}
}
