package inferencer

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// The hierarchical reasoning harness: one entry point (Harness.Do) over every model
// KNIRV runs. A deterministic routing table is the authority; HEAD may only choose
// among the models the table already allows. Every call returns a Trace saying what
// was considered, what ran, what failed and why, so nothing is silently substituted.

// Kind is what a task needs done, not which model does it.
type Kind string

const (
	KindDecide       Kind = "decide"        // closed-set supervision advice          -> HEAD
	KindAnalyze      Kind = "analyze"       // read and reason over text               -> HEART, llama, external
	KindGenerate     Kind = "generate"      // produce text                            -> HEART, llama, external
	KindValidate     Kind = "validate"      // check a prediction against domain rules -> HASHER
	KindServerTask   Kind = "server_task"   // operational prompts, dream passes       -> llama (base)
	KindExecuteSkill Kind = "execute_skill" // run a skill's ULoRA on llama            -> llama + adapter
	KindPlan         Kind = "plan"          // multi-step planning                     -> external orchestrator
)

// SkillRef names the compiled bundle an ExecuteSkill task runs.
type SkillRef struct {
	BundleID    string
	ContentHash string // optional; ULoRA refuses a bundle whose hash differs
}

// Constraints are checked BEFORE routing: a model that violates them is never a
// candidate, so a request that must stay on the box cannot reach an external provider.
type Constraints struct {
	Budget    time.Duration // latency budget for the whole call (0 = caller's context only)
	LocalOnly bool          // must not leave this machine
	NoSpend   bool          // must not use a paid model
	// Pin restricts the task to one named model: an operator override ("force internal").
	// It narrows the routing table's chain; it cannot add a model the table does not list.
	Pin string
}

// Task is one unit of work. Input is typed per kind (see each model's Do).
type Task struct {
	Kind        Kind
	Input       any
	Skill       *SkillRef
	Constraints Constraints
}

// Readiness is a model's answer to "can you take work now?".
type Readiness struct {
	Ready   bool
	Version string
	Detail  string
}

// ModelInfo is what routing constraints need to know about a model.
type ModelInfo struct {
	External bool // data leaves this machine
	Paid     bool // each call costs money
}

// Result is a model's answer. Text is set for text-producing kinds, Value carries the
// typed payload for the others (e.g. models.SupervisorAdvice for Decide).
type Result struct {
	Text    string
	Value   any
	Model   string
	Version string
	// Adapter is the ULoRA adapter that actually ran ("" when none did).
	Adapter string
	// Notes are facts the caller must not lose, e.g. that a skill did not run.
	Notes []string
	// Degraded is true when the result is NOT what was asked for (a skill task answered
	// by the base model). The caller decides whether that is acceptable.
	Degraded bool
}

// HarnessModel is one thing the harness can route to.
type HarnessModel interface {
	Name() string
	Info() ModelInfo
	Ready(ctx context.Context) Readiness
	Do(ctx context.Context, t Task) (Result, error)
}

// Chooser is an advisory selector (HEAD). It must answer from the eligible set it is
// given; the harness re-checks that and ignores anything else.
type Chooser interface {
	Choose(ctx context.Context, t Task, eligible []string) (Choice, error)
}

// Choice is a Chooser's recommendation.
type Choice struct {
	Model      string
	Confidence float32
	Abstain    bool
	Reason     string
	Version    string
}

// ErrNoRoute means no model could take the task: every candidate was excluded by a
// constraint, not ready, or failed. The Trace says which.
var ErrNoRoute = errors.New("no model available for task")

// Harness routes tasks over registered models.
type Harness struct {
	mu       sync.RWMutex
	models   map[string]HarnessModel
	routes   map[Kind][]string
	chooser  Chooser
	minConf  float32
	breakers map[string]*Breaker
	traces   *TraceLog
	stats    map[string]*modelStats
	newBreak func() *Breaker
}

// DefaultRoutes is the authoritative routing table: for each kind, the models in the
// order they are tried. Changing it is a code change on purpose.
func DefaultRoutes() map[Kind][]string {
	return map[Kind][]string{
		KindDecide:       {"head"},
		KindAnalyze:      {"heart", "llama", "external"},
		KindGenerate:     {"heart", "llama", "external"},
		KindValidate:     {"hasher"},
		KindServerTask:   {"llama"},
		KindExecuteSkill: {"llama"},
		KindPlan:         {"external"},
	}
}

// UserFacingRoutes is DefaultRoutes for a deployment where Generate/Analyze answer a
// person (the backend's chat): llama is the small embedded "server tasks" model and is
// not a substitute for the provider chain there, so it is left out of those chains. It
// still serves ServerTask and ExecuteSkill.
func UserFacingRoutes() map[Kind][]string {
	r := DefaultRoutes()
	r[KindAnalyze] = []string{"heart", "external"}
	r[KindGenerate] = []string{"heart", "external"}
	return r
}

// ambiguousKinds are the kinds with more than one sensible eligible model, where
// HEAD is consulted. The rest have a single answer and need no recommendation.
var ambiguousKinds = map[Kind]bool{KindAnalyze: true, KindGenerate: true}

// HarnessOption configures NewHarness.
type HarnessOption func(*Harness)

// WithRoutes replaces the routing table (tests, or a deployment without llama).
func WithRoutes(r map[Kind][]string) HarnessOption { return func(h *Harness) { h.routes = r } }

// WithChooser sets the advisory selector; minConfidence is the floor below which its
// recommendation is ignored.
func WithChooser(c Chooser, minConfidence float32) HarnessOption {
	return func(h *Harness) { h.chooser, h.minConf = c, minConfidence }
}

// WithBreaker sets how breakers are made (tests inject a fast clock).
func WithBreaker(f func() *Breaker) HarnessOption { return func(h *Harness) { h.newBreak = f } }

// WithTraceCapacity sets how many traces the ring buffer keeps.
func WithTraceCapacity(n int) HarnessOption { return func(h *Harness) { h.traces = NewTraceLog(n) } }

// NewHarness builds a harness over the given models.
func NewHarness(models []HarnessModel, opts ...HarnessOption) *Harness {
	h := &Harness{
		models:   map[string]HarnessModel{},
		routes:   DefaultRoutes(),
		minConf:  0.6,
		breakers: map[string]*Breaker{},
		traces:   NewTraceLog(256),
		stats:    map[string]*modelStats{},
		newBreak: func() *Breaker { return NewBreaker(3, 30*time.Second) },
	}
	for _, o := range opts {
		o(h)
	}
	for _, m := range models {
		h.models[m.Name()] = m
		h.breakers[m.Name()] = h.newBreak()
		h.stats[m.Name()] = &modelStats{}
	}
	return h
}

// Model returns a registered model by name.
func (h *Harness) Model(name string) (HarnessModel, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	m, ok := h.models[name]
	return m, ok
}

// Do routes and runs one task. The Trace is returned even on error.
func (h *Harness) Do(ctx context.Context, t Task) (Result, Trace, error) {
	start := time.Now()
	if t.Constraints.Budget > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, t.Constraints.Budget)
		defer cancel()
	}
	tr := Trace{ID: newTraceID(), Time: start.UTC(), Kind: t.Kind, LocalOnly: t.Constraints.LocalOnly, NoSpend: t.Constraints.NoSpend}
	if t.Skill != nil {
		tr.Skill = t.Skill.BundleID
	}

	chain, ok := h.routes[t.Kind]
	if !ok {
		tr.Error = fmt.Sprintf("no route for kind %q", t.Kind)
		return h.finish(Result{}, tr, start, fmt.Errorf("%w: %s", ErrNoRoute, tr.Error))
	}

	// 1. Constraints, then readiness and breaker state, in the table's order.
	var eligible []string
	for _, name := range chain {
		c := Candidate{Model: name}
		m, registered := h.Model(name)
		switch {
		case !registered:
			c.Excluded = "not registered"
		case t.Constraints.Pin != "" && name != t.Constraints.Pin:
			c.Excluded = "constraint: pinned to " + t.Constraints.Pin
		case t.Constraints.LocalOnly && m.Info().External:
			c.Excluded = "constraint: must stay local"
		case t.Constraints.NoSpend && m.Info().Paid:
			c.Excluded = "constraint: no spend"
		case !h.breakers[name].Allow():
			c.Excluded = "circuit open"
		default:
			rd := m.Ready(ctx)
			c.Version = rd.Version
			if !rd.Ready {
				c.Excluded = "not ready"
				if rd.Detail != "" {
					c.Excluded += ": " + rd.Detail
				}
			} else {
				eligible = append(eligible, name)
			}
		}
		tr.Candidates = append(tr.Candidates, c)
	}
	if len(eligible) == 0 {
		tr.Error = "no eligible model"
		return h.finish(Result{}, tr, start, fmt.Errorf("%w: %s", ErrNoRoute, summarise(tr.Candidates)))
	}

	// 2. HEAD recommends among the eligible; the rules decide whether to listen.
	order := eligible
	tr.Reason = "rule default: " + eligible[0]
	if ambiguousKinds[t.Kind] && len(eligible) > 1 && h.chooser != nil {
		order, tr.Advice = h.consult(ctx, t, eligible)
		if tr.Advice != nil && tr.Advice.Applied {
			tr.Reason = "HEAD recommended " + order[0] + " (rules allowed it)"
		} else if tr.Advice != nil {
			tr.Reason = "rule default: " + eligible[0] + " (" + tr.Advice.Note + ")"
		}
	}

	// 3. Run down the chain; every hop is recorded.
	var lastErr error
	for _, name := range order {
		m, _ := h.Model(name)
		if !h.breakers[name].Acquire() {
			lastErr = fmt.Errorf("%s: circuit open", name)
			tr.Fallbacks = append(tr.Fallbacks, Fallback{From: name, Error: "circuit open"})
			continue
		}
		hopStart := time.Now()
		res, err := m.Do(ctx, t)
		lat := time.Since(hopStart)
		h.record(name, lat, err)
		if err == nil {
			tr.Chosen, tr.Version, tr.Adapter = name, res.Version, res.Adapter
			tr.Degraded, tr.Notes = res.Degraded, res.Notes
			if res.Model == "" {
				res.Model = name
			}
			return h.finish(res, tr, start, nil)
		}
		lastErr = err
		tr.Fallbacks = append(tr.Fallbacks, Fallback{From: name, Error: err.Error(), LatencyMs: float64(lat.Microseconds()) / 1000})
		if ctx.Err() != nil {
			break
		}
	}
	tr.Error = lastErr.Error()
	return h.finish(Result{}, tr, start, fmt.Errorf("%w: %v", ErrNoRoute, lastErr))
}

// consult asks the chooser, validates the answer against the eligible set, and
// returns the order to try plus what happened.
func (h *Harness) consult(ctx context.Context, t Task, eligible []string) ([]string, *Advice) {
	adv := &Advice{Asked: true}
	c, err := h.chooser.Choose(ctx, t, eligible)
	adv.Version = c.Version
	switch {
	case err != nil:
		adv.Note = "HEAD could not answer: " + err.Error()
	case c.Abstain || c.Model == "":
		adv.Note = "HEAD abstained"
		if c.Reason != "" {
			adv.Note += ": " + c.Reason
		}
	case !contains(eligible, c.Model):
		adv.Note = fmt.Sprintf("HEAD named %q, which the rules do not allow", c.Model)
		adv.Model = c.Model
	case c.Confidence < h.minConf:
		adv.Note = fmt.Sprintf("HEAD confidence %.2f below %.2f", c.Confidence, h.minConf)
		adv.Model, adv.Confidence = c.Model, c.Confidence
	default:
		adv.Applied, adv.Model, adv.Confidence = true, c.Model, c.Confidence
		order := []string{c.Model}
		for _, n := range eligible {
			if n != c.Model {
				order = append(order, n)
			}
		}
		return order, adv
	}
	return eligible, adv
}

func (h *Harness) record(name string, lat time.Duration, err error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	st := h.stats[name]
	st.record(lat, err)
	if err != nil {
		h.breakers[name].Failure()
	} else {
		h.breakers[name].Success()
	}
}

func (h *Harness) finish(res Result, tr Trace, start time.Time, err error) (Result, Trace, error) {
	tr.LatencyMs = float64(time.Since(start).Microseconds()) / 1000
	h.traces.Add(tr)
	return res, tr, err
}

// Traces returns the most recent traces, newest first.
func (h *Harness) Traces(limit int) []Trace { return h.traces.Recent(limit) }

// Stats returns per-model counters.
func (h *Harness) Stats() map[string]ModelStats {
	h.mu.RLock()
	defer h.mu.RUnlock()
	out := make(map[string]ModelStats, len(h.stats))
	for name, st := range h.stats {
		s := st.snapshot()
		s.Circuit = h.breakers[name].State()
		out[name] = s
	}
	return out
}

func contains(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}

func summarise(cs []Candidate) string {
	s := ""
	for i, c := range cs {
		if i > 0 {
			s += "; "
		}
		s += c.Model + ": " + c.Excluded
	}
	return s
}

// ExecuteSkill runs prompt under the skill's ULoRA adapter. The returned Result and
// Trace say whether the skill actually ran: Result.Adapter is non-empty only if it did,
// and Degraded is true when the base model answered instead.
func (h *Harness) ExecuteSkill(ctx context.Context, skill SkillRef, prompt string, c Constraints) (Result, Trace, error) {
	return h.Do(ctx, Task{Kind: KindExecuteSkill, Input: prompt, Skill: &skill, Constraints: c})
}
