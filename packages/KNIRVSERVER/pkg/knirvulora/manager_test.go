package knirvulora

import (
	"bytes"
	"compress/gzip"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

// A stand-in ulorad: serves ULORA_SOCKET_PATH, and spawns a child of its own so the
// test can check that Stop reaches the whole process group.
const fakeUlorad = `#!/usr/bin/env python3
import os, signal, socket, subprocess, sys
child = subprocess.Popen(["sleep", "300"])
open(os.environ["ULORA_DATA_DIR"] + "/child.pid", "w").write(str(child.pid))
if os.environ.get("FAKE_IGNORE_TERM"):
    signal.signal(signal.SIGTERM, signal.SIG_IGN)
s = socket.socket(socket.AF_UNIX)
s.bind(os.environ["ULORA_SOCKET_PATH"])
s.listen(4)
while True:
    c, _ = s.accept()
    c.close()
`

func newFake(t *testing.T, env map[string]string) (*Manager, string) {
	t.Helper()
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 not available")
	}
	dir, err := os.MkdirTemp("", "ulora")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	bin := filepath.Join(dir, "ulorad-fake")
	if err := os.WriteFile(bin, []byte(fakeUlorad), 0o755); err != nil {
		t.Fatal(err)
	}
	m := NewManager(ManagerConfig{
		BinaryPath:   bin,
		SocketPath:   filepath.Join(dir, "s", "ulora.sock"),
		DataDir:      filepath.Join(dir, "data"),
		AuthToken:    "tok",
		EnvOverrides: env,
		StartTimeout: 10 * time.Second,
		StopTimeout:  1 * time.Second,
	})
	t.Cleanup(func() { _ = m.Stop() })
	return m, dir
}

func childPID(t *testing.T, dir string) int {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, "data", "child.pid"))
	if err != nil {
		t.Fatal(err)
	}
	var pid int
	for _, c := range string(b) {
		pid = pid*10 + int(c-'0')
	}
	return pid
}

func alive(pid int) bool {
	if syscall.Kill(pid, 0) != nil {
		return false
	}
	b, _ := os.ReadFile("/proc/" + itoa(pid) + "/stat")
	for i := len(b) - 1; i >= 0; i-- {
		if b[i] == ')' {
			return i+2 < len(b) && b[i+2] != 'Z'
		}
	}
	return true
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var d []byte
	for ; n > 0; n /= 10 {
		d = append([]byte{byte('0' + n%10)}, d...)
	}
	return string(d)
}

func TestStartStopStopsTheWholeGroup(t *testing.T) {
	m, dir := newFake(t, nil)
	if err := m.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !m.IsRunning() || m.PID() == 0 {
		t.Fatal("not running after Start")
	}
	child, parent := childPID(t, dir), m.PID()
	if err := m.Stop(); err != nil {
		t.Fatal(err)
	}
	time.Sleep(200 * time.Millisecond)
	if alive(parent) || alive(child) {
		t.Fatalf("survived Stop: ulorad alive=%v child alive=%v", alive(parent), alive(child))
	}
	if m.IsRunning() {
		t.Fatal("still reports running")
	}
	if _, err := os.Stat(m.cfg.SocketPath); !os.IsNotExist(err) {
		t.Fatal("socket not removed")
	}
}

func TestStopKillsWhatIgnoresSIGTERM(t *testing.T) {
	m, _ := newFake(t, map[string]string{"FAKE_IGNORE_TERM": "1"})
	if err := m.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	pid := m.PID()
	if err := m.Stop(); err != nil {
		t.Fatal(err)
	}
	time.Sleep(200 * time.Millisecond)
	if alive(pid) {
		t.Fatal("ulorad ignoring SIGTERM survived Stop")
	}
}

func TestStartFailsWhenTheDaemonExits(t *testing.T) {
	m, dir := newFake(t, nil)
	if err := os.WriteFile(m.cfg.BinaryPath, []byte("#!/bin/sh\nexit 3\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	_ = dir
	if err := m.Start(context.Background()); err == nil {
		t.Fatal("Start succeeded for a daemon that exited")
	}
	if m.IsRunning() {
		t.Fatal("reports running")
	}
}

func TestStopWithoutStartIsANoOp(t *testing.T) {
	if err := NewManager(ManagerConfig{}).Stop(); err != nil {
		t.Fatal(err)
	}
}

func TestDecompressAcceptsGzipAndPlain(t *testing.T) {
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	zw.Write([]byte("hello"))
	zw.Close()
	if got, err := decompress(buf.Bytes()); err != nil || string(got) != "hello" {
		t.Fatalf("gzip: %q %v", got, err)
	}
	if got, err := decompress([]byte("plain")); err != nil || string(got) != "plain" {
		t.Fatalf("plain: %q %v", got, err)
	}
}

func TestExtractEmbeddedBinary(t *testing.T) {
	dir := t.TempDir()
	p, err := ExtractEmbeddedBinary(dir)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(p) != BinaryName {
		t.Fatalf("got %s", p)
	}
	if st, err := os.Stat(p); err != nil || st.Mode()&0o111 == 0 {
		t.Fatalf("not an executable file: %v %v", st, err)
	}
}

// The embedded daemon itself: extracted, started with the manager's environment,
// serving its socket, then stopped.
func TestEmbeddedUloradStartsAndStops(t *testing.T) {
	dir, err := os.MkdirTemp("", "ulora")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	bin, err := ExtractEmbeddedBinary(dir)
	if err != nil {
		t.Fatal(err)
	}
	m := NewManager(ManagerConfig{
		BinaryPath: bin,
		SocketPath: filepath.Join(dir, "sockets", "ulora.sock"),
		DataDir:    filepath.Join(dir, "data"),
		AuthToken:  "test-token",
	})
	if err := m.Start(context.Background()); err != nil {
		t.Fatalf("real ulorad did not start: %v", err)
	}
	pid := m.PID()
	if err := m.Stop(); err != nil {
		t.Fatal(err)
	}
	time.Sleep(200 * time.Millisecond)
	if alive(pid) {
		t.Fatal("ulorad survived Stop")
	}
}
