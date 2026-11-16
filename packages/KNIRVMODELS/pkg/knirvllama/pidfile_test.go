package knirvllama

import (
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"
	"time"
)

// startDummy starts a harmless `sleep` that this test owns. Its argv[0] is set to
// binPath, so ownedProcess sees it as a KNIRVLLAMA started from binPath. The returned
// channel closes once the child has been reaped.
func startDummy(t *testing.T, binPath string) (*exec.Cmd, <-chan struct{}) {
	t.Helper()
	if runtime.GOOS != "linux" {
		t.Skip("pid-file ownership uses /proc/<pid>/cmdline (Linux only)")
	}
	cmd := exec.Command("sleep", "30")
	cmd.Args[0] = binPath
	if err := cmd.Start(); err != nil {
		t.Fatalf("start dummy child: %v", err)
	}
	done := make(chan struct{})
	go func() { _ = cmd.Wait(); close(done) }()
	t.Cleanup(func() {
		// Only the dummy this test started; ignore errors if it already exited.
		_ = cmd.Process.Kill()
		<-done
	})
	return cmd, done
}

func waitGone(t *testing.T, done <-chan struct{}, within time.Duration) bool {
	t.Helper()
	select {
	case <-done:
		return true
	case <-time.After(within):
		return false
	}
}

func TestPIDFileRoundTripAndMode(t *testing.T) {
	path := filepath.Join(t.TempDir(), pidFileName)
	if err := writePIDFile(path, 4242); err != nil {
		t.Fatalf("writePIDFile: %v", err)
	}
	pid, err := readPIDFile(path)
	if err != nil || pid != 4242 {
		t.Fatalf("readPIDFile = %d, %v; want 4242", pid, err)
	}
	if runtime.GOOS == "linux" {
		st, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if mode := st.Mode().Perm(); mode != 0600 {
			t.Fatalf("pid file mode = %o, want 0600", mode)
		}
	}
	if _, err := os.Stat(path + ".tmp"); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("temp file left behind: %v", err)
	}
}

func TestPIDFilePathUsesDataDir(t *testing.T) {
	got := pidFilePath("/data/knirvllama")
	if want := filepath.Join("/data/knirvllama", "knirvllama.pid"); got != want {
		t.Fatalf("pidFilePath = %q, want %q", got, want)
	}
}

func TestReadPIDFileRejectsGarbageAndNonPositive(t *testing.T) {
	dir := t.TempDir()
	for _, content := range []string{"", "not-a-pid", "0", "1", "-7"} {
		path := filepath.Join(dir, "p.pid")
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := readPIDFile(path); err == nil {
			t.Errorf("readPIDFile(%q) accepted invalid content", content)
		}
	}
}

func TestRemovePIDFileOnlyWhenPIDMatches(t *testing.T) {
	path := filepath.Join(t.TempDir(), pidFileName)
	if err := writePIDFile(path, 500); err != nil {
		t.Fatal(err)
	}
	removePIDFile(path, 501) // a different start's record: must stay
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("pid file for another pid was removed: %v", err)
	}
	removePIDFile(path, 500)
	if _, err := os.Stat(path); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("matching pid file not removed: %v", err)
	}
}

func TestOwnedProcessRejectsUnsafePIDs(t *testing.T) {
	for _, pid := range []int{0, 1, -1} {
		if ownedProcess(pid, "/bin/anything") {
			t.Errorf("ownedProcess(%d) = true; must never target init, self or invalid pids", pid)
		}
	}
	if ownedProcess(os.Getpid(), "") {
		t.Error("ownedProcess with empty binary path must be false")
	}
}

func TestStopStaleChildStopsOwnedChildAndRemovesFile(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "knirvllama-dummy")
	cmd, done := startDummy(t, bin)
	pidPath := pidFilePath(t.TempDir())
	if err := writePIDFile(pidPath, cmd.Process.Pid); err != nil {
		t.Fatal(err)
	}

	stopStaleChild(pidPath, bin)

	if !waitGone(t, done, 5*time.Second) {
		t.Fatal("owned stale child was not stopped")
	}
	if _, err := os.Stat(pidPath); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("pid file not removed after stop: %v", err)
	}
}

func TestStopStaleChildLeavesProcessWithDifferentArgv0Alone(t *testing.T) {
	ownBin := filepath.Join(t.TempDir(), "knirvllama-dummy")
	cmd, done := startDummy(t, ownBin)
	pidPath := pidFilePath(t.TempDir())
	if err := writePIDFile(pidPath, cmd.Process.Pid); err != nil {
		t.Fatal(err)
	}

	// Same pid, but the manager now executes a different binary path: the recorded
	// pid is not our process, so it must not be signalled.
	stopStaleChild(pidPath, filepath.Join(t.TempDir(), "other-binary"))

	if waitGone(t, done, 300*time.Millisecond) {
		t.Fatal("process with a different argv[0] was signalled")
	}
	if err := syscall.Kill(cmd.Process.Pid, 0); err != nil {
		t.Fatalf("dummy child should still be alive: %v", err)
	}
	if _, err := os.Stat(pidPath); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("stale pid file should be removed: %v", err)
	}
}

func TestStopStaleChildMissingFileIsNoOp(t *testing.T) {
	pidPath := pidFilePath(t.TempDir())
	stopStaleChild(pidPath, "/bin/anything") // must not panic or create anything
	if _, err := os.Stat(pidPath); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("missing pid file should stay missing: %v", err)
	}
}

func TestStopStaleChildGarbageFileRemoved(t *testing.T) {
	pidPath := pidFilePath(t.TempDir())
	if err := os.WriteFile(pidPath, []byte("junk"), 0600); err != nil {
		t.Fatal(err)
	}
	stopStaleChild(pidPath, "/bin/anything")
	if _, err := os.Stat(pidPath); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("garbage pid file should be removed: %v", err)
	}
}

func TestStopStaleChildDeadPIDFileRemoved(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "knirvllama-dummy")
	cmd, done := startDummy(t, bin)
	pid := cmd.Process.Pid
	_ = cmd.Process.Kill()
	if !waitGone(t, done, 5*time.Second) {
		t.Fatal("dummy did not exit")
	}
	pidPath := pidFilePath(t.TempDir())
	if err := writePIDFile(pidPath, pid); err != nil {
		t.Fatal(err)
	}
	stopStaleChild(pidPath, bin) // dead pid: nothing to signal, file still cleaned up
	if _, err := os.Stat(pidPath); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("pid file for a dead process should be removed: %v", err)
	}
}
