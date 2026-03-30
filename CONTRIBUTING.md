# Contributing to llmkit

## Development

### Prerequisites

- Go 1.24+
- golangci-lint

### Setup

```bash
git clone https://github.com/randalmurphal/llmkit.git
cd llmkit
go test ./...
make verify  # Run all checks
```

### Commands

```bash
make build     # Build all packages
make test      # Run tests
make coverage  # Run tests with coverage
make lint      # Run linter
make verify    # Run all checks
```

### Code Quality

- Run `make verify` before submitting PRs
- Maintain 90%+ test coverage on all packages
- Follow existing code patterns

### Package Structure

| Package | Purpose |
|---------|---------|
| root `llmkit/` | Shared client/config/types/registry/model helpers |
| `claude/` | Claude CLI wrapper and sessions |
| `codex/` | Codex CLI wrapper and sessions |
| `claudeconfig/` | Claude ecosystem parsing |
| `codexconfig/` | Codex ecosystem parsing |
| `env/` | Scoped hook/MCP/env lifecycle |
| `worktree/` | Git worktree lifecycle |
| `providers/` | Blank-import convenience package for registry registration |
| `tokens/` | Token counting and budget management |
| `template/` | Prompt template rendering |
| `parser/` | LLM response parsing |
| `truncate/` | Token-aware text truncation |

### Versioning

- The repository is `github.com/randalmurphal/llmkit`
- The V2 Go module path is `github.com/randalmurphal/llmkit/v2`
- V1 maintenance fixes can continue landing on `main`, but all V2 code and docs must use `/v2` imports

### Testing

Each package has comprehensive tests. Run with:

```bash
go test ./...                    # All tests
go test ./claude/...             # Single package
go test -cover ./...             # With coverage
```

### Adding Features

1. Add tests first
2. Implement the feature
3. Run `make verify`
4. Submit PR
