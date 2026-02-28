package blockchain

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewNRNClient(t *testing.T) {
	baseURL := "http://localhost:8080"
	client, err := NewNRNClient(baseURL, false, "")
	if err != nil {
		t.Fatalf("Failed to create NRN client: %v", err)
	}

	if client == nil {
		t.Error("Expected client to be initialized")
	}
	
	// Note: The actual NRNClient is gRPC-based, not HTTP-based
	// so we can't check for baseURL or httpClient fields
}

func TestVerifyPaymentTransaction(t *testing.T) {
	// Create a mock blockchain server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/chain" {
			block := Block{
				BlockNumber: 1,
				Transactions: []*Transaction{
					{
						TransactionHash: "tx123",
						From:            "sender",
						To:              "recipient",
						Value:           1000,
						Timestamp:       time.Now().Unix(),
					},
				},
				Timestamp: time.Now().Unix(),
				Hash:      "blockhash",
				PrevHash:  "prevhash",
			}
			response := struct {
				Blocks []*Block `json:"blocks"`
			}{
				Blocks: []*Block{&block},
			}
			json.NewEncoder(w).Encode(response)
		}
	}))
	defer server.Close()

	client, err := NewNRNClient(server.URL, false, "")
	if err != nil {
		t.Fatalf("Failed to create NRN client: %v", err)
	}

	// Test successful verification
	payment, err := client.VerifyPaymentTransaction("tx123", 1000, "recipient")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if payment.ID != "tx123" {
		t.Errorf("Expected ID tx123, got %s", payment.ID)
	}

	if payment.Amount != 1000 {
		t.Errorf("Expected amount 1000, got %d", payment.Amount)
	}

	if payment.Status != "confirmed" {
		t.Errorf("Expected status confirmed, got %s", payment.Status)
	}

	// Test transaction not found
	_, err = client.VerifyPaymentTransaction("nonexistent", 1000, "recipient")
	if err == nil {
		t.Error("Expected error for nonexistent transaction")
	}

	// Test amount mismatch
	_, err = client.VerifyPaymentTransaction("tx123", 2000, "recipient")
	if err == nil {
		t.Error("Expected error for amount mismatch")
	}

	// Test recipient mismatch
	_, err = client.VerifyPaymentTransaction("tx123", 1000, "wrongrecipient")
	if err == nil {
		t.Error("Expected error for recipient mismatch")
	}
}

func TestGetTransactionPool(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/txn_pool" {
			pool := []*Transaction{
				{
					TransactionHash: "pending_tx",
					From:            "sender",
					To:              "recipient",
					Value:           500,
					Timestamp:       time.Now().Unix(),
				},
			}
			json.NewEncoder(w).Encode(pool)
		}
	}))
	defer server.Close()

	client, err := NewNRNClient(server.URL, false, "")
	if err != nil {
		t.Fatalf("Failed to create NRN client: %v", err)
	}

	pool, err := client.GetTransactionPool()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(pool) != 1 {
		t.Errorf("Expected 1 transaction in pool, got %d", len(pool))
	}

	if pool[0].TransactionHash != "pending_tx" {
		t.Errorf("Expected transaction hash pending_tx, got %s", pool[0].TransactionHash)
	}
}

func TestSubmitTransaction(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/transaction" && r.Method == "POST" {
			var tx Transaction
			if err := json.NewDecoder(r.Body).Decode(&tx); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			response := struct {
				TxHash string `json:"tx_hash"`
			}{
				TxHash: "submitted_tx_hash",
			}
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(response)
		}
	}))
	defer server.Close()

	client, err := NewNRNClient(server.URL, false, "")
	if err != nil {
		t.Fatalf("Failed to create NRN client: %v", err)
	}

	tx := &Transaction{
		TransactionHash: "test_tx",
		From:            "sender",
		To:              "recipient",
		Value:           1000,
		Timestamp:       time.Now().Unix(),
	}

	txHash, err := client.SubmitTransaction(tx)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if txHash != "submitted_tx_hash" {
		t.Errorf("Expected tx hash submitted_tx_hash, got %s", txHash)
	}
}

func TestGetAccountBalance(t *testing.T) {
	client, err := NewNRNClient("http://localhost:8080", false, "")
	if err != nil {
		t.Fatalf("Failed to create NRN client: %v", err)
	}

	// Test valid address
	balance, err := client.GetAccountBalance("valid_address")
	if err != nil {
		t.Fatalf("Expected no error for valid address, got %v", err)
	}

	if balance != 1000000 {
		t.Errorf("Expected balance 1000000, got %d", balance)
	}

	// Test empty address
	_, err = client.GetAccountBalance("")
	if err == nil {
		t.Error("Expected error for empty address")
	}
}

func TestBlockAndTransactionTypes(t *testing.T) {
	// Test Block struct
	block := Block{
		BlockNumber: 1,
		Transactions: []*Transaction{
			{
				TransactionHash: "tx1",
				From:            "alice",
				To:              "bob",
				Value:           100,
				Data:            []byte("test data"),
				Timestamp:       time.Now().Unix(),
				Signature:       []byte("signature"),
				PublicKey:       "pubkey",
				Type:            "transfer",
				Fee:             10,
				Status:          "confirmed",
			},
		},
		Timestamp: time.Now().Unix(),
		Hash:      "blockhash",
		PrevHash:  "prevhash",
	}

	// Test JSON marshaling/unmarshaling
	data, err := json.Marshal(block)
	if err != nil {
		t.Fatalf("Failed to marshal block: %v", err)
	}

	var unmarshaled Block
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal block: %v", err)
	}

	if unmarshaled.BlockNumber != block.BlockNumber {
		t.Errorf("Expected block number %d, got %d", block.BlockNumber, unmarshaled.BlockNumber)
	}

	if len(unmarshaled.Transactions) != len(block.Transactions) {
		t.Errorf("Expected %d transactions, got %d", len(block.Transactions), len(unmarshaled.Transactions))
	}

	if unmarshaled.Transactions[0].TransactionHash != block.Transactions[0].TransactionHash {
		t.Errorf("Expected transaction hash %s, got %s", block.Transactions[0].TransactionHash, unmarshaled.Transactions[0].TransactionHash)
	}
}