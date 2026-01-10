package claudeconfig

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMCPServer_GetTransportType(t *testing.T) {
	tests := []struct {
		name     string
		server   MCPServer
		expected string
	}{
		{
			name:     "explicit stdio",
			server:   MCPServer{Type: "stdio", Command: "npx"},
			expected: "stdio",
		},
		{
			name:     "explicit http",
			server:   MCPServer{Type: "http", URL: "https://example.com"},
			expected: "http",
		},
		{
			name:     "explicit sse",
			server:   MCPServer{Type: "sse", URL: "https://example.com/sse"},
			expected: "sse",
		},
		{
			name:     "inferred stdio from command",
			server:   MCPServer{Command: "npx"},
			expected: "stdio",
		},
		{
			name:     "inferred http from url",
			server:   MCPServer{URL: "https://example.com"},
			expected: "http",
		},
		{
			name:     "default stdio",
			server:   MCPServer{},
			expected: "stdio",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.server.GetTransportType()
			if got != tt.expected {
				t.Errorf("GetTransportType() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestMCPServer_IsValid(t *testing.T) {
	tests := []struct {
		name    string
		server  MCPServer
		wantErr bool
	}{
		{
			name:    "valid stdio",
			server:  MCPServer{Type: "stdio", Command: "npx", Args: []string{"-y", "@pkg/name"}},
			wantErr: false,
		},
		{
			name:    "valid http",
			server:  MCPServer{Type: "http", URL: "https://example.com/mcp"},
			wantErr: false,
		},
		{
			name:    "valid sse",
			server:  MCPServer{Type: "sse", URL: "https://example.com/sse"},
			wantErr: false,
		},
		{
			name:    "stdio missing command",
			server:  MCPServer{Type: "stdio"},
			wantErr: true,
		},
		{
			name:    "http missing url",
			server:  MCPServer{Type: "http"},
			wantErr: true,
		},
		{
			name:    "sse missing url",
			server:  MCPServer{Type: "sse"},
			wantErr: true,
		},
		{
			name:    "invalid transport type",
			server:  MCPServer{Type: "invalid"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.server.IsValid()
			if (err != nil) != tt.wantErr {
				t.Errorf("IsValid() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMCPServer_Clone(t *testing.T) {
	original := &MCPServer{
		Type:     "stdio",
		Command:  "npx",
		Args:     []string{"-y", "@pkg/name"},
		Env:      map[string]string{"KEY": "value"},
		Headers:  []string{"Authorization: Bearer token"},
		Disabled: true,
	}

	clone := original.Clone()

	// Verify values are equal
	if clone.Type != original.Type {
		t.Errorf("Type = %q, want %q", clone.Type, original.Type)
	}
	if clone.Command != original.Command {
		t.Errorf("Command = %q, want %q", clone.Command, original.Command)
	}
	if clone.Disabled != original.Disabled {
		t.Errorf("Disabled = %v, want %v", clone.Disabled, original.Disabled)
	}

	// Verify slices are independent
	original.Args[0] = "changed"
	if clone.Args[0] == "changed" {
		t.Error("Args slice is not independent")
	}

	// Verify maps are independent
	original.Env["KEY"] = "changed"
	if clone.Env["KEY"] == "changed" {
		t.Error("Env map is not independent")
	}
}

func TestMCPConfig_AddServer(t *testing.T) {
	config := NewMCPConfig()

	server := &MCPServer{
		Command: "npx",
		Args:    []string{"-y", "@pkg/name"},
	}

	err := config.AddServer("test", server)
	if err != nil {
		t.Fatalf("AddServer() error = %v", err)
	}

	got := config.GetServer("test")
	if got == nil {
		t.Fatal("GetServer() returned nil")
	}
	if got.Command != "npx" {
		t.Errorf("Command = %q, want %q", got.Command, "npx")
	}
}

func TestMCPConfig_AddServer_Validation(t *testing.T) {
	config := NewMCPConfig()

	tests := []struct {
		name    string
		srvName string
		server  *MCPServer
		wantErr bool
	}{
		{
			name:    "empty name",
			srvName: "",
			server:  &MCPServer{Command: "npx"},
			wantErr: true,
		},
		{
			name:    "nil server",
			srvName: "test",
			server:  nil,
			wantErr: true,
		},
		{
			name:    "invalid server",
			srvName: "test",
			server:  &MCPServer{Type: "http"}, // missing URL
			wantErr: true,
		},
		{
			name:    "valid server",
			srvName: "test",
			server:  &MCPServer{Command: "npx"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := config.AddServer(tt.srvName, tt.server)
			if (err != nil) != tt.wantErr {
				t.Errorf("AddServer() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMCPConfig_RemoveServer(t *testing.T) {
	config := NewMCPConfig()
	config.MCPServers["test"] = &MCPServer{Command: "npx"}

	// Remove existing
	if !config.RemoveServer("test") {
		t.Error("RemoveServer() returned false for existing server")
	}
	if config.GetServer("test") != nil {
		t.Error("Server still exists after removal")
	}

	// Remove non-existing
	if config.RemoveServer("nonexistent") {
		t.Error("RemoveServer() returned true for non-existing server")
	}
}

func TestMCPConfig_ListServers(t *testing.T) {
	config := NewMCPConfig()
	config.MCPServers["server1"] = &MCPServer{Command: "cmd1"}
	config.MCPServers["server2"] = &MCPServer{Command: "cmd2"}

	names := config.ListServers()
	if len(names) != 2 {
		t.Errorf("ListServers() returned %d names, want 2", len(names))
	}

	// Check both names exist (order not guaranteed)
	found := make(map[string]bool)
	for _, name := range names {
		found[name] = true
	}
	if !found["server1"] || !found["server2"] {
		t.Errorf("ListServers() = %v, want [server1, server2]", names)
	}
}

func TestLoadProjectMCPConfig(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()

	// Test loading non-existent file (should return empty config)
	config, err := LoadProjectMCPConfig(tmpDir)
	if err != nil {
		t.Fatalf("LoadProjectMCPConfig() error = %v", err)
	}
	if config == nil {
		t.Fatal("LoadProjectMCPConfig() returned nil")
	}
	if len(config.MCPServers) != 0 {
		t.Errorf("Expected empty servers, got %d", len(config.MCPServers))
	}

	// Create a valid .mcp.json file
	mcpJSON := `{
  "mcpServers": {
    "github": {
      "type": "stdio",
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-github"],
      "env": {
        "GITHUB_TOKEN": "${GITHUB_TOKEN}"
      }
    },
    "remote": {
      "type": "http",
      "url": "https://example.com/mcp",
      "headers": ["Authorization: Bearer ${TOKEN}"]
    }
  }
}`
	mcpPath := filepath.Join(tmpDir, ".mcp.json")
	if err := os.WriteFile(mcpPath, []byte(mcpJSON), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Load the file
	config, err = LoadProjectMCPConfig(tmpDir)
	if err != nil {
		t.Fatalf("LoadProjectMCPConfig() error = %v", err)
	}

	// Verify github server
	github := config.GetServer("github")
	if github == nil {
		t.Fatal("github server not found")
	}
	if github.Command != "npx" {
		t.Errorf("github.Command = %q, want %q", github.Command, "npx")
	}
	if github.GetTransportType() != "stdio" {
		t.Errorf("github transport = %q, want %q", github.GetTransportType(), "stdio")
	}

	// Verify remote server
	remote := config.GetServer("remote")
	if remote == nil {
		t.Fatal("remote server not found")
	}
	if remote.URL != "https://example.com/mcp" {
		t.Errorf("remote.URL = %q, want %q", remote.URL, "https://example.com/mcp")
	}
	if remote.GetTransportType() != "http" {
		t.Errorf("remote transport = %q, want %q", remote.GetTransportType(), "http")
	}
}

func TestSaveProjectMCPConfig(t *testing.T) {
	tmpDir := t.TempDir()

	config := NewMCPConfig()
	config.MCPServers["test"] = &MCPServer{
		Command: "npx",
		Args:    []string{"-y", "@pkg/name"},
		Env:     map[string]string{"KEY": "value"},
	}

	err := SaveProjectMCPConfig(tmpDir, config)
	if err != nil {
		t.Fatalf("SaveProjectMCPConfig() error = %v", err)
	}

	// Verify file exists
	mcpPath := MCPConfigPath(tmpDir)
	if _, err := os.Stat(mcpPath); os.IsNotExist(err) {
		t.Fatal(".mcp.json file not created")
	}

	// Load and verify
	loaded, err := LoadProjectMCPConfig(tmpDir)
	if err != nil {
		t.Fatalf("LoadProjectMCPConfig() error = %v", err)
	}

	server := loaded.GetServer("test")
	if server == nil {
		t.Fatal("test server not found after reload")
	}
	if server.Command != "npx" {
		t.Errorf("Command = %q, want %q", server.Command, "npx")
	}
}

func TestMCPConfig_Merge(t *testing.T) {
	base := NewMCPConfig()
	base.MCPServers["shared"] = &MCPServer{Command: "base-cmd"}
	base.MCPServers["base-only"] = &MCPServer{Command: "base-only-cmd"}

	override := NewMCPConfig()
	override.MCPServers["shared"] = &MCPServer{Command: "override-cmd"}
	override.MCPServers["override-only"] = &MCPServer{Command: "override-only-cmd"}

	merged := base.Merge(override)

	// shared should have override value
	if merged.GetServer("shared").Command != "override-cmd" {
		t.Error("shared server not overridden")
	}

	// base-only should exist
	if merged.GetServer("base-only") == nil {
		t.Error("base-only server missing")
	}

	// override-only should exist
	if merged.GetServer("override-only") == nil {
		t.Error("override-only server missing")
	}

	// Verify independence
	base.MCPServers["shared"].Command = "changed"
	if merged.GetServer("shared").Command == "changed" {
		t.Error("merged config not independent from base")
	}
}

func TestMCPConfig_GetServerInfo(t *testing.T) {
	config := NewMCPConfig()
	config.MCPServers["test"] = &MCPServer{
		Command:  "npx",
		Args:     []string{"-y", "@pkg/name"},
		Env:      map[string]string{"KEY": "value", "KEY2": "value2"},
		Disabled: true,
	}

	info := config.GetServerInfo("test")
	if info == nil {
		t.Fatal("GetServerInfo() returned nil")
	}

	if info.Name != "test" {
		t.Errorf("Name = %q, want %q", info.Name, "test")
	}
	if info.Type != "stdio" {
		t.Errorf("Type = %q, want %q", info.Type, "stdio")
	}
	if info.Command != "npx" {
		t.Errorf("Command = %q, want %q", info.Command, "npx")
	}
	if !info.Disabled {
		t.Error("Disabled should be true")
	}
	if !info.HasEnv {
		t.Error("HasEnv should be true")
	}
	if info.EnvCount != 2 {
		t.Errorf("EnvCount = %d, want 2", info.EnvCount)
	}
	if info.ArgsCount != 2 {
		t.Errorf("ArgsCount = %d, want 2", info.ArgsCount)
	}
}

func TestMCPConfigExists(t *testing.T) {
	tmpDir := t.TempDir()

	// Should not exist initially
	if MCPConfigExists(tmpDir) {
		t.Error("MCPConfigExists() returned true for non-existent file")
	}

	// Create file
	mcpPath := MCPConfigPath(tmpDir)
	if err := os.WriteFile(mcpPath, []byte("{}"), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Should exist now
	if !MCPConfigExists(tmpDir) {
		t.Error("MCPConfigExists() returned false for existing file")
	}
}
