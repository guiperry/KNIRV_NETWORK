package transaction_turnserver

import (
	"encoding/json"
	"fmt"
	"log"
	"time"
)

// MockTxPoolAdapter implements the TxSubmitter interface for testing
// when a real blockchain transaction pool is not available
//
// Deprecated: Use BlockchainAdapter with NewBlockchainAdapterWithBlockchain instead.
// This mock adapter is only kept for backward compatibility and testing.
type MockTxPoolAdapter struct {
	minerAddress string
}

// NewMockTxPoolAdapter creates a new mock adapter
//
// Deprecated: Use NewBlockchainAdapterWithBlockchain instead for production code.
// This function is only kept for backward compatibility and testing.
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

// SubmitNRNMintTx implements the TxSubmitter interface
func (m *MockTxPoolAdapter) SubmitNRNMintTx(recipient, amount, reason, proofID string) error {
	log.Printf("MOCK NRN MINT TRANSACTION:")
	log.Printf("  Recipient: %s", recipient)
	log.Printf("  Amount: %s", amount)
	log.Printf("  Reason: %s", reason)
	log.Printf("  ProofID: %s", proofID)
	log.Printf("Transaction would be sent from 'TURN_SERVER' to '%s'", m.minerAddress)

	return nil
}

// SubmitConnectivityProofReward implements the TxSubmitter interface
func (m *MockTxPoolAdapter) SubmitConnectivityProofReward(nodeID, proofID string, score float64, amount string) error {
	log.Printf("MOCK CONNECTIVITY PROOF REWARD TRANSACTION:")
	log.Printf("  NodeID: %s", nodeID)
	log.Printf("  ProofID: %s", proofID)
	log.Printf("  Score: %.2f", score)
	log.Printf("  Amount: %s", amount)
	log.Printf("Transaction would be sent from 'TURN_SERVER' to '%s'", m.minerAddress)

	return nil
}

// SubmitParticipationReward implements the TxSubmitter interface
func (m *MockTxPoolAdapter) SubmitParticipationReward(nodeID, participationType, amount string) error {
	log.Printf("MOCK PARTICIPATION REWARD TRANSACTION:")
	log.Printf("  NodeID: %s", nodeID)
	log.Printf("  ParticipationType: %s", participationType)
	log.Printf("  Amount: %s", amount)
	log.Printf("Transaction would be sent from 'TURN_SERVER' to '%s'", m.minerAddress)

	return nil
}

// GetMintingStats implements the TxSubmitter interface
func (m *MockTxPoolAdapter) GetMintingStats() map[string]interface{} {
	return map[string]interface{}{
		"total_turn_sessions":         42,
		"total_nrn_minted":            "1000000000000000000000", // 1000 NRN in wei
		"total_connectivity_rewards":  "500000000000000000000",  // 500 NRN in wei
		"total_participation_rewards": "250000000000000000000",  // 250 NRN in wei
		"mock_adapter":                true,
		"miner_address":               m.minerAddress,
	}
}
