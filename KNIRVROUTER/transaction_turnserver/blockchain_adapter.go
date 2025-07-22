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
	minerAddress    string
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