package inferencer

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	models "github.com/guiperry/knirv-sdk-go/models"
)

// fakeDaemon serves the routes the three daemon-backed models use.
type fakeDaemon struct {
	mu        sync.Mutex
	registry  []models.ModelStatus
	regHits   int
	routeHead bool // whether HEAD's reasoning has a "route" decision
	routePick string
	reasons   int
}

func (d *fakeDaemon) client(t *testing.T) *models.Client {
	mux := http.NewServeMux()
	mux.HandleFunc(models.RegistryPath, func(w http.ResponseWriter, _ *http.Request) {
		d.mu.Lock()
		d.regHits++
		d.mu.Unlock()
		_ = json.NewEncoder(w).Encode(d.registry)
	})
	mux.HandleFunc(models.HeartPrefix+"/health", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"available":true,"status":"online","loaded":true,"ready":true,"progress":1}`))
	})
	mux.HandleFunc(models.HeartPrefix+"/generate", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"text":"from heart"}`))
	})
	mux.HandleFunc(models.HasherPrefix+"/validate", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"prediction":3,"valid":true,"domain":"d","rules_applied":2}`))
	})
	mux.HandleFunc(models.HeadPrefix+"/models", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"version":"head-9","config":{"max_seq_len":40,"summary_slots":8}}`))
	})
	mux.HandleFunc(models.HeadPrefix+"/reason", func(w http.ResponseWriter, _ *http.Request) {
		d.mu.Lock()
		d.reasons++
		d.mu.Unlock()
		dec := `[]`
		if d.routeHead {
			dec = `[{"name":"route","kind":"choice","options":["heart","llama"],"probabilities":[0.1,0.9],"selected":"` + d.routePick + `","confidence":0.9,"abstain":false}]`
		}
		_, _ = w.Write([]byte(`{"plan_tokens":[],"halting_steps":1,"confidence":0.5,"decisions":` + dec + `,"model_version":"head-9"}`))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return models.NewClient(srv.URL, nil)
}

func rows(ready ...bool) []models.ModelStatus {
	names := []string{"head", "heart", "hasher"}
	var out []models.ModelStatus
	for i, n := range names {
		out = append(out, models.ModelStatus{Name: n, Version: n + "-ver", Ready: ready[i]})
	}
	return out
}

func TestDaemonModelsReadinessSharesOneRegistryRead(t *testing.T) {
	d := &fakeDaemon{registry: rows(true, true, false)}
	head, heart, hasher := DaemonModels(d.client(t))
	ctx := context.Background()
	if r := head.Ready(ctx); !r.Ready || r.Version != "head-ver" {
		t.Fatalf("head %+v", r)
	}
	if r := hasher.Ready(ctx); r.Ready || r.Detail == "" {
		t.Fatalf("hasher not ready must say why: %+v", r)
	}
	if r := heart.Ready(ctx); !r.Ready || r.Version != "heart-ver" {
		t.Fatalf("heart %+v", r)
	}
	if d.regHits != 1 {
		t.Fatalf("three readiness checks should cost one registry read, got %d", d.regHits)
	}
}

func TestDaemonUnreachableIsNotReady(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	srv.Close()
	head, heart, _ := DaemonModels(models.NewClient(srv.URL, nil))
	if head.Ready(context.Background()).Ready || heart.Ready(context.Background()).Ready {
		t.Fatal("an unreachable daemon is not ready")
	}
}

func TestHeadDoDecideNeverErrorsOnAbstain(t *testing.T) {
	d := &fakeDaemon{registry: rows(true, true, true)}
	head, _, _ := DaemonModels(d.client(t))
	res, err := head.Do(context.Background(), Task{Kind: KindDecide, Input: models.AdviseContext{CurrentPhase: "build"}})
	if err != nil {
		t.Fatal(err)
	}
	adv, ok := res.Value.(models.SupervisorAdvice)
	if !ok || !adv.Abstain { // the fake reasoning has no action head
		t.Fatalf("expected an abstention, got %+v", res.Value)
	}
	if _, err := head.Do(context.Background(), Task{Kind: KindDecide, Input: "wrong type"}); err == nil {
		t.Fatal("wrong input type must be rejected")
	}
}

func TestHeadChooseRouteHead(t *testing.T) {
	d := &fakeDaemon{registry: rows(true, true, true), routeHead: true, routePick: "llama"}
	head, _, _ := DaemonModels(d.client(t))
	c, err := head.Choose(context.Background(), Task{Kind: KindGenerate}, []string{"heart", "llama"})
	if err != nil || c.Model != "llama" || c.Confidence < 0.8 || c.Version != "head-9" {
		t.Fatalf("%+v %v", c, err)
	}
}

func TestHeadChooseWithoutRouteHeadAbstainsAndIsRemembered(t *testing.T) {
	d := &fakeDaemon{registry: rows(true, true, true)}
	head, _, _ := DaemonModels(d.client(t))
	for i := 0; i < 3; i++ {
		c, err := head.Choose(context.Background(), Task{Kind: KindGenerate}, []string{"heart", "llama"})
		if err != nil || !c.Abstain || !strings.Contains(c.Reason, "no route head") {
			t.Fatalf("%+v %v", c, err)
		}
	}
	if d.reasons != 1 {
		t.Fatalf("the missing head should be remembered; reason calls = %d", d.reasons)
	}
}

func TestHeartAndHasherDo(t *testing.T) {
	d := &fakeDaemon{registry: rows(true, true, true)}
	_, heart, hasher := DaemonModels(d.client(t))
	ctx := context.Background()
	heart.Ready(ctx)
	res, err := heart.Do(ctx, Task{Kind: KindGenerate, Input: "hi"})
	if err != nil || res.Text != "from heart" || res.Version != "heart-ver" {
		t.Fatalf("%+v %v", res, err)
	}
	vr, err := hasher.Do(ctx, Task{Kind: KindValidate, Input: models.HasherValidateRequest{Prediction: 3, Domain: "d"}})
	if err != nil {
		t.Fatal(err)
	}
	if v, ok := vr.Value.(*models.HasherValidateResult); !ok || !v.Valid || vr.Version != "hasher-ver" {
		t.Fatalf("%+v", vr)
	}
	if _, err := heart.Do(ctx, Task{Kind: KindGenerate, Input: 42}); err == nil {
		t.Fatal("non-text input")
	}
}

// ---- llama ----------------------------------------------------------------

type fakeLlama struct {
	mu       sync.Mutex
	adapters []string // adapter field of each chat request ("" = none)
	rejectAd bool
	down     bool
}

func (f *fakeLlama) model(t *testing.T, skills SkillEnsurer) *LlamaModel {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		if f.down {
			w.WriteHeader(503)
			return
		}
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	mux.HandleFunc("/v1/adapters", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"base":{"path":"/m.gguf","sha256":"abcdef0123456789"},"adapters":[]}`))
	})
	mux.HandleFunc("/v1/chat/completions", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Adapter  string `json:"adapter"`
			Messages []struct {
				Role, Content string
			} `json:"messages"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		f.mu.Lock()
		f.adapters = append(f.adapters, req.Adapter)
		f.mu.Unlock()
		if req.Adapter != "" && f.rejectAd {
			w.WriteHeader(404)
			_, _ = w.Write([]byte(`{"error":"unknown adapter"}`))
			return
		}
		who := "base"
		if req.Adapter != "" {
			who = "adapter:" + req.Adapter
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]any{"content": who + " says " + req.Messages[len(req.Messages)-1].Content}}}})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	addr := srv.Listener.Addr().String()
	hc := &http.Client{Transport: &http.Transport{DialContext: func(ctx context.Context, n, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, n, addr)
	}}}
	return &LlamaModel{HTTP: hc, Skills: skills, Model: "m"}
}

type fakeSkills struct {
	ad  *SkillAdapter
	err error
	got [2]string
}

func (f *fakeSkills) Ensure(_ context.Context, id, hash string) (*SkillAdapter, error) {
	f.got = [2]string{id, hash}
	return f.ad, f.err
}

func TestLlamaReadyAndBaseTask(t *testing.T) {
	fl := &fakeLlama{}
	m := fl.model(t, nil)
	rd := m.Ready(context.Background())
	if !rd.Ready || rd.Version != "gguf-abcdef012345" {
		t.Fatalf("%+v", rd)
	}
	res, err := m.Do(context.Background(), Task{Kind: KindServerTask, Input: "dream"})
	if err != nil || res.Text != "base says dream" || res.Adapter != "" || res.Degraded {
		t.Fatalf("%+v %v", res, err)
	}
	fl.down = true
	m.lastCheck = m.lastCheck.AddDate(-1, 0, 0)
	if m.Ready(context.Background()).Ready {
		t.Fatal("llama down must not be ready")
	}
}

func TestLlamaBaseHookKeepsExistingProviderBehaviour(t *testing.T) {
	fl := &fakeLlama{}
	m := fl.model(t, nil)
	m.Base = func(_ context.Context, p string) (string, error) { return "provider:" + p, nil }
	res, _ := m.Do(context.Background(), Task{Kind: KindServerTask, Input: "x"})
	if res.Text != "provider:x" || len(fl.adapters) != 0 {
		t.Fatalf("server tasks must go through the existing provider: %+v adapters=%v", res, fl.adapters)
	}
}

func TestExecuteSkillRunsTheAdapterAndSaysSo(t *testing.T) {
	fl := &fakeLlama{}
	sk := &fakeSkills{ad: &SkillAdapter{Name: "skill-abc", BundleHash: "abc123", BaseSHA256: "def456"}}
	m := fl.model(t, sk)
	h := NewHarness([]HarnessModel{m})
	res, tr, err := h.ExecuteSkill(context.Background(), SkillRef{BundleID: "b1", ContentHash: "abc123"}, "solve", Constraints{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Text != "adapter:skill-abc says solve" || res.Adapter != "skill-abc" || res.Degraded {
		t.Fatalf("%+v", res)
	}
	if tr.Adapter != "skill-abc" || tr.Degraded || tr.Skill != "b1" || sk.got != [2]string{"b1", "abc123"} {
		t.Fatalf("trace must record the adapter: %+v got=%v", tr, sk.got)
	}
	if len(fl.adapters) != 1 || fl.adapters[0] != "skill-abc" {
		t.Fatalf("every skill request carries an explicit adapter: %v", fl.adapters)
	}
}

func TestExecuteSkillDegradesToBaseAndNeverPretends(t *testing.T) {
	cases := map[string]struct {
		skills SkillEnsurer
		reject bool
		why    string
	}{
		"ensure fails":     {&fakeSkills{err: errors.New("ulora refused to bind: base mismatch")}, false, "base mismatch"},
		"no ulora":         {nil, false, "no ULoRA"},
		"adapter rejected": {&fakeSkills{ad: &SkillAdapter{Name: "skill-x"}}, true, "adapter skill-x failed"},
	}
	for name, c := range cases {
		fl := &fakeLlama{rejectAd: c.reject}
		h := NewHarness([]HarnessModel{fl.model(t, c.skills)})
		res, tr, err := h.ExecuteSkill(context.Background(), SkillRef{BundleID: "b1"}, "solve", Constraints{})
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if res.Text != "base says solve" || res.Adapter != "" || !res.Degraded {
			t.Errorf("%s: must be the base answer, flagged: %+v", name, res)
		}
		if tr.Adapter != "" || !tr.Degraded || len(tr.Notes) != 1 || !strings.Contains(tr.Notes[0], "did NOT run") || !strings.Contains(tr.Notes[0], c.why) {
			t.Errorf("%s: trace must say the skill did not run: %+v", name, tr)
		}
	}
}

func TestExecuteSkillFailsWhenBaseFailsToo(t *testing.T) {
	fl := &fakeLlama{down: true}
	m := fl.model(t, &fakeSkills{err: errors.New("nope")})
	m.lastCheck = m.lastCheck.AddDate(-1, 0, 0)
	h := NewHarness([]HarnessModel{m})
	if _, tr, err := h.ExecuteSkill(context.Background(), SkillRef{BundleID: "b"}, "p", Constraints{}); err == nil || tr.Chosen != "" {
		t.Fatalf("llama down must error, not fake an answer: %+v %v", tr, err)
	}
	if _, err := m.Do(context.Background(), Task{Kind: KindExecuteSkill, Input: "p"}); err == nil {
		t.Fatal("a skill task without a bundle is invalid")
	}
}

func TestExternalModelInfoAndDispatch(t *testing.T) {
	e := &ExternalModel{GenerateFn: func(_ context.Context, p string) (string, error) { return "gen:" + p, nil },
		PlanFn: func(_ context.Context, p string) (string, error) { return "plan:" + p, nil }}
	if i := e.Info(); !i.External || !i.Paid {
		t.Fatalf("%+v", i)
	}
	g, _ := e.Do(context.Background(), Task{Kind: KindGenerate, Input: "a"})
	p, _ := e.Do(context.Background(), Task{Kind: KindPlan, Input: "a"})
	if g.Text != "gen:a" || p.Text != "plan:a" {
		t.Fatalf("%q %q", g.Text, p.Text)
	}
}
