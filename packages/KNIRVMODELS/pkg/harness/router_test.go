package harness

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type safeBuf struct {
	mu sync.Mutex
	b  strings.Builder
}

func (s *safeBuf) logf(f string, a ...any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	fmt.Fprintf(&s.b, f+"\n", a...)
}
func (s *safeBuf) String() string { s.mu.Lock(); defer s.mu.Unlock(); return s.b.String() }

// echoServer replies with the request's model and a marker naming the server.
func echoServer(name string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		var doc map[string]any
		json.Unmarshal(b, &doc)
		json.NewEncoder(w).Encode(map[string]any{"server": name, "model": doc["model"], "path": r.URL.Path})
	}))
}

func tcpBackend(name string, srv *httptest.Server, ready *atomic.Bool, model string) *Backend {
	addr := strings.TrimPrefix(srv.URL, "http://")
	b := &Backend{Name: name, Ready: ready.Load, Dial: func() (string, string) { return "tcp", addr }}
	if model != "" {
		b.Model = func() string { return model }
	}
	return b
}

func post(t *testing.T, h http.Handler, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// llama.cpp is listed first, so it is preferred whenever it is ready; Colibri serves
// while llama.cpp is not, with its own model id.
func TestRouterPrefersLlamaAndFallsBackToColibri(t *testing.T) {
	a, b := echoServer("colibri"), echoServer("llama")
	defer a.Close()
	defer b.Close()
	var aReady, bReady atomic.Bool
	r := NewRouter(nil, tcpBackend("llama.cpp", b, &bReady, ""), tcpBackend("colibri", a, &aReady, "colibri-small"))

	if rec := post(t, r, "/v1/chat/completions", `{"model":"x"}`); rec.Code != http.StatusServiceUnavailable || rec.Header().Get("Retry-After") == "" {
		t.Fatalf("nothing ready must be 503 with Retry-After, got %d", rec.Code)
	}

	aReady.Store(true)
	rec := post(t, r, "/v1/chat/completions", `{"model":"local-llama","messages":[]}`)
	var got map[string]any
	json.Unmarshal(rec.Body.Bytes(), &got)
	if got["server"] != "colibri" || got["model"] != "colibri-small" || got["path"] != "/v1/chat/completions" {
		t.Fatalf("colibri must serve while llama.cpp is down, with its own model id: %v", got)
	}

	bReady.Store(true)
	rec = post(t, r, "/v1/chat/completions", `{"model":"local-llama","messages":[]}`)
	got = nil
	json.Unmarshal(rec.Body.Bytes(), &got)
	if got["server"] != "llama" || got["model"] != "local-llama" {
		t.Fatalf("llama.cpp must be preferred when both are ready, untouched model: %v", got)
	}
	if rec.Header().Get(BackendHeader) != "llama.cpp" {
		t.Fatalf("backend header %q", rec.Header().Get(BackendHeader))
	}
}

func TestRouterStatusEndpoint(t *testing.T) {
	a := echoServer("colibri")
	defer a.Close()
	var ready atomic.Bool
	r := NewRouter(nil, tcpBackend("colibri", a, &ready, ""))
	r.Extra = func() map[string]any { return map[string]any{"note": "n"} }
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/harness/status", nil))
	var doc map[string]any
	json.Unmarshal(rec.Body.Bytes(), &doc)
	if doc["active"] != nil || doc["note"] != "n" || doc["backends"].(map[string]any)["colibri"] != false {
		t.Fatalf("%v", doc)
	}
	ready.Store(true)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/harness/status", nil))
	json.Unmarshal(rec.Body.Bytes(), &doc)
	if doc["active"] != "colibri" {
		t.Fatalf("%v", doc)
	}
}

func TestRouterStreamsWithoutBuffering(t *testing.T) {
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Write([]byte("data: one\n\n"))
		w.(http.Flusher).Flush()
		<-release
		w.Write([]byte("data: two\n\n"))
	}))
	defer srv.Close()
	var ready atomic.Bool
	ready.Store(true)
	front := httptest.NewServer(NewRouter(nil, tcpBackend("llama.cpp", srv, &ready, "")))
	defer front.Close()

	resp, err := http.Post(front.URL+"/v1/chat/completions", "application/json", strings.NewReader(`{"stream":true}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	buf := make([]byte, 64)
	done := make(chan string, 1)
	go func() { n, _ := resp.Body.Read(buf); done <- string(buf[:n]) }()
	select {
	case first := <-done:
		if !strings.Contains(first, "one") {
			t.Fatalf("first chunk %q", first)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("first event was held back until the response ended")
	}
	close(release)
}

func TestRouterServeCreatesSocketOnlyOnceABackendIsReady(t *testing.T) {
	a := echoServer("llama")
	defer a.Close()
	var ready atomic.Bool
	r := NewRouter(nil, tcpBackend("llama.cpp", a, &ready, ""))
	sock := filepath.Join(t.TempDir(), "sockets", "llama.sock")
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- r.Serve(ctx, sock, 0o660) }()

	time.Sleep(700 * time.Millisecond)
	if _, err := os.Stat(sock); err == nil {
		t.Fatal("socket must not exist while nothing can answer")
	}
	ready.Store(true)
	deadline := time.Now().Add(5 * time.Second)
	for {
		if conn, err := net.Dial("unix", sock); err == nil {
			conn.Close()
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("socket never appeared")
		}
		time.Sleep(50 * time.Millisecond)
	}
	if fi, _ := os.Stat(sock); fi.Mode().Perm() != 0o660 {
		t.Fatalf("perm %v", fi.Mode().Perm())
	}
	hc := &http.Client{Transport: &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", sock)
	}}}
	resp, err := hc.Post("http://x/v1/chat/completions", "application/json", strings.NewReader(`{"model":"m"}`))
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if !strings.Contains(string(b), `"server":"llama"`) {
		t.Fatalf("%s", b)
	}
	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Serve did not stop")
	}
	if _, err := os.Stat(sock); err == nil {
		t.Fatal("socket must be removed on shutdown")
	}
}

func TestRouterReplacesStaleSocket(t *testing.T) {
	sock := filepath.Join(t.TempDir(), "llama.sock")
	l, err := net.Listen("unix", sock)
	if err != nil {
		t.Skip(err)
	}
	l.(*net.UnixListener).SetUnlinkOnClose(false)
	l.Close() // leaves the socket file behind, as a crash does
	a := echoServer("x")
	defer a.Close()
	var ready atomic.Bool
	ready.Store(true)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go NewRouter(nil, tcpBackend("b", a, &ready, "")).Serve(ctx, sock, 0o660)
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if conn, err := net.Dial("unix", sock); err == nil {
			conn.Close()
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("stale socket was not replaced")
}

// plainChatOnly stands in for Colibri's qwen36 engine: chat, no grammar, no adapters.
func plainChatOnly(n Needs) bool { return !n.Adapter && !n.Grammar && !n.Tools && !n.Frame }

// llama.cpp supports everything, so with it ready every request goes there. Colibri
// (plain chat only) serves plain chat while llama.cpp is down; the rest waits for llama.
func TestRouterSendsStructuredOutputAndAdaptersToLlamaAndPlainChatToColibriWhenLlamaIsDown(t *testing.T) {
	a, b := echoServer("colibri"), echoServer("llama")
	defer a.Close()
	defer b.Close()
	var aReady, bReady atomic.Bool
	aReady.Store(true)
	bReady.Store(true)
	llama, colibri := tcpBackend("llama.cpp", b, &bReady, ""), tcpBackend("colibri", a, &aReady, "colibri-small")
	colibri.Supports = plainChatOnly
	r := NewRouter(nil, llama, colibri)
	r.OnDemandWait = 200 * time.Millisecond

	bodies := map[string]string{
		`{"model":"x","messages":[]}`:                                             "llama",
		`{"model":"x","response_format":{"type":"json_object"}}`:                  "llama",
		`{"model":"x","response_format":{"type":"json_schema","json_schema":{}}}`: "llama",
		`{"model":"x","response_format":{"type":"text"}}`:                         "llama",
		`{"model":"x","adapter":"skill-abc"}`:                                     "llama",
		`{"model":"x","tools":[{"type":"function"}]}`:                             "llama",
		`{"model":"x","tools":[]}`:                                                "llama",
		// A role-framed call is HEART/INFERENCER work: llama.cpp, always.
		`{"model":"x","messages":[],"knirv_stage":"heart.chat","knirv_frame_hash":"h"}`: "llama",
		`{"model":"x","knirv_stage":""}`:                                                "llama",
	}
	for body, want := range bodies {
		var got map[string]any
		json.Unmarshal(post(t, r, "/v1/chat/completions", body).Body.Bytes(), &got)
		if got["server"] != want {
			t.Errorf("%s served by %v, want %s", body, got["server"], want)
		}
	}
	var got map[string]any
	json.Unmarshal(post(t, r, "/v1/adapters", `{"name":"n","path":"/p"}`).Body.Bytes(), &got)
	if got["server"] != "llama" {
		t.Fatalf("adapter routes belong to llama.cpp: %v", got)
	}

	// llama.cpp down: plain chat goes to Colibri; a request only llama can serve waits
	// for it and then gets a 503 with states.
	bReady.Store(false)
	json.Unmarshal(post(t, r, "/v1/chat/completions", `{"model":"x","messages":[]}`).Body.Bytes(), &got)
	if got["server"] != "colibri" {
		t.Fatalf("plain chat must go to colibri while llama.cpp is down: %v", got)
	}
	rec := post(t, r, "/v1/chat/completions", `{"model":"x","response_format":{"type":"json_object"}}`)
	if rec.Code != http.StatusServiceUnavailable || !strings.Contains(rec.Body.String(), `"llama.cpp":false`) {
		t.Fatalf("structured output with llama.cpp down must be 503 with states: %d %s", rec.Code, rec.Body)
	}
}

// A request that needs llama.cpp while it is not ready waits for it (its Demand is
// called); plain chat is not held up by it and goes to Colibri meanwhile.
func TestRouterWaitsForLlamaForAStructuredRequest(t *testing.T) {
	a, b := echoServer("colibri"), echoServer("llama")
	defer a.Close()
	defer b.Close()
	var aReady, bReady atomic.Bool
	aReady.Store(true)
	llama, colibri := tcpBackend("llama.cpp", b, &bReady, ""), tcpBackend("colibri", a, &aReady, "")
	colibri.Supports = plainChatOnly
	var demands atomic.Int32
	llama.Demand = func(string) { demands.Add(1); go func() { time.Sleep(300 * time.Millisecond); bReady.Store(true) }() }
	r := NewRouter(nil, llama, colibri)

	start := time.Now()
	var got map[string]any
	json.Unmarshal(post(t, r, "/v1/chat/completions", `{"response_format":{"type":"json_object"}}`).Body.Bytes(), &got)
	if got["server"] != "llama" || time.Since(start) < 250*time.Millisecond {
		t.Fatalf("must wait for llama.cpp to come up and use it: %v after %s", got, time.Since(start))
	}
	// Plain chat during the wait goes to Colibri and does not wait on llama.cpp.
	bReady.Store(false)
	before := demands.Load()
	json.Unmarshal(post(t, r, "/v1/chat/completions", `{"messages":[]}`).Body.Bytes(), &got)
	if got["server"] != "colibri" || demands.Load() != before {
		t.Fatalf("plain chat must go to colibri while llama.cpp is down: %v", got)
	}
}

func TestRouterGivesUpWaitingWith503(t *testing.T) {
	a, b := echoServer("colibri"), echoServer("llama")
	defer a.Close()
	defer b.Close()
	var aReady, bReady atomic.Bool
	aReady.Store(true)
	llama, colibri := tcpBackend("llama.cpp", b, &bReady, ""), tcpBackend("colibri", a, &aReady, "")
	colibri.Supports = plainChatOnly
	r := NewRouter(nil, llama, colibri)
	r.OnDemandWait = 300 * time.Millisecond
	rec := post(t, r, "/v1/chat/completions", `{"adapter":"x"}`)
	if rec.Code != http.StatusServiceUnavailable || rec.Header().Get("Retry-After") == "" {
		t.Fatalf("got %d", rec.Code)
	}
}

// A llama.cpp that is out for good refuses the request at once instead of waiting.
func TestRouterRefusesAtOnceWhenTheNeededBackendIsGone(t *testing.T) {
	a, b := echoServer("colibri"), echoServer("llama")
	defer a.Close()
	defer b.Close()
	var aReady, bReady atomic.Bool
	aReady.Store(true)
	llama, colibri := tcpBackend("llama.cpp", b, &bReady, ""), tcpBackend("colibri", a, &aReady, "")
	colibri.Supports = plainChatOnly
	llama.Gone = func() bool { return true }
	llama.Demand = func(string) {}
	r := NewRouter(nil, llama, colibri)
	r.OnDemandWait = 10 * time.Second
	start := time.Now()
	rec := post(t, r, "/v1/chat/completions", `{"tools":[{"type":"function"}]}`)
	if rec.Code != http.StatusServiceUnavailable || time.Since(start) > 2*time.Second {
		t.Fatalf("got %d after %s", rec.Code, time.Since(start))
	}
}

func TestRouterForwardsToTheActiveBackendWhenNothingSupportsTheRequest(t *testing.T) {
	a := echoServer("colibri")
	defer a.Close()
	var ready atomic.Bool
	ready.Store(true)
	colibri := tcpBackend("colibri", a, &ready, "")
	colibri.Supports = plainChatOnly
	var got map[string]any
	json.Unmarshal(post(t, NewRouter(nil, colibri), "/v1/chat/completions", `{"response_format":{"type":"json_object"}}`).Body.Bytes(), &got)
	if got["server"] != "colibri" {
		t.Fatalf("%v", got)
	}
}

func TestRouterAnswersHealthAndAdapterListingItselfWhenLlamaIsStopped(t *testing.T) {
	a := echoServer("colibri")
	defer a.Close()
	var aReady, bReady atomic.Bool
	colibri := tcpBackend("colibri", a, &aReady, "")
	colibri.Supports = plainChatOnly
	llama := tcpBackend("llama.cpp", a, &bReady, "")
	var demands atomic.Int32
	llama.Demand = func(string) { demands.Add(1) }
	r := NewRouter(nil, llama, colibri)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("health with nothing ready: %d", rec.Code)
	}
	aReady.Store(true)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health", nil))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"backend":"colibri"`) {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}

	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/v1/adapters", nil))
	if rec.Code != http.StatusOK || demands.Load() != 0 || !strings.Contains(rec.Body.String(), `"adapters":[]`) {
		t.Fatalf("listing must not wait on llama.cpp: %d %s demands=%d", rec.Code, rec.Body, demands.Load())
	}
}

func postRole(t *testing.T, h http.Handler, role, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(RoleHeader, role)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// Teacher output is training data: it must come from the strong model or not at all.
// Teacher output is training data: it comes from llama.cpp, and from Colibri only once
// llama.cpp is out for good. A teacher request is never handed to the other backend
// while the teacher is down.
func TestTeacherRequestsGoToLlamaUntilItIsOutForGood(t *testing.T) {
	a, b := echoServer("colibri"), echoServer("llama")
	defer a.Close()
	defer b.Close()
	var aReady, bReady, out atomic.Bool
	llama := tcpBackend("llama.cpp", b, &bReady, "qwen")
	colibri := tcpBackend("colibri", a, &aReady, "colibri-small")
	llama.Teacher = func() bool { return !out.Load() }
	llama.Gone = out.Load
	colibri.Teacher = out.Load
	r := NewRouter(nil, llama, colibri)

	// llama.cpp down, colibri up: not the teacher, so the request is refused.
	aReady.Store(true)
	if rec := postRole(t, r, "teacher", `{"model":"x"}`); rec.Code != http.StatusServiceUnavailable || rec.Header().Get("Retry-After") == "" || strings.Contains(rec.Body.String(), `"server":"colibri"`) {
		t.Fatalf("a teacher request must not go to colibri while llama.cpp is the teacher: %d %s", rec.Code, rec.Body)
	}

	bReady.Store(true)
	rec := postRole(t, r, "teacher", `{"model":"x"}`)
	var got map[string]any
	json.Unmarshal(rec.Body.Bytes(), &got)
	if got["server"] != "llama" || got["model"] != "qwen" || rec.Header().Get(BackendHeader) != "llama.cpp" || rec.Header().Get(ModelHeader) != "qwen" {
		t.Fatalf("llama.cpp must answer teacher requests: %v %v", got, rec.Header())
	}

	// llama.cpp out for good: colibri becomes the teacher.
	bReady.Store(false)
	out.Store(true)
	rec = postRole(t, r, "teacher", `{"model":"x"}`)
	got = nil
	json.Unmarshal(rec.Body.Bytes(), &got)
	if got["server"] != "colibri" || got["model"] != "colibri-small" || rec.Header().Get(BackendHeader) != "colibri" {
		t.Fatalf("colibri must be the teacher once llama.cpp is out for good: %d %v %v", rec.Code, got, rec.Header())
	}
	// Ordinary traffic still sees the headers.
	if h := post(t, r, "/v1/chat/completions", `{"model":"x"}`).Header().Get(BackendHeader); h != "colibri" {
		t.Fatalf("backend header %q", h)
	}
}

func TestRoleFrameNeedIsDetectedFromHeaderAndBody(t *testing.T) {
	mk := func(hdr string) *http.Request {
		r := httptest.NewRequest("POST", "/v1/chat/completions", nil)
		if hdr != "" {
			r.Header.Set("X-KNIRV-Stage", hdr)
		}
		return r
	}
	if !detectNeeds(mk("heart.chat"), nil).Frame {
		t.Fatal("header not detected")
	}
	if detectNeeds(mk(""), nil).Frame {
		t.Fatal("unframed request flagged")
	}
	if !detectNeeds(mk(""), map[string]json.RawMessage{"knirv_stage": json.RawMessage(`"x"`)}).Frame {
		t.Fatal("body field not detected")
	}
	if !(Needs{Frame: true}).Any() || (Needs{Frame: true}).String() != "a role frame" {
		t.Fatal("Any/String")
	}
}

func TestTeacherRequestsAreNotTreatedAsFramedWork(t *testing.T) {
	r := httptest.NewRequest("POST", "/v1/chat/completions", nil)
	r.Header.Set(RoleHeader, RoleTeacher)
	r.Header.Set("X-KNIRV-Stage", "heart.chat")
	if detectNeeds(r, nil).Frame {
		t.Fatal("a teacher request must not be pinned to llama.cpp by a frame")
	}
}
