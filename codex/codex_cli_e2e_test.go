package codex_test

import (
	"bufio"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/randalmurphal/llmkit/codex"
)

func TestCodexCLI_Complete_E2EWithMockCLI(t *testing.T) {
	scriptPath := writeMockCodexScript(t)
	argsFile := filepath.Join(t.TempDir(), "args.txt")

	client := codex.NewCodexCLI(
		codex.WithCodexPath(scriptPath),
		codex.WithModel("gpt-5-codex"),
		codex.WithProfile("ci"),
		codex.WithLocalProvider("ollama"),
		codex.WithWebSearchMode(codex.WebSearchCached),
		codex.WithOutputSchema("/tmp/schema.json"),
		codex.WithOutputLastMessage("/tmp/last.txt"),
		codex.WithConfigOverride("foo", "bar"),
		codex.WithEnabledFeatures([]string{"project_doc"}),
		codex.WithDisabledFeatures([]string{"legacy_mode"}),
		codex.WithColorMode("always"),
		codex.WithOSS(),
		codex.WithSkipGitRepoCheck(),
		codex.WithAddDir("/tmp"),
		codex.WithEnv(map[string]string{
			"CODEX_TEST_MODE":      "success",
			"CODEX_TEST_ARGS_FILE": argsFile,
		}),
	)

	resp, err := client.Complete(context.Background(), codex.CompletionRequest{
		Messages: []codex.Message{{Role: codex.RoleUser, Content: "hello"}},
	})
	if err != nil {
		t.Fatalf("Complete returned error: %v", err)
	}

	if resp.Content != "Hello world" {
		t.Fatalf("resp.Content = %q, want %q", resp.Content, "Hello world")
	}
	if resp.SessionID != "thr_test" {
		t.Fatalf("resp.SessionID = %q, want %q", resp.SessionID, "thr_test")
	}
	if resp.Usage.InputTokens != 3 || resp.Usage.OutputTokens != 2 || resp.Usage.TotalTokens != 5 {
		t.Fatalf("unexpected usage: %+v", resp.Usage)
	}

	args := readLinesFile(t, argsFile)
	assertArgAt(t, args, 0, "exec")
	assertArgAt(t, args, 1, "--json")
	assertArgPair(t, args, "--model", "gpt-5-codex")
	assertArgPair(t, args, "--profile", "ci")
	assertArgPair(t, args, "--local-provider", "ollama")
	assertArgPair(t, args, "--output-schema", "/tmp/schema.json")
	assertArgPair(t, args, "--output-last-message", "/tmp/last.txt")
	assertArgPair(t, args, "--color", "always")
	assertContainsArg(t, args, "--oss")
	assertContainsArg(t, args, "--skip-git-repo-check")
	assertArgPair(t, args, "--enable", "project_doc")
	assertArgPair(t, args, "--disable", "legacy_mode")
	assertArgPair(t, args, "--add-dir", "/tmp")
	assertConfigOverride(t, args, `foo="bar"`)
	assertConfigOverride(t, args, `web_search="cached"`)
}

func TestCodexCLI_Stream_E2EWithMockCLI(t *testing.T) {
	scriptPath := writeMockCodexScript(t)

	client := codex.NewCodexCLI(
		codex.WithCodexPath(scriptPath),
		codex.WithEnv(map[string]string{
			"CODEX_TEST_MODE": "success",
		}),
	)

	stream, err := client.Stream(context.Background(), codex.CompletionRequest{
		Messages: []codex.Message{{Role: codex.RoleUser, Content: "hello"}},
	})
	if err != nil {
		t.Fatalf("Stream returned error: %v", err)
	}

	var gotContent strings.Builder
	var finalUsage *codex.TokenUsage
	var gotDone bool
	for chunk := range stream {
		if chunk.Error != nil {
			t.Fatalf("stream chunk error: %v", chunk.Error)
		}
		if chunk.Content != "" {
			gotContent.WriteString(chunk.Content)
		}
		if chunk.Done {
			gotDone = true
			finalUsage = chunk.Usage
		}
	}

	if !gotDone {
		t.Fatal("expected stream done chunk")
	}
	if gotContent.String() != "Hello world" {
		t.Fatalf("stream content = %q, want %q", gotContent.String(), "Hello world")
	}
	if finalUsage == nil {
		t.Fatal("expected final usage in done chunk")
	}
	if finalUsage.InputTokens != 3 || finalUsage.OutputTokens != 2 || finalUsage.TotalTokens != 5 {
		t.Fatalf("unexpected final usage: %+v", *finalUsage)
	}
}

func TestCodexCLI_ResumeViaComplete_E2EWithMockCLI(t *testing.T) {
	scriptPath := writeMockCodexScript(t)
	argsFile := filepath.Join(t.TempDir(), "args.txt")

	client := codex.NewCodexCLI(
		codex.WithCodexPath(scriptPath),
		codex.WithSessionID("last"),
		codex.WithResumeAll(),
		codex.WithModel("gpt-5.3-codex"),
		codex.WithDangerouslyBypassApprovalsAndSandbox(),
		codex.WithOutputSchema("/tmp/schema.json"),
		codex.WithProfile("ci"),
		codex.WithAddDir("/extra"),
		codex.WithColorMode("always"),
		codex.WithEnv(map[string]string{
			"CODEX_TEST_MODE":      "success",
			"CODEX_TEST_ARGS_FILE": argsFile,
		}),
	)

	_, err := client.Complete(context.Background(), codex.CompletionRequest{
		Messages: []codex.Message{{Role: codex.RoleUser, Content: "continue"}},
	})
	if err != nil {
		t.Fatalf("Complete (resume) returned error: %v", err)
	}

	args := readLinesFile(t, argsFile)

	// Resume prefix must come first
	assertArgAt(t, args, 0, "exec")
	assertArgAt(t, args, 1, "resume")
	assertArgAt(t, args, 2, "--last")
	assertArgAt(t, args, 3, "--all")
	assertArgAt(t, args, 4, "--json")

	// Flags accepted by `exec resume`
	assertArgPair(t, args, "--model", "gpt-5.3-codex")
	assertContainsArg(t, args, "--dangerously-bypass-approvals-and-sandbox")

	// Flags NOT accepted by `exec resume` — must be absent
	for _, forbidden := range []string{"--output-schema", "--profile", "--add-dir", "--color", "--cd"} {
		for _, arg := range args {
			if arg == forbidden {
				t.Fatalf("resume args should NOT contain %q, but got: %v", forbidden, args)
			}
		}
	}
}

func TestCodexCLI_Complete_RetryableErrorClassification(t *testing.T) {
	scriptPath := writeMockCodexScript(t)

	client := codex.NewCodexCLI(
		codex.WithCodexPath(scriptPath),
		codex.WithEnv(map[string]string{
			"CODEX_TEST_MODE": "stderr_rate_limit",
		}),
	)

	_, err := client.Complete(context.Background(), codex.CompletionRequest{
		Messages: []codex.Message{{Role: codex.RoleUser, Content: "hello"}},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var codexErr *codex.Error
	if ok := errorAs(err, &codexErr); !ok {
		t.Fatalf("expected *codex.Error, got %T (%v)", err, err)
	}
	if !codexErr.Retryable {
		t.Fatalf("expected retryable codex error, got: %+v", codexErr)
	}
}

func TestCodexCLI_Complete_SetsWorkdirAndEnv(t *testing.T) {
	scriptPath := writeMockCodexScript(t)
	tmp := t.TempDir()
	workdir := filepath.Join(tmp, "work")
	if err := os.MkdirAll(workdir, 0o755); err != nil {
		t.Fatalf("failed to create workdir: %v", err)
	}
	cwdFile := filepath.Join(tmp, "cwd.txt")
	envFile := filepath.Join(tmp, "env.txt")

	client := codex.NewCodexCLI(
		codex.WithCodexPath(scriptPath),
		codex.WithWorkdir(workdir),
		codex.WithEnv(map[string]string{
			"CODEX_TEST_MODE":     "success",
			"CODEX_TEST_CWD_FILE": cwdFile,
			"CODEX_TEST_ENV_FILE": envFile,
			"CUSTOM_TEST_ENV":     "expected-value",
		}),
	)

	_, err := client.Complete(context.Background(), codex.CompletionRequest{
		Messages: []codex.Message{{Role: codex.RoleUser, Content: "hello"}},
	})
	if err != nil {
		t.Fatalf("Complete returned error: %v", err)
	}

	cwd := strings.TrimSpace(readTextFile(t, cwdFile))
	if cwd != workdir {
		t.Fatalf("cwd = %q, want %q", cwd, workdir)
	}

	envText := readTextFile(t, envFile)
	if !strings.Contains(envText, "CUSTOM_TEST_ENV=expected-value") {
		t.Fatalf("expected CUSTOM_TEST_ENV in env dump, got:\n%s", envText)
	}
}

func writeMockCodexScript(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "mock-codex")

	script := `#!/bin/sh
set -eu

if [ "${CODEX_TEST_ARGS_FILE:-}" != "" ]; then
  : > "$CODEX_TEST_ARGS_FILE"
  for arg in "$@"; do
    printf '%s\n' "$arg" >> "$CODEX_TEST_ARGS_FILE"
  done
fi

if [ "${CODEX_TEST_CWD_FILE:-}" != "" ]; then
  pwd > "$CODEX_TEST_CWD_FILE"
fi

if [ "${CODEX_TEST_ENV_FILE:-}" != "" ]; then
  env > "$CODEX_TEST_ENV_FILE"
fi

mode="${CODEX_TEST_MODE:-success}"
case "$mode" in
  success)
    cat <<'JSON'
{"type":"thread.started","thread_id":"thr_test"}
{"type":"item.updated","item":{"type":"agent_message","delta":"Hello "}}
{"type":"item.completed","item":{"type":"agent_message","text":"world"}}
{"type":"turn.completed","usage":{"input_tokens":3,"output_tokens":2},"output":[{"type":"message","content":[{"type":"output_text","text":"Hello world"}]}]}
JSON
    ;;
  stderr_rate_limit)
    echo "rate limit exceeded" >&2
    exit 1
    ;;
  *)
    echo "unsupported CODEX_TEST_MODE=$mode" >&2
    exit 2
    ;;
esac
`

	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("failed to write mock codex script: %v", err)
	}
	return path
}

func readTextFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read %s: %v", path, err)
	}
	return string(b)
}

func readLinesFile(t *testing.T, path string) []string {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("failed to open %s: %v", path, err)
	}
	defer f.Close()

	var lines []string
	s := bufio.NewScanner(f)
	for s.Scan() {
		lines = append(lines, s.Text())
	}
	if err := s.Err(); err != nil {
		t.Fatalf("failed to scan %s: %v", path, err)
	}
	return lines
}

func assertArgAt(t *testing.T, args []string, index int, want string) {
	t.Helper()
	if index >= len(args) {
		t.Fatalf("args index %d out of range (len=%d): %v", index, len(args), args)
	}
	if args[index] != want {
		t.Fatalf("args[%d]=%q, want %q (args=%v)", index, args[index], want, args)
	}
}

func assertContainsArg(t *testing.T, args []string, want string) {
	t.Helper()
	for _, arg := range args {
		if arg == want {
			return
		}
	}
	t.Fatalf("arg %q not found in %v", want, args)
}

func assertArgPair(t *testing.T, args []string, key, value string) {
	t.Helper()
	for i := 0; i < len(args)-1; i++ {
		if args[i] == key && args[i+1] == value {
			return
		}
	}
	t.Fatalf("arg pair %q %q not found in %v", key, value, args)
}

func assertConfigOverride(t *testing.T, args []string, expected string) {
	t.Helper()
	for i := 0; i < len(args)-1; i++ {
		if args[i] == "-c" && args[i+1] == expected {
			return
		}
	}
	t.Fatalf("config override -c %q not found in %v", expected, args)
}

func errorAs(err error, target **codex.Error) bool {
	if err == nil {
		return false
	}
	if e, ok := err.(*codex.Error); ok {
		*target = e
		return true
	}
	if unwrapped, ok := err.(interface{ Unwrap() error }); ok {
		return errorAs(unwrapped.Unwrap(), target)
	}
	return false
}
