package knirvproof

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

var ErrDependencyUnavailable = errors.New("proof dependency unavailable")

type Verifier interface {
	Verify(context.Context, ProofSubmission, *FileStore) (*ValidationCertificate, error)
}

type Replicator interface {
	Replicate(context.Context, ProofSubmission, *FileStore) ([]StorageConfirmation, error)
}

type Minter interface {
	Mint(context.Context, MintRequest) (*ChainReceipt, error)
}

type Service struct {
	store            *FileStore
	verifier         Verifier
	replicator       Replicator
	minter           Minter
	requiredReplicas int
	now              func() time.Time
	mu               sync.Mutex
	submitMu         sync.Mutex
	processing       map[string]struct{}
}

func NewService(store *FileStore, verifier Verifier, replicator Replicator, minter Minter, requiredReplicas int) (*Service, error) {
	if store == nil {
		return nil, fmt.Errorf("proof store is required")
	}
	if requiredReplicas < 1 {
		return nil, fmt.Errorf("at least one replica is required")
	}
	return &Service{
		store:            store,
		verifier:         verifier,
		replicator:       replicator,
		minter:           minter,
		requiredReplicas: requiredReplicas,
		now:              func() time.Time { return time.Now().UTC() },
		processing:       make(map[string]struct{}),
	}, nil
}

func (s *Service) Store() *FileStore { return s.store }

func (s *Service) Batch(request ObjectBatchRequest) (*ObjectBatchResponse, error) {
	if err := validateProjectID(request.ProjectID); err != nil {
		return nil, err
	}
	response := &ObjectBatchResponse{}
	seen := make(map[string]struct{}, len(request.Objects))
	for _, object := range request.Objects {
		cid, err := NormalizeSHA256(object.CID)
		if err != nil {
			return nil, err
		}
		if object.Size < 0 {
			return nil, fmt.Errorf("negative object size")
		}
		if _, duplicate := seen[cid]; duplicate {
			return nil, fmt.Errorf("duplicate object %s", cid)
		}
		seen[cid] = struct{}{}
		if err := s.store.ReserveObject(request.ProjectID, cid); err != nil {
			return nil, err
		}
		exists, size, err := s.store.HasObject(cid)
		if err != nil {
			return nil, err
		}
		if exists && size != object.Size {
			return nil, fmt.Errorf("%w: object %s size is %d, descriptor says %d", ErrConflict, cid, size, object.Size)
		}
		if !exists {
			digest, _ := digestHex(cid)
			response.Missing = append(response.Missing, cid)
			response.Uploads = append(response.Uploads, UploadTarget{
				CID: cid, Method: "PUT", Path: "/api/v1/knirv/cas/objects/" + digest + "?project_id=" + request.ProjectID,
			})
		}
	}
	return response, nil
}

func (s *Service) Submit(submission ProofSubmission) (*SubmitResponse, bool, error) {
	s.submitMu.Lock()
	defer s.submitMu.Unlock()

	normalized, err := s.validateSubmission(submission)
	if err != nil {
		return nil, false, err
	}
	if existing, err := s.store.GetProof(normalized.ProjectID, normalized.ProofRoot); err == nil {
		same, compareErr := sameSubmission(existing.Submission, normalized)
		if compareErr != nil {
			return nil, false, compareErr
		}
		if !same {
			return nil, false, fmt.Errorf("%w: proof root already has a different descriptor", ErrConflict)
		}
		return &SubmitResponse{OperationID: existing.OperationID, ProofRoot: normalized.ProofRoot, Status: existing.Status}, false, nil
	} else if !errors.Is(err, ErrNotFound) {
		return nil, false, err
	}

	for _, object := range normalized.Objects {
		reserved, err := s.store.ProjectHasObject(normalized.ProjectID, object.CID)
		if err != nil {
			return nil, false, err
		}
		if !reserved {
			return nil, false, fmt.Errorf("ciphertext object %s is not reserved for project", object.CID)
		}
		exists, size, err := s.store.HasObject(object.CID)
		if err != nil {
			return nil, false, err
		}
		if !exists {
			return nil, false, fmt.Errorf("missing ciphertext object %s", object.CID)
		}
		if size != object.Size {
			return nil, false, fmt.Errorf("object %s size mismatch", object.CID)
		}
	}

	operationID, err := newOperationID()
	if err != nil {
		return nil, false, err
	}
	now := s.now()
	operation := &Operation{
		ID: operationID, ProjectID: normalized.ProjectID, ProofRoot: normalized.ProofRoot,
		Status: StatusUploaded, Retryable: true, CreatedAt: now, UpdatedAt: now,
	}
	record := &ProofRecord{
		Submission: normalized, OperationID: operationID, Status: StatusUploaded,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := s.store.BindCommit(normalized.ProjectID, normalized.Git.RawSHA256, normalized.ProofRoot); err != nil {
		return nil, false, err
	}
	if err := s.store.SaveOperation(operation); err != nil {
		return nil, false, err
	}
	if err := s.store.SaveProof(record); err != nil {
		return nil, false, err
	}
	return &SubmitResponse{OperationID: operationID, ProofRoot: normalized.ProofRoot, Status: StatusUploaded}, true, nil
}

func (s *Service) validateSubmission(submission ProofSubmission) (ProofSubmission, error) {
	if submission.SchemaVersion != SchemaProofSubmission {
		return submission, fmt.Errorf("unsupported schema version %q", submission.SchemaVersion)
	}
	if submission.BundleSchema != "dve.bundle.v1" {
		return submission, fmt.Errorf("unsupported bundle schema %q", submission.BundleSchema)
	}
	if err := validateProjectID(submission.ProjectID); err != nil {
		return submission, err
	}
	if !safeIDPattern.MatchString(submission.SessionID) {
		return submission, fmt.Errorf("invalid session id")
	}
	var err error
	if submission.ProofRoot, err = NormalizeSHA256(submission.ProofRoot); err != nil {
		return submission, fmt.Errorf("proof root: %w", err)
	}
	if submission.StorageRoot, err = NormalizeSHA256(submission.StorageRoot); err != nil {
		return submission, fmt.Errorf("storage root: %w", err)
	}
	if submission.EncryptedManifestCID, err = NormalizeSHA256(submission.EncryptedManifestCID); err != nil {
		return submission, fmt.Errorf("encrypted manifest cid: %w", err)
	}
	if submission.Git.RawSHA256, err = NormalizeSHA256(submission.Git.RawSHA256); err != nil {
		return submission, fmt.Errorf("git raw sha256: %w", err)
	}
	if submission.PolicyHash, err = NormalizeSHA256(submission.PolicyHash); err != nil {
		return submission, fmt.Errorf("policy hash: %w", err)
	}
	if submission.EventBundleRoot != "" {
		if submission.EventBundleRoot, err = NormalizeSHA256(submission.EventBundleRoot); err != nil {
			return submission, fmt.Errorf("event bundle root: %w", err)
		}
	}
	for label, hash := range map[string]*string{
		"workspace base hash":  &submission.Workspace.BaseHash,
		"workspace final hash": &submission.Workspace.FinalHash,
		"workspace diff hash":  &submission.Workspace.DiffHash,
	} {
		if *hash, err = NormalizeSHA256(*hash); err != nil {
			return submission, fmt.Errorf("%s: %w", label, err)
		}
	}
	if strings.TrimSpace(submission.RepositoryFingerprint) == "" {
		return submission, fmt.Errorf("repository fingerprint is required")
	}
	if err := validateGitCommit(submission.Git); err != nil {
		return submission, err
	}
	if submission.SignerID == "" || submission.SigningKeyID == "" || submission.Signature == "" {
		return submission, fmt.Errorf("signer, signing key, and signature are required")
	}
	if len(submission.KeyEnvelopes) < 2 {
		return submission, fmt.Errorf("owner and validator key envelopes are required")
	}
	recipients := make(map[string]struct{}, len(submission.KeyEnvelopes))
	for _, envelope := range submission.KeyEnvelopes {
		if envelope.RecipientID == "" || envelope.KeyID == "" || envelope.Algorithm == "" || envelope.WrappedKey == "" {
			return submission, fmt.Errorf("key envelope is incomplete")
		}
		if _, exists := recipients[envelope.RecipientID]; exists {
			return submission, fmt.Errorf("duplicate key envelope recipient %q", envelope.RecipientID)
		}
		recipients[envelope.RecipientID] = struct{}{}
	}
	manifestFound := false
	for index := range submission.Objects {
		if submission.Objects[index].CID, err = NormalizeSHA256(submission.Objects[index].CID); err != nil {
			return submission, fmt.Errorf("object %d: %w", index, err)
		}
		if submission.Objects[index].CID == submission.EncryptedManifestCID {
			manifestFound = true
		}
	}
	if !manifestFound {
		return submission, fmt.Errorf("encrypted manifest is not in objects")
	}
	storageRoot, err := StorageRoot(submission.Objects)
	if err != nil {
		return submission, err
	}
	if storageRoot != submission.StorageRoot {
		return submission, fmt.Errorf("storage root mismatch: expected %s", storageRoot)
	}
	for index, related := range submission.RelatedProofs {
		submission.RelatedProofs[index], err = NormalizeSHA256(related)
		if err != nil {
			return submission, fmt.Errorf("related proof %d: %w", index, err)
		}
	}
	return submission, nil
}

func validateGitCommit(commit GitCommit) error {
	length := 0
	switch commit.ObjectFormat {
	case "sha1":
		length = 40
	case "sha256":
		length = 64
	default:
		return fmt.Errorf("unsupported git object format %q", commit.ObjectFormat)
	}
	for label, value := range map[string]string{"oid": commit.OID, "tree oid": commit.TreeOID} {
		if !isHexLength(value, length) {
			return fmt.Errorf("invalid git %s", label)
		}
	}
	for _, parent := range commit.ParentOIDs {
		if !isHexLength(parent, length) {
			return fmt.Errorf("invalid git parent oid")
		}
	}
	return nil
}

func isHexLength(value string, length int) bool {
	if len(value) != length {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func sameSubmission(left, right ProofSubmission) (bool, error) {
	leftJSON, err := json.Marshal(left)
	if err != nil {
		return false, err
	}
	rightJSON, err := json.Marshal(right)
	if err != nil {
		return false, err
	}
	return string(leftJSON) == string(rightJSON), nil
}

func (s *Service) Process(ctx context.Context, operationID string) error {
	s.mu.Lock()
	if _, active := s.processing[operationID]; active {
		s.mu.Unlock()
		return nil
	}
	s.processing[operationID] = struct{}{}
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		delete(s.processing, operationID)
		s.mu.Unlock()
	}()

	operation, err := s.store.GetOperation(operationID)
	if err != nil {
		return err
	}
	record, err := s.store.GetProof(operation.ProjectID, operation.ProofRoot)
	if err != nil {
		return err
	}
	if operation.Status.Terminal() {
		return nil
	}
	operation.Attempts++
	operation.Retryable = true
	operation.LastError = ""
	operation.UpdatedAt = s.now()
	if err := s.store.SaveOperation(operation); err != nil {
		return err
	}

	if record.ValidationCertificate == nil {
		if err := s.transition(operation, record, StatusVerifying, "", true); err != nil {
			return err
		}
		if s.verifier == nil {
			return s.retryable(operation, record, "validator is not configured")
		}
		certificate, err := s.verifier.Verify(ctx, record.Submission, s.store)
		if err != nil {
			if errors.Is(err, ErrDependencyUnavailable) {
				return s.retryable(operation, record, err.Error())
			}
			_ = s.transition(operation, record, StatusRejected, err.Error(), false)
			return err
		}
		if err := validateCertificate(certificate, record.Submission); err != nil {
			_ = s.transition(operation, record, StatusRejected, err.Error(), false)
			return err
		}
		record.ValidationCertificate = certificate
		if err := s.transition(operation, record, StatusReplicating, "", true); err != nil {
			return err
		}
	}

	if len(record.StorageConfirmations) < s.requiredReplicas {
		if err := s.transition(operation, record, StatusReplicating, "", true); err != nil {
			return err
		}
		if s.replicator == nil {
			return s.retryable(operation, record, "storage replicator is not configured")
		}
		confirmations, err := s.replicator.Replicate(ctx, record.Submission, s.store)
		if err != nil {
			return s.retryable(operation, record, err.Error())
		}
		record.StorageConfirmations = uniqueConfirmations(confirmations, record.Submission.StorageRoot)
		if len(record.StorageConfirmations) < s.requiredReplicas {
			return s.retryable(operation, record, fmt.Sprintf("need %d storage replicas, have %d", s.requiredReplicas, len(record.StorageConfirmations)))
		}
		if err := s.transition(operation, record, StatusMintPending, "", true); err != nil {
			return err
		}
	}

	if s.minter == nil {
		return s.retryable(operation, record, "KNIRVCHAIN minter is not configured")
	}
	if err := s.transition(operation, record, StatusFinalizing, "", true); err != nil {
		return err
	}
	receipt, err := s.minter.Mint(ctx, MintRequest{
		SchemaVersion: SchemaMint, Submission: record.Submission,
		ValidationCertificate: *record.ValidationCertificate,
		StorageConfirmations:  record.StorageConfirmations,
	})
	if err != nil {
		if errors.Is(err, ErrConflict) {
			_ = s.transition(operation, record, StatusRejected, err.Error(), false)
			return err
		}
		return s.retryable(operation, record, err.Error())
	}
	if receipt == nil || receipt.TransactionID == "" {
		return s.retryable(operation, record, "KNIRVCHAIN returned no transaction receipt")
	}
	record.ChainReceipt = receipt
	if !receipt.Final || receipt.BlockID == "" || receipt.FinalizedAt.IsZero() {
		return s.retryable(operation, record, "KNIRVCHAIN transaction is not final")
	}
	return s.transition(operation, record, StatusCertified, "", false)
}

func (s *Service) retryable(operation *Operation, record *ProofRecord, message string) error {
	if err := s.transition(operation, record, operation.Status, message, true); err != nil {
		return err
	}
	return fmt.Errorf("%w: %s", ErrDependencyUnavailable, message)
}

func (s *Service) transition(operation *Operation, record *ProofRecord, status OperationStatus, message string, retryable bool) error {
	now := s.now()
	operation.Status = status
	operation.LastError = message
	operation.Retryable = retryable
	operation.UpdatedAt = now
	record.Status = status
	record.UpdatedAt = now
	if err := s.store.SaveProof(record); err != nil {
		return err
	}
	return s.store.SaveOperation(operation)
}

func validateCertificate(certificate *ValidationCertificate, submission ProofSubmission) error {
	if certificate == nil {
		return fmt.Errorf("validator returned no certificate")
	}
	if certificate.SchemaVersion == "" || certificate.ValidatorID == "" || certificate.ValidatedAt.IsZero() {
		return fmt.Errorf("validation certificate is incomplete")
	}
	if normalized, err := NormalizeSHA256(certificate.CertificateHash); err != nil {
		return fmt.Errorf("certificate hash: %w", err)
	} else {
		certificate.CertificateHash = normalized
	}
	if certificate.ProofRoot != submission.ProofRoot || certificate.CommitSHA256 != submission.Git.RawSHA256 || certificate.PolicyHash != submission.PolicyHash {
		return fmt.Errorf("validation certificate does not bind the submitted proof")
	}
	if certificate.EventBundleRoot != submission.EventBundleRoot {
		return fmt.Errorf("validation certificate does not bind the submitted event bundle root")
	}
	return nil
}

func uniqueConfirmations(confirmations []StorageConfirmation, storageRoot string) []StorageConfirmation {
	result := make([]StorageConfirmation, 0, len(confirmations))
	seen := make(map[string]struct{}, len(confirmations))
	for _, confirmation := range confirmations {
		if confirmation.Location == "" || confirmation.Root != storageRoot || confirmation.At.IsZero() {
			continue
		}
		if _, exists := seen[confirmation.Location]; exists {
			continue
		}
		seen[confirmation.Location] = struct{}{}
		result = append(result, confirmation)
	}
	return result
}

func (s *Service) AddEnvelope(projectID, proofRoot string, envelope KeyEnvelope) (*ProofRecord, error) {
	if envelope.RecipientID == "" || envelope.KeyID == "" || envelope.Algorithm == "" || envelope.WrappedKey == "" {
		return nil, fmt.Errorf("key envelope is incomplete")
	}
	record, err := s.store.GetProof(projectID, proofRoot)
	if err != nil {
		return nil, err
	}
	for _, existing := range append(record.Submission.KeyEnvelopes, record.AdditionalEnvelopes...) {
		if existing.RecipientID == envelope.RecipientID {
			if existing.KeyID == envelope.KeyID && existing.Algorithm == envelope.Algorithm && existing.WrappedKey == envelope.WrappedKey {
				return record, nil
			}
			return nil, fmt.Errorf("%w: recipient already has a different envelope", ErrConflict)
		}
	}
	record.AdditionalEnvelopes = append(record.AdditionalEnvelopes, envelope)
	record.UpdatedAt = s.now()
	if err := s.store.SaveProof(record); err != nil {
		return nil, err
	}
	return record, nil
}

func (s *Service) PublicProof(projectID, commitDigest string) (*PublicProof, error) {
	record, err := s.store.ProofForCommit(projectID, commitDigest)
	if err != nil {
		return nil, err
	}
	return &PublicProof{
		ProjectID: record.Submission.ProjectID, SessionID: record.Submission.SessionID,
		RepositoryFingerprint: record.Submission.RepositoryFingerprint, Git: record.Submission.Git,
		ProofRoot: record.Submission.ProofRoot, StorageRoot: record.Submission.StorageRoot,
		PolicyHash: record.Submission.PolicyHash, Status: record.Status,
		Certificate: record.ValidationCertificate, Receipt: record.ChainReceipt,
		EventBundleRoot: record.Submission.EventBundleRoot,
	}, nil
}

func (s *Service) Resume(ctx context.Context) error {
	operations, err := s.store.ListOperations()
	if err != nil {
		return err
	}
	for _, operation := range operations {
		if operation.Status.Terminal() || !operation.Retryable {
			continue
		}
		operationID := operation.ID
		go func() { _ = s.Process(ctx, operationID) }()
	}
	return nil
}
