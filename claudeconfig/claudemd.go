package claudeconfig

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ClaudeMD represents a CLAUDE.md file.
type ClaudeMD struct {
	Path     string `json:"path"`      // Full file path
	Content  string `json:"content"`   // Markdown content
	IsGlobal bool   `json:"is_global"` // true if ~/.claude/CLAUDE.md
	Source   string `json:"source"`    // "global", "user", "project", "local"
}

// ClaudeMDHierarchy represents the CLAUDE.md inheritance chain.
// Files are applied in order: global -> user -> project -> local
type ClaudeMDHierarchy struct {
	Global  *ClaudeMD   `json:"global,omitempty"`  // ~/.claude/CLAUDE.md
	User    *ClaudeMD   `json:"user,omitempty"`    // ~/CLAUDE.md
	Project *ClaudeMD   `json:"project,omitempty"` // {project}/CLAUDE.md
	Local   []*ClaudeMD `json:"local,omitempty"`   // Per-directory overrides
}

// LoadClaudeMD reads a CLAUDE.md file from the given path.
func LoadClaudeMD(path string) (*ClaudeMD, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read CLAUDE.md: %w", err)
	}

	return &ClaudeMD{
		Path:    path,
		Content: string(data),
	}, nil
}

// LoadClaudeMDHierarchy loads the full CLAUDE.md hierarchy for a project.
func LoadClaudeMDHierarchy(projectRoot string) (*ClaudeMDHierarchy, error) {
	hierarchy := &ClaudeMDHierarchy{}

	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("get home dir: %w", err)
	}

	// Load global (~/.claude/CLAUDE.md)
	globalPath := filepath.Join(home, ".claude", "CLAUDE.md")
	if global, err := LoadClaudeMD(globalPath); err != nil {
		return nil, fmt.Errorf("load global CLAUDE.md: %w", err)
	} else if global != nil {
		global.IsGlobal = true
		global.Source = "global"
		hierarchy.Global = global
	}

	// Load user (~/CLAUDE.md)
	userPath := filepath.Join(home, "CLAUDE.md")
	if user, err := LoadClaudeMD(userPath); err != nil {
		return nil, fmt.Errorf("load user CLAUDE.md: %w", err)
	} else if user != nil {
		user.Source = "user"
		hierarchy.User = user
	}

	// Load project ({project}/CLAUDE.md)
	projectPath := filepath.Join(projectRoot, "CLAUDE.md")
	if project, err := LoadClaudeMD(projectPath); err != nil {
		return nil, fmt.Errorf("load project CLAUDE.md: %w", err)
	} else if project != nil {
		project.Source = "project"
		hierarchy.Project = project
	}

	return hierarchy, nil
}

// LoadProjectClaudeMD loads only the project's CLAUDE.md file.
func LoadProjectClaudeMD(projectRoot string) (*ClaudeMD, error) {
	path := filepath.Join(projectRoot, "CLAUDE.md")
	claudemd, err := LoadClaudeMD(path)
	if err != nil {
		return nil, err
	}
	if claudemd != nil {
		claudemd.Source = "project"
	}
	return claudemd, nil
}

// SaveProjectClaudeMD saves the project's CLAUDE.md file.
func SaveProjectClaudeMD(projectRoot string, content string) error {
	path := filepath.Join(projectRoot, "CLAUDE.md")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("write CLAUDE.md: %w", err)
	}
	return nil
}

// CombinedContent returns the combined content of all CLAUDE.md files in the hierarchy.
// Files are separated by a header comment indicating the source.
func (h *ClaudeMDHierarchy) CombinedContent() string {
	var parts []string

	if h.Global != nil && h.Global.Content != "" {
		parts = append(parts, fmt.Sprintf("<!-- Global: %s -->\n%s", h.Global.Path, h.Global.Content))
	}

	if h.User != nil && h.User.Content != "" {
		parts = append(parts, fmt.Sprintf("<!-- User: %s -->\n%s", h.User.Path, h.User.Content))
	}

	if h.Project != nil && h.Project.Content != "" {
		parts = append(parts, fmt.Sprintf("<!-- Project: %s -->\n%s", h.Project.Path, h.Project.Content))
	}

	for _, local := range h.Local {
		if local != nil && local.Content != "" {
			parts = append(parts, fmt.Sprintf("<!-- Local: %s -->\n%s", local.Path, local.Content))
		}
	}

	return strings.Join(parts, "\n\n---\n\n")
}

// HasProject returns true if a project CLAUDE.md exists.
func (h *ClaudeMDHierarchy) HasProject() bool {
	return h != nil && h.Project != nil && h.Project.Content != ""
}

// HasGlobal returns true if a global CLAUDE.md exists.
func (h *ClaudeMDHierarchy) HasGlobal() bool {
	return h != nil && h.Global != nil && h.Global.Content != ""
}

// Count returns the number of CLAUDE.md files in the hierarchy.
func (h *ClaudeMDHierarchy) Count() int {
	count := 0
	if h.Global != nil {
		count++
	}
	if h.User != nil {
		count++
	}
	if h.Project != nil {
		count++
	}
	count += len(h.Local)
	return count
}

// ClaudeMDPath returns the expected path for a project's CLAUDE.md file.
func ClaudeMDPath(projectRoot string) string {
	return filepath.Join(projectRoot, "CLAUDE.md")
}

// GlobalClaudeMDPath returns the path to the global CLAUDE.md file.
func GlobalClaudeMDPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home dir: %w", err)
	}
	return filepath.Join(home, ".claude", "CLAUDE.md"), nil
}
