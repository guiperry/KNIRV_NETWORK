package models

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"knirvmodels/internal/policy"
	"knirvmodels/pkg/hands"
	"knirvmodels/pkg/hands/gpt"
	"knirvmodels/pkg/heartqwen"
	"knirvmodels/pkg/training/nrvtrainer"

	sdk "github.com/guiperry/knirv-sdk-go/models"
	"github.com/guiperry/knirv-sdk-go/models/frame"
)

// Hands is the registry facade over the HANDS inner GPT. Like HEART it is built
// on first use; health/status requests never trigger the build. It is entirely
// gated by KNIRV_HANDS_ENABLED (see flags.go): when disabled it reports a
// "disabled" status and mounts no routes, so the rest of the daemon is
// unaffected by the migration in progress.
//
// The framed handlers (/plan, /act) run the model inside the shared
// policy-gated tool loop (heartqwen.Engine.Run): the model emits an action
// envelope, every tool request passes through policy.Gate before anything
// executes, T2 (mutating) calls are returned as proposals, T3/forbidden calls
// are denied, and each decision is appended to the audit chain. This is the
// same loop HEART uses, so HANDS cannot exceed the frame's authority.
type Hands struct {
	build func() (generator, error)

	gate  *policy.Gate
	audit *policy.Audit
	exec  heartqwen.Executor
	reg   *frame.Registry

	mu     sync.Mutex
	eng    generator
	failed error
	failAt time.Time
	start  time.Time
	job    *TrainJob

	// runMu serialises training against generation. Training mutates the model's
	// persisted tensors in place; a concurrent Forward would observe a
	// half-updated weight set. Generation takes a read lock and returns 503 if
	// training holds the write lock.
	runMu sync.RWMutex
}

// generator is the HANDS model surface the handlers drive. *hands.Engine
// satisfies it; tests inject a scripted stand-in.
type generator interface {
	IsReady() bool
	Path() string
	Progress() float64
	GenerateText(prompt string, n int, temperature float32, topK int) (string, error)
}

// Generator is the exported alias of generator so callers outside the package
// (the headless daemon) can inject a scripted stand-in for HANDS in tests.
type Generator = generator

// maxBodyBytes bounds a hands request body so a hostile or buggy caller cannot
// exhaust memory before the JSON decodes.
const maxBodyBytes = 64 << 10

// maxOutputTokens is the ceiling on the number of tokens a single generation may
// ask for; it matches the frame default and the hands.* stages.
const maxOutputTokens = 512

// Training ceilings. /models/hands/train is reachable by the backend over the
// daemon's Unix socket, so every bound is enforced here rather than trusted to
// the caller.
const (
	maxTrainFrames  = 50_000
	maxTrainWindows = 200_000
	maxTrainEpochs  = 100
	maxTrainSteps   = 10_000
	maxTrainContext = 4096
)

// TrainJob is the observable state of one /models/hands/train request.
type TrainJob struct {
	ID         string     `json:"id"`
	State      string     `json:"state"`
	Input      string     `json:"input"`
	StartedAt  time.Time  `json:"startedAt"`
	FinishedAt *time.Time `json:"finishedAt,omitempty"`
	Frames     int        `json:"frames,omitempty"`
	Windows    int        `json:"windows,omitempty"`
	Steps      int        `json:"steps,omitempty"`
	Checkpoint string     `json:"checkpoint,omitempty"`
	Error      string     `json:"error,omitempty"`
}

// trainable is the optional surface a HANDS engine exposes for the trainer.
// Test stand-ins that only generate simply do not implement it.
type trainable interface {
	TrainWindows(windows [][]int, opts hands.TrainOptions) (hands.TrainResult, error)
}

// NewHands wires the HANDS facade with the same concrete executor, gate, audit
// and frame registry the daemon's HEART path uses, so a real core-authorized T0
// tool actually executes (instead of hitting the empty ops.New executor) and
// every decision lands in the audit chain. build returns the (possibly untrained)
// HANDS engine; tests use NewHandsWith directly to inject a scripted generator.
func NewHands(build func() (*hands.Engine, error), g *policy.Gate, a *policy.Audit, ex heartqwen.Executor, reg *frame.Registry) *Hands {
	return NewHandsWith(
		func() (generator, error) { return build() },
		g,
		a,
		ex,
		reg,
	)
}

// NewHandsWith injects the policy gate, audit log, tool executor and frame
// registry the framed handlers run against. The production daemon passes the
// same gate/audit/executor it gives HEART.
func NewHandsWith(build func() (generator, error), g *policy.Gate, a *policy.Audit, ex heartqwen.Executor, reg *frame.Registry) *Hands {
	return &Hands{build: build, gate: g, audit: a, exec: ex, reg: reg}
}

func (h *Hands) Name() string { return "hands" }

func (h *Hands) loaded() generator {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.eng
}

// Status is cheap and never builds the model.
func (h *Hands) Status() sdk.ModelStatus {
	if !HandsEnabled() {
		return sdk.ModelStatus{
			Name:    "hands",
			Role:    sdk.RoleHands,
			Ready:   false,
			Loaded:  false,
			Backend: "disabled",
			Error:   "HANDS disabled (set KNIRV_HANDS_ENABLED=1 to enable)",
		}
	}
	h.mu.Lock()
	eng, failed, failAt, start := h.eng, h.failed, h.failAt, h.start
	h.mu.Unlock()

	out := sdk.ModelStatus{
		Name:    "hands",
		Role:    sdk.RoleHands,
		Version: "gorgonite",
		Backend: "software",
	}
	if eng != nil {
		out.Loaded = true
		out.Ready = eng.IsReady()
		out.Trained = eng.IsReady()
		out.Since = since(start)
	} else if failed != nil {
		if time.Since(failAt) < retryAfterFailure {
			out.Error = failed.Error()
		}
	}
	return out
}

// handleHealth is the cheap readiness/liveness surface required by Phase 1: it
// never builds the model (Status only reads the cached engine state).
func (h *Hands) handleHealth(w http.ResponseWriter, r *http.Request) {
	WriteJSON(w, http.StatusOK, h.Status())
}

// Mount registers the HANDS routes under prefix. When HANDS is disabled nothing
// is mounted. Phase 4 mounts status/models/weights/generate/plan/act; the
// policy-gated tool loop and the .nrv trainer endpoint land with the tool
// executor and the unified .nrv pipeline (cross-repo handoffs, see migration
// plan §8).
func (h *Hands) Mount(mux *http.ServeMux, prefix string) {
	if !HandsEnabled() {
		return
	}
	mux.HandleFunc(prefix+"/status", func(w http.ResponseWriter, r *http.Request) {
		WriteJSON(w, http.StatusOK, h.statusPayload())
	})
	mux.HandleFunc(prefix+"/health", h.handleHealth)
	mux.HandleFunc(prefix+"/models", h.handleModels)
	mux.HandleFunc(prefix+"/weights", h.handleWeights)
	mux.HandleFunc(prefix+"/generate", h.handleGenerate)
	mux.HandleFunc(prefix+"/plan", h.handlePlan)
	mux.HandleFunc(prefix+"/act", h.handleAct)
	mux.HandleFunc(prefix+"/train", h.handleTrain)
}

// statusPayload is sdk.ModelStatus plus the current training job, if any.
func (h *Hands) statusPayload() any {
	return struct {
		sdk.ModelStatus
		Training *TrainJob `json:"training,omitempty"`
	}{ModelStatus: h.Status(), Training: h.trainingSnapshot()}
}

func (h *Hands) trainingSnapshot() *TrainJob {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.job == nil {
		return nil
	}
	copy := *h.job
	return &copy
}

func (h *Hands) handleModels(w http.ResponseWriter, r *http.Request) {
	WriteJSON(w, http.StatusOK, map[string]any{
		"name":         "hands",
		"role":         sdk.RoleHands,
		"backend":      "software",
		"version":      "gorgonite",
		"capabilities": []string{"generate", "plan", "act", "train"},
		"endpoints":    []string{"/status", "/health", "/models", "/weights", "/generate", "/plan", "/act", "/train"},
	})
}

func (h *Hands) handleWeights(w http.ResponseWriter, r *http.Request) {
	eng, err := h.engine()
	if err != nil || eng == nil || !eng.IsReady() {
		WriteJSON(w, http.StatusServiceUnavailable, map[string]string{
			"error": "HANDS untrained (set KNIRV_HANDS_ENABLED=1 and provide a checkpoint)",
		})
		return
	}
	WriteJSON(w, http.StatusOK, map[string]any{
		"name":       "hands",
		"version":    "gorgonite",
		"backend":    "software",
		"checkpoint": eng.Path(),
		"progress":   eng.Progress(),
	})
}

// decodeHandsBody reads a bounded request body and returns the generation
// parameters. N is capped at maxOutputTokens; larger values are rejected so a
// caller cannot drive one-token-per-loop generation to an unbounded length.
func decodeHandsBody(r *http.Request) (prompt string, n int, temperature float32, topK int, err error) {
	var req struct {
		Prompt      string  `json:"prompt"`
		N           int     `json:"n"`
		Temperature float32 `json:"temperature"`
		TopK        int     `json:"topK"`
	}
	if derr := json.NewDecoder(r.Body).Decode(&req); derr != nil {
		return "", 0, 0, 0, fmt.Errorf("bad request body: %w", derr)
	}
	if req.N <= 0 {
		req.N = 32
	}
	if req.N > maxOutputTokens {
		return "", 0, 0, 0, fmt.Errorf("n %d exceeds max output tokens %d", req.N, maxOutputTokens)
	}
	return req.Prompt, req.N, req.Temperature, req.TopK, nil
}

// handleGenerate runs the bare HANDS model with no role frame. N is capped at the
// stage ceiling. Untrained HANDS returns 503.
func (h *Hands) handleGenerate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	prompt, n, temperature, topK, err := decodeHandsBody(r)
	if err != nil {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	eng, err := h.engine()
	if err != nil || eng == nil || !eng.IsReady() {
		WriteJSON(w, http.StatusServiceUnavailable, map[string]string{
			"error": "HANDS untrained (set KNIRV_HANDS_ENABLED=1 and provide a checkpoint)",
		})
		return
	}
	if !h.runMu.TryRLock() {
		WriteJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "HANDS training in progress"})
		return
	}
	text, err := eng.GenerateText(prompt, n, temperature, topK)
	h.runMu.RUnlock()
	if err != nil {
		WriteJSON(w, http.StatusServiceUnavailable, map[string]string{"error": err.Error()})
		return
	}
	WriteJSON(w, http.StatusOK, map[string]any{"text": text})
}

// handlePlan builds the hands.plan frame and runs the HANDS model through the
// policy-gated tool loop to break a server task into bounded sub-tasks. Untrained
// HANDS returns 503.
func (h *Hands) handlePlan(w http.ResponseWriter, r *http.Request) {
	h.handleFramed(w, r, "hands.plan")
}

// handleAct builds the hands.assistant frame and runs the HANDS model through the
// policy-gated tool loop to carry out an operator request about server tasks.
// Untrained HANDS returns 503.
func (h *Hands) handleAct(w http.ResponseWriter, r *http.Request) {
	h.handleFramed(w, r, "hands.assistant")
}

// handleFramed builds the named HANDS stage frame and runs the model inside the
// shared policy-gated tool loop. It is gated exactly like /generate: disabled or
// untrained HANDS returns 503, non-POST returns 405. Every tool request the
// model emits is checked by policy.Gate; T2 calls become proposals, T3/forbidden
// calls are denied, and each decision is written to the audit chain.
func (h *Hands) handleFramed(w http.ResponseWriter, r *http.Request, stage string) {
	if r.Method != http.MethodPost {
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	prompt, _, _, _, err := decodeHandsBody(r)
	if err != nil {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	eng, err := h.engine()
	if err != nil || eng == nil || !eng.IsReady() {
		WriteJSON(w, http.StatusServiceUnavailable, map[string]string{
			"error": "HANDS untrained (set KNIRV_HANDS_ENABLED=1 and provide a checkpoint)",
		})
		return
	}

	if !h.runMu.TryRLock() {
		WriteJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "HANDS training in progress"})
		return
	}
	defer h.runMu.RUnlock()

	e := hands.NewLoop(eng, h.reg, h.gate, h.exec)
	in := heartqwen.Input{
		Stage:     stage,
		Task:      prompt,
		Caller:    callerFrom(r),
		Privilege: privilegeFrom(r),
		Tenant:    tenantFrom(r),
	}
	res, err := e.Run(r.Context(), in)
	if err != nil {
		WriteJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	out := map[string]any{
		"stage":      stage,
		"frame_hash": res.FrameHash,
		"trace":      res.Trace,
		"proposals":  res.Proposals,
	}
	if res.Text != "" {
		out["text"] = res.Text
	}
	if res.Output != nil {
		out["output"] = res.Output
	}
	WriteJSON(w, http.StatusOK, out)
}

// callerFrom/privilegeFrom/tenantFrom lift the authenticated request context
// into the policy gate. These come from the backend auth layer as headers, never
// from model output.
func callerFrom(r *http.Request) string {
	if c := strings.TrimSpace(r.Header.Get("X-KNIRV-Caller")); c != "" {
		return c
	}
	return "anonymous"
}

func privilegeFrom(r *http.Request) policy.Privilege {
	switch strings.ToLower(strings.TrimSpace(r.Header.Get("X-KNIRV-Privilege"))) {
	case "operator":
		return policy.Operator
	case "admin":
		return policy.Admin
	default:
		return policy.User
	}
}

func tenantFrom(r *http.Request) string {
	return strings.TrimSpace(r.Header.Get("X-KNIRV-Tenant"))
}

// engine builds (once) and returns the HANDS engine, honouring the same
// failure-remembered-for-retryAfterFailure contract as HEART. It is only called
// by the generation/weights routes; Status never builds.
func (h *Hands) engine() (generator, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.eng != nil {
		return h.eng, nil
	}
	if h.failed != nil && time.Since(h.failAt) < retryAfterFailure {
		return nil, h.failed
	}
	h.start = time.Now()
	eng, err := h.build()
	if err != nil {
		h.failed = err
		h.failAt = time.Now()
		return nil, err
	}
	h.eng = eng
	return eng, nil
}

type trainRequest struct {
	Input        string  `json:"input"`
	Epochs       int     `json:"epochs"`
	LearningRate float64 `json:"learningRate"`
	MaxSteps     int     `json:"maxSteps"`
	ContextLen   int     `json:"contextLength"`
	Stride       int     `json:"stride"`
}

// handleTrain starts one bounded training pass over a signed .nrv corpus. It is
// POST-only, returns 202 with a job id, and refuses (409) while a job is already
// running. The job is visible under /models/hands/status; nothing here opens a
// new listener, so the route stays reachable only through the daemon's Unix
// socket and KNIRVGATEWAY.
func (h *Hands) handleTrain(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	var req trainRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("bad request body: %v", err)})
		return
	}

	input := filepath.Clean(strings.TrimSpace(req.Input))
	if input == "" || input == "." {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "input .nrv path is required"})
		return
	}
	if !filepath.IsAbs(input) {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "input must be an absolute path"})
		return
	}
	if !strings.EqualFold(filepath.Ext(input), ".nrv") {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "input must be a .nrv asset"})
		return
	}
	info, err := os.Stat(input)
	if err != nil {
		if os.IsNotExist(err) {
			WriteJSON(w, http.StatusNotFound, map[string]string{"error": "input asset not found"})
			return
		}
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("stat input: %v", err)})
		return
	}
	if !info.Mode().IsRegular() || info.Size() == 0 {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "input must be a non-empty regular file"})
		return
	}

	epochs := req.Epochs
	if epochs <= 0 {
		epochs = 1
	}
	if epochs > maxTrainEpochs {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("epochs %d exceeds limit %d", epochs, maxTrainEpochs)})
		return
	}
	learningRate := req.LearningRate
	if learningRate <= 0 {
		learningRate = 3e-4
	}
	if learningRate > 1 {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "learningRate must be <= 1"})
		return
	}
	maxSteps := req.MaxSteps
	if maxSteps <= 0 {
		maxSteps = 128
	}
	if maxSteps > maxTrainSteps {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("maxSteps %d exceeds limit %d", maxSteps, maxTrainSteps)})
		return
	}
	contextLen := req.ContextLen
	if contextLen <= 0 {
		contextLen = 64
	}
	if contextLen < 2 || contextLen > maxTrainContext {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("contextLength must be in [2,%d]", maxTrainContext)})
		return
	}
	stride := req.Stride
	if stride <= 0 {
		stride = contextLen
	}
	if stride > contextLen {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "stride must be <= contextLength"})
		return
	}

	asset, err := nrvtrainer.LoadAsset(input)
	if err != nil {
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if len(asset.Frames) > maxTrainFrames {
		WriteJSON(w, http.StatusRequestEntityTooLarge, map[string]string{
			"error": fmt.Sprintf("asset has %d frames, exceeds limit %d", len(asset.Frames), maxTrainFrames),
		})
		return
	}
	windows := asset.TokenWindows(contextLen, stride)
	if len(windows) == 0 {
		WriteJSON(w, http.StatusUnprocessableEntity, map[string]string{
			"error": "asset has no Z3-valid token_window payloads usable for HANDS training",
		})
		return
	}
	if len(windows) > maxTrainWindows {
		WriteJSON(w, http.StatusRequestEntityTooLarge, map[string]string{
			"error": fmt.Sprintf("asset yields %d windows, exceeds limit %d", len(windows), maxTrainWindows),
		})
		return
	}

	eng, err := h.engine()
	if err != nil || eng == nil {
		WriteJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "HANDS engine unavailable"})
		return
	}
	trainer, ok := eng.(trainable)
	if !ok {
		WriteJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "HANDS engine does not support training"})
		return
	}
	// Reject out-of-vocabulary tokens here rather than letting GPT wrap them
	// modulo VocabSize; the JSONL path and the .nrv batch trainer reject them too.
	if cfgr, ok := eng.(interface{ Config() *gpt.Config }); ok {
		if cfg := cfgr.Config(); cfg != nil && cfg.VocabSize > 0 {
			if err := asset.CheckTokenRange(cfg.VocabSize); err != nil {
				WriteJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
				return
			}
		}
	}

	job := &TrainJob{
		ID:        fmt.Sprintf("hands-%d", time.Now().UnixNano()),
		State:     "running",
		Input:     input,
		StartedAt: time.Now(),
		Frames:    len(asset.Frames),
		Windows:   len(windows),
	}
	if !h.beginTrain(job) {
		WriteJSON(w, http.StatusConflict, map[string]string{"error": "a HANDS training job is already running"})
		return
	}

	savePath := ""
	if dir := HandsDir(); dir != "" {
		savePath = filepath.Join(dir, "model-latest")
	}
	opts := hands.TrainOptions{
		Epochs:       epochs,
		LearningRate: learningRate,
		MaxSteps:     maxSteps,
		SavePath:     savePath,
	}
	// Snapshot the job before the training goroutine can mutate it under h.mu.
	h.mu.Lock()
	accepted := *job
	h.mu.Unlock()
	go h.runTrain(job, trainer, windows, opts)
	WriteJSON(w, http.StatusAccepted, &accepted)
}

func (h *Hands) beginTrain(job *TrainJob) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.job != nil && h.job.State == "running" {
		return false
	}
	h.job = job
	return true
}

func (h *Hands) runTrain(job *TrainJob, trainer trainable, windows [][]int, opts hands.TrainOptions) {
	var (
		result hands.TrainResult
		err    error
	)
	func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				err = fmt.Errorf("training panicked: %v", recovered)
			}
		}()
		h.runMu.Lock()
		defer h.runMu.Unlock()
		result, err = trainer.TrainWindows(windows, opts)
	}()

	h.mu.Lock()
	defer h.mu.Unlock()
	finished := time.Now()
	job.FinishedAt = &finished
	job.Windows = result.Windows
	job.Steps = result.Steps
	job.Checkpoint = result.Checkpoint
	if err != nil {
		job.State = "failed"
		job.Error = err.Error()
		return
	}
	job.State = "succeeded"
}
