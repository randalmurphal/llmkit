package codexcontract

import (
	"strings"
	"testing"
)

func TestFlagNameFormat(t *testing.T) {
	flags := []string{
		FlagJSON,
		FlagModel,
		FlagSandbox,
		FlagAskForApproval,
		FlagFullAuto,
		FlagYolo,
		FlagDangerouslyBypassApprovalsSandbox,
		FlagCD,
		FlagAddDir,
		FlagImage,
		FlagSearch,
		FlagProfile,
		FlagLocalProvider,
		FlagSkipGitRepoCheck,
		FlagOutputSchema,
		FlagOutputLastMessage,
		FlagAll,
		FlagOSS,
		FlagEnable,
		FlagDisable,
		FlagColor,
		FlagVersion,
	}
	for _, flag := range flags {
		if !strings.HasPrefix(flag, "--") {
			t.Errorf("flag %q should start with '--'", flag)
		}
	}

	if FlagConfig != "-c" {
		t.Errorf("FlagConfig = %q, want -c", FlagConfig)
	}
}

func TestCriticalFlagValues(t *testing.T) {
	tests := []struct {
		name string
		got  string
		want string
	}{
		{"FlagJSON", FlagJSON, "--json"},
		{"FlagModel", FlagModel, "--model"},
		{"FlagSandbox", FlagSandbox, "--sandbox"},
		{"FlagAskForApproval", FlagAskForApproval, "--ask-for-approval"},
		{"FlagFullAuto", FlagFullAuto, "--full-auto"},
		{"FlagDangerouslyBypassApprovalsSandbox", FlagDangerouslyBypassApprovalsSandbox, "--dangerously-bypass-approvals-and-sandbox"},
		{"FlagProfile", FlagProfile, "--profile"},
		{"FlagLocalProvider", FlagLocalProvider, "--local-provider"},
		{"FlagSkipGitRepoCheck", FlagSkipGitRepoCheck, "--skip-git-repo-check"},
		{"FlagOutputSchema", FlagOutputSchema, "--output-schema"},
		{"FlagOutputLastMessage", FlagOutputLastMessage, "--output-last-message"},
		{"FlagEnable", FlagEnable, "--enable"},
		{"FlagDisable", FlagDisable, "--disable"},
		{"FlagColor", FlagColor, "--color"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Fatalf("%s = %q, want %q", tt.name, tt.got, tt.want)
			}
		})
	}
}

func TestEventConstants(t *testing.T) {
	events := []string{
		EventThreadStarted,
		EventTurnStarted,
		EventTurnCompleted,
		EventTurnFailed,
		EventItemStarted,
		EventItemUpdated,
		EventItemCompleted,
		EventError,
	}
	for _, event := range events {
		if event == EventError {
			continue
		}
		if !strings.Contains(event, ".") {
			t.Errorf("event type %q should use namespaced form", event)
		}
	}
}
