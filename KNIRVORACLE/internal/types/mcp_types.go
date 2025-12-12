package types

// CapabilityType defines the type of MCP primitive.
type CapabilityType string

const (
	CapabilityTypeUnspecified   CapabilityType = "UNSPECIFIED"
	CapabilityTypeResource      CapabilityType = "RESOURCE"
	CapabilityTypeTool          CapabilityType = "TOOL"
	CapabilityTypePrompt        CapabilityType = "PROMPT"
	CapabilityTypeMemoryService CapabilityType = "MEMORY_SERVICE"
)

// IsValidCapabilityType checks if the provided capability type is valid
func IsValidCapabilityType(capType CapabilityType) bool {
	switch capType {
	case CapabilityTypeUnspecified, CapabilityTypeResource, CapabilityTypeTool, CapabilityTypePrompt, CapabilityTypeMemoryService:
		return true
	default:
		return false
	}
}
