package proofasset

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

// ProofSystem identifies the formal proof assistant used to verify an asset.
type ProofSystem string

const (
	ProofSystemLean ProofSystem = "lean"
)

// ArtifactRef names an imported proof artifact by normalized name and digest.
type ArtifactRef struct {
	Name    string `json:"name"`
	Digest  string `json:"digest"`
}

// CandidateProvenance records how a proof candidate was produced. It is audit
// metadata only and must never affect checker acceptance.
type CandidateProvenance struct {
	SchemaVersion      uint32      `json:"schema_version"`
	ModelVersion       string      `json:"model_version,omitempty"`
	PromptHash         string      `json:"prompt_hash,omitempty"`
	ContextHash        string      `json:"context_hash,omitempty"`
	NrvBracketID       string      `json:"nrv_bracket_id,omitempty"`
	SourceDocumentIDs  []string    `json:"source_document_ids,omitempty"`
	GeneratedAt        time.Time   `json:"generated_at"`
}

// ProofAsset is the canonical, content-addressed representation of a theorem
// and its proof source. All fields are explicitly typed so that a pinned
// checker can reproduce the verification environment exactly.
type ProofAsset struct {
	SchemaVersion        uint32        `json:"schema_version"`
	ProofSystem          ProofSystem   `json:"proof_system"`
	ToolchainDigest      string        `json:"toolchain_digest"`
	DependencyLockDigest string        `json:"dependency_lock_digest"`
	TheoremSource        []byte        `json:"theorem_source"`
	ProofSource          []byte        `json:"proof_source"`
	Imports              []ArtifactRef `json:"imports"`
	CandidateProvenance  CandidateProvenance `json:"candidate_provenance"`
}

// VerificationReceipt is produced by the formal checker worker and is the
// only authority that may mark an assertion as FORMALLY_VERIFIED.
type VerificationReceipt struct {
	SchemaVersion     uint32    `json:"schema_version"`
	ProofAssetID      string    `json:"proof_asset_id"`
	Status            string    `json:"status"`
	CheckerDigest     string    `json:"checker_digest"`
	EnvironmentDigest string    `json:"environment_digest"`
	CheckedAt         time.Time `json:"checked_at"`
	DiagnosticDigest  string    `json:"diagnostic_digest,omitempty"`
}

// ProofAssetID is the content-addressed identity of a ProofAsset.
const ProofAssetIDLength = 64 // sha256 hex

// TheoremID is the content-addressed identity of the theorem source within a
// specific proof system and environment.
const TheoremIDLength = 64 // sha256 hex

// CanonicalProofAssetBytes returns a stable, versioned byte representation of
// the asset suitable for hashing. The serialization enforces:
//   - valid UTF-8 only, LF line endings
//   - explicit proof-system and toolchain identity
//   - imports sorted by normalized name then digest
//   - no network-resolved or floating dependencies
//   - bounded artifact size and exact length prefixes
func CanonicalProofAssetBytes(asset *ProofAsset) ([]byte, error) {
	if asset == nil {
		return nil, fmt.Errorf("proof asset is nil")
	}
	if asset.SchemaVersion == 0 {
		return nil, fmt.Errorf("proof asset schema_version is required")
	}
	if asset.ProofSystem == "" {
		return nil, fmt.Errorf("proof asset proof_system is required")
	}
	if len(asset.TheoremSource) == 0 {
		return nil, fmt.Errorf("proof asset theorem_source is required")
	}
	if len(asset.ProofSource) == 0 {
		return nil, fmt.Errorf("proof asset proof_source is required")
	}
	if len(asset.TheoremSource) > 65536 {
		return nil, fmt.Errorf("theorem_source exceeds max_source_bytes (65536): got %d", len(asset.TheoremSource))
	}
	if len(asset.ProofSource) > 65536 {
		return nil, fmt.Errorf("proof_source exceeds max_source_bytes (65536): got %d", len(asset.ProofSource))
	}

	// Normalize line endings to LF and validate UTF-8.
	normalize := func(b []byte) ([]byte, error) {
		s := strings.ReplaceAll(string(b), "\r\n", "\n")
		s = strings.ReplaceAll(s, "\r", "\n")
		if !utf8.ValidString(s) {
			return nil, fmt.Errorf("source contains invalid UTF-8")
		}
		return []byte(s), nil
	}

	theorem, err := normalize(asset.TheoremSource)
	if err != nil {
		return nil, fmt.Errorf("theorem_source normalization: %w", err)
	}
	proof, err := normalize(asset.ProofSource)
	if err != nil {
		return nil, fmt.Errorf("proof_source normalization: %w", err)
	}

	// Sort imports by normalized name then digest for deterministic output.
	sortedImports := make([]ArtifactRef, len(asset.Imports))
	copy(sortedImports, asset.Imports)
	sort.Slice(sortedImports, func(i, j int) bool {
		if sortedImports[i].Name != sortedImports[j].Name {
			return sortedImports[i].Name < sortedImports[j].Name
		}
		return sortedImports[i].Digest < sortedImports[j].Digest
	})

	// Build a self-contained JSON document with explicit length prefixes so
	// the serialization is unambiguous even for byte payloads that contain
	// JSON-special characters.
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("schema_version=%d\n", asset.SchemaVersion))
	buf.WriteString(fmt.Sprintf("proof_system=%s\n", asset.ProofSystem))
	buf.WriteString(fmt.Sprintf("toolchain_digest=%s\n", asset.ToolchainDigest))
	buf.WriteString(fmt.Sprintf("dependency_lock_digest=%s\n", asset.DependencyLockDigest))
	buf.WriteString(fmt.Sprintf("theorem_source_length=%d\n", len(theorem)))
	buf.Write(theorem)
	buf.WriteByte('\n')
	buf.WriteString(fmt.Sprintf("proof_source_length=%d\n", len(proof)))
	buf.Write(proof)
	buf.WriteByte('\n')
	buf.WriteString(fmt.Sprintf("imports_count=%d\n", len(sortedImports)))
	for _, imp := range sortedImports {
		buf.WriteString(fmt.Sprintf("import_name=%s\n", imp.Name))
		buf.WriteString(fmt.Sprintf("import_digest=%s\n", imp.Digest))
	}
	buf.WriteString(fmt.Sprintf("provenance_schema_version=%d\n", asset.CandidateProvenance.SchemaVersion))
	if asset.CandidateProvenance.ModelVersion != "" {
		buf.WriteString(fmt.Sprintf("provenance_model_version=%s\n", asset.CandidateProvenance.ModelVersion))
	}
	if asset.CandidateProvenance.PromptHash != "" {
		buf.WriteString(fmt.Sprintf("provenance_prompt_hash=%s\n", asset.CandidateProvenance.PromptHash))
	}
	if asset.CandidateProvenance.ContextHash != "" {
		buf.WriteString(fmt.Sprintf("provenance_context_hash=%s\n", asset.CandidateProvenance.ContextHash))
	}
	if asset.CandidateProvenance.NrvBracketID != "" {
		buf.WriteString(fmt.Sprintf("provenance_nrv_bracket_id=%s\n", asset.CandidateProvenance.NrvBracketID))
	}
	if len(asset.CandidateProvenance.SourceDocumentIDs) > 0 {
		buf.WriteString(fmt.Sprintf("provenance_source_document_ids_count=%d\n", len(asset.CandidateProvenance.SourceDocumentIDs)))
		for _, id := range asset.CandidateProvenance.SourceDocumentIDs {
			buf.WriteString(fmt.Sprintf("provenance_source_document_id=%s\n", id))
		}
	}
	buf.WriteString(fmt.Sprintf("provenance_generated_at=%s\n", asset.CandidateProvenance.GeneratedAt.UTC().Format(time.RFC3339Nano)))

	return buf.Bytes(), nil
}

// ComputeProofAssetID returns the content-addressed identity of the asset.
func ComputeProofAssetID(asset *ProofAsset) (string, error) {
	canonical, err := CanonicalProofAssetBytes(asset)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", sha256.Sum256(canonical)), nil
}

// ComputeTheoremID returns the theorem-level identity that is independent of
// the proof source but bound to a specific proof system and environment.
func ComputeTheoremID(proofSystem ProofSystem, toolchainDigest string, theoremSource []byte) (string, error) {
	if proofSystem == "" {
		return "", fmt.Errorf("proof system is required")
	}
	if toolchainDigest == "" {
		return "", fmt.Errorf("toolchain digest is required")
	}
	if len(theoremSource) == 0 {
		return "", fmt.Errorf("theorem source is required")
	}

	s := strings.ReplaceAll(string(theoremSource), "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	if !utf8.ValidString(s) {
		return "", fmt.Errorf("theorem source contains invalid UTF-8")
	}

	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("proof_system=%s\n", proofSystem))
	buf.WriteString(fmt.Sprintf("toolchain_digest=%s\n", toolchainDigest))
	buf.WriteString(fmt.Sprintf("theorem_source_length=%d\n", len(s)))
	buf.WriteString(s)
	buf.WriteByte('\n')

	return fmt.Sprintf("%x", sha256.Sum256(buf.Bytes())), nil
}

// ValidateProofAsset performs structural validation of a ProofAsset before it
// is submitted to the formal checker. It checks type invariants, size bounds,
// and import allowlist constraints. It does not invoke any checker and must
// never return FORMALLY_VERIFIED.
func ValidateProofAsset(asset *ProofAsset, allowedImports []string) error {
	if asset == nil {
		return fmt.Errorf("proof asset is nil")
	}
	if asset.SchemaVersion != 1 {
		return fmt.Errorf("unsupported schema_version %d: expected 1", asset.SchemaVersion)
	}
	if asset.ProofSystem != ProofSystemLean {
		return fmt.Errorf("unsupported proof system %q: only %q is supported", asset.ProofSystem, ProofSystemLean)
	}
	if asset.ToolchainDigest == "" {
		return fmt.Errorf("toolchain_digest is required")
	}
	if asset.DependencyLockDigest == "" {
		return fmt.Errorf("dependency_lock_digest is required")
	}
	if len(asset.TheoremSource) == 0 {
		return fmt.Errorf("theorem_source is required")
	}
	if len(asset.TheoremSource) > 65536 {
		return fmt.Errorf("theorem_source exceeds 65536 bytes: got %d", len(asset.TheoremSource))
	}
	if len(asset.ProofSource) == 0 {
		return fmt.Errorf("proof_source is required")
	}
	if len(asset.ProofSource) > 65536 {
		return fmt.Errorf("proof_source exceeds 65536 bytes: got %d", len(asset.ProofSource))
	}

	// Validate UTF-8 and LF normalization.
	if !utf8.ValidString(string(asset.TheoremSource)) {
		return fmt.Errorf("theorem_source contains invalid UTF-8")
	}
	if !utf8.ValidString(string(asset.ProofSource)) {
		return fmt.Errorf("proof_source contains invalid UTF-8")
	}

	// Check imports against allowlist.
	if len(allowedImports) > 0 {
		allowSet := make(map[string]struct{}, len(allowedImports))
		for _, imp := range allowedImports {
			allowSet[imp] = struct{}{}
		}
		for _, imp := range asset.Imports {
			if _, ok := allowSet[imp.Name]; !ok {
				return fmt.Errorf("import %q is not in the allowed import list", imp.Name)
			}
		}
	}

	// Provenance timestamps must be sensible.
	if asset.CandidateProvenance.GeneratedAt.IsZero() {
		return fmt.Errorf("candidate_provenance.generated_at is required")
	}

	return nil
}

// ProofLedgerEntry is a single append-only record in proof_writes.jsonl.
type ProofLedgerEntry struct {
	SchemaVersion        uint32     `json:"schema_version"`
	ProofAssetID         string     `json:"proof_asset_id"`
	TheoremID            string     `json:"theorem_id"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
	CanonicalAssetPath   string     `json:"canonical_asset_path,omitempty"`
	CanonicalAssetDigest string     `json:"canonical_asset_digest,omitempty"`
	ProofSystem          ProofSystem `json:"proof_system"`
	CheckerDigest        string     `json:"checker_digest"`
	DependencyLockDigest string     `json:"dependency_lock_digest"`
	EnvironmentDigest    string     `json:"environment_digest"`
	FinalStatus          string     `json:"final_status"`
	DiagnosticDigest     string     `json:"diagnostic_digest,omitempty"`
	NrvBracketID         string     `json:"nrv_bracket_id,omitempty"`
	AsicAttestation      *AsicAttestationRecord `json:"asic_attestation,omitempty"`
	Receipt              *VerificationReceipt   `json:"receipt,omitempty"`
}

// AsicAttestationRecord is an optional hardware attestation attached to a
// verified proof asset.
type AsicAttestationRecord struct {
	HeaderBytes     string    `json:"header_bytes"`
	HeaderVersion   uint32    `json:"header_version"`
	Target          uint32    `json:"target"`
	Nonce           uint32    `json:"nonce"`
	DoubleSHA256    string    `json:"double_sha256"`
	DeviceFirmware  string    `json:"device_firmware"`
	AttestedAt      time.Time `json:"attested_at"`
}

// ProofLedger manages an append-only proof_writes.jsonl ledger. It is safe for
// concurrent use.
type ProofLedger struct {
	path string
	mu   sync.Mutex
}

// NewProofLedger creates a ProofLedger that writes to the given path.
func NewProofLedger(path string) *ProofLedger {
	return &ProofLedger{path: path}
}

// Append adds a new entry to the proof ledger. The ledger only accepts
// FORMALLY_VERIFIED entries when the receipt validates against the exact
// stored artifact IDs.
func (l *ProofLedger) Append(entry ProofLedgerEntry) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if entry.FinalStatus == StatusFormallyVerified {
		if entry.Receipt == nil {
			return fmt.Errorf("ledger rejects FORMALLY_VERIFIED entry without a verification receipt")
		}
		if entry.Receipt.ProofAssetID != entry.ProofAssetID {
			return fmt.Errorf("ledger rejects FORMALLY_VERIFIED entry: receipt proof_asset_id %q does not match entry %q",
				entry.Receipt.ProofAssetID, entry.ProofAssetID)
		}
	}

	entry.UpdatedAt = time.Now().UTC()
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = entry.UpdatedAt
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshal proof ledger entry: %w", err)
	}

	dir := filepath.Dir(l.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create proof ledger directory: %w", err)
	}

	f, err := os.OpenFile(l.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("open proof ledger: %w", err)
	}
	defer f.Close()

	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("append proof ledger: %w", err)
	}

	return nil
}

// ReadAll reads all entries from the proof ledger. It tolerates interrupted
// historical rows.
func (l *ProofLedger) ReadAll() ([]ProofLedgerEntry, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	data, err := os.ReadFile(l.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read proof ledger: %w", err)
	}

	var entries []ProofLedgerEntry
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var entry ProofLedgerEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue // append-only ledgers may contain interrupted historical rows
		}
		entries = append(entries, entry)
	}

	return entries, nil
}
