package knirvmodels

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

// TestMain makes the test binary double as the knirvmodels binary in inference mode:
// it records each start in FAKE_INF_STATE, optionally exits at once for the first N
// starts, otherwise waits for SIGTERM.
func TestMain(m *testing.M) {
	if os.Getenv("KNIRV_FAKE_INFERENCE") == "1" {
		state := os.Getenv("FAKE_INF_STATE")
		b, _ := os.ReadFile(state)
		n, _ := strconv.Atoi(strings.TrimSpace(string(b)))
		os.WriteFile(state, []byte(strconv.Itoa(n+1)), 0o644)
		os.WriteFile(state+".args", []byte(strings.Join(os.Args[1:], " ")), 0o644)
		if crash, _ := strconv.Atoi(os.Getenv("FAKE_INF_CRASH_FIRST")); n < crash {
			os.Exit(3)
		}
		ch := make(chan os.Signal, 1)
		sigs(ch)
		<-ch
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func sigs(ch chan os.Signal) { signalNotify(ch, syscall.SIGTERM, syscall.SIGINT) }

type syncBuf struct {
	mu sync.Mutex
	b  strings.Builder
}

func (s *syncBuf) logf(f string, a ...any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.b.WriteString(strings.TrimSpace(strings.ReplaceAll(f, "%", "")) + "\n")
}
func (s *syncBuf) String() string { s.mu.Lock(); defer s.mu.Unlock(); return s.b.String() }

func newInference(t *testing.T, env ...string) (*Inference, string, *syncBuf) {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	state := filepath.Join(t.TempDir(), "state")
	logs := &syncBuf{}
	inf := NewInference(InferenceConfig{
		BinaryPath:     exe,
		SocketPath:     filepath.Join(t.TempDir(), "sockets", "llama.sock"),
		Env:            append([]string{"KNIRV_FAKE_INFERENCE=1", "FAKE_INF_STATE=" + state}, env...),
		RestartBackoff: []time.Duration{20 * time.Millisecond},
		StopTimeout:    5 * time.Second,
		Logf:           logs.logf,
	})
	return inf, state, logs
}

func readCount(state string) int {
	b, _ := os.ReadFile(state)
	n, _ := strconv.Atoi(strings.TrimSpace(string(b)))
	return n
}

func TestInferenceStartsChildWithInferenceFlags(t *testing.T) {
	inf, state, _ := newInference(t)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- inf.Run(ctx) }()
	waitFor(t, "child start", func() bool { return readCount(state) == 1 && inf.PID() != 0 })
	args, _ := os.ReadFile(state + ".args")
	if !strings.Contains(string(args), "--inference") || !strings.Contains(string(args), "--inference-socket "+inf.SocketPath()) {
		t.Fatalf("args %q", args)
	}
	pid := inf.PID()
	cancel()
	select {
	case err := <-done:
		if err != context.Canceled {
			t.Fatalf("err=%v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Run did not return on cancel")
	}
	if err := syscall.Kill(pid, 0); err == nil {
		t.Fatal("child must be stopped with the context")
	}
}

// The reason this exists: a child that dies is started again.
func TestInferenceRestartsChildThatExits(t *testing.T) {
	inf, state, logs := newInference(t, "FAKE_INF_CRASH_FIRST=2")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go inf.Run(ctx)
	waitFor(t, "third start to stay up", func() bool { return readCount(state) == 3 && inf.PID() != 0 })
	if got := strings.Count(logs.String(), "restarting in"); got != 2 {
		t.Fatalf("expected 2 restarts, got %d:\n%s", got, logs.String())
	}
}

func TestInferenceMissingBinaryEndsRun(t *testing.T) {
	inf := NewInference(InferenceConfig{BinaryPath: "/nonexistent/knirvmodels", SocketPath: filepath.Join(t.TempDir(), "s", "llama.sock"), Logf: func(string, ...any) {}})
	if err := inf.Run(context.Background()); err == nil || !strings.Contains(err.Error(), "start") {
		t.Fatalf("err=%v", err)
	}
}

func TestInferenceCancelDuringBackoff(t *testing.T) {
	inf, _, _ := newInference(t, "FAKE_INF_CRASH_FIRST=100")
	inf.cfg.RestartBackoff = []time.Duration{time.Hour}
	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(500*time.Millisecond, cancel)
	t0 := time.Now()
	if err := inf.Run(ctx); err != context.Canceled || time.Since(t0) > 10*time.Second {
		t.Fatalf("err=%v after %s", err, time.Since(t0))
	}
}

func TestExtractAsUsesADifferentNameFromTheDriverBinary(t *testing.T) {
	if len(embeddedBinary) == 0 {
		t.Skip("no embedded binary")
	}
	dir := t.TempDir()
	p, err := ExtractEmbeddedBinaryAs(dir, InferenceBinaryName)
	if err != nil || filepath.Base(p) == "knirvmodels" {
		t.Fatalf("%q %v", p, err)
	}
	if len(InferenceBinaryName) > 15 && InferenceBinaryName[:15] == "knirvmodels" {
		t.Fatal("process name must not collide with pkill -x knirvmodels")
	}
}

func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(20 * time.Millisecond)
	}
}
