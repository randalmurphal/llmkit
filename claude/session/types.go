// Package session provides long-running Claude CLI session management
// with bidirectional stream-json I/O.
package session

import (
	"encoding/json"
	"time"
)

// SessionStatus represents the current state of a session.
type SessionStatus string

// Session status constants.
const (
	StatusCreating    SessionStatus = "creating"
	StatusActive      SessionStatus = "active"
	StatusClosing     SessionStatus = "closing"
	StatusClosed      SessionStatus = "closed"
	StatusError       SessionStatus = "error"
	StatusTerminating SessionStatus = "terminating"
)

// OutputMessage represents any message received from Claude CLI stdout.
// Use Type to determine the specific message kind.
type OutputMessage struct {
	Type      string `json:"type"`              // "system", "assistant", "result", "user"
	Subtype   string `json:"subtype,omitempty"` // "init", "success", "error", "hook_response"
	SessionID string `json:"session_id"`        // Session identifier
	UUID      string `json:"uuid,omitempty"`    // Message UUID
	Raw       []byte `json:"-"`                 // Original JSON for advanced parsing

	// Populated for type="system", subtype="init"
	Init *InitMessage `json:"-"`

	// Populated for type="assistant"
	Assistant *AssistantMessage `json:"-"`

	// Populated for type="result"
	Result *ResultMessage `json:"-"`

	// Populated for type="system", subtype="hook_response"
	Hook *HookMessage `json:"-"`

	// Populated for type="user" (echoed back in some modes)
	User *UserMessage `json:"-"`
}

// InitMessage contains session initialization data.
type InitMessage struct {
	CWD               string      `json:"cwd"`
	SessionID         string      `json:"session_id"`
	Model             string      `json:"model"`
	PermissionMode    string      `json:"permissionMode"`
	Tools             []string    `json:"tools"`
	MCPServers        []MCPServer `json:"mcp_servers,omitempty"`
	SlashCommands     []string    `json:"slash_commands,omitempty"`
	Agents            []string    `json:"agents,omitempty"`
	Skills            []string    `json:"skills,omitempty"`
	ClaudeCodeVersion string      `json:"claude_code_version"`
	APIKeySource      string      `json:"apiKeySource"`
}

// MCPServer represents an MCP server connection status.
type MCPServer struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

// AssistantMessage contains Claude's response.
type AssistantMessage struct {
	Message         ClaudeMessage `json:"message"`
	ParentToolUseID string        `json:"parent_tool_use_id,omitempty"`
	SessionID       string        `json:"session_id"`
}

// ClaudeMessage is the inner message structure from Claude API.
type ClaudeMessage struct {
	ID           string         `json:"id"`
	Type         string         `json:"type"`
	Role         string         `json:"role"`
	Model        string         `json:"model"`
	Content      []ContentBlock `json:"content"`
	StopReason   *string        `json:"stop_reason"`
	StopSequence *string        `json:"stop_sequence"`
	Usage        MessageUsage   `json:"usage"`
}

// ContentBlock represents a content block in a message.
type ContentBlock struct {
	Type  string          `json:"type"`            // "text", "tool_use", "tool_result"
	Text  string          `json:"text,omitempty"`  // For text blocks
	ID    string          `json:"id,omitempty"`    // For tool_use blocks
	Name  string          `json:"name,omitempty"`  // Tool name for tool_use
	Input json.RawMessage `json:"input,omitempty"` // Tool input for tool_use
}

// MessageUsage tracks token usage for a single message.
type MessageUsage struct {
	InputTokens              int    `json:"input_tokens"`
	OutputTokens             int    `json:"output_tokens"`
	CacheCreationInputTokens int    `json:"cache_creation_input_tokens,omitempty"`
	CacheReadInputTokens     int    `json:"cache_read_input_tokens,omitempty"`
	ServiceTier              string `json:"service_tier,omitempty"`
}

// ResultMessage contains the final result of a session turn.
type ResultMessage struct {
	Subtype      string                `json:"subtype"` // "success" or "error"
	IsError      bool                  `json:"is_error"`
	Result       string                `json:"result"` // Text result
	SessionID    string                `json:"session_id"`
	DurationMS   int                   `json:"duration_ms"`
	DurationAPI  int                   `json:"duration_api_ms"`
	NumTurns     int                   `json:"num_turns"`
	TotalCostUSD float64               `json:"total_cost_usd"`
	Usage        ResultUsage           `json:"usage"`
	ModelUsage   map[string]ModelUsage `json:"modelUsage,omitempty"`
}

// ResultUsage contains aggregate token usage.
type ResultUsage struct {
	InputTokens              int    `json:"input_tokens"`
	OutputTokens             int    `json:"output_tokens"`
	CacheCreationInputTokens int    `json:"cache_creation_input_tokens,omitempty"`
	CacheReadInputTokens     int    `json:"cache_read_input_tokens,omitempty"`
	ServiceTier              string `json:"service_tier,omitempty"`
}

// ModelUsage contains per-model usage and cost.
type ModelUsage struct {
	InputTokens              int     `json:"inputTokens"`
	OutputTokens             int     `json:"outputTokens"`
	CacheReadInputTokens     int     `json:"cacheReadInputTokens,omitempty"`
	CacheCreationInputTokens int     `json:"cacheCreationInputTokens,omitempty"`
	CostUSD                  float64 `json:"costUSD"`
	ContextWindow            int     `json:"contextWindow,omitempty"`
}

// HookMessage contains hook execution output.
type HookMessage struct {
	SessionID string `json:"session_id"`
	HookName  string `json:"hook_name"`
	HookEvent string `json:"hook_event"`
	Stdout    string `json:"stdout"`
	Stderr    string `json:"stderr"`
	ExitCode  int    `json:"exit_code"`
}

// UserMessage is a message to send to Claude.
type UserMessage struct {
	Type    string `json:"type"`    // Always "user"
	Content string `json:"content"` // The user's message text
}

// NewUserMessage creates a new user message for sending to Claude.
func NewUserMessage(content string) UserMessage {
	return UserMessage{
		Type:    "user",
		Content: content,
	}
}

// SessionInfo contains metadata about a session.
type SessionInfo struct {
	ID           string        `json:"id"`
	Status       SessionStatus `json:"status"`
	Model        string        `json:"model"`
	CWD          string        `json:"cwd"`
	CreatedAt    time.Time     `json:"created_at"`
	LastActivity time.Time     `json:"last_activity"`
	TurnCount    int           `json:"turn_count"`
	TotalCostUSD float64       `json:"total_cost_usd"`
}

// parseTypedMessage parses the raw JSON and populates type-specific fields.
func (m *OutputMessage) parseTypedMessage() error {
	if m.Raw == nil {
		return nil
	}

	switch m.Type {
	case "system":
		switch m.Subtype {
		case "init":
			m.Init = &InitMessage{}
			return json.Unmarshal(m.Raw, m.Init)
		case "hook_response":
			m.Hook = &HookMessage{}
			return json.Unmarshal(m.Raw, m.Hook)
		}
	case "assistant":
		m.Assistant = &AssistantMessage{}
		return json.Unmarshal(m.Raw, m.Assistant)
	case "result":
		m.Result = &ResultMessage{}
		return json.Unmarshal(m.Raw, m.Result)
	case "user":
		m.User = &UserMessage{}
		return json.Unmarshal(m.Raw, m.User)
	}
	return nil
}

// ParseOutputMessage parses a JSON line from Claude CLI stream output.
func ParseOutputMessage(data []byte) (*OutputMessage, error) {
	var msg OutputMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}
	msg.Raw = data
	if err := msg.parseTypedMessage(); err != nil {
		return nil, err
	}
	return &msg, nil
}

// IsInit returns true if this is an initialization message.
func (m *OutputMessage) IsInit() bool {
	return m.Type == "system" && m.Subtype == "init"
}

// IsAssistant returns true if this is an assistant message.
func (m *OutputMessage) IsAssistant() bool {
	return m.Type == "assistant"
}

// IsResult returns true if this is a result message.
func (m *OutputMessage) IsResult() bool {
	return m.Type == "result"
}

// IsHook returns true if this is a hook response.
func (m *OutputMessage) IsHook() bool {
	return m.Type == "system" && m.Subtype == "hook_response"
}

// IsSuccess returns true if this is a successful result.
func (m *OutputMessage) IsSuccess() bool {
	return m.Type == "result" && m.Subtype == "success"
}

// IsError returns true if this is an error result.
func (m *OutputMessage) IsError() bool {
	return m.Type == "result" && (m.Subtype == "error" || (m.Result != nil && m.Result.IsError))
}

// GetText extracts the text content from the message.
// For assistant messages, concatenates all text blocks.
// For result messages, returns the result text.
func (m *OutputMessage) GetText() string {
	if m.Assistant != nil {
		var text string
		for _, block := range m.Assistant.Message.Content {
			if block.Type == "text" {
				text += block.Text
			}
		}
		return text
	}
	if m.Result != nil {
		return m.Result.Result
	}
	return ""
}
