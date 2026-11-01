package cloudflare

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func stubNetwork(t *testing.T, tag string, binary string) {
	t.Helper()
	oldGet, oldOut := httpGet, versionOutput
	t.Cleanup(func() { httpGet, versionOutput = oldGet, oldOut })
	httpGet = func(url string) (*http.Response, error) {
		body := binary
		if url == latestReleaseURL {
			body = fmt.Sprintf(`{"tag_name":"%s"}`, tag)
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}, nil
	}
	versionOutput = func(path string) (string, error) {
		b, _ := os.ReadFile(path)
		return string(b), nil // the fake binary's content is its --version output
	}
}

func writeFake(t *testing.T, version string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "cloudflared")
	if err := os.WriteFile(p, []byte("cloudflared version "+version+" (built 2025)"), 0755); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestVersionLess(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"2025.9.1", "2025.10.0", true},
		{"2025.10.1", "2025.10.1", false},
		{"2025.10.2", "2025.10.1", false},
		{"2025.10", "2025.10.1", true},
	}
	for _, c := range cases {
		a, _ := parseVersion(c.a)
		b, _ := parseVersion(c.b)
		if got := versionLess(a, b); got != c.want {
			t.Errorf("versionLess(%s,%s) = %v", c.a, c.b, got)
		}
	}
}

func TestUpdateCloudflaredKeepsCurrent(t *testing.T) {
	p := writeFake(t, "2025.10.1")
	stubNetwork(t, "2025.10.1", "unused")
	httpGetCalls := 0
	inner := httpGet
	httpGet = func(u string) (*http.Response, error) {
		if u != latestReleaseURL {
			httpGetCalls++
		}
		return inner(u)
	}
	updateCloudflared(p)
	if httpGetCalls != 0 {
		t.Fatal("up to date: nothing should be downloaded")
	}
}

func TestUpdateCloudflaredReplacesOlder(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("auto-update unsupported on this OS")
	}
	p := writeFake(t, "2025.9.0")
	stubNetwork(t, "2025.10.1", "cloudflared version 2025.10.1 (built 2025)")
	if runtime.GOOS == "darwin" {
		t.Skip("darwin downloads a tgz")
	}
	if got := updateCloudflared(p); got != p {
		t.Fatalf("path = %s", got)
	}
	if b, _ := os.ReadFile(p); !strings.Contains(string(b), "2025.10.1") {
		t.Fatalf("binary not replaced: %q", b)
	}
	if _, err := os.Stat(p + ".update"); !os.IsNotExist(err) {
		t.Fatal("staged file left behind")
	}
}

func TestUpdateCloudflaredKeepsBinaryWhenDownloadIsStale(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("linux download path")
	}
	p := writeFake(t, "2025.9.0")
	stubNetwork(t, "2025.10.1", "cloudflared version 2025.9.5 (built 2025)")
	updateCloudflared(p)
	if b, _ := os.ReadFile(p); !strings.Contains(string(b), "2025.9.0") {
		t.Fatalf("working binary was replaced by a stale one: %q", b)
	}
	if _, err := os.Stat(p + ".update"); !os.IsNotExist(err) {
		t.Fatal("staged file left behind")
	}
}

func TestUpdateCloudflaredOfflineKeepsBinary(t *testing.T) {
	p := writeFake(t, "2025.9.0")
	old := httpGet
	t.Cleanup(func() { httpGet = old })
	httpGet = func(string) (*http.Response, error) { return nil, io.ErrUnexpectedEOF }
	if got := updateCloudflared(p); got != p {
		t.Fatalf("path = %s", got)
	}
}
