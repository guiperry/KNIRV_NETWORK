package transaction_turnserver

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"KNIRVROUTER/internal/constants"
	"KNIRVROUTER/internal/types"
)

// BlockchainInterface defines the methods we need from the blockchain
type BlockchainInterface interface {
	AddTransactionToTransactionPool(transaction *types.Transaction) error
	GetTransactionPool() []*types.Transaction
}

// BlockchainAdapter implements the TxSubmitter interface to connect
// the TURN server with the blockchain's transaction pool
type BlockchainAdapter struct {
	// Reference to the blockchain instance
	blockchain   BlockchainInterface
	minerAddress string

	// Legacy support for function-based submission (deprecated)
	txPoolSubmitFunc func(from, to string, data []byte) error
}

// NewBlockchainAdapterWithBlockchain creates a new adapter with a blockchain instance
func NewBlockchainAdapterWithBlockchain(
	blockchain BlockchainInterface,
	minerAddress string,
) *BlockchainAdapter {
	return &BlockchainAdapter{
		blockchain:   blockchain,
		minerAddress: minerAddress,
	}
}

// NewBlockchainAdapter creates a new adapter with the given transaction submission function (deprecated)
// Use NewBlockchainAdapterWithBlockchain for new implementations
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
	// Add additional metadata
	sessionData["recorded_at"] = time.Now().UTC().Format(time.RFC3339)

	// Convert the session data to JSON
	jsonData, err := json.Marshal(sessionData)
	if err != nil {
		return fmt.Errorf("failed to marshal session data: %w", err)
	}

	// Use blockchain interface if available (preferred)
	if a.blockchain != nil {
		// Create a proper blockchain transaction
		transaction := types.NewTransaction(
			"TURN_SERVER",
			a.minerAddress,
			0, // No value transfer for TURN session data
			jsonData,
			constants.ORIGIN_PUBLIC,
		)

		// Submit to blockchain
		err = a.blockchain.AddTransactionToTransactionPool(transaction)
		if err != nil {
			return fmt.Errorf("failed to submit transaction to blockchain: %w", err)
		}

		log.Printf("TURN session transaction submitted to blockchain for client %s",
			sessionData["client_addr"])
		return nil
	}

	// Fall back to legacy function-based submission
	if a.txPoolSubmitFunc == nil {
		return fmt.Errorf("neither blockchain interface nor transaction submission function is set")
	}

	// Submit using legacy function
	err = a.txPoolSubmitFunc("TURN_SERVER", a.minerAddress, jsonData)
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

	// Use blockchain interface if available (preferred)
	if a.blockchain != nil {
		// Create a proper blockchain transaction
		transaction := types.NewTransaction(
			"NRN_MINTER",
			recipient,
			0, // Value will be handled by smart contract logic
			jsonData,
			constants.ORIGIN_PUBLIC,
		)

		// Submit to blockchain
		err = a.blockchain.AddTransactionToTransactionPool(transaction)
		if err != nil {
			return fmt.Errorf("failed to submit NRN mint transaction to blockchain: %w", err)
		}

		log.Printf("NRN mint transaction submitted to blockchain: recipient=%s, amount=%s, reason=%s",
			recipient, amount, reason)
		return nil
	}

	// Fall back to legacy function-based submission
	if a.txPoolSubmitFunc == nil {
		return fmt.Errorf("neither blockchain interface nor transaction submission function is set")
	}

	// Submit using legacy function
	err = a.txPoolSubmitFunc("NRN_MINTER", recipient, jsonData)
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
	stats := map[string]interface{}{
		"minter_address":  "NRN_MINTER",
		"minting_enabled": true,
		"last_updated":    time.Now(),
	}

	// If we have a blockchain interface, get real statistics
	if a.blockchain != nil {
		txPool := a.blockchain.GetTransactionPool()

		// Count different types of transactions
		turnSessions := 0
		nrnMints := 0
		connectivityRewards := 0
		participationRewards := 0

		for _, tx := range txPool {
			switch tx.From {
			case "TURN_SERVER":
				turnSessions++
			case "NRN_MINTER":
				nrnMints++

				// Try to parse the transaction data to categorize
				var mintRequest NRNMintRequest
				if err := json.Unmarshal(tx.Data, &mintRequest); err == nil {
					if mintRequest.Reason != "" {
						if len(mintRequest.Reason) >= 20 && mintRequest.Reason[:20] == "connectivity_proof_r" { // "connectivity_proof_reward_score_"
							connectivityRewards++
						} else if len(mintRequest.Reason) >= 19 && mintRequest.Reason[:19] == "participation_rewar" { // "participation_reward_"
							participationRewards++
						}
					}
				}
			}
		}

		stats["total_turn_sessions"] = turnSessions
		stats["total_nrn_mints"] = nrnMints
		stats["total_connectivity_rewards"] = connectivityRewards
		stats["total_participation_rewards"] = participationRewards
		stats["transaction_pool_size"] = len(txPool)
		stats["data_source"] = "blockchain"
	} else {
		// Return placeholder data for legacy mode
		stats["total_turn_sessions"] = 0
		stats["total_nrn_mints"] = 0
		stats["total_connectivity_rewards"] = 0
		stats["total_participation_rewards"] = 0
		stats["transaction_pool_size"] = 0
		stats["data_source"] = "placeholder"
	}

	return stats
}
