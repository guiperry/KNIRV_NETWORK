package main

import (
	"KNIRVCHAIN/internal/types"
	"encoding/json"
	"testing"
	"time"
)

// TestMCPStructSerialization tests the serialization and deserialization of MCP structs
func TestMCPStructSerialization(t *testing.T) {
	// Test BaseDescriptor
	baseDesc := types.BaseDescriptor{
		ID:             "test-id-123",
		Name:           "Test Capability",
		Owner:          "owner-address-123",
		Version:        "1.0.0",
		Description:    "Test capability for unit testing",
		GasFeeNRN:      100,
		Timestamp:      time.Now().Unix(),
		CustomMetadata: map[string]interface{}{"key1": "value1", "key2": 42},
	}

	// Serialize BaseDescriptor
	baseDescJSON, err := json.Marshal(baseDesc)
	if err != nil {
		t.Fatalf("Failed to serialize BaseDescriptor: %v", err)
	}

	// Deserialize BaseDescriptor
	var deserializedBaseDesc types.BaseDescriptor
	err = json.Unmarshal(baseDescJSON, &deserializedBaseDesc)
	if err != nil {
		t.Fatalf("Failed to deserialize BaseDescriptor: %v", err)
	}

	// Verify BaseDescriptor fields
	if deserializedBaseDesc.ID != baseDesc.ID {
		t.Errorf("BaseDescriptor ID mismatch: expected %s, got %s", baseDesc.ID, deserializedBaseDesc.ID)
	}
	if deserializedBaseDesc.Name != baseDesc.Name {
		t.Errorf("BaseDescriptor Name mismatch: expected %s, got %s", baseDesc.Name, deserializedBaseDesc.Name)
	}
	if deserializedBaseDesc.Owner != baseDesc.Owner {
		t.Errorf("BaseDescriptor Owner mismatch: expected %s, got %s", baseDesc.Owner, deserializedBaseDesc.Owner)
	}
	if deserializedBaseDesc.GasFeeNRN != baseDesc.GasFeeNRN {
		t.Errorf("BaseDescriptor GasFeeNRN mismatch: expected %d, got %d", baseDesc.GasFeeNRN, deserializedBaseDesc.GasFeeNRN)
	}

	// Test ResourceDescriptor
	baseDesc.CapabilityType = types.CapabilityTypeResource
	resourceDesc := types.ResourceDescriptor{
		BaseDescriptor: baseDesc,
		ResourceType:   types.ResourceTypeFile,
		ContentHash:    "sha256:1234567890abcdef",
		Schema: types.PluginSchemaDetail{
			Summary:       "This is resource 2 (API)",
			LocationHints: []string{"http://api.example.com/res2"},
			// ManifestFile and ExecutableFile might be empty for ResourceTypeAPI
		},
	}

	// Serialize ResourceDescriptor
	resourceDescJSON, err := json.Marshal(resourceDesc)
	if err != nil {
		t.Fatalf("Failed to serialize ResourceDescriptor: %v", err)
	}

	// Deserialize ResourceDescriptor
	var deserializedResourceDesc types.ResourceDescriptor
	err = json.Unmarshal(resourceDescJSON, &deserializedResourceDesc)
	if err != nil {
		t.Fatalf("Failed to deserialize ResourceDescriptor: %v", err)
	}

	// Verify ResourceDescriptor fields
	if deserializedResourceDesc.ID != resourceDesc.ID {
		t.Errorf("ResourceDescriptor ID mismatch: expected %s, got %s", resourceDesc.ID, deserializedResourceDesc.ID)
	}
	if deserializedResourceDesc.BaseDescriptor.CapabilityType != resourceDesc.BaseDescriptor.CapabilityType {
		t.Errorf("ResourceDescriptor CapabilityType mismatch: expected %s, got %s", resourceDesc.BaseDescriptor.CapabilityType, deserializedResourceDesc.BaseDescriptor.CapabilityType)
	}
	if deserializedResourceDesc.ResourceType != resourceDesc.ResourceType {
		t.Errorf("ResourceDescriptor ResourceType mismatch: expected %s, got %s", resourceDesc.ResourceType, deserializedResourceDesc.ResourceType)
	}
	if deserializedResourceDesc.ContentHash != resourceDesc.ContentHash {
		t.Errorf("ResourceDescriptor ContentHash mismatch: expected %s, got %s", resourceDesc.ContentHash, deserializedResourceDesc.ContentHash)
	}
	if len(deserializedResourceDesc.Schema.LocationHints) != len(resourceDesc.Schema.LocationHints) {
		t.Errorf("ResourceDescriptor LocationHints length mismatch: expected %d, got %d", len(resourceDesc.Schema.LocationHints), len(deserializedResourceDesc.Schema.LocationHints))
	}

	// Test ToolDescriptor
	baseDesc.CapabilityType = types.CapabilityTypeTool
	toolDesc := types.ToolDescriptor{
		BaseDescriptor:   baseDesc,
		InputSchemaJSON:  `{"type": "object", "properties": {"input": {"type": "string"}}}`,
		OutputSchemaJSON: `{"type": "object", "properties": {"output": {"type": "string"}}}`,
		ExecutionPointer: "resource:plugin-123",
	}

	// Serialize ToolDescriptor
	toolDescJSON, err := json.Marshal(toolDesc)
	if err != nil {
		t.Fatalf("Failed to serialize ToolDescriptor: %v", err)
	}

	// Deserialize ToolDescriptor
	var deserializedToolDesc types.ToolDescriptor
	err = json.Unmarshal(toolDescJSON, &deserializedToolDesc)
	if err != nil {
		t.Fatalf("Failed to deserialize ToolDescriptor: %v", err)
	}

	// Verify ToolDescriptor fields
	if deserializedToolDesc.ID != toolDesc.ID {
		t.Errorf("ToolDescriptor ID mismatch: expected %s, got %s", toolDesc.ID, deserializedToolDesc.ID)
	}
	if deserializedToolDesc.BaseDescriptor.CapabilityType != toolDesc.BaseDescriptor.CapabilityType {
		t.Errorf("ToolDescriptor CapabilityType mismatch: expected %s, got %s", toolDesc.BaseDescriptor.CapabilityType, deserializedToolDesc.BaseDescriptor.CapabilityType)
	}
	if deserializedToolDesc.InputSchemaJSON != toolDesc.InputSchemaJSON {
		t.Errorf("ToolDescriptor InputSchemaJSON mismatch: expected %s, got %s", toolDesc.InputSchemaJSON, deserializedToolDesc.InputSchemaJSON)
	}
	if deserializedToolDesc.OutputSchemaJSON != toolDesc.OutputSchemaJSON {
		t.Errorf("ToolDescriptor OutputSchemaJSON mismatch: expected %s, got %s", toolDesc.OutputSchemaJSON, deserializedToolDesc.OutputSchemaJSON)
	}
	if deserializedToolDesc.ExecutionPointer != toolDesc.ExecutionPointer {
		t.Errorf("ToolDescriptor ExecutionPointer mismatch: expected %s, got %s", toolDesc.ExecutionPointer, deserializedToolDesc.ExecutionPointer)
	}

	// Test PromptDescriptor
	baseDesc.CapabilityType = types.CapabilityTypePrompt
	promptDesc := types.PromptDescriptor{
		BaseDescriptor:       baseDesc,
		Template:             "Hello, {{name}}! How are you?",
		ParametersSchemaJSON: `{"type": "object", "properties": {"name": {"type": "string"}}}`,
	}

	// Serialize PromptDescriptor
	promptDescJSON, err := json.Marshal(promptDesc)
	if err != nil {
		t.Fatalf("Failed to serialize PromptDescriptor: %v", err)
	}

	// Deserialize PromptDescriptor
	var deserializedPromptDesc types.PromptDescriptor
	err = json.Unmarshal(promptDescJSON, &deserializedPromptDesc)
	if err != nil {
		t.Fatalf("Failed to deserialize PromptDescriptor: %v", err)
	}

	// Verify PromptDescriptor fields
	if deserializedPromptDesc.ID != promptDesc.ID {
		t.Errorf("PromptDescriptor ID mismatch: expected %s, got %s", promptDesc.ID, deserializedPromptDesc.ID)
	}
	if deserializedPromptDesc.BaseDescriptor.CapabilityType != promptDesc.BaseDescriptor.CapabilityType {
		t.Errorf("PromptDescriptor CapabilityType mismatch: expected %s, got %s", promptDesc.BaseDescriptor.CapabilityType, deserializedPromptDesc.BaseDescriptor.CapabilityType)
	}
	if deserializedPromptDesc.Template != promptDesc.Template {
		t.Errorf("PromptDescriptor Template mismatch: expected %s, got %s", promptDesc.Template, deserializedPromptDesc.Template)
	}
	if deserializedPromptDesc.ParametersSchemaJSON != promptDesc.ParametersSchemaJSON {
		t.Errorf("PromptDescriptor ParametersSchemaJSON mismatch: expected %s, got %s", promptDesc.ParametersSchemaJSON, deserializedPromptDesc.ParametersSchemaJSON)
	}

	// Test MemoryServiceDescriptor
	baseDesc.CapabilityType = types.CapabilityTypeMemoryService
	memoryDesc := types.MemoryServiceDescriptor{
		BaseDescriptor: baseDesc,
		GraphSchema:    `{"type": "object", "properties": {"nodes": {"type": "array"}}}`,
	}

	// Serialize MemoryServiceDescriptor
	memoryDescJSON, err := json.Marshal(memoryDesc)
	if err != nil {
		t.Fatalf("Failed to serialize MemoryServiceDescriptor: %v", err)
	}

	// Deserialize MemoryServiceDescriptor
	var deserializedMemoryDesc types.MemoryServiceDescriptor
	err = json.Unmarshal(memoryDescJSON, &deserializedMemoryDesc)
	if err != nil {
		t.Fatalf("Failed to deserialize MemoryServiceDescriptor: %v", err)
	}

	// Verify MemoryServiceDescriptor fields
	if deserializedMemoryDesc.ID != memoryDesc.ID {
		t.Errorf("MemoryServiceDescriptor ID mismatch: expected %s, got %s", memoryDesc.ID, deserializedMemoryDesc.ID)
	}
	if deserializedMemoryDesc.CapabilityType != memoryDesc.CapabilityType {
		t.Errorf("MemoryServiceDescriptor CapabilityType mismatch: expected %s, got %s", memoryDesc.CapabilityType, deserializedMemoryDesc.CapabilityType)
	}
	if deserializedMemoryDesc.GraphSchema != memoryDesc.GraphSchema {
		t.Errorf("MemoryServiceDescriptor GraphSchema mismatch: expected %s, got %s", memoryDesc.GraphSchema, deserializedMemoryDesc.GraphSchema)
	}

	// Test ContextRecord
	contextRecord := types.ContextRecord{
		ID:              "context-123",
		CapabilityID:    "capability-123",
		InteractionType: types.InteractionTypeToolInvocation,
		Initiator:       "initiator-address-123",
		Timestamp:       time.Now().Unix(),
		InputHash:       "sha256:input-hash",
		OutputHash:      "sha256:output-hash",
		Details:         map[string]interface{}{"param1": "value1", "param2": 42},
		Signature:       []byte("signature-bytes"),
	}

	// Serialize ContextRecord
	contextRecordJSON, err := json.Marshal(contextRecord)
	if err != nil {
		t.Fatalf("Failed to serialize ContextRecord: %v", err)
	}

	// Deserialize ContextRecord
	var deserializedContextRecord types.ContextRecord
	err = json.Unmarshal(contextRecordJSON, &deserializedContextRecord)
	if err != nil {
		t.Fatalf("Failed to deserialize ContextRecord: %v", err)
	}

	// Verify ContextRecord fields
	if deserializedContextRecord.ID != contextRecord.ID {
		t.Errorf("ContextRecord ID mismatch: expected %s, got %s", contextRecord.ID, deserializedContextRecord.ID)
	}
	if deserializedContextRecord.CapabilityID != contextRecord.CapabilityID {
		t.Errorf("ContextRecord CapabilityID mismatch: expected %s, got %s", contextRecord.CapabilityID, deserializedContextRecord.CapabilityID)
	}
	if deserializedContextRecord.InteractionType != contextRecord.InteractionType {
		t.Errorf("ContextRecord InteractionType mismatch: expected %s, got %s", contextRecord.InteractionType, deserializedContextRecord.InteractionType)
	}
	if deserializedContextRecord.Initiator != contextRecord.Initiator {
		t.Errorf("ContextRecord Initiator mismatch: expected %s, got %s", contextRecord.Initiator, deserializedContextRecord.Initiator)
	}
	if deserializedContextRecord.InputHash != contextRecord.InputHash {
		t.Errorf("ContextRecord InputHash mismatch: expected %s, got %s", contextRecord.InputHash, deserializedContextRecord.InputHash)
	}
	if deserializedContextRecord.OutputHash != contextRecord.OutputHash {
		t.Errorf("ContextRecord OutputHash mismatch: expected %s, got %s", contextRecord.OutputHash, deserializedContextRecord.OutputHash)
	}
}

// TestNewContextRecord tests the NewContextRecord function
func TestNewContextRecord(t *testing.T) {
	id := "context-123"
	capabilityID := "capability-123"
	interactionType := types.InteractionTypeToolInvocation
	initiator := "initiator-address-123"
	inputHash := "sha256:input-hash"
	outputHash := "sha256:output-hash"
	details := map[string]interface{}{"param1": "value1", "param2": 42}

	// Create a new ContextRecord
	contextRecord := types.NewContextRecord(id, capabilityID, interactionType, initiator, inputHash, outputHash, details)

	// Verify the fields
	if contextRecord.ID != id {
		t.Errorf("ContextRecord ID mismatch: expected %s, got %s", id, contextRecord.ID)
	}
	if contextRecord.CapabilityID != capabilityID {
		t.Errorf("ContextRecord CapabilityID mismatch: expected %s, got %s", capabilityID, contextRecord.CapabilityID)
	}
	if contextRecord.InteractionType != interactionType {
		t.Errorf("ContextRecord InteractionType mismatch: expected %s, got %s", interactionType, contextRecord.InteractionType)
	}
	if contextRecord.Initiator != initiator {
		t.Errorf("ContextRecord Initiator mismatch: expected %s, got %s", initiator, contextRecord.Initiator)
	}
	if contextRecord.InputHash != inputHash {
		t.Errorf("ContextRecord InputHash mismatch: expected %s, got %s", inputHash, contextRecord.InputHash)
	}
	if contextRecord.OutputHash != outputHash {
		t.Errorf("ContextRecord OutputHash mismatch: expected %s, got %s", outputHash, contextRecord.OutputHash)
	}
	if contextRecord.Timestamp == 0 {
		t.Errorf("ContextRecord Timestamp should not be 0")
	}
}
