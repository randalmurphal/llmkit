package codex

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"sort"
)

func writeSchemaFile(schema json.RawMessage) (string, error) {
	normalized, err := normalizeSchema(schema)
	if err != nil {
		return "", fmt.Errorf("normalize schema: %w", err)
	}

	file, err := os.CreateTemp("", "llmkit-codex-schema-*.json")
	if err != nil {
		return "", fmt.Errorf("create temp schema: %w", err)
	}
	defer file.Close()

	if _, err := file.Write(normalized); err != nil {
		_ = os.Remove(file.Name())
		return "", fmt.Errorf("write temp schema: %w", err)
	}
	if _, err := file.Write([]byte("\n")); err != nil {
		_ = os.Remove(file.Name())
		return "", fmt.Errorf("finalize temp schema: %w", err)
	}
	return file.Name(), nil
}

func normalizeSchema(schema json.RawMessage) ([]byte, error) {
	if len(schema) == 0 {
		return json.Marshal(map[string]any{"type": "object"})
	}

	var doc any
	if err := json.Unmarshal(schema, &doc); err != nil {
		return nil, err
	}

	normalizeSchemaNode(doc)
	return json.Marshal(doc)
}

func normalizeSchemaNode(node any) {
	switch typed := node.(type) {
	case map[string]any:
		for _, key := range []string{"definitions", "$defs"} {
			if raw, ok := typed[key].(map[string]any); ok {
				for _, child := range raw {
					normalizeSchemaNode(child)
				}
			}
		}
		for _, key := range []string{"items", "additionalProperties", "contains", "if", "then", "else", "not"} {
			if child, ok := typed[key]; ok {
				normalizeSchemaNode(child)
			}
		}
		for _, key := range []string{"allOf", "anyOf", "oneOf", "prefixItems"} {
			if raw, ok := typed[key].([]any); ok {
				for _, child := range raw {
					normalizeSchemaNode(child)
				}
			}
		}
		if isObjectSchema(typed) {
			if _, ok := typed["additionalProperties"]; !ok {
				typed["additionalProperties"] = false
			}
			if props, ok := typed["properties"].(map[string]any); ok {
				requiredSet := requiredNames(typed["required"])
				required := make([]string, 0, len(props))
				for name, child := range props {
					normalizeSchemaNode(child)
					if !requiredSet[name] {
						makePropertyNullable(child)
					}
					required = append(required, name)
				}
				sort.Strings(required)
				typed["required"] = required
			}
		}
	case []any:
		for _, child := range typed {
			normalizeSchemaNode(child)
		}
	}
}

func isObjectSchema(node map[string]any) bool {
	switch typ := node["type"].(type) {
	case string:
		return typ == "object"
	case []any:
		for _, value := range typ {
			if text, ok := value.(string); ok && text == "object" {
				return true
			}
		}
	}
	_, hasProps := node["properties"]
	return hasProps
}

func requiredNames(value any) map[string]bool {
	out := map[string]bool{}
	values, ok := value.([]any)
	if !ok {
		return out
	}
	for _, item := range values {
		if name, ok := item.(string); ok {
			out[name] = true
		}
	}
	return out
}

func makePropertyNullable(node any) {
	typed, ok := node.(map[string]any)
	if !ok {
		return
	}
	if typ, ok := typed["type"].(string); ok {
		if typ != "null" {
			typed["type"] = []any{typ, "null"}
		}
		return
	}
	if types, ok := typed["type"].([]any); ok {
		for _, value := range types {
			if text, ok := value.(string); ok && text == "null" {
				return
			}
		}
		typed["type"] = append(types, "null")
		return
	}
	if anyOf, ok := typed["anyOf"].([]any); ok {
		if branchHasNull(anyOf) {
			return
		}
		typed["anyOf"] = append(anyOf, map[string]any{"type": "null"})
		return
	}
	if oneOf, ok := typed["oneOf"].([]any); ok {
		if branchHasNull(oneOf) {
			return
		}
		typed["oneOf"] = append(oneOf, map[string]any{"type": "null"})
	}
}

func branchHasNull(branches []any) bool {
	for _, branch := range branches {
		m, ok := branch.(map[string]any)
		if !ok {
			continue
		}
		if typ, ok := m["type"].(string); ok && typ == "null" {
			return true
		}
	}
	return false
}

func extractLastJSONValue(content string) string {
	data := bytes.TrimSpace([]byte(content))
	if len(data) == 0 || json.Valid(data) {
		return string(data)
	}
	for start := len(data) - 1; start >= 0; start-- {
		switch data[start] {
		case '{', '[':
			if json.Valid(bytes.TrimSpace(data[start:])) {
				return string(bytes.TrimSpace(data[start:]))
			}
		}
	}
	return string(data)
}
