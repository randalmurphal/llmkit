package claudeconfig

// ToolInfo provides information about an available Claude Code tool.
type ToolInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"` // file, code, web, system, etc.
}

// AvailableTools returns the list of known Claude Code tools.
// This is a static list of core tools - MCP tools are discovered dynamically.
func AvailableTools() []ToolInfo {
	return []ToolInfo{
		// File tools
		{Name: "Read", Description: "Read file contents", Category: "file"},
		{Name: "Write", Description: "Write content to a file", Category: "file"},
		{Name: "Edit", Description: "Edit file with string replacement", Category: "file"},
		{Name: "Glob", Description: "Find files matching patterns", Category: "file"},
		{Name: "Grep", Description: "Search file contents with regex", Category: "file"},
		{Name: "NotebookEdit", Description: "Edit Jupyter notebook cells", Category: "file"},

		// System tools
		{Name: "Bash", Description: "Execute bash commands", Category: "system"},
		{Name: "Task", Description: "Launch sub-agents for complex tasks", Category: "system"},
		{Name: "TodoWrite", Description: "Manage task lists", Category: "system"},
		{Name: "KillShell", Description: "Kill a running background shell", Category: "system"},
		{Name: "TaskOutput", Description: "Get output from background tasks", Category: "system"},

		// Web tools
		{Name: "WebFetch", Description: "Fetch and process web content", Category: "web"},
		{Name: "WebSearch", Description: "Search the web", Category: "web"},

		// Code tools
		{Name: "LSP", Description: "Language Server Protocol operations", Category: "code"},

		// Interaction tools
		{Name: "AskUserQuestion", Description: "Ask user questions during execution", Category: "interaction"},

		// Planning tools
		{Name: "EnterPlanMode", Description: "Enter planning mode for complex tasks", Category: "planning"},
		{Name: "ExitPlanMode", Description: "Exit planning mode and request approval", Category: "planning"},

		// Skill tools
		{Name: "Skill", Description: "Execute a skill within the conversation", Category: "skill"},
	}
}

// ToolsByCategory returns tools grouped by category.
func ToolsByCategory() map[string][]ToolInfo {
	tools := AvailableTools()
	result := make(map[string][]ToolInfo)

	for _, tool := range tools {
		result[tool.Category] = append(result[tool.Category], tool)
	}

	return result
}

// GetTool returns information about a specific tool by name.
func GetTool(name string) *ToolInfo {
	for _, tool := range AvailableTools() {
		if tool.Name == name {
			return &tool
		}
	}
	return nil
}

// ToolCategories returns the list of tool categories.
func ToolCategories() []string {
	return []string{
		"file",
		"system",
		"web",
		"code",
		"interaction",
		"planning",
		"skill",
		"mcp",
	}
}
