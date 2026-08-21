package lean

import (
	"os"
	"strings"
	"testing"
	"time"

	"knirvhasher/pkg/hashing/proofasset"
)

func TestWorkerSubmitProof_CommandInjectionRejected(t *testing.T) {
	w := NewWorker(DefaultConfig(), &FakeRunner{Responses: []FakeResponse{FakeRunnerSuccess()}})
	asset := &proofasset.ProofAsset{
		SchemaVersion:        1,
		ProofSystem:          proofasset.ProofSystemLean,
		ToolchainDigest:      "lean-dev",
		DependencyLockDigest: "lean-2-imports",
		TheoremSource:        []byte("theorem x : True := by\n  trivial\n`whoami`\n"),
		ProofSource:          []byte("theorem x : True := by\n  trivial\n"),
		Imports: []proofasset.ArtifactRef{
			{Name: "Mathlib.Algebra.Group.Basic", Digest: "sha256:unknown"},
		},
		CandidateProvenance: proofasset.CandidateProvenance{
			SchemaVersion: 1,
			GeneratedAt:   time.Now().UTC(),
		},
	}
	_, err := w.SubmitProof(asset)
	if err == nil {
		t.Fatal("expected command injection to be rejected")
	}
	if !strings.Contains(err.Error(), "command injection") {
		t.Fatalf("expected command injection error, got: %v", err)
	}
}

func TestWorkerSubmitProof_NativeBuildHookRejected(t *testing.T) {
	w := NewWorker(DefaultConfig(), &FakeRunner{Responses: []FakeResponse{FakeRunnerSuccess()}})
	asset := &proofasset.ProofAsset{
		SchemaVersion:        1,
		ProofSystem:          proofasset.ProofSystemLean,
		ToolchainDigest:      "lean-dev",
		DependencyLockDigest: "lean-2-imports",
		TheoremSource:        []byte("theorem x : True := by\n  trivial\n"),
		ProofSource:          []byte("#if __builtin_expect(1,1)\ntrivial\n"),
		Imports: []proofasset.ArtifactRef{
			{Name: "Mathlib.Algebra.Group.Basic", Digest: "sha256:unknown"},
		},
		CandidateProvenance: proofasset.CandidateProvenance{
			SchemaVersion: 1,
			GeneratedAt:   time.Now().UTC(),
		},
	}
	_, err := w.SubmitProof(asset)
	if err == nil {
		t.Fatal("expected native build hook to be rejected")
	}
}

func TestWorkerSubmitProof_ToolchainDigestMismatch(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ToolchainDigest = "sha256:deadbeef"
	w := NewWorker(cfg, &FakeRunner{Responses: []FakeResponse{FakeRunnerSuccess()}})
	asset := &proofasset.ProofAsset{
		SchemaVersion:        1,
		ProofSystem:          proofasset.ProofSystemLean,
		ToolchainDigest:      "lean-dev",
		DependencyLockDigest: "lean-2-imports",
		TheoremSource:        []byte("theorem x : True := by\n  trivial\n"),
		ProofSource:          []byte("theorem x : True := by\n  trivial\n"),
		Imports: []proofasset.ArtifactRef{
			{Name: "Mathlib.Algebra.Group.Basic", Digest: "sha256:unknown"},
		},
		CandidateProvenance: proofasset.CandidateProvenance{
			SchemaVersion: 1,
			GeneratedAt:   time.Now().UTC(),
		},
	}
	_, err := w.SubmitProof(asset)
	if err == nil {
		t.Fatal("expected toolchain mismatch to fail")
	}
}

func TestWorkerSubmitProof_RelativeBinaryRejected(t *testing.T) {
	cfg := DefaultConfig()
	cfg.LeanBinary = "lean"
	cfg.ToolchainDigest = "sha256:abcd"
	w := NewWorker(cfg, &FakeRunner{Responses: []FakeResponse{FakeRunnerSuccess()}})
	asset := &proofasset.ProofAsset{
		SchemaVersion:        1,
		ProofSystem:          proofasset.ProofSystemLean,
		ToolchainDigest:      "lean-dev",
		DependencyLockDigest: "lean-2-imports",
		TheoremSource:        []byte("theorem x : True := by\n  trivial\n"),
		ProofSource:          []byte("theorem x : True := by\n  trivial\n"),
		Imports: []proofasset.ArtifactRef{
			{Name: "Mathlib.Algebra.Group.Basic", Digest: "sha256:unknown"},
		},
		CandidateProvenance: proofasset.CandidateProvenance{
			SchemaVersion: 1,
			GeneratedAt:   time.Now().UTC(),
		},
	}
	_, err := w.SubmitProof(asset)
	if err == nil {
		t.Fatal("expected relative binary path to be rejected")
	}
}

func TestWorkerSubmitProof_JobDirCleanup(t *testing.T) {
	cfg := DefaultConfig()
	cfg.WorkDir = t.TempDir()
	w := NewWorker(cfg, &FakeRunner{Responses: []FakeResponse{FakeRunnerSuccess()}})
	asset := &proofasset.ProofAsset{
		SchemaVersion:        1,
		ProofSystem:          proofasset.ProofSystemLean,
		ToolchainDigest:      "lean-dev",
		DependencyLockDigest: "lean-2-imports",
		TheoremSource:        []byte("theorem x : True := by\n  trivial\n"),
		ProofSource:          []byte("theorem x : True := by\n  trivial\n"),
		Imports: []proofasset.ArtifactRef{
			{Name: "Mathlib.Algebra.Group.Basic", Digest: "sha256:unknown"},
		},
		CandidateProvenance: proofasset.CandidateProvenance{
			SchemaVersion: 1,
			GeneratedAt:   time.Now().UTC(),
		},
	}
	_, err := w.SubmitProof(asset)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	entries, _ := os.ReadDir(cfg.WorkDir)
	for _, e := range entries {
		if e.IsDir() && strings.HasPrefix(e.Name(), "lean-job-") {
			t.Fatalf("job directory not cleaned up: %s", e.Name())
		}
	}
}
