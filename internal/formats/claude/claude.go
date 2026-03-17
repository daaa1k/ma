// Package claude implements encoding and decoding of Claude Code MCP
// configuration files (JSON with "mcpServers" wrapper).
//
// Supported transports: stdio, http (Streamable HTTP), sse.
//
// Reference schema:
//
//	{
//	  "mcpServers": {
//	    "name": {
//	      "type": "stdio|http|sse",
//	      "command": "...",       // stdio
//	      "args":    [...],       // stdio
//	      "env":     {...},       // stdio
//	      "url":     "...",       // http|sse
//	      "headers": {...}        // http|sse
//	    }
//	  }
//	}
package claude

import (
	"encoding/json"
	"fmt"

	"github.com/daaa1k/ma/internal/model"
)

// serverEntry is the JSON representation of one server in a Claude config file.
type serverEntry struct {
	Type    string            `json:"type"`
	Command string            `json:"command,omitempty"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
	URL     string            `json:"url,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
}

// configFile is the top-level JSON structure of a Claude Code config file.
type configFile struct {
	MCPServers map[string]serverEntry `json:"mcpServers"`
}

// Decode parses Claude Code JSON config bytes into the canonical model.
func Decode(data []byte) (*model.Config, error) {
	var f configFile
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("claude: parse JSON: %w", err)
	}

	cfg := &model.Config{Servers: make(map[string]model.Server, len(f.MCPServers))}
	for name, e := range f.MCPServers {
		s, err := toServer(name, e)
		if err != nil {
			return nil, err
		}
		cfg.Servers[name] = s
	}
	return cfg, nil
}

func toServer(name string, e serverEntry) (model.Server, error) {
	switch e.Type {
	case "stdio":
		return model.Server{
			Type:    model.TypeStdio,
			Command: e.Command,
			Args:    e.Args,
			Env:     e.Env,
		}, nil
	case "http":
		return model.Server{
			Type:    model.TypeHTTP,
			URL:     e.URL,
			Headers: e.Headers,
		}, nil
	case "sse":
		return model.Server{
			Type:    model.TypeSSE,
			URL:     e.URL,
			Headers: e.Headers,
		}, nil
	default:
		return model.Server{}, fmt.Errorf("claude: server %q: unknown type %q", name, e.Type)
	}
}

// Encode serialises the canonical config into Claude Code JSON format.
func Encode(cfg *model.Config) ([]byte, error) {
	f := configFile{MCPServers: make(map[string]serverEntry, len(cfg.Servers))}
	for name, s := range cfg.Servers {
		e, err := fromServer(name, s)
		if err != nil {
			return nil, err
		}
		f.MCPServers[name] = e
	}
	out, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("claude: marshal JSON: %w", err)
	}
	return out, nil
}

func fromServer(name string, s model.Server) (serverEntry, error) {
	switch s.Type {
	case model.TypeStdio:
		return serverEntry{
			Type:    "stdio",
			Command: s.Command,
			Args:    s.Args,
			Env:     s.Env,
		}, nil
	case model.TypeHTTP:
		return serverEntry{
			Type:    "http",
			URL:     s.URL,
			Headers: s.Headers,
		}, nil
	case model.TypeSSE:
		return serverEntry{
			Type:    "sse",
			URL:     s.URL,
			Headers: s.Headers,
		}, nil
	default:
		return serverEntry{}, fmt.Errorf("claude: server %q: unsupported type %q", name, s.Type)
	}
}
