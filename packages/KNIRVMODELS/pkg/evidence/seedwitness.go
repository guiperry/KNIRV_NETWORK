package evidence

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// SeederWitnesser witnesses a batch root by asking the 3_DATA_SEEDER daemon to
// mine a PoW assertion committing to it, over a Unix socket (repo rule 20: no
// HTTP URL). The daemon serves SubmitRoot / Pending / CommitRootAssertion
// (pkg/seedwitness). Until the seeder's ASIC/PoW mining of roots is wired (the
// recorded follow-up), the seeder never commits, so Witness returns an error and
// the root stays pending for a later pass.
type SeederWitnesser struct {
	// SocketPath is discovered automatically; set it only in tests.
	SocketPath string
	// HTTP overrides the client; nil selects the default unix-dialing client.
	HTTP *http.Client
	// PollTimeout bounds how long Witness waits for the seeder to commit. Zero
	// selects the default (8s).
	PollTimeout time.Duration
}

// DefaultWitnessSocketPath returns the automatic socket location, shared with the
// seeder daemon: the system app-data directory's sockets dir. KNIRV_APP_DATA_DIR
// is resolved the same way on both ends, so no user-set socket env var is needed.
func DefaultWitnessSocketPath() string {
	return filepath.Join(appDataDir(), "sockets", "seed-witness.sock")
}

func appDataDir() string {
	if d := strings.TrimSpace(os.Getenv("KNIRV_APP_DATA_DIR")); d != "" {
		return d
	}
	return "/var/lib/knirvserver"
}

// Witness submits the root and, once the seeder has committed an assertion for
// it, returns that attestation. It polls for a bounded time so a quickly-mined
// root is witnessed in a single Drain pass; if the seeder has not committed
// within the bound it returns an error and the root remains pending.
func (s *SeederWitnesser) Witness(ctx context.Context, e WitnessEntry) (WitnessAttestation, error) {
	root, err := parseRoot(e.Root)
	if err != nil {
		return WitnessAttestation{}, fmt.Errorf("evidence: witness root not decodable: %w", err)
	}
	id, err := s.submit(ctx, root, e.BatchID)
	if err != nil {
		return WitnessAttestation{}, err
	}

	deadline := s.PollTimeout
	if deadline <= 0 {
		deadline = 8 * time.Second
	}
	if dl, ok := ctx.Deadline(); ok {
		if rem := time.Until(dl); rem < deadline {
			deadline = rem
		}
	}
	pollCtx, cancel := context.WithTimeout(ctx, deadline)
	defer cancel()
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		a, ok, err := s.committed(pollCtx, root, id)
		if err != nil {
			return WitnessAttestation{}, err
		}
		if ok {
			return a, nil
		}
		select {
		case <-pollCtx.Done():
			return WitnessAttestation{}, fmt.Errorf("evidence: seeder has not witnessed root %s yet (pending)", e.Root)
		case <-ticker.C:
		}
	}
}

func (s *SeederWitnesser) client() *http.Client {
	if s.HTTP != nil {
		return s.HTTP
	}
	transport := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "unix", s.socketPath())
		},
	}
	return &http.Client{Transport: transport}
}

func (s *SeederWitnesser) socketPath() string {
	if s.SocketPath != "" {
		return s.SocketPath
	}
	return DefaultWitnessSocketPath()
}

func (s *SeederWitnesser) submit(ctx context.Context, root []byte, ref string) (string, error) {
	// Server contract (3_DATA_SEEDER/pkg/seedwitness/server.go handleSubmit): Root is a
	// JSON []byte (base64 on the wire), Ref a string. Send root as raw bytes so json
	// base64-encodes it the way the server decodes back into []byte.
	body, _ := json.Marshal(map[string]any{"root": root, "ref": ref})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://unix/witness/submit", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	resp, err := s.client().Do(req)
	if err != nil {
		return "", fmt.Errorf("evidence: seeder submit: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("evidence: seeder submit status %d", resp.StatusCode)
	}
	var out struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	return out.ID, nil
}

func (s *SeederWitnesser) committed(ctx context.Context, root []byte, id string) (WitnessAttestation, bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://unix/witness/assertion?id="+url.QueryEscape(id), nil)
	if err != nil {
		return WitnessAttestation{}, false, err
	}
	resp, err := s.client().Do(req)
	if err != nil {
		return WitnessAttestation{}, false, fmt.Errorf("evidence: seeder assertion: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return WitnessAttestation{}, false, nil
	}
	if resp.StatusCode != http.StatusOK {
		return WitnessAttestation{}, false, fmt.Errorf("evidence: seeder assertion status %d", resp.StatusCode)
	}
	var out struct {
		Kind string `json:"kind"`
		Ref  string `json:"ref"`
		Root string `json:"root"`
		Seed string `json:"seed"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return WitnessAttestation{}, false, err
	}
	// Revalidate the attestation independently of the seeder's 200 response so a
	// local caller cannot manufacture a "seeder-pow" witness (pi_3_9 audit,
	// finding 2): the returned root must match what we queued, the seed must be
	// exactly 32 bytes, and the canonical root PoW must meet difficulty.
	// The server returns Root and Seed as base64 (see 3_DATA_SEEDER/pkg/storage/
	// seed_writer.go RootAssertionView: Root = base64(root), Seed = base64(seed)); decode
	// accordingly rather than treating Root as hex.
	gotRoot, err := base64.StdEncoding.DecodeString(out.Root)
	if err != nil {
		return WitnessAttestation{}, false, fmt.Errorf("evidence: seeder assertion root undecodable: %w", err)
	}
	if !bytes.Equal(gotRoot, root) {
		return WitnessAttestation{}, false, fmt.Errorf("evidence: seeder assertion root %s does not match queued root %s", out.Root, hex.EncodeToString(root))
	}
	seed, err := base64.StdEncoding.DecodeString(out.Seed)
	if err != nil {
		return WitnessAttestation{}, false, fmt.Errorf("evidence: seeder assertion seed not base64: %w", err)
	}
	if err := verifyRootPoW(root, seed); err != nil {
		return WitnessAttestation{}, false, fmt.Errorf("evidence: seeder assertion rejected: %w", err)
	}
	detail, _ := json.Marshal(map[string]string{"seed": out.Seed})
	return WitnessAttestation{Kind: "seeder-pow", Ref: out.Ref, Detail: detail}, true, nil
}

// parseRoot decodes a Digest/String() form ("sha256:<hex>") or a bare/0x hex
// string into raw root bytes.
func parseRoot(s string) ([]byte, error) {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "sha256:")
	s = strings.TrimPrefix(s, "0x")
	s = strings.TrimPrefix(s, "0X")
	return hex.DecodeString(s)
}

// SeedWitnessSeed decodes a base64 seed returned by the seeder, used by tests
// and tooling that round-trips an attestation's detail.
func SeedWitnessSeed(b64 string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(b64)
}

// witnessRootPoWBits is the required leading-zero-bit difficulty for a committed
// root PoW seed. It MUST stay in lockstep with the seeder's canonical verifier:
//
//	3_DATA_SEEDER/pkg/storage/seed_writer.go  -> const rootPoWBits = 8
//	3_DATA_SEEDER/pkg/rootwitness/rootwitness.go -> VerifyRootPoW (double-SHA256(root||seed))
//
// The seeder client revalidates a "seeder-pow" attestation with this same rule
// (witnessRootPoWBits, leadingZeroBits, double-SHA256 over root||seed) so a local
// caller cannot forge one (pi_3_9 audit, finding 2). If the seeder's difficulty
// changes, change this constant to the same value and update the comment — they
// are duplicated on purpose (cross-module) and must not drift independently.
const witnessRootPoWBits = 8

// verifyRootPoW revalidates a seeder "seeder-pow" attestation the way the
// seeder's canonical verifier does: the seed must be exactly 32 bytes and the
// double-SHA256 over root||seed must carry at least witnessRootPoWBits leading
// zero bits. It returns an error describing the first failed check.
func verifyRootPoW(root, seed []byte) error {
	if len(seed) != 32 {
		return fmt.Errorf("seed length %d != 32", len(seed))
	}
	first := sha256.Sum256(append(append([]byte(nil), root...), seed...))
	second := sha256.Sum256(first[:])
	if leadingZeroBits(second[:]) < witnessRootPoWBits {
		return fmt.Errorf("root PoW hash has %d leading zero bits, need %d", leadingZeroBits(second[:]), witnessRootPoWBits)
	}
	return nil
}

// leadingZeroBits returns the number of leading zero bits in b. It MUST match the
// seeder's implementation in 3_DATA_SEEDER/pkg/storage/seed_writer.go so the
// client's revalidation agrees with the miner.
func leadingZeroBits(b []byte) int {
	bits := 0
	for _, by := range b {
		if by == 0 {
			bits += 8
			continue
		}
		for i := 7; i >= 0; i-- {
			if by&(1<<uint(i)) != 0 {
				return bits
			}
			bits++
		}
		return bits
	}
	return bits
}
