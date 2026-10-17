package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"gopkg.in/yaml.v3"
)

// ---- host -----------------------------------------------------------------

type procInfo struct {
	PID   int    `json:"pid"`
	Comm  string `json:"comm"`
	State string `json:"state"`
	Args  string `json:"args"`
	RSSMB int    `json:"rss_mb"`
}

// findProcs lists processes whose command name contains any of the needles.
func findProcs(needles ...string) []procInfo {
	var out []procInfo
	ents, _ := os.ReadDir("/proc")
	for _, e := range ents {
		pid, err := strconv.Atoi(e.Name())
		if err != nil {
			continue
		}
		stat, err := os.ReadFile(filepath.Join("/proc", e.Name(), "stat"))
		if err != nil {
			continue
		}
		s := string(stat)
		lp, rp := strings.IndexByte(s, '('), strings.LastIndexByte(s, ')')
		if lp < 0 || rp < lp {
			continue
		}
		comm := s[lp+1 : rp]
		match := false
		for _, n := range needles {
			if strings.Contains(strings.ToLower(comm), n) {
				match = true
			}
		}
		if !match {
			continue
		}
		fields := strings.Fields(s[rp+2:])
		pi := procInfo{PID: pid, Comm: comm}
		if len(fields) > 0 {
			pi.State = fields[0]
		}
		if cl, err := os.ReadFile(filepath.Join("/proc", e.Name(), "cmdline")); err == nil {
			pi.Args = redactArgs(strings.TrimSpace(strings.ReplaceAll(string(cl), "\x00", " ")))
		}
		if st, err := os.ReadFile(filepath.Join("/proc", e.Name(), "status")); err == nil {
			for _, l := range strings.Split(string(st), "\n") {
				if strings.HasPrefix(l, "VmRSS:") {
					kb, _ := strconv.Atoi(strings.Fields(l)[1])
					pi.RSSMB = kb / 1024
				}
			}
		}
		out = append(out, pi)
	}
	return out
}

func memInfoMB() (total, avail int) {
	b, _ := os.ReadFile("/proc/meminfo")
	for _, l := range strings.Split(string(b), "\n") {
		f := strings.Fields(l)
		if len(f) < 2 {
			continue
		}
		kb, _ := strconv.Atoi(f[1])
		switch f[0] {
		case "MemTotal:":
			total = kb / 1024
		case "MemAvailable:":
			avail = kb / 1024
		}
	}
	return
}

// runtimeConfig is the slice of the server config these probes care about.
type runtimeConfig struct {
	Llama struct {
		Enabled  bool `yaml:"enabled"`
		Parallel int  `yaml:"parallel"`
		CtxSize  int  `yaml:"ctx_size"`
		Threads  int  `yaml:"threads"`
	} `yaml:"llama"`
	Embeddings map[string]any `yaml:"embeddings"`
}

func readConfigKeys(path string) map[string]any {
	out := map[string]any{"path": path}
	b, err := os.ReadFile(path)
	if err != nil {
		out["error"] = err.Error()
		return out
	}
	var rc runtimeConfig
	if err := yaml.Unmarshal(b, &rc); err == nil {
		out["llama"] = rc.Llama
	}
	// The embedder keys sit at varying depths; find them textually rather than guess a schema.
	for _, l := range strings.Split(string(b), "\n") {
		t := strings.TrimSpace(l)
		for _, k := range []string{"cloudflare_embeddings_url", "ollama_base_url", "ollama_embedding_model"} {
			if strings.HasPrefix(t, k+":") {
				out[k] = strings.Trim(strings.TrimSpace(strings.TrimPrefix(t, k+":")), `"'`)
			}
		}
	}
	return out
}

func cmdHost(args []string) error {
	fs := flag.NewFlagSet("host", flag.ExitOnError)
	dir := fs.String("dir", "measure", "working directory")
	cfg := fs.String("config", "", "server config yaml to read llama/embedder settings from")
	dataDir := fs.String("data", "/var/lib/knirvserver", "server data dir")
	fs.Parse(args)

	r := newResult("host")
	r.Data["cores"] = runtime.NumCPU()
	total, avail := memInfoMB()
	r.Data["mem_total_mb"], r.Data["mem_available_mb"] = total, avail
	if b, err := os.ReadFile("/proc/loadavg"); err == nil {
		r.Data["loadavg"] = strings.Fields(string(b))[:3]
	}
	var st syscall.Statfs_t
	if syscall.Statfs(*dataDir, &st) == nil {
		r.Data["disk_free_gb"] = int(st.Bavail * uint64(st.Bsize) >> 30)
	}
	llamaProcs := findProcs("knirvllama", "llama-server", "llama_server")
	r.Data["llama_procs"] = llamaProcs
	r.Data["ollama_procs"] = findProcs("ollama")
	sock := filepath.Join(*dataDir, "sockets", "llama.sock")
	_, serr := os.Stat(sock)
	r.Data["llama_sock"] = map[string]any{"path": sock, "exists": serr == nil}
	live := false
	for _, p := range llamaProcs {
		if p.State != "Z" {
			live = true
		}
	}
	r.check("a live (non-zombie) llama process exists", live, fmt.Sprintf("%d match(es)", len(llamaProcs)))
	if *cfg != "" {
		r.Data["config"] = readConfigKeys(*cfg)
	}
	if avail < 8000 {
		r.note("only %d MB RAM available: spawning extra llama-server instances (llama-sweep) will swap or OOM", avail)
	}
	return r.write(*dir)
}

// ---- ollama ---------------------------------------------------------------

type listener struct {
	Addr  string `json:"addr"`
	Port  int    `json:"port"`
	Inode string `json:"inode"`
}

// parseProcNetTCP returns listening sockets from /proc/net/tcp{,6} content.
func parseProcNetTCP(content string, v6 bool) []listener {
	var out []listener
	lines := strings.Split(content, "\n")
	for i, l := range lines {
		if i == 0 {
			continue
		}
		f := strings.Fields(l)
		if len(f) < 10 || f[3] != "0A" { // 0A = LISTEN
			continue
		}
		hp := strings.Split(f[1], ":")
		if len(hp) != 2 {
			continue
		}
		port, _ := strconv.ParseInt(hp[1], 16, 32)
		out = append(out, listener{Addr: hexAddr(hp[0], v6), Port: int(port), Inode: f[9]})
	}
	return out
}

func hexAddr(h string, v6 bool) string {
	b := make([]byte, len(h)/2)
	for i := range b {
		v, _ := strconv.ParseUint(h[2*i:2*i+2], 16, 8)
		b[i] = byte(v)
	}
	// /proc stores each 32-bit word in host (little-endian) byte order.
	for i := 0; i+3 < len(b); i += 4 {
		b[i], b[i+1], b[i+2], b[i+3] = b[i+3], b[i+2], b[i+1], b[i]
	}
	if !v6 && len(b) == 4 {
		return net.IP(b).String()
	}
	return net.IP(b).String()
}

func listeningTCP() []listener {
	var out []listener
	for _, f := range []struct {
		p  string
		v6 bool
	}{{"/proc/net/tcp", false}, {"/proc/net/tcp6", true}} {
		if b, err := os.ReadFile(f.p); err == nil {
			out = append(out, parseProcNetTCP(string(b), f.v6)...)
		}
	}
	return out
}

func isLoopback(addr string) bool {
	ip := net.ParseIP(addr)
	return ip != nil && ip.IsLoopback()
}

type ollamaTags struct {
	Models []struct {
		Name    string `json:"name"`
		Digest  string `json:"digest"`
		Size    int64  `json:"size"`
		Details struct {
			Family string `json:"family"`
			Params string `json:"parameter_size"`
			Quant  string `json:"quantization_level"`
		} `json:"details"`
	} `json:"models"`
}

// probeOllama reports whether host:port answers like Ollama.
func probeOllama(hc *http.Client, base string) (banner string, ok bool) {
	resp, err := hc.Get(base + "/")
	if err != nil {
		return "", false
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 256))
	return strings.TrimSpace(string(b)), strings.Contains(string(b), "Ollama is running")
}

func cmdOllama(args []string) error {
	fs := flag.NewFlagSet("ollama", flag.ExitOnError)
	dir := fs.String("dir", "measure", "working directory")
	model := fs.String("model", "nomic-embed-text", "embedding model to time")
	n := fs.Int("n", 16, "texts to embed for the throughput figure")
	configured := fs.String("configured", "http://localhost:11434", "the URL the config points at (ollama_base_url)")
	fs.Parse(args)

	r := newResult("ollama")
	hc := &http.Client{Timeout: 3 * time.Second}

	// Every listener that answers like Ollama, whatever its port.
	type found struct {
		listener
		URL      string `json:"url"`
		Loopback bool   `json:"loopback_only"`
		Version  string `json:"version,omitempty"`
		Models   any    `json:"models,omitempty"`
	}
	var instances []found
	seenPort := map[int]bool{}
	for _, l := range listeningTCP() {
		if seenPort[l.Port] {
			continue
		}
		host := l.Addr
		if ip := net.ParseIP(host); ip == nil || ip.IsUnspecified() {
			host = "127.0.0.1"
		}
		base := "http://" + net.JoinHostPort(host, strconv.Itoa(l.Port))
		if _, ok := probeOllama(hc, base); !ok {
			continue
		}
		seenPort[l.Port] = true
		f := found{listener: l, URL: base, Loopback: isLoopback(l.Addr)}
		var v struct{ Version string }
		if resp, err := hc.Get(base + "/api/version"); err == nil {
			json.NewDecoder(resp.Body).Decode(&v)
			resp.Body.Close()
			f.Version = v.Version
		}
		var tags ollamaTags
		if resp, err := hc.Get(base + "/api/tags"); err == nil {
			json.NewDecoder(resp.Body).Decode(&tags)
			resp.Body.Close()
			var ms []map[string]any
			for _, m := range tags.Models {
				ms = append(ms, map[string]any{"name": m.Name, "digest": m.Digest[:min(12, len(m.Digest))], "params": m.Details.Params, "quant": m.Details.Quant})
			}
			f.Models = ms
		}
		instances = append(instances, f)
	}
	r.Data["instances"] = instances
	r.Data["configured_url"] = *configured
	r.Data["processes"] = findProcs("ollama")

	if len(instances) == 0 {
		return r.blocked("no listener answers like Ollama; start it (the server launcher does not manage it)").write(*dir)
	}
	r.check("exactly one Ollama instance answers", len(instances) == 1, fmt.Sprintf("%d found", len(instances)))

	cu := strings.TrimRight(*configured, "/")
	matches := false
	for _, in := range instances {
		if strings.TrimRight(in.URL, "/") == cu || strings.Replace(in.URL, "127.0.0.1", "localhost", 1) == cu {
			matches = true
		}
	}
	r.check("configured ollama_base_url reaches it", matches, "config "+*configured)
	for _, in := range instances {
		r.check("binds loopback only ("+in.URL+")", in.Loopback, "bound "+in.Addr)
	}

	// Embedding speed on the first instance.
	base := instances[0].URL
	texts := make([]string, *n)
	for i := range texts {
		texts[i] = fmt.Sprintf("Measurement sample %d: the quick brown fox jumps over the lazy dog near the river bank.", i)
	}
	ec := &http.Client{Timeout: 120 * time.Second}
	var lat []float64
	dims := 0
	start := time.Now()
	for _, t := range texts {
		t0 := time.Now()
		v, err := ollamaEmbed(ec, base, *model, t)
		if err != nil {
			r.note("embed failed: %v", err)
			break
		}
		dims = len(v)
		lat = append(lat, float64(time.Since(t0).Milliseconds()))
	}
	if len(lat) > 0 {
		el := time.Since(start).Seconds()
		r.Data["embed"] = map[string]any{"model": *model, "dims": dims, "texts": len(lat),
			"texts_per_s": r3(float64(len(lat)) / el), "p50_ms": percentile(lat, .5), "p95_ms": percentile(lat, .95),
			"note": "first call includes model load; rerun for warm figures"}
		r.check("embedding returns 768 dims", dims == 768, fmt.Sprint(dims))
	}
	return r.write(*dir)
}

func ollamaEmbed(hc *http.Client, base, model, text string) ([]float32, error) {
	body, _ := json.Marshal(map[string]string{"model": model, "prompt": text})
	resp, err := hc.Post(strings.TrimRight(base, "/")+"/api/embeddings", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 200))
		return nil, fmt.Errorf("ollama %d: %s", resp.StatusCode, b)
	}
	var o struct {
		Embedding []float32 `json:"embedding"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&o); err != nil {
		return nil, err
	}
	return o.Embedding, nil
}

// ---- cloudflare -----------------------------------------------------------

func cloudflareEmbed(hc *http.Client, url, text string) (vec []float32, model string, hdr http.Header, status int, err error) {
	body, _ := json.Marshal(map[string]string{"text": text})
	resp, err := hc.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, "", nil, 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<16))
		return nil, "", resp.Header, resp.StatusCode, fmt.Errorf("cloudflare %d", resp.StatusCode)
	}
	var o struct {
		Success   bool      `json:"success"`
		Model     string    `json:"model"`
		Embedding []float32 `json:"embedding"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&o); err != nil {
		return nil, "", resp.Header, resp.StatusCode, err
	}
	if !o.Success {
		return nil, o.Model, resp.Header, resp.StatusCode, fmt.Errorf("cloudflare reported success=false")
	}
	return o.Embedding, o.Model, resp.Header, resp.StatusCode, nil
}

func cmdCloudflare(args []string) error {
	fs := flag.NewFlagSet("cloudflare", flag.ExitOnError)
	dir := fs.String("dir", "measure", "working directory")
	url := fs.String("url", "https://embeddings.knirv.com", "embeddings endpoint (cloudflare_embeddings_url)")
	burst := fs.Int("burst", 5, "sequential requests to make; stops at the first non-200 (use to observe the limit, keep small)")
	capCfg := fs.Int("configured-cap", 5000, "daily cap the mapper assumes (CloudflareLimit default)")
	fs.Parse(args)

	r := newResult("cloudflare")
	hc := &http.Client{Timeout: 20 * time.Second}
	// Only a fixed public sentence is ever sent: no gold or user text.
	const sample = "The Eiffel Tower is located in Paris, France."
	var lat []float64
	var first []float32
	var model string
	hdrs := map[string]string{}
	stopped := ""
	for i := 0; i < *burst; i++ {
		t0 := time.Now()
		v, m, h, code, err := cloudflareEmbed(hc, *url, sample)
		for k, vs := range h {
			lk := strings.ToLower(k)
			if strings.Contains(lk, "ratelimit") || lk == "retry-after" || strings.HasPrefix(lk, "cf-") {
				hdrs[k] = strings.Join(vs, ",")
			}
		}
		if err != nil {
			stopped = fmt.Sprintf("request %d: HTTP %d: %v", i+1, code, err)
			break
		}
		lat = append(lat, float64(time.Since(t0).Milliseconds()))
		if first == nil {
			first, model = v, m
		}
	}
	r.Data["url"] = *url
	r.Data["requests_ok"] = len(lat)
	r.Data["model"] = model
	r.Data["dims"] = len(first)
	r.Data["p50_ms"] = percentile(lat, .5)
	r.Data["rate_headers"] = hdrs
	r.Data["configured_daily_cap"] = *capCfg
	r.Data["cap_arithmetic"] = fmt.Sprintf("12,000,000 chunks / %d per day = %.0f days", *capCfg, 12e6/float64(*capCfg))
	if stopped != "" {
		r.Data["stopped"] = stopped
	}
	if len(lat) == 0 {
		return r.failed("%s", stopped).write(*dir)
	}
	r.check("returns 768 dims", len(first) == 768, fmt.Sprint(len(first)))
	if 12e6/float64(*capCfg) > 30 {
		r.note("at %d requests/day a 12M-chunk index takes %.0f days: bulk indexing has to be local (E2)", *capCfg, 12e6/float64(*capCfg))
	}
	r.note("the server-side cap is not observable without exhausting it; this confirms only the client-side figure and any rate headers")
	return r.write(*dir)
}

var _ = bytes.NewReader

// redactArgs masks the value after any credential-looking flag so a result file
// never carries a server's API key.
func redactArgs(args string) string {
	f := strings.Fields(args)
	for i := 0; i+1 < len(f); i++ {
		switch f[i] {
		case "--api-key", "-api-key", "--key", "-key", "--token", "-token":
			f[i+1] = "REDACTED"
		}
	}
	return strings.Join(f, " ")
}
