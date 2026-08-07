package blockchain

import (
	"bytes"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"KNIRVCHAIN/internal/utils"
)

const (
	validationProofMintSchema    = "validation_proof_mint.v1"
	validationProofReceiptSchema = "validation_proof_receipt.v1"
	validationProofKeyPrefix     = "validation_proof:proof:"
	validationCommitKeyPrefix    = "validation_proof:commit:"
)

var validationProofSafeID = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)

type validationProofMintRequest struct {
	SchemaVersion         string                     `json:"schema_version"`
	Submission            validationProofSubmission  `json:"submission"`
	ValidationCertificate validationProofCertificate `json:"validation_certificate"`
	StorageConfirmations  []validationStorageConfirm `json:"storage_confirmations"`
}

type validationProofSubmission struct {
	SchemaVersion         string                  `json:"schema_version"`
	BundleSchema          string                  `json:"bundle_schema"`
	ProjectID             string                  `json:"project_id"`
	SessionID             string                  `json:"session_id"`
	ProofRoot             string                  `json:"proof_root"`
	StorageRoot           string                  `json:"storage_root"`
	EncryptedManifestCID  string                  `json:"encrypted_manifest_cid"`
	Objects               []validationObjectRef   `json:"objects"`
	KeyEnvelopes          []validationKeyEnvelope `json:"key_envelopes"`
	RepositoryFingerprint string                  `json:"repository_fingerprint"`
	Git                   validationGit           `json:"git"`
	Workspace             validationWorkspace     `json:"workspace"`
	PolicyHash            string                  `json:"policy_hash"`
	SignerID              string                  `json:"signer_id"`
	SigningKeyID          string                  `json:"signing_key_id"`
	Signature             string                  `json:"signature"`
	RelatedProofs         []string                `json:"related_proofs,omitempty"`
	// EventBundleRoot is the Merkle root over every KNIRVCHAIN event-bundle
	// NFT hash minted for this session's decisions (chain_refactor.md §3.2/
	// §3.3) — the Validation chain's commit schema binding to the minted
	// bundle(s). Empty for sessions with no resolvable resource usage.
	EventBundleRoot string `json:"event_bundle_root,omitempty"`
}

type validationObjectRef struct {
	CID  string `json:"cid"`
	Size int64  `json:"size"`
}

type validationKeyEnvelope struct {
	RecipientID string `json:"recipient_id"`
	KeyID       string `json:"key_id"`
	Algorithm   string `json:"algorithm"`
	WrappedKey  string `json:"wrapped_key"`
}

type validationWorkspace struct {
	BaseHash  string `json:"base_hash"`
	FinalHash string `json:"final_hash"`
	DiffHash  string `json:"diff_hash"`
}

type validationGit struct {
	ObjectFormat string   `json:"object_format"`
	OID          string   `json:"oid"`
	RawSHA256    string   `json:"raw_sha256"`
	TreeOID      string   `json:"tree_oid"`
	ParentOIDs   []string `json:"parent_oids,omitempty"`
}

type validationProofCertificate struct {
	SchemaVersion   string    `json:"schema_version"`
	CertificateHash string    `json:"certificate_hash"`
	ValidatorID     string    `json:"validator_id"`
	ValidatedAt     time.Time `json:"validated_at"`
	ProofRoot       string    `json:"proof_root"`
	CommitSHA256    string    `json:"commit_sha256"`
	PolicyHash      string    `json:"policy_hash"`
	EventBundleRoot string    `json:"event_bundle_root,omitempty"`
}

type validationStorageConfirm struct {
	Location string    `json:"location"`
	Root     string    `json:"root"`
	At       time.Time `json:"at"`
}

type validationProofRecord struct {
	ProjectID     string    `json:"project_id"`
	CommitSHA256  string    `json:"commit_sha256"`
	ProofRoot     string    `json:"proof_root"`
	RequestHash   string    `json:"request_hash"`
	TransactionID string    `json:"transaction_id"`
	AcceptedAt    time.Time `json:"accepted_at"`
}

type validationProofReceipt struct {
	SchemaVersion string    `json:"schema_version"`
	TransactionID string    `json:"transaction_id"`
	BlockID       string    `json:"block_id"`
	Final         bool      `json:"final"`
	FinalizedAt   time.Time `json:"finalized_at"`
}

func (bcs *BlockchainServer) handleValidationProofMint(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if bcs.checkNetworkPauseAndRejectIfPaused(w, "validation_proof_mint") {
		return
	}
	expectedToken := strings.TrimSpace(os.Getenv("KNIRV_INTERNAL_AUTH_TOKEN"))
	providedToken := strings.TrimSpace(r.Header.Get("X-KNIRV-Internal-Token"))
	if expectedToken == "" {
		http.Error(w, "internal proof minting is not configured", http.StatusServiceUnavailable)
		return
	}
	if subtle.ConstantTimeCompare([]byte(expectedToken), []byte(providedToken)) != 1 {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var request validationProofMintRequest
	if err := decoder.Decode(&request); err != nil {
		http.Error(w, "invalid validation proof: "+err.Error(), http.StatusBadRequest)
		return
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		http.Error(w, "validation proof must contain one JSON value", http.StatusBadRequest)
		return
	}
	if err := validateValidationProofMint(request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	bcs.validationProofMu.Lock()
	defer bcs.validationProofMu.Unlock()
	record, exists, err := bcs.validationProofRecord(request.Submission.ProofRoot)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	canonical, err := json.Marshal(request)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	requestHash := sha256Digest(canonical)
	if exists {
		if record.ProjectID != request.Submission.ProjectID || record.CommitSHA256 != request.Submission.Git.RawSHA256 || record.RequestHash != requestHash {
			http.Error(w, "proof root is already assigned", http.StatusConflict)
			return
		}
		bcs.writeValidationProofReceipt(w, record)
		return
	}
	if conflicting, err := bcs.validationCommitRecord(request.Submission.ProjectID, request.Submission.Git.RawSHA256); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else if conflicting != nil && conflicting.ProofRoot != request.Submission.ProofRoot {
		http.Error(w, "commit already has a primary proof", http.StatusConflict)
		return
	}

	transaction := NewTransaction(utils.BLOCKCHAIN_ADDRESS, request.Submission.ProofRoot, 0, canonical)
	transaction.Type = "protocol_validation_proof"
	bcs.BlockchainPtr.addVerifiedTxnToPoolAndSignal(transaction)
	record = &validationProofRecord{
		ProjectID: request.Submission.ProjectID, CommitSHA256: request.Submission.Git.RawSHA256,
		ProofRoot: request.Submission.ProofRoot, RequestHash: requestHash,
		TransactionID: transaction.TransactionHash, AcceptedAt: time.Now().UTC(),
	}
	if err := bcs.persistValidationProofRecord(record); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	bcs.BlockchainPtr.BroadcastTransaction(transaction)
	bcs.writeValidationProofReceipt(w, record)
}

func validateValidationProofMint(request validationProofMintRequest) error {
	if request.SchemaVersion != validationProofMintSchema {
		return fmt.Errorf("unsupported validation proof schema")
	}
	submission := request.Submission
	if submission.SchemaVersion != "knirv.proof-submission.v1" || submission.BundleSchema != "dve.bundle.v1" {
		return fmt.Errorf("unsupported proof or bundle schema")
	}
	if !validationProofSafeID.MatchString(submission.ProjectID) || !validationProofSafeID.MatchString(submission.SessionID) {
		return fmt.Errorf("invalid project or session id")
	}
	for label, digest := range map[string]string{
		"proof root": submission.ProofRoot, "storage root": submission.StorageRoot,
		"encrypted manifest cid": submission.EncryptedManifestCID,
		"commit digest":          submission.Git.RawSHA256, "policy hash": submission.PolicyHash,
		"certificate hash": request.ValidationCertificate.CertificateHash,
	} {
		if !validSHA256Digest(digest) {
			return fmt.Errorf("invalid %s", label)
		}
	}
	if submission.RepositoryFingerprint == "" || submission.SignerID == "" || submission.SigningKeyID == "" || submission.Signature == "" {
		return fmt.Errorf("proof identity is incomplete")
	}
	for label, digest := range map[string]string{
		"workspace base hash":  submission.Workspace.BaseHash,
		"workspace final hash": submission.Workspace.FinalHash,
		"workspace diff hash":  submission.Workspace.DiffHash,
	} {
		if !validSHA256Digest(digest) {
			return fmt.Errorf("invalid %s", label)
		}
	}
	calculatedStorageRoot, err := validationStorageRoot(submission.Objects)
	if err != nil || calculatedStorageRoot != submission.StorageRoot {
		return fmt.Errorf("storage root mismatch")
	}
	manifestFound := false
	for _, object := range submission.Objects {
		if object.CID == submission.EncryptedManifestCID {
			manifestFound = true
		}
	}
	recipients := make(map[string]struct{}, len(submission.KeyEnvelopes))
	for _, envelope := range submission.KeyEnvelopes {
		if envelope.RecipientID == "" || envelope.KeyID == "" || envelope.Algorithm == "" || envelope.WrappedKey == "" {
			return fmt.Errorf("incomplete key envelope")
		}
		if _, exists := recipients[envelope.RecipientID]; exists {
			return fmt.Errorf("duplicate key-envelope recipient")
		}
		recipients[envelope.RecipientID] = struct{}{}
	}
	if !manifestFound || len(recipients) < 2 {
		return fmt.Errorf("proof object or key-envelope descriptor is invalid")
	}
	if submission.Git.ObjectFormat != "sha1" && submission.Git.ObjectFormat != "sha256" {
		return fmt.Errorf("unsupported Git object format")
	}
	oidLength := 40
	if submission.Git.ObjectFormat == "sha256" {
		oidLength = 64
	}
	if !validHex(submission.Git.OID, oidLength) || !validHex(submission.Git.TreeOID, oidLength) {
		return fmt.Errorf("invalid Git object binding")
	}
	for _, parent := range submission.Git.ParentOIDs {
		if !validHex(parent, oidLength) {
			return fmt.Errorf("invalid Git parent binding")
		}
	}
	certificate := request.ValidationCertificate
	if certificate.SchemaVersion != "knirv.validation-certificate.v1" || certificate.ValidatorID == "" || certificate.ValidatedAt.IsZero() {
		return fmt.Errorf("validation certificate is incomplete")
	}
	if certificate.ProofRoot != submission.ProofRoot || certificate.CommitSHA256 != submission.Git.RawSHA256 || certificate.PolicyHash != submission.PolicyHash {
		return fmt.Errorf("validation certificate binding mismatch")
	}
	if certificate.EventBundleRoot != submission.EventBundleRoot {
		return fmt.Errorf("validation certificate event bundle root mismatch")
	}
	certificateClaim := certificate
	certificateClaim.CertificateHash = ""
	claimBytes, err := json.Marshal(certificateClaim)
	if err != nil || sha256Digest(claimBytes) != certificate.CertificateHash {
		return fmt.Errorf("validation certificate hash mismatch")
	}
	requiredReplicas := 1
	if configured, err := strconv.Atoi(os.Getenv("KNIRV_PROOF_REQUIRED_REPLICAS")); err == nil && configured > 0 {
		requiredReplicas = configured
	}
	locations := make(map[string]struct{}, len(request.StorageConfirmations))
	for _, confirmation := range request.StorageConfirmations {
		if confirmation.Location == "" || confirmation.Root != submission.StorageRoot || confirmation.At.IsZero() {
			return fmt.Errorf("invalid storage confirmation")
		}
		locations[confirmation.Location] = struct{}{}
	}
	if len(locations) < requiredReplicas {
		return fmt.Errorf("insufficient storage confirmations")
	}
	return nil
}

func (bc *BlockchainStruct) validateValidationProofMintsInBlock(block *Block) error {
	proofs := make(map[string]string)
	commits := make(map[string]string)
	for _, existingBlock := range bc.Blocks {
		for _, transaction := range existingBlock.Transactions {
			request, ok := decodeValidationProofMint(transaction.Data)
			if !ok {
				continue
			}
			proofs[request.Submission.ProofRoot] = request.Submission.Git.RawSHA256
			commits[request.Submission.ProjectID+"\x00"+request.Submission.Git.RawSHA256] = request.Submission.ProofRoot
		}
	}
	for _, transaction := range block.Transactions {
		request, ok := decodeValidationProofMint(transaction.Data)
		if !ok {
			continue
		}
		if transaction.From != utils.BLOCKCHAIN_ADDRESS || transaction.To != request.Submission.ProofRoot {
			return fmt.Errorf("validation proof transaction identity mismatch")
		}
		if err := validateValidationProofMint(request); err != nil {
			return err
		}
		if commit, exists := proofs[request.Submission.ProofRoot]; exists && commit != request.Submission.Git.RawSHA256 {
			return fmt.Errorf("proof root reassignment is forbidden")
		}
		commitKey := request.Submission.ProjectID + "\x00" + request.Submission.Git.RawSHA256
		if proof, exists := commits[commitKey]; exists && proof != request.Submission.ProofRoot {
			return fmt.Errorf("commit already has a primary proof")
		}
		proofs[request.Submission.ProofRoot] = request.Submission.Git.RawSHA256
		commits[commitKey] = request.Submission.ProofRoot
	}
	return nil
}

func decodeValidationProofMint(data []byte) (validationProofMintRequest, bool) {
	var envelope struct {
		SchemaVersion string `json:"schema_version"`
	}
	if json.Unmarshal(data, &envelope) != nil || envelope.SchemaVersion != validationProofMintSchema {
		return validationProofMintRequest{}, false
	}
	var request validationProofMintRequest
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&request) != nil {
		return validationProofMintRequest{}, false
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return validationProofMintRequest{}, false
	}
	return request, true
}

func validationStorageRoot(objects []validationObjectRef) (string, error) {
	if len(objects) == 0 {
		return "", fmt.Errorf("no ciphertext objects")
	}
	level := make([][sha256.Size]byte, 0, len(objects))
	seen := make(map[string]struct{}, len(objects))
	for _, object := range objects {
		if !validSHA256Digest(object.CID) || object.Size < 0 {
			return "", fmt.Errorf("invalid ciphertext object")
		}
		cid := "sha256:" + strings.TrimPrefix(strings.ToLower(object.CID), "sha256:")
		if _, exists := seen[cid]; exists {
			return "", fmt.Errorf("duplicate ciphertext object")
		}
		seen[cid] = struct{}{}
		level = append(level, sha256.Sum256([]byte(cid+"\n"+strconv.FormatInt(object.Size, 10))))
	}
	for len(level) > 1 {
		next := make([][sha256.Size]byte, 0, (len(level)+1)/2)
		for index := 0; index < len(level); index += 2 {
			right := level[index]
			if index+1 < len(level) {
				right = level[index+1]
			}
			pair := append(append(make([]byte, 0, sha256.Size*2), level[index][:]...), right[:]...)
			next = append(next, sha256.Sum256(pair))
		}
		level = next
	}
	return "sha256:" + hex.EncodeToString(level[0][:]), nil
}

func (bcs *BlockchainServer) validationProofRecord(proofRoot string) (*validationProofRecord, bool, error) {
	exists, err := bcs.db.KeyExists(validationProofKeyPrefix + strings.TrimPrefix(proofRoot, "sha256:"))
	if err != nil || !exists {
		return nil, false, err
	}
	raw, err := bcs.db.GetBytes(validationProofKeyPrefix + strings.TrimPrefix(proofRoot, "sha256:"))
	if err != nil {
		return nil, false, err
	}
	var record validationProofRecord
	if err := json.Unmarshal(raw, &record); err != nil {
		return nil, false, err
	}
	return &record, true, nil
}

func (bcs *BlockchainServer) validationCommitRecord(projectID, commitDigest string) (*validationProofRecord, error) {
	key := validationCommitKeyPrefix + projectID + ":" + strings.TrimPrefix(commitDigest, "sha256:")
	exists, err := bcs.db.KeyExists(key)
	if err != nil || !exists {
		return nil, err
	}
	raw, err := bcs.db.GetBytes(key)
	if err != nil {
		return nil, err
	}
	var record validationProofRecord
	if err := json.Unmarshal(raw, &record); err != nil {
		return nil, err
	}
	return &record, nil
}

func (bcs *BlockchainServer) persistValidationProofRecord(record *validationProofRecord) error {
	raw, err := json.Marshal(record)
	if err != nil {
		return err
	}
	proofKey := validationProofKeyPrefix + strings.TrimPrefix(record.ProofRoot, "sha256:")
	commitKey := validationCommitKeyPrefix + record.ProjectID + ":" + strings.TrimPrefix(record.CommitSHA256, "sha256:")
	if err := bcs.db.PutBytes(proofKey, raw); err != nil {
		return err
	}
	return bcs.db.PutBytes(commitKey, raw)
}

func (bcs *BlockchainServer) writeValidationProofReceipt(w http.ResponseWriter, record *validationProofRecord) {
	receipt := validationProofReceipt{SchemaVersion: validationProofReceiptSchema, TransactionID: record.TransactionID}
	bcs.BlockchainPtr.Lock()
	for _, block := range bcs.BlockchainPtr.Blocks {
		for _, transaction := range block.Transactions {
			if transaction.TransactionHash == record.TransactionID {
				receipt.Final = true
				receipt.BlockID = hex.EncodeToString(block.BlockHash)
				receipt.FinalizedAt = time.Unix(block.Timestamp, 0).UTC()
				break
			}
		}
		if receipt.Final {
			break
		}
	}
	bcs.BlockchainPtr.Unlock()
	w.Header().Set("Content-Type", "application/json")
	if receipt.Final {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusAccepted)
	}
	_ = json.NewEncoder(w).Encode(receipt)
}

func validSHA256Digest(value string) bool {
	value = strings.TrimPrefix(strings.ToLower(strings.TrimSpace(value)), "sha256:")
	return validHex(value, sha256.Size*2)
}

func validHex(value string, length int) bool {
	if len(value) != length {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func sha256Digest(value []byte) string {
	digest := sha256.Sum256(value)
	return "sha256:" + hex.EncodeToString(digest[:])
}
