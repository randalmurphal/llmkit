# llmkit

**Go library for LLM utilities.** Multi-provider toolkit with unified interface, Claude CLI wrapper, token counting, prompt templates, and response parsing.

---

## Package Overview

| Package | Purpose | Key Types |
|---------|---------|-----------|
| `provider/` | Unified multi-provider interface | `Client`, `Config`, `Request`, `Response` |
| `claude/` | Claude CLI wrapper with streaming | `ClaudeCLI`, `StreamEvent`, `Config` |
| `claudecontract/` | Claude CLI interface constants | `Flag*`, `Event*`, `Permission*`, `Tool*` |
| `claudeconfig/` | Claude config file parsing | `Settings`, `Skill`, `MCPConfig`, `Plugin` |
| `gemini/` | Gemini CLI wrapper | `GeminiCLI`, `Config` |
| `codex/` | OpenAI Codex CLI wrapper | `CodexCLI`, `Config` |
| `opencode/` | OpenCode CLI wrapper | `OpenCodeCLI`, `Config` |
| `continue/` | Continue.dev CLI wrapper | `ContinueCLI`, `Config` |
| `aider/` | Aider CLI wrapper | `AiderCLI`, `Config` |
| `template/` | Prompt template rendering | `Engine`, `Render` |
| `tokens/` | Token counting/budgeting | `Counter`, `Budget` |
| `parser/` | Extract structured data | `ExtractJSON`, `ExtractCodeBlocks` |
| `truncate/` | Token-aware truncation | `Truncator`, `FromEnd` |
| `model/` | Model selection/cost tracking | `Selector`, `CostTracker` |

---

## Quick Start

### Unified Provider Interface

```go
import (
    "github.com/randalmurphal/llmkit/provider"
    _ "github.com/randalmurphal/llmkit/claude"  // Auto-registers
)

cfg := provider.Config{
    Provider: "claude",
    Model:    "claude-sonnet-4-20250514",
    WorkDir:  "/path/to/project",
}
client, _ := provider.New(cfg.Provider, cfg)
defer client.Close()

resp, _ := client.Complete(ctx, provider.Request{
    SystemPrompt: "You are a helpful assistant.",
    Messages:     []provider.Message{{Role: provider.RoleUser, Content: "Hello!"}},
})
```

### Claude CLI Direct

```go
import "github.com/randalmurphal/llmkit/claude"

client := claude.NewClaudeCLI(
    claude.WithModel("claude-sonnet-4-20250514"),
    claude.WithDangerouslySkipPermissions(),
)

resp, _ := client.Complete(ctx, claude.CompletionRequest{
    Messages: []claude.Message{{Role: claude.RoleUser, Content: "Hello!"}},
})
```

### Streaming with Events

```go
events, result := client.StreamJSON(ctx, req)
for event := range events {
    switch event.Type {
    case claude.StreamEventAssistant:
        fmt.Print(event.Assistant.Text)
    case claude.StreamEventUser:
        // Tool result
    }
}
final, _ := result.Wait(ctx)
```

---

## Internal Dependencies

```
llmkit/
├── provider/        # Core interface, no deps
├── claudecontract/  # CLI constants, no deps
├── claude/          # Imports provider/, claudecontract/
├── claudeconfig/    # Imports claudecontract/
├── gemini/          # Imports provider/
├── template/        # No deps
├── tokens/          # No deps
├── parser/          # No deps
├── truncate/        # Imports tokens/
└── model/           # No deps
```

---

## Testing

```bash
# All tests
go test ./...

# Claude behavioral tests (uses API credits)
TEST_BEHAVIORAL=1 go test ./claude/... -run Behavioral -v -timeout 15m

# Contract validation against real CLI
TEST_REAL_CLI=1 go test ./claudecontract/... -run Real -v

# Coverage
go test ./... -coverprofile=coverage.out
```

---

## Design Principles

1. **Zero external dependencies** - Only Go stdlib
2. **À la carte imports** - Use only what you need
3. **Stable API** - Semver-friendly
4. **Configuration optional** - Sensible defaults

---

## Package Documentation

| Package | Doc |
|---------|-----|
| `claude/` | `claude/CLAUDE.md` - Streaming events, behavioral tests |
| `claudecontract/` | `claudecontract/CLAUDE.md` - CLI constants, version detection |
| `claudeconfig/` | `claudeconfig/CLAUDE.md` - Config file parsing |
| `model/` | `model/CLAUDE.md` - Model selection, cost tracking |
| `tokens/` | `tokens/CLAUDE.md` - Token counting |
| `template/` | `template/CLAUDE.md` - Template rendering |
| `parser/` | `parser/CLAUDE.md` - Response parsing |
| `truncate/` | `truncate/CLAUDE.md` - Text truncation |
