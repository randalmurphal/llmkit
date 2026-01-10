package claudeconfig

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadClaudeMD(t *testing.T) {
	tmpDir := t.TempDir()

	content := `# Project Instructions

This is a test CLAUDE.md file.

## Guidelines

- Follow these rules
- Be helpful
`
	filePath := filepath.Join(tmpDir, "CLAUDE.md")
	require.NoError(t, os.WriteFile(filePath, []byte(content), 0644))

	claudemd, err := LoadClaudeMD(filePath)
	require.NoError(t, err)

	assert.Equal(t, filePath, claudemd.Path)
	assert.Contains(t, claudemd.Content, "Project Instructions")
	assert.Contains(t, claudemd.Content, "Be helpful")
}

func TestLoadClaudeMD_NotExists(t *testing.T) {
	claudemd, err := LoadClaudeMD("/nonexistent/CLAUDE.md")
	require.NoError(t, err)
	assert.Nil(t, claudemd)
}

func TestLoadProjectClaudeMD(t *testing.T) {
	tmpDir := t.TempDir()

	content := "# Project specific instructions"
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "CLAUDE.md"), []byte(content), 0644))

	claudemd, err := LoadProjectClaudeMD(tmpDir)
	require.NoError(t, err)
	require.NotNil(t, claudemd)

	assert.Equal(t, "project", claudemd.Source)
	assert.Contains(t, claudemd.Content, "Project specific instructions")
}

func TestSaveProjectClaudeMD(t *testing.T) {
	tmpDir := t.TempDir()

	content := "# New Instructions\n\nSaved content."
	err := SaveProjectClaudeMD(tmpDir, content)
	require.NoError(t, err)

	// Verify file exists and content is correct
	filePath := filepath.Join(tmpDir, "CLAUDE.md")
	assert.FileExists(t, filePath)

	data, err := os.ReadFile(filePath)
	require.NoError(t, err)
	assert.Equal(t, content, string(data))
}

func TestClaudeMDHierarchy_CombinedContent(t *testing.T) {
	hierarchy := &ClaudeMDHierarchy{
		Global: &ClaudeMD{
			Path:    "/home/user/.claude/CLAUDE.md",
			Content: "# Global\n\nGlobal content.",
		},
		Project: &ClaudeMD{
			Path:    "/project/CLAUDE.md",
			Content: "# Project\n\nProject content.",
		},
	}

	combined := hierarchy.CombinedContent()

	assert.Contains(t, combined, "<!-- Global:")
	assert.Contains(t, combined, "Global content.")
	assert.Contains(t, combined, "<!-- Project:")
	assert.Contains(t, combined, "Project content.")
	assert.Contains(t, combined, "---") // Separator
}

func TestClaudeMDHierarchy_CombinedContent_Empty(t *testing.T) {
	hierarchy := &ClaudeMDHierarchy{}
	combined := hierarchy.CombinedContent()
	assert.Empty(t, combined)
}

func TestClaudeMDHierarchy_CombinedContent_SkipsEmpty(t *testing.T) {
	hierarchy := &ClaudeMDHierarchy{
		Global: &ClaudeMD{
			Path:    "/path",
			Content: "", // Empty content
		},
		Project: &ClaudeMD{
			Path:    "/project/CLAUDE.md",
			Content: "Project content.",
		},
	}

	combined := hierarchy.CombinedContent()

	assert.NotContains(t, combined, "<!-- Global:")
	assert.Contains(t, combined, "<!-- Project:")
}

func TestClaudeMDHierarchy_HasProject(t *testing.T) {
	assert.False(t, (*ClaudeMDHierarchy)(nil).HasProject())
	assert.False(t, (&ClaudeMDHierarchy{}).HasProject())
	assert.False(t, (&ClaudeMDHierarchy{Project: &ClaudeMD{}}).HasProject())
	assert.True(t, (&ClaudeMDHierarchy{Project: &ClaudeMD{Content: "content"}}).HasProject())
}

func TestClaudeMDHierarchy_HasGlobal(t *testing.T) {
	assert.False(t, (*ClaudeMDHierarchy)(nil).HasGlobal())
	assert.False(t, (&ClaudeMDHierarchy{}).HasGlobal())
	assert.True(t, (&ClaudeMDHierarchy{Global: &ClaudeMD{Content: "content"}}).HasGlobal())
}

func TestClaudeMDHierarchy_Count(t *testing.T) {
	hierarchy := &ClaudeMDHierarchy{
		Global:  &ClaudeMD{},
		User:    &ClaudeMD{},
		Project: &ClaudeMD{},
		Local:   []*ClaudeMD{{}, {}},
	}

	assert.Equal(t, 5, hierarchy.Count())
}

func TestClaudeMDPath(t *testing.T) {
	path := ClaudeMDPath("/home/user/project")
	assert.Equal(t, "/home/user/project/CLAUDE.md", path)
}

func TestLoadClaudeMDHierarchy_Integration(t *testing.T) {
	// Create a project directory with CLAUDE.md
	tmpDir := t.TempDir()
	projectContent := "# Project Instructions"
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "CLAUDE.md"), []byte(projectContent), 0644))

	hierarchy, err := LoadClaudeMDHierarchy(tmpDir)
	require.NoError(t, err)

	// Should have project, may or may not have global depending on environment
	assert.NotNil(t, hierarchy.Project)
	assert.Equal(t, "project", hierarchy.Project.Source)
	assert.Equal(t, projectContent, hierarchy.Project.Content)
}
