// Package vlm is HEAD's vision / multimodal backend after the migration. HEAD is no
// longer the decision core (that moved to HEART); it is the VLM and the mirrored text
// generator. A Backend turns the SDK VLM contract (HEADStatus / HeadGenerate /
// HeadVision) into a concrete model source.
//
// Owner-level decisions (which weights, which sidecar) are deferred (M2). Until then
// only the "none" backend ships: it reports no capability and returns a structured
// 503 on every operation, so HEAD never fabricates output. The onnx / llama / openai
// backends are provisional scaffolds that probe a configured endpoint and otherwise
// behave like "none"; they do not bundle or download any weights.
package vlm

import (
	"context"
	"errors"

	sdk "github.com/guiperry/knirv-sdk-go/models"
)

// ErrNoModel is returned by every operation when no backend is configured or the
// configured backend is unavailable. The daemon maps it to HTTP 503.
var ErrNoModel = errors.New("no HEAD vision backend is configured")

// Backend is one vision / mirror-text source HEAD can serve.
type Backend interface {
	// Name is the backend id ("none", "onnx", "llama", "openai") for status.
	Name() string
	// Ready reports whether the backend can answer now and why not if not.
	Ready(ctx context.Context) (bool, string)
	// Capabilities reports what the backend can do.
	Capabilities() sdk.HeadCapabilities
	// Generate runs mirrored text generation (POST /models/head/generate).
	Generate(ctx context.Context, req sdk.HeadGenerateRequest) (sdk.HeadGenerateResponse, error)
	// Vision runs one operation: embed / describe / ground / visual-reason.
	Vision(ctx context.Context, req sdk.VisionRequest) (sdk.VisionResponse, error)
	// PromotedVersion returns the version string of the promoted VLM weight set
	// HEAD currently serves at /models/head/weights/{ver}, or "" when HEAD has no
	// promoted weights (the default "none" backend, which never fabricates
	// output). It mirrors the decision service's model_version contract for HEAD's
	// own artefacts after Phase 3.
	PromotedVersion() string
}

// Status returns the backend's status for GET /models/head/status.
func Status(b Backend, ctx context.Context) sdk.HeadStatus {
	ready, detail := b.Ready(ctx)
	return sdk.HeadStatus{
		Backend:      b.Name(),
		Ready:        ready,
		Capabilities: b.Capabilities(),
		Detail:       detail,
	}
}
