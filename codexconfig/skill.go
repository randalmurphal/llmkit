package codexconfig

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type StringOrArray []string

func (s *StringOrArray) UnmarshalYAML(value *yaml.Node) error {
	var arr []string
	if err := value.Decode(&arr); err == nil {
		*s = arr
		return nil
	}

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

type Skill struct {
	Name          string         `yaml:"name" json:"name"`
	Description   string         `yaml:"description" json:"description"`
	AllowedTools  StringOrArray  `yaml:"allowed-tools,omitempty" json:"allowed_tools,omitempty"`
	Version       string         `yaml:"version,omitempty" json:"version,omitempty"`
	Content       string         `yaml:"-" json:"content"`
	Path          string         `yaml:"-" json:"path"`
	HasReferences bool           `yaml:"-" json:"has_references"`
	HasScripts    bool           `yaml:"-" json:"has_scripts"`
	HasAssets     bool           `yaml:"-" json:"has_assets"`
	Metadata      *SkillMetadata `yaml:"-" json:"metadata,omitempty"`
}

type SkillInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Path        string `json:"path"`
}

type SkillMetadata struct {
	Interface *SkillInterface `yaml:"interface,omitempty" json:"interface,omitempty"`
	Policy    *SkillPolicy    `yaml:"policy,omitempty" json:"policy,omitempty"`
}

type SkillInterface struct {
	DisplayName      string `yaml:"display_name,omitempty" json:"display_name,omitempty"`
	ShortDescription string `yaml:"short_description,omitempty" json:"short_description,omitempty"`
	IconSmall        string `yaml:"icon_small,omitempty" json:"icon_small,omitempty"`
	IconLarge        string `yaml:"icon_large,omitempty" json:"icon_large,omitempty"`
	BrandColor       string `yaml:"brand_color,omitempty" json:"brand_color,omitempty"`
	DefaultPrompt    string `yaml:"default_prompt,omitempty" json:"default_prompt,omitempty"`
}

type SkillPolicy struct {
	AllowImplicitInvocation bool `yaml:"allow_implicit_invocation,omitempty" json:"allow_implicit_invocation,omitempty"`
}

type SkillToggle struct {
	Path    string `toml:"path"`
	Enabled bool   `toml:"enabled"`
}

func (s *Skill) Info() SkillInfo {
	return SkillInfo{Name: s.Name, Description: s.Description, Path: s.Path}
}

func (s *Skill) Validate() error {
	if s.Name == "" {
		return errors.New("skill name is required")
	}
	if s.Description == "" {
		return errors.New("skill description is required")
	}
	return nil
}

func ParseSkillMD(path string) (*Skill, error) {
	filePath := path
	dirPath := path

	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("stat path: %w", err)
	}
	if info.IsDir() {
		filePath = filepath.Join(path, FileSkillMD)
	} else {
		dirPath = filepath.Dir(path)
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", FileSkillMD, err)
	}

	skill, err := parseSkillContent(data)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", FileSkillMD, err)
	}

	skill.Path = dirPath
	skill.HasReferences = dirExists(filepath.Join(dirPath, "references"))
	skill.HasScripts = dirExists(filepath.Join(dirPath, "scripts"))
	skill.HasAssets = dirExists(filepath.Join(dirPath, "assets"))

	metaPath := filepath.Join(dirPath, "agents", "openai.yaml")
	if fileExists(metaPath) {
		metaData, err := os.ReadFile(metaPath)
		if err == nil {
			var meta SkillMetadata
			if err := yaml.Unmarshal(metaData, &meta); err == nil {
				skill.Metadata = &meta
			}
		}
	}

	return skill, nil
}

func parseSkillContent(data []byte) (*Skill, error) {
	if !bytes.HasPrefix(data, []byte("---")) {
		return nil, errors.New("SKILL.md must start with YAML frontmatter (---)")
	}

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

	skill := &Skill{}
	frontmatter := strings.Join(frontmatterLines, "\n")
	if err := yaml.Unmarshal([]byte(frontmatter), skill); err != nil {
		return nil, fmt.Errorf("parse frontmatter: %w", err)
	}
	skill.Content = strings.TrimSpace(strings.Join(contentLines, "\n"))
	return skill, nil
}

func WriteSkillMD(skill *Skill, dir string) error {
	if err := skill.Validate(); err != nil {
		return fmt.Errorf("validate skill: %w", err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

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

	var buf bytes.Buffer
	buf.WriteString("---\n")
	buf.Write(fmBytes)
	buf.WriteString("---\n\n")
	buf.WriteString(skill.Content)
	if !strings.HasSuffix(skill.Content, "\n") {
		buf.WriteString("\n")
	}

	filePath := filepath.Join(dir, FileSkillMD)
	if err := os.WriteFile(filePath, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", FileSkillMD, err)
	}
	return nil
}

func DiscoverSkills(projectRoot, cwd string) ([]*Skill, error) {
	searchRoots := SkillSearchRoots(projectRoot, cwd)
	var skills []*Skill
	for _, root := range searchRoots {
		dir := RepoSkillsDir(root)
		found, err := discoverSkillsInDir(dir)
		if err != nil {
			return nil, err
		}
		skills = append(skills, found...)
	}

	userDir, err := UserSkillsDir()
	if err == nil {
		found, err := discoverSkillsInDir(userDir)
		if err != nil {
			return nil, err
		}
		skills = append(skills, found...)
	}

	sort.Slice(skills, func(i, j int) bool {
		if skills[i].Name == skills[j].Name {
			return skills[i].Path < skills[j].Path
		}
		return skills[i].Name < skills[j].Name
	})
	return skills, nil
}

func SkillSearchRoots(projectRoot, cwd string) []string {
	projectRoot = cleanPath(projectRoot)
	cwd = cleanPath(cwd)
	if cwd == "" {
		cwd = projectRoot
	}
	if cwd == "" {
		return nil
	}

	var roots []string
	seen := map[string]bool{}
	cur := cwd
	for {
		if !seen[cur] {
			roots = append(roots, cur)
			seen[cur] = true
		}
		if projectRoot == "" || cur == projectRoot {
			break
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			break
		}
		cur = parent
	}

	// Return from root to cwd to match Codex precedence.
	for i, j := 0, len(roots)-1; i < j; i, j = i+1, j-1 {
		roots[i], roots[j] = roots[j], roots[i]
	}
	return roots
}

func discoverSkillsInDir(skillsDir string) ([]*Skill, error) {
	if !dirExists(skillsDir) {
		return nil, nil
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
		skill, err := ParseSkillMD(skillPath)
		if err != nil {
			continue
		}
		skills = append(skills, skill)
	}
	return skills, nil
}
