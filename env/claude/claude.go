package claude

import (
	rootenv "github.com/randalmurphal/llmkit/v2/env"
)

func LoadSettings(workDir string) (*rootenv.Settings, error) {
	return rootenv.LoadSettings("claude", workDir)
}

func SaveSettings(workDir string, settings *rootenv.Settings) error {
	return rootenv.SaveSettings("claude", workDir, settings)
}

func NewScope(workDir string, cfg rootenv.ScopeConfig) (*rootenv.Scope, error) {
	return rootenv.NewScope("claude", workDir, cfg)
}
