# claude

**Claude CLI wrapper with OAuth credential management.** Provides a Go interface for invoking the Claude CLI binary with full configuration support.

---

## Package Contents

| File | Purpose |
|------|---------|
| `client.go` | `Client` interface definition |
| `claude_cli.go` | `ClaudeCLI` implementation with functional options |
| `config.go` | `Config` struct, validation, env loading, `ToOptions()` |
| `factory.go` | `NewFromConfig()`, `NewFromEnv()` |
| `singleton.go` | Default client management (`GetDefaultClient`, etc.) |
| `context.go` | Context injection (`ContextWithClient`, `ClientFromContext`) |
| `credentials.go` | OAuth credential loading/validation |
| `request.go` | Request/response types |
| `errors.go` | Error types and sentinels |
| `mock.go` | `MockClient` for testing |

---

## Key Types

### Client Interface

```go
type Client interface {
    Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error)
    Stream(ctx context.Context, req CompletionRequest) (<-chan StreamChunk, error)
}
```

### ClaudeCLI

```go
type ClaudeCLI struct {
    // Configuration fields (private)
}

// Functional options pattern
client := NewCLI(
    WithModel("sonnet"),
    WithTimeout(5*time.Minute),
    WithWorkdir("/path/to/project"),
    WithHomeDir("/custom/home"),        // For containers
    WithDangerouslySkipPermissions(),   // Non-interactive mode
)
```

### Request/Response

```go
type CompletionRequest struct {
    SystemPrompt string
    Messages     []Message
    Model        string
    MaxTokens    int
    Temperature  float64
    Tools        []Tool
    Options      map[string]any
}

type CompletionResponse struct {
    Content      string
    ToolCalls    []ToolCall
    Usage        TokenUsage
    Model        string
    FinishReason string
    Duration     time.Duration
    SessionID    string   // Claude CLI specific
    CostUSD      float64  // Claude CLI specific
}
```

### Credentials

```go
type Credentials struct {
    AccessToken      string
    RefreshToken     string
    ExpiresAt        int64  // Unix timestamp (ms)
    Scopes           []string
    SubscriptionType string
    RateLimitTier    string
}

// Load from default path (~/.claude/.credentials.json)
creds, err := LoadCredentials("")

// Load from custom directory (for containers)
creds, err := LoadCredentialsFromDir("/home/worker/.claude")

// Check expiration
if creds.IsExpired() { ... }
if creds.IsExpiringSoon(10 * time.Minute) { ... }
```

---

## Usage Examples

### Basic Completion

```go
client := claude.NewCLI()

resp, err := client.Complete(ctx, claude.CompletionRequest{
    SystemPrompt: "You are a helpful assistant.",
    Messages: []claude.Message{
        {Role: claude.RoleUser, Content: "What is 2+2?"},
    },
})
if err != nil {
    log.Fatal(err)
}
fmt.Println(resp.Content)
```

### Streaming

```go
stream, err := client.Stream(ctx, claude.CompletionRequest{
    Messages: []claude.Message{
        {Role: claude.RoleUser, Content: "Tell me a story."},
    },
})
if err != nil {
    log.Fatal(err)
}

for chunk := range stream {
    if chunk.Error != nil {
        log.Fatal(chunk.Error)
    }
    fmt.Print(chunk.Content)
    if chunk.Done {
        fmt.Printf("\n\nTokens: %d\n", chunk.Usage.TotalTokens)
    }
}
```

### Container Usage

```go
// When Claude config is mounted to a custom location
client := claude.NewCLI(
    claude.WithHomeDir("/home/worker"),
    claude.WithDangerouslySkipPermissions(),
)

// Verify credentials before use
creds, err := claude.LoadCredentialsFromDir("/home/worker/.claude")
if err != nil {
    log.Fatal("credentials not available:", err)
}
if creds.IsExpired() {
    log.Fatal("credentials expired")
}
```

### Testing with Mock

```go
mock := &claude.MockClient{
    CompleteFunc: func(ctx context.Context, req claude.CompletionRequest) (*claude.CompletionResponse, error) {
        return &claude.CompletionResponse{
            Content: "Mocked response",
            Usage:   claude.TokenUsage{TotalTokens: 10},
        }, nil
    },
}

// Use mock in tests
resp, _ := mock.Complete(ctx, claude.CompletionRequest{})
```

---

## Configuration Options

| Option | Description |
|--------|-------------|
| `WithModel(name)` | Set the Claude model to use |
| `WithTimeout(dur)` | Set request timeout |
| `WithWorkdir(path)` | Set working directory for CLI |
| `WithHomeDir(path)` | Override HOME for credential discovery |
| `WithDangerouslySkipPermissions()` | Skip permission prompts (non-interactive) |
| `WithSessionID(id)` | Continue existing session |
| `WithSystemPrompt(prompt)` | Set default system prompt |
| `WithMaxBudgetUSD(amount)` | Set spending limit |
| `WithAllowedTools(tools...)` | Restrict available tools |

---

## Error Handling

```go
// Sentinel errors
var (
    ErrUnavailable    = errors.New("LLM service unavailable")
    ErrContextTooLong = errors.New("context exceeds maximum length")
    ErrRateLimited    = errors.New("rate limited")
    ErrTimeout        = errors.New("request timed out")
)

// Check error type
if errors.Is(err, claude.ErrRateLimited) {
    // Wait and retry
}

// Credential errors
var (
    ErrCredentialsNotFound = errors.New("credentials file not found")
    ErrCredentialsExpired  = errors.New("credentials expired")
)
```

---

## Notes

- Requires Claude CLI binary in PATH
- Credentials auto-discovered from `~/.claude/.credentials.json`
- For containers, mount host's `~/.claude` and use `WithHomeDir`
- Mock client provided for testing without actual CLI
