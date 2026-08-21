package transformer

import (
	"bufio"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sync"

	"knirvhasher/pkg/hashing/proofasset"
)

// AttestationCandidate is the assertion proposed by the LM. ContextEmbedding
// must be produced by the same embedding model/configuration used by
// 2_DATA_ENCODER; its first four signal dimensions are quantized exactly as
// slots 0-3 in mapper.TensorPacker.
type AttestationCandidate struct {
	ContextTokens    []int32
	AssertionSpan    []int32
	ContextEmbedding []float32
}

// AttestationProof is the portable witness retained by the canonical seeder
// ledger. BestSeed is the mined PoW witness; verification of its difficulty is
// deliberately left to the node's attestation verifier/hardware path.
type AttestationProof struct {
	SchemaVersion    int32      `json:"schema_version"`
	SourceFile       string     `json:"source_file"`
	AssertionKey     string     `json:"assertion_key"`
	CommitmentTarget uint32     `json:"commitment_target"`
	ContextHash      uint32     `json:"context_hash"`
	AsicSlots        [12]uint32 `json:"asic_slots"`
	BestSeed         []byte     `json:"best_seed"`
}

// AttestationResult never blocks generation. A miss is explicit and is queued
// for mining, while a hit carries the proof required for cross-node checking.
type AttestationResult struct {
	Candidate     AttestationCandidate `json:"candidate"`
	Bucket        [4]uint32            `json:"bucket"`
	Hit           bool                 `json:"hit"`
	LowConfidence bool                 `json:"low_confidence"`
	Confidence    float32              `json:"confidence"`
	Proof         *AttestationProof    `json:"proof,omitempty"`
	Queued        bool                 `json:"queued"`
}

// AttestationBridge is the Phase 5 boundary between the LM and the
// append-only span-attestation ledger. It intentionally does not import the
// 3_DATA_SEEDER module: pipeline stages are independent Go modules.
//
// After the formal proof enhancement, the bridge also supports independent
// formal-proof lookup keyed by theorem/proof asset IDs, separate from the
// PoW span-attestation path.
type AttestationBridge struct {
	ledgerPath string
	signalDims [4]int

	mu       sync.RWMutex
	byBucket map[[4]uint32][]AttestationProof
	queued   map[string]struct{}
	pending  chan AttestationCandidate

	// Formal proof lookups
	proofLedgerPath string
	proofLedger     *proofasset.ProofLedger
}

// NewAttestationBridge loads a seed_writes.jsonl ledger. Missing ledgers are
// returned as errors so callers can distinguish an empty new node from a
// malformed configured path; use NewEmptyAttestationBridge for the former.
// proofLedgerPath is the optional path to proof_writes.jsonl for formal proof lookup.
func NewAttestationBridge(framesDir string, signalIndices []int, queueSize int, proofLedgerPath string) (*AttestationBridge, error) {
	b := NewEmptyAttestationBridge(framesDir, signalIndices, queueSize, proofLedgerPath)
	if err := b.Reload(); err != nil {
		return b, err
	}
	return b, nil
}

// NewEmptyAttestationBridge provides the non-blocking miss/queue behavior
// before the first mining run has produced a ledger.
func NewEmptyAttestationBridge(framesDir string, signalIndices []int, queueSize int, proofLedgerPath string) *AttestationBridge {
	if framesDir == "" {
		framesDir = DefaultFramesDir
	}
	if queueSize < 1 {
		queueSize = 128
	}
	dims := [4]int{0, 1, 2, 3}
	for i := 0; i < len(signalIndices) && i < len(dims); i++ {
		dims[i] = signalIndices[i]
	}
	b := &AttestationBridge{
		ledgerPath: filepath.Join(framesDir, seedWritesFile),
		signalDims: dims,
		byBucket:   make(map[[4]uint32][]AttestationProof),
		queued:     make(map[string]struct{}),
		pending:    make(chan AttestationCandidate, queueSize),
	}
	if proofLedgerPath != "" {
		b.proofLedgerPath = proofLedgerPath
		b.proofLedger = proofasset.NewProofLedger(proofLedgerPath)
	}
	return b
}

// Pending returns the receive-only mining queue. Consumers must persist or
// submit each candidate to EvolutionaryHarness; no generation path waits for it.
func (b *AttestationBridge) Pending() <-chan AttestationCandidate { return b.pending }

// Reload atomically rebuilds the bucket index from valid v2 span assertions.
func (b *AttestationBridge) Reload() error {
	f, err := os.Open(b.ledgerPath)
	if err != nil {
		return fmt.Errorf("open attestation ledger: %w", err)
	}
	defer f.Close()

	next := make(map[[4]uint32][]AttestationProof)
	s := bufio.NewScanner(f)
	// A ledger entry is normally tiny, but allow historical rows with large spans.
	s.Buffer(make([]byte, 64*1024), 4*1024*1024)
	for s.Scan() {
		var entry ledgerEntry
		if err := json.Unmarshal(s.Bytes(), &entry); err != nil {
			continue // append-only ledgers may contain interrupted historical rows
		}
		if entry.SchemaVersion < 2 || len(entry.ContextTokens) == 0 || len(entry.AssertionSpan) == 0 {
			continue // v1 token-only entries are not span attestations
		}
		if entry.AssertionKey != assertionIdentity(entry.ContextTokens, entry.AssertionSpan) {
			continue // never trust a ledger identity that doesn't bind its payload
		}
		seed, err := base64.StdEncoding.DecodeString(entry.BestSeed)
		if err != nil || len(seed) == 0 {
			continue
		}
		bucket := [4]uint32{entry.AsicSlots[0], entry.AsicSlots[1], entry.AsicSlots[2], entry.AsicSlots[3]}
		next[bucket] = append(next[bucket], AttestationProof{
			SchemaVersion: entry.SchemaVersion, SourceFile: entry.SourceFile,
			AssertionKey: entry.AssertionKey, CommitmentTarget: entry.CommitmentTarget,
			ContextHash: entry.ContextHash, AsicSlots: entry.AsicSlots, BestSeed: seed,
		})
	}
	if err := s.Err(); err != nil {
		return fmt.Errorf("read attestation ledger: %w", err)
	}
	b.mu.Lock()
	b.byBucket = next
	b.queued = make(map[string]struct{})
	b.mu.Unlock()
	return nil
}

// Ground looks up an LM proposal in its Slot 0-3 LSH bucket and then requires
// exact context/span identity within that bucket. This prevents an LSH
// collision from being reported as a witnessed assertion.
func (b *AttestationBridge) Ground(candidate AttestationCandidate) (AttestationResult, error) {
	bucket, err := b.Bucket(candidate.ContextEmbedding)
	if err != nil {
		return AttestationResult{Candidate: candidate, LowConfidence: true}, err
	}
	result := AttestationResult{Candidate: cloneCandidate(candidate), Bucket: bucket, LowConfidence: true}
	key := assertionIdentity(candidate.ContextTokens, candidate.AssertionSpan)

	b.mu.RLock()
	proofs := append([]AttestationProof(nil), b.byBucket[bucket]...)
	b.mu.RUnlock()
	for _, proof := range proofs {
		if proof.AssertionKey == key {
			proof.BestSeed = append([]byte(nil), proof.BestSeed...)
			result.Hit, result.LowConfidence, result.Confidence, result.Proof = true, false, 1, &proof
			return result, nil
		}
	}

	// The queue is deliberately best-effort and non-blocking: an unavailable
	// miner must lower confidence, never prevent the LM from generating.
	b.mu.Lock()
	if _, exists := b.queued[key]; !exists {
		select {
		case b.pending <- cloneCandidate(candidate):
			b.queued[key] = struct{}{}
			result.Queued = true
		default:
		}
	}
	b.mu.Unlock()
	return result, nil
}

// Bucket applies the exact [-1,1] -> uint32 quantization used by
// 2_DATA_ENCODER's TensorPacker for identity slots 0-3.
func (b *AttestationBridge) Bucket(embedding []float32) ([4]uint32, error) {
	var bucket [4]uint32
	for i, dim := range b.signalDims {
		if dim < 0 || dim >= len(embedding) {
			return bucket, fmt.Errorf("context embedding has %d dimensions; signal dimension %d is unavailable", len(embedding), dim)
		}
		v := embedding[dim]
		if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
			return bucket, fmt.Errorf("context embedding dimension %d is non-finite", dim)
		}
		if v < -1 {
			v = -1
		}
		if v > 1 {
			v = 1
		}
		bucket[i] = uint32(((float64(v)+1)/2)*4294967295.0 + 0.5)
	}
	return bucket, nil
}

func cloneCandidate(c AttestationCandidate) AttestationCandidate {
	c.ContextTokens = append([]int32(nil), c.ContextTokens...)
	c.AssertionSpan = append([]int32(nil), c.AssertionSpan...)
	c.ContextEmbedding = append([]float32(nil), c.ContextEmbedding...)
	return c
}

// assertionIdentity is intentionally byte-for-byte equivalent to
// 3_DATA_SEEDER/storage.assertionKey, retained locally because Go modules are
// independent. Length prefixes make context/span encoding unambiguous.
func assertionIdentity(contextTokens, assertionSpan []int32) string {
	h := sha256.New()
	write := func(tokens []int32) {
		var n [4]byte
		binary.BigEndian.PutUint32(n[:], uint32(len(tokens)))
		h.Write(n[:])
		for _, token := range tokens {
			binary.BigEndian.PutUint32(n[:], uint32(token))
			h.Write(n[:])
		}
	}
	write(contextTokens)
	write(assertionSpan)
	return fmt.Sprintf("assertion-v2:%x", h.Sum(nil))
}

// LookupProofAsset retrieves the most recent proof ledger entry for a given
// proof asset ID. It returns nil if no entry is found. This lookup is
// independent of the PoW span-attestation path.
func (b *AttestationBridge) LookupProofAsset(proofAssetID string) *proofasset.ProofLedgerEntry {
	if b.proofLedger == nil || proofAssetID == "" {
		return nil
	}
	entries, err := b.proofLedger.ReadAll()
	if err != nil {
		return nil
	}
	// Return the most recent entry for the given proof asset ID.
	var latest *proofasset.ProofLedgerEntry
	for i := range entries {
		if entries[i].ProofAssetID == proofAssetID {
			cp := entries[i]
			latest = &cp
		}
	}
	return latest
}

// ProofLedger returns the proof ledger for direct access.
func (b *AttestationBridge) ProofLedger() *proofasset.ProofLedger {
	return b.proofLedger
}
