package codexcontract

import (
	"regexp"
	"strings"
	"testing"
)

func TestVersionMatches(t *testing.T) {
	codexPath, err := findCodexCLI()
	if err != nil {
		t.Skip("codex CLI not found")
	}

	version, err := DetectCLIVersion(codexPath)
	if err != nil {
		t.Fatalf("failed to detect codex version: %v", err)
	}

	if version.Raw != TestedCLIVersion {
		t.Logf("WARNING: Installed codex version %s differs from tested version %s", version.Raw, TestedCLIVersion)
	}
}

func TestRootHelpContainsCriticalFlagsAndCommands(t *testing.T) {
	codexPath, err := findCodexCLI()
	if err != nil {
		t.Skip("codex CLI not found")
	}

	helpText := runCodexOutput(t, codexPath, "--help")

	criticalFlags := []string{
		FlagConfig,
		FlagEnable,
		FlagDisable,
		FlagImage,
		FlagModel,
		FlagOSS,
		FlagLocalProvider,
		FlagProfile,
		FlagSandbox,
		FlagAskForApproval,
		FlagFullAuto,
		FlagDangerouslyBypassApprovalsSandbox,
		FlagCD,
		FlagSearch,
		FlagAddDir,
		"--no-alt-screen",
		"--help",
		"--version",
	}
	for _, flag := range criticalFlags {
		if !strings.Contains(helpText, flag) {
			t.Errorf("flag %q not found in codex --help", flag)
		}
	}

	criticalCommands := []string{"exec", "review", "apply", "resume", "fork", "features"}
	for _, cmd := range criticalCommands {
		if !strings.Contains(helpText, cmd) {
			t.Errorf("command %q not found in codex --help", cmd)
		}
	}
}

func TestExecHelpContainsHeadlessFlags(t *testing.T) {
	codexPath, err := findCodexCLI()
	if err != nil {
		t.Skip("codex CLI not found")
	}

	helpText := runCodexOutput(t, codexPath, "exec", "--help")

	criticalExecFlags := []string{
		FlagConfig,
		FlagEnable,
		FlagDisable,
		FlagImage,
		FlagModel,
		FlagOSS,
		FlagLocalProvider,
		FlagSandbox,
		FlagProfile,
		FlagFullAuto,
		FlagDangerouslyBypassApprovalsSandbox,
		FlagCD,
		FlagSkipGitRepoCheck,
		FlagAddDir,
		FlagOutputSchema,
		FlagColor,
		FlagJSON,
		FlagOutputLastMessage,
		"--help",
		"--version",
	}
	for _, flag := range criticalExecFlags {
		if !strings.Contains(helpText, flag) {
			t.Errorf("exec flag %q not found in codex exec --help", flag)
		}
	}

	if !strings.Contains(helpText, "resume") {
		t.Error("expected exec subcommand 'resume' not found in codex exec --help")
	}
}

func TestDetectNewFlags(t *testing.T) {
	codexPath, err := findCodexCLI()
	if err != nil {
		t.Skip("codex CLI not found")
	}

	rootHelp := runCodexOutput(t, codexPath, "--help")
	execHelp := runCodexOutput(t, codexPath, "exec", "--help")
	combined := rootHelp + "\n" + execHelp

	flagPattern := regexp.MustCompile(`--([a-zA-Z][a-zA-Z0-9-]*)`)
	matches := flagPattern.FindAllStringSubmatch(combined, -1)

	knownFlags := map[string]bool{
		"json":             true,
		"model":            true,
		"sandbox":          true,
		"ask-for-approval": true,
		"full-auto":        true,
		"yolo":             true,
		"dangerously-bypass-approvals-and-sandbox": true,
		"cd":                  true,
		"add-dir":             true,
		"image":               true,
		"search":              true,
		"profile":             true,
		"local-provider":      true,
		"skip-git-repo-check": true,
		"output-schema":       true,
		"output-last-message": true,
		"all":                 true,
		"oss":                 true,
		"enable":              true,
		"disable":             true,
		"color":               true,
		"version":             true,
		"help":                true,
		"no-alt-screen":       true,
		"config":              true,
		"last":                true,
		"approve":             true,
		"name":                true,
	}

	unknownFlags := map[string]bool{}
	for _, match := range matches {
		flag := match[1]
		if !knownFlags[flag] {
			unknownFlags[flag] = true
		}
	}

	if len(unknownFlags) > 0 {
		for flag := range unknownFlags {
			t.Logf("INFO: new flag found in codex help: --%s", flag)
		}
	}
}
