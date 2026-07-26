package p2pconsensus

import (
	"bytes"
	"context"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// fakeHandler records calls and returns canned data.
type fakeHandler struct {
	ops     int
	syncs   int
	lastOp  OperationEnvelope
	lastReq SyncRequest
}

func (f *fakeHandler) HandleOperation(op OperationEnvelope) error {
	f.ops++
	f.lastOp = op
	return nil
}

func (f *fakeHandler) HandleSyncRequest(req SyncRequest) (*SyncResponse, error) {
	f.syncs++
	f.lastReq = req
	return &SyncResponse{NetworkID: req.NetworkID, Collection: req.Collection}, nil
}

func waitForSocket(t *testing.T, path string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); err == nil {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("socket %s never appeared", path)
}

func TestSocketServerRejectsUnsignedWhenSecretSet(t *testing.T) {
	dir := t.TempDir()
	sock := filepath.Join(dir, "knirv.sock")
	secret := "shared-secret"
	h := &fakeHandler{}
	ss := NewSocketServer(sock, h, "net", secret)

	go func() { _ = ss.Serve() }()
	defer ss.Stop(context.Background())
	waitForSocket(t, sock)

	client := UnixSocketDialer(sock)

	// Unsigned request must be rejected with 401.
	req, _ := http.NewRequest(http.MethodPost, "http://unix/internal/p2p/received-op",
		strings.NewReader(`{"collection":"c","document_id":"d"}`))
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 for unsigned request, got %d", resp.StatusCode)
	}
	if h.ops != 0 {
		t.Fatalf("handler must not be called for unauthorized request, got %d", h.ops)
	}

	// Signed request must be accepted.
	body := []byte(`{"collection":"c","document_id":"d","data":"ewoJCSJ4IjoxCgkJfQ=="}`)
	sig := SignMessage("net", secret, body)
	sreq, _ := http.NewRequest(http.MethodPost, "http://unix/internal/p2p/received-op", bytes.NewReader(body))
	sreq.Header.Set("X-KNIRV-Signature", sig)
	sresp, err := client.Do(sreq)
	if err != nil {
		t.Fatalf("signed request: %v", err)
	}
	sresp.Body.Close()
	if sresp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for signed request, got %d", sresp.StatusCode)
	}
	if h.ops != 1 {
		t.Fatalf("handler should be called once, got %d", h.ops)
	}
}

func TestSocketServerOpenNetworkAcceptsUnsigned(t *testing.T) {
	dir := t.TempDir()
	sock := filepath.Join(dir, "knirv.sock")
	h := &fakeHandler{}
	ss := NewSocketServer(sock, h, "net", "") // no secret

	go func() { _ = ss.Serve() }()
	defer ss.Stop(context.Background())
	waitForSocket(t, sock)

	client := UnixSocketDialer(sock)
	body := []byte(`{"collection":"c","document_id":"d"}`)
	req, _ := http.NewRequest(http.MethodPost, "http://unix/internal/p2p/received-op", bytes.NewReader(body))
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("open network should accept unsigned, got %d", resp.StatusCode)
	}
	if h.ops != 1 {
		t.Fatalf("handler should be called, got %d", h.ops)
	}
}

func TestSocketServerPermissionsAreRestrictive(t *testing.T) {
	dir := t.TempDir()
	sock := filepath.Join(dir, "knirv.sock")
	ss := NewSocketServer(sock, &fakeHandler{}, "net", "")
	go func() { _ = ss.Serve() }()
	defer ss.Stop(context.Background())
	waitForSocket(t, sock)

	info, err := os.Stat(sock)
	if err != nil {
		t.Fatalf("stat socket: %v", err)
	}
	// Must be a socket, owned by the user only (0600), never world-writable.
	if info.Mode()&os.ModeSocket == 0 {
		t.Fatalf("expected socket file, got mode %v", info.Mode())
	}
	if info.Mode().Perm()&0077 != 0 {
		t.Fatalf("socket must not be group/other accessible, got mode %v", info.Mode().Perm())
	}
}

func TestSocketServerCleansUpStaleSocket(t *testing.T) {
	dir := t.TempDir()
	sock := filepath.Join(dir, "knirv.sock")
	// Pre-create a stale socket file (simulating a leftover from a crash).
	// Use a bound-but-closed listener and recreate a plain file, since closing a
	// Unix listener removes its socket file.
	if c, err := net.Listen("unix", sock); err != nil {
		t.Fatalf("pre-create socket: %v", err)
	} else {
		c.Close()
	}
	// Recreate a leftover file at the same path.
	if err := os.WriteFile(sock, []byte("stale"), 0600); err != nil {
		t.Fatalf("write stale file: %v", err)
	}
	if _, err := os.Stat(sock); err != nil {
		t.Fatalf("stale socket should exist: %v", err)
	}

	ss := NewSocketServer(sock, &fakeHandler{}, "net", "")
	go func() { _ = ss.Serve() }()
	defer ss.Stop(context.Background())
	waitForSocket(t, sock) // server should have removed and re-created it
}

func TestSocketServerStopsAndRemovesSocket(t *testing.T) {
	dir := t.TempDir()
	sock := filepath.Join(dir, "knirv.sock")
	ss := NewSocketServer(sock, &fakeHandler{}, "net", "")
	go func() { _ = ss.Serve() }()
	waitForSocket(t, sock)
	if err := ss.Stop(context.Background()); err != nil {
		t.Fatalf("stop: %v", err)
	}
	// Allow the cleanup goroutine a moment to remove the file.
	time.Sleep(50 * time.Millisecond)
	if _, err := os.Stat(sock); !os.IsNotExist(err) {
		t.Fatalf("socket file should be removed after Stop, err=%v", err)
	}
}
