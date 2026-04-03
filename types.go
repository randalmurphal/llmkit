package llmkit

import (
	"encoding/json"
	"time"

	"github.com/randalmurphal/llmkit/v2/contract"
)

// Request configures a provider-agnostic completion call using only shared fields.
type Request struct {
	SystemPrompt string          `json:"system_prompt,omitempty"`
	Messages     []Message       `json:"messages"`
	Model        string          `json:"model,omitempty"`
	MaxTokens    int             `json:"max_tokens,omitempty"`
	Temperature  float64         `json:"temperature,omitempty"`
	Tools        []Tool          `json:"tools,omitempty"`
	JSONSchema   json.RawMessage `json:"json_schema,omitempty"`
}

// Message is a conversation turn.
type Message struct {
	Role         Role          `json:"role"`
	Content      string        `json:"content"`
	Name         string        `json:"name,omitempty"`
	ContentParts []ContentPart `json:"content_parts,omitempty"`
}

// ContentPart represents a piece of multimodal content.
type ContentPart struct {
	Type        string `json:"type"`
	Text        string `json:"text,omitempty"`
	ImageURL    string `json:"image_url,omitempty"`
	ImageBase64 string `json:"image_base64,omitempty"`
	MediaType   string `json:"media_type,omitempty"`
	FilePath    string `json:"file_path,omitempty"`
}

func NewTextMessage(role Role, content string) Message {
	return Message{Role: role, Content: content}
}

func NewImageMessage(role Role, text, imageURL string) Message {
	return Message{
		Role: role,
		ContentParts: []ContentPart{
			{Type: "text", Text: text},
			{Type: "image", ImageURL: imageURL},
		},
	}
}

func NewImageBase64Message(role Role, text, base64Data, mediaType string) Message {
	return Message{
		Role: role,
		ContentParts: []ContentPart{
			{Type: "text", Text: text},
			{Type: "image", ImageBase64: base64Data, MediaType: mediaType},
		},
	}
}

func (m Message) IsMultimodal() bool {
	return len(m.ContentParts) > 0
}

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

type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
	RoleSystem    Role = "system"
)

type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

// MCPServerConfig defines an MCP server using the shared llmkit contract.
type MCPServerConfig = contract.MCPServerConfig

type SessionMetadata struct {
	Provider string          `json:"provider"`
	Data     json.RawMessage `json:"data"`
}

type Response struct {
	Content      string           `json:"content"`
	ToolCalls    []ToolCall       `json:"tool_calls,omitempty"`
	ToolResults  []ToolResult     `json:"tool_results,omitempty"`
	Usage        TokenUsage       `json:"usage"`
	Model        string           `json:"model"`
	FinishReason string           `json:"finish_reason"`
	Duration     time.Duration    `json:"duration"`
	SessionID    string           `json:"session_id,omitempty"`
	Session      *SessionMetadata `json:"session,omitempty"`
	CostUSD      float64          `json:"cost_usd,omitempty"`
	NumTurns     int              `json:"num_turns,omitempty"`
	Metadata     map[string]any   `json:"metadata,omitempty"`
}

type ToolCall struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

type ToolResult struct {
	ID       string `json:"id,omitempty"`
	Name     string `json:"name"`
	Output   string `json:"output,omitempty"`
	Status   string `json:"status,omitempty"`
	ExitCode *int   `json:"exit_code,omitempty"`
}

type TokenUsage struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	TotalTokens              int `json:"total_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens,omitempty"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens,omitempty"`
}

func (u *TokenUsage) Add(other TokenUsage) {
	u.InputTokens += other.InputTokens
	u.OutputTokens += other.OutputTokens
	u.TotalTokens += other.TotalTokens
	u.CacheCreationInputTokens += other.CacheCreationInputTokens
	u.CacheReadInputTokens += other.CacheReadInputTokens
}

type StreamChunk struct {
	Type         string           `json:"type,omitempty"`
	Content      string           `json:"content,omitempty"`
	FinalContent string           `json:"final_content,omitempty"`
	MessageID    string           `json:"message_id,omitempty"`
	Role         string           `json:"role,omitempty"`
	Model        string           `json:"model,omitempty"`
	SessionID    string           `json:"session_id,omitempty"`
	Session      *SessionMetadata `json:"session,omitempty"`
	ToolCalls    []ToolCall       `json:"tool_calls,omitempty"`
	ToolResults  []ToolResult     `json:"tool_results,omitempty"`
	Usage        *TokenUsage      `json:"usage,omitempty"`
	CostUSD      float64          `json:"cost_usd,omitempty"`
	NumTurns     int              `json:"num_turns,omitempty"`
	Done         bool             `json:"done"`
	Metadata     map[string]any   `json:"metadata,omitempty"`
	Error        error            `json:"-"`
}
