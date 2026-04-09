package unit

import (
	"encoding/hex"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/KNIRV/KNIRV_NETWORK/KNIRVSHELL/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransactionSigning(t *testing.T) {
	// Generate a test key pair
	privateKey, err := crypto.GenerateKey()
	require.NoError(t, err)
	publicKey := privateKey.PublicKey
	publicKeyBytes := crypto.FromECDSAPub(&publicKey)
	publicKeyHex := "0x" + hex.EncodeToString(publicKeyBytes)

	// Create unsigned transaction details
	unsignedTxDetails := core.UnsignedTransactionDetails{
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
	signature, txHash, err := core.SignTransactionData(privateKey, unsignedTxDetails, dataBytes)
	require.NoError(t, err)
	assert.NotNil(t, signature)
	assert.NotEmpty(t, txHash)

	// Verify the signature
	valid, err := core.VerifySignature(publicKeyHex, signature, txHash)
	require.NoError(t, err)
	assert.True(t, valid)

	// Modify the signature and verify that it fails
	signature[0] = signature[0] + 1
	valid, err = core.VerifySignature(publicKeyHex, signature, txHash)
	require.NoError(t, err)
	assert.False(t, valid)
}

func TestTransactionCreation(t *testing.T) {
	// Create a transaction
	from := "0x1234567890abcdef1234567890abcdef12345678"
	to := "0xabcdef1234567890abcdef1234567890abcdef12"
	value := uint64(100)
	data := map[string]interface{}{"test": "data"}
	fee := uint64(10)
	txType := "TEST"

	unsignedTx, err := core.CreateTransaction(from, to, value, data, fee, txType)
	require.NoError(t, err)
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

func TestTransactionHashing(t *testing.T) {
	// Create a test transaction
	tx := &core.Transaction{
		From:      "0x1234567890abcdef1234567890abcdef12345678",
		To:        "0xabcdef1234567890abcdef1234567890abcdef12",
		Value:     100,
		Data:      []byte(`{"test":"data"}`),
		Timestamp: 1625097600,
		Fee:       10,
		Type:      "TEST",
	}

	// Calculate hash
	hash, err := core.CalculateTransactionHash(tx)
	require.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.True(t, len(hash) > 2)
	assert.Equal(t, "0x", hash[0:2])

	// Verify that the hash is deterministic
	hash2, err := core.CalculateTransactionHash(tx)
	require.NoError(t, err)
	assert.Equal(t, hash, hash2)

	// Modify the transaction and verify that the hash changes
	tx.Value = 200
	hash3, err := core.CalculateTransactionHash(tx)
	require.NoError(t, err)
	assert.NotEqual(t, hash, hash3)
}
