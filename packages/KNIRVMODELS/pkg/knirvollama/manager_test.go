package knirvollama

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
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

// TestMain doubles as the fake `ollama` program: the installed "binary" is a shell
// script that re-executes this test binary with FAKE_OLLAMA=1.
func TestMain(m *testing.M) {
	if os.Getenv("FAKE_OLLAMA") == "1" {
		runFakeOllama()
		return
	}
	os.Exit(m.Run())
}

func runFakeOllama() {
	if os.Getenv("FAKE_OLLAMA_EXIT") == "1" {
		os.Exit(3)
	}
	var mu sync.Mutex
	have := os.Getenv("FAKE_OLLAMA_HAVE_MODEL") == "1"
	digest := os.Getenv("FAKE_OLLAMA_DIGEST")
	mux := http.NewServeMux()
	fakeHandlers(mux, &mu, &have, digest, nil)
	if err := http.ListenAndServe(os.Getenv("OLLAMA_HOST"), mux); err != nil {
		os.Exit(4)
	}
}

// fakeHandlers installs the endpoints the manager uses.
func fakeHandlers(mux *http.ServeMux, mu *sync.Mutex, have *bool, digest string, pulls *int32) {
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			fmt.Fprint(w, "Ollama is running")
			return
		}
		http.NotFound(w, r)
	})
	mux.HandleFunc("/api/version", func(w http.ResponseWriter, _ *http.Request) { fmt.Fprint(w, `{"version":"0.9.9"}`) })
	mux.HandleFunc("/api/tags", func(w http.ResponseWriter, _ *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		models := []map[string]string{}
		if *have {
			models = append(models, map[string]string{"name": "nomic-embed-text:latest", "digest": digest})
		}
		json.NewEncoder(w).Encode(map[string]any{"models": models})
	})
	mux.HandleFunc("/api/pull", func(w http.ResponseWriter, _ *http.Request) {
		if pulls != nil {
			atomic.AddInt32(pulls, 1)
		}
		mu.Lock()
		*have = true
		mu.Unlock()
		fmt.Fprintln(w, `{"status":"pulling manifest"}`)
		fmt.Fprintln(w, `{"status":"success"}`)
	})
}

func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

const goodDigest = "0a109f422b47e3a30ba2b10eca18548e944e8a23073ee3f3e947efcf3c45e59f"

// releaseServer serves the fake ollama script and counts downloads.
type releaseServer struct {
	*httptest.Server
	body   []byte
	sha    string
	hits   *int32
	asTgz  bool
	tgzURL string
}

func newRelease(t *testing.T, tgz bool) *releaseServer {
	t.Helper()
	script := []byte(fmt.Sprintf("#!/bin/sh\nexec %q \"$@\"\n", os.Args[0]))
	body := script
	if tgz {
		var buf bytes.Buffer
		gz := gzip.NewWriter(&buf)
		tw := tar.NewWriter(gz)
		add := func(name string, data []byte, mode int64) {
			tw.WriteHeader(&tar.Header{Name: name, Mode: mode, Size: int64(len(data)), Typeflag: tar.TypeReg})
			tw.Write(data)
		}
		add("bin/ollama", script, 0o755)
		add("lib/ollama/libfake.so", []byte("x"), 0o644)
		tw.Close()
		gz.Close()
		body = buf.Bytes()
	}
	sum := sha256.Sum256(body)
	var hits int32
	rs := &releaseServer{body: body, sha: hex.EncodeToString(sum[:]), hits: &hits, asTgz: tgz}
	rs.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.Write(body)
	}))
	t.Cleanup(rs.Close)
	return rs
}

func (r *releaseServer) url() string {
	if r.asTgz {
		return r.URL + "/ollama-linux-amd64.tgz"
	}
	return r.URL + "/ollama"
}

func testCfg(t *testing.T, port int) (Config, *[]string) {
	t.Helper()
	var mu sync.Mutex
	var lines []string
	cfg := DefaultConfig()
	cfg.DataDir = filepath.Join(t.TempDir(), "knirvollama")
	cfg.BaseURL = fmt.Sprintf("http://127.0.0.1:%d", port)
	cfg.StartTimeout = 20 * time.Second
	cfg.PullTimeout = 20 * time.Second
	cfg.StopTimeout = 5 * time.Second
	cfg.RetryBackoff = []time.Duration{10 * time.Millisecond}
	cfg.MaxStartAttempts = 2
	cfg.Logf = func(f string, a ...any) {
		mu.Lock()
		lines = append(lines, fmt.Sprintf(f, a...))
		mu.Unlock()
	}
	return cfg, &lines
}

func TestAdoptsExistingInstanceUntouched(t *testing.T) {
	var mu sync.Mutex
	have := true
	var pulls int32
	mux := http.NewServeMux()
	fakeHandlers(mux, &mu, &have, goodDigest, &pulls)
	srv := httptest.NewServer(mux)
	defer srv.Close()
	rel := newRelease(t, false)

	cfg, _ := testCfg(t, 0)
	cfg.BaseURL = srv.URL
	cfg.Enabled = true // enabled, and still nothing is installed or started
	cfg.ServerURL, cfg.ServerSHA256 = rel.url(), rel.sha
	cfg.EmbeddingDigest = goodDigest[:16]
	m := New(cfg)
	if err := m.Start(t.Context()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	st := m.Status()
	if !st.Ready || !st.Adopted || st.Phase != PhaseReady || !st.Pinned || st.Digest != goodDigest || st.Version != "0.9.9" || st.Endpoint != srv.URL {
		t.Fatalf("status %+v", st)
	}
	if atomic.LoadInt32(rel.hits) != 0 || pulls != 0 {
		t.Fatal("an existing correct instance was touched (download or pull)")
	}
	if _, err := os.Stat(cfg.binDir()); err == nil {
		t.Fatal("something was installed next to an adopted instance")
	}
	m.Stop()
	if _, ok := probe(t.Context(), http.DefaultClient, srv.URL); !ok {
		t.Fatal("Stop killed an instance this manager did not start")
	}
}

func TestFirstStartProvisionsStartsAndPulls(t *testing.T) {
	for _, tgz := range []bool{false, true} {
		t.Run(fmt.Sprintf("tgz=%t", tgz), func(t *testing.T) {
			t.Setenv("FAKE_OLLAMA", "1")
			t.Setenv("FAKE_OLLAMA_DIGEST", goodDigest)
			rel := newRelease(t, tgz)
			cfg, lines := testCfg(t, freePort(t))
			cfg.Enabled = true
			cfg.ServerURL, cfg.ServerSHA256 = rel.url(), rel.sha
			cfg.EmbeddingDigest = goodDigest
			cfg.NumParallel, cfg.KeepAlive = 2, "10m"
			m := New(cfg)
			defer m.Stop()
			if err := m.Start(t.Context()); err != nil {
				t.Fatalf("Start: %v\n%v", err, *lines)
			}
			st := m.Status()
			if !st.Ready || st.Adopted || !st.Pinned || st.Digest != goodDigest || st.Phase != PhaseReady || st.Bind != "127.0.0.1" {
				t.Fatalf("status %+v", st)
			}
			if tgz {
				if _, err := os.Stat(filepath.Join(cfg.binDir(), "lib", "ollama", "libfake.so")); err != nil {
					t.Fatal("archive libraries were not unpacked beside the binary")
				}
			}
			// Visible phases, in order.
			joined := strings.Join(*lines, "\n")
			last := -1
			for _, p := range []string{"adopting", "downloading", "starting", "pulling", "ready at"} {
				i := strings.Index(joined, p)
				if i < 0 || i < last {
					t.Fatalf("phase %q missing or out of order in:\n%s", p, joined)
				}
				last = i
			}
		})
	}
}

func TestChecksumMismatchInstallsNothing(t *testing.T) {
	t.Setenv("FAKE_OLLAMA", "1")
	rel := newRelease(t, false)
	cfg, _ := testCfg(t, freePort(t))
	cfg.Enabled = true
	cfg.ServerURL = rel.url()
	cfg.ServerSHA256 = strings.Repeat("0", 64)
	m := New(cfg)
	err := m.Start(t.Context())
	if err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("want a checksum error, got %v", err)
	}
	if st := m.Status(); st.Phase != PhaseFailed || st.Ready {
		t.Fatalf("status %+v", st)
	}
	if _, err := os.Stat(cfg.binDir()); err == nil {
		t.Fatal("a mismatching artifact was installed")
	}
	if _, err := os.Stat(cfg.recordPath()); err == nil {
		t.Fatal("an install record was written for a mismatching artifact")
	}
}

func TestUnpinnedInstallIsRefused(t *testing.T) {
	cfg, _ := testCfg(t, freePort(t))
	cfg.Enabled = true
	m := New(cfg)
	if err := m.Start(t.Context()); err == nil || !strings.Contains(err.Error(), "ollama_server_sha256") {
		t.Fatalf("want ErrUnpinned, got %v", err)
	}
}

func TestRerunDoesNotDownloadAgain(t *testing.T) {
	t.Setenv("FAKE_OLLAMA", "1")
	t.Setenv("FAKE_OLLAMA_DIGEST", goodDigest)
	rel := newRelease(t, false)
	cfg, _ := testCfg(t, freePort(t))
	cfg.Enabled = true
	cfg.ServerURL, cfg.ServerSHA256 = rel.url(), rel.sha
	m := New(cfg)
	if err := m.Start(t.Context()); err != nil {
		t.Fatal(err)
	}
	m.Stop()
	m2 := New(cfg) // a restart of the daemon
	if err := m2.Start(t.Context()); err != nil {
		t.Fatal(err)
	}
	defer m2.Stop()
	if got := atomic.LoadInt32(rel.hits); got != 1 {
		t.Fatalf("release downloaded %d times, want 1", got)
	}
	// A changed pin is a new install.
	cfg.ServerSHA256 = strings.Repeat("1", 64)
	if _, err := EnsureBinary(t.Context(), cfg, http.DefaultClient); err == nil {
		t.Fatal("a different pin must not reuse the old install")
	}
}

func TestWildcardBindIsRefused(t *testing.T) {
	rel := newRelease(t, false)
	cfg, _ := testCfg(t, 0)
	cfg.Enabled = true
	cfg.ServerURL, cfg.ServerSHA256 = rel.url(), rel.sha

	// Configured to listen on all interfaces: nothing is downloaded or started.
	cfg.BaseURL = fmt.Sprintf("http://0.0.0.0:%d", freePort(t))
	m := New(cfg)
	if err := m.Start(t.Context()); err == nil {
		t.Fatal("0.0.0.0 accepted")
	}
	if st := m.Status(); st.Phase != PhaseDegraded || st.Ready {
		t.Fatalf("status %+v", st)
	}
	if atomic.LoadInt32(rel.hits) != 0 {
		t.Fatal("downloaded for a wildcard endpoint")
	}

	// An adopted loopback URL whose listener is in fact bound to all interfaces.
	var mu sync.Mutex
	have := true
	mux := http.NewServeMux()
	fakeHandlers(mux, &mu, &have, goodDigest, nil)
	srv := httptest.NewServer(mux)
	defer srv.Close()
	old := listenerWildcard
	listenerWildcard = func(int) (bool, bool) { return true, true }
	defer func() { listenerWildcard = old }()
	cfg.BaseURL = srv.URL
	m2 := New(cfg)
	if err := m2.Start(t.Context()); err == nil || m2.Status().Phase != PhaseDegraded {
		t.Fatalf("a wildcard listener was adopted: %v %+v", err, m2.Status())
	}
}

func TestDigestMismatchIsDegradedNotSilent(t *testing.T) {
	var mu sync.Mutex
	have := true
	mux := http.NewServeMux()
	fakeHandlers(mux, &mu, &have, "ffff"+goodDigest[4:], nil)
	srv := httptest.NewServer(mux)
	defer srv.Close()
	cfg, _ := testCfg(t, 0)
	cfg.BaseURL = srv.URL
	cfg.EmbeddingDigest = goodDigest
	m := New(cfg)
	err := m.Start(t.Context())
	st := m.Status()
	if err == nil || st.Ready || st.Phase != PhaseDegraded || !strings.Contains(st.Error, "does not match the pinned") || st.Pinned {
		t.Fatalf("err=%v status=%+v", err, st)
	}
	// With no pin the digest is reported but not enforced.
	cfg.EmbeddingDigest = ""
	m2 := New(cfg)
	if err := m2.Start(t.Context()); err != nil {
		t.Fatal(err)
	}
	if st := m2.Status(); !st.Ready || st.Pinned || st.Digest == "" {
		t.Fatalf("unpinned status %+v", st)
	}
}

func TestStartFailureIsNonFatal(t *testing.T) {
	t.Setenv("FAKE_OLLAMA", "1")
	t.Setenv("FAKE_OLLAMA_EXIT", "1")
	rel := newRelease(t, false)
	cfg, _ := testCfg(t, freePort(t))
	cfg.Enabled = true
	cfg.ServerURL, cfg.ServerSHA256 = rel.url(), rel.sha
	m := New(cfg)
	err := m.Start(t.Context())
	if err == nil || !strings.Contains(err.Error(), "exited during start") {
		t.Fatalf("err = %v", err)
	}
	if st := m.Status(); st.Phase != PhaseFailed || st.Ready || st.Error == "" {
		t.Fatalf("status %+v", st)
	}
	m.Stop() // must not hang or panic
}

func TestNotEnabledAndNothingRunningIsIdleNotAnError(t *testing.T) {
	cfg, _ := testCfg(t, freePort(t))
	m := New(cfg)
	if err := m.Start(t.Context()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if st := m.Status(); st.Phase != PhaseDisabled || st.Ready {
		t.Fatalf("status %+v", st)
	}
}

func TestExtractRefusesEscapingEntries(t *testing.T) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	tw.WriteHeader(&tar.Header{Name: "../evil", Mode: 0o644, Size: 1, Typeflag: tar.TypeReg})
	tw.Write([]byte("x"))
	tw.Close()
	gz.Close()
	archive := filepath.Join(t.TempDir(), "a.tgz")
	os.WriteFile(archive, buf.Bytes(), 0o600)
	if err := extractTarGz(archive, t.TempDir()); err == nil || !strings.Contains(err.Error(), "escapes") {
		t.Fatalf("want an escape error, got %v", err)
	}
}

func TestConfigFromEnvAndEndpointForms(t *testing.T) {
	t.Setenv("KNIRV_OLLAMA_ENABLED", "1")
	t.Setenv("OLLAMA_HOST", "127.0.0.1")
	t.Setenv("KNIRV_OLLAMA_EMBEDDING_DIGEST", "SHA256:ABC")
	t.Setenv("KNIRV_OLLAMA_NUM_PARALLEL", "4")
	c := FromEnv(nil).normalize()
	if !c.Enabled || c.BaseURL != "http://127.0.0.1:11434" || c.NumParallel != 4 || c.EmbeddingModel != "nomic-embed-text" {
		t.Fatalf("%+v", c)
	}
	for in, want := range map[string]string{"localhost:9": "http://localhost:9", "http://0.0.0.0:11434/": "http://0.0.0.0:11434", "[::1]": "http://[::1]:11434"} {
		if got := normalizeBaseURL(in); got != want {
			t.Errorf("normalizeBaseURL(%q) = %q, want %q", in, got, want)
		}
	}
	if isLoopbackHost("0.0.0.0") || isLoopbackHost("") || isLoopbackHost("::") || !isLoopbackHost("127.0.0.1") || !isLoopbackHost("::1") {
		t.Fatal("loopback classification is wrong")
	}
	if !digestMatches("sha256:"+goodDigest, goodDigest[:12]) || digestMatches(goodDigest, "abc") || digestMatches("", "x") {
		t.Fatal("digestMatches")
	}
}
