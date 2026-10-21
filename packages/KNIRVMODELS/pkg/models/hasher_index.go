package models

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"knirvmodels/pkg/evidence"
	"knirvmodels/pkg/livesearch"
	"knirvmodels/pkg/orgontology"
)

// indexHolder opens the evidence index on first use, like the other HASHER parts,
// so a daemon that never answers an index request never touches the directory.
type indexHolder struct {
	dir        string
	embedderID string
	mu         sync.Mutex
	ix         *evidence.Index
	err        error

	// live builds the retriever once the index is open; nil disables live lookup.
	live func(ix *evidence.Index, dir string) (*livesearch.Retriever, error)
	ret  *livesearch.Retriever
}

// UseIndex points HASHER at an evidence index directory. embedderID is only used
// when the directory is new; an existing index keeps the embedder it was built with.
func (h *Hasher) UseIndex(dir, embedderID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.index = &indexHolder{dir: dir, embedderID: embedderID}
}

// UseLive enables live lookup (Wikipedia/arXiv) on top of the index. build is called once,
// after the index opens; pass livesearch.NewProduction in the daemon.
func (h *Hasher) UseLive(build func(ix *evidence.Index, dir string) (*livesearch.Retriever, error)) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.index != nil {
		h.index.live = build
	}
}

// Retriever returns the live retriever, or nil when live lookup is not configured.
func (h *Hasher) Retriever() (*livesearch.Retriever, error) {
	ix, err := h.Index()
	if err != nil {
		return nil, err
	}
	h.mu.Lock()
	holder := h.index
	h.mu.Unlock()
	holder.mu.Lock()
	defer holder.mu.Unlock()
	if holder.ret == nil && holder.live != nil {
		holder.ret, err = holder.live(ix, holder.dir)
		if err != nil {
			return nil, err
		}
	}
	return holder.ret, nil
}

// Index returns the evidence index, opening it on first use. It is nil, with an
// error, when no directory was configured or the index cannot be opened.
func (h *Hasher) Index() (*evidence.Index, error) {
	h.mu.Lock()
	holder := h.index
	h.mu.Unlock()
	if holder == nil {
		return nil, errNoIndex
	}
	holder.mu.Lock()
	defer holder.mu.Unlock()
	if holder.ix == nil && holder.err == nil {
		holder.ix, holder.err = evidence.Open(holder.dir, holder.embedderID)
	}
	return holder.ix, holder.err
}

type indexError string

func (e indexError) Error() string { return string(e) }

const errNoIndex = indexError("no evidence index is configured")

// Request headers set by the authenticating backend, as for HEART.
const TenantHeader = "X-KNIRV-Tenant"

func (h *Hasher) mountIndex(mux *http.ServeMux, prefix string) {
	mux.HandleFunc(prefix+"/index/status", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}
		ix, err := h.Index()
		if err != nil {
			WriteJSON(w, http.StatusServiceUnavailable, map[string]any{"error": err.Error()})
			return
		}
		WriteJSON(w, http.StatusOK, ix.Status())
	})
	mux.HandleFunc(prefix+"/index/search", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}
		ix, err := h.Index()
		if err != nil {
			WriteJSON(w, http.StatusServiceUnavailable, map[string]any{"error": err.Error()})
			return
		}
		var req struct {
			evidence.Claim
			K                 int    `json:"k"`
			IncludeQuarantine bool   `json:"include_quarantine"`
			Mode              string `json:"mode"` // index (default) | flash | hybrid
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil || req.Text == "" {
			WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "a claim text is required"})
			return
		}
		if req.K > 50 {
			req.K = 50
		}
		if m := livesearch.Mode(req.Mode); m == livesearch.ModeFlash || m == livesearch.ModeHybrid {
			ret, err := h.Retriever()
			if err != nil || ret == nil {
				WriteJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "live lookup is not configured"})
				return
			}
			// Bounded: a claim must not hold the route open for longer than a verdict can wait.
			ctx, cancel := context.WithTimeout(r.Context(), 45*time.Second)
			defer cancel()
			res, err := ret.Retrieve(ctx, req.Claim, livesearch.Options{K: req.K, Tenant: r.Header.Get(TenantHeader), Mode: m})
			if err != nil {
				WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
			WriteJSON(w, http.StatusOK, res)
			return
		}
		opt := evidence.Options{K: req.K, Tenant: r.Header.Get(TenantHeader)}
		// Quarantine (live-web write-through) is for operators; it is never mixed into
		// an ordinary user's evidence.
		if req.IncludeQuarantine && privilegeOf(r.Header.Get(PrivilegeHeader)) >= 1 {
			opt.Tiers = []evidence.Tier{evidence.TierReference, evidence.TierUser, evidence.TierQuarantine}
		}
		res, err := ix.Search(req.Claim, opt)
		if err != nil {
			WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		WriteJSON(w, http.StatusOK, res)
	})
}

// mountOntology serves the organization-ontology routes (plan §4.7):
//
//	POST   /ontology/sync   operator: load an organization's onboarding ontology (body: orgontology.Config)
//	DELETE /ontology        operator: retire an organization's ontology (?tenant=ORG)
//	POST   /ontology/check  caller's tenant: compare {subject, property, value} with the ontology, no model involved
//
// sync and delete are for the backend that owns onboarding; check runs as the
// caller's tenant and can only ever see that tenant's ontology.
func (h *Hasher) mountOntology(mux *http.ServeMux, prefix string) {
	op := func(w http.ResponseWriter, r *http.Request) bool {
		if privilegeOf(r.Header.Get(PrivilegeHeader)) < 1 {
			WriteJSON(w, http.StatusForbidden, map[string]string{"error": "operator privilege is required"})
			return false
		}
		return true
	}
	mux.HandleFunc(prefix+"/ontology/sync", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}
		if !op(w, r) {
			return
		}
		ix, err := h.Index()
		if err != nil {
			WriteJSON(w, http.StatusServiceUnavailable, map[string]any{"error": err.Error()})
			return
		}
		var cfg orgontology.Config
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<20)).Decode(&cfg); err != nil {
			WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "body must be an organization configuration: " + err.Error()})
			return
		}
		res, err := orgontology.Sync(ix, cfg)
		if err != nil {
			WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		WriteJSON(w, http.StatusOK, res)
	})
	mux.HandleFunc(prefix+"/ontology", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}
		if !op(w, r) {
			return
		}
		ix, err := h.Index()
		if err != nil {
			WriteJSON(w, http.StatusServiceUnavailable, map[string]any{"error": err.Error()})
			return
		}
		tenant := r.URL.Query().Get("tenant")
		if tenant == "" {
			WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "tenant is required"})
			return
		}
		if err := orgontology.Remove(ix, tenant); err != nil {
			WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		WriteJSON(w, http.StatusOK, map[string]string{"tenant": tenant, "status": "removed"})
	})
	mux.HandleFunc(prefix+"/ontology/check", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}
		ix, err := h.Index()
		if err != nil {
			WriteJSON(w, http.StatusServiceUnavailable, map[string]any{"error": err.Error()})
			return
		}
		var c evidence.Claim
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&c); err != nil {
			WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "a claim with subject, property and value is required"})
			return
		}
		v, err := orgontology.Check(ix, r.Header.Get(TenantHeader), c)
		if err != nil {
			WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		WriteJSON(w, http.StatusOK, v)
	})
}
