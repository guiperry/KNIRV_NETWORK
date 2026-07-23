package transactionchain

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

var (
	instance *TransactionChain
	once     sync.Once
)

// TransactionChain manages the embedded Node.js transaction chain service as
// a subprocess, bound to a Unix domain socket. This is the sole owner of its
// lifecycle — backend_server (in the separate KNIRV_CORP repo) only holds an
// HTTP client dialing the same socket; it does not spawn its own copy and
// nothing exposes a TCP port for it.
type TransactionChain struct {
	ctx        context.Context
	cancelFunc context.CancelFunc
	cmd        *exec.Cmd
	socketPath string
	restartMu  sync.Mutex
	running    bool
}

// Get returns the singleton TransactionChain instance.
func Get() *TransactionChain {
	once.Do(func() {
		instance = &TransactionChain{}
	})
	return instance
}

// resolveNodeBinary locates a usable `node` executable. Processes that run
// under sudo/systemd do not inherit a user's interactive-shell PATH (e.g.
// nvm's PATH export lives in ~/.bashrc, which sudo's env_reset skips), so a
// bare exec.Command("node", ...) silently fails to find node even when it is
// installed. KNIRV_NODE_BINARY_PATH overrides everything; otherwise this
// checks PATH, common system install locations, then falls back to
// scanning nvm installs across all home directories for the newest version.
func resolveNodeBinary() (string, error) {
	return resolveNodeTool("node", "KNIRV_NODE_BINARY_PATH")
}

// resolveNpmBinary locates npm to pair with the already-resolved nodeBin.
// It deliberately checks nodeBin's own directory before falling back to
// resolveNodeTool's generic PATH/system search: a bare PATH lookup can
// resolve node and npm from two different installs (e.g. an nvm-installed
// node alongside Debian's system-packaged npm), and a mismatched system npm
// can be missing dependencies (like semver) that only ship with its own
// paired node version. bin/npm sitting next to bin/node holds for nvm,
// nodejs.org tarballs, and most system packages alike.
func resolveNpmBinary(nodeBin string) (string, error) {
	if override := strings.TrimSpace(os.Getenv("KNIRV_NPM_BINARY_PATH")); override != "" {
		if info, err := os.Stat(override); err == nil && !info.IsDir() {
			return override, nil
		}
	}
	if nodeBin != "" {
		colocated := filepath.Join(filepath.Dir(nodeBin), "npm")
		if info, err := os.Stat(colocated); err == nil && !info.IsDir() {
			return colocated, nil
		}
	}
	return resolveNodeTool("npm", "KNIRV_NPM_BINARY_PATH")
}

func resolveNodeTool(name, overrideEnv string) (string, error) {
	if override := strings.TrimSpace(os.Getenv(overrideEnv)); override != "" {
		if info, err := os.Stat(override); err == nil && !info.IsDir() {
			return override, nil
		}
	}
	if path, err := exec.LookPath(name); err == nil {
		return path, nil
	}
	candidates := []string{
		filepath.Join("/usr/local/bin", name),
		filepath.Join("/usr/bin", name),
		filepath.Join("/opt/node/bin", name),
	}
	for _, path := range candidates {
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			return path, nil
		}
	}
	if nvmPath := latestNvmBinary(name); nvmPath != "" {
		return nvmPath, nil
	}
	return "", fmt.Errorf("%s not found: not on PATH, not in common install locations, no nvm install found (set %s to override)", name, overrideEnv)
}

// latestNvmBinary scans nvm-style installs (~/.nvm/versions/node/<version>/bin/<name>)
// across every home directory it can find, returning the path from the
// highest-numbered version directory. Root-run processes have their own
// (usually empty) $HOME, so a plain os.UserHomeDir() lookup is not enough —
// nvm is almost always installed under the operator's own home directory.
func latestNvmBinary(name string) string {
	var homeDirs []string
	if home, err := os.UserHomeDir(); err == nil {
		homeDirs = append(homeDirs, home)
	}
	if entries, err := os.ReadDir("/home"); err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				homeDirs = append(homeDirs, filepath.Join("/home", entry.Name()))
			}
		}
	}

	var versions []string
	versionToPath := map[string]string{}
	for _, home := range homeDirs {
		matches, err := filepath.Glob(filepath.Join(home, ".nvm", "versions", "node", "v*", "bin", name))
		if err != nil {
			continue
		}
		for _, match := range matches {
			if info, err := os.Stat(match); err != nil || info.IsDir() {
				continue
			}
			version := filepath.Base(filepath.Dir(filepath.Dir(match)))
			versions = append(versions, version)
			versionToPath[version] = match
		}
	}
	if len(versions) == 0 {
		return ""
	}
	sort.Slice(versions, func(i, j int) bool { return compareNvmVersions(versions[i], versions[j]) < 0 })
	return versionToPath[versions[len(versions)-1]]
}

// compareNvmVersions compares "vMAJOR.MINOR.PATCH" strings numerically.
func compareNvmVersions(a, b string) int {
	pa := strings.Split(strings.TrimPrefix(a, "v"), ".")
	pb := strings.Split(strings.TrimPrefix(b, "v"), ".")
	for i := 0; i < len(pa) && i < len(pb); i++ {
		na, _ := strconv.Atoi(pa[i])
		nb, _ := strconv.Atoi(pb[i])
		if na != nb {
			return na - nb
		}
	}
	return len(pa) - len(pb)
}

// nodeToolEnv builds the environment for spawning node/npm, prepending
// nodeBin's own directory to PATH. Resolving npm's absolute path is not
// enough on its own: npm is itself a Node script invoked via a
// "#!/usr/bin/env node" shebang, so without node's directory on PATH, npm
// fails immediately with exit 127 ("node: No such file or directory") even
// though we execed npm by its full path. Any child process npm or main.js
// spawns (npm install's own node invocations, native module builds, etc.)
// needs this too.
func nodeToolEnv(nodeBin string) []string {
	nodeDir := filepath.Dir(nodeBin)
	env := os.Environ()
	pathSet := false
	for i, kv := range env {
		if rest, ok := strings.CutPrefix(kv, "PATH="); ok {
			env[i] = "PATH=" + nodeDir + string(os.PathListSeparator) + rest
			pathSet = true
			break
		}
	}
	if !pathSet {
		env = append(env, "PATH="+nodeDir)
	}
	return env
}

// Start launches the transaction chain service bound to socketPath,
// extracting the embedded bundle first if needed and running `npm install`
// before the initial start.
func (tc *TransactionChain) Start(ctx context.Context, socketPath string) error {
	tc.restartMu.Lock()
	defer tc.restartMu.Unlock()

	if tc.running {
		return fmt.Errorf("transaction chain already running")
	}

	tc.ctx, tc.cancelFunc = context.WithCancel(ctx)
	tc.socketPath = socketPath

	destDir, err := ExtractEmbeddedApp()
	if err != nil {
		return err
	}

	nodeBin, err := resolveNodeBinary()
	if err != nil {
		return err
	}
	npmBin, err := resolveNpmBinary(nodeBin)
	if err != nil {
		return err
	}

	installCmd := exec.CommandContext(tc.ctx, npmBin, "install", "--production")
	installCmd.Dir = destDir
	installCmd.Env = nodeToolEnv(nodeBin)
	installCmd.Stdout = os.Stdout
	installCmd.Stderr = os.Stderr
	if err := installCmd.Run(); err != nil {
		return fmt.Errorf("npm install failed: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(socketPath), 0o700); err != nil {
		return fmt.Errorf("create socket directory: %w", err)
	}
	_ = os.Remove(socketPath) // clear a stale socket file from a previous run

	tc.cmd = exec.CommandContext(tc.ctx, nodeBin, "main.js")
	tc.cmd.Dir = destDir
	tc.cmd.Env = append(nodeToolEnv(nodeBin), fmt.Sprintf("SOCKET_PATH=%s", socketPath))
	tc.cmd.Stdout = os.Stdout
	tc.cmd.Stderr = os.Stderr
	tc.cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := tc.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start transaction chain: %w", err)
	}

	tc.running = true

	go tc.monitorProcess(destDir, nodeBin)

	return nil
}

func (tc *TransactionChain) monitorProcess(destDir, nodeBin string) {
	for {
		_ = tc.cmd.Wait()
		tc.restartMu.Lock()

		if !tc.running {
			tc.restartMu.Unlock()
			return
		}

		select {
		case <-tc.ctx.Done():
			tc.running = false
			tc.restartMu.Unlock()
			return
		default:
		}

		time.Sleep(2 * time.Second)

		_ = os.Remove(tc.socketPath)
		tc.cmd = exec.CommandContext(tc.ctx, nodeBin, "main.js")
		tc.cmd.Dir = destDir
		tc.cmd.Env = append(nodeToolEnv(nodeBin), fmt.Sprintf("SOCKET_PATH=%s", tc.socketPath))
		tc.cmd.Stdout = os.Stdout
		tc.cmd.Stderr = os.Stderr
		tc.cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

		if err := tc.cmd.Start(); err != nil {
			tc.restartMu.Unlock()
			continue
		}

		tc.restartMu.Unlock()
	}
}

// Stop gracefully stops the transaction chain service.
func (tc *TransactionChain) Stop() error {
	tc.restartMu.Lock()
	defer tc.restartMu.Unlock()

	if !tc.running {
		return nil
	}

	tc.running = false
	tc.cancelFunc()

	if tc.cmd != nil && tc.cmd.Process != nil {
		syscall.Kill(-tc.cmd.Process.Pid, syscall.SIGTERM)
	}
	_ = os.Remove(tc.socketPath)

	return nil
}

// HealthCheck verifies the transaction chain service is running.
func (tc *TransactionChain) HealthCheck() error {
	tc.restartMu.Lock()
	defer tc.restartMu.Unlock()

	if !tc.running {
		return fmt.Errorf("transaction chain not running")
	}

	if tc.cmd == nil || tc.cmd.Process == nil {
		return fmt.Errorf("transaction chain process not initialized")
	}

	select {
	case <-tc.ctx.Done():
		return fmt.Errorf("transaction chain context cancelled")
	default:
	}

	return nil
}

// SocketPath returns the Unix domain socket path the transaction chain
// listens on, valid once Start has succeeded.
func (tc *TransactionChain) SocketPath() string {
	return tc.socketPath
}
