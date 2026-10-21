package head

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"os"
	"path/filepath"
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
