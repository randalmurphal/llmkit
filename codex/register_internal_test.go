package codex

import (
	"context"
	"testing"

	"github.com/randalmurphal/llmkit/v2"
)

func TestNewFromProviderConfig_MapsSharedFields(t *testing.T) {
	client, err := newFromProviderConfig(llmkit.Config{
		Provider: "codex",
		Model:    "gpt-5-codex",
		WorkDir:  "/tmp/work",
	})
	if err != nil {
		t.Fatalf("newFromProviderConfig returned error: %v", err)
	}

	adapter, ok := client.(*codexProviderAdapter)
	if !ok {
		t.Fatalf("unexpected client type: %T", client)
	}

	args := adapter.cli.buildExecArgs(CompletionRequest{Messages: []Message{{Role: RoleUser, Content: "hi"}}})
	assertArgPair(t, args, "--model", "gpt-5-codex")
	assertArgPair(t, args, "--cd", "/tmp/work")
}

func TestCodexProviderAdapter_BuildCompletionRequest_UsesDefaultSystemPrompt(t *testing.T) {
	adapter := &codexProviderAdapter{defaultSystemPrompt: "default system"}

	req := adapter.buildCompletionRequest(llmkit.Request{
		Messages: []llmkit.Message{llmkit.NewTextMessage(llmkit.RoleUser, "hi")},
	})

	if req.SystemPrompt != "default system" {
		t.Fatalf("system prompt = %q, want default system", req.SystemPrompt)
	}
}

func TestCodexProviderAdapter_BuildCompletionRequest_RequestPromptWins(t *testing.T) {
	adapter := &codexProviderAdapter{defaultSystemPrompt: "default system"}

	req := adapter.buildCompletionRequest(llmkit.Request{
		SystemPrompt: "request system",
		Messages:     []llmkit.Message{llmkit.NewTextMessage(llmkit.RoleUser, "hi")},
	})

	if req.SystemPrompt != "request system" {
		t.Fatalf("system prompt = %q, want request system", req.SystemPrompt)
	}
}

func TestEmitStreamChunk_StopsWhenContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	ch := make(chan llmkit.StreamChunk)
	if emitStreamChunk(ctx, ch, llmkit.StreamChunk{Type: "assistant", Content: "ignored"}) {
		t.Fatal("emitStreamChunk should stop when the stream context is cancelled")
	}
}
