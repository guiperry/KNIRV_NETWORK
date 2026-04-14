# KNIRVSERVER — Production Runtime Update Strategy

## Overview

This document proposes a self-update mechanism for `knirv-server` that automatically applies new versions when a commit is pushed to the `main` branch on GitHub. The implementation uses only the Go standard library — no new module dependencies.

---

## Constraints & Key Observations

| Constraint | Impact |
|---|---|
| Single unified binary (embeds frontend, backend, sub-service bins, config) | Updates replace the whole binary, not individual components |
| `root.key` embedded at build time via `go:embed` | Oracle root-node builds require special CI handling |
| `Version`, `BuildTime`, `GitCommit` injected via `-ldflags` | Version comparison is reliable without metadata files |
| No new Go module dependencies | All HTTP, crypto, and syscall work uses stdlib |
| Runs as a systemd service (per `docs/SYSTEMD_SERVICE.md`) | Restart strategy must cooperate with the process supervisor |

---

## Architecture Decision: GitHub Releases + Atomic Binary Swap

Two viable patterns exist:

**A. Pull (polling):** Server periodically checks GitHub Releases API for a newer release asset, downloads, and self-replaces.

**B. Push (webhook):** CI calls an admin endpoint after a successful build; server downloads and self-replaces.

**Decision: A (Pull) as primary, B (Push) as optional fast-path.**

Polling is simpler to operate — no inbound webhook secret to manage on the CI side, works behind NAT/firewall, and is resilient to CI flakiness. An optional `POST /api/v1/update/trigger` endpoint can be added later for zero-latency deploys.

---

## End-to-End Flow

```
git push → main
    │
    ▼
GitHub Actions (auto-release.yml)
    ├── make binary          (full build with ldflags)
    ├── sha256sum knirv-server > knirv-server.sha256
    ├── gh release create v<sha> --prerelease
    └── upload: knirv-server, knirv-server.sha256
    
    ┌── (every N minutes, default 10) ──────────────────┐
    │   GET api.github.com/repos/KNIRV/…/releases/latest│
    │   compare release.tag_name vs GitCommit            │
    │   if newer → download asset → verify sha256        │
    │            → atomic swap → graceful restart        │
    └────────────────────────────────────────────────────┘
```

---

## Component Design

### 1. Updater Package: `backend/internal/updater/`

Three files, no new imports beyond stdlib.

#### `updater.go` — core types and entry point

```go
package updater

import (
    "crypto/sha256"
    "encoding/hex"
    "fmt"
    "io"
    "log"
    "net/http"
    "os"
    "os/exec"
    "path/filepath"
    "syscall"
    "time"
)

type Config struct {
    Enabled        bool
    PollInterval   time.Duration // default: 10 minutes
    GitHubRepo     string        // "KNIRV/KNIRV_NETWORK"
    GitHubToken    string        // optional, for private repos
    AssetName      string        // "knirv-server" (matches release asset filename)
    CurrentCommit  string        // injected from main.Version / GitCommit at startup
    BinaryPath     string        // absolute path to the running binary (os.Executable())
    HealthEndpoint string        // e.g. "http://localhost:8084/api/v1/health"
}

type Updater struct {
    cfg    Config
    client *http.Client
}

func New(cfg Config) *Updater {
    return &Updater{
        cfg:    cfg,
        client: &http.Client{Timeout: 30 * time.Second},
    }
}

// Start runs the background polling loop. Call in a goroutine.
func (u *Updater) Start() {
    if !u.cfg.Enabled {
        log.Println("[updater] disabled")
        return
    }
    log.Printf("[updater] polling every %s for repo %s", u.cfg.PollInterval, u.cfg.GitHubRepo)
    ticker := time.NewTicker(u.cfg.PollInterval)
    defer ticker.Stop()
    for range ticker.C {
        if err := u.checkAndApply(); err != nil {
            log.Printf("[updater] check failed: %v", err)
        }
    }
}
```

#### `github.go` — GitHub Releases API (stdlib only)

```go
package updater

import (
    "encoding/json"
    "fmt"
    "io"
    "net/http"
)

type githubRelease struct {
    TagName string          `json:"tag_name"`  // e.g. "abc1234" (short commit SHA)
    Assets  []githubAsset   `json:"assets"`
    Body    string          `json:"body"`      // can embed full commit SHA in release notes
}

type githubAsset struct {
    Name               string `json:"name"`
    BrowserDownloadURL string `json:"browser_download_url"`
    Size               int64  `json:"size"`
}

func (u *Updater) fetchLatestRelease() (*githubRelease, error) {
    url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", u.cfg.GitHubRepo)
    req, _ := http.NewRequest("GET", url, nil)
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
    return &rel, json.NewDecoder(resp.Body).Decode(&rel)
}
```

#### `apply.go` — download, verify, atomic swap, exec-replace

```go
package updater

import (
    "crypto/sha256"
    "encoding/hex"
    "fmt"
    "io"
    "log"
    "os"
    "os/exec"
    "path/filepath"
    "strings"
    "syscall"
)

func (u *Updater) checkAndApply() error {
    rel, err := u.fetchLatestRelease()
    if err != nil {
        return fmt.Errorf("fetch release: %w", err)
    }
    // tag_name is the short commit SHA set by CI: "$(git rev-parse --short HEAD)"
    if rel.TagName == u.cfg.CurrentCommit {
        return nil // already up-to-date
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

    dir := filepath.Dir(u.cfg.BinaryPath)
    newPath := u.cfg.BinaryPath + ".new"
    prevPath := u.cfg.BinaryPath + ".prev"

    if err := u.download(binaryURL, newPath); err != nil {
        return fmt.Errorf("download: %w", err)
    }
    defer os.Remove(newPath) // cleanup on any failure path

    if checksumURL != "" {
        expected, err := u.fetchChecksum(checksumURL)
        if err != nil {
            return fmt.Errorf("fetch checksum: %w", err)
        }
        if err := verifyChecksum(newPath, expected); err != nil {
            return fmt.Errorf("checksum mismatch: %w", err)
        }
    }

    if err := os.Chmod(newPath, 0755); err != nil {
        return err
    }

    // Rotate: current → .prev, .new → current
    // os.Rename is atomic on Linux (same filesystem)
    _ = os.Rename(u.cfg.BinaryPath, prevPath)
    if err := os.Rename(newPath, u.cfg.BinaryPath); err != nil {
        // attempt rollback
        _ = os.Rename(prevPath, u.cfg.BinaryPath)
        return fmt.Errorf("rename new binary: %w", err)
    }

    log.Printf("[updater] binary updated to %s, restarting...", rel.TagName)
    // syscall.Exec replaces the current process image — the OS re-reads the
    // binary from disk, so the new version starts with the same PID/fd set.
    // systemd sees no process death; it just observes the PID continue running.
    return syscall.Exec(u.cfg.BinaryPath, os.Args, os.Environ())
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
    // sha256sum format: "<hex>  filename"
    return strings.Fields(string(b))[0], err
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
```

### 2. Wire-up in `main.go`

In `main.go`, after loading config and before starting the HTTP servers:

```go
import "github.com/KNIRV/KNIRV_NETWORK/packages/KNIRVSERVER/backend/internal/updater"

// ...inside main(), after config is loaded:
selfPath, _ := os.Executable()
upd := updater.New(updater.Config{
    Enabled:       viper.GetBool("updater.enabled"),
    PollInterval:  viper.GetDuration("updater.poll_interval"),
    GitHubRepo:    viper.GetString("updater.github_repo"),
    GitHubToken:   os.Getenv("GITHUB_TOKEN"),
    AssetName:     "knirv-server",
    CurrentCommit: GitCommit,  // already injected via ldflags
    BinaryPath:    selfPath,
})
go upd.Start()
```

### 3. Config (`config/production.yaml`)

```yaml
updater:
  enabled: true
  poll_interval: 10m
  github_repo: "KNIRV/KNIRV_NETWORK"
  # github token read from GITHUB_TOKEN env var (fine-grained, read:packages + read:contents)
```

---

## GitHub Actions Workflow

Create `.github/workflows/auto-release.yml`:

```yaml
name: Auto Release

on:
  push:
    branches: [main]

jobs:
  build-and-release:
    runs-on: ubuntu-latest
    permissions:
      contents: write   # needed to create releases and upload assets

    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - uses: actions/setup-go@v5
        with:
          go-version: '1.21'

      - name: Set up Node
        uses: actions/setup-node@v4
        with:
          node-version: '18'

      - name: Build unified binary
        working-directory: packages/KNIRVSERVER
        env:
          # root.key is NOT available in CI — oracle builds are separate (see below)
          SKIP_ROOT_KEY: "1"
        run: make binary

      - name: Checksum
        working-directory: packages/KNIRVSERVER/dist
        run: sha256sum knirv-server > knirv-server.sha256

      - name: Create / update release
        env:
          GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
        run: |
          TAG=$(git rev-parse --short HEAD)
          gh release delete "$TAG" --yes 2>/dev/null || true
          gh release create "$TAG" \
            --title "auto-release $TAG" \
            --notes "Automated build from commit ${{ github.sha }}" \
            --prerelease \
            packages/KNIRVSERVER/dist/knirv-server \
            packages/KNIRVSERVER/dist/knirv-server.sha256
```

---

## root.key / Oracle Node Consideration

The current build embeds `bin/root.key` at compile time via `go:embed`. This creates a conflict with automated CI builds because:

1. The key must not be stored in the repository or CI secrets in plaintext.
2. Oracle nodes need the key; non-oracle nodes do not.

**Recommended resolution:**

Move to a **sidecar key path** for production oracle nodes instead of embedding:

```go
// In main.go, replace the go:embed approach with:
func loadRootKey() []byte {
    // 1. Try sidecar path (production)
    if path := os.Getenv("ROOT_KEY_PATH"); path != "" {
        if b, err := os.ReadFile(path); err == nil {
            return b
        }
    }
    // 2. Fall back to embedded bytes (dev/local builds where bin/root.key exists)
    return rootKeyBytes // still go:embed, but nil when file absent
}
```

Non-oracle builds (what CI produces) simply have `bin/root.key` absent — the `go:embed` directive already tolerates this if the file is missing (the byte slice stays nil). Oracle nodes keep their key on disk at `ROOT_KEY_PATH`, which the new binary picks up after restart without needing a special build.

---

## Rollback Strategy

### Passive rollback (binary)
`apply.go` renames the current binary to `knirv-server.prev` before swapping in the new one. To roll back manually:

```bash
mv /usr/local/bin/knirv-server /usr/local/bin/knirv-server.bad
mv /usr/local/bin/knirv-server.prev /usr/local/bin/knirv-server
systemctl restart knirv-server
```

### Active rollback (post-restart health check)
A small watchdog can be added to `main.go` that, after startup, waits 30 seconds and probes `GET /api/v1/health`. If the probe fails, it calls rollback:

```go
func rollbackIfUnhealthy(binaryPath, healthURL string) {
    time.Sleep(30 * time.Second)
    resp, err := http.Get(healthURL)
    if err != nil || resp.StatusCode != 200 {
        prevPath := binaryPath + ".prev"
        if _, err := os.Stat(prevPath); err == nil {
            log.Println("[updater] health check failed, rolling back...")
            _ = os.Rename(binaryPath, binaryPath+".bad")
            _ = os.Rename(prevPath, binaryPath)
            _ = syscall.Exec(binaryPath, os.Args, os.Environ())
        }
    }
}
```

---

## Admin API Endpoints

Add these to the existing Gin router:

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/update/status` | Current commit, latest available release tag, update enabled/disabled |
| `POST` | `/api/v1/update/trigger` | Immediately trigger an update check (protected, admin auth required) |
| `GET` | `/api/v1/update/history` | Last N update events (in-memory ring buffer) |

---

## Security Considerations

- **GitHub token scope:** Fine-grained PAT with `contents: read` on the KNIRV_NETWORK repo only. Never give write access to a token that lives on a server.
- **Asset integrity:** SHA256 checksum is verified before the binary is made executable. If the checksum file is absent the download is still accepted (log a warning). Consider adding GPG signing for stronger guarantees.
- **Admin trigger endpoint:** Must sit behind the same auth middleware as other admin routes. Rate-limit to 1 request/minute.
- **Filesystem permissions:** The binary directory should be owned by the `knirv-server` service user. The `.prev` and `.new` files are written to the same directory, so no cross-filesystem renames occur (atomic guarantee holds).
- **Private repo:** Set `GITHUB_TOKEN` env var on each production node. The GitHub Releases API for a private repo requires authentication; the updater already passes the token as a `Bearer` header.

---

## Implementation Order

1. Create `backend/internal/updater/` with the three files above.
2. Add `updater.*` keys to `config/production.yaml` and `config/development.yaml` (disabled in dev).
3. Wire `updater.New(...).Start()` into `main.go`.
4. Create `.github/workflows/auto-release.yml`.
5. Adjust `bin/root.key` embed to tolerate absence (already works if file is simply not present in CI workspace).
6. Test with a dummy commit: observe polling, download, and `syscall.Exec` restart.
7. Add rollback watchdog.
8. Add admin API endpoints.
