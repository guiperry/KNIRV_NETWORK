package evidence

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultWitnessSocketPathParity(t *testing.T) {
	// Pin parity with pipeline/3_DATA_SEEDER/pkg/seedwitness.SocketPath (separate
	// module, cannot import). Both must resolve to the same seed-witness socket for
	// the same app-data dir so the verifier reaches the seeder (repo rule 20).
	const wantRel = "sockets/seed-witness.sock"
	const fallback = "/var/lib/knirvserver"

	t.Setenv("KNIRV_APP_DATA_DIR", "/tmp/knirv-appdata")
	if got, want := DefaultWitnessSocketPath(), filepath.Join("/tmp/knirv-appdata", wantRel); got != want {
		t.Fatalf("DefaultWitnessSocketPath() with KNIRV_APP_DATA_DIR set = %q, want %q (must match seedwitness.SocketPath)", got, want)
	}
	t.Setenv("KNIRV_APP_DATA_DIR", "")
	if got, want := DefaultWitnessSocketPath(), filepath.Join(fallback, wantRel); got != want {
		t.Fatalf("DefaultWitnessSocketPath() fallback = %q, want %q (must match seedwitness.SocketPath fallback)", got, want)
	}
}

func TestParseRoot(t *testing.T) {
	raw, _ := hex.DecodeString("deadbeef")
	cases := []string{
		"sha256:" + hex.EncodeToString(raw),
		"0x" + hex.EncodeToString(raw),
		hex.EncodeToString(raw),
	}
	for _, c := range cases {
		got, err := parseRoot(c)
		if err != nil {
			t.Fatalf("parseRoot(%q): %v", c, err)
		}
		if string(got) != string(raw) {
			t.Fatalf("parseRoot(%q)=%x want %x", c, got, raw)
		}
	}
}

// mineSeed finds a 32-byte seed whose root PoW meets the difficulty, using the
// same verifyRootPoW the production client relies on, so the "valid" test always
// exercises an attestation the client will accept.
func mineSeed(t *testing.T, root []byte) []byte {
	t.Helper()
	var seed [32]byte
	for n := uint32(0); n < 1<<22; n++ {
		binary.BigEndian.PutUint32(seed[:4], n)
		if err := verifyRootPoW(root, seed[:]); err == nil {
			return seed[:]
		}
	}
	t.Fatal("mineSeed: no valid seed found within bound")
	return nil
}

// badSeed finds a 32-byte seed whose root PoW definitely fails the difficulty,
// so the "invalid PoW" negative test is deterministic.
func badSeed(t *testing.T, root []byte) []byte {
	t.Helper()
	var seed [32]byte
	for n := uint32(0); n < 1<<22; n++ {
		binary.BigEndian.PutUint32(seed[:4], n)
		if err := verifyRootPoW(root, seed[:]); err != nil {
			return seed[:]
		}
	}
	t.Fatal("badSeed: could not find a failing seed")
	return nil
}

// assertionJSON builds a seeder /witness/assertion response body for a committed
// root. It mirrors the real server contract (3_DATA_SEEDER/pkg/storage/seed_writer.go
// RootAssertionView): Root and Seed are base64.
func assertionJSON(root, seed []byte) string {
	return fmt.Sprintf(`{"id":"abc","kind":"seeder-pow","root":%q,"ref":"rec-1","seed":%q}`,
		base64.StdEncoding.EncodeToString(root), base64.StdEncoding.EncodeToString(seed))
}

// startFakeSeeder serves an in-process fake of the seeder socket protocol (no
// CUDA, no real seeder) over the canonical /witness/* routes the real server mounts
// (pkg/seedwitness/server.go). Every /witness/assertion returns assertionJSON verbatim.
func startFakeSeeder(t *testing.T, assertionJSON string) string {
	t.Helper()
	dir := t.TempDir()
	sock := filepath.Join(dir, "seed-witness.sock")
	ln, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/witness/submit", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"id": "abc"})
	})
	mux.HandleFunc("/witness/assertion", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("id") != "abc" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(assertionJSON))
	})
	srv := &http.Server{Handler: mux}
	go srv.Serve(ln)
	t.Cleanup(func() { srv.Close(); ln.Close() })
	return sock
}

// TestSeederWitnesser exercises the client against a valid PoW attestation.
func TestSeederWitnesser(t *testing.T) {
	root := []byte{1, 2, 3, 4}
	seed := mineSeed(t, root)
	sock := startFakeSeeder(t, assertionJSON(root, seed))
	w := &SeederWitnesser{SocketPath: sock, PollTimeout: 2 * time.Second}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	att, err := w.Witness(ctx, WitnessEntry{BatchID: "b1", Root: "sha256:" + hex.EncodeToString(root)})
	if err != nil {
		t.Fatalf("Witness: %v", err)
	}
	if att.Kind != "seeder-pow" {
		t.Fatalf("kind = %q want seeder-pow", att.Kind)
	}
	if att.Ref != "rec-1" {
		t.Fatalf("ref = %q want rec-1", att.Ref)
	}
}

// TestSeederWitnesserRejectsForgedAttestation ensures the client does NOT trust a
// 200 from the seeder when the returned root/seed/PoW do not revalidate
// (pi_3_9 audit, finding 2).
func TestSeederWitnesserRejectsForgedAttestation(t *testing.T) {
	root := []byte{1, 2, 3, 4}
	goodSeed := mineSeed(t, root)

	cases := []struct {
		name string
		json string
	}{
		{
			name: "wrong root",
			json: assertionJSON([]byte{9, 9, 9, 9}, goodSeed),
		},
		{
			name: "seed not 32 bytes",
			json: fmt.Sprintf(`{"id":"abc","kind":"seeder-pow","root":%q,"ref":"rec-1","seed":%q}`,
				hex.EncodeToString(root), base64.StdEncoding.EncodeToString([]byte("short"))),
		},
		{
			name: "invalid PoW",
			json: fmt.Sprintf(`{"id":"abc","kind":"seeder-pow","root":%q,"ref":"rec-1","seed":%q}`,
				hex.EncodeToString(root), base64.StdEncoding.EncodeToString(badSeed(t, root))),
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			sock := startFakeSeeder(t, c.json)
			w := &SeederWitnesser{SocketPath: sock, PollTimeout: 2 * time.Second}
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			if _, err := w.Witness(ctx, WitnessEntry{BatchID: "b1", Root: "sha256:" + hex.EncodeToString(root)}); err == nil {
				t.Fatalf("Witness accepted forged attestation (%s)", c.name)
			}
		})
	}
}
