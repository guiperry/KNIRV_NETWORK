package blockchain

import (
	"encoding/json"
	"testing"
	"time"

	"KNIRVCHAIN/internal/utils"
)

func TestValidateValidationProofMint(t *testing.T) {
	t.Setenv("KNIRV_PROOF_REQUIRED_REPLICAS", "1")
	request := validValidationProofMint("proof-one", "commit-one")
	if err := validateValidationProofMint(request); err != nil {
		t.Fatalf("valid mint rejected: %v", err)
	}
	request.ValidationCertificate.PolicyHash = proofDigest("different-policy")
	if err := validateValidationProofMint(request); err == nil {
		t.Fatal("certificate binding mismatch accepted")
	}
}

func TestValidationProofConsensusRejectsCommitRebinding(t *testing.T) {
	t.Setenv("KNIRV_PROOF_REQUIRED_REPLICAS", "1")
	first := validValidationProofMint("proof-one", "same-commit")
	second := validValidationProofMint("proof-two", "same-commit")
	firstBytes, _ := json.Marshal(first)
	secondBytes, _ := json.Marshal(second)
	chain := &BlockchainStruct{Blocks: []*Block{{Transactions: []*Transaction{{
		From: utils.BLOCKCHAIN_ADDRESS, To: first.Submission.ProofRoot, Data: firstBytes,
	}}}}}
	block := &Block{Transactions: []*Transaction{{
		From: utils.BLOCKCHAIN_ADDRESS, To: second.Submission.ProofRoot, Data: secondBytes,
	}}}
	if err := chain.validateValidationProofMintsInBlock(block); err == nil {
		t.Fatal("consensus accepted a second primary proof for one commit")
	}
}

func TestValidationProofConsensusRejectsProofReassignment(t *testing.T) {
	t.Setenv("KNIRV_PROOF_REQUIRED_REPLICAS", "1")
	first := validValidationProofMint("same-proof", "commit-one")
	second := validValidationProofMint("same-proof", "commit-two")
	firstBytes, _ := json.Marshal(first)
	secondBytes, _ := json.Marshal(second)
	chain := &BlockchainStruct{Blocks: []*Block{{Transactions: []*Transaction{{
		From: utils.BLOCKCHAIN_ADDRESS, To: first.Submission.ProofRoot, Data: firstBytes,
	}}}}}
	block := &Block{Transactions: []*Transaction{{
		From: utils.BLOCKCHAIN_ADDRESS, To: second.Submission.ProofRoot, Data: secondBytes,
	}}}
	if err := chain.validateValidationProofMintsInBlock(block); err == nil {
		t.Fatal("consensus accepted reassignment of a proof root")
	}
}

func validValidationProofMint(proofSeed, commitSeed string) validationProofMintRequest {
	now := time.Unix(1700000000, 0).UTC()
	request := validationProofMintRequest{
		SchemaVersion: validationProofMintSchema,
		Submission: validationProofSubmission{
			SchemaVersion: "knirv.proof-submission.v1", BundleSchema: "dve.bundle.v1",
			ProjectID: "project-one", SessionID: "session-one", ProofRoot: proofDigest(proofSeed),
			EncryptedManifestCID: proofDigest("manifest"),
			Objects:              []validationObjectRef{{CID: proofDigest("manifest"), Size: 42}},
			KeyEnvelopes: []validationKeyEnvelope{
				{RecipientID: "owner", KeyID: "owner-key", Algorithm: "owner-wrap", WrappedKey: "owner-ciphertext"},
				{RecipientID: "validator", KeyID: "validator-key", Algorithm: "x25519-xchacha20poly1305", WrappedKey: "validator-ciphertext"},
			},
			RepositoryFingerprint: "repo:one",
			Git:                   validationGit{ObjectFormat: "sha1", OID: proofHex("a", 40), RawSHA256: proofDigest(commitSeed), TreeOID: proofHex("b", 40)},
			Workspace: validationWorkspace{
				BaseHash: proofDigest("base"), FinalHash: proofDigest("final"), DiffHash: proofDigest("diff"),
			},
			PolicyHash: proofDigest("policy"),
			SignerID:   "user-one", SigningKeyID: "key-one", Signature: "signature",
		},
		StorageConfirmations: []validationStorageConfirm{{Location: "node-one", Root: proofDigest("storage"), At: now}},
	}
	request.Submission.StorageRoot, _ = validationStorageRoot(request.Submission.Objects)
	request.StorageConfirmations[0].Root = request.Submission.StorageRoot
	request.ValidationCertificate = validationProofCertificate{
		SchemaVersion: "knirv.validation-certificate.v1", ValidatorID: "validator-one", ValidatedAt: now,
		ProofRoot: request.Submission.ProofRoot, CommitSHA256: request.Submission.Git.RawSHA256,
		PolicyHash: request.Submission.PolicyHash,
	}
	claimBytes, _ := json.Marshal(request.ValidationCertificate)
	request.ValidationCertificate.CertificateHash = sha256Digest(claimBytes)
	return request
}

func proofDigest(seed string) string { return sha256Digest([]byte(seed)) }

func proofHex(character string, length int) string {
	result := ""
	for len(result) < length {
		result += character
	}
	return result[:length]
}
