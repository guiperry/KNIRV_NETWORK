// Package models is the daemon's one namespace for its three models. Each model
// is a facade (HASHER validates, HEART generates, HEAD reasons) behind the same
// small interface, and the Registry is the single source for which model is
// serving what.
//
//	GET /models/registry   -> []sdk.ModelStatus
//	GET /models/health     -> sdk.Health
//	/models/{head,heart,hasher}/...   each model's own routes
//
// The routes carry no authentication: they are meant for backend_server over the
// daemon's unix socket (mode 0660), which authenticates callers.
package models

import (
	"encoding/json"
	"net/http"
	"sort"
	"sync"
	"time"

	sdk "github.com/guiperry/knirv-sdk-go/models"
	"github.com/guiperry/knirv-sdk-go/models/frame"
)

// Model is one served model. Status must be cheap and must never build the
// model: the registry is polled.
type Model interface {
	Name() string
	Status() sdk.ModelStatus
	// Mount registers the model's routes under prefix (e.g. "/models/heart").
	Mount(mux *http.ServeMux, prefix string)
}

// Registry holds the models the daemon serves.
type Registry struct {
	mu     sync.RWMutex
	models map[string]Model

	// frames and facts generate GET /models/manifest. facts is called per
	// request so the manifest reflects the running node, not startup.
	frames *frame.Registry
	facts  func() frame.LiveFacts
}

// SetManifest enables /models/manifest from the stage registry and a live-facts source.
func (r *Registry) SetManifest(frames *frame.Registry, facts func() frame.LiveFacts) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.frames, r.facts = frames, facts
}

func NewRegistry() *Registry { return &Registry{models: map[string]Model{}} }

// Register adds a model; a later registration of the same name replaces it.
func (r *Registry) Register(m Model) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.models[m.Name()] = m
}

// Statuses returns every model's status, sorted by name.
func (r *Registry) Statuses() []sdk.ModelStatus {
	r.mu.RLock()
	ms := make([]Model, 0, len(r.models))
	for _, m := range r.models {
		ms = append(ms, m)
	}
	r.mu.RUnlock()
	out := make([]sdk.ModelStatus, 0, len(ms))
	for _, m := range ms {
		out = append(out, m.Status())
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// Mount registers /models/registry, /models/health and every model's routes.
func (r *Registry) Mount(mux *http.ServeMux) {
	mux.HandleFunc(sdk.RegistryPath, func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet {
			WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}
		WriteJSON(w, http.StatusOK, r.Statuses())
	})
	mux.HandleFunc(sdk.HealthPath, func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet {
			WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}
		r.mu.RLock()
		n := len(r.models)
		r.mu.RUnlock()
		WriteJSON(w, http.StatusOK, sdk.Health{Status: "ok", Time: time.Now().UTC().Format(time.RFC3339), Models: n})
	})
	mux.HandleFunc(sdk.ManifestPath, func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet {
			WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}
		r.mu.RLock()
		frames, facts := r.frames, r.facts
		r.mu.RUnlock()
		if frames == nil {
			WriteJSON(w, http.StatusNotImplemented, map[string]string{"error": "no manifest configured"})
			return
		}
		var f frame.LiveFacts
		if facts != nil {
			f = facts()
		}
		WriteJSON(w, http.StatusOK, frames.Manifest(f))
	})
	r.mu.RLock()
	defer r.mu.RUnlock()
	for name, m := range r.models {
		m.Mount(mux, "/models/"+name)
	}
}

// WriteJSON writes v as a JSON response.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func since(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}
