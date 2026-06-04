// File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

package transaction

import (
	"context"
	"net/http"

	"github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction/internal/apijson"
	"github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction/internal/requestconfig"
	"github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction/option"
	"github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction/packages/param"
	"github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction/packages/respjson"
)

// URIGeneratorService contains methods and other services that help with
// interacting with the knirvchain-transaction-sdk API.
//
// Note, unlike clients, this service does not read variables from the environment
// automatically. You should not instantiate this service directly, and instead use
// the [NewURIGeneratorService] method instead.
type URIGeneratorService struct {
	Options []option.RequestOption
}

// NewURIGeneratorService generates a new service that applies the given options to
// each request. These options are applied after the parent client's options (if
// there is one), and before any request-specific options.
func NewURIGeneratorService(opts ...option.RequestOption) (r URIGeneratorService) {
	r = URIGeneratorService{}
	r.Options = opts
	return
}

// Generates a new URI and announces it to the network
func (r *URIGeneratorService) New(ctx context.Context, body URIGeneratorNewParams, opts ...option.RequestOption) (res *URIGeneratorNewResponse, err error) {
	opts = append(r.Options[:], opts...)
	path := "uriGenerator"
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodPost, path, body, &res, opts...)
	return
}

type URIGeneratorNewResponse struct {
	// Additional information
	Message string `json:"message"`
	// Resource ID
	ResourceID string `json:"resource_id"`
	// Whether the operation was successful
	Success bool `json:"success"`
	// Generated URI
	URI string `json:"uri"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Message     respjson.Field
		ResourceID  respjson.Field
		Success     respjson.Field
		URI         respjson.Field `json:"uri"`
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r URIGeneratorNewResponse) RawJSON() string { return r.JSON.raw }
func (r *URIGeneratorNewResponse) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type URIGeneratorNewParams struct {
	// Hash of the content
	ContentHash param.Opt[string] `json:"content_hash,omitzero"`
	// Owner address
	Owner param.Opt[string] `json:"owner,omitzero"`
	// Type of resource
	ResourceType param.Opt[string] `json:"resource_type,omitzero"`
	// Additional metadata
	Metadata map[string]any `json:"metadata,omitzero"`
	paramObj
}

func (r URIGeneratorNewParams) MarshalJSON() (data []byte, err error) {
	type shadow URIGeneratorNewParams
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *URIGeneratorNewParams) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}
