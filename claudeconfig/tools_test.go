package claudeconfig

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAvailableTools(t *testing.T) {
	tools := AvailableTools()

	// Should have tools
	assert.NotEmpty(t, tools)

	// Check some expected tools exist
	toolNames := make(map[string]bool)
	for _, tool := range tools {
		toolNames[tool.Name] = true
	}

	assert.True(t, toolNames["Read"], "should have Read tool")
	assert.True(t, toolNames["Write"], "should have Write tool")
	assert.True(t, toolNames["Bash"], "should have Bash tool")
	assert.True(t, toolNames["Task"], "should have Task tool")
	assert.True(t, toolNames["WebFetch"], "should have WebFetch tool")
}

func TestToolsByCategory(t *testing.T) {
	byCategory := ToolsByCategory()

	// Should have file category
	fileTools := byCategory["file"]
	assert.NotEmpty(t, fileTools)

	// Check Read is in file category
	hasRead := false
	for _, tool := range fileTools {
		if tool.Name == "Read" {
			hasRead = true
			break
		}
	}
	assert.True(t, hasRead, "Read should be in file category")

	// Should have system category
	systemTools := byCategory["system"]
	assert.NotEmpty(t, systemTools)

	// Check Bash is in system category
	hasBash := false
	for _, tool := range systemTools {
		if tool.Name == "Bash" {
			hasBash = true
			break
		}
	}
	assert.True(t, hasBash, "Bash should be in system category")
}

func TestGetTool(t *testing.T) {
	// Existing tool
	tool := GetTool("Read")
	assert.NotNil(t, tool)
	assert.Equal(t, "Read", tool.Name)
	assert.Equal(t, "file", tool.Category)

	// Non-existent tool
	tool = GetTool("NonExistent")
	assert.Nil(t, tool)
}

func TestToolCategories(t *testing.T) {
	categories := ToolCategories()

	assert.Contains(t, categories, "file")
	assert.Contains(t, categories, "system")
	assert.Contains(t, categories, "web")
	assert.Contains(t, categories, "code")
	assert.Contains(t, categories, "interaction")
}

func TestToolInfo_Fields(t *testing.T) {
	tools := AvailableTools()

	for _, tool := range tools {
		assert.NotEmpty(t, tool.Name, "tool should have name")
		assert.NotEmpty(t, tool.Description, "tool should have description")
		assert.NotEmpty(t, tool.Category, "tool should have category")
	}
}
