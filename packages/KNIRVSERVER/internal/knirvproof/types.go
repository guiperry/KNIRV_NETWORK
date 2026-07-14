package knirvproof

import "time"

const (
	SchemaProofSubmission = "knirv.proof-submission.v1"
	SchemaMint            = "validation_proof_mint.v1"
)

type OperationStatus string

const (
	StatusPrepared    OperationStatus = "prepared"
	StatusUploading   OperationStatus = "uploading"
	StatusUploaded    OperationStatus = "uploaded"
	StatusVerifying   OperationStatus = "verifying"
	StatusReplicating OperationStatus = "replicating"
	StatusMintPending OperationStatus = "mint_pending"
	StatusFinalizing  OperationStatus = "finalizing"
	StatusCertified   OperationStatus = "certified"
	StatusRejected    OperationStatus = "rejected"
	StatusQuarantined OperationStatus = "quarantined"
)

func (s OperationStatus) Terminal() bool {
	return s == StatusCertified || s == StatusRejected || s == StatusQuarantined
}

type ObjectRef struct {
	CID  string `json:"cid"`
	Size int64  `json:"size"`
}

type ObjectBatchRequest struct {
	ProjectID string      `json:"project_id"`
	Objects   []ObjectRef `json:"objects"`
}

type UploadTarget struct {
	CID    string `json:"cid"`
	Method string `json:"method"`
	Path   string `json:"path"`
}

type ObjectBatchResponse struct {
	Missing []string       `json:"missing"`
	Uploads []UploadTarget `json:"uploads"`
}

type KeyEnvelope struct {
	RecipientID string `json:"recipient_id"`
	KeyID       string `json:"key_id"`
	Algorithm   string `json:"algorithm"`
	WrappedKey  string `json:"wrapped_key"`
}

type ValidatorKey struct {
	RecipientID string `json:"recipient_id"`
	KeyID       string `json:"key_id"`
	Algorithm   string `json:"algorithm"`
	WrapSchema  string `json:"wrap_schema"`
	PublicKey   string `json:"public_key"`
}

type GitCommit struct {
	ObjectFormat string   `json:"object_format"`
	OID          string   `json:"oid"`
	RawSHA256    string   `json:"raw_sha256"`
	TreeOID      string   `json:"tree_oid"`
	ParentOIDs   []string `json:"parent_oids,omitempty"`
}

type WorkspaceBinding struct {
	BaseHash  string `json:"base_hash"`
	FinalHash string `json:"final_hash"`
	DiffHash  string `json:"diff_hash"`
}

type ProofSubmission struct {
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
	Signature             string           `json:"signature"`
	RelatedProofs         []string         `json:"related_proofs,omitempty"`
}

type ValidationCertificate struct {
	SchemaVersion   string    `json:"schema_version"`
	CertificateHash string    `json:"certificate_hash"`
	ValidatorID     string    `json:"validator_id"`
	ValidatedAt     time.Time `json:"validated_at"`
	ProofRoot       string    `json:"proof_root"`
	CommitSHA256    string    `json:"commit_sha256"`
	PolicyHash      string    `json:"policy_hash"`
}

type StorageConfirmation struct {
	Location string    `json:"location"`
	Root     string    `json:"root"`
	At       time.Time `json:"at"`
}

type ChainReceipt struct {
	SchemaVersion string    `json:"schema_version"`
	TransactionID string    `json:"transaction_id"`
	BlockID       string    `json:"block_id"`
	Final         bool      `json:"final"`
	FinalizedAt   time.Time `json:"finalized_at"`
}

type ProofRecord struct {
	Submission            ProofSubmission        `json:"submission"`
	AdditionalEnvelopes   []KeyEnvelope          `json:"additional_envelopes,omitempty"`
	OperationID           string                 `json:"operation_id"`
	Status                OperationStatus        `json:"status"`
	ValidationCertificate *ValidationCertificate `json:"validation_certificate,omitempty"`
	StorageConfirmations  []StorageConfirmation  `json:"storage_confirmations,omitempty"`
	ChainReceipt          *ChainReceipt          `json:"chain_receipt,omitempty"`
	CreatedAt             time.Time              `json:"created_at"`
	UpdatedAt             time.Time              `json:"updated_at"`
}

type Operation struct {
	ID        string          `json:"id"`
	ProjectID string          `json:"project_id"`
	ProofRoot string          `json:"proof_root"`
	Status    OperationStatus `json:"status"`
	Attempts  int             `json:"attempts"`
	Retryable bool            `json:"retryable"`
	LastError string          `json:"last_error,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

type SubmitResponse struct {
	OperationID string          `json:"operation_id"`
	ProofRoot   string          `json:"proof_root"`
	Status      OperationStatus `json:"status"`
}

type AccessEnvelopeRequest struct {
	Envelope KeyEnvelope `json:"envelope"`
}

type PublicProof struct {
	ProjectID             string                 `json:"project_id"`
	SessionID             string                 `json:"session_id"`
	RepositoryFingerprint string                 `json:"repository_fingerprint"`
	Git                   GitCommit              `json:"git"`
	ProofRoot             string                 `json:"proof_root"`
	StorageRoot           string                 `json:"storage_root"`
	PolicyHash            string                 `json:"policy_hash"`
	Status                OperationStatus        `json:"status"`
	Certificate           *ValidationCertificate `json:"validation_certificate,omitempty"`
	Receipt               *ChainReceipt          `json:"chain_receipt,omitempty"`
}

type MintRequest struct {
	SchemaVersion         string                `json:"schema_version"`
	Submission            ProofSubmission       `json:"submission"`
	ValidationCertificate ValidationCertificate `json:"validation_certificate"`
	StorageConfirmations  []StorageConfirmation `json:"storage_confirmations"`
}
