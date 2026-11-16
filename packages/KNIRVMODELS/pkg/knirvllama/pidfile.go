package knirvllama

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// pidFileName is the record of the KNIRVLLAMA child this manager started. It lives in
// the manager's DataDir, beside the other KNIRVLLAMA state.
const pidFileName = "knirvllama.pid"

// staleStopWait is how long a stale child gets to exit after SIGTERM before SIGKILL.
const staleStopWait = 3 * time.Second

func pidFilePath(dataDir string) string {
	return filepath.Join(dataDir, pidFileName)
}

// writePIDFile records pid at path. It writes a temp file and renames it into place,
// so a crash mid-write never leaves a half-written pid behind.
func writePIDFile(path string, pid int) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(strconv.Itoa(pid)+"\n"), 0600); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

// readPIDFile returns the pid recorded at path. A missing file is fs.ErrNotExist; any
// other content is an error, and the caller treats the file as stale.
func readPIDFile(path string) (int, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(raw)))
	if err != nil || pid <= 1 {
		return 0, fmt.Errorf("pid file %s holds no valid pid", path)
	}
	return pid, nil
}

// removePIDFile deletes path only when it still records pid. A newer record written by
// another start is left in place.
func removePIDFile(path string, pid int) {
	if got, err := readPIDFile(path); err == nil && got == pid {
		_ = os.Remove(path)
	}
}

// processArgv0 returns argv[0] of pid from /proc/<pid>/cmdline. It returns "" when the
// process is gone, is a zombie (empty cmdline), or /proc is unavailable.
func processArgv0(pid int) string {
	raw, err := os.ReadFile(filepath.Join("/proc", strconv.Itoa(pid), "cmdline"))
	if err != nil || len(raw) == 0 {
		return ""
	}
	argv0, _, _ := strings.Cut(string(raw), "\x00")
	return argv0
}

// ownedProcess reports whether pid is still the KNIRVLLAMA child started from
// binaryPath: its argv[0] is exactly the path the manager executes. A reused pid or an
// unrelated process fails this check and must not be signalled.
func ownedProcess(pid int, binaryPath string) bool {
	if pid <= 1 || binaryPath == "" {
		return false
	}
	return processArgv0(pid) == binaryPath
}

// stopStaleChild handles the pid file left by an earlier manager that died without
// stopping its child. It signals that pid only when the process is still ours (see
// ownedProcess): SIGTERM, a short wait, then SIGKILL if it is still ours. A missing,
// unreadable or stale file is removed or ignored; no other process is ever signalled.
func stopStaleChild(pidPath, binaryPath string) {
	pid, err := readPIDFile(pidPath)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			_ = os.Remove(pidPath)
		}
		return
	}
	if ownedProcess(pid, binaryPath) {
		_ = syscall.Kill(pid, syscall.SIGTERM)
		deadline := time.Now().Add(staleStopWait)
		for ownedProcess(pid, binaryPath) && time.Now().Before(deadline) {
			time.Sleep(100 * time.Millisecond)
		}
		if ownedProcess(pid, binaryPath) {
			_ = syscall.Kill(pid, syscall.SIGKILL)
		}
	}
	_ = os.Remove(pidPath)
}
