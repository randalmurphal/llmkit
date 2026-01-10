package claude

import (
	"strings"
	"sync"

	"github.com/randalmurphal/llmkit/parser"
)

// StreamAccumulator collects streaming content and provides analysis.
// It accumulates chunks as they arrive and optionally checks for markers
// in real-time, enabling early detection of completion signals.
//
// Thread-safe for concurrent append and read operations.
type StreamAccumulator struct {
	content strings.Builder
	markers *parser.MarkerMatcher
	usage   *TokenUsage
	done    bool
	err     error
	mu      sync.RWMutex
}

// NewStreamAccumulator creates an accumulator with optional marker detection.
// If markers is nil, marker detection methods will return false/empty.
//
// Example:
//
//	markers := parser.NewMarkerMatcher("phase_complete", "phase_blocked")
//	acc := NewStreamAccumulator(markers)
//
//	for chunk := range stream {
//	    acc.Append(chunk)
//	    if acc.HasMarker("phase_complete") {
//	        // Early completion detection
//	        break
//	    }
//	}
func NewStreamAccumulator(markers *parser.MarkerMatcher) *StreamAccumulator {
	return &StreamAccumulator{
		markers: markers,
	}
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

// HasMarker checks if a marker with the given tag has been detected
// in the accumulated content so far.
//
// Returns false if no markers matcher was provided.
func (a *StreamAccumulator) HasMarker(tag string) bool {
	if a.markers == nil {
		return false
	}

	a.mu.RLock()
	content := a.content.String()
	a.mu.RUnlock()

	return a.markers.Contains(content, tag)
}

// HasMarkerValue checks if a marker with the specific tag and value exists
// in the accumulated content.
func (a *StreamAccumulator) HasMarkerValue(tag, value string) bool {
	if a.markers == nil {
		return false
	}

	a.mu.RLock()
	content := a.content.String()
	a.mu.RUnlock()

	return a.markers.ContainsValue(content, tag, value)
}

// GetMarker returns the first marker with the given tag found in the
// accumulated content. Returns false if not found or no matcher configured.
func (a *StreamAccumulator) GetMarker(tag string) (parser.Marker, bool) {
	if a.markers == nil {
		return parser.Marker{}, false
	}

	a.mu.RLock()
	content := a.content.String()
	a.mu.RUnlock()

	return a.markers.FindFirst(content, tag)
}

// GetMarkerValue returns the value of a marker with the given tag.
// Returns empty string if not found.
func (a *StreamAccumulator) GetMarkerValue(tag string) string {
	if a.markers == nil {
		return ""
	}

	a.mu.RLock()
	content := a.content.String()
	a.mu.RUnlock()

	return a.markers.GetValue(content, tag)
}

// AllMarkers returns all markers found in the accumulated content.
// Returns nil if no matcher configured or no markers found.
func (a *StreamAccumulator) AllMarkers() []parser.Marker {
	if a.markers == nil {
		return nil
	}

	a.mu.RLock()
	content := a.content.String()
	a.mu.RUnlock()

	return a.markers.FindAll(content)
}

// IsPhaseComplete checks if a phase_complete marker with value "true"
// has been detected. Uses the global PhaseMarkers matcher.
func (a *StreamAccumulator) IsPhaseComplete() bool {
	a.mu.RLock()
	content := a.content.String()
	a.mu.RUnlock()

	return parser.IsPhaseComplete(content)
}

// IsPhaseBlocked checks if a phase_blocked marker has been detected.
// Uses the global PhaseMarkers matcher.
func (a *StreamAccumulator) IsPhaseBlocked() bool {
	a.mu.RLock()
	content := a.content.String()
	a.mu.RUnlock()

	return parser.IsPhaseBlocked(content)
}

// GetBlockedReason returns the reason from a phase_blocked marker, if present.
// Uses the global PhaseMarkers matcher.
func (a *StreamAccumulator) GetBlockedReason() string {
	a.mu.RLock()
	content := a.content.String()
	a.mu.RUnlock()

	return parser.GetBlockedReason(content)
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
//	acc := NewStreamAccumulator(nil)
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
//	acc := NewStreamAccumulator(markers)
//	err := acc.ConsumeStreamWithCallback(stream, func(chunk StreamChunk) bool {
//	    fmt.Print(chunk.Content) // Print as we receive
//	    return !acc.IsPhaseComplete() // Stop early on completion
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
