// Package model defines the canonical internal representation of MCP server
// configurations used as the interchange format between all supported tools.
package model

// ServerType represents the transport type for an MCP server.
type ServerType string

const (
	// TypeStdio is a local process communicating over stdin/stdout.
	TypeStdio ServerType = "stdio"
	// TypeHTTP is a remote server using Streamable HTTP transport.
	TypeHTTP ServerType = "http"
	// TypeSSE is a remote server using Server-Sent Events transport (legacy HTTP).
	TypeSSE ServerType = "sse"
)

// Server is the canonical representation of a single MCP server entry.
// All format-specific fields are normalized into this structure.
type Server struct {
	// Type indicates the transport mechanism.
	Type ServerType

	// Stdio fields (TypeStdio only).
	Command string
	Args    []string
	Env     map[string]string
	CWD     string

	// Remote fields (TypeHTTP and TypeSSE).
	URL     string
	Headers map[string]string

	// Tools is an optional allowlist of tool names.
	// Only Copilot CLI supports this natively; other formats drop it with a warning.
	Tools []string
}

// Config is the canonical representation of a complete MCP configuration,
// mapping server names to their definitions.
type Config struct {
	Servers map[string]Server
}
