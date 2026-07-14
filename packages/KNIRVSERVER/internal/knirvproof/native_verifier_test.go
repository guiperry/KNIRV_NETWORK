package knirvproof

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"testing"
	"time"

	dveevidence "knirv-server/internal/dveevidence"
)

func TestNativeVerifierDecryptsAndValidatesCommitProof(t *testing.T) {
	store := newTestStore(t)
	dek := bytes.Repeat([]byte{0x42}, 32)
	submission, ciphertext, publicKey := buildNativeProof(t, dek)
	if err := store.ReserveObject(submission.ProjectID, submission.EncryptedManifestCID); err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.PutObject(context.Background(), submission.EncryptedManifestCID, bytes.NewReader(ciphertext)); err != nil {
		t.Fatal(err)
	}
	verifier := &NativeVerifier{
		DEKs: DEKResolverFunc(func(context.Context, ProofSubmission) ([]byte, error) {
			return append([]byte(nil), dek...), nil
		}),
		SigningKeys: func(keyID string) (ed25519.PublicKey, bool) {
			return publicKey, keyID == "key-1"
		},
		ValidatorID: "validator-1",
		Now:         func() time.Time { return time.Unix(1700000000, 0) },
	}
	certificate, err := verifier.Verify(context.Background(), submission, store)
	if err != nil {
		t.Fatalf("Verify(): %v", err)
	}
	if certificate.ProofRoot != submission.ProofRoot || certificate.CommitSHA256 != submission.Git.RawSHA256 {
		t.Fatalf("certificate does not bind submission: %+v", certificate)
	}
	if _, err := NormalizeSHA256(certificate.CertificateHash); err != nil {
		t.Fatalf("invalid certificate hash: %v", err)
	}
}

func TestNativeVerifierRejectsWrongEncryptionKey(t *testing.T) {
	store := newTestStore(t)
	dek := bytes.Repeat([]byte{0x42}, 32)
	submission, ciphertext, publicKey := buildNativeProof(t, dek)
	if err := store.ReserveObject(submission.ProjectID, submission.EncryptedManifestCID); err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.PutObject(context.Background(), submission.EncryptedManifestCID, bytes.NewReader(ciphertext)); err != nil {
		t.Fatal(err)
	}
	verifier := &NativeVerifier{
		DEKs: DEKResolverFunc(func(context.Context, ProofSubmission) ([]byte, error) {
			return bytes.Repeat([]byte{0x24}, 32), nil
		}),
		SigningKeys: func(string) (ed25519.PublicKey, bool) { return publicKey, true },
		ValidatorID: "validator-1",
	}
	if _, err := verifier.Verify(context.Background(), submission, store); err == nil {
		t.Fatal("Verify accepted a manifest encrypted with a different key")
	}
}

func buildNativeProof(t *testing.T, dek []byte) (ProofSubmission, []byte, ed25519.PublicKey) {
	t.Helper()
	seed := bytes.Repeat([]byte{0x17}, ed25519.SeedSize)
	privateKey := ed25519.NewKeyFromSeed(seed)
	publicKey := privateKey.Public().(ed25519.PublicKey)
	signer, err := dveevidence.SignerFromSeed("key-1", seed)
	if err != nil {
		t.Fatal(err)
	}
	policy := dveevidence.Policy{AllowedCommands: []string{"go test"}, DeniedCommands: []string{"rm -rf"}, AllowNetwork: false}
	policyBytes, _ := json.Marshal(policy)
	policyHash := HashBytes(policyBytes)
	events, eventRoot, err := dveevidence.BuildEventLog([]dveevidence.Event{{Timestamp: "2026-07-13T12:00:00Z", Type: "command"}})
	if err != nil {
		t.Fatal(err)
	}
	artifactHash := HashBytes([]byte("artifact"))
	bundle := dveevidence.Bundle{
		SchemaVersion: dveevidence.SchemaVersion, SessionID: "session-one", DVEID: "dve-one",
		UserID: "user-one", ProjectID: "project-one", StartedAt: "2026-07-13T11:00:00Z",
		CompletedAt: "2026-07-13T12:00:00Z", WorkspaceBaseHash: HashBytes([]byte("base")),
		WorkspaceFinalHash: HashBytes([]byte("final")), PolicyHash: policyHash,
		Artifacts:    []dveevidence.ArtifactRef{{Name: "report", Class: "report", Hash: artifactHash, Size: 8}},
		EventLogRoot: eventRoot, ArtifactMerkleRoot: dveevidence.MerkleRoot([]string{artifactHash}),
	}
	if err := signer.Sign(&bundle); err != nil {
		t.Fatal(err)
	}
	treeOID := repeatHex("a", 40)
	commitContent := []byte("tree " + treeOID + "\nauthor Test <test@example.com> 1700000000 +0000\ncommitter Test <test@example.com> 1700000000 +0000\n\nTest commit\n\nKNIRV-Project: project-one\nKNIRV-Session: session-one\n")
	rawCommit := append([]byte("commit "+itoa(len(commitContent))+"\x00"), commitContent...)
	gitOID := sha1.Sum(rawCommit)
	diff := []byte("diff --git a/a b/a\n")
	git := GitCommit{
		ObjectFormat: "sha1", OID: hex.EncodeToString(gitOID[:]), RawSHA256: HashBytes(rawCommit),
		TreeOID: treeOID,
	}
	workspace := WorkspaceBinding{BaseHash: bundle.WorkspaceBaseHash, FinalHash: bundle.WorkspaceFinalHash, DiffHash: HashBytes(diff)}
	manifest := ProofManifest{
		SchemaVersion: SchemaProofManifest, ProjectID: bundle.ProjectID, SessionID: bundle.SessionID,
		RepositoryFingerprint: "repo:one", Git: git, GitCommitObject: rawCommit, GitDiff: diff,
		Workspace: workspace, Policy: policy, Bundle: bundle,
		Evidence: dveevidence.Evidence{Events: events, ArtifactHashes: []string{artifactHash}},
	}
	manifestBytes, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	submission := ProofSubmission{
		SchemaVersion: SchemaProofSubmission, BundleSchema: dveevidence.SchemaVersion,
		ProjectID: bundle.ProjectID, SessionID: bundle.SessionID,
		ProofRoot: HashBytes(manifestBytes), RepositoryFingerprint: manifest.RepositoryFingerprint,
		Git: git, Workspace: workspace, PolicyHash: policyHash, SignerID: bundle.UserID, SigningKeyID: "key-1",
		KeyEnvelopes: []KeyEnvelope{
			{RecipientID: "owner", KeyID: "owner-key", Algorithm: "x25519-xchacha20poly1305", WrappedKey: "owner-wrapped"},
			{RecipientID: "validator", KeyID: "validator-key", Algorithm: "x25519-xchacha20poly1305", WrappedKey: "validator-wrapped"},
		},
	}
	nonce := bytes.Repeat([]byte{0x33}, 24)
	ciphertext, err := EncryptProofManifest(dek, submission, manifestBytes, nonce)
	if err != nil {
		t.Fatal(err)
	}
	object := ObjectRef{CID: HashBytes(ciphertext), Size: int64(len(ciphertext))}
	submission.EncryptedManifestCID = object.CID
	submission.Objects = []ObjectRef{object}
	submission.StorageRoot, err = StorageRoot(submission.Objects)
	if err != nil {
		t.Fatal(err)
	}
	signingMessage, err := SubmissionSigningMessage(submission)
	if err != nil {
		t.Fatal(err)
	}
	submission.Signature = base64.StdEncoding.EncodeToString(ed25519.Sign(privateKey, signingMessage))
	return submission, ciphertext, publicKey
}

func itoa(value int) string {
	if value == 0 {
		return "0"
	}
	result := make([]byte, 0, 20)
	for value > 0 {
		result = append(result, byte('0'+value%10))
		value /= 10
	}
	for left, right := 0, len(result)-1; left < right; left, right = left+1, right-1 {
		result[left], result[right] = result[right], result[left]
	}
	return string(result)
}
