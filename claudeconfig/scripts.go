package claudeconfig

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Script errors
var (
	ErrScriptNameRequired        = errors.New("script name is required")
	ErrScriptPathRequired        = errors.New("script path is required")
	ErrScriptDescriptionRequired = errors.New("script description is required")
	ErrScriptNotFound            = errors.New("script not found")
	ErrScriptAlreadyExists       = errors.New("script already exists")
)

// ProjectScript defines a script available to Claude Code agents.
// Scripts are registered so agents know what tools are available in the project.
type ProjectScript struct {
	Name        string `json:"name"`
	Path        string `json:"path"`                  // Relative path from project root
	Description string `json:"description"`           // What the script does
	Language    string `json:"language,omitempty"`    // python, bash, go, etc.
	Executable  bool   `json:"executable,omitempty"`  // Whether the script is executable
}

// Validate checks that the script has required fields.
func (s *ProjectScript) Validate() error {
	if s.Name == "" {
		return ErrScriptNameRequired
	}
	if s.Path == "" {
		return ErrScriptPathRequired
	}
	if s.Description == "" {
		return ErrScriptDescriptionRequired
	}
	return nil
}

// ScriptService manages project script registrations stored in .claude/settings.json.
type ScriptService struct {
	projectRoot   string
	extensionName string // Key in settings.json extensions (default: "scripts")
}

// ScriptServiceOption configures the ScriptService.
type ScriptServiceOption func(*ScriptService)

// WithScriptExtensionName sets a custom extension name for storing scripts.
func WithScriptExtensionName(name string) ScriptServiceOption {
	return func(s *ScriptService) {
		s.extensionName = name
	}
}

// NewScriptService creates a new script service for the given project root.
func NewScriptService(projectRoot string, opts ...ScriptServiceOption) *ScriptService {
	s := &ScriptService{
		projectRoot:   projectRoot,
		extensionName: "scripts",
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// List returns all registered scripts.
func (s *ScriptService) List() ([]ProjectScript, error) {
	scripts, err := s.loadScripts()
	if err != nil {
		return nil, fmt.Errorf("load scripts: %w", err)
	}

	sort.Slice(scripts, func(i, j int) bool {
		return scripts[i].Name < scripts[j].Name
	})

	return scripts, nil
}

// Get returns a script by name.
func (s *ScriptService) Get(name string) (*ProjectScript, error) {
	scripts, err := s.List()
	if err != nil {
		return nil, err
	}

	for _, script := range scripts {
		if script.Name == name {
			return &script, nil
		}
	}

	return nil, ErrScriptNotFound
}

// Create registers a new script.
func (s *ScriptService) Create(script ProjectScript) error {
	if err := script.Validate(); err != nil {
		return fmt.Errorf("validate script: %w", err)
	}

	scripts, err := s.loadScripts()
	if err != nil {
		return fmt.Errorf("load scripts: %w", err)
	}

	// Check for duplicates
	for _, existing := range scripts {
		if existing.Name == script.Name {
			return ErrScriptAlreadyExists
		}
	}

	// Detect language if not set
	if script.Language == "" {
		script.Language = detectLanguage(script.Path)
	}

	// Check if executable
	fullPath := filepath.Join(s.projectRoot, script.Path)
	if info, err := os.Stat(fullPath); err == nil {
		script.Executable = info.Mode()&0111 != 0
	}

	scripts = append(scripts, script)

	if err := s.saveScripts(scripts); err != nil {
		return fmt.Errorf("save scripts: %w", err)
	}

	return nil
}

// Update modifies an existing script registration.
func (s *ScriptService) Update(name string, script ProjectScript) error {
	if err := script.Validate(); err != nil {
		return fmt.Errorf("validate script: %w", err)
	}

	scripts, err := s.loadScripts()
	if err != nil {
		return fmt.Errorf("load scripts: %w", err)
	}

	found := false
	for i, existing := range scripts {
		if existing.Name == name {
			// Check for rename conflicts
			if script.Name != name {
				for _, other := range scripts {
					if other.Name == script.Name {
						return ErrScriptAlreadyExists
					}
				}
			}
			scripts[i] = script
			found = true
			break
		}
	}

	if !found {
		return ErrScriptNotFound
	}

	if err := s.saveScripts(scripts); err != nil {
		return fmt.Errorf("save scripts: %w", err)
	}

	return nil
}

// Delete removes a script registration by name.
func (s *ScriptService) Delete(name string) error {
	scripts, err := s.loadScripts()
	if err != nil {
		return fmt.Errorf("load scripts: %w", err)
	}

	found := false
	result := make([]ProjectScript, 0, len(scripts))
	for _, script := range scripts {
		if script.Name == name {
			found = true
			continue
		}
		result = append(result, script)
	}

	if !found {
		return ErrScriptNotFound
	}

	if err := s.saveScripts(result); err != nil {
		return fmt.Errorf("save scripts: %w", err)
	}

	return nil
}

// Exists checks if a script with the given name is registered.
func (s *ScriptService) Exists(name string) bool {
	_, err := s.Get(name)
	return err == nil
}

// Discover scans the .claude/scripts/ directory for scripts and returns them.
// It does not automatically register them - call Create for each to register.
func (s *ScriptService) Discover() ([]ProjectScript, error) {
	scriptsDir := filepath.Join(s.projectRoot, ".claude", "scripts")

	if !dirExists(scriptsDir) {
		return nil, nil
	}

	entries, err := os.ReadDir(scriptsDir)
	if err != nil {
		return nil, fmt.Errorf("read scripts directory: %w", err)
	}

	var scripts []ProjectScript
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		path := filepath.Join(".claude", "scripts", name)
		fullPath := filepath.Join(s.projectRoot, path)

		info, err := entry.Info()
		if err != nil {
			continue
		}

		script := ProjectScript{
			Name:        strings.TrimSuffix(name, filepath.Ext(name)),
			Path:        path,
			Description: fmt.Sprintf("Script: %s", name), // Placeholder
			Language:    detectLanguage(name),
			Executable:  info.Mode()&0111 != 0,
		}

		// Try to read description from first line comment
		if desc := extractScriptDescription(fullPath); desc != "" {
			script.Description = desc
		}

		scripts = append(scripts, script)
	}

	sort.Slice(scripts, func(i, j int) bool {
		return scripts[i].Name < scripts[j].Name
	})

	return scripts, nil
}

// loadScripts loads scripts from settings.json extension.
func (s *ScriptService) loadScripts() ([]ProjectScript, error) {
	settings, err := LoadProjectSettings(s.projectRoot)
	if err != nil {
		return nil, err
	}

	var scripts []ProjectScript
	if err := settings.GetExtension(s.extensionName, &scripts); err != nil {
		return nil, fmt.Errorf("get %s extension: %w", s.extensionName, err)
	}

	if scripts == nil {
		scripts = []ProjectScript{}
	}

	return scripts, nil
}

// saveScripts saves scripts to settings.json extension.
func (s *ScriptService) saveScripts(scripts []ProjectScript) error {
	settings, err := LoadProjectSettings(s.projectRoot)
	if err != nil {
		return err
	}

	settings.SetExtension(s.extensionName, scripts)

	if err := SaveProjectSettings(s.projectRoot, settings); err != nil {
		return fmt.Errorf("save project settings: %w", err)
	}

	return nil
}

// detectLanguage guesses the language from the file extension.
func detectLanguage(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".py":
		return "python"
	case ".sh", ".bash":
		return "bash"
	case ".go":
		return "go"
	case ".js":
		return "javascript"
	case ".ts":
		return "typescript"
	case ".rb":
		return "ruby"
	case ".pl":
		return "perl"
	case ".php":
		return "php"
	case ".rs":
		return "rust"
	default:
		return ""
	}
}

// extractScriptDescription tries to read a description from the script's first comment.
func extractScriptDescription(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip shebang
		if strings.HasPrefix(line, "#!") {
			continue
		}

		// Skip empty lines
		if line == "" {
			continue
		}

		// Check for comment (various languages)
		if desc, ok := strings.CutPrefix(line, "#"); ok {
			return strings.TrimSpace(desc)
		}
		if desc, ok := strings.CutPrefix(line, "//"); ok {
			return strings.TrimSpace(desc)
		}
		if strings.HasPrefix(line, "\"\"\"") || strings.HasPrefix(line, "'''") {
			// Python docstring - get next line
			continue
		}

		// Non-comment line reached, stop looking
		break
	}

	return ""
}
