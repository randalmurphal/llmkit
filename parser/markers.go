package parser

import (
	"regexp"
	"strings"
	"sync"
)

// Marker represents a detected XML-style marker in content.
// Markers are used for structured signaling in LLM responses,
// such as phase completion or blocking indicators.
type Marker struct {
	// Tag is the marker name (e.g., "phase_complete", "phase_blocked").
	Tag string

	// Value is the content between the opening and closing tags.
	Value string

	// Raw is the full matched text including tags.
	Raw string
}

// MarkerMatcher finds XML-style markers in content.
// It compiles regex patterns for each registered tag and caches them
// for efficient repeated matching.
//
// Example markers:
//
//	<phase_complete>true</phase_complete>
//	<phase_blocked>reason: need clarification</phase_blocked>
//	<implement_complete>true</implement_complete>
type MarkerMatcher struct {
	tags     []string
	patterns map[string]*regexp.Regexp
	mu       sync.RWMutex
}

// NewMarkerMatcher creates a matcher for the given tag names.
// Tags should be provided without angle brackets.
//
// Example:
//
//	matcher := NewMarkerMatcher("phase_complete", "phase_blocked")
func NewMarkerMatcher(tags ...string) *MarkerMatcher {
	m := &MarkerMatcher{
		tags:     make([]string, 0, len(tags)),
		patterns: make(map[string]*regexp.Regexp, len(tags)),
	}

	for _, tag := range tags {
		m.addTag(tag)
	}

	return m
}

// addTag compiles and caches a regex pattern for the tag.
func (m *MarkerMatcher) addTag(tag string) {
	// Pattern: <tag>content</tag> with optional whitespace
	// Uses (?s) for dot-matches-newline (multiline content)
	pattern := regexp.MustCompile(`(?s)<` + regexp.QuoteMeta(tag) + `>(.*?)</` + regexp.QuoteMeta(tag) + `>`)

	m.mu.Lock()
	m.tags = append(m.tags, tag)
	m.patterns[tag] = pattern
	m.mu.Unlock()
}

// AddTag adds a new tag to match. This is safe for concurrent use.
func (m *MarkerMatcher) AddTag(tag string) {
	m.mu.RLock()
	_, exists := m.patterns[tag]
	m.mu.RUnlock()

	if !exists {
		m.addTag(tag)
	}
}

// Tags returns the list of tags this matcher looks for.
func (m *MarkerMatcher) Tags() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]string, len(m.tags))
	copy(result, m.tags)
	return result
}

// FindAll returns all markers found in content for all registered tags.
func (m *MarkerMatcher) FindAll(content string) []Marker {
	var markers []Marker

	m.mu.RLock()
	defer m.mu.RUnlock()

	for tag, pattern := range m.patterns {
		matches := pattern.FindAllStringSubmatch(content, -1)
		for _, match := range matches {
			if len(match) >= 2 {
				markers = append(markers, Marker{
					Tag:   tag,
					Value: strings.TrimSpace(match[1]),
					Raw:   match[0],
				})
			}
		}
	}

	return markers
}

// FindAllForTag returns all markers for a specific tag.
func (m *MarkerMatcher) FindAllForTag(content, tag string) []Marker {
	m.mu.RLock()
	pattern, ok := m.patterns[tag]
	m.mu.RUnlock()

	if !ok {
		return nil
	}

	var markers []Marker
	matches := pattern.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) >= 2 {
			markers = append(markers, Marker{
				Tag:   tag,
				Value: strings.TrimSpace(match[1]),
				Raw:   match[0],
			})
		}
	}

	return markers
}

// FindFirst returns the first marker found for the given tag.
// Returns false if no marker is found.
func (m *MarkerMatcher) FindFirst(content, tag string) (Marker, bool) {
	m.mu.RLock()
	pattern, ok := m.patterns[tag]
	m.mu.RUnlock()

	if !ok {
		return Marker{}, false
	}

	match := pattern.FindStringSubmatch(content)
	if len(match) < 2 {
		return Marker{}, false
	}

	return Marker{
		Tag:   tag,
		Value: strings.TrimSpace(match[1]),
		Raw:   match[0],
	}, true
}

// Contains checks if any marker with the given tag exists in content.
func (m *MarkerMatcher) Contains(content, tag string) bool {
	m.mu.RLock()
	pattern, ok := m.patterns[tag]
	m.mu.RUnlock()

	if !ok {
		return false
	}

	return pattern.MatchString(content)
}

// ContainsValue checks if a marker with the specific tag and value exists.
// The value comparison is case-insensitive and trims whitespace.
func (m *MarkerMatcher) ContainsValue(content, tag, value string) bool {
	marker, found := m.FindFirst(content, tag)
	if !found {
		return false
	}

	return strings.EqualFold(strings.TrimSpace(marker.Value), strings.TrimSpace(value))
}

// ContainsAny checks if any of the registered markers exist in content.
func (m *MarkerMatcher) ContainsAny(content string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, pattern := range m.patterns {
		if pattern.MatchString(content) {
			return true
		}
	}

	return false
}

// GetValue extracts the value of a marker with the given tag.
// Returns empty string if the marker is not found.
func (m *MarkerMatcher) GetValue(content, tag string) string {
	marker, found := m.FindFirst(content, tag)
	if !found {
		return ""
	}
	return marker.Value
}

// Common marker matchers for orchestration systems.
var (
	// PhaseMarkers matches common phase completion markers.
	PhaseMarkers = NewMarkerMatcher(
		"phase_complete",
		"phase_blocked",
	)

	// TaskMarkers matches task-level markers.
	TaskMarkers = NewMarkerMatcher(
		"task_complete",
		"task_blocked",
		"task_failed",
	)
)

// IsPhaseComplete checks if the content contains a phase completion marker.
// This is a convenience function using the default PhaseMarkers matcher.
func IsPhaseComplete(content string) bool {
	return PhaseMarkers.ContainsValue(content, "phase_complete", "true")
}

// IsPhaseBlocked checks if the content contains a phase blocked marker.
// This is a convenience function using the default PhaseMarkers matcher.
func IsPhaseBlocked(content string) bool {
	return PhaseMarkers.Contains(content, "phase_blocked")
}

// GetBlockedReason extracts the reason from a phase_blocked marker.
// Returns empty string if no blocked marker is found.
func GetBlockedReason(content string) string {
	return PhaseMarkers.GetValue(content, "phase_blocked")
}
