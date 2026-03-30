package codexconfig

import (
	"fmt"
	"os"
	"path/filepath"
)

type InstructionFile struct {
	Path    string `json:"path"`
	Source  string `json:"source"`
	Content string `json:"content"`
}

type InstructionsHierarchy struct {
	Global  *InstructionFile   `json:"global,omitempty"`
	Project []*InstructionFile `json:"project,omitempty"`
}

func ResolveInstructions(projectRoot, cwd string, cfg *ConfigFile) (*InstructionsHierarchy, error) {
	h := &InstructionsHierarchy{}

	home, err := codexHomeDir()
	if err != nil {
		return nil, err
	}
	global, err := loadFirstNonEmpty(home, []string{FileAgentsOverride, FileAgentsMD})
	if err != nil {
		return nil, err
	}
	if global != nil {
		global.Source = "global"
		h.Global = global
	}

	if cfg != nil && cfg.ModelInstructionsFile != "" {
		path := cfg.ModelInstructionsFile
		if !filepath.IsAbs(path) && projectRoot != "" {
			path = filepath.Join(projectRoot, path)
		}
		file, err := loadInstructionFile(path, "project")
		if err != nil {
			return nil, err
		}
		if file != nil {
			h.Project = append(h.Project, file)
		}
		return h, nil
	}

	roots := SkillSearchRoots(projectRoot, cwd)
	fallbacks := []string{}
	maxBytes := 32 * 1024
	if cfg != nil {
		fallbacks = append(fallbacks, cfg.ProjectDocFallbackFilenames...)
		if cfg.ProjectDocMaxBytes > 0 {
			maxBytes = cfg.ProjectDocMaxBytes
		}
	}

	totalBytes := 0
	for _, root := range roots {
		names := append([]string{FileAgentsOverride, FileAgentsMD}, fallbacks...)
		file, err := loadFirstNonEmpty(root, names)
		if err != nil {
			return nil, err
		}
		if file == nil {
			continue
		}
		if totalBytes+len(file.Content) > maxBytes {
			break
		}
		totalBytes += len(file.Content)
		file.Source = "project"
		h.Project = append(h.Project, file)
	}

	return h, nil
}

func loadFirstNonEmpty(dir string, names []string) (*InstructionFile, error) {
	for _, name := range names {
		path := filepath.Join(dir, name)
		file, err := loadInstructionFile(path, "")
		if err != nil {
			return nil, err
		}
		if file != nil {
			return file, nil
		}
	}
	return nil, nil
}

func loadInstructionFile(path, source string) (*InstructionFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read instructions file: %w", err)
	}
	if len(bytesTrimSpace(data)) == 0 {
		return nil, nil
	}
	return &InstructionFile{
		Path:    path,
		Source:  source,
		Content: string(data),
	}, nil
}

func bytesTrimSpace(data []byte) []byte {
	start := 0
	for start < len(data) && (data[start] == ' ' || data[start] == '\n' || data[start] == '\t' || data[start] == '\r') {
		start++
	}
	end := len(data)
	for end > start && (data[end-1] == ' ' || data[end-1] == '\n' || data[end-1] == '\t' || data[end-1] == '\r') {
		end--
	}
	return data[start:end]
}
