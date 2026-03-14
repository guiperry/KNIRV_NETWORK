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

// PeerService contains methods and other services that help with interacting with
// the knirvchain-transaction-sdk API.
//
// Note, unlike clients, this service does not read variables from the environment
// automatically. You should not instantiate this service directly, and instead use
// the [NewPeerService] method instead.
type PeerService struct {
	Options []option.RequestOption
}

// NewPeerService generates a new service that applies the given options to each
// request. These options are applied after the parent client's options (if there
// is one), and before any request-specific options.
func NewPeerService(opts ...option.RequestOption) (r PeerService) {
	r = PeerService{}
	r.Options = opts
	return
}

// Retrieves the list of connected peers
func (r *PeerService) List(ctx context.Context, opts ...option.RequestOption) (res *[]PeerListResponse, err error) {
	opts = append(r.Options[:], opts...)
	path := "peers"
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodGet, path, nil, &res, opts...)
	return
}

type PeerListResponse struct {
	// Peer ID
	ID string `json:"id"`
	// Peer address
	Address string `json:"address"`
	// Unix timestamp when the peer connected
	ConnectedSince int64 `json:"connected_since"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		ID             respjson.Field
		Address        respjson.Field
		ConnectedSince respjson.Field
		ExtraFields    map[string]respjson.Field
		raw            string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r PeerListResponse) RawJSON() string { return r.JSON.raw }
func (r *PeerListResponse) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}
