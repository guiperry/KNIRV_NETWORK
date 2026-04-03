package web

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"backend_server/internal/services/rollup"

	"github.com/gorilla/mux"
)

type testChainReader struct {
	snapshot *rollup.ChainSnapshot
}

func (r *testChainReader) GetChainSnapshot(context.Context) (*rollup.ChainSnapshot, error) {
	return r.snapshot, nil
}

type testOracleClient struct{}

func (c *testOracleClient) SubmitRollup(batch *rollup.RollupBatch) (string, error) {
	return "oracle-" + batch.ID, nil
}

func (c *testOracleClient) GetRollup(string) (map[string]interface{}, error) {
	return nil, nil
}

func (c *testOracleClient) FinalizeRollup(string) error {
	return nil
}

func (c *testOracleClient) DisputeRollup(string, string) error {
	return nil
}

func TestRollupHandlersStatusAndBatchRoutes(t *testing.T) {
	service := rollup.NewService(&testChainReader{
		snapshot: &rollup.ChainSnapshot{
			ChainID: "knirv-transaction-1",
			Height:  2,
			Blocks: []rollup.BlockRecord{
				{
					BlockNumber: 1,
					Hash:        "block-1",
					Transactions: []rollup.TransactionRecord{
						{TransactionHash: "tx-1", Amount: 1},
					},
				},
				{
					BlockNumber: 2,
					Hash:        "block-2",
					Transactions: []rollup.TransactionRecord{
						{TransactionHash: "tx-2", Amount: 2},
					},
				},
			},
		},
	}, &testOracleClient{})

	batch, err := service.BuildNextBatch(context.Background())
	if err != nil {
		t.Fatalf("BuildNextBatch failed: %v", err)
	}
	if batch == nil {
		t.Fatal("expected a batch to be built")
	}
	if _, err := service.SubmitBatch(context.Background(), batch.ID); err != nil {
		t.Fatalf("SubmitBatch failed: %v", err)
	}

	handlers := NewRollupHandlers(service, 45*time.Second)
	router := mux.NewRouter()
	handlers.RegisterRoutes(router, nil)

	statusReq := httptest.NewRequest(http.MethodGet, "/api/rollups/status", nil)
	statusResp := httptest.NewRecorder()
	router.ServeHTTP(statusResp, statusReq)

	if statusResp.Code != http.StatusOK {
		t.Fatalf("expected status route 200, got %d", statusResp.Code)
	}

	var statusPayload RollupResponse
	if err := json.Unmarshal(statusResp.Body.Bytes(), &statusPayload); err != nil {
		t.Fatalf("failed to decode status response: %v", err)
	}
	if !statusPayload.Success {
		t.Fatalf("expected success status response, got %+v", statusPayload)
	}

	batchReq := httptest.NewRequest(http.MethodGet, "/api/rollups/"+batch.ID, nil)
	batchResp := httptest.NewRecorder()
	router.ServeHTTP(batchResp, batchReq)

	if batchResp.Code != http.StatusOK {
		t.Fatalf("expected batch route 200, got %d", batchResp.Code)
	}
}
