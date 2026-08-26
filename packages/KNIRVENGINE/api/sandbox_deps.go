package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// DependencyStatus describes the availability of a single system binary the
// sandbox pipeline depends on (bubblewrap, Xvfb, x11vnc).
type DependencyStatus struct {
	Binary         string `json:"binary"`
	Present        bool   `json:"present"`
	Package        string `json:"package,omitempty"`
	InstallCommand string `json:"installCommand,omitempty"`
	Error          string `json:"error,omitempty"`
}

func isRequiredSandboxDependency(binary string) bool {
	switch binary {
	case "bwrap", "Xvfb", "x11vnc":
		return true
	default:
		return false
	}
}

// packageManagers is the ordered list of managers we know how to drive, by the
// command used to invoke them. Order matters: prefer dnf over yum, and
// apt-get over apt.
var packageManagers = []struct {
	command string
	manager string
}{
	{"apt-get", "apt-get"},
	{"dnf", "dnf"},
	{"microdnf", "microdnf"},
	{"yum", "yum"},
	{"pacman", "pacman"},
	{"zypper", "zypper"},
	{"apk", "apk"},
}

// commandRunner abstracts command execution so tests can stub it. The signature
// mirrors exec.Command(name, args...).
type commandRunner func(name string, args ...string) error

// realCommandRunner executes a system command, elevating only that package
// operation when needed. It never changes the privilege of the Electron host
// or the long-lived engine process.
func realCommandRunner(name string, args ...string) error {
	cmdArgs := args
	if os.Geteuid() != 0 {
		cmdArgs = append([]string{"-n", "--", name}, args...)
		name = "sudo"
	}
	cmd := exec.Command(name, cmdArgs...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s %v: %v (%s)", name, cmdArgs, err, strings.TrimSpace(string(out)))
	}
	return nil
}

// detectPackageManager returns the package manager available on the host by
// scanning PATH for known manager commands, in priority order.
func detectPackageManager() (string, error) {
	for _, pm := range packageManagers {
		if _, err := exec.LookPath(pm.command); err == nil {
			return pm.manager, nil
		}
	}
	return "", fmt.Errorf("no supported package manager found (looked for apt-get, dnf, microdnf, yum, pacman, zypper, apk)")
}

// readOSRelease parses /etc/os-release and returns the ID and ID_LIKE fields.
func readOSRelease() (id string, idLike string) {
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return "", ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "ID="):
			id = strings.Trim(strings.TrimPrefix(line, "ID="), "\"")
		case strings.HasPrefix(line, "ID_LIKE="):
			idLike = strings.Trim(strings.TrimPrefix(line, "ID_LIKE="), "\"")
		}
	}
	return id, idLike
}

// isRHELLike reports whether the distro is in the RHEL/Fedora family and thus
// needs EPEL for x11vnc.
func isRHELLike(id, idLike string) bool {
	if id == "fedora" || id == "rhel" || id == "centos" || id == "rocky" || id == "alma" || id == "ol" {
		return true
	}
	for _, f := range strings.Fields(idLike) {
		if f == "rhel" || f == "fedora" || f == "centos" {
			return true
		}
	}
	return false
}

// enableExtraRepos best-effort enables the repository that provides x11vnc on
// the detected platform (EPEL on RHEL/Fedora, universe on Ubuntu). Failures are
// ignored because the package may already be reachable.
func enableExtraRepos(pm, id, idLike string, runner commandRunner) {
	switch pm {
	case "dnf", "microdnf", "yum":
		if isRHELLike(id, idLike) {
			_ = runner(pm, "install", "-y", "epel-release")
		}
	case "apt-get":
		if id == "ubuntu" {
			_ = runner("add-apt-repository", "-y", "universe")
			_ = runner("apt-get", "update")
		}
	}
}

// enableExtraReposForTool extends the existing EPEL/universe precedent with
// Zeek's upstream repository. Zeek is absent from most default distributions,
// so its repository must be enabled before the normal package strategy runs.
func enableExtraReposForTool(binary, pm, id, idLike string, runner commandRunner) {
	if binary == "x11vnc" {
		enableExtraRepos(pm, id, idLike, runner)
		return
	}
	if binary != "zeek" {
		return
	}
	// Repository URLs follow the project's openSUSE Build Service convention.
	// They are best-effort: a distro package may already provide Zeek, and the
	// subsequent install reports the actionable failure if neither does.
	switch pm {
	case "apt-get":
		_ = runner("sh", "-c", "curl -fsSL https://download.opensuse.org/repositories/security:zeek/Debian_12/Release.key | gpg --dearmor -o /usr/share/keyrings/security_zeek.gpg && echo 'deb [signed-by=/usr/share/keyrings/security_zeek.gpg] https://download.opensuse.org/repositories/security:/zeek/Debian_12/ /' > /etc/apt/sources.list.d/security:zeek.list")
		_ = runner("apt-get", "update")
	case "dnf", "microdnf", "yum":
		_ = runner(pm, "config-manager", "--add-repo", "https://download.opensuse.org/repositories/security:/zeek/Fedora_39/security:zeek.repo")
	case "zypper":
		_ = runner("zypper", "--non-interactive", "ar", "https://download.opensuse.org/repositories/security:/zeek/openSUSE_Tumbleweed/security:zeek.repo", "security-zeek")
	}
}

// manualInstallCommand returns a human-runnable command to install the given
// packages, always shown with `sudo` for clarity.
func manualInstallCommand(pm string, pkgs []string) string {
	joined := strings.Join(pkgs, " ")
	switch pm {
	case "apt-get":
		return "sudo apt-get update && sudo apt-get install -y " + joined
	case "dnf", "microdnf":
		return "sudo " + pm + " install -y " + joined
	case "yum":
		return "sudo yum install -y " + joined
	case "pacman":
		return "sudo pacman -S --noconfirm " + joined
	case "zypper":
		return "sudo zypper install -y " + joined
	case "apk":
		return "sudo apk add --no-cache " + joined
	}
	return ""
}

// installPackages installs the given packages using the detected manager.
func installPackages(pm string, pkgs []string, runner commandRunner) error {
	switch pm {
	case "apt-get":
		if err := runner("apt-get", "update"); err != nil {
			return err
		}
		return runner("apt-get", append([]string{"install", "-y"}, pkgs...)...)
	case "dnf", "microdnf", "yum":
		return runner(pm, append([]string{"install", "-y"}, pkgs...)...)
	case "pacman":
		return runner("pacman", append([]string{"-S", "--noconfirm"}, pkgs...)...)
	case "zypper":
		return runner("zypper", append([]string{"install", "-y"}, pkgs...)...)
	case "apk":
		return runner("apk", append([]string{"add", "--no-cache"}, pkgs...)...)
	}
	return fmt.Errorf("unsupported package manager: %s", pm)
}

// EnsureSandboxDependencies prepares only Bubble Wrap's runtime foundation.
// Optional analysis tools are acquired by their own tool route when the user
// selects them; installing every optional tool merely by opening Bubble Wrap
// makes sandbox launch unpredictable and slow.
//
// The runner is injectable (realCommandRunner in production, a stub in tests).
func EnsureSandboxDependencies(runner commandRunner) ([]DependencyStatus, error) {
	if runtime.GOOS != "linux" {
		return nil, fmt.Errorf("sandbox dependencies are only supported on Linux (current OS: %s)", runtime.GOOS)
	}

	binaries := []string{"bwrap", "Xvfb", "x11vnc"}
	statuses := make([]DependencyStatus, 0, len(binaries))
	for _, binary := range binaries {
		statuses = append(statuses, EnsureToolDependency(binary, runner))
	}
	return statuses, nil
}

// binaryExists reports whether name is available, preferring a copy bundled in
// the engine's tools/ directory before falling back to PATH.
func binaryExists(name string) bool {
	if resolved := resolveSandboxTool(name); resolved != name {
		// resolveSandboxTool only returns a path when the file exists and is
		// executable, so a bundled copy is present.
		return true
	}
	_, err := exec.LookPath(name)
	return err == nil
}

// GetDependencyStatus returns the cached dependency check, running it once.
func (m *SandboxManager) GetDependencyStatus() ([]DependencyStatus, error) {
	m.depsOnce.Do(func() {
		m.depsMutex.Lock()
		defer m.depsMutex.Unlock()
		m.depsStatus, m.depsErr = EnsureSandboxDependencies(realCommandRunner)
	})
	return m.getCachedDeps()
}

func (m *SandboxManager) getCachedDeps() ([]DependencyStatus, error) {
	m.depsMutex.Lock()
	defer m.depsMutex.Unlock()
	return m.depsStatus, m.depsErr
}

// InstallDependencies re-runs the dependency check + install and caches the
// result (used by the API install endpoint and after a failed auto-attempt).
func (m *SandboxManager) InstallDependencies() ([]DependencyStatus, error) {
	m.depsMutex.Lock()
	m.depsStatus, m.depsErr = EnsureSandboxDependencies(realCommandRunner)
	m.depsMutex.Unlock()
	m.depsOnce.Do(func() {}) // mark the once-guard so Get returns the cached result
	return m.getCachedDeps()
}

func (m *SandboxManager) handleGetDeps(w http.ResponseWriter, r *http.Request) {
	statuses, err := m.GetDependencyStatus()
	writeDependencyReport(w, statuses, err)
}

func (m *SandboxManager) handleInstallDeps(w http.ResponseWriter, r *http.Request) {
	statuses, err := m.InstallDependencies()
	writeDependencyReport(w, statuses, err)
}

func writeDependencyReport(w http.ResponseWriter, statuses []DependencyStatus, err error) {
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ok":           false,
			"error":        err.Error(),
			"dependencies": statuses,
		})
		return
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"ok":           true,
		"dependencies": statuses,
	})
}
