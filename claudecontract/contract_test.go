package claudecontract

import (
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"
)

// TestVersionMatches verifies our tested version constant matches the installed CLI.
func TestVersionMatches(t *testing.T) {
	// Skip if claude CLI is not available
	claudePath, err := findClaudeCLI()
	if err != nil {
		t.Skip("Claude CLI not found, skipping version check")
	}

	version, err := DetectCLIVersion(claudePath)
	if err != nil {
		t.Fatalf("Failed to detect CLI version: %v", err)
	}

	// Check if we're on the tested version
	if version.Raw != TestedCLIVersion {
		t.Logf("WARNING: Installed CLI version %s differs from tested version %s", version.Raw, TestedCLIVersion)
		t.Logf("Some constants may be out of date. Run 'make update-golden' to update.")
	}
}

// TestFlagsExistInHelp validates that our flag constants appear in CLI --help output.
func TestFlagsExistInHelp(t *testing.T) {
	claudePath, err := findClaudeCLI()
	if err != nil {
		t.Skip("Claude CLI not found, skipping flag validation")
	}

	// Get help output
	cmd := claudeCmd(claudePath, "--help")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to get CLI help: %v", err)
	}
	helpText := string(output)

	// Critical flags that must exist
	criticalFlags := []string{
		FlagPrint,
		FlagOutputFormat,
		FlagInputFormat,
		FlagModel,
		FlagSystemPrompt,
		FlagSessionID,
		FlagResume,
		FlagContinue,
		FlagAllowedTools,
		FlagDisallowedTools,
		FlagDangerouslySkipPermissions,
		FlagPermissionMode,
		FlagMaxBudgetUSD,
		FlagMCPConfig,
		FlagJSONSchema,
		FlagVerbose,
	}

	for _, flag := range criticalFlags {
		if !strings.Contains(helpText, flag) {
			t.Errorf("Critical flag %q not found in CLI help output", flag)
		}
	}
}

// TestOutputFormatsValid validates our output format constants.
func TestOutputFormatsValid(t *testing.T) {
	claudePath, err := findClaudeCLI()
	if err != nil {
		t.Skip("Claude CLI not found, skipping format validation")
	}

	cmd := claudeCmd(claudePath, "--help")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to get CLI help: %v", err)
	}
	helpText := string(output)

	// Check output formats are documented
	formats := []string{FormatText, FormatJSON, FormatStreamJSON}
	for _, format := range formats {
		if !strings.Contains(helpText, format) {
			t.Errorf("Output format %q not found in CLI help output", format)
		}
	}
}

// TestPermissionModesValid validates our permission mode constants.
func TestPermissionModesValid(t *testing.T) {
	claudePath, err := findClaudeCLI()
	if err != nil {
		t.Skip("Claude CLI not found, skipping permission mode validation")
	}

	cmd := claudeCmd(claudePath, "--help")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to get CLI help: %v", err)
	}
	helpText := string(output)

	// Check permission modes are documented
	modes := []PermissionMode{
		PermissionDefault,
		PermissionAcceptEdits,
		PermissionBypassPermissions,
		PermissionPlan,
	}
	for _, mode := range modes {
		if mode == PermissionDefault {
			continue // "default" is the default, may not be explicitly listed
		}
		if !strings.Contains(helpText, string(mode)) {
			t.Errorf("Permission mode %q not found in CLI help output", mode)
		}
	}
}

// TestDetectNewFlags looks for flags in --help that we don't have constants for.
// This helps catch when the CLI adds new flags we should support.
func TestDetectNewFlags(t *testing.T) {
	claudePath, err := findClaudeCLI()
	if err != nil {
		t.Skip("Claude CLI not found, skipping new flag detection")
	}

	cmd := claudeCmd(claudePath, "--help")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to get CLI help: %v", err)
	}
	helpText := string(output)

	// Extract all --flag patterns from help output
	flagPattern := regexp.MustCompile(`--([a-zA-Z][a-zA-Z0-9-]*)`)
	matches := flagPattern.FindAllStringSubmatch(helpText, -1)

	// Known flags we have constants for
	knownFlags := map[string]bool{
		"print":                              true,
		"output-format":                      true,
		"input-format":                       true,
		"model":                              true,
		"fallback-model":                     true,
		"system-prompt":                      true,
		"append-system-prompt":               true,
		"session-id":                         true,
		"resume":                             true,
		"continue":                           true,
		"no-session-persistence":             true,
		"allowedTools":                       true,
		"allowed-tools":                      true, // alias
		"disallowedTools":                    true,
		"disallowed-tools":                   true, // alias
		"tools":                              true,
		"dangerously-skip-permissions":       true,
		"allow-dangerously-skip-permissions": true,
		"permission-mode":                    true,
		"setting-sources":                    true,
		"settings":                           true,
		"add-dir":                            true,
		"mcp-config":                         true,
		"strict-mcp-config":                  true,
		"mcp-debug":                          true,
		"max-budget-usd":                     true,
		"json-schema":                        true,
		"verbose":                            true,
		"debug":                              true,
		"help":                               true,
		"version":                            true,
		"agent":                              true,
		"agents":                             true,
		"fork-session":                       true,
		"betas":                              true,
		"chrome":                             true,
		"no-chrome":                          true,
		"disable-slash-commands":             true,
		"file":                               true,
		"ide":                                true,
		"include-partial-messages":           true,
		"replay-user-messages":               true,
		"plugin-dir":                         true,
		"debug-file":                         true,
		"effort":                             true,
		"from-pr":                            true,
	}

	// Report any flags we don't know about
	unknownFlags := make(map[string]bool)
	for _, match := range matches {
		flag := match[1]
		if !knownFlags[flag] {
			unknownFlags[flag] = true
		}
	}

	if len(unknownFlags) > 0 {
		var flags []string
		for flag := range unknownFlags {
			flags = append(flags, "--"+flag)
		}
		t.Logf("INFO: Found %d flags in CLI help that are not in claudecontract:", len(unknownFlags))
		for _, flag := range flags {
			t.Logf("  - %s", flag)
		}
		t.Log("Consider adding constants for these flags if they are needed.")
	}
}

// findClaudeCLI locates the claude CLI binary.
func findClaudeCLI() (string, error) {
	home, _ := os.UserHomeDir()

	// Try common locations
	locations := []string{
		"claude",                             // In PATH
		home + "/.local/bin/claude",          // User install
		home + "/.claude/local/claude",       // Claude Code internal
		"/usr/local/bin/claude",              // System install
	}

	for _, loc := range locations {
		path, err := exec.LookPath(loc)
		if err == nil {
			return path, nil
		}
	}

	return "", exec.ErrNotFound
}

// claudeCmd creates an exec.Command for the claude CLI that is safe to run
// inside a Claude Code session by unsetting the CLAUDECODE env var.
func claudeCmd(claudePath string, args ...string) *exec.Cmd {
	cmd := exec.Command(claudePath, args...)
	// Build env without CLAUDECODE so the CLI doesn't refuse to start
	// when tests are run inside a Claude Code session.
	env := os.Environ()
	filtered := env[:0]
	for _, e := range env {
		if !strings.HasPrefix(e, "CLAUDECODE=") {
			filtered = append(filtered, e)
		}
	}
	cmd.Env = filtered
	return cmd
}
