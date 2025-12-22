# Phase 3: flowgraph Integration

**Status**: Pending

---

## Objective

Update flowgraph to depend on llmkit, removing duplicated code.

---

## Prerequisites

- Phase 2 complete (llmkit functional and tested)

---

## Tasks

### 3.1 Update go.mod

```go
require github.com/randalmurphal/llmkit v1.0.0
```

For local development:
```go
replace github.com/randalmurphal/llmkit => ../llmkit
```

### 3.2 Update Imports

#### pkg/flowgraph/llm/ consumers

Find all files importing flowgraph's llm package:

```bash
grep -r "flowgraph/pkg/flowgraph/llm" ~/repos/flowgraph/
```

Update imports:
- `flowgraph/pkg/flowgraph/llm` → `github.com/randalmurphal/llmkit/claude`
- `flowgraph/pkg/flowgraph/llm/template` → `github.com/randalmurphal/llmkit/template`
- `flowgraph/pkg/flowgraph/llm/tokens` → `github.com/randalmurphal/llmkit/tokens`
- `flowgraph/pkg/flowgraph/llm/parser` → `github.com/randalmurphal/llmkit/parser`
- `flowgraph/pkg/flowgraph/llm/truncate` → `github.com/randalmurphal/llmkit/truncate`

#### pkg/flowgraph/model/ consumers

```bash
grep -r "flowgraph/pkg/flowgraph/model" ~/repos/flowgraph/
```

Update imports:
- `flowgraph/pkg/flowgraph/model` → `github.com/randalmurphal/llmkit/model`

### 3.3 Update flowgraph Core

Files in flowgraph that use LLM utilities:

| File | LLM Usage | Update Required |
|------|-----------|-----------------|
| context.go | WithLLM option | Import llmkit/claude |
| options.go | LLM client option | Import llmkit/claude |
| execute.go | LLM context access | Import llmkit/claude |

### 3.4 Backward Compatibility (Optional)

If maintaining backward compatibility, create re-export files:

```go
// pkg/flowgraph/llm/compat.go
package llm

import "github.com/randalmurphal/llmkit/claude"

// Deprecated: Use github.com/randalmurphal/llmkit/claude instead.
type Client = claude.Client

// Deprecated: Use github.com/randalmurphal/llmkit/claude instead.
type ClaudeCLI = claude.ClaudeCLI
```

**Recommendation**: Skip backward compatibility, make clean break.

### 3.5 Remove Extracted Code

After successful integration:

```bash
rm -rf ~/repos/flowgraph/pkg/flowgraph/llm/template/
rm -rf ~/repos/flowgraph/pkg/flowgraph/llm/tokens/
rm -rf ~/repos/flowgraph/pkg/flowgraph/llm/parser/
rm -rf ~/repos/flowgraph/pkg/flowgraph/llm/truncate/
rm -rf ~/repos/flowgraph/pkg/flowgraph/model/
rm ~/repos/flowgraph/pkg/flowgraph/llm/client.go
rm ~/repos/flowgraph/pkg/flowgraph/llm/claude_cli.go
rm ~/repos/flowgraph/pkg/flowgraph/llm/credentials.go
rm ~/repos/flowgraph/pkg/flowgraph/llm/request.go
rm ~/repos/flowgraph/pkg/flowgraph/llm/errors.go
rm ~/repos/flowgraph/pkg/flowgraph/llm/mock.go
rm ~/repos/flowgraph/pkg/flowgraph/llm/*_test.go
```

### 3.6 Update Documentation

Update flowgraph CLAUDE.md:
- Remove LLM package documentation
- Add llmkit as dependency
- Update import examples

---

## Validation

1. `go build ./...` in flowgraph
2. `go test ./...` in flowgraph
3. All existing flowgraph tests pass
4. No import cycles

---

## Success Criteria

1. flowgraph builds and tests pass
2. No duplicated code between flowgraph and llmkit
3. flowgraph CLAUDE.md updated
