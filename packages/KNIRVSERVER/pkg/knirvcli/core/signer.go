package core

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
)

// Transaction represents a KNIRVCHAIN transaction
type Transaction struct {
	From            string `json:"from"`
	To              string `json:"to,omitempty"`
	Value           uint64 `json:"value"`
	Data            []byte `json:"data"`
	Timestamp       int64  `json:"timestamp"`
	Fee             uint64 `json:"fee"`
	Type            string `json:"type"`
	PublicKey       string `json:"publicKey,omitempty"`
	Signature       []byte `json:"signature,omitempty"`
	TransactionHash string `json:"transactionHash,omitempty"`
}

// UnsignedTransactionDetails represents the details of an unsigned transaction
type UnsignedTransactionDetails struct {
	From      string      `json:"from"`
	To        string      `json:"to,omitempty"`
	Value     uint64      `json:"value"`
	Data      interface{} `json:"data"` // Contains the MCPRegisterCapabilityData
	Timestamp int64       `json:"timestamp"`
	Fee       uint64      `json:"fee"`
	Type      string      `json:"type"`
}

// GetCanonicalBytesForHashing returns the canonical bytes for hashing
func GetCanonicalBytesForHashing(tx *Transaction) ([]byte, error) {
	// Create a map with the transaction fields in a specific order
	txMap := map[string]interface{}{
		"from":      tx.From,
		"timestamp": tx.Timestamp,
		"fee":       tx.Fee,
		"type":      tx.Type,
	}

	// Add optional fields only if they are present
	if tx.To != "" {
		txMap["to"] = tx.To
	}
	if tx.Value > 0 {
		txMap["value"] = tx.Value
	}
	if len(tx.Data) > 0 {
		txMap["data"] = tx.Data
	}

	// Marshal to JSON with deterministic ordering
	canonicalBytes, err := json.Marshal(txMap)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal transaction: %w", err)
	}

	return canonicalBytes, nil
}

// CalculateTransactionHash calculates the hash of a transaction
func CalculateTransactionHash(tx *Transaction) (string, error) {
	canonicalBytes, err := GetCanonicalBytesForHashing(tx)
	if err != nil {
		return "", fmt.Errorf("failed to get canonical bytes: %w", err)
	}

	hash := sha256.Sum256(canonicalBytes)
	return "0x" + hex.EncodeToString(hash[:]), nil
}

// SignTransactionData signs transaction data with a private key
func SignTransactionData(
	privateKey *ecdsa.PrivateKey,
	unsignedTxDetails UnsignedTransactionDetails,
	mcpPayloadBytes []byte,
) ([]byte, string, error) {
	// Create a transaction object from unsigned details
	tx := &Transaction{
		From:      unsignedTxDetails.From,
		To:        unsignedTxDetails.To,
		Value:     unsignedTxDetails.Value,
		Data:      mcpPayloadBytes,
		Timestamp: unsignedTxDetails.Timestamp,
		Fee:       unsignedTxDetails.Fee,
		Type:      unsignedTxDetails.Type,
	}

	// Get canonical bytes for hashing
	canonicalBytes, err := GetCanonicalBytesForHashing(tx)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get canonical bytes: %w", err)
	}

	// Calculate hash
	hash := sha256.Sum256(canonicalBytes)

	// Sign the hash
	signature, err := crypto.Sign(hash[:], privateKey)
	if err != nil {
		return nil, "", fmt.Errorf("failed to sign transaction: %w", err)
	}

	// Calculate transaction hash
	txHash := "0x" + hex.EncodeToString(hash[:])

	return signature, txHash, nil
}

// VerifySignature verifies a transaction signature
func VerifySignature(
	publicKeyHex string,
	signature []byte,
	transactionHash string,
) (bool, error) {
	// Decode public key
	publicKeyBytes, err := hex.DecodeString(strings.TrimPrefix(publicKeyHex, "0x"))
	if err != nil {
		return false, fmt.Errorf("failed to decode public key: %w", err)
	}

	// Decode transaction hash
	hashBytes, err := hex.DecodeString(strings.TrimPrefix(transactionHash, "0x"))
	if err != nil {
		return false, fmt.Errorf("failed to decode transaction hash: %w", err)
	}

	// Verify signature
	return crypto.VerifySignature(publicKeyBytes, hashBytes, signature[:len(signature)-1]), nil
}

// AssembleSignedTransaction assembles a signed transaction
func AssembleSignedTransaction(
	unsignedTxDetails UnsignedTransactionDetails,
	publicKeyHex string,
	signature []byte,
	transactionHash string,
) (*Transaction, error) {
	// Convert data interface to bytes if needed
	var dataBytes []byte
	var err error

	switch data := unsignedTxDetails.Data.(type) {
	case []byte:
		dataBytes = data
	case string:
		dataBytes = []byte(data)
	default:
		dataBytes, err = json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal transaction data: %w", err)
		}
	}

	// Create signed transaction
	tx := &Transaction{
		From:            unsignedTxDetails.From,
		To:              unsignedTxDetails.To,
		Value:           unsignedTxDetails.Value,
		Data:            dataBytes,
		Timestamp:       unsignedTxDetails.Timestamp,
		Fee:             unsignedTxDetails.Fee,
		Type:            unsignedTxDetails.Type,
		PublicKey:       publicKeyHex,
		Signature:       signature,
		TransactionHash: transactionHash,
	}

	return tx, nil
}

// CreateTransaction creates a new transaction
func CreateTransaction(
	from string,
	to string,
	value uint64,
	data interface{},
	fee uint64,
	txType string,
) (*UnsignedTransactionDetails, error) {
	// Create unsigned transaction details
	unsignedTxDetails := UnsignedTransactionDetails{
		From:      from,
		To:        to,
		Value:     value,
		Data:      data,
		Timestamp: time.Now().Unix(),
		Fee:       fee,
		Type:      txType,
	}

	return &unsignedTxDetails, nil
}
