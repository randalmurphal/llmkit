package codexconfig

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type RuleDecision string

const (
	RuleAllow     RuleDecision = "allow"
	RulePrompt    RuleDecision = "prompt"
	RuleForbidden RuleDecision = "forbidden"
)

// PrefixPattern models Codex's prefix_rule pattern as positional alternatives.
// A literal token is represented as a one-element slice.
type PrefixPattern [][]string

func LiteralPattern(tokens ...string) PrefixPattern {
	pattern := make(PrefixPattern, 0, len(tokens))
	for _, token := range tokens {
		pattern = append(pattern, []string{token})
	}
	return pattern
}

type PrefixRule struct {
	Pattern       PrefixPattern `json:"pattern"`
	Decision      RuleDecision  `json:"decision,omitempty"`
	Justification string        `json:"justification,omitempty"`
	Match         []string      `json:"match,omitempty"`
	NotMatch      []string      `json:"not_match,omitempty"`
}

type RuleFile struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

func LoadRuleFile(path string) (*RuleFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &RuleFile{Path: path}, nil
		}
		return nil, fmt.Errorf("read rule file: %w", err)
	}
	return &RuleFile{Path: path, Content: string(data)}, nil
}

func SaveRuleFile(path string, file *RuleFile) error {
	if file == nil {
		file = &RuleFile{Path: path}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create rules dir: %w", err)
	}
	return os.WriteFile(path, []byte(file.Content), 0o644)
}

func (r PrefixRule) Render() string {
	var b strings.Builder
	b.WriteString("prefix_rule(\n")
	b.WriteString("    pattern = ")
	b.WriteString(renderPattern(r.Pattern))
	b.WriteString(",\n")
	if r.Decision != "" {
		b.WriteString("    decision = ")
		b.WriteString(renderString(string(r.Decision)))
		b.WriteString(",\n")
	}
	if r.Justification != "" {
		b.WriteString("    justification = ")
		b.WriteString(renderString(r.Justification))
		b.WriteString(",\n")
	}
	if len(r.Match) > 0 {
		b.WriteString("    match = ")
		b.WriteString(renderStringSlice(r.Match))
		b.WriteString(",\n")
	}
	if len(r.NotMatch) > 0 {
		b.WriteString("    not_match = ")
		b.WriteString(renderStringSlice(r.NotMatch))
		b.WriteString(",\n")
	}
	b.WriteString(")\n")
	return b.String()
}

func UpsertManagedPrefixRule(file *RuleFile, id string, rule PrefixRule) {
	if file == nil {
		return
	}
	begin := managedRuleBegin(id)
	end := managedRuleEnd(id)
	block := begin + "\n" + rule.Render() + end + "\n"

	if strings.Contains(file.Content, begin) && strings.Contains(file.Content, end) {
		start := strings.Index(file.Content, begin)
		finish := strings.Index(file.Content[start:], end)
		if finish >= 0 {
			finish += start + len(end)
			file.Content = file.Content[:start] + block + strings.TrimLeft(file.Content[finish:], "\n")
			return
		}
	}

	if file.Content != "" && !strings.HasSuffix(file.Content, "\n") {
		file.Content += "\n"
	}
	file.Content += block
}

func RemoveManagedRule(file *RuleFile, id string) bool {
	if file == nil {
		return false
	}
	begin := managedRuleBegin(id)
	end := managedRuleEnd(id)
	start := strings.Index(file.Content, begin)
	if start < 0 {
		return false
	}
	finish := strings.Index(file.Content[start:], end)
	if finish < 0 {
		return false
	}
	finish += start + len(end)
	file.Content = strings.TrimLeft(file.Content[:start]+file.Content[finish:], "\n")
	return true
}

func managedRuleBegin(id string) string { return "# BEGIN llmkit:" + id }
func managedRuleEnd(id string) string   { return "# END llmkit:" + id }

func renderPattern(pattern PrefixPattern) string {
	parts := make([]string, 0, len(pattern))
	for _, token := range pattern {
		if len(token) == 1 {
			parts = append(parts, renderString(token[0]))
			continue
		}
		parts = append(parts, renderStringSlice(token))
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

func renderString(value string) string {
	data, _ := json.Marshal(value)
	return string(data)
}

func renderStringSlice(values []string) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, renderString(value))
	}
	return "[" + strings.Join(parts, ", ") + "]"
}
