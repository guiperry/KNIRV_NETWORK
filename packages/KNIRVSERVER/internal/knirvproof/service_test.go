package knirvproof

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	dveevidence "knirv-server/internal/dveevidence"
)

type verifierFunc func(context.Context, ProofSubmission, *FileStore) (*ValidationCertificate, error)

func (fn verifierFunc) Verify(ctx context.Context, submission ProofSubmission, store *FileStore) (*ValidationCertificate, error) {
	return fn(ctx, submission, store)
}

type minterFunc func(context.Context, MintRequest) (*ChainReceipt, error)

func (fn minterFunc) Mint(ctx context.Context, request MintRequest) (*ChainReceipt, error) {
	return fn(ctx, request)
}

func TestFileStoreRejectsCiphertextCIDMismatch(t *testing.T) {
	store := newTestStore(t)
	expected := HashBytes([]byte("expected"))
	if _, _, err := store.PutObject(context.Background(), expected, bytes.NewBufferString("different")); err == nil {
		t.Fatal("PutObject accepted content with a different digest")
	}
	if exists, _, err := store.HasObject(expected); err != nil || exists {
		t.Fatalf("mismatched object persisted: exists=%v err=%v", exists, err)
	}
}

func TestProofLifecycleRequiresFinalChainReceipt(t *testing.T) {
	store := newTestStore(t)
	submission := putTestSubmission(t, store, "project-one", "a")
	verifier := verifierFunc(func(_ context.Context, submitted ProofSubmission, _ *FileStore) (*ValidationCertificate, error) {
		return testCertificate(submitted), nil
	})
	final := false
	minter := minterFunc(func(_ context.Context, _ MintRequest) (*ChainReceipt, error) {
		return &ChainReceipt{
			SchemaVersion: SchemaMint, TransactionID: "tx-1", BlockID: conditionalString(final, "block-1"),
			Final: final, FinalizedAt: conditionalTime(final),
		}, nil
	})
	service, err := NewService(store, verifier, LocalReplicator{Location: "node-a"}, minter, 1)
	if err != nil {
		t.Fatal(err)
	}
	response, created, err := service.Submit(submission)
	if err != nil || !created {
		t.Fatalf("Submit() created=%v err=%v", created, err)
	}
	if err := service.Process(context.Background(), response.OperationID); !errors.Is(err, ErrDependencyUnavailable) {
		t.Fatalf("Process() before finality error = %v", err)
	}
	operation, err := store.GetOperation(response.OperationID)
	if err != nil {
		t.Fatal(err)
	}
	if operation.Status != StatusFinalizing {
		t.Fatalf("status before finality = %s, want %s", operation.Status, StatusFinalizing)
	}
	if operation.Status == StatusCertified {
		t.Fatal("operation certified before a final chain receipt")
	}
	if _, err := service.PublicProof(submission.ProjectID, submission.Git.RawSHA256); err != nil {
		t.Fatalf("private commit index disappeared: %v", err)
	}

	final = true
	if err := service.Process(context.Background(), response.OperationID); err != nil {
		t.Fatalf("Process() after finality: %v", err)
	}
	operation, err = store.GetOperation(response.OperationID)
	if err != nil {
		t.Fatal(err)
	}
	if operation.Status != StatusCertified {
		t.Fatalf("final status = %s, want %s", operation.Status, StatusCertified)
	}
	public, err := service.PublicProof(submission.ProjectID, submission.Git.RawSHA256)
	if err != nil {
		t.Fatal(err)
	}
	if public.Receipt == nil || !public.Receipt.Final {
		t.Fatal("certified public proof has no final receipt")
	}

	reopened, err := NewFileStore(store.root, store.maxObjectBytes)
	if err != nil {
		t.Fatal(err)
	}
	restored, err := reopened.GetOperation(response.OperationID)
	if err != nil {
		t.Fatalf("operation did not survive restart: %v", err)
	}
	if restored.Status != StatusCertified {
		t.Fatalf("restored status = %s", restored.Status)
	}
}

func TestSubmitIsIdempotentAndRejectsCommitRebinding(t *testing.T) {
	store := newTestStore(t)
	service, err := NewService(store, nil, nil, nil, 1)
	if err != nil {
		t.Fatal(err)
	}
	first := putTestSubmission(t, store, "project-one", "first")
	response, created, err := service.Submit(first)
	if err != nil || !created {
		t.Fatalf("first Submit() created=%v err=%v", created, err)
	}
	retry, created, err := service.Submit(first)
	if err != nil || created {
		t.Fatalf("retry Submit() created=%v err=%v", created, err)
	}
	if retry.OperationID != response.OperationID {
		t.Fatalf("retry operation = %s, want %s", retry.OperationID, response.OperationID)
	}

	conflict := putTestSubmission(t, store, "project-one", "second")
	conflict.Git = first.Git
	if _, _, err := service.Submit(conflict); !errors.Is(err, ErrConflict) {
		t.Fatalf("commit rebind error = %v, want conflict", err)
	}
}

func TestSubmissionRejectsMissingOrUnreservedObjects(t *testing.T) {
	store := newTestStore(t)
	service, err := NewService(store, nil, nil, nil, 1)
	if err != nil {
		t.Fatal(err)
	}
	submission := testSubmission("project-one", []byte("ciphertext"), "missing")
	if _, _, err := service.Submit(submission); err == nil {
		t.Fatal("Submit accepted an unreserved, missing object")
	}
	if err := store.ReserveObject(submission.ProjectID, submission.Objects[0].CID); err != nil {
		t.Fatal(err)
	}
	if _, _, err := service.Submit(submission); err == nil {
		t.Fatal("Submit accepted a reserved but missing object")
	}
}

func newTestStore(t *testing.T) *FileStore {
	t.Helper()
	store, err := NewFileStore(t.TempDir(), 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	return store
}

func putTestSubmission(t *testing.T, store *FileStore, projectID, seed string) ProofSubmission {
	t.Helper()
	ciphertext := []byte("encrypted-" + seed)
	submission := testSubmission(projectID, ciphertext, seed)
	if err := store.ReserveObject(projectID, submission.Objects[0].CID); err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.PutObject(context.Background(), submission.Objects[0].CID, bytes.NewReader(ciphertext)); err != nil {
		t.Fatal(err)
	}
	return submission
}

func testSubmission(projectID string, ciphertext []byte, seed string) ProofSubmission {
	object := ObjectRef{CID: HashBytes(ciphertext), Size: int64(len(ciphertext))}
	storageRoot, _ := StorageRoot([]ObjectRef{object})
	digest := HashBytes([]byte("commit-" + seed))
	digestHexValue, _ := digestHex(digest)
	return ProofSubmission{
		SchemaVersion: SchemaProofSubmission, BundleSchema: dveevidence.SchemaVersion,
		ProjectID: projectID, SessionID: "session-" + seed,
		ProofRoot: HashBytes([]byte("proof-" + seed)), StorageRoot: storageRoot,
		EncryptedManifestCID: object.CID, Objects: []ObjectRef{object},
		KeyEnvelopes: []KeyEnvelope{
			{RecipientID: "owner", KeyID: "owner-key", Algorithm: "x25519-xchacha20poly1305", WrappedKey: "owner-wrapped"},
			{RecipientID: "validator", KeyID: "validator-key", Algorithm: "x25519-xchacha20poly1305", WrappedKey: "validator-wrapped"},
		},
		RepositoryFingerprint: "repo:" + seed,
		Git:                   GitCommit{ObjectFormat: "sha256", OID: digestHexValue, RawSHA256: digest, TreeOID: repeatHex("1", 64)},
		Workspace: WorkspaceBinding{
			BaseHash: HashBytes([]byte("base-" + seed)), FinalHash: HashBytes([]byte("final-" + seed)),
			DiffHash: HashBytes([]byte("diff-" + seed)),
		},
		PolicyHash: HashBytes([]byte("policy-" + seed)), SignerID: "user-1", SigningKeyID: "signing-key",
		Signature: "opaque-signature",
	}
}

func testCertificate(submission ProofSubmission) *ValidationCertificate {
	return &ValidationCertificate{
		SchemaVersion: "knirv.validation-certificate.v1", CertificateHash: HashBytes([]byte("certificate")),
		ValidatorID: "validator-1", ValidatedAt: time.Now().UTC(), ProofRoot: submission.ProofRoot,
		CommitSHA256: submission.Git.RawSHA256, PolicyHash: submission.PolicyHash,
	}
}

func repeatHex(value string, count int) string {
	result := ""
	for len(result) < count {
		result += value
	}
	return result[:count]
}

func conditionalString(condition bool, value string) string {
	if condition {
		return value
	}
	return ""
}

func conditionalTime(condition bool) time.Time {
	if condition {
		return time.Now().UTC()
	}
	return time.Time{}
}
