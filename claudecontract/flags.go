package claudecontract

// CLI flag names - update here when CLI changes.
// These are the exact flag names as used by the claude CLI binary.
const (
	// Core flags
	FlagPrint        = "--print"         // -p, Run in non-interactive mode
	FlagOutputFormat = "--output-format" // Output format: text, json, stream-json
	FlagInputFormat  = "--input-format"  // Input format for streaming
	FlagVerbose      = "--verbose"       // Enable verbose output in stream-json

	// Model flags
	FlagModel         = "--model"          // Claude model to use
	FlagFallbackModel = "--fallback-model" // Fallback model if primary fails

	// Session flags
	FlagSessionID            = "--session-id"             // UUID for session
	FlagContinue             = "--continue"               // -c, Continue most recent conversation
	FlagResume               = "--resume"                 // -r, Resume specific session by ID
	FlagForkSession          = "--fork-session"           // Fork session instead of reusing
	FlagNoSessionPersistence = "--no-session-persistence" // Don't persist session

	// Agent flags
	FlagAgent  = "--agent"  // Select agent for session
	FlagAgents = "--agents" // Define custom agents (JSON)

	// Tool flags (note: camelCase in CLI, not kebab-case!)
	FlagAllowedTools    = "--allowedTools"    // Tools to allow (repeatable)
	FlagDisallowedTools = "--disallowedTools" // Tools to disallow (repeatable)
	FlagTools           = "--tools"           // Comma-separated tool list

	// Prompt flags
	FlagSystemPrompt           = "--system-prompt"             // Set system prompt
	FlagAppendSystemPrompt     = "--append-system-prompt"      // Append to system prompt
	FlagSystemPromptFile       = "--system-prompt-file"        // Load system prompt from file
	FlagAppendSystemPromptFile = "--append-system-prompt-file" // Load append prompt from file

	// Permission flags
	FlagDangerouslySkipPermissions      = "--dangerously-skip-permissions"       // Skip all permission prompts
	FlagAllowDangerouslySkipPermissions = "--allow-dangerously-skip-permissions" // Enable the option
	FlagPermissionMode                  = "--permission-mode"                    // Permission mode
	FlagPermissionPromptTool            = "--permission-prompt-tool"             // MCP tool for permission prompts

	// Settings flags
	FlagSettings       = "--settings"        // Load settings file/JSON
	FlagSettingSources = "--setting-sources" // Comma-separated: user, project, local
	FlagPluginDir      = "--plugin-dir"      // Plugin directories (repeatable)

	// MCP flags
	FlagMCPConfig       = "--mcp-config"        // MCP configuration (file path or JSON)
	FlagStrictMCPConfig = "--strict-mcp-config" // Enforce strict MCP validation
	FlagMCPDebug        = "--mcp-debug"         // [DEPRECATED] Enable MCP debug mode

	// Directory flags
	FlagAddDir = "--add-dir" // Additional directories Claude can access (repeatable)

	// Budget and limits flags
	FlagMaxBudgetUSD = "--max-budget-usd" // Maximum budget in USD
	FlagMaxTurns     = "--max-turns"      // Maximum conversation turns

	// Schema flags
	FlagJSONSchema = "--json-schema" // JSON Schema for structured output

	// Streaming flags
	FlagIncludePartialMessages = "--include-partial-messages" // Include partial message events
	FlagReplayUserMessages     = "--replay-user-messages"     // Re-emit user messages

	// Debug flags
	FlagDebug = "--debug" // Debug mode with optional filter
	FlagBetas = "--betas" // Beta headers (repeatable)

	// Skill flags
	FlagDisableSlashCommands = "--disable-slash-commands" // Disable all skills

	// Session lifecycle flags
	FlagInit        = "--init"        // Run setup hooks
	FlagInitOnly    = "--init-only"   // Run setup hooks and exit
	FlagMaintenance = "--maintenance" // Run maintenance hooks

	// Web session flags
	FlagRemote   = "--remote"   // Create web session
	FlagTeleport = "--teleport" // Resume web session locally

	// IDE integration flags
	FlagChrome   = "--chrome"    // Enable Chrome integration
	FlagNoChrome = "--no-chrome" // Disable Chrome integration
	FlagIDE      = "--ide"       // IDE integration mode

	// File flags
	FlagFile = "--file" // File resources (repeatable)

	// Version flag
	FlagVersion = "--version" // -v, Show version
	FlagHelp    = "--help"    // -h, Show help
)
