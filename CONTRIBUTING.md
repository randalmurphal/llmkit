# Contributing to llmkit

## Development

### Prerequisites

- Go 1.24+
- golangci-lint

### Setup

```bash
git clone https://github.com/randalmurphal/llmkit.git
cd llmkit
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
| `claude/` | Claude CLI wrapper and client interface |
| `tokens/` | Token counting and budget management |
| `template/` | Prompt template rendering |
| `parser/` | LLM response parsing |
| `truncate/` | Token-aware text truncation |
| `model/` | Model selection and cost tracking |

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
