package claudecontract

// CLI flag names - update here when CLI changes.
// These are the exact flag names as used by the claude CLI binary.
//
// Source: https://code.claude.com/docs/en/cli-reference
//
// Test coverage legend:
//   [TESTED] - Has behavioral test in claude/behavioral_test.go
//   [PARSING] - Tested via golden file parsing
//   [MANUAL] - Requires manual testing (browser, web session, etc.)
//   [UNTESTED] - No automated test coverage
const (
	// Core flags
	FlagPrint        = "--print"         // -p, Run in non-interactive mode [TESTED]
	FlagOutputFormat = "--output-format" // Output format: text, json, stream-json [TESTED]
	FlagInputFormat  = "--input-format"  // Input format for streaming [UNTESTED]
	FlagVerbose      = "--verbose"       // Enable verbose output [TESTED]

	// Model flags
	FlagModel         = "--model"          // Claude model to use [TESTED]
	FlagFallbackModel = "--fallback-model" // Fallback model if primary fails [TESTED]

	// Session flags
	FlagSessionID            = "--session-id"             // UUID for session [TESTED]
	FlagContinue             = "--continue"               // -c, Continue most recent conversation [TESTED]
	FlagResume               = "--resume"                 // -r, Resume specific session by ID [TESTED]
	FlagForkSession          = "--fork-session"           // Fork session instead of reusing [TESTED]
	FlagNoSessionPersistence = "--no-session-persistence" // Don't persist session [TESTED]

	// Agent flags
	FlagAgent  = "--agent"  // Select agent for session [TESTED via AgentListInInit]
	FlagAgents = "--agents" // Define custom agents (JSON) [TESTED]

	// Tool flags (note: CLI accepts both camelCase and kebab-case!)
	FlagAllowedTools    = "--allowedTools"    // Tools to allow (repeatable) [TESTED]
	FlagDisallowedTools = "--disallowedTools" // Tools to disallow (repeatable) [TESTED]
	FlagTools           = "--tools"           // Restrict available tools [TESTED]

	// Prompt flags
	FlagSystemPrompt           = "--system-prompt"             // Set system prompt [TESTED]
	FlagAppendSystemPrompt     = "--append-system-prompt"      // Append to system prompt [TESTED]
	FlagSystemPromptFile       = "--system-prompt-file"        // Load system prompt from file [TESTED]
	FlagAppendSystemPromptFile = "--append-system-prompt-file" // Load append prompt from file [TESTED]

	// Permission flags
	FlagDangerouslySkipPermissions      = "--dangerously-skip-permissions"       // Skip all permission prompts [TESTED]
	FlagAllowDangerouslySkipPermissions = "--allow-dangerously-skip-permissions" // Enable the option [UNTESTED]
	FlagPermissionMode                  = "--permission-mode"                    // Permission mode [TESTED]
	FlagPermissionPromptTool            = "--permission-prompt-tool"             // MCP tool for permission prompts [UNTESTED]

	// Settings flags
	FlagSettings       = "--settings"        // Load settings file/JSON [UNTESTED]
	FlagSettingSources = "--setting-sources" // Comma-separated: user, project, local [UNTESTED]
	FlagPluginDir      = "--plugin-dir"      // Plugin directories (repeatable) [UNTESTED]

	// MCP flags
	FlagMCPConfig       = "--mcp-config"        // MCP configuration (file path or JSON) [TESTED via MCPServers]
	FlagStrictMCPConfig = "--strict-mcp-config" // Enforce strict MCP validation [UNTESTED]
	FlagMCPDebug        = "--mcp-debug"         // [DEPRECATED] Enable MCP debug mode [UNTESTED]

	// Directory flags
	FlagAddDir = "--add-dir" // Additional directories Claude can access (repeatable) [TESTED]

	// Budget and limits flags
	FlagMaxBudgetUSD = "--max-budget-usd" // Maximum budget in USD [TESTED]
	FlagMaxTurns     = "--max-turns"      // Maximum conversation turns [TESTED]

	// Schema flags
	FlagJSONSchema = "--json-schema" // JSON Schema for structured output [TESTED]

	// Streaming flags
	FlagIncludePartialMessages = "--include-partial-messages" // Include partial message events [UNTESTED]
	FlagReplayUserMessages     = "--replay-user-messages"     // Re-emit user messages [UNTESTED]

	// Debug flags
	FlagDebug     = "--debug"      // Debug mode with optional filter [UNTESTED]
	FlagDebugFile = "--debug-file" // Write debug output to file [UNTESTED]
	FlagBetas     = "--betas"      // Beta headers (repeatable) [UNTESTED]

	// Reasoning flags
	FlagEffort = "--effort" // Reasoning effort level [UNTESTED]

	// PR integration flags
	FlagFromPR = "--from-pr" // Load PR context [UNTESTED]

	// Skill flags
	FlagDisableSlashCommands = "--disable-slash-commands" // Disable all skills [UNTESTED]

	// Session lifecycle flags
	FlagInit        = "--init"        // Run setup hooks [UNTESTED]
	FlagInitOnly    = "--init-only"   // Run setup hooks and exit [UNTESTED]
	FlagMaintenance = "--maintenance" // Run maintenance hooks [UNTESTED]

	// Web session flags
	FlagRemote   = "--remote"   // Create web session [MANUAL]
	FlagTeleport = "--teleport" // Resume web session locally [MANUAL]

	// IDE integration flags
	FlagChrome   = "--chrome"    // Enable Chrome integration [MANUAL]
	FlagNoChrome = "--no-chrome" // Disable Chrome integration [MANUAL]
	FlagIDE      = "--ide"       // IDE integration mode [MANUAL]

	// File flags
	FlagFile = "--file" // File resources (repeatable) [UNTESTED]

	// Version flag
	FlagVersion = "--version" // -v, Show version [UNTESTED]
	FlagHelp    = "--help"    // -h, Show help [UNTESTED]
)

// TestedFlags returns a list of flags that have behavioral test coverage.
func TestedFlags() []string {
	return []string{
		FlagPrint,
		FlagOutputFormat,
		FlagVerbose,
		FlagModel,
		FlagFallbackModel,
		FlagSessionID,
		FlagContinue,
		FlagResume,
		FlagForkSession,
		FlagNoSessionPersistence,
		FlagAgent,
		FlagAgents,
		FlagAllowedTools,
		FlagDisallowedTools,
		FlagTools,
		FlagSystemPrompt,
		FlagAppendSystemPrompt,
		FlagSystemPromptFile,
		FlagAppendSystemPromptFile,
		FlagDangerouslySkipPermissions,
		FlagPermissionMode,
		FlagMCPConfig,
		FlagAddDir,
		FlagMaxBudgetUSD,
		FlagMaxTurns,
		FlagJSONSchema,
	}
}

// UntestedFlags returns a list of flags without behavioral test coverage.
func UntestedFlags() []string {
	return []string{
		FlagInputFormat,
		FlagAllowDangerouslySkipPermissions,
		FlagPermissionPromptTool,
		FlagSettings,
		FlagSettingSources,
		FlagPluginDir,
		FlagStrictMCPConfig,
		FlagMCPDebug,
		FlagIncludePartialMessages,
		FlagReplayUserMessages,
		FlagDebug,
		FlagDebugFile,
		FlagBetas,
		FlagEffort,
		FlagFromPR,
		FlagDisableSlashCommands,
		FlagInit,
		FlagInitOnly,
		FlagMaintenance,
		FlagFile,
		FlagVersion,
		FlagHelp,
	}
}

// ManualTestFlags returns flags that require manual testing.
func ManualTestFlags() []string {
	return []string{
		FlagRemote,
		FlagTeleport,
		FlagChrome,
		FlagNoChrome,
		FlagIDE,
	}
}
