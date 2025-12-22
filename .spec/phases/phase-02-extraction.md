# Phase 2: Code Extraction

**Status**: Pending

---

## Objective

Extract LLM utilities from flowgraph into llmkit with full test coverage.

---

## Prerequisites

- Phase 1 complete (structure ready)

---

## Tasks

### 2.1 claude/ Package

Extract from `flowgraph/pkg/flowgraph/llm/`:

| Source File | Destination | Changes Required |
|-------------|-------------|------------------|
| client.go | claude/client.go | Update package name |
| claude_cli.go | claude/cli.go | Update package, imports |
| credentials.go | claude/credentials.go | Update package name |
| request.go | claude/request.go | Update package name |
| errors.go | claude/errors.go | Update package name |
| mock.go | claude/mock.go | Update package name |
| claude_cli_test.go | claude/cli_test.go | Update imports |
| claude_cli_stream_test.go | claude/cli_stream_test.go | Update imports |
| mock_test.go | claude/mock_test.go | Update imports |
| internal_test.go | claude/internal_test.go | Update imports |
| credentials_test.go | claude/credentials_test.go | Update imports |

### 2.2 template/ Package

Extract from `flowgraph/pkg/flowgraph/llm/template/`:

| Source File | Destination | Changes Required |
|-------------|-------------|------------------|
| doc.go | template/doc.go | Update module path |
| engine.go | template/engine.go | Update package path |
| syntax.go | template/syntax.go | Direct copy |
| funcs.go | template/funcs.go | Direct copy |
| errors.go | template/errors.go | Direct copy |
| engine_test.go | template/engine_test.go | Update imports |

### 2.3 tokens/ Package

Extract from `flowgraph/pkg/flowgraph/llm/tokens/`:

| Source File | Destination | Changes Required |
|-------------|-------------|------------------|
| doc.go | tokens/doc.go | Update module path |
| counter.go | tokens/counter.go | Direct copy |
| budget.go | tokens/budget.go | Direct copy |
| counter_test.go | tokens/counter_test.go | Update imports |
| budget_test.go | tokens/budget_test.go | Update imports |

### 2.4 parser/ Package

Extract from `flowgraph/pkg/flowgraph/llm/parser/`:

| Source File | Destination | Changes Required |
|-------------|-------------|------------------|
| doc.go | parser/doc.go | Update module path |
| parser.go | parser/parser.go | Direct copy |
| extraction.go | parser/extraction.go | Direct copy |
| parser_test.go | parser/parser_test.go | Update imports |

### 2.5 truncate/ Package

Extract from `flowgraph/pkg/flowgraph/llm/truncate/`:

| Source File | Destination | Changes Required |
|-------------|-------------|------------------|
| doc.go | truncate/doc.go | Update module path |
| truncator.go | truncate/truncator.go | Update imports (uses tokens) |
| strategies.go | truncate/strategies.go | Direct copy |
| convenience.go | truncate/convenience.go | Update imports (uses tokens) |
| truncator_test.go | truncate/truncator_test.go | Update imports |

**Note**: truncate/ imports tokens/ - update to `github.com/randalmurphal/llmkit/tokens`

### 2.6 model/ Package

Extract from `flowgraph/pkg/flowgraph/model/`:

| Source File | Destination | Changes Required |
|-------------|-------------|------------------|
| names.go | model/names.go | Update package path |
| selector.go | model/selector.go | Direct copy |
| escalation.go | model/escalation.go | Direct copy |
| cost.go | model/cost.go | Direct copy |
| model_test.go | model/model_test.go | Update imports |

---

## Validation

For each package:

1. Run `go build ./...`
2. Run `go test ./...`
3. Check coverage: `go test -cover ./...`
4. Run `golangci-lint run`

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

Only `truncate/` has internal dependencies (on `tokens/`).

---

## Success Criteria

1. All packages build: `go build ./...`
2. All tests pass: `go test ./...`
3. Coverage 90%+
4. Lint clean: `golangci-lint run`
