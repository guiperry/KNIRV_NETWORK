package install

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// bbResp builds a canned HTTP response for the build-bound tests.
func bbResp(status int, etag string, body []byte) *http.Response {
	h := http.Header{}
	if etag != "" {
		h.Set("ETag", etag)
	}
	return &http.Response{StatusCode: status, Status: fmt.Sprintf("%d %s", status, http.StatusText(status)), Header: h, Body: io.NopCloser(bytes.NewReader(body))}
}

// bbRun counts source-build starts (git) and makes a staged candidate fail its
// --version check, so a broken re-upload is refused as the numeric path refuses it.
func bbRun(builds *[]string) func(context.Context, string, ...string) error {
	return func(_ context.Context, name string, _ ...string) error {
		if name == "git" {
			*builds = append(*builds, name)
		}
		if name == "git" || name == "cmake" {
			return io.ErrUnexpectedEOF
		}
		if strings.HasSuffix(name, ".update") {
			return errors.New("exec format error")
		}
		return nil
	}
}

// writeBuildAttempts writes the per-tag source-build sidecar for a test.
func writeBuildAttempts(t *testing.T, dataDir string, attempts map[string]string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dataDir, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := writeJSONAtomic(buildAttemptsPath(dataDir), attempts); err != nil {
		t.Fatal(err)
	}
}

// Issue 1 (V3): no marker, the artifact is 404, and the source build fails. The
// plain download is repeated on every start, but the source build is attempted once
// per recheck interval.
func TestBuildBoundNoMarker404BuildsOncePerInterval(t *testing.T) {
	dir, server := refusalServer(t)
	gets := 0
	var builds []string
	i := New()
	i.Get = releaseOnly(t, "b6100", func(string) (*http.Response, error) {
		gets++
		return bbResp(404, "", nil), nil
	})
	i.Output = artifactOutput("6050")
	i.LookPath = func(string) (string, error) { return "/usr/bin/x", nil }
	i.Run = countSourceBuilds(&builds)
	opts := Options{DataDir: dir, ServerURL: "http://x/llama-server.gz"}
	for s := 0; s < 4; s++ {
		if got := i.updateServer(context.Background(), opts, server); got != server {
			t.Fatalf("start %d: server = %q, want the current server kept", s, got)
		}
	}
	if len(builds) != 1 {
		t.Fatalf("source builds = %d over 4 starts, want 1", len(builds))
	}
	if gets != 4 {
		t.Fatalf("artifact GETs = %d, want 4 (the plain download still runs each start)", gets)
	}
}

// Issue 1 (V4): a marker for another tag, the new tag's artifact is 404: one build per interval.
func TestBuildBoundOtherTagMarker404BuildsOncePerInterval(t *testing.T) {
	dir, server := refusalServer(t)
	writeRejectedFile(t, dir, "b6000", "", "", sha([]byte("x")), time.Now())
	var builds []string
	i := New()
	i.Get = releaseOnly(t, "b6100", func(string) (*http.Response, error) {
		return bbResp(404, "", nil), nil
	})
	i.Output = artifactOutput("6050")
	i.LookPath = func(string) (string, error) { return "/usr/bin/x", nil }
	i.Run = countSourceBuilds(&builds)
	opts := Options{DataDir: dir, ServerURL: "http://x/llama-server.gz"}
	for s := 0; s < 4; s++ {
		i.updateServer(context.Background(), opts, server)
	}
	if len(builds) != 1 {
		t.Fatalf("source builds = %d over 4 starts, want 1", len(builds))
	}
}

// Issue 2 (V5): the held marker is within the interval, the server changes its ETag
// on every response, and the body is broken. Each start stages a changed validator
// and refuses it; the build is bounded to one per interval.
func TestBuildBoundChurnBrokenBodyBuildsOncePerInterval(t *testing.T) {
	dir, server := heldRefusalServer(t, `"v1"`, time.Now().Add(-1*time.Hour))
	n := 0
	var builds []string
	i := New()
	i.Get = releaseOnly(t, "b6100", func(string) (*http.Response, error) {
		t.Fatal("full GET within the interval")
		return nil, nil
	})
	i.Conditional = func(string, http.Header) (*http.Response, error) {
		n++
		return bbResp(200, `"c`+string(rune('0'+n))+`"`, gzipBytes(t, []byte("junk"))), nil
	}
	i.Output = artifactOutput("6050")
	i.LookPath = func(string) (string, error) { return "/usr/bin/x", nil }
	i.Run = bbRun(&builds)
	opts := Options{DataDir: dir, ServerURL: "http://x/llama-server.gz"}
	for s := 0; s < 3; s++ {
		if got := i.updateServer(context.Background(), opts, server); got != server {
			t.Fatalf("start %d: server = %q, want the current server kept", s, got)
		}
	}
	if len(builds) != 1 {
		t.Fatalf("source builds = %d over 3 starts, want 1", len(builds))
	}
}

// Issue 3 (V6): a due 404 builds once (and fails); on the next start, within the
// interval, a re-upload with a new ETag and a broken body must not build again.
func TestBuildBoundDueAbsentThenBrokenReuploadBuildsOnce(t *testing.T) {
	dir, server := heldRefusalServer(t, `"v1"`, time.Now().Add(-25*time.Hour))
	var builds []string
	i := New()
	i.Get = releaseOnly(t, "b6100", func(string) (*http.Response, error) {
		return bbResp(404, "", nil), nil
	})
	i.Conditional = func(string, http.Header) (*http.Response, error) {
		return bbResp(200, `"new"`, gzipBytes(t, []byte("junk"))), nil
	}
	i.Output = artifactOutput("6050")
	i.LookPath = func(string) (string, error) { return "/usr/bin/x", nil }
	i.Run = bbRun(&builds)
	opts := Options{DataDir: dir, ServerURL: "http://x/llama-server.gz"}
	i.updateServer(context.Background(), opts, server)
	i.updateServer(context.Background(), opts, server)
	if len(builds) != 1 {
		t.Fatalf("source builds = %d over 2 starts, want 1", len(builds))
	}
}

// The bound expires: an attempt older than the recheck interval allows a new build.
func TestBuildBoundAllowsBuildAfterInterval(t *testing.T) {
	dir, server := refusalServer(t)
	writeBuildAttempts(t, dir, map[string]string{
		"b6100": time.Now().Add(-25 * time.Hour).UTC().Format(time.RFC3339),
	})
	var builds []string
	i := New()
	i.Get = releaseOnly(t, "b6100", func(string) (*http.Response, error) {
		return bbResp(404, "", nil), nil
	})
	i.Output = artifactOutput("6050")
	i.LookPath = func(string) (string, error) { return "/usr/bin/x", nil }
	i.Run = countSourceBuilds(&builds)
	opts := Options{DataDir: dir, ServerURL: "http://x/llama-server.gz"}
	i.updateServer(context.Background(), opts, server)
	if len(builds) != 1 {
		t.Fatalf("source builds = %d after the interval, want 1", len(builds))
	}
}

// Another tag's recent attempt does not block this tag's build.
func TestBuildBoundIsPerTag(t *testing.T) {
	dir, server := refusalServer(t)
	writeBuildAttempts(t, dir, map[string]string{
		"b6000": time.Now().UTC().Format(time.RFC3339),
	})
	var builds []string
	i := New()
	i.Get = releaseOnly(t, "b6100", func(string) (*http.Response, error) {
		return bbResp(404, "", nil), nil
	})
	i.Output = artifactOutput("6050")
	i.LookPath = func(string) (string, error) { return "/usr/bin/x", nil }
	i.Run = countSourceBuilds(&builds)
	opts := Options{DataDir: dir, ServerURL: "http://x/llama-server.gz"}
	i.updateServer(context.Background(), opts, server)
	if len(builds) != 1 {
		t.Fatalf("source builds = %d, want 1 (a different tag's attempt must not block)", len(builds))
	}
}

// A future-dated or unreadable attempt does not block a build forever.
func TestReadBuildAttemptsDropsFutureAndCorrupt(t *testing.T) {
	dir := t.TempDir()
	now := time.Now()
	writeBuildAttempts(t, dir, map[string]string{
		"future": now.Add(48 * time.Hour).UTC().Format(time.RFC3339),
		"recent": now.Add(-time.Hour).UTC().Format(time.RFC3339),
		"junk":   "not-a-time",
	})
	got := readBuildAttempts(dir, now)
	if _, ok := got["future"]; ok {
		t.Error("future-dated attempt was kept")
	}
	if _, ok := got["junk"]; ok {
		t.Error("unparseable attempt was kept")
	}
	if _, ok := got["recent"]; !ok {
		t.Error("recent attempt was dropped")
	}
	if err := os.WriteFile(buildAttemptsPath(dir), []byte("{broken"), 0o644); err != nil {
		t.Fatal(err)
	}
	if n := len(readBuildAttempts(dir, now)); n != 0 {
		t.Fatalf("corrupt sidecar read as %d entries, want 0", n)
	}
}

// errBuildBounded must be recognisable by callers.
func TestBuildBoundedErrorType(t *testing.T) {
	var err error = &errBuildBounded{tag: "b6100", last: time.Now()}
	var b *errBuildBounded
	if !errors.As(err, &b) || b.tag != "b6100" {
		t.Fatal("errBuildBounded not matched by errors.As")
	}
}

// The stamped-build path (refreshStampedBuild) shares the bound: a source build of
// the newer tag that keeps failing is attempted once per interval.
func TestBuildBoundStampedBuildFailsOncePerInterval(t *testing.T) {
	dir, server := refusalServer(t)
	if err := writeStamp(server, installStamp{Source: sourceGitBuild, Tag: "b6000"}); err != nil {
		t.Fatal(err)
	}
	var builds []string
	i := New()
	i.Get = releaseOnly(t, "b6100", nil)
	// The installed server reports no build number, so it is judged by its stamp.
	i.Output = func(context.Context, string, ...string) (string, error) { return "version: 1.0.0-dev (build 0)\n", nil }
	i.LookPath = func(string) (string, error) { return "/usr/bin/x", nil }
	i.Run = countSourceBuilds(&builds)
	opts := Options{DataDir: dir, ServerURL: "http://x/llama-server.gz"}
	for s := 0; s < 3; s++ {
		if got := i.updateServer(context.Background(), opts, server); got != server {
			t.Fatalf("start %d: server = %q, want the current server kept", s, got)
		}
	}
	if len(builds) != 1 {
		t.Fatalf("source builds = %d over 3 starts, want 1", len(builds))
	}
}

// A failed write of the attempt record fails open: the build still runs on every
// start, and the failure is logged rather than blocking the build.
func TestBuildBoundAttemptWriteFailsOpen(t *testing.T) {
	dir, server := refusalServer(t)
	if err := os.MkdirAll(buildAttemptsPath(dir), 0o755); err != nil {
		t.Fatal(err)
	}
	var builds []string
	i := New()
	i.Get = releaseOnly(t, "b6100", func(string) (*http.Response, error) { return bbResp(404, "", nil), nil })
	i.Output = artifactOutput("6050")
	i.LookPath = func(string) (string, error) { return "/usr/bin/x", nil }
	i.Run = countSourceBuilds(&builds)
	opts := Options{DataDir: dir, ServerURL: "http://x/llama-server.gz"}
	for s := 0; s < 3; s++ {
		if got := i.updateServer(context.Background(), opts, server); got != server {
			t.Fatalf("start %d: server = %q, want kept", s, got)
		}
	}
	if len(builds) != 3 {
		t.Fatalf("builds = %d over 3 starts with an unwritable record, want 3 (fail open)", len(builds))
	}
}

// A build that fails still counts: its attempt is recorded before the build runs.
func TestBuildBoundFailedBuildIsRecorded(t *testing.T) {
	dir, server := refusalServer(t)
	var builds []string
	i := New()
	i.Get = releaseOnly(t, "b6100", func(string) (*http.Response, error) { return bbResp(404, "", nil), nil })
	i.Output = artifactOutput("6050")
	i.LookPath = func(string) (string, error) { return "/usr/bin/x", nil }
	i.Run = countSourceBuilds(&builds)
	opts := Options{DataDir: dir, ServerURL: "http://x/llama-server.gz"}
	i.updateServer(context.Background(), opts, server)
	if _, ok := readBuildAttempts(dir, time.Now())["b6100"]; !ok {
		t.Fatal("failed build left no attempt record")
	}
}

// An interrupted build (context cancelled during the git step) still counts: the
// next start does not build again within the interval.
func TestBuildBoundInterruptedBuildCounts(t *testing.T) {
	dir, server := refusalServer(t)
	ctx, cancel := context.WithCancel(context.Background())
	var builds []string
	i := New()
	i.Get = releaseOnly(t, "b6100", func(string) (*http.Response, error) { return bbResp(404, "", nil), nil })
	i.Output = artifactOutput("6050")
	i.LookPath = func(string) (string, error) { return "/usr/bin/x", nil }
	i.Run = func(c context.Context, name string, args ...string) error {
		if name == "git" {
			builds = append(builds, name)
			cancel()
			return context.Canceled
		}
		return nil
	}
	opts := Options{DataDir: dir, ServerURL: "http://x/llama-server.gz"}
	i.updateServer(ctx, opts, server)
	i.updateServer(context.Background(), opts, server)
	if len(builds) != 1 {
		t.Fatalf("builds = %d after an interrupted build then a new start, want 1", len(builds))
	}
}

// An attempt for an old tag must not block a genuinely new tag that appears later.
func TestBuildBoundNewTagNotBlockedByOldTagAttempt(t *testing.T) {
	dir, server := refusalServer(t)
	writeBuildAttempts(t, dir, map[string]string{
		"b6100": time.Now().Add(-time.Hour).UTC().Format(time.RFC3339),
	})
	var builds []string
	i := New()
	i.Get = releaseOnly(t, "b6200", func(string) (*http.Response, error) { return bbResp(404, "", nil), nil })
	i.Output = artifactOutput("6050")
	i.LookPath = func(string) (string, error) { return "/usr/bin/x", nil }
	i.Run = countSourceBuilds(&builds)
	opts := Options{DataDir: dir, ServerURL: "http://x/llama-server.gz"}
	i.updateServer(context.Background(), opts, server)
	if len(builds) != 1 {
		t.Fatalf("builds = %d for b6200 after a recent b6100 attempt, want 1", len(builds))
	}
	if _, ok := readBuildAttempts(dir, time.Now())["b6200"]; !ok {
		t.Fatal("new tag's attempt was not recorded")
	}
}

// One malformed entry (a non-string value) must not discard the valid attempts for
// the other tags: the valid b6100 attempt still bounds the build.
func TestReadBuildAttemptsKeepsValidWhenOneEntryMalformed(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	recent := now.Add(-time.Hour).UTC().Format(time.RFC3339)
	if err := os.WriteFile(buildAttemptsPath(dir), []byte(`{"b6100": "`+recent+`", "b6000": 5, "b5900": null}`), 0o644); err != nil {
		t.Fatal(err)
	}
	got := readBuildAttempts(dir, now)
	if _, ok := got["b6100"]; !ok {
		t.Fatal("a malformed entry for another tag dropped the valid b6100 attempt")
	}
	if _, ok := got["b6000"]; ok {
		t.Fatal("malformed entry b6000 was kept")
	}
	// And the bound holds end to end: no source build for b6100 on the next start.
	dir2, server := refusalServer(t)
	if err := os.WriteFile(buildAttemptsPath(dir2), []byte(`{"b6100": "`+recent+`", "b6000": 5}`), 0o644); err != nil {
		t.Fatal(err)
	}
	var builds []string
	i := New()
	i.Get = releaseOnly(t, "b6100", func(string) (*http.Response, error) { return bbResp(404, "", nil), nil })
	i.Output = artifactOutput("6050")
	i.LookPath = func(string) (string, error) { return "/usr/bin/x", nil }
	i.Run = countSourceBuilds(&builds)
	i.updateServer(context.Background(), Options{DataDir: dir2, ServerURL: "http://x/llama-server.gz"}, server)
	if len(builds) != 0 {
		t.Fatalf("source builds = %d with a valid recent b6100 attempt beside a malformed entry, want 0", len(builds))
	}
}

// Two concurrent callers (two processes sharing one data directory) for the same tag
// produce one source build: the lock makes the check and the record one step.
func TestBuildTagOnceConcurrentCallersBuildOnce(t *testing.T) {
	const callers = 8
	const rounds = 20
	for r := 0; r < rounds; r++ {
		dir, _ := refusalServer(t)
		var mu sync.Mutex
		builds := 0
		start := make(chan struct{})
		var wg sync.WaitGroup
		for c := 0; c < callers; c++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				i := New()
				i.Run = func(_ context.Context, name string, _ ...string) error {
					if name == "git" {
						mu.Lock()
						builds++
						mu.Unlock()
						return io.ErrUnexpectedEOF
					}
					return nil
				}
				<-start
				_, _ = i.buildTagOnce(context.Background(), dir, "b6100")
			}()
		}
		close(start)
		wg.Wait()
		if builds != 1 {
			t.Fatalf("round %d: source builds = %d for %d concurrent callers, want 1", r, builds, callers)
		}
		if _, err := os.Stat(buildAttemptsPath(dir) + ".lock"); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("round %d: build-attempt lock file left behind (err=%v)", r, err)
		}
	}
}

// A lock that cannot be taken fails open: the check runs without it and the build
// proceeds. Here the lock path is occupied by a directory, so the lock never succeeds.
func TestBuildBoundLockFailureFailsOpen(t *testing.T) {
	dir, server := refusalServer(t)
	if err := os.MkdirAll(buildAttemptsPath(dir)+".lock", 0o755); err != nil {
		t.Fatal(err)
	}
	var builds []string
	i := New()
	i.Get = releaseOnly(t, "b6100", func(string) (*http.Response, error) { return bbResp(404, "", nil), nil })
	i.Output = artifactOutput("6050")
	i.LookPath = func(string) (string, error) { return "/usr/bin/x", nil }
	i.Run = countSourceBuilds(&builds)
	if got := i.updateServer(context.Background(), Options{DataDir: dir, ServerURL: "http://x/llama-server.gz"}, server); got != server {
		t.Fatalf("server = %q, want kept", got)
	}
	if len(builds) != 1 {
		t.Fatalf("builds = %d with an unlockable record, want 1 (fail open)", len(builds))
	}
}

// A lock left behind by a process that died is broken once it is stale, so it does
// not block the bound's record forever.
func TestBuildBoundStaleLockIsBroken(t *testing.T) {
	dir, server := refusalServer(t)
	lock := buildAttemptsPath(dir) + ".lock"
	if err := os.MkdirAll(filepath.Dir(lock), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(lock, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-2 * buildAttemptsLockStale)
	if err := os.Chtimes(lock, old, old); err != nil {
		t.Fatal(err)
	}
	var builds []string
	i := New()
	i.Get = releaseOnly(t, "b6100", func(string) (*http.Response, error) { return bbResp(404, "", nil), nil })
	i.Output = artifactOutput("6050")
	i.LookPath = func(string) (string, error) { return "/usr/bin/x", nil }
	i.Run = countSourceBuilds(&builds)
	i.updateServer(context.Background(), Options{DataDir: dir, ServerURL: "http://x/llama-server.gz"}, server)
	if len(builds) != 1 {
		t.Fatalf("builds = %d behind a stale lock, want 1", len(builds))
	}
	if _, err := os.Stat(lock); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("stale lock file not cleaned up (err=%v)", err)
	}
}

// A rename obstruction (the verified candidate cannot be renamed into place) falls
// through to the bounded build: the build runs at most once per interval while the
// obstruction persists.
func TestBuildBoundRenameObstructionBuildBounded(t *testing.T) {
	dir := t.TempDir()
	server := filepath.Join(dir, "srv", "llama-server")
	_ = os.MkdirAll(filepath.Dir(server), 0755)
	_ = os.WriteFile(server, []byte("numeric"), 0755)
	dest := filepath.Join(dir, "bin", "llama-server")
	if err := os.MkdirAll(filepath.Join(dest, "occupied"), 0755); err != nil {
		t.Fatal(err)
	}
	fixed := []byte("#!/bin/sh\necho 'version: 6100 (fixed)'\n")
	var builds []string
	i := New()
	i.Get = releaseOnly(t, "b6100", func(string) (*http.Response, error) {
		return bbResp(200, `"v1"`, gzipBytes(t, fixed)), nil
	})
	i.Conditional = func(string, http.Header) (*http.Response, error) {
		return bbResp(304, "", nil), nil
	}
	i.Output = artifactOutput("6100")
	i.LookPath = func(string) (string, error) { return "/usr/bin/x", nil }
	i.Run = bbRun(&builds)
	opts := Options{DataDir: dir, ServerURL: "http://x/llama-server.gz"}
	for s := 0; s < 3; s++ {
		i.updateServer(context.Background(), opts, server)
	}
	if len(builds) != 1 {
		t.Fatalf("source builds = %d over 3 starts with rename obstruction, want 1", len(builds))
	}
}
