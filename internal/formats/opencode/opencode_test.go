package opencode_test

import (
	"encoding/json"
	"testing"

	"github.com/daaa1k/ma/internal/formats/opencode"
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
			name: "local_stdio",
			input: `{
				"mcp": {
					"my-server": {
						"type": "local",
						"command": ["npx", "-y", "my-mcp"],
						"enabled": true,
						"environment": {"MY_VAR": "val"}
					}
				}
			}`,
			wantServers: map[string]model.Server{
				"my-server": {
					Type:    model.TypeStdio,
					Command: "npx",
					Args:    []string{"-y", "my-mcp"},
					Env:     map[string]string{"MY_VAR": "val"},
				},
			},
		},
		{
			name: "remote_http",
			input: `{
				"mcp": {
					"remote-srv": {
						"type": "remote",
						"url": "https://my-mcp-server.com",
						"headers": {"Authorization": "Bearer MY_API_KEY"}
					}
				}
			}`,
			wantServers: map[string]model.Server{
				"remote-srv": {
					Type: model.TypeHTTP,
					URL:  "https://my-mcp-server.com",
					Headers: map[string]string{
						"Authorization": "Bearer MY_API_KEY",
					},
				},
			},
		},
		{
			name: "disabled_server_warns",
			input: `{
				"mcp": {
					"off-srv": {
						"type": "local",
						"command": ["node", "server.js"],
						"enabled": false
					}
				}
			}`,
			wantServers: map[string]model.Server{
				"off-srv": {Type: model.TypeStdio, Command: "node", Args: []string{"server.js"}},
			},
			wantWarnings: 1,
		},
		{
			name:    "unknown_type",
			input:   `{"mcp":{"s":{"type":"magic"}}}`,
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cfg, warnings, err := opencode.Decode([]byte(tc.input))
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
				if got.Command != want.Command {
					t.Errorf("[%s] Command: got %q, want %q", name, got.Command, want.Command)
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
		checkOutput  func(t *testing.T, data []byte)
	}{
		{
			name: "stdio_command_array",
			input: &model.Config{Servers: map[string]model.Server{
				"srv": {Type: model.TypeStdio, Command: "npx", Args: []string{"-y", "pkg"}},
			}},
			checkOutput: func(t *testing.T, data []byte) {
				t.Helper()
				var raw map[string]interface{}
				if err := json.Unmarshal(data, &raw); err != nil {
					t.Fatalf("unmarshal: %v", err)
				}
				mcp := raw["mcp"].(map[string]interface{})
				srv := mcp["srv"].(map[string]interface{})
				cmd := srv["command"].([]interface{})
				if cmd[0] != "npx" {
					t.Errorf("command[0]: got %v, want npx", cmd[0])
				}
				if cmd[1] != "-y" {
					t.Errorf("command[1]: got %v, want -y", cmd[1])
				}
			},
		},
		{
			name: "sse_becomes_remote_with_warning",
			input: &model.Config{Servers: map[string]model.Server{
				"sse-srv": {Type: model.TypeSSE, URL: "https://example.com/sse"},
			}},
			wantWarnings: 1,
			checkOutput: func(t *testing.T, data []byte) {
				t.Helper()
				var raw map[string]interface{}
				if err := json.Unmarshal(data, &raw); err != nil {
					t.Fatalf("unmarshal: %v", err)
				}
				mcp := raw["mcp"].(map[string]interface{})
				srv := mcp["sse-srv"].(map[string]interface{})
				if srv["type"] != "remote" {
					t.Errorf("type: got %v, want remote", srv["type"])
				}
			},
		},
		{
			name: "tools_field_warned_and_dropped",
			input: &model.Config{Servers: map[string]model.Server{
				"h": {Type: model.TypeHTTP, URL: "https://x.com/mcp", Tools: []string{"tool1"}},
			}},
			wantWarnings: 1,
			checkOutput: func(t *testing.T, data []byte) {
				t.Helper()
				var raw map[string]interface{}
				if err := json.Unmarshal(data, &raw); err != nil {
					t.Fatalf("unmarshal: %v", err)
				}
				mcp := raw["mcp"].(map[string]interface{})
				srv := mcp["h"].(map[string]interface{})
				if _, ok := srv["tools"]; ok {
					t.Error("tools field should not appear in OpenCode output")
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			data, warnings, err := opencode.Encode(tc.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(warnings) != tc.wantWarnings {
				t.Errorf("warnings count: got %d, want %d: %v", len(warnings), tc.wantWarnings, warnings)
			}
			if tc.checkOutput != nil {
				tc.checkOutput(t, data)
			}
		})
	}
}
