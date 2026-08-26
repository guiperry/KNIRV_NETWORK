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
	if binary == "x11vnc" || binary == "zeek" {
		id, idLike := readOSRelease()
		enableExtraReposForTool(binary, pm, id, idLike, runner)
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

// dotnetToolStrategy mirrors pipStrategy for .NET global tools.  It keeps
// ILSpy self-contained under the engine bundle instead of modifying the
// operator's global dotnet tool store.
type dotnetToolStrategy struct {
	packageName string
}

func (d dotnetToolStrategy) toolDir() string { return filepath.Join(sandboxToolsDir(), "dotnettools") }

func (d dotnetToolStrategy) present(binary string) bool {
	if info, err := os.Stat(filepath.Join(d.toolDir(), binary)); err == nil && !info.IsDir() {
		return true
	}
	return binaryExists(binary)
}

func (d dotnetToolStrategy) install(_ string, _ commandRunner) error {
	dotnet := resolveSandboxTool("dotnet")
	if dotnet == "dotnet" {
		var err error
		dotnet, err = exec.LookPath("dotnet")
		if err != nil {
			return fmt.Errorf("dotnet runtime not found: %v", err)
		}
	}
	if err := os.MkdirAll(d.toolDir(), 0o755); err != nil {
		return fmt.Errorf("failed to create managed dotnet tools directory: %v", err)
	}
	return unprivilegedCommandRunner(dotnet, "tool", "install", "--tool-path", d.toolDir(), d.packageName)
}

func (d dotnetToolStrategy) manualCommand(_ string) string {
	return fmt.Sprintf("dotnet tool install --tool-path %s %s", d.toolDir(), d.packageName)
}

func (p pipStrategy) present(binary string) bool {
	venvBin := filepath.Join(p.venvDir(), "bin", binary)
	if _, err := os.Stat(venvBin); err == nil {
		return true
	}
	return binaryExists(binary)
}

func (p pipStrategy) install(_ string, _ commandRunner) error {
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
	if err := unprivilegedCommandRunner(python, "-m", "venv", venvDir); err != nil {
		return fmt.Errorf("venv creation failed: %v", err)
	}
	pip := filepath.Join(venvDir, "bin", "pip")
	module := p.module
	if module == "" {
		module = p.binaryName()
	}
	return unprivilegedCommandRunner(pip, "install", module)
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

func (g githubReleaseStrategy) install(binary string, _ commandRunner) error {
	pattern, err := regexp.Compile(g.assetPattern)
	if err != nil {
		return fmt.Errorf("invalid asset pattern: %v", err)
	}
	asset, err := g.findReleaseAsset(pattern)
	if err != nil {
		return err
	}
	archivePath := filepath.Join(os.TempDir(), fmt.Sprintf("tool-%s-%s", binary, asset.Name))
	if err := downloadFile(asset.URL, archivePath); err != nil {
		return fmt.Errorf("download failed: %v", err)
	}
	defer os.Remove(archivePath)
	return g.extractNamedBinary(archivePath, sandboxToolsDir(), binary, unprivilegedCommandRunner)
}

func (g githubReleaseStrategy) manualCommand(binary string) string {
	return fmt.Sprintf("Download %s release from https://github.com/%s and extract to %s",
		binary, g.repo, sandboxToolsDir())
}

// toolAcquireStrategy maps a tool name to its acquisition strategy.
// Tools not listed here fall back to packageManagerStrategy.
var toolAcquireStrategies = map[string]acquireStrategy{
	"semgrep":      pipStrategy{module: "semgrep"},
	"jwt_tool.py":  pipStrategy{module: "jwt_tool"},
	"frida":        pipStrategy{module: "frida-tools"},
	"frida-server": githubReleaseStrategy{repo: "frida/frida", assetPattern: fridaServerAssetPattern()},
	"jadx":         githubReleaseStrategy{repo: "skylot/jadx", assetPattern: `^jadx-[0-9]+\.[0-9]+\.[0-9]+\.zip$`},
	"ilspycmd":     dotnetToolStrategy{packageName: "ilspycmd"},
	"java":         packageManagerStrategy{},
	"dotnet":       packageManagerStrategy{},
	"bpftrace":     packageManagerStrategy{},
	"tshark":       packageManagerStrategy{},
	"zeek":         packageManagerStrategy{},
	"proxychains4": packageManagerStrategy{},
	"afl-fuzz":     packageManagerStrategy{},
	"rizin":        packageManagerStrategy{},
	"rz-bin":       packageManagerStrategy{},
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
		"rz-bin":       {"apt-get": "rizin", "dnf": "rizin", "microdnf": "rizin", "yum": "rizin", "pacman": "rizin", "zypper": "rizin", "apk": "rizin"},
		"bwrap":        {"apt-get": "bubblewrap", "dnf": "bubblewrap", "microdnf": "bubblewrap", "yum": "bubblewrap", "pacman": "bubblewrap", "zypper": "bubblewrap", "apk": "bubblewrap"},
		"Xvfb":         {"apt-get": "xvfb", "dnf": "xorg-x11-server-Xvfb", "microdnf": "xorg-x11-server-Xvfb", "yum": "xorg-x11-server-Xvfb", "pacman": "xorg-server-xvfb", "zypper": "xorg-x11-server", "apk": "xvfb"},
		"x11vnc":       {"apt-get": "x11vnc", "dnf": "x11vnc", "microdnf": "x11vnc", "yum": "x11vnc", "pacman": "x11vnc", "zypper": "x11vnc", "apk": "x11vnc"},
		"java":         {"apt-get": "default-jre", "dnf": "java-17-openjdk-headless", "microdnf": "java-17-openjdk-headless", "yum": "java-17-openjdk-headless", "pacman": "jre-openjdk-headless", "zypper": "java-17-openjdk-headless", "apk": "openjdk17-jre-headless"},
		"dotnet":       {"apt-get": "dotnet-runtime-8.0", "dnf": "dotnet-runtime-8.0", "microdnf": "dotnet-runtime-8.0", "yum": "dotnet-runtime-8.0", "pacman": "dotnet-runtime", "zypper": "dotnet-runtime-8.0", "apk": "dotnet8-runtime"},
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
	if _, ok := strategy.(packageManagerStrategy); ok {
		if pm, err := detectPackageManager(); err == nil {
			st.Package = packageForManager(binary, pm)
		}
	}
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

// ensureToolAvailable is the execution-lane gate. It gives every real console
// the same acquire-on-first-use behavior as Bubble Wrap's dependency setup,
// while retaining an actionable error when the host cannot provide a tool.
func ensureToolAvailable(binary string) error {
	for _, prerequisite := range toolPrerequisites[binary] {
		if err := ensureSingleToolAvailable(prerequisite); err != nil {
			return fmt.Errorf("%s prerequisite: %w", binary, err)
		}
	}
	return ensureSingleToolAvailable(binary)
}

// Some acquisition methods install only the tool itself. Make their runtime
// prerequisites explicit so direct tool API calls are as complete as the
// dependency banner path.
var toolPrerequisites = map[string][]string{
	"jadx":     {"java"},
	"ilspycmd": {"dotnet"},
	"frida":    {"frida-server"},
}

func ensureSingleToolAvailable(binary string) error {
	status := EnsureToolDependency(binary, realCommandRunner)
	if status.Present {
		return nil
	}
	if status.Error != "" {
		return fmt.Errorf("%s is unavailable: %s", binary, status.Error)
	}
	if status.InstallCommand != "" {
		return fmt.Errorf("%s is unavailable; install it with: %s", binary, status.InstallCommand)
	}
	return fmt.Errorf("%s is unavailable", binary)
}

// unprivilegedCommandRunner is deliberately used by managed venv, dotnet, and
// release installs. Unlike package managers, these strategies must never sudo:
// they only write within the engine's own tools directory.
func unprivilegedCommandRunner(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s %v: %v (%s)", name, args, err, strings.TrimSpace(string(out)))
	}
	return nil
}

type githubReleaseAsset struct {
	Name string
	URL  string
}

// findReleaseAsset queries the GitHub releases API for the latest matching
// asset and keeps the name so extraction can select the correct archive type.
func (g githubReleaseStrategy) findReleaseAsset(pattern *regexp.Regexp) (githubReleaseAsset, error) {
	if g.repo == "" {
		return githubReleaseAsset{}, fmt.Errorf("no repo specified")
	}
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", g.repo)
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return githubReleaseAsset{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return githubReleaseAsset{}, fmt.Errorf("GitHub API request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return githubReleaseAsset{}, fmt.Errorf("GitHub API returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var release struct {
		Assets []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return githubReleaseAsset{}, fmt.Errorf("failed to decode GitHub release: %v", err)
	}
	for _, asset := range release.Assets {
		if pattern.MatchString(asset.Name) {
			return githubReleaseAsset{Name: asset.Name, URL: asset.BrowserDownloadURL}, nil
		}
	}
	return githubReleaseAsset{}, fmt.Errorf("no asset matching %q in %s release", g.assetPattern, g.repo)
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

func (g githubReleaseStrategy) extractNamedBinary(archivePath, destDir, binary string, runner commandRunner) error {
	if binary == "" {
		return fmt.Errorf("release binary name is required")
	}
	if strings.HasSuffix(archivePath, ".xz") {
		if err := runner("xz", "-dkf", archivePath); err != nil {
			return err
		}
		return copyExecutable(strings.TrimSuffix(archivePath, ".xz"), filepath.Join(destDir, binary))
	}

	// JADX's launcher is a script with sibling lib/ resources. Preserve the
	// release layout and create a stable top-level launcher for resolver use.
	if binary == "jadx" && strings.HasSuffix(archivePath, ".zip") {
		installDir := filepath.Join(destDir, "jadx-release")
		if err := os.RemoveAll(installDir); err != nil {
			return err
		}
		if err := os.MkdirAll(installDir, 0o755); err != nil {
			return err
		}
		if err := runner("unzip", "-o", archivePath, "-d", installDir); err != nil {
			return err
		}
		source, err := findReleaseBinary(installDir, binary)
		if err != nil {
			return err
		}
		launcher := "#!/bin/sh\nexec " + shellQuote(source) + " \"$@\"\n"
		return os.WriteFile(filepath.Join(destDir, binary), []byte(launcher), 0o755)
	}

	extractDir, err := os.MkdirTemp(destDir, ".tool-extract-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(extractDir)
	if strings.HasSuffix(archivePath, ".zip") {
		err = runner("unzip", "-o", archivePath, "-d", extractDir)
	} else {
		err = runner("tar", "-xzf", archivePath, "-C", extractDir)
	}
	if err != nil {
		return err
	}
	source, err := findReleaseBinary(extractDir, binary)
	if err != nil {
		return err
	}
	return copyExecutable(source, filepath.Join(destDir, binary))
}

func findReleaseBinary(root, binary string) (string, error) {
	var found string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() && filepath.Base(path) == binary {
			found = path
			return filepath.SkipAll
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if found == "" {
		return "", fmt.Errorf("release archive did not contain %q", binary)
	}
	return found, nil
}

func copyExecutable(source, dest string) error {
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

func fridaServerAssetPattern() string {
	arch := "x86_64"
	if runtime.GOARCH == "arm64" {
		arch = "arm64"
	}
	return `^frida-server-[0-9.]+-linux-` + arch + `\.xz$`
}
