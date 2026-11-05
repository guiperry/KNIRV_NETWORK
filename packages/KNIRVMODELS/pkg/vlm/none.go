package vlm

import (
	"context"

	sdk "github.com/guiperry/knirv-sdk-go/models"
)

// NoneBackend is the default: it reports no capability and fails every operation with
// ErrNoModel so HEAD returns a structured 503 and never fabricates output. It carries
// no weights and makes no network calls.
type NoneBackend struct{}

func (NoneBackend) Name() string { return "none" }

func (NoneBackend) Ready(context.Context) (bool, string) {
	return false, "KNIRV_VLM_BACKEND=none: HEAD returns 503, no vision backend configured"
}

func (NoneBackend) Capabilities() sdk.HeadCapabilities { return sdk.HeadCapabilities{} }

func (NoneBackend) Generate(context.Context, sdk.HeadGenerateRequest) (sdk.HeadGenerateResponse, error) {
	return sdk.HeadGenerateResponse{}, ErrNoModel
}

func (NoneBackend) Vision(context.Context, sdk.VisionRequest) (sdk.VisionResponse, error) {
	return sdk.VisionResponse{}, ErrNoModel
}

func (NoneBackend) PromotedVersion() string { return "" }
