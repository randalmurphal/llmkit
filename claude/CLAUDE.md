# claude

**Claude CLI wrapper with streaming event parsing and OAuth credential management.** Provides a Go interface for invoking the Claude CLI binary with full configuration support.

---

## Package Contents

| File | Purpose | Key Types |
|------|---------|-----------|
| `client.go` | Client interface | `Client` |
| `claude_cli.go` | CLI implementation | `ClaudeCLI`, `NewClaudeCLI()` |
| `stream_types.go` | Streaming event types | `StreamEvent`, `InitEvent`, `AssistantEvent`, `UserEvent`, `ResultEvent` |
| `stream.go` | Stream accumulation | `StreamAccumulator` |
| `config.go` | Configuration | `Config`, `FromEnv()`, `Validate()` |
| `factory.go` | Client factories | `NewFromConfig()`, `NewFromEnv()` |
| `singleton.go` | Default client | `GetDefaultClient()`, `SetDefaultClient()` |
| `context.go` | Context injection | `ContextWithClient()`, `ClientFromContext()` |
| `credentials.go` | OAuth credentials | `Credentials`, `LoadCredentials()` |
| `request.go` | Request/response | `CompletionRequest`, `CompletionResponse` |
| `errors.go` | Error types | `ErrRateLimited`, `ErrTimeout` |
| `mock.go` | Test mock | `MockClient` |

---

## Streaming Events (stream_types.go)

Claude CLI `--output-format stream-json` emits JSONL with these event types:

| Event Type | Struct | When Emitted |
|------------|--------|--------------|
| `init` | `InitEvent` | Session start, contains tools/model/session_id |
| `assistant` | `AssistantEvent` | Claude responses with content blocks |
| `user` | `UserEvent` | Tool results after execution |
| `result` | `ResultEvent` | Final result with usage/cost |
| `hook` | `HookEvent` | Hook script output |

### Key Streaming Types

```go
type StreamEvent struct {
    Type      StreamEventType  // init, assistant, user, result, hook, error
    SessionID string
    Init      *InitEvent       // When Type == StreamEventInit
    Assistant *AssistantEvent  // When Type == StreamEventAssistant
    User      *UserEvent       // When Type == StreamEventUser
    Result    *ResultEvent     // When Type == StreamEventResult
    Hook      *HookEvent       // When Type == StreamEventHook
    Raw       json.RawMessage  // Original JSON for advanced parsing
}

type UserEvent struct {
    SessionID       string
    Message         UserEventMessage
    ParentToolUseID *string           // For subagent results
    ToolUseResultRaw json.RawMessage  // String (error) or object (result)
}

// Access tool result (handles string vs object)
result := userEvent.GetToolUseResult()  // *ToolUseResult or nil
errStr := userEvent.GetToolUseResultError()  // string if error
```

### Content Blocks

```go
type ContentBlock struct {
    Type  string          // "text", "tool_use", "tool_result"
    Text  string          // For text blocks
    ID    string          // For tool_use
    Name  string          // Tool name for tool_use
    Input json.RawMessage // Tool input for tool_use
}
```

---

## Configuration Options

| Option | Description |
|--------|-------------|
| `WithModel(name)` | Model selection |
| `WithTimeout(dur)` | Request timeout |
| `WithWorkdir(path)` | Working directory |
| `WithHomeDir(path)` | Override HOME (containers) |
| `WithDangerouslySkipPermissions()` | Non-interactive mode |
| `WithSessionID(id)` | Continue existing session |
| `WithSystemPrompt(prompt)` | System prompt |
| `WithAppendSystemPrompt(prompt)` | Append to system prompt |
| `WithMaxBudgetUSD(amount)` | Spending limit |
| `WithMaxTurns(n)` | Conversation turn limit |
| `WithAllowedTools(tools...)` | Restrict tools |
| `WithDisallowedTools(tools...)` | Block specific tools |
| `WithPermissionMode(mode)` | Permission behavior |
| `WithJSONSchema(schema)` | Structured output |

---

## Testing

### Test Files

| File | Purpose | Run With |
|------|---------|----------|
| `behavioral_test.go` | Real CLI validation (26 tests) | `TEST_BEHAVIORAL=1 go test -run Behavioral` |
| `golden_test.go` | Parsing validation against captured output | `go test -run Golden` |
| `mock_test.go` | Mock client tests | `go test -run Mock` |
| `internal_test.go` | Unit tests | `go test` |

### Behavioral Tests (behavioral_test.go)

Tests real CLI behavior. **Uses API credits.** Run intentionally:

```bash
TEST_BEHAVIORAL=1 go test ./claude/... -run Behavioral -v -timeout 15m
```

| Test | What It Verifies |
|------|------------------|
| `TestBehavioralMaxTurns` | `--max-turns` limits turns |
| `TestBehavioralAllowedTools` | `--allowedTools` restricts usage |
| `TestBehavioralJSONSchema` | `--json-schema` structured output |
| `TestBehavioralSessionContinuity` | `--resume` continues sessions |
| `TestBehavioralSubagentExecution` | Task tool spawns subagents |
| `TestBehavioralSystemPrompt` | `--system-prompt` replaces prompt |
| `TestBehavioralCostTracking` | Cost reported in result |
| ... | (26 total tests) |

**Design**: Tests FAIL on parse errors or unexpected CLI behavior. No silent warnings.

### Golden Tests (golden_test.go)

Tests event parsing against `testdata/golden/*.jsonl`:

```bash
go test ./claude/... -run Golden -v
```

---

## Error Handling

```go
// Sentinel errors
var (
    ErrUnavailable    = errors.New("LLM service unavailable")
    ErrRateLimited    = errors.New("rate limited")
    ErrTimeout        = errors.New("request timed out")
)

// Credential errors
var (
    ErrCredentialsNotFound = errors.New("credentials file not found")
    ErrCredentialsExpired  = errors.New("credentials expired")
)
```

---

## Dependencies

- Imports `claudecontract/` for flag and event constants
- Imports `provider/` for unified interface registration
- No external dependencies (stdlib only)
