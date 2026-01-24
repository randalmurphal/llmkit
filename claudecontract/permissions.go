package claudecontract

// PermissionMode represents a permission mode for tool execution.
// From official SDK documentation.
type PermissionMode string

const (
	// PermissionDefault is standard permission behavior (prompts for each action).
	PermissionDefault PermissionMode = "default"

	// PermissionAcceptEdits auto-accepts file edits without prompting.
	PermissionAcceptEdits PermissionMode = "acceptEdits"

	// PermissionBypassPermissions bypasses all permission checks.
	// Requires --dangerously-skip-permissions or --allow-dangerously-skip-permissions.
	PermissionBypassPermissions PermissionMode = "bypassPermissions"

	// PermissionPlan is planning mode - no execution, only planning.
	PermissionPlan PermissionMode = "plan"
)

// Note: "delegate" and "dontAsk" appear in CLI --help but are NOT in official SDK docs.
// They may be internal or deprecated modes.

// ValidPermissionModes returns all valid permission modes.
func ValidPermissionModes() []PermissionMode {
	return []PermissionMode{
		PermissionDefault,
		PermissionAcceptEdits,
		PermissionBypassPermissions,
		PermissionPlan,
	}
}

// IsValid returns true if the permission mode is valid.
func (m PermissionMode) IsValid() bool {
	switch m {
	case PermissionDefault, PermissionAcceptEdits, PermissionBypassPermissions, PermissionPlan:
		return true
	default:
		return false
	}
}

// String returns the string value of the permission mode.
func (m PermissionMode) String() string {
	return string(m)
}

// PermissionBehavior represents the behavior for a permission rule.
type PermissionBehavior string

const (
	// PermissionBehaviorAllow allows the action.
	PermissionBehaviorAllow PermissionBehavior = "allow"

	// PermissionBehaviorDeny denies the action.
	PermissionBehaviorDeny PermissionBehavior = "deny"

	// PermissionBehaviorAsk prompts the user.
	PermissionBehaviorAsk PermissionBehavior = "ask"
)

// PermissionDestination represents where permission updates are saved.
type PermissionDestination string

const (
	// PermissionDestinationUserSettings saves to global user settings.
	PermissionDestinationUserSettings PermissionDestination = "userSettings"

	// PermissionDestinationProjectSettings saves to project settings.
	PermissionDestinationProjectSettings PermissionDestination = "projectSettings"

	// PermissionDestinationLocalSettings saves to local (gitignored) settings.
	PermissionDestinationLocalSettings PermissionDestination = "localSettings"

	// PermissionDestinationSession applies only to current session.
	PermissionDestinationSession PermissionDestination = "session"
)
