package knirvproof

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha1" // Git's legacy object format, not used as a security digest.
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"golang.org/x/crypto/chacha20poly1305"

	dveevidence "knirv-server/internal/dveevidence"
)

const (
	SchemaProofManifest     = "knirv.proof-manifest.v1"
	SchemaValidationCert    = "knirv.validation-certificate.v1"
	encryptedObjectMagic    = "KNIRV1"
	defaultMaxManifestBytes = 32 << 20
)

type ProofManifest struct {
	SchemaVersion         string               `json:"schema_version"`
	ProjectID             string               `json:"project_id"`
	SessionID             string               `json:"session_id"`
	RepositoryFingerprint string               `json:"repository_fingerprint"`
	Git                   GitCommit            `json:"git"`
	GitCommitObject       []byte               `json:"git_commit_object"`
	GitDiff               []byte               `json:"git_diff"`
	Workspace             WorkspaceBinding     `json:"workspace"`
	Policy                dveevidence.Policy   `json:"policy"`
	Bundle                dveevidence.Bundle   `json:"bundle"`
	Evidence              dveevidence.Evidence `json:"evidence"`
	RelatedProofs         []string             `json:"related_proofs,omitempty"`
}

type DEKResolver interface {
	ResolveDEK(context.Context, ProofSubmission) ([]byte, error)
}

type DEKResolverFunc func(context.Context, ProofSubmission) ([]byte, error)

func (fn DEKResolverFunc) ResolveDEK(ctx context.Context, submission ProofSubmission) ([]byte, error) {
	return fn(ctx, submission)
}

// EventBundleFetcher independently confirms a KNIRVCHAIN event-bundle NFT
// referenced by a PermissionDecision.EventBundleHash actually exists on
// chain (chain_refactor.md Open Decision #2), rather than trusting the
// CLI-supplied hash under the submission's signature alone.
type EventBundleFetcher interface {
	FetchBundleHash(ctx context.Context, eventID string) (hash string, ok bool, err error)
}

type NativeVerifier struct {
	DEKs             DEKResolver
	SigningKeys      dveevidence.KeyResolver
	ValidatorID      string
	MaxManifestBytes int64
	Now              func() time.Time
	// EventBundles is optional: when nil, event-bundle existence is not
	// independently re-verified (the manifest-internal hash/root checks in
	// validateManifest still apply). Production deployments should always
	// set this once KNIRVCHAIN's event-bundle mint endpoint is reachable.
	EventBundles EventBundleFetcher
}

func (v *NativeVerifier) Verify(ctx context.Context, submission ProofSubmission, store *FileStore) (*ValidationCertificate, error) {
	if v == nil || v.DEKs == nil {
		return nil, fmt.Errorf("%w: validator key resolver is not configured", ErrDependencyUnavailable)
	}
	if v.SigningKeys == nil {
		return nil, fmt.Errorf("%w: signing key resolver is not configured", ErrDependencyUnavailable)
	}
	if v.ValidatorID == "" {
		return nil, fmt.Errorf("%w: validator identity is not configured", ErrDependencyUnavailable)
	}
	dek, err := v.DEKs.ResolveDEK(ctx, submission)
	if err != nil {
		return nil, fmt.Errorf("%w: unwrap validator key: %v", ErrDependencyUnavailable, err)
	}
	defer wipe(dek)
	if len(dek) != chacha20poly1305.KeySize {
		return nil, fmt.Errorf("validator returned an invalid data-encryption key")
	}

	file, encryptedSize, err := store.OpenObject(submission.EncryptedManifestCID)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	maxPlaintext := v.MaxManifestBytes
	if maxPlaintext <= 0 {
		maxPlaintext = defaultMaxManifestBytes
	}
	maximumEncrypted := maxPlaintext + int64(len(encryptedObjectMagic)+chacha20poly1305.NonceSizeX+chacha20poly1305.Overhead)
	if encryptedSize > maximumEncrypted {
		return nil, fmt.Errorf("encrypted proof manifest exceeds validation limit")
	}
	encrypted, err := io.ReadAll(io.LimitReader(file, maximumEncrypted+1))
	if err != nil {
		return nil, err
	}
	defer wipe(encrypted)
	plaintext, err := decryptProofManifest(dek, submission, encrypted)
	if err != nil {
		return nil, err
	}
	defer wipe(plaintext)
	if int64(len(plaintext)) > maxPlaintext {
		return nil, fmt.Errorf("proof manifest exceeds validation limit")
	}

	var manifest ProofManifest
	decoder := json.NewDecoder(bytes.NewReader(plaintext))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&manifest); err != nil {
		return nil, fmt.Errorf("decode proof manifest: %w", err)
	}
	if err := ensureJSONEOF(decoder); err != nil {
		return nil, err
	}
	canonical, err := json.Marshal(manifest)
	if err != nil {
		return nil, err
	}
	defer wipe(canonical)
	if HashBytes(canonical) != submission.ProofRoot {
		return nil, fmt.Errorf("plaintext proof root mismatch")
	}
	if err := validateManifest(manifest, submission, v.SigningKeys); err != nil {
		return nil, err
	}
	if v.EventBundles != nil {
		if err := verifyEventBundles(ctx, v.EventBundles, manifest.Bundle.PermissionDecisions); err != nil {
			return nil, err
		}
	}

	now := time.Now().UTC()
	if v.Now != nil {
		now = v.Now().UTC()
	}
	certificate := &ValidationCertificate{
		SchemaVersion: SchemaValidationCert, ValidatorID: v.ValidatorID, ValidatedAt: now,
		ProofRoot: submission.ProofRoot, CommitSHA256: submission.Git.RawSHA256,
		PolicyHash: submission.PolicyHash, EventBundleRoot: submission.EventBundleRoot,
	}
	certificateBytes, err := json.Marshal(certificate)
	if err != nil {
		return nil, err
	}
	certificate.CertificateHash = HashBytes(certificateBytes)
	wipe(certificateBytes)
	return certificate, nil
}

func ProofManifestAAD(submission ProofSubmission) []byte {
	return []byte(SchemaProofManifest + "|" + submission.ProjectID + "|" + submission.ProofRoot)
}

func EncryptProofManifest(dek []byte, submission ProofSubmission, plaintext []byte, nonce []byte) ([]byte, error) {
	aead, err := chacha20poly1305.NewX(dek)
	if err != nil {
		return nil, err
	}
	if len(nonce) != aead.NonceSize() {
		return nil, fmt.Errorf("nonce must be %d bytes", aead.NonceSize())
	}
	result := make([]byte, 0, len(encryptedObjectMagic)+len(nonce)+len(plaintext)+aead.Overhead())
	result = append(result, encryptedObjectMagic...)
	result = append(result, nonce...)
	result = aead.Seal(result, nonce, plaintext, ProofManifestAAD(submission))
	return result, nil
}

func decryptProofManifest(dek []byte, submission ProofSubmission, encrypted []byte) ([]byte, error) {
	aead, err := chacha20poly1305.NewX(dek)
	if err != nil {
		return nil, err
	}
	headerLength := len(encryptedObjectMagic) + aead.NonceSize()
	if len(encrypted) < headerLength+aead.Overhead() || string(encrypted[:len(encryptedObjectMagic)]) != encryptedObjectMagic {
		return nil, fmt.Errorf("invalid encrypted proof object")
	}
	nonce := encrypted[len(encryptedObjectMagic):headerLength]
	plaintext, err := aead.Open(nil, nonce, encrypted[headerLength:], ProofManifestAAD(submission))
	if err != nil {
		return nil, fmt.Errorf("authenticate proof manifest: %w", err)
	}
	return plaintext, nil
}

func validateManifest(manifest ProofManifest, submission ProofSubmission, resolver dveevidence.KeyResolver) error {
	if manifest.SchemaVersion != SchemaProofManifest {
		return fmt.Errorf("unsupported proof manifest schema %q", manifest.SchemaVersion)
	}
	if manifest.ProjectID != submission.ProjectID || manifest.SessionID != submission.SessionID || manifest.RepositoryFingerprint != submission.RepositoryFingerprint {
		return fmt.Errorf("proof manifest identity does not match submission")
	}
	if !equalJSON(manifest.Git, submission.Git) || !equalJSON(manifest.Workspace, submission.Workspace) || !equalStrings(manifest.RelatedProofs, submission.RelatedProofs) {
		return fmt.Errorf("proof manifest bindings do not match submission")
	}
	policyBytes, err := json.Marshal(manifest.Policy)
	if err != nil {
		return err
	}
	if HashBytes(policyBytes) != submission.PolicyHash {
		return fmt.Errorf("policy hash does not match the exact embedded policy")
	}
	if HashBytes(manifest.GitDiff) != submission.Workspace.DiffHash {
		return fmt.Errorf("git diff hash mismatch")
	}
	if err := validateRawGitCommit(manifest.GitCommitObject, submission); err != nil {
		return err
	}
	bundle := &manifest.Bundle
	if bundle.SchemaVersion != dveevidence.SchemaVersion || bundle.ProjectID != submission.ProjectID || bundle.SessionID != submission.SessionID {
		return fmt.Errorf("DVE bundle identity does not match submission")
	}
	if bundle.UserID != submission.SignerID || bundle.PolicyHash != submission.PolicyHash || bundle.WorkspaceBaseHash != submission.Workspace.BaseHash || bundle.WorkspaceFinalHash != submission.Workspace.FinalHash {
		return fmt.Errorf("DVE bundle bindings do not match submission")
	}
	if bundle.Signature == nil || bundle.Signature.KeyID != submission.SigningKeyID {
		return fmt.Errorf("DVE bundle signing key does not match submission")
	}
	verified, err := dveevidence.VerifySignature(bundle, resolver)
	if err != nil || !verified {
		if err == nil {
			err = fmt.Errorf("signature is not verifiable")
		}
		return fmt.Errorf("DVE bundle signature: %w", err)
	}
	if err := verifySubmissionSignature(submission, resolver); err != nil {
		return err
	}
	if err := dveevidence.VerifyEventLog(manifest.Evidence.Events, bundle.EventLogRoot); err != nil {
		return fmt.Errorf("event log: %w", err)
	}
	artifactHashes := make([]string, 0, len(bundle.Artifacts))
	for _, artifact := range bundle.Artifacts {
		artifactHashes = append(artifactHashes, artifact.Hash)
	}
	if !equalStrings(artifactHashes, manifest.Evidence.ArtifactHashes) {
		return fmt.Errorf("artifact evidence does not match bundle")
	}
	if dveevidence.MerkleRoot(artifactHashes) != bundle.ArtifactMerkleRoot {
		return fmt.Errorf("artifact Merkle root mismatch")
	}
	if violations := dveevidence.ReplayPolicy(bundle, &manifest.Policy); len(violations) > 0 {
		return fmt.Errorf("policy replay found %d violation(s)", len(violations))
	}
	if bundle.EventBundleRoot != submission.EventBundleRoot {
		return fmt.Errorf("event bundle root does not match submission")
	}
	if expectedRoot := expectedEventBundleRoot(bundle.PermissionDecisions); expectedRoot != bundle.EventBundleRoot {
		return fmt.Errorf("event bundle root does not match recorded decisions")
	}
	return nil
}

// expectedEventBundleRoot mirrors the CLI's dve.Bundle.EventBundleRoot
// construction exactly: empty when no decision minted an event bundle
// (chain_refactor.md Design Invariant #3 — "0 for sessions with no
// MCP/skill/capability/asset/credential involvement"), otherwise the Merkle
// root over the per-decision hashes in order. dveevidence.MerkleRoot's own
// empty-input behavior (a hash of nothing) is deliberately NOT used here —
// that would wrongly require every pre-existing, bundle-less session to
// carry a non-empty root.
func expectedEventBundleRoot(decisions []dveevidence.PermissionDecision) string {
	hashes := eventBundleHashes(decisions)
	if len(hashes) == 0 {
		return ""
	}
	return dveevidence.MerkleRoot(hashes)
}

// eventBundleHashes extracts the per-decision KNIRVCHAIN event-bundle hashes
// in decision order, matching the CLI's dve.Bundle.EventBundleRoot
// construction (chain_refactor.md §3.3) so both sides compute the identical
// Merkle root over the identical leaf set.
func eventBundleHashes(decisions []dveevidence.PermissionDecision) []string {
	out := make([]string, 0, len(decisions))
	for _, decision := range decisions {
		if decision.EventBundleHash != "" {
			out = append(out, decision.EventBundleHash)
		}
	}
	return out
}

// verifyEventBundles independently re-fetches every KNIRVCHAIN event-bundle
// NFT a submission's decisions reference and confirms the bundle actually
// exists with the claimed hash (chain_refactor.md Open Decision #2: "the
// verifier re-fetches the bundle from KNIRVCHAIN by ID to confirm. If the
// mint doesn't exist the commit is rejected until fixed"). A missing bundle
// is reported as ErrDependencyUnavailable (retryable — minting is
// asynchronous relative to the commit that references it); a hash mismatch
// is a hard validation failure.
// safeEventBundleID mirrors the CLI's proofwire.SafeEventBundleID exactly —
// both sides must derive the identical KNIRVCHAIN lookup key from a
// PermissionDecision.ID (see that function's doc for why the raw ID isn't
// used directly).
func safeEventBundleID(decisionID string) string {
	sum := sha256.Sum256([]byte(decisionID))
	return hex.EncodeToString(sum[:])
}

func verifyEventBundles(ctx context.Context, fetcher EventBundleFetcher, decisions []dveevidence.PermissionDecision) error {
	for _, decision := range decisions {
		if decision.EventBundleHash == "" {
			continue
		}
		hash, ok, err := fetcher.FetchBundleHash(ctx, safeEventBundleID(decision.ID))
		if err != nil {
			return fmt.Errorf("%w: fetch event bundle for decision %s: %v", ErrDependencyUnavailable, decision.ID, err)
		}
		if !ok {
			return fmt.Errorf("%w: event bundle for decision %s is not minted yet", ErrDependencyUnavailable, decision.ID)
		}
		if hash != decision.EventBundleHash {
			return fmt.Errorf("event bundle hash mismatch for decision %s", decision.ID)
		}
	}
	return nil
}

type submissionSigningClaim struct {
	SchemaVersion         string           `json:"schema_version"`
	BundleSchema          string           `json:"bundle_schema"`
	ProjectID             string           `json:"project_id"`
	SessionID             string           `json:"session_id"`
	ProofRoot             string           `json:"proof_root"`
	StorageRoot           string           `json:"storage_root"`
	EncryptedManifestCID  string           `json:"encrypted_manifest_cid"`
	Objects               []ObjectRef      `json:"objects"`
	KeyEnvelopes          []KeyEnvelope    `json:"key_envelopes"`
	RepositoryFingerprint string           `json:"repository_fingerprint"`
	Git                   GitCommit        `json:"git"`
	Workspace             WorkspaceBinding `json:"workspace"`
	PolicyHash            string           `json:"policy_hash"`
	SignerID              string           `json:"signer_id"`
	SigningKeyID          string           `json:"signing_key_id"`
	RelatedProofs         []string         `json:"related_proofs,omitempty"`
	EventBundleRoot       string           `json:"event_bundle_root,omitempty"`
}

func SubmissionSigningMessage(submission ProofSubmission) ([]byte, error) {
	return json.Marshal(submissionSigningClaim{
		SchemaVersion: submission.SchemaVersion, BundleSchema: submission.BundleSchema,
		ProjectID: submission.ProjectID, SessionID: submission.SessionID,
		ProofRoot: submission.ProofRoot, StorageRoot: submission.StorageRoot,
		EncryptedManifestCID: submission.EncryptedManifestCID, Objects: submission.Objects,
		KeyEnvelopes: submission.KeyEnvelopes, RepositoryFingerprint: submission.RepositoryFingerprint,
		Git: submission.Git, Workspace: submission.Workspace, PolicyHash: submission.PolicyHash,
		SignerID: submission.SignerID, SigningKeyID: submission.SigningKeyID, RelatedProofs: submission.RelatedProofs,
		EventBundleRoot: submission.EventBundleRoot,
	})
}

func verifySubmissionSignature(submission ProofSubmission, resolver dveevidence.KeyResolver) error {
	publicKey, ok := resolver(submission.SigningKeyID)
	if !ok {
		return fmt.Errorf("unknown submission signing key")
	}
	signature, err := base64.StdEncoding.DecodeString(submission.Signature)
	if err != nil || len(signature) != ed25519.SignatureSize {
		return fmt.Errorf("invalid submission signature encoding")
	}
	message, err := SubmissionSigningMessage(submission)
	if err != nil {
		return err
	}
	if !ed25519.Verify(publicKey, message, signature) {
		return fmt.Errorf("submission signature verification failed")
	}
	return nil
}

func validateRawGitCommit(raw []byte, submission ProofSubmission) error {
	prefixEnd := bytes.IndexByte(raw, 0)
	if prefixEnd < 0 {
		return fmt.Errorf("raw Git commit is missing object header")
	}
	header := string(raw[:prefixEnd])
	content := raw[prefixEnd+1:]
	if header != fmt.Sprintf("commit %d", len(content)) {
		return fmt.Errorf("raw Git commit header is invalid")
	}
	if HashBytes(raw) != submission.Git.RawSHA256 {
		return fmt.Errorf("raw Git commit SHA-256 mismatch")
	}
	var oid string
	switch submission.Git.ObjectFormat {
	case "sha1":
		sum := sha1.Sum(raw)
		oid = hex.EncodeToString(sum[:])
	case "sha256":
		sum := sha256.Sum256(raw)
		oid = hex.EncodeToString(sum[:])
	default:
		return fmt.Errorf("unsupported Git object format")
	}
	if oid != strings.ToLower(submission.Git.OID) {
		return fmt.Errorf("Git object id mismatch")
	}
	parts := bytes.SplitN(content, []byte("\n\n"), 2)
	if len(parts) != 2 {
		return fmt.Errorf("Git commit has no message")
	}
	var tree string
	parents := make([]string, 0)
	for _, line := range strings.Split(string(parts[0]), "\n") {
		switch {
		case strings.HasPrefix(line, "tree "):
			tree = strings.TrimPrefix(line, "tree ")
		case strings.HasPrefix(line, "parent "):
			parents = append(parents, strings.TrimPrefix(line, "parent "))
		}
	}
	if tree != submission.Git.TreeOID || !equalStrings(parents, submission.Git.ParentOIDs) {
		return fmt.Errorf("Git tree or parent binding mismatch")
	}
	trailers := parseTrailers(string(parts[1]))
	if trailers["KNIRV-Project"] != submission.ProjectID || trailers["KNIRV-Session"] != submission.SessionID {
		return fmt.Errorf("Git commit is missing matching KNIRV trailers")
	}
	return nil
}

func parseTrailers(message string) map[string]string {
	result := make(map[string]string)
	lines := strings.Split(strings.TrimSpace(message), "\n")
	for index := len(lines) - 1; index >= 0; index-- {
		line := strings.TrimSpace(lines[index])
		if line == "" {
			if len(result) > 0 {
				break
			}
			continue
		}
		key, value, found := strings.Cut(line, ":")
		if !found {
			if len(result) > 0 {
				break
			}
			continue
		}
		result[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	return result
}

func ensureJSONEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return fmt.Errorf("proof manifest must contain one JSON value")
	}
	return nil
}

func equalJSON(left, right any) bool {
	leftBytes, _ := json.Marshal(left)
	rightBytes, _ := json.Marshal(right)
	return bytes.Equal(leftBytes, rightBytes)
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func wipe(value []byte) {
	for index := range value {
		value[index] = 0
	}
}
