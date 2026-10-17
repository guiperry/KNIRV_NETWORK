package teacher

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// fakeHarness answers like the router in front of Colibri. reply picks the text for a
// request; backend is what the X-Knirv-Backend header says.
type fakeHarness struct {
	srv     *httptest.Server
	calls   atomic.Int32
	backend string
	reply   func(system, user string, call int) string
	// busy makes the first n calls answer 503 (Colibri still loading).
	busy atomic.Int32
	role atomic.Value
}

func newFake(t *testing.T, reply func(system, user string, call int) string) (*fakeHarness, *Client) {
	t.Helper()
	f := &fakeHarness{backend: "colibri", reply: reply}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/harness/status":
			w.Write([]byte(`{"active":"colibri","colibri_model":"qwen36-35b"}`))
		case "/v1/chat/completions":
			f.role.Store(r.Header.Get(roleHeader))
			if f.busy.Add(-1) >= 0 {
				w.Header().Set("Retry-After", "1")
				http.Error(w, `{"error":{"message":"no teacher backend"}}`, http.StatusServiceUnavailable)
				return
			}
			var req struct {
				Messages []struct{ Role, Content string } `json:"messages"`
				Max      int                              `json:"max_tokens"`
			}
			json.NewDecoder(r.Body).Decode(&req)
			var sys, usr string
			for _, m := range req.Messages {
				if m.Role == "system" {
					sys = m.Content
				} else {
					usr = m.Content
				}
			}
			n := int(f.calls.Add(1))
			text := f.reply(sys, usr, n)
			w.Header().Set(backendHeader, f.backend)
			w.Header().Set(modelHeader, "qwen3.6-colibri")
			json.NewEncoder(w).Encode(map[string]any{
				"choices": []any{map[string]any{"message": map[string]string{"content": text}}},
				"usage":   map[string]int{"completion_tokens": len(strings.Fields(text))},
			})
		}
	}))
	t.Cleanup(f.srv.Close)
	return f, &Client{HTTP: f.srv.Client(), Base: f.srv.URL, Patience: 5 * time.Second}
}

func TestChatSendsTheTeacherRoleAndRecordsProvenance(t *testing.T) {
	f, cl := newFake(t, func(string, string, int) string { return "<think>hmm</think> hello " })
	rep, err := cl.Chat(context.Background(), Request{User: "hi", MaxTokens: 8})
	if err != nil {
		t.Fatal(err)
	}
	if f.role.Load() != "teacher" || rep.Text != "hello" {
		t.Fatalf("role=%v text=%q", f.role.Load(), rep.Text)
	}
	p := rep.Prov
	if p.Teacher != "colibri" || p.ModelID != "qwen3.6-colibri" || p.Catalog != "qwen36-35b" || p.PromptHash == "" || p.Version != Version {
		t.Fatalf("%+v", p)
	}
}

// If the harness ever answers from the fallback, nothing may be recorded as the teacher's.
func TestAnswerFromAnotherBackendIsRejected(t *testing.T) {
	f, cl := newFake(t, func(string, string, int) string { return "x" })
	f.backend = "llama.cpp"
	if _, err := cl.Chat(context.Background(), Request{User: "hi"}); err == nil || !strings.Contains(err.Error(), "not from the teacher backend") {
		t.Fatalf("got %v", err)
	}
}

func TestChatWaitsThroughUnavailableAndGivesUpWithAClearError(t *testing.T) {
	f, cl := newFake(t, func(string, string, int) string { return "ok" })
	f.busy.Store(1)
	cl.retryWait = 10 * time.Millisecond
	if rep, err := cl.Chat(context.Background(), Request{User: "hi"}); err != nil || rep.Text != "ok" {
		t.Fatalf("%v %v", rep, err)
	}
	f.busy.Store(1 << 20)
	cl.Patience = 100 * time.Millisecond
	if _, err := cl.Chat(context.Background(), Request{User: "hi"}); !errorsIs(err, ErrUnavailable) {
		t.Fatalf("got %v", err)
	}
}

func TestVoteBuildsADistributionAndCancelsPositionBias(t *testing.T) {
	// The fake always prefers the option whose text is "the evidence states ...": it
	// finds the letter in whatever position the prompt put it.
	_, cl := newFake(t, func(_, user string, _ int) string {
		for _, l := range strings.Split(user, "\n") {
			if strings.Contains(l, "clearly entails") {
				return l[:1]
			}
		}
		return "Z"
	})
	d, err := SuggestFactuality(context.Background(), cl, nil, "the sky is blue", "The sky is blue.", 6)
	if err != nil {
		t.Fatal(err)
	}
	if k, margin := d.Top(); k != Supported || margin != 1 || !d.Agree || d.Valid != 6 {
		t.Fatalf("%v top=%s margin=%v agree=%v", d, k, margin, d.Agree)
	}
}

func TestVoteDropsUnparseableAnswersAndFailsWhenNoneParse(t *testing.T) {
	_, cl := newFake(t, func(string, string, int) string { return "I think it is probably fine." })
	if _, err := SuggestFactuality(context.Background(), cl, nil, "c", "e", 2); err == nil {
		t.Fatal("expected an error when nothing parses")
	}
}

func TestCacheMakesRerunsFreeAndSurvivesRestart(t *testing.T) {
	f, cl := newFake(t, func(string, string, int) string { return "B" })
	path := filepath.Join(t.TempDir(), "cache.jsonl")
	c, err := OpenCache(path)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if _, err := JudgeRelevance(context.Background(), cl, c, "q", "p", 2); err != nil {
			t.Fatal(err)
		}
	}
	if f.calls.Load() != 2 {
		t.Fatalf("second run must be served from the cache: %d calls", f.calls.Load())
	}
	c.Close()
	c2, _ := OpenCache(path)
	defer c2.Close()
	if _, err := JudgeRelevance(context.Background(), cl, c2, "q", "p", 2); err != nil || f.calls.Load() != 2 || c2.Len() != 2 {
		t.Fatalf("a reopened cache must keep its answers: err=%v calls=%d len=%d", err, f.calls.Load(), c2.Len())
	}
}

func TestGenerateSFTFiltersAndRecordsProvenance(t *testing.T) {
	_, cl := newFake(t, func(sys, user string, _ int) string {
		switch {
		case strings.Contains(sys, "write realistic user requests"):
			return "1. Summarize the quarterly report for me\n2. summarize the QUARTERLY report for me\n3. Explain the secret gold claim text here\n- short"
		case strings.Contains(user, "Tiny"):
			return "ok"
		}
		return "A complete helpful answer that is long enough."
	})
	var out bytes.Buffer
	st, err := GenerateSFT(context.Background(), cl, nil, SFTSpec{
		Skill: "summaries", System: "Be brief.", Seeds: []string{"Summarize this memo please", "Tiny seed prompt"},
		ExpandPer: 4, Exclude: []string{"Explain the secret gold claim text here"},
	}, &out)
	if err != nil {
		t.Fatal(err)
	}
	// seeds (2) + expanded (3 valid lines, one a case-duplicate, one excluded) per seed.
	if st.Written != 2 || st.Excluded == 0 || st.Duplicates == 0 || st.TooShort != 1 {
		t.Fatalf("%+v\n%s", st, out.String())
	}
	var rec SFTRecord
	if err := json.Unmarshal(bytes.Split(out.Bytes(), []byte("\n"))[0], &rec); err != nil {
		t.Fatal(err)
	}
	if rec.Schema != "ulora-sft/v1" || !rec.Synthetic || rec.Provenance.Teacher != "colibri" || rec.Skill != "summaries" || rec.ID == "" {
		t.Fatalf("%+v", rec)
	}
}

func TestSoftLabelsHideTheLabelledFieldAndUseOnlyVerifiedEpisodes(t *testing.T) {
	var prompts []string
	_, cl := newFake(t, func(_, user string, _ int) string {
		prompts = append(prompts, user)
		return "A"
	})
	episodes := strings.Join([]string{
		`{"episode_id":"e1","task":{"request":"fix the build","intent":"review","risk":"low"},"state_before":{"phase":"develop"},"evidence":[{"class":"diff"}],"trace":{"agent":"codex"},"decision":{"recommended_action":"go","next_phase":"verify"},"outcome":{"verified":true}}`,
		`{"episode_id":"e2","task":{"request":"add a feature","intent":"implement"},"state_before":{"phase":"plan"},"trace":{"agent":"claude"},"decision":{"recommended_action":"go","next_phase":"develop"},"outcome":{"verified":true}}`,
		`{"episode_id":"e3","task":{"request":"x","intent":"review"},"decision":{"recommended_action":"go"},"outcome":{"verified":false}}`,
		`not json`,
	}, "\n")
	var out bytes.Buffer
	st, err := SoftLabels(context.Background(), cl, nil, strings.NewReader(episodes), SoftHeads, 2, &out)
	if err != nil {
		t.Fatal(err)
	}
	if st.Read != 4 || st.Used != 2 || st.Skipped != 2 || st.Heads["intent"] != 2 {
		t.Fatalf("%+v", st)
	}
	var line SoftLine
	json.Unmarshal(bytes.Split(out.Bytes(), []byte("\n"))[0], &line)
	if line.EpisodeID != "e1" || len(line.Labels["intent"]) == 0 || line.Provenance.Teacher != "colibri" {
		t.Fatalf("%+v", line)
	}
	for _, p := range prompts {
		if strings.Contains(p, `"intent"`) && strings.Contains(p, "Intent: ") {
			t.Fatalf("the intent head must not be shown the intent:\n%s", p)
		}
	}
}

func errorsIs(err, target error) bool { return errors.Is(err, target) }
