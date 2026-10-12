package models

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"knirvmodels/pkg/hashing/transformer"

	sdk "github.com/guiperry/knirv-sdk-go/models"
)

func do(mux *http.ServeMux, method, path, body string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(method, path, strings.NewReader(body)))
	return rec
}

func newMux(t *testing.T, build func() (*transformer.HEARTService, error)) (*http.ServeMux, *Heart) {
	t.Helper()
	reg := NewRegistry()
	h := NewHeart(build)
	reg.Register(h)
	reg.Register(NewHasher(func() (*transformer.UnifiedHasherEngine, error) {
		// Small: the production default is a ~2.5 GB seed store.
		cfg := &transformer.UnifiedConfig{VocabSize: 64, EmbedDim: 16, NumHeads: 2, NumLayers: 1, ContextLen: 8, Hidden1: 8, Hidden2: 4, OutputSize: 4, FFNHiddenDim: 32, Activation: "hash", Passes: 3}
		return transformer.NewUnifiedHasherEngineWithConfig(cfg, transformer.BuildDefaultSeedStore(cfg), nil, transformer.ModeTransformer), nil
	}))
	mux := http.NewServeMux()
	reg.Mount(mux)
	return mux, h
}

func TestHeartIsLazyAndHealthNeverBuildsIt(t *testing.T) {
	var builds atomic.Int32
	mux, _ := newMux(t, func() (*transformer.HEARTService, error) {
		builds.Add(1)
		return transformer.NewHEARTService(nil, nil), nil
	})
	for i := 0; i < 3; i++ {
		if rec := do(mux, "GET", sdk.HeartPrefix+"/health", ""); rec.Code != 200 || !strings.Contains(rec.Body.String(), `"loaded":false`) {
			t.Fatalf("idle health: %d %s", rec.Code, rec.Body)
		}
		do(mux, "GET", sdk.RegistryPath, "")
	}
	if builds.Load() != 0 {
		t.Fatalf("health/registry built HEART %d times", builds.Load())
	}
	// A real request builds it once; with no trained generator it is a 503, not a crash.
	for i := 0; i < 3; i++ {
		rec := do(mux, "POST", sdk.HeartPrefix+"/generate", `{"prompt":"hi"}`)
		if rec.Code != http.StatusServiceUnavailable || !strings.Contains(rec.Body.String(), `"ready":false`) {
			t.Fatalf("generate: %d %s", rec.Code, rec.Body)
		}
	}
	if builds.Load() != 1 {
		t.Fatalf("HEART built %d times, want 1", builds.Load())
	}
	rec := do(mux, "GET", sdk.HeartPrefix+"/health", "")
	if !strings.Contains(rec.Body.String(), `"loaded":true`) {
		t.Fatalf("loaded health: %s", rec.Body)
	}
	var rows []sdk.ModelStatus
	_ = json.Unmarshal(do(mux, "GET", sdk.RegistryPath, "").Body.Bytes(), &rows)
	for _, r := range rows {
		if r.Name == "heart" && (!r.Loaded || r.Since == "") {
			t.Fatalf("heart row %+v", r)
		}
	}
}

func TestHeartBuildFailureIs503AndBackedOff(t *testing.T) {
	var builds atomic.Int32
	mux, _ := newMux(t, func() (*transformer.HEARTService, error) {
		builds.Add(1)
		return nil, errors.New("tokenizer missing")
	})
	for i := 0; i < 4; i++ {
		rec := do(mux, "POST", sdk.HeartPrefix+"/generate", `{"prompt":"x"}`)
		if rec.Code != http.StatusServiceUnavailable || !strings.Contains(rec.Body.String(), "tokenizer missing") {
			t.Fatalf("%d %s", rec.Code, rec.Body)
		}
	}
	if builds.Load() != 1 {
		t.Fatalf("a failing build was retried %d times inside the back-off window", builds.Load())
	}
	var rows []sdk.ModelStatus
	_ = json.Unmarshal(do(mux, "GET", sdk.RegistryPath, "").Body.Bytes(), &rows)
	for _, r := range rows {
		if r.Name == "heart" && (r.Ready || r.Error == "") {
			t.Fatalf("a failed HEART must show its error: %+v", r)
		}
	}
}

func TestRegistryIsSortedAndMatchesTheSharedFixtureShape(t *testing.T) {
	mux, _ := newMux(t, func() (*transformer.HEARTService, error) { return nil, errors.New("unused") })
	raw := do(mux, "GET", sdk.RegistryPath, "").Body.Bytes()
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.DisallowUnknownFields()
	var rows []sdk.ModelStatus
	if err := dec.Decode(&rows); err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || rows[0].Name != "hasher" || rows[1].Name != "heart" {
		t.Fatalf("%+v", rows)
	}
	// The same decoder accepts the fixture the SDK's Go and TS tests use.
	fixture, err := os.ReadFile(filepath.Join("..", "..", "..", "KNIRVSDK", "testvectors", "models", "registry.json"))
	if err != nil {
		t.Fatal(err)
	}
	var fx []sdk.ModelStatus
	if err := json.Unmarshal(fixture, &fx); err != nil || len(fx) != 3 {
		t.Fatalf("fixture %v %v", fx, err)
	}
}

func TestHasherValidateInferStatus(t *testing.T) {
	mux, _ := newMux(t, func() (*transformer.HEARTService, error) { return nil, errors.New("unused") })
	if rec := do(mux, "GET", sdk.HasherPrefix+"/status", ""); !strings.Contains(rec.Body.String(), `"loaded":false`) {
		t.Fatalf("status must not build the engine: %s", rec.Body)
	}
	if rec := do(mux, "POST", sdk.HasherPrefix+"/validate", `{"prediction":1}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("domain is required: %d", rec.Code)
	}
	rec := do(mux, "POST", sdk.HasherPrefix+"/validate", `{"prediction":1,"domain":"nope"}`)
	var v sdk.HasherValidateResult
	if rec.Code != 200 || json.Unmarshal(rec.Body.Bytes(), &v) != nil || v.Domain != "nope" {
		t.Fatalf("validate %d %s", rec.Code, rec.Body)
	}
	if rec := do(mux, "GET", sdk.HasherPrefix+"/infer", ""); rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET infer: %d", rec.Code)
	}
	done := make(chan *httptest.ResponseRecorder, 1)
	go func() { done <- do(mux, "POST", sdk.HasherPrefix+"/infer", `{"text":"x"}`) }()
	select {
	case rec := <-done:
		var r sdk.HasherInferResult
		if rec.Code != 200 || json.Unmarshal(rec.Body.Bytes(), &r) != nil || r.Mode != "transformer" || r.TotalPasses != 1 {
			t.Fatalf("infer %d %s", rec.Code, rec.Body)
		}
	case <-time.After(2 * time.Minute):
		t.Fatal("infer did not return")
	}
	if rec := do(mux, "GET", sdk.HasherPrefix+"/status", ""); !strings.Contains(rec.Body.String(), `"loaded":true`) {
		t.Fatalf("status after infer: %s", rec.Body)
	}
}
