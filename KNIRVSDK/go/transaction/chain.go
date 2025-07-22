// File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

package transaction

import (
	"context"
	"net/http"

	"github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction/internal/apijson"
	"github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction/internal/requestconfig"
	"github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction/option"
	"github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction/packages/respjson"
)

// ChainService contains methods and other services that help with interacting with
// the knirvchain-transaction-sdk API.
//
// Note, unlike clients, this service does not read variables from the environment
// automatically. You should not instantiate this service directly, and instead use
// the [NewChainService] method instead.
type ChainService struct {
	Options []option.RequestOption
}

// NewChainService generates a new service that applies the given options to each
// request. These options are applied after the parent client's options (if there
// is one), and before any request-specific options.
func NewChainService(opts ...option.RequestOption) (r ChainService) {
	r = ChainService{}
	r.Options = opts
	return
}

// Retrieves the current state of the blockchain including blocks, transaction
// pool, and reflections
func (r *ChainService) Get(ctx context.Context, opts ...option.RequestOption) (res *ChainGetResponse, err error) {
	opts = append(r.Options[:], opts...)
	path := "chain"
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodGet, path, nil, &res, opts...)
	return
}

type ChainGetResponse struct {
	Blocks []Block `json:"blocks"`
	// The blockchain's address
	ChainAddress string `json:"chain_address"`
	// Unique identifier for the blockchain
	ChainID string `json:"chain_id"`
	// Map of reflection URLs
	Reflections     map[string]bool `json:"reflections"`
	TransactionPool []Transaction   `json:"transaction_pool"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Blocks          respjson.Field
		ChainAddress    respjson.Field
		ChainID         respjson.Field
		Reflections     respjson.Field
		TransactionPool respjson.Field
		ExtraFields     map[string]respjson.Field
		raw             string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r ChainGetResponse) RawJSON() string { return r.JSON.raw }
func (r *ChainGetResponse) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}
