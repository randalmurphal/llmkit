package worktree

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func installHooks(path string, protectedBranches []string) error {
	hooksDir := filepath.Join(path, ".githooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(hooksDir, "pre-push"), []byte(prePushHook(protectedBranches)), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(hooksDir, "pre-commit"), []byte(preCommitHook()), 0o755); err != nil {
		return err
	}
	_, err := runGit(path, "config", "core.hooksPath", ".githooks")
	return err
}

func prePushHook(protectedBranches []string) string {
	return "#!/bin/sh\n" +
		"branch=$(git rev-parse --abbrev-ref HEAD)\n" +
		strings.Join(compareBranches(protectedBranches), "") +
		"exit 0\n"
}

func compareBranches(branches []string) []string {
	out := make([]string, 0, len(branches))
	for _, branch := range branches {
		out = append(out, fmt.Sprintf("if [ \"$branch\" = %q ]; then\n  echo \"push blocked from protected branch %s\" >&2\n  exit 1\nfi\n", branch, branch))
	}
	return out
}

func preCommitHook() string {
	return "#!/bin/sh\n" +
		"if [ -n \"$(git status --porcelain)\" ]; then\n" +
		"  echo \"worktree has uncommitted changes\" >&2\n" +
		"fi\n" +
		"exit 0\n"
}
