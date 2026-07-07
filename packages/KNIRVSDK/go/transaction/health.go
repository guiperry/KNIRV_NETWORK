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

// HealthService contains methods and other services that help with interacting
// with the knirvchain-transaction-sdk API.
//
// Note, unlike clients, this service does not read variables from the environment
// automatically. You should not instantiate this service directly, and instead use
// the [NewHealthService] method instead.
type HealthService struct {
	Options []option.RequestOption
}

// NewHealthService generates a new service that applies the given options to each
// request. These options are applied after the parent client's options (if there
// is one), and before any request-specific options.
func NewHealthService(opts ...option.RequestOption) (r HealthService) {
	r = HealthService{}
	r.Options = opts
	return
}

// Checks the health status of the blockchain node
func (r *HealthService) Check(ctx context.Context, opts ...option.RequestOption) (res *HealthCheckResponse, err error) {
	opts = append(r.Options[:], opts...)
	path := "health"
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodGet, path, nil, &res, opts...)
	return
}

type HealthCheckResponse struct {
	// Blockchain subsystem status
	Blockchain bool `json:"blockchain"`
	// Database subsystem status
	Database bool `json:"database"`
	// Additional status information
	Message string `json:"message"`
	// Network subsystem status
	Network bool `json:"network"`
	// Overall health status
	//
	// Any of "healthy", "unhealthy".
	Status HealthCheckResponseStatus `json:"status"`
	// JSON contains metadata for fields, check presence with [respjson.Field.Valid].
	JSON struct {
		Blockchain  respjson.Field
		Database    respjson.Field
		Message     respjson.Field
		Network     respjson.Field
		Status      respjson.Field
		ExtraFields map[string]respjson.Field
		raw         string
	} `json:"-"`
}

// Returns the unmodified JSON received from the API
func (r HealthCheckResponse) RawJSON() string { return r.JSON.raw }
func (r *HealthCheckResponse) UnmarshalJSON(data []byte) error {
	return apijson.UnmarshalRoot(data, r)
}

// Overall health status
type HealthCheckResponseStatus string

const (
	HealthCheckResponseStatusHealthy   HealthCheckResponseStatus = "healthy"
	HealthCheckResponseStatusUnhealthy HealthCheckResponseStatus = "unhealthy"
)
