package claudecontract

import (
	"strings"
	"testing"
)

// TestFlagNameFormat validates all flag constants have correct format.
func TestFlagNameFormat(t *testing.T) {
	// All CLI flags should start with --
	flags := []string{
		FlagPrint,
		FlagOutputFormat,
		FlagInputFormat,
		FlagModel,
		FlagFallbackModel,
		FlagSystemPrompt,
		FlagAppendSystemPrompt,
		FlagSessionID,
		FlagResume,
		FlagContinue,
		FlagNoSessionPersistence,
		FlagAllowedTools,
		FlagDisallowedTools,
		FlagTools,
		FlagDangerouslySkipPermissions,
		FlagAllowDangerouslySkipPermissions,
		FlagPermissionMode,
		FlagSettingSources,
		FlagSettings,
		FlagAddDir,
		FlagMCPConfig,
		FlagStrictMCPConfig,
		FlagMCPDebug,
		FlagMaxBudgetUSD,
		FlagJSONSchema,
		FlagVerbose,
		FlagDebug,
		FlagHelp,
		FlagVersion,
		FlagAgent,
		FlagAgents,
		FlagForkSession,
		FlagBetas,
		FlagChrome,
		FlagNoChrome,
		FlagDisableSlashCommands,
		FlagFile,
		FlagIDE,
		FlagIncludePartialMessages,
		FlagReplayUserMessages,
		FlagPluginDir,
		FlagDebugFile,
		FlagEffort,
		FlagFromPR,
	}

	for _, flag := range flags {
		if !strings.HasPrefix(flag, "--") {
			t.Errorf("Flag %q should start with '--'", flag)
		}
		if strings.HasPrefix(flag, "---") {
			t.Errorf("Flag %q should not start with '---'", flag)
		}
	}
}

// TestCriticalFlagValues validates critical flag constant values.
func TestCriticalFlagValues(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{"FlagPrint", FlagPrint, "--print"},
		{"FlagOutputFormat", FlagOutputFormat, "--output-format"},
		{"FlagInputFormat", FlagInputFormat, "--input-format"},
		{"FlagModel", FlagModel, "--model"},
		{"FlagSystemPrompt", FlagSystemPrompt, "--system-prompt"},
		{"FlagSessionID", FlagSessionID, "--session-id"},
		{"FlagResume", FlagResume, "--resume"},
		{"FlagContinue", FlagContinue, "--continue"},
		{"FlagAllowedTools", FlagAllowedTools, "--allowedTools"},
		{"FlagDisallowedTools", FlagDisallowedTools, "--disallowedTools"},
		{"FlagDangerouslySkipPermissions", FlagDangerouslySkipPermissions, "--dangerously-skip-permissions"},
		{"FlagPermissionMode", FlagPermissionMode, "--permission-mode"},
		{"FlagMaxBudgetUSD", FlagMaxBudgetUSD, "--max-budget-usd"},
		{"FlagJSONSchema", FlagJSONSchema, "--json-schema"},
		{"FlagVerbose", FlagVerbose, "--verbose"},
		{"FlagMCPConfig", FlagMCPConfig, "--mcp-config"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("%s = %q, want %q", tt.name, tt.constant, tt.expected)
			}
		})
	}
}

// TestToolFlagsCamelCase validates tool flags use camelCase.
func TestToolFlagsCamelCase(t *testing.T) {
	// The CLI uses camelCase for tool flags (unusual for CLI but documented)
	if FlagAllowedTools != "--allowedTools" {
		t.Errorf("FlagAllowedTools = %q, want %q (camelCase)", FlagAllowedTools, "--allowedTools")
	}
	if FlagDisallowedTools != "--disallowedTools" {
		t.Errorf("FlagDisallowedTools = %q, want %q (camelCase)", FlagDisallowedTools, "--disallowedTools")
	}
}

// TestSessionManagementFlags validates session-related flags.
func TestSessionManagementFlags(t *testing.T) {
	// Session management flags
	sessionFlags := []struct {
		name     string
		flag     string
		expected string
	}{
		{"SessionID", FlagSessionID, "--session-id"},
		{"Resume", FlagResume, "--resume"},
		{"Continue", FlagContinue, "--continue"},
		{"ForkSession", FlagForkSession, "--fork-session"},
		{"NoSessionPersistence", FlagNoSessionPersistence, "--no-session-persistence"},
	}

	for _, sf := range sessionFlags {
		t.Run(sf.name, func(t *testing.T) {
			if sf.flag != sf.expected {
				t.Errorf("%s = %q, want %q", sf.name, sf.flag, sf.expected)
			}
		})
	}
}

// TestOutputFormatFlags validates output format flag values.
func TestOutputFormatFlags(t *testing.T) {
	// --output-format accepts: text, json, stream-json
	validFormats := []string{FormatText, FormatJSON, FormatStreamJSON}
	expectedValues := []string{"text", "json", "stream-json"}

	for i, format := range validFormats {
		if format != expectedValues[i] {
			t.Errorf("Format %d = %q, want %q", i, format, expectedValues[i])
		}
	}
}

// TestInputFormatFlags validates input format flag values.
func TestInputFormatFlags(t *testing.T) {
	// --input-format accepts: text, stream-json
	if FormatText != "text" {
		t.Errorf("FormatText = %q, want %q", FormatText, "text")
	}
	if FormatStreamJSON != "stream-json" {
		t.Errorf("FormatStreamJSON = %q, want %q", FormatStreamJSON, "stream-json")
	}
}

// TestPermissionModeFlags validates permission mode flag values.
func TestPermissionModeFlags(t *testing.T) {
	// --permission-mode accepts these values
	modes := []PermissionMode{
		PermissionDefault,
		PermissionAcceptEdits,
		PermissionBypassPermissions,
		PermissionPlan,
	}

	expectedValues := []string{
		"default",
		"acceptEdits",
		"bypassPermissions",
		"plan",
	}

	for i, mode := range modes {
		if string(mode) != expectedValues[i] {
			t.Errorf("PermissionMode %d = %q, want %q", i, mode, expectedValues[i])
		}
	}
}

// TestMCPFlags validates MCP-related flags.
func TestMCPFlags(t *testing.T) {
	mcpFlags := []struct {
		name     string
		flag     string
		expected string
	}{
		{"MCPConfig", FlagMCPConfig, "--mcp-config"},
		{"StrictMCPConfig", FlagStrictMCPConfig, "--strict-mcp-config"},
		{"MCPDebug", FlagMCPDebug, "--mcp-debug"},
	}

	for _, mf := range mcpFlags {
		t.Run(mf.name, func(t *testing.T) {
			if mf.flag != mf.expected {
				t.Errorf("%s = %q, want %q", mf.name, mf.flag, mf.expected)
			}
		})
	}
}

// TestBooleanFlags validates flags that take no value.
func TestBooleanFlags(t *testing.T) {
	// These flags are boolean - presence means true
	booleanFlags := []string{
		FlagPrint,
		FlagContinue,
		FlagDangerouslySkipPermissions,
		FlagAllowDangerouslySkipPermissions,
		FlagNoSessionPersistence,
		FlagStrictMCPConfig,
		FlagVerbose,
		FlagChrome,
		FlagNoChrome,
		FlagDisableSlashCommands,
		FlagIDE,
		FlagIncludePartialMessages,
		FlagReplayUserMessages,
		FlagForkSession,
	}

	// All should be valid flag format
	for _, flag := range booleanFlags {
		if !strings.HasPrefix(flag, "--") {
			t.Errorf("Boolean flag %q should start with '--'", flag)
		}
	}
}

// TestValueFlags validates flags that require a value.
func TestValueFlags(t *testing.T) {
	// These flags require a value
	valueFlags := []struct {
		flag        string
		description string
	}{
		{FlagOutputFormat, "output format (text/json/stream-json)"},
		{FlagInputFormat, "input format (text/stream-json)"},
		{FlagModel, "model name"},
		{FlagFallbackModel, "fallback model name"},
		{FlagSystemPrompt, "system prompt text"},
		{FlagAppendSystemPrompt, "additional system prompt"},
		{FlagSessionID, "session UUID"},
		{FlagResume, "session ID to resume"},
		{FlagPermissionMode, "permission mode"},
		{FlagSettingSources, "comma-separated sources"},
		{FlagSettings, "settings file path or JSON"},
		{FlagMaxBudgetUSD, "dollar amount"},
		{FlagJSONSchema, "JSON schema string"},
		{FlagAgent, "agent name"},
		{FlagAgents, "agents JSON"},
		{FlagDebug, "debug filter"},
		{FlagBetas, "beta headers"},
		{FlagFile, "file specs"},
	}

	for _, vf := range valueFlags {
		t.Run(vf.flag, func(t *testing.T) {
			if !strings.HasPrefix(vf.flag, "--") {
				t.Errorf("Value flag %q should start with '--'", vf.flag)
			}
		})
	}
}

// TestRepeatableFlags validates flags that can be specified multiple times.
func TestRepeatableFlags(t *testing.T) {
	// These flags can be specified multiple times
	repeatableFlags := []string{
		FlagAllowedTools,    // --allowedTools can appear multiple times
		FlagDisallowedTools, // --disallowedTools can appear multiple times
		FlagAddDir,          // --add-dir can appear multiple times
		FlagMCPConfig,       // --mcp-config can appear multiple times
		FlagPluginDir,       // --plugin-dir can appear multiple times
		FlagFile,            // --file can appear multiple times
		FlagBetas,           // --betas can appear multiple times
	}

	for _, flag := range repeatableFlags {
		if !strings.HasPrefix(flag, "--") {
			t.Errorf("Repeatable flag %q should start with '--'", flag)
		}
	}
}
