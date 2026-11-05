package seedwitness

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	evidence "knirvmodels/pkg/evidence"
	rootwitness "github.com/lab/hasher/data-seeder/pkg/rootwitness"
)

// fakeStore is an in-memory WitnessStore used to exercise the server handlers
// without linking CUDA. It mirrors the real store's PoW guard so the negative
// tests validate the same contract.
type fakeStore struct {
	q          *rootwitness.RootWitnessQueue
	assertions map[string]RootAssertionView
}

func newFakeStore(t *testing.T) *fakeStore {
	t.Helper()
	dir := t.TempDir()
	q := rootwitness.OpenRootWitnessQueue(filepath.Join(dir, "queue.jsonl"))
	return &fakeStore{q: q, assertions: map[string]RootAssertionView{}}
}

func (f *fakeStore) SubmitRoot(root []byte, ref string) string {
	return f.q.Submit(root, ref)
}

func (f *fakeStore) Pending() []rootwitness.PendingRoot {
	return f.q.Pending()
}

func (f *fakeStore) CommitRootAssertion(id string, seed []byte) (string, error) {
	if len(seed) != 32 {
		return "", fmt.Errorf("seed must be 32 bytes")
	}
	for _, p := range f.q.Pending() {
		if p.ID == id {
			if !rootwitness.VerifyRootPoW(p.Root, seed) {
				return "", fmt.Errorf("seed does not satisfy root PoW")
			}
			f.assertions[id] = RootAssertionView{
				ID:   id,
				Kind: "root",
				Root: base64.StdEncoding.EncodeToString(p.Root),
				Ref:  p.Ref,
				Seed: base64.StdEncoding.EncodeToString(seed),
			}
			f.q.Remove(id)
			return id, nil
		}
	}
	return "", fmt.Errorf("root not pending")
}

func (f *fakeStore) RootAssertionView(id string) (RootAssertionView, bool) {
	v, ok := f.assertions[id]
	return v, ok
}

func unixClient(sock string) *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", sock)
			},
		},
	}
}

func startServer(t *testing.T, store WitnessStore) (string, func()) {
	t.Helper()
	sock := filepath.Join(t.TempDir(), "seedwitness.sock")
	s := New(sock, store)
	l, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	go func() { _ = http.Serve(l, s.Handler()) }()
	return sock, func() { _ = l.Close(); _ = os.Remove(sock) }
}

func postJSON(t *testing.T, sock, path string, body interface{}) *http.Response {
	t.Helper()
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	resp, err := unixClient(sock).Post("http://unix"+path, "application/json", bytes.NewReader(data))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	return resp
}

// TestWitnessCommitRejectsForgedSeeds verifies the /witness/commit endpoint rejects
// the audit finding 2 forge attempts over a real Unix socket: wrong seed length,
// a 32-byte seed that is not a real root PoW, and an unknown root id. A valid mined
// seed commits and is then retrievable via /witness/assertion.
func TestWitnessCommitRejectsForgedSeeds(t *testing.T) {
	store := newFakeStore(t)
	root := []byte("batch-root-xyz")
	id := store.SubmitRoot(root, "ref-1")

	sock, stop := startServer(t, store)
	defer stop()

	// 1) Non-32-byte seed -> rejected.
	resp := postJSON(t, sock, "/witness/commit", map[string]string{
		"id":        id,
		"best_seed": base64.StdEncoding.EncodeToString(make([]byte, 31)),
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("31-byte seed: want 400, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// 2) 32-byte seed that is NOT a real PoW -> rejected.
	resp = postJSON(t, sock, "/witness/commit", map[string]string{
		"id":        id,
		"best_seed": base64.StdEncoding.EncodeToString(make([]byte, 32)),
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("non-PoW seed: want 400, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// 3) Unknown root id -> rejected.
	resp = postJSON(t, sock, "/witness/commit", map[string]string{
		"id":        "root-assertion:unknown",
		"best_seed": base64.StdEncoding.EncodeToString(make([]byte, 32)),
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("unknown id: want 400, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// 4) Valid mined seed -> committed and retrievable.
	seed, err := rootwitness.MineRootSeed(root)
	if err != nil {
		t.Fatalf("mine: %v", err)
	}
	resp = postJSON(t, sock, "/witness/commit", map[string]string{
		"id":        id,
		"best_seed": base64.StdEncoding.EncodeToString(seed),
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("valid seed: want 200, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	resp, err = unixClient(sock).Get("http://unix/witness/assertion?id=" + id)
	if err != nil {
		t.Fatalf("assertion fetch: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("assertion fetch: want 200, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

// TestSocketPath pins the pi_3_9 socket-path alignment: the server binds at exactly
// the path pkg/evidence's DefaultWitnessSocketPath dials, so the verifier's Witnesser
// reaches this server end to end — including in the default (KNIRV_APP_DATA_DIR unset)
// configuration, where the server fallback must equal the client's appDataDir() default.
func TestSocketPath(t *testing.T) {
	if got, want := SocketPath("/data/knirv"), filepath.Join("/data/knirv", "sockets", "seed-witness.sock"); got != want {
		t.Fatalf("SocketPath(/data/knirv) = %q, want %q", got, want)
	}
	// Default (no KNIRV_APP_DATA_DIR): the server fallback must match the client's
	// DefaultWitnessSocketPath exactly, so the verifier reaches the seeder end to end.
	t.Setenv("KNIRV_APP_DATA_DIR", "")
	if got, want := SocketPath(""), evidence.DefaultWitnessSocketPath(); got != want {
		t.Fatalf("SocketPath(\"\") = %q, want client default %q", got, want)
	}
}
