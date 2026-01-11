package provider

import (
	"encoding/json"
	"time"

	"github.com/randalmurphal/llmkit/claudeconfig"
)

// Request configures an LLM completion call.
// This is the provider-agnostic request format used across all CLI providers.
type Request struct {
	// SystemPrompt sets the system message that guides the model's behavior.
	SystemPrompt string `json:"system_prompt,omitempty"`

	// Messages is the conversation history to send to the model.
	Messages []Message `json:"messages"`

	// Model specifies which model to use (provider-specific name).
	// Examples: "claude-sonnet-4-20250514", "gemini-2.5-pro", "gpt-5-codex"
	Model string `json:"model,omitempty"`

	// MaxTokens limits the response length.
	MaxTokens int `json:"max_tokens,omitempty"`

	// Temperature controls response randomness (0.0 = deterministic, 1.0 = creative).
	Temperature float64 `json:"temperature,omitempty"`

	// Tools lists available tools the model can invoke.
	Tools []Tool `json:"tools,omitempty"`

	// MCP configures MCP servers to enable for this request.
	// MCP is the universal tool extension mechanism supported by all providers.
	MCP *claudeconfig.MCPConfig `json:"mcp,omitempty"`

	// Options holds provider-specific configuration not covered by standard fields.
	// See each provider's documentation for available options.
	Options map[string]any `json:"options,omitempty"`
}

// Message is a conversation turn.
// For simple text messages, use Content. For multimodal messages (images, files),
// use ContentParts instead.
type Message struct {
	Role    Role   `json:"role"`
	Content string `json:"content"`
	Name    string `json:"name,omitempty"` // For tool results

	// ContentParts enables multimodal content (text + images).
	// If set, takes precedence over Content for providers that support images.
	// Each part can be text, image (base64 or URL), or other media types.
	ContentParts []ContentPart `json:"content_parts,omitempty"`
}

// ContentPart represents a piece of multimodal content.
type ContentPart struct {
	// Type indicates the content type: "text", "image", "file"
	Type string `json:"type"`

	// Text content (when Type == "text")
	Text string `json:"text,omitempty"`

	// ImageURL for remote images (when Type == "image")
	ImageURL string `json:"image_url,omitempty"`

	// ImageBase64 for inline images (when Type == "image")
	// Format: base64-encoded image data
	ImageBase64 string `json:"image_base64,omitempty"`

	// MediaType specifies the MIME type (e.g., "image/png", "image/jpeg")
	MediaType string `json:"media_type,omitempty"`

	// FilePath for local file references (when Type == "file")
	FilePath string `json:"file_path,omitempty"`
}

// NewTextMessage creates a simple text message.
func NewTextMessage(role Role, content string) Message {
	return Message{Role: role, Content: content}
}

// NewImageMessage creates a message with an image from a URL.
func NewImageMessage(role Role, text, imageURL string) Message {
	return Message{
		Role: role,
		ContentParts: []ContentPart{
			{Type: "text", Text: text},
			{Type: "image", ImageURL: imageURL},
		},
	}
}

// NewImageBase64Message creates a message with an inline base64 image.
func NewImageBase64Message(role Role, text, base64Data, mediaType string) Message {
	return Message{
		Role: role,
		ContentParts: []ContentPart{
			{Type: "text", Text: text},
			{Type: "image", ImageBase64: base64Data, MediaType: mediaType},
		},
	}
}

// IsMultimodal returns true if the message has multimodal content.
func (m Message) IsMultimodal() bool {
	return len(m.ContentParts) > 0
}

// GetText returns the text content of the message.
// For multimodal messages, concatenates all text parts.
func (m Message) GetText() string {
	if !m.IsMultimodal() {
		return m.Content
	}
	var text string
	for _, part := range m.ContentParts {
		if part.Type == "text" {
			text += part.Text
		}
	}
	return text
}

// Role identifies the message sender.
type Role string

// Standard message roles supported across all providers.
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

// Response is the output of a completion call.
type Response struct {
	// Content is the text response from the model.
	Content string `json:"content"`

	// ToolCalls contains any tool invocations requested by the model.
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`

	// Usage tracks token consumption for this request.
	Usage TokenUsage `json:"usage"`

	// Model is the actual model used (may differ from requested).
	Model string `json:"model"`

	// FinishReason indicates why the model stopped generating.
	// Common values: "stop", "length", "tool_calls"
	FinishReason string `json:"finish_reason"`

	// Duration is the time taken for the completion.
	Duration time.Duration `json:"duration"`

	// SessionID is the session identifier (for providers that support sessions).
	// Empty if the provider doesn't support sessions.
	SessionID string `json:"session_id,omitempty"`

	// CostUSD is the estimated cost in USD (for providers that track cost).
	// Zero if cost tracking is not available.
	CostUSD float64 `json:"cost_usd,omitempty"`

	// NumTurns is the number of conversation turns (for multi-turn sessions).
	NumTurns int `json:"num_turns,omitempty"`

	// Metadata holds provider-specific response data.
	// See each provider's documentation for available fields.
	Metadata map[string]any `json:"metadata,omitempty"`
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

	// Cache-related tokens (provider-specific, may be zero)
	CacheCreationInputTokens int `json:"cache_creation_input_tokens,omitempty"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens,omitempty"`
}

// Add combines token usage from another TokenUsage.
func (u *TokenUsage) Add(other TokenUsage) {
	u.InputTokens += other.InputTokens
	u.OutputTokens += other.OutputTokens
	u.TotalTokens += other.TotalTokens
	u.CacheCreationInputTokens += other.CacheCreationInputTokens
	u.CacheReadInputTokens += other.CacheReadInputTokens
}

// StreamChunk is a piece of a streaming response.
type StreamChunk struct {
	// Content is the text content in this chunk.
	Content string `json:"content,omitempty"`

	// ToolCalls contains tool invocations (usually only in final chunks).
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`

	// Usage is the token usage (only set in final chunk).
	Usage *TokenUsage `json:"usage,omitempty"`

	// Done indicates this is the final chunk.
	Done bool `json:"done"`

	// Error is non-nil if streaming failed.
	Error error `json:"-"`
}
