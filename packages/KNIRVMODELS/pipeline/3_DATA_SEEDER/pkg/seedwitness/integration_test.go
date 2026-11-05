package seedwitness

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	evidence "knirvmodels/pkg/evidence"
	rootwitness "github.com/lab/hasher/data-seeder/pkg/rootwitness"
)

// startServerAtSocketPath starts the REAL seedwitness.Server on a real Unix socket at
// sock (the caller is responsible for putting a <appDataDir>/sockets/seed-witness.sock
// path here) and waits for it to come up. stop() tears the socket down.
func startServerAtSocketPath(t *testing.T, store WitnessStore, sock string) func() {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(sock), 0o755); err != nil {
		t.Fatalf("mkdir sockets dir: %v", err)
	}
	s := New(sock, store)
	go func() { _ = s.Serve() }()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(sock); err == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if _, err := os.Stat(sock); err != nil {
		t.Fatalf("server socket %s never came up: %v", sock, err)
	}
	return func() { _ = os.Remove(sock) }
}

// startMiner runs the seeder's mining role against the store in the background: for
// every pending root it mines a valid PoW seed and commits it, so the verifier's
// Witness() can observe a seeder-pow attestation. It stops when ctx is cancelled.
func startMiner(ctx context.Context, store WitnessStore) {
	ticker := time.NewTicker(20 * time.Millisecond)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				for _, p := range store.Pending() {
					seed, err := rootwitness.MineRootSeed(p.Root)
					if err != nil {
						continue
					}
					if _, err := store.CommitRootAssertion(p.ID, seed); err != nil {
						continue
					}
				}
			}
		}
	}()
}

// TestSeedWitnessEndToEnd is the end-to-end test that cannot live in KNIRVMODELS
// (importing the seeder would be a circular module dependency: 3_DATA_SEEDER already
// depends on knirvmodels). It starts the REAL seedwitness.Server on a real Unix socket
// at <appDataDir>/sockets/seed-witness.sock — the same path the verifier's
// SeederWitnesser dials via pkg/evidence.DefaultWitnessSocketPath — backed by a
// rootwitness-backed fake store, then drives the REAL evidence.SeederWitnesser.Witness()
// (the client routes/encoding are aligned with the server: pi_3_9 audit findings A/B)
// with a background mining goroutine, expecting a "seeder-pow" attestation whose root
// PoW the client revalidates independently. It also keeps the negative cases: a
// never-mined root stays 404, and a 32-byte non-PoW seed is rejected at /commit.
func TestSeedWitnessEndToEnd(t *testing.T) {
	store := newFakeStore(t)
	root := []byte("batch-root-merkle-9f3c")
	ref := "batch-001"

	appDataDir := t.TempDir()
	sock := SocketPath(appDataDir) // <appDataDir>/sockets/seed-witness.sock
	stop := startServerAtSocketPath(t, store, sock)
	defer stop()

	client := &evidence.SeederWitnesser{SocketPath: sock, PollTimeout: 5 * time.Second}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	startMiner(ctx, store)

	att, err := client.Witness(ctx, evidence.WitnessEntry{Root: hex.EncodeToString(root), BatchID: ref})
	if err != nil {
		t.Fatalf("Witness: %v", err)
	}
	if att.Kind != "seeder-pow" {
		t.Fatalf("Witness returned kind %q, want seeder-pow", att.Kind)
	}
	t.Logf("seeder-pow witness ok: ref=%q", ref)

	// The happy-path root is committed; stop mining so the negatives below stay pending.
	cancel()

	// Negative 1: a root submitted but never mined stays pending -> assertion 404.
	pendingID := store.SubmitRoot([]byte("never-mined-root"), "ref-2")
	resp, err := unixClient(sock).Get("http://unix/witness/assertion?id=" + url.QueryEscape(pendingID))
	if err != nil {
		t.Fatalf("pending assertion fetch: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("pending assertion: want 404, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// Negative 2: committing a 32-byte but non-PoW seed is rejected.
	badID := store.SubmitRoot([]byte("bad-seed-root"), "ref-3")
	resp = postJSON(t, sock, "/witness/commit", map[string]string{
		"id":        badID,
		"best_seed": base64.StdEncoding.EncodeToString(make([]byte, 32)), // 32 bytes, no PoW
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("non-PoW seed: want 400, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}
