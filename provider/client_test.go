package provider

import "testing"

func TestCapabilities_HasTool(t *testing.T) {
	caps := Capabilities{
		NativeTools: []string{"Read", "Write", "Edit", "Bash"},
	}

	tests := []struct {
		name string
		tool string
		want bool
	}{
		{"existing tool Read", "Read", true},
		{"existing tool Bash", "Bash", true},
		{"missing tool Glob", "Glob", false},
		{"case sensitive", "read", false},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := caps.HasTool(tt.tool); got != tt.want {
				t.Errorf("HasTool(%q) = %v, want %v", tt.tool, got, tt.want)
			}
		})
	}
}

func TestCapabilities_HasTool_Empty(t *testing.T) {
	caps := Capabilities{NativeTools: nil}

	if caps.HasTool("Read") {
		t.Error("expected HasTool to return false for nil NativeTools")
	}
}

func TestClaudeCapabilities(t *testing.T) {
	caps := ClaudeCapabilities

	if !caps.Streaming {
		t.Error("Claude should support streaming")
	}
	if !caps.Tools {
		t.Error("Claude should support tools")
	}
	if !caps.MCP {
		t.Error("Claude should support MCP")
	}
	if !caps.Sessions {
		t.Error("Claude should support sessions")
	}
	if !caps.Images {
		t.Error("Claude should support images")
	}
	if caps.ContextFile != "CLAUDE.md" {
		t.Errorf("expected ContextFile='CLAUDE.md', got %q", caps.ContextFile)
	}

	// Check some expected native tools
	expectedTools := []string{"Read", "Write", "Edit", "Bash", "Glob", "Grep"}
	for _, tool := range expectedTools {
		if !caps.HasTool(tool) {
			t.Errorf("Claude should have native tool %q", tool)
		}
	}
}

func TestGeminiCapabilities(t *testing.T) {
	caps := GeminiCapabilities

	if !caps.Streaming {
		t.Error("Gemini should support streaming")
	}
	if !caps.MCP {
		t.Error("Gemini should support MCP")
	}
	if caps.Sessions {
		t.Error("Gemini should not support sessions")
	}
	if caps.ContextFile != "GEMINI.md" {
		t.Errorf("expected ContextFile='GEMINI.md', got %q", caps.ContextFile)
	}

	// Check some expected native tools
	if !caps.HasTool("read_file") {
		t.Error("Gemini should have native tool 'read_file'")
	}
	if !caps.HasTool("google_web_search") {
		t.Error("Gemini should have native tool 'google_web_search'")
	}
}

func TestLocalCapabilities(t *testing.T) {
	caps := LocalCapabilities

	if !caps.Streaming {
		t.Error("Local should support streaming")
	}
	if caps.Tools {
		t.Error("Local should not support tools natively")
	}
	if !caps.MCP {
		t.Error("Local should support MCP")
	}
	if len(caps.NativeTools) != 0 {
		t.Errorf("Local should have no native tools, got %v", caps.NativeTools)
	}
}
