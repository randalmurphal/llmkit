package llmkit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/invopop/jsonschema"
)

// TypedResponse is a strict parsed response paired with the raw provider response.
type TypedResponse[T any] struct {
	Value    T         `json:"value"`
	Response *Response `json:"response"`
}

// CompleteTyped calls Complete and strictly unmarshals the resulting content into T.
// The provider response must already be constrained to valid JSON for the target type.
func CompleteTyped[T any](ctx context.Context, client Client, req Request) (*TypedResponse[T], error) {
	if client == nil {
		return nil, fmt.Errorf("client is required")
	}
	if len(req.JSONSchema) == 0 {
		schema, err := schemaFor[T]()
		if err != nil {
			return nil, fmt.Errorf("generate schema: %w", err)
		}
		req.JSONSchema = schema
	}

	resp, err := client.Complete(ctx, req)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, fmt.Errorf("nil response")
	}
	if strings.TrimSpace(resp.Content) == "" {
		return nil, fmt.Errorf("empty structured response")
	}

	data, err := extractStructuredJSON(resp.Content)
	if err != nil {
		return nil, err
	}

	var value T
	if err := decodeStructuredJSON(data, &value); err != nil {
		return nil, fmt.Errorf("parse structured response: %w", err)
	}

	return &TypedResponse[T]{
		Value:    value,
		Response: resp,
	}, nil
}

func schemaFor[T any]() ([]byte, error) {
	typ := reflect.TypeOf((*T)(nil)).Elem()
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	if typ.Kind() == reflect.Map || typ.Kind() == reflect.Interface {
		return json.Marshal(map[string]any{"type": "object"})
	}
	if typ.Kind() != reflect.Struct {
		return json.Marshal(map[string]any{"type": schemaTypeForKind(typ.Kind())})
	}
	reflector := jsonschema.Reflector{
		DoNotReference: true,
		ExpandedStruct: true,
	}
	return json.Marshal(reflector.ReflectFromType(typ))
}

func schemaTypeForKind(kind reflect.Kind) string {
	switch kind {
	case reflect.Bool:
		return "boolean"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "integer"
	case reflect.Float32, reflect.Float64:
		return "number"
	case reflect.Slice, reflect.Array:
		return "array"
	case reflect.String:
		return "string"
	default:
		return "object"
	}
}

func decodeStructuredJSON[T any](data []byte, out *T) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(out); err != nil {
		return err
	}
	if err := dec.Decode(new(any)); err != io.EOF {
		if err == nil {
			return fmt.Errorf("multiple JSON values")
		}
		return err
	}
	return nil
}

func extractStructuredJSON(content string) ([]byte, error) {
	data := bytes.TrimSpace([]byte(content))
	if len(data) == 0 {
		return nil, fmt.Errorf("empty structured response")
	}
	if json.Valid(data) {
		return data, nil
	}

	for start := len(data) - 1; start >= 0; start-- {
		switch data[start] {
		case '{', '[':
			candidate := bytes.TrimSpace(data[start:])
			if json.Valid(candidate) {
				return candidate, nil
			}
		}
	}

	return nil, fmt.Errorf("response does not contain valid JSON")
}
