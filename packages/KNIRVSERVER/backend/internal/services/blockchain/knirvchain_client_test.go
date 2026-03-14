package blockchain

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"backend_server/internal/objects"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewKNIRVCHAINClient(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/genesis" {
			resp := GenesisBlock{
				ChainID:     "knirv-test-1",
				BlockHeight: 100,
				Timestamp:   time.Now().Unix(),
			}
			json.NewEncoder(w).Encode(resp)
		}
	}))
	defer server.Close()

	client, err := NewKNIRVCHAINClient(server.URL, "", false, nil)
	require.NoError(t, err)
	require.NotNil(t, client)

	chainID, err := client.GetChainID()
	assert.NoError(t, err)
	assert.Equal(t, "knirv-test-1", chainID)
}

func TestKNIRVCHAINClient_GetBlockHeight(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/chain/height" {
			resp := struct {
				Height uint64 `json:"height"`
			}{Height: 500}
			json.NewEncoder(w).Encode(resp)
		}
	}))
	defer server.Close()

	client, err := NewKNIRVCHAINClient(server.URL, "", false, nil)
	require.NoError(t, err)

	height, err := client.GetBlockHeight()
	assert.NoError(t, err)
	assert.Equal(t, uint64(500), height)
}

func TestKNIRVCHAINClient_VerifyPaymentTransaction(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/chain/height":
			heightResp := struct {
				Height uint64 `json:"height"`
			}{Height: 1000}
			json.NewEncoder(w).Encode(heightResp)
		case "/chain/tx/test-tx-hash":
			tx := TransactionResponse{
				TransactionHash: "test-tx-hash",
				From:            "sender",
				To:              "knirv1system",
				Amount:          1000,
				BlockHeight:     990,
				Timestamp:       time.Now().Unix() - 60,
				Confirmations:   10,
				Status:          "confirmed",
			}
			json.NewEncoder(w).Encode(tx)
		}
	}))
	defer server.Close()

	client, err := NewKNIRVCHAINClient(server.URL, "", false, nil)
	require.NoError(t, err)

	payment, err := client.VerifyPaymentTransaction("test-tx-hash", 1000, "knirv1system")
	require.NoError(t, err)
	assert.NotNil(t, payment)
	assert.Equal(t, "test-tx-hash", payment.TxHash)
	assert.Equal(t, int64(1000), payment.Amount)
	assert.Equal(t, "confirmed", payment.Status)
}

func TestKNIRVCHAINClient_VerifyPaymentTransaction_InsufficientConfirmations(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/chain/height":
			heightResp := struct {
				Height uint64 `json:"height"`
			}{Height: 100}
			json.NewEncoder(w).Encode(heightResp)
		case "/chain/tx/test-tx-hash":
			tx := TransactionResponse{
				TransactionHash: "test-tx-hash",
				From:            "sender",
				To:              "knirv1system",
				Amount:          1000,
				BlockHeight:     99,
				Timestamp:       time.Now().Unix() - 60,
				Confirmations:   1,
				Status:          "confirmed",
			}
			json.NewEncoder(w).Encode(tx)
		}
	}))
	defer server.Close()

	client, err := NewKNIRVCHAINClient(server.URL, "", false, nil)
	require.NoError(t, err)

	_, err = client.VerifyPaymentTransaction("test-tx-hash", 1000, "knirv1system")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "insufficient confirmations")
}

func TestKNIRVCHAINClient_VerifyPaymentTransaction_AmountMismatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/chain/tx/test-tx-hash" {
			tx := TransactionResponse{
				TransactionHash: "test-tx-hash",
				From:            "sender",
				To:              "knirv1system",
				Amount:          500,
				BlockHeight:     990,
				Timestamp:       time.Now().Unix() - 60,
				Confirmations:   10,
				Status:          "confirmed",
			}
			json.NewEncoder(w).Encode(tx)
		}
	}))
	defer server.Close()

	client, err := NewKNIRVCHAINClient(server.URL, "", false, nil)
	require.NoError(t, err)

	_, err = client.VerifyPaymentTransaction("test-tx-hash", 1000, "knirv1system")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "amount mismatch")
}

func TestKNIRVCHAINClient_CreateChainSession(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/chain/height" {
			resp := struct {
				Height uint64 `json:"height"`
			}{Height: 100}
			json.NewEncoder(w).Encode(resp)
		}
	}))
	defer server.Close()

	client, err := NewKNIRVCHAINClient(server.URL, "", false, nil)
	require.NoError(t, err)

	session, err := client.CreateChainSession("dve-node-1", "knirv1owner123")
	require.NoError(t, err)
	require.NotNil(t, assert.NotEmpty(t, session.SessionID))
	assert.Equal(t, "dve-node-1", session.DVENodeID)
	assert.Equal(t, "knirv1owner123", session.OwnerAddress)
	assert.Equal(t, "active", session.Status)
	assert.NotEmpty(t, session.SessionKey)
	assert.NotEmpty(t, session.PQCSignature)
}

func TestKNIRVCHAINClient_ValidateSession(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/chain/height" {
			resp := struct {
				Height uint64 `json:"height"`
			}{Height: 100}
			json.NewEncoder(w).Encode(resp)
		}
	}))
	defer server.Close()

	client, err := NewKNIRVCHAINClient(server.URL, "", false, nil)
	require.NoError(t, err)

	session, err := client.CreateChainSession("dve-node-1", "knirv1owner123")
	require.NoError(t, err)

	validatedSession, err := client.ValidateSession(session.SessionID)
	require.NoError(t, err)
	assert.Equal(t, session.SessionID, validatedSession.SessionID)
}

func TestKNIRVCHAINClient_ValidateSession_Expired(t *testing.T) {
	client, err := NewKNIRVCHAINClient("http://localhost:1", "", false, nil)
	require.NoError(t, err)

	client.sessions["expired-session"] = &objects.ChainSession{
		SessionID:   "expired-session",
		DVENodeID:   "dve-node-1",
		Status:      "active",
		CreatedAt:   time.Now().Add(-2 * SessionTimeout),
		ExpiresAt:   time.Now().Add(-time.Hour),
		BlockHeight: 100,
	}

	_, err = client.ValidateSession("expired-session")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expired")
}

func TestKNIRVCHAINClient_RegisterDVENode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/chain/height":
			resp := struct {
				Height uint64 `json:"height"`
			}{Height: 100}
			json.NewEncoder(w).Encode(resp)
		case "/transaction":
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(struct {
				TxHash string `json:"tx_hash"`
			}{TxHash: "registered-tx-hash"})
		}
	}))
	defer server.Close()

	client, err := NewKNIRVCHAINClient(server.URL, "", false, nil)
	require.NoError(t, err)

	txHash, err := client.RegisterDVENode("dve-node-1", "knirv1owner123", 1000)
	require.NoError(t, err)
	assert.Equal(t, "registered-tx-hash", txHash)
}

func TestKNIRVCHAINClient_GetAccountBalance(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/account/knirv1owner123/balance" {
			resp := struct {
				Balance int64 `json:"balance"`
			}{Balance: 50000}
			json.NewEncoder(w).Encode(resp)
		}
	}))
	defer server.Close()

	client, err := NewKNIRVCHAINClient(server.URL, "", false, nil)
	require.NoError(t, err)

	balance, err := client.GetAccountBalance("knirv1owner123")
	require.NoError(t, err)
	assert.Equal(t, int64(50000), balance)
}

func TestKNIRVCHAINClient_GetAccountBalance_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/account/unknown/balance" {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client, err := NewKNIRVCHAINClient(server.URL, "", false, nil)
	require.NoError(t, err)

	balance, err := client.GetAccountBalance("unknown")
	require.NoError(t, err)
	assert.Equal(t, int64(0), balance)
}

func TestKNIRVCHAINClient_Close(t *testing.T) {
	client, err := NewKNIRVCHAINClient("http://localhost:1", "", false, nil)
	require.NoError(t, err)

	err = client.Close()
	assert.NoError(t, err)
}

func TestTransaction_BytesToSign(t *testing.T) {
	tx := &Transaction{
		Type:        "dve_registration",
		From:        "sender",
		To:          "knirv1system",
		Value:       1000,
		Data:        []byte("dve-node-1"),
		Timestamp:   1234567890,
		ChainID:     "knirv-test-1",
		BlockHeight: 100,
	}

	bytesToSign := tx.BytesToSign()
	assert.NotEmpty(t, bytesToSign)
	assert.Greater(t, len(bytesToSign), 0)
}

func TestPQCKeyPair(t *testing.T) {
	pair := &PQCKeyPair{
		PublicKey:  []byte("test-public-key"),
		PrivateKey: []byte("test-private-key"),
		KeyType:    "dilithium",
		KeyID:      "test-key-id",
	}

	assert.NotNil(t, pair)
	assert.Equal(t, "dilithium", pair.KeyType)
}

func TestChainClientInterface(t *testing.T) {
	var _ ChainClientInterface = (*KNIRVCHAINClient)(nil)
}
