package claudecontract

import (
	"testing"
)

// TestBuiltinToolConstants validates all built-in tool name constants.
func TestBuiltinToolConstants(t *testing.T) {
	// These are the exact tool names Claude CLI expects
	expectedTools := map[string]string{
		"ToolRead":             "Read",
		"ToolWrite":            "Write",
		"ToolEdit":             "Edit",
		"ToolGlob":             "Glob",
		"ToolGrep":             "Grep",
		"ToolNotebookEdit":     "NotebookEdit",
		"ToolBash":             "Bash",
		"ToolTask":             "Task",
		"ToolTaskOutput":       "TaskOutput",
		"ToolTaskStop":         "TaskStop",
		"ToolKillBash":         "KillBash",
		"ToolTodoWrite":        "TodoWrite",
		"ToolTaskCreate":       "TaskCreate",
		"ToolTaskUpdate":       "TaskUpdate",
		"ToolTaskList":         "TaskList",
		"ToolTaskGet":          "TaskGet",
		"ToolWebFetch":         "WebFetch",
		"ToolWebSearch":        "WebSearch",
		"ToolAskUserQuestion":  "AskUserQuestion",
		"ToolLSP":              "LSP",
		"ToolSkill":            "Skill",
		"ToolEnterPlanMode":    "EnterPlanMode",
		"ToolExitPlanMode":     "ExitPlanMode",
		"ToolToolSearch":       "ToolSearch",
		"ToolListMcpResources": "ListMcpResources",
		"ToolReadMcpResource":  "ReadMcpResource",
	}

	actualTools := map[string]string{
		"ToolRead":             ToolRead,
		"ToolWrite":            ToolWrite,
		"ToolEdit":             ToolEdit,
		"ToolGlob":             ToolGlob,
		"ToolGrep":             ToolGrep,
		"ToolNotebookEdit":     ToolNotebookEdit,
		"ToolBash":             ToolBash,
		"ToolTask":             ToolTask,
		"ToolTaskOutput":       ToolTaskOutput,
		"ToolTaskStop":         ToolTaskStop,
		"ToolKillBash":         ToolKillBash,
		"ToolTodoWrite":        ToolTodoWrite,
		"ToolTaskCreate":       ToolTaskCreate,
		"ToolTaskUpdate":       ToolTaskUpdate,
		"ToolTaskList":         ToolTaskList,
		"ToolTaskGet":          ToolTaskGet,
		"ToolWebFetch":         ToolWebFetch,
		"ToolWebSearch":        ToolWebSearch,
		"ToolAskUserQuestion":  ToolAskUserQuestion,
		"ToolLSP":              ToolLSP,
		"ToolSkill":            ToolSkill,
		"ToolEnterPlanMode":    ToolEnterPlanMode,
		"ToolExitPlanMode":     ToolExitPlanMode,
		"ToolToolSearch":       ToolToolSearch,
		"ToolListMcpResources": ToolListMcpResources,
		"ToolReadMcpResource":  ToolReadMcpResource,
	}

	for name, expected := range expectedTools {
		actual, ok := actualTools[name]
		if !ok {
			t.Errorf("Missing constant: %s", name)
			continue
		}
		if actual != expected {
			t.Errorf("%s = %q, want %q", name, actual, expected)
		}
	}
}

// TestBuiltinToolsFunction validates BuiltinTools returns all tools.
func TestBuiltinToolsFunction(t *testing.T) {
	tools := BuiltinTools()

	// Verify we have a reasonable number of tools
	if len(tools) < 20 {
		t.Errorf("BuiltinTools() returned only %d tools, expected at least 20", len(tools))
	}

	// Verify each tool has required fields
	for _, tool := range tools {
		if tool.Name == "" {
			t.Error("Tool with empty name found")
		}
		if tool.Category == "" {
			t.Errorf("Tool %s has empty category", tool.Name)
		}
		if tool.Description == "" {
			t.Errorf("Tool %s has empty description", tool.Name)
		}
	}

	// Verify critical tools are present
	criticalTools := []string{
		ToolRead, ToolWrite, ToolEdit, ToolBash,
		ToolGlob, ToolGrep, ToolTask, ToolWebFetch,
	}

	toolNames := make(map[string]bool)
	for _, tool := range tools {
		toolNames[tool.Name] = true
	}

	for _, critical := range criticalTools {
		if !toolNames[critical] {
			t.Errorf("Critical tool %s missing from BuiltinTools()", critical)
		}
	}
}

// TestToolCategories validates tool categories are valid.
func TestToolCategories(t *testing.T) {
	validCategories := map[ToolCategory]bool{
		ToolCategoryFile:        true,
		ToolCategorySystem:      true,
		ToolCategoryTask:        true,
		ToolCategoryWeb:         true,
		ToolCategoryInteraction: true,
		ToolCategoryCode:        true,
		ToolCategorySkill:       true,
		ToolCategoryPlanning:    true,
		ToolCategoryMCP:         true,
	}

	tools := BuiltinTools()
	for _, tool := range tools {
		if !validCategories[tool.Category] {
			t.Errorf("Tool %s has invalid category: %s", tool.Name, tool.Category)
		}
	}
}

// TestToolNamesForCLIAllowedTools validates tool names work with --allowedTools.
func TestToolNamesForCLIAllowedTools(t *testing.T) {
	// Tool names must be exact matches for --allowedTools flag
	// They should be PascalCase without spaces
	tools := BuiltinTools()
	for _, tool := range tools {
		// Check no spaces
		for _, c := range tool.Name {
			if c == ' ' {
				t.Errorf("Tool %s contains spaces", tool.Name)
			}
		}

		// Check PascalCase (first letter uppercase)
		if tool.Name != "" && tool.Name[0] >= 'a' && tool.Name[0] <= 'z' {
			t.Errorf("Tool %s should start with uppercase letter", tool.Name)
		}
	}
}

// TestToolPatternsForBash validates Bash tool pattern format.
func TestToolPatternsForBash(t *testing.T) {
	// The CLI supports patterns like "Bash(git:*)" for allowing specific commands
	// This test documents the expected pattern format

	validPatterns := []string{
		"Bash",               // Allow all bash
		"Bash(git:*)",        // Allow all git commands
		"Bash(npm:*)",        // Allow all npm commands
		"Bash(git commit:*)", // Allow git commit with any args
	}

	// These patterns should be usable with --allowedTools
	for _, pattern := range validPatterns {
		// Pattern should start with tool name
		if len(pattern) < 4 || pattern[:4] != "Bash" {
			t.Errorf("Invalid pattern format: %s", pattern)
		}
	}
}
