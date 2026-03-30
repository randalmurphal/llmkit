package worktree

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateAndRemoveWorktree(t *testing.T) {
	repo := initRepo(t)

	tree, err := Create(CreateOptions{
		RepoDir:      repo,
		TaskID:       "demo",
		BaseBranch:   "main",
		InstallHooks: Bool(true),
		PruneStale:   Bool(true),
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if _, err := os.Stat(tree.Path()); err != nil {
		t.Fatalf("expected worktree path to exist: %v", err)
	}
	if !strings.HasPrefix(tree.Branch(), "llmkit/") {
		t.Fatalf("unexpected branch %q", tree.Branch())
	}
	if _, err := os.Stat(filepath.Join(tree.Path(), ".githooks", "pre-push")); err != nil {
		t.Fatalf("expected pre-push hook: %v", err)
	}
	changed, err := tree.HasUncommittedChanges()
	if err != nil {
		t.Fatalf("HasUncommittedChanges: %v", err)
	}
	if changed {
		t.Fatal("expected clean worktree")
	}

	if err := os.WriteFile(filepath.Join(tree.Path(), "new.txt"), []byte("demo\n"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	changed, err = tree.HasUncommittedChanges()
	if err != nil {
		t.Fatalf("HasUncommittedChanges after edit: %v", err)
	}
	if !changed {
		t.Fatal("expected dirty worktree")
	}

	if err := tree.Remove(); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if _, err := os.Stat(tree.Path()); !os.IsNotExist(err) {
		t.Fatalf("expected worktree path removed, got %v", err)
	}
}

func TestListWorktrees(t *testing.T) {
	repo := initRepo(t)

	tree, err := Create(CreateOptions{
		RepoDir:      repo,
		TaskID:       "list",
		InstallHooks: Bool(true),
		PruneStale:   Bool(true),
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer tree.Remove()

	trees, err := List(repo)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	found := false
	for _, item := range trees {
		if item.Path() == tree.Path() {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected worktree %s in list", tree.Path())
	}
}

func TestCreateCanDisableHooks(t *testing.T) {
	repo := initRepo(t)

	tree, err := Create(CreateOptions{
		RepoDir:      repo,
		TaskID:       "no-hooks",
		InstallHooks: Bool(false),
		PruneStale:   Bool(true),
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer tree.Remove()

	if _, err := os.Stat(filepath.Join(tree.Path(), ".githooks")); !os.IsNotExist(err) {
		t.Fatalf("expected no hooks directory, got %v", err)
	}
}

func initRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	if _, err := runGit(repo, "init", "-b", "main"); err != nil {
		t.Fatalf("git init: %v", err)
	}
	if _, err := runGit(repo, "config", "user.name", "llmkit"); err != nil {
		t.Fatalf("git config name: %v", err)
	}
	if _, err := runGit(repo, "config", "user.email", "llmkit@example.com"); err != nil {
		t.Fatalf("git config email: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("demo\n"), 0o644); err != nil {
		t.Fatalf("write README: %v", err)
	}
	if _, err := runGit(repo, "add", "."); err != nil {
		t.Fatalf("git add: %v", err)
	}
	if _, err := runGit(repo, "commit", "-m", "initial"); err != nil {
		t.Fatalf("git commit: %v", err)
	}
	return repo
}
