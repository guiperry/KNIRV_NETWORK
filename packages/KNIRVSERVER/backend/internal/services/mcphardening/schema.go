package mcphardening

type SchemaValidator struct {
	schemas map[string]*ToolSchema
}

type ToolSchema struct {
	Name        string              `json:"name"`
	Description string              `json:"description"`
	InputSchema *JSONSchema         `json:"inputSchema"`
	OutputSchema *JSONSchema        `json:"outputSchema,omitempty"`
}

type JSONSchema struct {
	Type       string                `json:"type"`
	Properties map[string]*SchemaProp `json:"properties,omitempty"`
	Required   []string              `json:"required,omitempty"`
	Items      *JSONSchema           `json:"items,omitempty"`
}

type SchemaProp struct {
	Type        string              `json:"type"`
	Description string              `json:"description,omitempty"`
	Pattern     string              `json:"pattern,omitempty"`
	MinLength   int                 `json:"minLength,omitempty"`
	MaxLength   int                 `json:"maxLength,omitempty"`
	Minimum     float64             `json:"minimum,omitempty"`
	Maximum     float64             `json:"maximum,omitempty"`
	Enum        []string            `json:"enum,omitempty"`
	Properties  map[string]*SchemaProp `json:"properties,omitempty"`
	Items       *JSONSchema         `json:"items,omitempty"`
}

func NewSchemaValidator() *SchemaValidator {
	return &SchemaValidator{
		schemas: make(map[string]*ToolSchema),
	}
}

func (sv *SchemaValidator) RegisterSchema(schema *ToolSchema) {
	sv.schemas[schema.Name] = schema
}

func (sv *SchemaValidator) GetSchema(name string) (*ToolSchema, bool) {
	s, ok := sv.schemas[name]
	return s, ok
}

func (sv *SchemaValidator) ValidateArgs(toolName string, args map[string]interface{}) (bool, string) {
	schema, ok := sv.schemas[toolName]
	if !ok {
		return false, "no schema registered for tool: " + toolName
	}
	if schema.InputSchema == nil {
		return true, ""
	}
	for _, req := range schema.InputSchema.Required {
		if _, ok := args[req]; !ok {
			return false, "missing required argument: " + req
		}
	}
	return true, ""
}
