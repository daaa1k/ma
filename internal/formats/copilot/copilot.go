// Package copilot implements encoding and decoding of GitHub Copilot CLI MCP
// configuration files (JSON with "mcpServers" wrapper).
//
// Supported transports: stdio (type "local"), http, sse.
//
// Reference schema:
//
//	{
//	  "mcpServers": {
//	    "name": {
//	      "type":    "local|http|sse",
//	      "command": "...",      // local
//	      "args":    [...],      // local
//	      "env":     {...},      // local
//	      "url":     "...",      // http|sse
//	      "headers": {...},      // http|sse
//	      "tools":   ["*"]       // optional allowlist
//	    }
//	  }
//	}
package copilot

import (
	"encoding/json"
	"fmt"

	"github.com/daaa1k/ma/internal/model"
)

// serverEntry is the JSON representation of one server in a Copilot CLI config.
type serverEntry struct {
	Type    string            `json:"type"`
	Command string            `json:"command,omitempty"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
	URL     string            `json:"url,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
	Tools   []string          `json:"tools,omitempty"`
}

// configFile is the top-level JSON structure of a Copilot CLI config.
type configFile struct {
	MCPServers map[string]serverEntry `json:"mcpServers"`
}

// Warning is a non-fatal issue encountered during conversion.
type Warning struct {
	Server  string
	Message string
}

func (w Warning) Error() string {
	return fmt.Sprintf("copilot: server %q: %s", w.Server, w.Message)
}

// Decode parses Copilot CLI JSON config bytes into the canonical model.
func Decode(data []byte) (*model.Config, []Warning, error) {
	var f configFile
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, nil, fmt.Errorf("copilot: parse JSON: %w", err)
	}

	cfg := &model.Config{Servers: make(map[string]model.Server, len(f.MCPServers))}
	var warnings []Warning

	for name, e := range f.MCPServers {
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

	switch e.Type {
	case "local":
		return model.Server{
			Type:    model.TypeStdio,
			Command: e.Command,
			Args:    e.Args,
			Env:     e.Env,
			Tools:   e.Tools,
		}, warnings, nil

	case "http":
		return model.Server{
			Type:    model.TypeHTTP,
			URL:     e.URL,
			Headers: e.Headers,
			Tools:   e.Tools,
		}, warnings, nil

	case "sse":
		return model.Server{
			Type:    model.TypeSSE,
			URL:     e.URL,
			Headers: e.Headers,
			Tools:   e.Tools,
		}, warnings, nil

	default:
		return model.Server{}, warnings, fmt.Errorf("copilot: server %q: unknown type %q", name, e.Type)
	}
}

// Encode serialises the canonical config into Copilot CLI JSON format.
// The "tools" field is preserved if present in the canonical model.
// When converting from formats that do not support "tools", the field is omitted.
func Encode(cfg *model.Config) ([]byte, []Warning, error) {
	f := configFile{MCPServers: make(map[string]serverEntry, len(cfg.Servers))}
	var warnings []Warning

	for name, s := range cfg.Servers {
		e, w, err := fromServer(name, s)
		if err != nil {
			return nil, warnings, err
		}
		warnings = append(warnings, w...)
		f.MCPServers[name] = e
	}

	out, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return nil, warnings, fmt.Errorf("copilot: marshal JSON: %w", err)
	}
	return out, warnings, nil
}

func fromServer(name string, s model.Server) (serverEntry, []Warning, error) {
	var warnings []Warning

	switch s.Type {
	case model.TypeStdio:
		return serverEntry{
			Type:    "local",
			Command: s.Command,
			Args:    s.Args,
			Env:     s.Env,
			Tools:   s.Tools,
		}, warnings, nil

	case model.TypeHTTP:
		return serverEntry{
			Type:    "http",
			URL:     s.URL,
			Headers: s.Headers,
			Tools:   s.Tools,
		}, warnings, nil

	case model.TypeSSE:
		return serverEntry{
			Type:    "sse",
			URL:     s.URL,
			Headers: s.Headers,
			Tools:   s.Tools,
		}, warnings, nil

	default:
		return serverEntry{}, warnings, fmt.Errorf("copilot: server %q: unsupported type %q", name, s.Type)
	}
}

