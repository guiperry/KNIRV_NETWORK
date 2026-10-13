package inferencer

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

type fakeModel struct {
	name  string
	info  ModelInfo
	ready Readiness
	do    func(Task) (Result, error)

	mu    sync.Mutex
	calls int
}

func (f *fakeModel) Name() string                    { return f.name }
func (f *fakeModel) Info() ModelInfo                 { return f.info }
func (f *fakeModel) Ready(context.Context) Readiness { return f.ready }
func (f *fakeModel) Calls() int                      { f.mu.Lock(); defer f.mu.Unlock(); return f.calls }
func (f *fakeModel) Do(_ context.Context, t Task) (Result, error) {
	f.mu.Lock()
	f.calls++
	f.mu.Unlock()
	if f.do != nil {
		return f.do(t)
	}
	return Result{Text: f.name + " answer", Version: f.ready.Version}, nil
}

func up(name string) *fakeModel {
	return &fakeModel{name: name, ready: Readiness{Ready: true, Version: name + "-v1"}}
}

func ext() *fakeModel {
	m := up("external")
	m.info = ModelInfo{External: true, Paid: true}
	return m
}

type fakeChooser struct {
	mu  sync.Mutex
	c   Choice
	err error
	got []string
}

func (f *fakeChooser) Choose(_ context.Context, _ Task, eligible []string) (Choice, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.got = eligible
	return f.c, f.err
}

func TestRoutingTable(t *testing.T) {
	models := []HarnessModel{up("head"), up("heart"), up("hasher"), up("llama"), ext()}
	h := NewHarness(models)
	want := map[Kind]string{
		KindDecide: "head", KindAnalyze: "heart", KindGenerate: "heart", KindValidate: "hasher",
		KindServerTask: "llama", KindExecuteSkill: "llama", KindPlan: "external",
	}
	for k, w := range want {
		_, tr, err := h.Do(context.Background(), Task{Kind: k, Input: "x", Skill: &SkillRef{BundleID: "b"}})
		if err != nil || tr.Chosen != w {
			t.Errorf("%s: chose %q err %v, want %q", k, tr.Chosen, err, w)
		}
		if tr.Version != w+"-v1" {
			t.Errorf("%s: trace version %q", k, tr.Version)
		}
	}
	if _, tr, err := h.Do(context.Background(), Task{Kind: "bogus"}); err == nil || !errors.Is(err, ErrNoRoute) || tr.Error == "" {
		t.Fatalf("unknown kind must fail with ErrNoRoute and a trace error: %v %+v", err, tr)
	}
}

func TestFallbackIsRecorded(t *testing.T) {
	heart := up("heart")
	heart.do = func(Task) (Result, error) { return Result{}, errors.New("heart exploded") }
	llama, e := up("llama"), ext()
	h := NewHarness([]HarnessModel{heart, llama, e})
	res, tr, err := h.Do(context.Background(), Task{Kind: KindGenerate, Input: "p"})
	if err != nil || res.Text != "llama answer" || tr.Chosen != "llama" {
		t.Fatalf("res %+v tr %+v err %v", res, tr, err)
	}
	if len(tr.Fallbacks) != 1 || tr.Fallbacks[0].From != "heart" || !strings.Contains(tr.Fallbacks[0].Error, "exploded") {
		t.Fatalf("fallbacks: %+v", tr.Fallbacks)
	}
	if e.Calls() != 0 {
		t.Fatal("external must not be called when llama answered")
	}
}

func TestNotReadyIsSkippedAndSaidSo(t *testing.T) {
	heart := up("heart")
	heart.ready = Readiness{Detail: "idle (0% trained)"}
	h := NewHarness([]HarnessModel{heart, up("llama"), ext()})
	_, tr, err := h.Do(context.Background(), Task{Kind: KindGenerate, Input: "p"})
	if err != nil || tr.Chosen != "llama" {
		t.Fatalf("%+v %v", tr, err)
	}
	if heart.Calls() != 0 || !strings.Contains(tr.Candidates[0].Excluded, "idle") {
		t.Fatalf("heart should be excluded with its reason: %+v", tr.Candidates)
	}
}

func TestPrivacyConstraintsNeverReachExternal(t *testing.T) {
	heart := up("heart")
	heart.ready = Readiness{Detail: "down"}
	llama := up("llama")
	llama.ready = Readiness{Detail: "down"}
	e := ext()
	h := NewHarness([]HarnessModel{heart, llama, e})

	_, tr, err := h.Do(context.Background(), Task{Kind: KindGenerate, Input: "secret", Constraints: Constraints{LocalOnly: true}})
	if !errors.Is(err, ErrNoRoute) || e.Calls() != 0 {
		t.Fatalf("local-only task must fail rather than leave the box: err=%v external calls=%d", err, e.Calls())
	}
	if !strings.Contains(tr.Candidates[2].Excluded, "local") {
		t.Fatalf("trace must say why: %+v", tr.Candidates)
	}
	_, tr, err = h.Do(context.Background(), Task{Kind: KindPlan, Input: "p", Constraints: Constraints{NoSpend: true}})
	if !errors.Is(err, ErrNoRoute) || e.Calls() != 0 || !strings.Contains(tr.Candidates[0].Excluded, "spend") {
		t.Fatalf("no-spend task must not use the paid model: %v %+v", err, tr.Candidates)
	}
	// Without the constraint the same situation is served externally.
	if _, tr, err = h.Do(context.Background(), Task{Kind: KindGenerate, Input: "p"}); err != nil || tr.Chosen != "external" {
		t.Fatalf("%+v %v", tr, err)
	}
}

type clock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *clock) now() time.Time      { c.mu.Lock(); defer c.mu.Unlock(); return c.t }
func (c *clock) add(d time.Duration) { c.mu.Lock(); c.t = c.t.Add(d); c.mu.Unlock() }

func TestBreakerOpensAndRecovers(t *testing.T) {
	clk := &clock{t: time.Unix(1000, 0)}
	mk := func() *Breaker { b := NewBreaker(2, time.Minute); b.now = clk.now; return b }
	flaky := up("heart")
	failing := true
	flaky.do = func(Task) (Result, error) {
		if failing {
			return Result{}, errors.New("boom")
		}
		return Result{Text: "heart ok"}, nil
	}
	llama := up("llama")
	h := NewHarness([]HarnessModel{flaky, llama}, WithBreaker(mk), WithRoutes(map[Kind][]string{KindGenerate: {"heart", "llama"}}))
	ctx := context.Background()
	task := Task{Kind: KindGenerate, Input: "p"}

	for i := 0; i < 2; i++ {
		if _, tr, _ := h.Do(ctx, task); tr.Chosen != "llama" || len(tr.Fallbacks) != 1 {
			t.Fatalf("call %d: %+v", i, tr)
		}
	}
	if st := h.Stats()["heart"]; st.Circuit != "open" || st.Failures != 2 {
		t.Fatalf("stats %+v", st)
	}
	before := flaky.Calls()
	_, tr, _ := h.Do(ctx, task)
	if flaky.Calls() != before || tr.Candidates[0].Excluded != "circuit open" {
		t.Fatalf("open circuit must not call the model: %+v", tr.Candidates)
	}
	clk.add(2 * time.Minute)
	failing = false
	if _, tr, _ := h.Do(ctx, task); tr.Chosen != "heart" {
		t.Fatalf("half-open probe should reach heart: %+v", tr)
	}
	if st := h.Stats()["heart"]; st.Circuit != "closed" {
		t.Fatalf("a successful probe closes the circuit: %+v", st)
	}
}

func TestHalfOpenAllowsOneProbe(t *testing.T) {
	clk := &clock{t: time.Unix(0, 0)}
	b := NewBreaker(1, time.Second)
	b.now = clk.now
	b.Failure()
	clk.add(2 * time.Second)
	if !b.Acquire() || b.Acquire() {
		t.Fatal("exactly one probe may pass while half-open")
	}
	b.Failure()
	if b.Acquire() {
		t.Fatal("a failed probe reopens the circuit")
	}
}

func TestChooserRecommendationIsAdvisory(t *testing.T) {
	mk := func(c Choice, err error) (*Harness, *fakeModel, *fakeModel, *fakeChooser) {
		heart, llama := up("heart"), up("llama")
		ch := &fakeChooser{c: c, err: err}
		return NewHarness([]HarnessModel{heart, llama, ext()}, WithChooser(ch, 0.6)), heart, llama, ch
	}
	run := func(h *Harness) Trace {
		_, tr, err := h.Do(context.Background(), Task{Kind: KindGenerate, Input: "p"})
		if err != nil {
			t.Fatal(err)
		}
		return tr
	}

	h, _, _, ch := mk(Choice{Model: "llama", Confidence: 0.9}, nil)
	if tr := run(h); tr.Chosen != "llama" || tr.Advice == nil || !tr.Advice.Applied {
		t.Fatalf("a confident, allowed recommendation is followed: %+v", tr)
	}
	if len(ch.got) != 3 {
		t.Fatalf("chooser must be offered exactly the eligible set: %v", ch.got)
	}
	cases := map[string]Choice{
		"abstain":      {Abstain: true},
		"rules do not": {Model: "openai", Confidence: 0.99},
		"below":        {Model: "llama", Confidence: 0.2},
	}
	for want, c := range cases {
		h, _, _, _ := mk(c, nil)
		tr := run(h)
		if tr.Chosen != "heart" || tr.Advice == nil || tr.Advice.Applied || !strings.Contains(tr.Advice.Note, want) {
			t.Errorf("%s: rule default must win: chosen %q advice %+v", want, tr.Chosen, tr.Advice)
		}
	}
	h, _, _, _ = mk(Choice{}, errors.New("head down"))
	if tr := run(h); tr.Chosen != "heart" || !strings.Contains(tr.Advice.Note, "head down") {
		t.Fatalf("chooser failure falls back to rules: %+v", tr)
	}
}

func TestChooserNotConsultedWhenNothingToChoose(t *testing.T) {
	ch := &fakeChooser{c: Choice{Model: "llama", Confidence: 1}}
	h := NewHarness([]HarnessModel{up("hasher"), up("llama")}, WithChooser(ch, 0.6))
	if _, tr, _ := h.Do(context.Background(), Task{Kind: KindValidate, Input: "x"}); tr.Advice != nil || ch.got != nil {
		t.Fatalf("validate has one route; HEAD must not be asked: %+v", tr.Advice)
	}
}

func TestBudgetBoundsTheCall(t *testing.T) {
	slow := up("heart")
	slow.do = func(Task) (Result, error) { time.Sleep(300 * time.Millisecond); return Result{}, nil }
	// A model that honours ctx.
	h := NewHarness([]HarnessModel{&ctxModel{fakeModel: slow}}, WithRoutes(map[Kind][]string{KindGenerate: {"heart"}}))
	start := time.Now()
	_, tr, err := h.Do(context.Background(), Task{Kind: KindGenerate, Input: "p", Constraints: Constraints{Budget: 30 * time.Millisecond}})
	if err == nil || time.Since(start) > 250*time.Millisecond || tr.Error == "" {
		t.Fatalf("budget not enforced: %v after %v", err, time.Since(start))
	}
}

type ctxModel struct{ *fakeModel }

func (c *ctxModel) Do(ctx context.Context, t Task) (Result, error) {
	select {
	case <-ctx.Done():
		return Result{}, ctx.Err()
	case <-time.After(300 * time.Millisecond):
		return Result{Text: "late"}, nil
	}
}

func TestTraceRingKeepsNewest(t *testing.T) {
	h := NewHarness([]HarnessModel{up("llama")}, WithTraceCapacity(3), WithRoutes(map[Kind][]string{KindServerTask: {"llama"}}))
	var ids []string
	for i := 0; i < 5; i++ {
		_, tr, _ := h.Do(context.Background(), Task{Kind: KindServerTask, Input: "p"})
		ids = append(ids, tr.ID)
	}
	got := h.Traces(0)
	if len(got) != 3 || got[0].ID != ids[4] || got[2].ID != ids[2] {
		t.Fatalf("want newest-first last 3, got %v (ids %v)", got, ids)
	}
	if len(h.Traces(2)) != 2 {
		t.Fatal("limit")
	}
	st := h.Stats()["llama"]
	if st.Calls != 5 || st.Failures != 0 || st.P95Ms < 0 {
		t.Fatalf("%+v", st)
	}
}

func TestConcurrentDoIsRaceFree(t *testing.T) {
	flaky := up("heart")
	var n int
	var mu sync.Mutex
	flaky.do = func(Task) (Result, error) {
		mu.Lock()
		defer mu.Unlock()
		n++
		if n%3 == 0 {
			return Result{}, errors.New("x")
		}
		return Result{Text: "ok"}, nil
	}
	h := NewHarness([]HarnessModel{flaky, up("llama")}, WithChooser(&fakeChooser{c: Choice{Abstain: true}}, 0.5))
	var wg sync.WaitGroup
	for i := 0; i < 40; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, _, err := h.Do(context.Background(), Task{Kind: KindGenerate, Input: "p"}); err != nil {
				t.Error(err)
			}
			_ = h.Stats()
			_ = h.Traces(5)
		}()
	}
	wg.Wait()
}
