package claude

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
)

// StreamEventType identifies the type of streaming event.
type StreamEventType string

const (
	StreamEventInit      StreamEventType = "init"
	StreamEventAssistant StreamEventType = "assistant"
	StreamEventResult    StreamEventType = "result"
	StreamEventHook      StreamEventType = "hook"
	StreamEventError     StreamEventType = "error"
)

// StreamEvent represents a single event from Claude CLI stream-json output.
// Check Type to determine which field is populated.
type StreamEvent struct {
	// Type identifies which field is populated.
	Type StreamEventType

	// SessionID is available on all events after init.
	SessionID string

	// Init is populated when Type == StreamEventInit.
	Init *InitEvent

	// Assistant is populated when Type == StreamEventAssistant.
	Assistant *AssistantEvent

	// Result is populated when Type == StreamEventResult.
	Result *ResultEvent

	// Hook is populated when Type == StreamEventHook.
	Hook *HookEvent

	// Error is populated when Type == StreamEventError.
	Error error

	// Raw contains the original JSON for advanced parsing.
	Raw json.RawMessage
}

// InitEvent contains session initialization data.
// Emitted once at the start of streaming, after the first message is sent.
type InitEvent struct {
	SessionID         string      `json:"session_id"`
	Model             string      `json:"model"`
	CWD               string      `json:"cwd"`
	Tools             []string    `json:"tools"`
	MCPServers        []MCPServer `json:"mcp_servers,omitempty"`
	ClaudeCodeVersion string      `json:"claude_code_version"`
	PermissionMode    string      `json:"permissionMode"`
}

// MCPServer represents an MCP server's connection status.
type MCPServer struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

// AssistantEvent contains Claude's response (one per API message).
type AssistantEvent struct {
	// MessageID is the unique message identifier from the API.
	MessageID string `json:"id"`

	// Content contains the content blocks (text, tool_use, etc.).
	Content []ContentBlock `json:"content"`

	// Model is the model that generated this message.
	Model string `json:"model"`

	// StopReason indicates why generation stopped ("end_turn", "tool_use", etc.).
	StopReason string `json:"stop_reason,omitempty"`

	// Usage contains per-message token usage (includes cache tokens).
	Usage MessageUsage `json:"usage"`

	// Text is a convenience field with concatenated text content.
	Text string `json:"-"`
}

// ContentBlock represents a block of content in a message.
type ContentBlock struct {
	Type  string          `json:"type"` // "text", "tool_use", "tool_result"
	Text  string          `json:"text,omitempty"`
	ID    string          `json:"id,omitempty"`    // For tool_use
	Name  string          `json:"name,omitempty"`  // For tool_use
	Input json.RawMessage `json:"input,omitempty"` // For tool_use
}

// MessageUsage tracks token usage for a single message.
type MessageUsage struct {
	InputTokens              int           `json:"input_tokens"`
	OutputTokens             int           `json:"output_tokens"`
	CacheCreationInputTokens int           `json:"cache_creation_input_tokens,omitempty"`
	CacheReadInputTokens     int           `json:"cache_read_input_tokens,omitempty"`
	CacheCreation            *CacheDetails `json:"cache_creation,omitempty"`
	ServiceTier              string        `json:"service_tier,omitempty"`
}

// CacheDetails contains detailed cache token breakdown.
type CacheDetails struct {
	Ephemeral5mInputTokens int `json:"ephemeral_5m_input_tokens,omitempty"`
	Ephemeral1hInputTokens int `json:"ephemeral_1h_input_tokens,omitempty"`
}

// ResultEvent contains the final result of a streaming request.
// This is the last event emitted and contains cumulative totals.
type ResultEvent struct {
	// Subtype is "success" or "error".
	Subtype string `json:"subtype"`

	// IsError indicates if this is an error result.
	IsError bool `json:"is_error"`

	// Result is the text result (for non-schema requests).
	Result string `json:"result"`

	// StructuredOutput contains JSON when --json-schema was used.
	StructuredOutput json.RawMessage `json:"structured_output,omitempty"`

	// SessionID is the session identifier.
	SessionID string `json:"session_id"`

	// DurationMS is the total request duration in milliseconds.
	DurationMS int `json:"duration_ms"`

	// DurationAPIMS is the time spent in API calls.
	DurationAPIMS int `json:"duration_api_ms"`

	// NumTurns is the number of agentic turns taken.
	NumTurns int `json:"num_turns"`

	// TotalCostUSD is the total cost of this request.
	TotalCostUSD float64 `json:"total_cost_usd"`

	// Usage contains cumulative token usage.
	Usage ResultUsage `json:"usage"`

	// ModelUsage contains per-model usage breakdown.
	ModelUsage map[string]ModelUsageDetail `json:"modelUsage,omitempty"`
}

// ResultUsage contains aggregate token usage from a result.
type ResultUsage struct {
	InputTokens              int           `json:"input_tokens"`
	OutputTokens             int           `json:"output_tokens"`
	CacheCreationInputTokens int           `json:"cache_creation_input_tokens,omitempty"`
	CacheReadInputTokens     int           `json:"cache_read_input_tokens,omitempty"`
	CacheCreation            *CacheDetails `json:"cache_creation,omitempty"`
	ServiceTier              string        `json:"service_tier,omitempty"`
}

// ModelUsageDetail contains per-model token usage and cost.
type ModelUsageDetail struct {
	InputTokens              int     `json:"inputTokens"`
	OutputTokens             int     `json:"outputTokens"`
	CacheReadInputTokens     int     `json:"cacheReadInputTokens,omitempty"`
	CacheCreationInputTokens int     `json:"cacheCreationInputTokens,omitempty"`
	WebSearchRequests        int     `json:"webSearchRequests,omitempty"`
	CostUSD                  float64 `json:"costUSD"`
	ContextWindow            int     `json:"contextWindow,omitempty"`
	MaxOutputTokens          int     `json:"maxOutputTokens,omitempty"`
}

// HookEvent contains hook execution output.
type HookEvent struct {
	SessionID string `json:"session_id"`
	HookName  string `json:"hook_name"`
	HookEvent string `json:"hook_event"`
	Stdout    string `json:"stdout"`
	Stderr    string `json:"stderr"`
	ExitCode  int    `json:"exit_code"`
}

// StreamResult is a future that resolves when streaming completes.
// Use Wait() to block until the final result is available.
type StreamResult struct {
	done   chan struct{}
	result *ResultEvent
	err    error
	mu     sync.Mutex
}

// newStreamResult creates a new StreamResult.
func newStreamResult() *StreamResult {
	return &StreamResult{
		done: make(chan struct{}),
	}
}

// NewTestStreamResult creates a StreamResult for testing.
// Use TestComplete() to complete it from test code.
func NewTestStreamResult() *StreamResult {
	return newStreamResult()
}

// TestComplete is a test helper to complete a StreamResult.
// Only use in tests.
func (sr *StreamResult) TestComplete(result *ResultEvent, err error) {
	sr.complete(result, err)
}

// Wait blocks until streaming completes and returns the final result.
func (sr *StreamResult) Wait(ctx context.Context) (*ResultEvent, error) {
	select {
	case <-sr.done:
		sr.mu.Lock()
		defer sr.mu.Unlock()
		return sr.result, sr.err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Done returns a channel that closes when streaming completes.
func (sr *StreamResult) Done() <-chan struct{} {
	return sr.done
}

// complete sets the result and closes the done channel.
func (sr *StreamResult) complete(result *ResultEvent, err error) {
	sr.mu.Lock()
	sr.result = result
	sr.err = err
	sr.mu.Unlock()
	close(sr.done)
}

// StreamToComplete converts streaming events to a CompletionResponse.
// This drains the events channel and waits for the final result.
func StreamToComplete(ctx context.Context, events <-chan StreamEvent, result *StreamResult) (*CompletionResponse, error) {
	return StreamToCompleteWithCallback(ctx, events, result, nil)
}

// StreamToCompleteWithCallback converts streaming events to a CompletionResponse,
// calling the optional onEvent callback for each event as it arrives.
// Use this to capture transcripts in real-time, track progress, or log activity.
func StreamToCompleteWithCallback(ctx context.Context, events <-chan StreamEvent, result *StreamResult, onEvent func(StreamEvent)) (*CompletionResponse, error) {
	var content strings.Builder
	var sessionID string
	var model string
	var totalUsage TokenUsage

	for event := range events {
		// Call event handler first (before any processing) so caller sees raw events
		if onEvent != nil {
			onEvent(event)
		}

		if event.SessionID != "" && sessionID == "" {
			sessionID = event.SessionID
		}
		if event.Type == StreamEventAssistant && event.Assistant != nil {
			content.WriteString(event.Assistant.Text)
			if event.Assistant.Model != "" {
				model = event.Assistant.Model
			}
			// Accumulate usage from each message
			totalUsage.InputTokens += event.Assistant.Usage.InputTokens
			totalUsage.OutputTokens += event.Assistant.Usage.OutputTokens
			totalUsage.CacheCreationInputTokens += event.Assistant.Usage.CacheCreationInputTokens
			totalUsage.CacheReadInputTokens += event.Assistant.Usage.CacheReadInputTokens
		}
	}
	totalUsage.TotalTokens = totalUsage.InputTokens + totalUsage.OutputTokens

	final, err := result.Wait(ctx)
	if err != nil {
		return nil, err
	}

	resp := &CompletionResponse{
		Content:   content.String(),
		SessionID: sessionID,
		Model:     model,
		NumTurns:  final.NumTurns,
		CostUSD:   final.TotalCostUSD,
		Usage:     totalUsage,
	}

	// Handle structured output - it overrides content
	if len(final.StructuredOutput) > 0 {
		resp.Content = string(final.StructuredOutput)
	}

	// Use session ID from result if not captured from events
	if resp.SessionID == "" {
		resp.SessionID = final.SessionID
	}

	if final.IsError {
		resp.FinishReason = "error"
	} else {
		resp.FinishReason = "stop"
	}

	return resp, nil
}
