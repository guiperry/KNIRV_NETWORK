package models

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"knirvmodels/pkg/heart/decision"
	"knirvmodels/pkg/vlm"

	sdk "github.com/guiperry/knirv-sdk-go/models"
)

// fakeVLM is a test backend with a promoted VLM weight version, so the promoted
// weight contract can be exercised without real weights or the SDK core wasm.
type fakeVLM struct {
	ready bool
	ver   string
}

func (f fakeVLM) Name() string { return "fake" }
func (f fakeVLM) Ready(context.Context) (bool, string) {
	if f.ready {
		return true, ""
	}
	return false, "not ready"
}
func (f fakeVLM) Capabilities() sdk.HeadCapabilities {
	return sdk.HeadCapabilities{Vision: f.ready, Embed: f.ready}
}
func (f fakeVLM) Generate(context.Context, sdk.HeadGenerateRequest) (sdk.HeadGenerateResponse, error) {
	return sdk.HeadGenerateResponse{}, vlm.ErrNoModel
}
func (f fakeVLM) Vision(context.Context, sdk.VisionRequest) (sdk.VisionResponse, error) {
	return sdk.VisionResponse{}, vlm.ErrNoModel
}
func (f fakeVLM) PromotedVersion() string { return f.ver }

// decisionFromGolden builds a decision.Service backed by golden testdata so the
// HEART decision prefix actually serves decision metadata in the test.
func decisionFromGolden(t *testing.T) *decision.Service {
	t.Helper()
	dir := t.TempDir()
	const v = "v3-cafef00d"
	cfg, _ := os.ReadFile("../heart/decision/testdata/golden_config.json")
	var m map[string]any
	_ = json.Unmarshal(cfg, &m)
	m["model_version"] = v
	cfg, _ = json.Marshal(m)
	w, _ := os.ReadFile("../heart/decision/testdata/golden.safetensors")
	_ = os.WriteFile(filepath.Join(dir, "head_config.json"), cfg, 0o644)
	_ = os.WriteFile(filepath.Join(dir, "model-"+v+".safetensors"), w, 0o644)
	_ = os.Symlink("model-"+v+".safetensors", filepath.Join(dir, "model_latest"))
	svc := decision.NewService(dir)
	t.Cleanup(func() { svc.Close(context.Background()) })
	return svc
}

// TestHeadVLMAndDecisionNamespacesDoNotCollide proves FIX 5: after Phase 3,
// /models/head/models and /models/head/weights/{ver} serve HEAD's VLM artefacts,
// while the decision metadata is reachable only behind the HEART decision prefix
// (/models/heart/decide) and never at the HEAD paths.
func TestHeadVLMAndDecisionNamespacesDoNotCollide(t *testing.T) {
	const promoted = "v9-promoted"

	// HEAD VLM facade owns its prefix; the decision core owns only the HEART prefix.
	head := NewHead(fakeVLM{ready: true, ver: promoted})

	mux := http.NewServeMux()
	head.Mount(mux, sdk.HeadPrefix)
	decision.RoutesAt(decisionFromGolden(t), mux, sdk.HeartDecisionPrefix)

	srv := httptest.NewServer(mux)
	defer srv.Close()
	c := http.Client{}

	// 1. /models/head/models returns the VLM manifest, not decision metadata.
	res, err := c.Get(srv.URL + sdk.HeadPrefix + "/models")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != 200 {
		t.Fatalf("HEAD /models -> %d: %s", res.StatusCode, body)
	}
	var hm HeadManifest
	if err := json.Unmarshal(body, &hm); err != nil {
		t.Fatalf("HEAD /models not VLM manifest: %v (%s)", err, body)
	}
	if hm.Kind != "vlm" {
		t.Fatalf("HEAD /models kind = %q, want vlm (decision metadata leaked onto HEAD path)", hm.Kind)
	}
	if hm.PromotedVer != promoted {
		t.Fatalf("HEAD /models promoted_version = %q, want %q", hm.PromotedVer, promoted)
	}

	// 2. /models/head/weights/{promoted} returns the promoted-weight contract.
	res, err = c.Get(srv.URL + sdk.HeadPrefix + "/weights/" + promoted)
	if err != nil {
		t.Fatal(err)
	}
	body, _ = io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != 200 {
		t.Fatalf("HEAD /weights/%s -> %d: %s", promoted, res.StatusCode, body)
	}
	var hw HeadWeights
	if err := json.Unmarshal(body, &hw); err != nil {
		t.Fatalf("HEAD /weights not promoted-weight contract: %v (%s)", err, body)
	}
	if hw.Kind != "vlm-promoted-weights" || !hw.Available || hw.Version != promoted {
		t.Fatalf("HEAD /weights contract wrong: %+v", hw)
	}

	// 3. An unknown promoted version is rejected (404), never served as decision.
	res, _ = c.Get(srv.URL + sdk.HeadPrefix + "/weights/nope")
	res.Body.Close()
	if res.StatusCode != 404 {
		t.Fatalf("HEAD /weights/nope -> %d, want 404", res.StatusCode)
	}

	// 4. Decision metadata is reachable only via the HEART decision prefix.
	res, err = c.Get(srv.URL + sdk.HeartDecisionPrefix + "/models")
	if err != nil {
		t.Fatal(err)
	}
	body, _ = io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != 200 {
		t.Fatalf("HEART decision /models -> %d: %s", res.StatusCode, body)
	}
	var dm decision.Manifest
	if err := json.Unmarshal(body, &dm); err != nil {
		t.Fatalf("HEART decision /models not decision manifest: %v (%s)", err, body)
	}
	if dm.Version != "v3-cafef00d" {
		t.Fatalf("HEART decision /models version = %q, want %q", dm.Version, "v3-cafef00d")
	}
}
