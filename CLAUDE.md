# llmkit

**Go library for LLM utilities.** Standalone toolkit for token counting, prompt templates, response parsing, model selection, and Claude CLI integration.

**Status**: Structure ready, pending code extraction from flowgraph.

---

## Package Overview

| Package | Purpose | Key Types |
|---------|---------|-----------|
| `claude/` | Claude CLI wrapper | `Client`, `ClaudeCLI`, `Credentials`, `CompletionRequest` |
| `template/` | Prompt template rendering | `Engine`, `Template`, `Render` |
| `tokens/` | Token counting and budgeting | `Counter`, `Budget`, `Estimate` |
| `parser/` | Extract structured data from LLM responses | `ExtractJSON`, `ExtractYAML`, `ExtractCodeBlocks` |
| `truncate/` | Token-aware text truncation | `Truncator`, `Strategy`, `FromEnd`, `FromMiddle` |
| `model/` | Model selection and cost tracking | `Selector`, `CostTracker`, `EscalationChain` |

---

## Quick Reference

### Claude CLI

```go
import "github.com/randalmurphal/llmkit/claude"

// Create client with options
client := claude.NewCLI(
    claude.WithModel("sonnet"),
    claude.WithTimeout(5*time.Minute),
)

// Make a completion request
resp, err := client.Complete(ctx, claude.CompletionRequest{
    SystemPrompt: "You are a helpful assistant.",
    Messages: []claude.Message{
        {Role: claude.RoleUser, Content: "Hello!"},
    },
})

// Stream responses
stream, err := client.Stream(ctx, req)
for chunk := range stream {
    if chunk.Error != nil { ... }
    fmt.Print(chunk.Content)
    if chunk.Done { break }
}
```

### Template Rendering

```go
import "github.com/randalmurphal/llmkit/template"

engine := template.NewEngine()
result, err := engine.Render("Hello {{name}}, you have {{count}} messages.", map[string]any{
    "name":  "Alice",
    "count": 5,
})
// result: "Hello Alice, you have 5 messages."
```

### Token Counting

```go
import "github.com/randalmurphal/llmkit/tokens"

// Estimate token count (~4 chars per token)
counter := tokens.NewEstimatingCounter()
count := counter.Count("Hello, World!")  // ~4 tokens

// Budget management
budget := tokens.NewBudget(100000)
budget.Use(5000)
remaining := budget.Remaining()  // 95000
if budget.Exhausted() { ... }
```

### Response Parsing

```go
import "github.com/randalmurphal/llmkit/parser"

response := `Here's the JSON:
` + "```json" + `
{"name": "test", "value": 42}
` + "```"

// Extract JSON
data, err := parser.ExtractJSON(response)
// data: map[string]any{"name": "test", "value": 42}

// Extract code blocks
blocks := parser.ExtractCodeBlocks(response)
// blocks[0].Language: "json", blocks[0].Content: `{"name": "test", "value": 42}`
```

### Truncation

```go
import "github.com/randalmurphal/llmkit/truncate"
import "github.com/randalmurphal/llmkit/tokens"

counter := tokens.NewEstimatingCounter()
truncator := truncate.NewTruncator(counter, truncate.FromEnd)

// Truncate to fit token limit
result := truncator.Truncate(longText, 1000)

// Or use convenience functions
result = truncate.FromEndWithLimit(longText, 1000, counter)
```

### Model Selection

```go
import "github.com/randalmurphal/llmkit/model"

selector := model.NewSelector(
    model.WithThinkingModel(model.ModelOpus),
    model.WithDefaultModel(model.ModelSonnet),
    model.WithFastModel(model.ModelHaiku),
)

// Select by tier
m := selector.SelectForTier(model.TierThinking)  // ModelOpus
m = selector.SelectForTier(model.TierDefault)    // ModelSonnet
m = selector.SelectForTier(model.TierFast)       // ModelHaiku

// Cost tracking
tracker := model.NewCostTracker()
tracker.Record(model.ModelSonnet, 1000, 500)  // input, output tokens
cost := tracker.EstimatedCost()  // USD
```

---

## Design Principles

1. **Zero external dependencies** - Only Go stdlib
2. **À la carte imports** - Use only what you need
3. **Stable API** - Semver-friendly, rarely changes
4. **No forced patterns** - Interfaces for flexibility, concrete types for simplicity
5. **Configuration optional** - Sensible defaults, full configurability available

---

## Internal Dependencies

```
llmkit/
├── claude/      # No internal deps
├── template/    # No internal deps
├── tokens/      # No internal deps
├── parser/      # No internal deps
├── truncate/    # Imports tokens/
└── model/       # No internal deps
```

Only `truncate/` depends on another package (`tokens/`).

---

## Testing

```bash
# Run all tests
make test

# With coverage
make coverage

# Lint
make lint

# All checks
make verify
```

---

## Related Documentation

| File | Purpose |
|------|---------|
| `.spec/SPEC.md` | Full specification |
| `.spec/tracking/PROGRESS.md` | Implementation progress |
| `.spec/phases/*.md` | Phase-by-phase plan |
| `claude/CLAUDE.md` | Claude package details |
| `template/CLAUDE.md` | Template package details |
| `tokens/CLAUDE.md` | Tokens package details |
| `parser/CLAUDE.md` | Parser package details |
| `truncate/CLAUDE.md` | Truncate package details |
| `model/CLAUDE.md` | Model package details |
