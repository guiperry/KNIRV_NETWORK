// Package rootwitness holds the root-assertion queue and mined-root PoW logic
// for Task 3.9's provenance follow-up. It is deliberately cgo-free (no CUDA,
// no Arrow, no knirvmodels proof asset) so its tests run in the normal
// non-CUDA CI target. The CUDA-dependent span-seed materialization lives in
// pkg/storage; this package only handles batch Merkle roots and the PoW that
// commits a seed to each root.
package rootwitness

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// PendingRoot is one batch root awaiting a witness (a mined PoW seed committing
// to it) before it becomes a recorded assertion.
type PendingRoot struct {
	ID   string
	Root []byte
	Ref  string
}

// pendingQueueLine is one append-only record of the durable root-witness queue.
// A removed line is a tombstone (idempotent under rewrite); a reload skips ids
// that have a tombstone or were already committed to the ledger.
type pendingQueueLine struct {
	ID      string `json:"id"`
	Root    []byte `json:"root"`
	Ref     string `json:"ref"`
	Removed bool   `json:"removed,omitempty"`
}

// RootWitnessQueue buffers batch roots queued for witnessing so a caller (the
// 3.9 Witnesser) can submit roots out of the request path and commit the mined
// assertion once the seed is produced. The queue is durable: every Submit/Remove
// is appended to an append-only file under the seeder's data dir and reloaded at
// start, so a restart does not lose pending roots (Task 3.9 follow-up).
type RootWitnessQueue struct {
	mu      sync.Mutex
	pending []PendingRoot
	path    string
}

// OpenRootWitnessQueue loads a durable queue from path, reconstructing pending
// roots from any prior run (skipping tombstoned or already-committed ids).
func OpenRootWitnessQueue(path string) *RootWitnessQueue {
	q := &RootWitnessQueue{path: path}
	q.reload()
	return q
}

func (q *RootWitnessQueue) reload() {
	if q.path == "" {
		return
	}
	data, err := os.ReadFile(q.path)
	if err != nil {
		return
	}
	removed := make(map[string]bool)
	var pend []PendingRoot
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var e pendingQueueLine
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			continue
		}
		if e.Removed {
			removed[e.ID] = true
			// A tombstone may arrive after the submit line was already loaded;
			// drop the in-memory pending entry if present so a reload never
			// resurrects a committed/removed root.
			kept := pend[:0]
			for _, p := range pend {
				if p.ID != e.ID {
					kept = append(kept, p)
				}
			}
			pend = kept
			continue
		}
		if removed[e.ID] {
			continue
		}
		pend = append(pend, PendingRoot{ID: e.ID, Root: append([]byte(nil), e.Root...), Ref: e.Ref})
	}
	q.pending = pend
}

func (q *RootWitnessQueue) appendLine(e pendingQueueLine) {
	if q.path == "" {
		return
	}
	if err := os.MkdirAll(filepath.Dir(q.path), 0o755); err != nil {
		return
	}
	f, err := os.OpenFile(q.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	data, err := json.Marshal(e)
	if err != nil {
		return
	}
	_, _ = f.Write(append(data, '\n'))
}

// Submit enqueues a root and returns its assertion id (idempotent: the same root
// maps to the same id and is not queued twice).
func (q *RootWitnessQueue) Submit(root []byte, ref string) string {
	q.mu.Lock()
	defer q.mu.Unlock()
	id := RootAssertionKey(root)
	for _, p := range q.pending {
		if p.ID == id {
			return id
		}
	}
	rootCopy := append([]byte(nil), root...)
	q.pending = append(q.pending, PendingRoot{ID: id, Root: rootCopy, Ref: ref})
	q.appendLine(pendingQueueLine{ID: id, Root: rootCopy, Ref: ref})
	return id
}

// Pending returns a snapshot of the queued roots.
func (q *RootWitnessQueue) Pending() []PendingRoot {
	q.mu.Lock()
	defer q.mu.Unlock()
	out := make([]PendingRoot, len(q.pending))
	copy(out, q.pending)
	return out
}

// Remove drops a committed root from the queue and writes a tombstone so a reload
// will not resurrect it.
func (q *RootWitnessQueue) Remove(id string) {
	q.mu.Lock()
	defer q.mu.Unlock()
	kept := q.pending[:0]
	for _, p := range q.pending {
		if p.ID != id {
			kept = append(kept, p)
		}
	}
	q.pending = kept
	q.appendLine(pendingQueueLine{ID: id, Removed: true})
}

// rootPoWBits is the number of leading zero bits a root PoW seed must produce
// (double-SHA256 over root||seed). Conservative/easy difficulty; the 3.9 plan
// leaves the exact PoW form to the owner, so this is provisional.
const rootPoWBits = 8

// rootPoWMaxIter bounds the search so a hostile or unlucky root can never stall
// the miner; beyond it we report exhaustion rather than returning an invalid
// (non-PoW) fallback seed.
const rootPoWMaxIter = 1 << 22

// RootAssertionKey is the canonical ledger identity for a batch Merkle root. It
// mirrors the storage package's key so both agree on how a root is identified.
func RootAssertionKey(root []byte) string {
	h := sha256.New()
	h.Write([]byte("KNIRV-ROOT-ASSERTION-V2\x00"))
	var length [4]byte
	binary.BigEndian.PutUint32(length[:], uint32(len(root)))
	h.Write(length[:])
	h.Write(root)
	return fmt.Sprintf("root-assertion:%x", h.Sum(nil))
}

// rootPoWHash is the canonical root-PoW hash: double-SHA256 over root||seed.
func rootPoWHash(root, seed []byte) [32]byte {
	first := sha256.Sum256(append(append([]byte(nil), root...), seed...))
	return sha256.Sum256(first[:])
}

// LeadingZeroBits returns the number of leading zero bits in b.
func LeadingZeroBits(b []byte) int {
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

// VerifyRootPoW reports whether seed is a valid 32-byte PoW commitment to root
// under the canonical double-SHA256 rule at the required difficulty. A seed that
// is not exactly 32 bytes or that does not meet rootPoWBits is rejected.
func VerifyRootPoW(root, seed []byte) bool {
	if len(seed) != 32 {
		return false
	}
	h := rootPoWHash(root, seed)
	return LeadingZeroBits(h[:]) >= rootPoWBits
}

// MineRootSeed searches for a 32-byte PoW seed committing to root: a seed whose
// double-SHA256 over root||seed has at least rootPoWBits leading zero bits. It is
// bounded; if no valid seed is found within maxIter it returns an error instead
// of an invalid fallback, so a caller can never record a root assertion with no
// real PoW. The seed layout is deterministic for a given root (nonce in the
// first 4 bytes, sha256(root||nonce) filling the rest), so the same root always
// yields the same valid seed.
func MineRootSeed(root []byte) ([]byte, error) {
	return mineRootSeed(root, rootPoWMaxIter, rootPoWBits)
}

// mineRootSeed is the injectable core of MineRootSeed, used by tests to force
// exhaustion (small maxIter) or impossible difficulty (large bits).
func mineRootSeed(root []byte, maxIter uint32, bits int) ([]byte, error) {
	for nonce := uint32(0); nonce < maxIter; nonce++ {
		seed := make([]byte, 32)
		binary.BigEndian.PutUint32(seed[:4], nonce)
		fill := sha256.Sum256(append(append([]byte(nil), root...),
			byte(nonce), byte(nonce>>8), byte(nonce>>16), byte(nonce>>24)))
		copy(seed[4:], fill[:])
		h := rootPoWHash(root, seed)
		if LeadingZeroBits(h[:]) >= bits {
			return seed, nil
		}
	}
	return nil, fmt.Errorf("root PoW search exhausted after %d iterations without a valid seed", maxIter)
}
