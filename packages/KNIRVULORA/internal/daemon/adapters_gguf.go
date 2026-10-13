package daemon

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"ulora/internal/bundling"
	"ulora/internal/gguf"
	"ulora/internal/llamatarget"
	"ulora/internal/runtime"
)

// GGUFAdapterRequest asks for a bundle to be bound to a llama base model and
// written as a GGUF LoRA adapter llama-server can load.
type GGUFAdapterRequest struct {
	BundlePath  string `json:"bundle_path,omitempty"`
	BundleID    string `json:"bundle_id,omitempty"`
	ContentHash string `json:"content_hash,omitempty"`
	// BaseModelPath is the GGUF the adapter must match. Its sha256 is recorded in
	// the adapter and checked again by the llama side.
	BaseModelPath string `json:"base_model_path"`
}

// GGUFAdapterResponse records exactly what was produced and from what.
type GGUFAdapterResponse struct {
	AdapterPath     string   `json:"adapter_path"`
	BaseSHA256      string   `json:"base_sha256"`
	ContentHash     string   `json:"content_hash"`
	SkillName       string   `json:"skill_name"`
	Architecture    string   `json:"architecture"`
	Rank            int      `json:"rank"`
	Alpha           float64  `json:"alpha"`
	Modules         []string `json:"modules"`
	Layers          int      `json:"layers"`
	ConnectorDigest string   `json:"connector_digest"`
	// Cached is true when this (bundle, base) pair had already been written.
	Cached           bool `json:"cached"`
	ConnectorDerived bool `json:"connector_derived"`
}

var hashMu sync.Mutex

// baseSHA256 hashes a base model once per (path, size, mtime): a 4 GB model takes
// seconds to hash and is asked about on every adapter request.
func (s *Server) baseSHA256(path string) (string, error) {
	st, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	cachePath := filepath.Join(s.dataDir(), "adapters", "hash-cache.json")
	key := fmt.Sprintf("%s|%d|%d", path, st.Size(), st.ModTime().UnixNano())
	hashMu.Lock()
	defer hashMu.Unlock()
	cache := map[string]string{}
	if raw, err := os.ReadFile(cachePath); err == nil {
		_ = json.Unmarshal(raw, &cache)
	}
	if v := cache[key]; v != "" {
		return v, nil
	}
	sum, err := gguf.SHA256(path)
	if err != nil {
		return "", err
	}
	cache[key] = sum
	if raw, err := json.Marshal(cache); err == nil {
		_ = os.MkdirAll(filepath.Dir(cachePath), 0o755)
		_ = os.WriteFile(cachePath, raw, 0o644)
	}
	return sum, nil
}

func (s *Server) handleGGUFAdapter(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req GGUFAdapterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("invalid JSON: %v", err), http.StatusBadRequest)
		return
	}
	resp, status, err := s.buildGGUFAdapter(req)
	if err != nil {
		http.Error(w, err.Error(), status)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (s *Server) buildGGUFAdapter(req GGUFAdapterRequest) (*GGUFAdapterResponse, int, error) {
	if strings.TrimSpace(req.BaseModelPath) == "" {
		return nil, http.StatusBadRequest, fmt.Errorf("base_model_path is required")
	}
	bundlePath, err := s.resolveBundlePath(BindRequest{BundlePath: req.BundlePath, BundleID: req.BundleID})
	if err != nil {
		return nil, http.StatusBadRequest, err
	}
	contentHash, err := bundling.ContentHashOfFile(bundlePath)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("hash bundle: %w", err)
	}
	if d := strings.TrimSpace(req.ContentHash); d != "" && !strings.EqualFold(d, contentHash) {
		return nil, http.StatusConflict, fmt.Errorf("bundle content hash mismatch: %s hashes to %s, caller expected %s", bundlePath, contentHash, d)
	}

	sha, err := s.baseSHA256(req.BaseModelPath)
	if err != nil {
		return nil, http.StatusBadRequest, fmt.Errorf("base model: %w", err)
	}
	target, err := llamatarget.FromGGUF(req.BaseModelPath, sha)
	if err != nil {
		return nil, http.StatusUnprocessableEntity, err
	}

	binder, err := runtime.NewBinder(runtime.WithBundlePath(bundlePath), runtime.WithConnectorProvider(s.connectorCache()))
	if err != nil {
		return nil, http.StatusUnprocessableEntity, fmt.Errorf("open bundle: %w", err)
	}
	manifest := binder.Manifest()
	core := manifest.CanonicalCore

	// Refuse modules this target cannot take BEFORE any expensive work.
	mods := make(map[string]llamatarget.Module, len(core.TargetModules))
	for _, m := range core.TargetModules {
		mod, err := target.Module(m)
		if err != nil {
			return nil, http.StatusUnprocessableEntity, fmt.Errorf("bundle %s cannot be bound to %s: %w", manifest.Name, filepath.Base(req.BaseModelPath), err)
		}
		mods[m] = mod
	}

	out := filepath.Join(s.dataDir(), "adapters", sha[:12], contentHash[:12]+".gguf")
	conn, derived, err := s.ensureGGUFConnector(target, core.CanonicalDim)
	if err != nil {
		return nil, http.StatusUnprocessableEntity, err
	}
	resp := &GGUFAdapterResponse{
		AdapterPath: out, BaseSHA256: sha, ContentHash: contentHash, SkillName: manifest.Name,
		Architecture: target.Arch, Rank: core.AdapterRank, Alpha: core.ScalingFactorAlpha,
		Modules: core.TargetModules, Layers: target.Spec.NumLayers, ConnectorDigest: conn.Digest(), ConnectorDerived: derived,
	}
	if info, _, err := gguf.ReadAdapterInfo(out); err == nil && info.BaseSHA256 == sha && info.BundleHash == contentHash {
		resp.Cached = true
		return resp, http.StatusOK, nil
	}

	bound, err := binder.Bind(target.Spec)
	if err != nil {
		return nil, http.StatusUnprocessableEntity, fmt.Errorf("bind to %s: %w", filepath.Base(req.BaseModelPath), err)
	}
	var layers []gguf.LayerLoRA
	for _, name := range core.TargetModules {
		lw, ok := bound[name]
		if !ok {
			return nil, http.StatusInternalServerError, fmt.Errorf("binder returned no factors for %s", name)
		}
		mod := mods[name]
		if len(lw.A) == 0 || len(lw.A[0]) != mod.In || len(lw.B) != mod.Out {
			return nil, http.StatusInternalServerError, fmt.Errorf("%s: bound factors are A %dx%d / B %dx%d, want A r x %d / B %d x r",
				name, len(lw.A), colsOf(lw.A), len(lw.B), colsOf(lw.B), mod.In, mod.Out)
		}
		// A bundle's core is per semantic module, not per layer, so every layer gets
		// the same projected factors. Per-layer specialisation is a future compiler
		// feature, not something the writer may invent.
		for l := 0; l < target.Spec.NumLayers; l++ {
			layers = append(layers, gguf.LayerLoRA{Weight: mod.WeightName(l), A: lw.A, B: lw.B})
		}
	}
	err = gguf.WriteLoRA(out, gguf.AdapterInfo{
		Architecture: target.Arch, Alpha: float32(core.ScalingFactorAlpha),
		BaseSHA256: sha, BundleHash: contentHash, Skill: manifest.Name, Modules: strings.Join(core.TargetModules, ","),
	}, layers)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}
	return resp, http.StatusOK, nil
}

// ensureGGUFConnector returns the connector cached for this exact base model,
// deriving it from the model's own weights on first use.
func (s *Server) ensureGGUFConnector(t *llamatarget.Target, k int) (conn interface{ Digest() string }, derived bool, err error) {
	cache := s.connectorCache()
	if c, err := cache.Get(t.Spec.Family, t.Spec.ParamCount, k); err != nil {
		return nil, false, err
	} else if c != nil {
		return c, false, nil
	}
	f, err := gguf.Open(t.Path)
	if err != nil {
		return nil, false, err
	}
	m32, err := f.Matrix(t.ConnectorSource())
	if err != nil {
		return nil, false, fmt.Errorf("derive connector: %w", err)
	}
	if len(m32) != t.Spec.HiddenSize || len(m32[0]) != t.Spec.HiddenSize {
		return nil, false, fmt.Errorf("derive connector: %s is %dx%d, expected hidden-sized %d x %d", t.ConnectorSource(), len(m32), len(m32[0]), t.Spec.HiddenSize, t.Spec.HiddenSize)
	}
	m := make([][]float64, len(m32))
	for i, row := range m32 {
		m[i] = make([]float64, len(row))
		for j, v := range row {
			m[i][j] = float64(v)
		}
	}
	c, err := s.compiler.DeriveConnector(t.Spec, m, k, 0)
	if err != nil {
		return nil, false, fmt.Errorf("derive connector from %s: %w", t.ConnectorSource(), err)
	}
	if err := cache.Put(c); err != nil {
		return nil, false, err
	}
	return c, true, nil
}
