// Command measure is the baseline-measurement harness for three_layer_inference.md.
//
// It produces numbers, not features: every probe writes one JSON result under
// <dir>/results/ and `measure report` turns them into a go/no-go table. A probe that
// cannot run (llama down, no gold labels yet) records status "blocked" with the reason,
// so the report states what is missing rather than omitting the item.
//
// Plan item -> command:
//
//	(a) llama latency/cache/grammar/tool envelope   llama, llama-sweep
//	(a) HEART-on-Qwen quality gate                  heartbench
//	(b) factuality gold set (owner labels, D4)      gold add|label2|import|export|stats
//	(c) lexical-key retrieval recall                retrieval
//	(d) Qwen judge accuracy / false-support         judge
//	(e) embedder bake-off, Cloudflare cap           embed, cloudflare
//	(f) LoRA capability probe                       lora
//	(g) authoritative Ollama instance               ollama
//	    host snapshot                               host
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	statusMeasured = "measured"
	statusBlocked  = "blocked"
	statusFailed   = "failed"
)

// Result is the unit every probe writes. Data holds the numbers; Checks are the
// pass/fail statements `report` rolls up (thresholds marked placeholder are the
// plan's initial values, to be refit from these very measurements).
type Result struct {
	Probe   string         `json:"probe"`
	At      string         `json:"at"`
	Status  string         `json:"status"`
	Reason  string         `json:"reason,omitempty"`
	Data    map[string]any `json:"data,omitempty"`
	Checks  []Check        `json:"checks,omitempty"`
	Notes   []string       `json:"notes,omitempty"`
	Elapsed float64        `json:"elapsed_s"`
}

// Check is one go/no-go statement.
type Check struct {
	Name string `json:"name"`
	Pass bool   `json:"pass"`
	Info string `json:"info,omitempty"`
}

func newResult(probe string) *Result {
	return &Result{Probe: probe, At: time.Now().UTC().Format(time.RFC3339), Status: statusMeasured, Data: map[string]any{}}
}

func (r *Result) blocked(format string, a ...any) *Result {
	r.Status, r.Reason = statusBlocked, fmt.Sprintf(format, a...)
	return r
}

func (r *Result) failed(format string, a ...any) *Result {
	r.Status, r.Reason = statusFailed, fmt.Sprintf(format, a...)
	return r
}

func (r *Result) check(name string, pass bool, info string) {
	r.Checks = append(r.Checks, Check{Name: name, Pass: pass, Info: info})
}

func (r *Result) note(format string, a ...any) { r.Notes = append(r.Notes, fmt.Sprintf(format, a...)) }

// write stores the result and prints a short summary.
func (r *Result) write(dir string) error {
	rd := filepath.Join(dir, "results")
	if err := os.MkdirAll(rd, 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	p := filepath.Join(rd, r.Probe+".json")
	if err := os.WriteFile(p, append(b, '\n'), 0o644); err != nil {
		return err
	}
	fmt.Printf("%-14s %-9s %s\n", r.Probe, r.Status, r.Reason)
	for _, c := range r.Checks {
		fmt.Printf("  [%s] %s %s\n", map[bool]string{true: "pass", false: "FAIL"}[c.Pass], c.Name, c.Info)
	}
	for _, n := range r.Notes {
		fmt.Printf("  note: %s\n", n)
	}
	fmt.Printf("  -> %s\n", p)
	return nil
}

func usage() {
	fmt.Fprint(os.Stderr, `usage: measure <command> [flags]

  host                          host snapshot (cores, RAM, load, llama/ollama process state, config)
  ollama                        find the authoritative Ollama instance, bind address, models, embed speed   (g)
  cloudflare                    probe the Cloudflare embeddings endpoint and its limit behaviour           (e)
  embed                         embedder bake-off on the gold set: Cloudflare vs Ollama, prefix schemes    (e)
  llama                         llama latency, prefix cache, grammar, tool envelope, parallel slots        (a)
  llama-sweep                   spawn llama-server per ctx/threads/parallel setting and measure            (a)
  lora                          LoRA hot-swap capability probe                                             (f)
  heartbench                    HEART-on-Qwen quality gate over decision cases                             (a)
  gold add|label2|import|export|stats|accept   factuality gold set (SUPPORTED/REFUTED/NEI)                 (b)
  suggest                       Colibri suggests labels for candidate cases (suggestions.jsonl; the owner decides)  (b)
  retrieval-judge               Colibri grades retrieved passages to find relevant ones the gold set lacks         (c)
  retrieval                     lexical retrieval recall on the gold set                                   (c)
  judge                         Qwen judge accuracy, false-support rate, latency                           (d)
  report                        roll all results up into <dir>/REPORT.md

Every command accepts -dir (default "measure"; gold set and results live there).
`)
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	cmd, args := os.Args[1], os.Args[2:]
	var err error
	switch cmd {
	case "host":
		err = cmdHost(args)
	case "ollama":
		err = cmdOllama(args)
	case "cloudflare":
		err = cmdCloudflare(args)
	case "embed":
		err = cmdEmbed(args)
	case "llama":
		err = cmdLlama(args)
	case "llama-sweep":
		err = cmdLlamaSweep(args)
	case "lora":
		err = cmdLora(args)
	case "heartbench":
		err = cmdHeartBench(args)
	case "gold":
		err = cmdGold(args)
	case "suggest":
		err = cmdSuggest(args)
	case "retrieval-judge":
		err = cmdRetrievalJudge(args)
	case "retrieval":
		err = cmdRetrieval(args)
	case "judge":
		err = cmdJudge(args)
	case "report":
		err = cmdReport(args)
	case "-h", "--help", "help":
		usage()
	default:
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "measure:", err)
		os.Exit(1)
	}
}

// ---- shared helpers -------------------------------------------------------

// httpClient returns a client that dials a unix socket when sock is set.
func httpClient(sock string, timeout time.Duration) *http.Client {
	tr := &http.Transport{DisableKeepAlives: false}
	if sock != "" {
		tr.DialContext = func(ctx context.Context, _, _ string) (net.Conn, error) {
			var d net.Dialer
			return d.DialContext(ctx, "unix", sock)
		}
	}
	return &http.Client{Transport: tr, Timeout: timeout}
}

func percentile(v []float64, p float64) float64 {
	if len(v) == 0 {
		return math.NaN()
	}
	s := append([]float64(nil), v...)
	sort.Float64s(s)
	idx := int(math.Ceil(p*float64(len(s)))) - 1
	if idx < 0 {
		idx = 0
	}
	return s[idx]
}

func mean(v []float64) float64 {
	if len(v) == 0 {
		return math.NaN()
	}
	var s float64
	for _, x := range v {
		s += x
	}
	return s / float64(len(v))
}

// r3 rounds for readable JSON; NaN becomes nil-safe 0 with the caller noting emptiness.
func r3(x float64) float64 {
	if math.IsNaN(x) || math.IsInf(x, 0) {
		return 0
	}
	return math.Round(x*1000) / 1000
}

func ms(d time.Duration) float64 { return r3(float64(d.Microseconds()) / 1000) }

func splitList(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func readJSONL[T any](path string) ([]T, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var out []T
	for i, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var v T
		if err := json.Unmarshal([]byte(line), &v); err != nil {
			return nil, fmt.Errorf("%s line %d: %w", path, i+1, err)
		}
		out = append(out, v)
	}
	return out, nil
}

func appendJSONL(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	_, err = f.Write(append(b, '\n'))
	return err
}
