package models

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"time"

	"knirvmodels/internal/policy"
	"knirvmodels/pkg/hashing/transformer"
	"knirvmodels/pkg/heartqwen"

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

	// qwen, when set, answers generate. HEART is the shared local Qwen behind a
	// role frame; the Gorgonia GPT stays behind the other routes until a stage
	// is promoted (spec §3.6).
	qwen *heartqwen.Engine
	live func(ctx context.Context) bool
	fin  *transformer.WASMFinisher
}

// UseQwen makes generate (and health) come from the Qwen backend. live reports
// whether the llama wrapper is up.
func (h *Heart) UseQwen(e *heartqwen.Engine, live func(ctx context.Context) bool, fin *transformer.WASMFinisher) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.qwen, h.live, h.fin = e, live, fin
}

func (h *Heart) qwenEngine() (*heartqwen.Engine, func(ctx context.Context) bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.qwen, h.live
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
		if q, live := h.qwenEngine(); q != nil {
			ok := live != nil && live(r.Context())
			status := "qwen"
			if !ok {
				status = "qwen-unavailable"
			}
			WriteJSON(w, http.StatusOK, sdk.HeartHealth{Available: ok, Status: status, Loaded: ok, Ready: ok, Progress: 1})
			return
		}
		if h.loaded() == nil {
			WriteJSON(w, http.StatusOK, sdk.HeartHealth{Available: true, Status: "idle"})
			return
		}
		h.serve(prefix, w, r)
	})
	mux.HandleFunc(prefix+"/generate", func(w http.ResponseWriter, r *http.Request) {
		if q, _ := h.qwenEngine(); q != nil {
			h.generateQwen(q, w, r)
			return
		}
		h.serve(prefix, w, r)
	})
	// With the Qwen backend these four are answered by the model under its stage
	// frame; the system renders, verifies and compiles whatever fields it chooses.
	qwenRoute := func(route string, qwen http.HandlerFunc) {
		mux.HandleFunc(prefix+"/"+route, func(w http.ResponseWriter, r *http.Request) {
			if q, _ := h.qwenEngine(); q != nil {
				qwen(w, r)
				return
			}
			h.serve(prefix, w, r)
		})
	}
	qwenRoute("analyze", h.analyzeQwen)
	qwenRoute("advise", h.wasmHandler(transformer.WASMTypeRule))
	qwenRoute("resolve", h.wasmHandler(transformer.WASMTypeResolution))
	qwenRoute("patch", h.wasmHandler(transformer.WASMTypePatch))
	for _, route := range []string{"stats", "reload-seeds"} {
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

// Caller identity comes from the backend, which authenticates the user and
// reaches this daemon over its unix socket (see the package comment).
const (
	CallerHeader    = "X-KNIRV-Caller"
	PrivilegeHeader = "X-KNIRV-Privilege"
)

func privilegeOf(s string) policy.Privilege {
	switch s {
	case "admin":
		return policy.Admin
	case "operator":
		return policy.Operator
	}
	return policy.User
}

// generateQwen runs a stage on the Qwen backend. The stage defaults to
// heart.chat; naming ops.assistant requires an operator, which is checked
// here as well as per tool in the gate.
func (h *Heart) generateQwen(q *heartqwen.Engine, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed", "ready": false})
		return
	}
	var req struct {
		Prompt string `json:"prompt"`
		Stage  string `json:"stage"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil || req.Prompt == "" {
		WriteJSON(w, http.StatusBadRequest, map[string]any{"error": "a prompt is required", "ready": false})
		return
	}
	if req.Stage == "" {
		req.Stage = "heart.chat"
	}
	priv := privilegeOf(r.Header.Get(PrivilegeHeader))
	if req.Stage == "ops.assistant" && priv < policy.Operator {
		WriteJSON(w, http.StatusForbidden, map[string]any{"error": "ops.assistant needs operator privilege", "ready": true})
		return
	}
	res, err := q.Run(r.Context(), heartqwen.Input{Stage: req.Stage, Task: req.Prompt,
		Caller: r.Header.Get(CallerHeader), Privilege: priv, Tenant: r.Header.Get(TenantHeader)})
	if err != nil {
		// 503 so the harness falls through to the external generator and the
		// Trace says why.
		WriteJSON(w, http.StatusServiceUnavailable, map[string]any{"error": err.Error(), "ready": false, "backend": heartqwen.Backend})
		return
	}
	text := res.Text
	if text == "" && len(res.Output) > 0 {
		text = string(res.Output)
	}
	WriteJSON(w, http.StatusOK, map[string]any{"text": text, "ready": true, "backend": res.Backend,
		"stage": res.Stage, "frame_hash": res.FrameHash, "trace": res.Trace, "proposals": res.Proposals})
}
