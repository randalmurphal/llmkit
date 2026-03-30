package codexconfig

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverSkillsSearchesRepoAndUserScopes(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	projectRoot := t.TempDir()
	cwd := filepath.Join(projectRoot, "service", "api")
	if err := os.MkdirAll(cwd, 0o755); err != nil {
		t.Fatalf("MkdirAll cwd: %v", err)
	}

	projectSkillDir := filepath.Join(projectRoot, ".agents", "skills", "repo-skill")
	userSkillDir := filepath.Join(home, ".agents", "skills", "user-skill")
	if err := os.MkdirAll(projectSkillDir, 0o755); err != nil {
		t.Fatalf("MkdirAll project skill: %v", err)
	}
	if err := os.MkdirAll(userSkillDir, 0o755); err != nil {
		t.Fatalf("MkdirAll user skill: %v", err)
	}

	if err := os.WriteFile(filepath.Join(projectSkillDir, "SKILL.md"), []byte("---\nname: repo-skill\ndescription: repo\n---\n\nrepo"), 0o644); err != nil {
		t.Fatalf("write project skill: %v", err)
	}
	if err := os.WriteFile(filepath.Join(userSkillDir, "SKILL.md"), []byte("---\nname: user-skill\ndescription: user\n---\n\nuser"), 0o644); err != nil {
		t.Fatalf("write user skill: %v", err)
	}

	skills, err := DiscoverSkills(projectRoot, cwd)
	if err != nil {
		t.Fatalf("DiscoverSkills: %v", err)
	}
	if len(skills) != 2 {
		t.Fatalf("len(skills) = %d, want 2", len(skills))
	}
}
