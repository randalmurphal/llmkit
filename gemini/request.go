package gemini

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

	// Model specifies which model to use (e.g., "gemini-2.5-pro").
	Model string `json:"model,omitempty"`

	// MaxTokens limits the response length.
	MaxTokens int `json:"max_tokens,omitempty"`

	// Temperature controls response randomness (0.0 = deterministic, 1.0 = creative).
	Temperature float64 `json:"temperature,omitempty"`

	// Tools lists available tools the model can invoke.
	Tools []Tool `json:"tools,omitempty"`

	// Options holds provider-specific configuration not covered by standard fields.
	Options map[string]any `json:"options,omitempty"`
}

// Message is a conversation turn.
type Message struct {
	Role    Role   `json:"role"`
	Content string `json:"content"`
	Name    string `json:"name,omitempty"` // For tool results

	// ContentParts enables multimodal content (text + images).
	// If set, takes precedence over Content.
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

	// Gemini CLI specific fields
	NumTurns int `json:"num_turns,omitempty"`
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
}

// Add calculates total tokens and adds to existing usage.
func (u *TokenUsage) Add(other TokenUsage) {
	u.InputTokens += other.InputTokens
	u.OutputTokens += other.OutputTokens
	u.TotalTokens += other.TotalTokens
}

// StreamChunk is a piece of a streaming response.
type StreamChunk struct {
	Content   string      `json:"content,omitempty"`
	ToolCalls []ToolCall  `json:"tool_calls,omitempty"`
	Usage     *TokenUsage `json:"usage,omitempty"` // Only set in final chunk
	Done      bool        `json:"done"`
	Error     error       `json:"-"` // Non-nil if streaming failed
}

// Capabilities describes what this provider natively supports.
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

	// ContextFile is the filename for project-specific context (e.g., "GEMINI.md").
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
