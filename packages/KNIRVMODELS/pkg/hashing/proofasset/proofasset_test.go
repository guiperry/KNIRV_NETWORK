package proofasset

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStatusConstants(t *testing.T) {
	assert.Equal(t, "STRUCTURALLY_VALID", StatusStructurallyValid)
	assert.Equal(t, "STRUCTURALLY_REJECTED", StatusStructurallyRejected)
	assert.Equal(t, "PROOF_PENDING", StatusProofPending)
	assert.Equal(t, "FORMALLY_VERIFIED", StatusFormallyVerified)
	assert.Equal(t, "FORMALLY_REJECTED", StatusFormallyRejected)
	assert.Equal(t, "CHECKER_UNAVAILABLE", StatusCheckerUnavailable)
	assert.Equal(t, "ATTESTATION_PENDING", StatusAttestationPending)
	assert.Equal(t, "HARDWARE_ATTESTED", StatusHardwareAttested)
}

func TestDiagnosticConstants(t *testing.T) {
	assert.Equal(t, "PARSE_ERROR", string(DiagnosticParseError))
	assert.Equal(t, "UNKNOWN_IDENTIFIER", string(DiagnosticUnknownIdentifier))
	assert.Equal(t, "TYPE_MISMATCH", string(DiagnosticTypeMismatch))
	assert.Equal(t, "UNSOLVED_GOAL", string(DiagnosticUnsolvedGoal))
	assert.Equal(t, "IMPORT_POLICY_DENIED", string(DiagnosticImportPolicyDenied))
	assert.Equal(t, "RESOURCE_LIMIT", string(DiagnosticResourceLimit))
	assert.Equal(t, "CHECKER_FAILURE", string(DiagnosticCheckerFailure))
}

func TestCanonicalProofAssetBytes(t *testing.T) {
	asset := &ProofAsset{
		SchemaVersion:        1,
		ProofSystem:          ProofSystemLean,
		ToolchainDigest:      "lean-4.3.0",
		DependencyLockDigest: "lean-lock-abc123",
		TheoremSource:        []byte("theorem example : True := by trivial"),
		ProofSource:          []byte("theorem example : True := by trivial"),
		Imports: []ArtifactRef{
			{Name: "Mathlib.Data.Real.Basic", Digest: "sha256:def456"},
		},
		CandidateProvenance: CandidateProvenance{
			SchemaVersion: 1,
			GeneratedAt:   time.Now().UTC(),
		},
	}

	canonical, err := CanonicalProofAssetBytes(asset)
	require.NoError(t, err)
	assert.NotEmpty(t, canonical)

	// Verify determinism: same asset produces same canonical bytes.
	canonical2, err := CanonicalProofAssetBytes(asset)
	require.NoError(t, err)
	assert.Equal(t, canonical, canonical2)
}

func TestCanonicalProofAssetBytesStable(t *testing.T) {
	now := time.Now().UTC()
	asset1 := &ProofAsset{
		SchemaVersion:        1,
		ProofSystem:          ProofSystemLean,
		ToolchainDigest:      "lean-4.3.0",
		DependencyLockDigest: "lean-lock-abc123",
		TheoremSource:        []byte("theorem t : True := by trivial"),
		ProofSource:          []byte("theorem t : True := by trivial"),
		Imports:              []ArtifactRef{},
		CandidateProvenance: CandidateProvenance{
			SchemaVersion: 1,
			GeneratedAt:   now,
		},
	}
	asset2 := &ProofAsset{
		SchemaVersion:        1,
		ProofSystem:          ProofSystemLean,
		ToolchainDigest:      "lean-4.3.0",
		DependencyLockDigest: "lean-lock-abc123",
		TheoremSource:        []byte("theorem t : True := by trivial"),
		ProofSource:          []byte("theorem t : True := by trivial"),
		Imports:              []ArtifactRef{},
		CandidateProvenance: CandidateProvenance{
			SchemaVersion: 1,
			GeneratedAt:   now,
		},
	}

	c1, err := CanonicalProofAssetBytes(asset1)
	require.NoError(t, err)
	c2, err := CanonicalProofAssetBytes(asset2)
	require.NoError(t, err)
	assert.Equal(t, c1, c2, "canonical serialization must be stable across processes")
}

func TestCanonicalProofAssetBytesImportsSorted(t *testing.T) {
	asset := &ProofAsset{
		SchemaVersion:        1,
		ProofSystem:          ProofSystemLean,
		ToolchainDigest:      "lean-4.3.0",
		DependencyLockDigest: "lean-lock-abc123",
		TheoremSource:        []byte("theorem t : True := by trivial"),
		ProofSource:          []byte("theorem t : True := by trivial"),
		Imports: []ArtifactRef{
			{Name: "Z.Import", Digest: "sha256:zzz"},
			{Name: "A.Import", Digest: "sha256:aaa"},
		},
		CandidateProvenance: CandidateProvenance{
			SchemaVersion: 1,
			GeneratedAt:   time.Now().UTC(),
		},
	}

	canonical, err := CanonicalProofAssetBytes(asset)
	require.NoError(t, err)
	assert.Contains(t, string(canonical), "A.Import")
	assert.Contains(t, string(canonical), "Z.Import")
	// A.Import should appear before Z.Import in canonical form.
	idxA := bytes.Index(canonical, []byte("A.Import"))
	idxZ := bytes.Index(canonical, []byte("Z.Import"))
	assert.Less(t, idxA, idxZ, "imports must be sorted by name")
}

func TestCanonicalProofAssetBytesIDChanges(t *testing.T) {
	base := &ProofAsset{
		SchemaVersion:        1,
		ProofSystem:          ProofSystemLean,
		ToolchainDigest:      "lean-4.3.0",
		DependencyLockDigest: "lean-lock-abc123",
		TheoremSource:        []byte("theorem t : True := by trivial"),
		ProofSource:          []byte("theorem t : True := by trivial"),
		Imports:              []ArtifactRef{},
		CandidateProvenance: CandidateProvenance{
			SchemaVersion: 1,
			GeneratedAt:   time.Now().UTC(),
		},
	}
	baseID, err := ComputeProofAssetID(base)
	require.NoError(t, err)

	// Change theorem source: ID must change.
	modTheorem := &ProofAsset{
		SchemaVersion:        1,
		ProofSystem:          ProofSystemLean,
		ToolchainDigest:      "lean-4.3.0",
		DependencyLockDigest: "lean-lock-abc123",
		TheoremSource:        []byte("theorem t2 : True := by trivial"),
		ProofSource:          []byte("theorem t : True := by trivial"),
		Imports:              []ArtifactRef{},
		CandidateProvenance: CandidateProvenance{
			SchemaVersion: 1,
			GeneratedAt:   time.Now().UTC(),
		},
	}
	modID, err := ComputeProofAssetID(modTheorem)
	require.NoError(t, err)
	assert.NotEqual(t, baseID, modID, "changing theorem source must change asset ID")

	// Change proof source: ID must change.
	modProof := &ProofAsset{
		SchemaVersion:        1,
		ProofSystem:          ProofSystemLean,
		ToolchainDigest:      "lean-4.3.0",
		DependencyLockDigest: "lean-lock-abc123",
		TheoremSource:        []byte("theorem t : True := by trivial"),
		ProofSource:          []byte("theorem t2 : True := by trivial"),
		Imports:              []ArtifactRef{},
		CandidateProvenance: CandidateProvenance{
			SchemaVersion: 1,
			GeneratedAt:   time.Now().UTC(),
		},
	}
	modID2, err := ComputeProofAssetID(modProof)
	require.NoError(t, err)
	assert.NotEqual(t, baseID, modID2, "changing proof source must change asset ID")

	// Change toolchain: ID must change.
	modToolchain := &ProofAsset{
		SchemaVersion:        1,
		ProofSystem:          ProofSystemLean,
		ToolchainDigest:      "lean-4.4.0",
		DependencyLockDigest: "lean-lock-abc123",
		TheoremSource:        []byte("theorem t : True := by trivial"),
		ProofSource:          []byte("theorem t : True := by trivial"),
		Imports:              []ArtifactRef{},
		CandidateProvenance: CandidateProvenance{
			SchemaVersion: 1,
			GeneratedAt:   time.Now().UTC(),
		},
	}
	modID3, err := ComputeProofAssetID(modToolchain)
	require.NoError(t, err)
	assert.NotEqual(t, baseID, modID3, "changing toolchain must change asset ID")
}

func TestComputeTheoremID(t *testing.T) {
	id1, err := ComputeTheoremID(ProofSystemLean, "lean-4.3.0", []byte("theorem t : True := by trivial"))
	require.NoError(t, err)
	id2, err := ComputeTheoremID(ProofSystemLean, "lean-4.3.0", []byte("theorem t : True := by trivial"))
	require.NoError(t, err)
	assert.Equal(t, id1, id2, "same inputs must produce same theorem ID")

	id3, err := ComputeTheoremID(ProofSystemLean, "lean-4.4.0", []byte("theorem t : True := by trivial"))
	require.NoError(t, err)
	assert.NotEqual(t, id1, id3, "different toolchain must change theorem ID")

	id4, err := ComputeTheoremID("coq", "lean-4.3.0", []byte("theorem t : True := by trivial"))
	require.NoError(t, err)
	assert.NotEqual(t, id1, id4, "different proof system must change theorem ID")
}

func TestValidateProofAsset(t *testing.T) {
	t.Run("nil asset", func(t *testing.T) {
		err := ValidateProofAsset(nil, []string{})
		assert.Error(t, err)
	})

	t.Run("missing schema version", func(t *testing.T) {
		asset := &ProofAsset{
			ProofSystem:     ProofSystemLean,
			TheoremSource:   []byte("theorem t : True := by trivial"),
			ProofSource:     []byte("theorem t : True := by trivial"),
		}
		err := ValidateProofAsset(asset, []string{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "schema_version")
	})

	t.Run("unsupported proof system", func(t *testing.T) {
		asset := &ProofAsset{
			SchemaVersion:  1,
			ProofSystem:    "coq",
			TheoremSource:  []byte("theorem t : True := by trivial"),
			ProofSource:    []byte("theorem t : True := by trivial"),
		}
		err := ValidateProofAsset(asset, []string{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported proof system")
	})

	t.Run("missing toolchain digest", func(t *testing.T) {
		asset := &ProofAsset{
			SchemaVersion:  1,
			ProofSystem:    ProofSystemLean,
			TheoremSource:  []byte("theorem t : True := by trivial"),
			ProofSource:    []byte("theorem t : True := by trivial"),
		}
		err := ValidateProofAsset(asset, []string{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "toolchain_digest")
	})

	t.Run("oversized source", func(t *testing.T) {
		asset := &ProofAsset{
			SchemaVersion:        1,
			ProofSystem:          ProofSystemLean,
			ToolchainDigest:      "lean-4.3.0",
			DependencyLockDigest: "lean-lock-abc123",
			TheoremSource:        make([]byte, 65537),
			ProofSource:          []byte("theorem t : True := by trivial"),
		}
		err := ValidateProofAsset(asset, []string{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "exceeds")
	})

	t.Run("invalid UTF-8", func(t *testing.T) {
		asset := &ProofAsset{
			SchemaVersion:        1,
			ProofSystem:          ProofSystemLean,
			ToolchainDigest:      "lean-4.3.0",
			DependencyLockDigest: "lean-lock-abc123",
			TheoremSource:        []byte{0xff, 0xfe},
			ProofSource:          []byte("theorem t : True := by trivial"),
		}
		err := ValidateProofAsset(asset, []string{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid UTF-8")
	})

	t.Run("import not in allowlist", func(t *testing.T) {
		asset := &ProofAsset{
			SchemaVersion:        1,
			ProofSystem:          ProofSystemLean,
			ToolchainDigest:      "lean-4.3.0",
			DependencyLockDigest: "lean-lock-abc123",
			TheoremSource:        []byte("theorem t : True := by trivial"),
			ProofSource:          []byte("theorem t : True := by trivial"),
			Imports: []ArtifactRef{
				{Name: "Forbidden.Lib", Digest: "sha256:abc"},
			},
		}
		err := ValidateProofAsset(asset, []string{"Allowed.Lib"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not in the allowed import list")
	})

	t.Run("valid asset", func(t *testing.T) {
		asset := &ProofAsset{
			SchemaVersion:        1,
			ProofSystem:          ProofSystemLean,
			ToolchainDigest:      "lean-4.3.0",
			DependencyLockDigest: "lean-lock-abc123",
			TheoremSource:        []byte("theorem t : True := by trivial"),
			ProofSource:          []byte("theorem t : True := by trivial"),
			Imports: []ArtifactRef{
				{Name: "Mathlib.Data.Real.Basic", Digest: "sha256:def456"},
			},
			CandidateProvenance: CandidateProvenance{
				SchemaVersion: 1,
				GeneratedAt:   time.Now().UTC(),
			},
		}
		err := ValidateProofAsset(asset, []string{"Mathlib.Data.Real.Basic"})
		assert.NoError(t, err)
	})
}

func TestProofLedger(t *testing.T) {
	dir := t.TempDir()
	ledgerPath := filePathJoin(dir, "proof_writes.jsonl")
	ledger := NewProofLedger(ledgerPath)

	entry := ProofLedgerEntry{
		SchemaVersion:        1,
		ProofAssetID:         "abc123",
		TheoremID:            "theorem-xyz",
		ProofSystem:          ProofSystemLean,
		CheckerDigest:        "checker-abc",
		DependencyLockDigest: "lock-abc",
		EnvironmentDigest:    "env-abc",
		FinalStatus:          StatusFormallyVerified,
		NrvBracketID:         "nrv-001",
		Receipt: &VerificationReceipt{
			SchemaVersion:     1,
			ProofAssetID:      "abc123",
			Status:            StatusFormallyVerified,
			CheckerDigest:     "checker-abc",
			EnvironmentDigest: "env-abc",
			CheckedAt:         time.Now().UTC(),
		},
	}

	err := ledger.Append(entry)
	require.NoError(t, err)

	entries, err := ledger.ReadAll()
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "abc123", entries[0].ProofAssetID)
	assert.Equal(t, StatusFormallyVerified, entries[0].FinalStatus)
}

func TestProofLedgerRejectsVerifiedWithoutReceipt(t *testing.T) {
	dir := t.TempDir()
	ledgerPath := filePathJoin(dir, "proof_writes.jsonl")
	ledger := NewProofLedger(ledgerPath)

	entry := ProofLedgerEntry{
		SchemaVersion:  1,
		ProofAssetID:   "abc123",
		FinalStatus:    StatusFormallyVerified,
		Receipt:        nil, // missing receipt
	}

	err := ledger.Append(entry)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "without a verification receipt")
}

func TestProofLedgerRejectsMismatchedReceipt(t *testing.T) {
	dir := t.TempDir()
	ledgerPath := filePathJoin(dir, "proof_writes.jsonl")
	ledger := NewProofLedger(ledgerPath)

	entry := ProofLedgerEntry{
		SchemaVersion:  1,
		ProofAssetID:   "abc123",
		FinalStatus:    StatusFormallyVerified,
		Receipt: &VerificationReceipt{
			SchemaVersion: 1,
			ProofAssetID:  "wrong-id",
			Status:        StatusFormallyVerified,
		},
	}

	err := ledger.Append(entry)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not match entry")
}

func filePathJoin(dir, file string) string {
	return dir + "/" + file
}
