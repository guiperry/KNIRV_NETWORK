package mcphardening

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewSchemaValidator(t *testing.T) {
	sv := NewSchemaValidator()
	assert.NotNil(t, sv)
}

func TestRegisterAndGetSchema(t *testing.T) {
	sv := NewSchemaValidator()
	schema := &ToolSchema{
		Name:        "test-tool",
		Description: "A test tool",
		InputSchema: &JSONSchema{
			Type: "object",
			Properties: map[string]*SchemaProp{
				"name": {Type: "string", Description: "The name"},
			},
			Required: []string{"name"},
		},
	}

	sv.RegisterSchema(schema)
	got, ok := sv.GetSchema("test-tool")
	assert.True(t, ok)
	assert.Equal(t, "test-tool", got.Name)
	assert.Equal(t, "A test tool", got.Description)
}

func TestGetSchemaNotFound(t *testing.T) {
	sv := NewSchemaValidator()
	_, ok := sv.GetSchema("nonexistent")
	assert.False(t, ok)
}

func TestValidateArgs(t *testing.T) {
	sv := NewSchemaValidator()
	schema := &ToolSchema{
		Name: "greet",
		InputSchema: &JSONSchema{
			Type:     "object",
			Required: []string{"name", "greeting"},
		},
	}
	sv.RegisterSchema(schema)

	tests := []struct {
		name    string
		tool    string
		args    map[string]interface{}
		wantOK  bool
		wantMsg string
	}{
		{"all required", "greet", map[string]interface{}{"name": "Alice", "greeting": "hello"}, true, ""},
		{"missing required", "greet", map[string]interface{}{"name": "Alice"}, false, "missing required argument: greeting"},
		{"tool not registered", "unknown", map[string]interface{}{}, false, "no schema registered for tool: unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok, msg := sv.ValidateArgs(tt.tool, tt.args)
			assert.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.wantMsg, msg)
		})
	}
}

func TestValidateArgsNoInputSchema(t *testing.T) {
	sv := NewSchemaValidator()
	sv.RegisterSchema(&ToolSchema{Name: "no-input"})
	ok, msg := sv.ValidateArgs("no-input", nil)
	assert.True(t, ok)
	assert.Empty(t, msg)
}
