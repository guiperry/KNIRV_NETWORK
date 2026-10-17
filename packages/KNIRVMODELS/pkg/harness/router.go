package harness

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Backend is one place inference can be served from. Backends are tried in the order
// the Router was given them, so the first ready one is the preferred one.
type Backend struct {
	Name string
	// Ready reports whether the backend can take a request now.
	Ready func() bool
	// Dial returns the network and address to connect to ("unix", "/path.sock" or
	// "tcp", "127.0.0.1:8090"). It is evaluated per request because Colibri's port can
	// move.
	Dial func() (network, addr string)
	// Model, when it returns a non-empty id, replaces the "model" field of JSON
	// requests: each engine accepts only its own model id.
	Model func() string
	// Supports reports whether the backend can serve a request with these needs. Nil
	// means everything. A backend that cannot is skipped for such requests.
	Supports func(Needs) bool
	// Teacher reports whether the backend takes teacher-role requests now. Nil means
	// the backend never does (teacher output must come from the strong model only).
	Teacher func() bool
	// Demand is called when a request needs this backend, whether or not it is ready
	// yet: the owner starts it if it is stopped and records that it is in use. Nil
	// means the backend cannot be started on demand.
	Demand func(reason string)

	inflight atomic.Int32
}

// Busy reports whether a request is being served by the backend right now.
func (b *Backend) Busy() bool { return b.inflight.Load() > 0 }

// Needs is what a request requires beyond plain chat. Colibri serves plain chat; the
// rest goes to a backend that supports it (llama.cpp), started on demand.
type Needs struct {
	// Adapter: a ULoRA skill adapter (the /v1/adapters routes or an "adapter" field).
	Adapter bool
	// Grammar: output forced to a JSON object, JSON schema or grammar (response_format).
	Grammar bool
	// Tools: tool/function calls.
	Tools bool
}

// Any reports whether the request needs anything beyond plain chat.
func (n Needs) Any() bool { return n.Adapter || n.Grammar || n.Tools }

func (n Needs) String() string {
	var parts []string
	if n.Adapter {
		parts = append(parts, "a skill adapter")
	}
	if n.Grammar {
		parts = append(parts, "structured output")
	}
	if n.Tools {
		parts = append(parts, "tool calls")
	}
	return strings.Join(parts, " and ")
}

// RoleHeader marks the kind of caller. "teacher" is batch work whose output is kept
// (synthetic training data, suggested labels, retrieval judgments): it is served only
// by a backend that takes the teacher role, and is refused rather than answered by
// the fallback, so what the caller records about its source is true.
const RoleHeader = "X-Knirv-Role"

// RoleTeacher is the RoleHeader value for teacher requests.
const RoleTeacher = "teacher"

// BackendHeader and ModelHeader are set on every response: which backend answered
// and the model id it used, for provenance records.
const (
	BackendHeader = "X-Knirv-Backend"
	ModelHeader   = "X-Knirv-Model"
)

// RequireBackendHeader marks a request that must be answered by a backend with
// adapter support even when it would otherwise be answered locally (the skill adapter
// orchestrator sets it on its first call).
const RequireBackendHeader = "X-Knirv-Require-Backend"

// detectNeeds reads what a request requires from its path, headers and JSON body.
func detectNeeds(req *http.Request, doc map[string]json.RawMessage) Needs {
	var n Needs
	if strings.HasPrefix(req.URL.Path, "/v1/adapters") || req.Header.Get(RequireBackendHeader) != "" {
		n.Adapter = true
	}
	if raw, ok := doc["adapter"]; ok {
		var name string
		if json.Unmarshal(raw, &name) == nil && name != "" {
			n.Adapter = true
		}
	}
	if raw, ok := doc["response_format"]; ok {
		var rf struct {
			Type string `json:"type"`
		}
		if json.Unmarshal(raw, &rf) == nil && rf.Type != "" && rf.Type != "text" {
			n.Grammar = true
		}
	}
	for _, k := range []string{"grammar", "json_schema"} {
		if raw, ok := doc[k]; ok && len(raw) > 0 && string(raw) != "null" && string(raw) != `""` {
			n.Grammar = true
		}
	}
	for _, k := range []string{"tools", "functions"} {
		if raw, ok := doc[k]; ok && len(raw) > 0 && string(raw) != "null" && string(raw) != "[]" {
			n.Tools = true
		}
	}
	return n
}

// Router is the one inference endpoint callers use (the socket backend_server already
// registers). It forwards each request to the first ready backend, streaming
// responses unbuffered, and answers /harness/status itself.
type Router struct {
	backends []*Backend
	logf     func(string, ...any)
	// Extra is merged into the /harness/status document.
	Extra func() map[string]any

	// OnDemandWait bounds how long a request waits for a backend that has to be
	// started for it (default 10m: llama.cpp loads its model from cold).
	OnDemandWait time.Duration

	mu     sync.Mutex
	last   string
	proxy  *httputil.ReverseProxy
	bodies int64
}

// NewRouter builds a Router over backends in priority order.
func NewRouter(logf func(string, ...any), backends ...*Backend) *Router {
	if logf == nil {
		logf = func(string, ...any) {}
	}
	r := &Router{backends: backends, logf: logf}
	r.proxy = &httputil.ReverseProxy{
		Director:      func(*http.Request) {},
		FlushInterval: -1,
		ErrorHandler: func(w http.ResponseWriter, req *http.Request, err error) {
			writeJSONError(w, http.StatusBadGateway, "model backend error: "+err.Error(), r.states())
		},
	}
	return r
}

// Active is the backend that would serve a request now, or nil.
func (r *Router) Active() *Backend {
	for _, b := range r.backends {
		if b.Ready != nil && b.Ready() {
			return b
		}
	}
	return nil
}

func (r *Router) states() map[string]bool {
	out := make(map[string]bool, len(r.backends))
	for _, b := range r.backends {
		out[b.Name] = b.Ready != nil && b.Ready()
	}
	return out
}

// Status is the /harness/status document.
func (r *Router) Status() map[string]any {
	doc := map[string]any{"backends": r.states()}
	if b := r.Active(); b != nil {
		doc["active"] = b.Name
	} else {
		doc["active"] = nil
	}
	if r.Extra != nil {
		for k, v := range r.Extra() {
			doc[k] = v
		}
	}
	return doc
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	switch req.URL.Path {
	case "/harness/status":
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(r.Status())
		return
	case "/health":
		// Answered here: every backend has its own idea of /health, and callers want
		// to know whether any model can answer.
		w.Header().Set("Content-Type", "application/json")
		if b := r.Active(); b != nil {
			json.NewEncoder(w).Encode(map[string]any{"status": "ok", "backend": b.Name})
			return
		}
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]any{"status": "unavailable", "backends": r.states()})
		return
	}

	body, doc, err := bufferJSON(req)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error(), nil)
		return
	}
	needs := detectNeeds(req, doc)

	// Adapter state is llama.cpp's. Asking about it must not start llama.cpp: while it
	// is not running there are simply no adapters.
	if req.Method == http.MethodGet && req.URL.Path == "/v1/adapters" && req.Header.Get(RequireBackendHeader) == "" {
		if b := r.pick(needs); b == nil {
			w.Header().Set("Content-Type", "application/json")
			io.WriteString(w, `{"base":{"path":"","sha256":""},"adapters":[],"unavailable":"adapters need llama.cpp, which is not running"}`+"\n")
			return
		}
	}

	var b *Backend
	if strings.EqualFold(req.Header.Get(RoleHeader), RoleTeacher) {
		if b = r.teacher(); b == nil {
			w.Header().Set("Retry-After", "30")
			writeJSONError(w, http.StatusServiceUnavailable, "no teacher backend is available (Colibri is not serving on this host); teacher requests are never answered by the fallback", r.states())
			return
		}
	} else {
		b = r.pick(needs)
		if needs.Any() {
			b = r.awaitSupporting(req.Context(), needs, b)
		}
	}
	if b == nil {
		if needs.Any() && r.hasSupporting(needs) {
			w.Header().Set("Retry-After", "30")
			writeJSONError(w, http.StatusServiceUnavailable, "the model backend that supports "+needs.String()+" is not ready yet", r.states())
			return
		}
		b = r.Active() // nothing supports it: the active backend answers, or refuses clearly
	}
	if b == nil {
		w.Header().Set("Retry-After", "5")
		writeJSONError(w, http.StatusServiceUnavailable, "no model backend is ready yet", r.states())
		return
	}
	if needs.Any() && b.Demand != nil {
		b.Demand(needs.String()) // keeps an on-demand backend alive while it is used
	}
	b.inflight.Add(1)
	defer b.inflight.Add(-1)

	r.noteActive(b.Name)
	network, addr := b.Dial()
	if b.Model != nil {
		if m := b.Model(); m != "" {
			setModel(req, body, doc, m)
		}
	}
	out := req.Clone(req.Context())
	out.URL = &url.URL{Scheme: "http", Host: "backend", Path: req.URL.Path, RawQuery: req.URL.RawQuery}
	out.Host = "backend"
	if network == "tcp" {
		out.URL.Host, out.Host = addr, addr
	}
	// A per-request transport keeps the backend choice and its dial target together.
	rp := *r.proxy
	rp.ModifyResponse = func(resp *http.Response) error {
		resp.Header.Set(BackendHeader, b.Name)
		if b.Model != nil {
			if m := b.Model(); m != "" {
				resp.Header.Set(ModelHeader, m)
			}
		}
		return nil
	}
	rp.Transport = &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			var d net.Dialer
			return d.DialContext(ctx, network, addr)
		},
		DisableKeepAlives:     true,
		ResponseHeaderTimeout: 0,
	}
	defer rp.Transport.(*http.Transport).CloseIdleConnections()
	rp.ServeHTTP(w, out)
}

// pick is the first ready backend that supports the needs, or nil.
func (r *Router) pick(n Needs) *Backend {
	for _, b := range r.backends {
		if b.Ready != nil && b.Ready() && (b.Supports == nil || b.Supports(n)) {
			return b
		}
	}
	return nil
}

// teacher is the first backend that is ready and takes the teacher role, or nil.
func (r *Router) teacher() *Backend {
	for _, b := range r.backends {
		if b.Teacher != nil && b.Teacher() && b.Ready != nil && b.Ready() {
			return b
		}
	}
	return nil
}

func (r *Router) hasSupporting(n Needs) bool {
	for _, b := range r.backends {
		if b.Supports == nil || b.Supports(n) {
			return true
		}
	}
	return false
}

// awaitSupporting returns ready when it already supports the needs, else starts the
// first backend that does (Demand) and waits for it. It returns nil when none is
// configured, or when the wait runs out or the request is cancelled.
func (r *Router) awaitSupporting(ctx context.Context, n Needs, ready *Backend) *Backend {
	if ready != nil {
		return ready
	}
	var want *Backend
	for _, b := range r.backends {
		if b.Supports == nil || b.Supports(n) {
			want = b
			break
		}
	}
	if want == nil {
		return nil
	}
	wait := r.OnDemandWait
	if wait <= 0 {
		wait = 10 * time.Minute
	}
	if want.Demand != nil {
		r.logf("starting %s for a request that needs %s", want.Name, n.String())
		want.Demand(n.String())
	}
	deadline := time.NewTimer(wait)
	defer deadline.Stop()
	tick := time.NewTicker(250 * time.Millisecond)
	defer tick.Stop()
	for {
		if b := r.pick(n); b != nil {
			return b
		}
		select {
		case <-ctx.Done():
			return nil
		case <-deadline.C:
			return nil
		case <-tick.C:
		}
	}
}

// noteActive logs when traffic moves from one backend to another.
func (r *Router) noteActive(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.last != name {
		if r.last != "" {
			r.logf("inference now served by %s (was %s)", name, r.last)
		} else {
			r.logf("inference served by %s", name)
		}
		r.last = name
	}
}

// bufferJSON reads a JSON POST body once so the request can be inspected and still be
// forwarded. Other requests are returned untouched with a nil document.
func bufferJSON(req *http.Request) ([]byte, map[string]json.RawMessage, error) {
	if req.Body == nil || req.Method != http.MethodPost || !strings.Contains(req.Header.Get("Content-Type"), "json") {
		return nil, nil, nil
	}
	body, err := io.ReadAll(io.LimitReader(req.Body, 64<<20))
	req.Body.Close()
	if err != nil {
		return nil, nil, fmt.Errorf("read request body: %w", err)
	}
	var doc map[string]json.RawMessage
	if json.Unmarshal(body, &doc) != nil {
		doc = nil
	}
	replaceBody(req, body)
	return body, doc, nil
}

func replaceBody(req *http.Request, body []byte) {
	req.Body = io.NopCloser(bytes.NewReader(body))
	req.ContentLength = int64(len(body))
	req.Header.Set("Content-Length", strconv.Itoa(len(body)))
	req.GetBody = func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(body)), nil }
}

// setModel sets the "model" field of a buffered JSON request body.
func setModel(req *http.Request, body []byte, doc map[string]json.RawMessage, model string) {
	if body == nil {
		return
	}
	if doc != nil {
		doc["model"], _ = json.Marshal(model)
		if nb, err := json.Marshal(doc); err == nil {
			body = nb
		}
	}
	replaceBody(req, body)
}

func writeJSONError(w http.ResponseWriter, code int, msg string, backends map[string]bool) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	doc := map[string]any{"error": map[string]any{"message": msg, "type": "model_backend_unavailable"}}
	if backends != nil {
		doc["backends"] = backends
	}
	json.NewEncoder(w).Encode(doc)
}

// Serve listens on socketPath and serves until ctx ends. The socket appears only once
// some backend is ready, so a caller that waits for it (backend_server does) does not
// register a provider that cannot answer yet.
func (r *Router) Serve(ctx context.Context, socketPath string, perm os.FileMode) error {
	for r.Active() == nil {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}
	}
	if err := os.MkdirAll(filepath.Dir(socketPath), 0o755); err != nil {
		return err
	}
	if fi, err := os.Lstat(socketPath); err == nil && fi.Mode()&os.ModeSocket != 0 {
		os.Remove(socketPath) // stale socket from a previous run
	}
	ln, err := net.Listen("unix", socketPath)
	if err != nil {
		return fmt.Errorf("listen %s: %w", socketPath, err)
	}
	if err := os.Chmod(socketPath, perm); err != nil {
		ln.Close()
		return err
	}
	srv := &http.Server{Handler: r}
	go func() {
		<-ctx.Done()
		sctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv.Shutdown(sctx)
		os.Remove(socketPath)
	}()
	r.logf("inference endpoint listening on unix://%s", socketPath)
	if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}
