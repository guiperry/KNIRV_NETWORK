package blockchain

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"KNIRVCHAIN/internal/database"
	"KNIRVCHAIN/internal/ulora"
)

// newULoRAMintServer builds a BlockchainServer backed by a real temp LevelDB and
// a real temp app-data directory, so registration exercises genuine persistence
// and the genuine content-addressed blob store rather than doubles.
func newULoRAMintServer(t *testing.T) (*BlockchainServer, string) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("KNIRV_APP_DATA_DIR", dir)

	db, err := database.NewLevelDB(filepath.Join(dir, "chain.db"))
	if err != nil {
		t.Fatalf("open leveldb: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	return &BlockchainServer{db: db, testMode: true}, dir
}

// mintBody builds a valid registration request for bundleBytes.
func mintBody(t *testing.T, bundleID string, bundleBytes []byte, declaredHash string) []byte {
	t.Helper()
	request := map[string]any{
		"schema_version":    uloraBundleMintSchema,
		"bundle_id":         bundleID,
		"source_cluster_id": "cluster-1",
		"source_skill_ids":  []string{"skill-1"},
		"target_models":     []string{"llama-3-8b"},
		"manifest_version":  "1.0.0",
		"content_hash":      declaredHash,
		"bundle":            base64.StdEncoding.EncodeToString(bundleBytes),
	}
	raw, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	return raw
}

func postMint(server *BlockchainServer, body []byte) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ulora-bundles/mint", bytes.NewReader(body))
	req.Header.Set("X-KNIRV-Internal-Token", "test-internal-token")
	server.handleULoRABundleMint(rec, req)
	return rec
}

func TestULoRABundleMintRequiresInternalToken(t *testing.T) {
	t.Setenv("KNIRV_INTERNAL_AUTH_TOKEN", "test-internal-token")
	server, _ := newULoRAMintServer(t)

	t.Run("non-POST", func(t *testing.T) {
		rec := httptest.NewRecorder()
		server.handleULoRABundleMint(rec, httptest.NewRequest(http.MethodGet, "/api/v1/ulora-bundles/mint", nil))
		if rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("status = %d, want 405", rec.Code)
		}
	})

	t.Run("wrong token", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/ulora-bundles/mint", strings.NewReader(`{}`))
		req.Header.Set("X-KNIRV-Internal-Token", "wrong")
		server.handleULoRABundleMint(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", rec.Code)
		}
	})

	t.Run("fails closed when unconfigured", func(t *testing.T) {
		// A missing server-side token must refuse, not open the route.
		t.Setenv("KNIRV_INTERNAL_AUTH_TOKEN", "")
		server2, _ := newULoRAMintServer(t)
		rec := postMint(server2, []byte(`{}`))
		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("status = %d, want 503", rec.Code)
		}
	})
}

// A declared hash that does not describe the received bytes must be rejected,
// and nothing may be written — otherwise ContentHash stops describing stored
// bytes, which is the whole point of content addressing.
func TestULoRABundleMintRejectsHashMismatch(t *testing.T) {
	t.Setenv("KNIRV_INTERNAL_AUTH_TOKEN", "test-internal-token")
	server, dir := newULoRAMintServer(t)

	bundleBytes := []byte("ulora-bundle-bytes")
	rec := postMint(server, mintBody(t, "bundle-bad-hash", bundleBytes, strings.Repeat("a", 64)))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 for a hash mismatch", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "content hash mismatch") {
		t.Fatalf("expected a hash-mismatch error, got %q", rec.Body.String())
	}

	// Nothing stored, no pointer recorded.
	store, err := ulora.OpenDefault(filepath.Join(dir, "chain.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	entries, _ := os.ReadDir(store.Root())
	if len(entries) != 0 {
		t.Fatalf("blob store has %d entries after a rejected mint, want 0", len(entries))
	}
	if _, found, _ := server.uloraBundlePointerByID("bundle-bad-hash"); found {
		t.Fatal("a rejected mint must not record a pointer")
	}
}

func TestULoRABundleMintValidatesRequest(t *testing.T) {
	t.Setenv("KNIRV_INTERNAL_AUTH_TOKEN", "test-internal-token")
	server, _ := newULoRAMintServer(t)
	bundle := []byte("bytes")
	digest := sha256.Sum256(bundle)
	good := hex.EncodeToString(digest[:])

	cases := []struct {
		name   string
		mutate func(map[string]any)
	}{
		{"missing schema", func(m map[string]any) { m["schema_version"] = "" }},
		{"wrong schema", func(m map[string]any) { m["schema_version"] = "other" }},
		{"missing bundle id", func(m map[string]any) { m["bundle_id"] = "" }},
		{"missing cluster id", func(m map[string]any) { m["source_cluster_id"] = "" }},
		{"missing manifest version", func(m map[string]any) { m["manifest_version"] = "" }},
		{"no source skills", func(m map[string]any) { m["source_skill_ids"] = []string{} }},
		{"no target models", func(m map[string]any) { m["target_models"] = []string{} }},
		{"empty bundle", func(m map[string]any) { m["bundle"] = "" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var request map[string]any
			if err := json.Unmarshal(mintBody(t, "bundle-1", bundle, good), &request); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			tc.mutate(request)
			raw, _ := json.Marshal(request)
			if rec := postMint(server, raw); rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400 (%s)", rec.Code, rec.Body.String())
			}
		})
	}
}

// The end-to-end point of this route: register bytes, then stream them back
// through the reader, with the pointer's hash describing the served bytes.
func TestULoRABundleMintStoresAndServes(t *testing.T) {
	t.Setenv("KNIRV_INTERNAL_AUTH_TOKEN", "test-internal-token")
	server, dir := newULoRAMintServer(t)

	bundleBytes := []byte("a-real-ulora-bundle")
	digest := sha256.Sum256(bundleBytes)
	expectedHash := hex.EncodeToString(digest[:])

	rec := postMint(server, mintBody(t, "bundle-1", bundleBytes, expectedHash))
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201: %s", rec.Code, rec.Body.String())
	}

	var receipt uloraBundleMintReceipt
	if err := json.Unmarshal(rec.Body.Bytes(), &receipt); err != nil {
		t.Fatalf("decode receipt: %v", err)
	}
	if receipt.ContentHash != expectedHash {
		t.Fatalf("receipt hash = %q, want the recomputed %q", receipt.ContentHash, expectedHash)
	}
	if receipt.Pointer.CMU != "knirv://network/ulora_"+expectedHash {
		t.Fatalf("CMU = %q", receipt.Pointer.CMU)
	}
	if receipt.AlreadyRegistered {
		t.Fatal("first registration must not report already_registered")
	}

	// The bytes are on disk under their real digest, and readable back.
	store, err := ulora.OpenDefault(filepath.Join(dir, "chain.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	file, err := store.Open(expectedHash)
	if err != nil {
		t.Fatalf("stored bundle not readable at its hash: %v", err)
	}
	defer file.Close()
	served, err := os.ReadFile(file.Name())
	if err != nil {
		t.Fatalf("read stored bundle: %v", err)
	}
	if !bytes.Equal(served, bundleBytes) {
		t.Fatal("served bytes differ from the registered bundle")
	}

	// And the pointer is persisted for later lookup.
	pointer, found, err := server.uloraBundlePointerByID("bundle-1")
	if err != nil || !found {
		t.Fatalf("pointer not persisted (found=%v err=%v)", found, err)
	}
	if pointer.ContentHash != expectedHash {
		t.Fatalf("persisted hash = %q, want %q", pointer.ContentHash, expectedHash)
	}
}

func TestULoRABundleMintIsIdempotent(t *testing.T) {
	t.Setenv("KNIRV_INTERNAL_AUTH_TOKEN", "test-internal-token")
	server, _ := newULoRAMintServer(t)

	bundle := []byte("same-bytes")
	digest := sha256.Sum256(bundle)
	hash := hex.EncodeToString(digest[:])
	body := mintBody(t, "bundle-1", bundle, hash)

	if rec := postMint(server, body); rec.Code != http.StatusCreated {
		t.Fatalf("first mint status = %d, want 201", rec.Code)
	}
	rec := postMint(server, body)
	if rec.Code != http.StatusOK {
		t.Fatalf("second mint status = %d, want 200", rec.Code)
	}
	var receipt uloraBundleMintReceipt
	if err := json.Unmarshal(rec.Body.Bytes(), &receipt); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !receipt.AlreadyRegistered {
		t.Fatal("re-registering identical bytes must report already_registered")
	}
}

// Reusing a bundle id for different content must be refused, not silently
// overwritten — the id is the cluster's mint identity.
func TestULoRABundleMintConflictsOnDifferentContent(t *testing.T) {
	t.Setenv("KNIRV_INTERNAL_AUTH_TOKEN", "test-internal-token")
	server, _ := newULoRAMintServer(t)

	first := []byte("first-bytes")
	firstDigest := sha256.Sum256(first)
	if rec := postMint(server, mintBody(t, "bundle-1", first, hex.EncodeToString(firstDigest[:]))); rec.Code != http.StatusCreated {
		t.Fatalf("first mint status = %d, want 201", rec.Code)
	}

	second := []byte("different-bytes")
	secondDigest := sha256.Sum256(second)
	rec := postMint(server, mintBody(t, "bundle-1", second, hex.EncodeToString(secondDigest[:])))
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409 for a reused bundle id with different content", rec.Code)
	}
}
