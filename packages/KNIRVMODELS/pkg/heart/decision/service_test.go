package decision

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Lays out testdata the way 5_HEAD_TRAINER publishes it.
func publish(t *testing.T, version string) string {
	t.Helper()
	dir := t.TempDir()
	cfg, _ := os.ReadFile("testdata/golden_config.json")
	var m map[string]any
	json.Unmarshal(cfg, &m)
	m["model_version"] = version
	cfg, _ = json.Marshal(m)
	w, _ := os.ReadFile("testdata/golden.safetensors")
	os.WriteFile(filepath.Join(dir, "head_config.json"), cfg, 0o644)
	os.WriteFile(filepath.Join(dir, "model-"+version+".safetensors"), w, 0o644)
	if err := os.Symlink("model-"+version+".safetensors", filepath.Join(dir, "model_latest")); err != nil {
		t.Fatal(err)
	}
	return dir
}

type reasoning struct {
	PlanTokens   []uint32  `json:"plan_tokens"`
	HaltingSteps int       `json:"halting_steps"`
	Confidence   float64   `json:"confidence"`
	L            []float64 `json:"l_module_activations"`
	Decisions    []struct {
		Name          string    `json:"name"`
		Probabilities []float64 `json:"probabilities"`
	} `json:"decisions"`
	ModelVersion string `json:"model_version"`
}

func close(a, b float64) bool { return math.Abs(a-b) <= 1e-5*math.Max(1, math.Abs(b)) }

// The same fixture, input and golden.json the Rust and Node (WASM) tests use.
func TestWazeroAgreesWithNativeGolden(t *testing.T) {
	const v = "v1-deadbeef"
	svc := NewService(publish(t, v))
	defer svc.Close(context.Background())

	raw, err := svc.Reason(context.Background(), []uint32{3, 1, 4, 1, 5, 9, 2, 6})
	if err != nil {
		t.Fatal(err)
	}
	var got, want reasoning
	json.Unmarshal(raw, &got)
	g, _ := os.ReadFile("testdata/golden.json")
	json.Unmarshal(g, &want)

	if got.ModelVersion != v {
		t.Fatalf("model_version %q", got.ModelVersion)
	}
	if len(got.PlanTokens) != len(want.PlanTokens) || got.HaltingSteps != want.HaltingSteps {
		t.Fatalf("plan/halting differ: %v/%d vs %v/%d", got.PlanTokens, got.HaltingSteps, want.PlanTokens, want.HaltingSteps)
	}
	for i := range want.PlanTokens {
		if got.PlanTokens[i] != want.PlanTokens[i] {
			t.Fatalf("plan token %d: %d vs %d", i, got.PlanTokens[i], want.PlanTokens[i])
		}
	}
	for i := range want.L {
		if !close(got.L[i], want.L[i]) {
			t.Fatalf("l_act[%d]: %v vs %v", i, got.L[i], want.L[i])
		}
	}
	for i, d := range want.Decisions {
		for j := range d.Probabilities {
			if !close(got.Decisions[i].Probabilities[j], d.Probabilities[j]) {
				t.Fatalf("decision %s[%d]: %v vs %v", d.Name, j, got.Decisions[i].Probabilities[j], d.Probabilities[j])
			}
		}
	}
}

func TestManifestAndWeights(t *testing.T) {
	svc := NewService(publish(t, "v7-0a0a0a0a"))
	m, err := svc.Manifest()
	if err != nil {
		t.Fatal(err)
	}
	if m.Version != "v7-0a0a0a0a" || len(m.SHA256) != 64 || m.Size == 0 {
		t.Fatalf("%+v", m)
	}
	p, err := svc.WeightsPath(m.Version)
	if err != nil || filepath.Base(p) != "model-v7-0a0a0a0a.safetensors" {
		t.Fatal(p, err)
	}
	for _, bad := range []string{"../etc/passwd", "v1-xyz", "", "v1-deadbeef/../../x", "latest"} {
		if _, err := svc.WeightsPath(bad); err == nil {
			t.Errorf("version %q accepted", bad)
		}
	}
}

func TestNoModelAndBadInput(t *testing.T) {
	empty := NewService(t.TempDir())
	if _, err := empty.Manifest(); err != ErrNoModel {
		t.Fatalf("manifest: %v", err)
	}
	if _, err := empty.Reason(context.Background(), []uint32{1}); err != ErrNoModel {
		t.Fatalf("reason: %v", err)
	}
	svc := NewService(publish(t, "v1-deadbeef"))
	defer svc.Close(context.Background())
	if _, err := svc.Reason(context.Background(), nil); err == nil {
		t.Fatal("empty input accepted")
	}
	if _, err := svc.Reason(context.Background(), []uint32{9999}); err == nil {
		t.Fatal("out-of-vocab token accepted")
	}
	// the module survives a rejected request
	if _, err := svc.Reason(context.Background(), []uint32{1, 2}); err != nil {
		t.Fatal(err)
	}
}

func TestHotSwapOnPromotion(t *testing.T) {
	dir := publish(t, "v1-aaaaaaaa")
	svc := NewService(dir)
	defer svc.Close(context.Background())
	r1, err := svc.Reason(context.Background(), []uint32{1, 2, 3})
	if err != nil {
		t.Fatal(err)
	}
	// promote v2 (same weights, new version string), as the trainer does.
	cfg, _ := os.ReadFile(filepath.Join(dir, "head_config.json"))
	var m map[string]any
	json.Unmarshal(cfg, &m)
	m["model_version"] = "v2-bbbbbbbb"
	cfg, _ = json.Marshal(m)
	w, _ := os.ReadFile(filepath.Join(dir, "model-v1-aaaaaaaa.safetensors"))
	os.WriteFile(filepath.Join(dir, "head_config.json"), cfg, 0o644)
	os.WriteFile(filepath.Join(dir, "model-v2-bbbbbbbb.safetensors"), w, 0o644)
	os.Remove(filepath.Join(dir, "model_latest"))
	os.Symlink("model-v2-bbbbbbbb.safetensors", filepath.Join(dir, "model_latest"))
	r2, err := svc.Reason(context.Background(), []uint32{1, 2, 3})
	if err != nil {
		t.Fatal(err)
	}
	var a, b reasoning
	json.Unmarshal(r1, &a)
	json.Unmarshal(r2, &b)
	if a.ModelVersion != "v1-aaaaaaaa" || b.ModelVersion != "v2-bbbbbbbb" {
		t.Fatalf("versions %q -> %q", a.ModelVersion, b.ModelVersion)
	}
}

func TestReasonRebuildsAfterTheModuleDies(t *testing.T) {
	svc := NewService(publish(t, "v1-deadbeef"))
	defer svc.Close(context.Background())
	in := []uint32{3, 1, 4, 1, 5, 9, 2, 6}
	if _, err := svc.Reason(context.Background(), in); err != nil {
		t.Fatal(err)
	}
	// Break the pipe the way a crashed module would (H11: this used to be permanent).
	svc.mu.Lock()
	dead := svc.runner
	svc.mu.Unlock()
	dead.mu.Lock()
	dead.kill(errors.New("test: module crashed"))
	dead.mu.Unlock()
	if dead.Dead() == nil {
		t.Fatal("runner should be dead")
	}
	if _, err := svc.Reason(context.Background(), in); err != nil {
		t.Fatalf("service did not recover: %v", err)
	}
	svc.mu.Lock()
	defer svc.mu.Unlock()
	if svc.runner == dead || svc.runner.Dead() != nil {
		t.Fatal("expected a fresh, live runner")
	}
}

func TestHungCallTimesOutAndTheServiceRecovers(t *testing.T) {
	svc := NewService(publish(t, "v1-deadbeef"))
	defer svc.Close(context.Background())
	svc.CallTimeout = time.Nanosecond // every call "hangs"
	svc.Attempts = 2
	start := time.Now()
	_, err := svc.Reason(context.Background(), []uint32{3, 1, 4})
	if !errors.Is(err, ErrRunnerDead) {
		t.Fatalf("want ErrRunnerDead, got %v", err)
	}
	if time.Since(start) > 5*time.Second {
		t.Fatalf("hung call was not bounded: %s", time.Since(start))
	}
	svc.CallTimeout = DefaultCallTimeout
	if _, err := svc.Reason(context.Background(), []uint32{3, 1, 4, 1, 5, 9, 2, 6}); err != nil {
		t.Fatalf("service stayed down after a timeout: %v", err)
	}
}

func TestModelErrorsAreNotRetriedOrFatal(t *testing.T) {
	svc := NewService(publish(t, "v1-deadbeef"))
	defer svc.Close(context.Background())
	if _, err := svc.Reason(context.Background(), nil); err == nil || errors.Is(err, ErrRunnerDead) {
		t.Fatalf("empty input should be a plain error, got %v", err)
	}
	if _, err := svc.Reason(context.Background(), []uint32{3, 1, 4, 1, 5, 9, 2, 6}); err != nil {
		t.Fatalf("a bad request must not break the runner: %v", err)
	}
}

type adapterLayer struct {
	A [][]float32 `json:"a"`
	B [][]float32 `json:"b"`
}

func goldenDModel(t *testing.T) int {
	t.Helper()
	b, err := os.ReadFile("testdata/golden_config.json")
	if err != nil {
		t.Fatal(err)
	}
	var cfg struct {
		DModel int `json:"d_model"`
	}
	if err := json.Unmarshal(b, &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.DModel == 0 {
		t.Fatal("golden config has no d_model")
	}
	return cfg.DModel
}

func goldenBaseSHA(t *testing.T) string {
	t.Helper()
	w, err := os.ReadFile("testdata/golden.safetensors")
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(w)
	return hex.EncodeToString(sum[:])
}

// testAdapter builds a real bind-shaped adapter for the golden model.
func testAdapter(t *testing.T, baseSHA string, strength float64, roles ...string) []byte {
	t.Helper()
	d := goldenDModel(t)
	layers := make(map[string]adapterLayer, len(roles))
	for k, role := range roles {
		a := make([][]float32, 4)
		for i := range a {
			a[i] = make([]float32, d)
			for j := range a[i] {
				a[i][j] = float32(0.3 * math.Sin(float64(i*7+j*3+k)))
			}
		}
		b := make([][]float32, d)
		for i := range b {
			b[i] = make([]float32, 4)
			for j := range b[i] {
				b[i][j] = float32(0.3 * math.Sin(float64(i*7+j*3+9+k)))
			}
		}
		layers[role] = adapterLayer{A: a, B: b}
	}
	raw, err := json.Marshal(map[string]any{
		"adapter_rank":  4,
		"scaling_alpha": 64.0,
		"content_hash":  "test-adapter",
		"stacks":        []string{"l", "h"},
		"strength":      strength,
		"base_sha256":   baseSHA,
		"layers":        layers,
	})
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func mustReason(t *testing.T, svc *Service, tokens []uint32) reasoning {
	t.Helper()
	raw, err := svc.Reason(context.Background(), tokens)
	if err != nil {
		t.Fatal(err)
	}
	var r reasoning
	if err := json.Unmarshal(raw, &r); err != nil {
		t.Fatal(err)
	}
	return r
}

func probsEqual(a, b reasoning) bool {
	if len(a.Decisions) != len(b.Decisions) {
		return false
	}
	for i := range a.Decisions {
		if len(a.Decisions[i].Probabilities) != len(b.Decisions[i].Probabilities) {
			return false
		}
		for j := range a.Decisions[i].Probabilities {
			if a.Decisions[i].Probabilities[j] != b.Decisions[i].Probabilities[j] {
				return false
			}
		}
	}
	return true
}

func TestAdapterChangesOutputRevertsAndIsCached(t *testing.T) {
	const v = "v1-deadbeef"
	svc := NewService(publish(t, v))
	defer svc.Close(context.Background())

	tokens := []uint32{3, 1, 4, 1, 5, 9, 2, 6}
	base := mustReason(t, svc, tokens)
	ad := testAdapter(t, goldenBaseSHA(t), 1.0,
		"self_attn.q_proj", "self_attn.k_proj", "self_attn.v_proj", "self_attn.o_proj")
	if err := svc.SetAdapter(ad); err != nil {
		t.Fatal(err)
	}
	tuned := mustReason(t, svc, tokens)
	if probsEqual(base, tuned) {
		t.Fatal("adapter did not change the server-side output")
	}

	sum := sha256.Sum256(ad)
	hash := hex.EncodeToString(sum[:])
	svc.mu.Lock()
	loaded, applied := svc.loaded, svc.info.Adapter
	svc.mu.Unlock()
	if loaded != v+"@"+hash {
		t.Fatalf("loaded key %q, want %q", loaded, v+"@"+hash)
	}
	if applied == nil || len(applied.RolesApplied) != 4 || applied.TensorsChanged != 8 {
		t.Fatalf("applied adapter = %+v", applied)
	}
	if got := svc.Status()["adapter"]; got != hash {
		t.Fatalf("status adapter = %v, want %s", got, hash)
	}

	// The same adapter hash is a cache hit: the live merged model is reused, so
	// the load counter must not move (the output alone would also match a reload).
	svc.mu.Lock()
	loadsBefore := svc.loads
	svc.mu.Unlock()
	again := mustReason(t, svc, tokens)
	if !probsEqual(tuned, again) {
		t.Fatal("the same adapter produced different output on a second call")
	}
	svc.mu.Lock()
	loadsAfter := svc.loads
	svc.mu.Unlock()
	if loadsAfter != loadsBefore {
		t.Fatalf("same adapter hash reloaded the model: loads %d -> %d", loadsBefore, loadsAfter)
	}

	// strength=0 merges a zero delta, so the output is the base model again.
	zero := testAdapter(t, goldenBaseSHA(t), 0.0,
		"self_attn.q_proj", "self_attn.k_proj", "self_attn.v_proj", "self_attn.o_proj")
	if err := svc.SetAdapter(zero); err != nil {
		t.Fatal(err)
	}
	if got := mustReason(t, svc, tokens); !probsEqual(base, got) {
		t.Fatal("strength=0 did not revert to the base output")
	}

	if err := svc.SetAdapter(nil); err != nil {
		t.Fatal(err)
	}
	if got := mustReason(t, svc, tokens); !probsEqual(base, got) {
		t.Fatal("clearing the adapter did not revert to the base output")
	}
}

// SetAdapter must refuse input that is not an adapter object at install time,
// not at the next request, and must leave the installed adapter untouched.
func TestSetAdapterRejectsNonAdapterJSON(t *testing.T) {
	svc := NewService(publish(t, "v1-deadbeef"))
	defer svc.Close(context.Background())

	good := testAdapter(t, goldenBaseSHA(t), 1.0, "self_attn.q_proj")
	if err := svc.SetAdapter(good); err != nil {
		t.Fatal(err)
	}
	for name, raw := range map[string]string{
		"null":             "null",
		"object no layers": `{"adapter_rank":2}`,
		"layers null":      `{"layers":null}`,
		"not an object":    `[1,2]`,
		"malformed":        `{"layers":`,
	} {
		if err := svc.SetAdapter([]byte(raw)); err == nil {
			t.Fatalf("%s: SetAdapter accepted %q", name, raw)
		}
	}
	svc.mu.Lock()
	kept := svc.adapterHash
	svc.mu.Unlock()
	sum := sha256.Sum256(good)
	if kept != hex.EncodeToString(sum[:]) {
		t.Fatal("a refused SetAdapter replaced the installed adapter")
	}
}

func TestWrongBaseAdapterIsRefusedAndDoesNotPoisonTheService(t *testing.T) {
	svc := NewService(publish(t, "v1-deadbeef"))
	defer svc.Close(context.Background())

	tokens := []uint32{3, 1, 4, 1, 5, 9, 2, 6}
	base := mustReason(t, svc, tokens)
	wrong := testAdapter(t, strings.Repeat("0", 64), 1.0, "self_attn.q_proj")
	if err := svc.SetAdapter(wrong); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Reason(context.Background(), tokens); err == nil || !strings.Contains(err.Error(), "targets base") {
		t.Fatalf("wrong-base adapter was not refused: %v", err)
	}

	if err := svc.SetAdapter(nil); err != nil {
		t.Fatal(err)
	}
	if got := mustReason(t, svc, tokens); !probsEqual(base, got) {
		t.Fatal("the service did not recover after a refused adapter")
	}
}
