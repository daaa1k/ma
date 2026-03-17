// Package opencode implements encoding and decoding of OpenCode MCP
// configuration files (JSON format).
//
// Supported transports:
//   - stdio  → type "local",  command is an array (cmd + args combined)
//   - http/sse → type "remote", no distinction between the two transports
//
// Reference schema:
//
//	{
//	  "mcp": {
//	    "name": {
//	      "type": "local|remote",
//	      "command":     ["npx", "-y", "pkg"],  // local only
//	      "environment": {"VAR": "val"},         // local only
//	      "url":         "https://...",          // remote only
//	      "headers":     {"Key": "Val"},         // remote only
//	      "enabled":     true
//	    }
//	  }
//	}
package opencode

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/daaa1k/ma/internal/model"
)

// serverEntry is the JSON representation of one server in an OpenCode config.
type serverEntry struct {
	Type        string            `json:"type"`
	Command     []string          `json:"command,omitempty"`
	Environment map[string]string `json:"environment,omitempty"`
	URL         string            `json:"url,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	Enabled     *bool             `json:"enabled,omitempty"`
}

// configFile is the top-level JSON structure of an OpenCode config.
type configFile struct {
	Schema string                 `json:"$schema,omitempty"`
	MCP    map[string]serverEntry `json:"mcp"`
}

// Warning is a non-fatal issue encountered during conversion.
type Warning struct {
	Server  string
	Message string
}

func (w Warning) Error() string {
	return fmt.Sprintf("opencode: server %q: %s", w.Server, w.Message)
}

// Decode parses OpenCode JSON config bytes into the canonical model.
func Decode(data []byte) (*model.Config, []Warning, error) {
	var f configFile
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, nil, fmt.Errorf("opencode: parse JSON: %w", err)
	}

	cfg := &model.Config{Servers: make(map[string]model.Server, len(f.MCP))}
	var warnings []Warning

	for name, e := range f.MCP {
		s, w, err := toServer(name, e)
		if err != nil {
			return nil, warnings, err
		}
		warnings = append(warnings, w...)
		cfg.Servers[name] = s
	}
	return cfg, warnings, nil
}

func toServer(name string, e serverEntry) (model.Server, []Warning, error) {
	var warnings []Warning

	// Warn if the server is disabled – other formats have no equivalent.
	if e.Enabled != nil && !*e.Enabled {
		warnings = append(warnings, Warning{
			Server:  name,
			Message: `"enabled: false" has no equivalent in target format; server will be included as active`,
		})
	}

	switch e.Type {
	case "local":
		var command string
		var args []string
		if len(e.Command) > 0 {
			command = e.Command[0]
			args = e.Command[1:]
		}
		return model.Server{
			Type:    model.TypeStdio,
			Command: command,
			Args:    args,
			Env:     e.Environment,
		}, warnings, nil

	case "remote":
		return model.Server{
			Type:    model.TypeHTTP, // OpenCode uses "remote" for both HTTP and SSE.
			URL:     e.URL,
			Headers: e.Headers,
		}, warnings, nil

	default:
		return model.Server{}, warnings, fmt.Errorf("opencode: server %q: unknown type %q", name, e.Type)
	}
}

// Encode serialises the canonical config into OpenCode JSON format.
// Both HTTP and SSE are encoded as type "remote" (OpenCode makes no distinction).
func Encode(cfg *model.Config) ([]byte, []Warning, error) {
	f := configFile{
		Schema: "https://opencode.ai/config.json",
		MCP:    make(map[string]serverEntry, len(cfg.Servers)),
	}
	var warnings []Warning

	for name, s := range cfg.Servers {
		e, w := fromServer(name, s)
		warnings = append(warnings, w...)
		f.MCP[name] = e
	}

	out, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return nil, warnings, fmt.Errorf("opencode: marshal JSON: %w", err)
	}
	return out, warnings, nil
}

func fromServer(name string, s model.Server) (serverEntry, []Warning) {
	var warnings []Warning
	enabled := true

	switch s.Type {
	case model.TypeStdio:
		cmd := make([]string, 0, 1+len(s.Args))
		if s.Command != "" {
			cmd = append(cmd, s.Command)
		}
		cmd = append(cmd, s.Args...)
		return serverEntry{
			Type:        "local",
			Command:     cmd,
			Environment: s.Env,
			Enabled:     &enabled,
		}, warnings

	case model.TypeHTTP, model.TypeSSE:
		if s.Type == model.TypeSSE {
			warnings = append(warnings, Warning{
				Server:  name,
				Message: "SSE transport encoded as \"remote\" type (OpenCode does not distinguish HTTP from SSE)",
			})
		}
		if len(s.Tools) > 0 {
			warnings = append(warnings, Warning{
				Server:  name,
				Message: "\"tools\" field is not supported by OpenCode and will be dropped",
			})
		}
		return serverEntry{
			Type:    "remote",
			URL:     s.URL,
			Headers: s.Headers,
			Enabled: &enabled,
		}, warnings

	default:
		warnings = append(warnings, Warning{
			Server:  name,
			Message: fmt.Sprintf("unsupported transport type %q; server skipped", s.Type),
		})
		return serverEntry{}, warnings
	}
}

// WriteWarnings writes warnings to w in a human-readable format.
func WriteWarnings(w io.Writer, warnings []Warning) {
	for _, warn := range warnings {
		_, _ = fmt.Fprintf(w, "warning: %s\n", warn.Error())
	}
}
