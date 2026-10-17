package knirvllama

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

// TestMain makes the test binary double as a fake `knirvllama`: with
// KNIRV_FAKE_KNIRVLLAMA=1 it parses -listen, optionally crashes the first N starts
// (counted in FAKE_KL_STATE), then serves /health (503 while "loading", then 200).
func TestMain(m *testing.M) {
	if os.Getenv("KNIRV_FAKE_KNIRVLLAMA") == "1" {
		runFakeKnirvllama()
		return
	}
	os.Exit(m.Run())
}

func runFakeKnirvllama() {
	listen := ""
	for i, a := range os.Args {
		if a == "-listen" && i+1 < len(os.Args) {
			listen = os.Args[i+1]
		}
	}
	if n, _ := strconv.Atoi(os.Getenv("FAKE_KL_FAIL_FIRST")); n > 0 {
		state := os.Getenv("FAKE_KL_STATE")
		b, _ := os.ReadFile(state)
		seen, _ := strconv.Atoi(strings.TrimSpace(string(b)))
		if seen < n {
			os.WriteFile(state, []byte(strconv.Itoa(seen+1)), 0o644)
			fmt.Fprintln(os.Stderr, "boom: llama-server did not become healthy within 2m0s")
			os.Exit(1)
		}
	}
	if os.Getenv("FAKE_KL_CRASH_ALWAYS") == "1" {
		fmt.Fprintln(os.Stderr, "boom: out of memory")
		os.Exit(1)
	}
	if n, _ := strconv.Atoi(os.Getenv("FAKE_KL_DIE_AFTER_START")); n > 0 {
		// Serve briefly, then die: the first N runs crash after becoming healthy.
		state := os.Getenv("FAKE_KL_STATE") + ".runs"
		b, _ := os.ReadFile(state)
		seen, _ := strconv.Atoi(strings.TrimSpace(string(b)))
		if seen < n {
			os.WriteFile(state, []byte(strconv.Itoa(seen+1)), 0o644)
			go func() { time.Sleep(700 * time.Millisecond); fmt.Fprintln(os.Stderr, "boom: killed"); os.Exit(2) }()
		}
	}
	load, _ := strconv.Atoi(os.Getenv("FAKE_KL_LOAD_MS"))
	never := os.Getenv("FAKE_KL_NEVER") == "1"
	start := time.Now()
	fmt.Println("model loading")
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		if never || time.Since(start) < time.Duration(load)*time.Millisecond {
			http.Error(w, "loading", http.StatusServiceUnavailable)
			return
		}
		w.Write([]byte(`{"status":"ok"}`))
	})
	_ = http.ListenAndServe(listen, mux)
}

type lockedBuf struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (l *lockedBuf) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.Write(p)
}
func (l *lockedBuf) String() string { l.mu.Lock(); defer l.mu.Unlock(); return l.b.String() }

func freeTCP(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("no local listeners: %v", err)
	}
	defer l.Close()
	return l.Addr().String()
}

type harness struct {
	m    *Manager
	logs *lockedBuf
	out  *lockedBuf
}

// newHarness builds a Manager whose knirvllama is this test binary.
func newHarness(t *testing.T, cfg ManagerConfig, env ...string) *harness {
	t.Helper()
	killStale = func(string) {} // pkill -x would match, and kill, this very test binary
	t.Cleanup(func() { killStale = killStaleLlama })
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("KNIRV_LLAMA_BINARY_PATH", exe)
	t.Setenv("KNIRV_LLAMA_BINARY_DIR", t.TempDir())
	t.Setenv("KNIRV_APP_DATA_DIR", t.TempDir())
	t.Setenv("KNIRV_FAKE_KNIRVLLAMA", "1")
	t.Setenv("FAKE_KL_STATE", filepath.Join(t.TempDir(), "state"))
	for _, e := range env {
		k, v, _ := strings.Cut(e, "=")
		t.Setenv(k, v)
	}
	h := &harness{logs: &lockedBuf{}, out: &lockedBuf{}}
	cfg.ListenAddr = freeTCP(t)
	cfg.DataDir = t.TempDir()
	cfg.Stdout, cfg.Stderr = h.out, h.out
	cfg.Logf = func(f string, a ...any) { fmt.Fprintf(h.logs, f+"\n", a...) }
	if cfg.RetryBackoff == nil {
		cfg.RetryBackoff = []time.Duration{20 * time.Millisecond}
	}
	h.m = NewManager(&cfg, nil)
	t.Cleanup(func() { h.m.Stop(context.Background()) })
	return h
}

func processGone(pid int) bool {
	err := syscall.Kill(pid, 0)
	return errors.Is(err, syscall.ESRCH)
}

func TestStartSucceedsAfterSlowLoadAndStopReapsChild(t *testing.T) {
	h := newHarness(t, ManagerConfig{StartTimeout: 20 * time.Second}, "FAKE_KL_LOAD_MS=1500")
	if err := h.m.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	st := h.m.GetStatus()
	if !st.Running || st.Phase != "running" || st.Attempt != 1 {
		t.Fatalf("%+v", st)
	}
	pid := h.m.GetPID()
	if err := h.m.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	time.Sleep(100 * time.Millisecond)
	if !processGone(pid) {
		t.Fatal("stopped child is still present (zombie?)")
	}
	if h.m.GetStatus().Phase != "stopped" {
		t.Fatalf("%+v", h.m.GetStatus())
	}
	if !strings.Contains(h.out.String(), "[KNIRVLLAMA] model loading") {
		t.Fatalf("child output must be forwarded with a prefix:\n%s", h.out.String())
	}
}

// The reported failure: the first attempt dies, the second (warm) one works.
func TestStartRetriesAfterEarlyExit(t *testing.T) {
	h := newHarness(t, ManagerConfig{StartTimeout: 20 * time.Second}, "FAKE_KL_FAIL_FIRST=1")
	if err := h.m.Start(context.Background()); err != nil {
		t.Fatalf("a crash on the first attempt must be retried: %v", err)
	}
	st := h.m.GetStatus()
	if st.Attempt != 2 || !st.Running {
		t.Fatalf("%+v", st)
	}
	logs := h.logs.String()
	if !strings.Contains(logs, "retrying in") || !strings.Contains(logs, "boom") {
		t.Fatalf("the failure and the retry must be logged with the child's last output:\n%s", logs)
	}
}

// Regression for the 35-minute silent hang: a dead child is reported at once.
func TestStartReportsCrashFastWithChildOutput(t *testing.T) {
	h := newHarness(t, ManagerConfig{StartTimeout: time.Minute, MaxStartAttempts: 3}, "FAKE_KL_CRASH_ALWAYS=1")
	t0 := time.Now()
	err := h.m.Start(context.Background())
	if time.Since(t0) > 15*time.Second {
		t.Fatalf("took %s: must not wait out StartTimeout", time.Since(t0))
	}
	var se *StartError
	if !errors.As(err, &se) || !se.Exited || se.Attempt != 3 {
		t.Fatalf("want StartError exited on attempt 3, got %#v", err)
	}
	if !strings.Contains(err.Error(), "boom: out of memory") {
		t.Fatalf("error must carry the child's last output: %v", err)
	}
	st := h.m.GetStatus()
	if st.Running || st.Phase != "failed" || !strings.Contains(st.LastError, "boom") {
		t.Fatalf("%+v", st)
	}
	if got := strings.Count(h.logs.String(), "retrying in"); got != 2 {
		t.Fatalf("expected 2 retries, got %d", got)
	}
}

func TestHealthTimeoutIsNotRetriedAndKillsChild(t *testing.T) {
	h := newHarness(t, ManagerConfig{StartTimeout: time.Second, MaxStartAttempts: 3}, "FAKE_KL_NEVER=1")
	err := h.m.Start(context.Background())
	var se *StartError
	if !errors.As(err, &se) || !se.Timeout || se.Attempt != 1 {
		t.Fatalf("want a single timed-out attempt, got %#v", err)
	}
	time.Sleep(100 * time.Millisecond)
	if !processGone(h.m.GetPID()) {
		t.Fatal("a child that never became healthy must be killed")
	}
}

func TestMissingBinaryIsPermanent(t *testing.T) {
	h := newHarness(t, ManagerConfig{MaxStartAttempts: 3})
	t.Setenv("KNIRV_LLAMA_BINARY_PATH", "/nonexistent/knirvllama")
	t.Setenv("KNIRV_LLAMA_BINARY_DIR", "/nonexistent")
	h.m.config.BinaryPath = "/nonexistent/knirvllama"
	// extraction of the embedded binary cannot land anywhere, and no candidate exists
	err := h.m.Start(context.Background())
	if err == nil {
		t.Skip("embedded binary resolved on this machine")
	}
	if strings.Count(h.logs.String(), "retrying") != 0 {
		t.Fatalf("a missing binary must not be retried:\n%s", h.logs.String())
	}
}

func TestContextCancelDuringBackoffReturnsPromptly(t *testing.T) {
	h := newHarness(t, ManagerConfig{StartTimeout: time.Minute, RetryBackoff: []time.Duration{time.Hour}}, "FAKE_KL_CRASH_ALWAYS=1")
	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(500*time.Millisecond, cancel)
	t0 := time.Now()
	err := h.m.Start(ctx)
	if !errors.Is(err, context.Canceled) || time.Since(t0) > 10*time.Second {
		t.Fatalf("err=%v after %s", err, time.Since(t0))
	}
}

func TestStaleSocketIsRemovedBeforeStart(t *testing.T) {
	sock := filepath.Join(t.TempDir(), "llama.sock")
	l, err := net.Listen("unix", sock)
	if err != nil {
		t.Skip(err)
	}
	l.(*net.UnixListener).SetUnlinkOnClose(false)
	l.Close() // leaves the file behind, exactly like a killed process
	if _, err := os.Stat(sock); err != nil {
		t.Fatal("setup: stale socket file missing")
	}
	if err := removeStaleSocket(sock); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(sock); !os.IsNotExist(err) {
		t.Fatal("stale socket must be removed")
	}
}

func TestLiveSocketIsNotRemoved(t *testing.T) {
	sock := filepath.Join(t.TempDir(), "live.sock")
	l, err := net.Listen("unix", sock)
	if err != nil {
		t.Skip(err)
	}
	defer l.Close()
	go func() {
		for {
			c, err := l.Accept()
			if err != nil {
				return
			}
			c.Close()
		}
	}()
	if err := removeStaleSocket(sock); err == nil || !strings.Contains(err.Error(), "live process") {
		t.Fatalf("got %v", err)
	}
	if _, err := os.Stat(sock); err != nil {
		t.Fatal("a live process's socket must be left alone")
	}
}

func TestRemoveStaleSocketRefusesRegularFile(t *testing.T) {
	p := filepath.Join(t.TempDir(), "x")
	os.WriteFile(p, []byte("data"), 0o644)
	if err := removeStaleSocket(p); err == nil {
		t.Fatal("must not delete a regular file")
	}
	if err := removeStaleSocket(filepath.Join(t.TempDir(), "absent")); err != nil {
		t.Fatalf("absent path is fine: %v", err)
	}
}

func TestLineWriterPrefixesSplitsOnCRAndRecordsTail(t *testing.T) {
	var out bytes.Buffer
	ring := newTailRing(3)
	w := &lineWriter{prefix: "[P] ", out: &out, ring: ring}
	w.Write([]byte("one\ntwo\rthr"))
	w.Write([]byte("ee\nfour\nfive"))
	w.Flush()
	want := "[P] one\n[P] two\n[P] three\n[P] four\n[P] five\n"
	if out.String() != want {
		t.Fatalf("got %q want %q", out.String(), want)
	}
	if got := ring.tail(10); strings.Join(got, ",") != "three,four,five" {
		t.Fatalf("ring keeps only the last 3: %v", got)
	}
}

func TestBackoffSchedule(t *testing.T) {
	s := []time.Duration{5 * time.Second, 30 * time.Second}
	if backoffFor(s, 1) != 5*time.Second || backoffFor(s, 2) != 30*time.Second || backoffFor(s, 9) != 30*time.Second || backoffFor(nil, 1) != 0 {
		t.Fatal("schedule")
	}
}

func TestDefaultsAddRetryPolicy(t *testing.T) {
	m := NewManager(&ManagerConfig{}, nil)
	if m.config.MaxStartAttempts != 3 || len(m.config.RetryBackoff) != 2 {
		t.Fatalf("%+v", m.config)
	}
}

func fastSupervise(cfg ManagerConfig) ManagerConfig {
	cfg.SuperviseBackoff = []time.Duration{50 * time.Millisecond}
	cfg.StableAfter = time.Hour
	return cfg
}

// A start that exhausts its own retries is tried again by Supervise.
func TestSuperviseRestartsAfterStartFailure(t *testing.T) {
	// Attempts 1-2 crash inside the first Start (MaxStartAttempts=2); Supervise starts again.
	h := newHarness(t, fastSupervise(ManagerConfig{StartTimeout: 20 * time.Second, MaxStartAttempts: 2}), "FAKE_KL_FAIL_FIRST=2")
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- h.m.Supervise(ctx) }()
	deadline := time.Now().Add(30 * time.Second)
	for !h.m.IsRunning() && time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
	}
	if !h.m.IsRunning() {
		t.Fatalf("never came up:\n%s", h.logs.String())
	}
	if !strings.Contains(h.logs.String(), "restarting in") {
		t.Fatalf("restart not logged:\n%s", h.logs.String())
	}
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("err=%v", err)
	}
}

// A crash after a successful start is restarted too (nothing did that before).
func TestSuperviseRestartsAfterCrashWhileRunning(t *testing.T) {
	h := newHarness(t, fastSupervise(ManagerConfig{StartTimeout: 20 * time.Second}), "FAKE_KL_DIE_AFTER_START=1")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go h.m.Supervise(ctx)
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		if strings.Contains(h.logs.String(), "exited unexpectedly") && strings.Contains(h.logs.String(), "restarting in") && h.m.IsRunning() {
			if st := h.m.GetStatus(); st.Phase == "running" {
				return
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("not restarted after a crash:\n%s", h.logs.String())
}

func TestSuperviseStopsOnPermanentFailure(t *testing.T) {
	h := newHarness(t, fastSupervise(ManagerConfig{}))
	resolveBinary = func(string) (string, error) { return "", errors.New("no such binary") }
	t.Cleanup(func() { resolveBinary = resolveLlamaBinaryPath })
	done := make(chan error, 1)
	go func() { done <- h.m.Supervise(context.Background()) }()
	select {
	case err := <-done:
		if err == nil || !strings.Contains(err.Error(), "binary not found") {
			t.Fatalf("err=%v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("a missing binary must end supervision, not loop")
	}
	if strings.Contains(h.logs.String(), "restarting in") {
		t.Fatalf("must not restart:\n%s", h.logs.String())
	}
}

func TestSuperviseCancelDuringBackoff(t *testing.T) {
	h := newHarness(t, ManagerConfig{StartTimeout: time.Minute, MaxStartAttempts: 1, SuperviseBackoff: []time.Duration{time.Hour}}, "FAKE_KL_CRASH_ALWAYS=1")
	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(700*time.Millisecond, cancel)
	t0 := time.Now()
	if err := h.m.Supervise(ctx); !errors.Is(err, context.Canceled) || time.Since(t0) > 10*time.Second {
		t.Fatalf("err=%v after %s", err, time.Since(t0))
	}
}
