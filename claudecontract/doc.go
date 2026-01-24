// Package claudecontract provides a single source of truth for all Claude CLI
// interface details including flag names, event types, file paths, permission
// modes, tool names, and other volatile strings that may change between CLI versions.
//
// # Purpose
//
// This package centralizes all the "contract" between the llmkit Go code and the
// Claude CLI binary. When the Claude CLI changes its interface (flag names, JSON
// field names, event types, etc.), only this package needs to be updated.
//
// # Package Contents
//
//   - version.go: CLI version detection and compatibility checking
//   - flags.go: CLI flag name constants (--print, --model, etc.)
//   - events.go: Stream event types (system, assistant, user, result)
//   - permissions.go: Permission mode constants
//   - tools.go: Built-in tool names and categories
//   - paths.go: File and directory names (.claude, settings.json, etc.)
//   - formats.go: Output formats, transport types, hook events
//
// # Usage
//
// Both the claude/ and claudeconfig/ packages import from here:
//
//	import "github.com/randalmurphal/llmkit/claudecontract"
//
//	// Use flag constants
//	args := []string{claudecontract.FlagPrint, claudecontract.FlagOutputFormat, claudecontract.FormatStreamJSON}
//
//	// Check event types
//	if eventType == claudecontract.EventTypeAssistant { ... }
//
//	// Check permission modes
//	if mode == claudecontract.PermissionBypassPermissions { ... }
//
// # Version Compatibility
//
// The TestedCLIVersion constant indicates which Claude CLI version this code was
// tested against. Use CheckVersion() to detect the installed CLI version and warn
// if it's newer than the tested version:
//
//	v := claudecontract.CheckVersion("claude")
//	// Logs a warning if CLI version is newer than TestedCLIVersion
//
// # Sources
//
// These constants are derived from:
//   - Claude CLI --help output
//   - Official TypeScript SDK documentation: https://platform.claude.com/docs/en/agent-sdk/typescript
//   - Official Python SDK documentation: https://platform.claude.com/docs/en/agent-sdk/python
//   - Official Hooks documentation: https://code.claude.com/docs/en/hooks
//   - Official CLI Reference: https://code.claude.com/docs/en/cli-reference
package claudecontract
