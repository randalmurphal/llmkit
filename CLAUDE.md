# llmkit

**Go library for LLM utilities.** V2 exposes a shared root API, Claude and Codex provider packages, provider-native config ecosystems, and focused utility packages.

---

## Package Overview

| Package | Purpose | Key Types |
|---------|---------|-----------|
| root `llmkit/` | Shared client/config/types/registry/model helpers | `Client`, `Config`, `Request`, `Response` |
| `claude/` | Claude CLI wrapper with streaming | `ClaudeCLI`, `Config` |
| `codex/` | Codex CLI wrapper with streaming | `CodexCLI`, `Config` |
| `claudeconfig/` | Claude config and ecosystem parsing | `Settings`, `Skill`, `MCPConfig`, `Plugin` |
| `codexconfig/` | Codex config and ecosystem parsing | `ConfigFile`, `HookConfig`, `Skill`, `Plugin` |
| `env/` | Scoped hook/MCP/env lifecycle management | `Scope`, `Settings` |
| `worktree/` | Git worktree management | `Worktree`, `CreateOptions` |
| `claudecontract/` | Claude CLI constants/contracts | `Flag*`, `Event*`, `Permission*`, `Tool*` |
| `codexcontract/` | Codex CLI constants/contracts | `Flag*`, `Event*`, `Hook*` |
| `template/` | Prompt template rendering | `Engine`, `Render` |
| `tokens/` | Token counting/budgeting | `Counter`, `Budget` |
| `parser/` | Extract structured data | `ExtractJSON`, `ExtractCodeBlocks` |
| `truncate/` | Token-aware truncation | `Truncator`, `FromEnd` |

---

## Quick Start

### Unified Root API

```go
import (
    "github.com/randalmurphal/llmkit/v2"
    _ "github.com/randalmurphal/llmkit/v2/claude"  // Auto-registers
)

cfg := llmkit.Config{
    Model:   "claude-sonnet-4-20250514",
    WorkDir: "/path/to/project",
}
client, _ := llmkit.New("claude", cfg)
defer client.Close()

resp, _ := client.Complete(ctx, llmkit.Request{
    SystemPrompt: "You are a helpful assistant.",
    Messages:     []llmkit.Message{{Role: llmkit.RoleUser, Content: "Hello!"}},
})
```

### Claude CLI Direct

```go
import "github.com/randalmurphal/llmkit/v2/claude"

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

`llmkit/` owns the shared contracts. Provider packages register themselves with the root registry. `claudeconfig/` and `codexconfig/` own provider-native ecosystem parsing. `env/` and `worktree/` are opt-in lifecycle helpers.

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

1. **Shared root contracts** - no duplicate cross-provider request/response/config types
2. **Provider-native ecosystems** - Claude and Codex config surfaces stay explicit
3. **Strict structured output** - parse failures are errors, not silent fallbacks
4. **Opt-in side effects** - `env/` and `worktree/` never mutate projects implicitly

---

## Package Documentation

| Package | Doc |
|---------|-----|
| root `llmkit/` | `README.md` - shared API overview |
| `claude/` | `claude/CLAUDE.md` - streaming, sessions, behavioral tests |
| `codex/` | `codex/doc.go` - Codex CLI usage |
| `claudecontract/` | `claudecontract/CLAUDE.md` - CLI constants, version detection |
| `codexcontract/` | `codexcontract/doc.go` - Codex CLI constants and hooks |
| `claudeconfig/` | `claudeconfig/CLAUDE.md` - Claude ecosystem parsing |
| `codexconfig/` | `codexconfig/` - Codex ecosystem parsing |
| `env/` | source files and tests in `env/` - scoped lifecycle helpers |
| `worktree/` | source files and tests in `worktree/` - git worktree helpers |
| `tokens/` | `tokens/CLAUDE.md` - Token counting |
| `template/` | `template/CLAUDE.md` - Template rendering |
| `parser/` | `parser/CLAUDE.md` - Response parsing |
| `truncate/` | `truncate/CLAUDE.md` - Text truncation |
