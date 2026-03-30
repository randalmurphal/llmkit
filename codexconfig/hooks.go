package codexconfig

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
)

type HookEvent string

const (
	HookSessionStart     HookEvent = "SessionStart"
	HookPreToolUse       HookEvent = "PreToolUse"
	HookPostToolUse      HookEvent = "PostToolUse"
	HookUserPromptSubmit HookEvent = "UserPromptSubmit"
	HookStop             HookEvent = "Stop"
)

func ValidHookEvents() []HookEvent {
	return []HookEvent{
		HookSessionStart,
		HookPreToolUse,
		HookPostToolUse,
		HookUserPromptSubmit,
		HookStop,
	}
}

func (h HookEvent) IsValid() bool {
	return slices.Contains(ValidHookEvents(), h)
}

type HookConfig struct {
	Hooks map[string][]HookMatcher `json:"hooks"`
}

type HookMatcher struct {
	Matcher string      `json:"matcher,omitempty"`
	Hooks   []HookEntry `json:"hooks"`
}

type HookEntry struct {
	Type          string `json:"type"`
	Command       string `json:"command,omitempty"`
	Timeout       int    `json:"timeout,omitempty"`
	StatusMessage string `json:"statusMessage,omitempty"`
}

type SessionStartSource string

const (
	SessionStartSourceStartup SessionStartSource = "startup"
	SessionStartSourceResume  SessionStartSource = "resume"
	SessionStartSourceClear   SessionStartSource = "clear"
)

type HookContext struct {
	SessionID      string `json:"session_id,omitempty"`
	TurnID         string `json:"turn_id,omitempty"`
	TranscriptPath string `json:"transcript_path,omitempty"`
	CWD            string `json:"cwd,omitempty"`
	HookEventName  string `json:"hook_event_name,omitempty"`
	Model          string `json:"model,omitempty"`
	PermissionMode string `json:"permission_mode,omitempty"`
}

type SessionStartInput struct {
	HookContext
	Source SessionStartSource `json:"source"`
}

type UserPromptSubmitInput struct {
	HookContext
	Prompt string `json:"prompt"`
}

type ToolHookInput struct {
	HookContext
	ToolName   string          `json:"tool_name,omitempty"`
	ToolInput  json.RawMessage `json:"tool_input,omitempty"`
	ToolOutput json.RawMessage `json:"tool_output,omitempty"`
}

type StopInput struct {
	HookContext
	StopHookActive       bool   `json:"stop_hook_active"`
	LastAssistantMessage string `json:"last_assistant_message,omitempty"`
}

type HookDecision string

const (
	HookDecisionBlock HookDecision = "block"
)

type HookOutput struct {
	Continue       *bool           `json:"continue,omitempty"`
	StopReason     string          `json:"stopReason,omitempty"`
	SuppressOutput bool            `json:"suppressOutput,omitempty"`
	SystemMessage  string          `json:"systemMessage,omitempty"`
	Decision       HookDecision    `json:"decision,omitempty"`
	Reason         string          `json:"reason,omitempty"`
	Specific       json.RawMessage `json:"hookSpecificOutput,omitempty"`
}

func ContinueOutput() HookOutput {
	t := true
	return HookOutput{Continue: &t}
}

func AbortOutput(reason string) HookOutput {
	f := false
	return HookOutput{Continue: &f, StopReason: reason}
}

func (h *HookOutput) ShouldContinue() bool {
	return h.Continue == nil || *h.Continue
}

func LoadHooks(projectRoot string) (*HookConfig, error) {
	path := HooksPath(projectRoot)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &HookConfig{Hooks: map[string][]HookMatcher{}}, nil
		}
		return nil, fmt.Errorf("read hooks file: %w", err)
	}

	var cfg HookConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse hooks file: %w", err)
	}
	if cfg.Hooks == nil {
		cfg.Hooks = map[string][]HookMatcher{}
	}
	return &cfg, nil
}

func SaveHooks(projectRoot string, cfg *HookConfig) error {
	if cfg == nil {
		cfg = &HookConfig{Hooks: map[string][]HookMatcher{}}
	}
	if cfg.Hooks == nil {
		cfg.Hooks = map[string][]HookMatcher{}
	}
	path := HooksPath(projectRoot)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create hooks dir: %w", err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal hooks file: %w", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write hooks file: %w", err)
	}
	return nil
}
