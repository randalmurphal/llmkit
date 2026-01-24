// Package claudeconfig provides utilities for parsing Claude Code's native configuration formats.
// This includes SKILL.md files, settings.json, and CLAUDE.md files.
package claudeconfig

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/randalmurphal/llmkit/claudecontract"
	"gopkg.in/yaml.v3"
)

// StringOrArray is a custom type that can unmarshal from either a string
// (comma-separated) or a YAML array.
type StringOrArray []string

// UnmarshalYAML implements yaml.Unmarshaler to handle both string and array formats.
func (s *StringOrArray) UnmarshalYAML(value *yaml.Node) error {
	// Try as array first
	var arr []string
	if err := value.Decode(&arr); err == nil {
		*s = arr
		return nil
	}

	// Try as comma-separated string
	var str string
	if err := value.Decode(&str); err == nil {
		parts := strings.Split(str, ",")
		result := make([]string, 0, len(parts))
		for _, p := range parts {
			if trimmed := strings.TrimSpace(p); trimmed != "" {
				result = append(result, trimmed)
			}
		}
		*s = result
		return nil
	}

	return errors.New("allowed-tools must be a string or array")
}

// Skill represents a Claude Code skill parsed from a SKILL.md file.
// The file format uses YAML frontmatter followed by markdown content.
type Skill struct {
	// Frontmatter fields (from YAML between --- delimiters)
	Name         string        `yaml:"name" json:"name"`
	Description  string        `yaml:"description" json:"description"`
	AllowedTools StringOrArray `yaml:"allowed-tools,omitempty" json:"allowed_tools,omitempty"`
	Version      string        `yaml:"version,omitempty" json:"version,omitempty"`

	// Content is the markdown body after the frontmatter
	Content string `yaml:"-" json:"content"`

	// Path is the directory containing the SKILL.md file
	Path string `yaml:"-" json:"path"`

	// Resource flags indicate presence of subdirectories
	HasReferences bool `yaml:"-" json:"has_references"`
	HasScripts    bool `yaml:"-" json:"has_scripts"`
	HasAssets     bool `yaml:"-" json:"has_assets"`
}

// SkillInfo provides summary information for listing skills.
type SkillInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Path        string `json:"path"`
}

// Info returns summary information for this skill.
func (s *Skill) Info() SkillInfo {
	return SkillInfo{
		Name:        s.Name,
		Description: s.Description,
		Path:        s.Path,
	}
}

// Validate checks that the skill has required fields.
func (s *Skill) Validate() error {
	if s.Name == "" {
		return errors.New("skill name is required")
	}
	if s.Description == "" {
		return errors.New("skill description is required")
	}
	return nil
}

// ParseSkillMD reads and parses a SKILL.md file from the given path.
// The path can be either the SKILL.md file itself or the directory containing it.
func ParseSkillMD(path string) (*Skill, error) {
	// Determine the actual file path
	filePath := path
	dirPath := path

	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("stat path: %w", err)
	}

	if info.IsDir() {
		filePath = filepath.Join(path, claudecontract.FileSkillMD)
	} else {
		dirPath = filepath.Dir(path)
	}

	// Read the file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", claudecontract.FileSkillMD, err)
	}

	// Parse the content
	skill, err := parseSkillContent(data)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", claudecontract.FileSkillMD, err)
	}

	skill.Path = dirPath

	// Check for resource subdirectories
	skill.HasReferences = dirExists(filepath.Join(dirPath, claudecontract.DirReferences))
	skill.HasScripts = dirExists(filepath.Join(dirPath, claudecontract.DirScripts))
	skill.HasAssets = dirExists(filepath.Join(dirPath, claudecontract.DirAssets))

	return skill, nil
}

// parseSkillContent parses the SKILL.md content with YAML frontmatter.
func parseSkillContent(data []byte) (*Skill, error) {
	// Check for frontmatter delimiter
	if !bytes.HasPrefix(data, []byte("---")) {
		return nil, errors.New("SKILL.md must start with YAML frontmatter (---)")
	}

	// Find the end of frontmatter
	scanner := bufio.NewScanner(bytes.NewReader(data))
	var frontmatterLines []string
	var contentLines []string
	inFrontmatter := false
	foundEnd := false

	lineNum := 0
	for scanner.Scan() {
		line := scanner.Text()
		lineNum++

		if lineNum == 1 && line == "---" {
			inFrontmatter = true
			continue
		}

		if inFrontmatter && line == "---" {
			inFrontmatter = false
			foundEnd = true
			continue
		}

		if inFrontmatter {
			frontmatterLines = append(frontmatterLines, line)
		} else if foundEnd {
			contentLines = append(contentLines, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan content: %w", err)
	}

	if !foundEnd {
		return nil, errors.New("SKILL.md frontmatter not closed (missing ---)")
	}

	// Parse YAML frontmatter
	skill := &Skill{}
	frontmatter := strings.Join(frontmatterLines, "\n")
	if err := yaml.Unmarshal([]byte(frontmatter), skill); err != nil {
		return nil, fmt.Errorf("parse frontmatter: %w", err)
	}

	// Set content (trim leading/trailing blank lines)
	skill.Content = strings.TrimSpace(strings.Join(contentLines, "\n"))

	return skill, nil
}

// WriteSkillMD writes a skill to a SKILL.md file in the given directory.
// It creates the directory if it doesn't exist.
func WriteSkillMD(skill *Skill, dir string) error {
	if err := skill.Validate(); err != nil {
		return fmt.Errorf("validate skill: %w", err)
	}

	// Create directory if needed
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	// Build frontmatter
	frontmatter := struct {
		Name         string   `yaml:"name"`
		Description  string   `yaml:"description"`
		AllowedTools []string `yaml:"allowed-tools,omitempty"`
		Version      string   `yaml:"version,omitempty"`
	}{
		Name:         skill.Name,
		Description:  skill.Description,
		AllowedTools: skill.AllowedTools,
		Version:      skill.Version,
	}

	fmBytes, err := yaml.Marshal(frontmatter)
	if err != nil {
		return fmt.Errorf("marshal frontmatter: %w", err)
	}

	// Build full content
	var buf bytes.Buffer
	buf.WriteString("---\n")
	buf.Write(fmBytes)
	buf.WriteString("---\n\n")
	buf.WriteString(skill.Content)
	if !strings.HasSuffix(skill.Content, "\n") {
		buf.WriteString("\n")
	}

	// Write file
	filePath := filepath.Join(dir, claudecontract.FileSkillMD)
	if err := os.WriteFile(filePath, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("write %s: %w", claudecontract.FileSkillMD, err)
	}

	return nil
}

// DiscoverSkills finds all SKILL.md files in the given .claude directory.
// It searches in the skills/ subdirectory.
func DiscoverSkills(claudeDir string) ([]*Skill, error) {
	skillsDir := filepath.Join(claudeDir, claudecontract.DirSkills)

	// Check if skills directory exists
	if !dirExists(skillsDir) {
		return nil, nil // No skills directory, return empty list
	}

	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		return nil, fmt.Errorf("read skills directory: %w", err)
	}

	var skills []*Skill
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		skillPath := filepath.Join(skillsDir, entry.Name())
		skillFile := filepath.Join(skillPath, claudecontract.FileSkillMD)

		// Check if SKILL.md exists in this directory
		if !fileExists(skillFile) {
			continue
		}

		skill, err := ParseSkillMD(skillPath)
		if err != nil {
			// Log warning but continue discovering other skills
			continue
		}

		skills = append(skills, skill)
	}

	// Sort by name for consistent ordering
	sort.Slice(skills, func(i, j int) bool {
		return skills[i].Name < skills[j].Name
	})

	return skills, nil
}

// ListSkillResources returns the files in a skill's resource subdirectory.
func ListSkillResources(skillDir, resourceType string) ([]string, error) {
	resourceDir := filepath.Join(skillDir, resourceType)
	if !dirExists(resourceDir) {
		return nil, nil
	}

	entries, err := os.ReadDir(resourceDir)
	if err != nil {
		return nil, fmt.Errorf("read %s directory: %w", resourceType, err)
	}

	var files []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		files = append(files, entry.Name())
	}

	sort.Strings(files)
	return files, nil
}

// dirExists checks if a directory exists.
func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// fileExists checks if a file exists.
func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
