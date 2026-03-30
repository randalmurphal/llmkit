package codex

import (
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
