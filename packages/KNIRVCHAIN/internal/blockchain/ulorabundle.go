package blockchain

import (
	"bytes"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"KNIRVCHAIN/internal/types"
	"KNIRVCHAIN/internal/ulora"
)

// ULoRA bundle registration (ulora_implementation.md §2.1/§2.7).
//
// This is the missing writer for the content-addressed blob store at
// <appDataDir>/ulora_blobs/<sha256>. GET /api/ulora/{hash} could already stream
// a bundle, and internal/ulora could already store one, but nothing ever called
// Put — so the serving route could only ever 404. This route closes that loop:
// KNIRVGRAPH's DRQ pipeline compiles a cluster's `.ulora` via KNIRVULORA
// (POST /ulora/v1/compile-cluster) and registers the resulting bytes here.
//
// The bundle travels as base64 inside JSON, matching the existing
// event-bundle/validation-proof mint idiom (a sibling route on this same
// server, also internal-token gated) rather than introducing a new transport
// pattern. The chain recomputes SHA-256 over the bytes it actually received and
// rejects a mismatch, so ContentHash always describes stored bytes — the
// property §2.1 requires and that internal/mining's cosmetic generateIPFSCID
// lacked.

const (
	// uloraBundleMintSchema versions the wire request.
	uloraBundleMintSchema = "knirv.ulora-bundle-mint.v1"
	// uloraBundleReceiptSchema versions the response.
	uloraBundleReceiptSchema = "knirv.ulora-bundle-receipt.v1"
	// uloraBundlePointerKeyPrefix stores registered pointers by bundle id.
	uloraBundlePointerKeyPrefix = "ulora_bundle:pointer:"

	// uloraBundleMaxBytes bounds the decoded bundle. Adapter bundles hold
	// safetensors of per-target-model projection matrices; 512MiB is a ceiling
	// that refuses absurd payloads without constraining real ones.
	uloraBundleMaxBytes = 512 << 20
)

// uloraBundleMintRequest is what KNIRVGRAPH posts after a successful compile.
type uloraBundleMintRequest struct {
	SchemaVersion   string   `json:"schema_version"`
	BundleID        string   `json:"bundle_id"`
	SourceClusterID string   `json:"source_cluster_id"`
	SourceSkillIDs  []string `json:"source_skill_ids"`
	TargetModels    []string `json:"target_models"`
	ManifestVersion string   `json:"manifest_version"`
	// ContentHash is the digest the compiler reported for these bytes. It is
	// cross-checked against the digest of the bytes actually received.
	ContentHash string `json:"content_hash"`
	// Bundle is the raw `.ulora` archive, base64-encoded.
	Bundle string `json:"bundle"`
}

// uloraBundleMintReceipt is returned to the caller and is also what the
// serving route's callers see once registration completes.
type uloraBundleMintReceipt struct {
	SchemaVersion string `json:"schema_version"`
	BundleID      string `json:"bundle_id"`
	ContentHash   string `json:"content_hash"`
	Pointer       struct {
		CMU             string   `json:"cmu"`
		SourceClusterID string   `json:"source_cluster_id"`
		SourceSkillIDs  []string `json:"source_skill_ids"`
		TargetModels    []string `json:"target_models"`
		ManifestVersion string   `json:"manifest_version"`
		CreatedAt       int64    `json:"created_at"`
	} `json:"pointer"`
	// AlreadyRegistered is true when an identical bundle was already
	// registered, making the mint idempotent per (bundle_id, content_hash).
	AlreadyRegistered bool `json:"already_registered,omitempty"`
}

// handleULoRABundleMint registers a compiled `.ulora` bundle.
func (bcs *BlockchainServer) handleULoRABundleMint(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if bcs.checkNetworkPauseAndRejectIfPaused(w, "ulora_bundle_mint") {
		return
	}

	// Same internal-service-token gate as the sibling mint routes: an unset
	// token refuses rather than opening the route.
	expectedToken := strings.TrimSpace(os.Getenv("KNIRV_INTERNAL_AUTH_TOKEN"))
	providedToken := strings.TrimSpace(r.Header.Get("X-KNIRV-Internal-Token"))
	if expectedToken == "" {
		http.Error(w, "internal ulora bundle registration is not configured", http.StatusServiceUnavailable)
		return
	}
	if subtle.ConstantTimeCompare([]byte(expectedToken), []byte(providedToken)) != 1 {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, uloraBundleMaxEncodedBytes())
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var request uloraBundleMintRequest
	if err := decoder.Decode(&request); err != nil {
		http.Error(w, "invalid ulora bundle registration: "+err.Error(), http.StatusBadRequest)
		return
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		http.Error(w, "ulora bundle registration must contain one JSON value", http.StatusBadRequest)
		return
	}
	if err := validateULoRABundleMint(request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	bundleBytes, err := base64.StdEncoding.DecodeString(request.Bundle)
	if err != nil {
		http.Error(w, "bundle is not valid base64", http.StatusBadRequest)
		return
	}
	if len(bundleBytes) == 0 {
		http.Error(w, "bundle is empty", http.StatusBadRequest)
		return
	}
	if len(bundleBytes) > uloraBundleMaxBytes {
		http.Error(w, fmt.Sprintf("bundle exceeds %d bytes", uloraBundleMaxBytes), http.StatusRequestEntityTooLarge)
		return
	}

	// Content addressing is computed here, over exactly the bytes received —
	// never trusted from the caller. A mismatch means the bundle was corrupted
	// or truncated in transit, which must not be registered as if intact.
	digest := sha256.Sum256(bundleBytes)
	computedHash := hex.EncodeToString(digest[:])
	if !strings.EqualFold(computedHash, strings.TrimSpace(request.ContentHash)) {
		http.Error(w, fmt.Sprintf("content hash mismatch: declared %s, received bytes hash to %s",
			strings.TrimSpace(request.ContentHash), computedHash), http.StatusBadRequest)
		return
	}

	pointer, err := types.NewULoRABundlePointer(
		request.BundleID, computedHash, request.SourceClusterID,
		request.SourceSkillIDs, request.TargetModels, request.ManifestVersion,
	)
	if err != nil {
		http.Error(w, "invalid bundle pointer: "+err.Error(), http.StatusBadRequest)
		return
	}

	bcs.validationProofMu.Lock()
	defer bcs.validationProofMu.Unlock()

	// Idempotent per bundle id: re-registering the same bytes returns the
	// existing pointer instead of duplicating it, and a *different* payload
	// under an already-used id is refused rather than silently overwritten.
	if existing, found, err := bcs.uloraBundlePointerByID(request.BundleID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else if found {
		if !strings.EqualFold(existing.ContentHash, computedHash) {
			http.Error(w, fmt.Sprintf("bundle %s is already registered with a different content hash", request.BundleID),
				http.StatusConflict)
			return
		}
		writeULoRABundleReceipt(w, http.StatusOK, existing, true)
		return
	}

	store, err := ulora.OpenDefault(bcs.databasePath())
	if err != nil {
		http.Error(w, "bundle store unavailable: "+err.Error(), http.StatusInternalServerError)
		return
	}
	storedHash, err := store.Put(bytes.NewReader(bundleBytes))
	if err != nil {
		http.Error(w, "store bundle: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if !strings.EqualFold(storedHash, computedHash) {
		// The store hashes the bytes it wrote; a disagreement means the file on
		// disk is not what was digested, so the pointer must not be recorded.
		http.Error(w, fmt.Sprintf("bundle store hash mismatch: stored %s, computed %s", storedHash, computedHash),
			http.StatusInternalServerError)
		return
	}

	if err := bcs.saveULoRABundlePointer(pointer); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeULoRABundleReceipt(w, http.StatusCreated, pointer, false)
}

// databasePath exposes the chain database's path so the blob store can be
// resolved relative to the same app data directory the reader uses.
func (bcs *BlockchainServer) databasePath() string {
	if bcs == nil || bcs.db == nil {
		return ""
	}
	return bcs.db.Path()
}

// uloraBundlePointerByID looks up a registered pointer.
func (bcs *BlockchainServer) uloraBundlePointerByID(bundleID string) (*types.ULoRABundlePointer, bool, error) {
	if bcs.db == nil {
		return nil, false, fmt.Errorf("blockchain database is not available")
	}
	key := uloraBundlePointerKeyPrefix + bundleID
	exists, err := bcs.db.KeyExists(key)
	if err != nil || !exists {
		return nil, false, err
	}
	raw, err := bcs.db.GetBytes(key)
	if err != nil {
		return nil, false, err
	}
	var pointer types.ULoRABundlePointer
	if err := json.Unmarshal(raw, &pointer); err != nil {
		return nil, false, err
	}
	return &pointer, true, nil
}

// saveULoRABundlePointer persists a registered pointer.
func (bcs *BlockchainServer) saveULoRABundlePointer(pointer *types.ULoRABundlePointer) error {
	if bcs.db == nil {
		return fmt.Errorf("blockchain database is not available")
	}
	raw, err := json.Marshal(pointer)
	if err != nil {
		return fmt.Errorf("marshal ulora bundle pointer: %w", err)
	}
	if err := bcs.db.PutBytes(uloraBundlePointerKeyPrefix+pointer.BundleID, raw); err != nil {
		return fmt.Errorf("persist ulora bundle pointer: %w", err)
	}
	return nil
}

// writeULoRABundleReceipt renders the registration receipt.
func writeULoRABundleReceipt(w http.ResponseWriter, status int, pointer *types.ULoRABundlePointer, already bool) {
	receipt := uloraBundleMintReceipt{
		SchemaVersion:     uloraBundleReceiptSchema,
		BundleID:          pointer.BundleID,
		ContentHash:       pointer.ContentHash,
		AlreadyRegistered: already,
	}
	receipt.Pointer.CMU = pointer.CMU
	receipt.Pointer.SourceClusterID = pointer.SourceClusterID
	receipt.Pointer.SourceSkillIDs = pointer.SourceSkillIDs
	receipt.Pointer.TargetModels = pointer.TargetModels
	receipt.Pointer.ManifestVersion = pointer.ManifestVersion
	receipt.Pointer.CreatedAt = pointer.CreatedAt
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(receipt)
}

// validateULoRABundleMint checks required fields before any bytes are stored,
// so a malformed request cannot leave a blob on disk with no pointer.
func validateULoRABundleMint(request uloraBundleMintRequest) error {
	if request.SchemaVersion != uloraBundleMintSchema {
		return fmt.Errorf("unsupported schema_version %q, want %q", request.SchemaVersion, uloraBundleMintSchema)
	}
	if strings.TrimSpace(request.BundleID) == "" {
		return fmt.Errorf("bundle_id is required")
	}
	if strings.TrimSpace(request.SourceClusterID) == "" {
		return fmt.Errorf("source_cluster_id is required")
	}
	if strings.TrimSpace(request.ManifestVersion) == "" {
		return fmt.Errorf("manifest_version is required")
	}
	if len(request.SourceSkillIDs) == 0 {
		return fmt.Errorf("source_skill_ids cannot be empty")
	}
	if len(request.TargetModels) == 0 {
		return fmt.Errorf("target_models cannot be empty")
	}
	if strings.TrimSpace(request.ContentHash) == "" {
		return fmt.Errorf("content_hash is required")
	}
	if strings.TrimSpace(request.Bundle) == "" {
		return fmt.Errorf("bundle is required")
	}
	return nil
}

// uloraBundleMaxEncodedBytes is the request-body ceiling: base64 inflates the
// payload by 4/3, plus JSON overhead.
func uloraBundleMaxEncodedBytes() int64 {
	return int64(uloraBundleMaxBytes/3*4) + (1 << 20)
}
