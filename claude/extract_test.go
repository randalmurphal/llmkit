package claude

import (
	"context"
	"encoding/json"
	"testing"
)

func TestFindJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantJSON bool
	}{
		{
			name:     "pure JSON object",
			input:    `{"status": "pass", "summary": "All tests pass"}`,
			wantJSON: true,
		},
		{
			name:     "pure JSON array",
			input:    `[{"name": "test1"}, {"name": "test2"}]`,
			wantJSON: true,
		},
		{
			name:     "JSON in code block",
			input:    "Here are the results:\n```json\n{\"status\": \"pass\"}\n```\nDone.",
			wantJSON: true,
		},
		{
			name:     "JSON mixed with text",
			input:    "The review findings are:\n{\"round\": 1, \"summary\": \"Found issues\"}\nPlease address these.",
			wantJSON: true,
		},
		{
			name:     "no JSON",
			input:    "Just some plain text without any JSON.",
			wantJSON: false,
		},
		{
			name:     "invalid JSON",
			input:    `{"status": "pass", summary: "missing quotes"}`,
			wantJSON: false,
		},
		{
			name:     "nested JSON",
			input:    `{"outer": {"inner": {"deep": "value"}}}`,
			wantJSON: true,
		},
		{
			name:     "JSON with escaped quotes",
			input:    `{"message": "He said \"hello\""}`,
			wantJSON: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findJSON(tt.input)
			if tt.wantJSON && result == "" {
				t.Errorf("findJSON() returned empty, want JSON")
			}
			if !tt.wantJSON && result != "" {
				t.Errorf("findJSON() = %q, want empty", result)
			}
			if result != "" && !json.Valid([]byte(result)) {
				t.Errorf("findJSON() returned invalid JSON: %s", result)
			}
		})
	}
}

type testStruct struct {
	Status  string `json:"status"`
	Summary string `json:"summary"`
	Count   int    `json:"count,omitempty"`
}

func TestExtractStructured_DirectParsing(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantStatus string
		wantErr    bool
	}{
		{
			name:       "pure JSON",
			input:      `{"status": "pass", "summary": "All good"}`,
			wantStatus: "pass",
		},
		{
			name:       "JSON in code block",
			input:      "```json\n{\"status\": \"fail\", \"summary\": \"Issues found\"}\n```",
			wantStatus: "fail",
		},
		{
			name:       "JSON with surrounding text",
			input:      "Here's the result:\n{\"status\": \"pending\", \"summary\": \"In progress\"}",
			wantStatus: "pending",
		},
	}

	schema := `{
		"type": "object",
		"properties": {
			"status": {"type": "string"},
			"summary": {"type": "string"},
			"count": {"type": "integer"}
		},
		"required": ["status", "summary"]
	}`

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result testStruct
			err := ExtractStructured(context.Background(), nil, tt.input, schema, &result, nil)

			if tt.wantErr {
				if err == nil {
					t.Error("ExtractStructured() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("ExtractStructured() error = %v", err)
				return
			}

			if result.Status != tt.wantStatus {
				t.Errorf("Status = %q, want %q", result.Status, tt.wantStatus)
			}
		})
	}
}

func TestExtractStructured_NoJSON_NoClient(t *testing.T) {
	var result testStruct
	err := ExtractStructured(
		context.Background(),
		nil, // no client
		"This is just plain text with no JSON",
		`{"type": "object"}`,
		&result,
		nil,
	)

	if err == nil {
		t.Error("ExtractStructured() expected error when no JSON and no client")
	}
}

func TestExtractStructured_FallbackExtraction(t *testing.T) {
	// Create a mock client that returns valid JSON
	mockClient := NewMockClient(`{"status": "extracted", "summary": "From LLM"}`)

	var result testStruct
	err := ExtractStructured(
		context.Background(),
		mockClient,
		"This text has no valid JSON but describes that status is extracted and summary is From LLM",
		`{"type": "object", "properties": {"status": {"type": "string"}, "summary": {"type": "string"}}}`,
		&result,
		nil,
	)

	if err != nil {
		t.Errorf("ExtractStructured() error = %v", err)
		return
	}

	if result.Status != "extracted" {
		t.Errorf("Status = %q, want %q", result.Status, "extracted")
	}
	if result.Summary != "From LLM" {
		t.Errorf("Summary = %q, want %q", result.Summary, "From LLM")
	}
}

func TestExtractStructured_WithOptions(t *testing.T) {
	mockClient := NewMockClient(`{"status": "done", "summary": "Test"}`)

	var result testStruct
	err := ExtractStructured(
		context.Background(),
		mockClient,
		"No JSON here",
		`{"type": "object"}`,
		&result,
		&ExtractStructuredOptions{
			Model:       "opus",
			MaxTokens:   500,
			Temperature: 0.1,
			Context:     "This is test review findings",
		},
	)

	if err != nil {
		t.Errorf("ExtractStructured() error = %v", err)
	}
}

func TestBuildExtractionPrompt(t *testing.T) {
	prompt := buildExtractionPrompt("Some content", "Review findings context")

	if !contains(prompt, "Some content") {
		t.Error("Prompt should contain the input content")
	}
	if !contains(prompt, "Review findings context") {
		t.Error("Prompt should contain the context")
	}
	if !contains(prompt, "Extract") {
		t.Error("Prompt should contain extraction instruction")
	}
}

func TestBuildExtractionPrompt_LongInput(t *testing.T) {
	// Create a very long input
	longInput := make([]byte, 20000)
	for i := range longInput {
		longInput[i] = 'x'
	}

	prompt := buildExtractionPrompt(string(longInput), "")

	// Should be truncated
	if len(prompt) > 16000 {
		t.Errorf("Prompt should be truncated, got length %d", len(prompt))
	}
	if !contains(prompt, "[truncated]") {
		t.Error("Truncated prompt should contain truncation marker")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
