package crypto

import (
	"time"
)

// CryptoProvider defines the interface for cryptographic operations
type CryptoProvider interface {
	// Key generation
	GenerateKeyPair() (*KeyPair, error)
	GeneratePrivateKey() (*PrivateKey, error)
	DerivePublicKey(privateKey *PrivateKey) (*PublicKey, error)

	// Signing and verification
	Sign(data []byte, privateKey *PrivateKey) (*Signature, error)
	Verify(data []byte, signature *Signature, publicKey *PublicKey) (bool, error)

	// Encryption and decryption
	Encrypt(data []byte, publicKey *PublicKey) ([]byte, error)
	Decrypt(encryptedData []byte, privateKey *PrivateKey) ([]byte, error)

	// Hashing
	Hash(data []byte) ([]byte, error)
	HashWithSalt(data, salt []byte) ([]byte, error)
}

// EmbeddingProvider defines the interface for deterministic embeddings
type EmbeddingProvider interface {
	// Embedding generation
	GenerateEmbedding(text string) ([]float32, error)
	GenerateBatchEmbeddings(texts []string) ([][]float32, error)

	// Deterministic operations
	GenerateDeterministicEmbedding(text string, seed []byte) ([]float32, error)
	ValidateEmbedding(text string, embedding []float32, seed []byte) (bool, error)

	// Embedding operations
	ComputeSimilarity(embedding1, embedding2 []float32) (float64, error)
	FindSimilar(query []float32, embeddings [][]float32, threshold float64) ([]int, error)

	// Configuration
	SetModel(modelName string) error
	GetModel() string
	GetDimensions() int
}

// PoAuDValidator defines the interface for Proof of Authentic Data validation
type PoAuDValidator interface {
	// Validation operations
	ValidatePoAuD(proof *PoAuDProof) (*ValidationResult, error)
	ValidateStandalone(data []byte, proof *PoAuDProof) (*ValidationResult, error)

	// Proof generation
	GenerateProof(data []byte, metadata *ProofMetadata) (*PoAuDProof, error)
	GenerateProofWithKey(data []byte, privateKey *PrivateKey, metadata *ProofMetadata) (*PoAuDProof, error)

	// Proof verification
	VerifyProof(proof *PoAuDProof) (bool, error)
	VerifyProofChain(proofs []*PoAuDProof) (bool, error)

	// Authenticity checking
	CheckAuthenticity(data []byte, proof *PoAuDProof) (*AuthenticityResult, error)
	CheckDataIntegrity(data []byte, proof *PoAuDProof) (bool, error)
}

// CerebrasProvider defines the interface for Cerebras-specific operations
type CerebrasProvider interface {
	// Cerebras embedding operations
	GenerateCerebrasEmbedding(text string, config *CerebrasConfig) ([]float32, error)
	GenerateDeterministicCerebrasEmbedding(text string, seed []byte, config *CerebrasConfig) ([]float32, error)

	// Model operations
	LoadModel(modelPath string) error
	UnloadModel() error
	GetModelInfo() (*ModelInfo, error)

	// Batch operations
	ProcessBatch(texts []string, config *CerebrasConfig) (*BatchResult, error)

	// Configuration
	SetCerebrasConfig(config *CerebrasConfig) error
	GetCerebrasConfig() *CerebrasConfig
}

// KeyManager defines the interface for key management operations
type KeyManager interface {
	// Key storage
	StoreKey(keyID string, key *PrivateKey) error
	RetrieveKey(keyID string) (*PrivateKey, error)
	DeleteKey(keyID string) error
	ListKeys() ([]string, error)

	// Key rotation
	RotateKey(keyID string) (*KeyPair, error)
	ScheduleRotation(keyID string, interval time.Duration) error

	// Key derivation
	DeriveKey(masterKey *PrivateKey, path string) (*PrivateKey, error)
	DeriveKeyFromSeed(seed []byte, path string) (*PrivateKey, error)

	// Security
	LockKeystore() error
	UnlockKeystore(password string) error
	IsLocked() bool
}

// KeyPair represents a cryptographic key pair
type KeyPair struct {
	PrivateKey *PrivateKey `json:"private_key"`
	PublicKey  *PublicKey  `json:"public_key"`
	Algorithm  string      `json:"algorithm"`
	CreatedAt  time.Time   `json:"created_at"`
}

// PrivateKey represents a private key
type PrivateKey struct {
	Data      []byte    `json:"data"`
	Algorithm string    `json:"algorithm"`
	KeyID     string    `json:"key_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// PublicKey represents a public key
type PublicKey struct {
	Data      []byte    `json:"data"`
	Algorithm string    `json:"algorithm"`
	KeyID     string    `json:"key_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// Signature represents a cryptographic signature
type Signature struct {
	Data      []byte    `json:"data"`
	Algorithm string    `json:"algorithm"`
	KeyID     string    `json:"key_id,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// PoAuDProof represents a Proof of Authentic Data
type PoAuDProof struct {
	ID         string                 `json:"id"`
	DataHash   []byte                 `json:"data_hash"`
	Signature  *Signature             `json:"signature"`
	Timestamp  time.Time              `json:"timestamp"`
	Metadata   *ProofMetadata         `json:"metadata"`
	ChainProof *ChainProof            `json:"chain_proof,omitempty"`
	Embeddings []float32              `json:"embeddings,omitempty"`
	Attributes map[string]interface{} `json:"attributes,omitempty"`
}

// ProofMetadata represents metadata for a PoAuD proof
type ProofMetadata struct {
	Source      string                 `json:"source"`
	Creator     string                 `json:"creator"`
	Version     string                 `json:"version"`
	Description string                 `json:"description,omitempty"`
	Tags        []string               `json:"tags,omitempty"`
	Context     map[string]interface{} `json:"context,omitempty"`
}

// ChainProof represents a chain of proofs for verification
type ChainProof struct {
	PreviousHash []byte     `json:"previous_hash"`
	MerkleRoot   []byte     `json:"merkle_root"`
	BlockHeight  uint64     `json:"block_height"`
	Witnesses    []*Witness `json:"witnesses"`
}

// Witness represents a witness in the proof chain
type Witness struct {
	ID        string     `json:"id"`
	PublicKey *PublicKey `json:"public_key"`
	Signature *Signature `json:"signature"`
	Timestamp time.Time  `json:"timestamp"`
}

// ValidationResult represents the result of PoAuD validation
type ValidationResult struct {
	Valid      bool                   `json:"valid"`
	Score      float64                `json:"score"`
	Confidence float64                `json:"confidence"`
	Errors     []string               `json:"errors,omitempty"`
	Warnings   []string               `json:"warnings,omitempty"`
	Details    map[string]interface{} `json:"details,omitempty"`
	Timestamp  time.Time              `json:"timestamp"`
}

// AuthenticityResult represents the result of authenticity checking
type AuthenticityResult struct {
	Authentic  bool                   `json:"authentic"`
	Confidence float64                `json:"confidence"`
	Factors    []AuthenticityFactor   `json:"factors"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	Timestamp  time.Time              `json:"timestamp"`
}

// AuthenticityFactor represents a factor in authenticity determination
type AuthenticityFactor struct {
	Type        string  `json:"type"`
	Score       float64 `json:"score"`
	Weight      float64 `json:"weight"`
	Description string  `json:"description"`
}

// CerebrasConfig represents configuration for Cerebras operations
type CerebrasConfig struct {
	ModelName     string                 `json:"model_name"`
	Temperature   float64                `json:"temperature"`
	MaxTokens     int                    `json:"max_tokens"`
	Deterministic bool                   `json:"deterministic"`
	Seed          []byte                 `json:"seed,omitempty"`
	Parameters    map[string]interface{} `json:"parameters,omitempty"`
}

// ModelInfo represents information about a loaded model
type ModelInfo struct {
	Name         string    `json:"name"`
	Version      string    `json:"version"`
	Architecture string    `json:"architecture"`
	Parameters   int64     `json:"parameters"`
	LoadedAt     time.Time `json:"loaded_at"`
	MemoryUsage  int64     `json:"memory_usage"`
}

// BatchResult represents the result of batch processing
type BatchResult struct {
	Results     [][]float32            `json:"results"`
	ProcessedAt time.Time              `json:"processed_at"`
	Duration    time.Duration          `json:"duration"`
	Errors      []string               `json:"errors,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// HashConfig represents configuration for hashing operations
type HashConfig struct {
	Algorithm string `json:"algorithm"`
	SaltSize  int    `json:"salt_size"`
	Rounds    int    `json:"rounds"`
}

// EncryptionConfig represents configuration for encryption operations
type EncryptionConfig struct {
	Algorithm string                 `json:"algorithm"`
	KeySize   int                    `json:"key_size"`
	Mode      string                 `json:"mode"`
	Padding   string                 `json:"padding"`
	Options   map[string]interface{} `json:"options,omitempty"`
}

// CryptoEvent represents cryptographic events
type CryptoEvent struct {
	Type      string                 `json:"type"`
	Operation string                 `json:"operation"`
	KeyID     string                 `json:"key_id,omitempty"`
	Success   bool                   `json:"success"`
	Error     string                 `json:"error,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// CryptoMetrics represents cryptographic operation metrics
type CryptoMetrics struct {
	SignOperations    int64         `json:"sign_operations"`
	VerifyOperations  int64         `json:"verify_operations"`
	EncryptOperations int64         `json:"encrypt_operations"`
	DecryptOperations int64         `json:"decrypt_operations"`
	HashOperations    int64         `json:"hash_operations"`
	AverageLatency    time.Duration `json:"average_latency"`
	ErrorCount        int64         `json:"error_count"`
	LastOperation     time.Time     `json:"last_operation"`
}

// CryptoConfig represents overall cryptographic configuration
type CryptoConfig struct {
	DefaultAlgorithm string            `json:"default_algorithm"`
	KeySize          int               `json:"key_size"`
	HashConfig       *HashConfig       `json:"hash_config"`
	EncryptionConfig *EncryptionConfig `json:"encryption_config"`
	CerebrasConfig   *CerebrasConfig   `json:"cerebras_config,omitempty"`
	SecurityLevel    string            `json:"security_level"`
	EnableHardware   bool              `json:"enable_hardware"`
}

// EventHandler defines the function signature for crypto event handlers
type EventHandler func(event *CryptoEvent) error

// Error types for cryptographic operations
var (
	ErrInvalidKey       = NewCryptoError("invalid key")
	ErrInvalidSignature = NewCryptoError("invalid signature")
	ErrEncryptionFailed = NewCryptoError("encryption failed")
	ErrDecryptionFailed = NewCryptoError("decryption failed")
	ErrKeyNotFound      = NewCryptoError("key not found")
	ErrProofInvalid     = NewCryptoError("proof invalid")
	ErrEmbeddingFailed  = NewCryptoError("embedding generation failed")
	ErrModelNotLoaded   = NewCryptoError("model not loaded")
)

// CryptoError represents a cryptographic error
type CryptoError struct {
	Message string
	Code    string
}

func (e *CryptoError) Error() string {
	return e.Message
}

func NewCryptoError(message string) *CryptoError {
	return &CryptoError{Message: message}
}
