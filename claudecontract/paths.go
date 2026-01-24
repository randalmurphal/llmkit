package claudecontract

// File names used by Claude Code.
const (
	// FileSettings is the settings file name.
	FileSettings = "settings.json"

	// FileSettingsLocal is the local (gitignored) settings file name.
	FileSettingsLocal = "settings.local.json"

	// FileMCPConfig is the MCP configuration file name.
	FileMCPConfig = ".mcp.json"

	// FileSkillMD is the skill definition file name.
	FileSkillMD = "SKILL.md"

	// FilePluginJSON is the plugin manifest file name.
	FilePluginJSON = "plugin.json"

	// FileCredentials is the credentials file name.
	FileCredentials = ".credentials.json"

	// FileClaudeMD is the project instructions file name.
	FileClaudeMD = "CLAUDE.md"

	// FileAgentsMD is the agents definition file name.
	FileAgentsMD = "AGENTS.md"

	// FileHooksJSON is the hooks configuration file name.
	FileHooksJSON = "hooks.json"
)

// Directory names used by Claude Code.
const (
	// DirClaude is the main Claude configuration directory.
	DirClaude = ".claude"

	// DirClaudePlugin is the plugin marker directory inside plugins.
	DirClaudePlugin = ".claude-plugin"

	// DirSkills is the skills directory.
	DirSkills = "skills"

	// DirPlugins is the plugins directory.
	DirPlugins = "plugins"

	// DirAgents is the agents directory.
	DirAgents = "agents"

	// DirHooks is the hooks directory.
	DirHooks = "hooks"

	// DirCommands is the plugin commands directory.
	DirCommands = "commands"

	// DirScripts is the scripts directory.
	DirScripts = "scripts"

	// DirReferences is the skill references directory.
	DirReferences = "references"

	// DirAssets is the skill assets directory.
	DirAssets = "assets"

	// DirCache is the cache directory.
	DirCache = "cache"

	// DirPlans is the plans directory.
	DirPlans = "plans"

	// DirProjects is the projects directory for session storage.
	DirProjects = "projects"
)

// SettingSource represents a source for loading settings.
type SettingSource string

const (
	// SettingSourceUser is the global user settings (~/.claude/settings.json).
	SettingSourceUser SettingSource = "user"

	// SettingSourceProject is the project settings (.claude/settings.json).
	SettingSourceProject SettingSource = "project"

	// SettingSourceLocal is the local settings (.claude/settings.local.json).
	SettingSourceLocal SettingSource = "local"
)

// ValidSettingSources returns all valid setting sources.
func ValidSettingSources() []SettingSource {
	return []SettingSource{
		SettingSourceUser,
		SettingSourceProject,
		SettingSourceLocal,
	}
}
