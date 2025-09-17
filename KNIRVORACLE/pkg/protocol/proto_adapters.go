package protocol

import (
	"time"

	pb "KNIRVORACLE/pkg/protocol/proto"
	"KNIRVORACLE/pkg/types"

	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ProtoToContextRecord converts a ContextRecordProto to a ContextRecord
func ProtoToContextRecord(proto *pb.ContextRecordProto) *types.ContextRecord {
	if proto == nil {
		return nil
	}

	var details map[string]interface{}
	if proto.Details != nil {
		details = proto.Details.AsMap()
	}

	var timestamp int64
	if proto.Timestamp != nil {
		timestamp = proto.Timestamp.AsTime().Unix()
	}

	return &types.ContextRecord{
		ID:              proto.Id,
		CapabilityID:    proto.CapabilityId,
		InteractionType: types.InteractionType(protoInteractionTypeToString(proto.InteractionType)),
		Initiator:       proto.Initiator,
		Timestamp:       timestamp,
		InputHash:       proto.InputHash,
		OutputHash:      proto.OutputHash,
		Details:         details,
		Signature:       proto.Signature,
	}
}

// ContextRecordToProto converts a ContextRecord to a ContextRecordProto
func ContextRecordToProto(record *types.ContextRecord) (*pb.ContextRecordProto, error) {
	if record == nil {
		return nil, nil
	}

	var details *structpb.Struct
	var err error
	if record.Details != nil {
		details, err = structpb.NewStruct(record.Details)
		if err != nil {
			return nil, err
		}
	}

	timestamp := timestamppb.New(time.Unix(record.Timestamp, 0))

	return &pb.ContextRecordProto{
		Id:              record.ID,
		CapabilityId:    record.CapabilityID,
		InteractionType: stringToProtoInteractionType(string(record.InteractionType)),
		Initiator:       record.Initiator,
		Timestamp:       timestamp,
		InputHash:       record.InputHash,
		OutputHash:      record.OutputHash,
		Details:         details,
		Signature:       record.Signature,
	}, nil
}

// Helper function to convert proto enum to string
func protoInteractionTypeToString(interactionType pb.InteractionTypeProto) string {
	switch interactionType {
	case pb.InteractionTypeProto_INTERACTION_TYPE_PROTO_TOOL_INVOCATION:
		return string(types.InteractionTypeToolInvocation)
	case pb.InteractionTypeProto_INTERACTION_TYPE_PROTO_PROMPT_USAGE:
		return string(types.InteractionTypePromptUsage)
	case pb.InteractionTypeProto_INTERACTION_TYPE_PROTO_RESOURCE_ACCESS:
		return string(types.InteractionTypeResourceAccess)
	case pb.InteractionTypeProto_INTERACTION_TYPE_PROTO_PLUGIN_EXECUTION:
		return string(types.InteractionTypePluginExecution)
	case pb.InteractionTypeProto_INTERACTION_TYPE_PROTO_SAMPLING_REQUEST_SENT:
		return string(types.InteractionTypeSamplingRequestSent)
	case pb.InteractionTypeProto_INTERACTION_TYPE_PROTO_SAMPLING_RESPONSE_RECEIVED:
		return string(types.InteractionTypeSamplingResponseReceived)
	case pb.InteractionTypeProto_INTERACTION_TYPE_PROTO_MEMORY_WRITE:
		return string(types.InteractionTypeMemoryWrite)
	case pb.InteractionTypeProto_INTERACTION_TYPE_PROTO_CAPABILITY_REGISTRATION:
		return string(types.InteractionTypeCapabilityRegistration)
	default:
		return ""
	}
}

// Helper function to convert string to proto enum
func stringToProtoInteractionType(interactionType string) pb.InteractionTypeProto {
	switch interactionType {
	case string(types.InteractionTypeToolInvocation):
		return pb.InteractionTypeProto_INTERACTION_TYPE_PROTO_TOOL_INVOCATION
	case string(types.InteractionTypePromptUsage):
		return pb.InteractionTypeProto_INTERACTION_TYPE_PROTO_PROMPT_USAGE
	case string(types.InteractionTypeResourceAccess):
		return pb.InteractionTypeProto_INTERACTION_TYPE_PROTO_RESOURCE_ACCESS
	case string(types.InteractionTypePluginExecution):
		return pb.InteractionTypeProto_INTERACTION_TYPE_PROTO_PLUGIN_EXECUTION
	case string(types.InteractionTypeSamplingRequestSent):
		return pb.InteractionTypeProto_INTERACTION_TYPE_PROTO_SAMPLING_REQUEST_SENT
	case string(types.InteractionTypeSamplingResponseReceived):
		return pb.InteractionTypeProto_INTERACTION_TYPE_PROTO_SAMPLING_RESPONSE_RECEIVED
	case string(types.InteractionTypeMemoryWrite):
		return pb.InteractionTypeProto_INTERACTION_TYPE_PROTO_MEMORY_WRITE
	case string(types.InteractionTypeCapabilityRegistration):
		return pb.InteractionTypeProto_INTERACTION_TYPE_PROTO_CAPABILITY_REGISTRATION
	default:
		return pb.InteractionTypeProto_INTERACTION_TYPE_PROTO_UNSPECIFIED
	}
}

// GetCapabilityFromProtoInternal extracts a capability interface from a proto container
// This is the original implementation
func GetCapabilityFromProtoInternal(container *pb.CapabilityDescriptorContainerProto) (interface{}, error) {
	if container == nil {
		return nil, nil
	}

	switch {
	case container.GetResource() != nil:
		resource := container.GetResource()
		baseDesc := types.BaseDescriptor{
			ID:             resource.BaseDescriptor.Id,
			CapabilityType: types.CapabilityTypeResource,
			Name:           resource.BaseDescriptor.Name,
			Owner:          resource.BaseDescriptor.Owner,
			Version:        resource.BaseDescriptor.Version,
			Description:    resource.BaseDescriptor.Description,
			GasFeeNRN:      resource.BaseDescriptor.GasFeeNrn,
		}
		if resource.BaseDescriptor.Timestamp != nil {
			baseDesc.Timestamp = resource.BaseDescriptor.Timestamp.AsTime().Unix()
		}
		if resource.BaseDescriptor.CustomMetadata != nil {
			baseDesc.CustomMetadata = resource.BaseDescriptor.CustomMetadata.AsMap()
		}

		var schema types.PluginSchemaDetail
		if resource.Schema != nil {
			schema = types.PluginSchemaDetail{
				Summary:             resource.Schema.Summary,
				ManifestFile:        resource.Schema.ManifestFile,
				ExecutableFile:      resource.Schema.ExecutableFile,
				OutputDirectoryHint: resource.Schema.OutputDirectoryHint,
				LocationHints:       resource.Schema.LocationHints,
			}
			if resource.Schema.AccessInfo != nil {
				schema.AccessInfo = resource.Schema.AccessInfo.AsMap()
			}
		}

		return types.ResourceDescriptor{
			BaseDescriptor: baseDesc,
			ResourceType:   types.DiscoveryResourceType(protoResourceTypeToString(resource.ResourceType)),
			ContentHash:    resource.ContentHash,
			Schema:         schema,
		}, nil

	case container.GetTool() != nil:
		tool := container.GetTool()
		baseDesc := types.BaseDescriptor{
			ID:             tool.BaseDescriptor.Id,
			CapabilityType: types.CapabilityTypeTool,
			Name:           tool.BaseDescriptor.Name,
			Owner:          tool.BaseDescriptor.Owner,
			Version:        tool.BaseDescriptor.Version,
			Description:    tool.BaseDescriptor.Description,
			GasFeeNRN:      tool.BaseDescriptor.GasFeeNrn,
		}
		if tool.BaseDescriptor.Timestamp != nil {
			baseDesc.Timestamp = tool.BaseDescriptor.Timestamp.AsTime().Unix()
		}
		if tool.BaseDescriptor.CustomMetadata != nil {
			baseDesc.CustomMetadata = tool.BaseDescriptor.CustomMetadata.AsMap()
		}

		return types.ToolDescriptor{
			BaseDescriptor:   baseDesc,
			InputSchemaJSON:  tool.InputSchemaJson,
			OutputSchemaJSON: tool.OutputSchemaJson,
			ExecutionPointer: tool.ExecutionPointer,
		}, nil

	case container.GetPrompt() != nil:
		prompt := container.GetPrompt()
		baseDesc := types.BaseDescriptor{
			ID:             prompt.BaseDescriptor.Id,
			CapabilityType: types.CapabilityTypePrompt,
			Name:           prompt.BaseDescriptor.Name,
			Owner:          prompt.BaseDescriptor.Owner,
			Version:        prompt.BaseDescriptor.Version,
			Description:    prompt.BaseDescriptor.Description,
			GasFeeNRN:      prompt.BaseDescriptor.GasFeeNrn,
		}
		if prompt.BaseDescriptor.Timestamp != nil {
			baseDesc.Timestamp = prompt.BaseDescriptor.Timestamp.AsTime().Unix()
		}
		if prompt.BaseDescriptor.CustomMetadata != nil {
			baseDesc.CustomMetadata = prompt.BaseDescriptor.CustomMetadata.AsMap()
		}

		return types.PromptDescriptor{
			BaseDescriptor:       baseDesc,
			Template:             prompt.Template,
			ParametersSchemaJSON: prompt.ParametersSchemaJson,
		}, nil

	case container.GetMemoryService() != nil:
		memory := container.GetMemoryService()
		baseDesc := types.BaseDescriptor{
			ID:             memory.BaseDescriptor.Id,
			CapabilityType: types.CapabilityTypeMemoryService,
			Name:           memory.BaseDescriptor.Name,
			Owner:          memory.BaseDescriptor.Owner,
			Version:        memory.BaseDescriptor.Version,
			Description:    memory.BaseDescriptor.Description,
			GasFeeNRN:      memory.BaseDescriptor.GasFeeNrn,
		}
		if memory.BaseDescriptor.Timestamp != nil {
			baseDesc.Timestamp = memory.BaseDescriptor.Timestamp.AsTime().Unix()
		}
		if memory.BaseDescriptor.CustomMetadata != nil {
			baseDesc.CustomMetadata = memory.BaseDescriptor.CustomMetadata.AsMap()
		}

		return types.MemoryServiceDescriptor{
			BaseDescriptor: baseDesc,
			GraphSchema:    memory.GraphSchema,
		}, nil
	}

	return nil, nil
}

// Helper function to convert proto enum to string
func protoResourceTypeToString(resourceType pb.DiscoveryResourceTypeProto) string {
	switch resourceType {
	case pb.DiscoveryResourceTypeProto_DISCOVERY_RESOURCE_TYPE_PROTO_FILE:
		return string(types.DiscoveryResourceTypeFile)
	case pb.DiscoveryResourceTypeProto_DISCOVERY_RESOURCE_TYPE_PROTO_API:
		return string(types.DiscoveryResourceTypeAPI)
	case pb.DiscoveryResourceTypeProto_DISCOVERY_RESOURCE_TYPE_PROTO_PLUGIN:
		return string(types.DiscoveryResourceTypePlugin)
	case pb.DiscoveryResourceTypeProto_DISCOVERY_RESOURCE_TYPE_PROTO_GENERATED_DOCUMENT:
		return string(types.DiscoveryResourceTypeGeneratedDoc)
	case pb.DiscoveryResourceTypeProto_DISCOVERY_RESOURCE_TYPE_PROTO_DATASET:
		return string(types.DiscoveryResourceTypeDataset)
	case pb.DiscoveryResourceTypeProto_DISCOVERY_RESOURCE_TYPE_PROTO_MODEL_ARTIFACT:
		return string(types.DiscoveryResourceTypeModelArtifact)
	case pb.DiscoveryResourceTypeProto_DISCOVERY_RESOURCE_TYPE_PROTO_SERVICE:
		return string(types.DiscoveryResourceTypeService)
	default:
		return ""
	}
}

// ConvertProtoCapabilitiesToInterfaces converts a slice of proto capabilities to a slice of interfaces
func ConvertProtoCapabilitiesToInterfaces(protoCapabilities []*pb.CapabilityDescriptorContainerProto) ([]interface{}, error) {
	result := make([]interface{}, 0, len(protoCapabilities))

	for _, protoCapability := range protoCapabilities {
		capability, err := GetCapabilityFromProto(protoCapability)
		if err != nil {
			return nil, err
		}
		if capability != nil {
			result = append(result, capability)
		}
	}

	return result, nil
}

// ConvertProtoContextRecordsToContextRecords converts a slice of proto context records to a slice of ContextRecord
func ConvertProtoContextRecordsToContextRecords(protoRecords []*pb.ContextRecordProto) []*types.ContextRecord {
	result := make([]*types.ContextRecord, 0, len(protoRecords))

	for _, protoRecord := range protoRecords {
		record := ProtoToContextRecord(protoRecord)
		if record != nil {
			result = append(result, record)
		}
	}

	return result
}

// ResourceDescriptorToProto converts a ResourceDescriptor to a ResourceDescriptorProto
func ResourceDescriptorToProto(resource types.ResourceDescriptor) (*pb.ResourceDescriptorProto, error) {
	baseProto, err := BaseDescriptorToProto(resource.BaseDescriptor)
	if err != nil {
		return nil, err
	}

	var schemaProto *pb.PluginSchemaDetailProto
	if resource.Schema.Summary != "" || len(resource.Schema.LocationHints) > 0 {
		schemaProto = &pb.PluginSchemaDetailProto{
			Summary:             resource.Schema.Summary,
			ManifestFile:        resource.Schema.ManifestFile,
			ExecutableFile:      resource.Schema.ExecutableFile,
			OutputDirectoryHint: resource.Schema.OutputDirectoryHint,
			LocationHints:       resource.Schema.LocationHints,
		}

		if resource.Schema.AccessInfo != nil {
			accessInfo, err := structpb.NewStruct(resource.Schema.AccessInfo)
			if err != nil {
				return nil, err
			}
			schemaProto.AccessInfo = accessInfo
		}
	}

	return &pb.ResourceDescriptorProto{
		BaseDescriptor: baseProto,
		ResourceType:   stringToProtoResourceType(string(resource.ResourceType)),
		ContentHash:    resource.ContentHash,
		Schema:         schemaProto,
	}, nil
}

// ToolDescriptorToProto converts a ToolDescriptor to a ToolDescriptorProto
func ToolDescriptorToProto(tool types.ToolDescriptor) (*pb.ToolDescriptorProto, error) {
	baseProto, err := BaseDescriptorToProto(tool.BaseDescriptor)
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

// PromptDescriptorToProto converts a PromptDescriptor to a PromptDescriptorProto
func PromptDescriptorToProto(prompt types.PromptDescriptor) (*pb.PromptDescriptorProto, error) {
	baseProto, err := BaseDescriptorToProto(prompt.BaseDescriptor)
	if err != nil {
		return nil, err
	}

	return &pb.PromptDescriptorProto{
		BaseDescriptor:       baseProto,
		Template:             prompt.Template,
		ParametersSchemaJson: prompt.ParametersSchemaJSON,
	}, nil
}

// MemoryServiceDescriptorToProto converts a MemoryServiceDescriptor to a MemoryServiceDescriptorProto
func MemoryServiceDescriptorToProto(memory types.MemoryServiceDescriptor) (*pb.MemoryServiceDescriptorProto, error) {
	baseProto, err := BaseDescriptorToProto(memory.BaseDescriptor)
	if err != nil {
		return nil, err
	}

	return &pb.MemoryServiceDescriptorProto{
		BaseDescriptor: baseProto,
		GraphSchema:    memory.GraphSchema,
	}, nil
}

// BaseDescriptorToProto converts a BaseDescriptor to a BaseDescriptorProto
func BaseDescriptorToProto(base types.BaseDescriptor) (*pb.BaseDescriptorProto, error) {
	var customMetadata *structpb.Struct
	var err error
	if base.CustomMetadata != nil {
		customMetadata, err = structpb.NewStruct(base.CustomMetadata)
		if err != nil {
			return nil, err
		}
	}

	timestamp := timestamppb.New(time.Unix(base.Timestamp, 0))

	return &pb.BaseDescriptorProto{
		Id:             base.ID,
		CapabilityType: stringToProtoCapabilityType(string(base.CapabilityType)),
		Name:           base.Name,
		Owner:          base.Owner,
		Version:        base.Version,
		Description:    base.Description,
		GasFeeNrn:      base.GasFeeNRN,
		Timestamp:      timestamp,
		CustomMetadata: customMetadata,
	}, nil
}

// CapabilityToProto converts a capability interface to a CapabilityDescriptorContainerProto
func CapabilityToProto(capability interface{}) (*pb.CapabilityDescriptorContainerProto, error) {
	if capability == nil {
		return nil, nil
	}

	container := &pb.CapabilityDescriptorContainerProto{}

	switch c := capability.(type) {
	case types.ResourceDescriptor:
		resourceProto, err := ResourceDescriptorToProto(c)
		if err != nil {
			return nil, err
		}
		// Create a new container with the resource descriptor
		container = &pb.CapabilityDescriptorContainerProto{
			Descriptor_: &pb.CapabilityDescriptorContainerProto_Resource{
				Resource: resourceProto,
			},
		}
	case types.ToolDescriptor:
		toolProto, err := ToolDescriptorToProto(c)
		if err != nil {
			return nil, err
		}
		// Create a new container with the tool descriptor
		container = &pb.CapabilityDescriptorContainerProto{
			Descriptor_: &pb.CapabilityDescriptorContainerProto_Tool{
				Tool: toolProto,
			},
		}
	case types.PromptDescriptor:
		promptProto, err := PromptDescriptorToProto(c)
		if err != nil {
			return nil, err
		}
		// Create a new container with the prompt descriptor
		container = &pb.CapabilityDescriptorContainerProto{
			Descriptor_: &pb.CapabilityDescriptorContainerProto_Prompt{
				Prompt: promptProto,
			},
		}
	case types.MemoryServiceDescriptor:
		memoryProto, err := MemoryServiceDescriptorToProto(c)
		if err != nil {
			return nil, err
		}
		// Create a new container with the memory service descriptor
		container = &pb.CapabilityDescriptorContainerProto{
			Descriptor_: &pb.CapabilityDescriptorContainerProto_MemoryService{
				MemoryService: memoryProto,
			},
		}
	default:
		return nil, nil
	}

	return container, nil
}

// ConvertInterfacesToProtoCapabilities converts a slice of capability interfaces to a slice of proto capabilities
func ConvertInterfacesToProtoCapabilities(capabilities []interface{}) ([]*pb.CapabilityDescriptorContainerProto, error) {
	result := make([]*pb.CapabilityDescriptorContainerProto, 0, len(capabilities))

	for _, capability := range capabilities {
		protoCapability, err := CapabilityToProto(capability)
		if err != nil {
			return nil, err
		}
		if protoCapability != nil {
			result = append(result, protoCapability)
		}
	}

	return result, nil
}

// ConvertContextRecordsToProtoContextRecords converts a slice of ContextRecord to a slice of proto context records
func ConvertContextRecordsToProtoContextRecords(records []*types.ContextRecord) ([]*pb.ContextRecordProto, error) {
	result := make([]*pb.ContextRecordProto, 0, len(records))

	for _, record := range records {
		protoRecord, err := ContextRecordToProto(record)
		if err != nil {
			return nil, err
		}
		if protoRecord != nil {
			result = append(result, protoRecord)
		}
	}

	return result, nil
}

// Helper function to convert string to proto enum
func stringToProtoCapabilityType(capabilityType string) pb.CapabilityTypeProto {
	switch capabilityType {
	case string(types.CapabilityTypeResource):
		return pb.CapabilityTypeProto_CAPABILITY_TYPE_PROTO_RESOURCE
	case string(types.CapabilityTypeTool):
		return pb.CapabilityTypeProto_CAPABILITY_TYPE_PROTO_TOOL
	case string(types.CapabilityTypePrompt):
		return pb.CapabilityTypeProto_CAPABILITY_TYPE_PROTO_PROMPT
	case string(types.CapabilityTypeMemoryService):
		return pb.CapabilityTypeProto_CAPABILITY_TYPE_PROTO_MEMORY_SERVICE
	default:
		return pb.CapabilityTypeProto_CAPABILITY_TYPE_PROTO_UNSPECIFIED
	}
}

// Helper function to convert string to proto enum
func stringToProtoResourceType(resourceType string) pb.DiscoveryResourceTypeProto {
	switch resourceType {
	case string(types.DiscoveryResourceTypeFile):
		return pb.DiscoveryResourceTypeProto_DISCOVERY_RESOURCE_TYPE_PROTO_FILE
	case string(types.DiscoveryResourceTypeAPI):
		return pb.DiscoveryResourceTypeProto_DISCOVERY_RESOURCE_TYPE_PROTO_API
	case string(types.DiscoveryResourceTypePlugin):
		return pb.DiscoveryResourceTypeProto_DISCOVERY_RESOURCE_TYPE_PROTO_PLUGIN
	case string(types.DiscoveryResourceTypeGeneratedDoc):
		return pb.DiscoveryResourceTypeProto_DISCOVERY_RESOURCE_TYPE_PROTO_GENERATED_DOCUMENT
	case string(types.DiscoveryResourceTypeDataset):
		return pb.DiscoveryResourceTypeProto_DISCOVERY_RESOURCE_TYPE_PROTO_DATASET
	case string(types.DiscoveryResourceTypeModelArtifact):
		return pb.DiscoveryResourceTypeProto_DISCOVERY_RESOURCE_TYPE_PROTO_MODEL_ARTIFACT
	case string(types.DiscoveryResourceTypeService):
		return pb.DiscoveryResourceTypeProto_DISCOVERY_RESOURCE_TYPE_PROTO_SERVICE
	default:
		return pb.DiscoveryResourceTypeProto_DISCOVERY_RESOURCE_TYPE_PROTO_UNSPECIFIED
	}
}

// GetCapabilityFromProto is the public API for converting proto to capability
// It uses the internal implementation
func GetCapabilityFromProto(protoCapability *pb.CapabilityDescriptorContainerProto) (interface{}, error) {
	// Use the internal implementation
	return GetCapabilityFromProtoInternal(protoCapability)
}
