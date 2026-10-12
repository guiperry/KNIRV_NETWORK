package supervision

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// AttestationHeaderVersion is the current version of the attestation header
// commitment format. Incrementing this makes old records unverifiable under
// the new logic, providing forward compatibility.
const AttestationHeaderVersion uint32 = 1

// Nonces are appended to the canonical commitment. Keeping them outside the
// header removes fragile offset arithmetic when variable-length lists evolve.
const attestationHeaderNonceOffset = 0 // retained for source compatibility
const attestationHeaderMaxHexLen = 512

// AttestationRequest binds the provenance of a supervision model recommendation.
// An attestation proves that a specific model artifact was evaluated against a
// specific dataset manifest and policy bundle with specific evidence, at a
// specific point in time, without claiming that the recommendation itself is
// correct or safe.
type AttestationRequest struct {
	ModelArtifactHash      string   `json:"model_artifact_hash"`                // sha256 hex of the model artifact weights
	DatasetManifestHash    string   `json:"dataset_manifest_hash"`              // sha256 hex of the DatasetManifest
	PolicyBundleHash       string   `json:"policy_bundle_hash"`                 // sha256 hex of the active policy bundle
	GoldEpisodeIDs         []string `json:"gold_episode_ids,omitempty"`         // sha256 hex of Gold episodes used as retrieval basis
	EvidenceArtifactHashes []string `json:"evidence_artifact_hashes,omitempty"` // sha256 hex of evidence artifacts cited
	DeviceFirmware         string   `json:"device_firmware,omitempty"`          // opaque firmware identifier for hardware attestation
	NonceStart             uint32   `json:"nonce_start"`
	NonceEnd               uint32   `json:"nonce_end"`
	Target                 uint32   `json:"target"` // difficulty target (leading Uint32 < Target)
}

// AttestationRecord is an optional hardware/software attestation that can be
// attached to a model recommendation in the CLI advisory path. It proves data
// linkage, not factual or policy correctness. Attestation confidence is kept
// separate from semantic confidence: a valid PoW witness proves the
// recommendation came from a particular model trained on a particular manifest
// evaluated against a named policy bundle, nothing more.
type AttestationRecord struct {
	HeaderHex      string    `json:"header_hex"` // hex-encoded canonical commitment
	Nonce          uint32    `json:"nonce"`
	DoubleSHA256   string    `json:"double_sha256"` // double-SHA256 of header+nonce
	AttestedAt     time.Time `json:"attested_at"`
	DeviceFirmware string    `json:"device_firmware,omitempty"`
}

// AttestationResult is the outcome of computing a supervision attestation.
type AttestationResult struct {
	Record     *AttestationRecord
	Attested   bool
	Diagnostic string
}

// Attestor produces and verifies supervision attestations. By default it uses
// software SHA-256 PoW; if the operator provides an ASIC-backed service it can
// produce a hardware witness.
type Attestor struct {
	enabled bool
}

// NewAttestor creates a supervision attestor. If enabled is false, all
// attestation calls return Attested=false with a diagnostic.
func NewAttestor(enabled bool) *Attestor {
	return &Attestor{enabled: enabled}
}

// IsEnabled reports whether the attestor will produce records.
func (a *Attestor) IsEnabled() bool { return a.enabled }

// ComputeAttestation computes a PoW attestation for the given request.
// It returns a record that can be independently verified by a second node.
func (a *Attestor) ComputeAttestation(req AttestationRequest) *AttestationResult {
	if !a.enabled {
		return &AttestationResult{
			Attested:   false,
			Diagnostic: "supervision attestation disabled",
		}
	}

	if err := validateAttestationRequest(req); err != nil {
		return &AttestationResult{
			Attested:   false,
			Diagnostic: fmt.Sprintf("invalid request: %s", err),
		}
	}

	header := buildAttestationHeader(req)
	headerBytes := header.Bytes()

	nonce, doubleSHA256 := mineAttestationHeader(headerBytes, req.Target, req.NonceStart, req.NonceEnd)
	if !lessThanTarget(doubleSHA256, req.Target) {
		return &AttestationResult{Attested: false, Diagnostic: "no qualifying nonce in requested range"}
	}

	record := &AttestationRecord{
		HeaderHex:      fmt.Sprintf("%x", headerBytes),
		Nonce:          nonce,
		DoubleSHA256:   fmt.Sprintf("%x", doubleSHA256),
		AttestedAt:     time.Now().UTC(),
		DeviceFirmware: req.DeviceFirmware,
	}

	return &AttestationResult{
		Record:     record,
		Attested:   true,
		Diagnostic: fmt.Sprintf("attestation mined nonce=%d hash=%x", nonce, doubleSHA256[:8]),
	}
}

// VerifyAttestation independently verifies an attestation record. A second
// node (or the CLI itself) can call this to validate the witness without
// trusting the producing node's report.
func (a *Attestor) VerifyAttestation(record *AttestationRecord, req AttestationRequest) bool {
	if record == nil {
		return false
	}
	if err := validateAttestationRequest(req); err != nil || record.Nonce < req.NonceStart || record.Nonce > req.NonceEnd {
		return false
	}

	headerBytes, err := decodeAttestationHeader(record.HeaderHex)
	if err != nil {
		return false
	}
	if string(headerBytes) != string(buildAttestationHeader(req).Bytes()) {
		return false
	}

	expectedHash := doubleSHA256Attestation(headerBytes, record.Nonce)
	return lessThanTarget(expectedHash, req.Target) && fmt.Sprintf("%x", expectedHash) == record.DoubleSHA256
}

func validateAttestationRequest(req AttestationRequest) error {
	for name, value := range map[string]string{"model_artifact_hash": req.ModelArtifactHash, "dataset_manifest_hash": req.DatasetManifestHash, "policy_bundle_hash": req.PolicyBundleHash} {
		if _, err := decodeHash(value); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
	}
	for _, value := range append(append([]string{}, req.GoldEpisodeIDs...), req.EvidenceArtifactHashes...) {
		if _, err := decodeHash(value); err != nil {
			return fmt.Errorf("reference hash: %w", err)
		}
	}
	if req.Target == 0 {
		return fmt.Errorf("target (difficulty) is required")
	}
	if req.NonceEnd < req.NonceStart {
		return fmt.Errorf("nonce_end must be >= nonce_start")
	}
	return nil
}

func decodeHash(value string) ([32]byte, error) {
	var out [32]byte
	if len(value) != 64 {
		return out, fmt.Errorf("must be a 64-char sha256 hex")
	}
	b, err := hex.DecodeString(value)
	if err != nil {
		return out, fmt.Errorf("invalid hexadecimal hash")
	}
	copy(out[:], b)
	return out, nil
}

// AttestationHeader is the commitment structure for a supervision attestation.
// It records the version and all provenance references that the PoW binds to.
type AttestationHeader struct {
	Version              uint32
	ModelArtifactHash    [32]byte
	DatasetManifestHash  [32]byte
	PolicyBundleHash     [32]byte
	GoldEpisodeCount     uint32
	GoldEpisodeHashes    [][32]byte
	EvidenceCount        uint32
	EvidenceHashes       [][32]byte
	Timestamp            uint64
	Target               uint32
	Nonce                uint32
	Reserved             [16]byte
	DeviceFirmwareLength uint32
	DeviceFirmware       string
}

func (h *AttestationHeader) Bytes() []byte {
	var buf []byte

	ver := make([]byte, 4)
	binary.BigEndian.PutUint32(ver, h.Version)
	buf = append(buf, ver...)

	buf = append(buf, h.ModelArtifactHash[:]...)
	buf = append(buf, h.DatasetManifestHash[:]...)
	buf = append(buf, h.PolicyBundleHash[:]...)

	goldCount := make([]byte, 4)
	binary.BigEndian.PutUint32(goldCount, h.GoldEpisodeCount)
	buf = append(buf, goldCount...)
	for _, gh := range h.GoldEpisodeHashes {
		buf = append(buf, gh[:]...)
	}

	evCount := make([]byte, 4)
	binary.BigEndian.PutUint32(evCount, h.EvidenceCount)
	buf = append(buf, evCount...)
	for _, eh := range h.EvidenceHashes {
		buf = append(buf, eh[:]...)
	}

	ts := make([]byte, 8)
	binary.BigEndian.PutUint64(ts, h.Timestamp)
	buf = append(buf, ts...)

	tgt := make([]byte, 4)
	binary.BigEndian.PutUint32(tgt, h.Target)
	buf = append(buf, tgt...)

	nonce := make([]byte, 4)
	binary.BigEndian.PutUint32(nonce, h.Nonce)
	buf = append(buf, nonce...)

	buf = append(buf, h.Reserved[:]...)

	fwLen := make([]byte, 4)
	binary.BigEndian.PutUint32(fwLen, h.DeviceFirmwareLength)
	buf = append(buf, fwLen...)
	buf = append(buf, []byte(h.DeviceFirmware)...)

	return buf
}

func buildAttestationHeader(req AttestationRequest) *AttestationHeader {
	var modelHash, datasetHash, policyHash [32]byte
	modelHash, _ = decodeHash(req.ModelArtifactHash)
	datasetHash, _ = decodeHash(req.DatasetManifestHash)
	policyHash, _ = decodeHash(req.PolicyBundleHash)

	sortedGold := make([]string, len(req.GoldEpisodeIDs))
	copy(sortedGold, req.GoldEpisodeIDs)
	sort.Strings(sortedGold)

	sortedEvidence := make([]string, len(req.EvidenceArtifactHashes))
	copy(sortedEvidence, req.EvidenceArtifactHashes)
	sort.Strings(sortedEvidence)

	goldHashes := make([][32]byte, len(sortedGold))
	for i, gh := range sortedGold {
		goldHashes[i], _ = decodeHash(gh)
	}

	evidenceHashes := make([][32]byte, len(sortedEvidence))
	for i, eh := range sortedEvidence {
		evidenceHashes[i], _ = decodeHash(eh)
	}

	return &AttestationHeader{
		Version:             AttestationHeaderVersion,
		ModelArtifactHash:   modelHash,
		DatasetManifestHash: datasetHash,
		PolicyBundleHash:    policyHash,
		GoldEpisodeCount:    uint32(len(goldHashes)),
		GoldEpisodeHashes:   goldHashes,
		EvidenceCount:       uint32(len(evidenceHashes)),
		EvidenceHashes:      evidenceHashes,
		// The request is the commitment. Wall-clock receipt time belongs to the
		// record and must not make independently rebuilt commitments differ.
		Timestamp:            0,
		Target:               req.Target,
		Nonce:                req.NonceStart,
		Reserved:             [16]byte{},
		DeviceFirmware:       req.DeviceFirmware,
		DeviceFirmwareLength: uint32(len(req.DeviceFirmware)),
	}
}

func decodeAttestationHeader(hexStr string) ([]byte, error) {
	if len(hexStr)%2 != 0 {
		return nil, fmt.Errorf("invalid hex length: %d", len(hexStr))
	}
	if len(hexStr) > attestationHeaderMaxHexLen {
		return nil, fmt.Errorf("header exceeds max length %d", attestationHeaderMaxHexLen)
	}
	header := make([]byte, len(hexStr)/2)
	for i := 0; i < len(header); i++ {
		b := hexStr[i*2 : i*2+2]
		var v byte
		if _, err := fmt.Sscanf(b, "%02x", &v); err != nil {
			return nil, err
		}
		header[i] = v
	}
	if len(header) < 4 {
		return nil, fmt.Errorf("header too short")
	}
	if binary.BigEndian.Uint32(header[0:4]) != AttestationHeaderVersion {
		return nil, fmt.Errorf("version mismatch: got %d, expected %d",
			binary.BigEndian.Uint32(header[0:4]), AttestationHeaderVersion)
	}
	return header, nil
}

func mineAttestationHeader(headerBytes []byte, target, start, end uint32) (uint32, [32]byte) {
	for nonce := start; ; nonce++ {
		h := doubleSHA256Attestation(headerBytes, nonce)
		if lessThanTarget(h, target) {
			return nonce, h
		}
		if nonce == end {
			break
		}
	}
	return end, [32]byte{}
}

func doubleSHA256Attestation(data []byte, nonce uint32) [32]byte {
	buf := make([]byte, len(data)+4)
	copy(buf, data)
	binary.BigEndian.PutUint32(buf[len(data):], nonce)
	first := sha256.Sum256(buf)
	return sha256.Sum256(first[:])
}

func lessThanTarget(h [32]byte, target uint32) bool {
	return binary.BigEndian.Uint32(h[0:4]) < target
}

// AttestationLedgerEntry is a single append-only record in the attestation ledger.
type AttestationLedgerEntry struct {
	SchemaVersion     uint32             `json:"schema_version"`
	Request           AttestationRequest `json:"request"`
	Record            *AttestationRecord `json:"record,omitempty"`
	Verified          bool               `json:"verified"`
	AttestedAt        time.Time          `json:"attested_at"`
	RecordFingerprint string             `json:"record_fingerprint"`
}

// AttestationLedger manages append-only attestation records with rotation
// and replay protection. Records are stored as JSON lines in a daily file.
// Replay protection prevents the same attestation from being accepted twice.
const attestMaxRecordsPerFile = 10000

type AttestationLedger struct {
	dir        string
	mu         sync.Mutex
	seen       map[string]bool // replay protection: header_hex+nonce
	maxPerFile int
}

// NewAttestationLedger creates a ledger that stores attestation entries in
// the given directory, one JSONL file per rotation window.
func NewAttestationLedger(dir string) *AttestationLedger {
	l := &AttestationLedger{
		dir:        dir,
		seen:       make(map[string]bool),
		maxPerFile: attestMaxRecordsPerFile,
	}
	// Best-effort hydration prevents replay after restart; malformed persisted
	// ledgers are rejected on reads rather than silently skipped.
	if entries, err := l.readAllUnlocked(); err == nil {
		for _, e := range entries {
			if e.RecordFingerprint != "" {
				l.seen[e.RecordFingerprint] = true
			}
		}
	}
	return l
}

// Append adds an entry to the attestation ledger with replay protection.
// It returns an error if the same record has already been accepted.
func (l *AttestationLedger) Append(entry AttestationLedgerEntry) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	entry.SchemaVersion = AttestationHeaderVersion
	entry.RecordFingerprint = attestRecordFingerprint(entry.Record)
	if entry.AttestedAt.IsZero() && entry.Record != nil {
		entry.AttestedAt = entry.Record.AttestedAt
	}

	if entry.Record != nil {
		if l.seen[entry.RecordFingerprint] {
			return fmt.Errorf("replay detected: record already accepted")
		}
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshal attestation entry: %w", err)
	}

	filename := l.currentFilename()
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create attestation ledger directory: %w", err)
	}

	f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("open attestation ledger: %w", err)
	}
	defer f.Close()

	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("append attestation ledger: %w", err)
	}
	if err := f.Sync(); err != nil {
		return fmt.Errorf("sync attestation ledger: %w", err)
	}
	if entry.Record != nil {
		l.seen[entry.RecordFingerprint] = true
	}

	return nil
}

// PruneBefore removes ledger entries whose AttestedAt is before the cutoff.
// This supports retention policies.
func (l *AttestationLedger) PruneBefore(cutoff time.Time) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	entries, err := l.readAllUnlocked()
	if err != nil {
		return err
	}

	kept := make([]AttestationLedgerEntry, 0, len(entries))
	for _, e := range entries {
		if e.AttestedAt.After(cutoff) || e.AttestedAt.Equal(cutoff) {
			kept = append(kept, e)
		}
	}

	return l.rewrite(kept)
}

// ReadAll reads all entries from the current attestation ledger file.
func (l *AttestationLedger) ReadAll() ([]AttestationLedgerEntry, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.readAllUnlocked()
}

func (l *AttestationLedger) readAllUnlocked() ([]AttestationLedgerEntry, error) {
	data, err := os.ReadFile(l.currentFilename())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read attestation ledger: %w", err)
	}

	var entries []AttestationLedgerEntry
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var entry AttestationLedgerEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			return nil, fmt.Errorf("decode attestation ledger: %w", err)
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

func (l *AttestationLedger) currentFilename() string {
	now := time.Now().UTC().Format("2006-01-02")
	return filepath.Join(l.dir, fmt.Sprintf("attestation_%s.jsonl", now))
}

func (l *AttestationLedger) rewrite(entries []AttestationLedgerEntry) error {
	filename := l.currentFilename()
	var buf strings.Builder
	for _, e := range entries {
		data, err := json.Marshal(e)
		if err != nil {
			return fmt.Errorf("marshal entry during prune: %w", err)
		}
		buf.Write(data)
		buf.WriteByte('\n')
	}

	if err := os.WriteFile(filename, []byte(buf.String()), 0644); err != nil {
		return fmt.Errorf("rewrite attestation ledger: %w", err)
	}
	return nil
}

func attestRecordFingerprint(rec *AttestationRecord) string {
	if rec == nil {
		return ""
	}
	h := sha256.Sum256([]byte(rec.HeaderHex + fmt.Sprintf("%d", rec.Nonce)))
	return fmt.Sprintf("%x", h)
}
