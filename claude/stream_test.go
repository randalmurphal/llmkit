package claude

import (
	"testing"
)

func TestStreamAccumulator_Append(t *testing.T) {
	acc := NewStreamAccumulator()

	acc.Append(StreamEvent{
		Type:      StreamEventAssistant,
		SessionID: "test-session",
		Assistant: &AssistantEvent{Text: "Hello "},
	})
	acc.Append(StreamEvent{
		Type:      StreamEventAssistant,
		SessionID: "test-session",
		Assistant: &AssistantEvent{
			Text: "World",
			Usage: MessageUsage{
				InputTokens:  10,
				OutputTokens: 5,
			},
		},
	})

	if got := acc.Content(); got != "Hello World" {
		t.Errorf("Content() = %q, want %q", got, "Hello World")
	}

	if acc.Done() {
		t.Error("Done() should be false before result event")
	}

	acc.Append(StreamEvent{
		Type:      StreamEventResult,
		SessionID: "test-session",
		Result: &ResultEvent{
			Subtype:   "success",
			SessionID: "test-session",
			Usage: ResultUsage{
				InputTokens:  10,
				OutputTokens: 5,
			},
		},
	})

	if !acc.Done() {
		t.Error("Done() should be true after result event")
	}

	if acc.Usage() == nil {
		t.Error("Usage() should not be nil after events with usage")
	}

	if acc.Usage().InputTokens != 10 {
		t.Errorf("Usage().InputTokens = %d, want 10", acc.Usage().InputTokens)
	}
}

func TestStreamAccumulator_SessionAndModel(t *testing.T) {
	acc := NewStreamAccumulator()

	acc.Append(StreamEvent{
		Type:      StreamEventInit,
		SessionID: "test-session-123",
		Init: &InitEvent{
			SessionID: "test-session-123",
			Model:     "claude-opus-4-5",
		},
	})

	if acc.SessionID() != "test-session-123" {
		t.Errorf("SessionID() = %q, want %q", acc.SessionID(), "test-session-123")
	}

	if acc.Model() != "claude-opus-4-5" {
		t.Errorf("Model() = %q, want %q", acc.Model(), "claude-opus-4-5")
	}
}

func TestStreamAccumulator_Error(t *testing.T) {
	acc := NewStreamAccumulator()

	acc.Append(StreamEvent{
		Type:      StreamEventAssistant,
		Assistant: &AssistantEvent{Text: "partial"},
	})
	acc.Append(StreamEvent{
		Type:  StreamEventError,
		Error: ErrUnavailable,
	})

	if acc.Error() == nil {
		t.Error("Error() should not be nil after error event")
	}
}

func TestStreamAccumulator_Reset(t *testing.T) {
	acc := NewStreamAccumulator()

	acc.Append(StreamEvent{
		Type:      StreamEventInit,
		SessionID: "test",
		Init:      &InitEvent{SessionID: "test", Model: "model"},
	})
	acc.Append(StreamEvent{
		Type:      StreamEventAssistant,
		Assistant: &AssistantEvent{Text: "data", Usage: MessageUsage{InputTokens: 10}},
	})
	acc.Append(StreamEvent{Type: StreamEventResult, Result: &ResultEvent{Subtype: "success"}})

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

	if acc.SessionID() != "" {
		t.Error("SessionID() should be empty after reset")
	}

	if acc.Model() != "" {
		t.Error("Model() should be empty after reset")
	}
}

func TestStreamAccumulator_ToResponse(t *testing.T) {
	acc := NewStreamAccumulator()

	acc.Append(StreamEvent{
		Type:      StreamEventInit,
		SessionID: "session-abc",
		Init:      &InitEvent{SessionID: "session-abc", Model: "claude-opus-4-5"},
	})
	acc.Append(StreamEvent{
		Type: StreamEventAssistant,
		Assistant: &AssistantEvent{
			Text: "Response text",
			Usage: MessageUsage{
				InputTokens:  100,
				OutputTokens: 50,
			},
		},
	})
	acc.Append(StreamEvent{
		Type:   StreamEventResult,
		Result: &ResultEvent{Subtype: "success"},
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

	if resp.SessionID != "session-abc" {
		t.Errorf("ToResponse().SessionID = %q, want %q", resp.SessionID, "session-abc")
	}

	if resp.Model != "claude-opus-4-5" {
		t.Errorf("ToResponse().Model = %q, want %q", resp.Model, "claude-opus-4-5")
	}
}

func TestStreamAccumulator_ToResponse_Error(t *testing.T) {
	acc := NewStreamAccumulator()

	acc.Append(StreamEvent{
		Type:      StreamEventAssistant,
		Assistant: &AssistantEvent{Text: "partial"},
	})
	acc.Append(StreamEvent{
		Type:  StreamEventError,
		Error: ErrUnavailable,
	})

	resp := acc.ToResponse()

	if resp.FinishReason != "error" {
		t.Errorf("ToResponse().FinishReason = %q, want %q", resp.FinishReason, "error")
	}
}

func TestStreamAccumulator_ConsumeStream(t *testing.T) {
	acc := NewStreamAccumulator()

	ch := make(chan StreamEvent)
	go func() {
		ch <- StreamEvent{
			Type:      StreamEventInit,
			SessionID: "test",
			Init:      &InitEvent{SessionID: "test"},
		}
		ch <- StreamEvent{
			Type:      StreamEventAssistant,
			Assistant: &AssistantEvent{Text: "Hello "},
		}
		ch <- StreamEvent{
			Type:      StreamEventAssistant,
			Assistant: &AssistantEvent{Text: "World"},
		}
		ch <- StreamEvent{
			Type:   StreamEventResult,
			Result: &ResultEvent{Subtype: "success"},
		}
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

	ch := make(chan StreamEvent)
	go func() {
		ch <- StreamEvent{
			Type:      StreamEventAssistant,
			Assistant: &AssistantEvent{Text: "partial"},
		}
		ch <- StreamEvent{
			Type:  StreamEventError,
			Error: ErrUnavailable,
		}
		close(ch)
	}()

	err := acc.ConsumeStream(ch)
	if err == nil {
		t.Error("ConsumeStream() should return error")
	}
}

func TestStreamAccumulator_ConsumeStreamWithCallback(t *testing.T) {
	acc := NewStreamAccumulator()

	ch := make(chan StreamEvent)
	go func() {
		ch <- StreamEvent{
			Type:      StreamEventAssistant,
			Assistant: &AssistantEvent{Text: "Working..."},
		}
		ch <- StreamEvent{
			Type:      StreamEventAssistant,
			Assistant: &AssistantEvent{Text: " Done!"},
		}
		ch <- StreamEvent{
			Type:      StreamEventAssistant,
			Assistant: &AssistantEvent{Text: " Extra content"},
		}
		ch <- StreamEvent{
			Type:   StreamEventResult,
			Result: &ResultEvent{Subtype: "success"},
		}
		close(ch)
	}()

	receivedEvents := 0
	err := acc.ConsumeStreamWithCallback(ch, func(event StreamEvent) bool {
		receivedEvents++
		// Stop after 2 events
		return receivedEvents < 2
	})

	if err != nil {
		t.Errorf("ConsumeStreamWithCallback() error = %v", err)
	}

	// Should have stopped after 2 events
	if receivedEvents != 2 {
		t.Errorf("received %d events, want 2", receivedEvents)
	}

	// Content should only include the first 2 events
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
			acc.Append(StreamEvent{
				Type:      StreamEventAssistant,
				Assistant: &AssistantEvent{Text: "x"},
			})
		}
		done <- true
	}()

	// Reader goroutine
	go func() {
		for i := 0; i < 100; i++ {
			_ = acc.Content()
			_ = acc.Len()
			_ = acc.Done()
			_ = acc.SessionID()
			_ = acc.Model()
		}
		done <- true
	}()

	<-done
	<-done
}

// Benchmark
func BenchmarkStreamAccumulator_Append(b *testing.B) {
	acc := NewStreamAccumulator()
	event := StreamEvent{
		Type:      StreamEventAssistant,
		Assistant: &AssistantEvent{Text: "Some content to append"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		acc.Append(event)
	}
}
