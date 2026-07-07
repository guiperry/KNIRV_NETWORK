// File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

package transaction

import (
	"context"
	"net/http"

	"github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction/internal/requestconfig"
	"github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction/option"
)

// TxnPoolService contains methods and other services that help with interacting
// with the knirvchain-transaction-sdk API.
//
// Note, unlike clients, this service does not read variables from the environment
// automatically. You should not instantiate this service directly, and instead use
// the [NewTxnPoolService] method instead.
type TxnPoolService struct {
	Options []option.RequestOption
}

// NewTxnPoolService generates a new service that applies the given options to each
// request. These options are applied after the parent client's options (if there
// is one), and before any request-specific options.
func NewTxnPoolService(opts ...option.RequestOption) (r TxnPoolService) {
	r = TxnPoolService{}
	r.Options = opts
	return
}

// Retrieves the current transaction pool
func (r *TxnPoolService) Get(ctx context.Context, opts ...option.RequestOption) (res *[]Transaction, err error) {
	opts = append(r.Options[:], opts...)
	path := "txn_pool"
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodGet, path, nil, &res, opts...)
	return
}
