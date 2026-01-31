package wallet

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockXIONServer creates a mock XION API server for testing
func mockXIONServer(_ *testing.T, balance uint64, txHash string, shouldFail bool) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if shouldFail {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "mock error"}`))
			return
		}

		// Handle balance query
		if r.URL.Path == "/cosmos/bank/v1beta1/balances/xion1test" {
			resp := BalanceResponse{
				Balances: []Balance{
					{Denom: "unrn", Amount: fmt.Sprintf("%d", balance)},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}

		// Handle transaction broadcast
		if r.URL.Path == "/cosmos/tx/v1beta1/txs" && r.Method == "POST" {
			resp := BroadcastTxResponse{
				TxResponse: TxResponse{
					Height:    "12345",
					TxHash:    txHash,
					Code:      0,
					RawLog:    "success",
					Codespace: "",
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}

		// Handle transaction query
		if r.URL.Path == "/cosmos/tx/v1beta1/txs/"+txHash && r.Method == "GET" {
			resp := GetTxResponse{
				TxResponse: TxResponse{
					Height:    "12345",
					TxHash:    txHash,
					Code:      0,
					RawLog:    "success",
					Codespace: "",
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}

		// Handle node info query
		if r.URL.Path == "/cosmos/base/tendermint/v1beta1/node_info" {
			resp := NodeInfoResponse{
				NodeInfo: NodeInfo{
					Network: "xion-testnet-1",
					Version: "v1.0.0",
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
}

func TestNewNRNWallet_Mock(t *testing.T) {
	wallet, err := NewNRNWallet("mock")
	require.NoError(t, err)
	assert.NotNil(t, wallet)
	assert.Equal(t, "xion1mockaddress", wallet.Address())
}

func TestNewNRNWallet_EmptyKey(t *testing.T) {
	_, err := NewNRNWallet("")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "private key required")
}

func TestNRNWallet_Balance(t *testing.T) {
	// Create mock server with 1M micro-NRN (1 NRN)
	server := mockXIONServer(t, 1000000, "", false)
	defer server.Close()

	// Create client pointing to mock server
	client := NewXIONClient(server.URL, server.URL, "xion-testnet-1")

	// Create wallet with mock client
	privKey := NewPrivateKey([]byte("test_key"))
	wallet := NewNRNWalletWithClient("xion1test", privKey, client)

	ctx := context.Background()

	// Test balance query
	balance, err := wallet.Balance(ctx)
	require.NoError(t, err)
	assert.Equal(t, uint64(1), balance) // 1 NRN
}

func TestNRNWallet_BalanceCache(t *testing.T) {
	server := mockXIONServer(t, 1000000, "", false)
	defer server.Close()

	client := NewXIONClient(server.URL, server.URL, "xion-testnet-1")
	privKey := NewPrivateKey([]byte("test_key"))
	wallet := NewNRNWalletWithClient("xion1test", privKey, client)

	// Set short cache TTL for testing
	wallet.cacheTTL = 100 * time.Millisecond

	ctx := context.Background()

	// First balance query - should hit network
	balance1, err := wallet.Balance(ctx)
	require.NoError(t, err)
	assert.Equal(t, uint64(1), balance1)

	// Second query immediately - should use cache
	balance2, err := wallet.Balance(ctx)
	require.NoError(t, err)
	assert.Equal(t, balance1, balance2)

	// Wait for cache to expire
	time.Sleep(150 * time.Millisecond)

	// Third query - should hit network again
	balance3, err := wallet.Balance(ctx)
	require.NoError(t, err)
	assert.Equal(t, uint64(1), balance3)
}

func TestNRNWallet_HasBalance(t *testing.T) {
	server := mockXIONServer(t, 5000000, "", false) // 5 NRN
	defer server.Close()

	client := NewXIONClient(server.URL, server.URL, "xion-testnet-1")
	privKey := NewPrivateKey([]byte("test_key"))
	wallet := NewNRNWalletWithClient("xion1test", privKey, client)

	ctx := context.Background()

	// Test sufficient balance
	hasBalance, err := wallet.HasBalance(ctx, 3)
	require.NoError(t, err)
	assert.True(t, hasBalance)

	// Test insufficient balance
	hasBalance, err = wallet.HasBalance(ctx, 10)
	require.NoError(t, err)
	assert.False(t, hasBalance)

	// Test exact balance
	hasBalance, err = wallet.HasBalance(ctx, 5)
	require.NoError(t, err)
	assert.True(t, hasBalance)
}

func TestNRNWallet_Spend(t *testing.T) {
	server := mockXIONServer(t, 10000000, "ABC123", false) // 10 NRN
	defer server.Close()

	client := NewXIONClient(server.URL, server.URL, "xion-testnet-1")
	privKey := NewPrivateKey([]byte("test_key"))
	wallet := NewNRNWalletWithClient("xion1test", privKey, client)

	ctx := context.Background()

	// Test successful spend
	txHash, err := wallet.Spend(ctx, 5, "test payment")
	require.NoError(t, err)
	assert.Equal(t, "ABC123", txHash)

	// Check transaction history
	history := wallet.GetTransactionHistory(10)
	assert.Len(t, history, 1)
	assert.Equal(t, "ABC123", history[0].TxHash)
	assert.Equal(t, uint64(5), history[0].Amount)
	assert.Equal(t, "test payment", history[0].Memo)
	assert.Equal(t, "broadcasted", history[0].Status)
}

func TestNRNWallet_Spend_InsufficientBalance(t *testing.T) {
	server := mockXIONServer(t, 1000000, "", false) // 1 NRN
	defer server.Close()

	client := NewXIONClient(server.URL, server.URL, "xion-testnet-1")
	privKey := NewPrivateKey([]byte("test_key"))
	wallet := NewNRNWalletWithClient("xion1test", privKey, client)

	ctx := context.Background()

	// Test spending more than balance
	_, err := wallet.Spend(ctx, 5, "test payment")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "insufficient balance")
}

func TestNRNWallet_GetTransactionHistory(t *testing.T) {
	server := mockXIONServer(t, 100000000, "TX1", false)
	defer server.Close()

	client := NewXIONClient(server.URL, server.URL, "xion-testnet-1")
	privKey := NewPrivateKey([]byte("test_key"))
	wallet := NewNRNWalletWithClient("xion1test", privKey, client)

	ctx := context.Background()

	// Make multiple transactions
	wallet.Spend(ctx, 1, "payment 1")
	wallet.Spend(ctx, 2, "payment 2")
	wallet.Spend(ctx, 3, "payment 3")

	// Get recent history
	history := wallet.GetTransactionHistory(2)
	assert.Len(t, history, 2)
	assert.Equal(t, "payment 3", history[1].Memo) // Most recent
	assert.Equal(t, "payment 2", history[0].Memo)
}

func TestNRNWallet_RefreshBalance(t *testing.T) {
	server := mockXIONServer(t, 5000000, "", false)
	defer server.Close()

	client := NewXIONClient(server.URL, server.URL, "xion-testnet-1")
	privKey := NewPrivateKey([]byte("test_key"))
	wallet := NewNRNWalletWithClient("xion1test", privKey, client)

	ctx := context.Background()

	// Refresh balance
	err := wallet.RefreshBalance(ctx)
	require.NoError(t, err)

	// Check cached balance
	balance, err := wallet.Balance(ctx)
	require.NoError(t, err)
	assert.Equal(t, uint64(5), balance)
}

func TestXIONClient_QueryBalance(t *testing.T) {
	server := mockXIONServer(t, 7500000, "", false)
	defer server.Close()

	client := NewXIONClient(server.URL, server.URL, "xion-testnet-1")
	ctx := context.Background()

	balance, err := client.QueryBalance(ctx, "xion1test", "unrn")
	require.NoError(t, err)
	assert.Greater(t, balance, uint64(0))
}

func TestXIONClient_QueryBalance_Error(t *testing.T) {
	server := mockXIONServer(t, 0, "", true) // Should fail
	defer server.Close()

	client := NewXIONClient(server.URL, server.URL, "xion-testnet-1")
	ctx := context.Background()

	_, err := client.QueryBalance(ctx, "xion1test", "unrn")
	assert.Error(t, err)
}

func TestXIONClient_BroadcastTx(t *testing.T) {
	server := mockXIONServer(t, 0, "TXHASH123", false)
	defer server.Close()

	client := NewXIONClient(server.URL, server.URL, "xion-testnet-1")
	ctx := context.Background()

	// Create dummy transaction bytes
	txBytes := []byte("dummy_transaction_data")

	txHash, err := client.BroadcastTx(ctx, txBytes)
	require.NoError(t, err)
	assert.Equal(t, "TXHASH123", txHash)
}

func TestXIONClient_GetTransaction(t *testing.T) {
	server := mockXIONServer(t, 0, "TXHASH456", false)
	defer server.Close()

	client := NewXIONClient(server.URL, server.URL, "xion-testnet-1")
	ctx := context.Background()

	txResp, err := client.GetTransaction(ctx, "TXHASH456")
	require.NoError(t, err)
	assert.NotNil(t, txResp)
	assert.Equal(t, "TXHASH456", txResp.TxHash)
	assert.Equal(t, uint32(0), txResp.Code) // Success
}

func TestXIONClient_GetNodeInfo(t *testing.T) {
	server := mockXIONServer(t, 0, "", false)
	defer server.Close()

	client := NewXIONClient(server.URL, server.URL, "xion-testnet-1")
	ctx := context.Background()

	nodeInfo, err := client.GetNodeInfo(ctx)
	require.NoError(t, err)
	assert.NotNil(t, nodeInfo)
	assert.Equal(t, "xion-testnet-1", nodeInfo.Network)
}

func TestTransaction_Builder(t *testing.T) {
	builder := NewTransactionBuilder("xion-testnet-1", 123, 456)

	tx := builder.
		WithMemo("test transaction").
		WithFee(1000, "unrn", 200000).
		WithSend("xion1from", "xion1to", 5000000, "unrn").
		Build()

	assert.Equal(t, "xion-testnet-1", tx.ChainID)
	assert.Equal(t, uint64(123), tx.AccountNumber)
	assert.Equal(t, uint64(456), tx.Sequence)
	assert.Equal(t, "test transaction", tx.Memo)
	assert.Len(t, tx.Msgs, 1)
	assert.Equal(t, uint64(200000), tx.Fee.Gas)
}

func TestTransaction_SignAndSerialize(t *testing.T) {
	// Create transaction
	tx := NewTransactionBuilder("xion-testnet-1", 100, 200).
		WithMemo("test").
		WithFee(1000, "unrn", 200000).
		WithSend("xion1from", "xion1to", 1000000, "unrn").
		Build()

	// Create private key
	privKey := NewPrivateKey([]byte("test_private_key"))

	// Sign transaction
	signedTx, err := SignTransaction(tx, privKey)
	require.NoError(t, err)
	assert.NotNil(t, signedTx)
	assert.Len(t, signedTx.Signatures, 1)

	// Serialize
	txBytes, err := signedTx.Serialize()
	require.NoError(t, err)
	assert.Greater(t, len(txBytes), 0)
}

func TestPrivateKey_Sign(t *testing.T) {
	privKey := NewPrivateKey([]byte("test_key"))

	msg := []byte("message to sign")
	signature, err := privKey.Sign(msg)
	require.NoError(t, err)
	assert.NotNil(t, signature)
	assert.Greater(t, len(signature), 0)

	// Same message should produce same signature (deterministic)
	signature2, err := privKey.Sign(msg)
	require.NoError(t, err)
	assert.Equal(t, signature, signature2)

	// Different message should produce different signature
	signature3, err := privKey.Sign([]byte("different message"))
	require.NoError(t, err)
	assert.NotEqual(t, signature, signature3)
}

func TestPrivateKey_GetPublicKey(t *testing.T) {
	privKey := NewPrivateKey([]byte("test_key"))

	pubKey := privKey.GetPublicKey()
	assert.NotNil(t, pubKey)
	assert.Equal(t, "/cosmos.crypto.secp256k1.PubKey", pubKey.Type)
	assert.Greater(t, len(pubKey.Key), 0)
}

func TestTransactionHistory(t *testing.T) {
	history := &TransactionHistory{
		Transactions: make([]HistoricalTransaction, 0),
	}

	// Add transactions
	history.AddTransaction(HistoricalTransaction{
		TxHash:    "TX1",
		Timestamp: time.Now(),
		Amount:    100,
		Memo:      "payment 1",
		Status:    "confirmed",
	})

	history.AddTransaction(HistoricalTransaction{
		TxHash:    "TX2",
		Timestamp: time.Now(),
		Amount:    200,
		Memo:      "payment 2",
		Status:    "pending",
	})

	history.AddTransaction(HistoricalTransaction{
		TxHash:    "TX3",
		Timestamp: time.Now(),
		Amount:    300,
		Memo:      "payment 3",
		Status:    "confirmed",
	})

	// Get recent transactions
	recent := history.GetRecent(2)
	assert.Len(t, recent, 2)
	assert.Equal(t, "TX3", recent[1].TxHash) // Most recent
	assert.Equal(t, "TX2", recent[0].TxHash)

	// Get all transactions
	all := history.GetRecent(10)
	assert.Len(t, all, 3)
}

func TestDeriveAddress(t *testing.T) {
	privKey := NewPrivateKey([]byte("test_key"))
	pubKey := privKey.GetPublicKey()

	address := deriveAddress(pubKey)
	assert.NotEmpty(t, address)
	assert.Contains(t, address, "xion1")
}

func TestEncodeBase64(t *testing.T) {
	data := []byte("hello world")
	encoded := encodeBase64(data)
	assert.NotEmpty(t, encoded)
	// Should be base64 encoded
	assert.Greater(t, len(encoded), len(data))
}
