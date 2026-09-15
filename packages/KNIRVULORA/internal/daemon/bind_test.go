package daemon

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ulora/internal/api"
	"ulora/internal/bundling"
	"ulora/internal/config"
	"ulora/internal/connector"
	"ulora/internal/safetensors"
)

// writeTestBundle builds a real .ulora archive (manifest + weights) at the path
// the compiler would produce for bundleID, so the route is exercised against the
// genuine bundle format rather than a directory of loose files.
const (
	// canonicalDim is K, the shared latent space of the canonical core.
	canonicalDim = 1024
	rank         = 2
	hiddenSize   = 2
)

func writeTestBundle(t *testing.T, dataDir, bundleID string, modules []string) string {
	t.Helper()

	bundleDir := filepath.Join(dataDir, "bundles", bundleID+"-src")
	if err := os.MkdirAll(bundleDir, 0o755); err != nil {
		t.Fatalf("mkdir bundle dir: %v", err)
	}

	// A complete manifest: ManifestFromJSON validates it, so a partial one is
	// rejected before the route ever gets to projection.
	manifest := map[string]any{
		"$schema": "https://ulora.org/schema/v1/manifest.json",
		"name":    "test-bundle",
		"version": "1.0.0",
		"provenance": map[string]any{
			"source_id":          "cluster-1",
			"source_dataset_ids": []string{"skill-1"},
		},
		"base_source_model": map[string]any{
			"family":              "llama",
			"param_count":         "8b",
			"hidden_size":         2,
			"intermediate_size":   4,
			"num_attention_heads": 1,
			"num_key_value_heads": 1,
			"head_dim":            2,
			"num_layers":          1,
			"attention_mechanism": "gqa",
			"activation_func":     "silu",
		},
		"canonical_core": map[string]any{
			"adapter_rank":         2,
			"scaling_factor_alpha": 32.0,
			// K: the shared latent space the canonical core lives in.
			"canonical_dim":  1024,
			"target_modules": modules,
		},
		"routing_policy": map[string]any{
			"similarity_threshold":   0.5,
			"allowed_transfer_paths": []string{"subspace_svd"},
		},
	}
	manifestJSON, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(bundleDir, bundling.ManifestFile), manifestJSON, 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	// Real safetensors, because that is what the compiler writes: it merges via
	// safetensors.numpy.save_file. Writing JSON here would verify the fixture
	// rather than the system — which is exactly how the binder's JSON read went
	// unnoticed.
	//
	// The artifact carries the canonical core only. Connectors are a property of
	// the model, so they live in the runtime's cache and are supplied to the
	// binder as a provider — not embedded in the bundle.
	weights := map[string][][]float32{}
	for _, module := range modules {
		weights[module+"/lora_A"] = mat(rank, canonicalDim) // r × K
		weights[module+"/lora_B"] = mat(canonicalDim, rank) // K × r
	}

	weightsData, err := safetensors.Write(weights, map[string]string{"producer": "test"})
	if err != nil {
		t.Fatalf("encode weights: %v", err)
	}
	if err := os.WriteFile(filepath.Join(bundleDir, bundling.WeightsFile), weightsData, 0o644); err != nil {
		t.Fatalf("write weights: %v", err)
	}

	bundlePath := filepath.Join(dataDir, "bundles", "ulora-"+bundleID+".ulora")
	if err := bundling.WriteBundle(bundleDir, bundlePath); err != nil {
		t.Fatalf("write bundle: %v", err)
	}
	return bundlePath
}

// rawSocketCall issues a request without adding an Authorization header, so the
// auth guard can be tested.
func rawSocketCall(t *testing.T, cfg *config.Config, method, path string, body []byte, token string) *http.Response {
	t.Helper()
	client := &http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", cfg.SocketPath)
			},
		},
	}
	var reader *bytes.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	} else {
		reader = bytes.NewReader(nil)
	}
	req, err := http.NewRequest(method, "http://ulora"+path, reader)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("socket call: %v", err)
	}
	return resp
}

// mat builds a deterministic, non-zero matrix so a projection result can be
// distinguished from an all-zero one.
func mat(rows, cols int) [][]float32 {
	out := make([][]float32, rows)
	for i := range out {
		out[i] = make([]float32, cols)
		for j := range out[i] {
			out[i][j] = float32(i%7+1) * 0.01
		}
	}
	return out
}

func sha256File(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func targetModel() api.BaseModelSpec {
	return api.BaseModelSpec{
		Family:             "llama",
		ParamCount:         "8b",
		HiddenSize:         2,
		IntermediateSize:   4,
		NumAttentionHeads:  1,
		NumKeyValueHeads:   1,
		HeadDim:            2,
		NumLayers:          1,
		AttentionMechanism: "gqa",
		ActivationFunc:     "silu",
	}
}

// The bind route does real work on behalf of a caller, so it must be gated like
// the other mutating routes.
func TestBindRequiresAuth(t *testing.T) {
	_, cfg, cleanup := startTestServer(t)
	defer cleanup()

	resp := rawSocketCall(t, cfg, http.MethodPost, "/ulora/v1/bind", []byte(`{}`), "")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized && resp.StatusCode != http.StatusForbidden {
		t.Fatalf("unauthenticated bind returned %d, want 401/403", resp.StatusCode)
	}
}

// A caller that names no bundle, or a bundle that does not exist, must be
// refused rather than bound against whatever happens to be around.
func TestBindRejectsUnresolvableBundle(t *testing.T) {
	_, cfg, cleanup := startTestServer(t)
	defer cleanup()

	body := `{"target_model":{"family":"llama","hidden_size":2,"num_layers":1}}`
	resp := rawSocketCall(t, cfg, http.MethodPost, "/ulora/v1/bind", []byte(body), cfg.AuthToken)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("no bundle specified returned %d, want 400", resp.StatusCode)
	}

	body = `{"bundle_id":"does-not-exist","target_model":{"family":"llama","hidden_size":2,"num_layers":1}}`
	resp2 := rawSocketCall(t, cfg, http.MethodPost, "/ulora/v1/bind", []byte(body), cfg.AuthToken)
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusBadRequest {
		t.Fatalf("unknown bundle id returned %d, want 400", resp2.StatusCode)
	}
}

// A declared hash that does not describe the bundle must be refused: binding a
// stale bundle while believing it is another one is precisely the silent failure
// this check exists to prevent.
func TestBindRefusesContentHashMismatch(t *testing.T) {
	_, cfg, cleanup := startTestServer(t)
	defer cleanup()

	writeTestBundle(t, cfg.DataDir, "cluster-1", []string{"q_proj"})

	body := `{"bundle_id":"cluster-1","content_hash":"` + strings.Repeat("0", 64) + `","target_model":{"family":"llama","hidden_size":2,"num_layers":1}}`
	resp := rawSocketCall(t, cfg, http.MethodPost, "/ulora/v1/bind", []byte(body), cfg.AuthToken)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("hash mismatch returned %d, want 409", resp.StatusCode)
	}
}

// The happy path: a bundle resolves by id, is projected, and the response reports
// the hash of the bytes actually bound — which is what a validator attributes a
// verdict to.
func TestBindProjectsBundleAndReportsRealHash(t *testing.T) {
	_, cfg, cleanup := startTestServer(t)
	defer cleanup()

	bundlePath := writeTestBundle(t, cfg.DataDir, "cluster-1", []string{"q_proj"})
	wantHash := sha256File(t, bundlePath)

	// Seed the connector cache for the target model, standing in for the
	// derivation the runtime performs once per model family. The directory
	// matches the daemon's fallback when ConnectorDir is unset.
	cache := connector.NewCache(filepath.Join(cfg.DataDir, "connectors"))
	if err := cache.Put(&connector.Connector{
		Family:       "llama",
		ParamCount:   "8b",
		CanonicalDim: canonicalDim,
		PIn:          mat(canonicalDim, hiddenSize),
		POut:         mat(hiddenSize, canonicalDim),
	}); err != nil {
		t.Fatalf("seed connector cache: %v", err)
	}

	payload := map[string]any{
		"bundle_id":    "cluster-1",
		"content_hash": wantHash,
		"target_model": targetModel(),
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	resp := rawSocketCall(t, cfg, http.MethodPost, "/ulora/v1/bind", raw, cfg.AuthToken)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		buf := new(bytes.Buffer)
		_, _ = buf.ReadFrom(resp.Body)
		t.Fatalf("bind returned %d: %s", resp.StatusCode, buf.String())
	}

	var out BindResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if out.ContentHash != wantHash {
		t.Fatalf("ContentHash = %q, want the bundle's real digest %q", out.ContentHash, wantHash)
	}
	if out.AdapterRank != 2 {
		t.Fatalf("AdapterRank = %d, want 2", out.AdapterRank)
	}
	if out.TargetModules != 1 {
		t.Fatalf("TargetModules = %d, want 1", out.TargetModules)
	}
	if len(out.Layers) == 0 {
		t.Fatal("no layers were projected")
	}
	for name, layer := range out.Layers {
		if len(layer.A) == 0 || len(layer.B) == 0 {
			t.Fatalf("layer %q has empty A/B matrices: %+v", name, layer)
		}
	}
	if out.ManifestVersion != "1.0.0" {
		t.Fatalf("ManifestVersion = %q, want 1.0.0", out.ManifestVersion)
	}
}
