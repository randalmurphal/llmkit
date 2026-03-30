# claudeconfig

Provider-native parsing and mutation helpers for Claude ecosystem files under `.claude/` plus project `CLAUDE.md`.

## Main Surfaces

| Surface | Purpose |
|---------|---------|
| `Settings` | Read/write `settings.json`, hooks, permissions, env, plugin enablement |
| `MCPConfig` | Read/write `.mcp.json` server definitions |
| `Skill` | Parse and discover `SKILL.md` skills |
| `ClaudeMD` | Parse project instruction files |
| `SubAgent` / `AgentService` | Discover and manage Claude agent files |
| `ProjectScript` / `ScriptService` | Discover and manage reusable scripts |
| `Plugin` / `PluginService` | Discover plugins and manage enablement |
| `Marketplace` helpers | Work with Claude plugin marketplaces |

## Important Paths

| Path | Purpose |
|------|---------|
| `~/.claude/settings.json` | Global settings |
| `<project>/.claude/settings.json` | Project settings |
| `<project>/.mcp.json` | Project MCP config |
| `~/.claude/skills/*/SKILL.md` | User skills |
| `<project>/.claude/skills/*/SKILL.md` | Project skills |
| `<project>/CLAUDE.md` | Project instructions |

## Common Operations

| Operation | API |
|----------|-----|
| Load merged settings | `LoadSettings(workDir)` |
| Load project settings only | `LoadProjectSettings(workDir)` |
| Save project settings | `SaveProjectSettings(workDir, settings)` |
| Load project MCP config | `LoadProjectMCPConfig(workDir)` |
| Save project MCP config | `SaveProjectMCPConfig(workDir, cfg)` |
| Parse one skill | `ParseSkillMD(path)` |
| Discover skills | `DiscoverSkills(dir)` |
| Load instruction hierarchy | `LoadClaudeMDHierarchy(workDir)` |
| Create agent manager | `NewAgentService(projectRoot)` |
| Create script manager | `NewScriptService(projectRoot)` |
| Discover plugins | `DiscoverPlugins(claudeDir)` |

## Hook Events

The settings helper re-exports hook event constants from `claudecontract`, including:

- `HookPreToolUse`
- `HookPostToolUse`
- `HookPreCompact`
- `HookStop`

Use `ValidHookEvents()` when validating user-provided values.

## V2 Notes

- This package remains public in V2 because Claude’s file formats are part of the supported ecosystem surface.
- Root `llmkit.Config` and `llmkit.Request` do not depend on `claudeconfig`; config-file mutation stays explicit and opt-in.

## Testing

```bash
go test ./claudeconfig/...
```
