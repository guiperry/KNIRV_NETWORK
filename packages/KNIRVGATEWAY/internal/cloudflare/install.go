package cloudflare

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// latestReleaseURL reports the newest cloudflared release; its tag is the
// date-based version (e.g. "2025.10.1") that `cloudflared --version` prints.
const latestReleaseURL = "https://api.github.com/repos/cloudflare/cloudflared/releases/latest"

// Seams for tests.
var (
	httpGet       = http.Get
	versionOutput = func(path string) (string, error) {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		out, err := exec.CommandContext(ctx, path, "--version").CombinedOutput()
		return string(out), err
	}
)

// EnsureCloudflared returns the path to cloudflared, installing it if missing
// and updating it when a newer release exists. A failed check or update keeps
// the working binary, so an offline host still starts. CLOUDFLARED_PATH pins an
// explicit binary and CLOUDFLARED_UPDATE=0 disables the check.
func EnsureCloudflared() (string, error) {
	p, err := findCloudflared()
	if err != nil {
		return installCloudflared()
	}
	if os.Getenv("CLOUDFLARED_PATH") == "" && os.Getenv("CLOUDFLARED_UPDATE") != "0" {
		return updateCloudflared(p), nil
	}
	return p, nil
}

var versionPattern = regexp.MustCompile(`(\d+(?:\.\d+)+)`)

// parseVersion extracts the numeric parts of a version such as "2025.10.1".
func parseVersion(s string) ([]int, bool) {
	m := versionPattern.FindString(s)
	if m == "" {
		return nil, false
	}
	var v []int
	for _, part := range strings.Split(m, ".") {
		n, err := strconv.Atoi(part)
		if err != nil {
			return nil, false
		}
		v = append(v, n)
	}
	return v, true
}

// versionLess reports whether a is older than b.
func versionLess(a, b []int) bool {
	for i := 0; i < len(a) || i < len(b); i++ {
		var x, y int
		if i < len(a) {
			x = a[i]
		}
		if i < len(b) {
			y = b[i]
		}
		if x != y {
			return x < y
		}
	}
	return false
}

func latestCloudflaredVersion() (string, []int, error) {
	resp, err := httpGet(latestReleaseURL) // #nosec G107
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	var rel struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&rel); err != nil {
		return "", nil, err
	}
	v, ok := parseVersion(rel.TagName)
	if !ok {
		return "", nil, fmt.Errorf("unrecognised release tag %q", rel.TagName)
	}
	return rel.TagName, v, nil
}

// updateCloudflared replaces the binary at path with the latest release when it
// is older and returns the path to use, which is path itself whenever nothing
// changed or the update failed. The new binary is staged beside the old one and
// verified before it is renamed over it, so the working binary is never lost.
func updateCloudflared(path string) string {
	tag, latest, err := latestCloudflaredVersion()
	if err != nil {
		log.Printf("[cloudflare] could not check the latest cloudflared release (%v); keeping %s", err, path)
		return path
	}
	out, err := versionOutput(path)
	installed, ok := parseVersion(out)
	if !ok {
		log.Printf("[cloudflare] could not read the installed cloudflared version (%v); updating to %s", err, tag)
	} else if !versionLess(installed, latest) {
		log.Printf("[cloudflare] cloudflared %s is up to date (latest %s)", versionString(installed), tag)
		return path
	} else {
		log.Printf("[cloudflare] cloudflared %s is older than %s; updating", versionString(installed), tag)
	}

	staged := path + ".update"
	if err := fetchCloudflared(staged); err != nil {
		_ = os.Remove(staged)
		log.Printf("[cloudflare] cloudflared update to %s failed (%v); keeping %s", tag, err, path)
		return path
	}
	out, err = versionOutput(staged)
	got, ok := parseVersion(out)
	if !ok || versionLess(got, latest) {
		_ = os.Remove(staged)
		log.Printf("[cloudflare] downloaded cloudflared failed verification (%v); keeping %s", err, path)
		return path
	}
	if err := os.Rename(staged, path); err != nil {
		_ = os.Remove(staged)
		log.Printf("[cloudflare] could not replace %s (%v); keeping the current version", path, err)
		return path
	}
	log.Printf("[cloudflare] updated cloudflared to %s", versionString(got))
	return path
}

func versionString(v []int) string {
	parts := make([]string, len(v))
	for i, n := range v {
		parts[i] = strconv.Itoa(n)
	}
	return strings.Join(parts, ".")
}

// fetchCloudflared downloads the latest release for this platform to dest.
func fetchCloudflared(dest string) error {
	switch runtime.GOOS {
	case "linux":
		switch runtime.GOARCH {
		case "amd64", "arm64", "arm":
			return downloadBinary(fmt.Sprintf("https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-%s", runtime.GOARCH), dest)
		}
	case "darwin":
		switch runtime.GOARCH {
		case "amd64", "arm64":
			return downloadTarGzBinary(fmt.Sprintf("https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-darwin-%s.tgz", runtime.GOARCH), "cloudflared", dest)
		}
	}
	return fmt.Errorf("auto-update unsupported on %s/%s", runtime.GOOS, runtime.GOARCH)
}

func installCloudflared() (string, error) {
	switch runtime.GOOS {
	case "linux":
		return installCloudflaredLinux()
	case "darwin":
		return installCloudflaredDarwin()
	default:
		return "", fmt.Errorf(
			"cloudflared not found; auto-install unsupported on %s — install from %s",
			runtime.GOOS, cloudflaredInstallURL,
		)
	}
}

const cloudflaredInstallURL = "https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/downloads/"

func installCloudflaredLinux() (string, error) {
	arch := runtime.GOARCH
	switch arch {
	case "amd64", "arm64", "arm":
	default:
		return "", fmt.Errorf("cloudflared auto-install unsupported on linux/%s — install from %s", arch, cloudflaredInstallURL)
	}

	url := fmt.Sprintf(
		"https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-%s",
		arch,
	)

	installPath := "/usr/local/bin/cloudflared"
	if err := downloadBinary(url, installPath); err != nil {
		// Fall back to user-local path if /usr/local/bin requires root.
		local := filepath.Join(os.Getenv("HOME"), ".local", "bin", "cloudflared")
		if err2 := os.MkdirAll(filepath.Dir(local), 0755); err2 != nil {
			return "", fmt.Errorf("cloudflared install: %w (fallback mkdir: %v)", err, err2)
		}
		if err2 := downloadBinary(url, local); err2 != nil {
			return "", fmt.Errorf("cloudflared install to %s: %w", local, err2)
		}
		installPath = local
	}

	return installPath, nil
}

func installCloudflaredDarwin() (string, error) {
	// Prefer homebrew — cleaner and handles updates.
	if brew, err := exec.LookPath("brew"); err == nil {
		if out, err := exec.Command(brew, "install", "cloudflared").CombinedOutput(); err != nil {
			return "", fmt.Errorf("brew install cloudflared: %w\n%s", err, out)
		}
		if p, err := findCloudflared(); err == nil {
			return p, nil
		}
	}

	// Fall back to direct tgz download.
	arch := runtime.GOARCH
	switch arch {
	case "amd64", "arm64":
	default:
		return "", fmt.Errorf("cloudflared auto-install unsupported on darwin/%s", arch)
	}

	url := fmt.Sprintf(
		"https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-darwin-%s.tgz",
		arch,
	)

	installPath := "/usr/local/bin/cloudflared"
	if err := downloadTarGzBinary(url, "cloudflared", installPath); err != nil {
		local := filepath.Join(os.Getenv("HOME"), ".local", "bin", "cloudflared")
		if err2 := os.MkdirAll(filepath.Dir(local), 0755); err2 != nil {
			return "", fmt.Errorf("cloudflared install: %w (fallback mkdir: %v)", err, err2)
		}
		if err2 := downloadTarGzBinary(url, "cloudflared", local); err2 != nil {
			return "", fmt.Errorf("cloudflared install to %s: %w", local, err2)
		}
		installPath = local
	}

	return installPath, nil
}

func downloadBinary(url, dest string) error {
	resp, err := httpGet(url) // #nosec G107
	if err != nil {
		return fmt.Errorf("download %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download %s: HTTP %d", url, resp.StatusCode)
	}

	f, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	return err
}

func downloadTarGzBinary(url, binaryName, dest string) error {
	resp, err := httpGet(url) // #nosec G107
	if err != nil {
		return fmt.Errorf("download %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download %s: HTTP %d", url, resp.StatusCode)
	}

	gz, err := gzip.NewReader(resp.Body)
	if err != nil {
		return fmt.Errorf("gzip reader: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("tar read: %w", err)
		}
		if filepath.Base(hdr.Name) != binaryName {
			continue
		}
		f, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = io.Copy(f, tr) // #nosec G110
		return err
	}

	return fmt.Errorf("binary %q not found in archive", binaryName)
}
