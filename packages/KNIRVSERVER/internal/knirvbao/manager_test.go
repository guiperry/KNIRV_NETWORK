package knirvbao

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	if os.Getenv("KNIRVBAO_FAKE_CHILD") == "1" {
		fakeBao()
		return
	}
	os.Exit(m.Run())
}

func fakeBao() {
	if os.Getenv("KNIRVBAO_FAKE_FAIL") == "1" {
		os.Exit(2)
	}
	if len(os.Args) < 4 {
		os.Exit(2)
	}
	config, err := os.ReadFile(os.Args[3])
	if err != nil {
		os.Exit(2)
	}
	match := regexp.MustCompile(`(?m)^  address = ("[^\n]+/bao\.sock")$`).FindSubmatch(config)
	if len(match) != 2 {
		os.Exit(2)
	}
	socket, err := strconv.Unquote(string(match[1]))
	if err != nil {
		os.Exit(2)
	}
	root := filepath.Dir(socket)
	var failStep string
	if data, rerr := os.ReadFile(filepath.Join(root, "fake-fail")); rerr == nil {
		failStep = strings.TrimSpace(string(data))
	}
	listener, err := net.Listen("unix", socket)
	if err != nil {
		os.Exit(2)
	}
	if err := os.Chmod(socket, 0600); err != nil {
		os.Exit(2)
	}
	if os.Getenv("KNIRV_BAO_SEAL_KEY") == "" || os.Getenv("ORACLE_KEY_PASSWORD") != "" || os.Getenv("BOOT_KEY_PASSWORD") != "" {
		os.Exit(2)
	}
	if err := os.WriteFile(filepath.Join(root, "fake-env-ok"), []byte("ok"), 0600); err != nil {
		os.Exit(2)
	}
	var sealed atomic.Bool
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sys/seal-status", func(w http.ResponseWriter, _ *http.Request) {
		_, err := os.Stat(filepath.Join(root, "fake-initialized"))
		_ = json.NewEncoder(w).Encode(map[string]bool{"initialized": err == nil, "sealed": sealed.Load()})
	})
	mux.HandleFunc("/v1/sys/init", func(w http.ResponseWriter, _ *http.Request) {
		if err := os.WriteFile(filepath.Join(root, "fake-initialized"), []byte("yes"), 0600); err != nil {
			w.WriteHeader(500)
			return
		}
		_, _ = io.WriteString(w, `{"root_token":"synthetic-test-token","recovery_keys":["synthetic-recovery-key-1"]}`)
	})
	mux.HandleFunc("/v1/sys/health", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	mux.HandleFunc("/v1/test/crash", func(http.ResponseWriter, *http.Request) { os.Exit(2) })
	mux.HandleFunc("/v1/test/seal", func(w http.ResponseWriter, _ *http.Request) { sealed.Store(true); w.WriteHeader(http.StatusNoContent) })
	mux.HandleFunc("/v1/test/unseal", func(w http.ResponseWriter, _ *http.Request) { sealed.Store(false); w.WriteHeader(http.StatusNoContent) })
	mux.HandleFunc("/v1/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && (r.URL.Path == "/v1/sys/mounts" || r.URL.Path == "/v1/sys/auth") {
			_, _ = io.WriteString(w, `{"data":{}}`)
			return
		}
		switch {
		case failStep == "mount" && r.Method == http.MethodPost && (strings.HasPrefix(r.URL.Path, "/v1/sys/mounts/") || strings.HasPrefix(r.URL.Path, "/v1/sys/auth/")):
			w.WriteHeader(http.StatusInternalServerError)
			return
		case failStep == "policy" && r.Method == http.MethodPut && strings.HasPrefix(r.URL.Path, "/v1/sys/policies/acl/"):
			w.WriteHeader(http.StatusInternalServerError)
			return
		case failStep == "role" && r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/v1/auth/approle/role/"):
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if r.Header.Get("X-Vault-Token") != "synthetic-test-token" {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	server := &http.Server{Handler: mux}
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGTERM)
	go func() { <-signals; _ = server.Close() }()
	_ = server.Serve(listener)
}

type staticSeal struct {
	key []byte
	err error
}

type flakySeal struct{ calls atomic.Int32 }

func (s *flakySeal) SealKey(context.Context) ([]byte, error) {
	if s.calls.Add(1) == 1 {
		return nil, errors.New("temporary seal failure")
	}
	return make([]byte, 32), nil
}

func TestManagerRetriesTransientStartupFailure(t *testing.T) {
	m := fakeManager(t)
	m.source = &flakySeal{}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	t.Cleanup(func() {
		stop, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = m.Stop(stop)
	})
	if err := m.Start(ctx); err == nil {
		t.Fatal("initial seal failure was hidden")
	}
	if status := m.Health(); status.State != Degraded {
		t.Fatal("initial failure was not degraded")
	}
	for deadline := time.Now().Add(30 * time.Second); time.Now().Before(deadline); {
		m.mu.Lock()
		state := m.status.State
		m.mu.Unlock()
		if state == Ready {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatal("manager did not recover after a transient startup failure")
}

func TestManagerWritesRecoveryEnvOnFirstBoot(t *testing.T) {
	m := fakeManager(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	t.Cleanup(func() {
		stop, stopCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer stopCancel()
		_ = m.Stop(stop)
	})
	var logBuf bytes.Buffer
	prev := log.Writer()
	log.SetOutput(&logBuf)
	defer log.SetOutput(prev)

	if err := m.Start(ctx); err != nil {
		t.Fatalf("first boot was blocked: %v", err)
	}
	root, _ := m.config.root()
	if _, err := os.Stat(filepath.Join(root, "fake-initialized")); err != nil {
		t.Fatal("first boot did not initialize storage")
	}
	envPath := filepath.Join(m.config.ConfigDir, "openbao.env")
	data, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("recovery env not written: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "OPENBAO_ROOT_TOKEN=synthetic-test-token") {
		t.Fatalf("root token missing from recovery env:\n%s", content)
	}
	if !strings.Contains(content, "OPENBAO_RECOVERY_KEY_1=synthetic-recovery-key-1") {
		t.Fatalf("recovery key missing from recovery env:\n%s", content)
	}
	info, err := os.Stat(envPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("recovery env is not private: %04o", info.Mode().Perm())
	}
	logged := logBuf.String()
	if strings.Contains(logged, "synthetic-test-token") || strings.Contains(logged, "synthetic-recovery-key-1") {
		t.Fatalf("recovery material was logged:\n%s", logged)
	}
	if !strings.Contains(logged, envPath) {
		t.Fatalf("first boot did not log the recovery env path:\n%s", logged)
	}
}

func (s staticSeal) SealKey(context.Context) ([]byte, error) {
	return append([]byte(nil), s.key...), s.err
}

func fakeManager(t *testing.T) *Manager {
	t.Helper()
	t.Setenv("ORACLE_KEY_PASSWORD", "synthetic-root-password")
	t.Setenv("BOOT_KEY_PASSWORD", "synthetic-boot-password")
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	quoted := "'" + strings.ReplaceAll(executable, "'", "'\\''") + "'"
	binary := []byte("#!/bin/sh\nKNIRVBAO_FAKE_CHILD=1 exec " + quoted + " \"$@\"\n")
	plugin := []byte("synthetic test plugin")
	digest := sha256.Sum256(plugin)
	m := NewManager(Config{Enabled: true, DataDir: t.TempDir(), ConfigDir: t.TempDir()}, staticSeal{key: make([]byte, 32)})
	m.baoBinary = func() []byte { return binary }
	m.pluginBinary = func() []byte { return plugin }
	m.pluginSHA = func() string { return hex.EncodeToString(digest[:]) }
	return m
}

func TestManagerLifecycleAndRestart(t *testing.T) {
	m := fakeManager(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	t.Cleanup(func() {
		stop, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = m.Stop(stop)
	})
	if err := m.Start(ctx); err != nil {
		t.Fatal(err)
	}
	if status := m.Health(); status.State != Ready {
		t.Fatalf("initial health: %+v", status)
	}
	second := NewManager(m.config, staticSeal{key: make([]byte, 32)})
	if err := second.Start(ctx); err == nil {
		t.Fatal("second manager acquired the active data directory")
	}
	if status := m.Health(); status.State != Ready {
		t.Fatal("second manager disturbed the first")
	}
	root, _ := m.config.root()
	if _, err := os.Stat(filepath.Join(root, "fake-env-ok")); err != nil {
		t.Fatal("child environment was not isolated")
	}
	api := newBaoAPI(m.Socket())
	if _, err := api.request(ctx, http.MethodPost, "test/seal", "", nil, nil); err != nil {
		t.Fatal(err)
	}
	if status := m.Health(); status.State != Degraded || !strings.Contains(status.Reason, "sealed") {
		t.Fatal("sealed daemon did not degrade health")
	}
	if _, err := api.request(ctx, http.MethodPost, "test/unseal", "", nil, nil); err != nil {
		t.Fatal(err)
	}
	if status := m.Health(); status.State != Ready {
		t.Fatal("health did not recover after unseal")
	}
	_, _ = api.request(ctx, http.MethodPost, "test/crash", "", nil, nil)
	api.client.CloseIdleConnections()
	deadline := time.Now().Add(20 * time.Second)
	sawDegraded := false
	for time.Now().Before(deadline) {
		m.mu.Lock()
		state := m.status.State
		m.mu.Unlock()
		if state == Degraded {
			sawDegraded = true
		}
		if sawDegraded && state == Ready {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if !sawDegraded || m.Health().State != Ready {
		t.Fatal("owned child did not degrade and restart")
	}
	stop, stopCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer stopCancel()
	if err := m.Stop(stop); err != nil {
		t.Fatal(err)
	}
	if m.Health().State != Absent {
		t.Fatal("stopped manager is not absent")
	}
}

func TestManagerAbsentAndExistingSocket(t *testing.T) {
	base := t.TempDir()
	m := NewManager(Config{Enabled: true, DataDir: base}, staticSeal{})
	if err := m.Start(context.Background()); err != nil || m.Health().State != Absent {
		t.Fatal("keyless node did not stay absent")
	}
	root, _ := m.config.root()
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Fatal("keyless node created OpenBao files")
	}
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "bao.sock")
	if err := os.WriteFile(path, []byte("untouched"), 0600); err != nil {
		t.Fatal(err)
	}
	m.source = staticSeal{key: make([]byte, 32)}
	if err := m.Start(context.Background()); err == nil || m.Health().State != Degraded {
		t.Fatal("existing socket path was accepted")
	}
	contents, err := os.ReadFile(path)
	if err != nil || !strings.EqualFold(string(contents), "untouched") {
		t.Fatal("existing path was modified")
	}
}

// F1: an existing world-readable openbao.env must be re-privatized to 0600.
func TestWriteRecoveryEnvOverwrites0644With0600(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "openbao.env")
	if err := os.WriteFile(path, []byte("OLD"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := writeRecoveryEnv(dir, "", "tok", []string{"k1"}); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("recovery env not re-privatized: %04o", info.Mode().Perm())
	}
	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "OPENBAO_ROOT_TOKEN=tok") {
		t.Fatalf("recovery env content not written:\n%s", data)
	}
	if strings.Contains(string(data), "OLD") {
		t.Fatal("old content was not replaced")
	}
}

// F1: a permissive umask must not leak the credentials with a looser mode.
func TestWriteRecoveryEnvRespectsPermissiveUmask(t *testing.T) {
	old := syscall.Umask(0)
	defer syscall.Umask(old)
	dir := t.TempDir()
	if _, err := writeRecoveryEnv(dir, "", "tok", nil); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filepath.Join(dir, "openbao.env"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("recovery env not 0600 under umask 0: %04o", info.Mode().Perm())
	}
}

// F1: an existing symlink target must never be followed or overwritten.
func TestWriteRecoveryEnvRejectsSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	if err := os.WriteFile(target, []byte("SECRET"), 0600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "openbao.env")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if _, err := writeRecoveryEnv(dir, "", "tok", nil); err == nil {
		t.Fatal("symlinked recovery env was not rejected")
	}
	data, _ := os.ReadFile(target)
	if string(data) != "SECRET" {
		t.Fatal("symlink target was modified")
	}
	info, _ := os.Lstat(link)
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatal("symlink was replaced with a real file")
	}
}

// F1: a write/rename failure must leave the previously existing file intact and leave
// no leftover temp file behind.
func TestWriteRecoveryEnvFailureKeepsPriorFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "openbao.env")
	if err := os.WriteFile(path, []byte("PRIOR"), 0600); err != nil {
		t.Fatal(err)
	}
	recoveryEnvFailPreRename = errors.New("injected write failure")
	defer func() { recoveryEnvFailPreRename = nil }()
	if _, err := writeRecoveryEnv(dir, "", "tok", nil); err == nil {
		t.Fatal("expected injected failure")
	}
	data, _ := os.ReadFile(path)
	if string(data) != "PRIOR" {
		t.Fatalf("prior recovery file was modified: %q", data)
	}
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".openbao-env-") {
			t.Fatalf("leftover temp file %s", e.Name())
		}
	}
}

// F2 + F3: an interrupted bootstrap (here, a mount failure) persists the recovery
// credentials before failing, and the same manager resumes from them once the failure
// is cleared. The persisted token is never revoked and never appears in logs.
func TestManagerResumesInterruptedBootstrapFromEnv(t *testing.T) {
	m := fakeManager(t)
	root, _ := m.config.root()
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}
	failFile := filepath.Join(root, "fake-fail")
	if err := os.WriteFile(failFile, []byte("mount"), 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	t.Cleanup(func() {
		stop, c := context.WithTimeout(context.Background(), 10*time.Second)
		defer c()
		_ = m.Stop(stop)
	})
	var logBuf bytes.Buffer
	prev := log.Writer()
	log.SetOutput(&logBuf)
	defer log.SetOutput(prev)

	if err := m.Start(ctx); err == nil {
		t.Fatal("initial bootstrap was expected to fail before the marker")
	}
	envPath := filepath.Join(m.config.ConfigDir, "openbao.env")
	if _, err := os.Stat(envPath); err != nil {
		t.Fatalf("recovery env not persisted before the failure: %v", err)
	}
	if markerPresent(root) {
		t.Fatal("marker was written despite the bootstrap failure")
	}
	if err := os.Remove(failFile); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(90 * time.Second)
	for time.Now().Before(deadline) {
		if markerPresent(root) && m.Health().State == Ready {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	if !markerPresent(root) {
		t.Fatal("marker not written after the failure was cleared")
	}
	if status := m.Health(); status.State != Ready {
		t.Fatalf("recovered manager not ready: %+v", status)
	}
	if strings.Contains(logBuf.String(), "synthetic-test-token") {
		t.Fatalf("persisted token leaked into logs:\n%s", logBuf.String())
	}
}

// F2: when the credential file itself cannot be persisted, return a precise degraded
// error. The vault is left initialized and recoverable, but no (false) success is claimed.
func TestManagerReportsDegradedWhenEnvPersistenceFails(t *testing.T) {
	m := fakeManager(t)
	recoveryEnvFailPreRename = errors.New("disk full")
	defer func() { recoveryEnvFailPreRename = nil }()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	t.Cleanup(func() {
		stop, c := context.WithTimeout(context.Background(), 10*time.Second)
		defer c()
		_ = m.Stop(stop)
	})
	err := m.Start(ctx)
	if err == nil {
		t.Fatal("expected a degraded error when env persistence failed")
	}
	if !strings.Contains(err.Error(), "recovery credentials could not be persisted") {
		t.Fatalf("not the precise degraded error: %v", err)
	}
	root, _ := m.config.root()
	if _, statErr := os.Stat(filepath.Join(root, "fake-initialized")); statErr != nil {
		t.Fatal("vault was left uninitialized despite a precise degraded error")
	}
	if _, statErr := os.Stat(filepath.Join(m.config.ConfigDir, "openbao.env")); !os.IsNotExist(statErr) {
		t.Fatal("recovery env should be absent when persistence failed")
	}
}

// ---------------------------------------------------------------------------
// Socket-free supervisor tests (injectable clock + fake-child factory). These run
// under the auditor sandbox because they start no listeners.
// ---------------------------------------------------------------------------

// fakeClock records every backoff interval requested and fires immediately.
type fakeClock struct {
	mu sync.Mutex
	d  []time.Duration
}

func (c *fakeClock) After(d time.Duration) <-chan time.Time {
	c.mu.Lock()
	c.d = append(c.d, d)
	c.mu.Unlock()
	return time.After(0)
}

func (c *fakeClock) durations() []time.Duration {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]time.Duration, len(c.d))
	copy(out, c.d)
	return out
}

// crashingFakeChild is a child whose process "exits" immediately.
func crashingFakeChild() *child {
	c := &child{wait: make(chan error, 1)}
	c.wait <- nil
	return c
}

func mgrState(m *Manager) State {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.status.State
}

func mgrReason(m *Manager) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.status.Reason
}

func waitForState(m *Manager, want State, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if mgrState(m) == want {
			return nil
		}
		time.Sleep(5 * time.Millisecond)
	}
	return fmt.Errorf("state %s not reached (current %s)", want, mgrState(m))
}

func waitForSpawn(spawns *atomic.Int64, want int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if int(spawns.Load()) >= want {
			return nil
		}
		time.Sleep(5 * time.Millisecond)
	}
	return fmt.Errorf("spawn count %d did not reach %d", spawns.Load(), want)
}

// newSuperviseHarness builds a Manager wired to an injectable clock and child
// factory, bypassing Start (so no real OpenBao binary is launched).
func newSuperviseHarness(t *testing.T, stableWindow time.Duration, makeChild func(context.Context) (*child, bool, error)) (*Manager, *fakeClock, *atomic.Int64) {
	t.Helper()
	clk := &fakeClock{}
	var spawns atomic.Int64
	m := &Manager{
		config:       Config{Enabled: true},
		maxRestarts:  5,
		status:       Status{State: Absent},
		clock:        clk,
		now:          time.Now,
		stableWindow: stableWindow,
	}
	m.spawn = func(ctx context.Context) (*child, bool, error) {
		spawns.Add(1)
		return makeChild(ctx)
	}
	return m, clk, &spawns
}

// Backoff starts at 1s and doubles to the 30s cap, and a continuously crashing
// vault is given up after maxRestarts failures (terminal Degraded, no more spawns).
func TestSuperviseExhaustsAfterMaxRestarts(t *testing.T) {
	m, clk, spawns := newSuperviseHarness(t, time.Minute, func(context.Context) (*child, bool, error) {
		return crashingFakeChild(), false, nil
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go m.supervise(ctx, nil, done)

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("supervisor did not terminate after exhausting restarts")
	}
	if got := int(spawns.Load()); got != m.maxRestarts+1 {
		t.Fatalf("spawn count = %d, want %d", got, m.maxRestarts+1)
	}
	st := m.Health()
	if st.State != Degraded || !strings.Contains(st.Reason, "restarts exhausted") {
		t.Fatalf("expected terminal degraded, got %+v", st)
	}
	got := clk.durations()
	want := []time.Duration{time.Second, 2 * time.Second, 4 * time.Second, 8 * time.Second, 16 * time.Second}
	if len(got) != len(want) {
		t.Fatalf("recorded backoffs = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("backoff[%d] = %v, want %v (full %v)", i, got[i], want[i], got)
		}
	}
}

// A child that crashes leaves the manager Degraded while it is down, and once a
// fresh child is up again the manager recovers to Ready. The restart spawn blocks
// on a gate so the Degraded window is observable deterministically.
func TestSuperviseDegradedWhileChildDownThenRecovers(t *testing.T) {
	var mu sync.Mutex
	var waits []chan error
	restartGate := make(chan struct{})
	m, _, _ := newSuperviseHarness(t, time.Minute, func(ctx context.Context) (*child, bool, error) {
		mu.Lock()
		defer mu.Unlock()
		if len(waits) == 0 {
			w := make(chan error, 1)
			waits = append(waits, w)
			return &child{wait: w}, false, nil
		}
		select {
		case <-restartGate:
		case <-ctx.Done():
			return nil, false, ctx.Err()
		}
		w := make(chan error, 1)
		waits = append(waits, w)
		return &child{wait: w}, false, nil
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go m.supervise(ctx, nil, done)
	if err := waitForState(m, Ready, 2*time.Second); err != nil {
		t.Fatalf("did not become ready: %v", err)
	}
	mu.Lock()
	first := waits[0]
	mu.Unlock()
	close(first)
	if err := waitForState(m, Degraded, 2*time.Second); err != nil {
		t.Fatalf("did not become degraded while child down: %v", err)
	}
	close(restartGate)
	if err := waitForState(m, Ready, 2*time.Second); err != nil {
		t.Fatalf("did not recover after outage: %v", err)
	}
	cancel()
	<-done
}

// A sustained healthy run clears the consecutive-failure budget, so the manager
// does not give up at the fixed maxRestarts boundary. (Mutation of nextBackoff
// would make the backoff sequence constant and this contract observable as a
// change; the reset is what keeps it alive past spawn 6.)
func TestSuperviseResetsCounterAfterSustainedSuccess(t *testing.T) {
	var mu sync.Mutex
	var stableWait chan error
	var spawnN int64
	m, clk, spawns := newSuperviseHarness(t, 20*time.Millisecond, func(context.Context) (*child, bool, error) {
		n := atomic.AddInt64(&spawnN, 1)
		mu.Lock()
		defer mu.Unlock()
		if n == 4 {
			stableWait = make(chan error, 1)
			return &child{wait: stableWait}, false, nil
		}
		return crashingFakeChild(), false, nil
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go m.supervise(ctx, nil, done)

	if err := waitForSpawn(spawns, 4, 2*time.Second); err != nil {
		t.Fatalf("stable child not started: %v", err)
	}
	// Let it run past the stable window, then crash it.
	time.Sleep(100 * time.Millisecond)
	mu.Lock()
	sw := stableWait
	mu.Unlock()
	close(sw)

	// Without the reset, restarts would hit maxRestarts at the 6th spawn and the
	// manager would give up (spawns would stall at exactly 6). With the reset it
	// keeps going well past that boundary.
	if err := waitForSpawn(spawns, 8, 2*time.Second); err != nil {
		t.Fatalf("counter did not reset after sustained success; gave up early: %v", err)
	}
	cancel()
	<-done
	if got := int(spawns.Load()); got <= m.maxRestarts+1 {
		t.Fatalf("counter did not reset after sustained success; supervisor stalled at spawns=%d (would exhaust at %d without reset)", got, m.maxRestarts+1)
	}
	// The backoff sequence must show a reset to 1s after the sustained success
	// (climbs 1,2,4 then restarts at 1s again).
	if !slicesContains(clk.durations(), time.Second) {
		t.Fatalf("backoff never reset to 1s after sustained success; got %v", clk.durations())
	}
}

func slicesContains(s []time.Duration, v time.Duration) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}
