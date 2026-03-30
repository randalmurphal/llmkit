package worktree

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Worktree struct {
	path   string
	branch string
	repo   string
}

func (w *Worktree) Path() string   { return w.path }
func (w *Worktree) Branch() string { return w.branch }

func Create(opts CreateOptions) (*Worktree, error) {
	if err := opts.withDefaults(); err != nil {
		return nil, err
	}
	repoDir, err := repoRoot(opts.RepoDir)
	if err != nil {
		return nil, err
	}
	if opts.PruneStale != nil && *opts.PruneStale {
		if err := Prune(repoDir); err != nil {
			return nil, err
		}
	}
	if err := os.MkdirAll(opts.Dir, 0o755); err != nil {
		return nil, err
	}

	worktreePath := filepath.Join(opts.Dir, sanitizeName(opts.Branch))
	if _, err := runGit(repoDir, "show-ref", "--verify", "--quiet", "refs/heads/"+opts.Branch); err == nil {
		if _, err := runGit(repoDir, "worktree", "add", worktreePath, opts.Branch); err != nil {
			return nil, err
		}
	} else {
		if _, err := runGit(repoDir, "worktree", "add", "-b", opts.Branch, worktreePath, opts.BaseBranch); err != nil {
			return nil, err
		}
	}
	if resolved, err := filepath.EvalSymlinks(worktreePath); err == nil {
		worktreePath = resolved
	}

	if opts.InstallHooks != nil && *opts.InstallHooks {
		if err := installHooks(worktreePath, opts.ProtectedBranches); err != nil {
			return nil, err
		}
	}

	return &Worktree{
		path:   worktreePath,
		branch: opts.Branch,
		repo:   repoDir,
	}, nil
}

func (w *Worktree) Remove() error {
	if w == nil {
		return nil
	}
	if _, err := runGit(w.repo, "worktree", "remove", w.path); err == nil {
		return nil
	}
	_, err := runGit(w.repo, "worktree", "remove", "--force", w.path)
	return err
}

func (w *Worktree) HasUncommittedChanges() (bool, error) {
	out, err := runGit(w.path, "status", "--porcelain")
	if err != nil {
		return false, err
	}
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || line == "?? .githooks/" {
			continue
		}
		return true, nil
	}
	return false, nil
}

func List(repoDir string) ([]*Worktree, error) {
	repoDir, err := repoRoot(repoDir)
	if err != nil {
		return nil, err
	}
	out, err := runGit(repoDir, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}

	var trees []*Worktree
	var current *Worktree
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if line == "" {
			if current != nil {
				current.repo = repoDir
				trees = append(trees, current)
				current = nil
			}
			continue
		}
		if strings.HasPrefix(line, "worktree ") {
			if current != nil {
				current.repo = repoDir
				trees = append(trees, current)
			}
			current = &Worktree{path: strings.TrimPrefix(line, "worktree ")}
			continue
		}
		if strings.HasPrefix(line, "branch refs/heads/") && current != nil {
			current.branch = strings.TrimPrefix(line, "branch refs/heads/")
		}
	}
	if current != nil {
		current.repo = repoDir
		trees = append(trees, current)
	}
	return trees, nil
}

func Prune(repoDir string) error {
	repoDir, err := repoRoot(repoDir)
	if err != nil {
		return err
	}
	_, err = runGit(repoDir, "worktree", "prune")
	return err
}

func repoRoot(path string) (string, error) {
	out, err := runGit(path, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", err
	}
	root := strings.TrimSpace(out)
	if resolved, err := filepath.EvalSymlinks(root); err == nil {
		root = resolved
	}
	return root, nil
}

func runGit(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), msg)
	}
	return stdout.String(), nil
}
