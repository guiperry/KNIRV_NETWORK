// File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

package transaction

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction/internal/apijson"
	"github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction/internal/requestconfig"
	"github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction/option"
	"github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction/packages/param"
	"github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction/packages/respjson"
)

// McpCapabilityService contains methods and other services that help with
// interacting with the knirvchain-transaction-sdk API.
//
// Note, unlike clients, this service does not read variables from the environment
// automatically. You should not instantiate this service directly, and instead use
// the [NewMcpCapabilityService] method instead.
type McpCapabilityService struct {
	Options []option.RequestOption
}

// NewMcpCapabilityService generates a new service that applies the given options
// to each request. These options are applied after the parent client's options (if
// there is one), and before any request-specific options.
func NewMcpCapabilityService(opts ...option.RequestOption) (r McpCapabilityService) {
	r = McpCapabilityService{}
	r.Options = opts
	return
}

// Retrieves a specific capability by ID
func (r *McpCapabilityService) Get(ctx context.Context, capabilityID string, opts ...option.RequestOption) (res *CapabilityDescriptorUnion, err error) {
	opts = append(r.Options[:], opts...)
	if capabilityID == "" {
		err = errors.New("missing required capability_id parameter")
		return
	}
	path := fmt.Sprintf("mcp/capability/%s", capabilityID)
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodGet, path, nil, &res, opts...)
	return
}

// Submits a pre-signed transaction to update an existing capability. The sender
// must be the owner of the capability. The `Transaction.data` within the signed
// transaction should contain `MCPUpdateCapabilityData`.
func (r *McpCapabilityService) Update(ctx context.Context, body McpCapabilityUpdateParams, opts ...option.RequestOption) (res *McpCapabilityUpdateResponse, err error) {
	opts = append(r.Options[:], opts...)
	path := "mcp/capability/update"
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodPost, path, body, &res, opts...)
	return
}

// Submits a pre-signed transaction to invoke an existing capability. The
// `Transaction.data` within the signed transaction should contain
// `MCPInvokeCapabilityData`.
func (r *McpCapabilityService) Invoke(ctx context.Context, body McpCapabilityInvokeParams, opts ...option.RequestOption) (res *McpCapabilityInvokeResponse, err error) {
	opts = append(r.Options[:], opts...)
	path := "mcp/capability/invoke"
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodPost, path, body, &res, opts...)
	return
}

// Allows a client to get the necessary data (e.g., a hash or structured unsigned
// transaction) that needs to be signed to register a new MCP capability. The
// client signs this data locally and then submits the complete, signed transaction
// via the general /transaction endpoint.
func (r *McpCapabilityService) PrepareRegistration(ctx context.Context, body McpCapabilityPrepareRegistrationParams, opts ...option.RequestOption) (res *McpCapabilityPrepareRegistrationResponse, err error) {
	opts = append(r.Options[:], opts...)
	path := "mcp/capability/prepare_registration"
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodPost, path, body, &res, opts...)
	return
}

// Lists all invocations of a specific capability
func (r *McpCapabilityService) GetInvocations(ctx context.Context, capabilityID string, opts ...option.RequestOption) (res *[]ContextRecord, err error) {
	opts = append(r.Options[:], opts...)
	if capabilityID == "" {
		err = errors.New("missing required capability_id parameter")
		return
	}
	path := fmt.Sprintf("mcp/capability/%s/invocations", capabilityID)
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodGet, path, nil, &res, opts...)
	return
}

type BaseDescriptor struct {
	// Unique identifier
	ID string `json:"id"`
	// Type of capability
	//
	// Any of "RESOURCE", "TOOL", "PROMPT", "MEMORY_SERVICE".
	CapabilityType BaseDescriptorCapabilityType `json:"capabilityType"`
	// Custom metadata
	CustomMetadata map[string]any `json:"customMetadata"`
	// Description of the capability
	Description string `json:"description"`
	// NRN token gas fee for invoking/using this capability
	GasFeeNrn int64 `json:"gasFeeNRN"`
	// Human-readable name
	Name string `json:"name"`
	// Public key or address of the owner/root
	Owner string `json:"owner"`
	// Creation/registration timestamp
	Timestamp int64 `json:"timestamp"`
	// Version string
	Version string `json:"version"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		ID             respjson.Field
		CapabilityType respjson.Field
		CustomMetadata respjson.Field
		Description    respjson.Field
		GasFeeNrn      respjson.Field
		Name           respjson.Field
		Owner          respjson.Field
		Timestamp      respjson.Field
		Version        respjson.Field
		ExtraFields    map[string]respjson.Field
		raw            string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r BaseDescriptor) RawJSON() string { return r.JSON.raw }
func (r *BaseDescriptor) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// ToParam converts this BaseDescriptor to a BaseDescriptorParam.
//
// Warning: the fields of the param type will not be present. ToParam should only
// be used at the last possible moment before sending a request. Test for this with
// BaseDescriptorParam.Overrides()
func (r BaseDescriptor) ToParam() BaseDescriptorParam {
	return param.Override[BaseDescriptorParam](json.RawMessage(r.RawJSON()))
}

// Type of capability
type BaseDescriptorCapabilityType string

const (
	BaseDescriptorCapabilityTypeResource      BaseDescriptorCapabilityType = "RESOURCE"
	BaseDescriptorCapabilityTypeTool          BaseDescriptorCapabilityType = "TOOL"
	BaseDescriptorCapabilityTypePrompt        BaseDescriptorCapabilityType = "PROMPT"
	BaseDescriptorCapabilityTypeMemoryService BaseDescriptorCapabilityType = "MEMORY_SERVICE"
)

type BaseDescriptorParam struct {
	// Unique identifier
	ID param.Opt[string] `json:"id,omitzero"`
	// Description of the capability
	Description param.Opt[string] `json:"description,omitzero"`
	// NRN token gas fee for invoking/using this capability
	GasFeeNrn param.Opt[int64] `json:"gasFeeNRN,omitzero"`
	// Human-readable name
	Name param.Opt[string] `json:"name,omitzero"`
	// Public key or address of the owner/root
	Owner param.Opt[string] `json:"owner,omitzero"`
	// Creation/registration timestamp
	Timestamp param.Opt[int64] `json:"timestamp,omitzero"`
	// Version string
	Version param.Opt[string] `json:"version,omitzero"`
	// Type of capability
	//
	// Any of "RESOURCE", "TOOL", "PROMPT", "MEMORY_SERVICE".
	CapabilityType BaseDescriptorCapabilityType `json:"capabilityType,omitzero"`
	// Custom metadata
	CustomMetadata map[string]any `json:"customMetadata,omitzero"`
	paramObj
}

func (r BaseDescriptorParam) MarshalJSON() (data []byte, err error) {
	type shadow BaseDescriptorParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BaseDescriptorParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// CapabilityDescriptorUnion contains all possible properties and values from
// [CapabilityDescriptorResourceDescriptor], [CapabilityDescriptorToolDescriptor],
// [CapabilityDescriptorPromptDescriptor],
// [CapabilityDescriptorMemoryServiceDescriptor].
//
// Use the [CapabilityDescriptorUnion.AsAny] method to switch on the variant.
//
// Use the methods beginning with 'As' to cast the union to one of its variants.
type CapabilityDescriptorUnion struct {
	// This field is from variant [CapabilityDescriptorResourceDescriptor],
	// [CapabilityDescriptorToolDescriptor], [CapabilityDescriptorPromptDescriptor],
	// [CapabilityDescriptorMemoryServiceDescriptor].
	ID string `json:"id"`
	// Any of nil, nil, nil, nil.
	CapabilityType BaseDescriptorCapabilityType `json:"capabilityType"`
	// This field is from variant [CapabilityDescriptorResourceDescriptor],
	// [CapabilityDescriptorToolDescriptor], [CapabilityDescriptorPromptDescriptor],
	// [CapabilityDescriptorMemoryServiceDescriptor].
	CustomMetadata map[string]any `json:"customMetadata"`
	// This field is from variant [CapabilityDescriptorResourceDescriptor],
	// [CapabilityDescriptorToolDescriptor], [CapabilityDescriptorPromptDescriptor],
	// [CapabilityDescriptorMemoryServiceDescriptor].
	Description string `json:"description"`
	// This field is from variant [CapabilityDescriptorResourceDescriptor],
	// [CapabilityDescriptorToolDescriptor], [CapabilityDescriptorPromptDescriptor],
	// [CapabilityDescriptorMemoryServiceDescriptor].
	GasFeeNrn int64 `json:"gasFeeNRN"`
	// This field is from variant [CapabilityDescriptorResourceDescriptor],
	// [CapabilityDescriptorToolDescriptor], [CapabilityDescriptorPromptDescriptor],
	// [CapabilityDescriptorMemoryServiceDescriptor].
	Name string `json:"name"`
	// This field is from variant [CapabilityDescriptorResourceDescriptor],
	// [CapabilityDescriptorToolDescriptor], [CapabilityDescriptorPromptDescriptor],
	// [CapabilityDescriptorMemoryServiceDescriptor].
	Owner string `json:"owner"`
	// This field is from variant [CapabilityDescriptorResourceDescriptor],
	// [CapabilityDescriptorToolDescriptor], [CapabilityDescriptorPromptDescriptor],
	// [CapabilityDescriptorMemoryServiceDescriptor].
	Timestamp int64 `json:"timestamp"`
	// This field is from variant [CapabilityDescriptorResourceDescriptor],
	// [CapabilityDescriptorToolDescriptor], [CapabilityDescriptorPromptDescriptor],
	// [CapabilityDescriptorMemoryServiceDescriptor].
	Version string `json:"version"`
	// This field is from variant [CapabilityDescriptorResourceDescriptor].
	ContentHash string `json:"contentHash"`
	// This field is from variant [CapabilityDescriptorResourceDescriptor].
	ResourceType string `json:"resourceType"`
	// This field is from variant [CapabilityDescriptorResourceDescriptor].
	Schema CapabilityDescriptorResourceDescriptorSchema `json:"schema"`
	// This field is from variant [CapabilityDescriptorToolDescriptor].
	ExecutionPointer string `json:"executionPointer"`
	// This field is from variant [CapabilityDescriptorToolDescriptor].
	InputSchemaJson string `json:"inputSchemaJSON"`
	// This field is from variant [CapabilityDescriptorToolDescriptor].
	OutputSchemaJson string `json:"outputSchemaJSON"`
	// This field is from variant [CapabilityDescriptorPromptDescriptor].
	ParametersSchemaJson string `json:"parametersSchemaJSON"`
	// This field is from variant [CapabilityDescriptorPromptDescriptor].
	Template string `json:"template"`
	// This field is from variant [CapabilityDescriptorMemoryServiceDescriptor].
	GraphSchema string `json:"graphSchema"`
	JSON        struct {
		ID                   respjson.Field
		CapabilityType       respjson.Field
		CustomMetadata       respjson.Field
		Description          respjson.Field
		GasFeeNrn            respjson.Field
		Name                 respjson.Field
		Owner                respjson.Field
		Timestamp            respjson.Field
		Version              respjson.Field
		ContentHash          respjson.Field
		ResourceType         respjson.Field
		Schema               respjson.Field
		ExecutionPointer     respjson.Field
		InputSchemaJson      respjson.Field
		OutputSchemaJson     respjson.Field
		ParametersSchemaJson respjson.Field
		Template             respjson.Field
		GraphSchema          respjson.Field
		raw                  string
	} `json:"-"`
}

func (u CapabilityDescriptorUnion) AsCapabilityDescriptorResourceDescriptor() (v CapabilityDescriptorResourceDescriptor) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u CapabilityDescriptorUnion) AsCapabilityDescriptorToolDescriptor() (v CapabilityDescriptorToolDescriptor) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u CapabilityDescriptorUnion) AsCapabilityDescriptorPromptDescriptor() (v CapabilityDescriptorPromptDescriptor) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

func (u CapabilityDescriptorUnion) AsCapabilityDescriptorMemoryServiceDescriptor() (v CapabilityDescriptorMemoryServiceDescriptor) {
	apijson.UnmarshalRoot(json.RawMessage(u.JSON.raw), &v)
	return
}

// Returns the unmodified JSON received from the API
func (u CapabilityDescriptorUnion) RawJSON() string { return u.JSON.raw }

func (r *CapabilityDescriptorUnion) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// ToParam converts this CapabilityDescriptorUnion to a
// CapabilityDescriptorUnionParam.
//
// Warning: the fields of the param type will not be present. ToParam should only
// be used at the last possible moment before sending a request. Test for this with
// CapabilityDescriptorUnionParam.Overrides()
func (r CapabilityDescriptorUnion) ToParam() CapabilityDescriptorUnionParam {
	return param.Override[CapabilityDescriptorUnionParam](json.RawMessage(r.RawJSON()))
}

type CapabilityDescriptorResourceDescriptor struct {
	// SHA256 hash of the Capability package
	ContentHash string `json:"contentHash"`
	// Specific kind of resource
	//
	// Any of "FILE", "API", "PLUGIN", "GENERATED_DOCUMENT", "DATASET",
	// "MODEL_ARTIFACT", "SERVICE".
	ResourceType string                                       `json:"resourceType"`
	Schema       CapabilityDescriptorResourceDescriptorSchema `json:"schema"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		ContentHash  respjson.Field
		ResourceType respjson.Field
		Schema       respjson.Field
		ExtraFields  map[string]respjson.Field
		raw          string
	} `json:"-"`
	BaseDescriptor
}

// Returns the unmodified JSON received from the API
func (r CapabilityDescriptorResourceDescriptor) RawJSON() string { return r.JSON.raw }
func (r *CapabilityDescriptorResourceDescriptor) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type CapabilityDescriptorResourceDescriptorSchema struct {
	// For APIs - endpoint, auth details, etc.
	AccessInfo map[string]any `json:"accessInfo"`
	// Relative path to the main executable file within the downloaded package
	ExecutableFile string `json:"executableFile"`
	// Array of URIs where the Capability package can be downloaded
	LocationHints []string `json:"locationHints"`
	// Relative path to the manifest file within the downloaded package
	ManifestFile string `json:"manifestFile"`
	// Optional hint for plugin's output/config directory
	OutputDirectoryHint string `json:"outputDirectoryHint"`
	// Brief, human-readable summary of the plugin's purpose
	Summary string `json:"summary"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		AccessInfo          respjson.Field
		ExecutableFile      respjson.Field
		LocationHints       respjson.Field
		ManifestFile        respjson.Field
		OutputDirectoryHint respjson.Field
		Summary             respjson.Field
		ExtraFields         map[string]respjson.Field
		raw                 string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r CapabilityDescriptorResourceDescriptorSchema) RawJSON() string { return r.JSON.raw }
func (r *CapabilityDescriptorResourceDescriptorSchema) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type CapabilityDescriptorToolDescriptor struct {
	// Reference to a Plugin, API endpoint, or embedded logic
	ExecutionPointer string `json:"executionPointer"`
	// JSON Schema for tool input
	InputSchemaJson string `json:"inputSchemaJSON"`
	// JSON Schema for tool output
	OutputSchemaJson string `json:"outputSchemaJSON"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		ExecutionPointer respjson.Field
		InputSchemaJson  respjson.Field
		OutputSchemaJson respjson.Field
		ExtraFields      map[string]respjson.Field
		raw              string
	} `json:"-"`
	BaseDescriptor
}

// Returns the unmodified JSON received from the API
func (r CapabilityDescriptorToolDescriptor) RawJSON() string { return r.JSON.raw }
func (r *CapabilityDescriptorToolDescriptor) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type CapabilityDescriptorPromptDescriptor struct {
	// JSON Schema for template parameters
	ParametersSchemaJson string `json:"parametersSchemaJSON"`
	// The prompt template string
	Template string `json:"template"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		ParametersSchemaJson respjson.Field
		Template             respjson.Field
		ExtraFields          map[string]respjson.Field
		raw                  string
	} `json:"-"`
	BaseDescriptor
}

// Returns the unmodified JSON received from the API
func (r CapabilityDescriptorPromptDescriptor) RawJSON() string { return r.JSON.raw }
func (r *CapabilityDescriptorPromptDescriptor) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type CapabilityDescriptorMemoryServiceDescriptor struct {
	// Describes the structure/ontology of the memory graph if applicable
	GraphSchema string `json:"graphSchema"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		GraphSchema respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
	BaseDescriptor
}

// Returns the unmodified JSON received from the API
func (r CapabilityDescriptorMemoryServiceDescriptor) RawJSON() string { return r.JSON.raw }
func (r *CapabilityDescriptorMemoryServiceDescriptor) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// Only one field can be non-zero.
//
// Use [param.IsOmitted] to confirm if a field is set.
type CapabilityDescriptorUnionParam struct {
	OfCapabilityDescriptorResourceDescriptor      *CapabilityDescriptorResourceDescriptorParam      `json:",omitzero,inline"`
	OfCapabilityDescriptorToolDescriptor          *CapabilityDescriptorToolDescriptorParam          `json:",omitzero,inline"`
	OfCapabilityDescriptorPromptDescriptor        *CapabilityDescriptorPromptDescriptorParam        `json:",omitzero,inline"`
	OfCapabilityDescriptorMemoryServiceDescriptor *CapabilityDescriptorMemoryServiceDescriptorParam `json:",omitzero,inline"`
	paramUnion
}

func (u CapabilityDescriptorUnionParam) MarshalJSON() ([]byte, error) {
	return param.MarshalUnion(u, u.OfCapabilityDescriptorResourceDescriptor, u.OfCapabilityDescriptorToolDescriptor, u.OfCapabilityDescriptorPromptDescriptor, u.OfCapabilityDescriptorMemoryServiceDescriptor)
}
func (u *CapabilityDescriptorUnionParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, u)
}

// Returns a pointer to the underlying variant's property, if present.
func (u CapabilityDescriptorUnionParam) GetContentHash() *string {
	if vt := u.OfCapabilityDescriptorResourceDescriptor; vt != nil && vt.ContentHash.Valid() {
		return &vt.ContentHash.Value
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u CapabilityDescriptorUnionParam) GetResourceType() *string {
	if vt := u.OfCapabilityDescriptorResourceDescriptor; vt != nil {
		return &vt.ResourceType
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u CapabilityDescriptorUnionParam) GetSchema() *CapabilityDescriptorResourceDescriptorSchemaParam {
	if vt := u.OfCapabilityDescriptorResourceDescriptor; vt != nil {
		return &vt.Schema
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u CapabilityDescriptorUnionParam) GetExecutionPointer() *string {
	if vt := u.OfCapabilityDescriptorToolDescriptor; vt != nil && vt.ExecutionPointer.Valid() {
		return &vt.ExecutionPointer.Value
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u CapabilityDescriptorUnionParam) GetInputSchemaJson() *string {
	if vt := u.OfCapabilityDescriptorToolDescriptor; vt != nil && vt.InputSchemaJson.Valid() {
		return &vt.InputSchemaJson.Value
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u CapabilityDescriptorUnionParam) GetOutputSchemaJson() *string {
	if vt := u.OfCapabilityDescriptorToolDescriptor; vt != nil && vt.OutputSchemaJson.Valid() {
		return &vt.OutputSchemaJson.Value
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u CapabilityDescriptorUnionParam) GetParametersSchemaJson() *string {
	if vt := u.OfCapabilityDescriptorPromptDescriptor; vt != nil && vt.ParametersSchemaJson.Valid() {
		return &vt.ParametersSchemaJson.Value
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u CapabilityDescriptorUnionParam) GetTemplate() *string {
	if vt := u.OfCapabilityDescriptorPromptDescriptor; vt != nil && vt.Template.Valid() {
		return &vt.Template.Value
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u CapabilityDescriptorUnionParam) GetGraphSchema() *string {
	if vt := u.OfCapabilityDescriptorMemoryServiceDescriptor; vt != nil && vt.GraphSchema.Valid() {
		return &vt.GraphSchema.Value
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u CapabilityDescriptorUnionParam) GetID() *string {
	if vt := u.OfCapabilityDescriptorResourceDescriptor; vt != nil && vt.ID.Valid() {
		return &vt.ID.Value
	} else if vt := u.OfCapabilityDescriptorToolDescriptor; vt != nil && vt.ID.Valid() {
		return &vt.ID.Value
	} else if vt := u.OfCapabilityDescriptorPromptDescriptor; vt != nil && vt.ID.Valid() {
		return &vt.ID.Value
	} else if vt := u.OfCapabilityDescriptorMemoryServiceDescriptor; vt != nil && vt.ID.Valid() {
		return &vt.ID.Value
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u CapabilityDescriptorUnionParam) GetCapabilityType() *string {
	if vt := u.OfCapabilityDescriptorResourceDescriptor; vt != nil {
		return (*string)(&vt.CapabilityType)
	} else if vt := u.OfCapabilityDescriptorToolDescriptor; vt != nil {
		return (*string)(&vt.CapabilityType)
	} else if vt := u.OfCapabilityDescriptorPromptDescriptor; vt != nil {
		return (*string)(&vt.CapabilityType)
	} else if vt := u.OfCapabilityDescriptorMemoryServiceDescriptor; vt != nil {
		return (*string)(&vt.CapabilityType)
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u CapabilityDescriptorUnionParam) GetDescription() *string {
	if vt := u.OfCapabilityDescriptorResourceDescriptor; vt != nil && vt.Description.Valid() {
		return &vt.Description.Value
	} else if vt := u.OfCapabilityDescriptorToolDescriptor; vt != nil && vt.Description.Valid() {
		return &vt.Description.Value
	} else if vt := u.OfCapabilityDescriptorPromptDescriptor; vt != nil && vt.Description.Valid() {
		return &vt.Description.Value
	} else if vt := u.OfCapabilityDescriptorMemoryServiceDescriptor; vt != nil && vt.Description.Valid() {
		return &vt.Description.Value
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u CapabilityDescriptorUnionParam) GetGasFeeNrn() *int64 {
	if vt := u.OfCapabilityDescriptorResourceDescriptor; vt != nil && vt.GasFeeNrn.Valid() {
		return &vt.GasFeeNrn.Value
	} else if vt := u.OfCapabilityDescriptorToolDescriptor; vt != nil && vt.GasFeeNrn.Valid() {
		return &vt.GasFeeNrn.Value
	} else if vt := u.OfCapabilityDescriptorPromptDescriptor; vt != nil && vt.GasFeeNrn.Valid() {
		return &vt.GasFeeNrn.Value
	} else if vt := u.OfCapabilityDescriptorMemoryServiceDescriptor; vt != nil && vt.GasFeeNrn.Valid() {
		return &vt.GasFeeNrn.Value
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u CapabilityDescriptorUnionParam) GetName() *string {
	if vt := u.OfCapabilityDescriptorResourceDescriptor; vt != nil && vt.Name.Valid() {
		return &vt.Name.Value
	} else if vt := u.OfCapabilityDescriptorToolDescriptor; vt != nil && vt.Name.Valid() {
		return &vt.Name.Value
	} else if vt := u.OfCapabilityDescriptorPromptDescriptor; vt != nil && vt.Name.Valid() {
		return &vt.Name.Value
	} else if vt := u.OfCapabilityDescriptorMemoryServiceDescriptor; vt != nil && vt.Name.Valid() {
		return &vt.Name.Value
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u CapabilityDescriptorUnionParam) GetOwner() *string {
	if vt := u.OfCapabilityDescriptorResourceDescriptor; vt != nil && vt.Owner.Valid() {
		return &vt.Owner.Value
	} else if vt := u.OfCapabilityDescriptorToolDescriptor; vt != nil && vt.Owner.Valid() {
		return &vt.Owner.Value
	} else if vt := u.OfCapabilityDescriptorPromptDescriptor; vt != nil && vt.Owner.Valid() {
		return &vt.Owner.Value
	} else if vt := u.OfCapabilityDescriptorMemoryServiceDescriptor; vt != nil && vt.Owner.Valid() {
		return &vt.Owner.Value
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u CapabilityDescriptorUnionParam) GetTimestamp() *int64 {
	if vt := u.OfCapabilityDescriptorResourceDescriptor; vt != nil && vt.Timestamp.Valid() {
		return &vt.Timestamp.Value
	} else if vt := u.OfCapabilityDescriptorToolDescriptor; vt != nil && vt.Timestamp.Valid() {
		return &vt.Timestamp.Value
	} else if vt := u.OfCapabilityDescriptorPromptDescriptor; vt != nil && vt.Timestamp.Valid() {
		return &vt.Timestamp.Value
	} else if vt := u.OfCapabilityDescriptorMemoryServiceDescriptor; vt != nil && vt.Timestamp.Valid() {
		return &vt.Timestamp.Value
	}
	return nil
}

// Returns a pointer to the underlying variant's property, if present.
func (u CapabilityDescriptorUnionParam) GetVersion() *string {
	if vt := u.OfCapabilityDescriptorResourceDescriptor; vt != nil && vt.Version.Valid() {
		return &vt.Version.Value
	} else if vt := u.OfCapabilityDescriptorToolDescriptor; vt != nil && vt.Version.Valid() {
		return &vt.Version.Value
	} else if vt := u.OfCapabilityDescriptorPromptDescriptor; vt != nil && vt.Version.Valid() {
		return &vt.Version.Value
	} else if vt := u.OfCapabilityDescriptorMemoryServiceDescriptor; vt != nil && vt.Version.Valid() {
		return &vt.Version.Value
	}
	return nil
}

// Returns a pointer to the underlying variant's CustomMetadata property, if
// present.
func (u CapabilityDescriptorUnionParam) GetCustomMetadata() map[string]any {
	if vt := u.OfCapabilityDescriptorResourceDescriptor; vt != nil {
		return vt.CustomMetadata
	} else if vt := u.OfCapabilityDescriptorToolDescriptor; vt != nil {
		return vt.CustomMetadata
	} else if vt := u.OfCapabilityDescriptorPromptDescriptor; vt != nil {
		return vt.CustomMetadata
	} else if vt := u.OfCapabilityDescriptorMemoryServiceDescriptor; vt != nil {
		return vt.CustomMetadata
	}
	return nil
}

func init() {
	apijson.RegisterUnion[CapabilityDescriptorUnionParam](
		"capabilityType",
		apijson.Discriminator[CapabilityDescriptorResourceDescriptorParam]("RESOURCE"),
		apijson.Discriminator[CapabilityDescriptorResourceDescriptorParam]("TOOL"),
		apijson.Discriminator[CapabilityDescriptorResourceDescriptorParam]("PROMPT"),
		apijson.Discriminator[CapabilityDescriptorResourceDescriptorParam]("MEMORY_SERVICE"),
		apijson.Discriminator[CapabilityDescriptorToolDescriptorParam]("RESOURCE"),
		apijson.Discriminator[CapabilityDescriptorToolDescriptorParam]("TOOL"),
		apijson.Discriminator[CapabilityDescriptorToolDescriptorParam]("PROMPT"),
		apijson.Discriminator[CapabilityDescriptorToolDescriptorParam]("MEMORY_SERVICE"),
		apijson.Discriminator[CapabilityDescriptorPromptDescriptorParam]("RESOURCE"),
		apijson.Discriminator[CapabilityDescriptorPromptDescriptorParam]("TOOL"),
		apijson.Discriminator[CapabilityDescriptorPromptDescriptorParam]("PROMPT"),
		apijson.Discriminator[CapabilityDescriptorPromptDescriptorParam]("MEMORY_SERVICE"),
		apijson.Discriminator[CapabilityDescriptorMemoryServiceDescriptorParam]("RESOURCE"),
		apijson.Discriminator[CapabilityDescriptorMemoryServiceDescriptorParam]("TOOL"),
		apijson.Discriminator[CapabilityDescriptorMemoryServiceDescriptorParam]("PROMPT"),
		apijson.Discriminator[CapabilityDescriptorMemoryServiceDescriptorParam]("MEMORY_SERVICE"),
	)
}

type CapabilityDescriptorResourceDescriptorParam struct {
	// SHA256 hash of the Capability package
	ContentHash param.Opt[string] `json:"contentHash,omitzero"`
	// Specific kind of resource
	ResourceType string                                            `json:"resourceType,omitzero"`
	Schema       CapabilityDescriptorResourceDescriptorSchemaParam `json:"schema,omitzero"`
	BaseDescriptorParam
}

func (r CapabilityDescriptorResourceDescriptorParam) MarshalJSON() (data []byte, err error) {
	type shadow CapabilityDescriptorResourceDescriptorParam
	return param.MarshalObject(r, (*shadow)(&r))
}

type CapabilityDescriptorResourceDescriptorSchemaParam struct {
	// Relative path to the main executable file within the downloaded package
	ExecutableFile param.Opt[string] `json:"executableFile,omitzero"`
	// Relative path to the manifest file within the downloaded package
	ManifestFile param.Opt[string] `json:"manifestFile,omitzero"`
	// Optional hint for plugin's output/config directory
	OutputDirectoryHint param.Opt[string] `json:"outputDirectoryHint,omitzero"`
	// Brief, human-readable summary of the plugin's purpose
	Summary param.Opt[string] `json:"summary,omitzero"`
	// For APIs - endpoint, auth details, etc.
	AccessInfo map[string]any `json:"accessInfo,omitzero"`
	// Array of URIs where the Capability package can be downloaded
	LocationHints []string `json:"locationHints,omitzero"`
	paramObj
}

func (r CapabilityDescriptorResourceDescriptorSchemaParam) MarshalJSON() (data []byte, err error) {
	type shadow CapabilityDescriptorResourceDescriptorSchemaParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *CapabilityDescriptorResourceDescriptorSchemaParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type CapabilityDescriptorToolDescriptorParam struct {
	// Reference to a Plugin, API endpoint, or embedded logic
	ExecutionPointer param.Opt[string] `json:"executionPointer,omitzero"`
	// JSON Schema for tool input
	InputSchemaJson param.Opt[string] `json:"inputSchemaJSON,omitzero"`
	// JSON Schema for tool output
	OutputSchemaJson param.Opt[string] `json:"outputSchemaJSON,omitzero"`
	BaseDescriptorParam
}

func (r CapabilityDescriptorToolDescriptorParam) MarshalJSON() (data []byte, err error) {
	type shadow CapabilityDescriptorToolDescriptorParam
	return param.MarshalObject(r, (*shadow)(&r))
}

type CapabilityDescriptorPromptDescriptorParam struct {
	// JSON Schema for template parameters
	ParametersSchemaJson param.Opt[string] `json:"parametersSchemaJSON,omitzero"`
	// The prompt template string
	Template param.Opt[string] `json:"template,omitzero"`
	BaseDescriptorParam
}

func (r CapabilityDescriptorPromptDescriptorParam) MarshalJSON() (data []byte, err error) {
	type shadow CapabilityDescriptorPromptDescriptorParam
	return param.MarshalObject(r, (*shadow)(&r))
}

type CapabilityDescriptorMemoryServiceDescriptorParam struct {
	// Describes the structure/ontology of the memory graph if applicable
	GraphSchema param.Opt[string] `json:"graphSchema,omitzero"`
	BaseDescriptorParam
}

func (r CapabilityDescriptorMemoryServiceDescriptorParam) MarshalJSON() (data []byte, err error) {
	type shadow CapabilityDescriptorMemoryServiceDescriptorParam
	return param.MarshalObject(r, (*shadow)(&r))
}

type McpCapabilityUpdateResponse struct {
	Message         string `json:"message"`
	TransactionHash string `json:"transactionHash"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Message         respjson.Field
		TransactionHash respjson.Field
		ExtraFields     map[string]respjson.Field
		raw             string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r McpCapabilityUpdateResponse) RawJSON() string { return r.JSON.raw }
func (r *McpCapabilityUpdateResponse) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type McpCapabilityInvokeResponse struct {
	ContextID       string `json:"contextID"`
	Message         string `json:"message"`
	TransactionHash string `json:"transactionHash"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		ContextID       respjson.Field
		Message         respjson.Field
		TransactionHash respjson.Field
		ExtraFields     map[string]respjson.Field
		raw             string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r McpCapabilityInvokeResponse) RawJSON() string { return r.JSON.raw }
func (r *McpCapabilityInvokeResponse) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type McpCapabilityPrepareRegistrationResponse struct {
	// A server-side hash identifying this pending registration preparation.
	PendingTransactionHash string `json:"pendingTransactionHash,required"`
	// A structured representation of the unsigned transaction the client is expected
	// to form and sign.
	UnsignedTransaction McpCapabilityPrepareRegistrationResponseUnsignedTransaction `json:"unsignedTransaction,required"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		PendingTransactionHash respjson.Field
		UnsignedTransaction    respjson.Field
		ExtraFields            map[string]respjson.Field
		raw                    string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r McpCapabilityPrepareRegistrationResponse) RawJSON() string { return r.JSON.raw }
func (r *McpCapabilityPrepareRegistrationResponse) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// A structured representation of the unsigned transaction the client is expected
// to form and sign.
type McpCapabilityPrepareRegistrationResponseUnsignedTransaction struct {
	// Base64 encoded JSON bytes of MCPRegisterCapabilityData.
	Data      string `json:"data" format:"base64"`
	Fee       string `json:"fee"`
	From      string `json:"from"`
	Timestamp int64  `json:"timestamp"`
	To        string `json:"to,nullable"`
	Type      string `json:"type"`
	// Hash of the payload that needs to be signed.
	UnsignedTransactionPayloadHash string `json:"unsignedTransactionPayloadHash"`
	Value                          string `json:"value"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Data                           respjson.Field
		Fee                            respjson.Field
		From                           respjson.Field
		Timestamp                      respjson.Field
		To                             respjson.Field
		Type                           respjson.Field
		UnsignedTransactionPayloadHash respjson.Field
		Value                          respjson.Field
		ExtraFields                    map[string]respjson.Field
		raw                            string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r McpCapabilityPrepareRegistrationResponseUnsignedTransaction) RawJSON() string {
	return r.JSON.raw
}
func (r *McpCapabilityPrepareRegistrationResponseUnsignedTransaction) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type McpCapabilityUpdateParams struct {
	SignedTransaction TransactionParam `json:"signedTransaction,omitzero,required"`
	paramObj
}

func (r McpCapabilityUpdateParams) MarshalJSON() (data []byte, err error) {
	type shadow McpCapabilityUpdateParams
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *McpCapabilityUpdateParams) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type McpCapabilityInvokeParams struct {
	SignedTransaction TransactionParam `json:"signedTransaction,omitzero,required"`
	paramObj
}

func (r McpCapabilityInvokeParams) MarshalJSON() (data []byte, err error) {
	type shadow McpCapabilityInvokeParams
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *McpCapabilityInvokeParams) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type McpCapabilityPrepareRegistrationParams struct {
	// Network fee for the registration transaction.
	Fee int64 `json:"fee,required"`
	// Address of the account that will own the capability.
	OwnerAddress string `json:"ownerAddress,required"`
	// Additional information
	Message    param.Opt[string]              `json:"message,omitzero"`
	Descriptor CapabilityDescriptorUnionParam `json:"descriptor,omitzero"`
	paramObj
}

func (r McpCapabilityPrepareRegistrationParams) MarshalJSON() (data []byte, err error) {
	type shadow McpCapabilityPrepareRegistrationParams
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *McpCapabilityPrepareRegistrationParams) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}
