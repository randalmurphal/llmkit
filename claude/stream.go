package claude

import (
	"strings"
	"sync"
)

// StreamAccumulator collects streaming events and provides analysis.
// It accumulates events as they arrive.
//
// Thread-safe for concurrent append and read operations.
type StreamAccumulator struct {
	content   strings.Builder
	usage     *TokenUsage
	sessionID string
	model     string
	done      bool
	err       error
	mu        sync.RWMutex
}

// NewStreamAccumulator creates an accumulator for collecting stream events.
//
// Example:
//
//	acc := NewStreamAccumulator()
//
//	for event := range events {
//	    acc.Append(event)
//	}
func NewStreamAccumulator() *StreamAccumulator {
	return &StreamAccumulator{}
}

// Append adds an event's content to the accumulator.
// It captures assistant text, usage data, and session info.
func (a *StreamAccumulator) Append(event StreamEvent) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Capture session ID from any event
	if event.SessionID != "" && a.sessionID == "" {
		a.sessionID = event.SessionID
	}

	switch event.Type {
	case StreamEventInit:
		if event.Init != nil {
			if a.sessionID == "" {
				a.sessionID = event.Init.SessionID
			}
			if a.model == "" {
				a.model = event.Init.Model
			}
		}

	case StreamEventAssistant:
		if event.Assistant != nil {
			if event.Assistant.Text != "" {
				a.content.WriteString(event.Assistant.Text)
			}
			if event.Assistant.Model != "" {
				a.model = event.Assistant.Model
			}
			// Accumulate per-message usage
			if a.usage == nil {
				a.usage = &TokenUsage{}
			}
			a.usage.InputTokens += event.Assistant.Usage.InputTokens
			a.usage.OutputTokens += event.Assistant.Usage.OutputTokens
			a.usage.CacheCreationInputTokens += event.Assistant.Usage.CacheCreationInputTokens
			a.usage.CacheReadInputTokens += event.Assistant.Usage.CacheReadInputTokens
		}

	case StreamEventResult:
		a.done = true
		// Result usage is cumulative, prefer it over accumulated
		if event.Result != nil && a.usage == nil {
			a.usage = &TokenUsage{
				InputTokens:              event.Result.Usage.InputTokens,
				OutputTokens:             event.Result.Usage.OutputTokens,
				CacheCreationInputTokens: event.Result.Usage.CacheCreationInputTokens,
				CacheReadInputTokens:     event.Result.Usage.CacheReadInputTokens,
			}
		}

	case StreamEventError:
		a.err = event.Error
	}

	if a.usage != nil {
		a.usage.TotalTokens = a.usage.InputTokens + a.usage.OutputTokens
	}
}

// Content returns the accumulated content so far.
func (a *StreamAccumulator) Content() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.content.String()
}

// Usage returns the token usage, or nil if not yet received.
func (a *StreamAccumulator) Usage() *TokenUsage {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.usage
}

// SessionID returns the session ID from init/result events.
func (a *StreamAccumulator) SessionID() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.sessionID
}

// Model returns the model name from events.
func (a *StreamAccumulator) Model() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.model
}

// Done returns true if the result event has been received.
func (a *StreamAccumulator) Done() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.done
}

// Error returns any error from the stream.
func (a *StreamAccumulator) Error() error {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.err
}

// Len returns the current length of accumulated content.
func (a *StreamAccumulator) Len() int {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.content.Len()
}

// Reset clears the accumulator for reuse.
func (a *StreamAccumulator) Reset() {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.content.Reset()
	a.usage = nil
	a.sessionID = ""
	a.model = ""
	a.done = false
	a.err = nil
}

// ToResponse converts the accumulated stream into a CompletionResponse.
// This is useful after the stream is complete.
func (a *StreamAccumulator) ToResponse() *CompletionResponse {
	a.mu.RLock()
	defer a.mu.RUnlock()

	resp := &CompletionResponse{
		Content:      a.content.String(),
		SessionID:    a.sessionID,
		Model:        a.model,
		FinishReason: "stop",
	}

	if a.usage != nil {
		resp.Usage = *a.usage
	}

	if a.err != nil {
		resp.FinishReason = "error"
	}

	return resp
}

// ConsumeStream reads all events from a stream channel into the accumulator.
// It blocks until the channel is closed or an error is received.
// Returns the final error, if any.
//
// Example:
//
//	events, result, _ := client.StreamJSON(ctx, req)
//	acc := NewStreamAccumulator()
//	if err := acc.ConsumeStream(events); err != nil {
//	    // Handle error
//	}
//	response := acc.ToResponse()
func (a *StreamAccumulator) ConsumeStream(events <-chan StreamEvent) error {
	for event := range events {
		a.Append(event)
		if event.Error != nil {
			return event.Error
		}
	}
	return nil
}

// ConsumeStreamWithCallback reads events and calls the callback for each one.
// The callback can return false to stop consuming early.
// Returns the final error, if any.
//
// Example:
//
//	acc := NewStreamAccumulator()
//	err := acc.ConsumeStreamWithCallback(events, func(event StreamEvent) bool {
//	    if event.Type == StreamEventAssistant {
//	        fmt.Print(event.Assistant.Text) // Print as we receive
//	    }
//	    return true // Continue consuming
//	})
func (a *StreamAccumulator) ConsumeStreamWithCallback(events <-chan StreamEvent, callback func(StreamEvent) bool) error {
	for event := range events {
		a.Append(event)
		if event.Error != nil {
			return event.Error
		}
		if !callback(event) {
			return nil
		}
	}
	return nil
}
