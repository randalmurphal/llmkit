package codex

import (
	"testing"

	"github.com/randalmurphal/llmkit/provider"
)

func TestApplyRequestOverrides(t *testing.T) {
	req := &CompletionRequest{
		Options: map[string]any{
			"web_search":          "cached",
			"output_schema":       "/tmp/schema.json",
			"output_last_message": "/tmp/last.txt",
			"config_overrides": map[string]any{
				"foo": "bar",
			},
		},
	}

	applyRequestOverrides(req)

	if req.WebSearchMode != WebSearchCached {
		t.Fatalf("expected web_search_mode cached, got %q", req.WebSearchMode)
	}
	if req.OutputSchemaPath != "/tmp/schema.json" {
		t.Fatalf("unexpected output schema path: %q", req.OutputSchemaPath)
	}
	if req.OutputLastMessagePath != "/tmp/last.txt" {
		t.Fatalf("unexpected output last message path: %q", req.OutputLastMessagePath)
	}
	if req.ConfigOverrides["foo"] != "bar" {
		t.Fatalf("expected config override foo=bar, got %v", req.ConfigOverrides)
	}
}

func TestNewFromProviderConfig_MapsNewCodexOptions(t *testing.T) {
	client, err := newFromProviderConfig(provider.Config{
		Provider: "codex",
		Model:    "gpt-5-codex",
		Options: map[string]any{
			"sandbox":                "workspace-write",
			"ask_for_approval":       "never",
			"web_search":             "cached",
			"profile":                "ci",
			"local_provider":         "ollama",
			"skip_git_repo_check":    true,
			"output_schema":          "/tmp/schema.json",
			"output_last_message":    "/tmp/last.txt",
			"model_reasoning_effort": "low",
			"hide_agent_reasoning":   true,
			"resume_all":             true,
			"oss":                    true,
			"color":                  "always",
			"enable_features":        []string{"project_doc"},
			"disable_features":       []string{"legacy_mode"},
			"config_overrides": map[string]any{
				"foo": "bar",
			},
		},
	})
	if err != nil {
		t.Fatalf("newFromProviderConfig returned error: %v", err)
	}

	adapter, ok := client.(*codexProviderAdapter)
	if !ok {
		t.Fatalf("unexpected client type: %T", client)
	}

	args := adapter.cli.buildExecArgs(CompletionRequest{Messages: []Message{{Role: RoleUser, Content: "hi"}}})
	assertArgPair(t, args, "--profile", "ci")
	assertArgPair(t, args, "--local-provider", "ollama")
	assertArgPair(t, args, "--oss", "")
	assertArgPair(t, args, "--color", "always")
	assertArgPair(t, args, "--enable", "project_doc")
	assertArgPair(t, args, "--disable", "legacy_mode")
	assertArgPair(t, args, "--output-schema", "/tmp/schema.json")
	assertArgPair(t, args, "--output-last-message", "/tmp/last.txt")
	requireConfigOverride(t, args, `foo="bar"`)
	requireConfigOverride(t, args, `model_reasoning_effort="low"`)
	requireConfigOverride(t, args, "hide_agent_reasoning=true")
	requireConfigOverride(t, args, `web_search="cached"`)
}
