package parser

import (
	"testing"
)

func TestNewMarkerMatcher(t *testing.T) {
	m := NewMarkerMatcher("test_complete", "test_blocked")

	tags := m.Tags()
	if len(tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(tags))
	}
}

func TestMarkerMatcher_FindFirst(t *testing.T) {
	m := NewMarkerMatcher("phase_complete", "phase_blocked")

	tests := []struct {
		name      string
		content   string
		tag       string
		wantFound bool
		wantValue string
	}{
		{
			name:      "simple completion marker",
			content:   "Done! <phase_complete>true</phase_complete>",
			tag:       "phase_complete",
			wantFound: true,
			wantValue: "true",
		},
		{
			name:      "blocked marker with reason",
			content:   "Cannot proceed. <phase_blocked>reason: need clarification</phase_blocked>",
			tag:       "phase_blocked",
			wantFound: true,
			wantValue: "reason: need clarification",
		},
		{
			name:      "marker with whitespace in value",
			content:   "<phase_complete>  true  </phase_complete>",
			tag:       "phase_complete",
			wantFound: true,
			wantValue: "true", // whitespace should be trimmed
		},
		{
			name:      "marker not found",
			content:   "No markers here",
			tag:       "phase_complete",
			wantFound: false,
			wantValue: "",
		},
		{
			name:      "wrong tag",
			content:   "<phase_complete>true</phase_complete>",
			tag:       "phase_blocked",
			wantFound: false,
			wantValue: "",
		},
		{
			name:      "multiline value",
			content:   "<phase_blocked>\nreason: need info\ndetails: foo\n</phase_blocked>",
			tag:       "phase_blocked",
			wantFound: true,
			wantValue: "reason: need info\ndetails: foo",
		},
		{
			name:      "unregistered tag",
			content:   "<unknown_tag>value</unknown_tag>",
			tag:       "unknown_tag",
			wantFound: false, // tag not registered
			wantValue: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			marker, found := m.FindFirst(tt.content, tt.tag)

			if found != tt.wantFound {
				t.Errorf("FindFirst() found = %v, want %v", found, tt.wantFound)
			}

			if found && marker.Value != tt.wantValue {
				t.Errorf("FindFirst() value = %q, want %q", marker.Value, tt.wantValue)
			}
		})
	}
}

func TestMarkerMatcher_FindAll(t *testing.T) {
	m := NewMarkerMatcher("item")

	content := "First <item>one</item> and <item>two</item> and <item>three</item>"
	markers := m.FindAll(content)

	if len(markers) != 3 {
		t.Errorf("expected 3 markers, got %d", len(markers))
	}

	expectedValues := []string{"one", "two", "three"}
	for i, marker := range markers {
		if marker.Value != expectedValues[i] {
			t.Errorf("marker[%d].Value = %q, want %q", i, marker.Value, expectedValues[i])
		}
		if marker.Tag != "item" {
			t.Errorf("marker[%d].Tag = %q, want %q", i, marker.Tag, "item")
		}
	}
}

func TestMarkerMatcher_FindAllForTag(t *testing.T) {
	m := NewMarkerMatcher("a", "b")

	content := "<a>1</a> <b>2</b> <a>3</a>"

	aMarkers := m.FindAllForTag(content, "a")
	if len(aMarkers) != 2 {
		t.Errorf("expected 2 'a' markers, got %d", len(aMarkers))
	}

	bMarkers := m.FindAllForTag(content, "b")
	if len(bMarkers) != 1 {
		t.Errorf("expected 1 'b' marker, got %d", len(bMarkers))
	}
}

func TestMarkerMatcher_Contains(t *testing.T) {
	m := NewMarkerMatcher("phase_complete")

	tests := []struct {
		name    string
		content string
		want    bool
	}{
		{
			name:    "marker present",
			content: "Done <phase_complete>true</phase_complete>",
			want:    true,
		},
		{
			name:    "marker absent",
			content: "No marker here",
			want:    false,
		},
		{
			name:    "partial marker (no closing tag)",
			content: "<phase_complete>true",
			want:    false,
		},
		{
			name:    "partial marker (no opening tag)",
			content: "true</phase_complete>",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := m.Contains(tt.content, "phase_complete"); got != tt.want {
				t.Errorf("Contains() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMarkerMatcher_ContainsValue(t *testing.T) {
	m := NewMarkerMatcher("phase_complete")

	tests := []struct {
		name    string
		content string
		tag     string
		value   string
		want    bool
	}{
		{
			name:    "exact match",
			content: "<phase_complete>true</phase_complete>",
			tag:     "phase_complete",
			value:   "true",
			want:    true,
		},
		{
			name:    "case insensitive",
			content: "<phase_complete>TRUE</phase_complete>",
			tag:     "phase_complete",
			value:   "true",
			want:    true,
		},
		{
			name:    "whitespace tolerance",
			content: "<phase_complete>  true  </phase_complete>",
			tag:     "phase_complete",
			value:   "true",
			want:    true,
		},
		{
			name:    "different value",
			content: "<phase_complete>false</phase_complete>",
			tag:     "phase_complete",
			value:   "true",
			want:    false,
		},
		{
			name:    "marker not present",
			content: "no markers",
			tag:     "phase_complete",
			value:   "true",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := m.ContainsValue(tt.content, tt.tag, tt.value); got != tt.want {
				t.Errorf("ContainsValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMarkerMatcher_ContainsAny(t *testing.T) {
	m := NewMarkerMatcher("phase_complete", "phase_blocked")

	tests := []struct {
		name    string
		content string
		want    bool
	}{
		{
			name:    "first tag present",
			content: "<phase_complete>true</phase_complete>",
			want:    true,
		},
		{
			name:    "second tag present",
			content: "<phase_blocked>reason</phase_blocked>",
			want:    true,
		},
		{
			name:    "both tags present",
			content: "<phase_complete>true</phase_complete> <phase_blocked>reason</phase_blocked>",
			want:    true,
		},
		{
			name:    "no tags present",
			content: "nothing here",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := m.ContainsAny(tt.content); got != tt.want {
				t.Errorf("ContainsAny() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMarkerMatcher_GetValue(t *testing.T) {
	m := NewMarkerMatcher("phase_blocked")

	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "value present",
			content: "<phase_blocked>need more info</phase_blocked>",
			want:    "need more info",
		},
		{
			name:    "value absent",
			content: "no marker",
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := m.GetValue(tt.content, "phase_blocked"); got != tt.want {
				t.Errorf("GetValue() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMarkerMatcher_AddTag(t *testing.T) {
	m := NewMarkerMatcher("initial")

	// Tag shouldn't exist yet
	if m.Contains("<new_tag>value</new_tag>", "new_tag") {
		t.Error("new_tag should not be registered yet")
	}

	// Add the tag
	m.AddTag("new_tag")

	// Now it should work
	if !m.Contains("<new_tag>value</new_tag>", "new_tag") {
		t.Error("new_tag should be registered after AddTag")
	}

	// Adding same tag again should be idempotent
	m.AddTag("new_tag")
	tags := m.Tags()
	count := 0
	for _, tag := range tags {
		if tag == "new_tag" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected 1 'new_tag', got %d", count)
	}
}

func TestIsPhaseComplete(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    bool
	}{
		{
			name:    "complete true",
			content: "Done! <phase_complete>true</phase_complete>",
			want:    true,
		},
		{
			name:    "complete false",
			content: "Done! <phase_complete>false</phase_complete>",
			want:    false,
		},
		{
			name:    "no marker",
			content: "Still working",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsPhaseComplete(tt.content); got != tt.want {
				t.Errorf("IsPhaseComplete() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsPhaseBlocked(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    bool
	}{
		{
			name:    "blocked",
			content: "<phase_blocked>need clarification</phase_blocked>",
			want:    true,
		},
		{
			name:    "not blocked",
			content: "Working fine",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsPhaseBlocked(tt.content); got != tt.want {
				t.Errorf("IsPhaseBlocked() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetBlockedReason(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "has reason",
			content: "<phase_blocked>need more information about requirements</phase_blocked>",
			want:    "need more information about requirements",
		},
		{
			name:    "no blocked marker",
			content: "all good",
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetBlockedReason(tt.content); got != tt.want {
				t.Errorf("GetBlockedReason() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMarkerRaw(t *testing.T) {
	m := NewMarkerMatcher("test")

	content := "prefix <test>value</test> suffix"
	marker, found := m.FindFirst(content, "test")

	if !found {
		t.Fatal("marker not found")
	}

	if marker.Raw != "<test>value</test>" {
		t.Errorf("Raw = %q, want %q", marker.Raw, "<test>value</test>")
	}
}

// Test concurrent access
func TestMarkerMatcher_Concurrent(t *testing.T) {
	m := NewMarkerMatcher("tag1", "tag2")

	done := make(chan bool)

	// Reader goroutine
	go func() {
		for i := 0; i < 100; i++ {
			m.Contains("<tag1>value</tag1>", "tag1")
			m.FindFirst("<tag2>value</tag2>", "tag2")
			m.FindAll("<tag1>a</tag1><tag2>b</tag2>")
		}
		done <- true
	}()

	// Writer goroutine (adding tags)
	go func() {
		for i := 0; i < 10; i++ {
			m.AddTag("dynamic_tag")
		}
		done <- true
	}()

	<-done
	<-done
}

// Benchmark
func BenchmarkMarkerMatcher_FindFirst(b *testing.B) {
	m := NewMarkerMatcher("phase_complete", "phase_blocked")
	content := "Some text before <phase_complete>true</phase_complete> and after"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.FindFirst(content, "phase_complete")
	}
}

func BenchmarkMarkerMatcher_Contains(b *testing.B) {
	m := NewMarkerMatcher("phase_complete")
	content := "Some text before <phase_complete>true</phase_complete> and after"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Contains(content, "phase_complete")
	}
}

func BenchmarkMarkerMatcher_ContainsValue(b *testing.B) {
	m := NewMarkerMatcher("phase_complete")
	content := "Some text before <phase_complete>true</phase_complete> and after"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.ContainsValue(content, "phase_complete", "true")
	}
}
