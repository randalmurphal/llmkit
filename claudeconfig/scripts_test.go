package claudeconfig

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProjectScript_Validate(t *testing.T) {
	tests := []struct {
		name    string
		script  ProjectScript
		wantErr error
	}{
		{
			name:    "valid",
			script:  ProjectScript{Name: "test", Path: "scripts/test.sh", Description: "A test script"},
			wantErr: nil,
		},
		{
			name:    "missing name",
			script:  ProjectScript{Path: "scripts/test.sh", Description: "A test script"},
			wantErr: ErrScriptNameRequired,
		},
		{
			name:    "missing path",
			script:  ProjectScript{Name: "test", Description: "A test script"},
			wantErr: ErrScriptPathRequired,
		},
		{
			name:    "missing description",
			script:  ProjectScript{Name: "test", Path: "scripts/test.sh"},
			wantErr: ErrScriptDescriptionRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.script.Validate()
			if tt.wantErr == nil {
				assert.NoError(t, err)
			} else {
				assert.ErrorIs(t, err, tt.wantErr)
			}
		})
	}
}

func TestScriptService_CRUD(t *testing.T) {
	tmpDir := t.TempDir()
	svc := NewScriptService(tmpDir)

	// List empty
	scripts, err := svc.List()
	require.NoError(t, err)
	assert.Empty(t, scripts)

	// Create
	script1 := ProjectScript{
		Name:        "build",
		Path:        ".claude/scripts/build.sh",
		Description: "Build the project",
		Language:    "bash",
	}
	err = svc.Create(script1)
	require.NoError(t, err)

	script2 := ProjectScript{
		Name:        "test",
		Path:        ".claude/scripts/test.py",
		Description: "Run tests",
		Language:    "python",
	}
	err = svc.Create(script2)
	require.NoError(t, err)

	// List
	scripts, err = svc.List()
	require.NoError(t, err)
	assert.Len(t, scripts, 2)
	assert.Equal(t, "build", scripts[0].Name) // Sorted

	// Get
	script, err := svc.Get("build")
	require.NoError(t, err)
	assert.Equal(t, "Build the project", script.Description)
	assert.Equal(t, "bash", script.Language)

	// Get non-existent
	_, err = svc.Get("nonexistent")
	assert.ErrorIs(t, err, ErrScriptNotFound)

	// Update
	script1.Description = "Build the entire project"
	err = svc.Update("build", script1)
	require.NoError(t, err)

	script, err = svc.Get("build")
	require.NoError(t, err)
	assert.Equal(t, "Build the entire project", script.Description)

	// Delete
	err = svc.Delete("build")
	require.NoError(t, err)

	scripts, err = svc.List()
	require.NoError(t, err)
	assert.Len(t, scripts, 1)

	// Delete non-existent
	err = svc.Delete("build")
	assert.ErrorIs(t, err, ErrScriptNotFound)

	// Exists
	assert.True(t, svc.Exists("test"))
	assert.False(t, svc.Exists("build"))
}

func TestScriptService_CreateDuplicate(t *testing.T) {
	tmpDir := t.TempDir()
	svc := NewScriptService(tmpDir)

	script := ProjectScript{Name: "test", Path: "test.sh", Description: "Test"}
	err := svc.Create(script)
	require.NoError(t, err)

	err = svc.Create(script)
	assert.ErrorIs(t, err, ErrScriptAlreadyExists)
}

func TestScriptService_AutoDetectLanguage(t *testing.T) {
	tmpDir := t.TempDir()
	svc := NewScriptService(tmpDir)

	// Create script without language
	script := ProjectScript{
		Name:        "analyze",
		Path:        "scripts/analyze.py",
		Description: "Analyze data",
		// Language not set
	}
	err := svc.Create(script)
	require.NoError(t, err)

	got, err := svc.Get("analyze")
	require.NoError(t, err)
	assert.Equal(t, "python", got.Language)
}

func TestScriptService_Discover(t *testing.T) {
	tmpDir := t.TempDir()
	scriptsDir := filepath.Join(tmpDir, ".claude", "scripts")
	require.NoError(t, os.MkdirAll(scriptsDir, 0755))

	// Create some scripts
	script1 := `#!/bin/bash
# Build the project
echo "Building..."
`
	require.NoError(t, os.WriteFile(filepath.Join(scriptsDir, "build.sh"), []byte(script1), 0755))

	script2 := `#!/usr/bin/env python3
# Run the test suite
import unittest
`
	require.NoError(t, os.WriteFile(filepath.Join(scriptsDir, "test.py"), []byte(script2), 0644))

	svc := NewScriptService(tmpDir)
	discovered, err := svc.Discover()
	require.NoError(t, err)

	assert.Len(t, discovered, 2)

	// Check build script
	var buildScript *ProjectScript
	for i := range discovered {
		if discovered[i].Name == "build" {
			buildScript = &discovered[i]
			break
		}
	}
	require.NotNil(t, buildScript)
	assert.Equal(t, "bash", buildScript.Language)
	assert.Equal(t, "Build the project", buildScript.Description)
	assert.True(t, buildScript.Executable)

	// Check test script
	var testScript *ProjectScript
	for i := range discovered {
		if discovered[i].Name == "test" {
			testScript = &discovered[i]
			break
		}
	}
	require.NotNil(t, testScript)
	assert.Equal(t, "python", testScript.Language)
	assert.Equal(t, "Run the test suite", testScript.Description)
	assert.False(t, testScript.Executable)
}

func TestScriptService_DiscoverNoDir(t *testing.T) {
	tmpDir := t.TempDir()
	svc := NewScriptService(tmpDir)

	discovered, err := svc.Discover()
	require.NoError(t, err)
	assert.Nil(t, discovered)
}

func TestScriptService_WithCustomExtension(t *testing.T) {
	tmpDir := t.TempDir()
	svc := NewScriptService(tmpDir, WithScriptExtensionName("project_scripts"))

	script := ProjectScript{Name: "test", Path: "test.sh", Description: "Test"}
	err := svc.Create(script)
	require.NoError(t, err)

	// Verify it's stored in custom extension
	settings, err := LoadProjectSettings(tmpDir)
	require.NoError(t, err)

	var scripts []ProjectScript
	err = settings.GetExtension("project_scripts", &scripts)
	require.NoError(t, err)
	assert.Len(t, scripts, 1)
}

func TestDetectLanguage(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"script.py", "python"},
		{"script.sh", "bash"},
		{"script.bash", "bash"},
		{"script.go", "go"},
		{"script.js", "javascript"},
		{"script.ts", "typescript"},
		{"script.rb", "ruby"},
		{"script.pl", "perl"},
		{"script.php", "php"},
		{"script.rs", "rust"},
		{"script.unknown", ""},
		{"script", ""},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			assert.Equal(t, tt.expected, detectLanguage(tt.path))
		})
	}
}

func TestExtractScriptDescription(t *testing.T) {
	tmpDir := t.TempDir()

	// Bash script with description
	bashScript := `#!/bin/bash
# This is the description
echo "Hello"
`
	bashPath := filepath.Join(tmpDir, "bash.sh")
	require.NoError(t, os.WriteFile(bashPath, []byte(bashScript), 0644))
	assert.Equal(t, "This is the description", extractScriptDescription(bashPath))

	// Python script with description
	pyScript := `#!/usr/bin/env python3
# Python script description
import os
`
	pyPath := filepath.Join(tmpDir, "script.py")
	require.NoError(t, os.WriteFile(pyPath, []byte(pyScript), 0644))
	assert.Equal(t, "Python script description", extractScriptDescription(pyPath))

	// Go file with description
	goScript := `// Go script description
package main
`
	goPath := filepath.Join(tmpDir, "script.go")
	require.NoError(t, os.WriteFile(goPath, []byte(goScript), 0644))
	assert.Equal(t, "Go script description", extractScriptDescription(goPath))

	// No description
	noDesc := `#!/bin/bash
echo "No description"
`
	noDescPath := filepath.Join(tmpDir, "nodesc.sh")
	require.NoError(t, os.WriteFile(noDescPath, []byte(noDesc), 0644))
	assert.Equal(t, "", extractScriptDescription(noDescPath))

	// Non-existent file
	assert.Equal(t, "", extractScriptDescription("/nonexistent"))
}
