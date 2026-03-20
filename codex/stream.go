package codex

import (
	"strings"
	"sync"
	"time"
)

// StreamAccumulator accumulates streaming chunks into a complete response.
// Thread-safe for concurrent reads during accumulation.
type StreamAccumulator struct {
	mu        sync.RWMutex
	content   strings.Builder
	usage     *TokenUsage
	sessionID string
	done      bool
	err       error
}

// NewStreamAccumulator creates an accumulator for collecting stream chunks.
func NewStreamAccumulator() *StreamAccumulator {
	return &StreamAccumulator{}
}

// Append adds a streaming chunk's data to the accumulator.
// It captures text content, usage data, session info, and errors.
func (a *StreamAccumulator) Append(chunk StreamChunk) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if chunk.SessionID != "" && a.sessionID == "" {
		a.sessionID = chunk.SessionID
	}

	if chunk.Content != "" {
		a.content.WriteString(chunk.Content)
	}

	// FinalContent is the authoritative text when it differs from streamed deltas.
	// Replace accumulated content with the final version.
	if chunk.FinalContent != "" {
		a.content.Reset()
		a.content.WriteString(chunk.FinalContent)
	}

	if chunk.Usage != nil {
		a.usage = chunk.Usage
	}

	if chunk.Error != nil {
		a.err = chunk.Error
	}

	if chunk.Done {
		a.done = true
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

// SessionID returns the session ID from stream chunks.
func (a *StreamAccumulator) SessionID() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.sessionID
}

// Done returns true if the final chunk has been received.
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

// ConsumeStream reads all chunks from a stream channel into the accumulator.
// It blocks until the channel is closed or an error is received.
// Returns the first error encountered, if any.
func (a *StreamAccumulator) ConsumeStream(chunks <-chan StreamChunk) error {
	for chunk := range chunks {
		a.Append(chunk)
		if chunk.Error != nil {
			return chunk.Error
		}
	}
	return nil
}

// ConsumeStreamWithCallback reads chunks and calls the callback for each one.
// The callback can return false to stop consuming early.
// Returns the first error encountered, if any.
func (a *StreamAccumulator) ConsumeStreamWithCallback(chunks <-chan StreamChunk, callback func(StreamChunk) bool) error {
	for chunk := range chunks {
		a.Append(chunk)
		if chunk.Error != nil {
			return chunk.Error
		}
		if !callback(chunk) {
			return nil
		}
	}
	return nil
}

// ToResponse converts the accumulated stream into a CompletionResponse.
func (a *StreamAccumulator) ToResponse(duration time.Duration) *CompletionResponse {
	a.mu.RLock()
	defer a.mu.RUnlock()

	resp := &CompletionResponse{
		Content:      strings.TrimSpace(a.content.String()),
		SessionID:    a.sessionID,
		FinishReason: "stop",
		Duration:     duration,
	}

	if a.usage != nil {
		resp.Usage = *a.usage
	}

	if a.err != nil {
		resp.FinishReason = "error"
	}

	return resp
}
