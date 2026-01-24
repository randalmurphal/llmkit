package claude

import (
	"encoding/json"
	"time"
)

// CompletionRequest configures an LLM completion call.
type CompletionRequest struct {
	// SystemPrompt sets the system message that guides the model's behavior.
	SystemPrompt string `json:"system_prompt,omitempty"`

	// Messages is the conversation history to send to the model.
	Messages []Message `json:"messages"`

	// Model specifies which model to use (e.g., "claude-sonnet-4-20250514").
	Model string `json:"model,omitempty"`

	// MaxTokens limits the response length.
	MaxTokens int `json:"max_tokens,omitempty"`

	// Temperature controls response randomness (0.0 = deterministic, 1.0 = creative).
	Temperature float64 `json:"temperature,omitempty"`

	// Tools lists available tools the model can invoke.
	Tools []Tool `json:"tools,omitempty"`

	// Options holds provider-specific configuration not covered by standard fields.
	Options map[string]any `json:"options,omitempty"`

	// JSONSchema forces structured output matching the given JSON schema.
	// When set, Claude will output valid JSON conforming to this schema.
	// This overrides any client-level schema set via WithJSONSchema().
	JSONSchema string `json:"json_schema,omitempty"`

	// OnEvent is called for each streaming event during execution.
	// Use this to capture transcripts in real-time, track progress, or log activity.
	// Events include: StreamEventInit, StreamEventAssistant, StreamEventResult, StreamEventHook.
	// If nil, events are still processed internally but not exposed to caller.
	// This callback is invoked synchronously - keep handlers fast to avoid blocking.
	OnEvent func(StreamEvent) `json:"-"`
}

// Message is a conversation turn.
type Message struct {
	Role    Role   `json:"role"`
	Content string `json:"content"`
	Name    string `json:"name,omitempty"` // For tool results
}

// Role identifies the message sender.
type Role string

// Standard message roles.
const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
	RoleSystem    Role = "system"
)

// Tool defines an available tool for the LLM.
type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"` // JSON Schema
}

// CompletionResponse is the output of a completion call.
type CompletionResponse struct {
	Content      string        `json:"content"`
	ToolCalls    []ToolCall    `json:"tool_calls,omitempty"`
	Usage        TokenUsage    `json:"usage"`
	Model        string        `json:"model"`
	FinishReason string        `json:"finish_reason"`
	Duration     time.Duration `json:"duration"`

	// Claude CLI specific fields (populated when using JSON output)
	SessionID string  `json:"session_id,omitempty"`
	CostUSD   float64 `json:"cost_usd,omitempty"`
	NumTurns  int     `json:"num_turns,omitempty"`
}

// ToolCall represents a tool invocation request from the LLM.
type ToolCall struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// TokenUsage tracks token consumption.
type TokenUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`

	// Cache-related tokens (Claude specific)
	CacheCreationInputTokens int `json:"cache_creation_input_tokens,omitempty"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens,omitempty"`
}

// Add calculates total tokens and adds to existing usage.
func (u *TokenUsage) Add(other TokenUsage) {
	u.InputTokens += other.InputTokens
	u.OutputTokens += other.OutputTokens
	u.TotalTokens += other.TotalTokens
	u.CacheCreationInputTokens += other.CacheCreationInputTokens
	u.CacheReadInputTokens += other.CacheReadInputTokens
}

// Capabilities describes what a provider natively supports.
// This type mirrors provider.Capabilities for API compatibility.
type Capabilities struct {
	// Streaming indicates if the provider supports streaming responses.
	Streaming bool `json:"streaming"`

	// Tools indicates if the provider supports tool/function calling.
	Tools bool `json:"tools"`

	// MCP indicates if the provider supports MCP (Model Context Protocol) servers.
	MCP bool `json:"mcp"`

	// Sessions indicates if the provider supports multi-turn conversation sessions.
	Sessions bool `json:"sessions"`

	// Images indicates if the provider supports image inputs.
	Images bool `json:"images"`

	// NativeTools lists the provider's built-in tools by name.
	NativeTools []string `json:"native_tools"`

	// ContextFile is the filename for project-specific context (e.g., "CLAUDE.md").
	ContextFile string `json:"context_file,omitempty"`
}

// HasTool checks if a native tool is available by name.
func (c Capabilities) HasTool(name string) bool {
	for _, t := range c.NativeTools {
		if t == name {
			return true
		}
	}
	return false
}
