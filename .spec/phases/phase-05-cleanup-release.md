# Phase 5: Cleanup & Release

**Status**: Pending

---

## Objective

Finalize llmkit as a standalone library, remove deprecated code from flowgraph, and tag releases.

---

## Prerequisites

- Phase 4 complete (all consumers working)
- All tests passing across all repos

---

## Tasks

### 5.1 Final Verification

Run full test suite across all repos:

```bash
# llmkit
cd ~/repos/llmkit
go test -race -cover ./...
golangci-lint run

# flowgraph
cd ~/repos/flowgraph
go test -race ./...
golangci-lint run

# devflow
cd ~/repos/devflow
go test -race ./...

# task-keeper
cd ~/repos/task-keeper
make test
make lint
```

### 5.2 Remove Deprecated flowgraph Code

After confirming all consumers work:

```bash
cd ~/repos/flowgraph

# Remove extracted packages
rm -rf pkg/flowgraph/llm/template/
rm -rf pkg/flowgraph/llm/tokens/
rm -rf pkg/flowgraph/llm/parser/
rm -rf pkg/flowgraph/llm/truncate/
rm -rf pkg/flowgraph/model/

# Remove root llm files (now in llmkit/claude/)
rm pkg/flowgraph/llm/client.go
rm pkg/flowgraph/llm/claude_cli.go
rm pkg/flowgraph/llm/credentials.go
rm pkg/flowgraph/llm/request.go
rm pkg/flowgraph/llm/errors.go
rm pkg/flowgraph/llm/mock.go
rm pkg/flowgraph/llm/*_test.go

# Keep any files that aren't moving (check first!)
ls pkg/flowgraph/llm/
```

### 5.3 Update Documentation

#### llmkit

- Finalize README.md
- Ensure all CLAUDE.md files accurate
- Add CHANGELOG.md
- Add CONTRIBUTING.md (optional)

#### flowgraph

- Update CLAUDE.md to reflect llmkit dependency
- Update package overview table
- Remove llm package documentation
- Add migration notes

#### task-keeper

- Update CLAUDE.md with new imports
- Update Foundation Integration section

### 5.4 Version Tags

```bash
# llmkit - first release
cd ~/repos/llmkit
git add .
git commit -m "feat: initial release - LLM utilities extracted from flowgraph"
git tag v1.0.0

# flowgraph - update version
cd ~/repos/flowgraph
git add .
git commit -m "refactor: extract LLM utilities to llmkit

BREAKING CHANGE: LLM packages moved to github.com/randalmurphal/llmkit

Migration:
- flowgraph/llm -> llmkit/claude
- flowgraph/llm/template -> llmkit/template
- flowgraph/llm/tokens -> llmkit/tokens
- flowgraph/llm/parser -> llmkit/parser
- flowgraph/llm/truncate -> llmkit/truncate
- flowgraph/model -> llmkit/model"
git tag v2.0.0  # Major version bump for breaking change
```

### 5.5 Remove Replace Directives

Once packages are published:

```bash
# In each repo's go.mod, remove:
# replace github.com/randalmurphal/llmkit => ../llmkit

# Then update:
go get github.com/randalmurphal/llmkit@v1.0.0
go mod tidy
```

### 5.6 CI/CD Setup for llmkit

Create `.github/workflows/ci.yml`:

```yaml
name: CI

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.23'
      - run: go build ./...
      - run: go test -race -cover ./...

  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.23'
      - uses: golangci/golangci-lint-action@v4
        with:
          version: latest
```

---

## Checklist

- [ ] All repos build with `go build ./...`
- [ ] All tests pass with `go test ./...`
- [ ] Coverage meets targets (90%+)
- [ ] Lint clean in all repos
- [ ] No import cycles
- [ ] Documentation updated
- [ ] Version tags created
- [ ] CI/CD configured for llmkit
- [ ] Replace directives removed (after publishing)

---

## Success Criteria

1. llmkit v1.0.0 tagged and working independently
2. flowgraph v2.0.0 tagged with llmkit dependency
3. task-keeper working with both
4. No duplicate code
5. All documentation accurate
