package claude

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/randalmurphal/llmkit/claude/session"
)

// SessionClient wraps a session.Session to implement the Client interface.
// This allows using session-based execution (with multi-turn context retention)
// with code that expects the simpler Client interface.
//
// Unlike ClaudeCLI which spawns a new process per request, SessionClient
// maintains a long-running Claude CLI process with full conversation history.
type SessionClient struct {
	session session.Session
	manager session.SessionManager
	ownsSession bool // True if we created the session and should close it
}

// NewSessionClient creates a client from an existing session.
// The caller is responsible for closing the session when done.
func NewSessionClient(s session.Session) *SessionClient {
	return &SessionClient{
		session:     s,
		ownsSession: false,
	}
}

// NewSessionClientWithManager creates a SessionClient that manages its own session.
// Pass a sessionID to create or resume a session with that ID.
// The session will be closed when Close() is called.
//
// Example:
//
//	mgr := session.NewManager()
//	client, err := NewSessionClientWithManager(ctx, mgr, "task-001",
//	    session.WithModel("claude-opus-4-5-20251101"),
//	    session.WithWorkdir("/path/to/project"),
//	)
//	defer client.Close()
func NewSessionClientWithManager(ctx context.Context, mgr session.SessionManager, sessionID string, opts ...session.SessionOption) (*SessionClient, error) {
	// Check if session exists and can be resumed
	if sessionID != "" {
		if existing, ok := mgr.Get(sessionID); ok {
			return &SessionClient{
				session:     existing,
				manager:     mgr,
				ownsSession: false, // Don't close existing session
			}, nil
		}
	}

	// Add session ID option if provided
	if sessionID != "" {
		opts = append([]session.SessionOption{session.WithSessionID(sessionID)}, opts...)
	}

	// Create new session
	s, err := mgr.Create(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	return &SessionClient{
		session:     s,
		manager:     mgr,
		ownsSession: true,
	}, nil
}

// Session returns the underlying session for direct access.
// Use this when you need session-specific features not exposed by Client.
func (c *SessionClient) Session() session.Session {
	return c.session
}

// SessionID returns the session identifier.
func (c *SessionClient) SessionID() string {
	return c.session.ID()
}

// Complete implements Client by sending a message and waiting for the result.
// The conversation history is maintained across calls to the same SessionClient.
func (c *SessionClient) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	start := time.Now()

	// Convert messages to user message content
	prompt := messagesToPrompt(req.Messages)
	if prompt == "" {
		return nil, NewError("complete", fmt.Errorf("no message content"), false)
	}

	// Send the message
	userMsg := session.NewUserMessage(prompt)
	if err := c.session.Send(ctx, userMsg); err != nil {
		retryable := isSessionRetryable(err)
		return nil, NewError("complete", fmt.Errorf("send message: %w", err), retryable)
	}

	// Collect response until we get a result message
	var content strings.Builder
	var result *session.ResultMessage

	outputCh := c.session.Output()
collectLoop:
	for {
		select {
		case <-ctx.Done():
			return nil, NewError("complete", ctx.Err(), false)
		case msg, ok := <-outputCh:
			if !ok {
				// Channel closed without result
				break collectLoop
			}

			if msg.IsAssistant() {
				text := msg.GetText()
				content.WriteString(text)
			}

			if msg.IsResult() {
				result = msg.Result
				break collectLoop
			}
		}
	}

	// Build response
	resp := &CompletionResponse{
		Content:   content.String(),
		Duration:  time.Since(start),
		SessionID: c.session.ID(),
	}

	if result != nil {
		resp.NumTurns = result.NumTurns
		resp.CostUSD = result.TotalCostUSD
		resp.Usage = TokenUsage{
			InputTokens:              result.Usage.InputTokens,
			OutputTokens:             result.Usage.OutputTokens,
			TotalTokens:              result.Usage.InputTokens + result.Usage.OutputTokens,
			CacheCreationInputTokens: result.Usage.CacheCreationInputTokens,
			CacheReadInputTokens:     result.Usage.CacheReadInputTokens,
		}

		if result.IsError {
			resp.FinishReason = "error"
			// Use result text as error content if no assistant content
			if resp.Content == "" {
				resp.Content = result.Result
			}
		} else {
			resp.FinishReason = "stop"
			// Prefer result text over accumulated assistant content
			if result.Result != "" {
				resp.Content = result.Result
			}
		}

		// Extract model from model usage
		for model := range result.ModelUsage {
			resp.Model = model
			break
		}
	} else {
		// No result message - session may have ended unexpectedly
		resp.FinishReason = "error"
	}

	return resp, nil
}

// StreamJSON implements Client by sending a message and streaming responses.
// Each assistant message is yielded as a StreamEvent.
func (c *SessionClient) StreamJSON(ctx context.Context, req CompletionRequest) (<-chan StreamEvent, *StreamResult, error) {
	// Convert messages to user message content
	prompt := messagesToPrompt(req.Messages)
	if prompt == "" {
		return nil, nil, NewError("stream", fmt.Errorf("no message content"), false)
	}

	// Send the message
	userMsg := session.NewUserMessage(prompt)
	if err := c.session.Send(ctx, userMsg); err != nil {
		retryable := isSessionRetryable(err)
		return nil, nil, NewError("stream", fmt.Errorf("send message: %w", err), retryable)
	}

	events := make(chan StreamEvent)
	result := newStreamResult()

	go func() {
		defer close(events)

		info := c.session.Info()

		// Send init event
		events <- StreamEvent{
			Type:      StreamEventInit,
			SessionID: info.ID,
			Init: &InitEvent{
				SessionID: info.ID,
				Model:     info.Model,
			},
		}

		var totalUsage MessageUsage

		for msg := range c.session.Output() {
			select {
			case <-ctx.Done():
				events <- StreamEvent{Type: StreamEventError, Error: ctx.Err()}
				result.complete(nil, ctx.Err())
				return
			default:
			}

			if msg.IsAssistant() {
				text := msg.GetText()
				if text != "" {
					events <- StreamEvent{
						Type:      StreamEventAssistant,
						SessionID: info.ID,
						Assistant: &AssistantEvent{
							Text:  text,
							Model: info.Model,
							Usage: MessageUsage{}, // Session doesn't provide per-message usage
						},
					}
				}
			}

			if msg.IsResult() {
				msgResult := msg.Result
				if msgResult != nil {
					totalUsage = MessageUsage{
						InputTokens:              msgResult.Usage.InputTokens,
						OutputTokens:             msgResult.Usage.OutputTokens,
						CacheCreationInputTokens: msgResult.Usage.CacheCreationInputTokens,
						CacheReadInputTokens:     msgResult.Usage.CacheReadInputTokens,
					}
				}

				resultEvent := &ResultEvent{
					Subtype:   "success",
					IsError:   false,
					SessionID: info.ID,
					NumTurns:  1,
					Usage: ResultUsage{
						InputTokens:              totalUsage.InputTokens,
						OutputTokens:             totalUsage.OutputTokens,
						CacheCreationInputTokens: totalUsage.CacheCreationInputTokens,
						CacheReadInputTokens:     totalUsage.CacheReadInputTokens,
					},
				}
				result.complete(resultEvent, nil)
				return
			}
		}

		// Session output channel closed without result
		result.complete(nil, fmt.Errorf("session ended without result"))
	}()

	return events, result, nil
}

// Close closes the session if this client owns it.
// If the session was passed in via NewSessionClient, it is NOT closed.
func (c *SessionClient) Close() error {
	if c.ownsSession && c.session != nil {
		return c.session.Close()
	}
	return nil
}

// Info returns session information.
func (c *SessionClient) Info() session.SessionInfo {
	return c.session.Info()
}

// Status returns the current session status.
func (c *SessionClient) Status() session.SessionStatus {
	return c.session.Status()
}

// messagesToPrompt converts CompletionRequest messages to a single prompt string.
// This follows the same logic as ClaudeCLI.appendMessagePrompt.
func messagesToPrompt(messages []Message) string {
	var prompt strings.Builder

	for _, msg := range messages {
		switch msg.Role {
		case RoleUser:
			prompt.WriteString(msg.Content)
			prompt.WriteString("\n")
		case RoleAssistant:
			// For conversation history, format as context
			if prompt.Len() > 0 {
				prompt.WriteString("\nAssistant: ")
				prompt.WriteString(msg.Content)
				prompt.WriteString("\n\nUser: ")
			}
		case RoleSystem:
			// System messages go at the beginning
			prompt.WriteString(msg.Content)
			prompt.WriteString("\n\n")
		}
	}

	return strings.TrimSpace(prompt.String())
}

// isSessionRetryable checks if a session error is retryable.
func isSessionRetryable(err error) bool {
	if err == nil {
		return false
	}
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "connection") ||
		strings.Contains(errStr, "temporary")
}
