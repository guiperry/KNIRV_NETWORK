package transaction_turnserver

import (
	"encoding/json"
	"fmt"
	"log"
	"time"
)

// MockTxPoolAdapter implements the TxSubmitter interface for testing
// when a real blockchain transaction pool is not available
type MockTxPoolAdapter struct {
	minerAddress string
}

// NewMockTxPoolAdapter creates a new mock adapter
func NewMockTxPoolAdapter(minerAddress string) *MockTxPoolAdapter {
	return &MockTxPoolAdapter{
		minerAddress: minerAddress,
	}
}

// SubmitTurnSessionTx implements the TxSubmitter interface
func (m *MockTxPoolAdapter) SubmitTurnSessionTx(sessionData map[string]interface{}) error {
	// Add additional metadata
	sessionData["recorded_at"] = time.Now().UTC().Format(time.RFC3339)
	sessionData["mock"] = true
	
	// Convert the session data to JSON for logging
	jsonData, err := json.MarshalIndent(sessionData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal session data: %w", err)
	}
	
	// Log the transaction data (in a real implementation, this would be submitted to the blockchain)
	log.Printf("MOCK TURN SESSION TRANSACTION:\n%s", string(jsonData))
	log.Printf("Transaction would be sent from 'TURN_SERVER' to '%s'", m.minerAddress)
	
	return nil
}