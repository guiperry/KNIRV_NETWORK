package vlm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	sdk "github.com/guiperry/knirv-sdk-go/models"
)

// backendMiniCPM is the HEAD backend that runs MiniCPM-V 4.7 (35B-A3B MoE, Apache-2.0
// code licence) through a llama-server process started with --mmproj. HEAD talks to
// llama-server's OpenAI-compatible /v1/chat/completions endpoint; the server is the
// only component that touches model weights.
const backendMiniCPM = "minicpm"

const (
	minicpmEndpointEnv = "KNIRV_VLM_MINICPM_ENDPOINT"
	minicpmModelEnv    = "KNIRV_VLM_MINICPM_MODEL"
	minicpmVersionEnv  = "KNIRV_VLM_MINICPM_VERSION"

	// minicpmDefaultModel names the quantisation HEAD is provisioned with. It is
	// reported in status; llama-server does not need the name to match.
	minicpmDefaultModel = "MiniCPM-V-4_7-35B-A3B-IQ4_XS"

	// maxImageBytes caps one inline image (base64 payload decoded size). A larger
	// image is rejected before it reaches the model.
	maxImageBytes = 8 << 20
)

// ErrImageRejected is returned when an ImageRef is not an inline data: URI. HEAD never
// lets llama-server fetch a URL, so no request can make the host reach out or read a
// local path through the model.
var ErrImageRejected = errors.New("head minicpm backend accepts only inline data: image URLs")

// ErrOperationUnsupported is returned for a vision operation this backend does not run.
var ErrOperationUnsupported = errors.New("head minicpm backend does not support this operation")

// MiniCPMBackend relays HEAD requests to a llama-server running MiniCPM-V.
type MiniCPMBackend struct {
	endpoint string
	model    string
	version  string
	client   *http.Client
	// MaxTokens bounds one completion. Zero means defaultMiniCPMMaxTokens.
	MaxTokens int
}

const defaultMiniCPMMaxTokens = 512

// NewMiniCPMBackend builds the backend for an endpoint such as http://127.0.0.1:8090.
// An empty endpoint yields an inactive backend that reports not-ready.
func NewMiniCPMBackend(endpoint string) *MiniCPMBackend {
	model := strings.TrimSpace(os.Getenv(minicpmModelEnv))
	if model == "" {
		model = minicpmDefaultModel
	}
	version := strings.TrimSpace(os.Getenv(minicpmVersionEnv))
	if version == "" {
		version = model
	}
	return &MiniCPMBackend{
		endpoint: strings.TrimRight(strings.TrimSpace(endpoint), "/"),
		model:    model,
		version:  version,
		client:   &http.Client{Timeout: 5 * time.Minute},
	}
}

func (b *MiniCPMBackend) Name() string { return backendMiniCPM }

// Ready probes llama-server's /health. A llama-server that is still loading the model
// answers 503, which is reported as not ready rather than as an error.
func (b *MiniCPMBackend) Ready(ctx context.Context) (bool, string) {
	if b.endpoint == "" {
		return false, fmt.Sprintf("%s not set: minicpm backend is inactive (returns 503)", minicpmEndpointEnv)
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, b.endpoint+"/health", nil)
	if err != nil {
		return false, err.Error()
	}
	res, err := b.client.Do(req)
	if err != nil {
		return false, fmt.Sprintf("minicpm llama-server unreachable: %v", err)
	}
	defer res.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(res.Body, 4<<10))
	if res.StatusCode != http.StatusOK {
		return false, fmt.Sprintf("minicpm llama-server not ready: HTTP %d", res.StatusCode)
	}
	return true, ""
}

// Capabilities: MiniCPM-V reads images and writes text. It does not produce embeddings
// through this path, so Embed is false.
func (b *MiniCPMBackend) Capabilities() sdk.HeadCapabilities {
	if b.endpoint == "" {
		return sdk.HeadCapabilities{}
	}
	return sdk.HeadCapabilities{Vision: true, TextMirror: true, Embed: false}
}

// PromotedVersion is the configured version string, or the model name. It is never ""
// while the backend is configured, so HEAD reports which weights it serves.
func (b *MiniCPMBackend) PromotedVersion() string {
	if b.endpoint == "" {
		return ""
	}
	return b.version
}

func (b *MiniCPMBackend) Generate(ctx context.Context, req sdk.HeadGenerateRequest) (sdk.HeadGenerateResponse, error) {
	if b.endpoint == "" {
		return sdk.HeadGenerateResponse{}, ErrNoModel
	}
	parts, err := imageParts(req.Images)
	if err != nil {
		return sdk.HeadGenerateResponse{}, err
	}
	system := strings.TrimSpace(req.PerceptualContext)
	if system == "" {
		system = "You are HEAD, KNIRV's vision and text model. Answer only from the prompt and the images."
	}
	text, err := b.chat(ctx, system, req.Prompt, parts)
	if err != nil {
		return sdk.HeadGenerateResponse{}, err
	}
	return sdk.HeadGenerateResponse{Text: text, Model: backendMiniCPM, Version: b.version}, nil
}

func (b *MiniCPMBackend) Vision(ctx context.Context, req sdk.VisionRequest) (sdk.VisionResponse, error) {
	if b.endpoint == "" {
		return sdk.VisionResponse{}, ErrNoModel
	}
	var instruction string
	switch req.Operation {
	case "describe":
		instruction = "Describe the image content factually. Do not infer anything not visible."
	case "ground":
		instruction = "Locate the object named in the request. Reply with JSON {\"label\": string, \"bbox\": [x0, y0, x1, y1]} in normalised 0-1 coordinates, or {\"label\": null} if it is absent."
	case "visual-reason":
		instruction = "Answer the question using only the images. If the images do not settle it, say so."
	default:
		return sdk.VisionResponse{}, fmt.Errorf("%w: %q", ErrOperationUnsupported, req.Operation)
	}
	parts, err := imageParts(req.Images)
	if err != nil {
		return sdk.VisionResponse{}, err
	}
	user := strings.TrimSpace(req.Text)
	if user == "" {
		user = instruction
	} else {
		user = instruction + "\n\n" + user
	}
	if req.Context != "" {
		user = user + "\n\nContext:\n" + req.Context
	}
	text, err := b.chat(ctx, "You are HEAD's vision model.", user, parts)
	if err != nil {
		return sdk.VisionResponse{}, err
	}
	result, _ := json.Marshal(map[string]string{"text": text})
	return sdk.VisionResponse{
		Operation: req.Operation,
		Model:     backendMiniCPM,
		Version:   b.version,
		Result:    result,
	}, nil
}

// imageParts turns HEAD's ImageRef list into OpenAI-style image_url parts. Only inline
// data: URIs of image types are accepted, and each must decode within maxImageBytes.
func imageParts(images []sdk.ImageRef) ([]map[string]any, error) {
	var parts []map[string]any
	for _, img := range images {
		u := strings.TrimSpace(img.URL)
		if !strings.HasPrefix(u, "data:image/") || !strings.Contains(u, ";base64,") {
			return nil, ErrImageRejected
		}
		payload := u[strings.Index(u, ";base64,")+len(";base64,"):]
		if len(payload) > maxImageBytes*4/3+4 {
			return nil, fmt.Errorf("head minicpm backend: image exceeds %d bytes", maxImageBytes)
		}
		parts = append(parts, map[string]any{
			"type":      "image_url",
			"image_url": map[string]string{"url": u},
		})
	}
	return parts, nil
}

// chat sends one system + user turn with optional images and returns the reply text.
func (b *MiniCPMBackend) chat(ctx context.Context, system, user string, images []map[string]any) (string, error) {
	content := []map[string]any{{"type": "text", "text": user}}
	content = append(content, images...)
	maxTokens := b.MaxTokens
	if maxTokens <= 0 {
		maxTokens = defaultMiniCPMMaxTokens
	}
	body := map[string]any{
		"model": b.model,
		"messages": []map[string]any{
			{"role": "system", "content": system},
			{"role": "user", "content": content},
		},
		// Non-thinking defaults from the MiniCPM-V 4.7 model card.
		"temperature":    0.7,
		"top_p":          0.95,
		"top_k":          100,
		"repeat_penalty": 1.03,
		"max_tokens":     maxTokens,
		"stream":         false,
	}
	raw, _ := json.Marshal(body)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, b.endpoint+"/v1/chat/completions", bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	res, err := b.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("head minicpm backend unreachable: %w", err)
	}
	defer res.Body.Close()
	out, _ := io.ReadAll(io.LimitReader(res.Body, 8<<20))
	if res.StatusCode == http.StatusServiceUnavailable {
		return "", ErrNoModel
	}
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("head minicpm backend: HTTP %d: %s", res.StatusCode, strings.TrimSpace(string(out)))
	}
	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(out, &parsed); err != nil {
		return "", fmt.Errorf("head minicpm backend: bad response: %w", err)
	}
	if len(parsed.Choices) == 0 {
		return "", errors.New("head minicpm backend: response has no choices")
	}
	return strings.TrimSpace(parsed.Choices[0].Message.Content), nil
}
