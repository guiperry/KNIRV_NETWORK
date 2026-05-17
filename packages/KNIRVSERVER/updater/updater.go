package updater

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"syscall"
	"time"
)

// UpdateStatus is the cached result of a GitHub release check.
type UpdateStatus struct {
	Available      bool   `json:"available"`
	LatestTag      string `json:"latest_tag"`
	CurrentVersion string `json:"current_version"`
	Notes          string `json:"notes,omitempty"`
	CheckedAt      string `json:"checked_at"`
}

type Config struct {
	Enabled        bool
	PollInterval   time.Duration
	GitHubRepo    string
	GitHubToken   string
	AssetName     string
	CurrentCommit string
	BinaryPath    string
}

type Updater struct {
	cfg          Config
	client       *http.Client
	mu           sync.RWMutex
	cachedStatus *UpdateStatus
	cacheExpiry  time.Time
}

func New(cfg Config) *Updater {
	if cfg.PollInterval == 0 {
		cfg.PollInterval = 10 * time.Minute
	}
	if cfg.AssetName == "" {
		cfg.AssetName = "knirv-server"
	}
	return &Updater{
		cfg:    cfg,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (u *Updater) Start() {
	if !u.cfg.Enabled {
		log.Println("[updater] disabled")
		return
	}
	log.Printf("[updater] polling every %s for repo %s", u.cfg.PollInterval, u.cfg.GitHubRepo)
	ticker := time.NewTicker(u.cfg.PollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if err := u.checkAndApply(); err != nil {
				log.Printf("[updater] check failed: %v", err)
			}
		}
	}
}

func (u *Updater) GetStatus() map[string]interface{} {
	return map[string]interface{}{
		"enabled":        u.cfg.Enabled,
		"current_commit": u.cfg.CurrentCommit,
		"poll_interval":   u.cfg.PollInterval.String(),
		"github_repo":    u.cfg.GitHubRepo,
	}
}

type githubRelease struct {
	TagName string        `json:"tag_name"`
	Assets  []githubAsset `json:"assets"`
	Body   string        `json:"body"`
}

type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

func (u *Updater) fetchLatestRelease() (*githubRelease, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", u.cfg.GitHubRepo)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	if u.cfg.GitHubToken != "" {
		req.Header.Set("Authorization", "Bearer "+u.cfg.GitHubToken)
	}
	resp, err := u.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("github API returned %d", resp.StatusCode)
	}
	var rel githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, err
	}
	return &rel, nil
}

func (u *Updater) GetLatestReleaseInfo() (tag string, err error) {
	rel, err := u.fetchLatestRelease()
	if err != nil {
		return "", err
	}
	return rel.TagName, nil
}

func (u *Updater) download(url, dest string) error {
	resp, err := u.client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	f, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}

func (u *Updater) fetchChecksum(url string) (string, error) {
	resp, err := u.client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	fields := splitAndTrim(string(b))
	if len(fields) == 0 {
		return "", fmt.Errorf("empty checksum response")
	}
	return fields[0], nil
}

func splitAndTrim(s string) []string {
	var result []string
	var current []byte
	for _, c := range s {
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			if len(current) > 0 {
				result = append(result, string(current))
				current = nil
			}
		} else {
			current = append(current, byte(c))
		}
	}
	if len(current) > 0 {
		result = append(result, string(current))
	}
	return result
}

func (u *Updater) checkAndApply() error {
	rel, err := u.fetchLatestRelease()
	if err != nil {
		return fmt.Errorf("fetch release: %w", err)
	}
	if rel.TagName == u.cfg.CurrentCommit {
		return nil
	}
	log.Printf("[updater] new version available: %s (current: %s)", rel.TagName, u.cfg.CurrentCommit)

	binaryURL, checksumURL := "", ""
	for _, a := range rel.Assets {
		if a.Name == u.cfg.AssetName {
			binaryURL = a.BrowserDownloadURL
		}
		if a.Name == u.cfg.AssetName+".sha256" {
			checksumURL = a.BrowserDownloadURL
		}
	}
	if binaryURL == "" {
		return fmt.Errorf("asset %q not found in release %s", u.cfg.AssetName, rel.TagName)
	}

	newPath := u.cfg.BinaryPath + ".new"
	prevPath := u.cfg.BinaryPath + ".prev"

	if err := u.download(binaryURL, newPath); err != nil {
		return fmt.Errorf("download: %w", err)
	}
	defer os.Remove(newPath)

	if checksumURL != "" {
		expected, err := u.fetchChecksum(checksumURL)
		if err != nil {
			return fmt.Errorf("fetch checksum: %w", err)
		}
		if err := verifyChecksum(newPath, expected); err != nil {
			return fmt.Errorf("checksum mismatch: %w", err)
		}
	} else {
		log.Printf("[updater] warning: no checksum file found in release, skipping verification")
	}

	if err := os.Chmod(newPath, 0755); err != nil {
		return err
	}

	_ = os.Rename(u.cfg.BinaryPath, prevPath)
	if err := os.Rename(newPath, u.cfg.BinaryPath); err != nil {
		_ = os.Rename(prevPath, u.cfg.BinaryPath)
		return fmt.Errorf("rename new binary: %w", err)
	}

	log.Printf("[updater] binary updated to %s, restarting...", rel.TagName)
	return syscall.Exec(u.cfg.BinaryPath, os.Args, os.Environ())
}

func verifyChecksum(path, expected string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}
	got := hex.EncodeToString(h.Sum(nil))
	if got != expected {
		return fmt.Errorf("want %s, got %s", expected, got)
	}
	return nil
}

// CheckUpdateAvailable returns whether a newer release exists on GitHub without
// downloading or applying it. Results are cached for one poll interval (minimum
// 10 minutes) to avoid hammering the GitHub API.
// Returns a not-available status (no error) when the updater is disabled or
// no GitHub repo is configured.
func (u *Updater) CheckUpdateAvailable() (*UpdateStatus, error) {
	if !u.cfg.Enabled || u.cfg.GitHubRepo == "" {
		return &UpdateStatus{
			Available:      false,
			CurrentVersion: u.cfg.CurrentCommit,
			CheckedAt:      time.Now().UTC().Format(time.RFC3339),
		}, nil
	}

	u.mu.RLock()
	if u.cachedStatus != nil && time.Now().Before(u.cacheExpiry) {
		s := *u.cachedStatus
		u.mu.RUnlock()
		return &s, nil
	}
	u.mu.RUnlock()

	rel, err := u.fetchLatestRelease()
	if err != nil {
		return nil, fmt.Errorf("fetch release: %w", err)
	}

	cacheTTL := u.cfg.PollInterval
	if cacheTTL < 10*time.Minute {
		cacheTTL = 10 * time.Minute
	}

	status := &UpdateStatus{
		Available:      rel.TagName != "" && rel.TagName != u.cfg.CurrentCommit,
		LatestTag:      rel.TagName,
		CurrentVersion: u.cfg.CurrentCommit,
		Notes:          rel.Body,
		CheckedAt:      time.Now().UTC().Format(time.RFC3339),
	}

	u.mu.Lock()
	u.cachedStatus = status
	u.cacheExpiry = time.Now().Add(cacheTTL)
	u.mu.Unlock()

	return status, nil
}

func (u *Updater) TriggerUpdate() error {
	log.Println("[updater] manually triggered update check")
	return u.checkAndApply()
}