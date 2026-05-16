package updater

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestNewUpdater(t *testing.T) {
	cfg := Config{
		Enabled:        true,
		PollInterval:   5 * time.Minute,
		GitHubRepo:     "KNIRV/KNIRV_NETWORK",
		GitHubToken:    "test-token",
		AssetName:      "knirv-server",
		CurrentCommit:  "abc1234",
		BinaryPath:     "/usr/local/bin/knirv-server",
	}

	u := New(cfg)

	if u == nil {
		t.Fatal("New() returned nil")
	}

	if u.cfg.PollInterval != 5*time.Minute {
		t.Errorf("expected PollInterval 5m, got %v", u.cfg.PollInterval)
	}

	if u.cfg.GitHubRepo != "KNIRV/KNIRV_NETWORK" {
		t.Errorf("expected GitHubRepo KNIRV/KNIRV_NETWORK, got %s", u.cfg.GitHubRepo)
	}

	if u.client == nil {
		t.Error("expected http.Client to be initialized")
	}
}

func TestNewUpdaterDefaults(t *testing.T) {
	cfg := Config{
		Enabled:       true,
		GitHubRepo:    "test/repo",
		BinaryPath:   "/path/to/bin",
	}

	u := New(cfg)

	if u.cfg.PollInterval != 10*time.Minute {
		t.Errorf("expected default PollInterval 10m, got %v", u.cfg.PollInterval)
	}

	if u.cfg.AssetName != "knirv-server" {
		t.Errorf("expected default AssetName knirv-server, got %s", u.cfg.AssetName)
	}
}

func TestUpdaterStatus(t *testing.T) {
	cfg := Config{
		Enabled:        true,
		PollInterval: 15 * time.Minute,
		GitHubRepo:    "owner/repo",
		CurrentCommit: "def5678",
	}

	u := New(cfg)
	status := u.GetStatus()

	if status["enabled"] != true {
		t.Error("expected enabled=true in status")
	}

	if status["current_commit"] != "def5678" {
		t.Errorf("expected current_commit def5678, got %v", status["current_commit"])
	}

	if status["poll_interval"] != "15m0s" {
		t.Errorf("expected poll_interval 15m0s, got %v", status["poll_interval"])
	}

	if status["github_repo"] != "owner/repo" {
		t.Errorf("expected github_repo owner/repo, got %v", status["github_repo"])
	}
}

func TestVerifyChecksum(t *testing.T) {
	content := "hello world"
	expectedHash := "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"

	tmpFile, err := os.CreateTemp("", "checksum-test-*")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	if err := verifyChecksum(tmpFile.Name(), expectedHash); err != nil {
		t.Errorf("checksum verification failed: %v", err)
	}
}

func TestVerifyChecksumMismatch(t *testing.T) {
	content := "hello world"
	tmpFile, err := os.CreateTemp("", "checksum-test-*")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	wrongHash := "0000000000000000000000000000000000000000000000000000000000000000"
	if err := verifyChecksum(tmpFile.Name(), wrongHash); err == nil {
		t.Error("expected checksum mismatch error, got nil")
	}
}

func TestSplitAndTrim(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"abc  def", []string{"abc", "def"}},
		{"abc\tdef", []string{"abc", "def"}},
		{"abc\ndef", []string{"abc", "def"}},
		{"abc\r\ndef", []string{"abc", "def"}},
		{"  abc   def  ", []string{"abc", "def"}},
		{"abc", []string{"abc"}},
		{"", nil},
	}

	for _, tc := range tests {
		result := splitAndTrim(tc.input)
		if len(result) != len(tc.expected) {
			t.Errorf("splitAndTrim(%q): expected %v, got %v", tc.input, tc.expected, result)
			continue
		}
		for i, v := range result {
			if v != tc.expected[i] {
				t.Errorf("splitAndTrim(%q)[%d]: expected %s, got %s", tc.input, i, tc.expected[i], v)
			}
		}
	}
}

func TestDisabledUpdater(t *testing.T) {
	cfg := Config{
		Enabled:      false,
		GitHubRepo:  "owner/repo",
		BinaryPath:  "/path",
	}

	u := New(cfg)
	status := u.GetStatus()

	if status["enabled"] != false {
		t.Error("expected enabled=false in status for disabled updater")
	}
}

// newTestUpdater wires an Updater against a fake GitHub API server.
func newTestUpdater(serverURL, currentCommit string) *Updater {
	cfg := Config{
		Enabled:       true,
		PollInterval:  10 * time.Minute,
		GitHubRepo:    "owner/repo",
		CurrentCommit: currentCommit,
		BinaryPath:    "/path/to/bin",
	}
	u := New(cfg)
	// Override the HTTP client to talk to the test server.
	u.client = &http.Client{
		Timeout: 5 * time.Second,
		Transport: &hostRewriteTransport{
			base:       http.DefaultTransport,
			targetHost: serverURL,
		},
	}
	return u
}

// hostRewriteTransport rewrites the Host of every request to targetHost so
// the updater's hardcoded api.github.com URL hits the test server instead.
type hostRewriteTransport struct {
	base       http.RoundTripper
	targetHost string // e.g. "http://127.0.0.1:PORT"
}

func (t *hostRewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	cloned := req.Clone(req.Context())
	cloned.URL.Host = req.URL.Host
	// Replace scheme+host with the test server.
	cloned.URL.Scheme = "http"
	cloned.URL.Host = t.targetHost[len("http://"):]
	cloned.Host = cloned.URL.Host
	return t.base.RoundTrip(cloned)
}

func TestCheckUpdateAvailable_NewVersion(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(githubRelease{TagName: "v2.0.0", Body: "New features"})
	}))
	defer srv.Close()

	u := newTestUpdater(srv.URL, "v1.0.0")
	status, err := u.CheckUpdateAvailable()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !status.Available {
		t.Error("expected Available=true when tags differ")
	}
	if status.LatestTag != "v2.0.0" {
		t.Errorf("expected LatestTag v2.0.0, got %s", status.LatestTag)
	}
	if status.CurrentVersion != "v1.0.0" {
		t.Errorf("expected CurrentVersion v1.0.0, got %s", status.CurrentVersion)
	}
	if status.Notes != "New features" {
		t.Errorf("expected Notes 'New features', got %s", status.Notes)
	}
	if status.CheckedAt == "" {
		t.Error("expected CheckedAt to be set")
	}
}

func TestCheckUpdateAvailable_UpToDate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(githubRelease{TagName: "v1.0.0"})
	}))
	defer srv.Close()

	u := newTestUpdater(srv.URL, "v1.0.0")
	status, err := u.CheckUpdateAvailable()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status.Available {
		t.Error("expected Available=false when tags match")
	}
}

func TestCheckUpdateAvailable_Caching(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		json.NewEncoder(w).Encode(githubRelease{TagName: "v2.0.0"})
	}))
	defer srv.Close()

	u := newTestUpdater(srv.URL, "v1.0.0")

	if _, err := u.CheckUpdateAvailable(); err != nil {
		t.Fatalf("first call error: %v", err)
	}
	if _, err := u.CheckUpdateAvailable(); err != nil {
		t.Fatalf("second call error: %v", err)
	}

	if calls != 1 {
		t.Errorf("expected exactly 1 HTTP call (cache hit on second), got %d", calls)
	}
}

func TestCheckUpdateAvailable_CacheExpiry(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		json.NewEncoder(w).Encode(githubRelease{TagName: "v2.0.0"})
	}))
	defer srv.Close()

	u := newTestUpdater(srv.URL, "v1.0.0")
	// Force cache to be already expired.
	u.cacheExpiry = time.Now().Add(-1 * time.Second)
	u.cachedStatus = &UpdateStatus{Available: false, LatestTag: "v1.0.0"}

	status, err := u.CheckUpdateAvailable()
	if err != nil {
		t.Fatalf("error after forced expiry: %v", err)
	}
	if !status.Available {
		t.Error("expected fresh fetch to detect update after cache expiry")
	}
	if calls != 1 {
		t.Errorf("expected 1 HTTP call after cache expiry, got %d", calls)
	}
}

func TestCheckUpdateAvailable_GitHubError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	u := newTestUpdater(srv.URL, "v1.0.0")
	_, err := u.CheckUpdateAvailable()
	if err == nil {
		t.Error("expected error when GitHub API returns 500")
	}
}

func TestCheckUpdateAvailable_EmptyTagNotAvailable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(githubRelease{TagName: ""})
	}))
	defer srv.Close()

	u := newTestUpdater(srv.URL, "v1.0.0")
	status, err := u.CheckUpdateAvailable()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status.Available {
		t.Error("expected Available=false when latest tag is empty")
	}
}