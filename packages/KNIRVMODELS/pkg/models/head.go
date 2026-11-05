package models

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"knirvmodels/pkg/vlm"

	sdk "github.com/guiperry/knirv-sdk-go/models"
)

// Head is the registry facade for HEAD after the migration: the vision / multimodal
// model (VLM) and the mirrored text generator. It serves the SDK VLM contract at
// /models/head/* (status, capabilities, embed, describe, ground, visual-reason,
// generate). The decision routes that historically lived here moved to the HEART
// decision core (registerHEARTDecision, mounted at /models/heart/decide) and are
// removed from HEAD in Phase 5.
type Head struct {
	backend vlm.Backend
	start   time.Time
}

// NewHead builds the HEAD facade over a VLM backend. vlm.FromEnv() selects the backend
// from KNIRV_VLM_BACKEND (default "none": HEAD reports no capability and returns a
// structured 503 on every operation, so it never fabricates output).
func NewHead(backend vlm.Backend) *Head { return &Head{backend: backend, start: time.Now()} }

func (h *Head) Name() string { return "head" }

func (h *Head) Status() sdk.ModelStatus {
	ready, detail := h.backend.Ready(context.Background())
	cp := h.backend.Capabilities()
	detailBytes, _ := json.Marshal(cp)
	out := sdk.ModelStatus{
		Name: "head", Role: sdk.RoleVision, Version: h.backend.Name(),
		Ready: ready, Loaded: ready, Trained: ready, Backend: h.backend.Name(),
		Detail: detailBytes,
	}
	if ready {
		out.Since = since(h.start)
	}
	if !ready && detail != "" {
		out.Error = detail
	}
	return out
}

// Mount serves the HEAD VLM contract under prefix (/models/head by default). It does
// not mount the decision routes; those live only behind the HEART decision prefix
// (/models/heart/decide), served by the HEART service's registerHEARTDecision. Phase 3 moves
// HEAD's own artefacts (VLM manifest, promoted weights) onto /models/head/models and
// /models/head/weights/{ver}; the legacy decision manifest/weights that previously
// lived here are removed so the two namespaces cannot collide.
func (h *Head) Mount(mux *http.ServeMux, prefix string) {
	mux.HandleFunc(prefix+"/status", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, vlm.Status(h.backend, r.Context()))
	})
	mux.HandleFunc(prefix+"/capabilities", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, h.backend.Capabilities())
	})
	for _, op := range []string{"embed", "describe", "ground", "visual-reason"} {
		mux.HandleFunc(prefix+"/"+op, func(w http.ResponseWriter, r *http.Request) {
			var req sdk.VisionRequest
			_ = json.NewDecoder(r.Body).Decode(&req)
			req.Operation = op
			resp, err := h.backend.Vision(r.Context(), req)
			if errors.Is(err, vlm.ErrNoModel) {
				writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": err.Error()})
				return
			}
			if err != nil {
				writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, http.StatusOK, resp)
		})
	}
	mux.HandleFunc(prefix+"/generate", func(w http.ResponseWriter, r *http.Request) {
		var req sdk.HeadGenerateRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		resp, err := h.backend.Generate(r.Context(), req)
		if errors.Is(err, vlm.ErrNoModel) {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": err.Error()})
			return
		}
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, resp)
	})
	// Phase 3 VLM/HEAD artefact routes. HEAD is the vision model, so /models and
	// /weights here describe HEAD's own artefacts, not the decision core.
	mux.HandleFunc(prefix+"/models", h.handleVLMModels)
	mux.HandleFunc(prefix+"/weights/", func(w http.ResponseWriter, r *http.Request) {
		h.handleVLMWeights(w, r, prefix)
	})
}

// HeadManifest is the VLM/HEAD artefact manifest served at /models/head/models after
// Phase 3. It is distinct from the decision model manifest (now reachable only at
// /models/heart/decide/models): this one describes HEAD's vision artefacts.
type HeadManifest struct {
	Kind        string   `json:"kind"` // "vlm"
	Backend     string   `json:"backend"`
	Ready       bool     `json:"ready"`
	PromotedVer string   `json:"promoted_version,omitempty"`
	Operations  []string `json:"operations"`
}

func (h *Head) handleVLMModels(w http.ResponseWriter, r *http.Request) {
	ready, _ := h.backend.Ready(r.Context())
	writeJSON(w, http.StatusOK, HeadManifest{
		Kind:        "vlm",
		Backend:     h.backend.Name(),
		Ready:       ready,
		PromotedVer: h.backend.PromotedVersion(),
		Operations:  []string{"embed", "describe", "ground", "visual-reason"},
	})
}

// HeadWeights is the promoted-weight contract served at /models/head/weights/{ver}.
// It returns the promoted weight set / version info HEAD currently serves. HEAD never
// fabricates weights, so an unpromoted backend returns 503 and an unknown version 404.
type HeadWeights struct {
	Kind      string `json:"kind"` // "vlm-promoted-weights"
	Version   string `json:"version"`
	Backend   string `json:"backend"`
	Available bool   `json:"available"`
}

func (h *Head) handleVLMWeights(w http.ResponseWriter, r *http.Request, prefix string) {
	ver := strings.TrimPrefix(r.URL.Path, prefix+"/weights/")
	ready, _ := h.backend.Ready(r.Context())
	promoted := h.backend.PromotedVersion()
	if !ready || promoted == "" {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "HEAD has no promoted VLM weights"})
		return
	}
	if ver != promoted {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "unknown promoted version"})
		return
	}
	writeJSON(w, http.StatusOK, HeadWeights{
		Kind:      "vlm-promoted-weights",
		Version:   promoted,
		Backend:   h.backend.Name(),
		Available: true,
	})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
