package transaction_turnserver

import (
	"encoding/json"
	"fmt"
	"log"
	"time"
)

// BlockchainAdapter implements the TxSubmitter interface to connect
// the TURN server with the blockchain's transaction pool
type BlockchainAdapter struct {
	// Reference to the blockchain's transaction pool
	// This will be set when the adapter is created
	txPoolSubmitFunc func(from, to string, data []byte) error
	minerAddress     string
}

// NewBlockchainAdapter creates a new adapter with the given transaction submission function
func NewBlockchainAdapter(
	txPoolSubmitFunc func(from, to string, data []byte) error,
	minerAddress string,
) *BlockchainAdapter {
	return &BlockchainAdapter{
		txPoolSubmitFunc: txPoolSubmitFunc,
		minerAddress:     minerAddress,
	}
}

// SubmitTurnSessionTx implements the TxSubmitter interface
func (a *BlockchainAdapter) SubmitTurnSessionTx(sessionData map[string]interface{}) error {
	if a.txPoolSubmitFunc == nil {
		return fmt.Errorf("transaction submission function not set")
	}

	// Add additional metadata
	sessionData["recorded_at"] = time.Now().UTC().Format(time.RFC3339)

	// Convert the session data to JSON
	jsonData, err := json.Marshal(sessionData)
	if err != nil {
		return fmt.Errorf("failed to marshal session data: %w", err)
	}

	// The TURN server is the "from" address
	fromAddress := "TURN_SERVER"

	// Submit to the blockchain using the provided function
	// The transaction goes to the miner's address
	err = a.txPoolSubmitFunc(fromAddress, a.minerAddress, jsonData)
	if err != nil {
		return fmt.Errorf("failed to submit transaction: %w", err)
	}

	log.Printf("TURN session transaction submitted successfully for client %s",
		sessionData["client_addr"])

	return nil
}

// NRNMintRequest represents a request to mint NRN tokens
type NRNMintRequest struct {
	Recipient string `json:"recipient"`
	Amount    string `json:"amount"`
	Reason    string `json:"reason"`
	ProofID   string `json:"proof_id,omitempty"`
	Timestamp int64  `json:"timestamp"`
}

// SubmitNRNMintTx submits a transaction to mint NRN tokens
func (a *BlockchainAdapter) SubmitNRNMintTx(recipient, amount, reason, proofID string) error {
	if a.txPoolSubmitFunc == nil {
		return fmt.Errorf("transaction submission function not set")
	}

	// Create mint request
	mintRequest := NRNMintRequest{
		Recipient: recipient,
		Amount:    amount,
		Reason:    reason,
		ProofID:   proofID,
		Timestamp: time.Now().Unix(),
	}

	// Convert to JSON
	jsonData, err := json.Marshal(mintRequest)
	if err != nil {
		return fmt.Errorf("failed to marshal mint request: %w", err)
	}

	// Submit mint transaction
	fromAddress := "NRN_MINTER"
	err = a.txPoolSubmitFunc(fromAddress, recipient, jsonData)
	if err != nil {
		return fmt.Errorf("failed to submit NRN mint transaction: %w", err)
	}

	log.Printf("NRN mint transaction submitted: recipient=%s, amount=%s, reason=%s",
		recipient, amount, reason)

	return nil
}

// SubmitConnectivityProofReward submits a transaction to reward connectivity proof
func (a *BlockchainAdapter) SubmitConnectivityProofReward(nodeID, proofID string, score float64, amount string) error {
	reason := fmt.Sprintf("connectivity_proof_reward_score_%.2f", score)
	return a.SubmitNRNMintTx(nodeID, amount, reason, proofID)
}

// SubmitParticipationReward submits a transaction to reward network participation
func (a *BlockchainAdapter) SubmitParticipationReward(nodeID, participationType, amount string) error {
	reason := fmt.Sprintf("participation_reward_%s", participationType)
	return a.SubmitNRNMintTx(nodeID, amount, reason, "")
}

// GetMintingStats returns statistics about NRN minting
func (a *BlockchainAdapter) GetMintingStats() map[string]interface{} {
	// In a real implementation, this would query the blockchain for minting statistics
	// For now, return placeholder data
	return map[string]interface{}{
		"total_minted":     "0",
		"total_recipients": 0,
		"last_mint_time":   time.Now(),
		"minting_enabled":  true,
		"minter_address":   "NRN_MINTER",
	}
}
