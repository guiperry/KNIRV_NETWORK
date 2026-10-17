package runtime

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestMain doubles as a fake llama-server: when KNIRV_FAKE_LLAMA is set the test
// binary serves /health on --port the way llama-server does (503 while "loading the
// model", then 200), or exits early, so Supervisor is tested against a real child.
func TestMain(m *testing.M) {
	if os.Getenv("KNIRV_FAKE_LLAMA") == "1" {
		runFakeLlama()
		return
	}
	os.Exit(m.Run())
}

func runFakeLlama() {
	port := ""
	for i, a := range os.Args {
		if a == "--port" && i+1 < len(os.Args) {
			port = os.Args[i+1]
		}
	}
	ms := func(k string) time.Duration {
		n, _ := strconv.Atoi(os.Getenv(k))
		return time.Duration(n) * time.Millisecond
	}
	if d := ms("FAKE_EXIT_MS"); d > 0 {
		time.Sleep(d)
		os.Exit(3)
	}
	start := time.Now()
	load := ms("FAKE_LOAD_MS")
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		if time.Since(start) < load {
			http.Error(w, `{"error":{"message":"Loading model"}}`, http.StatusServiceUnavailable)
			return
		}
		w.Write([]byte(`{"status":"ok"}`))
	})
	_ = http.ListenAndServe("127.0.0.1:"+port, mux)
}

func freeAddr(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("no local listeners: %v", err)
	}
	defer l.Close()
	return l.Addr().String()
}

// fakeChild returns a Supervisor whose "llama-server" is this test binary.
func fakeChild(t *testing.T, env ...string) *Supervisor {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("KNIRV_FAKE_LLAMA", "1")
	for _, e := range env {
		k, v, _ := strings.Cut(e, "=")
		t.Setenv(k, v)
	}
	model := writeGGUF(t, 1024)
	return &Supervisor{Path: exe, Model: model, Address: freeAddr(t), NoWarm: true}
}

func writeGGUF(t *testing.T, size int) string {
	t.Helper()
	p := t.TempDir() + "/m.gguf"
	b := make([]byte, size)
	copy(b, "GGUF")
	if err := os.WriteFile(p, b, 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// A model that loads slowly (503 first) must succeed within the deadline.
func TestWaitHealthySlowLoadSucceedsAndReportsProgress(t *testing.T) {
	var ready atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if !ready.Load() {
			http.Error(w, "loading", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(200)
	}))
	defer srv.Close()
	time.AfterFunc(1200*time.Millisecond, func() { ready.Store(true) })

	var mu sync.Mutex
	var lines []string
	err := WaitHealthy(context.Background(), strings.TrimPrefix(srv.URL, "http://"), nil, WaitOptions{
		Timeout: 10 * time.Second, ProgressEvery: 300 * time.Millisecond,
		Logf: func(f string, a ...any) { mu.Lock(); lines = append(lines, fmt.Sprintf(f, a...)); mu.Unlock() },
	})
	if err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	defer mu.Unlock()
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "loading model (HTTP 503)") || !strings.Contains(joined, "healthy after") {
		t.Fatalf("expected progress then success lines, got:\n%s", joined)
	}
}

func TestWaitHealthyReturnsImmediatelyWhenChildExits(t *testing.T) {
	exited := make(chan struct{})
	close(exited)
	t0 := time.Now()
	err := WaitHealthy(context.Background(), freeAddr(t), exited, WaitOptions{Timeout: time.Minute})
	if !errors.Is(err, ErrExited) {
		t.Fatalf("want ErrExited, got %v", err)
	}
	if time.Since(t0) > 2*time.Second {
		t.Fatalf("took %s: a dead child must not wait for the deadline", time.Since(t0))
	}
}

func TestWaitHealthyTimeoutNamesLastState(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "loading", http.StatusServiceUnavailable)
	}))
	defer srv.Close()
	err := WaitHealthy(context.Background(), strings.TrimPrefix(srv.URL, "http://"), nil, WaitOptions{Timeout: 400 * time.Millisecond})
	if err == nil || !strings.Contains(err.Error(), "loading model (HTTP 503)") {
		t.Fatalf("got %v", err)
	}
}

func TestWaitHealthyHonoursContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(200*time.Millisecond, cancel)
	err := WaitHealthy(ctx, freeAddr(t), nil, WaitOptions{Timeout: time.Minute})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v", err)
	}
}

func TestHealthTimeoutFromEnv(t *testing.T) {
	t.Setenv("KNIRV_LLAMA_HEALTH_TIMEOUT", "")
	if HealthTimeoutFromEnv() != DefaultHealthTimeout {
		t.Fatal("default")
	}
	t.Setenv("KNIRV_LLAMA_HEALTH_TIMEOUT", "7m")
	if HealthTimeoutFromEnv() != 7*time.Minute {
		t.Fatal("override")
	}
	t.Setenv("KNIRV_LLAMA_HEALTH_TIMEOUT", "garbage")
	if HealthTimeoutFromEnv() != DefaultHealthTimeout {
		t.Fatal("invalid falls back")
	}
	if DefaultHealthTimeout <= 2*time.Minute {
		t.Fatal("the old 120s deadline is what failed cold starts")
	}
}

// The failure that motivated this: a model slower than the old 120s deadline. Scaled
// down, a load longer than the OLD style fixed wait but inside HealthTimeout must pass.
func TestSupervisorSlowLoadWithinTimeoutSucceeds(t *testing.T) {
	s := fakeChild(t, "FAKE_LOAD_MS=1500")
	s.HealthTimeout = 20 * time.Second
	defer s.Stop()
	if err := s.Start(context.Background(), nil); err != nil {
		t.Fatalf("slow but alive child must start: %v", err)
	}
}

func TestSupervisorChildExitIsReportedFast(t *testing.T) {
	s := fakeChild(t, "FAKE_EXIT_MS=300")
	s.HealthTimeout = time.Minute
	t0 := time.Now()
	err := s.Start(context.Background(), nil)
	if !errors.Is(err, ErrExited) {
		t.Fatalf("want ErrExited, got %v", err)
	}
	if time.Since(t0) > 10*time.Second {
		t.Fatalf("exit took %s to report", time.Since(t0))
	}
}

func TestSupervisorTimeoutStopsChild(t *testing.T) {
	s := fakeChild(t, "FAKE_LOAD_MS=60000")
	s.HealthTimeout = time.Second
	err := s.Start(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "did not become healthy") {
		t.Fatalf("got %v", err)
	}
	if s.child != nil {
		t.Fatal("child must be cleared after a timeout")
	}
}

func TestMlockFlag(t *testing.T) {
	if got := (Options{Mlock: true}).args("m", "1"); !contains(got, "--mlock") {
		t.Fatalf("%v", got)
	}
	if got := (Options{}).args("m", "1"); contains(got, "--mlock") {
		t.Fatalf("mlock must be opt-in: %v", got)
	}
}
