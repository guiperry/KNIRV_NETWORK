package install

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// refusalServer lays out a numeric-path install: a llama-server at bin/llama-server
// that reports build 6050, with no marker yet.
func refusalServer(t *testing.T) (dir, server string) {
	t.Helper()
	dir = t.TempDir()
	server = filepath.Join(dir, "bin", "llama-server")
	_ = os.MkdirAll(filepath.Dir(server), 0755)
	_ = os.WriteFile(server, []byte("numeric"), 0755)
	return dir, server
}

// artifactOutput reports build for a staged candidate and 6000 for the installed
// binary, so the candidate is the only thing that can be newer.
func artifactOutput(build string) func(context.Context, string, ...string) (string, error) {
	return func(_ context.Context, path string, _ ...string) (string, error) {
		if strings.HasSuffix(path, ".update") {
			return "version: " + build + " (x)\n", nil
		}
		return "version: 6000 (abc)\n", nil
	}
}

// failSourceBuild makes any git or cmake call fail the test: a source build must not
// start while the refusal holds.
func failSourceBuild(t *testing.T) func(context.Context, string, ...string) error {
	return func(_ context.Context, name string, args ...string) error {
		if name == "git" || name == "cmake" {
			t.Fatalf("source build started (%s %v); the refusal should hold", name, args)
		}
		return nil
	}
}

// releaseOnly answers the release lookup and fails any other plain GET.
func releaseOnly(t *testing.T, tag string, artifact func(string) (*http.Response, error)) func(string) (*http.Response, error) {
	return func(url string) (*http.Response, error) {
		if url == LatestReleaseURL {
			return releaseResponder(tag)(url)
		}
		if artifact == nil {
			t.Fatalf("unexpected plain GET of %s", url)
			return nil, nil
		}
		return artifact(url)
	}
}

// Issue 1 (probe M4): the server keeps the stored ETag across a re-upload of a fixed
// build. Within the interval the validator still holds the refusal; once checked_at
// is due, the artifact is compared in full and the fixed build is installed.
func TestNumericStableETagFixedArtifactPickedUpWhenDue(t *testing.T) {
	dir, server := refusalServer(t)
	old := []byte("#!/bin/sh\necho 'version: 6050 (x)'\n")
	fixed := []byte("#!/bin/sh\necho 'version: 6100 (fixed)'\n")
	writeRejectedFile(t, dir, "b6100", `"v1"`, "", sha(old), time.Now().Add(-25*time.Hour))
	i := New()
	i.Get = releaseOnly(t, "b6100", func(string) (*http.Response, error) {
		h := http.Header{}
		h.Set("ETag", `"v1"`)
		return &http.Response{StatusCode: 200, Status: "200 OK", Header: h, Body: io.NopCloser(bytes.NewReader(gzipBytes(t, fixed)))}, nil
	})
	i.Conditional = noNetwork2(t) // a due comparison is unconditional
	i.Output = artifactOutput("6100")
	i.LookPath = func(string) (string, error) { return "/usr/bin/x", nil }
	i.Run = failSourceBuild(t)
	i.updateServer(context.Background(), Options{DataDir: dir, ServerURL: "http://x/llama-server.gz"}, server)
	if b, _ := os.ReadFile(server); !bytes.Equal(b, fixed) {
		t.Fatal("fixed artifact behind a stable ETag was not installed once the recheck was due")
	}
	if readRejected(dir) != nil {
		t.Fatal("refusal marker still present after the fixed artifact was installed")
	}
}

// Issue 1: inside the recheck interval a stable validator still holds the refusal
// with a conditional GET, and neither a download nor a source build happens.
func TestNumericRefusalHoldsWithinIntervalWithoutBuild(t *testing.T) {
	dir, server := refusalServer(t)
	plain := []byte("#!/bin/sh\necho 'version: 6050 (x)'\n")
	writeRejectedFile(t, dir, "b6100", `"v1"`, "", sha(plain), time.Now())
	var sentINM []string
	i := New()
	i.Get = releaseOnly(t, "b6100", nil)
	i.Conditional = func(_ string, h http.Header) (*http.Response, error) {
		sentINM = append(sentINM, h.Get("If-None-Match"))
		return &http.Response{StatusCode: 304, Status: "304 Not Modified", Header: http.Header{}, Body: io.NopCloser(bytes.NewReader(nil))}, nil
	}
	i.Output = artifactOutput("6100")
	i.LookPath = func(string) (string, error) { return "/usr/bin/x", nil }
	i.Run = failSourceBuild(t)
	opts := Options{DataDir: dir, ServerURL: "http://x/llama-server.gz"}
	i.updateServer(context.Background(), opts, server)
	i.updateServer(context.Background(), opts, server)
	if len(sentINM) != 2 || sentINM[0] != `"v1"` || sentINM[1] != `"v1"` {
		t.Fatalf("conditional requests = %v; want two with If-None-Match v1", sentINM)
	}
	if b, _ := os.ReadFile(server); !bytes.Equal(b, []byte("numeric")) {
		t.Fatal("server replaced while the refusal held")
	}
}

// Issue 3 (kills m6b): the server ignores the conditional headers and answers 200
// with the same ETag. The body must not be read.
func TestNumericIgnoredConditionalSameETagNoBodyRead(t *testing.T) {
	dir, server := refusalServer(t)
	plain := []byte("#!/bin/sh\necho 'version: 6050 (x)'\n")
	writeRejectedFile(t, dir, "b6100", `"v1"`, "", sha(plain), time.Now())
	reads := 0
	i := New()
	i.Get = releaseOnly(t, "b6100", nil)
	i.Conditional = func(string, http.Header) (*http.Response, error) {
		h := http.Header{}
		h.Set("ETag", `"v1"`)
		return &http.Response{StatusCode: 200, Status: "200 OK", Header: h, Body: io.NopCloser(bodyCounter{bytes.NewReader(gzipBytes(t, plain)), &reads})}, nil
	}
	i.Output = artifactOutput("6100")
	i.LookPath = func(string) (string, error) { return "/usr/bin/x", nil }
	i.Run = failSourceBuild(t)
	i.updateServer(context.Background(), Options{DataDir: dir, ServerURL: "http://x/llama-server.gz"}, server)
	if reads != 0 {
		t.Fatalf("body read %d times for an unchanged ETag; want 0", reads)
	}
}

// Issue 3 (kills x2): a new ETag on bytes identical to the refusal. The marker takes
// the new validator, nothing is staged for verification, and the server stays.
func TestNumericNewETagSameBytesNoReinstall(t *testing.T) {
	dir, server := refusalServer(t)
	plain := []byte("#!/bin/sh\necho 'version: 6050 (x)'\n")
	writeRejectedFile(t, dir, "b6100", `"v1"`, "", sha(plain), time.Now())
	stagedChecks := 0
	i := New()
	i.Get = releaseOnly(t, "b6100", nil)
	i.Conditional = func(string, http.Header) (*http.Response, error) {
		h := http.Header{}
		h.Set("ETag", `"v2"`)
		return &http.Response{StatusCode: 200, Status: "200 OK", Header: h, Body: io.NopCloser(bytes.NewReader(gzipBytes(t, plain)))}, nil
	}
	i.Output = func(ctx context.Context, path string, args ...string) (string, error) {
		if strings.HasSuffix(path, ".update") {
			stagedChecks++
		}
		return artifactOutput("6050")(ctx, path, args...)
	}
	i.LookPath = func(string) (string, error) { return "/usr/bin/x", nil }
	i.Run = failSourceBuild(t)
	i.updateServer(context.Background(), Options{DataDir: dir, ServerURL: "http://x/llama-server.gz"}, server)
	if stagedChecks != 0 {
		t.Fatalf("identical bytes were staged and verified %d times; want 0", stagedChecks)
	}
	if b, _ := os.ReadFile(server); string(b) != "numeric" {
		t.Fatal("server replaced by identical bytes")
	}
	if m := readRejected(dir); m == nil || m.ETag != `"v2"` {
		t.Fatalf("marker = %+v; want the new ETag v2", m)
	}
}

// Issue 6: the candidate verifies, but it cannot be renamed into place. The bytes
// were not refused, so no refusal marker is written, and the next start downloads
// the artifact again and installs it once the obstruction is gone. The destination
// is a non-empty directory, which a file cannot replace whoever runs the test.
func TestRefusedCandidateRenameFailureRecordsNoRefusal(t *testing.T) {
	dir := t.TempDir()
	server := filepath.Join(dir, "srv", "llama-server")
	_ = os.MkdirAll(filepath.Dir(server), 0755)
	_ = os.WriteFile(server, []byte("numeric"), 0755)
	dest := filepath.Join(dir, "bin", "llama-server")
	if err := os.MkdirAll(filepath.Join(dest, "occupied"), 0755); err != nil {
		t.Fatal(err)
	}
	fixed := []byte("#!/bin/sh\necho 'version: 6100 (fixed)'\n")
	gets := 0
	i := New()
	i.Get = releaseOnly(t, "b6100", func(string) (*http.Response, error) {
		gets++
		h := http.Header{}
		h.Set("ETag", `"v1"`)
		return &http.Response{StatusCode: 200, Status: "200 OK", Header: h, Body: io.NopCloser(bytes.NewReader(gzipBytes(t, fixed)))}, nil
	})
	i.Conditional = func(string, http.Header) (*http.Response, error) {
		return &http.Response{StatusCode: 304, Status: "304 Not Modified", Header: http.Header{}, Body: io.NopCloser(bytes.NewReader(nil))}, nil
	}
	i.Output = artifactOutput("6100")
	i.LookPath = func(string) (string, error) { return "/usr/bin/x", nil }
	i.Run = func(_ context.Context, name string, args ...string) error {
		if name == "git" || name == "cmake" {
			return io.ErrUnexpectedEOF // the source fallback fails; the server must survive
		}
		return nil
	}
	opts := Options{DataDir: dir, ServerURL: "http://x/llama-server.gz"}
	i.updateServer(context.Background(), opts, server)
	if gets != 1 {
		t.Fatalf("first start downloaded the artifact %d times; want 1", gets)
	}
	// A rename failure is not a fault of the bytes, so no refusal is recorded.
	if m := readRejected(dir); m != nil {
		t.Fatalf("marker after rename failure = %+v; want none (the bytes were not refused)", m)
	}
	if b, _ := os.ReadFile(server); string(b) != "numeric" {
		t.Fatal("server replaced although the rename failed")
	}
	// Probe P1: the obstruction is removed and the same bytes are downloaded again
	// and installed, not held as refused.
	if err := os.RemoveAll(dest); err != nil {
		t.Fatal(err)
	}
	i.updateServer(context.Background(), opts, server)
	if gets != 2 {
		t.Fatalf("second start made %d artifact GETs; want 2 (one download per start)", gets)
	}
	b, err := os.ReadFile(dest)
	if err != nil || !bytes.Equal(b, fixed) {
		t.Fatalf("after the obstruction was removed, bin/llama-server = %q (err %v); want the verified artifact", b, err)
	}
}

// Issue 5: a successful source build of a tag clears the refused marker for that tag
// and leaves the marker of any other tag alone.
func TestInstallServerClearsRefusalForBuiltTag(t *testing.T) {
	sameTag := t.TempDir()
	_ = os.MkdirAll(filepath.Join(sameTag, "bin"), 0755)
	writeRejectedFile(t, sameTag, "b6100", `"v1"`, "", "abc", time.Now())
	buildTagForTest(t, sameTag, "b6100")
	if readRejected(sameTag) != nil {
		t.Fatal("refusal for b6100 still present after a successful b6100 source build")
	}

	otherTag := t.TempDir()
	_ = os.MkdirAll(filepath.Join(otherTag, "bin"), 0755)
	writeRejectedFile(t, otherTag, "b6200", `"v1"`, "", "abc", time.Now())
	buildTagForTest(t, otherTag, "b6100")
	if m := readRejected(otherTag); m == nil || m.Tag != "b6200" {
		t.Fatalf("refusal for b6200 = %+v after an unrelated b6100 build; want it kept", m)
	}
}

func buildTagForTest(t *testing.T, dataDir, tag string) {
	t.Helper()
	built := filepath.Join(dataDir, "llama.cpp", "build", "bin", "llama-server")
	i := New()
	i.LookPath = func(string) (string, error) { return "/usr/bin/x", nil }
	i.Output = func(context.Context, string, ...string) (string, error) { return "version: 6100 (head)\n", nil }
	i.Run = func(_ context.Context, name string, args ...string) error {
		if name == "cmake" && contains(args, "--build") {
			writeScript(t, built, "version: 6100 (head)")
		}
		return nil
	}
	if _, err := i.installServer(context.Background(), dataDir, tag); err != nil {
		t.Fatal(err)
	}
}

// Issue 4 (kills c8): KNIRV_LLAMA_CUDA=1 reaches the configure step as -DGGML_CUDA=ON.
func TestInstallServerConfigureUsesCUDAWhenForced(t *testing.T) {
	t.Setenv("KNIRV_LLAMA_CUDA", "1")
	dir := t.TempDir()
	built := filepath.Join(dir, "llama.cpp", "build", "bin", "llama-server")
	var configure []string
	i := New()
	i.LookPath = func(string) (string, error) { return "/usr/bin/x", nil }
	i.Output = func(context.Context, string, ...string) (string, error) { return "version: 6200 (head)\n", nil }
	i.Run = func(_ context.Context, name string, args ...string) error {
		if name == "cmake" && contains(args, "-B") {
			configure = args
		}
		if name == "cmake" && contains(args, "--build") {
			writeScript(t, built, "version: 6200 (head)")
		}
		return nil
	}
	if _, err := i.installServer(context.Background(), dir, ""); err != nil {
		t.Fatal(err)
	}
	if !contains(configure, "-DGGML_CUDA=ON") || contains(configure, "-DGGML_CUDA=OFF") {
		t.Fatalf("configure args = %v; want -DGGML_CUDA=ON", configure)
	}
}

// Issue 8: an unreadable last-check time is reported as due now, not as a next
// comparison at the current time.
func TestNextCheckNoteUnreadableTimestamp(t *testing.T) {
	note := nextCheckNote(installStamp{CheckedAt: "junk"})
	if strings.Contains(note, "next comparison after") || !strings.Contains(note, "due now") {
		t.Fatalf("note = %q; want a due-now message", note)
	}
	valid := nextCheckNote(installStamp{CheckedAt: "2026-11-01T00:00:00Z"})
	if !strings.Contains(valid, "next comparison after 2026-11-02T00:00:00Z") {
		t.Fatalf("note = %q; want next comparison 24h after the last one", valid)
	}
}

// Issue 10: a KNIRV_LLAMA_CUDA value other than 0 or 1 is logged once as ignored;
// 0 and 1 are not.
func TestInstallServerLogsUnrecognisedCUDAOverride(t *testing.T) {
	for _, c := range []struct {
		value   string
		wantLog bool
	}{{"maybe", true}, {"true", true}, {"1", false}, {"0", false}, {"", false}} {
		t.Setenv("KNIRV_LLAMA_CUDA", c.value)
		dir := t.TempDir()
		built := filepath.Join(dir, "llama.cpp", "build", "bin", "llama-server")
		var out bytes.Buffer
		i := New()
		i.Prog = NewProgress(&out)
		i.LookPath = func(string) (string, error) { return "/usr/bin/x", nil }
		i.Output = func(context.Context, string, ...string) (string, error) { return "version: 6200 (head)\n", nil }
		i.Run = func(_ context.Context, name string, args ...string) error {
			if name == "cmake" && contains(args, "--build") {
				writeScript(t, built, "version: 6200 (head)")
			}
			return nil
		}
		if _, err := i.installServer(context.Background(), dir, ""); err != nil {
			t.Fatal(err)
		}
		got := strings.Contains(out.String(), "is not 0 or 1")
		if got != c.wantLog {
			t.Errorf("KNIRV_LLAMA_CUDA=%q: logged=%v, want %v; output %q", c.value, got, c.wantLog, out.String())
		}
	}
}
