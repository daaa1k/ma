package convert_test

import (
	"strings"
	"testing"

	"github.com/daaa1k/ma/internal/convert"
)

// claudeStdioJSON is a minimal Claude Code config with stdio, http, and sse servers.
const claudeAllTypesJSON = `{
  "mcpServers": {
    "stdio-srv": {
      "type": "stdio",
      "command": "npx",
      "args": ["-y", "my-mcp"],
      "env": {"TOKEN": "abc"}
    },
    "http-srv": {
      "type": "http",
      "url": "https://api.example.com/mcp",
      "headers": {"Authorization": "Bearer ${API_KEY}"}
    },
    "sse-srv": {
      "type": "sse",
      "url": "https://api.example.com/sse"
    }
  }
}`

// copilotAllTypesJSON has stdio (local), http, and sse with tools.
const copilotAllTypesJSON = `{
  "mcpServers": {
    "stdio-srv": {
      "type": "local",
      "command": "npx",
      "args": ["-y", "my-mcp"],
      "tools": ["tool_a", "tool_b"]
    },
    "http-srv": {
      "type": "http",
      "url": "https://api.example.com/mcp",
      "headers": {"Authorization": "Bearer ${API_KEY}"},
      "tools": ["*"]
    },
    "sse-srv": {
      "type": "sse",
      "url": "https://example.com/sse",
      "tools": ["*"]
    }
  }
}`

func TestConvert(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		src          convert.Format
		dst          convert.Format
		input        string
		wantWarnings int
		// substrings that must appear in output
		wantContains []string
		// substrings that must NOT appear in output
		wantAbsent []string
	}{
		{
			name:         "claude_to_copilot_stdio",
			src:          convert.FormatClaude,
			dst:          convert.FormatCopilot,
			input:        claudeAllTypesJSON,
			wantContains: []string{`"local"`, `"http"`, `"sse"`, `"npx"`},
		},
		{
			name:  "claude_to_codex_sse_skipped",
			src:   convert.FormatClaude,
			dst:   convert.FormatCodex,
			input: claudeAllTypesJSON,
			// SSE server skipped → 1 warning
			wantWarnings: 1,
			wantContains: []string{`command = "npx"`, `url = "https://api.example.com/mcp"`},
			wantAbsent:   []string{"sse-srv"},
		},
		{
			name:         "claude_to_opencode_all_types",
			src:          convert.FormatClaude,
			dst:          convert.FormatOpenCode,
			input:        claudeAllTypesJSON,
			wantWarnings: 1, // SSE → remote with warning
			wantContains: []string{`"local"`, `"remote"`, `"npx"`},
		},
		{
			name:  "copilot_to_claude_tools_dropped",
			src:   convert.FormatCopilot,
			dst:   convert.FormatClaude,
			input: copilotAllTypesJSON,
			// tools field is preserved on the canonical model but claude encoder drops it silently (no warning)
			wantContains: []string{`"stdio"`, `"http"`, `"sse"`},
		},
		{
			name:  "copilot_to_codex_sse_skipped",
			src:   convert.FormatCopilot,
			dst:   convert.FormatCodex,
			input: copilotAllTypesJSON,
			// sse-srv skipped
			wantWarnings: 1,
			wantAbsent:   []string{"sse-srv"},
		},
		{
			name:         "codex_to_claude_bearer_converted",
			src:          convert.FormatCodex,
			dst:          convert.FormatClaude,
			input: `[mcp_servers.figma]
url = "https://mcp.figma.com/mcp"
bearer_token_env_var = "FIGMA_TOKEN"
`,
			wantWarnings: 1,
			wantContains: []string{`"Bearer ${FIGMA_TOKEN}"`},
		},
		{
			name:  "opencode_to_copilot",
			src:   convert.FormatOpenCode,
			dst:   convert.FormatCopilot,
			input: `{"mcp":{"srv":{"type":"local","command":["npx","-y","pkg"],"enabled":true}}}`,
			wantContains: []string{`"local"`, `"npx"`},
		},
		{
			name:    "unknown_src_format",
			src:     "unknown",
			dst:     convert.FormatClaude,
			input:   `{}`,
			wantWarnings: -1, // signal "expect error"
		},
		{
			name:    "unknown_dst_format",
			src:     convert.FormatClaude,
			dst:     "unknown",
			input:   `{"mcpServers":{}}`,
			wantWarnings: -1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := convert.Convert(tc.src, tc.dst, []byte(tc.input))

			// wantWarnings == -1 signals we expect an error
			if tc.wantWarnings == -1 {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(result.Warnings) != tc.wantWarnings {
				t.Errorf("warnings count: got %d, want %d:\n%v",
					len(result.Warnings), tc.wantWarnings, result.Warnings)
			}

			out := string(result.Data)
			for _, want := range tc.wantContains {
				// Use Contains with unquoted form as well.
				if !strings.Contains(out, want) {
					t.Errorf("output missing %q:\n%s", want, out)
				}
			}
			for _, absent := range tc.wantAbsent {
				if strings.Contains(out, absent) {
					t.Errorf("output should not contain %q:\n%s", absent, out)
				}
			}
		})
	}
}
