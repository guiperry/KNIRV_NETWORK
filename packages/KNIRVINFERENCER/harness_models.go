package inferencer

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	models "github.com/guiperry/knirv-sdk-go/models"
	"github.com/guiperry/knirv-sdk-go/models/frame"
)

// Adapters from each model to HarnessModel. HEAD, HEART and HASHER are reached through
// the KNIRVSDK models client (one unix socket); llama through its own socket; the
// external chain through hooks supplied by InferenceService.

const statusTTL = 5 * time.Second

// statusCache reads the daemon registry at most once per statusTTL and shares it
// between the three daemon-backed models.
type statusCache struct {
	client *models.Client

	mu   sync.Mutex
	at   time.Time
	rows map[string]models.ModelStatus
	err  error
}

func (c *statusCache) get(ctx context.Context, name string) (models.ModelStatus, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if time.Since(c.at) > statusTTL {
		cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		rows, err := c.client.Registry(cctx)
		cancel()
		c.at, c.err = time.Now(), err
		c.rows = map[string]models.ModelStatus{}
		for _, r := range rows {
			c.rows[r.Name] = r
		}
	}
	if c.err != nil {
		return models.ModelStatus{}, c.err
	}
	r, ok := c.rows[name]
	if !ok {
		return models.ModelStatus{}, fmt.Errorf("daemon does not serve %q", name)
	}
	return r, nil
}

func (c *statusCache) readiness(ctx context.Context, name string) Readiness {
	r, err := c.get(ctx, name)
	if err != nil {
		return Readiness{Detail: err.Error()}
	}
	d := r.Error
	if d == "" && !r.Ready {
		d = "not ready"
	}
	return Readiness{Ready: r.Ready, Version: r.Version, Detail: d}
}

// DaemonModels builds the three models served by the KNIRVMODELS daemon on one client.
// They share one registry read.
func DaemonModels(c *models.Client) (head *HeadModel, heart *HeartModel, hasher *HasherModel) {
	sc := &statusCache{client: c}
	return &HeadModel{client: c, status: sc}, &HeartModel{client: c, status: sc}, &HasherModel{client: c, status: sc}
}

// ---- HEAD -----------------------------------------------------------------

// HeadModel is HEAD after the migration: the vision / multimodal model (VLM) and the
// coordinator of mirrored text generation. It no longer decides or routes; those live
// in HEART (see HeartModel). Vision tasks carry a models.VisionRequest; mirror_generate
// tasks carry a models.HeadGenerateRequest and HEAD arbitrates its own candidate against
// HEART's (see GenerateMirrored).
type HeadModel struct {
	client *models.Client
	status *statusCache
}

func (h *HeadModel) Name() string    { return "head" }
func (h *HeadModel) Info() ModelInfo { return ModelInfo{} }
func (h *HeadModel) Ready(ctx context.Context) Readiness {
	return h.status.readiness(ctx, "head")
}

func (h *HeadModel) Do(ctx context.Context, t Task) (Result, error) {
	switch t.Kind {
	case KindVision:
		req, ok := t.Input.(models.VisionRequest)
		if !ok {
			return Result{}, fmt.Errorf("vision input must be models.VisionRequest, got %T", t.Input)
		}
		r, err := h.client.HeadVision(ctx, req)
		if err != nil {
			return Result{}, fmt.Errorf("head vision: %w", err)
		}
		return Result{Value: r, Model: "head"}, nil
	case KindMirrorGenerate:
		g, ok := t.Input.(models.HeadGenerateRequest)
		if !ok {
			return Result{}, fmt.Errorf("mirror_generate input must be models.HeadGenerateRequest, got %T", t.Input)
		}
		mr, err := GenerateMirrored(ctx, h.client, g.Prompt, g.PerceptualContext, g.Images, g.Frame, nil)
		if err != nil {
			return Result{}, fmt.Errorf("head mirror_generate: %w", err)
		}
		return Result{Text: mr.Text, Model: mr.Winner, Version: mr.Version}, nil
	default:
		return Result{}, fmt.Errorf("head does not take %q tasks", t.Kind)
	}
}

// routePrompt is the closed-set question the decision core is asked.
func routePrompt(t Task, eligible []string) string {
	return fmt.Sprintf("route kind=%s eligible=%s skill=%t", t.Kind, strings.Join(eligible, ","), t.Skill != nil)
}

// ---- HEART ----------------------------------------------------------------

// HeartModel is the generator over the daemon's HEART routes and, after the migration,
// the decision core (formerly HEAD's reasoning) and the harness's advisory Chooser.
// Input is a string (generate/analyze) or a models.AdviseContext (decide).
type HeartModel struct {
	client *models.Client
	status *statusCache

	mu        sync.Mutex
	ready     Readiness
	lastCheck time.Time
	noRoute   map[string]time.Time // model version -> when "no route head" was seen
}

func (h *HeartModel) Name() string    { return "heart" }
func (h *HeartModel) Info() ModelInfo { return ModelInfo{} }

// Ready uses HEART's own health route, which never builds the model; the registry row
// would say the same but HEART is lazy and the health route is the cheap probe.
func (h *HeartModel) Ready(ctx context.Context) Readiness {
	h.mu.Lock()
	defer h.mu.Unlock()
	if time.Since(h.lastCheck) < statusTTL {
		return h.ready
	}
	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	hh, err := h.client.HeartHealth(cctx)
	h.lastCheck = time.Now()
	if err != nil {
		h.ready = Readiness{Detail: err.Error()}
		return h.ready
	}
	ver := ""
	if row, e := h.status.get(ctx, "heart"); e == nil {
		ver = row.Version
	}
	d := ""
	if !hh.Ready {
		d = fmt.Sprintf("%s (%.0f%% trained)", hh.Status, hh.Progress*100)
	}
	h.ready = Readiness{Ready: hh.Ready, Version: ver, Detail: d}
	return h.ready
}

func (h *HeartModel) Do(ctx context.Context, t Task) (Result, error) {
	if t.Kind == KindDecide {
		ac, ok := t.Input.(models.AdviseContext)
		if !ok {
			return Result{}, fmt.Errorf("decide input must be models.AdviseContext, got %T", t.Input)
		}
		m, err := h.client.Manifest(ctx)
		if err != nil {
			return Result{}, fmt.Errorf("heart decision: %w", err)
		}
		width, err := m.InputWidth()
		if err != nil {
			return Result{}, fmt.Errorf("heart decision: %w", err)
		}
		r, err := h.client.Decide(ctx, models.EncodeByteV1(models.BuildAdvisePrompt(ac), width))
		if err != nil {
			return Result{}, fmt.Errorf("heart decision: %w", err)
		}
		adv := models.AdviceFromReasoning(*r, ac)
		return Result{Value: adv, Version: r.ModelVersion, Model: "heart"}, nil
	}
	prompt, ok := promptOf(t.Input)
	if !ok {
		return Result{}, fmt.Errorf("heart needs a text prompt, got %T", t.Input)
	}
	text, err := h.client.HeartGenerate(ctx, prompt)
	if errors.Is(err, models.ErrNotReady) {
		h.mu.Lock()
		h.lastCheck = time.Time{} // look again next time
		h.mu.Unlock()
	}
	if err != nil {
		return Result{}, fmt.Errorf("heart: %w", err)
	}
	h.mu.Lock()
	ver := h.ready.Version
	h.mu.Unlock()
	return Result{Text: text, Model: "heart", Version: ver}, nil
}

// Choose asks the decision core (now HEART) which eligible model should take the task.
// A model trained without that head abstains, which sends the harness to its rule
// default; that is remembered for a few minutes so each call is not two wasted requests.
func (h *HeartModel) Choose(ctx context.Context, t Task, eligible []string) (Choice, error) {
	m, err := h.client.Manifest(ctx)
	if err != nil {
		return Choice{}, err
	}
	h.mu.Lock()
	if at, ok := h.noRoute[m.Version]; ok && time.Since(at) < 5*time.Minute {
		h.mu.Unlock()
		return Choice{Abstain: true, Reason: "HEART has no route head", Version: m.Version}, nil
	}
	h.mu.Unlock()

	width, err := m.InputWidth()
	if err != nil {
		return Choice{}, err
	}
	r, err := h.client.Decide(ctx, models.EncodeByteV1(routePrompt(t, eligible), width))
	if err != nil {
		return Choice{}, err
	}
	d := r.Decision("route")
	if d == nil || d.Kind != "choice" {
		h.mu.Lock()
		if h.noRoute == nil {
			h.noRoute = map[string]time.Time{}
		}
		h.noRoute[r.ModelVersion] = time.Now()
		h.mu.Unlock()
		return Choice{Abstain: true, Reason: "HEART has no route head", Version: r.ModelVersion}, nil
	}
	if d.Abstain || d.Selected == nil {
		return Choice{Abstain: true, Reason: "route head abstained", Version: r.ModelVersion}, nil
	}
	c := Choice{Model: *d.Selected, Version: r.ModelVersion}
	if d.Confidence != nil {
		c.Confidence = *d.Confidence
	}
	return c, nil
}

// ---- HASHER ---------------------------------------------------------------

// HasherModel is the validator. Input is models.HasherValidateRequest; the verdict is
// Result.Value (*models.HasherValidateResult).
type HasherModel struct {
	client *models.Client
	status *statusCache
}

func (h *HasherModel) Name() string    { return "hasher" }
func (h *HasherModel) Info() ModelInfo { return ModelInfo{} }
func (h *HasherModel) Ready(ctx context.Context) Readiness {
	return h.status.readiness(ctx, "hasher")
}

func (h *HasherModel) Do(ctx context.Context, t Task) (Result, error) {
	switch t.Kind {
	case KindValidate:
		req, ok := t.Input.(models.HasherValidateRequest)
		if !ok {
			return Result{}, fmt.Errorf("validate input must be models.HasherValidateRequest, got %T", t.Input)
		}
		res, err := h.client.HasherValidate(ctx, req)
		if err != nil {
			return Result{}, fmt.Errorf("hasher: %w", err)
		}
		row, _ := h.status.get(ctx, "hasher")
		return Result{Value: res, Model: "hasher", Version: row.Version}, nil
	case KindVerify:
		var req models.VerifyRequest
		switch in := t.Input.(type) {
		case models.VerifyRequest:
			req = in
		case *models.VerifyRequest:
			if in == nil {
				return Result{}, fmt.Errorf("verify input is nil")
			}
			req = *in
		case string:
			req = models.VerifyRequest{Text: in}
		default:
			return Result{}, fmt.Errorf("verify input must be models.VerifyRequest or text, got %T", t.Input)
		}
		return h.doVerify(ctx, req, t.Constraints)
	}
	return Result{}, fmt.Errorf("hasher does not take %q tasks", t.Kind)
}

// Verify checks a text or claims against HASHER's evidence index through the SDK
// client, applying the constraints (see doVerify). It does not route or trace; use
// Harness.Verify for that.
func (h *HasherModel) Verify(ctx context.Context, req models.VerifyRequest, c Constraints) (*models.VerifyResponse, error) {
	res, err := h.doVerify(ctx, req, c)
	if err != nil {
		return nil, err
	}
	return res.Value.(*models.VerifyResponse), nil
}

// VerifyJob reads a deferred or out-of-budget job; only its tenant can.
func (h *HasherModel) VerifyJob(ctx context.Context, id, tenant string) (*models.VerifyResponse, error) {
	resp, err := h.client.VerifyJob(ctx, id, tenant)
	if err != nil {
		return nil, fmt.Errorf("hasher: %w", err)
	}
	if err := resp.Check(); err != nil {
		return nil, fmt.Errorf("hasher: malformed verify answer: %w", err)
	}
	return resp, nil
}

// doVerify applies the constraints, calls the daemon and refuses an answer that breaks
// the contract or a constraint:
//
//   - NoFlash (and LocalOnly, which implies it) forces mode "index"; a response that
//     says another mode ran is discarded as a constraint violation, so a privacy
//     request can never be answered by a live lookup.
//   - A latency Budget becomes the request's budget_ms and, unless the caller chose a
//     delivery, makes it inline. An answer that is still pending is returned marked
//     Degraded with the job id in the Trace: the caller has no verdict yet.
//   - An answer that fails VerifyResponse.Check is an error, never a result.
func (h *HasherModel) doVerify(ctx context.Context, req models.VerifyRequest, c Constraints) (Result, error) {
	var notes []string
	if req.Mode == "" {
		req.Mode = models.ModeIndex
	}
	noFlash := c.NoFlash || c.LocalOnly
	if noFlash && req.Mode != models.ModeIndex {
		notes = append(notes, fmt.Sprintf("constraint: no flash, mode %q changed to index", req.Mode))
		req.Mode = models.ModeIndex
	}
	if c.Budget > 0 {
		ms := int(c.Budget / time.Millisecond)
		if ms < 1 {
			ms = 1
		}
		req.BudgetMS = ms
		if req.Delivery == "" {
			req.Delivery = models.DeliveryInline
		}
	}
	resp, err := h.client.Verify(ctx, req)
	if err != nil {
		return Result{}, fmt.Errorf("hasher: %w", err)
	}
	if err := resp.Check(); err != nil {
		return Result{}, fmt.Errorf("hasher: malformed verify answer: %w", err)
	}
	if noFlash && (resp.Mode != models.ModeIndex || resp.FlashUsed || len(resp.OutboundQueries) > 0) {
		return Result{}, fmt.Errorf("hasher: constraint violated: no flash was required but the answer ran in mode %q (flash used: %t)", resp.Mode, resp.FlashUsed)
	}
	row, _ := h.status.get(ctx, "hasher")
	vt := verifyTrace(resp)
	res := Result{Value: resp, Model: "hasher", Version: row.Version, Verify: vt, Notes: notes}
	if resp.Degraded {
		res.Degraded = true
		res.Notes = append(res.Notes, "verification degraded: the answer is uncommitted and is not a verdict")
	}
	if resp.Status == models.VerifyPending {
		res.Degraded = true
		res.Notes = append(res.Notes, "no verdict yet: read job "+resp.JobID)
	}
	return res, nil
}

// verifyTrace summarises an answer for the Trace. It copies no text.
func verifyTrace(r *models.VerifyResponse) *VerifyTrace {
	vt := &VerifyTrace{Mode: r.Mode, Delivery: r.Delivery, Gate: r.Gate, GateLevel: r.GateLevel, Claims: len(r.Claims),
		Degraded: r.Degraded, Pending: r.Status == models.VerifyPending, JobID: r.JobID, IndexVersion: r.IndexVersion,
		FlashUsed: r.FlashUsed || r.Mode == models.ModeFlash || r.Mode == models.ModeHybrid, OutboundQueries: r.OutboundQueries}
	if r.Judge != nil {
		vt.Judge = r.Judge.Model
		if r.Judge.SHA256 != "" {
			vt.Judge += "@" + r.Judge.SHA256
		}
	}
	if len(r.Claims) > 0 {
		hit := 0
		for _, c := range r.Claims {
			if len(c.Evidence) > 0 {
				hit++
			}
		}
		vt.HitRate = float64(hit) / float64(len(r.Claims))
	}
	return vt
}

// ---- external -------------------------------------------------------------

// ExternalModel is the existing provider chain (Delegator/Orchestrator). It is the
// only model that sends data off the machine and the only paid one. The hooks are
// supplied by InferenceService so this file does not reach into its internals.
type ExternalModel struct {
	GenerateFn func(ctx context.Context, prompt string) (string, error)
	PlanFn     func(ctx context.Context, prompt string) (string, error)
}

func (e *ExternalModel) Name() string    { return "external" }
func (e *ExternalModel) Info() ModelInfo { return ModelInfo{External: true, Paid: true} }
func (e *ExternalModel) Ready(context.Context) Readiness {
	return Readiness{Ready: e.GenerateFn != nil || e.PlanFn != nil, Version: "provider-chain"}
}

func (e *ExternalModel) Do(ctx context.Context, t Task) (Result, error) {
	prompt, ok := promptOf(t.Input)
	if !ok {
		return Result{}, fmt.Errorf("external needs a text prompt, got %T", t.Input)
	}
	fn := e.GenerateFn
	if t.Kind == KindPlan {
		fn = e.PlanFn
	}
	if fn == nil {
		return Result{}, fmt.Errorf("external has no handler for %q", t.Kind)
	}
	text, err := fn(ctx, prompt)
	if err != nil {
		return Result{}, err
	}
	return Result{Text: text, Model: "external", Version: "provider-chain"}, nil
}

// ---- llama (+ ULoRA) ------------------------------------------------------

// LlamaInput is a prompt with optional system text and token limit.
type LlamaInput struct {
	Prompt string
	// System is task-specific context. It is sent inside the user turn, under
	// the stage frame, which is the only system message a call carries.
	System    string
	MaxTokens int
	// Stage overrides the stage chosen from the task kind (frame registry id).
	Stage string
}

// SkillEnsurer makes a bundle's adapter available in llama (*SkillAdapters).
type SkillEnsurer interface {
	Ensure(ctx context.Context, bundleID, contentHash string) (*SkillAdapter, error)
}

// LlamaModel is the embedded llama.cpp. Base tasks (ServerTask, and the generic
// fallbacks) go through Base when set, so the existing dream-pass provider behaviour
// (system prompt, JSON response format) is unchanged; otherwise through a direct chat
// call. ExecuteSkill tasks bind the skill's adapter first and ALWAYS send an explicit
// adapter name. If the adapter cannot be made available the task is answered by the
// base model and the Result says so (Degraded, no Adapter): a skill that did not run
// is never reported as having run.
type LlamaModel struct {
	HTTP   *http.Client // dials llama.sock
	Base   func(ctx context.Context, prompt string) (string, error)
	Skills SkillEnsurer
	Model  string // "model" field sent to llama (it ignores the value)

	mu        sync.Mutex
	lastCheck time.Time
	ready     Readiness
}

const llamaHost = "http://" + llamaSocketHost

// NewLlamaModel builds the model over llama's unix socket.
func NewLlamaModel(socket string, skills SkillEnsurer) *LlamaModel {
	return &LlamaModel{HTTP: models.NewUnixHTTPClient(socket, DefaultHTTPTimeout), Skills: skills, Model: "local-llama"}
}

func (l *LlamaModel) Name() string    { return "llama" }
func (l *LlamaModel) Info() ModelInfo { return ModelInfo{} }

// Ready is llama's own /health, plus the base model hash from /v1/adapters as version.
func (l *LlamaModel) Ready(ctx context.Context) Readiness {
	l.mu.Lock()
	defer l.mu.Unlock()
	if time.Since(l.lastCheck) < statusTTL {
		return l.ready
	}
	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	l.lastCheck = time.Now()
	code, _, err := l.get(cctx, "/health")
	if err != nil || code != 200 {
		d := "health check failed"
		if err != nil {
			d = err.Error()
		}
		l.ready = Readiness{Detail: d}
		return l.ready
	}
	l.ready = Readiness{Ready: true, Version: "llama"}
	var ad llamaAdapters
	if code, body, err := l.get(cctx, "/v1/adapters"); err == nil && code == 200 && json.Unmarshal(body, &ad) == nil && ad.Base.SHA256 != "" {
		sha := ad.Base.SHA256
		if len(sha) > 12 {
			sha = sha[:12]
		}
		l.ready.Version = "gguf-" + sha
	}
	return l.ready
}

func (l *LlamaModel) get(ctx context.Context, path string) (int, []byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, llamaHost+path, nil)
	if err != nil {
		return 0, nil, err
	}
	res, err := l.HTTP.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	return res.StatusCode, b, nil
}

// chat is one /v1/chat/completions call. adapter "" sends no adapter field.
func (l *LlamaModel) chat(ctx context.Context, in LlamaInput, adapter string) (string, error) {
	if in.Stage == "" {
		in.Stage = "inferencer.task"
	}
	fr, err := frame.Default().Build(in.Stage, frame.LiveFacts{Model: l.Model})
	if err != nil {
		return "", fmt.Errorf("llama: %w", err)
	}
	user := in.Prompt
	if in.System != "" {
		user = "CONTEXT:\n" + in.System + "\n\nTASK:\n" + in.Prompt
	}
	msgs := []map[string]string{
		{"role": "system", "content": fr.System},
		{"role": "user", "content": user},
	}
	req := map[string]any{"model": l.Model, "messages": msgs,
		"knirv_stage": fr.Stage, "knirv_frame_hash": fr.Hash, "cache_prompt": true}
	if s, ok := frame.Default().Stage(fr.Stage); ok {
		req["knirv_priority"] = int(s.Priority)
	}
	if in.MaxTokens > 0 {
		req["max_tokens"] = in.MaxTokens
	}
	if adapter != "" {
		req["adapter"] = adapter
	}
	raw, _ := json.Marshal(req)
	hr, err := http.NewRequestWithContext(ctx, http.MethodPost, llamaHost+"/v1/chat/completions", bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	hr.Header.Set("Content-Type", "application/json")
	res, err := l.HTTP.Do(hr)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(res.Body, 8<<20))
	if res.StatusCode != 200 {
		return "", fmt.Errorf("llama HTTP %d: %s", res.StatusCode, strings.TrimSpace(string(body)))
	}
	var out struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &out); err != nil || len(out.Choices) == 0 {
		return "", fmt.Errorf("llama returned no completion")
	}
	return out.Choices[0].Message.Content, nil
}

func (l *LlamaModel) base(ctx context.Context, in LlamaInput) (string, error) {
	if l.Base != nil && in.System == "" && in.MaxTokens == 0 {
		return l.Base(ctx, in.Prompt)
	}
	return l.chat(ctx, in, "")
}

func (l *LlamaModel) Do(ctx context.Context, t Task) (Result, error) {
	in, ok := llamaInputOf(t.Input)
	if !ok {
		return Result{}, fmt.Errorf("llama needs a prompt, got %T", t.Input)
	}
	version := l.Ready(ctx).Version
	if in.Stage == "" {
		in.Stage = "inferencer.task"
		if t.Kind == KindExecuteSkill && t.Skill != nil && t.Skill.BundleID != "" {
			in.Stage = frame.SkillPrefix + t.Skill.BundleID
		}
	}
	if t.Kind != KindExecuteSkill {
		text, err := l.base(ctx, in)
		if err != nil {
			return Result{}, fmt.Errorf("llama: %w", err)
		}
		return Result{Text: text, Model: "llama", Version: version}, nil
	}

	// ExecuteSkill.
	if t.Skill == nil || t.Skill.BundleID == "" {
		return Result{}, errors.New("execute_skill needs a skill bundle")
	}
	degrade := func(why error) (Result, error) {
		text, err := l.base(ctx, in)
		if err != nil {
			return Result{}, fmt.Errorf("skill %s did not run (%v) and the base model failed too: %w", t.Skill.BundleID, why, err)
		}
		return Result{Text: text, Model: "llama", Version: version, Degraded: true,
			Notes: []string{fmt.Sprintf("skill %s did NOT run: %v; this is the base model's answer", t.Skill.BundleID, why)}}, nil
	}
	if l.Skills == nil {
		return degrade(errors.New("no ULoRA adapter service configured"))
	}
	ad, err := l.Skills.Ensure(ctx, t.Skill.BundleID, t.Skill.ContentHash)
	if err != nil {
		return degrade(err)
	}
	text, err := l.chat(ctx, in, ad.Name)
	if err != nil {
		return degrade(fmt.Errorf("adapter %s failed: %w", ad.Name, err))
	}
	return Result{Text: text, Model: "llama", Version: version, Adapter: ad.Name,
		Notes: []string{fmt.Sprintf("adapter %s (bundle %s, base %s)", ad.Name, short(ad.BundleHash), short(ad.BaseSHA256))}}, nil
}

func short(s string) string {
	if len(s) > 12 {
		return s[:12]
	}
	return s
}

func promptOf(in any) (string, bool) {
	switch v := in.(type) {
	case string:
		return v, true
	case LlamaInput:
		return v.Prompt, true
	}
	return "", false
}

func llamaInputOf(in any) (LlamaInput, bool) {
	switch v := in.(type) {
	case string:
		return LlamaInput{Prompt: v}, true
	case LlamaInput:
		return v, true
	}
	return LlamaInput{}, false
}
