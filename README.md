# ma — MCP config adapter

`ma` は共通の MCP サーバー設定ファイルを読み込み、各 AI コーディングツールのネイティブ形式に変換してツールを起動するランチャーです。

```
ma copilot
ma opencode
ma codex -- --some-flag
ma --version
```

## 動機

Claude Code・GitHub Copilot CLI・OpenCode・Codex はそれぞれ独自の MCP 設定形式を持ちます。
`ma` は Claude Code JSON 形式の設定ファイル 1 つを管理するだけで、すべてのツールを同じ MCP サーバー構成で利用できるようにします。

## 対応ツールと設定注入方法

| サブコマンド | ツール | 変換形式 | 注入方法 |
|---|---|---|---|
| `ma copilot` | GitHub Copilot CLI | Copilot JSON | `--additional-mcp-config <tmpfile>` |
| `ma opencode` | OpenCode | OpenCode JSON | `OPENCODE_CONFIG=<tmpfile>` |
| `ma codex` | Codex CLI | TOML | `--config <tmpfile.toml>` |

## 対応トランスポート

| トランスポート | copilot | opencode | codex |
|---|---|---|---|
| stdio | ✅ | ✅ | ✅ |
| Streamable HTTP | ✅ | ✅ | ✅ |
| SSE | ✅ | ✅ (remote として) | ⚠️ スキップ＋警告 |

## インストール

### Homebrew

```sh
brew install daaa1k/tap/ma
```

### Nix (Home Manager)

`flake.nix` に追加：

```nix
inputs.ma.url = "github:daaa1k/ma";
```

Home Manager 設定：

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

## 設定ファイル

Claude Code JSON 形式で記述します。以下の順で検索されます：

1. `--config` フラグ
2. `./.mcp.json`（カレントディレクトリ）
3. `~/.mcp.json`（ホームディレクトリ）

### 記述例

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
      "url": "https://mcp.figma.com/mcp",
      "headers": {
        "Authorization": "Bearer ${FIGMA_TOKEN}"
      }
    },
    "cloudflare": {
      "type": "sse",
      "url": "https://docs.mcp.cloudflare.com/sse"
    }
  }
}
```

### 対応フィールド

**stdio サーバー**

| フィールド | 型 | 説明 |
|---|---|---|
| `type` | `"stdio"` | トランスポート種別 |
| `command` | string | 実行コマンド |
| `args` | string[] | コマンド引数 |
| `env` | object | 環境変数 |

**Streamable HTTP / SSE サーバー**

| フィールド | 型 | 説明 |
|---|---|---|
| `type` | `"http"` または `"sse"` | トランスポート種別 |
| `url` | string | エンドポイント URL |
| `headers` | object | HTTP ヘッダー（認証など） |

環境変数の参照は `${VAR_NAME}` 構文で記述します。

## 使い方

### 基本

```sh
# .mcp.json を読み込んで opencode を起動
ma opencode

# copilot を起動（-- 以降はツールへの追加引数）
ma copilot -- --disable-builtin-mcps

# 設定ファイルを明示指定
ma --config ~/work/mcp.json codex
```

### バージョン確認

```sh
ma --version
```

### ヘルプ

```sh
ma --help
ma copilot --help
```

## 変換の注意点

変換時に情報が失われる場合、`ma` は stderr に警告を出力します。

| 状況 | 挙動 |
|---|---|
| SSE サーバー → Codex | そのサーバーをスキップ＋警告 |
| SSE サーバー → OpenCode | `"remote"` 型に変換＋警告 |
| Copilot `tools` フィールド → 他形式 | フィールドを破棄＋警告 |
| Codex `bearer_token_env_var` → 他形式 | `Authorization: Bearer ${VAR}` ヘッダーに変換＋情報 |

## 開発

```sh
# ビルド
go build ./...

# テスト（race detector 付き）
go test -race ./...

# Lint
golangci-lint run ./...
```

## ライセンス

MIT
