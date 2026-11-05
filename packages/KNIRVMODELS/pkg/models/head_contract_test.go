package models

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"knirvmodels/pkg/vlm"

	sdk "github.com/guiperry/knirv-sdk-go/models"
)

// contractFakeVLM is a test backend that returns a synthetic VisionResponse per operation
// (or vlm.ErrNoModel when not ready, exercising the vlm=none 503 path through the
// real mounted Head).
type contractFakeVLM struct {
	ready bool
	calls map[string]int
	model string
}

func (f *contractFakeVLM) Name() string { return "fake" }
func (f *contractFakeVLM) Ready(ctx context.Context) (bool, string) {
	if f.ready {
		return true, ""
	}
	return false, "no backend"
}
func (f *contractFakeVLM) Capabilities() sdk.HeadCapabilities {
	return sdk.HeadCapabilities{Vision: true, TextMirror: true, Embed: true}
}
func (f *contractFakeVLM) Generate(ctx context.Context, req sdk.HeadGenerateRequest) (sdk.HeadGenerateResponse, error) {
	if !f.ready {
		return sdk.HeadGenerateResponse{}, vlm.ErrNoModel
	}
	return sdk.HeadGenerateResponse{Text: "gen", Model: "fake", Version: "1"}, nil
}
func (f *contractFakeVLM) Vision(ctx context.Context, req sdk.VisionRequest) (sdk.VisionResponse, error) {
	if !f.ready {
		return sdk.VisionResponse{}, vlm.ErrNoModel
	}
	f.calls[req.Operation]++
	return sdk.VisionResponse{Operation: req.Operation, Model: f.model, Result: []byte(`{"ok":true}`)}, nil
}
func (f *contractFakeVLM) PromotedVersion() string { return "" }

// TestHeadVisionContract drives the real SDK Client.HeadVision against the real
// mounted KNIRVMODELS Head (no hand-written fake mux): every supported operation
// must reach its own route and return the operation-tagged response. The prior
// SDK-side test stood in a fake mux; this pins the client<->server contract end to
// end.
func TestHeadVisionContract(t *testing.T) {
	backend := &contractFakeVLM{ready: true, calls: map[string]int{}, model: "fake"}
	h := NewHead(backend)
	mux := http.NewServeMux()
	h.Mount(mux, "/models/head")
	srv := httptest.NewServer(mux)
	defer srv.Close()
	c := sdk.NewClient(srv.URL, nil)

	for _, op := range []string{"embed", "describe", "ground", "visual-reason"} {
		res, err := c.HeadVision(context.Background(), sdk.VisionRequest{Operation: op, Text: "x"})
		if err != nil {
			t.Fatalf("HeadVision(%q): %v", op, err)
		}
		if res.Operation != op {
			t.Fatalf("HeadVision(%q): response operation = %q", op, res.Operation)
		}
		if backend.calls[op] != 1 {
			t.Fatalf("HeadVision(%q): backend invoked %d times, want 1", op, backend.calls[op])
		}
	}

	if _, err := c.HeadVision(context.Background(), sdk.VisionRequest{Operation: "hallucinate"}); err == nil {
		t.Fatal("HeadVision(unknown): expected client-side rejection, got nil")
	}
}

// TestHeadVisionNoneReturns503 pins the vlm=none contract from the real server: with
// no backend, every HEAD route returns 503 (vlm.ErrNoModel), so HEAD never
// fabricates output.
func TestHeadVisionNoneReturns503(t *testing.T) {
	backend := &contractFakeVLM{ready: false}
	h := NewHead(backend)
	mux := http.NewServeMux()
	h.Mount(mux, "/models/head")
	srv := httptest.NewServer(mux)
	defer srv.Close()
	c := sdk.NewClient(srv.URL, nil)

	for _, op := range []string{"embed", "describe", "ground", "visual-reason"} {
		if _, err := c.HeadVision(context.Background(), sdk.VisionRequest{Operation: op, Text: "x"}); err == nil {
			t.Fatalf("HeadVision(%q) with no backend: expected error, got nil", op)
		}
	}
}
