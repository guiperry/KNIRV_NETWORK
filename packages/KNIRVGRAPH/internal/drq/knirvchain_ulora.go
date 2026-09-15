package drq

import (
	"bytes"

	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Registration of a compiled `.ulora` bundle on KNIRVCHAIN.
//
// The chain stores the bytes in its content-addressed blob store and records a
// ULoRABundlePointer; it recomputes SHA-256 over what it received and rejects a
// mismatch, so the hash always describes stored bytes. The bytes travel base64
// inside JSON over the chain's Unix socket, matching the event-bundle mint idiom
// rather than introducing a second transport.
//
// The bundle file is written by KNIRVULORA under the shared app-data directory
// that KNIRVSERVER gives ulorad, KNIRVGRAPH and KNIRVCHAIN, which is why this can
// read what the daemon produced.

const (
	knirvchainULoRABundleMintRoute = "/api/v1/ulora-bundles/mint"
	// knirvchainULoRABundleMintSchema versions the mint request; the chain
	// rejects anything else.
	knirvchainULoRABundleMintSchema = "knirv.ulora-bundle-mint.v1"
	// knirvchainULoRABundleMaxBytes mirrors the chain's own ceiling.
	knirvchainULoRABundleMaxBytes = 512 << 20
	// knirvchainULoRAMintTimeout bounds the upload. Adapter bundles are
	// safetensors archives, so this is far longer than an ordinary RPC.
	knirvchainULoRAMintTimeout = 10 * time.Minute
)

type knirvchainULoRABundleMintRequest struct {
	SchemaVersion   string   `json:"schema_version"`
	BundleID        string   `json:"bundle_id"`
	SourceClusterID string   `json:"source_cluster_id"`
	SourceSkillIDs  []string `json:"source_skill_ids"`
	TargetModels    []string `json:"target_models"`
	ManifestVersion string   `json:"manifest_version"`
	ContentHash     string   `json:"content_hash"`
	Bundle          string   `json:"bundle"`
}

type knirvchainULoRABundleReceipt struct {
	BundleID    string `json:"bundle_id"`
	ContentHash string `json:"content_hash"`
	Pointer     struct {
		CMU             string `json:"cmu"`
		SourceClusterID string `json:"source_cluster_id"`
	} `json:"pointer"`
}

// MintULoRABundle registers a compiled bundle on KNIRVCHAIN. Idempotent per
// (bundleID, contentHash) on the chain side.
func (kc *KNIRVCHAINClient) MintULoRABundle(clusterID, manifestVersion string, bundleBytes []byte, sourceSkillIDs, targetModels []string, contentHash string) ([]byte, error) {
	if kc == nil {
		return nil, errors.New("no KNIRVCHAIN client configured")
	}
	clusterID = strings.TrimSpace(clusterID)
	if clusterID == "" {
		return nil, errors.New("cluster id is required to mint a uLoRA bundle")
	}
	if len(bundleBytes) == 0 {
		return nil, errors.New("bundle bytes are required to mint a uLoRA bundle")
	}
	if len(bundleBytes) > knirvchainULoRABundleMaxBytes {
		return nil, fmt.Errorf("bundle is %d bytes, over the %d-byte limit", len(bundleBytes), knirvchainULoRABundleMaxBytes)
	}
	if len(sourceSkillIDs) == 0 {
		return nil, errors.New("source skill ids are required to mint a uLoRA bundle")
	}
	if len(targetModels) == 0 {
		return nil, errors.New("target models are required to mint a uLoRA bundle")
	}
	if strings.TrimSpace(manifestVersion) == "" {
		return nil, errors.New("manifest version is required to mint a uLoRA bundle")
	}

	// Recompute locally and send the digest of the bytes actually being sent, so
	// a mismatch is caught before an upload rather than by the chain afterwards.
	digest := sha256.Sum256(bundleBytes)
	computed := hex.EncodeToString(digest[:])
	if declared := strings.TrimSpace(contentHash); declared != "" && !strings.EqualFold(declared, computed) {
		return nil, fmt.Errorf("bundle content hash mismatch: compiler reported %s, bytes hash to %s", declared, computed)
	}

	token := strings.TrimSpace(os.Getenv("KNIRV_INTERNAL_AUTH_TOKEN"))
	if token == "" {
		return nil, errors.New("KNIRV_INTERNAL_AUTH_TOKEN is not configured")
	}

	request := knirvchainULoRABundleMintRequest{
		SchemaVersion:   knirvchainULoRABundleMintSchema,
		BundleID:        clusterID,
		SourceClusterID: clusterID,
		SourceSkillIDs:  sourceSkillIDs,
		TargetModels:    targetModels,
		ManifestVersion: manifestVersion,
		ContentHash:     computed,
		Bundle:          base64.StdEncoding.EncodeToString(bundleBytes),
	}
	body, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("marshal uLoRA bundle mint request: %w", err)
	}

	httpRequest, err := http.NewRequest(http.MethodPost, knirvchainSocketSchemeAndHost+knirvchainULoRABundleMintRoute, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build uLoRA bundle mint request: %w", err)
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("X-KNIRV-Internal-Token", token)

	client := knirvchainUnixHTTPClient(knirvchainSocketPath(), knirvchainULoRAMintTimeout)
	response, err := client.Do(httpRequest)
	if err != nil {
		return nil, fmt.Errorf("mint uLoRA bundle on KNIRVCHAIN: %w", err)
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("read uLoRA bundle mint response: %w", err)
	}
	if response.StatusCode/100 != 2 {
		return nil, fmt.Errorf("KNIRVCHAIN rejected uLoRA bundle mint (%d): %s", response.StatusCode, strings.TrimSpace(string(responseBody)))
	}

	var receipt knirvchainULoRABundleReceipt
	if err := json.Unmarshal(responseBody, &receipt); err != nil {
		return nil, fmt.Errorf("decode uLoRA bundle mint receipt: %w", err)
	}
	if receipt.BundleID != clusterID {
		return nil, fmt.Errorf("KNIRVCHAIN returned a receipt for a different bundle (%s)", receipt.BundleID)
	}
	// The chain hashes what it received; if that disagrees with what we sent, the
	// pointer does not describe these bytes and the caller must not treat the
	// mint as done.
	if !strings.EqualFold(receipt.ContentHash, computed) {
		return nil, fmt.Errorf("KNIRVCHAIN registered a different hash (%s) than the uploaded bytes (%s)", receipt.ContentHash, computed)
	}

	return responseBody, nil
}

// ReadCompiledBundle reads the `.ulora` archive a compile produced.
//
// The compiler response reports the manifest and weights paths but not the
// archive path; the layout is fixed (bundles/ulora-<clusterID>.ulora beside the
// bundle directory), so it is derived from the manifest path. The bytes are
// re-hashed and checked against the compiler's reported hash before being
// returned, so a stale or partial file fails here rather than being registered
// on-chain under a wrong digest.
func ReadCompiledBundle(result *ULoRACompileResult) ([]byte, error) {
	if result == nil {
		return nil, errors.New("compile result is required")
	}
	manifestPath := strings.TrimSpace(result.ManifestPath)
	if manifestPath == "" {
		return nil, errors.New("compile result has no manifest path")
	}
	if strings.TrimSpace(result.BundleID) == "" {
		return nil, errors.New("compile result has no bundle id")
	}

	// The archive sits beside the per-bundle directory, not inside it:
	//   <dataDir>/bundles/<clusterID>-<hash>/manifest.json   (the manifest)
	//   <dataDir>/bundles/ulora-<clusterID>.ulora            (the archive)
	bundleDir := filepath.Dir(manifestPath)
	bundlesRoot := filepath.Dir(bundleDir)
	if bundlesRoot == "" || bundlesRoot == "." || bundlesRoot == string(filepath.Separator) {
		return nil, fmt.Errorf("cannot derive the uLoRA bundle directory from manifest path %q", manifestPath)
	}
	bundlePath := filepath.Join(bundlesRoot, fmt.Sprintf("ulora-%s.ulora", result.BundleID))

	info, err := os.Stat(bundlePath)
	if err != nil {
		return nil, fmt.Errorf("read compiled uLoRA bundle at %s: %w", bundlePath, err)
	}
	if info.IsDir() {
		return nil, fmt.Errorf("compiled uLoRA bundle path %s is a directory", bundlePath)
	}
	if info.Size() > knirvchainULoRABundleMaxBytes {
		return nil, fmt.Errorf("compiled uLoRA bundle is %d bytes, over the %d-byte limit", info.Size(), knirvchainULoRABundleMaxBytes)
	}

	bundleBytes, err := os.ReadFile(bundlePath)
	if err != nil {
		return nil, fmt.Errorf("read compiled uLoRA bundle %s: %w", bundlePath, err)
	}

	if declared := strings.TrimSpace(result.ContentHash); declared != "" {
		digest := sha256.Sum256(bundleBytes)
		if computed := hex.EncodeToString(digest[:]); !strings.EqualFold(declared, computed) {
			return nil, fmt.Errorf("compiled bundle at %s hashes to %s but the compiler reported %s", bundlePath, computed, declared)
		}
	}
	return bundleBytes, nil
}
