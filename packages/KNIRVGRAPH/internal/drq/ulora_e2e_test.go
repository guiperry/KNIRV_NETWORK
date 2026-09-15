package drq

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ============================================================================
// BundleValidator: every acceptance rule, and the ways it must refuse
// ============================================================================

// unixTransport dials a Unix socket regardless of the URL host, mirroring what
// the production clients do.
func unixTransport(socketPath string) *http.Transport {
	return &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "unix", socketPath)
		},
	}
}

// Echo modes for the fake validation service: what it reports as the bundle it
// actually executed.
const (
	echoSubmittedHash = "submitted" // echoes the hash it was asked about (a real pass)
	echoOtherHash     = "other"     // reports a different bundle's hash
	echoNoHash        = "none"      // reports no attribution at all
)

func verdictServer(t *testing.T, verdict map[string]any, echoMode string) string {
	t.Helper()
	var submittedHash string
	mux := http.NewServeMux()
	mux.HandleFunc("/validation/tasks", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Data map[string]any `json:"data"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "submit decode: "+err.Error(), http.StatusBadRequest)
			return
		}
		submittedHash, _ = body.Data[BundleRevalidationKey].(string)
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]string{"id": "task-1"})
	})
	mux.HandleFunc("/validation/tasks/task-1/results", func(w http.ResponseWriter, r *http.Request) {
		if submittedHash == "" {
			// The fake cannot attribute a verdict, so say so instead of
			// returning a misleading empty echo.
			http.Error(w, "fake validation service saw no bundle hash in the submission", http.StatusInternalServerError)
			return
		}
		payload := map[string]any{}
		for k, v := range verdict {
			payload[k] = v
		}
		// The attribution lives inside the result's `results` map, which is where
		// a real DVE would report the hash it actually executed.
		results, _ := payload["results"].(map[string]any)
		if results == nil {
			results = map[string]any{}
		}
		switch echoMode {
		case echoOtherHash:
			results[BundleRevalidationKey] = "some-other-hash"
		case echoNoHash:
			delete(results, BundleRevalidationKey)
		default:
			results[BundleRevalidationKey] = submittedHash
		}
		payload["results"] = results
		_ = json.NewEncoder(w).Encode(map[string]any{"result": payload})
	})
	return newUnixServer(t, mux)
}

func TestBundleValidatorAcceptsAttributableVerdict(t *testing.T) {
	socket := verdictServer(t, map[string]any{
		"task_id": "task-1",
		"status":  "success",
		"score":   0.9,
		"proof":   "proof-bytes",
		"results": map[string]any{},
	}, echoSubmittedHash)
	v := &DVEBundleValidator{
		BaseURL:      "http://unix",
		Token:        "tok",
		Client:       &http.Client{Transport: unixTransport(socket)},
		PollInterval: 1,
		PollTimeout:  2,
	}
	err := v.ValidateBundle(context.Background(), &ErrorCluster{ClusterID: "c1"},
		&ULoRACompileResult{ContentHash: "abc123"}, []ULoRADatasetRecord{{Context: "x"}})
	if err != nil {
		t.Fatalf("expected an attributable verdict to be accepted, got: %v", err)
	}
}

// A verdict for different content must never be accepted as a pass for ours.
func TestBundleValidatorRefusesVerdictForDifferentBundle(t *testing.T) {
	socket := verdictServer(t, map[string]any{
		"task_id": "task-1",
		"status":  "success",
		"score":   0.9,
		"proof":   "proof-bytes",
		"results": map[string]any{},
	}, echoOtherHash)
	v := &DVEBundleValidator{
		BaseURL: "http://unix", Token: "tok",
		Client: &http.Client{Transport: unixTransport(socket)},
	}
	err := v.ValidateBundle(context.Background(), &ErrorCluster{ClusterID: "c1"},
		&ULoRACompileResult{ContentHash: "abc123"}, []ULoRADatasetRecord{{Context: "x"}})
	if err == nil {
		t.Fatal("a verdict about different content must be refused")
	}
	if !strings.Contains(err.Error(), "different content") {
		t.Fatalf("error should explain the mismatch, got: %v", err)
	}
}

// Today's reality: no service can execute a bundle, so no hash comes back. The
// gate must refuse rather than treat a missing attribution as success.
func TestBundleValidatorRefusesUnattributableVerdict(t *testing.T) {
	socket := verdictServer(t, map[string]any{
		"task_id": "task-1",
		"status":  "success",
		"score":   0.9,
		"proof":   "proof-bytes",
		"results": map[string]any{},
	}, echoNoHash)
	v := &DVEBundleValidator{
		BaseURL: "http://unix", Token: "tok",
		Client: &http.Client{Transport: unixTransport(socket)},
	}
	err := v.ValidateBundle(context.Background(), &ErrorCluster{ClusterID: "c1"},
		&ULoRACompileResult{ContentHash: "abc123"}, []ULoRADatasetRecord{{Context: "x"}})
	if err == nil {
		t.Fatal("a verdict with no bundle attribution must be refused")
	}
	if !strings.Contains(err.Error(), BundleRevalidationKey) {
		t.Fatalf("error should name the missing attribution, got: %v", err)
	}
}

func TestBundleValidatorRefusesNoProofAndLowScore(t *testing.T) {
	for name, verdict := range map[string]map[string]any{
		"no proof": {
			"task_id": "task-1", "status": "success", "score": 0.9, "proof": "",
			"results": map[string]any{},
		},
		"failed status": {
			"task_id": "task-1", "status": "failure", "score": 0.1, "proof": "p",
			"results": map[string]any{},
		},
	} {
		t.Run(name, func(t *testing.T) {
			socket := verdictServer(t, verdict, echoSubmittedHash)
			v := &DVEBundleValidator{
				BaseURL: "http://unix", Token: "tok",
				Client: &http.Client{Transport: unixTransport(socket)},
			}
			if err := v.ValidateBundle(context.Background(), &ErrorCluster{ClusterID: "c1"},
				&ULoRACompileResult{ContentHash: "abc123"}, []ULoRADatasetRecord{{Context: "x"}}); err == nil {
				t.Fatal("must be refused")
			}
		})
	}
}

func TestBundleValidatorRefusesWithoutBackend(t *testing.T) {
	v := NewDVEBundleValidator("", "")
	if err := v.ValidateBundle(context.Background(), &ErrorCluster{ClusterID: "c1"},
		&ULoRACompileResult{ContentHash: "abc"}, nil); err != ErrDVEValidationUnavailable {
		t.Fatalf("err = %v, want ErrDVEValidationUnavailable", err)
	}
}

// ============================================================================
// End to end: corpus -> compile -> validate -> read bundle -> register
// ============================================================================

// TestULoRAMintingEndToEnd drives the whole uLoRA path over real transports:
// a real Unix-socket compile against a fake ulorad, a real bundle file on disk,
// a real Unix-socket validation round trip, and a real Unix-socket mint against
// a fake KNIRVCHAIN that recomputes the hash of what it received.
func TestULoRAMintingEndToEnd(t *testing.T) {
	appData := t.TempDir()
	clustersDir := filepath.Join(appData, "ulora")

	// --- fake ulorad: writes a real bundle and reports its true hash ---
	var compiledDataset []ULoRADatasetRecord
	uloraSocket := newUnixServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != uloraCompileRoute {
			http.Error(w, "unexpected route "+r.URL.Path, http.StatusNotFound)
			return
		}
		var req uloraCompileRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		compiledDataset = req.Dataset

		bundleDir := filepath.Join(clustersDir, "bundles", req.Provenance.SourceID+"-deadbeef")
		if err := os.MkdirAll(bundleDir, 0o755); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		manifestPath := filepath.Join(bundleDir, "manifest.json")
		if err := os.WriteFile(manifestPath, []byte(`{"manifest_version":"1.0.0"}`), 0o644); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		weightsPath := filepath.Join(bundleDir, "core_weights.safetensors")
		if err := os.WriteFile(weightsPath, []byte("weights"), 0o644); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		bundlePath := filepath.Join(clustersDir, "bundles", "ulora-"+req.Provenance.SourceID+".ulora")
		payload := []byte("ULORA-BUNDLE-BYTES-" + req.Provenance.SourceID)
		if err := os.WriteFile(bundlePath, payload, 0o644); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		sum := sha256.Sum256(payload)

		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(ULoRACompileResult{
			BundleID:     req.Provenance.SourceID,
			ContentHash:  hex.EncodeToString(sum[:]),
			ManifestPath: manifestPath,
			WeightsPath:  weightsPath,
			PathUsed:     "B",
			TargetModels: []string{"llama-3-8b"},
		})
	}))

	// --- fake validation service: echoes back the hash it was asked about ---
	validationSocket := verdictServer(t, map[string]any{
		"task_id": "task-1", "status": "success", "score": 0.95, "proof": "dve-proof",
		"results": map[string]any{},
	}, echoSubmittedHash)

	// --- fake KNIRVCHAIN: hashes what it received, like the real mint route ---
	var (
		registeredBytes []byte
		registeredHash  string
		registeredToken string
	)
	const internalToken = "internal-secret"
	chainSocket := newUnixServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != knirvchainULoRABundleMintRoute {
			http.Error(w, "unexpected route", http.StatusNotFound)
			return
		}
		registeredToken = r.Header.Get("X-KNIRV-Internal-Token")
		var req knirvchainULoRABundleMintRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		decoded, err := base64.StdEncoding.DecodeString(req.Bundle)
		if err != nil {
			http.Error(w, "bad base64", http.StatusBadRequest)
			return
		}
		registeredBytes = decoded
		sum := sha256.Sum256(decoded)
		registeredHash = hex.EncodeToString(sum[:])
		if !strings.EqualFold(req.ContentHash, registeredHash) {
			http.Error(w, "hash mismatch", http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"bundle_id": req.BundleID, "content_hash": registeredHash,
		})
	}))

	// --- environment the real clients read ---
	t.Setenv("ULORA_SOCKET_PATH", uloraSocket)
	t.Setenv("KNIRV_CHAIN_SOCKET_PATH", chainSocket)
	t.Setenv("KNIRV_INTERNAL_AUTH_TOKEN", internalToken)

	// --- the protocol, wired to the fakes through the real clients ---
	cluster := corpusCluster()
	cluster.OwnerAgent = "agent-owner"

	docs := &fakeSkillDocs{docs: map[string]string{
		"e1": "add a bounds check",
		"e2": "guard the pointer",
	}}

	protocol := NewSkillMintingProtocol(SkillMintingDeps{
		Chain:     &KNIRVCHAINClient{},
		SkillDocs: docs,
		Compiler:  NewKNIRVULORAClient(""),
		Validator: &DVEBundleValidator{
			BaseURL: "http://unix", Token: "tok",
			Client: &http.Client{Transport: unixTransport(validationSocket)},
		},
		TargetModels: map[string]ULoRATargetModel{
			"llama-3-8b": {Family: "llama", ParamCount: "8b", HiddenSize: 4096, IntermediateSize: 11008, NumLayers: 32, AttentionMechanism: "gqa", ActivationFunc: "silu"},
		},
		ManifestVersion: "1.0.0",
	})

	skillNode := &SkillNode{ID: "skill-1", Creator: "agent-a", Description: "d"}
	if err := protocol.mintULoRABundle(context.Background(), cluster, skillNode); err != nil {
		t.Fatalf("end-to-end mint failed: %v", err)
	}

	// The corpus reached the compiler with both members' fixes.
	if len(compiledDataset) != 2 {
		t.Fatalf("compiler received %d records, want 2", len(compiledDataset))
	}
	if compiledDataset[0].CorrectedCompletion != "add a bounds check" {
		t.Fatalf("corpus fix = %q", compiledDataset[0].CorrectedCompletion)
	}

	// The chain received the real bundle bytes, under the internal token, and the
	// hash it computed matches what was declared.
	if registeredToken != internalToken {
		t.Fatalf("chain saw token %q", registeredToken)
	}
	if string(registeredBytes) != "ULORA-BUNDLE-BYTES-cluster-1" {
		t.Fatalf("chain received %q", registeredBytes)
	}
	if registeredHash == "" {
		t.Fatal("chain computed no hash")
	}

	// And the skill now points at the adapter it produced.
	if !strings.Contains(skillNode.CodePackageURI, registeredHash) {
		t.Fatalf("skill CodePackageURI = %q, want it to reference %s", skillNode.CodePackageURI, registeredHash)
	}
}

// The gate must stop the whole path: with a validator that refuses, nothing may
// be registered on the chain.
func TestULoRAMintingEndToEndGateBlocksRegistration(t *testing.T) {
	appData := t.TempDir()
	clustersDir := filepath.Join(appData, "ulora")

	uloraSocket := newUnixServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req uloraCompileRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		bundleDir := filepath.Join(clustersDir, "bundles", req.Provenance.SourceID+"-x")
		_ = os.MkdirAll(bundleDir, 0o755)
		manifestPath := filepath.Join(bundleDir, "manifest.json")
		_ = os.WriteFile(manifestPath, []byte("{}"), 0o644)
		bundlePath := filepath.Join(clustersDir, "bundles", "ulora-"+req.Provenance.SourceID+".ulora")
		payload := []byte("bytes")
		_ = os.WriteFile(bundlePath, payload, 0o644)
		sum := sha256.Sum256(payload)
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(ULoRACompileResult{
			BundleID: req.Provenance.SourceID, ContentHash: hex.EncodeToString(sum[:]),
			ManifestPath: manifestPath, WeightsPath: manifestPath,
		})
	}))

	chainCalled := false
	chainSocket := newUnixServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		chainCalled = true
		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, `{"bundle_id":"cluster-1","content_hash":"x"}`)
	}))

	t.Setenv("ULORA_SOCKET_PATH", uloraSocket)
	t.Setenv("KNIRV_CHAIN_SOCKET_PATH", chainSocket)
	t.Setenv("KNIRV_INTERNAL_AUTH_TOKEN", "t")

	protocol := NewSkillMintingProtocol(SkillMintingDeps{
		Chain:           &KNIRVCHAINClient{},
		SkillDocs:       &fakeSkillDocs{docs: map[string]string{"e1": "fix", "e2": "fix"}},
		Compiler:        NewKNIRVULORAClient(""),
		Validator:       stubValidator{err: ErrDVEValidationUnavailable},
		TargetModels:    map[string]ULoRATargetModel{"llama-3-8b": {Family: "llama", HiddenSize: 4096, IntermediateSize: 11008, NumLayers: 32, AttentionMechanism: "gqa", ActivationFunc: "silu"}},
		ManifestVersion: "1.0.0",
	})

	err := protocol.mintULoRABundle(context.Background(), corpusCluster(), &SkillNode{ID: "skill-1"})
	if err == nil {
		t.Fatal("a refusing validator must fail the mint")
	}
	if chainCalled {
		t.Fatal("the bundle must not be registered when validation refuses")
	}
}
