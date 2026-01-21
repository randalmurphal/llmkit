package claude

import (
	"testing"
)

func TestStreamAccumulator_Append(t *testing.T) {
	acc := NewStreamAccumulator()

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
	acc := NewStreamAccumulator()

	acc.Append(StreamChunk{Content: "partial"})
	acc.Append(StreamChunk{Error: ErrUnavailable})

	if acc.Error() == nil {
		t.Error("Error() should not be nil after error chunk")
	}
}

func TestStreamAccumulator_Reset(t *testing.T) {
	acc := NewStreamAccumulator()

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
	acc := NewStreamAccumulator()

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
	acc := NewStreamAccumulator()

	acc.Append(StreamChunk{Content: "partial"})
	acc.Append(StreamChunk{Error: ErrUnavailable})

	resp := acc.ToResponse()

	if resp.FinishReason != "error" {
		t.Errorf("ToResponse().FinishReason = %q, want %q", resp.FinishReason, "error")
	}
}

func TestStreamAccumulator_ConsumeStream(t *testing.T) {
	acc := NewStreamAccumulator()

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
	acc := NewStreamAccumulator()

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
	acc := NewStreamAccumulator()

	ch := make(chan StreamChunk)
	go func() {
		ch <- StreamChunk{Content: "Working..."}
		ch <- StreamChunk{Content: " Done!"}
		ch <- StreamChunk{Content: " Extra content"}
		ch <- StreamChunk{Done: true}
		close(ch)
	}()

	receivedChunks := 0
	err := acc.ConsumeStreamWithCallback(ch, func(chunk StreamChunk) bool {
		receivedChunks++
		// Stop after 2 chunks
		return receivedChunks < 2
	})

	if err != nil {
		t.Errorf("ConsumeStreamWithCallback() error = %v", err)
	}

	// Should have stopped after 2 chunks
	if receivedChunks != 2 {
		t.Errorf("received %d chunks, want 2", receivedChunks)
	}

	// Content should only include the first 2 chunks
	expected := "Working... Done!"
	if got := acc.Content(); got != expected {
		t.Errorf("Content() = %q, want %q", got, expected)
	}
}

func TestStreamAccumulator_Concurrent(t *testing.T) {
	acc := NewStreamAccumulator()

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
		}
		done <- true
	}()

	<-done
	<-done
}

// Benchmark
func BenchmarkStreamAccumulator_Append(b *testing.B) {
	acc := NewStreamAccumulator()
	chunk := StreamChunk{Content: "Some content to append"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		acc.Append(chunk)
	}
}
