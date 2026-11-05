package vlm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	sdk "github.com/guiperry/knirv-sdk-go/models"
)

// VLM backend names (mirrors pkg/models flags; duplicated here so pkg/vlm does not
// depend on pkg/models, which is edited concurrently by other agents).
const (
	backendNone   = "none"
	backendLLAMA  = "llama"
	backendONNX   = "onnx"
	backendOpenAI = "openai"
)

// vlmBackendEnv is KNIRV_VLM_BACKEND.
const vlmBackendEnv = "KNIRV_VLM_BACKEND"

// FromEnv builds the HEAD backend named by KNIRV_VLM_BACKEND (default "none").
//
// The onnx / llama / openai backends are provisional scaffolds (M1): they talk to a
// sidecar over HTTP only when its endpoint env var is set (KNIRV_VLM_ONNX_ENDPOINT,
// KNIRV_VLM_LLAMA_ENDPOINT, KNIRV_LLAMA_SOCKET / KNIRV_VLM_OPENAI_*). When unset they
// behave exactly like "none" and return ErrNoModel, so HEAD never fabricates output
// and never downloads or bundles weights (M2 is the owner's call).
func FromEnv() Backend {
	return New(os.Getenv(vlmBackendEnv))
}

// New builds the named backend. Unknown names fall back to NoneBackend.
func New(name string) Backend {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "", backendNone:
		return NoneBackend{}
	case backendONNX:
		return newHTTPBackend(backendONNX, endpointEnv("KNIRV_VLM_ONNX_ENDPOINT"))
	case backendLLAMA:
		return newHTTPBackend(backendLLAMA, endpointEnv("KNIRV_VLM_LLAMA_ENDPOINT"))
	case backendOpenAI:
		return newOpenAIBackend()
	default:
		return NoneBackend{}
	}
}

// endpointEnv returns the configured sidecar endpoint, or "" if none.
func endpointEnv(key string) string {
	return strings.TrimSpace(os.Getenv(key))
}

// HTTPBackend is a provisional sidecar-backed backend. With no endpoint it is "none"
// in all but name; with an endpoint it relays Generate/Vision over HTTP. It does not
// own or bundle weights.
type HTTPBackend struct {
	name     string
	endpoint string
	client   *http.Client
}

func newHTTPBackend(name, endpoint string) *HTTPBackend {
	return &HTTPBackend{
		name:     name,
		endpoint: endpoint,
		client:   &http.Client{Timeout: 60 * time.Second},
	}
}

func (b *HTTPBackend) Name() string { return b.name }

func (b *HTTPBackend) Ready(context.Context) (bool, string) {
	if b.endpoint == "" {
		return false, fmt.Sprintf("KNIRV_VLM_%s_ENDPOINT not set: %s backend is inactive (returns 503)", strings.ToUpper(b.name), b.name)
	}
	return true, ""
}

func (b *HTTPBackend) Capabilities() sdk.HeadCapabilities {
	if b.endpoint == "" {
		return sdk.HeadCapabilities{}
	}
	return sdk.HeadCapabilities{Vision: true, TextMirror: true, Embed: true}
}

func (b *HTTPBackend) Generate(ctx context.Context, req sdk.HeadGenerateRequest) (sdk.HeadGenerateResponse, error) {
	if b.endpoint == "" {
		return sdk.HeadGenerateResponse{}, ErrNoModel
	}
	var out sdk.HeadGenerateResponse
	if err := b.post(ctx, b.endpoint+"/generate", req, &out); err != nil {
		return sdk.HeadGenerateResponse{}, err
	}
	return out, nil
}

func (b *HTTPBackend) Vision(ctx context.Context, req sdk.VisionRequest) (sdk.VisionResponse, error) {
	if b.endpoint == "" {
		return sdk.VisionResponse{}, ErrNoModel
	}
	var out sdk.VisionResponse
	if err := b.post(ctx, b.endpoint+"/vision", req, &out); err != nil {
		return sdk.VisionResponse{}, err
	}
	return out, nil
}

// PromotedVersion is "" for the sidecar backends: they relay to a sidecar and do
// not bundle or own HEAD's promoted weights (that is the owner's M2 call).
func (b *HTTPBackend) PromotedVersion() string { return "" }

func (b *HTTPBackend) post(ctx context.Context, url string, in, out any) error {
	body, _ := json.Marshal(in)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := b.client.Do(req)
	if err != nil {
		return fmt.Errorf("head %s backend unreachable: %w", b.name, err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(res.Body, 8<<20))
	if res.StatusCode == http.StatusServiceUnavailable {
		return ErrNoModel
	}
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("head %s backend: HTTP %d: %s", b.name, res.StatusCode, strings.TrimSpace(string(raw)))
	}
	return json.Unmarshal(raw, out)
}

// openAIBackend is the opt-in external backend (M1). It is explicitly off by default
// and must be configured with an endpoint and key; it is the only backend that may
// send image data off the host, and the caller is warned to do so deliberately.
type openAIBackend struct {
	*HTTPBackend
}

func newOpenAIBackend() Backend {
	ep := endpointEnv("KNIRV_VLM_OPENAI_ENDPOINT")
	if ep == "" {
		ep = endpointEnv("KNIRV_VLM_OPENAI_URL")
	}
	b := newHTTPBackend("openai", ep)
	return &openAIBackend{HTTPBackend: b}
}

func (b *openAIBackend) Ready(ctx context.Context) (bool, string) {
	if b.endpoint == "" {
		return false, "KNIRV_VLM_OPENAI_ENDPOINT not set: openai backend is off by default (data leaves the host)"
	}
	return true, "openai backend is external: image data may leave the host"
}
