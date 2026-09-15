package daemon

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"ulora/internal/api"
	"ulora/internal/bundling"
	"ulora/internal/runtime"
)

// BindRequest asks for a bundle to be projected onto a target architecture.
//
// This is the per-request adapter primitive: the caller names the bundle and the
// model it wants the adapter for, and gets back the LoRA A/B matrices to apply,
// without any model being loaded here. KNIRV-side callers reach it over this
// socket rather than importing the module, per ulora_implementation.md §2.4.
type BindRequest struct {
	// BundlePath is the bundle archive to bind. Either this or BundleID is
	// required.
	BundlePath string `json:"bundle_path,omitempty"`
	// BundleID resolves <data_dir>/bundles/ulora-<BundleID>.ulora, which is
	// where the compiler writes bundles.
	BundleID string `json:"bundle_id,omitempty"`
	// ContentHash, when supplied, must match the bundle actually read. This is
	// what lets a caller assert it is binding the bundle it thinks it is —
	// and what a validator can later attribute a verdict to. Must match despite
	// the name: it is a plain check for a wrong or stale bundle.
	ContentHash string `json:"content_hash,omitempty"`
	// TargetModel is the architecture to project onto.
	TargetModel api.BaseModelSpec `json:"target_model"`
}

// BindResponse is the projected adapter plus the provenance a caller needs to
// record what was actually executed.
type BindResponse struct {
	// ContentHash is computed from the bundle bytes on disk, not echoed from the
	// request, so it always describes what was really bound.
	ContentHash string `json:"content_hash"`
	BundleID    string `json:"bundle_id,omitempty"`
	BundlePath  string `json:"bundle_path"`

	ManifestVersion string `json:"manifest_version,omitempty"`
	ManifestName    string `json:"manifest_name,omitempty"`

	AdapterRank   int     `json:"adapter_rank"`
	ScalingAlpha  float64 `json:"scaling_alpha"`
	TargetModules int     `json:"target_modules"`

	// Layers maps a target module name to its LoRA A/B matrices.
	Layers map[string]runtime.LayerWeight `json:"layers"`
}

// handleBind projects a bundle onto a target model's architecture.
func (s *Server) handleBind(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req BindRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	bundlePath, err := s.resolveBundlePath(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Hash the bytes we are about to use rather than trusting the request, then
	// fail loudly if the caller expected different content. Binding a stale or
	// wrong bundle is exactly the failure that would silently mis-train or
	// mis-validate, so it is refused rather than warned about.
	contentHash, err := bundling.ContentHashOfFile(bundlePath)
	if err != nil {
		http.Error(w, fmt.Sprintf("hash bundle: %v", err), http.StatusInternalServerError)
		return
	}
	if declared := strings.TrimSpace(req.ContentHash); declared != "" && !strings.EqualFold(declared, contentHash) {
		http.Error(w, fmt.Sprintf("bundle content hash mismatch: %s hashes to %s, caller expected %s",
			bundlePath, contentHash, declared), http.StatusConflict)
		return
	}

	binder, err := runtime.NewBinder(
		runtime.WithBundlePath(bundlePath),
		// Connectors come from the runtime's per-model cache: they belong to the
		// model, not to the bundle, so one skill binds to every model the cache
		// can serve without the artifact embedding a projection per target.
		runtime.WithConnectorProvider(s.connectorCache()),
	)
	if err != nil {
		http.Error(w, fmt.Sprintf("open bundle: %v", err), http.StatusUnprocessableEntity)
		return
	}
	manifest := binder.Manifest()
	if manifest == nil {
		http.Error(w, "bundle manifest is missing", http.StatusUnprocessableEntity)
		return
	}

	layers, err := binder.Bind(req.TargetModel)
	if err != nil {
		// A topology mismatch is the caller's problem, not an internal fault.
		http.Error(w, fmt.Sprintf("bind to target model: %v", err), http.StatusUnprocessableEntity)
		return
	}

	resp := BindResponse{
		ContentHash:     contentHash,
		BundleID:        strings.TrimSpace(req.BundleID),
		BundlePath:      bundlePath,
		ManifestVersion: manifest.Version,
		ManifestName:    manifest.Name,
		AdapterRank:     manifest.CanonicalCore.AdapterRank,
		ScalingAlpha:    manifest.CanonicalCore.ScalingFactorAlpha,
		TargetModules:   len(manifest.CanonicalCore.TargetModules),
		Layers:          layers,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// resolveBundlePath turns a request into a concrete bundle path, refusing
// ambiguous or missing input rather than guessing.
func (s *Server) resolveBundlePath(req BindRequest) (string, error) {
	explicit := strings.TrimSpace(req.BundlePath)
	bundleID := strings.TrimSpace(req.BundleID)

	if explicit != "" && bundleID != "" {
		// Both were given; they must agree, otherwise the caller is confused
		// about which bundle it is asking for.
		derived := s.bundlePathForID(bundleID)
		if derived != explicit {
			return "", fmt.Errorf("bundle_path %q and bundle_id %q disagree", explicit, bundleID)
		}
	}
	if explicit != "" {
		if _, err := os.Stat(explicit); err != nil {
			return "", fmt.Errorf("bundle_path %q is not readable: %w", explicit, err)
		}
		return explicit, nil
	}
	if bundleID == "" {
		return "", fmt.Errorf("either bundle_path or bundle_id is required")
	}

	derived := s.bundlePathForID(bundleID)
	if _, err := os.Stat(derived); err != nil {
		return "", fmt.Errorf("no bundle for id %q at %s", bundleID, derived)
	}
	return derived, nil
}

// bundlePathForID mirrors the compiler's output naming:
// <data_dir>/bundles/ulora-<bundle_id>.ulora.
func (s *Server) bundlePathForID(bundleID string) string {
	return filepath.Join(s.dataDir(), "bundles", fmt.Sprintf("ulora-%s.ulora", bundleID))
}

// dataDir reports the bundle root. Config owns this; the accessor exists so the
// naming scheme lives in exactly one place.
func (s *Server) dataDir() string {
	return s.cfg.DataDir
}
