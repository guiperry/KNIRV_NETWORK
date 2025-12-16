package main

import (
	"errors"
	"fmt"
	"log"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	// Import the generated protobuf types from the correct location
	pb "KNIRVCHAIN/proto"
	"KNIRVCHAIN/types"
)

// Error definitions
var ErrUnsupportedCapabilityType = errors.New("unsupported capability type")

// ============================================================================
// Go struct to Protobuf message conversion functions
// ============================================================================

// ConvertBaseDescriptorToProto converts a BaseDescriptor to a BaseDescriptorProto
func ConvertBaseDescriptorToProto(base types.BaseDescriptor) (*pb.BaseDescriptorProto, error) {
	// Convert timestamp from int64 to protobuf Timestamp
	ts := timestamppb.New(time.Unix(base.Timestamp, 0)) // Use APIv2 timestamppb

	// Convert capability type
	var capType pb.CapabilityTypeProto
	switch base.CapabilityType {
	case types.CapabilityTypeResource:
		capType = pb.CapabilityTypeProto_CAPABILITY_TYPE_PROTO_RESOURCE
	case types.CapabilityTypeTool:
		capType = pb.CapabilityTypeProto_CAPABILITY_TYPE_PROTO_TOOL
	case types.CapabilityTypePrompt:
		capType = pb.CapabilityTypeProto_CAPABILITY_TYPE_PROTO_PROMPT
	case types.CapabilityTypeMemoryService:
		capType = pb.CapabilityTypeProto_CAPABILITY_TYPE_PROTO_MEMORY_SERVICE
	default:
		capType = pb.CapabilityTypeProto_CAPABILITY_TYPE_PROTO_UNSPECIFIED
	}

	// Convert custom metadata to protobuf Struct
	var customMetadataProto *structpb.Struct // Use APIv2 structpb
	if base.CustomMetadata != nil {
		var err error
		customMetadataProto, err = structpb.NewStruct(base.CustomMetadata)
		if err != nil {
			// Log the error for more details
			log.Printf("Error converting customMetadata map to structpb.Struct: %v. Metadata: %+v", err, base.CustomMetadata)
			// Depending on strictness, you might return an error or proceed with nil metadata
			// For now, let's proceed with nil if conversion fails, but log it.
			customMetadataProto = nil
		}
	}

	return &pb.BaseDescriptorProto{
		Id:             base.ID,
		CapabilityType: capType,
		Name:           base.Name,
		Owner:          base.Owner,
		Version:        base.Version,
		Description:    base.Description,
		GasFeeNrn:      base.GasFeeNRN,
		Timestamp:      ts,
		CustomMetadata: customMetadataProto,
	}, nil
}

// ConvertPluginSchemaDetailToProto converts a PluginSchemaDetail to a PluginSchemaDetailProto
func ConvertPluginSchemaDetailToProto(schema types.PluginSchemaDetail) (*pb.PluginSchemaDetailProto, error) {
	// Convert access info to protobuf Struct
	var accessInfoProto *structpb.Struct // Use APIv2 structpb
	if schema.AccessInfo != nil {
		var err error
		accessInfoProto, err = structpb.NewStruct(schema.AccessInfo)
		if err != nil {
			log.Printf("Error converting accessInfo map to structpb.Struct: %v. AccessInfo: %+v", err, schema.AccessInfo)
			// Proceed with nil if conversion fails, log it.
			accessInfoProto = nil
		}
	}

	return &pb.PluginSchemaDetailProto{
		Summary:             schema.Summary,
		AccessInfo:          accessInfoProto,
		LocationHints:       schema.LocationHints,
		ManifestFile:        schema.ManifestFile,
		ExecutableFile:      schema.ExecutableFile,
		OutputDirectoryHint: schema.OutputDirectoryHint,
	}, nil
}

// ConvertResourceDescriptorToProto converts a ResourceDescriptor to a ResourceDescriptorProto
func ConvertResourceDescriptorToProto(resource types.ResourceDescriptor) (*pb.ResourceDescriptorProto, error) {
	baseProto, err := ConvertBaseDescriptorToProto(resource.BaseDescriptor)
	if err != nil {
		return nil, err
	}

	// Convert resource type
	var resourceType pb.DiscoveryResourceTypeProto
	switch resource.ResourceType {
	case types.DiscoveryResourceTypeFile:
		resourceType = pb.DiscoveryResourceTypeProto_DISCOVERY_RESOURCE_TYPE_PROTO_FILE
	case types.DiscoveryResourceTypeAPI:
		resourceType = pb.DiscoveryResourceTypeProto_DISCOVERY_RESOURCE_TYPE_PROTO_API
	case types.DiscoveryResourceTypePlugin:
		resourceType = pb.DiscoveryResourceTypeProto_DISCOVERY_RESOURCE_TYPE_PROTO_PLUGIN
	case types.DiscoveryResourceTypeGeneratedDoc:
		resourceType = pb.DiscoveryResourceTypeProto_DISCOVERY_RESOURCE_TYPE_PROTO_GENERATED_DOCUMENT
	case types.DiscoveryResourceTypeDataset:
		resourceType = pb.DiscoveryResourceTypeProto_DISCOVERY_RESOURCE_TYPE_PROTO_DATASET
	case types.DiscoveryResourceTypeModelArtifact:
		resourceType = pb.DiscoveryResourceTypeProto_DISCOVERY_RESOURCE_TYPE_PROTO_MODEL_ARTIFACT
	case types.DiscoveryResourceTypeService:
		resourceType = pb.DiscoveryResourceTypeProto_DISCOVERY_RESOURCE_TYPE_PROTO_SERVICE
	default:
		resourceType = pb.DiscoveryResourceTypeProto_DISCOVERY_RESOURCE_TYPE_PROTO_UNSPECIFIED
	}

	schemaProto, err := ConvertPluginSchemaDetailToProto(resource.Schema)
	if err != nil {
		return nil, err
	}

	return &pb.ResourceDescriptorProto{
		BaseDescriptor: baseProto,
		ResourceType:   resourceType,
		ContentHash:    resource.ContentHash,
		Schema:         schemaProto,
	}, nil
}

// ConvertToolDescriptorToProto converts a ToolDescriptor to a ToolDescriptorProto
func ConvertToolDescriptorToProto(tool types.ToolDescriptor) (*pb.ToolDescriptorProto, error) {
	baseProto, err := ConvertBaseDescriptorToProto(tool.BaseDescriptor)
	if err != nil {
		return nil, err
	}

	return &pb.ToolDescriptorProto{
		BaseDescriptor:   baseProto,
		InputSchemaJson:  tool.InputSchemaJSON,
		OutputSchemaJson: tool.OutputSchemaJSON,
		ExecutionPointer: tool.ExecutionPointer,
	}, nil
}

// ConvertPromptDescriptorToProto converts a PromptDescriptor to a PromptDescriptorProto
func ConvertPromptDescriptorToProto(prompt types.PromptDescriptor) (*pb.PromptDescriptorProto, error) {
	baseProto, err := ConvertBaseDescriptorToProto(prompt.BaseDescriptor)
	if err != nil {
		return nil, err
	}

	return &pb.PromptDescriptorProto{
		BaseDescriptor:       baseProto,
		Template:             prompt.Template,
		ParametersSchemaJson: prompt.ParametersSchemaJSON,
	}, nil
}

// ConvertMemoryServiceDescriptorToProto converts a MemoryServiceDescriptor to a MemoryServiceDescriptorProto
func ConvertMemoryServiceDescriptorToProto(memory types.MemoryServiceDescriptor) (*pb.MemoryServiceDescriptorProto, error) {
	baseProto, err := ConvertBaseDescriptorToProto(memory.BaseDescriptor)
	if err != nil {
		return nil, err
	}

	return &pb.MemoryServiceDescriptorProto{
		BaseDescriptor: baseProto,
		GraphSchema:    memory.GraphSchema,
	}, nil
}

// ConvertContextRecordToProto converts a ContextRecord to a ContextRecordProto
func ConvertContextRecordToProto(record types.ContextRecord) (*pb.ContextRecordProto, error) {
	// Convert timestamp from int64 to protobuf Timestamp
	ts := timestamppb.New(time.Unix(record.Timestamp, 0)) // Use APIv2 timestamppb

	// Convert interaction type
	var interactionType pb.InteractionTypeProto
	switch record.InteractionType {
	case types.InteractionTypeToolInvocation:
		interactionType = pb.InteractionTypeProto_INTERACTION_TYPE_PROTO_TOOL_INVOCATION
	case types.InteractionTypePromptUsage:
		interactionType = pb.InteractionTypeProto_INTERACTION_TYPE_PROTO_PROMPT_USAGE
	case types.InteractionTypeResourceAccess:
		interactionType = pb.InteractionTypeProto_INTERACTION_TYPE_PROTO_RESOURCE_ACCESS
	case types.InteractionTypePluginExecution:
		interactionType = pb.InteractionTypeProto_INTERACTION_TYPE_PROTO_PLUGIN_EXECUTION
	case types.InteractionTypeSamplingRequestSent:
		interactionType = pb.InteractionTypeProto_INTERACTION_TYPE_PROTO_SAMPLING_REQUEST_SENT
	case types.InteractionTypeSamplingResponseReceived:
		interactionType = pb.InteractionTypeProto_INTERACTION_TYPE_PROTO_SAMPLING_RESPONSE_RECEIVED
	case types.InteractionTypeMemoryWrite:
		interactionType = pb.InteractionTypeProto_INTERACTION_TYPE_PROTO_MEMORY_WRITE
	case types.InteractionTypeCapabilityRegistration:
		interactionType = pb.InteractionTypeProto_INTERACTION_TYPE_PROTO_CAPABILITY_REGISTRATION
	default:
		interactionType = pb.InteractionTypeProto_INTERACTION_TYPE_PROTO_UNSPECIFIED
	}

	// Convert details to protobuf Struct
	var detailsProto *structpb.Struct // Use APIv2 structpb
	if record.Details != nil {
		var err error
		detailsProto, err = structpb.NewStruct(record.Details)
		if err != nil {
			log.Printf("Error converting details map to structpb.Struct: %v. Details: %+v", err, record.Details)
			// Proceed with nil if conversion fails, log it.
			detailsProto = nil
		}
	}

	return &pb.ContextRecordProto{
		Id:              record.ID,
		CapabilityId:    record.CapabilityID,
		InteractionType: interactionType,
		Initiator:       record.Initiator,
		Timestamp:       ts,
		InputHash:       record.InputHash,
		OutputHash:      record.OutputHash,
		Details:         detailsProto,
		Signature:       record.Signature,
	}, nil
}

// ConvertToCapabilityDescriptorContainerProto converts a capability descriptor to a CapabilityDescriptorContainerProto
func ConvertToCapabilityDescriptorContainerProto(capability interface{}) (*pb.CapabilityDescriptorContainerProto, error) {
	container := &pb.CapabilityDescriptorContainerProto{}

	switch c := capability.(type) {
	case types.ResourceDescriptor:
		resourceProto, err := ConvertResourceDescriptorToProto(c)
		if err != nil {
			return nil, err
		}
		container.Descriptor_ = &pb.CapabilityDescriptorContainerProto_Resource{
			Resource: resourceProto,
		}
	case types.ToolDescriptor:
		toolProto, err := ConvertToolDescriptorToProto(c)
		if err != nil {
			return nil, err
		}
		container.Descriptor_ = &pb.CapabilityDescriptorContainerProto_Tool{
			Tool: toolProto,
		}
	case types.PromptDescriptor:
		promptProto, err := ConvertPromptDescriptorToProto(c)
		if err != nil {
			return nil, err
		}
		container.Descriptor_ = &pb.CapabilityDescriptorContainerProto_Prompt{
			Prompt: promptProto,
		}
	case types.MemoryServiceDescriptor:
		memoryProto, err := ConvertMemoryServiceDescriptorToProto(c)
		if err != nil {
			return nil, err
		}
		container.Descriptor_ = &pb.CapabilityDescriptorContainerProto_MemoryService{
			MemoryService: memoryProto,
		}
	default:
		return nil, ErrUnsupportedCapabilityType
	}

	return container, nil
}

// ============================================================================
// Protobuf message to Go struct conversion functions
// ============================================================================

// ConvertProtoToBaseDescriptor converts a BaseDescriptorProto to a BaseDescriptor
func ConvertProtoToBaseDescriptor(proto *pb.BaseDescriptorProto) (types.BaseDescriptor, error) {
	// Convert timestamp from protobuf Timestamp to int64
	var ts int64
	if proto.Timestamp != nil {
		ts = proto.Timestamp.AsTime().Unix() // Use AsTime().Unix() for APIv2
	}

	// Convert capability type
	var capType types.CapabilityType
	switch proto.CapabilityType {
	case pb.CapabilityTypeProto_CAPABILITY_TYPE_PROTO_RESOURCE:
		capType = types.CapabilityTypeResource
	case pb.CapabilityTypeProto_CAPABILITY_TYPE_PROTO_TOOL:
		capType = types.CapabilityTypeTool
	case pb.CapabilityTypeProto_CAPABILITY_TYPE_PROTO_PROMPT:
		capType = types.CapabilityTypePrompt
	case pb.CapabilityTypeProto_CAPABILITY_TYPE_PROTO_MEMORY_SERVICE:
		capType = types.CapabilityTypeMemoryService
	default:
		capType = ""
	}

	// Convert custom metadata from protobuf Struct to map[string]interface{}
	var customMetadata map[string]interface{}
	if proto.CustomMetadata != nil {
		customMetadata = proto.CustomMetadata.AsMap() // AsMap() works for APIv2 structpb.Struct
	}

	return types.BaseDescriptor{
		ID:             proto.Id,
		CapabilityType: capType,
		Name:           proto.Name,
		Owner:          proto.Owner,
		Version:        proto.Version,
		Description:    proto.Description,
		GasFeeNRN:      proto.GasFeeNrn,
		Timestamp:      ts,
		CustomMetadata: customMetadata,
	}, nil
}

// ConvertProtoToPluginSchemaDetail converts a PluginSchemaDetailProto to a PluginSchemaDetail
func ConvertProtoToPluginSchemaDetail(proto *pb.PluginSchemaDetailProto) (types.PluginSchemaDetail, error) {
	// Convert access info from protobuf Struct to map[string]interface{}
	var accessInfo map[string]interface{}
	if proto.AccessInfo != nil {
		accessInfo = proto.AccessInfo.AsMap() // AsMap() works for APIv2 structpb.Struct
	}

	return types.PluginSchemaDetail{
		Summary:             proto.Summary,
		AccessInfo:          accessInfo,
		LocationHints:       proto.LocationHints,
		ManifestFile:        proto.ManifestFile,
		ExecutableFile:      proto.ExecutableFile,
		OutputDirectoryHint: proto.OutputDirectoryHint,
	}, nil
}

// ConvertProtoToResourceDescriptor converts a ResourceDescriptorProto to a ResourceDescriptor
func ConvertProtoToResourceDescriptor(proto *pb.ResourceDescriptorProto) (types.ResourceDescriptor, error) {
	baseDesc, err := ConvertProtoToBaseDescriptor(proto.BaseDescriptor)
	if err != nil {
		return types.ResourceDescriptor{}, err
	}

	// Convert resource type
	var resourceType types.DiscoveryResourceType
	switch proto.ResourceType {
	case pb.DiscoveryResourceTypeProto_DISCOVERY_RESOURCE_TYPE_PROTO_FILE:
		resourceType = types.DiscoveryResourceTypeFile
	case pb.DiscoveryResourceTypeProto_DISCOVERY_RESOURCE_TYPE_PROTO_API:
		resourceType = types.DiscoveryResourceTypeAPI
	case pb.DiscoveryResourceTypeProto_DISCOVERY_RESOURCE_TYPE_PROTO_PLUGIN:
		resourceType = types.DiscoveryResourceTypePlugin
	case pb.DiscoveryResourceTypeProto_DISCOVERY_RESOURCE_TYPE_PROTO_GENERATED_DOCUMENT:
		resourceType = types.DiscoveryResourceTypeGeneratedDoc
	case pb.DiscoveryResourceTypeProto_DISCOVERY_RESOURCE_TYPE_PROTO_DATASET:
		resourceType = types.DiscoveryResourceTypeDataset
	case pb.DiscoveryResourceTypeProto_DISCOVERY_RESOURCE_TYPE_PROTO_MODEL_ARTIFACT:
		resourceType = types.DiscoveryResourceTypeModelArtifact
	case pb.DiscoveryResourceTypeProto_DISCOVERY_RESOURCE_TYPE_PROTO_SERVICE:
		resourceType = types.DiscoveryResourceTypeService
	default:
		resourceType = ""
	}

	schema, err := ConvertProtoToPluginSchemaDetail(proto.Schema)
	if err != nil {
		return types.ResourceDescriptor{}, err
	}

	return types.ResourceDescriptor{
		BaseDescriptor: baseDesc,
		ResourceType:   resourceType,
		ContentHash:    proto.ContentHash,
		Schema:         schema,
	}, nil
}

// ConvertProtoToToolDescriptor converts a ToolDescriptorProto to a ToolDescriptor
func ConvertProtoToToolDescriptor(proto *pb.ToolDescriptorProto) (types.ToolDescriptor, error) {
	baseDesc, err := ConvertProtoToBaseDescriptor(proto.BaseDescriptor)
	if err != nil {
		return types.ToolDescriptor{}, err
	}

	return types.ToolDescriptor{
		BaseDescriptor:   baseDesc,
		InputSchemaJSON:  proto.InputSchemaJson,
		OutputSchemaJSON: proto.OutputSchemaJson,
		ExecutionPointer: proto.ExecutionPointer,
	}, nil
}

// ConvertProtoToPromptDescriptor converts a PromptDescriptorProto to a PromptDescriptor
func ConvertProtoToPromptDescriptor(proto *pb.PromptDescriptorProto) (types.PromptDescriptor, error) {
	baseDesc, err := ConvertProtoToBaseDescriptor(proto.BaseDescriptor)
	if err != nil {
		return types.PromptDescriptor{}, err
	}

	return types.PromptDescriptor{
		BaseDescriptor:       baseDesc,
		Template:             proto.Template,
		ParametersSchemaJSON: proto.ParametersSchemaJson,
	}, nil
}

// ConvertProtoToMemoryServiceDescriptor converts a MemoryServiceDescriptorProto to a MemoryServiceDescriptor
func ConvertProtoToMemoryServiceDescriptor(proto *pb.MemoryServiceDescriptorProto) (types.MemoryServiceDescriptor, error) {
	baseDesc, err := ConvertProtoToBaseDescriptor(proto.BaseDescriptor)
	if err != nil {
		return types.MemoryServiceDescriptor{}, err
	}

	return types.MemoryServiceDescriptor{
		BaseDescriptor: baseDesc,
		GraphSchema:    proto.GraphSchema,
	}, nil
}

// ConvertProtoToContextRecord converts a ContextRecordProto to a ContextRecord
func ConvertProtoToContextRecord(proto *pb.ContextRecordProto) (types.ContextRecord, error) {
	// Convert timestamp from protobuf Timestamp to int64
	var ts int64
	if proto.Timestamp != nil {
		ts = proto.Timestamp.AsTime().Unix() // Use AsTime().Unix() for APIv2
	}

	// Convert interaction type
	var interactionType types.InteractionType
	switch proto.InteractionType {
	case pb.InteractionTypeProto_INTERACTION_TYPE_PROTO_TOOL_INVOCATION:
		interactionType = types.InteractionTypeToolInvocation
	case pb.InteractionTypeProto_INTERACTION_TYPE_PROTO_PROMPT_USAGE:
		interactionType = types.InteractionTypePromptUsage
	case pb.InteractionTypeProto_INTERACTION_TYPE_PROTO_RESOURCE_ACCESS:
		interactionType = types.InteractionTypeResourceAccess
	case pb.InteractionTypeProto_INTERACTION_TYPE_PROTO_PLUGIN_EXECUTION:
		interactionType = types.InteractionTypePluginExecution
	case pb.InteractionTypeProto_INTERACTION_TYPE_PROTO_SAMPLING_REQUEST_SENT:
		interactionType = types.InteractionTypeSamplingRequestSent
	case pb.InteractionTypeProto_INTERACTION_TYPE_PROTO_SAMPLING_RESPONSE_RECEIVED:
		interactionType = types.InteractionTypeSamplingResponseReceived
	case pb.InteractionTypeProto_INTERACTION_TYPE_PROTO_MEMORY_WRITE:
		interactionType = types.InteractionTypeMemoryWrite
	case pb.InteractionTypeProto_INTERACTION_TYPE_PROTO_CAPABILITY_REGISTRATION:
		interactionType = types.InteractionTypeCapabilityRegistration
	default:
		interactionType = ""
	}

	// Convert details from protobuf Struct to map[string]interface{}
	var details map[string]interface{}
	if proto.Details != nil {
		details = proto.Details.AsMap() // AsMap() works for APIv2 structpb.Struct
	}

	return types.ContextRecord{
		ID:              proto.Id,
		CapabilityID:    proto.CapabilityId,
		InteractionType: interactionType,
		Initiator:       proto.Initiator,
		Timestamp:       ts,
		InputHash:       proto.InputHash,
		OutputHash:      proto.OutputHash,
		Details:         details,
		Signature:       proto.Signature,
	}, nil
}

// ConvertProtoToCapability converts a CapabilityDescriptorContainerProto to a capability descriptor
func ConvertProtoToCapability(proto *pb.CapabilityDescriptorContainerProto) (interface{}, error) {
	switch x := proto.GetDescriptor_().(type) {
	case *pb.CapabilityDescriptorContainerProto_Resource:
		return ConvertProtoToResourceDescriptor(x.Resource)
	case *pb.CapabilityDescriptorContainerProto_Tool:
		return ConvertProtoToToolDescriptor(x.Tool)
	case *pb.CapabilityDescriptorContainerProto_Prompt:
		return ConvertProtoToPromptDescriptor(x.Prompt)
	case *pb.CapabilityDescriptorContainerProto_MemoryService:
		return ConvertProtoToMemoryServiceDescriptor(x.MemoryService)
	default:
		return nil, ErrUnsupportedCapabilityType
	}
}

// ============================================================================
// Helper functions for timestamp handling
// ============================================================================

// TimeToProtoTimestamp converts a Go time.Time to a protobuf Timestamp
func TimeToProtoTimestamp(t time.Time) *timestamppb.Timestamp { // Return APIv2 type
	return timestamppb.New(t)
}

// ProtoTimestampToTime converts a protobuf Timestamp to a Go time.Time
func ProtoTimestampToTime(ts *timestamppb.Timestamp) time.Time { // Accept APIv2 type
	if ts == nil {
		return time.Time{}
	}
	return ts.AsTime()
}

// UnixTimeToProtoTimestamp converts a Unix timestamp (seconds since epoch) to a protobuf Timestamp
func UnixTimeToProtoTimestamp(unixTime int64) *timestamppb.Timestamp { // Return APIv2 type
	return timestamppb.New(time.Unix(unixTime, 0))
}

// ProtoTimestampToUnixTime converts a protobuf Timestamp to a Unix timestamp (seconds since epoch)
func ProtoTimestampToUnixTime(ts *timestamppb.Timestamp) int64 { // Accept APIv2 type
	if ts == nil {
		return 0
	}
	return ts.Seconds
}

// ============================================================================
// Helper functions for canonical JSON representation for hashing
// ============================================================================

// GetCanonicalJSONForHashing returns a canonical JSON representation of a protobuf message for hashing
func GetCanonicalBytesForHashing(msg proto.Message) ([]byte, error) {
	// Use the proto.Marshal function to get deterministic binary output for hashing
	return proto.Marshal(msg)
}

// ToProto converts the Go Transaction struct to its Protobuf representation.
// This is now an alias for ToProtoForStorage to ensure consistent behavior.
func (t *Transaction) ToProto() (*pb.TransactionProto, error) {
	return t.ToProtoForStorage()
}

// ToProtoForHashing creates a TransactionProto specifically for hashing (used in signing/verification).
// It omits fields that are not part of the signed content.
func (t *Transaction) ToProtoForHashing() (*pb.TransactionProto, error) {
	// Use the old API timestamp to match the generated TransactionProto struct
	// t.Timestamp is int64 Unix seconds
	ts := timestamppb.New(time.Unix(t.Timestamp, 0)) // Use APIv2 timestamppb
	return &pb.TransactionProto{
		From:      t.From,
		To:        t.To,
		Value:     t.Value,
		Data:      t.Data,
		Timestamp: ts,
		Fee:       t.Fee,
		Type:      t.Type,
	}, nil
}

// ToProtoForStorage creates a TransactionProto that includes all fields needed for storage
// including Signature and PublicKey
func (t *Transaction) ToProtoForStorage() (*pb.TransactionProto, error) {
	ts := timestamppb.New(time.Unix(t.Timestamp, 0)) // Use APIv2 timestamppb
	return &pb.TransactionProto{
		From:      t.From,
		To:        t.To,
		Value:     t.Value,
		Data:      t.Data,
		Timestamp: ts,
		Fee:       t.Fee,
		Type:      t.Type,
		Signature: []byte(t.Signature),
		PublicKey: []byte(t.PublicKey),
	}, nil
}

// ToProto converts the Go Block struct to its Protobuf representation for hashing.
func (b *Block) ToProto() (*pb.BlockProto, error) {
	var finalProtoTransactions []*pb.TransactionProto // Default to nil

	if len(b.Transactions) > 0 {
		finalProtoTransactions = make([]*pb.TransactionProto, len(b.Transactions))
		for i, tx := range b.Transactions {
			ptx, err := tx.ToProto()
			if err != nil {
				return nil, fmt.Errorf("failed to convert transaction %d to proto: %w", i, err)
			}
			finalProtoTransactions[i] = ptx
		}
	}

	// SmartContract (b.Data) field removed from BlockProto

	ts := timestamppb.New(time.Unix(b.Timestamp, 0)) // Use APIv2 timestamppb

	return &pb.BlockProto{
		BlockNumber:  b.BlockNumber,
		PrevHash:     b.PrevHash,
		Timestamp:    ts,
		Nonce:        int32(b.Nonce),         // Ensure Nonce type matches proto (e.g., int32)
		Transactions: finalProtoTransactions, // Will be nil if b.Transactions is empty
		// SmartContract field removed
		ProposerAddress: b.ProposerAddress,
	}, nil
}
