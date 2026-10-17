package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// llamaClient talks to llama over the wrapper's unix socket (chat + adapters only)
// and/or the raw llama-server address (also /props, /slots, /lora-adapters).
type llamaClient struct {
	chatHC, rawHC *http.Client
	chatBase      string // base URL for /v1/chat/completions
	rawBase       string // base URL for raw llama-server routes; "" when unknown
	key           string // bearer token for the raw upstream
	mode          string // working constrained-output mode, discovered lazily
}

type llamaFlags struct {
	sock, raw, key *string
	timeout        *time.Duration
}

func addLlamaFlags(fs *flag.FlagSet) llamaFlags {
	return llamaFlags{
		sock:    fs.String("llama-sock", "", "wrapper unix socket (e.g. /var/lib/knirvserver/sockets/llama.sock): chat and adapter routes only"),
		raw:     fs.String("llama-raw", "", "raw llama-server host:port (private upstream): adds /props, /slots, /lora-adapters"),
		key:     fs.String("llama-key", os.Getenv("MEASURE_LLAMA_KEY"), "API key for -llama-raw (default: $MEASURE_LLAMA_KEY)"),
		timeout: fs.Duration("timeout", 10*time.Minute, "per-request timeout (CPU-only Qwen is slow)"),
	}
}

func (f llamaFlags) client() (*llamaClient, error) {
	if *f.sock == "" && *f.raw == "" {
		return nil, fmt.Errorf("give -llama-sock and/or -llama-raw")
	}
	c := &llamaClient{}
	if *f.key != "" {
		c.key = *f.key
	}
	if *f.sock != "" {
		c.chatHC = httpClient(*f.sock, *f.timeout)
		c.chatBase = "http://llama"
	}
	if *f.raw != "" {
		c.rawHC = &http.Client{Timeout: *f.timeout}
		c.rawBase = "http://" + *f.raw
		if c.key != "" {
			c.rawHC.Transport = bearerTransport{c.key, http.DefaultTransport}
		}
		if c.chatHC == nil {
			c.chatHC, c.chatBase = c.rawHC, c.rawBase
		}
	}
	return c, nil
}

func (c *llamaClient) healthy() error {
	resp, err := c.chatHC.Get(c.chatBase + "/health")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("llama health: HTTP %d", resp.StatusCode)
	}
	return nil
}

type chatMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatOut struct {
	Content          string
	PromptTokens     int
	CompletionTokens int
	Timings          map[string]float64
	Wall             time.Duration
}

// chat sends one completion. extra is merged into the body (response_format, cache_prompt, ...).
func (c *llamaClient) chat(ctx context.Context, msgs []chatMsg, maxTokens int, extra map[string]any) (*chatOut, int, error) {
	body := map[string]any{"messages": msgs, "max_tokens": maxTokens, "temperature": 0, "stream": false}
	for k, v := range extra {
		body[k] = v
	}
	b, _ := json.Marshal(body)
	req, _ := http.NewRequestWithContext(ctx, "POST", c.chatBase+"/v1/chat/completions", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	t0 := time.Now()
	resp, err := c.chatHC.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, resp.StatusCode, fmt.Errorf("HTTP %d: %.200s", resp.StatusCode, raw)
	}
	var o struct {
		Choices []struct {
			Message chatMsg `json:"message"`
		} `json:"choices"`
		Usage struct {
			Prompt     int `json:"prompt_tokens"`
			Completion int `json:"completion_tokens"`
		} `json:"usage"`
		Timings map[string]float64 `json:"timings"`
	}
	if err := json.Unmarshal(raw, &o); err != nil {
		return nil, 200, fmt.Errorf("decode: %w", err)
	}
	out := &chatOut{PromptTokens: o.Usage.Prompt, CompletionTokens: o.Usage.Completion, Timings: o.Timings, Wall: time.Since(t0)}
	if len(o.Choices) > 0 {
		out.Content = o.Choices[0].Message.Content
	}
	return out, 200, nil
}

// constrained asks for output matching schema. llama-server builds differ in which
// form they accept, so the first one that works is remembered and reported.
func (c *llamaClient) constrained(ctx context.Context, msgs []chatMsg, schema map[string]any, maxTokens int) (*chatOut, error) {
	forms := map[string]map[string]any{
		"json_schema": {"response_format": map[string]any{"type": "json_schema", "json_schema": map[string]any{"name": "out", "strict": true, "schema": schema}}},
		"json_object": {"response_format": map[string]any{"type": "json_object", "schema": schema}},
		"grammar":     {"json_schema": schema},
	}
	order := []string{"json_schema", "json_object", "grammar"}
	if c.mode != "" {
		order = []string{c.mode}
	}
	var last error
	for _, m := range order {
		extra := map[string]any{"cache_prompt": true}
		for k, v := range forms[m] {
			extra[k] = v
		}
		out, code, err := c.chat(ctx, msgs, maxTokens, extra)
		if err == nil {
			c.mode = m
			return out, nil
		}
		last = err
		if code != 400 && code != 422 { // not a "this form is unsupported" answer
			return nil, err
		}
	}
	return nil, fmt.Errorf("no constrained-output form accepted: %w", last)
}

func (c *llamaClient) rawGet(path string, out any) (int, error) {
	if c.rawBase == "" {
		return 0, fmt.Errorf("needs -llama-raw")
	}
	resp, err := c.rawHC.Get(c.rawBase + path)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == 200 && out != nil {
		if err := json.Unmarshal(b, out); err != nil {
			return 200, err
		}
	}
	return resp.StatusCode, nil
}

// filler returns about n tokens of varied text (roughly 0.75 words per token).
func filler(n int, salt string) string {
	words := []string{"ledger", "node", "policy", "witness", "bucket", "skill", "audit", "guardrail", "inquiry", "template",
		"severity", "resolution", "verifier", "compile", "adapter", "prefix", "cache", "schema", "vector", "claim"}
	var sb strings.Builder
	sb.WriteString(salt + " ")
	for i := 0; sb.Len() < n*4; i++ {
		sb.WriteString(words[(i*7+len(salt))%len(words)])
		sb.WriteByte(' ')
		if i%12 == 11 {
			sb.WriteString("and then ")
		}
	}
	return sb.String()
}

func timing(t map[string]float64, k string) float64 { return r3(t[k]) }

// ---- llama probe ----------------------------------------------------------

func cmdLlama(args []string) error {
	fs := flag.NewFlagSet("llama", flag.ExitOnError)
	dir := fs.String("dir", "measure", "working directory")
	lf := addLlamaFlags(fs)
	trials := fs.Int("trials", 10, "constrained-output trials per check")
	sizes := fs.String("sizes", "128,512,1024,1800", "prompt sizes (tokens) for the prefill sweep; keep under ctx_size")
	fs.Parse(args)

	r := newResult("llama")
	c, err := lf.client()
	if err != nil {
		return err
	}
	if err := c.healthy(); err != nil {
		return r.blocked("llama is not reachable: %v (see `measure host`; knirvllama may be a zombie)", err).write(*dir)
	}
	ctx := context.Background()

	// Server properties: context size, slots, build.
	var props struct {
		Slots int    `json:"total_slots"`
		Build string `json:"build_info"`
		Gen   struct {
			NCtx int `json:"n_ctx"`
		} `json:"default_generation_settings"`
	}
	if code, err := c.rawGet("/props", &props); err == nil && code == 200 {
		r.Data["props"] = map[string]any{"n_ctx": props.Gen.NCtx, "total_slots": props.Slots, "build": props.Build}
	} else {
		r.note("/props unavailable (%v, HTTP %d): pass -llama-raw for n_ctx, slots and build", err, code)
	}
	var slots []map[string]any
	if code, _ := c.rawGet("/slots", &slots); code == 200 {
		r.Data["slots"] = len(slots)
	}
	for _, p := range findProcs("llama-server", "llama_server") {
		if p.State != "Z" {
			r.Data["server_rss_mb"] = p.RSSMB
			r.Data["server_args"] = p.Args
		}
	}

	// Prefill speed against prompt size.
	var rows []map[string]any
	for _, s := range splitList(*sizes) {
		n, _ := strconv.Atoi(s)
		out, _, err := c.chat(ctx, []chatMsg{{"user", filler(n, "size"+s) + "\nSay OK."}}, 1, map[string]any{"cache_prompt": false})
		if err != nil {
			rows = append(rows, map[string]any{"target_tokens": n, "error": err.Error()})
			continue
		}
		rows = append(rows, map[string]any{"target_tokens": n, "prompt_tokens": out.PromptTokens, "wall_ms": ms(out.Wall),
			"prompt_tok_per_s": timing(out.Timings, "prompt_per_second")})
	}
	r.Data["prefill"] = rows

	// Prefix cache: same long prefix, different tail.
	prefix := filler(900, "frame")
	var cacheWall [2]float64
	var cacheTim [2]map[string]float64
	cacheOK := true
	for i := 0; i < 2; i++ {
		out, _, err := c.chat(ctx, []chatMsg{{"system", prefix}, {"user", fmt.Sprintf("Task %d: reply OK.", i)}}, 1, map[string]any{"cache_prompt": true})
		if err != nil {
			cacheOK = false
			r.note("cache test: %v", err)
			break
		}
		cacheWall[i], cacheTim[i] = ms(out.Wall), out.Timings
	}
	if cacheOK {
		speedup := safeDiv(cacheWall[0], cacheWall[1])
		r.Data["prefix_cache"] = map[string]any{"cold_ms": cacheWall[0], "warm_ms": cacheWall[1], "speedup": r3(speedup),
			"warm_prompt_n": cacheTim[1]["prompt_n"], "warm_cache_n": cacheTim[1]["cache_n"]}
		r.check("prefix cache makes a repeated 900-token frame at least 2x faster", speedup >= 2, fmt.Sprintf("%.1fx", speedup))
	}

	// Constrained output validity: stage-like schema with an enum and a bounded number.
	schema := map[string]any{"type": "object", "additionalProperties": false, "required": []string{"severity", "action", "priority"},
		"properties": map[string]any{
			"severity": map[string]any{"enum": []string{"low", "medium", "high", "critical"}},
			"action":   map[string]any{"enum": []string{"allow", "request_review", "block"}},
			"priority": map[string]any{"type": "integer", "minimum": 1, "maximum": 5}}}
	valid := 0
	var genTok []float64
	for i := 0; i < *trials; i++ {
		out, err := c.constrained(ctx, []chatMsg{{"user", fmt.Sprintf("Case %d: an agent tried to delete /etc/passwd. Decide.", i)}}, schema, 64)
		if err != nil {
			r.note("constrained trial %d: %v", i, err)
			continue
		}
		genTok = append(genTok, timing(out.Timings, "predicted_per_second"))
		if validateDecision(out.Content) == nil {
			valid++
		}
	}
	r.Data["constrained"] = map[string]any{"mode": c.mode, "valid": valid, "trials": *trials, "gen_tok_per_s": r3(mean(genTok))}
	r.check("grammar/schema-constrained output is accepted by this build", c.mode != "", "mode "+c.mode)
	r.check("constrained output is 100% schema-valid", valid == *trials && *trials > 0, fmt.Sprintf("%d/%d", valid, *trials))

	// Action envelope (tool call or final).
	env := map[string]any{"type": "object", "required": []string{"action"}, "properties": map[string]any{
		"action": map[string]any{"enum": []string{"tool", "final"}},
		"tool":   map[string]any{"enum": []string{"manifest.get", "index.search", "wasm.verify"}},
		"args":   map[string]any{"type": "object"}, "output": map[string]any{"type": "string"}}}
	cases := []struct{ prompt, want string }{
		{"What models are loaded on this node? Use a tool if you need facts.", "tool"},
		{"Look up what the index says about the Eiffel Tower.", "tool"},
		{"Say hello. No tools needed.", "final"},
		{"Verify the generated module before answering.", "tool"},
		{"Reply with the word done.", "final"},
	}
	envValid, envRight := 0, 0
	for _, tc := range cases {
		out, err := c.constrained(ctx, []chatMsg{{"system", `Reply as {"action":"tool","tool":"<name>","args":{}} to call a tool or {"action":"final","output":"..."} to finish. Tools: manifest.get, index.search, wasm.verify.`}, {"user", tc.prompt}}, env, 96)
		if err != nil {
			continue
		}
		var e struct{ Action string }
		if json.Unmarshal([]byte(out.Content), &e) == nil && (e.Action == "tool" || e.Action == "final") {
			envValid++
			if e.Action == tc.want {
				envRight++
			}
		}
	}
	r.Data["action_envelope"] = map[string]any{"valid": envValid, "correct_choice": envRight, "cases": len(cases)}
	r.check("action envelope always parses", envValid == len(cases), fmt.Sprintf("%d/%d", envValid, len(cases)))
	r.check("action envelope picks tool vs final correctly (>= 4/5)", envRight >= 4, fmt.Sprintf("%d/%d", envRight, len(cases)))
	return r.write(*dir)
}

func validateDecision(s string) error {
	var d struct {
		Severity string  `json:"severity"`
		Action   string  `json:"action"`
		Priority float64 `json:"priority"`
	}
	dec := json.NewDecoder(strings.NewReader(s))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&d); err != nil {
		return err
	}
	in := func(v string, set ...string) bool {
		for _, x := range set {
			if v == x {
				return true
			}
		}
		return false
	}
	if !in(d.Severity, "low", "medium", "high", "critical") || !in(d.Action, "allow", "request_review", "block") ||
		d.Priority < 1 || d.Priority > 5 || d.Priority != float64(int(d.Priority)) {
		return fmt.Errorf("out of schema: %s", s)
	}
	return nil
}

// ---- lora probe -----------------------------------------------------------

type loraAdapter struct {
	ID    int     `json:"id"`
	Path  string  `json:"path"`
	Scale float64 `json:"scale"`
}

func (c *llamaClient) loraList() ([]loraAdapter, int, error) {
	var l []loraAdapter
	code, err := c.rawGet("/lora-adapters", &l)
	return l, code, err
}

func (c *llamaClient) loraPost(body any) (int, string, error) {
	b, _ := json.Marshal(body)
	resp, err := c.rawHC.Post(c.rawBase+"/lora-adapters", "application/json", bytes.NewReader(b))
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()
	rb, _ := io.ReadAll(io.LimitReader(resp.Body, 400))
	return resp.StatusCode, string(rb), nil
}

// cmdLora answers O3: can this build hot-swap? Scale changes and per-request `lora`
// are the "activation" half; loading a new adapter file into a running server is the half
// the repo records as unsupported on build 9e0e220.
func cmdLora(args []string) error {
	fs := flag.NewFlagSet("lora", flag.ExitOnError)
	dir := fs.String("dir", "measure", "working directory")
	lf := addLlamaFlags(fs)
	newAdapter := fs.String("new-adapter", "", "path of a GGUF LoRA NOT loaded at start, to test runtime loading")
	serverBin := fs.String("server-bin", "", "llama-server binary to record --version for (run once per build you want compared)")
	fs.Parse(args)

	r := newResult("lora")
	if *serverBin != "" {
		out, _ := exec.Command(*serverBin, "--version").CombinedOutput()
		r.Data["server_version"] = strings.TrimSpace(string(out))
	}
	c, err := lf.client()
	if err != nil {
		return err
	}
	if c.rawBase == "" {
		return r.blocked("the wrapper does not proxy /lora-adapters: pass -llama-raw host:port of the private llama-server").write(*dir)
	}
	if err := c.healthy(); err != nil {
		return r.blocked("llama is not reachable: %v", err).write(*dir)
	}
	before, code, err := c.loraList()
	if err != nil || code != 200 {
		return r.failed("GET /lora-adapters: HTTP %d %v (build too old for runtime adapter control?)", code, err).write(*dir)
	}
	r.Data["loaded_at_start"] = before
	r.check("GET /lora-adapters is exposed", true, fmt.Sprintf("%d adapter(s) loaded", len(before)))

	if len(before) == 0 {
		r.note("no adapter is loaded: start llama-server with --lora <gguf> --lora-init-without-apply to exercise scale switching")
	} else {
		// Scale switching: set the first adapter to 0, then 1, read back each time.
		okScale := true
		for _, s := range []float64{0, 1} {
			if code, body, err := c.loraPost([]loraAdapter{{ID: before[0].ID, Scale: s}}); err != nil || code != 200 {
				okScale = false
				r.note("POST scale %v: HTTP %d %v %s", s, code, err, body)
				break
			}
			now, _, _ := c.loraList()
			if len(now) == 0 || now[0].Scale != s {
				okScale = false
			}
		}
		r.check("adapter scale can be switched at runtime", okScale, "")

		// Per-request `lora` must change output at temperature 0 when the adapter does something.
		ctx := context.Background()
		prompt := []chatMsg{{"user", "Describe a guardrail in one sentence."}}
		off, _, e1 := c.chat(ctx, prompt, 40, map[string]any{"lora": []loraAdapter{{ID: before[0].ID, Scale: 0}}})
		on, _, e2 := c.chat(ctx, prompt, 40, map[string]any{"lora": []loraAdapter{{ID: before[0].ID, Scale: 1}}})
		if e1 == nil && e2 == nil {
			r.Data["per_request_lora_changes_output"] = off.Content != on.Content
			r.check("per-request `lora` field changes the output", off.Content != on.Content, "adapter must be non-trivial for this to differ")
		}
	}

	if *newAdapter == "" {
		r.note("runtime loading of a NEW adapter file not tested: pass -new-adapter <gguf-not-loaded-at-start>")
		return r.write(*dir)
	}
	code, body, err := c.loraPost([]loraAdapter{{ID: len(before), Path: *newAdapter, Scale: 1}})
	after, _, _ := c.loraList()
	appeared := len(after) > len(before)
	for _, a := range after {
		if a.Path == *newAdapter {
			appeared = true
		}
	}
	r.Data["new_adapter_probe"] = map[string]any{"post_http": code, "post_body": body, "post_err": fmt.Sprint(err), "adapters_after": after}
	r.Data["runtime_adapter_loading"] = appeared
	r.check("runtime loading of a new adapter file works on this build", appeared,
		"if false, adapters join the preloaded pool at the next restart (design §6.4)")
	if code == 200 && !appeared {
		r.note("POST returned success and changed nothing: the same behaviour the repo records for build 9e0e220")
	}
	return r.write(*dir)
}

// ---- llama sweep ----------------------------------------------------------

// cmdLlamaSweep spawns its own llama-server on a private port per setting, so it never
// touches the running one. It refuses to run without -yes because Qwen 7B on this host
// competes with the live services for the same CPU and RAM.
func cmdLlamaSweep(args []string) error {
	fs := flag.NewFlagSet("llama-sweep", flag.ExitOnError)
	dir := fs.String("dir", "measure", "working directory")
	bin := fs.String("server-bin", "", "llama-server binary")
	model := fs.String("model", "", "GGUF model path")
	matrix := fs.String("matrix", "ctx=2048,8192;threads=4,6;parallel=1", "settings to try; every combination runs")
	port := fs.Int("port", 18099, "private loopback port")
	yes := fs.Bool("yes", false, "confirm: this loads the model once per combination")
	frame := fs.Int("frame-tokens", 1200, "size of the stable prefix to test caching with")
	fs.Parse(args)

	r := newResult("llama-sweep")
	if *bin == "" || *model == "" {
		return r.blocked("needs -server-bin and -model (the model file under /var/lib/knirvserver/knirvllama/models is not readable by this user)").write(*dir)
	}
	if !*yes {
		return r.blocked("not run: re-run with -yes (loads the model once per combination)").write(*dir)
	}
	axes := map[string][]string{"ctx": {"2048"}, "threads": {"4"}, "parallel": {"1"}}
	for _, part := range strings.Split(*matrix, ";") {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) == 2 {
			axes[strings.TrimSpace(kv[0])] = splitList(kv[1])
		}
	}
	var runs []map[string]any
	for _, ctxs := range axes["ctx"] {
		for _, th := range axes["threads"] {
			for _, par := range axes["parallel"] {
				runs = append(runs, sweepOne(*bin, *model, *port, ctxs, th, par, *frame))
			}
		}
	}
	r.Data["runs"] = runs
	r.note("compare cold_ms/warm_ms and rss_mb across rows to choose ctx_size, threads and parallel (O14)")
	return r.write(*dir)
}

func sweepOne(bin, model string, port int, ctxSize, threads, parallel string, frameTokens int) map[string]any {
	row := map[string]any{"ctx": ctxSize, "threads": threads, "parallel": parallel}
	addr := net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
	cmd := exec.Command(bin, "-m", model, "--host", "127.0.0.1", "--port", strconv.Itoa(port),
		"-c", ctxSize, "-t", threads, "-np", parallel, "--cache-reuse", "256")
	if err := cmd.Start(); err != nil {
		row["error"] = err.Error()
		return row
	}
	defer func() { cmd.Process.Kill(); cmd.Wait() }()
	c := &llamaClient{chatHC: &http.Client{Timeout: 15 * time.Minute}, chatBase: "http://" + addr}
	c.rawHC, c.rawBase = c.chatHC, c.chatBase
	up := false
	for i := 0; i < 600; i++ {
		if c.healthy() == nil {
			up = true
			break
		}
		time.Sleep(time.Second)
	}
	if !up {
		row["error"] = "server did not become healthy in 10 minutes"
		return row
	}
	prefix := filler(frameTokens, "sweep"+ctxSize)
	var wall [2]float64
	for i := 0; i < 2; i++ {
		out, _, err := c.chat(context.Background(), []chatMsg{{"system", prefix}, {"user", fmt.Sprintf("Task %d: reply OK.", i)}}, 8, map[string]any{"cache_prompt": true})
		if err != nil {
			row["error"] = err.Error()
			return row
		}
		wall[i] = ms(out.Wall)
		if i == 0 {
			row["prompt_tokens"] = out.PromptTokens
			row["prompt_tok_per_s"] = timing(out.Timings, "prompt_per_second")
			row["gen_tok_per_s"] = timing(out.Timings, "predicted_per_second")
		}
	}
	row["cold_ms"], row["warm_ms"] = wall[0], wall[1]
	for _, p := range findProcs("llama-server", "llama_server") {
		if p.PID == cmd.Process.Pid {
			row["rss_mb"] = p.RSSMB
		}
	}
	_ = os.Stdout
	return row
}

// bearerTransport adds the llama-server API key to every request.
type bearerTransport struct {
	key  string
	next http.RoundTripper
}

func (b bearerTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	r.Header.Set("Authorization", "Bearer "+b.key)
	return b.next.RoundTrip(r)
}
