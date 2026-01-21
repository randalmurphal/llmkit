package claude

import (
	"strings"
	"sync"
)

// StreamAccumulator collects streaming content and provides analysis.
// It accumulates chunks as they arrive.
//
// Thread-safe for concurrent append and read operations.
type StreamAccumulator struct {
	content strings.Builder
	usage   *TokenUsage
	done    bool
	err     error
	mu      sync.RWMutex
}

// NewStreamAccumulator creates an accumulator for collecting stream chunks.
//
// Example:
//
//	acc := NewStreamAccumulator()
//
//	for chunk := range stream {
//	    acc.Append(chunk)
//	}
func NewStreamAccumulator() *StreamAccumulator {
	return &StreamAccumulator{}
}

// Append adds a chunk's content to the accumulator.
// It also captures usage data from the final chunk.
func (a *StreamAccumulator) Append(chunk StreamChunk) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if chunk.Content != "" {
		a.content.WriteString(chunk.Content)
	}

	if chunk.Usage != nil {
		a.usage = chunk.Usage
	}

	if chunk.Done {
		a.done = true
	}

	if chunk.Error != nil {
		a.err = chunk.Error
	}
}

// Content returns the accumulated content so far.
func (a *StreamAccumulator) Content() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.content.String()
}

// Usage returns the token usage from the final chunk, or nil if not yet received.
func (a *StreamAccumulator) Usage() *TokenUsage {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.usage
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

// ConsumeStream reads all chunks from a stream channel into the accumulator.
// It blocks until the channel is closed or an error is received.
// Returns the final error, if any.
//
// Example:
//
//	stream, _ := client.Stream(ctx, req)
//	acc := NewStreamAccumulator()
//	if err := acc.ConsumeStream(stream); err != nil {
//	    // Handle error
//	}
//	response := acc.ToResponse()
func (a *StreamAccumulator) ConsumeStream(stream <-chan StreamChunk) error {
	for chunk := range stream {
		a.Append(chunk)
		if chunk.Error != nil {
			return chunk.Error
		}
	}
	return nil
}

// ConsumeStreamWithCallback reads chunks and calls the callback for each one.
// The callback can return false to stop consuming early.
// Returns the final error, if any.
//
// Example:
//
//	acc := NewStreamAccumulator()
//	err := acc.ConsumeStreamWithCallback(stream, func(chunk StreamChunk) bool {
//	    fmt.Print(chunk.Content) // Print as we receive
//	    return true // Continue consuming
//	})
func (a *StreamAccumulator) ConsumeStreamWithCallback(stream <-chan StreamChunk, callback func(StreamChunk) bool) error {
	for chunk := range stream {
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
