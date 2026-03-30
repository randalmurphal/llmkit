package llmkit

import (
	"context"
	"encoding/json"
	"time"
)

// EventType categorizes provider-agnostic events.
type EventType int

const (
	EventText EventType = iota
	EventToolCall
	EventToolResult
	EventUsage
	EventError
	EventDone
	EventSessionStart
	EventHook
)

// Event represents a provider-agnostic event from any LLM interaction.
type Event struct {
	Type      EventType
	Provider  string
	Model     string
	SessionID string
	Timestamp time.Time
	Text      string
	ToolCall  *ToolCall
	Usage     *TokenUsage
	Error     error
	Done      bool
	Raw       json.RawMessage
}

// EventHandler receives events from any provider interaction.
type EventHandler func(Event)

// Compile-time interface check.
var _ Client = (*ObservableClient)(nil)

// ObservableClient wraps a Client and emits normalized events to a handler.
// The underlying Client behavior is unchanged — ObservableClient is a transparent proxy.
type ObservableClient struct {
	inner   Client
	handler EventHandler
}

// NewObservableClient creates an ObservableClient that wraps the given client
// and emits normalized events to the handler for every interaction.
func NewObservableClient(client Client, handler EventHandler) *ObservableClient {
	return &ObservableClient{
		inner:   client,
		handler: handler,
	}
}

// emit sends an event to the handler with common fields pre-filled.
func (c *ObservableClient) emit(e Event) {
	if e.Timestamp.IsZero() {
		e.Timestamp = time.Now()
	}
	if e.Provider == "" {
		e.Provider = c.inner.Provider()
	}
	c.handler(e)
}

// Complete wraps the inner Complete, emitting an EventDone with the response content.
func (c *ObservableClient) Complete(ctx context.Context, req Request) (*Response, error) {
	resp, err := c.inner.Complete(ctx, req)
	if err != nil {
		c.emit(Event{
			Type:  EventError,
			Error: err,
		})
		return nil, err
	}

	c.emit(Event{
		Type:      EventDone,
		Model:     resp.Model,
		SessionID: resp.SessionID,
		Text:      resp.Content,
		Usage:     &resp.Usage,
		Done:      true,
	})

	return resp, nil
}

// Stream wraps the inner Stream, reading chunks, emitting events for each,
// and re-publishing to a new channel returned to the caller.
func (c *ObservableClient) Stream(ctx context.Context, req Request) (<-chan StreamChunk, error) {
	innerCh, err := c.inner.Stream(ctx, req)
	if err != nil {
		c.emit(Event{
			Type:  EventError,
			Error: err,
		})
		return nil, err
	}

	outerCh := make(chan StreamChunk)
	go func() {
		defer close(outerCh)
		for chunk := range innerCh {
			c.emitChunkEvents(chunk)
			select {
			case outerCh <- chunk:
			case <-ctx.Done():
				c.emit(Event{Type: EventError, Error: ctx.Err()})
				// Drain inner channel in background to prevent the inner
				// goroutine from leaking when we stop consuming.
				go func() {
					for range innerCh {
					}
				}()
				return
			}
		}
	}()

	return outerCh, nil
}

// emitChunkEvents emits the appropriate events for a single StreamChunk.
func (c *ObservableClient) emitChunkEvents(chunk StreamChunk) {
	if chunk.Content != "" {
		c.emit(Event{
			Type:      EventText,
			Text:      chunk.Content,
			SessionID: chunk.SessionID,
		})
	}

	for i := range chunk.ToolCalls {
		tc := chunk.ToolCalls[i]
		c.emit(Event{
			Type:      EventToolCall,
			ToolCall:  &tc,
			SessionID: chunk.SessionID,
		})
	}

	if chunk.Usage != nil {
		c.emit(Event{
			Type:      EventUsage,
			Usage:     chunk.Usage,
			SessionID: chunk.SessionID,
		})
	}

	if chunk.Error != nil {
		c.emit(Event{
			Type:      EventError,
			Error:     chunk.Error,
			SessionID: chunk.SessionID,
		})
	}

	if chunk.Done {
		c.emit(Event{
			Type:      EventDone,
			Done:      true,
			SessionID: chunk.SessionID,
		})
	}
}

// Provider returns the provider name from the inner client.
func (c *ObservableClient) Provider() string {
	return c.inner.Provider()
}

// Capabilities returns the capabilities from the inner client.
func (c *ObservableClient) Capabilities() Capabilities {
	return c.inner.Capabilities()
}

// Close releases resources held by the inner client.
func (c *ObservableClient) Close() error {
	return c.inner.Close()
}
