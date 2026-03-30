package session

import (
	"testing"

	"github.com/randalmurphal/llmkit/v2/codexcontract"
)

func TestSessionBuildArgsIncludesReasoningEffort(t *testing.T) {
	s := &session{
		config: sessionConfig{
			enabledFeatures: []string{"codex_hooks"},
			reasoningEffort: "xhigh",
		},
	}

	args := s.buildArgs()

	assertContains(t, args, codexcontract.CommandAppServer)
	assertContains(t, args, codexcontract.FlagEnable)
	assertContains(t, args, "codex_hooks")
	assertContains(t, args, codexcontract.FlagConfig)
	assertContains(t, args, `model_reasoning_effort="xhigh"`)
}

func assertContains(t *testing.T, args []string, want string) {
	t.Helper()
	for _, arg := range args {
		if arg == want {
			return
		}
	}
	t.Fatalf("expected args to contain %q, got %v", want, args)
}
