package claude

import (
	"testing"

	"github.com/randalmurphal/llmkit/parser"
)

func TestStreamAccumulator_Append(t *testing.T) {
	acc := NewStreamAccumulator(nil)

	acc.Append(StreamChunk{Content: "Hello "})
	acc.Append(StreamChunk{Content: "World"})

	if got := acc.Content(); got != "Hello World" {
		t.Errorf("Content() = %q, want %q", got, "Hello World")
	}

	if acc.Done() {
		t.Error("Done() should be false before final chunk")
	}

	acc.Append(StreamChunk{Done: true, Usage: &TokenUsage{InputTokens: 10, OutputTokens: 5}})

	if !acc.Done() {
		t.Error("Done() should be true after final chunk")
	}

	if acc.Usage() == nil {
		t.Error("Usage() should not be nil after final chunk")
	}

	if acc.Usage().InputTokens != 10 {
		t.Errorf("Usage().InputTokens = %d, want 10", acc.Usage().InputTokens)
	}
}

func TestStreamAccumulator_Error(t *testing.T) {
	acc := NewStreamAccumulator(nil)

	acc.Append(StreamChunk{Content: "partial"})
	acc.Append(StreamChunk{Error: ErrUnavailable})

	if acc.Error() == nil {
		t.Error("Error() should not be nil after error chunk")
	}
}

func TestStreamAccumulator_HasMarker(t *testing.T) {
	markers := parser.NewMarkerMatcher("phase_complete", "phase_blocked")
	acc := NewStreamAccumulator(markers)

	acc.Append(StreamChunk{Content: "Working on it..."})

	if acc.HasMarker("phase_complete") {
		t.Error("HasMarker() should be false before marker appears")
	}

	acc.Append(StreamChunk{Content: " <phase_complete>true</phase_complete>"})

	if !acc.HasMarker("phase_complete") {
		t.Error("HasMarker() should be true after marker appears")
	}
}

func TestStreamAccumulator_HasMarkerValue(t *testing.T) {
	markers := parser.NewMarkerMatcher("phase_complete")
	acc := NewStreamAccumulator(markers)

	acc.Append(StreamChunk{Content: "<phase_complete>true</phase_complete>"})

	if !acc.HasMarkerValue("phase_complete", "true") {
		t.Error("HasMarkerValue() should match 'true'")
	}

	if acc.HasMarkerValue("phase_complete", "false") {
		t.Error("HasMarkerValue() should not match 'false'")
	}
}

func TestStreamAccumulator_GetMarker(t *testing.T) {
	markers := parser.NewMarkerMatcher("phase_blocked")
	acc := NewStreamAccumulator(markers)

	acc.Append(StreamChunk{Content: "<phase_blocked>need clarification</phase_blocked>"})

	marker, found := acc.GetMarker("phase_blocked")
	if !found {
		t.Error("GetMarker() should find marker")
	}

	if marker.Value != "need clarification" {
		t.Errorf("GetMarker().Value = %q, want %q", marker.Value, "need clarification")
	}
}

func TestStreamAccumulator_GetMarkerValue(t *testing.T) {
	markers := parser.NewMarkerMatcher("status")
	acc := NewStreamAccumulator(markers)

	acc.Append(StreamChunk{Content: "<status>in_progress</status>"})

	if got := acc.GetMarkerValue("status"); got != "in_progress" {
		t.Errorf("GetMarkerValue() = %q, want %q", got, "in_progress")
	}

	// Nonexistent marker
	if got := acc.GetMarkerValue("other"); got != "" {
		t.Errorf("GetMarkerValue() for missing = %q, want empty", got)
	}
}

func TestStreamAccumulator_NilMarkers(t *testing.T) {
	acc := NewStreamAccumulator(nil)

	acc.Append(StreamChunk{Content: "<phase_complete>true</phase_complete>"})

	// All marker methods should return false/empty without panic
	if acc.HasMarker("phase_complete") {
		t.Error("HasMarker() should return false with nil markers")
	}

	if acc.HasMarkerValue("phase_complete", "true") {
		t.Error("HasMarkerValue() should return false with nil markers")
	}

	if _, found := acc.GetMarker("phase_complete"); found {
		t.Error("GetMarker() should return false with nil markers")
	}

	if got := acc.GetMarkerValue("phase_complete"); got != "" {
		t.Error("GetMarkerValue() should return empty with nil markers")
	}

	if markers := acc.AllMarkers(); markers != nil {
		t.Error("AllMarkers() should return nil with nil markers")
	}
}

func TestStreamAccumulator_IsPhaseComplete(t *testing.T) {
	acc := NewStreamAccumulator(nil) // Uses global PhaseMarkers

	acc.Append(StreamChunk{Content: "Still working..."})

	if acc.IsPhaseComplete() {
		t.Error("IsPhaseComplete() should be false before marker")
	}

	acc.Append(StreamChunk{Content: " Done! <phase_complete>true</phase_complete>"})

	if !acc.IsPhaseComplete() {
		t.Error("IsPhaseComplete() should be true after marker")
	}
}

func TestStreamAccumulator_IsPhaseBlocked(t *testing.T) {
	acc := NewStreamAccumulator(nil)

	acc.Append(StreamChunk{Content: "<phase_blocked>need more info</phase_blocked>"})

	if !acc.IsPhaseBlocked() {
		t.Error("IsPhaseBlocked() should be true")
	}

	if got := acc.GetBlockedReason(); got != "need more info" {
		t.Errorf("GetBlockedReason() = %q, want %q", got, "need more info")
	}
}

func TestStreamAccumulator_Reset(t *testing.T) {
	acc := NewStreamAccumulator(nil)

	acc.Append(StreamChunk{Content: "data"})
	acc.Append(StreamChunk{Done: true, Usage: &TokenUsage{InputTokens: 10}})

	if acc.Len() == 0 {
		t.Error("Len() should be > 0 before reset")
	}

	acc.Reset()

	if acc.Len() != 0 {
		t.Error("Len() should be 0 after reset")
	}

	if acc.Done() {
		t.Error("Done() should be false after reset")
	}

	if acc.Usage() != nil {
		t.Error("Usage() should be nil after reset")
	}

	if acc.Error() != nil {
		t.Error("Error() should be nil after reset")
	}
}

func TestStreamAccumulator_ToResponse(t *testing.T) {
	acc := NewStreamAccumulator(nil)

	acc.Append(StreamChunk{Content: "Response text"})
	acc.Append(StreamChunk{
		Done: true,
		Usage: &TokenUsage{
			InputTokens:  100,
			OutputTokens: 50,
			TotalTokens:  150,
		},
	})

	resp := acc.ToResponse()

	if resp.Content != "Response text" {
		t.Errorf("ToResponse().Content = %q, want %q", resp.Content, "Response text")
	}

	if resp.Usage.InputTokens != 100 {
		t.Errorf("ToResponse().Usage.InputTokens = %d, want 100", resp.Usage.InputTokens)
	}

	if resp.FinishReason != "stop" {
		t.Errorf("ToResponse().FinishReason = %q, want %q", resp.FinishReason, "stop")
	}
}

func TestStreamAccumulator_ToResponse_Error(t *testing.T) {
	acc := NewStreamAccumulator(nil)

	acc.Append(StreamChunk{Content: "partial"})
	acc.Append(StreamChunk{Error: ErrUnavailable})

	resp := acc.ToResponse()

	if resp.FinishReason != "error" {
		t.Errorf("ToResponse().FinishReason = %q, want %q", resp.FinishReason, "error")
	}
}

func TestStreamAccumulator_ConsumeStream(t *testing.T) {
	acc := NewStreamAccumulator(nil)

	ch := make(chan StreamChunk)
	go func() {
		ch <- StreamChunk{Content: "Hello "}
		ch <- StreamChunk{Content: "World"}
		ch <- StreamChunk{Done: true}
		close(ch)
	}()

	err := acc.ConsumeStream(ch)
	if err != nil {
		t.Errorf("ConsumeStream() error = %v, want nil", err)
	}

	if got := acc.Content(); got != "Hello World" {
		t.Errorf("Content() = %q, want %q", got, "Hello World")
	}
}

func TestStreamAccumulator_ConsumeStream_Error(t *testing.T) {
	acc := NewStreamAccumulator(nil)

	ch := make(chan StreamChunk)
	go func() {
		ch <- StreamChunk{Content: "partial"}
		ch <- StreamChunk{Error: ErrUnavailable}
		close(ch)
	}()

	err := acc.ConsumeStream(ch)
	if err == nil {
		t.Error("ConsumeStream() should return error")
	}
}

func TestStreamAccumulator_ConsumeStreamWithCallback(t *testing.T) {
	markers := parser.NewMarkerMatcher("phase_complete")
	acc := NewStreamAccumulator(markers)

	ch := make(chan StreamChunk)
	go func() {
		ch <- StreamChunk{Content: "Working..."}
		ch <- StreamChunk{Content: " <phase_complete>true</phase_complete>"}
		ch <- StreamChunk{Content: " Extra content"}
		ch <- StreamChunk{Done: true}
		close(ch)
	}()

	receivedChunks := 0
	err := acc.ConsumeStreamWithCallback(ch, func(chunk StreamChunk) bool {
		receivedChunks++
		// Stop when we see the completion marker
		return !acc.HasMarker("phase_complete")
	})

	if err != nil {
		t.Errorf("ConsumeStreamWithCallback() error = %v", err)
	}

	// Should have stopped after 2 chunks (when marker was detected)
	if receivedChunks != 2 {
		t.Errorf("received %d chunks, want 2", receivedChunks)
	}

	// Content should only include the first 2 chunks
	expected := "Working... <phase_complete>true</phase_complete>"
	if got := acc.Content(); got != expected {
		t.Errorf("Content() = %q, want %q", got, expected)
	}
}

func TestStreamAccumulator_Concurrent(t *testing.T) {
	acc := NewStreamAccumulator(parser.NewMarkerMatcher("test"))

	done := make(chan bool)

	// Writer goroutine
	go func() {
		for i := 0; i < 100; i++ {
			acc.Append(StreamChunk{Content: "x"})
		}
		done <- true
	}()

	// Reader goroutine
	go func() {
		for i := 0; i < 100; i++ {
			_ = acc.Content()
			_ = acc.Len()
			_ = acc.Done()
			_ = acc.HasMarker("test")
		}
		done <- true
	}()

	<-done
	<-done
}

func TestStreamAccumulator_AllMarkers(t *testing.T) {
	markers := parser.NewMarkerMatcher("item")
	acc := NewStreamAccumulator(markers)

	acc.Append(StreamChunk{Content: "<item>one</item> <item>two</item>"})

	all := acc.AllMarkers()
	if len(all) != 2 {
		t.Errorf("AllMarkers() returned %d markers, want 2", len(all))
	}
}

// Benchmark
func BenchmarkStreamAccumulator_Append(b *testing.B) {
	acc := NewStreamAccumulator(nil)
	chunk := StreamChunk{Content: "Some content to append"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		acc.Append(chunk)
	}
}

func BenchmarkStreamAccumulator_HasMarker(b *testing.B) {
	markers := parser.NewMarkerMatcher("phase_complete")
	acc := NewStreamAccumulator(markers)
	acc.Append(StreamChunk{Content: "Some content <phase_complete>true</phase_complete> more content"})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		acc.HasMarker("phase_complete")
	}
}
