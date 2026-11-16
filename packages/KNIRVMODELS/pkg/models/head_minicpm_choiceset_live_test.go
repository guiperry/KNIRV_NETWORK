package models

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"knirvmodels/pkg/vlm"

	sdk "github.com/guiperry/knirv-sdk-go/models"
)

// This file is a test-only 4.3/4.4 smoke probe for the runtime HEAD path. The 4.3 set
// scorer itself is the native Rust set head (heart/decision; pipeline/5_HEAD_TRAINER)
// and never calls MiniCPM. What does call MiniCPM at runtime is HEAD's VLM/text surface
// (/models/head/*) through pkg/vlm's minicpm backend. This probe therefore drives that
// real surface: it asks HEAD to select one candidate string out of a choice set, on
// candidate strings that never appear in the worked example or in any other question
// ("unseen" strings), with and without a demonstrated example. It is a small smoke
// measurement (n per condition is single digits), not the plan's 600+600 gate run.
//
// It runs only when KNIRV_VLM_MINICPM_LIVE_ENDPOINT names a live llama-server, exactly
// like TestHEADMiniCPMLiveRouting; otherwise it skips with the reason.

type choicesetCase struct {
	prompt  string
	choices []string
	correct int
	task    string
}

type choicesetResult struct {
	task       string
	condition  string
	correct    bool
	got        int
	latency    time.Duration
	reply      string
	httpStatus int
}

// choicesetTaskA is a routing-style task: a state maps four keys to four values, and the
// question names one key. Every value string is unique across every case.
func choicesetTaskA() []choicesetCase {
	return []choicesetCase{
		{task: "A", prompt: "State:\n- zone red passphrase is ORBIT-7\n- zone blue passphrase is LATTICE-3\n- zone green passphrase is CINDER-9\n- zone amber passphrase is VECTOR-2\n\nQuestion: Which passphrase belongs to zone blue?\n",
			choices: []string{"ORBIT-7", "LATTICE-3", "CINDER-9", "VECTOR-2"}, correct: 1},
		{task: "A", prompt: "State:\n- north gate code is QUARTZ-8\n- south gate code is MAGNET-5\n- east gate code is PILOT-1\n- west gate code is HARBOR-6\n\nQuestion: Which code belongs to the west gate?\n",
			choices: []string{"HARBOR-6", "MAGNET-5", "QUARTZ-8", "PILOT-1"}, correct: 0},
		{task: "A", prompt: "State:\n- alfa cohort label is GLACIER-4\n- bravo cohort label is TUNDRA-9\n- charlie cohort label is SAHARA-2\n- delta cohort label is MEADOW-7\n\nQuestion: Which label belongs to the charlie cohort?\n",
			choices: []string{"GLACIER-4", "TUNDRA-9", "SAHARA-2", "MEADOW-7"}, correct: 2},
		{task: "A", prompt: "State:\n- spring bird is KESTREL-3\n- summer bird is OSPREY-8\n- autumn bird is MERLIN-5\n- winter bird is HARRIER-1\n\nQuestion: Which bird belongs to winter?\n",
			choices: []string{"KESTREL-3", "OSPREY-8", "MERLIN-5", "HARRIER-1"}, correct: 3},
		{task: "A", prompt: "State:\n- copper cat is FALCON-6\n- silver cat is JAGUAR-4\n- gold cat is PUMA-9\n- iron cat is LYNX-2\n\nQuestion: Which cat belongs to silver?\n",
			choices: []string{"FALCON-6", "PUMA-9", "JAGUAR-4", "LYNX-2"}, correct: 2},
		{task: "A", prompt: "State:\n- apple tree is BIRCH-7\n- banana tree is CEDAR-3\n- cherry tree is MAPLE-8\n- date tree is ASPEN-5\n\nQuestion: Which tree belongs to date?\n",
			choices: []string{"ASPEN-5", "BIRCH-7", "CEDAR-3", "MAPLE-8"}, correct: 0},
		{task: "A", prompt: "State:\n- mercury station is COMET-2\n- venus station is NEBULA-6\n- mars station is PULSAR-4\n- jupiter station is QUASAR-9\n\nQuestion: Which station belongs to venus?\n",
			choices: []string{"COMET-2", "NEBULA-6", "PULSAR-4", "QUASAR-9"}, correct: 1},
		{task: "A", prompt: "State:\n- paper sample is AMBER-5\n- glass sample is SLATE-1\n- stone sample is GRANITE-7\n- steel sample is BASALT-3\n\nQuestion: Which sample belongs to glass?\n",
			choices: []string{"AMBER-5", "GRANITE-7", "SLATE-1", "BASALT-3"}, correct: 2},
	}
}

// choicesetTaskB is an evidence-reranker-style task: pick the one span that supports the
// claim. Every span string is unique across every case.
func choicesetTaskB() []choicesetCase {
	return []choicesetCase{
		{task: "B", prompt: "Claim: The reactor output rose in June.\n\nSpans:\n1. Output in May was 40 MW.\n2. Output in June reached 55 MW.\n3. The turbine was serviced in July.\n4. Output in August was 52 MW.\n\nQuestion: Which span supports the claim?\n",
			choices: []string{"Output in May was 40 MW.", "Output in June reached 55 MW.", "The turbine was serviced in July.", "Output in August was 52 MW."}, correct: 1},
		{task: "B", prompt: "Claim: The bridge opened after 1985.\n\nSpans:\n1. The bridge opened in 1982.\n2. The bridge was repainted in 2001.\n3. Construction began in 1979.\n4. The bridge opened to traffic in 1988.\n\nQuestion: Which span supports the claim?\n",
			choices: []string{"The bridge opened in 1982.", "The bridge was repainted in 2001.", "Construction began in 1979.", "The bridge opened to traffic in 1988."}, correct: 3},
		{task: "B", prompt: "Claim: The candidate won the election.\n\nSpans:\n1. The candidate lost by 300 votes.\n2. The candidate conceded on Tuesday.\n3. The candidate won 54 percent of the vote.\n4. A recount was requested.\n\nQuestion: Which span supports the claim?\n",
			choices: []string{"The candidate lost by 300 votes.", "The candidate conceded on Tuesday.", "The candidate won 54 percent of the vote.", "A recount was requested."}, correct: 2},
		{task: "B", prompt: "Claim: The drug lowers blood pressure.\n\nSpans:\n1. The trial measured blood pressure for 12 weeks.\n2. The drug reduced systolic pressure by 9 mmHg.\n3. Side effects were mild.\n4. The placebo group was unchanged.\n\nQuestion: Which span supports the claim?\n",
			choices: []string{"The trial measured blood pressure for 12 weeks.", "The drug reduced systolic pressure by 9 mmHg.", "Side effects were mild.", "The placebo group was unchanged."}, correct: 1},
		{task: "B", prompt: "Claim: The comet passes Earth in 2061.\n\nSpans:\n1. The comet passed Earth in 1986.\n2. Its orbit period is 76 years.\n3. The comet will return in 2061.\n4. Amateur astronomers observed it.\n\nQuestion: Which span supports the claim?\n",
			choices: []string{"The comet passed Earth in 1986.", "Its orbit period is 76 years.", "The comet will return in 2061.", "Amateur astronomers observed it."}, correct: 2},
		{task: "B", prompt: "Claim: The library is open on Sunday.\n\nSpans:\n1. The library is closed on Sunday.\n2. Sunday hours are 10 to 4.\n3. The library opens at 9 on weekdays.\n4. The library has 40,000 books.\n\nQuestion: Which span supports the claim?\n",
			choices: []string{"The library is closed on Sunday.", "Sunday hours are 10 to 4.", "The library opens at 9 on weekdays.", "The library has 40,000 books."}, correct: 1},
	}
}

func letters(n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = string(rune('A' + i))
	}
	return out
}

func buildChoicesetPrompt(c choicesetCase, withExample bool) string {
	var b strings.Builder
	b.WriteString("You are choosing exactly one candidate from a fixed set. Do not invent candidates.\n\n")
	if withExample {
		b.WriteString("Example:\n")
		b.WriteString("State:\n- zone red passphrase is WIDGET-9\n- zone blue passphrase is GADGET-4\n")
		b.WriteString("Question: Which passphrase belongs to zone blue?\n")
		b.WriteString("Candidates:\nA. GADGET-4\nB. WIDGET-9\n")
		b.WriteString("Answer with exactly one letter. Example answer: A\n\n")
		b.WriteString("Now answer the real question in the same format.\n\n")
	}
	b.WriteString(c.prompt)
	b.WriteString("Candidates:\n")
	ls := letters(len(c.choices))
	for i, ch := range c.choices {
		b.WriteString(ls[i])
		b.WriteString(". ")
		b.WriteString(ch)
		b.WriteString("\n")
	}
	b.WriteString("\nAnswer with exactly one letter (")
	b.WriteString(strings.Join(ls, ", "))
	b.WriteString("). Answer:")
	return b.String()
}

var answerLetterRe = regexp.MustCompile(`(?:^|[\s(])([A-D])(?:[\).,:;\s]|$)`)

// parseChoicesetReply maps a free-form reply to a candidate index, or -1. A unique
// candidate-string match wins (the model may echo the text); otherwise the last
// standalone uppercase letter A-D is used.
func parseChoicesetReply(reply string, choices []string) int {
	found := -1
	ambiguous := false
	for i, ch := range choices {
		if ch != "" && strings.Contains(reply, ch) {
			if found >= 0 && found != i {
				ambiguous = true
			}
			found = i
		}
	}
	if found >= 0 && !ambiguous {
		return found
	}
	ms := answerLetterRe.FindAllStringSubmatch(reply, -1)
	if len(ms) == 0 {
		return -1
	}
	last := ms[len(ms)-1][1][0]
	idx := int(last - 'A')
	if idx < 0 || idx >= len(choices) {
		return -1
	}
	return idx
}

// TestHEADMiniCPMChoiceSetLive measures whether HEAD, backed by the real MiniCPM-V
// llama-server through pkg/vlm, selects the right candidate string from a choice set
// when the candidate strings never appeared in the demonstrated example. It reports
// accuracy per task and per condition (worked example / no example), the base rate
// (1/N), latency, and the request counts that left the backend. It is a smoke probe,
// not the plan's full 600+600 4.4 gate.
func TestHEADMiniCPMChoiceSetLive(t *testing.T) {
	ep := strings.TrimSpace(os.Getenv("KNIRV_VLM_MINICPM_LIVE_ENDPOINT"))
	if ep == "" {
		t.Skip("set KNIRV_VLM_MINICPM_LIVE_ENDPOINT (e.g. http://127.0.0.1:8093) to run the live MiniCPM choice-set probe")
	}
	target, err := url.Parse(ep)
	if err != nil {
		t.Fatalf("bad KNIRV_VLM_MINICPM_LIVE_ENDPOINT: %v", err)
	}

	var mu sync.Mutex
	chatCalls := 0
	rp := httputil.NewSingleHostReverseProxy(target)
	rp.Transport = &http.Transport{ResponseHeaderTimeout: 10 * time.Minute}
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/chat/completions" {
			mu.Lock()
			chatCalls++
			mu.Unlock()
		}
		rp.ServeHTTP(w, r)
	}))
	defer proxy.Close()

	backend := vlm.NewMiniCPMBackend(proxy.URL)
	backend.MaxTokens = 32
	head := NewHead(backend)
	mux := http.NewServeMux()
	head.Mount(mux, "/models/head")
	srv := httptest.NewServer(mux)
	defer srv.Close()

	run := func(c choicesetCase, withExample bool) choicesetResult {
		condition := "no-example"
		if withExample {
			condition = "example"
		}
		body, _ := json.Marshal(sdk.HeadGenerateRequest{Prompt: buildChoicesetPrompt(c, withExample)})
		start := time.Now()
		res, err := http.Post(srv.URL+"/models/head/generate", "application/json", bytes.NewReader(body))
		lat := time.Since(start)
		if err != nil {
			t.Errorf("%s/%s: %v", c.task, condition, err)
			return choicesetResult{task: c.task, condition: condition, got: -1, latency: lat}
		}
		raw, _ := io.ReadAll(res.Body)
		res.Body.Close()
		out := choicesetResult{task: c.task, condition: condition, got: -1, latency: lat, httpStatus: res.StatusCode}
		if res.StatusCode != http.StatusOK {
			t.Errorf("%s/%s: HTTP %d: %s", c.task, condition, res.StatusCode, raw)
			return out
		}
		var resp sdk.HeadGenerateResponse
		if err := json.Unmarshal(raw, &resp); err != nil {
			t.Errorf("%s/%s: bad response %s: %v", c.task, condition, raw, err)
			return out
		}
		out.reply = resp.Text
		out.got = parseChoicesetReply(resp.Text, c.choices)
		out.correct = out.got == c.correct
		return out
	}

	var results []choicesetResult
	for _, withExample := range []bool{false, true} {
		for _, c := range choicesetTaskA() {
			results = append(results, run(c, withExample))
		}
		for _, c := range choicesetTaskB() {
			results = append(results, run(c, withExample))
		}
	}

	type bucket struct {
		n, correct int
		totalLat   time.Duration
	}
	buckets := map[string]*bucket{}
	for _, r := range results {
		key := r.task + "/" + r.condition
		b := buckets[key]
		if b == nil {
			b = &bucket{}
			buckets[key] = b
		}
		b.n++
		if r.correct {
			b.correct++
		}
		b.totalLat += r.latency
		t.Logf("%s/%s: correct=%v got=%d reply=%q latency=%s http=%d",
			r.task, r.condition, r.correct, r.got, strings.TrimSpace(r.reply), r.latency.Round(time.Millisecond), r.httpStatus)
	}
	for _, key := range []string{"A/no-example", "A/example", "B/no-example", "B/example"} {
		b := buckets[key]
		if b == nil || b.n == 0 {
			continue
		}
		t.Logf("RESULT %s: %d/%d = %.3f (base rate %.3f), mean latency %s",
			key, b.correct, b.n, float64(b.correct)/float64(b.n), 0.25, (b.totalLat / time.Duration(b.n)).Round(time.Millisecond))
	}
	mu.Lock()
	calls := chatCalls
	mu.Unlock()
	if calls != len(results) {
		t.Fatalf("backend saw %d chat calls for %d probe questions", calls, len(results))
	}
	t.Logf("routing proof: %d /v1/chat/completions requests through pkg/vlm minicpm to the live llama-server", calls)
}
