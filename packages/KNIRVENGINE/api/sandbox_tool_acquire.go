package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

// acquireStrategy abstracts how a tool binary is acquired. The existing
// package-manager approach becomes packageManagerStrategy; pip and GitHub
// release binaries add two more. Every strategy still reports through the
// same DependencyStatus struct so Bubblewrap.tsx's existing missing-deps
// banner needs zero frontend changes.
type acquireStrategy interface {
	// present reports whether the binary is available (bundled, PATH, or venv).
	present(binary string) bool
	// install attempts to acquire the binary. Returns nil on success.
	install(binary string, runner commandRunner) error
	// manualCommand returns a human-runnable fallback command.
	manualCommand(binary string) string
}

// packageManagerStrategy acquires tools via the system package manager.
// This is the existing behavior, refactored out of EnsureSandboxDependencies.
type packageManagerStrategy struct{}

func (packageManagerStrategy) present(binary string) bool {
	return binaryExists(binary)
}

func (packageManagerStrategy) install(binary string, runner commandRunner) error {
	pm, err := detectPackageManager()
	if err != nil {
		return err
	}
	pkg := packageForManager(binary, pm)
	if pkg == "" {
		return fmt.Errorf("no package mapping for %s on %s", binary, pm)
	}
	return installPackages(pm, []string{pkg}, runner)
}

func (packageManagerStrategy) manualCommand(binary string) string {
	pm, err := detectPackageManager()
	if err != nil {
		return fmt.Sprintf("# install %s manually", binary)
	}
	pkg := packageForManager(binary, pm)
	return manualInstallCommand(pm, []string{pkg})
}

// pipStrategy installs Python tools into a managed venv under the sandbox
// tools directory (not the system Python, so no root needed).
type pipStrategy struct {
	module string // pip module name (defaults to binary if empty)
}

func (p pipStrategy) present(binary string) bool {
	venvBin := filepath.Join(p.venvDir(), "bin", binary)
	if _, err := os.Stat(venvBin); err == nil {
		return true
	}
	return binaryExists(binary)
}

func (p pipStrategy) install(_ string, runner commandRunner) error {
	venvDir := p.venvDir()
	if err := os.MkdirAll(venvDir, 0o755); err != nil {
		return fmt.Errorf("failed to create venv dir: %v", err)
	}
	python, err := exec.LookPath("python3")
	if err != nil {
		python, err = exec.LookPath("python")
		if err != nil {
			return fmt.Errorf("python not found: %v", err)
		}
	}
	if err := runner(python, "-m", "venv", venvDir); err != nil {
		return fmt.Errorf("venv creation failed: %v", err)
	}
	pip := filepath.Join(venvDir, "bin", "pip")
	module := p.module
	if module == "" {
		module = p.binaryName()
	}
	return runner(pip, "install", module)
}

func (p pipStrategy) manualCommand(_ string) string {
	module := p.module
	if module == "" {
		module = p.binaryName()
	}
	return fmt.Sprintf("python3 -m venv <dir> && <dir>/bin/pip install %s", module)
}

func (p pipStrategy) venvDir() string {
	return filepath.Join(sandboxToolsDir(), "pyenv")
}

func (p pipStrategy) binaryName() string {
	// Default: module name == binary name. Override via module field.
	return p.module
}

// githubReleaseStrategy downloads a release binary from a GitHub repo.
// assetPattern is a regex matched against release asset names; the first
// match is downloaded and extracted into sandboxToolsDir().
type githubReleaseStrategy struct {
	repo         string
	assetPattern string
}

func (g githubReleaseStrategy) present(binary string) bool {
	return binaryExists(binary)
}

func (g githubReleaseStrategy) install(binary string, runner commandRunner) error {
	pattern, err := regexp.Compile(g.assetPattern)
	if err != nil {
		return fmt.Errorf("invalid asset pattern: %v", err)
	}
	url, err := g.findReleaseURL(pattern)
	if err != nil {
		return err
	}
	archivePath := filepath.Join(os.TempDir(), fmt.Sprintf("tool-%s-release", binary))
	if err := downloadFile(url, archivePath); err != nil {
		return fmt.Errorf("download failed: %v", err)
	}
	defer os.Remove(archivePath)
	return g.extractBinary(archivePath, sandboxToolsDir(), runner)
}

func (g githubReleaseStrategy) manualCommand(binary string) string {
	return fmt.Sprintf("Download %s release from https://github.com/%s and extract to %s",
		binary, g.repo, sandboxToolsDir())
}

// toolAcquireStrategy maps a tool name to its acquisition strategy.
// Tools not listed here fall back to packageManagerStrategy.
var toolAcquireStrategies = map[string]acquireStrategy{
	"semgrep":      pipStrategy{module: "semgrep"},
	"jwt_tool":     pipStrategy{module: "jwt_tool"},
	"bpftrace":     packageManagerStrategy{},
	"tshark":       packageManagerStrategy{},
	"zeek":         packageManagerStrategy{},
	"proxychains4": packageManagerStrategy{},
	"afl-fuzz":     packageManagerStrategy{},
	"rizin":        packageManagerStrategy{},
}

// acquireStrategyFor returns the acquisition strategy for a tool.
func acquireStrategyFor(binary string) acquireStrategy {
	if s, ok := toolAcquireStrategies[binary]; ok {
		return s
	}
	return packageManagerStrategy{}
}

// packageForManager returns the package name for a binary on a given manager.
func packageForManager(binary, pm string) string {
	pkgs := map[string]map[string]string{
		"bpftrace":     {"apt-get": "bpftrace", "dnf": "bpftrace", "microdnf": "bpftrace", "yum": "bpftrace", "pacman": "bpftrace", "zypper": "bpftrace", "apk": "bpftrace"},
		"tshark":       {"apt-get": "tshark", "dnf": "wireshark-cli", "microdnf": "wireshark-cli", "yum": "wireshark-cli", "pacman": "wireshark-cli", "zypper": "wireshark", "apk": "tshark"},
		"zeek":         {"apt-get": "zeek", "dnf": "zeek", "microdnf": "zeek", "yum": "zeek", "pacman": "zeek", "zypper": "zeek", "apk": "zeek"},
		"proxychains4": {"apt-get": "proxychains4", "dnf": "proxychains-ng", "microdnf": "proxychains-ng", "yum": "proxychains-ng", "pacman": "proxychains4", "zypper": "proxychains-ng", "apk": "proxychains-ng"},
		"afl-fuzz":     {"apt-get": "afl++", "dnf": "aflplusplus", "microdnf": "aflplusplus", "yum": "aflplusplus", "pacman": "aflplusplus", "zypper": "aflplusplus", "apk": "afl++"},
		"rizin":        {"apt-get": "rizin", "dnf": "rizin", "microdnf": "rizin", "yum": "rizin", "pacman": "rizin", "zypper": "rizin", "apk": "rizin"},
		"jadx":         {}, // GitHub release only
		"ilspycmd":     {}, // dotnet tool only
	}
	if m, ok := pkgs[binary]; ok {
		if p, ok := m[pm]; ok {
			return p
		}
	}
	return ""
}

// EnsureToolDependency checks whether a single tool is present and attempts
// to install it using the appropriate strategy. Returns a DependencyStatus.
func EnsureToolDependency(binary string, runner commandRunner) DependencyStatus {
	if runtime.GOOS != "linux" {
		return DependencyStatus{
			Binary: binary,
			Error:  fmt.Sprintf("tool acquisition is only supported on Linux (current OS: %s)", runtime.GOOS),
		}
	}
	strategy := acquireStrategyFor(binary)
	st := DependencyStatus{Binary: binary, Present: strategy.present(binary)}
	if st.Present {
		return st
	}
	st.InstallCommand = strategy.manualCommand(binary)
	if err := strategy.install(binary, runner); err != nil {
		st.Error = err.Error()
		return st
	}
	if strategy.present(binary) {
		st.Present = true
		st.InstallCommand = ""
	} else {
		st.Error = "installation reported success but binary still not found"
	}
	return st
}

// findReleaseURL queries the GitHub releases API for the latest release whose
// asset name matches pattern, and returns the asset's download URL.
func (g githubReleaseStrategy) findReleaseURL(pattern *regexp.Regexp) (string, error) {
	if g.repo == "" {
		return "", fmt.Errorf("no repo specified")
	}
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", g.repo)
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("GitHub API request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("GitHub API returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var release struct {
		Assets []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", fmt.Errorf("failed to decode GitHub release: %v", err)
	}
	for _, asset := range release.Assets {
		if pattern.MatchString(asset.Name) {
			return asset.BrowserDownloadURL, nil
		}
	}
	return "", fmt.Errorf("no asset matching %q in %s release", g.assetPattern, g.repo)
}

// downloadFile downloads a URL to a local path.
func downloadFile(url, dest string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned %d", resp.StatusCode)
	}
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}

// extractBinary extracts a single binary from a tarball/zip into destDir.
// Supports .tar.gz and .zip archives.
func (g githubReleaseStrategy) extractBinary(archivePath, destDir string, runner commandRunner) error {
	if strings.HasSuffix(archivePath, ".zip") {
		return runner("unzip", "-o", archivePath, "-d", destDir)
	}
	return runner("tar", "-xzf", archivePath, "-C", destDir)
}
