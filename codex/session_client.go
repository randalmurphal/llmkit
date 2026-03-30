package codex

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/randalmurphal/llmkit/v2/codex/session"
)

// SessionClient wraps a session.Session to provide a simpler request/response interface.
// Unlike CodexCLI which spawns a new process per request, SessionClient maintains a
// long-running Codex app-server process with full conversation history.
//
// SessionClient is not safe for concurrent Complete or Stream calls.
// The underlying session's Output channel is shared, so concurrent
// callers would steal each other's messages. Serialize access externally
// if needed.
type SessionClient struct {
	session     session.Session
	manager     session.SessionManager
	ownsSession bool
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
// Pass a threadID to create or resume a session with that ID.
// The session will be closed when Close() is called.
func NewSessionClientWithManager(ctx context.Context, mgr session.SessionManager, threadID string, opts ...session.SessionOption) (*SessionClient, error) {
	if threadID != "" {
		if existing, ok := mgr.Get(threadID); ok {
			return &SessionClient{
				session:     existing,
				manager:     mgr,
				ownsSession: false,
			}, nil
		}
	}

	if threadID != "" {
		opts = append([]session.SessionOption{session.WithThreadID(threadID)}, opts...)
	}

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
func (c *SessionClient) Session() session.Session {
	return c.session
}

// SessionID returns the session/thread identifier.
func (c *SessionClient) SessionID() string {
	return c.session.ID()
}

// ThreadID returns the Codex thread UUID.
func (c *SessionClient) ThreadID() string {
	return c.session.ThreadID()
}

// Complete sends a message and waits for the turn to complete.
// Conversation history is maintained across calls.
func (c *SessionClient) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	start := time.Now()

	prompt := messagesToPrompt(req.Messages)
	if prompt == "" {
		return nil, fmt.Errorf("no message content")
	}

	userMsg := session.NewUserMessage(prompt)
	if err := c.session.Send(ctx, userMsg); err != nil {
		return nil, fmt.Errorf("send message: %w", err)
	}

	var content strings.Builder
	var turnDone bool

	outputCh := c.session.Output()
	for !turnDone {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case msg, ok := <-outputCh:
			if !ok {
				turnDone = true
				break
			}

			if msg.IsAgentMessage() && msg.Content != "" {
				content.WriteString(msg.Content)
			}

			if msg.IsTurnComplete() || msg.IsTurnFailed() {
				turnDone = true
			}
		}
	}

	resp := &CompletionResponse{
		Content:      content.String(),
		Duration:     time.Since(start),
		SessionID:    c.session.ID(),
		FinishReason: "stop",
	}

	if req.Model != "" {
		resp.Model = req.Model
	}

	return resp, nil
}

// Stream sends a message and returns a channel of streaming chunks.
func (c *SessionClient) Stream(ctx context.Context, req CompletionRequest) (<-chan StreamChunk, error) {
	prompt := messagesToPrompt(req.Messages)
	if prompt == "" {
		return nil, fmt.Errorf("no message content")
	}

	userMsg := session.NewUserMessage(prompt)
	if err := c.session.Send(ctx, userMsg); err != nil {
		return nil, fmt.Errorf("send message: %w", err)
	}

	chunks := make(chan StreamChunk)

	go func() {
		defer close(chunks)

		for msg := range c.session.Output() {
			select {
			case <-ctx.Done():
				chunks <- StreamChunk{Error: ctx.Err(), Done: true}
				return
			default:
			}

			if msg.IsAgentMessage() && msg.Content != "" {
				chunks <- StreamChunk{
					Content:   msg.Content,
					SessionID: c.session.ID(),
				}
			}

			if msg.IsTurnComplete() {
				chunks <- StreamChunk{
					Done:      true,
					SessionID: c.session.ID(),
				}
				return
			}

			if msg.IsTurnFailed() {
				chunks <- StreamChunk{
					Error:     fmt.Errorf("turn failed: %s", msg.Error),
					Done:      true,
					SessionID: c.session.ID(),
				}
				return
			}
		}

		// Channel closed without turn completion.
		chunks <- StreamChunk{
			Error: fmt.Errorf("session ended without turn completion"),
			Done:  true,
		}
	}()

	return chunks, nil
}

// Steer injects input into an actively running turn.
// This is a Codex-specific capability not available with other providers.
func (c *SessionClient) Steer(ctx context.Context, content string) error {
	return c.session.Steer(ctx, session.NewUserMessage(content))
}

// Close closes the session if this client owns it.
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
func messagesToPrompt(messages []Message) string {
	var prompt strings.Builder

	for _, msg := range messages {
		switch msg.Role {
		case RoleUser:
			prompt.WriteString(msg.Content)
			prompt.WriteString("\n")
		case RoleAssistant:
			if prompt.Len() > 0 {
				prompt.WriteString("\nAssistant: ")
				prompt.WriteString(msg.Content)
				prompt.WriteString("\n\nUser: ")
			}
		case RoleSystem:
			prompt.WriteString(msg.Content)
			prompt.WriteString("\n\n")
		}
	}

	return strings.TrimSpace(prompt.String())
}
