package device

// CGMinerHashMethod implements the jitter.HashMethod interface by routing the
// FINAL golden-nonce search through the live CGMiner ASIC while using pure-Go
// SHA-256 for the 21 per-pass intermediate hashes (avoiding 21 network round-trips).
//
// Architecture:
//   - Per-pass hashing uses crypto/sha256 (fast, no latency).
//   - After the 21-pass loop, the caller can call MineForGoldenNonce() to submit
//     the refined header to the ASIC.  The ASIC finds a nonce at 500 GH/s;
//     that nonce IS the semantic key (golden nonce) — the token address.
//   - IsAvailable() returns true only when CGMiner responds on the API port.

import (
	"crypto/sha256"
	"fmt"
	"net"
	"time"
)

// CGMinerHashMethod wraps a CGMinerClient and satisfies the jitter.HashMethod
// interface for per-pass hashing, plus exposes MineForGoldenNonce for ASIC
// mining of the final 80-byte header.
type CGMinerHashMethod struct {
	client     *CGMinerClient
	stratumSrv *StratumServer
	timeout    time.Duration
}

// NewCGMinerHashMethod creates a hash method backed by a CGMiner instance.
// host and port identify the CGMiner API endpoint (e.g. "192.168.12.151", 4028).
//
// It also registers a real local Stratum V1 pool with that CGMiner instance
// (see stratum_server.go) so MineForGoldenNonce submits real work to the
// ASIC and gets back a genuine, hardware-found, independently-verified
// nonce - the same mechanism Device.MineWork/ComputeBatch use - rather than
// falling back to a placeholder derived from CGMiner's accepted-share
// counter. If the pool can't be registered, construction fails rather than
// silently returning a hash method that would use that placeholder.
func NewCGMinerHashMethod(host string, port int, timeout time.Duration) (*CGMinerHashMethod, error) {
	if host == "" {
		host = CGMinerDefaultHost
	}
	if port == 0 {
		port = CGMinerDefaultPort
	}
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	client := NewCGMinerClient(host, port)
	if !client.IsAvailable() {
		return nil, fmt.Errorf("cgminer not reachable at %s:%d", host, port)
	}

	stratumSrv, err := startCGMinerStratumPool(client, host)
	if err != nil {
		return nil, fmt.Errorf("register real stratum pool with cgminer at %s:%d: %w", host, port, err)
	}

	return &CGMinerHashMethod{
		client:     client,
		stratumSrv: stratumSrv,
		timeout:    timeout,
	}, nil
}

// Close releases the local stratum pool server. CGMiner itself is left
// pointed at the now-dead pool URL; it will simply fail to reconnect,
// degrading gracefully rather than crashing.
func (m *CGMinerHashMethod) Close() error {
	if m.stratumSrv == nil {
		return nil
	}
	return m.stratumSrv.Close()
}

// IsAvailable returns true when CGMiner is responding on the API socket.
func (m *CGMinerHashMethod) IsAvailable() bool {
	if m.client == nil {
		return false
	}
	addr := net.JoinHostPort(m.client.host, fmt.Sprintf("%d", m.client.port))
	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// ComputeHash computes a single SHA-256 hash in software.
// Per-pass hashing uses crypto/sha256 to avoid round-trip latency.
func (m *CGMinerHashMethod) ComputeHash(data []byte) ([32]byte, error) {
	return sha256.Sum256(data), nil
}

// ComputeDoubleHash computes double SHA-256 in software.
// The 21 per-pass hashes inside the JitterEngine use this path.
// The ASIC is reserved for the final nonce search via MineForGoldenNonce().
func (m *CGMinerHashMethod) ComputeDoubleHash(data []byte) ([32]byte, error) {
	first := sha256.Sum256(data)
	return sha256.Sum256(first[:]), nil
}

// MineForGoldenNonce submits the 80-byte refined header to the CGMiner ASIC
// via the real local stratum pool (see stratum_server.go) and returns the
// golden nonce actually found by the hardware.
//
// This is the ASIC's core contribution: it hashes the semantically refined header
// at ~500 GH/s to find the nonce that IS the token address in the knowledge base.
//
// Note: real Stratum mining always derives the merkleroot (header bytes
// 36:68) from a pool-side coinbase transaction, so it cannot preserve the
// caller-supplied merkleroot bytes verbatim - version/prevhash/ntime/nbits
// pass through unchanged, but the returned nonce and hash are real and
// independently verified against the header KNIRVHASHER actually mined
// (which the caller does not otherwise observe), not against the exact
// original 80 bytes. Returns (nonce, hash, error).
func (m *CGMinerHashMethod) MineForGoldenNonce(header []byte) (uint32, [32]byte, error) {
	if len(header) != 80 {
		return 0, [32]byte{}, fmt.Errorf("header must be 80 bytes, got %d", len(header))
	}

	if !m.IsAvailable() {
		return 0, [32]byte{}, fmt.Errorf("cgminer not available")
	}

	if m.stratumSrv == nil {
		return 0, [32]byte{}, fmt.Errorf("no real work path to cgminer: local stratum pool failed to register at startup")
	}

	job := m.stratumSrv.NewJobFromHeader(header)
	if err := m.stratumSrv.SubmitJob(job); err != nil {
		return 0, [32]byte{}, fmt.Errorf("failed to publish stratum work: %w", err)
	}

	result, err := m.stratumSrv.WaitForResult(job, m.timeout)
	if err != nil {
		return 0, [32]byte{}, fmt.Errorf("cgminer real-hardware mining failed: %w", err)
	}

	return result.nonce, result.hash, nil
}

// GetStats fetches live statistics from CGMiner.
func (m *CGMinerHashMethod) GetStats() (map[string]interface{}, error) {
	return m.client.GetStatus()
}

// GetDevices fetches device info from CGMiner.
func (m *CGMinerHashMethod) GetDevices() ([]map[string]interface{}, error) {
	return m.client.GetDevices()
}
