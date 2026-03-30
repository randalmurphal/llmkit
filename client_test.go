package llmkit

import "testing"

func TestCapabilitiesHasTool(t *testing.T) {
	caps := Capabilities{
		Runtime: RuntimeCapabilities{
			NativeTools: []string{"Read", "Write", "Bash"},
		},
	}
	if !caps.HasTool("Read") {
		t.Fatal("expected Read to be supported")
	}
	if caps.HasTool("read") {
		t.Fatal("expected HasTool to be case-sensitive")
	}
}

func TestCodexCapabilitiesReportEnvironmentSurface(t *testing.T) {
	if !CodexCapabilities.Environment.Hooks || !CodexCapabilities.Environment.MCP {
		t.Fatal("expected Codex environment capabilities to include hooks and MCP")
	}
	if CodexCapabilities.Environment.ConfigFile != "config.toml" {
		t.Fatalf("ConfigFile = %q", CodexCapabilities.Environment.ConfigFile)
	}
}
