# claudeconfig

**Claude configuration file parsing.** Reads and writes Claude's `.claude/` directory structure including settings, skills, MCP servers, and plugins.

---

## Package Contents

| File | Purpose | Key Types |
|------|---------|-----------|
| `settings.go` | `settings.json` parsing | `Settings`, `HookConfig`, `AgentConfig` |
| `skill.go` | `SKILL.md` parsing | `Skill`, `ParseSkillMD()` |
| `mcp.go` | `.mcp.json` parsing | `MCPConfig`, `MCPServer` |
| `plugin.go` | Plugin discovery | `Plugin`, `DiscoverPlugins()` |
| `plugin_service.go` | Plugin service management | `PluginService` |
| `claudemd.go` | `CLAUDE.md` parsing | `ClaudeMD`, `ParseClaudeMD()` |
| `agents.go` | Agent definitions | `SubAgent`, `AgentStore` |
| `agent_discovery.go` | Agent `.md` discovery | `DiscoverAgents()` |
| `scripts.go` | Script discovery | `Script`, `DiscoverScripts()` |
| `tools.go` | Tool catalog | `Tool`, `BuiltinTools()` |
| `marketplace.go` | Plugin marketplace | `MarketplaceClient` |

---

## File Locations

Uses constants from `claudecontract/paths.go`:

| File | Default Path | Purpose |
|------|--------------|---------|
| `settings.json` | `~/.claude/settings.json` | User preferences, hooks |
| `.mcp.json` | `~/.claude/.mcp.json` | MCP server configuration |
| `.credentials.json` | `~/.claude/.credentials.json` | OAuth tokens |
| `SKILL.md` | `~/.claude/skills/*/SKILL.md` | Skill definitions |
| `CLAUDE.md` | Project root | Project instructions |

---

## Key Types

### Settings (settings.go)

```go
type Settings struct {
    Theme          string                 `json:"theme"`
    Model          string                 `json:"model"`
    Hooks          map[string]HookConfig  `json:"hooks"`
    AllowedTools   []string               `json:"allowedTools"`
    Agents         map[string]AgentConfig `json:"agents"`
}

// Load from default location
settings, err := LoadSettings("")

// Load from specific path
settings, err := LoadSettings("/path/to/settings.json")
```

### MCPConfig (mcp.go)

```go
type MCPConfig struct {
    Servers map[string]MCPServer `json:"mcpServers"`
}

type MCPServer struct {
    Command string            `json:"command"`
    Args    []string          `json:"args"`
    Env     map[string]string `json:"env"`
}

// Load MCP configuration
config, err := LoadMCPConfig("/project/.mcp.json")
```

### Skill (skill.go)

```go
type Skill struct {
    Name        string   // From directory name
    Description string   // From SKILL.md frontmatter
    Prompt      string   // Skill prompt content
    AllowedTools []string // Tools this skill can use
}

// Parse SKILL.md file
skill, err := ParseSkillMD("/path/to/SKILL.md")
```

### Plugin (plugin.go)

```go
type Plugin struct {
    Name    string
    Path    string
    Config  PluginConfig  // From plugin.json
}

// Discover all plugins in a directory
plugins, err := DiscoverPlugins("~/.claude/plugins")
```

---

## Hook Events

Supported hook events (from `claudecontract/formats.go`):

| Event | When Triggered |
|-------|----------------|
| `PreToolUse` | Before tool execution |
| `PostToolUse` | After tool succeeds |
| `PostToolUseFailure` | After tool fails |
| `PreCompact` | Before context compaction |
| `Stop` | Claude finishes responding |
| `SessionStart` | Session begins |
| `SessionEnd` | Session terminates |
| `UserPromptSubmit` | User submits prompt |

---

## Discovery Functions

| Function | Purpose | Returns |
|----------|---------|---------|
| `DiscoverPlugins(dir)` | Find all plugins | `[]Plugin` |
| `DiscoverAgents(dir)` | Find agent `.md` files | `[]Agent` |
| `DiscoverScripts(dir)` | Find hook scripts | `[]Script` |
| `DiscoverSkills(dir)` | Find skill directories | `[]Skill` |

---

## Testing

```bash
# Run all tests
go test ./claudeconfig/...

# Run specific test
go test ./claudeconfig/... -run TestLoadSettings -v
```

Test files use fixtures in `testdata/` directory.

---

## Dependencies

- Imports `claudecontract/` for path and hook event constants
- No external dependencies (stdlib only)
