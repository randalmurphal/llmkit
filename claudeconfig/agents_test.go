package claudeconfig

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSubAgent_Validate(t *testing.T) {
	tests := []struct {
		name    string
		agent   SubAgent
		wantErr error
	}{
		{
			name:    "valid",
			agent:   SubAgent{Name: "test-agent", Description: "A test agent"},
			wantErr: nil,
		},
		{
			name:    "missing name",
			agent:   SubAgent{Description: "A test agent"},
			wantErr: ErrSubAgentNameRequired,
		},
		{
			name:    "missing description",
			agent:   SubAgent{Name: "test-agent"},
			wantErr: ErrSubAgentDescriptionRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.agent.Validate()
			if tt.wantErr == nil {
				assert.NoError(t, err)
			} else {
				assert.ErrorIs(t, err, tt.wantErr)
			}
		})
	}
}

func TestAgentService_CRUD(t *testing.T) {
	tmpDir := t.TempDir()
	svc := NewAgentService(tmpDir)

	// List empty
	agents, err := svc.List()
	require.NoError(t, err)
	assert.Empty(t, agents)

	// Create
	agent1 := SubAgent{
		Name:        "builder",
		Description: "Builds things",
		Model:       "sonnet",
		Tools:       &ToolPermissions{Allow: []string{"Read", "Write", "Bash"}},
	}
	err = svc.Create(agent1)
	require.NoError(t, err)

	agent2 := SubAgent{
		Name:        "reviewer",
		Description: "Reviews code",
		Model:       "opus",
	}
	err = svc.Create(agent2)
	require.NoError(t, err)

	// List
	agents, err = svc.List()
	require.NoError(t, err)
	assert.Len(t, agents, 2)
	assert.Equal(t, "builder", agents[0].Name) // Sorted

	// Get
	agent, err := svc.Get("builder")
	require.NoError(t, err)
	assert.Equal(t, "Builds things", agent.Description)
	assert.Equal(t, "sonnet", agent.Model)

	// Get non-existent
	_, err = svc.Get("nonexistent")
	assert.ErrorIs(t, err, ErrSubAgentNotFound)

	// Update
	agent1.Description = "Builds amazing things"
	err = svc.Update("builder", agent1)
	require.NoError(t, err)

	agent, err = svc.Get("builder")
	require.NoError(t, err)
	assert.Equal(t, "Builds amazing things", agent.Description)

	// Update non-existent
	err = svc.Update("nonexistent", agent1)
	assert.ErrorIs(t, err, ErrSubAgentNotFound)

	// Delete
	err = svc.Delete("builder")
	require.NoError(t, err)

	agents, err = svc.List()
	require.NoError(t, err)
	assert.Len(t, agents, 1)

	// Delete non-existent
	err = svc.Delete("builder")
	assert.ErrorIs(t, err, ErrSubAgentNotFound)

	// Exists
	assert.True(t, svc.Exists("reviewer"))
	assert.False(t, svc.Exists("builder"))
}

func TestAgentService_CreateDuplicate(t *testing.T) {
	tmpDir := t.TempDir()
	svc := NewAgentService(tmpDir)

	agent := SubAgent{Name: "test", Description: "Test agent"}
	err := svc.Create(agent)
	require.NoError(t, err)

	err = svc.Create(agent)
	assert.ErrorIs(t, err, ErrSubAgentAlreadyExists)
}

func TestAgentService_UpdateRename(t *testing.T) {
	tmpDir := t.TempDir()
	svc := NewAgentService(tmpDir)

	// Create original
	err := svc.Create(SubAgent{Name: "original", Description: "Original agent"})
	require.NoError(t, err)

	// Create another
	err = svc.Create(SubAgent{Name: "other", Description: "Other agent"})
	require.NoError(t, err)

	// Rename to non-conflicting name
	err = svc.Update("original", SubAgent{Name: "renamed", Description: "Renamed agent"})
	require.NoError(t, err)

	assert.False(t, svc.Exists("original"))
	assert.True(t, svc.Exists("renamed"))

	// Try to rename to existing name
	err = svc.Update("renamed", SubAgent{Name: "other", Description: "Conflict"})
	assert.ErrorIs(t, err, ErrSubAgentAlreadyExists)
}

func TestAgentService_WithCustomExtension(t *testing.T) {
	tmpDir := t.TempDir()
	svc := NewAgentService(tmpDir, WithAgentExtensionName("custom_agents"))

	agent := SubAgent{Name: "test", Description: "Test"}
	err := svc.Create(agent)
	require.NoError(t, err)

	// Verify it's stored in custom extension
	settings, err := LoadProjectSettings(tmpDir)
	require.NoError(t, err)

	var agents []SubAgent
	err = settings.GetExtension("custom_agents", &agents)
	require.NoError(t, err)
	assert.Len(t, agents, 1)
}

func TestAgentService_PersistsToDisk(t *testing.T) {
	tmpDir := t.TempDir()

	// Create with one service instance
	svc1 := NewAgentService(tmpDir)
	err := svc1.Create(SubAgent{
		Name:        "persistent",
		Description: "Should persist",
		Model:       "haiku",
		SkillRefs:   []string{"python-style", "testing"},
	})
	require.NoError(t, err)

	// Verify file exists
	settingsPath := filepath.Join(tmpDir, ".claude", "settings.json")
	assert.FileExists(t, settingsPath)

	// Read with new service instance
	svc2 := NewAgentService(tmpDir)
	agent, err := svc2.Get("persistent")
	require.NoError(t, err)
	assert.Equal(t, "haiku", agent.Model)
	assert.Equal(t, []string{"python-style", "testing"}, agent.SkillRefs)
}

func TestAgentService_ValidationOnCreate(t *testing.T) {
	tmpDir := t.TempDir()
	svc := NewAgentService(tmpDir)

	err := svc.Create(SubAgent{Name: "", Description: "Missing name"})
	assert.ErrorIs(t, err, ErrSubAgentNameRequired)

	err = svc.Create(SubAgent{Name: "test", Description: ""})
	assert.ErrorIs(t, err, ErrSubAgentDescriptionRequired)
}

func TestAgentService_EmptyProjectDir(t *testing.T) {
	tmpDir := t.TempDir()

	// Remove any existing .claude directory
	os.RemoveAll(filepath.Join(tmpDir, ".claude"))

	svc := NewAgentService(tmpDir)

	// Should work even without existing settings
	agents, err := svc.List()
	require.NoError(t, err)
	assert.Empty(t, agents)
}
