package models

import (
	"errors"
	"net/http"
	"sync"
	"time"

	"knirvmodels/pkg/hashing/transformer"

	sdk "github.com/guiperry/knirv-sdk-go/models"
)

// retryAfterFailure stops a failing HEART build from being retried on every
// request (it loads a tokenizer, semantic memory and an engine).
const retryAfterFailure = 30 * time.Second

// Heart is the registry facade over transformer.HEARTService. The service is
// built on first use, so a daemon that never needs HEART never pays for it, and
// health/status requests never trigger the build.
type Heart struct {
	build func() (*transformer.HEARTService, error)

	mu      sync.Mutex
	svc     *transformer.HEARTService
	failed  error
	failAt  time.Time
	started time.Time
	handler *http.ServeMux
}

// NewHeart takes the constructor to run on first use.
func NewHeart(build func() (*transformer.HEARTService, error)) *Heart {
	return &Heart{build: build}
}

func (h *Heart) Name() string { return "heart" }

// loaded returns the built service without building it.
func (h *Heart) loaded() *transformer.HEARTService {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.svc
}

// ensure builds HEART once; a failure is remembered for retryAfterFailure.
func (h *Heart) ensure() (*http.ServeMux, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.handler != nil {
		return h.handler, nil
	}
	if h.failed != nil && time.Since(h.failAt) < retryAfterFailure {
		return nil, h.failed
	}
	svc, err := h.build()
	if err != nil {
		h.failed, h.failAt = err, time.Now()
		return nil, err
	}
	if svc == nil {
		h.failed, h.failAt = errors.New("heart constructor returned nothing"), time.Now()
		return nil, h.failed
	}
	mux := http.NewServeMux()
	svc.Mount(mux, "")
	h.svc, h.handler, h.failed, h.started = svc, mux, nil, time.Now()
	return mux, nil
}

func (h *Heart) Status() sdk.ModelStatus {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := sdk.ModelStatus{Name: "heart", Role: sdk.RoleGenerator, Version: "gorgonite", Backend: "software", Loaded: h.svc != nil}
	if h.svc != nil {
		out.Since = since(h.started)
		out.Ready = true
	}
	if h.failed != nil {
		out.Error = h.failed.Error()
	}
	return out
}

// Mount registers HEART's routes under prefix. Every route except health builds
// HEART first; a build failure is a 503 so callers fall through to another model.
func (h *Heart) Mount(mux *http.ServeMux, prefix string) {
	mux.HandleFunc(prefix+"/health", func(w http.ResponseWriter, r *http.Request) {
		if h.loaded() == nil {
			WriteJSON(w, http.StatusOK, sdk.HeartHealth{Available: true, Status: "idle"})
			return
		}
		h.serve(prefix, w, r)
	})
	for _, route := range []string{"analyze", "advise", "resolve", "patch", "generate", "stats", "reload-seeds"} {
		mux.HandleFunc(prefix+"/"+route, func(w http.ResponseWriter, r *http.Request) { h.serve(prefix, w, r) })
	}
}

func (h *Heart) serve(prefix string, w http.ResponseWriter, r *http.Request) {
	inner, err := h.ensure()
	if err != nil {
		WriteJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "heart unavailable: " + err.Error(), "ready": false})
		return
	}
	r2 := r.Clone(r.Context())
	u := *r.URL
	u.Path = u.Path[len(prefix):]
	r2.URL = &u
	inner.ServeHTTP(w, r2)
}
