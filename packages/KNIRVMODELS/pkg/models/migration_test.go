package models

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"knirvmodels/pkg/hashing/transformer"
	"knirvmodels/pkg/vlm"

	sdk "github.com/guiperry/knirv-sdk-go/models"
)

// TestMigrationFlagDefaults pins the Phase 0 defaults: legacy layout, HANDS off,
// VLM backend none, HEART backend qwen. Regression here means a default silently
// flipped and old clients could break.
func TestMigrationFlagDefaults(t *testing.T) {
	if ModelLayout() != ModelLayoutMigrated {
		t.Fatalf("KNIRV_MODEL_LAYOUT default = %q, want %q", ModelLayout(), ModelLayoutMigrated)
	}
	if HandsEnabled() {
		t.Fatal("KNIRV_HANDS_ENABLED default should be off")
	}
	if VLMBackend() != VLMBackendNone {
		t.Fatalf("KNIRV_VLM_BACKEND default = %q, want %q", VLMBackend(), VLMBackendNone)
	}
	if HeartBackend() != HeartBackendQwen {
		t.Fatalf("KNIRV_HEART_BACKEND default = %q, want %q", HeartBackend(), HeartBackendQwen)
	}
	if !MirroredGenerationUnsupportedByDefault(t) {
		t.Fatal("KNIRV_MIRRORED_GENERATION must default off in Phase 0")
	}
	if MirroredTimeoutMS() != 30000 {
		t.Fatalf("mirrored timeout default = %d, want 30000", MirroredTimeoutMS())
	}
	if MirrorTieMargin() != 0.05 {
		t.Fatalf("mirror tie margin default = %v, want 0.05", MirrorTieMargin())
	}
}

func MirroredGenerationUnsupportedByDefault(t *testing.T) bool {
	t.Helper()
	return !MirroredGeneration()
}

// TestHeartBackendHandsAlias pins that KNIRV_HEART_BACKEND=gpt and =hands select
// the same HANDS GPT backend (spec §3.3): both must report HeartBackendIsHands()
// true and only the default (qwen) keeps the Qwen wrapper.
func TestHeartBackendHandsAlias(t *testing.T) {
	for _, v := range []string{"gpt", "hands"} {
		t.Setenv("KNIRV_HEART_BACKEND", v)
		if !HeartBackendIsHands() {
			t.Fatalf("KNIRV_HEART_BACKEND=%s: HeartBackendIsHands() = false, want true", v)
		}
		if got := HeartBackend(); got != v {
			t.Fatalf("KNIRV_HEART_BACKEND=%s: HeartBackend() = %q, want %q", v, got, v)
		}
	}
	t.Setenv("KNIRV_HEART_BACKEND", "qwen")
	if HeartBackendIsHands() {
		t.Fatal("qwen must not be reported as a HANDS backend")
	}
}

// TestCompatibilityMatrixFixture pins the §7.4 compatibility matrix: it must parse,
// list every current route, and its declared flag defaults must match the code. The
// full legacy route/method/status matrix is declared below as a table; the test
// fails on any MISSING entry and on any UNEXPECTED entry, so the contract the code
// must preserve can never silently drift.
func TestCompatibilityMatrixFixture(t *testing.T) {
	path := filepath.Join("compatibility_matrix.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read matrix: %v", err)
	}
	var m struct {
		Flags []struct {
			Name    string `json:"name"`
			Default string `json:"default"`
		} `json:"flags"`
		Routes []struct {
			Method       string `json:"method"`
			LegacyPath   string `json:"legacy_path"`
			MigratedPath string `json:"migrated_path"`
			Status       string `json:"status"`
		} `json:"routes"`
	}
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("parse matrix: %v", err)
	}
	if len(m.Routes) == 0 {
		t.Fatal("matrix lists no routes")
	}

	// The complete legacy §7.4 route/method/status matrix. method + migrated_path
	// uniquely identifies each contract entry; status is asserted too.
	type expectedRoute struct {
		method, legacy, migrated, status string
	}
	want := []expectedRoute{
		{"Head.VisualReason", "", "/models/head/visual-reason", "added"},
		{"Head.Describe", "", "/models/head/describe", "added"},
		{"Head.Ground", "", "/models/head/ground", "added"},
		{"Head.Embed", "", "/models/head/embed", "added"},
		{"Head.Generate", "", "/models/head/generate", "added"},
		{"Heart.Decide", "", "/models/heart/decide", "added"},
		{"Heart.GenerateWithDecision", "", "/models/heart/generate", "added"},
		{"Heart.Generate", "/models/heart/generate", "/models/heart/generate", "unchanged"},
		{"Hands.Plan", "", "/models/hands/plan", "added"},
		{"Hands.Act", "", "/models/hands/act", "added"},
		{"Hands.Status", "", "/models/hands/status", "added"},
		{"Head.Manifest", "/models/head/models", "/models/heart/decide/models", "layout-switch"},
		{"Head.Weights", "/models/head/weights/{ver}", "/models/heart/decide/weights/{ver}", "layout-switch"},
	}

	// Fail on MISSING entries.
	got := make(map[string]expectedRoute, len(m.Routes))
	for _, r := range m.Routes {
		if r.Method == "" {
			t.Fatalf("matrix route with empty method")
		}
		if r.LegacyPath == "" && r.MigratedPath == "" {
			t.Fatalf("route %s has neither legacy nor migrated path", r.Method)
		}
		key := r.Method + "\x00" + r.MigratedPath
		got[key] = expectedRoute{r.Method, r.LegacyPath, r.MigratedPath, r.Status}
	}
	for _, w := range want {
		key := w.method + "\x00" + w.migrated
		have, ok := got[key]
		if !ok {
			t.Errorf("matrix MISSING route %s -> %s (status %s)", w.method, w.migrated, w.status)
			continue
		}
		if have.status != w.status {
			t.Errorf("matrix route %s: status = %q, want %q", w.method, have.status, w.status)
		}
	}
	// Fail on UNEXPECTED entries.
	if len(got) != len(want) {
		for key := range got {
			found := false
			for _, w := range want {
				if w.method+"\x00"+w.migrated == key {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("matrix has UNEXPECTED route entry %q", key)
			}
		}
		t.Fatalf("matrix declares %d routes, want exactly %d (see UNEXPECTED/MISSING above)", len(got), len(want))
	}

	for _, f := range m.Flags {
		switch f.Name {
		case "KNIRV_MODEL_LAYOUT":
			if f.Default != "migrated" {
				t.Errorf("matrix KNIRV_MODEL_LAYOUT default = %q, code default = migrated", f.Default)
			}
		case "KNIRV_HANDS_ENABLED":
			if f.Default != "0" {
				t.Errorf("matrix KNIRV_HANDS_ENABLED default = %q, want 0", f.Default)
			}
		case "KNIRV_VLM_BACKEND":
			if f.Default != "none" {
				t.Errorf("matrix KNIRV_VLM_BACKEND default = %q, want none", f.Default)
			}
		}
	}
}

// TestContractRoutesServedByRealRegistry independently pins the §7.4 contract: it
// builds the real registry and asserts every contract route is actually served (not
// 404). Unlike TestCompatibilityMatrixFixture — which only checks the matrix JSON
// against a list copied from the same file — this test derives the contract from the
// SDK route constants and the real handlers, so a route that vanishes from the code
// (or a matrix entry that misdescribes the code) is caught here.
func TestContractRoutesServedByRealRegistry(t *testing.T) {
	if os.Getenv("KNIRV_SDK_DATA_DIR") == "" {
		t.Fatalf("KNIRV_SDK_DATA_DIR unset and TestMain could not seed the local SDK core")
	}
	t.Setenv("KNIRV_HANDS_ENABLED", "1")
	t.Setenv("KNIRV_HEART_BACKEND", "gpt")
	t.Setenv("KNIRV_VLM_BACKEND", "none")

	reg := buildPhase0Registry(t)
	mux := http.NewServeMux()
	reg.Mount(mux)
	srv := httptest.NewServer(mux)
	defer srv.Close()
	// Trigger HEART's software build so /models/heart/decide/* is mounted.
	postJSON(srv, sdk.HeartPrefix+"/generate", `{"prompt":"hi"}`)

	// method, path, mounted = whether the real registry must serve it (404 = not).
	cases := []struct {
		method  string
		path    string
		mounted bool
	}{
		{"GET", sdk.RegistryPath, true},
		{"GET", sdk.HealthPath, true},
		{"GET", sdk.ManifestPath, true},
		{"GET", sdk.HasherPrefix + "/status", true},
		{"GET", sdk.HasherPrefix + "/ontology", true},
		{"GET", sdk.HasherPrefix + "/index/status", true},
		{"POST", sdk.HasherPrefix + "/validate", true},
		{"POST", sdk.HasherPrefix + "/verify", true},
		{"POST", sdk.HasherPrefix + "/infer", true},
		{"POST", sdk.HasherPrefix + "/index/search", true},
		{"POST", sdk.HasherPrefix + "/ontology/sync", true},
		{"POST", sdk.HasherPrefix + "/ontology/check", true},
		{"GET", sdk.HeartPrefix + "/health", true},
		{"POST", sdk.HeartPrefix + "/generate", true},
		{"POST", sdk.HeartDecisionPrefix + "/reason", true},
		{"GET", sdk.HeartDecisionPrefix + "/models", true},
		{"GET", sdk.HeartDecisionPrefix + "/matrix", true},
		{"GET", sdk.HeadPrefix + "/status", true},
		{"GET", sdk.HeadPrefix + "/capabilities", true},
		{"POST", sdk.HeadPrefix + "/embed", true},
		{"POST", sdk.HeadPrefix + "/describe", true},
		{"POST", sdk.HeadPrefix + "/ground", true},
		{"POST", sdk.HeadPrefix + "/visual-reason", true},
		{"POST", sdk.HeadPrefix + "/generate", true},
		{"GET", sdk.HeadPrefix + "/models", true},
		{"GET", sdk.HeadPrefix + "/weights/", true},
		{"GET", sdk.HandsPrefix + "/status", true},
		{"GET", sdk.HandsPrefix + "/health", true},
		{"GET", sdk.HandsPrefix + "/models", true},
		{"POST", sdk.HandsPrefix + "/generate", true},
		{"POST", sdk.HandsPrefix + "/plan", true},
		{"POST", sdk.HandsPrefix + "/act", true},
		// Legacy head.reason is a client-side alias only; the server does not mount it.
		{"POST", sdk.HeadPrefix + "/reason", false},
	}
	for _, c := range cases {
		var rec *httptest.ResponseRecorder
		if c.method == "GET" {
			rec = getReq(srv, c.path)
		} else {
			rec = postJSON(srv, c.path, `{}`)
		}
		// A route is "not mounted" only when the mux itself answers the
		// default 404; a handler that legitimately answers 404 (e.g. a missing
		// query param) is still mounted.
		mounted := !(rec.Code == http.StatusNotFound && rec.Body.String() == "404 page not found\n")
		if mounted != c.mounted {
			t.Errorf("contract route %s %s: mounted=%v, want %v (status %d)", c.method, c.path, mounted, c.mounted, rec.Code)
		}
	}
}

// buildPhase0Registry wires the four models the daemon serves, matching
// internal/cli/headless/models.go registerModels with small/fast configs so the
// test runs without the production ~2.5 GB seed store. HEART uses the software
// (gpt) backend so its decision core mounts under /models/heart/decide.
func buildPhase0Registry(t *testing.T) *Registry {
	t.Helper()
	reg := NewRegistry()
	hcfg := &transformer.UnifiedConfig{
		VocabSize: 64, EmbedDim: 16, NumHeads: 2, NumLayers: 1, ContextLen: 8,
		Hidden1: 8, Hidden2: 4, OutputSize: 4, FFNHiddenDim: 32, Activation: "hash", Passes: 3,
	}
	reg.Register(NewHasher(func() (*transformer.UnifiedHasherEngine, error) {
		return transformer.NewUnifiedHasherEngineWithConfig(hcfg, transformer.BuildDefaultSeedStore(hcfg), nil, transformer.ModeTransformer), nil
	}))
	reg.Register(NewHeart(func() (*transformer.HEARTService, error) {
		return transformer.NewHEARTService(nil, nil), nil
	}))
	reg.Register(NewHead(vlm.FromEnv()))
	reg.Register(newTestHands())
	return reg
}

// TestRouteContractMounted builds the real registry/daemon and pins that every
// declared legacy §7.4 route is actually served with its expected status and a
// minimal response shape. Routes that depend on a backend intentionally
// unavailable in this config (VLM=none, untrained HANDS/HEART) must still answer
// with a structured 503 (never 404). The HEART decision core (/models/heart/decide/*)
// mounts only after the software build; the test drives that prerequisite and FAILS
// if the decision routes are not mounted. Because mounting HEART needs the SDK core
// wasm (seeded by TestMain), the test fails loudly if the core is unavailable.
func TestRouteContractMounted(t *testing.T) {
	if os.Getenv("KNIRV_SDK_DATA_DIR") == "" {
		t.Fatalf("KNIRV_SDK_DATA_DIR unset and TestMain could not seed the local SDK core; set KNIRV_SDK_DATA_DIR or build KNIRVSDK/dist")
	}
	t.Setenv("KNIRV_HANDS_ENABLED", "1")
	t.Setenv("KNIRV_HEART_BACKEND", "gpt")
	t.Setenv("KNIRV_VLM_BACKEND", "none")

	reg := buildPhase0Registry(t)
	mux := http.NewServeMux()
	reg.Mount(mux)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	// Trigger HEART's software build so /models/heart/decide/* is mounted.
	postJSON(srv, sdk.HeartPrefix+"/generate", `{"prompt":"hi"}`)

	// Each declared route: method + path + the status this config must return.
	// A 503 is the structured "backend unavailable" answer; 404 means the route is
	// not mounted and is a hard failure, except the documented legacy alias
	// head.reason (client-side only). dynamic marks an optional subtree that may
	// legitimately be absent (a 404 there is tolerated, not a failure).
	routes := []struct {
		method  string
		path    string
		want    int
		dynamic bool
	}{
		// registry-level
		{"GET", sdk.RegistryPath, http.StatusOK, false},
		{"GET", sdk.HealthPath, http.StatusOK, false},
		{"GET", sdk.ManifestPath, http.StatusNotImplemented, false},
		// hasher: readiness GETs are 200; infer accepts an empty body and returns
		// 200; index/ontology subsystems are unbuilt in this config (503); ontology
		// check/sync are gated (403/503). These pin the current behavior.
		{"GET", sdk.HasherPrefix + "/status", http.StatusOK, false},
		{"POST", sdk.HasherPrefix + "/validate", http.StatusBadRequest, false},
		{"POST", sdk.HasherPrefix + "/verify", http.StatusBadRequest, false},
		{"POST", sdk.HasherPrefix + "/infer", http.StatusOK, false},
		{"GET", sdk.HasherPrefix + "/index/status", http.StatusServiceUnavailable, false},
		{"POST", sdk.HasherPrefix + "/index/search", http.StatusServiceUnavailable, false},
		{"POST", sdk.HasherPrefix + "/ontology/sync", http.StatusForbidden, false},
		{"GET", sdk.HasherPrefix + "/ontology", http.StatusMethodNotAllowed, false},
		{"POST", sdk.HasherPrefix + "/ontology/check", http.StatusServiceUnavailable, false},
		{"GET", sdk.HasherPrefix + "/verify/jobs/", 0, true},
		// heart (decision core is dynamic, mounts after software build — MUST mount)
		{"GET", sdk.HeartPrefix + "/health", http.StatusOK, false},
		{"POST", sdk.HeartPrefix + "/generate", http.StatusServiceUnavailable, false},
		{"POST", sdk.HeartDecisionPrefix + "/reason", http.StatusServiceUnavailable, false},
		{"GET", sdk.HeartDecisionPrefix + "/models", http.StatusServiceUnavailable, false},
		{"GET", sdk.HeartDecisionPrefix + "/matrix", http.StatusNotFound, false},
		// head (VLM backend none => ops 503, but routes are mounted)
		{"GET", sdk.HeadPrefix + "/status", http.StatusOK, false},
		{"GET", sdk.HeadPrefix + "/capabilities", http.StatusOK, false},
		{"POST", sdk.HeadPrefix + "/embed", http.StatusServiceUnavailable, false},
		{"POST", sdk.HeadPrefix + "/describe", http.StatusServiceUnavailable, false},
		{"POST", sdk.HeadPrefix + "/ground", http.StatusServiceUnavailable, false},
		{"POST", sdk.HeadPrefix + "/visual-reason", http.StatusServiceUnavailable, false},
		{"POST", sdk.HeadPrefix + "/generate", http.StatusServiceUnavailable, false},
		{"GET", sdk.HeadPrefix + "/models", http.StatusOK, false},
		{"GET", sdk.HeadPrefix + "/weights/", http.StatusServiceUnavailable, false},
		// hands (untrained => generate/plan/act 503; status/health/models 200)
		{"GET", sdk.HandsPrefix + "/status", http.StatusOK, false},
		{"GET", sdk.HandsPrefix + "/health", http.StatusOK, false},
		{"GET", sdk.HandsPrefix + "/models", http.StatusOK, false},
		{"POST", sdk.HandsPrefix + "/generate", http.StatusServiceUnavailable, false},
		{"POST", sdk.HandsPrefix + "/plan", http.StatusServiceUnavailable, false},
		{"POST", sdk.HandsPrefix + "/act", http.StatusServiceUnavailable, false},
		{"POST", sdk.HeadPrefix + "/reason", http.StatusNotFound, false},
	}

	for _, r := range routes {
		var rec *httptest.ResponseRecorder
		if r.method == "GET" {
			rec = getReq(srv, r.path)
		} else {
			rec = postJSON(srv, r.path, `{}`)
		}
		if r.want == 0 {
			if rec.Code == http.StatusNotFound {
				t.Logf("optional route %s %s not mounted in this config (documented)", r.method, r.path)
				continue
			}
		} else {
			if rec.Code == http.StatusNotFound && r.want != http.StatusNotFound {
				t.Errorf("declared route %s %s returned 404 (not mounted)", r.method, r.path)
			}
			if rec.Code != r.want {
				t.Errorf("route %s %s: got %d, want %d", r.method, r.path, rec.Code, r.want)
			}
			checkRouteShape(t, rec)
		}
	}
}

// checkRouteShape is the minimal response-shape check per route: a 2xx response
// must be a JSON object, and an error response (400/503) must carry an "error"
// field, so a route can never silently return an empty or non-JSON body.
func checkRouteShape(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	if rec.Code == http.StatusNotFound {
		return // documented removal/alias
	}
	ct := rec.Header().Get("Content-Type")
	if rec.Code >= 200 && rec.Code < 300 {
		if !strings.Contains(ct, "application/json") {
			t.Errorf("route returned %d with content-type %q (want JSON)", rec.Code, ct)
			return
		}
		var v any
		if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil {
			t.Errorf("route returned %d but body is not valid JSON: %v", rec.Code, err)
		}
		return
	}
	if rec.Code == http.StatusBadRequest || rec.Code == http.StatusServiceUnavailable {
		var m map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &m); err != nil || m["error"] == nil {
			t.Errorf("route returned %d but did not carry an \"error\" field", rec.Code)
		}
	}
}

func getReq(srv *httptest.Server, path string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, srv.URL+path, nil)
	srv.Config.Handler.ServeHTTP(rec, req)
	return rec
}

func postJSON(srv *httptest.Server, path, body string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, srv.URL+path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	srv.Config.Handler.ServeHTTP(rec, req)
	return rec
}

// TestCurrentDecisionRoute pins the existing contract: the SDK client's
// token-reasoning call hits /models/heart/decide/reason, and HEART generation hits
// /models/heart/generate.
func TestCurrentDecisionRoute(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if r.URL.Path == sdk.HeartDecisionPrefix+"/reason" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"decisions": []any{}, "model_version": "v"})
			return
		}
		if r.URL.Path == sdk.HeartPrefix+"/generate" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"text": "hi", "ready": true})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := sdk.NewClient(srv.URL, srv.Client())
	if _, err := c.Decide(context.Background(), []uint32{257, 104, 105}); err != nil {
		t.Fatalf("Decide: %v", err)
	}
	if gotPath != sdk.HeartDecisionPrefix+"/reason" {
		t.Fatalf("Decide path = %q, want %q", gotPath, sdk.HeartDecisionPrefix+"/reason")
	}
	if _, err := c.HeartGenerate(context.Background(), "hi"); err != nil {
		t.Fatalf("HeartGenerate: %v", err)
	}
	if gotPath != sdk.HeartPrefix+"/generate" {
		t.Fatalf("HeartGenerate path = %q, want %q", gotPath, sdk.HeartPrefix+"/generate")
	}
}
