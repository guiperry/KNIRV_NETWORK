package install

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// writeScript writes an executable that prints version, standing in for a
// llama-server binary with that `--version` output.
func writeScript(t *testing.T, path, version string) []byte {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	body := "#!/bin/sh\necho '" + version + "'\n"
	if err := os.WriteFile(path, []byte(body), 0755); err != nil {
		t.Fatal(err)
	}
	return []byte(body)
}

func gzipBytes(t *testing.T, data []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	if _, err := w.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func sha(b []byte) string {
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}

// artifactServer serves a gzipped llama-server with an ETag and answers
// conditional GETs with 304, counting the full (200) downloads.
type artifactServer struct {
	etag string
	body []byte // gzipped
	full atomic.Int32
	srv  *httptest.Server
}

func newArtifactServer(t *testing.T, etag string, plain []byte) *artifactServer {
	t.Helper()
	a := &artifactServer{etag: etag, body: gzipBytes(t, plain)}
	a.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a.etag != "" && r.Header.Get("If-None-Match") == a.etag {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		a.full.Add(1)
		w.Header().Set("ETag", a.etag)
		_, _ = w.Write(a.body)
	}))
	t.Cleanup(a.srv.Close)
	return a
}

func (a *artifactServer) url() string { return a.srv.URL + "/llama-server.gz" }

// noNetwork makes any fetch or command in the installer a test failure.
func noNetwork(t *testing.T, i *Installer) {
	t.Helper()
	fail := func(what string) func(string) (*http.Response, error) {
		return func(url string) (*http.Response, error) {
			t.Fatalf("unexpected %s of %s", what, url)
			return nil, nil
		}
	}
	i.Get = fail("GET")
	i.Conditional = func(url string, _ http.Header) (*http.Response, error) {
		t.Fatalf("unexpected conditional GET of %s", url)
		return nil, nil
	}
	i.Run = func(_ context.Context, name string, args ...string) error {
		t.Fatalf("unexpected command %s %v", name, args)
		return nil
	}
}

const (
	tarballLine = "version: 1.0.0-dev (build 0, commit unknown)"
)

// An unknown build with no stamp, outside the installer's own directory, is kept
// on every start, with no fetch and no build.
func TestUnknownBuildNoStampKeptWithoutLoop(t *testing.T) {
	dir := t.TempDir()
	server := filepath.Join(dir, "tools", "llama-server")
	writeScript(t, server, tarballLine)
	i := New()
	noNetwork(t, i)
	for start := 0; start < 3; start++ {
		if got := i.updateServer(context.Background(), Options{DataDir: dir}, server); got != server {
			t.Fatalf("start %d: server = %s, want %s", start, got, server)
		}
	}
	if st, _ := readStamp(server); st != nil {
		t.Fatalf("a stamp was written for a binary nobody installed: %+v", st)
	}
}

// An unstamped binary in the installer's own bin directory (installed before
// stamps existed) is compared with the release artifact once. The same bytes are
// stamped and kept. Later starts inside the recheck interval make no request.
func TestUnstampedManagedServerAdoptedOnce(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", t.TempDir())
	plain := []byte("#!/bin/sh\necho '" + tarballLine + "'\n")
	server := filepath.Join(dir, "bin", "llama-server")
	if err := os.MkdirAll(filepath.Dir(server), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(server, plain, 0755); err != nil {
		t.Fatal(err)
	}
	a := newArtifactServer(t, "", plain)
	i := New()
	i.Get = func(url string) (*http.Response, error) { t.Fatalf("unexpected plain GET %s", url); return nil, nil }
	i.Run = func(_ context.Context, name string, args ...string) error {
		if name != server && !strings.HasSuffix(name, "llama-server.update") {
			t.Fatalf("unexpected command %s %v", name, args)
		}
		return nil
	}
	opts := Options{DataDir: dir, ServerURL: a.url()}
	if got := i.updateServer(context.Background(), opts, server); got != server {
		t.Fatalf("start 1: server = %s, want %s", got, server)
	}
	if n := a.full.Load(); n != 1 {
		t.Fatalf("start 1: full downloads = %d, want 1 (the adoption comparison)", n)
	}
	st, err := readStamp(server)
	if err != nil || st == nil || st.Source != sourcePrebuilt || st.SHA256 != sha(plain) || st.CheckedAt == "" {
		t.Fatalf("stamp after adoption = %+v, %v; want prebuilt with the sha of the installed bytes", st, err)
	}
	if got := i.updateServer(context.Background(), opts, server); got != server {
		t.Fatalf("start 2: server = %s, want %s", got, server)
	}
	if n := a.full.Load(); n != 1 {
		t.Fatalf("start 2: full downloads = %d, want still 1 inside the recheck interval", n)
	}
}

// A source build stamped with an old tag is rebuilt at the latest tag, and the
// new build is stamped.
func TestStampedOldGitBuildRebuilds(t *testing.T) {
	dir := t.TempDir()
	server := filepath.Join(dir, "bin", "llama-server")
	writeScript(t, server, tarballLine)
	if err := writeStamp(server, installStamp{Source: sourceGitBuild, Tag: "b6000"}); err != nil {
		t.Fatal(err)
	}
	i := New()
	pinUpstream(t, "b6100")
	i.Get = releaseResponder("b6100")
	i.Conditional = noNetwork2(t)
	i.LookPath = func(string) (string, error) { return "/usr/bin/x", nil }
	built := filepath.Join(dir, "llama.cpp", "build", "bin", "llama-server")
	var ranBuild bool
	i.Run = func(_ context.Context, name string, args ...string) error {
		if name == "cmake" && contains(args, "--build") {
			ranBuild = true
			writeScript(t, built, "version: 6100 (abc)")
		}
		return nil
	}
	got := i.updateServer(context.Background(), Options{DataDir: dir}, server)
	if !ranBuild {
		t.Fatal("stamped b6000 with b6100 latest: expected a source build")
	}
	if got != built {
		t.Fatalf("server = %s, want %s", got, built)
	}
	st, err := readStamp(got)
	if err != nil || st == nil || st.Source != sourceGitBuild || st.Tag != "b6100" {
		t.Fatalf("stamp after rebuild = %+v, %v; want git-build b6100", st, err)
	}
}

// A source build stamped with the latest tag is not rebuilt.
func TestStampedCurrentGitBuildNotRebuilt(t *testing.T) {
	dir := t.TempDir()
	server := filepath.Join(dir, "bin", "llama-server")
	writeScript(t, server, tarballLine)
	if err := writeStamp(server, installStamp{Source: sourceGitBuild, Tag: "b6100"}); err != nil {
		t.Fatal(err)
	}
	i := New()
	pinUpstream(t, "b6100")
	i.Get = releaseResponder("b6100")
	i.Conditional = noNetwork2(t)
	i.Run = func(_ context.Context, name string, args ...string) error {
		t.Fatalf("up to date: unexpected command %s %v", name, args)
		return nil
	}
	if got := i.updateServer(context.Background(), Options{DataDir: dir}, server); got != server {
		t.Fatalf("server = %s, want %s", got, server)
	}
}

// A prebuilt artifact whose ETag is unchanged is not downloaded again.
func TestPrebuiltUnchangedETagNotRedownloaded(t *testing.T) {
	dir := t.TempDir()
	plain := []byte("#!/bin/sh\necho '" + tarballLine + "'\n")
	server := filepath.Join(dir, "bin", "llama-server")
	if err := os.MkdirAll(filepath.Dir(server), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(server, plain, 0755); err != nil {
		t.Fatal(err)
	}
	a := newArtifactServer(t, `"v1"`, plain)
	if err := writeStamp(server, installStamp{Source: sourcePrebuilt, ETag: `"v1"`, SHA256: sha(plain)}); err != nil {
		t.Fatal(err)
	}
	i := New()
	i.Get = func(url string) (*http.Response, error) {
		t.Fatalf("unchanged artifact: unexpected full GET of %s", url)
		return nil, nil
	}
	i.Run = func(_ context.Context, name string, args ...string) error {
		t.Fatalf("unchanged artifact: unexpected command %s %v", name, args)
		return nil
	}
	if got := i.updateServer(context.Background(), Options{DataDir: dir, ServerURL: a.url()}, server); got != server {
		t.Fatalf("server = %s, want %s", got, server)
	}
	if n := a.full.Load(); n != 0 {
		t.Fatalf("full downloads = %d, want 0", n)
	}
}

// A prebuilt artifact whose ETag changed is replaced. The staged binary reports
// build 0, and it is accepted because the artifact changed, not by version.
func TestPrebuiltChangedETagReplacedWithBuildZero(t *testing.T) {
	dir := t.TempDir()
	old := []byte("#!/bin/sh\necho 'version: 1.0.0-dev (build 0, commit unknown)'\necho old\n")
	server := filepath.Join(dir, "bin", "llama-server")
	if err := os.MkdirAll(filepath.Dir(server), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(server, old, 0755); err != nil {
		t.Fatal(err)
	}
	if err := writeStamp(server, installStamp{Source: sourcePrebuilt, ETag: `"v1"`, SHA256: sha(old)}); err != nil {
		t.Fatal(err)
	}
	newBin := []byte("#!/bin/sh\necho 'version: 1.0.0-dev (build 0, commit unknown)'\necho new\n")
	a := newArtifactServer(t, `"v2"`, newBin)
	i := New()
	i.Get = func(string) (*http.Response, error) { t.Fatal("unexpected plain GET"); return nil, nil }
	i.Conditional = New().Conditional
	got := i.updateServer(context.Background(), Options{DataDir: dir, ServerURL: a.url()}, server)
	if want := filepath.Join(dir, "bin", "llama-server"); got != want {
		t.Fatalf("server = %s, want %s", got, want)
	}
	if b, _ := os.ReadFile(got); string(b) != string(newBin) {
		t.Fatalf("binary was not replaced; content = %q", b)
	}
	st, err := readStamp(got)
	if err != nil || st == nil || st.ETag != `"v2"` || st.SHA256 != sha(newBin) {
		t.Fatalf("stamp = %+v, %v; want etag v2 and the new sha", st, err)
	}
}

// A changed ETag whose bytes hash the same as the stamp is not a real change:
// nothing is replaced, and the validator is updated.
func TestPrebuiltSameBytesNewETagNotReplaced(t *testing.T) {
	dir := t.TempDir()
	plain := []byte("#!/bin/sh\necho 'version: 1.0.0-dev (build 0, commit unknown)'\n")
	server := filepath.Join(dir, "bin", "llama-server")
	if err := os.MkdirAll(filepath.Dir(server), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(server, plain, 0755); err != nil {
		t.Fatal(err)
	}
	if err := writeStamp(server, installStamp{Source: sourcePrebuilt, ETag: `"v1"`, SHA256: sha(plain)}); err != nil {
		t.Fatal(err)
	}
	before, err := os.Stat(server)
	if err != nil {
		t.Fatal(err)
	}
	a := newArtifactServer(t, `"v2"`, plain)
	i := New()
	i.Get = func(string) (*http.Response, error) { t.Fatal("unexpected plain GET"); return nil, nil }
	i.Conditional = New().Conditional
	if got := i.updateServer(context.Background(), Options{DataDir: dir, ServerURL: a.url()}, server); got != server {
		t.Fatalf("server = %s, want %s", got, server)
	}
	after, err := os.Stat(server)
	if err != nil || !os.SameFile(before, after) {
		t.Fatalf("binary was replaced (inode changed): %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "bin", "llama-server.update")); !os.IsNotExist(err) {
		t.Fatalf("staged download left behind: %v", err)
	}
	st, err := readStamp(server)
	if err != nil || st == nil || st.ETag != `"v2"` {
		t.Fatalf("stamp = %+v, %v; want the new etag recorded", st, err)
	}
}

// Corrupt or unrecognised stamps are treated as absent: the binary is kept and
// nothing is fetched.
func TestCorruptStampTreatedAsAbsent(t *testing.T) {
	for name, content := range map[string]string{
		"truncated json": `{"source":"prebuilt","etag":`,
		"unknown source": `{"source":"magic","tag":"b9999"}`,
	} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			server := filepath.Join(dir, "tools", "llama-server")
			writeScript(t, server, tarballLine)
			if err := os.WriteFile(stampPath(server), []byte(content), 0644); err != nil {
				t.Fatal(err)
			}
			i := New()
			noNetwork(t, i)
			if got := i.updateServer(context.Background(), Options{DataDir: dir}, server); got != server {
				t.Fatalf("server = %s, want %s", got, server)
			}
		})
	}
}

// A prebuilt stamp with no validators, compared recently, is kept with no fetch.
func TestPrebuiltStampWithoutValidatorsKept(t *testing.T) {
	dir := t.TempDir()
	server := filepath.Join(dir, "bin", "llama-server")
	writeScript(t, server, tarballLine)
	recent := time.Now().UTC().Format(time.RFC3339)
	if err := writeStamp(server, installStamp{Source: sourcePrebuilt, SHA256: "abc", CheckedAt: recent}); err != nil {
		t.Fatal(err)
	}
	i := New()
	noNetwork(t, i)
	if got := i.updateServer(context.Background(), Options{DataDir: dir}, server); got != server {
		t.Fatalf("server = %s, want %s", got, server)
	}
}

// Installing a prebuilt through Ensure records the stamp with the artifact
// validators and the sha256 of the installed bytes.
func TestEnsureDownloadWritesStamp(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", t.TempDir())
	plain := []byte("#!/bin/sh\necho 'version: 1.0.0-dev (build 0, commit unknown)'\n")
	a := newArtifactServer(t, `"v1"`, plain)
	model := filepath.Join(dir, "model.gguf")
	if err := os.WriteFile(model, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	i := New()
	res, err := i.Ensure(context.Background(), Options{DataDir: dir, ServerURL: a.url(), ModelPath: model})
	if err != nil {
		t.Fatal(err)
	}
	st, err := readStamp(res.ServerPath)
	if err != nil || st == nil {
		t.Fatalf("stamp = %+v, %v", st, err)
	}
	if st.Source != sourcePrebuilt || st.ETag != `"v1"` || st.SHA256 != sha(plain) {
		t.Fatalf("stamp = %+v; want prebuilt, etag v1, sha of the installed bytes", st)
	}
}

// The stamp is replaced by rename, leaving no temp files behind.
func TestWriteStampLeavesNoTempFiles(t *testing.T) {
	dir := t.TempDir()
	server := filepath.Join(dir, "llama-server")
	for n := 0; n < 3; n++ {
		if err := writeStamp(server, installStamp{Source: sourceGitBuild, Tag: "b1"}); err != nil {
			t.Fatal(err)
		}
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".tmp") {
			t.Fatalf("temp file left behind: %s", e.Name())
		}
	}
	if st, err := readStamp(server); err != nil || st == nil || st.Tag != "b1" {
		t.Fatalf("stamp = %+v, %v", st, err)
	}
}

// A user-supplied server path with updates disabled is never fetched or replaced.
func TestUserSuppliedServerNeverReplaced(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", t.TempDir())
	server := filepath.Join(dir, "my-llama-server")
	writeScript(t, server, tarballLine)
	model := filepath.Join(dir, "model.gguf")
	if err := os.WriteFile(model, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	i := New()
	noNetwork(t, i)
	res, err := i.Ensure(context.Background(), Options{DataDir: dir, ServerPath: server, ModelPath: model, CheckUpdate: false})
	if err != nil {
		t.Fatal(err)
	}
	if res.ServerPath != server {
		t.Fatalf("server = %s, want %s", res.ServerPath, server)
	}
}

// noNetwork2 returns a conditional GET that fails the test, for tests that only
// expect the release list.
func noNetwork2(t *testing.T) func(string, http.Header) (*http.Response, error) {
	return func(url string, _ http.Header) (*http.Response, error) {
		t.Fatalf("unexpected conditional GET of %s", url)
		return nil, io.ErrUnexpectedEOF
	}
}

func contains(xs []string, want string) bool {
	for _, x := range xs {
		if x == want {
			return true
		}
	}
	return false
}

// A 200 with no validators must not erase the stamp's validators. Same bytes
// keep the ETag the stamp already had.
func TestHeaderlessSameBytesKeepsValidators(t *testing.T) {
	dir := t.TempDir()
	plain := []byte("#!/bin/sh\necho '" + tarballLine + "'\n")
	server := filepath.Join(dir, "bin", "llama-server")
	if err := os.MkdirAll(filepath.Dir(server), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(server, plain, 0755); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-48 * time.Hour).UTC().Format(time.RFC3339)
	if err := writeStamp(server, installStamp{Source: sourcePrebuilt, ETag: `"v1"`, SHA256: sha(plain), CheckedAt: old}); err != nil {
		t.Fatal(err)
	}
	a := newArtifactServer(t, "", plain) // headerless 200 every time
	i := New()
	i.Get = func(string) (*http.Response, error) { t.Fatal("unexpected plain GET"); return nil, nil }
	i.Run = func(context.Context, string, ...string) error { return nil }
	if got := i.updateServer(context.Background(), Options{DataDir: dir, ServerURL: a.url()}, server); got != server {
		t.Fatalf("server = %s, want %s", got, server)
	}
	st, err := readStamp(server)
	if err != nil || st == nil || st.ETag != `"v1"` || st.SHA256 != sha(plain) {
		t.Fatalf("stamp = %+v, %v; want the v1 validator kept", st, err)
	}
	if st.CheckedAt == old {
		t.Fatalf("CheckedAt not advanced after a full comparison")
	}
}

// The probe from validation: a headerless artifact replaces the binary, and the
// binary is later replaced by an artifact with a validator. The change must be
// picked up within the recheck interval, not skipped forever.
func TestHeaderlessChangeRecoveredByLaterComparison(t *testing.T) {
	dir := t.TempDir()
	v2 := []byte("#!/bin/sh\necho '" + tarballLine + "'\necho v2\n")
	v3 := []byte("#!/bin/sh\necho '" + tarballLine + "'\necho v3\n")
	server := filepath.Join(dir, "bin", "llama-server")
	if err := os.MkdirAll(filepath.Dir(server), 0755); err != nil {
		t.Fatal(err)
	}
	v1 := []byte("#!/bin/sh\necho '" + tarballLine + "'\necho v1\n")
	if err := os.WriteFile(server, v1, 0755); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-48 * time.Hour).UTC().Format(time.RFC3339)
	if err := writeStamp(server, installStamp{Source: sourcePrebuilt, ETag: `"v1"`, SHA256: sha(v1), CheckedAt: old}); err != nil {
		t.Fatal(err)
	}
	a := newArtifactServer(t, "", v2) // headerless: the artifact changed to v2
	i := New()
	i.Get = func(string) (*http.Response, error) { t.Fatal("unexpected plain GET"); return nil, nil }
	i.Run = func(context.Context, string, ...string) error { return nil }
	opts := Options{DataDir: dir, ServerURL: a.url()}
	if got := i.updateServer(context.Background(), opts, server); got != server {
		t.Fatalf("start 1: server = %s, want %s", got, server)
	}
	if b, _ := os.ReadFile(server); string(b) != string(v2) {
		t.Fatalf("start 1: binary not replaced by the changed headerless artifact")
	}
	// The artifact is now published with a validator, and the stamp's check is old.
	a.etag, a.body = `"v3"`, gzipBytes(t, v3)
	st, _ := readStamp(server)
	aged := *st
	aged.CheckedAt = time.Now().Add(-48 * time.Hour).UTC().Format(time.RFC3339)
	if err := writeStamp(server, aged); err != nil {
		t.Fatal(err)
	}
	if got := i.updateServer(context.Background(), opts, server); got != server {
		t.Fatalf("start 2: server = %s, want %s", got, server)
	}
	if b, _ := os.ReadFile(server); string(b) != string(v3) {
		t.Fatalf("start 2: the later v3 artifact was not installed")
	}
	st, err := readStamp(server)
	if err != nil || st == nil || st.ETag != `"v3"` {
		t.Fatalf("stamp = %+v, %v; want etag v3", st, err)
	}
}

// A validator change inside the recheck interval is not downloaded. The next
// comparison after the interval applies it.
func TestChangedValidatorInsideIntervalWaits(t *testing.T) {
	dir := t.TempDir()
	v1 := []byte("#!/bin/sh\necho '" + tarballLine + "'\necho v1\n")
	v2 := []byte("#!/bin/sh\necho '" + tarballLine + "'\necho v2\n")
	server := filepath.Join(dir, "bin", "llama-server")
	if err := os.MkdirAll(filepath.Dir(server), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(server, v1, 0755); err != nil {
		t.Fatal(err)
	}
	recent := time.Now().UTC().Format(time.RFC3339)
	if err := writeStamp(server, installStamp{Source: sourcePrebuilt, ETag: `"v1"`, SHA256: sha(v1), CheckedAt: recent}); err != nil {
		t.Fatal(err)
	}
	a := newArtifactServer(t, `"v2"`, v2)
	i := New()
	i.Get = func(string) (*http.Response, error) { t.Fatal("unexpected plain GET"); return nil, nil }
	i.Run = func(context.Context, string, ...string) error { return nil }
	if got := i.updateServer(context.Background(), Options{DataDir: dir, ServerURL: a.url()}, server); got != server {
		t.Fatalf("server = %s, want %s", got, server)
	}
	if b, _ := os.ReadFile(server); string(b) != string(v1) {
		t.Fatalf("binary replaced inside the recheck interval")
	}
}

// Test for the validator's mutant m4: a numeric install must not accept a staged
// artifact that reports build 0. It falls back to a source build, and the
// installed binary is left alone.
func TestNumericInstallRejectsStagedBuildZero(t *testing.T) {
	dir := t.TempDir()
	server := filepath.Join(dir, "bin", "llama-server")
	if err := os.MkdirAll(filepath.Dir(server), 0755); err != nil {
		t.Fatal(err)
	}
	numeric := []byte("numeric-6000")
	if err := os.WriteFile(server, numeric, 0755); err != nil {
		t.Fatal(err)
	}
	artifact := []byte("#!/bin/sh\necho '" + tarballLine + "'\n")
	i := New()
	pinUpstream(t, "b6100")
	i.Get = releaseResponderWithArtifact("b6100", gzipBytes(t, artifact))
	i.Output = func(_ context.Context, path string, _ ...string) (string, error) {
		if strings.HasSuffix(path, ".update") {
			return tarballLine + "\n", nil
		}
		return "version: 6000 (abc)\n", nil
	}
	i.LookPath = func(string) (string, error) { return "/usr/bin/x", nil }
	built := filepath.Join(dir, "llama.cpp", "build", "bin", "llama-server")
	var ranBuild bool
	i.Run = func(_ context.Context, name string, args ...string) error {
		if name == "cmake" && contains(args, "--build") {
			ranBuild = true
			writeScript(t, built, "version: 6100 (new)")
		}
		return nil
	}
	got := i.updateServer(context.Background(), Options{DataDir: dir, ServerURL: "https://example.test/llama-server.gz"}, server)
	if !ranBuild {
		t.Fatal("numeric 6000 with a build-0 artifact: expected a source build")
	}
	if got != built {
		t.Fatalf("server = %s, want the source build %s", got, built)
	}
	if b, _ := os.ReadFile(server); string(b) != string(numeric) {
		t.Fatalf("installed numeric build was replaced by a build-0 artifact")
	}
}

// releaseResponderWithArtifact serves the release list and a gzipped artifact.
func releaseResponderWithArtifact(_ string, artifact []byte) func(string) (*http.Response, error) {
	return func(url string) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Status: "200 OK", Body: io.NopCloser(bytes.NewReader(artifact))}, nil
	}
}

// A default-branch build records the build number it reports, not an empty tag.
func TestDefaultBranchBuildStampedWithReportedBuild(t *testing.T) {
	dir := t.TempDir()
	built := filepath.Join(dir, "llama.cpp", "build", "bin", "llama-server")
	i := New()
	i.LookPath = func(string) (string, error) { return "/usr/bin/x", nil }
	i.Run = func(_ context.Context, name string, args ...string) error {
		if name == "cmake" && contains(args, "--build") {
			writeScript(t, built, "version: 6200 (head)")
		}
		return nil
	}
	i.Output = func(context.Context, string, ...string) (string, error) { return "version: 6200 (head)\n", nil }
	if _, err := i.installServer(context.Background(), dir, ""); err != nil {
		t.Fatal(err)
	}
	st, err := readStamp(built)
	if err != nil || st == nil || st.Tag != "b6200" {
		t.Fatalf("stamp = %+v, %v; want tag b6200", st, err)
	}
}

// A git-build stamp with no resolvable tag is rebuilt once at the latest tag.
func TestStampWithoutTagRebuildsAtLatest(t *testing.T) {
	dir := t.TempDir()
	server := filepath.Join(dir, "bin", "llama-server")
	writeScript(t, server, tarballLine)
	if err := writeStamp(server, installStamp{Source: sourceGitBuild}); err != nil {
		t.Fatal(err)
	}
	i := New()
	pinUpstream(t, "b6100")
	i.Get = releaseResponder("b6100")
	i.Conditional = noNetwork2(t)
	i.LookPath = func(string) (string, error) { return "/usr/bin/x", nil }
	built := filepath.Join(dir, "llama.cpp", "build", "bin", "llama-server")
	i.Run = func(_ context.Context, name string, args ...string) error {
		if name == "cmake" && contains(args, "--build") {
			writeScript(t, built, "version: 6100 (abc)")
		}
		return nil
	}
	if got := i.updateServer(context.Background(), Options{DataDir: dir}, server); got != built {
		t.Fatalf("server = %s, want %s", got, built)
	}
	if st, _ := readStamp(built); st == nil || st.Tag != "b6100" {
		t.Fatalf("stamp = %+v; want b6100", st)
	}
}

// A `--version` line with a bare build number parses, and a semver never does.
func TestParseBuildBareNumber(t *testing.T) {
	if got, err := parseBuild("version: 7780\n"); err != nil || got != 7780 {
		t.Fatalf("bare 7780: got %d, %v", got, err)
	}
	if _, err := parseBuild("version: 1.0.0-dev\n"); err == nil || errors.Is(err, errUnknownBuild) {
		t.Fatalf("semver without build: err = %v, want a plain parse error", err)
	}
}

// A failed stamp write leaves the previous stamp intact and no temp file behind.
func TestFailedStampWriteKeepsPreviousStamp(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("directory permissions do not stop root")
	}
	dir := t.TempDir()
	server := filepath.Join(dir, "llama-server")
	if err := writeStamp(server, installStamp{Source: sourceGitBuild, Tag: "b1"}); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dir, 0500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0755) })
	if err := writeStamp(server, installStamp{Source: sourceGitBuild, Tag: "b2"}); err == nil {
		t.Fatal("write into a read-only directory succeeded")
	}
	if err := os.Chmod(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if st, err := readStamp(server); err != nil || st == nil || st.Tag != "b1" {
		t.Fatalf("stamp = %+v, %v; want the previous b1", st, err)
	}
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".tmp") {
			t.Fatalf("temp file left behind: %s", e.Name())
		}
	}
}

// Fix 1 (validator issue 1, probe P2): the stamp has only Last-Modified. The
// server now sends an ETag with the same Last-Modified and new bytes. Equal
// Last-Modified alone must not count as unchanged, so the new build is installed.
func TestLastModifiedOnlyChangedETagIsReplaced(t *testing.T) {
	dir := t.TempDir()
	oldBin := []byte("#!/bin/sh\necho '" + tarballLine + "'\n")
	newBin := []byte("#!/bin/sh\necho 'version: 6200 (new)'\n")
	server := filepath.Join(dir, "bin", "llama-server")
	_ = os.MkdirAll(filepath.Dir(server), 0755)
	_ = os.WriteFile(server, oldBin, 0755)
	lm := "Mon, 02 Nov 2026 00:00:00 GMT"
	old := time.Now().Add(-48 * time.Hour).UTC().Format(time.RFC3339)
	if err := writeStamp(server, installStamp{Source: sourcePrebuilt, LastModified: lm, SHA256: sha(oldBin), CheckedAt: old}); err != nil {
		t.Fatal(err)
	}
	gz := gzipBytes(t, newBin)
	i := New()
	i.Get = func(string) (*http.Response, error) { t.Fatal("unexpected plain GET"); return nil, nil }
	i.Conditional = func(_ string, h http.Header) (*http.Response, error) {
		if h.Get("If-None-Match") == `"v2"` {
			return &http.Response{StatusCode: 304, Status: "304 Not Modified", Header: http.Header{}, Body: io.NopCloser(bytes.NewReader(nil))}, nil
		}
		hd := http.Header{}
		hd.Set("ETag", `"v2"`)
		hd.Set("Last-Modified", lm)
		return &http.Response{StatusCode: 200, Status: "200 OK", Header: hd, Body: io.NopCloser(bytes.NewReader(gz))}, nil
	}
	i.Run = func(context.Context, string, ...string) error { return nil }
	i.Output = func(_ context.Context, path string, _ ...string) (string, error) {
		if strings.HasSuffix(path, ".update") {
			return "version: 6200 (new)\n", nil
		}
		return tarballLine + "\n", nil
	}
	for start := 1; start <= 2; start++ {
		i.updateServer(context.Background(), Options{DataDir: dir, ServerURL: "http://x/llama-server.gz"}, server)
	}
	if b, _ := os.ReadFile(server); !bytes.Equal(b, newBin) {
		t.Fatal("changed artifact (same Last-Modified, new ETag) not installed after two starts")
	}
}

// Fix 2 (validator issue 2, probe P1): the server ignores If-None-Match but repeats
// the stamp's ETag. The artifact is unchanged, so its body must not be read.
func TestIgnoredConditionalSameETagNoBodyRead(t *testing.T) {
	dir := t.TempDir()
	plain := []byte("#!/bin/sh\necho '" + tarballLine + "'\n")
	server := filepath.Join(dir, "bin", "llama-server")
	_ = os.MkdirAll(filepath.Dir(server), 0755)
	_ = os.WriteFile(server, plain, 0755)
	old := time.Now().Add(-48 * time.Hour).UTC().Format(time.RFC3339)
	if err := writeStamp(server, installStamp{Source: sourcePrebuilt, ETag: `"v1"`, SHA256: sha(plain), CheckedAt: old}); err != nil {
		t.Fatal(err)
	}
	gz := gzipBytes(t, plain)
	reads := 0
	i := New()
	i.Get = func(string) (*http.Response, error) { t.Fatal("unexpected plain GET"); return nil, nil }
	i.Conditional = func(string, http.Header) (*http.Response, error) {
		h := http.Header{}
		h.Set("ETag", `"v1"`)
		return &http.Response{StatusCode: 200, Status: "200 OK", Header: h, Body: io.NopCloser(bodyCounter{bytes.NewReader(gz), &reads})}, nil
	}
	i.Run = func(context.Context, string, ...string) error { return nil }
	i.updateServer(context.Background(), Options{DataDir: dir, ServerURL: "http://x/llama-server.gz"}, server)
	if reads != 0 {
		t.Fatalf("body read %d times for an unchanged ETag; want 0 (no full download)", reads)
	}
}

// bodyCounter counts Read calls on the body, so a test can tell whether a
// response body was downloaded at all.
type bodyCounter struct {
	r io.Reader
	n *int
}

func (c bodyCounter) Read(p []byte) (int, error) { *c.n++; return c.r.Read(p) }

// Fix 3: the numeric path. The artifact trails upstream (reports 6050, latest
// b6100) and is refused. Later starts must ask with the stored validators and
// must not download it again while the validator is unchanged.
func TestNumericRefusedArtifactNotRefetchedEachStart(t *testing.T) {
	dir := t.TempDir()
	server := filepath.Join(dir, "bin", "llama-server")
	_ = os.MkdirAll(filepath.Dir(server), 0755)
	_ = os.WriteFile(server, []byte("numeric"), 0755)
	artifact := gzipBytes(t, []byte("#!/bin/sh\necho 'version: 6050 (x)'\n"))
	artifactGets, conditionals := 0, 0
	var sentINM []string
	pinUpstream(t, "b6100")
	i := New()
	i.Get = func(url string) (*http.Response, error) {
		artifactGets++
		h := http.Header{}
		h.Set("ETag", `"v1"`)
		return &http.Response{StatusCode: 200, Status: "200 OK", Header: h, Body: io.NopCloser(bytes.NewReader(artifact))}, nil
	}
	i.Conditional = func(url string, h http.Header) (*http.Response, error) {
		conditionals++
		sentINM = append(sentINM, h.Get("If-None-Match"))
		return &http.Response{StatusCode: 304, Status: "304 Not Modified", Header: http.Header{}, Body: io.NopCloser(bytes.NewReader(nil))}, nil
	}
	i.Output = func(_ context.Context, path string, _ ...string) (string, error) {
		if strings.HasSuffix(path, ".update") {
			return "version: 6050 (x)\n", nil
		}
		return "version: 6000 (abc)\n", nil
	}
	i.LookPath = func(string) (string, error) { return "/usr/bin/x", nil }
	i.Run = func(_ context.Context, name string, args ...string) error {
		if name == "git" || name == "cmake" {
			return io.ErrUnexpectedEOF
		}
		return nil
	}
	for start := 1; start <= 3; start++ {
		i.updateServer(context.Background(), Options{DataDir: dir, ServerURL: "http://x/llama-server.gz"}, server)
	}
	if artifactGets != 1 {
		t.Fatalf("artifact downloaded %d times over 3 starts; want 1 (first start only)", artifactGets)
	}
	if conditionals != 2 {
		t.Fatalf("conditional GETs = %d; want 2 (starts 2 and 3)", conditionals)
	}
	for _, v := range sentINM {
		if v != `"v1"` {
			t.Fatalf("If-None-Match = %q; want the refused artifact's ETag", v)
		}
	}
}

// Fix 3: a refused artifact is retried when the validator changes.
func TestNumericRefusedArtifactRetriedWhenValidatorChanges(t *testing.T) {
	dir := t.TempDir()
	server := filepath.Join(dir, "bin", "llama-server")
	_ = os.MkdirAll(filepath.Dir(server), 0755)
	_ = os.WriteFile(server, []byte("numeric"), 0755)
	oldPlain := []byte("#!/bin/sh\necho 'version: 6050 (x)'\n")
	newPlain := []byte("#!/bin/sh\necho 'version: 6100 (new)'\n")
	writeRejectedFile(t, dir, "b6100", `"v1"`, "", sha(oldPlain), time.Now())
	pinUpstream(t, "b6100")
	i := New()
	i.Get = func(url string) (*http.Response, error) {
		t.Fatal("plain GET of the artifact: the validator-changed retry must be conditional first")
		return nil, nil
	}
	i.Conditional = func(_ string, h http.Header) (*http.Response, error) {
		if h.Get("If-None-Match") != `"v1"` {
			t.Fatalf("If-None-Match = %q; want v1", h.Get("If-None-Match"))
		}
		hd := http.Header{}
		hd.Set("ETag", `"v2"`)
		return &http.Response{StatusCode: 200, Status: "200 OK", Header: hd, Body: io.NopCloser(bytes.NewReader(gzipBytes(t, newPlain)))}, nil
	}
	i.Output = func(_ context.Context, path string, _ ...string) (string, error) {
		if strings.HasSuffix(path, ".update") {
			return "version: 6100 (new)\n", nil
		}
		return "version: 6000 (abc)\n", nil
	}
	i.LookPath = func(string) (string, error) { return "/usr/bin/x", nil }
	i.Run = func(_ context.Context, name string, args ...string) error {
		if name == "cmake" {
			t.Fatal("a changed artifact that reports b6100 must install, not build")
		}
		return nil
	}
	got := i.updateServer(context.Background(), Options{DataDir: dir, ServerURL: "http://x/llama-server.gz"}, server)
	if got != server {
		t.Fatalf("server = %s, want %s", got, server)
	}
	if b, _ := os.ReadFile(server); !bytes.Equal(b, newPlain) {
		t.Fatal("changed artifact that reports b6100 was not installed")
	}
}

// Fix 3: a refused artifact is retried when upstream publishes a newer tag.
func TestNumericRefusedArtifactRetriedWhenTagChanges(t *testing.T) {
	dir := t.TempDir()
	server := filepath.Join(dir, "bin", "llama-server")
	_ = os.MkdirAll(filepath.Dir(server), 0755)
	_ = os.WriteFile(server, []byte("numeric"), 0755)
	oldPlain := []byte("#!/bin/sh\necho 'version: 6050 (x)'\n")
	newPlain := []byte("#!/bin/sh\necho 'version: 6200 (new)'\n")
	writeRejectedFile(t, dir, "b6100", `"v1"`, "", sha(oldPlain), time.Now())
	gets := 0
	pinUpstream(t, "b6200")
	i := New()
	i.Get = func(url string) (*http.Response, error) {
		gets++
		return &http.Response{StatusCode: 200, Status: "200 OK", Body: io.NopCloser(bytes.NewReader(gzipBytes(t, newPlain)))}, nil
	}
	i.Conditional = noNetwork2(t)
	i.Output = func(_ context.Context, path string, _ ...string) (string, error) {
		if strings.HasSuffix(path, ".update") {
			return "version: 6200 (new)\n", nil
		}
		return "version: 6000 (abc)\n", nil
	}
	i.LookPath = func(string) (string, error) { return "/usr/bin/x", nil }
	i.Run = func(context.Context, string, ...string) error { return nil }
	i.updateServer(context.Background(), Options{DataDir: dir, ServerURL: "http://x/llama-server.gz"}, server)
	if gets != 1 {
		t.Fatalf("artifact GETs = %d; want 1 (new tag retries the artifact)", gets)
	}
	if b, _ := os.ReadFile(server); !bytes.Equal(b, newPlain) {
		t.Fatal("artifact for the newer tag was not installed")
	}
}

// Fix 3: without validators a refused artifact is compared again only when the
// recheck interval has passed. Same bytes then update the comparison time, and the
// next start within the interval makes no request.
func TestNumericRefusedArtifactWithoutValidatorsRecheckedOncePerInterval(t *testing.T) {
	dir := t.TempDir()
	server := filepath.Join(dir, "bin", "llama-server")
	_ = os.MkdirAll(filepath.Dir(server), 0755)
	_ = os.WriteFile(server, []byte("numeric"), 0755)
	plain := []byte("#!/bin/sh\necho 'version: 6050 (x)'\n")
	writeRejectedFile(t, dir, "b6100", "", "", sha(plain), time.Now().Add(-48*time.Hour))
	gets := 0
	pinUpstream(t, "b6100")
	i := New()
	i.Get = func(url string) (*http.Response, error) {
		gets++
		return &http.Response{StatusCode: 200, Status: "200 OK", Header: http.Header{}, Body: io.NopCloser(bytes.NewReader(gzipBytes(t, plain)))}, nil
	}
	i.Conditional = noNetwork2(t)
	i.Output = func(_ context.Context, path string, _ ...string) (string, error) {
		if strings.HasSuffix(path, ".update") {
			return "version: 6050 (x)\n", nil
		}
		return "version: 6000 (abc)\n", nil
	}
	i.LookPath = func(string) (string, error) { return "/usr/bin/x", nil }
	i.Run = func(_ context.Context, name string, args ...string) error {
		if name == "git" || name == "cmake" {
			return io.ErrUnexpectedEOF
		}
		return nil
	}
	opts := Options{DataDir: dir, ServerURL: "http://x/llama-server.gz"}
	i.updateServer(context.Background(), opts, server)
	if gets != 1 {
		t.Fatalf("after the interval: artifact downloads = %d, want 1", gets)
	}
	st := readRejectedFile(t, dir)
	if when, err := time.Parse(time.RFC3339, st.CheckedAt); err != nil || time.Since(when) > time.Minute {
		t.Fatalf("checked_at = %q after an identical comparison; want now", st.CheckedAt)
	}
	i.Conditional = noNetwork2(t)
	i.updateServer(context.Background(), opts, server)
}

// writeRejectedFile writes a refused-artifact marker the way a previous start would.
func writeRejectedFile(t *testing.T, dataDir, tag, etag, lastModified, sum string, checked time.Time) {
	t.Helper()
	b, err := json.Marshal(map[string]string{
		"tag": tag, "etag": etag, "last_modified": lastModified, "sha256": sum,
		"reason": "test", "checked_at": checked.UTC().Format(time.RFC3339),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "bin", "llama-server.rejected"), b, 0644); err != nil {
		t.Fatal(err)
	}
}

type rejectedFileJSON struct {
	Tag       string `json:"tag"`
	CheckedAt string `json:"checked_at"`
}

func readRejectedFile(t *testing.T, dataDir string) rejectedFileJSON {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dataDir, "bin", "llama-server.rejected"))
	if err != nil {
		t.Fatalf("refused-artifact marker missing: %v", err)
	}
	var r rejectedFileJSON
	if err := json.Unmarshal(b, &r); err != nil {
		t.Fatal(err)
	}
	return r
}
