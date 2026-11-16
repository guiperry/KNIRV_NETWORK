package nrvtrainer

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testBracket(t *testing.T, seed byte) []byte {
	t.Helper()
	out := make([]byte, bracketSize)
	out[0] = seed
	out[bracketSize-1] = seed
	return out
}

// writeNRV3 writes an asset from registry entries and payloads. Payloads are
// emitted the transitional way (registry member), matching F's documented
// choice until KNIRVBASE can write the canonical payload section.
func writeNRV3(t *testing.T, path string, frames []struct {
	id      string
	valid   bool
	tokens  []int
	payload FramePayload
}) {
	t.Helper()
	var brackets []byte
	registryFrames := make([]map[string]any, 0, len(frames))
	for i, frame := range frames {
		brackets = append(brackets, testBracket(t, byte(i+1))...)
		entry := map[string]any{
			"id":     frame.id,
			"count":  1,
			"offset": i * bracketSize,
			"z3": map[string]any{
				"satisfiable":      frame.valid,
				"constraint_count": 1,
			},
		}
		if frame.payload.TokenWindow != nil || frame.payload.ErrorTestCase != nil ||
			frame.payload.TRLSample != nil || frame.payload.SupervisionEpisode != nil {
			entry["payload"] = frame.payload
		}
		registryFrames = append(registryFrames, entry)
	}
	registry, err := json.Marshal(map[string]any{"version": "3.0", "frames": registryFrames})
	if err != nil {
		t.Fatalf("marshal registry: %v", err)
	}
	out := append([]byte(nrv3Magic), 0, 0, 0, 0)
	binary.LittleEndian.PutUint32(out[4:8], uint32(len(registry)))
	out = append(out, registry...)
	out = append(out, brackets...)
	if err := os.WriteFile(path, out, 0o600); err != nil {
		t.Fatalf("write asset: %v", err)
	}
}

func TestLoadAssetNRV3EmbeddedPayload(t *testing.T) {
	path := filepath.Join(t.TempDir(), "asset.nrv")
	writeNRV3(t, path, []struct {
		id      string
		valid   bool
		tokens  []int
		payload FramePayload
	}{
		{id: "f1", valid: true, payload: FramePayload{TokenWindow: &TokenWindowPayload{Tokens: []uint32{1, 2, 3, 4, 5}}}},
		{id: "f2", valid: false, payload: FramePayload{TokenWindow: &TokenWindowPayload{Tokens: []uint32{9, 9, 9}}}},
		{id: "f3", valid: true, payload: FramePayload{ErrorTestCase: &ErrorTestCasePayload{ID: "e1", Input: "in", Expected: "out"}}},
	})

	asset, err := LoadAsset(path)
	if err != nil {
		t.Fatalf("LoadAsset: %v", err)
	}
	if asset.Format != FormatNRV3 {
		t.Fatalf("format = %q", asset.Format)
	}
	if len(asset.Frames) != 3 {
		t.Fatalf("frames = %d", len(asset.Frames))
	}
	if !asset.Frames[0].Valid() || asset.Frames[1].Valid() || asset.Frames[2].Z3Status != Z3Valid {
		t.Fatalf("unexpected z3 statuses: %q %q %q", asset.Frames[0].Z3Status, asset.Frames[1].Z3Status, asset.Frames[2].Z3Status)
	}
	if asset.Frames[0].BracketCount != 1 || len(asset.Frames[0].RawBrackets) != bracketSize {
		t.Fatalf("brackets = %d/%d", asset.Frames[0].BracketCount, len(asset.Frames[0].RawBrackets))
	}
	windows := asset.TokenWindows(3, 2)
	if len(windows) != 2 {
		t.Fatalf("windows = %d, want 2 (valid frame only, len 5 context 3 stride 2)", len(windows))
	}
	counts := asset.PayloadCounts()
	if counts["token_window"] != 2 || counts["error_test_case"] != 1 {
		t.Fatalf("counts = %#v", counts)
	}
}

func TestLoadAssetNRV3Sidecar(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "asset.nrv")
	writeNRV3(t, path, []struct {
		id      string
		valid   bool
		tokens  []int
		payload FramePayload
	}{
		{id: "f1", valid: true},
		{id: "f2", valid: true},
	})
	sidecar := map[string]FramePayload{
		"f2": {TRLSample: &TRLSamplePayload{Prompt: "p", Chosen: "c"}},
	}
	data, _ := json.Marshal(sidecar)
	if err := os.WriteFile(path+".payload.json", data, 0o600); err != nil {
		t.Fatal(err)
	}

	asset, err := LoadAsset(path)
	if err != nil {
		t.Fatalf("LoadAsset: %v", err)
	}
	if asset.Frames[0].HasPayload {
		t.Fatal("f1 unexpectedly has a payload")
	}
	if asset.Frames[1].Payload.TRLSample == nil || asset.Frames[1].Payload.TRLSample.Chosen != "c" {
		t.Fatalf("sidecar payload not attached: %#v", asset.Frames[1].Payload)
	}
}

func TestLoadAssetRejectsRelativePath(t *testing.T) {
	if _, err := LoadAsset("relative.nrv"); err == nil {
		t.Fatal("expected relative path rejection")
	}
}

func TestDispatcherRoutesOnlyValidFrames(t *testing.T) {
	path := filepath.Join(t.TempDir(), "asset.nrv")
	writeNRV3(t, path, []struct {
		id      string
		valid   bool
		tokens  []int
		payload FramePayload
	}{
		{id: "f1", valid: true, payload: FramePayload{TokenWindow: &TokenWindowPayload{Tokens: []uint32{1, 2, 3, 4, 5, 6}}}},
		{id: "f2", valid: false, payload: FramePayload{TokenWindow: &TokenWindowPayload{Tokens: []uint32{7, 8, 9, 10, 11, 12}}}},
		{id: "f3", valid: true, payload: FramePayload{TokenWindow: &TokenWindowPayload{Tokens: []uint32{13, 14, 15, 16, 17, 18}}}},
	})

	var gotWindows int
	dispatcher := New(&HandsTrainer{Fn: func(_ context.Context, windows [][]int, opts HandsOptions) (HandsStats, error) {
		gotWindows = len(windows)
		if opts.MaxSteps != 10 {
			return HandsStats{}, fmt.Errorf("max steps = %d", opts.MaxSteps)
		}
		return HandsStats{Windows: len(windows), Steps: 3}, nil
	}})

	result, err := dispatcher.Train(context.Background(), path, Options{
		Targets: []Target{TargetHANDS},
		Hands:   HandsOptions{ContextLen: 4, Stride: 2, MaxSteps: 10},
	})
	if err != nil {
		t.Fatalf("Train: %v", err)
	}
	if result.Asset.ValidFrames != 2 {
		t.Fatalf("valid frames = %d", result.Asset.ValidFrames)
	}
	if gotWindows != 4 {
		t.Fatalf("trainer saw %d windows, want 4 (2 valid frames x 2 windows)", gotWindows)
	}
	if len(result.Targets) != 1 || result.Targets[0].Target != TargetHANDS || result.Targets[0].Steps != 3 {
		t.Fatalf("target result = %#v", result.Targets)
	}
}

func TestDispatcherRejectsInvalidOnlyAsset(t *testing.T) {
	path := filepath.Join(t.TempDir(), "asset.nrv")
	writeNRV3(t, path, []struct {
		id      string
		valid   bool
		tokens  []int
		payload FramePayload
	}{{id: "f1", valid: false}})

	dispatcher := New(&HandsTrainer{Fn: func(context.Context, [][]int, HandsOptions) (HandsStats, error) {
		t.Fatal("trainer must not be called")
		return HandsStats{}, nil
	}})
	if _, err := dispatcher.Train(context.Background(), path, Options{}); err == nil {
		t.Fatal("expected refusal for an asset with no valid frames")
	}
}

func TestDispatcherRejectsUnregisteredTarget(t *testing.T) {
	path := filepath.Join(t.TempDir(), "asset.nrv")
	writeNRV3(t, path, []struct {
		id      string
		valid   bool
		tokens  []int
		payload FramePayload
	}{{id: "f1", valid: true, payload: FramePayload{TokenWindow: &TokenWindowPayload{Tokens: []uint32{1, 2, 3}}}}})
	dispatcher := New()
	_, err := dispatcher.Train(context.Background(), path, Options{Targets: []Target{TargetHANDS}})
	if err == nil || !strings.Contains(err.Error(), "not registered") {
		t.Fatalf("err = %v", err)
	}
}

func TestLoadAssetCanonicalContainer(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "canonical.nrv")

	brackets := testBracket(t, 7)
	bracketOffset := int64(headerSize + registryPadding)
	payload, err := json.Marshal(FramePayload{
		SupervisionEpisode: &SupervisionEpisodePayload{State: "state", Decision: "decision"},
	})
	if err != nil {
		t.Fatal(err)
	}
	payloadOffset := bracketOffset + int64(len(brackets))

	registryJSON, err := json.Marshal(map[string]any{
		"version":    1,
		"dataset_id": "dataset-1",
		"frames": []any{map[string]any{
			"id": "f1",
			"z3": map[string]any{"status": "VALID"},
			"brackets": map[string]any{
				"count": 1, "offset": bracketOffset, "length": len(brackets),
			},
			"payload": map[string]any{"offset": payloadOffset, "length": len(payload)},
		}},
		"pqc_manifest": map[string]any{
			"key_id":         "key-1",
			"algorithm":      "ML-DSA-65",
			"file_signature": base64.StdEncoding.EncodeToString([]byte("sig")),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(registryJSON) > registryPadding {
		t.Fatal("registry too large for test")
	}

	out := make([]byte, headerSize+registryPadding, int(payloadOffset)+len(payload))
	binary.LittleEndian.PutUint32(out[0:4], canonicalMagic)
	binary.LittleEndian.PutUint32(out[4:8], 1)
	copy(out[headerSize:], registryJSON)
	out = append(out, brackets...)
	out = append(out, payload...)
	binary.LittleEndian.PutUint32(out[8:12], uint32(len(out)))
	if err := os.WriteFile(path, out, 0o600); err != nil {
		t.Fatal(err)
	}

	asset, err := LoadAsset(path)
	if err != nil {
		t.Fatalf("LoadAsset: %v", err)
	}
	if asset.Format != FormatCanonical || asset.Dataset != "dataset-1" {
		t.Fatalf("asset = %#v", asset)
	}
	if !asset.Signed || asset.KeyID != "key-1" || asset.Algorithm != "ML-DSA-65" || !asset.VerificationSkipped {
		t.Fatalf("pqc summary = %#v", asset)
	}
	if len(asset.Frames) != 1 || !asset.Frames[0].Valid() || !asset.Frames[0].HasPayload {
		t.Fatalf("frame = %#v", asset.Frames)
	}
	if asset.Frames[0].Payload.SupervisionEpisode == nil || asset.Frames[0].Payload.SupervisionEpisode.Decision != "decision" {
		t.Fatalf("payload = %#v", asset.Frames[0].Payload)
	}
	if asset.Frames[0].BracketCount != 1 || len(asset.Frames[0].RawBrackets) != len(brackets) {
		t.Fatalf("brackets = %d/%d", asset.Frames[0].BracketCount, len(asset.Frames[0].RawBrackets))
	}
}

func unixServer(t *testing.T, handler http.Handler) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "nrvtrainer-sock-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	socket := filepath.Join(dir, "ulora.sock")
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	server := &http.Server{Handler: handler}
	go server.Serve(listener)
	t.Cleanup(func() { server.Close() })
	return socket
}

func TestULoRATrainerCompilesWorkflowCorpus(t *testing.T) {
	path := filepath.Join(t.TempDir(), "workflow.nrv")
	writeNRV3(t, path, []struct {
		id      string
		valid   bool
		tokens  []int
		payload FramePayload
	}{
		{id: "f1", valid: true, payload: FramePayload{ErrorTestCase: &ErrorTestCasePayload{ID: "e1", Input: "ctx-b", Expected: "fix-b"}}},
		{id: "f2", valid: true, payload: FramePayload{TRLSample: &TRLSamplePayload{Prompt: "ctx-a", Chosen: "fix-a"}}},
		{id: "f3", valid: true, payload: FramePayload{SupervisionEpisode: &SupervisionEpisodePayload{State: "ctx-c", Decision: "fix-c"}}},
		{id: "f4", valid: false, payload: FramePayload{ErrorTestCase: &ErrorTestCasePayload{Input: "bad", Expected: "bad"}}},
	})

	const token = "internal-secret"
	var seen uloraCompileRequest
	var seenAuth string
	socket := unixServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != uloraCompileRoute {
			http.Error(w, "unexpected route", http.StatusNotFound)
			return
		}
		seenAuth = r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&seen); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(ULoRACompileResult{
			BundleID: "bundle-1", ContentHash: "hash-1",
			ManifestPath: "/tmp/manifest.json", WeightsPath: "/tmp/weights.safetensors",
			PathUsed: "B", TargetModels: []string{"llama-3-8b"},
		})
	}))

	trainer := &ULoRATrainer{Client: NewULoRAClient(socket, token)}
	result, err := trainer.Train(context.Background(), mustLoad(t, path), Options{
		ULoRA: ULoRAOptions{
			WorkflowID:  "workflow-1",
			TargetModel: "llama-3-8b",
			TargetModels: []ULoRATargetModel{{
				Family: "llama", ParamCount: "8b", HiddenSize: 4096, IntermediateSize: 11008,
				NumLayers: 32, AttentionMechanism: "gqa", ActivationFunc: "silu",
			}},
		},
	})
	if err != nil {
		t.Fatalf("Train: %v", err)
	}
	if seenAuth != "Bearer "+token {
		t.Fatalf("auth = %q", seenAuth)
	}
	if len(seen.Dataset) != 3 {
		t.Fatalf("dataset = %d, want 3 (invalid frame excluded)", len(seen.Dataset))
	}
	if seen.Dataset[0].Context != "ctx-a" || seen.Dataset[1].Context != "ctx-b" || seen.Dataset[2].Context != "ctx-c" {
		t.Fatalf("dataset order = %#v", seen.Dataset)
	}
	if seen.Provenance.SourceID != "workflow-1" {
		t.Fatalf("source = %q", seen.Provenance.SourceID)
	}
	if len(seen.TargetModels) != 1 || seen.TargetModels[0].ActivationFunc != "silu" {
		t.Fatalf("targets = %#v", seen.TargetModels)
	}
	if !strings.Contains(result.Detail, "hash-1") {
		t.Fatalf("detail = %q", result.Detail)
	}
}

func TestULoRATrainerValidatesTargetSpec(t *testing.T) {
	path := filepath.Join(t.TempDir(), "workflow.nrv")
	writeNRV3(t, path, []struct {
		id      string
		valid   bool
		tokens  []int
		payload FramePayload
	}{{id: "f1", valid: true, payload: FramePayload{ErrorTestCase: &ErrorTestCasePayload{Input: "in", Expected: "out"}}}})

	trainer := &ULoRATrainer{}
	_, err := trainer.Train(context.Background(), mustLoad(t, path), Options{
		ULoRA: ULoRAOptions{
			TargetModel:  "llama-3-8b",
			TargetModels: []ULoRATargetModel{{Family: "llama", HiddenSize: 4096}},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "attention_mechanism") {
		t.Fatalf("err = %v", err)
	}
}

func TestULoRAClientRequiresToken(t *testing.T) {
	t.Setenv("ULORA_AUTH_TOKEN", "")
	client := NewULoRAClient("/nonexistent.sock", "")
	_, err := client.CompileCluster(context.Background(), uloraCompileRequest{
		Dataset:      []ULoRADatasetRecord{{Context: "in", CorrectedCompletion: "out", TargetModel: "m"}},
		TargetModels: []ULoRATargetModel{{Family: "m"}},
	})
	if err == nil || !strings.Contains(err.Error(), "ULORA_AUTH_TOKEN") {
		t.Fatalf("err = %v", err)
	}
}

func mustLoad(t *testing.T, path string) *Asset {
	t.Helper()
	asset, err := LoadAsset(path)
	if err != nil {
		t.Fatalf("LoadAsset: %v", err)
	}
	return asset
}
