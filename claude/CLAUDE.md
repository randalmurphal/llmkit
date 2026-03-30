# claude

Claude CLI wrapper for V2. This package supports direct one-shot usage, root-registry registration, stream-json parsing, and long-running session management under [`session/`](./session/).

## Main Surfaces

| Surface | Purpose |
|---------|---------|
| `ClaudeCLI` | Direct Claude CLI client |
| `CompletionRequest` / `CompletionResponse` | One-shot request and response types |
| `StreamJSON` | Stream Claude `stream-json` events plus final result |
| `Config` | Serializable config that maps onto option functions |
| `session/` | Long-running Claude CLI sessions with `Send`, `Output`, and `WaitForInit` |
| `register.go` | V2 root registry adapter for `llmkit.New("claude", ...)` |

## Key Options

| Option | Purpose |
|--------|---------|
| `WithModel(name)` | Select model |
| `WithFallbackModel(name)` | Set fallback model |
| `WithTimeout(dur)` | Request timeout |
| `WithWorkdir(path)` | Working directory |
| `WithDangerouslySkipPermissions()` | Non-interactive trusted execution |
| `WithPermissionMode(mode)` | Permission behavior |
| `WithAllowedTools(tools)` | Allow-list tools |
| `WithDisallowedTools(tools)` | Block tools |
| `WithTools(tools)` | Set exact built-in tool set |
| `WithSystemPrompt(prompt)` | Replace system prompt |
| `WithAppendSystemPrompt(prompt)` | Append to system prompt |
| `WithJSONSchema(schema)` | Require structured output |
| `WithMCPServers(servers)` | Inline MCP server definitions |
| `WithSessionID(id)` / `WithResume(id)` | One-shot session continuity |

## Streaming

`StreamJSON` parses Claude `--output-format stream-json` output into typed events:

| Event | Meaning |
|-------|---------|
| `StreamEventInit` | Initial session metadata |
| `StreamEventAssistant` | Assistant message content blocks |
| `StreamEventUser` | Tool-result/user echo events |
| `StreamEventResult` | Final turn result with usage/cost |
| `StreamEventHook` | Hook output when enabled |

For long-running interactive sessions, use [`claude/session`](./session/), not `ClaudeCLI.StreamJSON`.

## V2 Integration

- Direct import: `github.com/randalmurphal/llmkit/v2/claude`
- Root registration: blank-import `github.com/randalmurphal/llmkit/v2/claude` or `github.com/randalmurphal/llmkit/v2/providers`
- Shared root request/response types live in `github.com/randalmurphal/llmkit/v2`

## Testing

```bash
go test ./claude/...
TEST_BEHAVIORAL=1 go test ./claude/... -run Behavioral -v -timeout 15m
go test ./claude/session/...
```

Behavioral tests use real Claude CLI execution and API credits. Golden and unit tests validate parsing and option construction without real network execution.
