# llmkit

[![Go Reference](https://pkg.go.dev/badge/github.com/randalmurphal/llmkit/v2.svg)](https://pkg.go.dev/github.com/randalmurphal/llmkit/v2)
[![Go Report Card](https://goreportcard.com/badge/github.com/randalmurphal/llmkit/v2)](https://goreportcard.com/report/github.com/randalmurphal/llmkit/v2)

Go library for LLM utilities and CLI integrations. V2 exposes a unified root API for Claude and Codex, provider-native config packages for `claudeconfig` and `codexconfig`, and opt-in project lifecycle helpers in `env` and `worktree`.

## Installation

```bash
go get github.com/randalmurphal/llmkit/v2
```

## Packages

| Package | Description |
|---------|-------------|
| [`claude`](./claude/) | Claude CLI wrapper with OAuth credential management |
| [`codex`](./codex/) | OpenAI Codex CLI wrapper with headless `exec --json` support |
| [`claudeconfig`](./claudeconfig/) | Claude local config and ecosystem file parsing |
| [`codexconfig`](./codexconfig/) | Codex local config, hooks, skills, plugins, and custom-agent parsing |
| [`env`](./env/) | Scoped hook, MCP, env var, and tempfile lifecycle helpers |
| [`worktree`](./worktree/) | Git worktree creation, pruning, and safety hooks |
| [`providers`](./providers/) | Convenience blank imports for Claude and Codex registry registration |
| [`template`](./template/) | Prompt template rendering with `{{variable}}` syntax |
| [`tokens`](./tokens/) | Token counting and budget management |
| [`parser`](./parser/) | Extract JSON, YAML, and code blocks from LLM responses |
| [`truncate`](./truncate/) | Token-aware text truncation strategies |

## Release Scope

- Module path: `github.com/randalmurphal/llmkit/v2`
- Supported runtime providers in V2: Claude and Codex
- Supported config ecosystems in V2: `claudeconfig` and `codexconfig`
- Removed from V2: `aider`, `continue`, `gemini`, `local`, `opencode`, and the old root `provider` / `model` packages

## Quick Start

### Token Counting

```go
import "github.com/randalmurphal/llmkit/v2/tokens"

counter := tokens.NewEstimatingCounter()
count := counter.Count("Hello, World!")  // ~4 tokens
```

### Template Rendering

```go
import "github.com/randalmurphal/llmkit/v2/template"

engine := template.NewEngine()
result, err := engine.Render("Hello {{name}}!", map[string]any{"name": "World"})
// result: "Hello World!"
```

### Unified Client

```go
import (
    "github.com/randalmurphal/llmkit/v2"
    _ "github.com/randalmurphal/llmkit/v2/providers"
)

client, err := llmkit.New("codex", llmkit.Config{
    Provider: "codex",
    Model:    "gpt-5-codex",
})
resp, err := client.Complete(ctx, llmkit.Request{
    Messages: []llmkit.Message{
        {Role: llmkit.RoleUser, Content: "What is 2+2?"},
    },
})
fmt.Println(resp.Content)
```

### Typed Structured Output

```go
import (
    "github.com/randalmurphal/llmkit/v2"
    _ "github.com/randalmurphal/llmkit/v2/providers"
)

type ReviewResult struct {
    Approved bool   `json:"approved"`
    Summary  string `json:"summary"`
}

typed, err := llmkit.CompleteTyped[ReviewResult](ctx, client, llmkit.Request{
    Messages: []llmkit.Message{
        {Role: llmkit.RoleUser, Content: "Review this patch."},
    },
})
_ = typed.Value
```

### Response Parsing

```go
import "github.com/randalmurphal/llmkit/v2/parser"

response := `Here's the JSON: ` + "```json\n{\"key\": \"value\"}\n```"
data, err := parser.ExtractJSON(response)
// data: map[string]any{"key": "value"}
```

### Model Selection

```go
import "github.com/randalmurphal/llmkit/v2"

selector := llmkit.NewSelector(
    llmkit.WithThinkingModel(llmkit.ModelOpus),
    llmkit.WithDefaultModel(llmkit.ModelSonnet),
    llmkit.WithFastModel(llmkit.ModelHaiku),
)
m := selector.SelectForTier(llmkit.TierThinking)  // ModelOpus
```

## Design Principles

- **À la carte imports** - Use only what you need
- **Stable API** - Semver-friendly, rarely changes
- **No forced patterns** - Interfaces for flexibility

## Development

```bash
# Run tests
make test

# Run tests with coverage
make coverage

# Lint
make lint

# All checks
make verify
```

## License

MIT License - see [LICENSE](./LICENSE) for details.
