// File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

package transaction

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction/internal/apijson"
	"github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction/internal/apiquery"
	"github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction/internal/requestconfig"
	"github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction/option"
	"github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction/packages/param"
	"github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction/packages/respjson"
)

// McpService contains methods and other services that help with interacting with
// the knirvchain-transaction-sdk API.
//
// Note, unlike clients, this service does not read variables from the environment
// automatically. You should not instantiate this service directly, and instead use
// the [NewMcpService] method instead.
type McpService struct {
	Options    []option.RequestOption
	Capability McpCapabilityService
}

// NewMcpService generates a new service that applies the given options to each
// request. These options are applied after the parent client's options (if there
// is one), and before any request-specific options.
func NewMcpService(opts ...option.RequestOption) (r McpService) {
	r = McpService{}
	r.Options = opts
	r.Capability = NewMcpCapabilityService(opts...)
	return
}

// Retrieves a specific context record by ID
func (r *McpService) Get(ctx context.Context, contextID string, opts ...option.RequestOption) (res *ContextRecord, err error) {
	opts = append(r.Options[:], opts...)
	if contextID == "" {
		err = errors.New("missing required context_id parameter")
		return
	}
	path := fmt.Sprintf("mcp/context/%s", contextID)
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodGet, path, nil, &res, opts...)
	return
}

// Lists all registered capabilities
func (r *McpService) GetCapabilities(ctx context.Context, query McpGetCapabilitiesParams, opts ...option.RequestOption) (res *[]CapabilityDescriptorUnion, err error) {
	opts = append(r.Options[:], opts...)
	path := "mcp/capabilities"
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodGet, path, query, &res, opts...)
	return
}

// Lists all context records
func (r *McpService) GetContexts(ctx context.Context, query McpGetContextsParams, opts ...option.RequestOption) (res *[]ContextRecord, err error) {
	opts = append(r.Options[:], opts...)
	path := "mcp/contexts"
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodGet, path, query, &res, opts...)
	return
}

type ContextRecord struct {
	// Unique identifier for the context record
	ID string `json:"id"`
	// Identifier of the MCP capability involved
	CapabilityID string    `json:"capabilityID"`
	CreatedAt    time.Time `json:"createdAt" format:"date-time"`
	// Primitive-specific information
	Details map[string]any `json:"details"`
	// Public key or address of the entity initiating the context
	Initiator string `json:"initiator"`
	// Optional hash of the input data to the interaction
	InputHash string `json:"inputHash"`
	// Type of interaction
	//
	// Any of "CAPABILITY_REGISTRATION", "CAPABILITY_INVOCATION", "CAPABILITY_UPDATE",
	// "GENERAL_MESSAGE", "TOOL_INVOCATION", "PROMPT_USAGE", "RESOURCE_ACCESS",
	// "PLUGIN_EXECUTION", "SAMPLING_REQUEST_SENT", "SAMPLING_RESPONSE_RECEIVED",
	// "MEMORY_WRITE".
	InteractionType ContextRecordInteractionType `json:"interactionType"`
	// Optional hash of the output data from the interaction
	OutputHash string `json:"outputHash"`
	// Cryptographic signature of the ContextRecord content by the initiator
	Signature string `json:"signature" format:"byte"`
	// Any of "pending", "success", "failed", "processing".
	Status ContextRecordStatus `json:"status"`
	// Timestamp of the event
	Timestamp int64     `json:"timestamp"`
	UpdatedAt time.Time `json:"updatedAt" format:"date-time"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		ID              respjson.Field
		CapabilityID    respjson.Field
		CreatedAt       respjson.Field
		Details         respjson.Field
		Initiator       respjson.Field
		InputHash       respjson.Field
		InteractionType respjson.Field
		OutputHash      respjson.Field
		Signature       respjson.Field
		Status          respjson.Field
		Timestamp       respjson.Field
		UpdatedAt       respjson.Field
		ExtraFields     map[string]respjson.Field
		raw             string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r ContextRecord) RawJSON() string { return r.JSON.raw }
func (r *ContextRecord) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// Type of interaction
type ContextRecordInteractionType string

const (
	ContextRecordInteractionTypeCapabilityRegistration   ContextRecordInteractionType = "CAPABILITY_REGISTRATION"
	ContextRecordInteractionTypeCapabilityInvocation     ContextRecordInteractionType = "CAPABILITY_INVOCATION"
	ContextRecordInteractionTypeCapabilityUpdate         ContextRecordInteractionType = "CAPABILITY_UPDATE"
	ContextRecordInteractionTypeGeneralMessage           ContextRecordInteractionType = "GENERAL_MESSAGE"
	ContextRecordInteractionTypeToolInvocation           ContextRecordInteractionType = "TOOL_INVOCATION"
	ContextRecordInteractionTypePromptUsage              ContextRecordInteractionType = "PROMPT_USAGE"
	ContextRecordInteractionTypeResourceAccess           ContextRecordInteractionType = "RESOURCE_ACCESS"
	ContextRecordInteractionTypePluginExecution          ContextRecordInteractionType = "PLUGIN_EXECUTION"
	ContextRecordInteractionTypeSamplingRequestSent      ContextRecordInteractionType = "SAMPLING_REQUEST_SENT"
	ContextRecordInteractionTypeSamplingResponseReceived ContextRecordInteractionType = "SAMPLING_RESPONSE_RECEIVED"
	ContextRecordInteractionTypeMemoryWrite              ContextRecordInteractionType = "MEMORY_WRITE"
)

type ContextRecordStatus string

const (
	ContextRecordStatusPending    ContextRecordStatus = "pending"
	ContextRecordStatusSuccess    ContextRecordStatus = "success"
	ContextRecordStatusFailed     ContextRecordStatus = "failed"
	ContextRecordStatusProcessing ContextRecordStatus = "processing"
)

type McpGetCapabilitiesParams struct {
	// Filter by owner address
	Owner param.Opt[string] `query:"owner,omitzero" json:"-"`
	// Filter by capability type
	//
	// Any of "RESOURCE", "TOOL", "PROMPT", "MEMORY_SERVICE".
	Type McpGetCapabilitiesParamsType `query:"type,omitzero" json:"-"`
	paramObj
}

// URLQuery serializes [McpGetCapabilitiesParams]'s query parameters as
// `url.Values`.
func (r McpGetCapabilitiesParams) URLQuery() (v url.Values, err error) {
	return apiquery.MarshalWithSettings(r, apiquery.QuerySettings{
		ArrayFormat:  apiquery.ArrayQueryFormatComma,
		NestedFormat: apiquery.NestedQueryFormatBrackets,
	})
}

// Filter by capability type
type McpGetCapabilitiesParamsType string

const (
	McpGetCapabilitiesParamsTypeResource      McpGetCapabilitiesParamsType = "RESOURCE"
	McpGetCapabilitiesParamsTypeTool          McpGetCapabilitiesParamsType = "TOOL"
	McpGetCapabilitiesParamsTypePrompt        McpGetCapabilitiesParamsType = "PROMPT"
	McpGetCapabilitiesParamsTypeMemoryService McpGetCapabilitiesParamsType = "MEMORY_SERVICE"
)

type McpGetContextsParams struct {
	// Filter by capability ID
	CapabilityID param.Opt[string] `query:"capabilityId,omitzero" json:"-"`
	// Filter by initiator address
	Initiator param.Opt[string] `query:"initiator,omitzero" json:"-"`
	// Filter by interaction type
	//
	// Any of "TOOL_INVOCATION", "PROMPT_USAGE", "RESOURCE_ACCESS", "PLUGIN_EXECUTION",
	// "SAMPLING_REQUEST_SENT", "SAMPLING_RESPONSE_RECEIVED", "MEMORY_WRITE",
	// "CAPABILITY_REGISTRATION".
	InteractionType McpGetContextsParamsInteractionType `query:"interactionType,omitzero" json:"-"`
	paramObj
}

// URLQuery serializes [McpGetContextsParams]'s query parameters as `url.Values`.
func (r McpGetContextsParams) URLQuery() (v url.Values, err error) {
	return apiquery.MarshalWithSettings(r, apiquery.QuerySettings{
		ArrayFormat:  apiquery.ArrayQueryFormatComma,
		NestedFormat: apiquery.NestedQueryFormatBrackets,
	})
}

// Filter by interaction type
type McpGetContextsParamsInteractionType string

const (
	McpGetContextsParamsInteractionTypeToolInvocation           McpGetContextsParamsInteractionType = "TOOL_INVOCATION"
	McpGetContextsParamsInteractionTypePromptUsage              McpGetContextsParamsInteractionType = "PROMPT_USAGE"
	McpGetContextsParamsInteractionTypeResourceAccess           McpGetContextsParamsInteractionType = "RESOURCE_ACCESS"
	McpGetContextsParamsInteractionTypePluginExecution          McpGetContextsParamsInteractionType = "PLUGIN_EXECUTION"
	McpGetContextsParamsInteractionTypeSamplingRequestSent      McpGetContextsParamsInteractionType = "SAMPLING_REQUEST_SENT"
	McpGetContextsParamsInteractionTypeSamplingResponseReceived McpGetContextsParamsInteractionType = "SAMPLING_RESPONSE_RECEIVED"
	McpGetContextsParamsInteractionTypeMemoryWrite              McpGetContextsParamsInteractionType = "MEMORY_WRITE"
	McpGetContextsParamsInteractionTypeCapabilityRegistration   McpGetContextsParamsInteractionType = "CAPABILITY_REGISTRATION"
)
