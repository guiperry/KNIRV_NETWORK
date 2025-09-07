package core

import (
	"encoding/hex"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
)

func TestGetCanonicalBytesForHashing(t *testing.T) {
	// Create a test transaction
	tx := &Transaction{
		From:      "0x1234567890abcdef1234567890abcdef12345678",
		To:        "0xabcdef1234567890abcdef1234567890abcdef12",
		Value:     100,
		Data:      []byte(`{"test":"data"}`),
		Timestamp: 1625097600,
		Fee:       10,
		Type:      "TEST",
	}

	// Get canonical bytes
	canonicalBytes, err := GetCanonicalBytesForHashing(tx)
	assert.NoError(t, err)
	assert.NotNil(t, canonicalBytes)

	// Verify that the canonical bytes are deterministic
	canonicalBytes2, err := GetCanonicalBytesForHashing(tx)
	assert.NoError(t, err)
	assert.Equal(t, canonicalBytes, canonicalBytes2)

	// Modify the transaction and verify that the canonical bytes change
	tx.Value = 200
	canonicalBytes3, err := GetCanonicalBytesForHashing(tx)
	assert.NoError(t, err)
	assert.NotEqual(t, canonicalBytes, canonicalBytes3)
}

func TestCalculateTransactionHash(t *testing.T) {
	// Create a test transaction
	tx := &Transaction{
		From:      "0x1234567890abcdef1234567890abcdef12345678",
		To:        "0xabcdef1234567890abcdef1234567890abcdef12",
		Value:     100,
		Data:      []byte(`{"test":"data"}`),
		Timestamp: 1625097600,
		Fee:       10,
		Type:      "TEST",
	}

	// Calculate hash
	hash, err := CalculateTransactionHash(tx)
	assert.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.True(t, len(hash) > 2)
	assert.Equal(t, "0x", hash[0:2])

	// Verify that the hash is deterministic
	hash2, err := CalculateTransactionHash(tx)
	assert.NoError(t, err)
	assert.Equal(t, hash, hash2)

	// Modify the transaction and verify that the hash changes
	tx.Value = 200
	hash3, err := CalculateTransactionHash(tx)
	assert.NoError(t, err)
	assert.NotEqual(t, hash, hash3)
}

func TestSignAndVerifyTransaction(t *testing.T) {
	// Generate a test key pair
	privateKey, err := crypto.GenerateKey()
	assert.NoError(t, err)
	publicKey := privateKey.PublicKey
	publicKeyBytes := crypto.FromECDSAPub(&publicKey)
	publicKeyHex := "0x" + hex.EncodeToString(publicKeyBytes)

	// Create unsigned transaction details
	unsignedTxDetails := UnsignedTransactionDetails{
		From:      "0x1234567890abcdef1234567890abcdef12345678",
		To:        "0xabcdef1234567890abcdef1234567890abcdef12",
		Value:     100,
		Data:      map[string]interface{}{"test": "data"},
		Timestamp: 1625097600,
		Fee:       10,
		Type:      "TEST",
	}

	// Convert data to bytes
	dataBytes := []byte(`{"test":"data"}`)

	// Sign the transaction
	signature, txHash, err := SignTransactionData(privateKey, unsignedTxDetails, dataBytes)
	assert.NoError(t, err)
	assert.NotNil(t, signature)
	assert.NotEmpty(t, txHash)

	// Verify the signature
	valid, err := VerifySignature(publicKeyHex, signature, txHash)
	assert.NoError(t, err)
	assert.True(t, valid)

	// Modify the signature and verify that it fails
	signature[0] = signature[0] + 1
	valid, err = VerifySignature(publicKeyHex, signature, txHash)
	assert.NoError(t, err)
	assert.False(t, valid)
}

func TestAssembleSignedTransaction(t *testing.T) {
	// Generate a test key pair
	privateKey, err := crypto.GenerateKey()
	assert.NoError(t, err)
	publicKey := privateKey.PublicKey
	publicKeyBytes := crypto.FromECDSAPub(&publicKey)
	publicKeyHex := "0x" + hex.EncodeToString(publicKeyBytes)

	// Create unsigned transaction details
	unsignedTxDetails := UnsignedTransactionDetails{
		From:      "0x1234567890abcdef1234567890abcdef12345678",
		To:        "0xabcdef1234567890abcdef1234567890abcdef12",
		Value:     100,
		Data:      map[string]interface{}{"test": "data"},
		Timestamp: 1625097600,
		Fee:       10,
		Type:      "TEST",
	}

	// Convert data to bytes
	dataBytes := []byte(`{"test":"data"}`)

	// Sign the transaction
	signature, txHash, err := SignTransactionData(privateKey, unsignedTxDetails, dataBytes)
	assert.NoError(t, err)

	// Assemble the signed transaction
	signedTx, err := AssembleSignedTransaction(unsignedTxDetails, publicKeyHex, signature, txHash)
	assert.NoError(t, err)
	assert.NotNil(t, signedTx)

	// Verify the signed transaction fields
	assert.Equal(t, unsignedTxDetails.From, signedTx.From)
	assert.Equal(t, unsignedTxDetails.To, signedTx.To)
	assert.Equal(t, unsignedTxDetails.Value, signedTx.Value)
	assert.Equal(t, unsignedTxDetails.Timestamp, signedTx.Timestamp)
	assert.Equal(t, unsignedTxDetails.Fee, signedTx.Fee)
	assert.Equal(t, unsignedTxDetails.Type, signedTx.Type)
	assert.Equal(t, publicKeyHex, signedTx.PublicKey)
	assert.Equal(t, signature, signedTx.Signature)
	assert.Equal(t, txHash, signedTx.TransactionHash)
}

func TestCreateTransaction(t *testing.T) {
	// Create a transaction
	from := "0x1234567890abcdef1234567890abcdef12345678"
	to := "0xabcdef1234567890abcdef1234567890abcdef12"
	value := uint64(100)
	data := map[string]interface{}{"test": "data"}
	fee := uint64(10)
	txType := "TEST"

	unsignedTx, err := CreateTransaction(from, to, value, data, fee, txType)
	assert.NoError(t, err)
	assert.NotNil(t, unsignedTx)

	// Verify the transaction fields
	assert.Equal(t, from, unsignedTx.From)
	assert.Equal(t, to, unsignedTx.To)
	assert.Equal(t, value, unsignedTx.Value)
	assert.Equal(t, data, unsignedTx.Data)
	assert.Greater(t, unsignedTx.Timestamp, int64(0))
	assert.Equal(t, fee, unsignedTx.Fee)
	assert.Equal(t, txType, unsignedTx.Type)
}
