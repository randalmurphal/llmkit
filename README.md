# llmkit

[![Go Reference](https://pkg.go.dev/badge/github.com/randalmurphal/llmkit.svg)](https://pkg.go.dev/github.com/randalmurphal/llmkit)
[![Go Report Card](https://goreportcard.com/badge/github.com/randalmurphal/llmkit)](https://goreportcard.com/report/github.com/randalmurphal/llmkit)

Go library for LLM utilities. Standalone toolkit for token counting, prompt templates, response parsing, model selection, and Claude CLI integration.

## Installation

```bash
go get github.com/randalmurphal/llmkit
```

## Packages

| Package | Description |
|---------|-------------|
| [`claude`](./claude/) | Claude CLI wrapper with OAuth credential management |
| [`template`](./template/) | Prompt template rendering with `{{variable}}` syntax |
| [`tokens`](./tokens/) | Token counting and budget management |
| [`parser`](./parser/) | Extract JSON, YAML, and code blocks from LLM responses |
| [`truncate`](./truncate/) | Token-aware text truncation strategies |
| [`model`](./model/) | Model selection, cost tracking, and escalation chains |

## Quick Start

### Token Counting

```go
import "github.com/randalmurphal/llmkit/tokens"

counter := tokens.NewEstimatingCounter()
count := counter.Count("Hello, World!")  // ~4 tokens
```

### Template Rendering

```go
import "github.com/randalmurphal/llmkit/template"

engine := template.NewEngine()
result, err := engine.Render("Hello {{name}}!", map[string]any{"name": "World"})
// result: "Hello World!"
```

### Claude CLI

```go
import "github.com/randalmurphal/llmkit/claude"

client := claude.NewCLI()
resp, err := client.Complete(ctx, claude.CompletionRequest{
    Messages: []claude.Message{
        {Role: claude.RoleUser, Content: "What is 2+2?"},
    },
})
fmt.Println(resp.Content)
```

### Response Parsing

```go
import "github.com/randalmurphal/llmkit/parser"

response := `Here's the JSON: ` + "```json\n{\"key\": \"value\"}\n```"
data, err := parser.ExtractJSON(response)
// data: map[string]any{"key": "value"}
```

### Model Selection

```go
import "github.com/randalmurphal/llmkit/model"

selector := model.NewSelector(
    model.WithThinkingModel(model.ModelOpus),
    model.WithDefaultModel(model.ModelSonnet),
    model.WithFastModel(model.ModelHaiku),
)
m := selector.SelectForTier(model.TierThinking)  // ModelOpus
```

## Design Principles

- **Zero external dependencies** - Only Go stdlib
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
