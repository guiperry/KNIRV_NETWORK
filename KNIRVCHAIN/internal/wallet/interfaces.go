package wallet

import (
	"context"
	"math/big"
	"time"
)

// WalletManager defines the interface for wallet management operations
type WalletManager interface {
	// Wallet lifecycle
	CreateWallet(name string, password string) (*Wallet, error)
	LoadWallet(path string, password string) (*Wallet, error)
	UnloadWallet(walletID string) error
	DeleteWallet(walletID string) error
	
	// Wallet operations
	GetWallet(walletID string) (*Wallet, error)
	GetAllWallets() ([]*Wallet, error)
	LockWallet(walletID string) error
	UnlockWallet(walletID string, password string) error
	
	// Key management
	GenerateAddress(walletID string) (*Address, error)
	ImportPrivateKey(walletID string, privateKey string) (*Address, error)
	ExportPrivateKey(walletID string, address string) (string, error)
	
	// Lifecycle
	Start(ctx context.Context) error
	Stop() error
}

// PaymentProcessor defines the interface for payment processing
type PaymentProcessor interface {
	// Payment operations
	CreatePayment(payment *Payment) (*PaymentResult, error)
	ProcessPayment(paymentID string) error
	CancelPayment(paymentID string) error
	RefundPayment(paymentID string, amount *big.Int) error
	
	// Payment queries
	GetPayment(paymentID string) (*Payment, error)
	GetPaymentsByWallet(walletID string) ([]*Payment, error)
	GetPaymentHistory(walletID string, limit int) ([]*Payment, error)
	
	// Payment validation
	ValidatePayment(payment *Payment) error
	EstimateFee(payment *Payment) (*big.Int, error)
	
	// Event handling
	OnPaymentCompleted(handler PaymentHandler) error
	OnPaymentFailed(handler PaymentHandler) error
}

// TransactionBuilder defines the interface for building transactions
type TransactionBuilder interface {
	// Transaction building
	NewTransaction() *TransactionBuilder
	SetFrom(address string) *TransactionBuilder
	SetTo(address string) *TransactionBuilder
	SetAmount(amount *big.Int) *TransactionBuilder
	SetFee(fee *big.Int) *TransactionBuilder
	SetData(data []byte) *TransactionBuilder
	SetNonce(nonce uint64) *TransactionBuilder
	
	// Transaction finalization
	Build() (*Transaction, error)
	Sign(privateKey string) (*Transaction, error)
	Validate() error
}

// WalletServer defines the interface for wallet server operations
type WalletServer interface {
	// Server operations
	Start(ctx context.Context) error
	Stop() error
	GetStatus() ServerStatus
	
	// API endpoints
	HandleCreateWallet(request *CreateWalletRequest) (*CreateWalletResponse, error)
	HandleSendTransaction(request *SendTransactionRequest) (*SendTransactionResponse, error)
	HandleGetBalance(request *GetBalanceRequest) (*GetBalanceResponse, error)
	HandleGetTransactionHistory(request *GetHistoryRequest) (*GetHistoryResponse, error)
}

// MasterWallet defines the interface for master wallet operations
type MasterWallet interface {
	// Master wallet operations
	Initialize(seed string) error
	DeriveWallet(index uint32) (*Wallet, error)
	GetMasterKey() (string, error)
	
	// Hierarchical operations
	DeriveAddress(walletIndex, addressIndex uint32) (*Address, error)
	GetDerivationPath(walletIndex, addressIndex uint32) string
	
	// Backup and recovery
	ExportSeed() (string, error)
	ImportSeed(seed string) error
	CreateBackup() (*WalletBackup, error)
	RestoreFromBackup(backup *WalletBackup) error
}

// Wallet represents a cryptocurrency wallet
type Wallet struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Type      string     `json:"type"`
	Addresses []*Address `json:"addresses"`
	Balance   *big.Int   `json:"balance"`
	Locked    bool       `json:"locked"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// Address represents a wallet address
type Address struct {
	Address    string    `json:"address"`
	PublicKey  string    `json:"public_key"`
	PrivateKey string    `json:"private_key,omitempty"`
	Balance    *big.Int  `json:"balance"`
	Nonce      uint64    `json:"nonce"`
	CreatedAt  time.Time `json:"created_at"`
	Label      string    `json:"label,omitempty"`
}

// Payment represents a payment transaction
type Payment struct {
	ID          string    `json:"id"`
	From        string    `json:"from"`
	To          string    `json:"to"`
	Amount      *big.Int  `json:"amount"`
	Fee         *big.Int  `json:"fee"`
	Currency    string    `json:"currency"`
	Status      string    `json:"status"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	TxHash      string    `json:"tx_hash,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// PaymentResult represents the result of a payment operation
type PaymentResult struct {
	PaymentID   string    `json:"payment_id"`
	TxHash      string    `json:"tx_hash"`
	Status      string    `json:"status"`
	Fee         *big.Int  `json:"fee"`
	Timestamp   time.Time `json:"timestamp"`
	BlockHeight uint64    `json:"block_height,omitempty"`
}

// Transaction represents a blockchain transaction
type Transaction struct {
	ID        string    `json:"id"`
	From      string    `json:"from"`
	To        string    `json:"to"`
	Amount    *big.Int  `json:"amount"`
	Fee       *big.Int  `json:"fee"`
	Nonce     uint64    `json:"nonce"`
	Data      []byte    `json:"data,omitempty"`
	Signature string    `json:"signature"`
	Hash      string    `json:"hash"`
	Timestamp time.Time `json:"timestamp"`
}

// ServerStatus represents the status of the wallet server
type ServerStatus struct {
	Running       bool      `json:"running"`
	WalletCount   int       `json:"wallet_count"`
	ActiveSessions int      `json:"active_sessions"`
	LastActivity  time.Time `json:"last_activity"`
	Version       string    `json:"version"`
}

// WalletBackup represents a wallet backup
type WalletBackup struct {
	Version   string    `json:"version"`
	Timestamp time.Time `json:"timestamp"`
	Wallets   []*Wallet `json:"wallets"`
	Seed      string    `json:"seed,omitempty"`
	Checksum  string    `json:"checksum"`
}

// API Request/Response types
type CreateWalletRequest struct {
	Name     string `json:"name"`
	Password string `json:"password"`
	Type     string `json:"type,omitempty"`
}

type CreateWalletResponse struct {
	WalletID string   `json:"wallet_id"`
	Address  string   `json:"address"`
	Mnemonic string   `json:"mnemonic,omitempty"`
}

type SendTransactionRequest struct {
	WalletID string   `json:"wallet_id"`
	From     string   `json:"from"`
	To       string   `json:"to"`
	Amount   *big.Int `json:"amount"`
	Fee      *big.Int `json:"fee,omitempty"`
	Data     []byte   `json:"data,omitempty"`
	Password string   `json:"password"`
}

type SendTransactionResponse struct {
	TxHash    string   `json:"tx_hash"`
	Status    string   `json:"status"`
	Fee       *big.Int `json:"fee"`
	Timestamp time.Time `json:"timestamp"`
}

type GetBalanceRequest struct {
	WalletID string `json:"wallet_id"`
	Address  string `json:"address,omitempty"`
}

type GetBalanceResponse struct {
	Balance   *big.Int `json:"balance"`
	Addresses []AddressBalance `json:"addresses,omitempty"`
}

type AddressBalance struct {
	Address string   `json:"address"`
	Balance *big.Int `json:"balance"`
}

type GetHistoryRequest struct {
	WalletID string `json:"wallet_id"`
	Address  string `json:"address,omitempty"`
	Limit    int    `json:"limit,omitempty"`
	Offset   int    `json:"offset,omitempty"`
}

type GetHistoryResponse struct {
	Transactions []*Transaction `json:"transactions"`
	Total        int            `json:"total"`
}

// Balance represents account balance information
type Balance struct {
	Address   string   `json:"address"`
	Available *big.Int `json:"available"`
	Pending   *big.Int `json:"pending"`
	Total     *big.Int `json:"total"`
	Currency  string   `json:"currency"`
	UpdatedAt time.Time `json:"updated_at"`
}

// WalletEvent represents wallet-related events
type WalletEvent struct {
	Type      string                 `json:"type"`
	WalletID  string                 `json:"wallet_id"`
	Data      map[string]interface{} `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
}

// PaymentHandler defines the function signature for payment event handlers
type PaymentHandler func(payment *Payment) error

// WalletEventHandler defines the function signature for wallet event handlers
type WalletEventHandler func(event *WalletEvent) error

// WalletConfig represents wallet configuration
type WalletConfig struct {
	DataDir           string        `json:"data_dir"`
	NetworkType       string        `json:"network_type"`
	DefaultFee        *big.Int      `json:"default_fee"`
	MaxFee            *big.Int      `json:"max_fee"`
	ConfirmationBlocks int          `json:"confirmation_blocks"`
	AutoLockTimeout   time.Duration `json:"auto_lock_timeout"`
	BackupEnabled     bool          `json:"backup_enabled"`
	BackupInterval    time.Duration `json:"backup_interval"`
}

// CryptoProvider defines the interface for cryptographic operations
type CryptoProvider interface {
	GenerateKeyPair() (publicKey, privateKey string, err error)
	SignTransaction(tx *Transaction, privateKey string) (string, error)
	VerifySignature(tx *Transaction, signature, publicKey string) bool
	DeriveAddress(publicKey string) (string, error)
	EncryptPrivateKey(privateKey, password string) (string, error)
	DecryptPrivateKey(encryptedKey, password string) (string, error)
}

// Error types for wallet operations
var (
	ErrWalletNotFound     = NewWalletError("wallet not found")
	ErrWalletLocked       = NewWalletError("wallet is locked")
	ErrInsufficientFunds  = NewWalletError("insufficient funds")
	ErrInvalidAddress     = NewWalletError("invalid address")
	ErrInvalidPassword    = NewWalletError("invalid password")
	ErrTransactionFailed  = NewWalletError("transaction failed")
	ErrPaymentFailed      = NewWalletError("payment failed")
)

// WalletError represents a wallet-specific error
type WalletError struct {
	Message string
	Code    string
}

func (e *WalletError) Error() string {
	return e.Message
}

func NewWalletError(message string) *WalletError {
	return &WalletError{Message: message}
}
