package models

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"knirvmodels/pkg/vlm"

	sdk "github.com/guiperry/knirv-sdk-go/models"
)

// textMirrorFakeBackend is an in-process active backend. It exists only to prove the
// text_mirror capability is reported when a backend is active; it runs no model,
// starts no process and makes no network call.
type textMirrorFakeBackend struct{}

func (textMirrorFakeBackend) Name() string { return "fake" }
func (textMirrorFakeBackend) Ready(context.Context) (bool, string) {
	return true, ""
}
func (textMirrorFakeBackend) Capabilities() sdk.HeadCapabilities {
	return sdk.HeadCapabilities{Vision: true, TextMirror: true, Embed: true}
}
func (textMirrorFakeBackend) Generate(context.Context, sdk.HeadGenerateRequest) (sdk.HeadGenerateResponse, error) {
	return sdk.HeadGenerateResponse{Text: "fake", Model: "fake", Version: "1"}, nil
}
func (textMirrorFakeBackend) Vision(context.Context, sdk.VisionRequest) (sdk.VisionResponse, error) {
	return sdk.VisionResponse{}, nil
}
func (textMirrorFakeBackend) PromotedVersion() string { return "" }

// getCapabilitiesJSON fetches /models/head/capabilities from a mounted Head and
// decodes it into a generic map, so the wire key "text_mirror" itself is asserted.
func getCapabilitiesJSON(t *testing.T, backend vlm.Backend) map[string]any {
	t.Helper()
	mux := http.NewServeMux()
	NewHead(backend).Mount(mux, "/models/head")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/models/head/capabilities", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /models/head/capabilities: status %d", rec.Code)
	}
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("capabilities body: %v (%q)", err, rec.Body.String())
	}
	return out
}

// TestHeadTextMirrorCapabilityAbsentUnderNone: with the default "none" backend the
// text_mirror capability must be absent (false) on both the capabilities route and
// the status route's embedded capabilities.
func TestHeadTextMirrorCapabilityAbsentUnderNone(t *testing.T) {
	caps := getCapabilitiesJSON(t, vlm.NoneBackend{})
	if v, _ := caps["text_mirror"].(bool); v {
		t.Fatalf("text_mirror reported under none backend: %v", caps)
	}

	status := NewHead(vlm.NoneBackend{}).Status()
	var detail sdk.HeadCapabilities
	if err := json.Unmarshal(status.Detail, &detail); err != nil {
		t.Fatalf("status detail: %v", err)
	}
	if detail.TextMirror {
		t.Fatalf("status reports TextMirror under none backend: %+v", detail)
	}
}

// TestHeadTextMirrorCapabilityPresentWhenBackendActive: when an active backend is
// configured, text_mirror is reported true on the capabilities route.
func TestHeadTextMirrorCapabilityPresentWhenBackendActive(t *testing.T) {
	caps := getCapabilitiesJSON(t, textMirrorFakeBackend{})
	if v, _ := caps["text_mirror"].(bool); !v {
		t.Fatalf("text_mirror not reported for active backend: %v", caps)
	}
}
