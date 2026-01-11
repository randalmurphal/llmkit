# llmkit

**Go library for LLM utilities.** Multi-provider toolkit for LLM integration with unified interface, token counting, prompt templates, response parsing, and model selection.

---

## Package Overview

| Package | Purpose | Key Types |
|---------|---------|-----------|
| `provider/` | Unified multi-provider interface | `Client`, `Config`, `Request`, `Response`, `Register` |
| `claude/` | Claude CLI wrapper | `ClaudeCLI`, `Config`, `MockClient` |
| `gemini/` | Gemini CLI wrapper | `GeminiCLI`, `Config` |
| `codex/` | OpenAI Codex CLI wrapper | `CodexCLI`, `Config` |
| `opencode/` | OpenCode CLI wrapper | `OpenCodeCLI`, `Config` |
| `continue/` | Continue.dev CLI wrapper (local models) | `ContinueCLI`, `Config` |
| `aider/` | Aider CLI wrapper (git-centric editing) | `AiderCLI`, `Config`, `EditMarker` |
| `template/` | Prompt template rendering | `Engine`, `Template`, `Render` |
| `tokens/` | Token counting and budgeting | `Counter`, `Budget`, `Estimate` |
| `parser/` | Extract structured data from LLM responses | `ExtractJSON`, `ExtractYAML`, `ExtractCodeBlocks` |
| `truncate/` | Token-aware text truncation | `Truncator`, `Strategy`, `FromEnd`, `FromMiddle` |
| `model/` | Model selection and cost tracking | `Selector`, `CostTracker`, `EscalationChain` |

---

## Quick Reference

### Unified Provider Interface

```go
import (
    "github.com/randalmurphal/llmkit/provider"
    // Import providers to auto-register them
    _ "github.com/randalmurphal/llmkit/claude"
    _ "github.com/randalmurphal/llmkit/gemini"
    _ "github.com/randalmurphal/llmkit/continue"
    _ "github.com/randalmurphal/llmkit/aider"
)

// Create any provider with unified config
cfg := provider.Config{
    Provider:     "claude",  // or "gemini", "continue", "aider"
    Model:        "claude-sonnet-4-20250514",
    Timeout:      5 * time.Minute,
    WorkDir:      "/path/to/project",
}

client, err := provider.New(cfg.Provider, cfg)
if err != nil { ... }
defer client.Close()

// Universal request/response
resp, err := client.Complete(ctx, provider.Request{
    SystemPrompt: "You are a helpful assistant.",
    Messages: []provider.Message{
        {Role: provider.RoleUser, Content: "Hello!"},
    },
})
fmt.Println(resp.Content)
```

### Claude CLI

```go
import "github.com/randalmurphal/llmkit/claude"

// Create client with options (original pattern)
client := claude.NewClaudeCLI(
    claude.WithModel("claude-sonnet-4-20250514"),
    claude.WithTimeout(5*time.Minute),
)

// Create client from config struct (new - enables YAML/JSON/env loading)
cfg := claude.Config{
    Model:        "claude-opus-4-5-20251101",
    SystemPrompt: "You are a code reviewer.",
    MaxTurns:     10,
}
client = claude.NewFromConfig(cfg)

// Create client from environment variables
client = claude.NewFromEnv()  // Uses CLAUDE_* env vars

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

### Continue.dev CLI (Local Models)

```go
import continuedev "github.com/randalmurphal/llmkit/continue"

// Create Continue.dev client for local model support
cli := continuedev.NewContinueCLI(
    continuedev.WithModel("llama3.2:latest"),
    continuedev.WithConfigPath("~/.continue/config.yaml"),
    continuedev.WithAllowedTools([]string{"Write()", "Edit()"}),
    continuedev.WithExcludedTools([]string{"Bash(rm *)"}),
)

// Or use provider interface
cfg := provider.Config{
    Provider: "continue",
    Model:    "llama3.2:latest",
    WorkDir:  "/path/to/project",
    Options: map[string]any{
        "config_path":    "~/.continue/config.yaml",
        "allowed_tools":  []string{"Write()", "Edit()"},
        "verbose":        true,
    },
}
client, _ := provider.New("continue", cfg)
```

### Aider CLI (Git-Centric Editing)

```go
import "github.com/randalmurphal/llmkit/aider"

// Create Aider client for git-aware code editing
cli := aider.NewAiderCLI(
    aider.WithModel("ollama_chat/llama3.2:latest"),
    aider.WithEditableFiles([]string{"src/main.go", "src/utils.go"}),
    aider.WithReadOnlyFiles([]string{"README.md"}),
    aider.WithYesAlways(),        // Auto-confirm for automation
    aider.WithNoAutoCommits(),    // Control commits manually
)

// Or use provider interface
cfg := provider.Config{
    Provider: "aider",
    Model:    "ollama_chat/llama3.2:latest",
    WorkDir:  "/path/to/project",
    Options: map[string]any{
        "editable_files":   []string{"src/main.go"},
        "read_only_files":  []string{"README.md"},
        "yes_always":       true,
        "no_auto_commits":  true,
        "ollama_api_base":  "http://localhost:11434",
    },
}
client, _ := provider.New("aider", cfg)
```

### Configuration & Dependency Injection

```go
import "github.com/randalmurphal/llmkit/claude"

// Struct-based config - serializable to YAML/JSON
cfg := claude.Config{
    Model:        "claude-opus-4-5-20251101",
    MaxTurns:     20,
    Timeout:      10*time.Minute,
    MaxBudgetUSD: 5.0,
    WorkDir:      "/app",
}

// Load from environment (CLAUDE_* prefix)
cfg = claude.FromEnv()

// Validate before use
if err := cfg.Validate(); err != nil {
    log.Fatal(err)
}

// Mix config with option overrides
client := claude.NewFromConfig(cfg,
    claude.WithDangerouslySkipPermissions(),  // option overrides config
)

// Singleton pattern for app-wide client
claude.SetDefaultConfig(cfg)
client = claude.GetDefaultClient()

// Context injection for DI
ctx := claude.ContextWithClient(context.Background(), client)
// Later in handlers:
client = claude.ClientFromContext(ctx)
client = claude.MustClientFromContext(ctx)  // panics if missing

// Testing with mock
mock := claude.NewMockClient("test response")
claude.SetDefaultClient(mock)
defer claude.ResetDefaultClient()
```

**Environment Variables** (CLAUDE_ prefix):
- `CLAUDE_MODEL` - Model name
- `CLAUDE_FALLBACK_MODEL` - Fallback model
- `CLAUDE_SYSTEM_PROMPT` - System prompt
- `CLAUDE_MAX_TURNS` - Max turns
- `CLAUDE_TIMEOUT` - Timeout (e.g., "5m")
- `CLAUDE_MAX_BUDGET_USD` - Budget limit
- `CLAUDE_WORK_DIR` - Working directory
- `CLAUDE_PATH` - Path to claude binary
- `CLAUDE_HOME_DIR` - Override HOME (containers)
- `CLAUDE_CONFIG_DIR` - Override .claude directory
- `CLAUDE_SKIP_PERMISSIONS` - Skip permissions (true/1)

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
├── provider/    # Core interface, no internal deps
├── claude/      # Imports provider/
├── gemini/      # Imports provider/
├── codex/       # Imports provider/
├── opencode/    # Imports provider/
├── continue/    # Imports provider/
├── aider/       # Imports provider/
├── template/    # No internal deps
├── tokens/      # No internal deps
├── parser/      # No internal deps
├── truncate/    # Imports tokens/
└── model/       # No internal deps
```

All CLI wrappers depend on `provider/` for the unified interface.

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
| `provider/doc.go` | Provider interface documentation |
| `claude/CLAUDE.md` | Claude package details |
| `continue/doc.go` | Continue.dev CLI documentation |
| `aider/doc.go` | Aider CLI documentation |
| `template/CLAUDE.md` | Template package details |
| `tokens/CLAUDE.md` | Tokens package details |
| `parser/CLAUDE.md` | Parser package details |
| `truncate/CLAUDE.md` | Truncate package details |
| `model/CLAUDE.md` | Model package details |
