# ma — MCP config adapter

`ma` reads a single shared MCP server config and launches AI coding tools with
the config automatically adapted to each tool's native format.

```
ma copilot
ma opencode
ma codex -- --some-flag
ma --version
```

## Motivation

Claude Code, GitHub Copilot CLI, OpenCode, and Codex each have their own MCP
configuration format. With `ma`, you maintain one config file in Claude Code
JSON format and every tool is launched with the same MCP server setup.

## Supported tools

| Subcommand | Tool | Target format | Injection method |
|---|---|---|---|
| `ma copilot` | GitHub Copilot CLI | Copilot JSON | `--additional-mcp-config <json-content>` |
| `ma opencode` | OpenCode | OpenCode JSON | `OPENCODE_CONFIG=<tmpfile>` |
| `ma codex` | Codex CLI | TOML | `-c mcp_servers.NAME={...}` (per server) |

## Transport support

| Transport | copilot | opencode | codex |
|---|---|---|---|
| stdio | ✅ | ✅ | ✅ |
| Streamable HTTP | ✅ | ✅ | ✅ |
| SSE | ✅ | ✅ (as `remote`) | ⚠️ skipped with warning |

## Installation

### Homebrew

```sh
brew install --cask daaa1k/tap/ma
```

### Nix (Home Manager)

Add to your `flake.nix`:

```nix
inputs.ma.url = "github:daaa1k/ma";
```

Home Manager configuration:

```nix
{ inputs, ... }: {
  imports = [ inputs.ma.homeManagerModules.default ];
  programs.ma.enable = true;
}
```

### go install

```sh
go install github.com/daaa1k/ma@latest
```

## Configuration

Write your MCP servers in Claude Code JSON format. `ma` searches for the config
file in the following order:

1. `--config` flag
2. `./.mcp.json` (current directory)
3. `~/.mcp.json` (home directory)

### Example

```json
{
  "mcpServers": {
    "context7": {
      "type": "stdio",
      "command": "npx",
      "args": ["-y", "@upstash/context7-mcp"]
    },
    "figma": {
      "type": "http",
      "url": "https://mcp.figma.com/mcp"
    },
    "cloudflare": {
      "type": "sse",
      "url": "https://docs.mcp.cloudflare.com/sse"
    }
  }
}
```

### Fields

**stdio servers**

| Field | Type | Description |
|---|---|---|
| `type` | `"stdio"` | Transport type |
| `command` | string | Executable to run |
| `args` | string[] | Command arguments |
| `env` | object | Environment variables |

**Streamable HTTP / SSE servers**

| Field | Type | Description |
|---|---|---|
| `type` | `"http"` or `"sse"` | Transport type |
| `url` | string | Endpoint URL |
| `headers` | object | HTTP headers (e.g. auth) |

Environment variable references use `${VAR_NAME}` syntax.

## Usage

### Basic

```sh
# Launch opencode using .mcp.json in the current directory
ma opencode

# Launch copilot and pass extra flags (everything after -- goes to the tool)
ma copilot -- --disable-builtin-mcps

# Specify a config file explicitly
ma --config ~/work/mcp.json codex
```

### Version

```sh
ma --version
```

### Help

```sh
ma --help
ma copilot --help
```

## Conversion caveats

When a field cannot be represented in the target format, `ma` prints a warning
to stderr and continues.

| Situation | Behavior |
|---|---|
| SSE server → Codex | Server skipped + warning |
| SSE server → OpenCode | Encoded as `"remote"` type + warning |
| Copilot `tools` field → OpenCode | Field dropped + warning |
| Copilot `tools` field → Codex / Claude Code | Field dropped silently |
| Codex `bearer_token_env_var` → other formats | Converted to `Authorization: Bearer ${VAR}` header + warning |

## Development

```sh
# Build
go build ./...

# Test (with race detector)
go test -race ./...

# Lint
golangci-lint run ./...
```

## License

MIT
