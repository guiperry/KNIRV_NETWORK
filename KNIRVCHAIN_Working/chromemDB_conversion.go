package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	pb "KNIRVCHAIN/proto"
	"KNIRVCHAIN/types"
)

// PrepareCapabilityDescriptorForChromemFromRegisterLocal prepares a capability descriptor for ChromemDB storage
// from register data (local version to avoid name conflict with sync_manager.go)
func PrepareCapabilityDescriptorForChromemFromRegisterLocal(
	registerData *pb.MCPRegisterCapabilityDataProto,
	txHash string,
	blockNumber uint64,
	timestamp int64,
) (string, string, map[string]interface{}, error) {
	if registerData == nil || registerData.GetCapabilityDescriptor() == nil {
		return "", "", nil, fmt.Errorf("PrepareCapabilityDescriptorForChromemFromRegisterLocal: registerData or its CapabilityDescriptor is nil")
	}

	// Properly serialize the descriptor to JSON
	desc := registerData.GetCapabilityDescriptor()
	descJSON, err := json.Marshal(desc)
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to marshal capability descriptor to JSON: %w", err)
	}
	doc := string(descJSON)

	var baseDesc *pb.BaseDescriptorProto
	var specificResourceType string // For resource type, if applicable

	switch d := desc.Descriptor_.(type) {
	case *pb.CapabilityDescriptorContainerProto_Resource:
		if d.Resource == nil {
			return "", "", nil, fmt.Errorf("resource descriptor in container is nil")
		}
		baseDesc = d.Resource.GetBaseDescriptor()
		specificResourceType = d.Resource.GetResourceType().String()
	case *pb.CapabilityDescriptorContainerProto_Tool:
		if d.Tool == nil {
			return "", "", nil, fmt.Errorf("tool descriptor in container is nil")
		}
		baseDesc = d.Tool.GetBaseDescriptor()
	case *pb.CapabilityDescriptorContainerProto_Prompt:
		if d.Prompt == nil {
			return "", "", nil, fmt.Errorf("prompt descriptor in container is nil")
		}
		baseDesc = d.Prompt.GetBaseDescriptor()
	case *pb.CapabilityDescriptorContainerProto_MemoryService:
		if d.MemoryService == nil {
			return "", "", nil, fmt.Errorf("memory service descriptor in container is nil")
		}
		baseDesc = d.MemoryService.GetBaseDescriptor()
	default:
		return "", "", nil, fmt.Errorf("unknown descriptor type in CapabilityDescriptorContainerProto: %T", d)
	}

	if baseDesc == nil {
		return "", "", nil, fmt.Errorf("baseDescriptor is nil after type switch")
	}

	meta := map[string]interface{}{
		"tx_hash":        txHash,
		"block_number":   blockNumber,
		"timestamp":      timestamp,
		"capability_id":  baseDesc.GetId(),
		"name":           baseDesc.GetName(), // Common fields
		"owner":          baseDesc.GetOwner(),
		"version":        baseDesc.GetVersion(),
		"description":    baseDesc.GetDescription(),
		"capabilityType": baseDesc.GetCapabilityType().String(), // This is the general CapabilityType
		"gasFeeNRN":      baseDesc.GetGasFeeNrn(),
	}

	// Add specific fields if it's a resource
	if resDesc := desc.GetResource(); resDesc != nil {
		meta["resourceType"] = specificResourceType // This is DiscoveryResourceType
		meta["contentHash"] = resDesc.GetContentHash()
	}

	// Use the capability ID as the document ID for direct lookup
	id := baseDesc.GetId()
	return id, doc, meta, nil
}

// PrepareCapabilityDescriptorForChromemFromUpdate prepares a capability descriptor for ChromemDB storage
// from update data
func PrepareCapabilityDescriptorForChromemFromUpdate(
	cdProto *pb.MCPUpdateCapabilityDataProto,
	txHash string,
	blockNumber uint64,
	timestamp int64,
) (string, string, map[string]interface{}, error) {
	if cdProto == nil || cdProto.GetCapabilityDescriptor() == nil {
		return "", "", nil, fmt.Errorf("PrepareCapabilityDescriptorForChromemFromUpdate: cdProto or its CapabilityDescriptor is nil")
	}

	// Properly serialize the descriptor to JSON
	desc := cdProto.GetCapabilityDescriptor()
	descJSON, err := json.Marshal(desc)
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to marshal capability descriptor to JSON: %w", err)
	}
	doc := string(descJSON)

	var baseDesc *pb.BaseDescriptorProto
	var specificResourceType string

	switch d := desc.Descriptor_.(type) {
	case *pb.CapabilityDescriptorContainerProto_Resource:
		if d.Resource == nil {
			return "", "", nil, fmt.Errorf("resource descriptor in container is nil for update")
		}
		baseDesc = d.Resource.GetBaseDescriptor()
		specificResourceType = d.Resource.GetResourceType().String()
	case *pb.CapabilityDescriptorContainerProto_Tool:
		if d.Tool == nil {
			return "", "", nil, fmt.Errorf("tool descriptor in container is nil for update")
		}
		baseDesc = d.Tool.GetBaseDescriptor()
	// Add other cases (Prompt, MemoryService) if updates for them are supported and indexed similarly
	default:
		return "", "", nil, fmt.Errorf("unknown or unsupported descriptor type for update in CapabilityDescriptorContainerProto: %T", d)
	}

	if baseDesc == nil {
		return "", "", nil, fmt.Errorf("baseDescriptor is nil after type switch for update")
	}

	meta := map[string]interface{}{
		"tx_hash":        txHash,
		"block_number":   blockNumber,
		"timestamp":      timestamp,
		"capability_id":  baseDesc.GetId(),
		"name":           baseDesc.GetName(), // Common fields
		"owner":          baseDesc.GetOwner(),
		"version":        baseDesc.GetVersion(),
		"description":    baseDesc.GetDescription(),
		"capabilityType": baseDesc.GetCapabilityType().String(), // General CapabilityType
		"gasFeeNRN":      baseDesc.GetGasFeeNrn(),
	}
	if resDesc := desc.GetResource(); resDesc != nil {
		meta["resourceType"] = specificResourceType // DiscoveryResourceType
		meta["contentHash"] = resDesc.GetContentHash()
	}

	// Use the capability ID as the document ID for direct lookup
	id := baseDesc.GetId()
	return id, doc, meta, nil
}

// PrepareContextRecordForChromemEnhanced converts a ContextRecord to ChromemDB format with enhanced metadata
// This is a local version to avoid name conflict with sync_manager.go
func PrepareContextRecordForChromemEnhanced(cr *types.ContextRecord, txHash string, blockHeight int64, blockTimestamp int64) (
	id string,
	documentForEmbedding string,
	metadata map[string]interface{},
) {
	// Check for nil context record
	if cr == nil {
		log.Printf("[ERROR] PrepareContextRecordForChromemEnhanced: received nil ContextRecord")
		return "", "", nil
	}

	id = cr.ID

	// Document for semantic search (natural language representation)
	documentForEmbedding = fmt.Sprintf("Context record for capability %s. Status: %s. Invoked by %s, provided by %s. Error: %s. Initiated at %s, completed at %s.",
		cr.CapabilityID,
		string(cr.Status),
		cr.InvokerNRN,
		cr.ProviderNRN,
		cr.Error,
		time.Unix(cr.TimestampInitiated, 0).Format(time.RFC3339),
		time.Unix(cr.TimestampCompleted, 0).Format(time.RFC3339))

	// Create full JSON representation for storage/retrieval
	documentObj := map[string]interface{}{
		"record_type":    "context_record",
		"schema_version": "1.0",
		"capability_id":  cr.CapabilityID,
		"status":         string(cr.Status),
		"invoker":        cr.InvokerNRN,
		"provider":       cr.ProviderNRN,
		"error":          cr.Error,
		"input_hash":     cr.InputHash,
		"output_hash":    cr.OutputHash,
	}

	// Safely add details if they exist
	if cr.Details != nil {
		// Create a safe copy of the details map to avoid modifying the original
		safeDetails := make(map[string]interface{})
		for k, v := range cr.Details {
			// Handle different value types safely
			switch val := v.(type) {
			case string:
				safeDetails[k] = val
			case float64, int, int64, uint64, bool:
				// Convert numeric types to string for consistent handling
				safeDetails[k] = fmt.Sprintf("%v", val)
			case nil:
				// Skip nil values or set to empty string
				safeDetails[k] = ""
			default:
				// For complex types, use JSON marshaling to get a string representation
				jsonBytes, err := json.Marshal(val)
				if err != nil {
					log.Printf("[WARNING] Failed to marshal detail value for key %s: %v", k, err)
					safeDetails[k] = fmt.Sprintf("%T", val) // Use type name as fallback
				} else {
					safeDetails[k] = string(jsonBytes)
				}
			}
		}
		documentObj["details"] = safeDetails
	}

	fullRecordJSONBytes, err := json.MarshalIndent(documentObj, "", "  ")
	if err != nil {
		log.Printf("[ERROR] PrepareContextRecordForChromemEnhanced: Failed to marshal document object: %v", err)
		return "", "", nil
	}

	// Metadata follows a consistent structure for filtering
	metadata = map[string]interface{}{
		"ID":                 cr.ID,
		"CapabilityID":       cr.CapabilityID,
		"InvokerNRN":         cr.InvokerNRN,
		"ProviderNRN":        cr.ProviderNRN,
		"Status":             string(cr.Status),
		"InputHash":          cr.InputHash,
		"OutputHash":         cr.OutputHash,
		"Error":              cr.Error,
		"TimestampInitiated": cr.TimestampInitiated,
		"TimestampCompleted": cr.TimestampCompleted,
		"GasFeeNRN":          cr.GasFeeNRN,
		"TransactionHash":    txHash,
		"BlockHeight":        blockHeight,
		"BlockTimestamp":     blockTimestamp,
		"SchemaVersion":      "1.0",
		"RecordType":         "context_record",
		"FullRecordJSON":     string(fullRecordJSONBytes),
	}
	return
}

// PrepareCapabilityDescriptorForChromem converts a CapabilityDescriptor to ChromemDB format
func PrepareCapabilityDescriptorForChromem(data interface{}, txHash string, blockHeight int64, blockTimestamp int64) (
	id string,
	documentForEmbedding string,
	metadata map[string]interface{},
) { // Document is a string
	var capabilityID, name, owner, version, description, capabilityType string
	var gasFeeNRN uint64
	var fullDescriptor interface{}

	// Extract fields based on data type
	switch d := data.(type) {
	case *types.MCPRegisterCapabilityData:
		capabilityID = d.CapabilityID
		name = d.Name
		owner = d.Owner
		version = d.Version
		description = d.Description
		capabilityType = string(d.CapabilityType)
		gasFeeNRN = d.GasFeeNRN
		fullDescriptor = d.Descriptor
	case *types.MCPUpdateCapabilityData:
		capabilityID = d.CapabilityID
		name = d.Name
		owner = d.Owner
		version = d.Version
		description = d.Description
		capabilityType = string(d.CapabilityType)
		gasFeeNRN = d.GasFeeNRN
		fullDescriptor = d.Descriptor
	default:
		// Handle unknown type
		return "", "", nil
	}

	id = capabilityID

	// Document for semantic search (natural language representation)
	documentForEmbedding = fmt.Sprintf("Capability '%s': %s. Type: %s. Owner: %s. Version: %s. Description: %s.",
		capabilityID,
		name,
		capabilityType,
		owner,
		version,
		description) // Use the raw description here

	// Create full JSON representation for storage/retrieval
	fullDescriptorJSONBytes, err := json.MarshalIndent(fullDescriptor, "", "  ")
	if err != nil {
		return "", "", nil
	}

	// Metadata follows a consistent structure for filtering
	metadata = map[string]interface{}{
		"ID":                 capabilityID,
		"Name":               name,
		"Owner":              owner,
		"Version":            version,
		"Description":        description,
		"CapabilityType":     capabilityType,
		"GasFeeNRN":          gasFeeNRN,
		"RegisteredAt":       blockTimestamp,
		"TransactionHash":    txHash,
		"BlockHeight":        blockHeight,
		"BlockTimestamp":     blockTimestamp,
		"IsLatest":           true, // New registrations or updates are always the latest
		"SchemaVersion":      "1.0",
		"RecordType":         "capability_descriptor",
		"FullDescriptorJSON": string(fullDescriptorJSONBytes),
	}
	return
}

// createDataDescriptionFormatted creates a human-readable description of transaction data
// This is a local version to avoid name conflict with sync_manager.go
func createDataDescriptionFormatted(dataObj interface{}, txType string) string {
	switch txType {
	case "MCP_REGISTER_CAPABILITY":
		if data, ok := dataObj.(map[string]interface{}); ok {
			capabilityID, _ := data["capability_id"].(string)
			name, _ := data["name"].(string)
			capabilityType, _ := data["capability_type"].(string)
			owner, _ := data["owner"].(string)
			version, _ := data["version"].(string)
			description, _ := data["description"].(string)

			// Create a more detailed formatted description
			formattedDesc := fmt.Sprintf("Registration of capability '%s' with ID '%s' of type '%s'", name, capabilityID, capabilityType)

			// Add additional details if available
			if owner != "" {
				formattedDesc += fmt.Sprintf("\nOwner: %s", owner)
			}
			if version != "" {
				formattedDesc += fmt.Sprintf("\nVersion: %s", version)
			}
			if description != "" {
				// Truncate description if too long
				if len(description) > 100 {
					formattedDesc += fmt.Sprintf("\nDescription: %s...", description[:100])
				} else {
					formattedDesc += fmt.Sprintf("\nDescription: %s", description)
				}
			}

			return formattedDesc
		}
	case "MCP_INVOKE_CAPABILITY":
		if data, ok := dataObj.(map[string]interface{}); ok {
			capabilityID, _ := data["capability_id"].(string)
			invoker, _ := data["invoker"].(string)
			provider, _ := data["provider"].(string)
			inputData, _ := data["input_data"].(string)

			// Create a more detailed formatted description
			formattedDesc := fmt.Sprintf("Invocation of capability with ID '%s'", capabilityID)

			// Add additional details if available
			if invoker != "" {
				formattedDesc += fmt.Sprintf("\nInvoker: %s", invoker)
			}
			if provider != "" {
				formattedDesc += fmt.Sprintf("\nProvider: %s", provider)
			}
			if inputData != "" {
				// Truncate input data if too long
				if len(inputData) > 50 {
					formattedDesc += fmt.Sprintf("\nInput data: %s...", inputData[:50])
				} else {
					formattedDesc += fmt.Sprintf("\nInput data: %s", inputData)
				}
			}

			return formattedDesc
		}
	case "MCP_UPDATE_CAPABILITY":
		if data, ok := dataObj.(map[string]interface{}); ok {
			capabilityID, _ := data["capability_id"].(string)
			name, _ := data["name"].(string)
			version, _ := data["version"].(string)
			owner, _ := data["owner"].(string)
			description, _ := data["description"].(string)

			// Create a more detailed formatted description
			formattedDesc := fmt.Sprintf("Update of capability with ID '%s'", capabilityID)

			// Add additional details if available
			if name != "" {
				formattedDesc += fmt.Sprintf("\nName: %s", name)
			}
			if version != "" {
				formattedDesc += fmt.Sprintf("\nNew version: %s", version)
			}
			if owner != "" {
				formattedDesc += fmt.Sprintf("\nOwner: %s", owner)
			}
			if description != "" {
				// Truncate description if too long
				if len(description) > 100 {
					formattedDesc += fmt.Sprintf("\nDescription: %s...", description[:100])
				} else {
					formattedDesc += fmt.Sprintf("\nDescription: %s", description)
				}
			}

			return formattedDesc
		}
	case "STANDARD_TRANSFER":
		if data, ok := dataObj.(map[string]interface{}); ok {
			from, _ := data["from"].(string)
			to, _ := data["to"].(string)
			value, _ := data["value"].(float64)

			formattedDesc := "Standard transfer of funds"

			// Add additional details if available
			if from != "" && to != "" {
				formattedDesc += fmt.Sprintf("\nFrom: %s\nTo: %s", from, to)
			}
			if value > 0 {
				formattedDesc += fmt.Sprintf("\nValue: %.2f", value)
			}

			return formattedDesc
		}
		return "Standard transfer of funds"
	}

	// Default description
	dataJSON, _ := json.Marshal(dataObj)
	if len(dataJSON) > 100 {
		return fmt.Sprintf("Transaction data: %s...", string(dataJSON[:100]))
	}
	return fmt.Sprintf("Transaction data: %s", string(dataJSON))
}
