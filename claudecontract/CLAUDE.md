# claudecontract

**Single source of truth for Claude CLI interface constants.** Centralizes all volatile strings (flags, events, paths) that may change between CLI versions.

---

## Package Contents

| File | Purpose | Key Exports |
|------|---------|-------------|
| `version.go` | CLI version detection | `TestedCLIVersion`, `CheckVersion()`, `ParseVersion()` |
| `flags.go` | CLI flag constants | `FlagModel`, `FlagPrint`, `FlagOutputFormat`, etc. |
| `events.go` | Stream event types | `EventTypeAssistant`, `EventTypeUser`, `EventTypeResult` |
| `permissions.go` | Permission modes | `PermissionDefault`, `PermissionBypassPermissions` |
| `tools.go` | Built-in tool names | `ToolRead`, `ToolWrite`, `ToolBash`, `BuiltinTools()` |
| `paths.go` | File/directory names | `FileSettings`, `DirClaude`, `FileMCPConfig` |
| `formats.go` | Output formats, hooks | `FormatStreamJSON`, `HookPreToolUse`, `TransportStdio` |

---

## Why This Package Exists

Claude CLI interface details are scattered across the codebase:
- Flag names in `claude/claude_cli.go`
- Event types in `claude/stream_types.go`
- File paths in `claudeconfig/*.go`

**Problem**: When Claude CLI changes, multiple files need updating.

**Solution**: Single source of truth. Both `claude/` and `claudeconfig/` import from here.

---

## Key Constants

### Flags (flags.go)

| Constant | Value | Used For |
|----------|-------|----------|
| `FlagPrint` | `--print` | Single-shot mode |
| `FlagOutputFormat` | `--output-format` | Output format selection |
| `FlagModel` | `--model` | Model selection |
| `FlagSessionID` | `--session-id` | Session management |
| `FlagAllowedTools` | `--allowedTools` | Tool restrictions |
| `FlagDangerouslySkipPermissions` | `--dangerously-skip-permissions` | Non-interactive mode |
| `FlagMaxTurns` | `--max-turns` | Conversation limits |
| `FlagJSONSchema` | `--json-schema` | Structured output |

### Events (events.go)

| Constant | Value | When Emitted |
|----------|-------|--------------|
| `EventTypeSystem` | `system` | Init, hooks, compact boundary |
| `EventTypeAssistant` | `assistant` | Claude responses |
| `EventTypeUser` | `user` | Tool results |
| `EventTypeResult` | `result` | Final result |

### Result Subtypes

| Constant | Value | Meaning |
|----------|-------|---------|
| `ResultSubtypeSuccess` | `success` | Normal completion |
| `ResultSubtypeErrorMaxTurns` | `error_max_turns` | Turn limit reached |
| `ResultSubtypeErrorMaxBudgetUSD` | `error_max_budget_usd` | Budget exceeded |
| `ResultSubtypeErrorDuringExecution` | `error_during_execution` | Runtime error |

### Paths (paths.go)

| Constant | Value | Purpose |
|----------|-------|---------|
| `FileSettings` | `settings.json` | User settings |
| `FileMCPConfig` | `.mcp.json` | MCP server config |
| `FileCredentials` | `.credentials.json` | OAuth credentials |
| `DirClaude` | `.claude` | Config directory |
| `DirSkills` | `skills` | Skills directory |

---

## Version Detection

```go
// Check installed CLI version against tested version
v := claudecontract.CheckVersion("claude")  // Logs warning if newer

// Parse version string
version, err := claudecontract.ParseVersion("2.1.19")

// Current tested version
const TestedCLIVersion = "2.1.19"
```

---

## Testing

| Test File | Purpose |
|-----------|---------|
| `contract_test.go` | Validates constants are non-empty |
| `flags_test.go` | Validates flag format (`--` prefix) |
| `events_test.go` | Validates event type constants |
| `paths_test.go` | Validates path constants |
| `tools_test.go` | Validates tool names |
| `real_cli_test.go` | Tests against real CLI (requires `TEST_REAL_CLI=1`) |
| `golden_test.go` | Tests against captured CLI help output |

```bash
# Unit tests
go test ./claudecontract/...

# Real CLI validation (uses installed claude binary)
TEST_REAL_CLI=1 go test ./claudecontract/... -run Real -v
```

---

## Sources

Constants derived from:
- Claude CLI `--help` output
- [TypeScript SDK docs](https://platform.claude.com/docs/en/agent-sdk/typescript)
- [Python SDK docs](https://platform.claude.com/docs/en/agent-sdk/python)
- [Hooks documentation](https://code.claude.com/docs/en/hooks)
- [CLI Reference](https://code.claude.com/docs/en/cli-reference)
