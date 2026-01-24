package claudecontract

// Tool names - canonical list of built-in Claude Code tools.
// These are the exact names used in allowedTools/disallowedTools.
const (
	// File tools
	ToolRead         = "Read"
	ToolWrite        = "Write"
	ToolEdit         = "Edit"
	ToolGlob         = "Glob"
	ToolGrep         = "Grep"
	ToolNotebookEdit = "NotebookEdit"

	// System tools
	ToolBash       = "Bash"
	ToolTask       = "Task"
	ToolTaskOutput = "TaskOutput"
	ToolTaskStop   = "TaskStop"
	ToolKillBash   = "KillBash"

	// Task management
	ToolTodoWrite  = "TodoWrite"
	ToolTaskCreate = "TaskCreate"
	ToolTaskUpdate = "TaskUpdate"
	ToolTaskList   = "TaskList"
	ToolTaskGet    = "TaskGet"

	// Web tools
	ToolWebFetch  = "WebFetch"
	ToolWebSearch = "WebSearch"

	// Interaction tools
	ToolAskUserQuestion = "AskUserQuestion"

	// Code tools
	ToolLSP = "LSP"

	// Skill tools
	ToolSkill = "Skill"

	// Planning tools
	ToolEnterPlanMode = "EnterPlanMode"
	ToolExitPlanMode  = "ExitPlanMode"

	// Search tools
	ToolToolSearch = "ToolSearch"

	// MCP tools
	ToolListMcpResources = "ListMcpResources"
	ToolReadMcpResource  = "ReadMcpResource"
)

// ToolCategory represents a category of tools.
type ToolCategory string

// Tool category constants.
const (
	ToolCategoryFile        ToolCategory = "file"
	ToolCategorySystem      ToolCategory = "system"
	ToolCategoryTask        ToolCategory = "task"
	ToolCategoryWeb         ToolCategory = "web"
	ToolCategoryInteraction ToolCategory = "interaction"
	ToolCategoryCode        ToolCategory = "code"
	ToolCategorySkill       ToolCategory = "skill"
	ToolCategoryPlanning    ToolCategory = "planning"
	ToolCategoryMCP         ToolCategory = "mcp"
)

// ToolInfo describes a built-in tool.
type ToolInfo struct {
	Name        string
	Description string
	Category    ToolCategory
}

// BuiltinTools returns information about all built-in tools.
func BuiltinTools() []ToolInfo {
	return []ToolInfo{
		// File tools
		{ToolRead, "Read files from the filesystem", ToolCategoryFile},
		{ToolWrite, "Write files to the filesystem", ToolCategoryFile},
		{ToolEdit, "Edit files using string replacement", ToolCategoryFile},
		{ToolGlob, "Find files matching glob patterns", ToolCategoryFile},
		{ToolGrep, "Search file contents with regex", ToolCategoryFile},
		{ToolNotebookEdit, "Edit Jupyter notebook cells", ToolCategoryFile},

		// System tools
		{ToolBash, "Execute bash commands", ToolCategorySystem},
		{ToolTask, "Launch subagents for complex tasks", ToolCategorySystem},
		{ToolTaskOutput, "Get output from background tasks", ToolCategorySystem},
		{ToolTaskStop, "Stop background tasks", ToolCategorySystem},
		{ToolKillBash, "Kill background bash processes", ToolCategorySystem},

		// Task management
		{ToolTodoWrite, "Manage todo lists", ToolCategoryTask},
		{ToolTaskCreate, "Create tasks", ToolCategoryTask},
		{ToolTaskUpdate, "Update tasks", ToolCategoryTask},
		{ToolTaskList, "List tasks", ToolCategoryTask},
		{ToolTaskGet, "Get task details", ToolCategoryTask},

		// Web tools
		{ToolWebFetch, "Fetch and analyze web content", ToolCategoryWeb},
		{ToolWebSearch, "Search the web", ToolCategoryWeb},

		// Interaction tools
		{ToolAskUserQuestion, "Ask clarifying questions", ToolCategoryInteraction},

		// Code tools
		{ToolLSP, "Language server protocol operations", ToolCategoryCode},

		// Skill tools
		{ToolSkill, "Invoke slash commands/skills", ToolCategorySkill},

		// Planning tools
		{ToolEnterPlanMode, "Enter planning mode", ToolCategoryPlanning},
		{ToolExitPlanMode, "Exit planning mode", ToolCategoryPlanning},

		// MCP tools
		{ToolListMcpResources, "List MCP resources", ToolCategoryMCP},
		{ToolReadMcpResource, "Read MCP resources", ToolCategoryMCP},
	}
}

// ToolCategories returns all tool categories.
func ToolCategories() []ToolCategory {
	return []ToolCategory{
		ToolCategoryFile,
		ToolCategorySystem,
		ToolCategoryTask,
		ToolCategoryWeb,
		ToolCategoryInteraction,
		ToolCategoryCode,
		ToolCategorySkill,
		ToolCategoryPlanning,
		ToolCategoryMCP,
	}
}

// ToolsByCategory returns tools grouped by category.
func ToolsByCategory() map[ToolCategory][]ToolInfo {
	result := make(map[ToolCategory][]ToolInfo)
	for _, tool := range BuiltinTools() {
		result[tool.Category] = append(result[tool.Category], tool)
	}
	return result
}

// AllToolNames returns a slice of all tool names.
func AllToolNames() []string {
	tools := BuiltinTools()
	names := make([]string, len(tools))
	for i, t := range tools {
		names[i] = t.Name
	}
	return names
}
