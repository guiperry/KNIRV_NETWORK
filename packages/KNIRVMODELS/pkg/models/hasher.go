package models

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"sync"
	"time"

	"knirvmodels/pkg/hashing/transformer"
	"knirvmodels/pkg/hashing/validation"

	sdk "github.com/guiperry/knirv-sdk-go/models"
)

// Hasher is the registry facade over the SHA-256 neural model: the logical
// validator (cheap, always available) and the unified inference engine (built
// on first use). It wraps only entry points that already exist; the gRPC
// validator that GuardrailManager uses is unchanged.
type Hasher struct {
	mu        sync.Mutex
	engine    *transformer.UnifiedHasherEngine
	newEngine func() (*transformer.UnifiedHasherEngine, error)
	validator *validation.LogicalValidator
	started   time.Time
	index     *indexHolder
}

// NewHasher takes the engine constructor (nil → the production default).
func NewHasher(newEngine func() (*transformer.UnifiedHasherEngine, error)) *Hasher {
	if newEngine == nil {
		newEngine = func() (*transformer.UnifiedHasherEngine, error) {
			// The default engine holds a ~100k x 768 seed table (about 2.5 GB). It is
			// built once, on first use, and shared with HEART.
			log.Printf("HASHER: building the unified engine (large; first use)")
			return transformer.NewUnifiedHasherEngineFromConfig(nil)
		}
	}
	return &Hasher{newEngine: newEngine}
}

func (h *Hasher) Name() string { return "hasher" }

// Engine builds the engine on first use. HEART shares it so the daemon holds
// one set of seeds.
func (h *Hasher) Engine() (*transformer.UnifiedHasherEngine, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.engine != nil {
		return h.engine, nil
	}
	e, err := h.newEngine()
	if err != nil {
		return nil, err
	}
	if e == nil {
		return nil, errors.New("hasher engine unavailable")
	}
	h.engine, h.started = e, time.Now()
	return e, nil
}

func (h *Hasher) loaded() *transformer.UnifiedHasherEngine {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.engine
}

func (h *Hasher) logical() (*validation.LogicalValidator, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.validator == nil {
		v, err := validation.NewLogicalValidator()
		if err != nil {
			return nil, err
		}
		h.validator = v
	}
	return h.validator, nil
}

func (h *Hasher) snapshot() sdk.HasherStatus {
	e := h.loaded()
	if e == nil {
		return sdk.HasherStatus{}
	}
	st := e.Stats()
	return sdk.HasherStatus{
		Loaded: true, Mode: string(e.Mode()), HardwareTier: e.HardwareTier(), UsingHardware: e.IsUsingHardware(),
		TotalInferences: st.TotalInferences, HardwareInferences: st.HardwareInferences, SoftwareFallbacks: st.SoftwareFallbacks,
	}
}

func (h *Hasher) Status() sdk.ModelStatus {
	snap := h.snapshot()
	detail, _ := json.Marshal(snap)
	backend := "software"
	if snap.UsingHardware {
		backend = snap.HardwareTier
	}
	out := sdk.ModelStatus{
		Name: "hasher", Role: sdk.RoleValidator, Version: "unified-transformer",
		Ready: true, Loaded: snap.Loaded, Backend: backend, Detail: detail,
	}
	if snap.Loaded {
		h.mu.Lock()
		out.Since = since(h.started)
		h.mu.Unlock()
	}
	return out
}

func (h *Hasher) Mount(mux *http.ServeMux, prefix string) {
	mux.HandleFunc(prefix+"/status", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}
		WriteJSON(w, http.StatusOK, h.snapshot())
	})
	h.mountIndex(mux, prefix)
	h.mountOntology(mux, prefix)
	mux.HandleFunc(prefix+"/validate", h.validate)
	mux.HandleFunc(prefix+"/infer", h.infer)
}

func (h *Hasher) validate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	var req sdk.HasherValidateRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "bad request: " + err.Error()})
		return
	}
	if req.Domain == "" {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "domain is required"})
		return
	}
	v, err := h.logical()
	if err != nil {
		WriteJSON(w, http.StatusServiceUnavailable, map[string]string{"error": err.Error()})
		return
	}
	res, err := v.Validate(req.Prediction, req.Domain, req.Context)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	WriteJSON(w, http.StatusOK, sdk.HasherValidateResult{
		Prediction: res.Prediction, Valid: res.Valid, Domain: res.Domain, RulesApplied: res.RulesApplied,
		Errors: res.Errors, LatencyMs: float64(res.Latency) / float64(time.Millisecond),
	})
}

func (h *Hasher) infer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	var req struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "bad request: " + err.Error()})
		return
	}
	e, err := h.Engine()
	if err != nil {
		WriteJSON(w, http.StatusServiceUnavailable, map[string]string{"error": err.Error()})
		return
	}
	res, err := e.Infer([]byte(req.Text))
	if err != nil || res == nil || res.Consensus == nil {
		msg := "no consensus"
		if err != nil {
			msg = err.Error()
		}
		WriteJSON(w, http.StatusServiceUnavailable, map[string]string{"error": msg})
		return
	}
	WriteJSON(w, http.StatusOK, sdk.HasherInferResult{
		Prediction: res.Consensus.Prediction, Confidence: res.Consensus.Confidence, Mode: string(e.Mode()),
		ValidPasses: res.ValidPasses, TotalPasses: res.TotalPasses, LatencyMs: float64(res.Latency) / float64(time.Millisecond),
	})
}
