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

// HeadModel is the reasoner. Decide tasks carry a models.AdviseContext and produce a
// models.SupervisorAdvice in Result.Value; HEAD never errors there, it abstains.
// HeadModel is also the harness's advisory Chooser.
type HeadModel struct {
	client *models.Client
	status *statusCache

	mu      sync.Mutex
	noRoute map[string]time.Time // model version -> when "no route head" was seen
}

func (h *HeadModel) Name() string    { return "head" }
func (h *HeadModel) Info() ModelInfo { return ModelInfo{} }
func (h *HeadModel) Ready(ctx context.Context) Readiness {
	return h.status.readiness(ctx, "head")
}

func (h *HeadModel) Do(ctx context.Context, t Task) (Result, error) {
	if t.Kind != KindDecide {
		return Result{}, fmt.Errorf("head does not take %q tasks", t.Kind)
	}
	ac, ok := t.Input.(models.AdviseContext)
	if !ok {
		return Result{}, fmt.Errorf("decide input must be models.AdviseContext, got %T", t.Input)
	}
	adv := h.client.Advise(ctx, ac)
	return Result{Value: adv, Version: adv.ModelVersion, Model: "head"}, nil
}

// routePrompt is the closed-set question HEAD is asked.
func routePrompt(t Task, eligible []string) string {
	return fmt.Sprintf("route kind=%s eligible=%s skill=%t", t.Kind, strings.Join(eligible, ","), t.Skill != nil)
}

// Choose asks HEAD's "route" decision head which eligible model should take the task.
// A model trained without that head abstains, which sends the harness to its rule
// default; that is remembered for a few minutes so each call is not two wasted requests.
func (h *HeadModel) Choose(ctx context.Context, t Task, eligible []string) (Choice, error) {
	m, err := h.client.Manifest(ctx)
	if err != nil {
		return Choice{}, err
	}
	h.mu.Lock()
	if at, ok := h.noRoute[m.Version]; ok && time.Since(at) < 5*time.Minute {
		h.mu.Unlock()
		return Choice{Abstain: true, Reason: "HEAD has no route head", Version: m.Version}, nil
	}
	h.mu.Unlock()

	width, err := m.InputWidth()
	if err != nil {
		return Choice{}, err
	}
	r, err := h.client.Reason(ctx, models.EncodeByteV1(routePrompt(t, eligible), width))
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
		return Choice{Abstain: true, Reason: "HEAD has no route head", Version: r.ModelVersion}, nil
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

// ---- HEART ----------------------------------------------------------------

// HeartModel is the generator over the daemon's HEART routes. Input is a string.
type HeartModel struct {
	client *models.Client
	status *statusCache

	mu        sync.Mutex
	ready     Readiness
	lastCheck time.Time
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
	if t.Kind != KindValidate {
		return Result{}, fmt.Errorf("hasher does not take %q tasks", t.Kind)
	}
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
