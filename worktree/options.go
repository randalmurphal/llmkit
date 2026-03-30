package worktree

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

type CreateOptions struct {
	RepoDir           string
	Branch            string
	BaseBranch        string
	Dir               string
	TaskID            string
	BranchPrefix      string
	ProtectedBranches []string
	InstallHooks      *bool
	PruneStale        *bool
}

func DefaultCreateOptions(repoDir string) CreateOptions {
	return CreateOptions{
		RepoDir:      repoDir,
		InstallHooks: Bool(true),
		PruneStale:   Bool(true),
	}
}

func (o *CreateOptions) withDefaults() error {
	if o.RepoDir == "" {
		return fmt.Errorf("repo dir is required")
	}
	if o.BaseBranch == "" {
		o.BaseBranch = "main"
	}
	if o.BranchPrefix == "" {
		o.BranchPrefix = "llmkit/"
	}
	if o.Dir == "" {
		o.Dir = filepath.Join(filepath.Dir(o.RepoDir), "worktrees")
	}
	if len(o.ProtectedBranches) == 0 {
		o.ProtectedBranches = []string{"main", "master"}
	}
	if o.InstallHooks == nil {
		o.InstallHooks = Bool(true)
	}
	if o.PruneStale == nil {
		o.PruneStale = Bool(true)
	}
	if o.Branch == "" {
		taskID := strings.TrimSpace(o.TaskID)
		if taskID == "" {
			taskID = fmt.Sprintf("session-%d", time.Now().UnixNano())
		}
		o.Branch = o.BranchPrefix + sanitizeName(taskID)
	}
	return nil
}

func sanitizeName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "worktree"
	}
	name = strings.ReplaceAll(name, " ", "-")
	name = strings.ReplaceAll(name, "/", "-")
	return name
}

func Bool(v bool) *bool { return &v }
