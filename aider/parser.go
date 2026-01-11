package aider

import (
	"regexp"
	"strings"

	"github.com/randalmurphal/llmkit/provider"
)

// ParseAiderOutput parses Aider's text output into a provider.Response.
// Aider does not have native JSON output, so we parse the text for patterns.
func ParseAiderOutput(output string) *provider.Response {
	resp := &provider.Response{
		Content:      strings.TrimSpace(output),
		FinishReason: "stop",
	}

	// Parse for edit markers
	edits := ParseEditMarkers(output)
	if len(edits) > 0 {
		resp.Metadata = map[string]any{
			"edits": edits,
		}
	}

	return resp
}

// EditMarker represents a detected file edit.
type EditMarker struct {
	Action string // "created", "modified", "applied"
	File   string
}

// ParseEditMarkers extracts file edit markers from Aider output.
// Aider outputs patterns like:
//   - "Applied edit to file.go"
//   - "Created file.go"
//   - "Modified file.go"
//   - "Wrote file.go"
func ParseEditMarkers(output string) []EditMarker {
	var markers []EditMarker

	// Pattern: "Applied edit to <file>"
	appliedRe := regexp.MustCompile(`(?i)Applied edit to (.+)`)
	for _, match := range appliedRe.FindAllStringSubmatch(output, -1) {
		if len(match) > 1 {
			markers = append(markers, EditMarker{
				Action: "applied",
				File:   strings.TrimSpace(match[1]),
			})
		}
	}

	// Pattern: "Created <file>"
	createdRe := regexp.MustCompile(`(?i)Created (.+\.[\w]+)`)
	for _, match := range createdRe.FindAllStringSubmatch(output, -1) {
		if len(match) > 1 {
			markers = append(markers, EditMarker{
				Action: "created",
				File:   strings.TrimSpace(match[1]),
			})
		}
	}

	// Pattern: "Wrote <file>"
	wroteRe := regexp.MustCompile(`(?i)Wrote (.+\.[\w]+)`)
	for _, match := range wroteRe.FindAllStringSubmatch(output, -1) {
		if len(match) > 1 {
			markers = append(markers, EditMarker{
				Action: "modified",
				File:   strings.TrimSpace(match[1]),
			})
		}
	}

	// Pattern: "Add <file> to the chat" (file added for editing)
	addedRe := regexp.MustCompile(`(?i)Add (.+\.[\w]+) to the chat`)
	for _, match := range addedRe.FindAllStringSubmatch(output, -1) {
		if len(match) > 1 {
			markers = append(markers, EditMarker{
				Action: "added",
				File:   strings.TrimSpace(match[1]),
			})
		}
	}

	// Deduplicate
	return deduplicateMarkers(markers)
}

// deduplicateMarkers removes duplicate edit markers.
func deduplicateMarkers(markers []EditMarker) []EditMarker {
	seen := make(map[string]bool)
	var result []EditMarker

	for _, m := range markers {
		key := m.Action + ":" + m.File
		if !seen[key] {
			seen[key] = true
			result = append(result, m)
		}
	}

	return result
}

// ExtractErrorMessage extracts error messages from Aider output.
func ExtractErrorMessage(output string) string {
	// Look for common error patterns
	errorPatterns := []string{
		`(?i)Error: (.+)`,
		`(?i)Exception: (.+)`,
		`(?i)Failed to (.+)`,
		`(?i)Could not (.+)`,
	}

	for _, pattern := range errorPatterns {
		re := regexp.MustCompile(pattern)
		if match := re.FindStringSubmatch(output); len(match) > 1 {
			return match[1]
		}
	}

	return ""
}

// ContainsCommit checks if the output indicates a git commit was made.
func ContainsCommit(output string) bool {
	commitPatterns := []string{
		`(?i)Committed`,
		`(?i)commit [a-f0-9]{7,40}`,
		`(?i)Created commit`,
	}

	for _, pattern := range commitPatterns {
		re := regexp.MustCompile(pattern)
		if re.MatchString(output) {
			return true
		}
	}

	return false
}

// ExtractCommitHash extracts a git commit hash from Aider output.
func ExtractCommitHash(output string) string {
	re := regexp.MustCompile(`[a-f0-9]{7,40}`)
	if match := re.FindString(output); match != "" {
		return match
	}
	return ""
}
