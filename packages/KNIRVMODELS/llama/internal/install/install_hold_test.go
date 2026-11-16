package install

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// heldRefusalServer lays out a numeric-path install at build 6000 with a refused
// marker for b6100. The artifact reports build 6050, so the refusal stands.
func heldRefusalServer(t *testing.T, etag string, checked time.Time) (dir, server string) {
	t.Helper()
	dir = t.TempDir()
	server = filepath.Join(dir, "bin", "llama-server")
	_ = os.MkdirAll(filepath.Dir(server), 0755)
	_ = os.WriteFile(server, []byte("numeric"), 0755)
	writeRejectedFile(t, dir, "b6100", etag, "", sha([]byte("refused-bytes")), checked)
	return dir, server
}

// countSourceBuilds counts the git clone calls made by the source fallback.
func countSourceBuilds(calls *[]string) func(context.Context, string, ...string) error {
	return func(_ context.Context, name string, args ...string) error {
		if name == "git" {
			*calls = append(*calls, name)
		}
		if name == "git" || name == "cmake" {
			return io.ErrUnexpectedEOF // the source build fails; the server must survive
		}
		return nil
	}
}

// Probe P2: the marker is due and the artifact host fails on every request. The
// failed full comparison must hold the refusal (stamping checked_at), so three
// starts run at most one source build.
func TestHeldRefusalArtifactHostErrorBuildsAtMostOnce(t *testing.T) {
	dir, server := heldRefusalServer(t, "", time.Now().Add(-25*time.Hour))
	artifactGets := 0
	var builds []string
	i := New()
	pinUpstream(t, "b6100")
	i.Get = releaseOnly(t, "b6100", func(string) (*http.Response, error) {
		artifactGets++
		return nil, errors.New("artifact host unreachable")
	})
	i.Conditional = noNetwork2(t)
	i.Output = artifactOutput("6050")
	i.LookPath = func(string) (string, error) { return "/usr/bin/x", nil }
	i.Run = countSourceBuilds(&builds)
	opts := Options{DataDir: dir, ServerURL: "http://x/llama-server.gz"}
	for start := 1; start <= 3; start++ {
		i.updateServer(context.Background(), opts, server)
	}
	if len(builds) > 1 {
		t.Fatalf("source builds over 3 starts = %d; want at most 1", len(builds))
	}
	if artifactGets != 1 {
		t.Fatalf("artifact GETs = %d; want 1 (the due comparison, then held)", artifactGets)
	}
	m := readRejected(dir)
	if m == nil || m.Tag != "b6100" || m.SHA256 == "" {
		t.Fatalf("marker after host error = %+v; want the refusal kept", m)
	}
	if when, err := time.Parse(time.RFC3339, m.CheckedAt); err != nil || time.Since(when) > time.Minute {
		t.Fatalf("checked_at = %q after host error; want stamped now", m.CheckedAt)
	}
	if b, _ := os.ReadFile(server); string(b) != "numeric" {
		t.Fatal("server replaced during a failed comparison")
	}
}

// Probe P3: the marker is within the interval with a stable ETag v1. The server now
// sends ETag v2 with a body that is not a valid archive, on every request. The
// staging error is recorded with the new validator, so later starts hold on the
// unchanged v2 without a source build each time.
func TestHeldRefusalBrokenBodyChangedETagBuildsAtMostOnce(t *testing.T) {
	dir, server := heldRefusalServer(t, `"v1"`, time.Now())
	conditionals := 0
	var builds []string
	i := New()
	pinUpstream(t, "b6100")
	i.Get = releaseOnly(t, "b6100", nil)
	i.Conditional = func(string, http.Header) (*http.Response, error) {
		conditionals++
		h := http.Header{}
		h.Set("ETag", `"v2"`)
		return &http.Response{StatusCode: 200, Status: "200 OK", Header: h, Body: io.NopCloser(strings.NewReader("not gzip"))}, nil
	}
	i.Output = artifactOutput("6050")
	i.LookPath = func(string) (string, error) { return "/usr/bin/x", nil }
	i.Run = countSourceBuilds(&builds)
	opts := Options{DataDir: dir, ServerURL: "http://x/llama-server.gz"}
	for start := 1; start <= 3; start++ {
		i.updateServer(context.Background(), opts, server)
	}
	if len(builds) > 1 {
		t.Fatalf("source builds over 3 starts = %d; want at most 1", len(builds))
	}
	if conditionals != 3 {
		t.Fatalf("conditional GETs = %d; want 3", conditionals)
	}
	m := readRejected(dir)
	if m == nil || m.ETag != `"v2"` {
		t.Fatalf("marker after broken body = %+v; want the new validator \"v2\" recorded", m)
	}
	if b, _ := os.ReadFile(server); string(b) != "numeric" {
		t.Fatal("server replaced by a broken artifact")
	}
}

// Issue 2: a host error on a request made within the recheck interval holds the
// refusal without a source build and without moving checked_at. The full comparison
// therefore still falls due 24 h after the last successful comparison, however close
// together the starts are.
func TestHeldRefusalWithinIntervalHostErrorKeepsCheckedAt(t *testing.T) {
	checked := time.Now().Add(-23 * time.Hour).UTC().Truncate(time.Second)
	dir, server := heldRefusalServer(t, `"v1"`, checked)
	conditionals := 0
	var builds []string
	i := New()
	pinUpstream(t, "b6100")
	i.Get = releaseOnly(t, "b6100", func(string) (*http.Response, error) {
		t.Fatal("full comparison ran within the recheck interval")
		return nil, nil
	})
	i.Conditional = func(string, http.Header) (*http.Response, error) {
		conditionals++
		if conditionals == 1 {
			return nil, errors.New("artifact host unreachable")
		}
		return &http.Response{StatusCode: 503, Status: "503 Service Unavailable", Header: http.Header{}, Body: io.NopCloser(strings.NewReader(""))}, nil
	}
	i.Output = artifactOutput("6050")
	i.LookPath = func(string) (string, error) { return "/usr/bin/x", nil }
	i.Run = countSourceBuilds(&builds)
	opts := Options{DataDir: dir, ServerURL: "http://x/llama-server.gz"}
	for start := 1; start <= 3; start++ {
		i.updateServer(context.Background(), opts, server)
	}
	if len(builds) != 0 {
		t.Fatalf("source builds within the interval after host errors = %d; want 0", len(builds))
	}
	if conditionals != 3 {
		t.Fatalf("conditional GETs = %d; want 3 (one per start, none a full comparison)", conditionals)
	}
	m := readRejected(dir)
	if m == nil || m.CheckedAt != checked.Format(time.RFC3339) {
		t.Fatalf("marker after host errors within the interval = %+v; want checked_at unchanged at %s", m, checked.Format(time.RFC3339))
	}
	if b, _ := os.ReadFile(server); string(b) != "numeric" {
		t.Fatal("server replaced during a failed comparison")
	}
}

// Issue 1: a permanent 404 for the artifact of a held tag is "no usable artifact", not
// a host outage. The due comparison falls back to one source build, stamps checked_at,
// and the next starts within the interval make no request and no further build.
func TestHeldRefusalArtifactAbsentBuildsOncePerInterval(t *testing.T) {
	dir, server := heldRefusalServer(t, "", time.Now().Add(-25*time.Hour))
	artifactGets := 0
	var builds []string
	i := New()
	pinUpstream(t, "b6100")
	i.Get = releaseOnly(t, "b6100", func(string) (*http.Response, error) {
		artifactGets++
		return &http.Response{StatusCode: 404, Status: "404 Not Found", Header: http.Header{}, Body: io.NopCloser(strings.NewReader(""))}, nil
	})
	i.Conditional = noNetwork2(t)
	i.Output = artifactOutput("6050")
	i.LookPath = func(string) (string, error) { return "/usr/bin/x", nil }
	i.Run = countSourceBuilds(&builds)
	opts := Options{DataDir: dir, ServerURL: "http://x/llama-server.gz"}
	for start := 1; start <= 3; start++ {
		i.updateServer(context.Background(), opts, server)
	}
	if artifactGets != 1 {
		t.Fatalf("artifact GETs = %d; want 1 (the due comparison only)", artifactGets)
	}
	if len(builds) != 1 {
		t.Fatalf("source builds over 3 starts = %d; want exactly 1 (the fallback)", len(builds))
	}
	m := readRejected(dir)
	if m == nil || m.Tag != "b6100" || !strings.Contains(m.Reason, "404") {
		t.Fatalf("marker after absent artifact = %+v; want the 404 reason recorded", m)
	}
	if when, err := time.Parse(time.RFC3339, m.CheckedAt); err != nil || time.Since(when) > time.Minute {
		t.Fatalf("checked_at = %q after the fallback; want stamped now", m.CheckedAt)
	}
	if b, _ := os.ReadFile(server); string(b) != "numeric" {
		t.Fatal("server replaced although no artifact was published")
	}
}

// Issue 3: within the interval a changed validator whose new bytes do not run is
// recorded with the new validator and its own reason, and built from source once.
// Later starts hold on the unchanged validator: no build, and the body is not read.
func TestHeldRefusalChangedETagRefusedBytesBuildsOnceWithNewReason(t *testing.T) {
	dir, server := heldRefusalServer(t, `"v1"`, time.Now())
	conditionals := 0
	var builds []string
	countBuilds := countSourceBuilds(&builds)
	i := New()
	pinUpstream(t, "b6100")
	i.Get = releaseOnly(t, "b6100", nil)
	badGz := gzipBytes(t, []byte("#!/bin/sh\nexit 1\n"))
	i.Conditional = func(_ string, h http.Header) (*http.Response, error) {
		conditionals++
		if h.Get("If-None-Match") == `"v2"` {
			return &http.Response{StatusCode: 304, Status: "304 Not Modified", Header: http.Header{}, Body: io.NopCloser(strings.NewReader(""))}, nil
		}
		hd := http.Header{}
		hd.Set("ETag", `"v2"`)
		return &http.Response{StatusCode: 200, Status: "200 OK", Header: hd, Body: io.NopCloser(strings.NewReader(string(badGz)))}, nil
	}
	i.Output = artifactOutput("6050")
	i.LookPath = func(string) (string, error) { return "/usr/bin/x", nil }
	i.Run = func(ctx context.Context, name string, args ...string) error {
		if strings.HasSuffix(name, ".update") {
			return errors.New("exit status 1") // the new bytes do not run
		}
		return countBuilds(ctx, name, args...)
	}
	opts := Options{DataDir: dir, ServerURL: "http://x/llama-server.gz"}
	for start := 1; start <= 3; start++ {
		i.updateServer(context.Background(), opts, server)
	}
	if len(builds) != 1 {
		t.Fatalf("source builds over 3 starts = %d; want exactly 1 for the new refusal", len(builds))
	}
	if conditionals != 3 {
		t.Fatalf("conditional GETs = %d; want 3", conditionals)
	}
	m := readRejected(dir)
	if m == nil || m.ETag != `"v2"` || m.Reason != "does not run" {
		t.Fatalf("marker = %+v; want ETag \"v2\" with reason \"does not run\"", m)
	}
	if b, _ := os.ReadFile(server); string(b) != "numeric" {
		t.Fatal("server replaced by refused bytes")
	}
}
