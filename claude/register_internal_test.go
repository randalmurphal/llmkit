package claude

import (
	"testing"

	"github.com/randalmurphal/llmkit/v2"
)

func TestAssistantChunkFromEvent_PreservesStructuredContentWithoutText(t *testing.T) {
	event := StreamEvent{
		Type:      StreamEventAssistant,
		SessionID: "sess-1",
		Assistant: &AssistantEvent{
			MessageID: "msg-1",
			Model:     "sonnet",
			Content: []ContentBlock{{
				Type:  "tool_use",
				ID:    "tool-1",
				Name:  "Read",
				Input: []byte(`{"file":"main.go"}`),
			}},
		},
	}

	chunk, ok := assistantChunkFromEvent(event, llmkit.SessionMetadataForID("claude", "sess-1"))
	if !ok {
		t.Fatal("expected assistant chunk to be emitted")
	}
	if chunk.Content != "" {
		t.Fatalf("content = %q, want empty text payload", chunk.Content)
	}
	if chunk.MessageID != "msg-1" {
		t.Fatalf("message id = %q, want msg-1", chunk.MessageID)
	}
	if chunk.Metadata == nil || chunk.Metadata["content_blocks"] == nil {
		t.Fatal("expected content_blocks metadata to be preserved")
	}
}

func TestAssistantChunkFromEvent_RejectsEmptyAssistantPayload(t *testing.T) {
	event := StreamEvent{
		Type:      StreamEventAssistant,
		SessionID: "sess-1",
		Assistant: &AssistantEvent{
			MessageID: "msg-1",
			Model:     "sonnet",
		},
	}

	if _, ok := assistantChunkFromEvent(event, nil); ok {
		t.Fatal("expected empty assistant payload to be skipped")
	}
}
