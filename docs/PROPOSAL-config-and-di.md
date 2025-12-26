# Proposal: Configuration Struct & Dependency Injection Patterns

**Status**: Implemented (2025-12-25)
**Author**: task-keeper integration analysis
**Date**: 2025-12-25

## Problem Statement

The current Claude client uses functional options exclusively, which is excellent for
one-off configuration but creates friction for:

1. **Serializable configuration** - Can't load from YAML/JSON/env easily
2. **Dependency injection** - No interface to mock in tests
3. **Application configuration** - Must convert struct config to options
4. **Default client patterns** - No standard way to have app-wide defaults

Current pattern requires converting config to options manually:

```go
// Current: 15+ lines for DI-friendly setup
type Config struct {
    Model        string
    SystemPrompt string
    MaxTurns     int
    // ...must mirror all options
}

func NewClient(cfg Config) claude.Client {
    opts := []claude.ClaudeOption{}
    if cfg.Model != "" {
        opts = append(opts, claude.WithModel(cfg.Model))
    }
    if cfg.SystemPrompt != "" {
        opts = append(opts, claude.WithSystemPrompt(cfg.SystemPrompt))
    }
    // ...repeat for every field
    return claude.NewClaudeCLI(opts...)
}
```

## Proposed Solution

Add a `Config` struct with a factory method, while preserving functional options:

```go
// New: Clean, serializable configuration
cfg := claude.Config{
    Model:        "claude-opus-4-5-20251101",
    SystemPrompt: "You are a helpful assistant.",
    MaxTurns:     10,
}

client := claude.NewFromConfig(cfg)
```

## Design Principles

### 1. Additive Only
- New `Config` struct and `NewFromConfig()` factory
- Existing `NewClaudeCLI()` with options unchanged
- Full backward compatibility

### 2. Config-First, Options-Compatible
- `Config` can be extended with options: `NewFromConfig(cfg, opts...)`
- Options override config values
- Best of both worlds

### 3. Environment Variable Loading
- Opt-in `LoadFromEnv()` method
- Standard `CLAUDE_` prefix
- Explicit, not magic

### 4. Interface for Mocking
- Export `Client` interface (already exists internally)
- Enable test doubles without import cycles

---

## Detailed Design

### Client Interface

Location: `claude/client.go` (additions to existing file)

```go
package claude

import "context"

// Client provides LLM completion and streaming capabilities.
// This interface enables dependency injection and testing.
type Client interface {
    // Complete sends a completion request and returns the response.
    Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error)

    // Stream sends a completion request and returns a channel of response chunks.
    // The channel is closed when streaming completes or an error occurs.
    // Check StreamChunk.Error for errors, StreamChunk.Done for completion.
    Stream(ctx context.Context, req CompletionRequest) (<-chan StreamChunk, error)
}

// Compile-time verification that ClaudeCLI implements Client.
var _ Client = (*ClaudeCLI)(nil)
```

### Config Struct

Location: `claude/config.go` (new file)

```go
package claude

import (
    "os"
    "strconv"
    "time"
)

// Config holds configuration for a Claude client.
// Zero values use sensible defaults where noted.
type Config struct {
    // --- Model Selection ---

    // Model is the primary model to use.
    // Default: "claude-sonnet-4-20250514"
    Model string `json:"model" yaml:"model" mapstructure:"model"`

    // FallbackModel is used when primary model is unavailable.
    // Optional.
    FallbackModel string `json:"fallback_model" yaml:"fallback_model" mapstructure:"fallback_model"`

    // --- Prompts ---

    // SystemPrompt is the system message prepended to all requests.
    // Optional.
    SystemPrompt string `json:"system_prompt" yaml:"system_prompt" mapstructure:"system_prompt"`

    // AppendSystemPrompt is appended to the existing system prompt.
    // Use when you want to add to defaults rather than replace.
    AppendSystemPrompt string `json:"append_system_prompt" yaml:"append_system_prompt" mapstructure:"append_system_prompt"`

    // --- Execution Limits ---

    // MaxTurns limits conversation turns (tool calls + responses).
    // 0 means no limit. Default: 10.
    MaxTurns int `json:"max_turns" yaml:"max_turns" mapstructure:"max_turns"`

    // Timeout is the maximum duration for a completion request.
    // 0 uses the default (5 minutes).
    Timeout time.Duration `json:"timeout" yaml:"timeout" mapstructure:"timeout"`

    // MaxBudgetUSD limits spending per request.
    // 0 means no limit.
    MaxBudgetUSD float64 `json:"max_budget_usd" yaml:"max_budget_usd" mapstructure:"max_budget_usd"`

    // --- Working Directory ---

    // WorkDir is the working directory for file operations.
    // Default: current directory.
    WorkDir string `json:"work_dir" yaml:"work_dir" mapstructure:"work_dir"`

    // --- Tool Control ---

    // AllowedTools limits which tools Claude can use.
    // Empty means all tools allowed.
    AllowedTools []string `json:"allowed_tools" yaml:"allowed_tools" mapstructure:"allowed_tools"`

    // DisallowedTools explicitly blocks certain tools.
    // Takes precedence over AllowedTools.
    DisallowedTools []string `json:"disallowed_tools" yaml:"disallowed_tools" mapstructure:"disallowed_tools"`

    // DangerouslySkipPermissions bypasses permission prompts.
    // Use with extreme caution, only in trusted environments.
    DangerouslySkipPermissions bool `json:"dangerously_skip_permissions" yaml:"dangerously_skip_permissions" mapstructure:"dangerously_skip_permissions"`

    // --- Session Management ---

    // SessionID enables session persistence with this ID.
    // Optional.
    SessionID string `json:"session_id" yaml:"session_id" mapstructure:"session_id"`

    // Continue resumes the last session.
    Continue bool `json:"continue" yaml:"continue" mapstructure:"continue"`

    // Resume resumes a specific session by ID.
    Resume string `json:"resume" yaml:"resume" mapstructure:"resume"`

    // NoSessionPersistence disables session saving.
    NoSessionPersistence bool `json:"no_session_persistence" yaml:"no_session_persistence" mapstructure:"no_session_persistence"`

    // --- Container Environment ---

    // HomeDir overrides the home directory (for containers).
    HomeDir string `json:"home_dir" yaml:"home_dir" mapstructure:"home_dir"`

    // ConfigDir overrides the .claude config directory.
    ConfigDir string `json:"config_dir" yaml:"config_dir" mapstructure:"config_dir"`

    // Env provides additional environment variables.
    Env map[string]string `json:"env" yaml:"env" mapstructure:"env"`

    // --- Advanced ---

    // ClaudePath is the path to the claude CLI binary.
    // Default: "claude" (found via PATH).
    ClaudePath string `json:"claude_path" yaml:"claude_path" mapstructure:"claude_path"`

    // OutputFormat controls CLI output format.
    // Default: "json".
    OutputFormat string `json:"output_format" yaml:"output_format" mapstructure:"output_format"`
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
    return Config{
        Model:        "claude-sonnet-4-20250514",
        MaxTurns:     10,
        Timeout:      5 * time.Minute,
        OutputFormat: "json",
    }
}

// LoadFromEnv populates config fields from environment variables.
// Environment variables use CLAUDE_ prefix and take precedence.
func (c *Config) LoadFromEnv() {
    if v := os.Getenv("CLAUDE_MODEL"); v != "" {
        c.Model = v
    }
    if v := os.Getenv("CLAUDE_FALLBACK_MODEL"); v != "" {
        c.FallbackModel = v
    }
    if v := os.Getenv("CLAUDE_SYSTEM_PROMPT"); v != "" {
        c.SystemPrompt = v
    }
    if v := os.Getenv("CLAUDE_MAX_TURNS"); v != "" {
        if n, err := strconv.Atoi(v); err == nil {
            c.MaxTurns = n
        }
    }
    if v := os.Getenv("CLAUDE_TIMEOUT"); v != "" {
        if d, err := time.ParseDuration(v); err == nil {
            c.Timeout = d
        }
    }
    if v := os.Getenv("CLAUDE_WORK_DIR"); v != "" {
        c.WorkDir = v
    }
    if v := os.Getenv("CLAUDE_MAX_BUDGET_USD"); v != "" {
        if f, err := strconv.ParseFloat(v, 64); err == nil {
            c.MaxBudgetUSD = f
        }
    }
    if v := os.Getenv("CLAUDE_PATH"); v != "" {
        c.ClaudePath = v
    }
    if v := os.Getenv("CLAUDE_HOME_DIR"); v != "" {
        c.HomeDir = v
    }
    if v := os.Getenv("CLAUDE_CONFIG_DIR"); v != "" {
        c.ConfigDir = v
    }
    if v := os.Getenv("CLAUDE_SKIP_PERMISSIONS"); v == "true" || v == "1" {
        c.DangerouslySkipPermissions = true
    }
}

// FromEnv creates a Config from environment variables with defaults.
func FromEnv() Config {
    cfg := DefaultConfig()
    cfg.LoadFromEnv()
    return cfg
}

// Validate checks if the configuration is valid.
func (c *Config) Validate() error {
    // Model is the only truly required field
    if c.Model == "" {
        return fmt.Errorf("Model is required")
    }
    // MaxTurns must be non-negative
    if c.MaxTurns < 0 {
        return fmt.Errorf("MaxTurns must be >= 0")
    }
    // MaxBudgetUSD must be non-negative
    if c.MaxBudgetUSD < 0 {
        return fmt.Errorf("MaxBudgetUSD must be >= 0")
    }
    return nil
}

// ToOptions converts the config to functional options.
// This enables mixing Config with additional options.
func (c *Config) ToOptions() []ClaudeOption {
    opts := make([]ClaudeOption, 0, 16)

    if c.Model != "" {
        opts = append(opts, WithModel(c.Model))
    }
    if c.FallbackModel != "" {
        opts = append(opts, WithFallbackModel(c.FallbackModel))
    }
    if c.SystemPrompt != "" {
        opts = append(opts, WithSystemPrompt(c.SystemPrompt))
    }
    if c.AppendSystemPrompt != "" {
        opts = append(opts, WithAppendSystemPrompt(c.AppendSystemPrompt))
    }
    if c.MaxTurns > 0 {
        opts = append(opts, WithMaxTurns(c.MaxTurns))
    }
    if c.Timeout > 0 {
        opts = append(opts, WithTimeout(c.Timeout))
    }
    if c.MaxBudgetUSD > 0 {
        opts = append(opts, WithMaxBudgetUSD(c.MaxBudgetUSD))
    }
    if c.WorkDir != "" {
        opts = append(opts, WithWorkdir(c.WorkDir))
    }
    if len(c.AllowedTools) > 0 {
        opts = append(opts, WithAllowedTools(c.AllowedTools))
    }
    if len(c.DisallowedTools) > 0 {
        opts = append(opts, WithDisallowedTools(c.DisallowedTools))
    }
    if c.DangerouslySkipPermissions {
        opts = append(opts, WithDangerouslySkipPermissions())
    }
    if c.SessionID != "" {
        opts = append(opts, WithSessionID(c.SessionID))
    }
    if c.Continue {
        opts = append(opts, WithContinue())
    }
    if c.Resume != "" {
        opts = append(opts, WithResume(c.Resume))
    }
    if c.NoSessionPersistence {
        opts = append(opts, WithNoSessionPersistence())
    }
    if c.HomeDir != "" {
        opts = append(opts, WithHomeDir(c.HomeDir))
    }
    if c.ConfigDir != "" {
        opts = append(opts, WithConfigDir(c.ConfigDir))
    }
    if len(c.Env) > 0 {
        opts = append(opts, WithEnv(c.Env))
    }
    if c.ClaudePath != "" {
        opts = append(opts, WithClaudePath(c.ClaudePath))
    }
    if c.OutputFormat != "" {
        opts = append(opts, WithOutputFormat(c.OutputFormat))
    }

    return opts
}
```

### Factory Functions

Location: `claude/factory.go` (new file)

```go
package claude

// NewFromConfig creates a Client from a Config struct.
// Additional options can be provided to override config values.
func NewFromConfig(cfg Config, opts ...ClaudeOption) Client {
    // Convert config to options
    configOpts := cfg.ToOptions()

    // Combine: config options first, then explicit overrides
    allOpts := make([]ClaudeOption, 0, len(configOpts)+len(opts))
    allOpts = append(allOpts, configOpts...)
    allOpts = append(allOpts, opts...)

    return NewClaudeCLI(allOpts...)
}

// NewFromEnv creates a Client configured from environment variables.
// Additional options can be provided to override env values.
func NewFromEnv(opts ...ClaudeOption) Client {
    cfg := FromEnv()
    return NewFromConfig(cfg, opts...)
}
```

### Singleton Pattern (Optional)

Location: `claude/singleton.go` (new file)

```go
package claude

import (
    "sync"
)

// Default client management for applications that want a global client.
// This is optional - applications can manage their own client lifecycle.

var (
    defaultClient     Client
    defaultClientOnce sync.Once
    defaultClientMu   sync.RWMutex
    defaultConfig     = DefaultConfig()
)

// SetDefaultConfig sets the configuration for the default client.
// Must be called before GetDefaultClient is first called.
// Not thread-safe with concurrent GetDefaultClient calls.
func SetDefaultConfig(cfg Config) {
    defaultClientMu.Lock()
    defer defaultClientMu.Unlock()
    defaultConfig = cfg
}

// GetDefaultClient returns a singleton default client.
// Creates the client lazily on first call using the default config.
// Thread-safe for concurrent access.
func GetDefaultClient() Client {
    defaultClientMu.RLock()
    if defaultClient != nil {
        defer defaultClientMu.RUnlock()
        return defaultClient
    }
    defaultClientMu.RUnlock()

    defaultClientOnce.Do(func() {
        defaultClientMu.Lock()
        defer defaultClientMu.Unlock()
        defaultClient = NewFromConfig(defaultConfig)
    })

    defaultClientMu.RLock()
    defer defaultClientMu.RUnlock()
    return defaultClient
}

// SetDefaultClient sets the singleton client directly.
// Useful for testing or when you want to manage the client lifecycle.
func SetDefaultClient(c Client) {
    defaultClientMu.Lock()
    defer defaultClientMu.Unlock()
    defaultClient = c
    // Mark as initialized so GetDefaultClient won't recreate
    defaultClientOnce.Do(func() {})
}

// ResetDefaultClient clears the singleton client.
// Useful for testing to ensure clean state between tests.
func ResetDefaultClient() {
    defaultClientMu.Lock()
    defer defaultClientMu.Unlock()
    defaultClient = nil
    defaultClientOnce = sync.Once{}
}
```

### Context Injection

Location: `claude/context.go` (new file)

```go
package claude

import "context"

type contextKey struct{ name string }

var clientContextKey = &contextKey{"claude-client"}

// ContextWithClient adds a Client to a context.
func ContextWithClient(ctx context.Context, c Client) context.Context {
    return context.WithValue(ctx, clientContextKey, c)
}

// ClientFromContext retrieves a Client from a context.
// Returns nil if no Client is present.
func ClientFromContext(ctx context.Context) Client {
    if c, ok := ctx.Value(clientContextKey).(Client); ok {
        return c
    }
    return nil
}

// MustClientFromContext retrieves a Client or panics.
// Use when client is required and missing is a programming error.
func MustClientFromContext(ctx context.Context) Client {
    c := ClientFromContext(ctx)
    if c == nil {
        panic("claude.Client not found in context")
    }
    return c
}
```

---

## Usage Examples

### Basic Usage with Config

```go
package main

import (
    "context"
    "log"

    "github.com/randalmurphal/llmkit/claude"
)

func main() {
    cfg := claude.Config{
        Model:        "claude-opus-4-5-20251101",
        SystemPrompt: "You are a code reviewer.",
        MaxTurns:     5,
    }

    client := claude.NewFromConfig(cfg)

    resp, err := client.Complete(context.Background(), claude.CompletionRequest{
        Messages: []claude.Message{{Role: "user", Content: "Review this code..."}},
    })
    if err != nil {
        log.Fatal(err)
    }

    log.Println(resp.Content)
}
```

### Environment-Based Configuration

```go
// Set environment variables:
// CLAUDE_MODEL=claude-opus-4-5-20251101
// CLAUDE_MAX_TURNS=20
// CLAUDE_WORK_DIR=/app

client := claude.NewFromEnv()
```

### Config with Option Overrides

```go
cfg := claude.DefaultConfig()
cfg.Model = "claude-sonnet-4-20250514"

// Override specific settings with options
client := claude.NewFromConfig(cfg,
    claude.WithMaxTurns(3),  // Override config's MaxTurns
    claude.WithDangerouslySkipPermissions(),
)
```

### Loading from YAML

```go
import "gopkg.in/yaml.v3"

var cfgYAML = `
model: claude-opus-4-5-20251101
system_prompt: You are a helpful assistant.
max_turns: 10
allowed_tools:
  - Read
  - Bash
`

var cfg claude.Config
if err := yaml.Unmarshal([]byte(cfgYAML), &cfg); err != nil {
    log.Fatal(err)
}

client := claude.NewFromConfig(cfg)
```

### Using Default Client

```go
// In main.go or init
claude.SetDefaultConfig(claude.Config{
    Model:    "claude-opus-4-5-20251101",
    MaxTurns: 10,
})

// Anywhere else in the application
client := claude.GetDefaultClient()
resp, _ := client.Complete(ctx, req)
```

### Testing with Mock

```go
func TestMyHandler(t *testing.T) {
    // Create mock client
    mock := claude.NewMockClient("mocked response")

    // Inject via context
    ctx := claude.ContextWithClient(context.Background(), mock)

    // Or inject via singleton
    claude.SetDefaultClient(mock)
    defer claude.ResetDefaultClient()

    // Test your code
    result := MyHandler(ctx)

    // Verify
    if mock.CallCount() != 1 {
        t.Errorf("expected 1 call, got %d", mock.CallCount())
    }
}
```

### Application Setup Pattern

```go
package main

import (
    "context"
    "log"
    "os"

    "github.com/randalmurphal/llmkit/claude"
)

func main() {
    // Load config with environment overrides
    cfg := claude.DefaultConfig()
    cfg.LoadFromEnv()

    // Apply application-specific overrides
    if os.Getenv("APP_ENV") == "production" {
        cfg.Model = "claude-opus-4-5-20251101"
        cfg.MaxBudgetUSD = 10.0
    }

    // Validate before use
    if err := cfg.Validate(); err != nil {
        log.Fatalf("Invalid config: %v", err)
    }

    // Create client
    client := claude.NewFromConfig(cfg)

    // Set as default for easy access
    claude.SetDefaultClient(client)

    // Run application
    runApp()
}

func runApp() {
    client := claude.GetDefaultClient()
    // Use client...
}
```

---

## API Summary

### New Types

| Type | Purpose |
|------|---------|
| `Client` | Interface for dependency injection |
| `Config` | Struct-based configuration |

### New Functions

| Function | Purpose |
|----------|---------|
| `DefaultConfig()` | Returns config with sensible defaults |
| `FromEnv()` | Creates config from environment variables |
| `NewFromConfig(cfg, opts...)` | Creates client from config |
| `NewFromEnv(opts...)` | Creates client from environment |
| `GetDefaultClient()` | Returns singleton client |
| `SetDefaultClient(c)` | Sets singleton (for testing) |
| `SetDefaultConfig(cfg)` | Sets singleton config |
| `ResetDefaultClient()` | Clears singleton (for testing) |
| `ContextWithClient(ctx, c)` | Adds client to context |
| `ClientFromContext(ctx)` | Gets client from context |

### Config Methods

| Method | Purpose |
|--------|---------|
| `LoadFromEnv()` | Loads from CLAUDE_* env vars |
| `Validate()` | Validates configuration |
| `ToOptions()` | Converts to functional options |

---

## Implementation Plan

### Phase 1: Core Config (DONE)
- [x] `Client` interface export (already existed)
- [x] `Config` struct with all fields (config.go)
- [x] `DefaultConfig()` function (config.go)
- [x] `Validate()` method (config.go)
- [x] `ToOptions()` method (config.go)

### Phase 2: Factory Functions (DONE)
- [x] `NewFromConfig()` (factory.go)
- [x] `LoadFromEnv()` (config.go)
- [x] `FromEnv()` (config.go)
- [x] `NewFromEnv()` (factory.go)

### Phase 3: Singleton Pattern (DONE)
- [x] `GetDefaultClient()` (singleton.go)
- [x] `SetDefaultClient()` (singleton.go)
- [x] `SetDefaultConfig()` (singleton.go)
- [x] `ResetDefaultClient()` (singleton.go)

### Phase 4: Context Injection (DONE)
- [x] `ContextWithClient()` (context.go)
- [x] `ClientFromContext()` (context.go)
- [x] `MustClientFromContext()` (context.go)

### Phase 5: Documentation (DONE)
- [x] Update CLAUDE.md
- [x] Add examples (in CLAUDE.md)
- [ ] Update README (optional)

---

## Migration Path

### For Existing Users

No changes required. `NewClaudeCLI()` with options unchanged.

### For task-keeper

Replace `internal/claude/` wrapper:

```go
// Old: internal wrapper with own Config type
import tkclaude "github.com/randalmurphal/task-keeper/internal/claude"
client := tkclaude.NewClient(tkclaude.Config{...})

// New: direct use
import "github.com/randalmurphal/llmkit/claude"
client := claude.NewFromConfig(claude.Config{...})
```

---

## Alternatives Considered

### 1. Only Interface, No Config

Export `Client` interface but leave config to users.

**Rejected because:**
- Config struct is the main pain point
- Interface alone doesn't solve serialization

### 2. Viper Integration

Add direct Viper support: `claude.LoadViper(v *viper.Viper)`

**Rejected because:**
- Adds dependency
- Easy to bridge with existing `LoadFromEnv()`
- Users with Viper can unmarshal to Config themselves

### 3. Builder Pattern

```go
client := claude.NewBuilder().
    Model("opus").
    MaxTurns(10).
    Build()
```

**Rejected because:**
- Not serializable
- Duplicates options pattern
- Doesn't solve config storage

---

## Open Questions

1. **Should `Config` support JSON tags?**
   - Current: Yes, for flexibility
   - Alternative: Separate ConfigJSON type

2. **Should singleton be in separate package?**
   - Current: Same package for convenience
   - Alternative: `claude/global` or `claude/singleton` package

3. **Required vs optional fields?**
   - Current: Only Model is required
   - Alternative: More strict validation

---

## References

- task-keeper integration: `internal/claude/client.go`
- Current llmkit API: `claude/claude.go`, `claude/options.go`
- Mock client: `claude/mock.go`
