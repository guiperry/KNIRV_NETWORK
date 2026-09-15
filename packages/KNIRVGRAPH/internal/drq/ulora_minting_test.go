package drq

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeSkillDocs resolves member errors to validated fixes.
type fakeSkillDocs struct {
	docs map[string]string
}

func (f *fakeSkillDocs) SkillDocForError(errorID string) (string, error) {
	return f.docs[errorID], nil
}

func corpusCluster() *ErrorCluster {
	return &ErrorCluster{
		ClusterID: "cluster-1",
		Errors: []*ErrorNode{
			{
				Id:          "e2",
				ErrorType:   "runtime",
				Domain:      "runtime",
				Description: "nil pointer",
				ModelOrigin: "llama-3-8b",
			},
			{
				Id:          "e1",
				ErrorType:   "runtime",
				Domain:      "runtime",
				Description: "index out of range",
				ModelOrigin: "llama-3-8b",
			},
		},
	}
}

// Only members with a resolvable fix become training records, and the order is
// deterministic so the same cluster always compiles to the same content hash.
func TestGatherClusterDatasets(t *testing.T) {
	docs := &fakeSkillDocs{docs: map[string]string{
		"e1": "add a bounds check",
		"e2": "guard the pointer",
	}}

	dataset, skillIDs, err := gatherClusterDatasets(corpusCluster(), docs)
	if err != nil {
		t.Fatalf("gather: %v", err)
	}
	if len(dataset) != 2 {
		t.Fatalf("records = %d, want 2", len(dataset))
	}
	if !strings.Contains(dataset[0].Context, "index out of range") {
		t.Fatalf("records must be ordered by error id; first = %q", dataset[0].Context)
	}
	if dataset[0].CorrectedCompletion != "add a bounds check" {
		t.Fatalf("CorrectedCompletion = %q, want the validated fix", dataset[0].CorrectedCompletion)
	}
	if dataset[0].TargetModel != "llama-3-8b" {
		t.Fatalf("TargetModel = %q, want model_origin", dataset[0].TargetModel)
	}
	if len(skillIDs) != 2 || skillIDs[0] != "e1" {
		t.Fatalf("skill ids = %v", skillIDs)
	}

	// Deterministic across runs.
	again, _, err := gatherClusterDatasets(corpusCluster(), docs)
	if err != nil {
		t.Fatalf("gather again: %v", err)
	}
	if again[0].Context != dataset[0].Context || again[1].Context != dataset[1].Context {
		t.Fatal("corpus ordering is not deterministic")
	}
}

// A member with no fix is skipped rather than filled with a placeholder, and a
// cluster with nothing resolvable is an error, not an empty corpus.
func TestGatherClusterDatasetsSkipsUnresolvedAndFailsWhenEmpty(t *testing.T) {
	docs := &fakeSkillDocs{docs: map[string]string{"e1": "fix"}}
	dataset, ids, err := gatherClusterDatasets(corpusCluster(), docs)
	if err != nil {
		t.Fatalf("gather: %v", err)
	}
	if len(dataset) != 1 || len(ids) != 1 {
		t.Fatalf("expected only the resolved member, got %d records %v", len(dataset), ids)
	}

	empty := &fakeSkillDocs{docs: map[string]string{}}
	if _, _, err := gatherClusterDatasets(corpusCluster(), empty); err == nil {
		t.Fatal("a cluster with no resolvable fix must error, not yield an empty corpus")
	}
}

func TestTargetModelsForCorpus(t *testing.T) {
	spec := ULoRATargetModel{Family: "llama", ParamCount: "8b", HiddenSize: 4096, NumLayers: 32}
	dataset := []ULoRADatasetRecord{
		{TargetModel: "llama-3-8b"},
		{TargetModel: "llama-3-8b"},
	}

	models, err := targetModelsForCorpus(dataset, map[string]ULoRATargetModel{"llama-3-8b": spec})
	if err != nil {
		t.Fatalf("target models: %v", err)
	}
	if len(models) != 1 {
		t.Fatalf("expected one distinct model, got %d", len(models))
	}

	// An unknown model must not be guessed at: inventing hidden sizes would
	// produce an adapter for a model that does not exist.
	if _, err := targetModelsForCorpus([]ULoRADatasetRecord{{TargetModel: "mystery"}}, nil); err == nil {
		t.Fatal("unknown target model must error")
	}

	// An incomplete spec is equally unusable.
	if _, err := targetModelsForCorpus(dataset, map[string]ULoRATargetModel{"llama-3-8b": {Family: "llama"}}); err == nil {
		t.Fatal("incomplete architecture spec must error")
	}
}

// newUnixServer serves handler on a temporary Unix socket, the same transport
// KNIRVULORA and KNIRVCHAIN use.
func newUnixServer(t *testing.T, handler http.Handler) (socketPath string) {
	t.Helper()
	dir := t.TempDir()
	socketPath = filepath.Join(dir, "test.sock")
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("listen unix: %v", err)
	}
	server := &httptest.Server{Listener: listener, Config: &http.Server{Handler: handler}}
	server.Start()
	t.Cleanup(server.Close)
	return socketPath
}

func TestKNIRVULORAClientCompile(t *testing.T) {
	var gotPath string
	var gotRequest uloraCompileRequest

	socketPath := newUnixServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&gotRequest); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(ULoRACompileResult{
			BundleID:     "cluster-1",
			ContentHash:  strings.Repeat("a", 64),
			ManifestPath: "/tmp/appdata/ulora/bundles/cluster-1-abc/manifest.json",
			WeightsPath:  "/tmp/appdata/ulora/bundles/cluster-1-abc/core_weights.safetensors",
			PathUsed:     "B",
			TargetModels: []string{"llama-3-8b"},
		})
	}))

	client := NewKNIRVULORAClient(socketPath)
	result, err := client.CompileCluster(context.Background(), "cluster-1",
		[]ULoRADatasetRecord{{Context: "ctx", CorrectedCompletion: "fix", TargetModel: "llama-3-8b"}},
		[]ULoRATargetModel{{Family: "llama", HiddenSize: 4096, NumLayers: 32}},
		ULoRAProvenance{SourceID: "cluster-1", SourceDatasetIDs: []string{"e1"}})
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if gotPath != uloraCompileRoute {
		t.Fatalf("posted to %q, want %q", gotPath, uloraCompileRoute)
	}
	if len(gotRequest.Dataset) != 1 || gotRequest.Provenance.SourceID != "cluster-1" {
		t.Fatalf("request body wrong: %+v", gotRequest)
	}
	if result.BundleID != "cluster-1" {
		t.Fatalf("bundle id = %q", result.BundleID)
	}
}

// A compile that reports no output must not be treated as success, or the caller
// goes looking for bundle files that were never produced.
func TestKNIRVULORAClientRejectsIncompleteResult(t *testing.T) {
	socketPath := newUnixServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(ULoRACompileResult{BundleID: "cluster-1"})
	}))

	client := NewKNIRVULORAClient(socketPath)
	_, err := client.CompileCluster(context.Background(), "cluster-1",
		[]ULoRADatasetRecord{{Context: "c", CorrectedCompletion: "f", TargetModel: "m"}},
		[]ULoRATargetModel{{Family: "llama", HiddenSize: 4096, NumLayers: 32}},
		ULoRAProvenance{})
	if err == nil {
		t.Fatal("a result with no content hash must be rejected")
	}
}

func TestKNIRVULORAClientRejectsEmptyInputs(t *testing.T) {
	client := NewKNIRVULORAClient("/nonexistent.sock")
	if _, err := client.CompileCluster(context.Background(), "c", nil, nil, ULoRAProvenance{}); err == nil {
		t.Fatal("empty dataset must be rejected before any request")
	}
}

// The gate that matters most: a configured compiler with no validation backend
// must stop the mint, because §2.2 forbids publishing on a placeholder validator.
func TestMintULoRABundleRequiresValidator(t *testing.T) {
	smp := &SkillMintingProtocol{
		uloraCompiler:   &KNIRVULORAClient{},
		manifestVersion: "1.0.0",
		// bundleValidator deliberately nil
	}
	err := smp.mintULoRABundle(context.Background(), corpusCluster(), &SkillNode{ID: "skill-1"})
	if !errors.Is(err, ErrDVEValidationUnavailable) {
		t.Fatalf("err = %v, want ErrDVEValidationUnavailable", err)
	}
}

func TestMintULoRABundleRequiresManifestVersion(t *testing.T) {
	smp := &SkillMintingProtocol{
		uloraCompiler:   &KNIRVULORAClient{},
		bundleValidator: stubValidator{},
	}
	if err := smp.mintULoRABundle(context.Background(), corpusCluster(), &SkillNode{ID: "skill-1"}); err == nil {
		t.Fatal("a missing manifest version must be rejected")
	}
}

type stubValidator struct{ err error }

func (s stubValidator) ValidateBundle(context.Context, *ErrorCluster, *ULoRACompileResult, []ULoRADatasetRecord) error {
	return s.err
}

func sha256HexOf(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// ReadCompiledBundle derives the archive path from the manifest path and refuses
// a bundle whose bytes do not match the compiler's reported hash.
func TestReadCompiledBundle(t *testing.T) {
	dir := t.TempDir()
	bundleDir := filepath.Join(dir, "bundles", "cluster-1-abc")
	if err := os.MkdirAll(bundleDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	manifestPath := filepath.Join(bundleDir, "manifest.json")
	if err := os.WriteFile(manifestPath, []byte("{}"), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	bundlePath := filepath.Join(dir, "bundles", "ulora-cluster-1.ulora")
	payload := []byte("not-really-a-tar")
	if err := os.WriteFile(bundlePath, payload, 0o644); err != nil {
		t.Fatalf("write bundle: %v", err)
	}

	good := ULoRACompileResult{
		BundleID:     "cluster-1",
		ManifestPath: manifestPath,
		ContentHash:  sha256HexOf(payload),
	}
	got, err := ReadCompiledBundle(&good)
	if err != nil {
		t.Fatalf("read bundle: %v", err)
	}
	if string(got) != string(payload) {
		t.Fatalf("bytes = %q, want %q", got, payload)
	}

	// A hash that does not describe the bytes must fail rather than be uploaded.
	bad := good
	bad.ContentHash = strings.Repeat("b", 64)
	if _, err := ReadCompiledBundle(&bad); err == nil {
		t.Fatal("a bundle whose bytes do not match the reported hash must be rejected")
	}

	// A missing archive must fail rather than silently minting nothing.
	missing := good
	missing.BundleID = "cluster-absent"
	if _, err := ReadCompiledBundle(&missing); err == nil {
		t.Fatal("a missing bundle archive must be rejected")
	}
}
