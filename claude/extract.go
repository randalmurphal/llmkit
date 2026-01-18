// Package claude provides the ExtractStructured helper for robust JSON extraction.
package claude

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// ExtractStructuredOptions configures the extraction behavior.
type ExtractStructuredOptions struct {
	// Model specifies which model to use for extraction fallback.
	// Defaults to "haiku" if empty.
	Model string

	// MaxTokens limits the extraction response length.
	// Defaults to 2000 if zero.
	MaxTokens int

	// Temperature for the extraction call. Defaults to 0.
	Temperature float64

	// Context provides additional context about what the data represents.
	// This helps the LLM understand the extraction task better.
	Context string
}

// ExtractStructured attempts to extract structured JSON data from LLM output.
//
// It follows a two-phase approach for robustness:
//  1. Try direct JSON parsing of the input
//  2. If parsing fails, use an LLM call with JSON schema to extract
//
// This is designed for session-based workflows where --json-schema cannot be
// used during generation (it only works with print mode). The session generates
// output guided by prompts, then this function extracts structured data.
//
// Parameters:
//   - client: Claude client for fallback extraction
//   - input: Raw text output from session or other source
//   - schema: JSON schema defining the expected structure
//   - target: Pointer to struct to unmarshal into (e.g., &ReviewFindings{})
//   - opts: Optional configuration (can be nil for defaults)
//
// Returns error if both direct parsing and extraction fail.
func ExtractStructured(
	ctx context.Context,
	client Client,
	input string,
	schema string,
	target any,
	opts *ExtractStructuredOptions,
) error {
	if opts == nil {
		opts = &ExtractStructuredOptions{}
	}

	// Apply defaults
	if opts.Model == "" {
		opts.Model = "haiku"
	}
	if opts.MaxTokens == 0 {
		opts.MaxTokens = 2000
	}

	// Phase 1: Try direct JSON parsing
	// Look for JSON in the input - it might be the whole thing or embedded
	jsonContent := findJSON(input)
	if jsonContent != "" {
		if err := json.Unmarshal([]byte(jsonContent), target); err == nil {
			return nil // Success - direct parsing worked
		}
	}

	// Phase 2: Fall back to LLM extraction with JSON schema
	if client == nil {
		return fmt.Errorf("direct JSON parsing failed and no client provided for fallback extraction")
	}

	prompt := buildExtractionPrompt(input, opts.Context)

	resp, err := client.Complete(ctx, CompletionRequest{
		Messages: []Message{
			{Role: RoleUser, Content: prompt},
		},
		Model:       opts.Model,
		MaxTokens:   opts.MaxTokens,
		Temperature: opts.Temperature,
		JSONSchema:  schema,
	})
	if err != nil {
		return fmt.Errorf("extraction API call failed: %w", err)
	}

	if resp == nil || resp.Content == "" {
		return fmt.Errorf("extraction returned empty response")
	}

	// Parse the extracted JSON
	if err := json.Unmarshal([]byte(resp.Content), target); err != nil {
		return fmt.Errorf("failed to parse extracted JSON: %w (content: %s)", err, truncate(resp.Content, 200))
	}

	return nil
}

// findJSON attempts to locate JSON content within text.
// It looks for JSON objects or arrays, handling common formats:
// - Pure JSON response
// - JSON in code blocks
// - JSON mixed with other text
func findJSON(input string) string {
	input = strings.TrimSpace(input)

	// Try the whole input as JSON first
	if (strings.HasPrefix(input, "{") && strings.HasSuffix(input, "}")) ||
		(strings.HasPrefix(input, "[") && strings.HasSuffix(input, "]")) {
		if json.Valid([]byte(input)) {
			return input
		}
	}

	// Look for JSON in code blocks (```json ... ```)
	if idx := strings.Index(input, "```json"); idx != -1 {
		start := idx + 7 // len("```json")
		// Skip any whitespace/newlines after ```json
		for start < len(input) && (input[start] == '\n' || input[start] == '\r' || input[start] == ' ') {
			start++
		}
		if endIdx := strings.Index(input[start:], "```"); endIdx != -1 {
			content := strings.TrimSpace(input[start : start+endIdx])
			if json.Valid([]byte(content)) {
				return content
			}
		}
	}

	// Look for JSON object anywhere in text (greedy match for outermost braces)
	if braceStart := strings.Index(input, "{"); braceStart != -1 {
		// Find matching closing brace
		depth := 0
		inString := false
		escape := false
		for i := braceStart; i < len(input); i++ {
			c := input[i]
			if escape {
				escape = false
				continue
			}
			if c == '\\' && inString {
				escape = true
				continue
			}
			if c == '"' {
				inString = !inString
				continue
			}
			if inString {
				continue
			}
			if c == '{' {
				depth++
			} else if c == '}' {
				depth--
				if depth == 0 {
					candidate := input[braceStart : i+1]
					if json.Valid([]byte(candidate)) {
						return candidate
					}
					break
				}
			}
		}
	}

	return ""
}

// buildExtractionPrompt creates the prompt for LLM-based extraction.
func buildExtractionPrompt(input string, context string) string {
	var sb strings.Builder

	sb.WriteString("Extract structured data from the following content.\n\n")

	if context != "" {
		sb.WriteString("Context: ")
		sb.WriteString(context)
		sb.WriteString("\n\n")
	}

	sb.WriteString("Content to extract from:\n")
	sb.WriteString("---\n")

	// Truncate very long inputs to avoid token limits
	content := input
	if len(content) > 15000 {
		content = content[:15000] + "\n...[truncated]"
	}
	sb.WriteString(content)

	sb.WriteString("\n---\n\n")
	sb.WriteString("Extract the relevant information and return it as JSON matching the required schema.")

	return sb.String()
}

// truncate shortens a string to maxLen characters, adding "..." if truncated.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
