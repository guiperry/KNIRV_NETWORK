package transaction

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
)

// Wallet represents a blockchain wallet with its associated keys and address
type Wallet struct {
	address    string
	publicKey  *ecdsa.PublicKey
	privateKey *ecdsa.PrivateKey
}

// GetAddress returns the wallet's address
func (w *Wallet) GetAddress() string {
	return w.address
}

// GetPublicKeyHex returns the wallet's public key in hexadecimal format
func (w *Wallet) GetPublicKeyHex() string {
	if w.publicKey == nil {
		return ""
	}
	return hex.EncodeToString(elliptic.Marshal(w.publicKey.Curve, w.publicKey.X, w.publicKey.Y))
}

// SignTransaction signs a transaction using the wallet's private key
func (w *Wallet) SignTransaction(txn *Transaction) error {
	if w.privateKey == nil {
		return errors.New("private key not available")
	}

	// Create the message to sign (transaction data)
	message := fmt.Sprintf("%s%s%s%s%d%s%d",
		txn.From,
		txn.To,
		txn.Value,
		txn.Type,
		txn.Timestamp,
		txn.Fee,
		txn.Version,
	)

	// Sign the message
	r, s, err := ecdsa.Sign(rand.Reader, w.privateKey, []byte(message))
	if err != nil {
		return fmt.Errorf("failed to sign transaction: %w", err)
	}

	// Encode the signature
	signature := append(r.Bytes(), s.Bytes()...)
	txn.Signature = hex.EncodeToString(signature)
	txn.PublicKey = w.GetPublicKeyHex()

	return nil
}

// NewWallet creates a new wallet with a new key pair
func NewWallet() (*Wallet, error) {
	// Generate a new private key
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	// Create the wallet
	wallet := &Wallet{
		privateKey: privateKey,
		publicKey:  &privateKey.PublicKey,
	}

	// Generate address from public key
	wallet.address = hex.EncodeToString(elliptic.Marshal(privateKey.Curve, privateKey.PublicKey.X, privateKey.PublicKey.Y))

	return wallet, nil
}

// TransactionType represents the type of transaction
type TransactionType string

const (
	TransactionTypeMCPRegisterCapability TransactionType = "mcp_register_capability"
	TransactionTypeTransfer              TransactionType = "transfer"
	TransactionTypeMCPInvokeCapability   TransactionType = "mcp_invoke_capability"
	TransactionTypeMCPUpdateCapability   TransactionType = "mcp_update_capability"
)

// ResourceType represents the type of resource
type ResourceType string

const (
	ResourceTypePlugin ResourceType = "plugin"
	ResourceTypeData   ResourceType = "data"
	ResourceTypeCode   ResourceType = "code"
)

// ResourceDescriptor represents a resource in the system
type ResourceDescriptor struct {
	ID           string       `json:"id"`
	ResourceType ResourceType `json:"resource_type"`
	ContentHash  string       `json:"content_hash"`
	Schema       struct {
		LocationHints []string `json:"location_hints"`
	} `json:"schema"`
}

// NewMCPTransaction creates a new MCP transaction
func NewMCPTransaction(from, to string, value uint64, data []byte, txType TransactionType, fee uint64, timestamp int64) *Transaction {
	return &Transaction{
		From:            from,
		To:              to,
		Value:           fmt.Sprintf("%d", value),
		Type:            string(txType),
		Fee:             fmt.Sprintf("%d", fee),
		Timestamp:       timestamp,
		Version:         1,
		Data:            make(map[string]any),
		TransactionHash: calculateHash(from, to, value, data, txType, fee, timestamp),
	}
}

// calculateHash generates a hash for the transaction
func calculateHash(from, to string, value uint64, data []byte, txType TransactionType, fee uint64, timestamp int64) string {
	// In a real implementation, this would use a proper cryptographic hash function
	// For now, we'll return a placeholder
	return fmt.Sprintf("%x", []byte(fmt.Sprintf("%s%s%d%s%d%d", from, to, value, txType, fee, timestamp)))
}
