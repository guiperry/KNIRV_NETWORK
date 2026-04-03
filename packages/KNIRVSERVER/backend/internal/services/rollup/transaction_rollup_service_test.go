package rollup

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

type stubChainReader struct {
	snapshot *ChainSnapshot
}

func (r *stubChainReader) GetChainSnapshot(context.Context) (*ChainSnapshot, error) {
	return r.snapshot, nil
}

type stubOracleClient struct{}

func (c *stubOracleClient) SubmitRollup(batch *RollupBatch) (string, error) {
	return "settlement-" + batch.ID, nil
}

func (c *stubOracleClient) GetRollup(string) (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}

func (c *stubOracleClient) FinalizeRollup(string) error {
	return nil
}

func (c *stubOracleClient) DisputeRollup(string, string) error {
	return nil
}

type reconcilingOracleClient struct {
	status  string
	reason  string
	records map[string]map[string]interface{}
}

func (c *reconcilingOracleClient) SubmitRollup(batch *RollupBatch) (string, error) {
	if c.records == nil {
		c.records = make(map[string]map[string]interface{})
	}
	c.records[batch.ID] = map[string]interface{}{
		"id":     batch.ID,
		"status": c.status,
	}
	if c.status == "disputed" && c.reason != "" {
		c.records[batch.ID]["dispute"] = c.reason
	}
	if c.status == "finalized" {
		c.records[batch.ID]["finalized_at"] = time.Now().UTC().Format(time.RFC3339)
	}
	return "settlement-" + batch.ID, nil
}

func (c *reconcilingOracleClient) GetRollup(id string) (map[string]interface{}, error) {
	return c.records[id], nil
}

func (c *reconcilingOracleClient) FinalizeRollup(string) error {
	return nil
}

func (c *reconcilingOracleClient) DisputeRollup(string, string) error {
	return nil
}

func TestServicePersistsAndReloadsState(t *testing.T) {
	t.Parallel()

	snapshot := &ChainSnapshot{
		ChainID: "knirv-transaction-1",
		Height:  1,
		Blocks: []BlockRecord{
			{
				BlockNumber: 1,
				Hash:        "block-1",
				Transactions: []TransactionRecord{
					{TransactionHash: "tx-1", Amount: 10},
				},
			},
		},
	}

	statePath := filepath.Join(t.TempDir(), "rollups.json")
	service := NewService(&stubChainReader{snapshot: snapshot}, &stubOracleClient{})
	if err := service.SetPersistencePath(statePath); err != nil {
		t.Fatalf("SetPersistencePath failed: %v", err)
	}

	batch, err := service.BuildNextBatch(context.Background())
	if err != nil {
		t.Fatalf("BuildNextBatch failed: %v", err)
	}
	if batch == nil {
		t.Fatal("expected batch to be built")
	}

	submitted, err := service.SubmitBatch(context.Background(), batch.ID)
	if err != nil {
		t.Fatalf("SubmitBatch failed: %v", err)
	}
	if submitted.Status != StatusSubmitted {
		t.Fatalf("expected submitted status, got %s", submitted.Status)
	}

	reloaded := NewService(&stubChainReader{snapshot: snapshot}, &stubOracleClient{})
	if err := reloaded.SetPersistencePath(statePath); err != nil {
		t.Fatalf("reloaded SetPersistencePath failed: %v", err)
	}

	status := reloaded.GetStatus()
	if status.LastProcessed != 1 {
		t.Fatalf("expected last processed 1, got %d", status.LastProcessed)
	}

	persistedBatch, ok := reloaded.GetBatch(batch.ID)
	if !ok {
		t.Fatalf("expected batch %s to be reloaded", batch.ID)
	}
	if persistedBatch.Status != StatusSubmitted {
		t.Fatalf("expected persisted batch status %s, got %s", StatusSubmitted, persistedBatch.Status)
	}
}

func TestServiceReconcilesOracleStatuses(t *testing.T) {
	t.Parallel()

	snapshot := &ChainSnapshot{
		ChainID: "knirv-transaction-1",
		Height:  1,
		Blocks: []BlockRecord{
			{BlockNumber: 1, Hash: "block-1"},
		},
	}

	tests := []struct {
		name           string
		oracleStatus   string
		disputeReason  string
		expectedStatus BatchStatus
		expectedError  string
	}{
		{
			name:           "finalized becomes settled",
			oracleStatus:   "finalized",
			expectedStatus: StatusSettled,
		},
		{
			name:           "disputed becomes disputed",
			oracleStatus:   "disputed",
			disputeReason:  "invalid batch root",
			expectedStatus: StatusDisputed,
			expectedError:  "invalid batch root",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			oracleClient := &reconcilingOracleClient{status: tc.oracleStatus, reason: tc.disputeReason}
			service := NewService(&stubChainReader{snapshot: snapshot}, oracleClient)

			batch, err := service.BuildNextBatch(context.Background())
			if err != nil {
				t.Fatalf("BuildNextBatch failed: %v", err)
			}
			if _, err := service.SubmitBatch(context.Background(), batch.ID); err != nil {
				t.Fatalf("SubmitBatch failed: %v", err)
			}

			updated, err := service.ReconcileWithOracle(context.Background())
			if err != nil {
				t.Fatalf("ReconcileWithOracle failed: %v", err)
			}
			if updated != 1 {
				t.Fatalf("expected one updated batch, got %d", updated)
			}

			reconciled, ok := service.GetBatch(batch.ID)
			if !ok {
				t.Fatalf("expected batch %s to exist", batch.ID)
			}
			if reconciled.Status != tc.expectedStatus {
				t.Fatalf("expected status %s, got %s", tc.expectedStatus, reconciled.Status)
			}
			if tc.expectedError != "" && reconciled.Error != tc.expectedError {
				t.Fatalf("expected dispute reason %q, got %q", tc.expectedError, reconciled.Error)
			}
		})
	}
}
