package headless

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"knirvmodels/internal/policy"
	handspkg "knirvmodels/pkg/hands"
	"knirvmodels/pkg/hands/gpt"
	"knirvmodels/pkg/hashing/transformer"
	"knirvmodels/pkg/models"

	sdk "github.com/guiperry/knirv-sdk-go/models"
)

// tinyHandsConfig keeps the engine small so the HANDS wiring test runs without the
// full GPT-2 (~2.5 GB) the production constructor builds.
func tinyHandsConfig() *gpt.Config {
	return &gpt.Config{
		VocabSize: 16, EmbedDim: 8, NumHeads: 2, NumLayers: 1,
		ContextLen: 16, FFNHiddenDim: 16, DecayAlpha: 1.0,
	}
}

// withHandsEnv sets the environment the HANDS wiring reads, on a temp tree so the
// test never touches real audit / checkpoint / app-data locations.
func withHandsEnv(t *testing.T, handsEnabled bool) {
	t.Helper()
	t.Setenv("KNIRV_APP_DATA_DIR", t.TempDir())
	t.Setenv("KNIRV_AUDIT_DIR", t.TempDir())
	t.Setenv("KNIRV_HANDS_DIR", t.TempDir())
	if handsEnabled {
		t.Setenv("KNIRV_HANDS_ENABLED", "1")
	} else {
		t.Setenv("KNIRV_HANDS_ENABLED", "0")
	}
}

// overrideHandsConfig swaps the production engine config (gpt.DefaultConfig) for a
// tiny one so the default (engineBuild == nil) constructor path is exercised
// cheaply, then restores it.
func overrideHandsConfig(t *testing.T, cfg *gpt.Config) {
	t.Helper()
	old := handsEngineConfig
	handsEngineConfig = func() *gpt.Config { return cfg }
	t.Cleanup(func() { handsEngineConfig = old })
}

// scriptedHandsGen is a generator that returns a fixed sequence of framed outputs,
// so a T0 tool call can be driven deterministically through the real gated loop
// (gate + audit + daemon tool set) without an untrained model.
type scriptedHandsGen struct {
	ready   bool
	outputs []string
	i       int
}

func (g *scriptedHandsGen) GenerateText(prompt string, n int, temperature float32, topK int) (string, error) {
	if g.i >= len(g.outputs) {
		return `{"action":"final","output":"done"}`, nil
	}
	s := g.outputs[g.i]
	g.i++
	return s, nil
}
func (g *scriptedHandsGen) IsReady() bool     { return g.ready }
func (g *scriptedHandsGen) Path() string      { return "test" }
func (g *scriptedHandsGen) Progress() float64 { return 1 }

// TestHandsRegisteredWhenEnabled exercises the REAL production constructor path
// (engineBuild == nil -> loadHandsEngine with HandsDir()/model-latest): a
// persistent audit log, the daemon tool set, the policy gate, the default frame,
// and the checkpoint load. With a checkpoint present the engine is ready and
// /models/hands/generate answers 200; the audit log directory is created and the
// hash chain verifies.
func TestHandsRegisteredWhenEnabled(t *testing.T) {
	withHandsEnv(t, true)
	overrideHandsConfig(t, tinyHandsConfig())
	s := NewServer(t.TempDir()+"/h.sock", 0o660)

	// Publish a checkpoint so the production load path (HandsDir()/model-latest)
	// flips the engine to ready.
	cp := filepath.Join(models.HandsDir(), "model-latest")
	saved := handspkg.NewEngine(tinyHandsConfig(), "")
	if err := saved.Save(cp); err != nil {
		t.Fatalf("save checkpoint: %v", err)
	}

	// engineBuild == nil -> production default constructor (loadHandsEngine).
	reg := models.NewRegistry()
	hasher := models.NewHasher(nil)
	fin := transformer.NewWASMFinisher(heartConfigFromEnv())
	h := s.buildHandsWith(reg, hasher, fin, nil)
	if h == nil {
		t.Fatal("buildHandsWith(nil) returned nil with HANDS enabled and a writable audit dir")
	}
	reg.Register(h)

	// The registry reports HANDS as the server_tasks role.
	roles := map[string]string{}
	for _, st := range reg.Statuses() {
		roles[st.Name] = st.Role
	}
	if roles["hands"] != sdk.RoleHands {
		t.Fatalf("registry hands role = %q, want %q", roles["hands"], sdk.RoleHands)
	}

	// Checkpoint present -> generate is ready (200).
	mux := http.NewServeMux()
	h.Mount(mux, sdk.HandsPrefix)
	srv := httptest.NewServer(mux)
	defer srv.Close()
	rec := httptest.NewRecorder()
	srv.Config.Handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, sdk.HandsPrefix+"/generate", strings.NewReader(`{"prompt":"hi","n":4}`)))
	if rec.Code != http.StatusOK {
		t.Fatalf("/models/hands/generate: got %d, want 200 (checkpoint present)", rec.Code)
	}

	// The persistent audit log directory was created by OpenAudit; the chain
	// verifies (valid even when empty).
	if fi, err := os.Stat(auditDir()); err != nil || !fi.IsDir() {
		t.Fatalf("audit dir not created at %s: %v", auditDir(), err)
	}
	auditPath := filepath.Join(auditDir(), "hands_tool_audit.jsonl")
	if _, _, err := policy.VerifyChain(auditPath); err != nil {
		t.Fatalf("audit chain invalid: %v", err)
	}
}

// TestHandsNoCheckpointReturns503 pins the production behavior: with no checkpoint
// the engine stays untrained and every trained-path route returns 503 (no training
// is triggered by the daemon). The audit log is still created. Exercises the
// engineBuild == nil default path with HandsDir empty of a checkpoint.
func TestHandsNoCheckpointReturns503(t *testing.T) {
	withHandsEnv(t, true)
	overrideHandsConfig(t, tinyHandsConfig())
	s := NewServer(t.TempDir()+"/h.sock", 0o660)
	// KNIRV_HANDS_DIR points at a fresh temp dir with no model-latest.

	// engineBuild == nil -> production default constructor with no checkpoint.
	reg := models.NewRegistry()
	hasher := models.NewHasher(nil)
	fin := transformer.NewWASMFinisher(heartConfigFromEnv())
	h := s.buildHandsWith(reg, hasher, fin, nil)
	if h == nil {
		t.Fatal("buildHandsWith(nil) returned nil")
	}
	reg.Register(h)

	mux := http.NewServeMux()
	h.Mount(mux, sdk.HandsPrefix)
	for _, p := range []string{sdk.HandsPrefix + "/generate", sdk.HandsPrefix + "/plan", sdk.HandsPrefix + "/act"} {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, p, strings.NewReader(`{"prompt":"hi"}`)))
		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("%s: got %d, want 503 (untrained, no checkpoint)", p, rec.Code)
		}
	}

	// The audit log directory is still created even when HANDS is untrained.
	if fi, err := os.Stat(auditDir()); err != nil || !fi.IsDir() {
		t.Fatalf("audit dir not created: %v", err)
	}
}

// TestHandsDisabledWhenAuditUnavailable pins that HANDS returns nil (unregistered)
// rather than serving unaudited when the audit log cannot be opened.
func TestHandsDisabledWhenAuditUnavailable(t *testing.T) {
	withHandsEnv(t, true)
	// Put KNIRV_AUDIT_DIR under an existing regular file so OpenAudit's
	// MkdirAll(filepath.Dir(path)) fails and the audit cannot be opened.
	blocker := filepath.Join(t.TempDir(), "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("KNIRV_AUDIT_DIR", filepath.Join(blocker, "hands_tool_audit.jsonl"))
	s := NewServer(t.TempDir()+"/h.sock", 0o660)
	reg := models.NewRegistry()
	hasher := models.NewHasher(nil)
	fin := transformer.NewWASMFinisher(heartConfigFromEnv())
	if h := s.buildHandsWith(reg, hasher, fin, nil); h != nil {
		t.Fatal("buildHandsWith should return nil when the audit log cannot be opened")
	}
	for _, st := range reg.Statuses() {
		if st.Name == "hands" {
			t.Fatal("hands registered despite unauditable")
		}
	}
}

// TestLoadHandsEngine pins the production checkpoint loader that the default
// (engineBuild == nil) constructor uses: absent -> untrained engine (no error),
// present -> ready engine, corrupt -> load error (not a silent fallback).
func TestLoadHandsEngine(t *testing.T) {
	cfg := tinyHandsConfig()

	t.Run("absent", func(t *testing.T) {
		dir := t.TempDir() // no model-latest
		eng, err := loadHandsEngine(dir, cfg)
		if err != nil {
			t.Fatalf("absent: loadHandsEngine returned error: %v", err)
		}
		if eng.IsReady() {
			t.Fatal("absent checkpoint should leave the engine untrained")
		}
	})

	t.Run("present", func(t *testing.T) {
		dir := t.TempDir()
		cp := filepath.Join(dir, "model-latest")
		saved := handspkg.NewEngine(cfg, "")
		if err := saved.Save(cp); err != nil {
			t.Fatalf("save: %v", err)
		}
		eng, err := loadHandsEngine(dir, cfg)
		if err != nil {
			t.Fatalf("present: loadHandsEngine returned error: %v", err)
		}
		if !eng.IsReady() {
			t.Fatal("present checkpoint should leave the engine ready")
		}
	})

	t.Run("corrupt", func(t *testing.T) {
		dir := t.TempDir()
		cp := filepath.Join(dir, "model-latest")
		if err := os.WriteFile(cp, []byte("not a model"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := loadHandsEngine(dir, cfg); err == nil {
			t.Fatal("corrupt checkpoint should be a load error")
		}
	})
}

// TestHandsActRecordsAuditEntry drives a REAL core-authorized T0 tool call (manifest.get)
// through the headless /models/hands/act path using a scripted generator, and proves
// the decision is written to the audit chain: policy.VerifyChain returns >=1 entry
// naming the tool and the execute decision. This is the headless-level counterpart of
// the pkg/models unit test, exercising the daemon's actual gate + audit + tool wiring
// (not a vacuous empty-chain check).
func TestHandsActRecordsAuditEntry(t *testing.T) {
	withHandsEnv(t, true)
	s := NewServer(t.TempDir()+"/h.sock", 0o660)

	reg := models.NewRegistry()
	hasher := models.NewHasher(nil)
	fin := transformer.NewWASMFinisher(heartConfigFromEnv())

	// Scripted generator: first emit a T0 tool call (manifest.get), then finish.
	gen := &scriptedHandsGen{
		ready: true,
		outputs: []string{
			`{"action":"tool","tool":"manifest.get","args":{}}`,
			`{"action":"final","output":"done"}`,
		},
	}
	h := s.buildHandsWith(reg, hasher, fin, func() (models.Generator, error) { return gen, nil })
	if h == nil {
		t.Fatal("buildHandsWith returned nil")
	}
	reg.Register(h)

	mux := http.NewServeMux()
	h.Mount(mux, sdk.HandsPrefix)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	rec := httptest.NewRecorder()
	srv.Config.Handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, sdk.HandsPrefix+"/act", strings.NewReader(`{"prompt":"show manifest"}`)))
	if rec.Code != http.StatusOK {
		t.Fatalf("/models/hands/act: got %d, want 200; body=%s", rec.Code, rec.Body.String())
	}

	// The gated decision must be recorded in the audit chain.
	auditPath := filepath.Join(auditDir(), "hands_tool_audit.jsonl")
	n, _, err := policy.VerifyChain(auditPath)
	if err != nil {
		t.Fatalf("audit chain invalid: %v", err)
	}
	if n < 1 {
		t.Fatalf("expected >=1 audit entry from the T0 tool call, got %d", n)
	}
	data, err := os.ReadFile(auditPath)
	if err != nil {
		t.Fatalf("read audit: %v", err)
	}
	if !strings.Contains(string(data), `"tool":"manifest.get"`) {
		t.Fatalf("audit entry does not name the tool manifest.get: %s", data)
	}
	if !strings.Contains(string(data), `"outcome":"execute"`) {
		t.Fatalf("audit entry does not record the execute decision: %s", data)
	}
}

// TestRegisterModelsHandsDisabled pins the models.HandsEnabled() == false branch of
// registration: when KNIRV_HANDS_ENABLED != 1, registerModels must NOT register the
// hands model and /models/hands/* must not be served.
func TestRegisterModelsHandsDisabled(t *testing.T) {
	t.Setenv("KNIRV_HANDS_ENABLED", "0")
	t.Setenv("KNIRV_APP_DATA_DIR", t.TempDir())
	t.Setenv("KNIRV_AUDIT_DIR", t.TempDir())
	// NewServer already runs registerModels(); the disabled branch is exercised
	// when KNIRV_HANDS_ENABLED != 1.
	s := NewServer(t.TempDir()+"/h.sock", 0o660)

	for _, st := range s.registry.Statuses() {
		if st.Name == "hands" {
			t.Fatal("hands registered despite KNIRV_HANDS_ENABLED=0")
		}
	}

	rec := httptest.NewRecorder()
	s.mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, sdk.HandsPrefix+"/status", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("/models/hands/status: got %d, want 404 when HANDS disabled", rec.Code)
	}
}
