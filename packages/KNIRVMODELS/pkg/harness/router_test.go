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

func TestRouterPrefersFirstReadyBackendAndRewritesModel(t *testing.T) {
	a, b := echoServer("colibri"), echoServer("llama")
	defer a.Close()
	defer b.Close()
	var aReady, bReady atomic.Bool
	r := NewRouter(nil, tcpBackend("colibri", a, &aReady, "colibri-small"), tcpBackend("llama.cpp", b, &bReady, ""))

	if rec := post(t, r, "/v1/chat/completions", `{"model":"x"}`); rec.Code != http.StatusServiceUnavailable || rec.Header().Get("Retry-After") == "" {
		t.Fatalf("nothing ready must be 503 with Retry-After, got %d", rec.Code)
	}

	bReady.Store(true)
	rec := post(t, r, "/v1/chat/completions", `{"model":"local-llama","messages":[]}`)
	var got map[string]any
	json.Unmarshal(rec.Body.Bytes(), &got)
	if got["server"] != "llama" || got["model"] != "local-llama" {
		t.Fatalf("fallback must serve untouched: %v", got)
	}

	aReady.Store(true)
	rec = post(t, r, "/v1/chat/completions", `{"model":"local-llama","messages":[]}`)
	got = nil
	json.Unmarshal(rec.Body.Bytes(), &got)
	if got["server"] != "colibri" || got["model"] != "colibri-small" || got["path"] != "/v1/chat/completions" {
		t.Fatalf("primary must serve with its own model id: %v", got)
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
func plainChatOnly(n Needs) bool { return !n.Adapter && !n.Grammar && !n.Tools }

func TestRouterSendsStructuredOutputAndAdaptersToTheBackendThatSupportsThem(t *testing.T) {
	a, b := echoServer("colibri"), echoServer("llama")
	defer a.Close()
	defer b.Close()
	var aReady, bReady atomic.Bool
	aReady.Store(true)
	bReady.Store(true)
	colibri, llama := tcpBackend("colibri", a, &aReady, "colibri-small"), tcpBackend("llama.cpp", b, &bReady, "")
	colibri.Supports = plainChatOnly
	r := NewRouter(nil, colibri, llama)

	for body, want := range map[string]string{
		`{"model":"x","messages":[]}`:                                             "colibri",
		`{"model":"x","response_format":{"type":"json_object"}}`:                  "llama",
		`{"model":"x","response_format":{"type":"json_schema","json_schema":{}}}`: "llama",
		`{"model":"x","response_format":{"type":"text"}}`:                         "colibri",
		`{"model":"x","adapter":"skill-abc"}`:                                     "llama",
		`{"model":"x","tools":[{"type":"function"}]}`:                             "llama",
		`{"model":"x","tools":[]}`:                                                "colibri",
	} {
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
}

func TestRouterStartsAnOnDemandBackendAndWaitsForIt(t *testing.T) {
	a, b := echoServer("colibri"), echoServer("llama")
	defer a.Close()
	defer b.Close()
	var aReady, bReady atomic.Bool
	aReady.Store(true)
	colibri, llama := tcpBackend("colibri", a, &aReady, ""), tcpBackend("llama.cpp", b, &bReady, "")
	colibri.Supports = plainChatOnly
	var demands atomic.Int32
	llama.Demand = func(string) { demands.Add(1); go func() { time.Sleep(300 * time.Millisecond); bReady.Store(true) }() }
	r := NewRouter(nil, colibri, llama)

	start := time.Now()
	var got map[string]any
	json.Unmarshal(post(t, r, "/v1/chat/completions", `{"response_format":{"type":"json_object"}}`).Body.Bytes(), &got)
	if got["server"] != "llama" || time.Since(start) < 250*time.Millisecond {
		t.Fatalf("must wait for llama.cpp to come up and use it: %v after %s", got, time.Since(start))
	}
	// A plain request keeps going to Colibri and does not touch llama.cpp.
	before := demands.Load()
	json.Unmarshal(post(t, r, "/v1/chat/completions", `{"messages":[]}`).Body.Bytes(), &got)
	if got["server"] != "colibri" || demands.Load() != before {
		t.Fatalf("plain chat must stay on colibri: %v", got)
	}
}

func TestRouterGivesUpWaitingWith503(t *testing.T) {
	a, b := echoServer("colibri"), echoServer("llama")
	defer a.Close()
	defer b.Close()
	var aReady, bReady atomic.Bool
	aReady.Store(true)
	colibri, llama := tcpBackend("colibri", a, &aReady, ""), tcpBackend("llama.cpp", b, &bReady, "")
	colibri.Supports = plainChatOnly
	r := NewRouter(nil, colibri, llama)
	r.OnDemandWait = 300 * time.Millisecond
	rec := post(t, r, "/v1/chat/completions", `{"adapter":"x"}`)
	if rec.Code != http.StatusServiceUnavailable || rec.Header().Get("Retry-After") == "" {
		t.Fatalf("got %d", rec.Code)
	}
}

// With no backend able to do what the request needs, the active one gets it and
// answers (or refuses) in its own words rather than the router inventing an outage.
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
	r := NewRouter(nil, colibri, llama)

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
		t.Fatalf("listing must not start llama.cpp: %d %s demands=%d", rec.Code, rec.Body, demands.Load())
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
func TestTeacherRequestsAreServedOnlyByTheTeacherBackend(t *testing.T) {
	a, b := echoServer("colibri"), echoServer("llama")
	defer a.Close()
	defer b.Close()
	var aReady, bReady atomic.Bool
	var teaching atomic.Bool
	bReady.Store(true)
	colibri, llama := tcpBackend("colibri", a, &aReady, "colibri-small"), tcpBackend("llama.cpp", b, &bReady, "")
	colibri.Teacher = teaching.Load
	r := NewRouter(nil, colibri, llama)

	rec := postRole(t, r, "teacher", `{"model":"x"}`)
	if rec.Code != http.StatusServiceUnavailable || rec.Header().Get("Retry-After") == "" || strings.Contains(rec.Body.String(), `"server":"llama"`) {
		t.Fatalf("with colibri down a teacher request must be refused, not answered by llama.cpp: %d %s", rec.Code, rec.Body)
	}
	aReady.Store(true) // loaded, but not (yet) qualified
	if rec = postRole(t, r, "teacher", `{"model":"x"}`); rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("an unqualified colibri must not teach: %d", rec.Code)
	}
	teaching.Store(true)
	rec = postRole(t, r, "teacher", `{"model":"x"}`)
	var got map[string]any
	json.Unmarshal(rec.Body.Bytes(), &got)
	if got["server"] != "colibri" || got["model"] != "colibri-small" {
		t.Fatalf("%v", got)
	}
	if rec.Header().Get(BackendHeader) != "colibri" || rec.Header().Get(ModelHeader) != "colibri-small" {
		t.Fatalf("provenance headers missing: %v", rec.Header())
	}
	// Ordinary traffic still sees the headers.
	if h := post(t, r, "/v1/chat/completions", `{"model":"x"}`).Header().Get(BackendHeader); h != "colibri" {
		t.Fatalf("backend header %q", h)
	}
}
