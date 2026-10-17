package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSplitIsDeterministicAndGrouped(t *testing.T) {
	if splitOf("g1") != splitOf("g1") {
		t.Fatal("split not deterministic")
	}
	dev := 0
	for i := 0; i < 1000; i++ {
		if splitOf(string(rune('a'+i%26))+strings.Repeat("x", i)) == "dev" {
			dev++
		}
	}
	if dev < 300 || dev > 500 {
		t.Fatalf("dev share %d/1000, want ~40%%", dev)
	}
}

func TestGoldNormalizeRejectsBadInput(t *testing.T) {
	good := GoldCase{Claim: "c", Evidence: "e", Label: "supported", Labeler: "me"}
	if err := good.normalize(); err != nil {
		t.Fatal(err)
	}
	if good.Label != LabelSupported || good.DataClass != "private" || good.ID == "" || good.Group == "" {
		t.Fatalf("defaults not applied: %+v", good)
	}
	for _, bad := range []GoldCase{
		{Claim: "", Evidence: "e", Label: "NEI", Labeler: "me"},
		{Claim: "c", Evidence: "e", Label: "MAYBE", Labeler: "me"},
		{Claim: "c", Evidence: "e", Label: "NEI"},
		{Claim: "c", Evidence: "e", Label: "NEI", Labeler: "me", DataClass: "secret"},
	} {
		if bad.normalize() == nil {
			t.Fatalf("accepted %+v", bad)
		}
	}
}

func TestSamePassageSharesGroupAndSplit(t *testing.T) {
	a := GoldCase{Claim: "c1", Evidence: "same passage", Label: "NEI", Labeler: "x"}
	b := GoldCase{Claim: "c2", Evidence: "same passage", Label: "NEI", Labeler: "x"}
	a.normalize()
	b.normalize()
	if a.Group != b.Group || splitOf(a.Group) != splitOf(b.Group) {
		t.Fatal("cases sharing evidence must share a split")
	}
}

func TestCohenKappa(t *testing.T) {
	cs := []GoldCase{
		{Label: "SUPPORTED", Label2: "SUPPORTED"}, {Label: "REFUTED", Label2: "REFUTED"},
		{Label: "NEI", Label2: "NEI"}, {Label: "SUPPORTED", Label2: "REFUTED"}, {Label: "NEI"},
	}
	k, n := cohenKappa(cs)
	if n != 4 || k <= 0 || k >= 1 {
		t.Fatalf("kappa %v n %d", k, n)
	}
	k, _ = cohenKappa([]GoldCase{{Label: "SUPPORTED", Label2: "SUPPORTED"}, {Label: "REFUTED", Label2: "REFUTED"}})
	if k != 1 {
		t.Fatalf("perfect agreement kappa %v", k)
	}
}

func TestBM25RecallFindsOwnPassage(t *testing.T) {
	cs := []GoldCase{
		{Claim: "The Eiffel Tower is in Paris", Evidence: "The Eiffel Tower stands in Paris, France.", Label: LabelSupported},
		{Claim: "Mount Everest is 8849 metres tall", Evidence: "Mount Everest rises 8849 metres above sea level.", Label: LabelSupported},
		{Claim: "Water boils at 100 C", Evidence: "At sea level water boils at 100 degrees Celsius.", Label: LabelRefuted},
		{Claim: "unrelated", Evidence: "Nothing about it.", Label: LabelNEI},
	}
	pool, qs, gold := retrievalPool(cs, []string{"A passage about sourdough bread.", "Another about tax law."})
	if len(qs) != 3 {
		t.Fatalf("NEI must not be a query: %d", len(qs))
	}
	idx := newBM25(pool)
	ranked := make([][]int, len(qs))
	for i, q := range qs {
		ranked[i] = idx.rank(q)
	}
	rec, mrr := recallAt(ranked, gold, []int{1, 5})
	if rec[1] != 1 || mrr != 1 {
		t.Fatalf("recall %v mrr %v", rec, mrr)
	}
}

func TestParseProcNetTCPListenOnly(t *testing.T) {
	in := `  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode
   0: 0100007F:2CAB 00000000:0000 0A 00000000:00000000 00:00000000 00000000     0        0 111 1
   1: 00000000:2CAC 00000000:0000 0A 00000000:00000000 00:00000000 00000000     0        0 222 1
   2: 0100007F:9999 0100007F:2CAB 01 00000000:00000000 00:00000000 00000000     0        0 333 1
`
	ls := parseProcNetTCP(in, false)
	if len(ls) != 2 {
		t.Fatalf("want 2 listeners, got %+v", ls)
	}
	if ls[0].Addr != "127.0.0.1" || ls[0].Port != 11435 || !isLoopback(ls[0].Addr) {
		t.Fatalf("first %+v", ls[0])
	}
	if ls[1].Addr != "0.0.0.0" || isLoopback(ls[1].Addr) {
		t.Fatalf("second %+v", ls[1])
	}
}

func TestValidateDecision(t *testing.T) {
	if validateDecision(`{"severity":"high","action":"block","priority":3}`) != nil {
		t.Fatal("valid rejected")
	}
	for _, bad := range []string{`{"severity":"huge","action":"block","priority":3}`, `{"severity":"low","action":"block","priority":9}`,
		`{"severity":"low","action":"block","priority":2,"extra":1}`, `not json`} {
		if validateDecision(bad) == nil {
			t.Fatalf("accepted %s", bad)
		}
	}
}

// fakeLlama answers the judge by reading the claim: it knows the right answer for
// each claim and reports it using whatever letter the prompt assigned that option.
func fakeLlama(t *testing.T, answers map[string]string, rejectJSONSchema bool) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte(`{"status":"ok"}`)) })
	mux.HandleFunc("/v1/chat/completions", func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		var req struct {
			Messages       []chatMsg      `json:"messages"`
			ResponseFormat map[string]any `json:"response_format"`
		}
		json.Unmarshal(b, &req)
		if rejectJSONSchema && req.ResponseFormat != nil && req.ResponseFormat["type"] == "json_schema" {
			http.Error(w, `{"error":"unsupported"}`, 400)
			return
		}
		user := req.Messages[len(req.Messages)-1].Content
		want := ""
		for claim, label := range answers {
			if strings.Contains(user, "CLAIM: "+claim) {
				want = label
			}
		}
		desc := map[string]string{LabelSupported: "entails the claim", LabelRefuted: "the evidence contradicts the claim", LabelNEI: "not enough information"}
		letter := "A"
		for _, line := range strings.Split(user, "\n") {
			if len(line) > 3 && line[1] == ':' && strings.Contains(line, desc[want]) {
				letter = line[:1]
			}
		}
		json.NewEncoder(w).Encode(map[string]any{
			"choices": []any{map[string]any{"message": map[string]string{"role": "assistant", "content": `{"verdict":"` + letter + `"}`}}},
			"usage":   map[string]int{"prompt_tokens": 50, "completion_tokens": 5},
			"timings": map[string]float64{"predicted_per_second": 7.5},
		})
	})
	return httptest.NewServer(mux)
}

func TestJudgeOrderSwapAndFallbackMode(t *testing.T) {
	srv := fakeLlama(t, map[string]string{"c-sup": LabelSupported, "c-ref": LabelRefuted, "c-nei": LabelNEI}, true)
	defer srv.Close()
	c := &llamaClient{chatHC: srv.Client(), chatBase: srv.URL}
	if err := c.healthy(); err != nil {
		t.Fatal(err)
	}
	for claim, want := range map[string]string{"c-sup": LabelSupported, "c-ref": LabelRefuted, "c-nei": LabelNEI} {
		got, dis, _, _, err := judgeOne(context.Background(), c, GoldCase{Claim: claim, Evidence: "ev"})
		if err != nil || got != want || dis {
			t.Fatalf("%s: got %q disagree=%v err=%v", claim, got, dis, err)
		}
	}
	if c.mode != "json_object" {
		t.Fatalf("expected fallback past unsupported json_schema, mode=%q", c.mode)
	}
}

func TestOrderSwapDisagreementBecomesNEI(t *testing.T) {
	// A judge that always answers "A" picks SUPPORTED then NEI under the swapped order.
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/chat/completions", func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"{\"verdict\":\"A\"}"}}]}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	c := &llamaClient{chatHC: srv.Client(), chatBase: srv.URL}
	got, dis, _, _, err := judgeOne(context.Background(), c, GoldCase{Claim: "x", Evidence: "y"})
	if err != nil || got != LabelNEI || !dis {
		t.Fatalf("got %q dis=%v err=%v", got, dis, err)
	}
}

func TestScoreJudgeFalseSupport(t *testing.T) {
	conf := map[string]map[string]int{
		LabelSupported: {LabelSupported: 9, LabelNEI: 1},
		LabelRefuted:   {LabelRefuted: 8, LabelSupported: 2},
		LabelNEI:       {LabelNEI: 9, LabelSupported: 1},
	}
	r := newResult("judge")
	scoreJudge(r, conf, []float64{100, 200, 300}, 2, 0)
	if r.Data["false_support_rate"].(float64) != r3(3.0/20.0) {
		t.Fatalf("fsr %v", r.Data["false_support_rate"])
	}
	if r.Checks[0].Pass {
		t.Fatal("15% false-support must fail the 2% gate")
	}
}

func TestGoldReadyBlockedWhenEmpty(t *testing.T) {
	_, blocked := goldReady(t.TempDir(), "judge", "test")
	if blocked == nil || blocked.Status != statusBlocked {
		t.Fatalf("expected blocked, got %+v", blocked)
	}
}

func TestCosineAndSchemes(t *testing.T) {
	if c := cosine([]float32{1, 0}, []float32{1, 0}); c < 0.999 {
		t.Fatal(c)
	}
	if cosine([]float32{1, 0}, []float32{0, 1}) != 0 {
		t.Fatal("orthogonal")
	}
	if embedSchemes["nomic"].Query != "search_query: " {
		t.Fatal("nomic prefix")
	}
}

func TestRedactArgs(t *testing.T) {
	got := redactArgs("llama-server --port 8000 --api-key SECRET --threads 4")
	if strings.Contains(got, "SECRET") || !strings.Contains(got, "--threads 4") {
		t.Fatal(got)
	}
}
