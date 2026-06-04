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

// TransactionService contains methods and other services that help with
// interacting with the knirvchain-transaction-sdk API.
//
// Note, unlike clients, this service does not read variables from the environment
// automatically. You should not instantiate this service directly, and instead use
// the [NewTransactionService] method instead.
type TransactionService struct {
	Options []option.RequestOption
}

// NewTransactionService generates a new service that applies the given options to
// each request. These options are applied after the parent client's options (if
// there is one), and before any request-specific options.
func NewTransactionService(opts ...option.RequestOption) (r TransactionService) {
	r = TransactionService{}
	r.Options = opts
	return
}

// Submits a pre-signed transaction to the blockchain network. This endpoint is
// used for various transaction types, including standard transfers and MCP
// operations like capability registration (after preparation), invocation, or
// updates. The `Transaction.type` field and `Transaction.data` structure will
// determine how it's processed.
func (r *TransactionService) Submit(ctx context.Context, body TransactionSubmitParams, opts ...option.RequestOption) (res *TransactionSubmitResponse, err error) {
	opts = append(r.Options[:], opts...)
	path := "transaction"
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodPost, path, body, &res, opts...)
	return
}

type Transaction struct {
	// Transaction hash/ID.
	ID string `json:"id,required"`
	// NRN token gas fee paid for this transaction
	Fee string `json:"fee,required"`
	// Sender address
	From string `json:"from,required"`
	// Public key of the sender
	PublicKey string `json:"public_key,required"`
	// Cryptographic signature
	Signature string `json:"signature,required" format:"byte"`
	// Unix timestamp (nanoseconds) when the transaction was created
	Timestamp int64 `json:"timestamp,required"`
	// Transaction type for MCP transactions
	Type    string `json:"type,required"`
	Version int64  `json:"version,required"`
	// Transaction payload, structure depends on transaction type. For MCP ops, this
	// will be JSON of MCPRegisterCapabilityData, MCPInvokeCapabilityData, etc.
	Data map[string]any `json:"data"`
	// Transaction status (PENDING, SUCCESS, FAILED)
	Status string `json:"status"`
	// Recipient address
	To string `json:"to,nullable"`
	// Hash of the transaction
	TransactionHash string `json:"transaction_hash"`
	// Amount of NRN transferred.
	Value string `json:"value"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		ID              respjson.Field
		Fee             respjson.Field
		From            respjson.Field
		PublicKey       respjson.Field
		Signature       respjson.Field
		Timestamp       respjson.Field
		Type            respjson.Field
		Version         respjson.Field
		Data            respjson.Field
		Status          respjson.Field
		To              respjson.Field
		TransactionHash respjson.Field
		Value           respjson.Field
		ExtraFields     map[string]respjson.Field
		raw             string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r Transaction) RawJSON() string { return r.JSON.raw }
func (r *Transaction) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// ToParam converts this Transaction to a TransactionParam.
//
// Warning: the fields of the param type will not be present. ToParam should only
// be used at the last possible moment before sending a request. Test for this with
// TransactionParam.Overrides()
func (r Transaction) ToParam() TransactionParam {
	return param.Override[TransactionParam](json.RawMessage(r.RawJSON()))
}

// The properties ID, Fee, From, PublicKey, Signature, Timestamp, Type, Version are
// required.
type TransactionParam struct {
	// Transaction hash/ID.
	ID string `json:"id,required"`
	// NRN token gas fee paid for this transaction
	Fee string `json:"fee,required"`
	// Sender address
	From string `json:"from,required"`
	// Public key of the sender
	PublicKey string `json:"public_key,required"`
	// Cryptographic signature
	Signature string `json:"signature,required" format:"byte"`
	// Unix timestamp (nanoseconds) when the transaction was created
	Timestamp int64 `json:"timestamp,required"`
	// Transaction type for MCP transactions
	Type    string `json:"type,required"`
	Version int64  `json:"version,required"`
	// Recipient address
	To param.Opt[string] `json:"to,omitzero"`
	// Transaction status (PENDING, SUCCESS, FAILED)
	Status param.Opt[string] `json:"status,omitzero"`
	// Hash of the transaction
	TransactionHash param.Opt[string] `json:"transaction_hash,omitzero"`
	// Amount of NRN transferred.
	Value param.Opt[string] `json:"value,omitzero"`
	// Transaction payload, structure depends on transaction type. For MCP ops, this
	// will be JSON of MCPRegisterCapabilityData, MCPInvokeCapabilityData, etc.
	Data map[string]any `json:"data,omitzero"`
	paramObj
}

func (r TransactionParam) MarshalJSON() (data []byte, err error) {
	type shadow TransactionParam
	return param.MarshalObject(r, (*shadow)(&r))
}
func (r *TransactionParam) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type TransactionSubmitResponse struct {
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
func (r TransactionSubmitResponse) RawJSON() string { return r.JSON.raw }
func (r *TransactionSubmitResponse) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

type TransactionSubmitParams struct {
	Transaction TransactionParam
	paramObj
}

func (r TransactionSubmitParams) MarshalJSON() (data []byte, err error) {
	return json.Marshal(r.Transaction)
}
func (r *TransactionSubmitParams) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &r.Transaction)
}
