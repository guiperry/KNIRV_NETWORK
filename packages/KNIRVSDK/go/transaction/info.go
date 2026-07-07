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

// InfoService contains methods and other services that help with interacting with
// the knirvchain-transaction-sdk API.
//
// Note, unlike clients, this service does not read variables from the environment
// automatically. You should not instantiate this service directly, and instead use
// the [NewInfoService] method instead.
type InfoService struct {
	Options []option.RequestOption
}

// NewInfoService generates a new service that applies the given options to each
// request. These options are applied after the parent client's options (if there
// is one), and before any request-specific options.
func NewInfoService(opts ...option.RequestOption) (r InfoService) {
	r = InfoService{}
	r.Options = opts
	return
}

// Retrieves information about the blockchain node
func (r *InfoService) Get(ctx context.Context, opts ...option.RequestOption) (res *InfoGetResponse, err error) {
	opts = append(r.Options[:], opts...)
	path := "info"
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodGet, path, nil, &res, opts...)
	return
}

type InfoGetResponse struct {
	// Current blockchain height
	ChainHeight int64 `json:"chain_height"`
	// Whether the node is currently mining
	IsMining bool `json:"is_mining"`
	// Unique identifier for the node
	NodeID string `json:"node_id"`
	// Number of connected peers
	PeerCount int64 `json:"peer_count"`
	// Node uptime in seconds
	Uptime int64 `json:"uptime"`
	// Software version
	Version string `json:"version"`
	// Node's wallet address
	WalletAddress string `json:"wallet_address"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		ChainHeight   respjson.Field
		IsMining      respjson.Field
		NodeID        respjson.Field
		PeerCount     respjson.Field
		Uptime        respjson.Field
		Version       respjson.Field
		WalletAddress respjson.Field
		ExtraFields   map[string]respjson.Field
		raw           string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r InfoGetResponse) RawJSON() string { return r.JSON.raw }
func (r *InfoGetResponse) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}
