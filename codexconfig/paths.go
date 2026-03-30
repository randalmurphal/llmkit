package codexconfig

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	DirCodex            = ".codex"
	DirAgentsDot        = ".agents"
	DirSkills           = "skills"
	DirPlugins          = "plugins"
	DirAgents           = "agents"
	DirRules            = "rules"
	FileConfigTOML      = "config.toml"
	FileHooksJSON       = "hooks.json"
	FileSkillMD         = "SKILL.md"
	FileAgentsMD        = "AGENTS.md"
	FileAgentsOverride  = "AGENTS.override.md"
	FileMarketplaceJSON = "marketplace.json"
	DirCodexPlugin      = ".codex-plugin"
	FilePluginJSON      = "plugin.json"
)

func codexHomeDir() (string, error) {
	if home := os.Getenv("CODEX_HOME"); home != "" {
		return home, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home dir: %w", err)
	}
	return filepath.Join(home, DirCodex), nil
}

func UserConfigPath() (string, error) {
	home, err := codexHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, FileConfigTOML), nil
}

func ProjectConfigPath(projectRoot string) string {
	return filepath.Join(projectRoot, DirCodex, FileConfigTOML)
}

func HooksPath(projectRoot string) string {
	return filepath.Join(projectRoot, DirCodex, FileHooksJSON)
}

func ProjectAgentsDir(projectRoot string) string {
	return filepath.Join(projectRoot, DirCodex, DirAgents)
}

func UserAgentsDir() (string, error) {
	home, err := codexHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, DirAgents), nil
}

func UserRulesDir() (string, error) {
	home, err := codexHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, DirRules), nil
}

func ProjectRulesDir(projectRoot string) string {
	return filepath.Join(projectRoot, DirCodex, DirRules)
}

func UserSkillsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home dir: %w", err)
	}
	return filepath.Join(home, DirAgentsDot, DirSkills), nil
}

func RepoSkillsDir(root string) string {
	return filepath.Join(root, DirAgentsDot, DirSkills)
}

func RepoMarketplacePath(root string) string {
	return filepath.Join(root, DirAgentsDot, DirPlugins, FileMarketplaceJSON)
}

func UserMarketplacePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home dir: %w", err)
	}
	return filepath.Join(home, DirAgentsDot, DirPlugins, FileMarketplaceJSON), nil
}

func UserPluginsDir() (string, error) {
	home, err := codexHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, DirPlugins), nil
}

func RepoPluginsDir(root string) string {
	return filepath.Join(root, DirPlugins)
}
