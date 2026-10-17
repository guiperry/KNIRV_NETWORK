package harness

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// fakeColi stands in for Colibri's `c/coli`. It is run with /bin/sh, records the
// arguments of `setup`, and answers `status --json` from marker files.
const fakeColi = `#!/bin/sh
home="$COLI_SETUP_HOME"
mkdir -p "$home"
case "$1" in
setup)
  case "$*" in
  *--list*)
    if [ -n "$FAKE_COLI_NOFIT" ]; then
      echo '[{"id":"qwen36-35b","fits":false,"recommended":true,"reason":"needs 20 GB of RAM"}]'
    else
      echo 'detecting hardware...'
      echo '[{"id":"huge","fits":false,"recommended":false},{"id":"small-model","fits":true,"recommended":true,"download_gb":19}]'
    fi ;;
  *) echo "$@" > "$home/setup.args"; echo "downloading model" ;;
  esac ;;
start) touch "$home/started" ;;
stop) rm -f "$home/started" ;;
status)
  if [ -f "$home/started" ]; then
    echo '{"install":{"phase":"started"},"server":{"state":"ready","model_id":"colibri-small","urls":{"openai_base_url":"'"$FAKE_COLI_BASE"'"}}}'
  else
    echo '{"install":{"phase":"ready"},"server":{"state":"stopped"}}'
  fi ;;
esac
`

func sourceTarball(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	add := func(name, body string, mode int64) {
		tw.WriteHeader(&tar.Header{Name: name, Mode: mode, Size: int64(len(body)), Typeflag: tar.TypeReg})
		tw.Write([]byte(body))
	}
	tw.WriteHeader(&tar.Header{Name: "colibri-2.0.0/", Typeflag: tar.TypeDir, Mode: 0o755})
	add("colibri-2.0.0/c/coli", fakeColi, 0o755)
	add("colibri-2.0.0/README.md", "hi", 0o644)
	tw.Close()
	gz.Close()
	return buf.Bytes()
}

type colibriFixture struct {
	c         *Colibri
	downloads *atomic.Int32
	logs      *safeBuf
	// apiDelay is how long the fake model takes to answer a chat request.
	apiDelay *atomic.Int64
}

func newColibriFixture(t *testing.T, cfg ColibriConfig, env ...string) *colibriFixture {
	t.Helper()
	var downloads atomic.Int32
	tarball := sourceTarball(t)
	src := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		downloads.Add(1)
		w.Write(tarball)
	}))
	t.Cleanup(src.Close)
	var apiDelay atomic.Int64
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/models" {
			w.Write([]byte(`{"data":[{"id":"colibri-small"}]}`))
			return
		}
		if r.URL.Path == "/v1/chat/completions" {
			var req struct {
				MaxTokens int `json:"max_tokens"`
			}
			json.NewDecoder(r.Body).Decode(&req)
			time.Sleep(time.Duration(apiDelay.Load()))
			w.Write([]byte(`{"choices":[{"message":{"content":"one two"}}],"usage":{"completion_tokens":` + strconv.Itoa(req.MaxTokens) + `}}`))
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(api.Close)
	logs := &safeBuf{}
	cfg.Home = t.TempDir()
	cfg.SourceURL = src.URL + "/colibri.tar.gz"
	cfg.Python = "/bin/sh"
	cfg.Port = 8199
	cfg.Logf = logs.logf
	if cfg.CPUInfoPath == "" {
		cfg.CPUInfoPath = writeCPUInfo(t, "Test CPU", "fpu sse4_2 avx avx2 fma")
	}
	cfg.Env = append(cfg.Env, "FAKE_COLI_BASE="+api.URL+"/v1")
	cfg.Env = append(cfg.Env, env...)
	return &colibriFixture{c: NewColibri(cfg), downloads: &downloads, logs: logs, apiDelay: &apiDelay}
}

func TestInstallSetsTheOneParameterToTheRecommendedModel(t *testing.T) {
	f := newColibriFixture(t, ColibriConfig{})
	if err := f.c.Install(context.Background()); err != nil {
		t.Fatalf("%v\n%s", err, f.logs.String())
	}
	if f.c.ChosenModel() != "small-model" {
		t.Fatalf("chosen %q", f.c.ChosenModel())
	}
	args, err := os.ReadFile(filepath.Join(f.c.cfg.Home, "state", "setup.args"))
	if err != nil {
		t.Fatal(err)
	}
	got := string(args)
	for _, want := range []string{"setup", "--yes", "--no-start", "--model small-model", "--port 8199", "--host 127.0.0.1", "--dir " + filepath.Join(f.c.cfg.Home, "models")} {
		if !strings.Contains(got, want) {
			t.Fatalf("setup args missing %q: %s", want, got)
		}
	}
	if !strings.Contains(f.logs.String(), "[COLIBRI] downloading model") {
		t.Fatalf("colibri output must be forwarded:\n%s", f.logs.String())
	}
}

func TestInstallIsIdempotentAndDownloadsSourceOnce(t *testing.T) {
	f := newColibriFixture(t, ColibriConfig{})
	for i := 0; i < 2; i++ {
		if err := f.c.Install(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	if n := f.downloads.Load(); n != 1 {
		t.Fatalf("source downloaded %d times", n)
	}
}

func TestExplicitModelOverridesRecommendation(t *testing.T) {
	f := newColibriFixture(t, ColibriConfig{Model: "qwen3-coder-30b"})
	if err := f.c.Install(context.Background()); err != nil {
		t.Fatal(err)
	}
	args, _ := os.ReadFile(filepath.Join(f.c.cfg.Home, "state", "setup.args"))
	if !strings.Contains(string(args), "--model qwen3-coder-30b") {
		t.Fatalf("%s", args)
	}
}

func TestModelFromEnvironment(t *testing.T) {
	t.Setenv("KNIRV_COLIBRI_MODEL", "olmoe")
	c := NewColibri(ColibriConfig{Home: t.TempDir()})
	if c.cfg.Model != "olmoe" {
		t.Fatalf("%q", c.cfg.Model)
	}
}

func TestNoModelFitsIsPermanent(t *testing.T) {
	f := newColibriFixture(t, ColibriConfig{}, "FAKE_COLI_NOFIT=1")
	err := f.c.Install(context.Background())
	if !errors.Is(err, ErrNoModelFits) || !IsPermanent(err) {
		t.Fatalf("err=%v", err)
	}
	if !strings.Contains(err.Error(), "needs 20 GB of RAM") {
		t.Fatalf("the reason must reach the operator: %v", err)
	}
}

func TestMissingPythonIsPermanent(t *testing.T) {
	c := NewColibri(ColibriConfig{Home: t.TempDir(), Python: "/nonexistent/python", CPUInfoPath: writeCPUInfo(t, "T", "avx2")})
	if err := c.Install(context.Background()); !IsPermanent(err) {
		t.Fatalf("err=%v", err)
	}
}

func TestDownloadFailureIsRetryable(t *testing.T) {
	src := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Error(w, "no", 502) }))
	defer src.Close()
	c := NewColibri(ColibriConfig{Home: t.TempDir(), SourceURL: src.URL, Python: "/bin/sh", CPUInfoPath: writeCPUInfo(t, "T", "avx2")})
	err := c.Install(context.Background())
	if err == nil || IsPermanent(err) {
		t.Fatalf("a network failure must be retried, got %v", err)
	}
	if _, statErr := os.Stat(c.srcDir()); statErr == nil {
		t.Fatal("a failed download must not leave something that looks installed")
	}
}

func TestStartWaitsForReadyAndReportsEndpoint(t *testing.T) {
	f := newColibriFixture(t, ColibriConfig{})
	ctx := context.Background()
	if err := f.c.Install(ctx); err != nil {
		t.Fatal(err)
	}
	if f.c.Healthy() {
		t.Fatal("not healthy before start")
	}
	if err := f.c.Start(ctx); err != nil {
		t.Fatalf("%v\n%s", err, f.logs.String())
	}
	if f.c.ModelID() != "colibri-small" || !strings.HasSuffix(f.c.BaseURL(), "/v1") || !f.c.Healthy() {
		t.Fatalf("model=%q base=%q healthy=%v", f.c.ModelID(), f.c.BaseURL(), f.c.Healthy())
	}
	if f.c.HostPort() == "" {
		t.Fatal("host:port")
	}
	if err := f.c.Stop(ctx); err != nil {
		t.Fatal(err)
	}
	if f.c.Healthy() || f.c.BaseURL() != "" {
		t.Fatal("stopped")
	}
}

func TestStartTimesOutWhenNeverReady(t *testing.T) {
	f := newColibriFixture(t, ColibriConfig{ReadyTimeout: 300 * time.Millisecond})
	if err := f.c.Install(context.Background()); err != nil {
		t.Fatal(err)
	}
	// status never reports ready.
	os.WriteFile(f.c.coli(), []byte("#!/bin/sh\ncase \"$1\" in status) echo '{\"server\":{\"state\":\"loading\"}}';; esac\n"), 0o755)
	if err := f.c.Start(context.Background()); err == nil || !strings.Contains(err.Error(), "not ready") {
		t.Fatalf("err=%v", err)
	}
}

func TestUntarSkipsEscapingPathsAndLinks(t *testing.T) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for _, h := range []tar.Header{
		{Name: "top/../../evil.txt", Typeflag: tar.TypeReg, Size: 1, Mode: 0o644},
		{Name: "top/link", Typeflag: tar.TypeSymlink, Linkname: "/etc/passwd"},
		{Name: "top/ok.txt", Typeflag: tar.TypeReg, Size: 1, Mode: 0o644},
	} {
		h := h
		tw.WriteHeader(&h)
		if h.Typeflag == tar.TypeReg {
			tw.Write([]byte("x"))
		}
	}
	tw.Close()
	gz.Close()
	root := t.TempDir()
	dest := filepath.Join(root, "dest")
	os.Mkdir(dest, 0o755)
	if err := untarGz(&buf, dest); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "evil.txt")); err == nil {
		t.Fatal("path traversal escaped the destination")
	}
	if _, err := os.Lstat(filepath.Join(dest, "link")); err == nil {
		t.Fatal("symlink extracted")
	}
	if _, err := os.Stat(filepath.Join(dest, "ok.txt")); err != nil {
		t.Fatal("regular file missing")
	}
}

func TestPickRecommendedAcceptsWrappedList(t *testing.T) {
	id, err := pickRecommended([]byte("log line\n{\"models\":[{\"id\":\"a\",\"fits\":true,\"recommended\":true}]}"))
	if err != nil || id != "a" {
		t.Fatalf("%q %v", id, err)
	}
}

// testdata holds the output of the real Colibri v2.0.0 `coli setup --list --json` and
// `coli status --json` (before setup), so the parsing is checked against what Colibri
// actually prints and not only against the fake above.
func TestPickRecommendedFromRealColibriOutput(t *testing.T) {
	out, err := os.ReadFile("testdata/colibri_setup_list_v2.0.0.json")
	if err != nil {
		t.Fatal(err)
	}
	id, err := pickRecommended(out)
	if err != nil || id != "qwen36-35b" {
		t.Fatalf("got %q, %v", id, err)
	}
}

func TestRealUnconfiguredStatusIsNotReady(t *testing.T) {
	out, err := os.ReadFile("testdata/colibri_status_unconfigured_v2.0.0.json")
	if err != nil {
		t.Fatal(err)
	}
	var st colibriStatus
	if err := json.Unmarshal(extractJSON(out), &st); err != nil {
		t.Fatal(err)
	}
	if st.Server.State == "ready" {
		t.Fatalf("%+v", st)
	}
}

func TestHealthyRejectsAnUnrelatedWebServer(t *testing.T) {
	web := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("<html>some dashboard</html>"))
	}))
	defer web.Close()
	c := NewColibri(ColibriConfig{Home: t.TempDir()})
	c.baseURL = web.URL + "/v1"
	if c.Healthy() {
		t.Fatal("an HTML 200 from another service on the port is not Colibri")
	}
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(`{"object":"list","data":[]}`)) }))
	defer api.Close()
	c.baseURL = api.URL + "/v1"
	if !c.Healthy() {
		t.Fatal("an OpenAI model list is healthy")
	}
}

func TestDefaultPortSkipsAPortInUse(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(DefaultColibriPort))
	if err != nil {
		t.Skipf("default port already busy on this machine: %v", err)
	}
	defer ln.Close()
	if p := freePort("127.0.0.1", DefaultColibriPort); p == DefaultColibriPort {
		t.Fatal("must not choose a port that is in use")
	}
	if NewColibri(ColibriConfig{Home: t.TempDir(), Port: 7777}).cfg.Port != 7777 {
		t.Fatal("an explicit port is kept")
	}
}

// A finished setup with its model files on disk must not run Colibri's setup, and so
// must not download the model, again.
func TestInstallSkipsSetupWhenModelAlreadyOnDisk(t *testing.T) {
	f := newColibriFixture(t, ColibriConfig{})
	ctx := context.Background()
	if err := f.c.Install(ctx); err != nil {
		t.Fatal(err)
	}
	// Stand in for the 19 GB the fake lists, as a sparse file.
	model := filepath.Join(f.c.modelsDir(), "small-model.bin")
	fh, err := os.Create(model)
	if err != nil {
		t.Fatal(err)
	}
	fh.Truncate(19e9)
	fh.Close()
	args := filepath.Join(f.c.stateDir(), "setup.args")

	os.Remove(args)
	if err := f.c.Install(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(args); err == nil {
		t.Fatal("setup ran again although the model is already installed")
	}
	if !strings.Contains(f.logs.String(), "not downloading again") {
		t.Fatalf("skip must be logged:\n%s", f.logs.String())
	}

	// A truncated model (interrupted or deleted files) resumes through setup.
	fh, _ = os.OpenFile(model, os.O_WRONLY, 0)
	fh.Truncate(1e9)
	fh.Close()
	if err := f.c.Install(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(args); err != nil {
		t.Fatal("short model files must run setup so the download resumes")
	}
}

func TestInstallRerunsSetupForADifferentModelOrNoRecord(t *testing.T) {
	f := newColibriFixture(t, ColibriConfig{})
	ctx := context.Background()
	if err := f.c.Install(ctx); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(f.c.modelsDir(), "x.bin"), []byte("x"), 0o644)
	args := filepath.Join(f.c.stateDir(), "setup.args")
	os.Remove(args)
	f.c.cfg.Model = "other-model"
	if err := f.c.Install(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(args); err != nil {
		t.Fatal("a different model must run setup")
	}
	os.Remove(f.c.recordPath())
	os.Remove(args)
	if err := f.c.Install(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(args); err != nil {
		t.Fatal("without a record setup must run so Colibri verifies the files")
	}
}

func TestPreparedAndStickyPortFollowTheInstallRecord(t *testing.T) {
	f := newColibriFixture(t, ColibriConfig{})
	if f.c.Prepared() {
		t.Fatal("nothing is installed yet")
	}
	if err := f.c.Install(context.Background()); err != nil {
		t.Fatal(err)
	}
	if f.c.Prepared() {
		t.Fatal("a record without model files is not prepared")
	}
	os.WriteFile(filepath.Join(f.c.modelsDir(), "weights.bin"), []byte("x"), 0o644)
	if !f.c.Prepared() {
		t.Fatal("a finished setup must be reported as prepared")
	}
	// A new instance over the same home picks the port the setup used (when free).
	free := freePort("127.0.0.1", 18400)
	f.c.cfg.Port = free
	if err := f.c.writeInstallRecord("small-model"); err != nil {
		t.Fatal(err)
	}
	if got := stickyPort(f.c.cfg.Home, "127.0.0.1"); got != free {
		t.Fatalf("port moved from %d to %d", free, got)
	}
	ln, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(free)))
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	if got := stickyPort(f.c.cfg.Home, "127.0.0.1"); got == free {
		t.Fatal("a port in use must not be reused")
	}
}

// What the served engine accepts comes from the arch its /health reports.
func TestFeaturesFollowTheServedEngineFamily(t *testing.T) {
	for arch, want := range map[string][2]bool{"qwen36": {false, false}, "glm": {true, true}, "kimi": {true, false}} {
		arch := arch
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`{"status":"ok","arch":"` + arch + `"}`))
		}))
		c := NewColibri(ColibriConfig{Home: t.TempDir(), Port: 8199})
		c.baseURL = srv.URL + "/v1"
		if c.SupportsTools() != want[0] || c.SupportsGrammar() != want[1] {
			t.Errorf("%s: tools=%v grammar=%v", arch, c.SupportsTools(), c.SupportsGrammar())
		}
		srv.Close()
	}
	c := NewColibri(ColibriConfig{Home: t.TempDir(), Port: 8199})
	if c.SupportsGrammar() || c.SupportsTools() {
		t.Fatal("a stopped server supports nothing")
	}
}

func writeCPUInfo(t *testing.T, model, flags string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "cpuinfo")
	if err := os.WriteFile(p, []byte("processor\t: 0\nmodel name\t: "+model+"\nflags\t\t: "+flags+"\n\nprocessor\t: 1\nmodel name\t: "+model+"\nflags\t\t: "+flags+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// ---- host requirements and the speed check ----

func TestHostWithoutAVX2IsRefusedBeforeAnythingIsDownloaded(t *testing.T) {
	f := newColibriFixture(t, ColibriConfig{CPUInfoPath: writeCPUInfo(t, "Intel(R) Core(TM) i7 CPU 950", "fpu sse4_2 popcnt")})
	err := f.c.Install(context.Background())
	if !IsPermanent(err) || !errors.Is(err, ErrHostTooWeak) {
		t.Fatalf("got %v", err)
	}
	for _, want := range []string{"i7 CPU 950", "no AVX2", "llama.cpp serves instead", "KNIRV_COLIBRI=force"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("message must say %q: %v", want, err)
		}
	}
	if f.downloads.Load() != 0 {
		t.Fatal("nothing may be downloaded for a host that cannot run it")
	}
}

func TestForceSkipsTheHostRequirement(t *testing.T) {
	f := newColibriFixture(t, ColibriConfig{Force: true, CPUInfoPath: writeCPUInfo(t, "Old CPU", "sse4_2")})
	if err := f.c.Install(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := f.c.Qualify(context.Background()); err != nil {
		t.Fatalf("forced hosts are not measured: %v", err)
	}
}

func TestUnreadableCPUInfoDoesNotBlock(t *testing.T) {
	f := newColibriFixture(t, ColibriConfig{CPUInfoPath: filepath.Join(t.TempDir(), "missing")})
	if err := f.c.Install(context.Background()); err != nil {
		t.Fatalf("an unreadable cpuinfo must not block: %v", err)
	}
}

func startedColibri(t *testing.T, cfg ColibriConfig) *colibriFixture {
	t.Helper()
	f := newColibriFixture(t, cfg)
	if err := f.c.Install(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := f.c.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	return f
}

func TestQualifyPassesAFastHostAndRemembersIt(t *testing.T) {
	f := startedColibri(t, ColibriConfig{MinTokensPerSec: 5})
	if err := f.c.Qualify(context.Background()); err != nil {
		t.Fatal(err)
	}
	// A second check must not talk to the model again: make it unusable.
	f.apiDelay.Store(int64(time.Hour))
	if err := f.c.Qualify(context.Background()); err != nil {
		t.Fatalf("a recorded pass must be reused: %v", err)
	}
	q := f.c.Qualification()
	if q["qualified"] != true || q["tokens_per_sec"].(float64) <= 0 {
		t.Fatalf("%v", q)
	}
}

func TestQualifyStopsAndRemembersASlowHost(t *testing.T) {
	// 16 tokens in 0.5s is 32 tokens/s; ask for far more than that.
	f := startedColibri(t, ColibriConfig{MinTokensPerSec: 100000})
	f.apiDelay.Store(int64(500 * time.Millisecond))
	err := f.c.Qualify(context.Background())
	if !IsPermanent(err) || !errors.Is(err, ErrHostTooWeak) || !strings.Contains(err.Error(), "tokens/s, below") {
		t.Fatalf("got %v", err)
	}
	if f.c.BaseURL() != "" {
		t.Fatal("a too-slow colibri must be stopped")
	}
	if !strings.Contains(f.logs.String(), "too slow") {
		t.Fatalf("must say why:\n%s", f.logs.String())
	}
	// The next start refuses before downloading or starting anything.
	if err := f.c.Install(context.Background()); !errors.Is(err, ErrHostTooWeak) {
		t.Fatalf("a recorded failure must stop the next install: %v", err)
	}
	if q := f.c.Qualification(); q["qualified"] != false || q["reason"] == "" {
		t.Fatalf("%v", q)
	}
	// Changing the minimum measures again.
	f.c.cfg.MinTokensPerSec = 0.001
	if err := f.c.Install(context.Background()); err != nil {
		t.Fatalf("a different minimum must be measured again: %v", err)
	}
}

func TestQualifyTimeoutCountsAsTooSlow(t *testing.T) {
	f := startedColibri(t, ColibriConfig{MinTokensPerSec: 1, WarmupTimeout: 200 * time.Millisecond})
	f.apiDelay.Store(int64(2 * time.Second))
	if err := f.c.Qualify(context.Background()); !errors.Is(err, ErrHostTooWeak) {
		t.Fatalf("got %v", err)
	}
}

func TestMinTPSFromEnvironment(t *testing.T) {
	t.Setenv("KNIRV_COLIBRI_MIN_TPS", "2.5")
	if c := NewColibri(ColibriConfig{Home: t.TempDir()}); c.minTPS() != 2.5 {
		t.Fatalf("%v", c.minTPS())
	}
	t.Setenv("KNIRV_COLIBRI_MIN_TPS", "0")
	if c := NewColibri(ColibriConfig{Home: t.TempDir()}); c.minTPS() != 0 {
		t.Fatal("0 must turn the speed check off")
	}
	t.Setenv("KNIRV_COLIBRI_MIN_TPS", "")
	if c := NewColibri(ColibriConfig{Home: t.TempDir()}); c.minTPS() != DefaultMinTokensPerSec {
		t.Fatal("default")
	}
}
