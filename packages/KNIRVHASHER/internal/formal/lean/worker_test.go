package lean

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"knirvhasher/pkg/hashing/proofasset"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	assert.Equal(t, "lean", cfg.LeanBinary)
	assert.Equal(t, 15, cfg.MaxCheckerSeconds)
	assert.Contains(t, cfg.WorkDir, "knirv-lean-worker")
	assert.NotEmpty(t, cfg.ImportAllowlist)
}

func TestWorkerSubmitProof_Success(t *testing.T) {
	fake := &FakeRunner{Responses: []FakeResponse{FakeRunnerSuccess()}}
	cfg := DefaultConfig()
	cfg.WorkDir = t.TempDir()
	worker := NewWorker(cfg, fake)

	asset := &proofasset.ProofAsset{
		SchemaVersion:        1,
		ProofSystem:          proofasset.ProofSystemLean,
		ToolchainDigest:      "lean-4.3.0",
		DependencyLockDigest: "lean-lock-abc123",
		TheoremSource:        []byte("theorem t : True := by trivial"),
		ProofSource:          []byte("theorem t : True := by trivial"),
		Imports:              []proofasset.ArtifactRef{},
		CandidateProvenance: proofasset.CandidateProvenance{
			SchemaVersion: 1,
			GeneratedAt:   time.Now().UTC(),
		},
	}

	result, err := worker.SubmitProof(asset)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Receipt)
	assert.Equal(t, proofasset.StatusFormallyVerified, result.Receipt.Status)
	assert.NotEmpty(t, result.Receipt.ProofAssetID)
	assert.NotEmpty(t, result.Receipt.CheckerDigest)
	assert.NotEmpty(t, result.Receipt.CheckedAt)
}

func TestWorkerSubmitProof_Rejected(t *testing.T) {
	fake := &FakeRunner{Responses: []FakeResponse{FakeRunnerRejected("unsolved goal")}}
	cfg := DefaultConfig()
	cfg.WorkDir = t.TempDir()
	worker := NewWorker(cfg, fake)

	asset := &proofasset.ProofAsset{
		SchemaVersion:        1,
		ProofSystem:          proofasset.ProofSystemLean,
		ToolchainDigest:      "lean-4.3.0",
		DependencyLockDigest: "lean-lock-abc123",
		TheoremSource:        []byte("theorem t : True := by trivial"),
		ProofSource:          []byte("theorem t : True := by trivial"),
		Imports:              []proofasset.ArtifactRef{},
		CandidateProvenance: proofasset.CandidateProvenance{
			SchemaVersion: 1,
			GeneratedAt:   time.Now().UTC(),
		},
	}

	result, err := worker.SubmitProof(asset)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Receipt)
	assert.Equal(t, proofasset.StatusFormallyRejected, result.Receipt.Status)
	assert.NotEmpty(t, result.Receipt.ProofAssetID)
}

func TestWorkerSubmitProof_ParseError(t *testing.T) {
	fake := &FakeRunner{Responses: []FakeResponse{FakeRunnerParseError()}}
	cfg := DefaultConfig()
	cfg.WorkDir = t.TempDir()
	worker := NewWorker(cfg, fake)

	asset := &proofasset.ProofAsset{
		SchemaVersion:        1,
		ProofSystem:          proofasset.ProofSystemLean,
		ToolchainDigest:      "lean-4.3.0",
		DependencyLockDigest: "lean-lock-abc123",
		TheoremSource:        []byte("theorem t : True := by trivial"),
		ProofSource:          []byte("theorem t : True := by trivial"),
		Imports:              []proofasset.ArtifactRef{},
		CandidateProvenance: proofasset.CandidateProvenance{
			SchemaVersion: 1,
			GeneratedAt:   time.Now().UTC(),
		},
	}

	result, err := worker.SubmitProof(asset)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Nil(t, result.Receipt)
	assert.Contains(t, result.Diagnostic, proofasset.DiagnosticParseError)
}

func TestWorkerSubmitProof_PrecheckFailure(t *testing.T) {
	fake := &FakeRunner{Responses: []FakeResponse{FakeRunnerSuccess()}}
	worker := NewWorker(DefaultConfig(), fake)

	asset := &proofasset.ProofAsset{
		SchemaVersion:        1,
		ProofSystem:          "unsupported",
		TheoremSource:        []byte("theorem t : True := by trivial"),
		ProofSource:          []byte("theorem t : True := by trivial"),
		Imports:              []proofasset.ArtifactRef{},
		CandidateProvenance: proofasset.CandidateProvenance{
			SchemaVersion: 1,
			GeneratedAt:   time.Now().UTC(),
		},
	}

	result, err := worker.SubmitProof(asset)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "precheck failed")
}

func TestWorkerSubmitProof_ImportDenied(t *testing.T) {
	fake := &FakeRunner{Responses: []FakeResponse{FakeRunnerSuccess()}}
	cfg := DefaultConfig()
	cfg.ImportAllowlist = []string{"Allowed.Lib"}
	worker := NewWorker(cfg, fake)

	asset := &proofasset.ProofAsset{
		SchemaVersion:        1,
		ProofSystem:          proofasset.ProofSystemLean,
		ToolchainDigest:      "lean-4.3.0",
		DependencyLockDigest: "lean-lock-abc123",
		TheoremSource:        []byte("theorem t : True := by trivial"),
		ProofSource:          []byte("theorem t : True := by trivial"),
		Imports: []proofasset.ArtifactRef{
			{Name: "Forbidden.Lib", Digest: "sha256:abc"},
		},
		CandidateProvenance: proofasset.CandidateProvenance{
			SchemaVersion: 1,
			GeneratedAt:   time.Now().UTC(),
		},
	}

	result, err := worker.SubmitProof(asset)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "not in the allowed import list")
}

func TestFakeRunner_CyclesResponses(t *testing.T) {
	fake := &FakeRunner{
		Responses: []FakeResponse{
			FakeRunnerSuccess(),
			FakeRunnerRejected("test"),
		},
	}

	assert.Equal(t, "KNIRV_STATUS=FORMALLY_VERIFIED\n", string(fake.Next().Stdout))
	assert.Equal(t, "KNIRV_STATUS=FORMALLY_REJECTED\ntest\n", string(fake.Next().Stdout))
	assert.Equal(t, "KNIRV_STATUS=FORMALLY_VERIFIED\n", string(fake.Next().Stdout)) // cycles back
}
