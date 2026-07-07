package cloudflare

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// EnsureCloudflared returns the path to cloudflared, installing it if needed.
func EnsureCloudflared() (string, error) {
	if p, err := findCloudflared(); err == nil {
		return p, nil
	}
	return installCloudflared()
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
	resp, err := http.Get(url) // #nosec G107
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
	resp, err := http.Get(url) // #nosec G107
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
