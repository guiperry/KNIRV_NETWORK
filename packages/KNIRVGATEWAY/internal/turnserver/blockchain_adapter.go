package turnserver

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"
)

type BlockchainInterface interface {
	AddTransactionToTransactionPool(transaction *Transaction) error
	GetTransactionPool() []*Transaction
}

type TxSubmitter interface {
	SubmitTurnSessionTx(sessionData map[string]interface{}) error
	SubmitNRNMintTx(recipient, amount, reason, proofID string) error
	SubmitConnectivityProofReward(nodeID, proofID string, score float64, amount string) error
	SubmitParticipationReward(nodeID, participationType, amount string) error
	GetMintingStats() map[string]interface{}
}

type BlockchainAdapter struct {
	blockchain       BlockchainInterface
	minerAddress     string
	txPoolSubmitFunc func(from, to string, data []byte) error

	statsMutex         sync.RWMutex
	sessionCount       int64
	mintCount          int64
	connectivityCount  int64
	participationCount int64
}

func NewBlockchainAdapter(blockchain BlockchainInterface, minerAddress string) *BlockchainAdapter {
	return &BlockchainAdapter{
		blockchain:   blockchain,
		minerAddress: minerAddress,
	}
}

func (a *BlockchainAdapter) SubmitTurnSessionTx(sessionData map[string]interface{}) error {
	sessionData["recorded_at"] = time.Now().UTC().Format(time.RFC3339)

	jsonData, err := json.Marshal(sessionData)
	if err != nil {
		return fmt.Errorf("failed to marshal session data: %w", err)
	}

	if a.blockchain != nil {
		transaction := &Transaction{
			From:      "TURN_SERVER",
			To:        a.minerAddress,
			Value:     0,
			Data:      jsonData,
			Status:    "pending",
			Timestamp: time.Now().UnixNano(),
			Origin:    OriginPublic,
		}

		err = a.blockchain.AddTransactionToTransactionPool(transaction)
		if err != nil {
			return fmt.Errorf("failed to submit transaction to blockchain: %w", err)
		}

		a.statsMutex.Lock()
		a.sessionCount++
		a.statsMutex.Unlock()

		log.Printf("TURN session transaction submitted to blockchain for client %s",
			sessionData["client_addr"])
		return nil
	}

	if a.txPoolSubmitFunc != nil {
		err = a.txPoolSubmitFunc("TURN_SERVER", a.minerAddress, jsonData)
		if err != nil {
			return fmt.Errorf("failed to submit transaction: %w", err)
		}
	}

	a.statsMutex.Lock()
	a.sessionCount++
	a.statsMutex.Unlock()

	log.Printf("TURN session recorded (mock mode) for client %s",
		sessionData["client_addr"])

	return nil
}

func (a *BlockchainAdapter) SubmitNRNMintTx(recipient, amount, reason, proofID string) error {
	mintRequest := NRNMintRequest{
		Recipient: recipient,
		Amount:    amount,
		Reason:    reason,
		ProofID:   proofID,
		Timestamp: time.Now().Unix(),
	}

	jsonData, err := json.Marshal(mintRequest)
	if err != nil {
		return fmt.Errorf("failed to marshal mint request: %w", err)
	}

	if a.blockchain != nil {
		transaction := &Transaction{
			From:      "NRN_MINTER",
			To:        recipient,
			Value:     0,
			Data:      jsonData,
			Status:    "pending",
			Timestamp: time.Now().UnixNano(),
			Origin:    OriginPublic,
		}

		err = a.blockchain.AddTransactionToTransactionPool(transaction)
		if err != nil {
			return fmt.Errorf("failed to submit NRN mint transaction to blockchain: %w", err)
		}

		a.statsMutex.Lock()
		a.mintCount++
		a.statsMutex.Unlock()

		log.Printf("NRN mint transaction submitted to blockchain: recipient=%s, amount=%s, reason=%s",
			recipient, amount, reason)
		return nil
	}

	a.statsMutex.Lock()
	a.mintCount++
	a.statsMutex.Unlock()

	log.Printf("NRN mint recorded (mock mode): recipient=%s, amount=%s, reason=%s",
		recipient, amount, reason)

	return nil
}

func (a *BlockchainAdapter) SubmitConnectivityProofReward(nodeID, proofID string, score float64, amount string) error {
	reason := fmt.Sprintf("connectivity_proof_reward_score_%.2f", score)
	return a.SubmitNRNMintTx(nodeID, amount, reason, proofID)
}

func (a *BlockchainAdapter) SubmitParticipationReward(nodeID, participationType, amount string) error {
	reason := fmt.Sprintf("participation_reward_%s", participationType)
	return a.SubmitNRNMintTx(nodeID, amount, reason, "")
}

func (a *BlockchainAdapter) GetMintingStats() map[string]interface{} {
	a.statsMutex.RLock()
	defer a.statsMutex.RUnlock()

	stats := map[string]interface{}{
		"minter_address":              "NRN_MINTER",
		"minting_enabled":             true,
		"last_updated":                time.Now(),
		"total_turn_sessions":         a.sessionCount,
		"total_nrn_mints":             a.mintCount,
		"total_connectivity_rewards":  a.connectivityCount,
		"total_participation_rewards": a.participationCount,
		"transaction_pool_size":       0,
		"data_source":                 "gateway",
	}

	if a.blockchain != nil {
		txPool := a.blockchain.GetTransactionPool()
		stats["transaction_pool_size"] = len(txPool)
		stats["data_source"] = "blockchain"
	}

	return stats
}

type MockBlockchainAdapter struct {
	transactions []*Transaction
	mu           sync.Mutex
}

func NewMockBlockchainAdapter() *MockBlockchainAdapter {
	return &MockBlockchainAdapter{
		transactions: make([]*Transaction, 0),
	}
}

func (m *MockBlockchainAdapter) AddTransactionToTransactionPool(transaction *Transaction) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.transactions = append(m.transactions, transaction)
	return nil
}

func (m *MockBlockchainAdapter) GetTransactionPool() []*Transaction {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.transactions
}
