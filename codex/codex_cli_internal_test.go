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

func TestBuildExecArgs_ResumeFiltersUnsupportedFlags(t *testing.T) {
	// Create a client with ALL options set — including those not supported by `exec resume`
	client := NewCodexCLI(
		WithSessionID("thr_abc123"),
		WithModel("gpt-5.3-codex"),
		WithProfile("ci"),
		WithLocalProvider("ollama"),
		WithSandboxMode(SandboxWorkspaceWrite),
		WithApprovalMode(ApprovalNever),
		WithOutputSchema("/tmp/schema.json"),
		WithOutputLastMessage("/tmp/last.txt"),
		WithOSS(),
		WithColorMode("always"),
		WithAddDir("/extra"),
		WithWorkdir("/work"),
		WithDangerouslyBypassApprovalsAndSandbox(),
		WithSkipGitRepoCheck(),
		WithEnabledFeatures([]string{"project_doc"}),
		WithDisabledFeatures([]string{"legacy_mode"}),
		WithConfigOverrides(map[string]any{"foo": "bar"}),
		WithImage("/tmp/i.png"),
	)

	args := client.buildExecArgs(CompletionRequest{Messages: []Message{{Role: RoleUser, Content: "continue"}}})

	// Must start with: exec resume <session_id> --json
	if len(args) < 4 {
		t.Fatalf("short args: %v", args)
	}
	if args[0] != "exec" || args[1] != "resume" || args[2] != "thr_abc123" || args[3] != "--json" {
		t.Fatalf("unexpected prefix: %v", args[:4])
	}

	// Flags that SHOULD be present on resume
	assertArgPair(t, args, "--model", "gpt-5.3-codex")
	if !slices.Contains(args, "--dangerously-bypass-approvals-and-sandbox") {
		t.Fatalf("expected --dangerously-bypass-approvals-and-sandbox in resume args: %v", args)
	}
	if !slices.Contains(args, "--skip-git-repo-check") {
		t.Fatalf("expected --skip-git-repo-check in resume args: %v", args)
	}
	assertArgPair(t, args, "--enable", "project_doc")
	assertArgPair(t, args, "--disable", "legacy_mode")
	assertArgPair(t, args, "--image", "/tmp/i.png")
	requireConfigOverride(t, args, `foo="bar"`)

	// Flags that must NOT be present on resume
	forbidden := []string{
		"--profile", "--local-provider", "--oss", "--color",
		"--sandbox", "--ask-for-approval",
		"--cd", "--add-dir",
		"--output-schema", "--output-last-message",
	}
	for _, flag := range forbidden {
		if slices.Contains(args, flag) {
			t.Fatalf("resume args must NOT contain %q, but got: %v", flag, args)
		}
	}
}

func TestBuildExecArgs_FreshExecIncludesAllFlags(t *testing.T) {
	// Same options as above but NO session ID — should include ALL flags
	client := NewCodexCLI(
		WithModel("gpt-5.3-codex"),
		WithProfile("ci"),
		WithLocalProvider("ollama"),
		WithSandboxMode(SandboxWorkspaceWrite),
		WithApprovalMode(ApprovalNever),
		WithOutputSchema("/tmp/schema.json"),
		WithOutputLastMessage("/tmp/last.txt"),
		WithOSS(),
		WithColorMode("always"),
		WithAddDir("/extra"),
		WithWorkdir("/work"),
		WithSkipGitRepoCheck(),
	)

	args := client.buildExecArgs(CompletionRequest{Messages: []Message{{Role: RoleUser, Content: "hello"}}})

	// Must start with: exec --json (no resume)
	if args[0] != "exec" || args[1] != "--json" {
		t.Fatalf("unexpected prefix for fresh exec: %v", args[:2])
	}

	// All flags should be present
	assertArgPair(t, args, "--model", "gpt-5.3-codex")
	assertArgPair(t, args, "--profile", "ci")
	assertArgPair(t, args, "--local-provider", "ollama")
	assertArgPair(t, args, "--sandbox", string(SandboxWorkspaceWrite))
	assertArgPair(t, args, "--ask-for-approval", string(ApprovalNever))
	assertArgPair(t, args, "--output-schema", "/tmp/schema.json")
	assertArgPair(t, args, "--output-last-message", "/tmp/last.txt")
	assertArgPair(t, args, "--color", "always")
	assertArgPair(t, args, "--add-dir", "/extra")
	assertArgPair(t, args, "--cd", "/work")
	if !slices.Contains(args, "--oss") {
		t.Fatalf("expected --oss in fresh exec args: %v", args)
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
	if len(e3.ToolCalls) != 0 {
		t.Fatalf("item.completed should not emit duplicate tool calls: %#v", e3.ToolCalls)
	}
	if len(e3.ToolResults) != 0 {
		t.Fatalf("tool_call completion should not emit tool results: %#v", e3.ToolResults)
	}

	e3Started, err := parseEventLine([]byte(`{"type":"item.started","item":{"id":"call_1","type":"tool_call","name":"shell","arguments":{"command":"ls"}}}`))
	if err != nil {
		t.Fatalf("parse item.started: %v", err)
	}
	if len(e3Started.ToolCalls) != 1 || e3Started.ToolCalls[0].Name != "shell" {
		t.Fatalf("unexpected tool calls: %#v", e3Started.ToolCalls)
	}
	if !json.Valid(e3Started.ToolCalls[0].Arguments) {
		t.Fatalf("tool call args should be valid json: %s", string(e3Started.ToolCalls[0].Arguments))
	}

	e3Result, err := parseEventLine([]byte(`{"type":"item.completed","item":{"id":"cmd_1","type":"command_execution","command":"/bin/zsh -lc pwd","aggregated_output":"/repo\n","exit_code":0,"status":"completed"}}`))
	if err != nil {
		t.Fatalf("parse command_execution result: %v", err)
	}
	if len(e3Result.ToolResults) != 1 {
		t.Fatalf("expected command_execution tool result, got %#v", e3Result.ToolResults)
	}
	if e3Result.ToolResults[0].Name != "/bin/zsh -lc pwd" || e3Result.ToolResults[0].Output != "/repo\n" {
		t.Fatalf("unexpected tool result: %#v", e3Result.ToolResults[0])
	}
	if e3Result.ToolResults[0].ExitCode == nil || *e3Result.ToolResults[0].ExitCode != 0 {
		t.Fatalf("unexpected exit code: %#v", e3Result.ToolResults[0].ExitCode)
	}

	e4, err := parseEventLine([]byte(`{"type":"turn.completed","usage":{"input_tokens":10,"output_tokens":4,"cached_input_tokens":7},"output":[{"type":"message","content":[{"type":"output_text","text":"Hello world"}]}]}`))
	if err != nil {
		t.Fatalf("parse turn.completed: %v", err)
	}
	if !e4.Done {
		t.Fatal("expected done=true for turn.completed")
	}
	if e4.Usage == nil || e4.Usage.TotalTokens != 14 || e4.Usage.CacheReadInputTokens != 7 {
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
		`{"type":"usage","usage":{"cached_input_tokens":8}}`,
		`{"type":"turn.completed","usage":{"input_tokens":5,"output_tokens":2},"output":[{"type":"message","content":[{"type":"output_text","text":"Hello world"}]}]}`,
	}, "\n")

	resp := client.parseResponse([]byte(data))
	if resp.SessionID != "thr_abc" {
		t.Fatalf("expected session id thr_abc, got %q", resp.SessionID)
	}
	if resp.Content != "Hello world" {
		t.Fatalf("expected content Hello world, got %q", resp.Content)
	}
	if resp.Usage.InputTokens != 5 || resp.Usage.OutputTokens != 2 || resp.Usage.TotalTokens != 7 || resp.Usage.CacheReadInputTokens != 8 {
		t.Fatalf("unexpected usage: %+v", resp.Usage)
	}
}

func TestParseResponse_UsesAuthoritativeTurnOutputWhenItDiffers(t *testing.T) {
	client := NewCodexCLI(WithModel("gpt-5-codex"))
	data := strings.Join([]string{
		`{"type":"thread.started","thread_id":"thr_replace"}`,
		`{"type":"item.updated","item":{"id":"m1","type":"agent_message","delta":"partial "}}`,
		`{"type":"turn.completed","usage":{"input_tokens":2,"output_tokens":3},"output":[{"type":"message","content":[{"type":"output_text","text":"final answer"}]}]}`,
	}, "\n")

	resp := client.parseResponse([]byte(data))
	if resp.SessionID != "thr_replace" {
		t.Fatalf("expected session id thr_replace, got %q", resp.SessionID)
	}
	if resp.Content != "final answer" {
		t.Fatalf("expected content final answer, got %q", resp.Content)
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
