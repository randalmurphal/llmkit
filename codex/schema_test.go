package codex

import (
	"encoding/json"
	"testing"
)

func TestNormalizeSchemaMakesOptionalFieldsNullable(t *testing.T) {
	input := json.RawMessage(`{
		"type": "object",
		"properties": {
			"name": {"type": "string"},
			"count": {"type": "integer"}
		},
		"required": ["name"]
	}`)

	data, err := normalizeSchema(input)
	if err != nil {
		t.Fatalf("normalizeSchema: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal normalized schema: %v", err)
	}
	if got["additionalProperties"] != false {
		t.Fatalf("expected additionalProperties=false, got %#v", got["additionalProperties"])
	}

	required := got["required"].([]any)
	if len(required) != 2 || required[0] != "count" || required[1] != "name" {
		t.Fatalf("unexpected required list: %#v", required)
	}

	props := got["properties"].(map[string]any)
	count := props["count"].(map[string]any)
	types := count["type"].([]any)
	if len(types) != 2 || types[0] != "integer" || types[1] != "null" {
		t.Fatalf("optional field was not made nullable: %#v", count["type"])
	}
}

func TestExtractLastJSONValue(t *testing.T) {
	got := extractLastJSONValue("thinking...\n{\"ok\":true}")
	if got != "{\"ok\":true}" {
		t.Fatalf("extractLastJSONValue = %q", got)
	}
}
