# Phase 4: Consumer Updates

**Status**: Pending

---

## Objective

Update task-keeper and devflow to use llmkit directly where appropriate.

---

## Prerequisites

- Phase 3 complete (flowgraph using llmkit)

---

## task-keeper Updates

### 4.1 Update go.mod

```go
require github.com/randalmurphal/llmkit v1.0.0
```

For local development:
```go
replace github.com/randalmurphal/llmkit => ../llmkit
```

### 4.2 Import Changes

| File | Current Import | New Import |
|------|----------------|------------|
| internal/service/prompt_service.go | flowgraph/llm/template | llmkit/template |
| internal/context/builder.go | flowgraph/llm/tokens | llmkit/tokens |
| internal/claude/transcript.go | flowgraph/llm/parser | llmkit/parser |
| internal/flow/claude_action.go | flowgraph/llm/template | llmkit/template |
| internal/flow/nodes.go | flowgraph/config, expr, template | flowgraph/config, flowgraph/expr, llmkit/template |
| internal/trigger/listener.go | flowgraph/expr, llm/template | flowgraph/expr, llmkit/template |
| internal/api/trigger_handler.go | flowgraph/expr, llm/template | flowgraph/expr, llmkit/template |

### 4.3 Files to Update

```bash
# Find all flowgraph/llm imports in task-keeper
grep -r "flowgraph/pkg/flowgraph/llm" ~/repos/task-keeper/internal/
```

Expected files:
- `internal/service/prompt_service.go`
- `internal/context/builder.go`
- `internal/claude/transcript.go`
- `internal/claude/client.go`
- `internal/flow/claude_action.go`
- `internal/flow/nodes.go`
- `internal/trigger/listener.go`
- `internal/api/trigger_handler.go`

### 4.4 Update CLAUDE.md

Update task-keeper CLAUDE.md:
- Add llmkit to dependencies section
- Update import examples
- Update "Uses flowgraph/llm/..." references to llmkit

---

## devflow Updates

### 4.5 Check Current Usage

```bash
grep -r "flowgraph/pkg/flowgraph/llm" ~/repos/devflow/
grep -r "flowgraph/pkg/flowgraph/model" ~/repos/devflow/
```

### 4.6 Update if Needed

If devflow imports LLM utilities directly (not via flowgraph context):
1. Add llmkit to go.mod
2. Update imports

**Note**: devflow may not need changes if it only uses flowgraph's context-injected LLM client.

---

## Validation

### task-keeper

```bash
cd ~/repos/task-keeper
go build ./...
make test
make lint
```

### devflow

```bash
cd ~/repos/devflow
go build ./...
go test ./...
```

---

## Import Update Script

```bash
#!/bin/bash
# Run from task-keeper root

# Template
find . -name "*.go" -exec sed -i \
  's|github.com/randalmurphal/flowgraph/pkg/flowgraph/llm/template|github.com/randalmurphal/llmkit/template|g' {} \;

# Tokens
find . -name "*.go" -exec sed -i \
  's|github.com/randalmurphal/flowgraph/pkg/flowgraph/llm/tokens|github.com/randalmurphal/llmkit/tokens|g' {} \;

# Parser
find . -name "*.go" -exec sed -i \
  's|github.com/randalmurphal/flowgraph/pkg/flowgraph/llm/parser|github.com/randalmurphal/llmkit/parser|g' {} \;

# Truncate
find . -name "*.go" -exec sed -i \
  's|github.com/randalmurphal/flowgraph/pkg/flowgraph/llm/truncate|github.com/randalmurphal/llmkit/truncate|g' {} \;

# Model
find . -name "*.go" -exec sed -i \
  's|github.com/randalmurphal/flowgraph/pkg/flowgraph/model|github.com/randalmurphal/llmkit/model|g' {} \;

# Claude (root llm package -> claude)
find . -name "*.go" -exec sed -i \
  's|github.com/randalmurphal/flowgraph/pkg/flowgraph/llm"|github.com/randalmurphal/llmkit/claude"|g' {} \;

# Run goimports to fix formatting
goimports -w .
```

---

## Success Criteria

1. task-keeper builds: `go build ./...`
2. task-keeper tests pass: `make test`
3. task-keeper lint clean: `make lint`
4. devflow builds and tests pass
5. All three repos work together with local replace directives
