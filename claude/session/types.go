// Package session provides long-running Claude CLI session management
// with bidirectional stream-json I/O.
package session

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/randalmurphal/llmkit/claudecontract"
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

// UserMessage is a message to send to Claude via stream-json input.
type UserMessage struct {
	Type    string             `json:"type"`    // Always "user"
	Message UserMessageContent `json:"message"` // The message content
}

// UserMessageContent contains the actual message content.
type UserMessageContent struct {
	Role    string `json:"role"`    // Always "user"
	Content string `json:"content"` // The user's message text
}

// NewUserMessage creates a new user message for sending to Claude.
func NewUserMessage(content string) UserMessage {
	return UserMessage{
		Type: "user",
		Message: UserMessageContent{
			Role:    "user",
			Content: content,
		},
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
	case claudecontract.EventTypeSystem:
		switch m.Subtype {
		case claudecontract.SubtypeInit:
			m.Init = &InitMessage{}
			return json.Unmarshal(m.Raw, m.Init)
		case claudecontract.SubtypeHookResponse:
			m.Hook = &HookMessage{}
			return json.Unmarshal(m.Raw, m.Hook)
		}
	case claudecontract.EventTypeAssistant:
		m.Assistant = &AssistantMessage{}
		return json.Unmarshal(m.Raw, m.Assistant)
	case claudecontract.EventTypeResult:
		m.Result = &ResultMessage{}
		return json.Unmarshal(m.Raw, m.Result)
	case claudecontract.EventTypeUser:
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
	return m.Type == claudecontract.EventTypeSystem && m.Subtype == claudecontract.SubtypeInit
}

// IsAssistant returns true if this is an assistant message.
func (m *OutputMessage) IsAssistant() bool {
	return m.Type == claudecontract.EventTypeAssistant
}

// IsResult returns true if this is a result message.
func (m *OutputMessage) IsResult() bool {
	return m.Type == claudecontract.EventTypeResult
}

// IsHook returns true if this is a hook response.
func (m *OutputMessage) IsHook() bool {
	return m.Type == claudecontract.EventTypeSystem && m.Subtype == claudecontract.SubtypeHookResponse
}

// IsSuccess returns true if this is a successful result.
func (m *OutputMessage) IsSuccess() bool {
	return m.Type == claudecontract.EventTypeResult && m.Subtype == claudecontract.ResultSubtypeSuccess
}

// IsError returns true if this is an error result.
// Error subtypes include: error_max_turns, error_during_execution, error_max_budget_usd, etc.
func (m *OutputMessage) IsError() bool {
	return m.Type == claudecontract.EventTypeResult && (strings.HasPrefix(m.Subtype, "error") || (m.Result != nil && m.Result.IsError))
}

// GetText extracts the text content from the message.
// For assistant messages, concatenates all text blocks.
// For result messages, returns the result text.
func (m *OutputMessage) GetText() string {
	if m.Assistant != nil {
		var sb strings.Builder
		for _, block := range m.Assistant.Message.Content {
			if block.Type == claudecontract.ContentTypeText {
				sb.WriteString(block.Text)
			}
		}
		return sb.String()
	}
	if m.Result != nil {
		return m.Result.Result
	}
	return ""
}

// =============================================================================
// JSONL File Types
// =============================================================================
//
// These types represent Claude Code's persisted session format written to:
//   ~/.claude/projects/{normalized-path}/{sessionId}.jsonl
//
// Each line is a JSON object with a "type" field determining the structure.
// This is different from the streaming OutputMessage format above.

// JSONLMessage represents a single line in Claude Code's session JSONL file.
// The file contains a sequence of these messages representing the full session history.
type JSONLMessage struct {
	Type       string              `json:"type"`                 // "user", "assistant", "queue-operation"
	Timestamp  string              `json:"timestamp"`            // ISO 8601 format
	SessionID  string              `json:"sessionId"`            // Session identifier (UUID)
	UUID       string              `json:"uuid"`                 // Unique message identifier
	ParentUUID *string             `json:"parentUuid,omitempty"` // Links to parent message (threading)
	Message    *JSONLMessageBody   `json:"message,omitempty"`    // Present for user/assistant types
	ToolResult *JSONLToolResult    `json:"toolUseResult,omitempty"` // Present when message contains tool results

	// Raw JSON for advanced parsing
	Raw json.RawMessage `json:"-"`
}

// JSONLMessageBody contains the actual message content in a JSONL entry.
type JSONLMessageBody struct {
	Role    string          `json:"role"`              // "user" or "assistant"
	Content json.RawMessage `json:"content"`           // Array of content blocks (text, tool_use, tool_result)
	Model   string          `json:"model,omitempty"`   // Model used (assistant messages only)
	Usage   *JSONLUsage     `json:"usage,omitempty"`   // Token usage (assistant messages only)
}

// JSONLUsage contains per-message token usage from JSONL files.
type JSONLUsage struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens,omitempty"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens,omitempty"`
}

// JSONLToolResult contains metadata about tool execution results in JSONL files.
// This appears alongside the message when tools were executed.
type JSONLToolResult struct {
	DurationMs int64 `json:"durationMs,omitempty"` // Tool execution time

	// TodoWrite-specific fields for progress tracking
	OldTodos []TodoItem `json:"oldTodos,omitempty"` // Previous todo state
	NewTodos []TodoItem `json:"newTodos,omitempty"` // Updated todo state
}

// TodoItem represents a single item from Claude's TodoWrite tool.
// These are extracted from JSONL files for progress tracking during execution.
type TodoItem struct {
	Content    string `json:"content"`     // Task description (imperative form)
	Status     string `json:"status"`      // "pending", "in_progress", "completed"
	ActiveForm string `json:"activeForm"`  // Present continuous form shown during execution
}

// JSONLContentBlock represents a content block in JSONL message content.
// This is used when fully parsing the content array.
type JSONLContentBlock struct {
	Type      string          `json:"type"`                 // "text", "tool_use", "tool_result"
	Text      string          `json:"text,omitempty"`       // For text blocks
	ID        string          `json:"id,omitempty"`         // For tool_use blocks
	Name      string          `json:"name,omitempty"`       // Tool name for tool_use
	Input     json.RawMessage `json:"input,omitempty"`      // Tool input for tool_use
	ToolUseID string          `json:"tool_use_id,omitempty"` // For tool_result blocks
	Content   json.RawMessage `json:"content,omitempty"`    // Tool result content (tool_result blocks)
	IsError   bool            `json:"is_error,omitempty"`   // Whether tool execution failed
}

// ParseJSONLMessage parses a single line from a Claude Code JSONL file.
func ParseJSONLMessage(data []byte) (*JSONLMessage, error) {
	var msg JSONLMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}
	msg.Raw = data
	return &msg, nil
}

// IsUser returns true if this is a user message.
// Returns false for nil receivers.
func (m *JSONLMessage) IsUser() bool {
	return m != nil && m.Type == claudecontract.EventTypeUser
}

// IsAssistant returns true if this is an assistant message.
// Returns false for nil receivers.
func (m *JSONLMessage) IsAssistant() bool {
	return m != nil && m.Type == claudecontract.EventTypeAssistant
}

// GetModel returns the model used for this message.
// Only assistant messages have model information.
func (m *JSONLMessage) GetModel() string {
	if m.Message != nil {
		return m.Message.Model
	}
	return ""
}

// GetUsage returns token usage for this message.
// Only assistant messages have usage information.
func (m *JSONLMessage) GetUsage() *JSONLUsage {
	if m.Message != nil {
		return m.Message.Usage
	}
	return nil
}

// GetContentBlocks parses and returns the content blocks from the message.
// Returns nil if the message has no content or parsing fails.
func (m *JSONLMessage) GetContentBlocks() []JSONLContentBlock {
	if m.Message == nil || len(m.Message.Content) == 0 {
		return nil
	}
	var blocks []JSONLContentBlock
	if err := json.Unmarshal(m.Message.Content, &blocks); err != nil {
		return nil
	}
	return blocks
}

// GetText extracts concatenated text from all text content blocks.
func (m *JSONLMessage) GetText() string {
	blocks := m.GetContentBlocks()
	if blocks == nil {
		return ""
	}
	var sb strings.Builder
	for _, b := range blocks {
		if b.Type == "text" {
			sb.WriteString(b.Text)
		}
	}
	return sb.String()
}

// GetToolCalls returns all tool_use blocks from the message.
func (m *JSONLMessage) GetToolCalls() []JSONLContentBlock {
	blocks := m.GetContentBlocks()
	if blocks == nil {
		return nil
	}
	var tools []JSONLContentBlock
	for _, b := range blocks {
		if b.Type == "tool_use" {
			tools = append(tools, b)
		}
	}
	return tools
}

// HasTodoUpdate returns true if this message contains a TodoWrite tool result.
func (m *JSONLMessage) HasTodoUpdate() bool {
	return m.ToolResult != nil && (len(m.ToolResult.OldTodos) > 0 || len(m.ToolResult.NewTodos) > 0)
}

// GetTodos returns the current todo state from this message.
// Returns NewTodos if available, otherwise OldTodos.
func (m *JSONLMessage) GetTodos() []TodoItem {
	if m.ToolResult == nil {
		return nil
	}
	if len(m.ToolResult.NewTodos) > 0 {
		return m.ToolResult.NewTodos
	}
	return m.ToolResult.OldTodos
}
