package core

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransactionFlow(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/transaction":
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{
				"transactionHash": "test-txn",
				"status": "pending",
				"blockHeight": 0,
				"blockHash": "",
				"timestamp": 0
			}`))
		default:
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error":"not found"}`))
		}
	}))
	defer server.Close()

	// Create logger
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	// Create API client
	client := NewAPIClient(
		server.URL,
		WithTimeout(5*time.Second),
		WithRetries(2),
		WithLogger(logger),
	)

	// Create context
	ctx := context.Background()

	// Generate a test key pair
	privateKey, err := crypto.GenerateKey()
	require.NoError(t, err)
	publicKey := privateKey.PublicKey
	publicKeyBytes := crypto.FromECDSAPub(&publicKey)
	publicKeyHex := "0x" + hex.EncodeToString(publicKeyBytes)

	// Create unsigned transaction details
	unsignedTxDetails := UnsignedTransactionDetails{
		From:      "0x1234567890abcdef1234567890abcdef12345678",
		To:        "0xabcdef1234567890abcdef1234567890abcdef12",
		Value:     100,
		Data:      map[string]interface{}{"test": "data"},
		Timestamp: time.Now().Unix(),
		Fee:       10,
		Type:      "TEST",
	}

	// Convert data to bytes
	dataBytes := []byte(`{"test":"data"}`)

	// Sign the transaction
	signature, txHash, err := SignTransactionData(privateKey, unsignedTxDetails, dataBytes)
	require.NoError(t, err)
	assert.NotNil(t, signature)
	assert.NotEmpty(t, txHash)

	// Verify the signature
	valid, err := VerifySignature(publicKeyHex, signature, txHash)
	require.NoError(t, err)
	assert.True(t, valid)

	// Assemble the signed transaction
	signedTx, err := AssembleSignedTransaction(unsignedTxDetails, publicKeyHex, signature, txHash)
	require.NoError(t, err)
	assert.NotNil(t, signedTx)

	// Submit the transaction
	response, err := client.SubmitTransaction(ctx, signedTx)
	require.NoError(t, err)
	assert.Equal(t, "test-txn", response.TransactionHash)
	assert.Equal(t, "pending", response.Status)
}

func TestCreateAndSignTransaction(t *testing.T) {
	// Generate a test key pair
	privateKey, err := crypto.GenerateKey()
	require.NoError(t, err)
	publicKey := privateKey.PublicKey
	publicKeyBytes := crypto.FromECDSAPub(&publicKey)
	publicKeyHex := "0x" + hex.EncodeToString(publicKeyBytes)

	// Create a transaction
	from := "0x1234567890abcdef1234567890abcdef12345678"
	to := "0xabcdef1234567890abcdef1234567890abcdef12"
	value := uint64(100)
	data := map[string]interface{}{"test": "data"}
	fee := uint64(10)
	txType := "TEST"

	// Create unsigned transaction
	unsignedTx, err := CreateTransaction(from, to, value, data, fee, txType)
	require.NoError(t, err)
	assert.NotNil(t, unsignedTx)

	// Convert data to bytes
	dataBytes, err := json.Marshal(data)
	require.NoError(t, err)

	// Sign the transaction
	signature, txHash, err := SignTransactionData(privateKey, *unsignedTx, dataBytes)
	require.NoError(t, err)
	assert.NotNil(t, signature)
	assert.NotEmpty(t, txHash)

	// Assemble the signed transaction
	signedTx, err := AssembleSignedTransaction(*unsignedTx, publicKeyHex, signature, txHash)
	require.NoError(t, err)
	assert.NotNil(t, signedTx)

	// Verify the transaction fields
	assert.Equal(t, from, signedTx.From)
	assert.Equal(t, to, signedTx.To)
	assert.Equal(t, value, signedTx.Value)
	assert.Equal(t, fee, signedTx.Fee)
	assert.Equal(t, txType, signedTx.Type)
	assert.Equal(t, publicKeyHex, signedTx.PublicKey)
	assert.Equal(t, signature, signedTx.Signature)
	assert.Equal(t, txHash, signedTx.TransactionHash)
}
