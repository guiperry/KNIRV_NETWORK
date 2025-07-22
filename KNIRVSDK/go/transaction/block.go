// File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

package transaction

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction/internal/apijson"
	"github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction/internal/requestconfig"
	"github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction/option"
	"github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction/packages/param"
	"github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction/packages/respjson"
)

// BlockService contains methods and other services that help with interacting with
// the knirvchain-transaction-sdk API.
//
// Note, unlike clients, this service does not read variables from the environment
// automatically. You should not instantiate this service directly, and instead use
// the [NewBlockService] method instead.
type BlockService struct {
	Options []option.RequestOption
}

// NewBlockService generates a new service that applies the given options to each
// request. These options are applied after the parent client's options (if there
// is one), and before any request-specific options.
func NewBlockService(opts ...option.RequestOption) (r BlockService) {
	r = BlockService{}
	r.Options = opts
	return
}

// Submits a new block to the blockchain
func (r *BlockService) Submit(ctx context.Context, body BlockSubmitParams, opts ...option.RequestOption) (res *BlockSubmitResponse, err error) {
	opts = append(r.Options[:], opts...)
	path := "block"
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodPost, path, body, &res, opts...)
	return
}

type Block struct {
	// Block number in the chain
	BlockNumber int64 `json:"block_number"`
	// Hash of the block
	Hash string `json:"hash" format:"byte"`
	// Nonce used for mining
	Nonce int64 `json:"nonce"`
	// Hash of the previous block
	PrevHash string `json:"prevHash" format:"byte"`
	// Address of the block proposer (miner/validator)
	ProposerAddress string `json:"proposer_address"`
	// Unix timestamp when the block was created
	Timestamp int64 `json:"timestamp"`
	// Transactions included in this block
	Transactions []Transaction `json:"transactions"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		BlockNumber     respjson.Field
		Hash            respjson.Field
		Nonce           respjson.Field
		PrevHash        respjson.Field
		ProposerAddress respjson.Field
		Timestamp       respjson.Field
		Transactions    respjson.Field
		ExtraFields     map[string]respjson.Field
		raw             string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r Block) RawJSON() string { return r.JSON.raw }
func (r *Block) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// ToParam converts this Block to a BlockParam.
//
// Warning: the fields of the param type will not be present. ToParam should only
// be used at the last possible moment before sending a request. Test for this with
// BlockParam.Overrides()
func (r Block) ToParam() BlockParam {
	return param.Override[BlockParam](json.RawMessage(r.RawJSON()))
}

type BlockParam struct {
	// Block number in the chain
	BlockNumber param.Opt[int64] `json:"block_number,omitzero"`
	// Hash of the block
	Hash param.Opt[string] `json:"hash,omitzero" format:"byte"`
	// Nonce used for mining
	Nonce param.Opt[int64] `json:"nonce,omitzero"`
	// Hash of the previous block
	PrevHash param.Opt[string] `json:"prevHash,omitzero" format:"byte"`
	// Address of the block proposer (miner/validator)
	ProposerAddress param.Opt[string] `json:"proposer_address,omitzero"`
	// Unix timestamp when the block was created
	Timestamp param.Opt[int64] `json:"timestamp,omitzero"`
	// Transactions included in this block
	Transactions []TransactionParam `json:"transactions,omitzero"`
	paramObj
}

func (r BlockParam) MarshalJSON() (data []byte, err error) {
	type shadow BlockParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *BlockParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type BlockSubmitResponse struct {
	Message string `json:"message"`
	Success bool   `json:"success"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Message     respjson.Field
		Success     respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r BlockSubmitResponse) RawJSON() string { return r.JSON.raw }
func (r *BlockSubmitResponse) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type BlockSubmitParams struct {
	Block BlockParam
	paramObj
}

func (r BlockSubmitParams) MarshalJSON() (data []byte, err error) {
	return json.Marshal(r.Block)
}
func (r *BlockSubmitParams) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &r.Block)
}
