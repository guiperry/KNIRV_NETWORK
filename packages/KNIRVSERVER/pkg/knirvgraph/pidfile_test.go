package knirvgraph

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"
)

// Fake children named "knirvgraph" so the name check in killStaleGraph sees the
// same comm a real child has. Each is started and reaped by the test; nothing
// here uses pkill or matches a process by name on the host.

type fakeChild struct {
	cmd  *exec.Cmd
	done chan struct{}
}

// startFakeKNIRVGraph runs a shell script named knirvgraph that blocks on stdin.
// Closing its stdin ends it. Cleanup always closes stdin and reaps the child.
func startFakeKNIRVGraph(t *testing.T) *fakeChild {
	t.Helper()
	path := filepath.Join(t.TempDir(), "knirvgraph")
	if err := os.WriteFile(path, []byte("#!/bin/sh\nread _\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(path)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	fc := &fakeChild{cmd: cmd, done: make(chan struct{})}
	go func() { _ = cmd.Wait(); close(fc.done) }()
	t.Cleanup(func() {
		_ = stdin.Close()
		<-fc.done
	})
	return fc
}

// startSleeper runs an unrelated process (name "sleep") that is never a knirvgraph.
func startSleeper(t *testing.T) *fakeChild {
	t.Helper()
	cmd := exec.Command("sleep", "60")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	fc := &fakeChild{cmd: cmd, done: make(chan struct{})}
	go func() { _ = cmd.Wait(); close(fc.done) }()
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		<-fc.done
	})
	return fc
}

func (f *fakeChild) exited(within time.Duration) bool {
	select {
	case <-f.done:
		return true
	case <-time.After(within):
		return false
	}
}

func TestKillStaleGraphSignalsOnlyRecordedPid(t *testing.T) {
	recorded := startFakeKNIRVGraph(t)
	unrecorded := startFakeKNIRVGraph(t)

	pidFile := filepath.Join(t.TempDir(), "run", "knirvgraph.pid")
	if err := writePIDFile(pidFile, recorded.cmd.Process.Pid); err != nil {
		t.Fatal(err)
	}

	killStaleGraph(pidFile, "knirvgraph")

	if !recorded.exited(3 * time.Second) {
		t.Fatal("recorded stale KNIRVGRAPH was not stopped")
	}
	if unrecorded.exited(200 * time.Millisecond) {
		t.Fatal("KNIRVGRAPH not recorded in the pid file was stopped")
	}
	if _, err := os.Stat(pidFile); !os.IsNotExist(err) {
		t.Fatalf("pid file still present after kill: %v", err)
	}
}

// A recorded pid that now belongs to some other process (pid reuse) is not signalled.
func TestKillStaleGraphSkipsReusedPid(t *testing.T) {
	other := startSleeper(t)
	pidFile := filepath.Join(t.TempDir(), "knirvgraph.pid")
	if err := writePIDFile(pidFile, other.cmd.Process.Pid); err != nil {
		t.Fatal(err)
	}

	killStaleGraph(pidFile, "knirvgraph")

	if other.exited(200 * time.Millisecond) {
		t.Fatal("a process not named knirvgraph was signalled from a stale pid file")
	}
	if _, err := os.Stat(pidFile); !os.IsNotExist(err) {
		t.Fatal("stale pid file was not removed")
	}
}

func TestKillStaleGraphWithoutPidFileIsNoop(t *testing.T) {
	unrecorded := startFakeKNIRVGraph(t)
	killStaleGraph(filepath.Join(t.TempDir(), "missing.pid"), "knirvgraph")
	killStaleGraph("", "knirvgraph")
	if unrecorded.exited(200 * time.Millisecond) {
		t.Fatal("a knirvgraph with no pid file record was stopped")
	}
}

// Start records its child in the app data dir and Stop removes the record.
func TestManagerStartRecordsAndStopRemovesPidFile(t *testing.T) {
	appDir := t.TempDir()
	fake := writeLoopKNIRVGraph(t)
	t.Setenv("KNIRV_GRAPH_BINARY_PATH", fake)

	m := NewManager(&ManagerConfig{
		BinaryPath:   fake,
		DataPath:     filepath.Join(appDir, "knirvgraph"),
		AppDataDir:   appDir,
		StopTimeout:  2 * time.Second,
		StartTimeout: time.Second,
	}, zap.NewNop())
	if err := m.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	pidFile := pidFilePath(appDir)
	pid, ok := readPIDFile(pidFile)
	if !ok || pid != m.cmd.Process.Pid {
		t.Fatalf("pid file records %d (ok=%v), want child pid %d", pid, ok, m.cmd.Process.Pid)
	}

	if err := m.Stop(); err != nil {
		t.Fatalf("stop: %v", err)
	}
	if _, err := os.Stat(pidFile); !os.IsNotExist(err) {
		t.Fatal("pid file still present after stop")
	}
}

// A stale KNIRVGRAPH recorded by an earlier run is stopped by the next Start, and
// an unrecorded one is left alone.
func TestManagerStartKillsOnlyRecordedStaleGraph(t *testing.T) {
	appDir := t.TempDir()
	fake := writeLoopKNIRVGraph(t)
	t.Setenv("KNIRV_GRAPH_BINARY_PATH", fake)

	stale := startFakeKNIRVGraph(t)
	unrecorded := startFakeKNIRVGraph(t)
	if err := writePIDFile(pidFilePath(appDir), stale.cmd.Process.Pid); err != nil {
		t.Fatal(err)
	}

	m := NewManager(&ManagerConfig{
		BinaryPath:   fake,
		DataPath:     filepath.Join(appDir, "knirvgraph"),
		AppDataDir:   appDir,
		StopTimeout:  2 * time.Second,
		StartTimeout: time.Second,
	}, zap.NewNop())
	if err := m.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	t.Cleanup(func() { _ = m.Stop() })

	if !stale.exited(3 * time.Second) {
		t.Fatal("start did not stop the recorded stale KNIRVGRAPH")
	}
	if unrecorded.exited(200 * time.Millisecond) {
		t.Fatal("start stopped a KNIRVGRAPH that was not recorded")
	}
}

// writeLoopKNIRVGraph is a long-running stand-in for the manager's child. It is
// named knirvgraph and ignores its arguments.
func writeLoopKNIRVGraph(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "knirvgraph")
	script := "#!/bin/sh\nwhile :; do sleep 0.05; done\n"
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

// The pid file holds only the number, with a trailing newline.
func TestPIDFileRoundTrip(t *testing.T) {
	p := filepath.Join(t.TempDir(), "run", "knirvgraph.pid")
	if err := writePIDFile(p, 4242); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(p)
	if strings.TrimSpace(string(raw)) != strconv.Itoa(4242) {
		t.Fatalf("pid file content %q", raw)
	}
	if got, ok := readPIDFile(p); !ok || got != 4242 {
		t.Fatalf("read = %d, %v", got, ok)
	}
	removePIDFileFor(p, 1) // a different pid must not remove it
	if _, ok := readPIDFile(p); !ok {
		t.Fatal("pid file removed for a pid it does not record")
	}
	removePIDFileFor(p, 4242)
	if _, err := os.Stat(p); !os.IsNotExist(err) {
		t.Fatal("pid file not removed for its own pid")
	}
}
