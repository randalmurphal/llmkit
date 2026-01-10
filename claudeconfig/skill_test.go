package claudeconfig

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSkillMD(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	skillDir := filepath.Join(tmpDir, "test-skill")
	require.NoError(t, os.MkdirAll(skillDir, 0755))

	// Create SKILL.md file
	content := `---
name: Test Skill
description: A test skill for unit testing
allowed-tools:
  - Read
  - Bash
version: "1.0"
---

# Test Skill

This is the skill content.

## Usage

Use this skill for testing.
`
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0644))

	// Create resource directories
	require.NoError(t, os.MkdirAll(filepath.Join(skillDir, "references"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(skillDir, "scripts"), 0755))

	// Parse the skill
	skill, err := ParseSkillMD(skillDir)
	require.NoError(t, err)

	assert.Equal(t, "Test Skill", skill.Name)
	assert.Equal(t, "A test skill for unit testing", skill.Description)
	assert.Equal(t, []string{"Read", "Bash"}, skill.AllowedTools)
	assert.Equal(t, "1.0", skill.Version)
	assert.Contains(t, skill.Content, "# Test Skill")
	assert.Contains(t, skill.Content, "Use this skill for testing.")
	assert.Equal(t, skillDir, skill.Path)
	assert.True(t, skill.HasReferences)
	assert.True(t, skill.HasScripts)
	assert.False(t, skill.HasAssets)
}

func TestParseSkillMD_DirectFile(t *testing.T) {
	tmpDir := t.TempDir()
	skillDir := filepath.Join(tmpDir, "skill")
	require.NoError(t, os.MkdirAll(skillDir, 0755))

	content := `---
name: Direct File Skill
description: Test direct file parsing
---

Content here.
`
	filePath := filepath.Join(skillDir, "SKILL.md")
	require.NoError(t, os.WriteFile(filePath, []byte(content), 0644))

	// Parse by file path directly
	skill, err := ParseSkillMD(filePath)
	require.NoError(t, err)

	assert.Equal(t, "Direct File Skill", skill.Name)
	assert.Equal(t, skillDir, skill.Path)
}

func TestParseSkillMD_NoFrontmatter(t *testing.T) {
	tmpDir := t.TempDir()
	content := `# Just Markdown

No frontmatter here.
`
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "SKILL.md"), []byte(content), 0644))

	_, err := ParseSkillMD(tmpDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must start with YAML frontmatter")
}

func TestParseSkillMD_UnclosedFrontmatter(t *testing.T) {
	tmpDir := t.TempDir()
	content := `---
name: Broken
description: Missing closing delimiter
`
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "SKILL.md"), []byte(content), 0644))

	_, err := ParseSkillMD(tmpDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "frontmatter not closed")
}

func TestWriteSkillMD(t *testing.T) {
	tmpDir := t.TempDir()
	skillDir := filepath.Join(tmpDir, "new-skill")

	skill := &Skill{
		Name:         "Written Skill",
		Description:  "A skill written by test",
		AllowedTools: []string{"Read", "Write"},
		Version:      "2.0",
		Content:      "# My Skill\n\nThis is the content.",
	}

	err := WriteSkillMD(skill, skillDir)
	require.NoError(t, err)

	// Verify file exists
	filePath := filepath.Join(skillDir, "SKILL.md")
	assert.FileExists(t, filePath)

	// Read back and verify
	parsed, err := ParseSkillMD(skillDir)
	require.NoError(t, err)

	assert.Equal(t, skill.Name, parsed.Name)
	assert.Equal(t, skill.Description, parsed.Description)
	assert.Equal(t, skill.AllowedTools, parsed.AllowedTools)
	assert.Equal(t, skill.Version, parsed.Version)
	assert.Equal(t, skill.Content, parsed.Content)
}

func TestWriteSkillMD_Validation(t *testing.T) {
	tmpDir := t.TempDir()

	// Missing name
	skill := &Skill{
		Description: "Has description but no name",
		Content:     "Content",
	}
	err := WriteSkillMD(skill, tmpDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "name is required")

	// Missing description
	skill = &Skill{
		Name:    "Has name",
		Content: "Content",
	}
	err = WriteSkillMD(skill, tmpDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "description is required")
}

func TestDiscoverSkills(t *testing.T) {
	tmpDir := t.TempDir()
	skillsDir := filepath.Join(tmpDir, "skills")
	require.NoError(t, os.MkdirAll(skillsDir, 0755))

	// Create skill 1
	skill1Dir := filepath.Join(skillsDir, "alpha-skill")
	require.NoError(t, os.MkdirAll(skill1Dir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(skill1Dir, "SKILL.md"), []byte(`---
name: Alpha Skill
description: First skill
---

Content A
`), 0644))

	// Create skill 2
	skill2Dir := filepath.Join(skillsDir, "beta-skill")
	require.NoError(t, os.MkdirAll(skill2Dir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(skill2Dir, "SKILL.md"), []byte(`---
name: Beta Skill
description: Second skill
---

Content B
`), 0644))

	// Create a directory without SKILL.md (should be skipped)
	require.NoError(t, os.MkdirAll(filepath.Join(skillsDir, "not-a-skill"), 0755))

	// Discover skills
	skills, err := DiscoverSkills(tmpDir)
	require.NoError(t, err)

	assert.Len(t, skills, 2)
	assert.Equal(t, "Alpha Skill", skills[0].Name) // Sorted alphabetically
	assert.Equal(t, "Beta Skill", skills[1].Name)
}

func TestDiscoverSkills_NoSkillsDir(t *testing.T) {
	tmpDir := t.TempDir()

	skills, err := DiscoverSkills(tmpDir)
	require.NoError(t, err)
	assert.Nil(t, skills)
}

func TestListSkillResources(t *testing.T) {
	tmpDir := t.TempDir()
	skillDir := filepath.Join(tmpDir, "skill")
	refsDir := filepath.Join(skillDir, "references")
	require.NoError(t, os.MkdirAll(refsDir, 0755))

	// Create some files
	require.NoError(t, os.WriteFile(filepath.Join(refsDir, "doc1.md"), []byte(""), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(refsDir, "doc2.md"), []byte(""), 0644))

	files, err := ListSkillResources(skillDir, "references")
	require.NoError(t, err)

	assert.Equal(t, []string{"doc1.md", "doc2.md"}, files)
}

func TestListSkillResources_NotExists(t *testing.T) {
	tmpDir := t.TempDir()

	files, err := ListSkillResources(tmpDir, "references")
	require.NoError(t, err)
	assert.Nil(t, files)
}

func TestSkill_Validate(t *testing.T) {
	tests := []struct {
		name    string
		skill   Skill
		wantErr string
	}{
		{
			name:    "valid",
			skill:   Skill{Name: "Test", Description: "Desc"},
			wantErr: "",
		},
		{
			name:    "missing name",
			skill:   Skill{Description: "Desc"},
			wantErr: "name is required",
		},
		{
			name:    "missing description",
			skill:   Skill{Name: "Test"},
			wantErr: "description is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.skill.Validate()
			if tt.wantErr == "" {
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, tt.wantErr)
			}
		})
	}
}

func TestSkill_Info(t *testing.T) {
	skill := &Skill{
		Name:        "Test",
		Description: "Test description",
		Path:        "/some/path",
		Content:     "Some content",
	}

	info := skill.Info()
	assert.Equal(t, "Test", info.Name)
	assert.Equal(t, "Test description", info.Description)
	assert.Equal(t, "/some/path", info.Path)
}
