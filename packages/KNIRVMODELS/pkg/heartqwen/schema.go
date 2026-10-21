package heartqwen

import (
	"encoding/json"
	"fmt"
)

// checkSchema validates out against the small JSON-schema subset the stage
// schemas use: type (object, string, integer, number, boolean, array),
// required, properties, enum. It is deliberately strict about what it knows
// and silent about keywords it does not (the engine's grammar enforces the
// rest at generation time).
func checkSchema(schema, out json.RawMessage) error {
	var s struct {
		Type       string                     `json:"type"`
		Required   []string                   `json:"required"`
		Properties map[string]json.RawMessage `json:"properties"`
		Enum       []json.RawMessage          `json:"enum"`
	}
	if err := json.Unmarshal(schema, &s); err != nil {
		return nil
	}
	if len(s.Enum) > 0 {
		for _, e := range s.Enum {
			if string(e) == string(out) {
				return nil
			}
		}
		return fmt.Errorf("value %s is not one of the allowed values", out)
	}
	var v any
	if err := json.Unmarshal(out, &v); err != nil {
		return err
	}
	switch s.Type {
	case "object":
		m, ok := v.(map[string]any)
		if !ok {
			return fmt.Errorf("want object")
		}
		for _, r := range s.Required {
			if _, ok := m[r]; !ok {
				return fmt.Errorf("missing required field %q", r)
			}
		}
		for k, sub := range s.Properties {
			if raw, ok := m[k]; ok {
				b, _ := json.Marshal(raw)
				if err := checkSchema(sub, b); err != nil {
					return fmt.Errorf("%s: %w", k, err)
				}
			}
		}
	case "string":
		if _, ok := v.(string); !ok {
			return fmt.Errorf("want string")
		}
	case "boolean":
		if _, ok := v.(bool); !ok {
			return fmt.Errorf("want boolean")
		}
	case "number":
		if _, ok := v.(float64); !ok {
			return fmt.Errorf("want number")
		}
	case "integer":
		if f, ok := v.(float64); !ok || f != float64(int64(f)) {
			return fmt.Errorf("want integer")
		}
	case "array":
		if _, ok := v.([]any); !ok {
			return fmt.Errorf("want array")
		}
	}
	return nil
}
