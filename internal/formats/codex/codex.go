// Package codex implements encoding and decoding of OpenAI Codex MCP
// configuration files (TOML format).
//
// Supported transports: stdio, http (Streamable HTTP).
// SSE is NOT supported by Codex; servers of type SSE are skipped with a warning.
//
// Reference schema:
//
//	[mcp_servers.name]
//	command            = "..."
//	args               = [...]
//	url                = "..."             # http
//	bearer_token_env_var = "ENV_VAR_NAME"  # http auth (env var name, not value)
//	http_headers       = { "Key" = "Val" } # http extra headers
package codex

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/daaa1k/ma/internal/model"
)

// serverEntry is the TOML representation of one server under [mcp_servers.*].
type serverEntry struct {
	Command           string            `toml:"command,omitempty"`
	Args              []string          `toml:"args,omitempty"`
	Env               map[string]string `toml:"env,omitempty"`
	URL               string            `toml:"url,omitempty"`
	BearerTokenEnvVar string            `toml:"bearer_token_env_var,omitempty"`
	HTTPHeaders       map[string]string `toml:"http_headers,omitempty"`
}

// configFile is the top-level TOML structure.
type configFile struct {
	MCPServers map[string]serverEntry `toml:"mcp_servers"`
}

// Warning is a non-fatal issue encountered during conversion.
type Warning struct {
	Server  string
	Message string
}

func (w Warning) Error() string {
	return fmt.Sprintf("codex: server %q: %s", w.Server, w.Message)
}

// Decode parses Codex TOML config bytes into the canonical model.
// It returns warnings for any servers that could not be represented faithfully.
func Decode(data []byte) (*model.Config, []Warning, error) {
	var f configFile
	if _, err := toml.NewDecoder(bytes.NewReader(data)).Decode(&f); err != nil {
		return nil, nil, fmt.Errorf("codex: parse TOML: %w", err)
	}

	cfg := &model.Config{Servers: make(map[string]model.Server, len(f.MCPServers))}
	var warnings []Warning

	for name, e := range f.MCPServers {
		s, w := toServer(name, e)
		warnings = append(warnings, w...)
		if s != nil {
			cfg.Servers[name] = *s
		}
	}
	return cfg, warnings, nil
}

func toServer(name string, e serverEntry) (*model.Server, []Warning) {
	var warnings []Warning

	if e.URL != "" {
		// HTTP server
		headers := make(map[string]string, len(e.HTTPHeaders)+1)
		for k, v := range e.HTTPHeaders {
			headers[k] = v
		}
		if e.BearerTokenEnvVar != "" {
			// bearer_token_env_var holds the name of the env var, not its value.
			// Expand to a standard Authorization header using ${VAR} syntax so the
			// receiving tool can substitute it at runtime.
			headers["Authorization"] = fmt.Sprintf("Bearer ${%s}", e.BearerTokenEnvVar)
			warnings = append(warnings, Warning{
				Server:  name,
				Message: fmt.Sprintf("bearer_token_env_var %q converted to Authorization header with ${%s}", e.BearerTokenEnvVar, e.BearerTokenEnvVar),
			})
		}
		if len(headers) == 0 {
			headers = nil
		}
		return &model.Server{
			Type:    model.TypeHTTP,
			URL:     e.URL,
			Headers: headers,
		}, warnings
	}

	// stdio server (command present)
	return &model.Server{
		Type:    model.TypeStdio,
		Command: e.Command,
		Args:    e.Args,
		Env:     e.Env,
	}, warnings
}

// Encode serialises the canonical config into Codex TOML format.
// SSE servers are skipped with a warning because Codex does not support SSE.
func Encode(cfg *model.Config) ([]byte, []Warning, error) {
	f := configFile{MCPServers: make(map[string]serverEntry, len(cfg.Servers))}
	var warnings []Warning

	// Sort for deterministic output.
	names := make([]string, 0, len(cfg.Servers))
	for n := range cfg.Servers {
		names = append(names, n)
	}
	sort.Strings(names)

	for _, name := range names {
		s := cfg.Servers[name]
		e, w, skip := fromServer(name, s)
		warnings = append(warnings, w...)
		if !skip {
			f.MCPServers[name] = e
		}
	}

	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(f); err != nil {
		return nil, warnings, fmt.Errorf("codex: marshal TOML: %w", err)
	}
	return cleanTOML(buf.Bytes()), warnings, nil
}

func fromServer(name string, s model.Server) (serverEntry, []Warning, bool) {
	var warnings []Warning

	switch s.Type {
	case model.TypeStdio:
		return serverEntry{
			Command: s.Command,
			Args:    s.Args,
			Env:     s.Env,
		}, warnings, false

	case model.TypeHTTP:
		e := serverEntry{URL: s.URL}
		// Extract bearer token if present as a simple "${VAR}" Authorization value.
		headers := make(map[string]string, len(s.Headers))
		for k, v := range s.Headers {
			if strings.EqualFold(k, "Authorization") {
				if strings.HasPrefix(v, "Bearer ${") && strings.HasSuffix(v, "}") {
					varName := v[len("Bearer ${") : len(v)-1]
					e.BearerTokenEnvVar = varName
					continue
				}
			}
			headers[k] = v
		}
		if len(headers) > 0 {
			e.HTTPHeaders = headers
		}
		return e, warnings, false

	case model.TypeSSE:
		warnings = append(warnings, Warning{
			Server:  name,
			Message: "SSE transport is not supported by Codex; server skipped",
		})
		return serverEntry{}, warnings, true

	default:
		warnings = append(warnings, Warning{
			Server:  name,
			Message: fmt.Sprintf("unknown transport type %q; server skipped", s.Type),
		})
		return serverEntry{}, warnings, true
	}
}

// cleanTOML rewrites the BurntSushi encoder output so that each
// [mcp_servers.*] table appears on its own line for readability.
func cleanTOML(b []byte) []byte {
	lines := strings.Split(string(b), "\n")
	var out []byte
	for i, line := range lines {
		if i > 0 && strings.HasPrefix(line, "[mcp_servers.") {
			out = append(out, '\n')
		}
		out = append(out, []byte(line)...)
		if i < len(lines)-1 {
			out = append(out, '\n')
		}
	}
	return out
}

// BuildConfigOverrides converts encoded Codex TOML config data into a list of
// -c key=value override arguments suitable for passing to the codex CLI.
// Each MCP server becomes a single "-c", "mcp_servers.NAME=<inline-table>" pair.
func BuildConfigOverrides(data []byte) ([]string, error) {
	var f configFile
	if _, err := toml.NewDecoder(bytes.NewReader(data)).Decode(&f); err != nil {
		return nil, fmt.Errorf("codex: parse TOML: %w", err)
	}

	names := make([]string, 0, len(f.MCPServers))
	for n := range f.MCPServers {
		names = append(names, n)
	}
	sort.Strings(names)

	var args []string
	for _, name := range names {
		entry := f.MCPServers[name]
		args = append(args, "-c", fmt.Sprintf("mcp_servers.%s=%s", name, serverToInlineTOML(entry)))
	}
	return args, nil
}

// serverToInlineTOML serialises a serverEntry as a TOML inline table string.
func serverToInlineTOML(e serverEntry) string {
	var parts []string
	if e.Command != "" {
		parts = append(parts, "command = "+tomlQuote(e.Command))
	}
	if len(e.Args) > 0 {
		parts = append(parts, "args = "+tomlStringArray(e.Args))
	}
	if len(e.Env) > 0 {
		parts = append(parts, "env = "+tomlStringMap(e.Env))
	}
	if e.URL != "" {
		parts = append(parts, "url = "+tomlQuote(e.URL))
	}
	if e.BearerTokenEnvVar != "" {
		parts = append(parts, "bearer_token_env_var = "+tomlQuote(e.BearerTokenEnvVar))
	}
	if len(e.HTTPHeaders) > 0 {
		parts = append(parts, "http_headers = "+tomlStringMap(e.HTTPHeaders))
	}
	return "{" + strings.Join(parts, ", ") + "}"
}

func tomlQuote(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return `"` + s + `"`
}

func tomlStringArray(ss []string) string {
	quoted := make([]string, len(ss))
	for i, s := range ss {
		quoted[i] = tomlQuote(s)
	}
	return "[" + strings.Join(quoted, ", ") + "]"
}

func tomlStringMap(m map[string]string) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, tomlQuote(k)+" = "+tomlQuote(m[k]))
	}
	return "{" + strings.Join(parts, ", ") + "}"
}

