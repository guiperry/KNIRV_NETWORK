package knirvgraph

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// knirvGraphPIDFileName is the pid file the manager keeps under the app data
// directory. It records the one KNIRVGRAPH child this manager started.
const knirvGraphPIDFileName = "knirvgraph.pid"

// linuxCommLen is the kernel's limit on a process name (/proc/<pid>/comm).
const linuxCommLen = 15

// pidFilePath is where the manager records its child's pid for appDataDir.
func pidFilePath(appDataDir string) string {
	return filepath.Join(appDataDir, "run", knirvGraphPIDFileName)
}

// writePIDFile records pid at path, creating the directory if needed.
func writePIDFile(path string, pid int) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(strconv.Itoa(pid)+"\n"), 0o644)
}

// readPIDFile returns the pid recorded at path, or false when there is none or it
// does not hold a positive integer.
func readPIDFile(path string) (int, bool) {
	b, err := os.ReadFile(path)
	if err != nil {
		return 0, false
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(b)))
	if err != nil || pid <= 0 {
		return 0, false
	}
	return pid, true
}

// removePIDFileFor removes path only when it still records pid. Another manager
// may have written its own pid there since, and that file is not ours to remove.
func removePIDFileFor(path string, pid int) {
	if recorded, ok := readPIDFile(path); ok && recorded == pid {
		_ = os.Remove(path)
	}
}

// processIsNamed reports whether pid is a live process whose kernel name is name.
// A zombie is not live. Without /proc the answer is false, so nothing is signalled.
func processIsNamed(pid int, name string) bool {
	if pid <= 0 || name == "" {
		return false
	}
	comm, err := os.ReadFile(fmt.Sprintf("/proc/%d/comm", pid))
	if err != nil {
		return false
	}
	want := name
	if len(want) > linuxCommLen {
		want = want[:linuxCommLen]
	}
	if strings.TrimSpace(string(comm)) != want {
		return false
	}
	stat, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return false
	}
	// The state field follows the last ')' of the comm field in /proc/<pid>/stat.
	end := strings.LastIndex(string(stat), ")")
	if end < 0 {
		return false
	}
	fields := strings.Fields(string(stat)[end+1:])
	if len(fields) == 0 {
		return false
	}
	return fields[0] != "Z" && fields[0] != "X"
}

// waitForProcessExit polls until pid is no longer named name or the deadline passes.
func waitForProcessExit(pid int, name string, deadline time.Duration) bool {
	end := time.Now().Add(deadline)
	for {
		if !processIsNamed(pid, name) {
			return true
		}
		if time.Now().After(end) {
			return false
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// killStaleGraph stops a KNIRVGRAPH left behind by an earlier run. It signals only
// the pid recorded in pidFile, and only while that pid is still a process named
// binaryName. A pid that has been reused, a dev instance, or another owner's child
// is never signalled. The pid file is removed either way.
func killStaleGraph(pidFile, binaryName string) {
	if pidFile == "" || binaryName == "" {
		return
	}
	pid, ok := readPIDFile(pidFile)
	if !ok {
		_ = os.Remove(pidFile)
		return
	}
	if processIsNamed(pid, binaryName) {
		_ = syscall.Kill(pid, syscall.SIGTERM)
		if !waitForProcessExit(pid, binaryName, time.Second) {
			_ = syscall.Kill(pid, syscall.SIGKILL)
			waitForProcessExit(pid, binaryName, 2*time.Second)
		}
	}
	_ = os.Remove(pidFile)
}
