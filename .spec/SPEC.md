# llmkit Specification

**Purpose**: Standalone Go library for LLM utilities - extracted from flowgraph to enable lightweight consumption.

**Status**: Planning

---

## Problem Statement

flowgraph bundles LLM utilities (template rendering, token counting, response parsing, Claude CLI) with graph execution. Projects that need only LLM utilities must import the entire graph engine.

### Current State
```
flowgraph (graph engine + LLM utilities)
    ↑
devflow (dev workflows)
    ↑
task-keeper (uses both)
```

### Target State
```
llmkit (pure LLM utilities - zero external deps)
    ↑              ↑
flowgraph        devflow
(graph engine)   (dev workflows)
    ↑              ↑
    └── task-keeper ──┘
```

---

## Design Principles

1. **Zero external dependencies** - Only Go stdlib + `golang.org/x/` packages
2. **À la carte imports** - Each subpackage usable independently
3. **Stable API** - These utilities rarely change, semver-friendly
4. **No forced patterns** - Interfaces for flexibility, concrete types for simplicity
5. **Configuration optional** - Sensible defaults, full configurability available

---

## Package Structure

```
llmkit/
├── go.mod                      # module github.com/randalmurphal/llmkit
├── CLAUDE.md                   # Root documentation
├── doc.go                      # Package documentation
│
├── claude/                     # Claude CLI wrapper
│   ├── CLAUDE.md
│   ├── doc.go
│   ├── client.go              # Client interface
│   ├── cli.go                 # ClaudeCLI implementation
│   ├── credentials.go         # OAuth credential management
│   ├── request.go             # Request/response types
│   ├── errors.go              # Error types
│   ├── mock.go                # MockClient for testing
│   └── *_test.go
│
├── template/                   # Prompt template rendering
│   ├── CLAUDE.md
│   ├── doc.go
│   ├── engine.go              # Template engine
│   ├── syntax.go              # {{var}} syntax parsing
│   ├── funcs.go               # Built-in template functions
│   ├── errors.go              # Template errors
│   └── *_test.go
│
├── tokens/                     # Token counting and budgeting
│   ├── CLAUDE.md
│   ├── doc.go
│   ├── counter.go             # Token estimation
│   ├── budget.go              # Token budget management
│   └── *_test.go
│
├── parser/                     # LLM response parsing
│   ├── CLAUDE.md
│   ├── doc.go
│   ├── parser.go              # Main parser
│   ├── extraction.go          # JSON/YAML/code extraction
│   └── *_test.go
│
├── truncate/                   # Token-aware truncation
│   ├── CLAUDE.md
│   ├── doc.go
│   ├── truncator.go           # Core truncator
│   ├── strategies.go          # FromEnd, FromMiddle, FromStart
│   ├── convenience.go         # Helper functions
│   └── *_test.go
│
├── model/                      # Model selection and cost tracking
│   ├── CLAUDE.md
│   ├── doc.go
│   ├── names.go               # ModelOpus, ModelSonnet, ModelHaiku
│   ├── selector.go            # Task-based model selection
│   ├── escalation.go          # Escalation chains
│   ├── cost.go                # Cost tracking
│   └── *_test.go
│
└── internal/
    └── testutil/              # Shared test utilities
        └── testutil.go
```

---

## Extraction Mapping

### From flowgraph/pkg/flowgraph/llm/

| Source | Destination | Notes |
|--------|-------------|-------|
| `client.go` | `claude/client.go` | Interface definition |
| `claude_cli.go` | `claude/cli.go` | ClaudeCLI implementation |
| `credentials.go` | `claude/credentials.go` | OAuth credential handling |
| `request.go` | `claude/request.go` | Request/response types |
| `errors.go` | `claude/errors.go` | Error types |
| `mock.go` | `claude/mock.go` | MockClient |
| `template/*` | `template/*` | Direct copy, update imports |
| `tokens/*` | `tokens/*` | Direct copy, update imports |
| `parser/*` | `parser/*` | Direct copy, update imports |
| `truncate/*` | `truncate/*` | Direct copy, update imports |

### From flowgraph/pkg/flowgraph/model/

| Source | Destination | Notes |
|--------|-------------|-------|
| `names.go` | `model/names.go` | ModelName, Tier types |
| `selector.go` | `model/selector.go` | Model selection |
| `escalation.go` | `model/escalation.go` | Escalation chains |
| `cost.go` | `model/cost.go` | Cost tracking |

---

## Consumer Changes Required

### flowgraph

After extraction, flowgraph will:
1. Add `require github.com/randalmurphal/llmkit` to go.mod
2. Update imports from internal llm/ to llmkit packages
3. Remove extracted packages from pkg/flowgraph/llm/ and model/
4. Re-export types for backward compatibility (optional, may deprecate)

### devflow

If devflow uses any LLM utilities directly:
1. Add `require github.com/randalmurphal/llmkit`
2. Update imports

### task-keeper

1. Add `require github.com/randalmurphal/llmkit`
2. Update imports:
   - `flowgraph/llm/template` → `llmkit/template`
   - `flowgraph/llm/tokens` → `llmkit/tokens`
   - `flowgraph/llm/parser` → `llmkit/parser`
   - `flowgraph/llm` (client) → `llmkit/claude`
   - `flowgraph/model` → `llmkit/model`

---

## API Design Decisions

### claude/ Package

**Client Interface** - Keep generic for future providers:
```go
type Client interface {
    Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error)
    Stream(ctx context.Context, req CompletionRequest) (<-chan StreamChunk, error)
}
```

**ClaudeCLI Options** - Functional options pattern:
```go
client := claude.NewCLI(
    claude.WithModel("sonnet"),
    claude.WithTimeout(5*time.Minute),
    claude.WithHomeDir("/custom/path"),  // For containers
)
```

### template/ Package

**Engine** - Simple, predictable:
```go
engine := template.NewEngine()
result, err := engine.Render("Hello {{name}}", map[string]any{"name": "World"})
```

**No dependencies on other llmkit packages** - template/ should not import tokens/.

### tokens/ Package

**Counter Interface** - Allow custom implementations:
```go
type Counter interface {
    Count(text string) int
}

// Default implementation uses estimation (~4 chars/token)
counter := tokens.NewEstimatingCounter()
```

**Budget** - Track usage against limits:
```go
budget := tokens.NewBudget(100000)
budget.Use(5000)
if budget.Exhausted() { ... }
```

### parser/ Package

**Extraction functions** - Clear, single-purpose:
```go
json, err := parser.ExtractJSON(response)
yaml, err := parser.ExtractYAML(response)
blocks := parser.ExtractCodeBlocks(response)
```

### truncate/ Package

**Strategy pattern**:
```go
truncator := truncate.NewTruncator(counter, truncate.FromEnd)
result := truncator.Truncate(text, maxTokens)
```

### model/ Package

**Selector** - Task-agnostic tier selection:
```go
selector := model.NewSelector(
    model.WithThinkingModel(model.ModelOpus),
    model.WithDefaultModel(model.ModelSonnet),
    model.WithFastModel(model.ModelHaiku),
)
m := selector.SelectForTier(model.TierThinking)
```

---

## Testing Strategy

1. **Unit tests** - Each package has comprehensive unit tests (extracted from flowgraph)
2. **No integration tests requiring Claude API** - Use MockClient
3. **Coverage target** - 90%+ (matching flowgraph coverage)

---

## Migration Path

### Phase 1: Create llmkit (this work)
- Create repo structure
- Document plan
- Prepare for extraction

### Phase 2: Extract code
- Copy files with import updates
- Run tests
- Achieve parity with flowgraph

### Phase 3: Update flowgraph
- Add llmkit dependency
- Update internal imports
- Deprecate old packages (or remove)

### Phase 4: Update consumers
- Update task-keeper imports
- Update devflow if needed
- Test all projects

### Phase 5: Cleanup
- Remove deprecated flowgraph packages
- Tag llmkit v1.0.0
- Update documentation

---

## Open Questions

1. **Backward compatibility in flowgraph** - Re-export llmkit types from old paths, or breaking change?
2. **model/ package location** - Currently in flowgraph, but purely about LLM model selection. Include in llmkit?
3. **Versioning strategy** - Start at v0.1.0 or v1.0.0? (Recommend v1.0.0 since API is stable)

---

## Success Criteria

1. llmkit can be imported standalone without flowgraph
2. All existing tests pass
3. task-keeper works with updated imports
4. flowgraph works with llmkit dependency
5. No circular dependencies
