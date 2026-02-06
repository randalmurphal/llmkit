package codex

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"
)

func TestBuildExecArgs_HeadlessOptions(t *testing.T) {
	client := NewCodexCLI(
		WithModel("gpt-5-codex"),
		WithProfile("ci"),
		WithLocalProvider("ollama"),
		WithSandboxMode(SandboxWorkspaceWrite),
		WithApprovalMode(ApprovalNever),
		WithWebSearchMode(WebSearchCached),
		WithSkipGitRepoCheck(),
		WithOutputSchema("/tmp/schema.json"),
		WithOutputLastMessage("/tmp/last.txt"),
		WithConfigOverrides(map[string]any{"foo": "bar", "count": 2}),
		WithReasoningEffort("medium"),
		WithHideAgentReasoning(),
		WithOSS(),
		WithColorMode("always"),
		WithEnabledFeatures([]string{"project_doc"}),
		WithDisabledFeatures([]string{"legacy_mode"}),
		WithAddDir("/extra"),
		WithImage("/tmp/i.png"),
	)

	args := client.buildExecArgs(CompletionRequest{
		Messages: []Message{{Role: RoleUser, Content: "hello"}},
		ConfigOverrides: map[string]any{
			"foo": "baz",
		},
		OutputLastMessagePath: "/tmp/last-override.txt",
	})

	if len(args) < 2 || args[0] != "exec" || args[1] != "--json" {
		t.Fatalf("unexpected exec args prefix: %v", args)
	}

	assertArgPair(t, args, "--model", "gpt-5-codex")
	assertArgPair(t, args, "--profile", "ci")
	assertArgPair(t, args, "--local-provider", "ollama")
	assertArgPair(t, args, "--color", "always")
	assertArgPair(t, args, "--sandbox", string(SandboxWorkspaceWrite))
	assertArgPair(t, args, "--ask-for-approval", string(ApprovalNever))
	assertArgPair(t, args, "--oss", "")
	assertArgPair(t, args, "--output-schema", "/tmp/schema.json")
	assertArgPair(t, args, "--output-last-message", "/tmp/last-override.txt")

	if !slices.Contains(args, "--oss") {
		t.Fatalf("expected --oss in args: %v", args)
	}
	assertArgPair(t, args, "--enable", "project_doc")
	assertArgPair(t, args, "--disable", "legacy_mode")

	if !slices.Contains(args, "--skip-git-repo-check") {
		t.Fatalf("expected --skip-git-repo-check in args: %v", args)
	}

	if !slices.Contains(args, "hello") {
		t.Fatalf("expected prompt in args: %v", args)
	}

	requireConfigOverride(t, args, `foo="baz"`)
	requireConfigOverride(t, args, "count=2")
	requireConfigOverride(t, args, `model_reasoning_effort="medium"`)
	requireConfigOverride(t, args, "hide_agent_reasoning=true")
	requireConfigOverride(t, args, `web_search="cached"`)
}

func TestBuildExecArgs_ResumeLast(t *testing.T) {
	client := NewCodexCLI(WithSessionID("last"), WithResumeAll())
	args := client.buildExecArgs(CompletionRequest{Messages: []Message{{Role: RoleUser, Content: "continue"}}})

	wantPrefix := []string{"exec", "resume", "--last", "--all", "--json"}
	if len(args) < len(wantPrefix) {
		t.Fatalf("short args: %v", args)
	}
	for i, want := range wantPrefix {
		if args[i] != want {
			t.Fatalf("args[%d]=%q want %q; args=%v", i, args[i], want, args)
		}
	}
}

func TestBuildExecArgs_SearchModes(t *testing.T) {
	t.Run("legacy search flag", func(t *testing.T) {
		client := NewCodexCLI(WithSearch())
		args := client.buildExecArgs(CompletionRequest{Messages: []Message{{Role: RoleUser, Content: "q"}}})
		requireConfigOverride(t, args, `web_search="live"`)
	})

	t.Run("web search mode via config override", func(t *testing.T) {
		client := NewCodexCLI(WithWebSearchMode(WebSearchDisabled))
		args := client.buildExecArgs(CompletionRequest{Messages: []Message{{Role: RoleUser, Content: "q"}}})
		requireConfigOverride(t, args, `web_search="disabled"`)
	})
}

func TestParseEventLine_ModernHeadlessEvents(t *testing.T) {
	e1, err := parseEventLine([]byte(`{"type":"thread.started","thread_id":"thr_123"}`))
	if err != nil {
		t.Fatalf("parse thread.started: %v", err)
	}
	if e1.SessionID != "thr_123" {
		t.Fatalf("expected session/thread id, got %q", e1.SessionID)
	}

	e2, err := parseEventLine([]byte(`{"type":"item.updated","item":{"id":"i1","type":"agent_message","delta":"Hello "}}`))
	if err != nil {
		t.Fatalf("parse item.updated: %v", err)
	}
	if e2.Text != "Hello " {
		t.Fatalf("unexpected item.updated text: %q", e2.Text)
	}

	e3, err := parseEventLine([]byte(`{"type":"item.completed","item":{"id":"call_1","type":"tool_call","name":"shell","arguments":{"command":"ls"}}}`))
	if err != nil {
		t.Fatalf("parse item.completed: %v", err)
	}
	if len(e3.ToolCalls) != 1 || e3.ToolCalls[0].Name != "shell" {
		t.Fatalf("unexpected tool calls: %#v", e3.ToolCalls)
	}
	if !json.Valid(e3.ToolCalls[0].Arguments) {
		t.Fatalf("tool call args should be valid json: %s", string(e3.ToolCalls[0].Arguments))
	}

	e4, err := parseEventLine([]byte(`{"type":"turn.completed","usage":{"input_tokens":10,"output_tokens":4},"output":[{"type":"message","content":[{"type":"output_text","text":"Hello world"}]}]}`))
	if err != nil {
		t.Fatalf("parse turn.completed: %v", err)
	}
	if !e4.Done {
		t.Fatal("expected done=true for turn.completed")
	}
	if e4.Usage == nil || e4.Usage.TotalTokens != 14 {
		t.Fatalf("unexpected usage: %#v", e4.Usage)
	}
	if e4.Text != "Hello world" || !e4.TextFromTurnOutput {
		t.Fatalf("unexpected turn output text: %+v", e4)
	}
}

func TestParseResponse_ModernJSONL(t *testing.T) {
	client := NewCodexCLI(WithModel("gpt-5-codex"))
	data := strings.Join([]string{
		`{"type":"thread.started","thread_id":"thr_abc"}`,
		`{"type":"item.updated","item":{"id":"m1","type":"agent_message","delta":"Hello "}}`,
		`{"type":"item.completed","item":{"id":"m1","type":"agent_message","text":"world"}}`,
		`{"type":"turn.completed","usage":{"input_tokens":5,"output_tokens":2},"output":[{"type":"message","content":[{"type":"output_text","text":"Hello world"}]}]}`,
	}, "\n")

	resp := client.parseResponse([]byte(data))
	if resp.SessionID != "thr_abc" {
		t.Fatalf("expected session id thr_abc, got %q", resp.SessionID)
	}
	if resp.Content != "Hello world" {
		t.Fatalf("expected content Hello world, got %q", resp.Content)
	}
	if resp.Usage.InputTokens != 5 || resp.Usage.OutputTokens != 2 || resp.Usage.TotalTokens != 7 {
		t.Fatalf("unexpected usage: %+v", resp.Usage)
	}
}

func assertArgPair(t *testing.T, args []string, key, value string) {
	t.Helper()
	if value == "" {
		for i := 0; i < len(args); i++ {
			if args[i] == key {
				return
			}
		}
		t.Fatalf("expected arg %s in %v", key, args)
	}
	for i := 0; i < len(args)-1; i++ {
		if args[i] == key && args[i+1] == value {
			return
		}
	}
	t.Fatalf("expected arg pair %s %s in %v", key, value, args)
}

func requireConfigOverride(t *testing.T, args []string, expected string) {
	t.Helper()
	for i := 0; i < len(args)-1; i++ {
		if args[i] == "-c" && args[i+1] == expected {
			return
		}
	}
	t.Fatalf("expected config override -c %s in %v", expected, args)
}
