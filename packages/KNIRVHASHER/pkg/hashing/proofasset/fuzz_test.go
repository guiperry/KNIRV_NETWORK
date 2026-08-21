package proofasset

import (
	"encoding/json"
	"testing"
	"time"
)

func FuzzCanonicalProofAssetBytes(f *testing.F) {
	f.Add("theorem x : True := by trivial", "theorem x : True := by trivial", "Mathlib.Data.Real.Basic")
	f.Add("", "", "")
	f.Add("theorem x : True := by\n  exact id\n", "theorem x : True := by\n  exact id\n", "Mathlib.Algebra.Group.Basic")

	f.Fuzz(func(t *testing.T, theorem, proof, importName string) {
		asset := &ProofAsset{
			SchemaVersion:        1,
			ProofSystem:          ProofSystemLean,
			ToolchainDigest:      "lean-dev",
			DependencyLockDigest: "lean-1-imports",
			TheoremSource:        []byte(theorem),
			ProofSource:          []byte(proof),
			Imports: []ArtifactRef{
				{Name: importName, Digest: "sha256:unknown"},
			},
			CandidateProvenance: CandidateProvenance{
				SchemaVersion: 1,
				GeneratedAt:   timeNow(),
			},
		}
		_, _ = CanonicalProofAssetBytes(asset)
		_, _ = ComputeProofAssetID(asset)
		_ = ValidateProofAsset(asset, []string{importName})
	})
}

func FuzzProofLedgerRead(f *testing.F) {
	f.Add(`{"schema_version":1,"proof_asset_id":"abc","final_status":"FORMALLY_VERIFIED"}`)
	f.Add("not json\n")
	f.Add("")

	f.Fuzz(func(t *testing.T, line string) {
		var entry ProofLedgerEntry
		_ = json.Unmarshal([]byte(line), &entry)
	})
}

func FuzzValidateProofAsset(f *testing.F) {
	f.Add("theorem x : True := by trivial", "theorem x : True := by trivial")

	f.Fuzz(func(t *testing.T, theorem, proof string) {
		asset := &ProofAsset{
			SchemaVersion:        1,
			ProofSystem:          ProofSystemLean,
			ToolchainDigest:      "lean-dev",
			DependencyLockDigest: "lean-1-imports",
			TheoremSource:        []byte(theorem),
			ProofSource:          []byte(proof),
			Imports: []ArtifactRef{
				{Name: "Mathlib.Data.Real.Basic", Digest: "sha256:unknown"},
			},
			CandidateProvenance: CandidateProvenance{
				SchemaVersion: 1,
				GeneratedAt:   timeNow(),
			},
		}
		_ = ValidateProofAsset(asset, []string{"Mathlib.Data.Real.Basic"})
	})
}

func timeNow() time.Time {
	return time.Now().UTC()
}
