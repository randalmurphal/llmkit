package codexcontract

import (
	"encoding/json"
	"testing"
)

func TestHookEventValidity(t *testing.T) {
	valid := []HookEvent{HookSessionStart, HookPreToolUse, HookPostToolUse, HookUserPromptSubmit, HookStop}
	for _, e := range valid {
		if !e.IsValid() {
			t.Errorf("HookEvent %q should be valid", e)
		}
	}

	invalid := HookEvent("Unknown")
	if invalid.IsValid() {
		t.Errorf("HookEvent %q should not be valid for Codex", invalid)
	}
}

func TestValidHookEvents(t *testing.T) {
	events := ValidHookEvents()
	if len(events) != 5 {
		t.Errorf("ValidHookEvents() returned %d events, want 5", len(events))
	}
}

func TestHookEventString(t *testing.T) {
	if HookSessionStart.String() != "SessionStart" {
		t.Errorf("HookSessionStart.String() = %q, want %q", HookSessionStart.String(), "SessionStart")
	}
}

func TestHookConfig_JSONRoundTrip(t *testing.T) {
	cfg := HookConfig{
		Hooks: map[string][]HookMatcher{
			"SessionStart": {
				{
					Matcher: "startup|resume",
					Hooks: []HookEntry{
						{
							Type:          "command",
							Command:       "my-script.sh",
							Timeout:       600,
							StatusMessage: "Running session init...",
						},
					},
				},
			},
			"Stop": {
				{
					Hooks: []HookEntry{
						{
							Type:    "command",
							Command: "check-debate.sh",
							Timeout: 30,
						},
					},
				},
			},
		},
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var roundTripped HookConfig
	if err := json.Unmarshal(data, &roundTripped); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if len(roundTripped.Hooks) != 2 {
		t.Errorf("got %d event groups, want 2", len(roundTripped.Hooks))
	}

	sessionStart := roundTripped.Hooks["SessionStart"]
	if len(sessionStart) != 1 {
		t.Fatalf("SessionStart: got %d matchers, want 1", len(sessionStart))
	}
	if sessionStart[0].Matcher != "startup|resume" {
		t.Errorf("SessionStart matcher = %q, want %q", sessionStart[0].Matcher, "startup|resume")
	}
	if sessionStart[0].Hooks[0].Timeout != 600 {
		t.Errorf("SessionStart timeout = %d, want 600", sessionStart[0].Hooks[0].Timeout)
	}

	stop := roundTripped.Hooks["Stop"]
	if len(stop) != 1 {
		t.Fatalf("Stop: got %d matchers, want 1", len(stop))
	}
	if stop[0].Matcher != "" {
		t.Errorf("Stop matcher should be empty, got %q", stop[0].Matcher)
	}
}

func TestStopInput_JSONRoundTrip(t *testing.T) {
	input := StopInput{
		HookContext: HookContext{
			SessionID:      "sess-123",
			TurnID:         "turn-456",
			CWD:            "/home/user/project",
			HookEventName:  "Stop",
			Model:          "gpt-5-codex",
			PermissionMode: "dontAsk",
		},
		StopHookActive:       true,
		LastAssistantMessage: "I've completed the task.",
	}

	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var roundTripped StopInput
	if err := json.Unmarshal(data, &roundTripped); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if roundTripped.SessionID != input.SessionID {
		t.Errorf("SessionID = %q, want %q", roundTripped.SessionID, input.SessionID)
	}
	if roundTripped.TurnID != input.TurnID {
		t.Errorf("TurnID = %q, want %q", roundTripped.TurnID, input.TurnID)
	}
	if !roundTripped.StopHookActive {
		t.Error("StopHookActive should be true")
	}
	if roundTripped.LastAssistantMessage != input.LastAssistantMessage {
		t.Errorf("LastAssistantMessage = %q, want %q", roundTripped.LastAssistantMessage, input.LastAssistantMessage)
	}
}

func TestHookOutput_BlockDecision(t *testing.T) {
	output := HookOutput{
		Decision: HookDecisionBlock,
		Reason:   "Debate is still active. You have 2 unread messages.",
	}

	data, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var roundTripped HookOutput
	if err := json.Unmarshal(data, &roundTripped); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if roundTripped.Decision != HookDecisionBlock {
		t.Errorf("Decision = %q, want %q", roundTripped.Decision, HookDecisionBlock)
	}
	if roundTripped.Reason == "" {
		t.Error("Reason should not be empty when decision=block")
	}
}

func TestHookOutput_ContinueFalse(t *testing.T) {
	output := AbortOutput("Session terminated by policy")

	data, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var roundTripped HookOutput
	if err := json.Unmarshal(data, &roundTripped); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if roundTripped.ShouldContinue() {
		t.Error("ShouldContinue() should be false")
	}
	if roundTripped.StopReason != output.StopReason {
		t.Errorf("StopReason = %q, want %q", roundTripped.StopReason, output.StopReason)
	}
}

func TestHookOutput_ContinueHelpers(t *testing.T) {
	cont := ContinueOutput()
	if !cont.ShouldContinue() {
		t.Error("ContinueOutput().ShouldContinue() should be true")
	}

	abort := AbortOutput("test reason")
	if abort.ShouldContinue() {
		t.Error("AbortOutput().ShouldContinue() should be false")
	}
	if abort.StopReason != "test reason" {
		t.Errorf("StopReason = %q, want %q", abort.StopReason, "test reason")
	}
}

func TestHookOutput_NilContinueDefaultsSafe(t *testing.T) {
	// Zero-value HookOutput should default to continue (safe default).
	var output HookOutput
	if !output.ShouldContinue() {
		t.Error("zero-value HookOutput.ShouldContinue() should be true (safe default)")
	}
}

func TestUserPromptSubmitInput_JSONRoundTrip(t *testing.T) {
	input := UserPromptSubmitInput{
		HookContext: HookContext{
			SessionID:      "sess-123",
			TurnID:         "turn-789",
			CWD:            "/project",
			HookEventName:  "UserPromptSubmit",
			Model:          "gpt-5-codex",
			PermissionMode: "default",
		},
		Prompt: "Fix the tests",
	}

	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var roundTripped UserPromptSubmitInput
	if err := json.Unmarshal(data, &roundTripped); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if roundTripped.Prompt != input.Prompt {
		t.Errorf("Prompt = %q, want %q", roundTripped.Prompt, input.Prompt)
	}
	if roundTripped.TurnID != input.TurnID {
		t.Errorf("TurnID = %q, want %q", roundTripped.TurnID, input.TurnID)
	}
}

func TestToolHookInput_JSONRoundTrip(t *testing.T) {
	input := ToolHookInput{
		HookContext: HookContext{
			SessionID:     "sess-123",
			TurnID:        "turn-1",
			HookEventName: "PreToolUse",
		},
		ToolName:  "shell",
		ToolInput: json.RawMessage(`{"cmd":"pwd"}`),
	}

	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var roundTripped ToolHookInput
	if err := json.Unmarshal(data, &roundTripped); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if roundTripped.ToolName != "shell" {
		t.Fatalf("ToolName = %q", roundTripped.ToolName)
	}
	if string(roundTripped.ToolInput) != `{"cmd":"pwd"}` {
		t.Fatalf("ToolInput = %s", roundTripped.ToolInput)
	}
}

func TestSessionStartSpecific_JSONRoundTrip(t *testing.T) {
	specific := SessionStartSpecific{
		HookEventName:     "SessionStart",
		AdditionalContext: "You are in a debate. Check the channel for messages.",
	}

	data, err := json.Marshal(specific)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var roundTripped SessionStartSpecific
	if err := json.Unmarshal(data, &roundTripped); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if roundTripped.AdditionalContext != specific.AdditionalContext {
		t.Errorf("AdditionalContext = %q, want %q", roundTripped.AdditionalContext, specific.AdditionalContext)
	}
}
