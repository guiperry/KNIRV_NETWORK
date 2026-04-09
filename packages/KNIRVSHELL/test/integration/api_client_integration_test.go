package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/KNIRV/KNIRV_NETWORK/KNIRVSHELL/core"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockKNIRVCHAINServer creates a mock KNIRVCHAIN server for testing
func MockKNIRVCHAINServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/health":
			json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		case "/info":
			json.NewEncoder(w).Encode(core.NodeInfo{
				Version:     "1.0.0",
				NodeID:      "mock-node",
				StartTime:   time.Now().Add(-24 * time.Hour),
				Uptime:      "24h",
				NetworkID:   "mock-network",
				PeerCount:   10,
				SyncStatus:  "synced",
				BlockHeight: 1000,
			})
		case "/chain":
			json.NewEncoder(w).Encode(core.ChainInfo{
				NetworkID:       "mock-network",
				BlockHeight:     1000,
				LastBlockTime:   time.Now().Add(-1 * time.Minute),
				TotalTxns:       5000,
				PendingTxns:     10,
				ActiveCapCount:  100,
				ValidatorCount:  5,
				ConsensusStatus: "active",
			})
		case "/block/1000":
			json.NewEncoder(w).Encode(core.BlockInfo{
				Height:       1000,
				Hash:         "mock-hash",
				PreviousHash: "mock-prev-hash",
				Timestamp:    time.Now().Add(-1 * time.Minute),
				TxnCount:     10,
				Size:         1024,
				Proposer:     "mock-proposer",
				Transactions: []string{"txn1", "txn2", "txn3"},
			})
		case "/block/hash/mock-hash":
			json.NewEncoder(w).Encode(core.BlockInfo{
				Height:       1000,
				Hash:         "mock-hash",
				PreviousHash: "mock-prev-hash",
				Timestamp:    time.Now().Add(-1 * time.Minute),
				TxnCount:     10,
				Size:         1024,
				Proposer:     "mock-proposer",
				Transactions: []string{"txn1", "txn2", "txn3"},
			})
		case "/transaction/txn1":
			json.NewEncoder(w).Encode(core.TransactionInfo{
				Hash:        "txn1",
				BlockHeight: 1000,
				BlockHash:   "mock-hash",
				From:        "mock-sender",
				To:          "mock-receiver",
				Value:       "100",
				Fee:         "10",
				Timestamp:   time.Now().Add(-1 * time.Minute),
				Status:      "confirmed",
				Type:        "transfer",
				Data:        "mock-data",
			})
		case "/txn_pool":
			json.NewEncoder(w).Encode(core.TransactionPoolInfo{
				PendingCount: 10,
				QueuedCount:  5,
				Transactions: []core.TransactionInfo{
					{
						Hash:      "txn-pending",
						From:      "mock-sender",
						To:        "mock-receiver",
						Value:     "100",
						Fee:       "10",
						Timestamp: time.Now(),
						Status:    "pending",
						Type:      "transfer",
						Data:      "mock-data",
					},
				},
			})
		case "/transaction":
			// Handle transaction submission
			var request core.SubmitTransactionRequest
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]string{"error": "invalid request"})
				return
			}

			json.NewEncoder(w).Encode(core.SubmitTransactionResponse{
				TransactionHash: "mock-txn-hash",
				Status:          "pending",
			})
		case "/mcp/capability/prepare_registration":
			// Handle capability registration preparation
			var request core.MCPPrepareCapabilityRegistrationRequest
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]string{"error": "invalid request"})
				return
			}

			json.NewEncoder(w).Encode(core.MCPPrepareCapabilityRegistrationResponse{
				CapabilityID: "mock-cap-id",
				TxnData:      "mock-txn-data",
				Fee:          "100",
				Descriptor:   request.Descriptor,
			})
		default:
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "not found"})
		}
	}))
}

func TestAPIClientIntegration(t *testing.T) {
	// Create mock server
	server := MockKNIRVCHAINServer()
	defer server.Close()

	// Create logger
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	// Create API client
	client := core.NewAPIClient(
		server.URL,
		core.WithTimeout(5*time.Second),
		core.WithRetries(2),
		core.WithLogger(logger),
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
		assert.Equal(t, "mock-node", info.NodeID)
		assert.Equal(t, "mock-network", info.NetworkID)
		assert.Equal(t, uint64(1000), info.BlockHeight)
	})

	t.Run("GetChainInfo", func(t *testing.T) {
		info, err := client.GetChainInfo(ctx)
		require.NoError(t, err)
		assert.Equal(t, "mock-network", info.NetworkID)
		assert.Equal(t, uint64(1000), info.BlockHeight)
		assert.Equal(t, uint64(5000), info.TotalTxns)
		assert.Equal(t, uint64(10), info.PendingTxns)
	})

	t.Run("GetBlockInfo", func(t *testing.T) {
		info, err := client.GetBlockInfo(ctx, 1000)
		require.NoError(t, err)
		assert.Equal(t, uint64(1000), info.Height)
		assert.Equal(t, "mock-hash", info.Hash)
		assert.Equal(t, "mock-prev-hash", info.PreviousHash)
		assert.Equal(t, 10, info.TxnCount)
	})

	t.Run("GetBlockByHash", func(t *testing.T) {
		info, err := client.GetBlockByHash(ctx, "mock-hash")
		require.NoError(t, err)
		assert.Equal(t, uint64(1000), info.Height)
		assert.Equal(t, "mock-hash", info.Hash)
		assert.Equal(t, "mock-prev-hash", info.PreviousHash)
		assert.Equal(t, 10, info.TxnCount)
	})

	t.Run("GetTransactionInfo", func(t *testing.T) {
		info, err := client.GetTransactionInfo(ctx, "txn1")
		require.NoError(t, err)
		assert.Equal(t, "txn1", info.Hash)
		assert.Equal(t, uint64(1000), info.BlockHeight)
		assert.Equal(t, "mock-hash", info.BlockHash)
		assert.Equal(t, "mock-sender", info.From)
		assert.Equal(t, "mock-receiver", info.To)
	})

	t.Run("GetTransactionPoolInfo", func(t *testing.T) {
		info, err := client.GetTransactionPoolInfo(ctx)
		require.NoError(t, err)
		assert.Equal(t, 10, info.PendingCount)
		assert.Equal(t, 5, info.QueuedCount)
		assert.Len(t, info.Transactions, 1)
		assert.Equal(t, "txn-pending", info.Transactions[0].Hash)
	})

	t.Run("SubmitTransaction", func(t *testing.T) {
		request := core.SubmitTransactionRequest{
			From:      "mock-sender",
			Data:      []byte("mock-data"),
			Signature: []byte("mock-sig"),
			Type:      "transfer",
		}
		txn := &core.Transaction{
			From:      request.From,
			Data:      request.Data,
			Signature: request.Signature,
			Type:      request.Type,
		}
		response, err := client.SubmitTransaction(ctx, txn)
		require.NoError(t, err)
		assert.Equal(t, "mock-txn-hash", response.TransactionHash)
		assert.Equal(t, "pending", response.Status)
	})

	t.Run("PrepareCapabilityRegistration", func(t *testing.T) {
		request := core.MCPPrepareCapabilityRegistrationRequest{
			Name:        "mock-cap",
			Version:     "1.0.0",
			Description: "Mock capability",
			Author:      "Mock Author",
			License:     "MIT",
			Descriptor: map[string]interface{}{
				"type": "tool",
			},
		}
		response, err := client.PrepareCapabilityRegistration(ctx, request, "mock-plugin.so", "mock-manifest.json")
		require.NoError(t, err)
		assert.Equal(t, "mock-cap-id", response.CapabilityID)
		assert.Equal(t, "mock-txn-data", response.TxnData)
		assert.Equal(t, "100", response.Fee)
		assert.NotNil(t, response.Descriptor)
		assert.NotNil(t, response.Descriptor["schema"])
	})
}
