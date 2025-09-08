package core

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPIClient(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"status":"ok"}`))
		case "/info":
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{
				"version": "1.0.0",
				"node_id": "test-node",
				"start_time": "2023-01-01T00:00:00Z",
				"uptime": "1h",
				"network_id": "test-network",
				"peer_count": 10,
				"sync_status": "synced",
				"block_height": 100
			}`))
		case "/chain":
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{
				"network_id": "test-network",
				"block_height": 100,
				"last_block_time": "2023-01-01T00:00:00Z",
				"total_txns": 1000,
				"pending_txns": 10,
				"active_cap_count": 50,
				"validator_count": 5,
				"consensus_status": "active"
			}`))
		case "/block/100":
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{
				"height": 100,
				"hash": "test-hash",
				"previous_hash": "prev-hash",
				"timestamp": "2023-01-01T00:00:00Z",
				"txn_count": 10,
				"size": 1000,
				"proposer": "test-proposer",
				"transactions": ["txn1", "txn2"]
			}`))
		case "/block/hash/test-hash":
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{
				"height": 100,
				"hash": "test-hash",
				"previous_hash": "prev-hash",
				"timestamp": "2023-01-01T00:00:00Z",
				"txn_count": 10,
				"size": 1000,
				"proposer": "test-proposer",
				"transactions": ["txn1", "txn2"]
			}`))
		case "/transaction/test-txn":
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{
				"hash": "test-txn",
				"block_height": 100,
				"block_hash": "test-hash",
				"from": "sender",
				"to": "receiver",
				"value": "100",
				"fee": "10",
				"timestamp": "2023-01-01T00:00:00Z",
				"status": "confirmed",
				"type": "transfer",
				"data": "test-data"
			}`))
		case "/txn_pool":
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{
				"pending_count": 10,
				"queued_count": 5,
				"transactions": [
					{
						"hash": "test-txn",
						"block_height": 0,
						"block_hash": "",
						"from": "sender",
						"to": "receiver",
						"value": "100",
						"fee": "10",
						"timestamp": "2023-01-01T00:00:00Z",
						"status": "pending",
						"type": "transfer",
						"data": "test-data"
					}
				]
			}`))
		case "/transaction":
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{
				"txn_hash": "test-txn",
				"status": "pending"
			}`))
		case "/mcp/capability/prepare_registration":
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{
				"capability_id": "test-cap",
				"txn_data": "test-data",
				"fee": "10",
				"descriptor": {
					"schema": {
						"manifestFile": "manifest.json",
						"executableFile": "plugin.so",
						"locationHints": ["file://manifest.json", "file://plugin.so"]
					}
				}
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

	t.Run("HealthCheck", func(t *testing.T) {
		err := client.HealthCheck(ctx)
		require.NoError(t, err)
	})

	t.Run("GetNodeInfo", func(t *testing.T) {
		info, err := client.GetNodeInfo(ctx)
		require.NoError(t, err)
		assert.Equal(t, "1.0.0", info.Version)
		assert.Equal(t, "test-node", info.NodeID)
		assert.Equal(t, "test-network", info.NetworkID)
		assert.Equal(t, uint64(100), info.BlockHeight)
	})

	t.Run("CheckVersionCompatibility", func(t *testing.T) {
		err := client.CheckVersionCompatibility(ctx, "1.0.0")
		require.NoError(t, err)
	})

	t.Run("GetChainInfo", func(t *testing.T) {
		info, err := client.GetChainInfo(ctx)
		require.NoError(t, err)
		assert.Equal(t, "test-network", info.NetworkID)
		assert.Equal(t, uint64(100), info.BlockHeight)
		assert.Equal(t, uint64(1000), info.TotalTxns)
		assert.Equal(t, uint64(10), info.PendingTxns)
	})

	t.Run("GetBlockInfo", func(t *testing.T) {
		info, err := client.GetBlockInfo(ctx, 100)
		require.NoError(t, err)
		assert.Equal(t, uint64(100), info.Height)
		assert.Equal(t, "test-hash", info.Hash)
		assert.Equal(t, "prev-hash", info.PreviousHash)
		assert.Equal(t, 10, info.TxnCount)
	})

	t.Run("GetBlockByHash", func(t *testing.T) {
		info, err := client.GetBlockByHash(ctx, "test-hash")
		require.NoError(t, err)
		assert.Equal(t, uint64(100), info.Height)
		assert.Equal(t, "test-hash", info.Hash)
		assert.Equal(t, "prev-hash", info.PreviousHash)
		assert.Equal(t, 10, info.TxnCount)
	})

	t.Run("GetTransactionInfo", func(t *testing.T) {
		info, err := client.GetTransactionInfo(ctx, "test-txn")
		require.NoError(t, err)
		assert.Equal(t, "test-txn", info.Hash)
		assert.Equal(t, uint64(100), info.BlockHeight)
		assert.Equal(t, "test-hash", info.BlockHash)
		assert.Equal(t, "sender", info.From)
		assert.Equal(t, "receiver", info.To)
	})

	t.Run("GetTransactionPoolInfo", func(t *testing.T) {
		info, err := client.GetTransactionPoolInfo(ctx)
		require.NoError(t, err)
		assert.Equal(t, 10, info.PendingCount)
		assert.Equal(t, 5, info.QueuedCount)
		assert.Len(t, info.Transactions, 1)
	})

	t.Run("SubmitTransaction", func(t *testing.T) {
		tx := &Transaction{
			From:            "sender",
			To:              "receiver",
			Value:           100,
			Data:            []byte("test-data"),
			Timestamp:       time.Now().Unix(),
			Fee:             10,
			Type:            "transfer",
			PublicKey:       "test-pubkey",
			Signature:       []byte("test-sig"),
			TransactionHash: "test-hash",
		}
		response, err := client.SubmitTransaction(ctx, tx)
		require.NoError(t, err)
		assert.Equal(t, "test-txn", response.TransactionHash)
		assert.Equal(t, "pending", response.Status)
	})
}
