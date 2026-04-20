package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"KNIRVCHAIN/config"
	"KNIRVCHAIN/internal/agent"
	"KNIRVCHAIN/internal/blockchain"
	"KNIRVCHAIN/internal/crypto"
	"KNIRVCHAIN/internal/database"
	"KNIRVCHAIN/internal/p2p"
	"KNIRVCHAIN/internal/wallet"
)

// Type aliases for easier use in tests
type Agent = agent.Agent
type BadgeAttachment = agent.BadgeAttachment
type AgentManager = agent.AgentManager
type ChromemManager = database.ChromemManager
type Wallet = agent.Wallet
type Capability = agent.Capability
type AgentRelationship = agent.AgentRelationship
type WalletManager = wallet.WalletManager

// Blockchain types
type BlockchainStruct = blockchain.BlockchainStruct
type BlockchainServer = blockchain.BlockchainServer
type Block = blockchain.Block
type Transaction = blockchain.Transaction
type LevelDB = blockchain.LevelDB
type MCPProcessor = blockchain.MCPProcessor
type ChromemSyncManager = blockchain.ChromemSyncManager
type ConsensusManager = agent.ConsensusManager

// Agent package types for consensus manager
type AgentBlockchainStruct = agent.BlockchainStruct
type AgentBlock = agent.Block

// P2P types
type BootnodeInfo = p2p.BootnodeInfo
type DiscoveryManager = p2p.DiscoveryManager

// Crypto types
type CerebrasEmbeddingClient = crypto.CerebrasEmbeddingClient

// Constructor function aliases
var NewAgentManager = agent.NewAgentManager

// NewChromemManager creates a new ChromemManager for tests.
func NewChromemManager(cfg *config.ChromemConfig) (*ChromemManager, error) {
	return database.NewChromemManager(cfg)
}

var NewWallet = wallet.NewWallet
var NewDiscoveryManager = p2p.NewDiscoveryManager
var NewCerebrasEmbeddingClient = crypto.NewCerebrasEmbeddingClient
var CreateCerebrasEmbeddingFunction = crypto.CreateCerebrasEmbeddingFunction
var FetchBootnodesFromRegistry = p2p.FetchBootnodesFromRegistry

// Blockchain functions
var NewBlock = blockchain.NewBlock
var NewMCPTransaction = blockchain.NewMCPTransaction
var NewBlockchain = blockchain.NewBlockchain
var NewLevelDB = database.NewLevelDB
var NewMCPProcessor = blockchain.NewMCPProcessor
var NewChromemSyncManager = blockchain.NewChromemSyncManager
var NewTransaction = blockchain.NewTransaction
var NewConsensusManager = agent.NewConsensusManager

// Wallet server functions
var NewWalletServer = wallet.NewWalletServer
var NewWalletManager = wallet.NewWalletManager

// Global variables
var BootnodeRegistryURL = p2p.BootnodeRegistryURL
var DiscoveryResourceTypeChain = p2p.DiscoveryResourceTypeChain

// Transaction type constants
var TransactionTypeMCPRegisterCapability = blockchain.TransactionTypeMCPRegisterCapability
var TransactionTypeMCPInvokeCapability = blockchain.TransactionTypeMCPInvokeCapability
var TransactionTypeMCPUpdateCapability = blockchain.TransactionTypeMCPUpdateCapability

// Transaction status constants
var TXN_VERIFICATION_SUCCESS = blockchain.TXN_VERIFICATION_SUCCESS
var TXN_VERIFICATION_FAILED = blockchain.TXN_VERIFICATION_FAILED

// Missing functions for PoAu-D tests
func ValidateTransactionForDelegation(tx *Transaction) bool {
	// Placeholder implementation for testing
	return tx.Type == TransactionTypeMCPInvokeCapability
}

func EnableDelegation(bc *BlockchainStruct) error {
	// Placeholder implementation for testing
	bc.PoAuDEnabled = true
	return nil
}

func DisableDelegation(bc *BlockchainStruct) error {
	// Placeholder implementation for testing
	bc.PoAuDEnabled = false
	return nil
}

var NewTransactionPoolManager = blockchain.NewTransactionPoolManager

// Missing functions for integration tests
func waitForShutdownSignal(cancel context.CancelFunc, wg *sync.WaitGroup, _ string, _ *config.Config, _ interface{}) {
	// Placeholder implementation for testing
	if cancel != nil {
		cancel()
	}
	if wg != nil {
		wg.Wait()
	}
}

// Missing functions for peer lifecycle tests
func GenerateChainURI(rootNodeURL string, desiredURI string) (string, string, string, error) {
	// Placeholder implementation for testing
	// Returns (extractedID, fullURI, txnHash, error)
	chainID := "test-chain-" + desiredURI
	fullURI := "knirv://" + chainID
	txnHash := "test-txn-hash"
	return chainID, fullURI, txnHash, nil
}

// Missing constants for P2P consensus tests
var NetworkControlTopic = "network-control"

// Type conversion functions for blockchain.Transaction <-> wallet.Transaction compatibility
func BlockchainToWalletTransaction(bcTx *blockchain.Transaction) *wallet.Transaction {
	if bcTx == nil {
		return nil
	}
	amount := big.NewInt(int64(bcTx.Value))
	timestamp := time.Unix(bcTx.Timestamp, 0)
	signature := fmt.Sprintf("%x", bcTx.Signature) // Convert []byte to hex string

	return &wallet.Transaction{
		From:      bcTx.From,
		To:        bcTx.To,
		Amount:    amount,
		Data:      bcTx.Data,
		Hash:      bcTx.TransactionHash,
		Timestamp: timestamp,
		Signature: signature,
	}
}

func WalletToBlockchainTransaction(walletTx *wallet.Transaction) *blockchain.Transaction {
	if walletTx == nil {
		return nil
	}
	value := uint64(walletTx.Amount.Uint64())
	timestamp := walletTx.Timestamp.Unix()
	signature, _ := hex.DecodeString(walletTx.Signature) // Convert hex string to []byte

	return &blockchain.Transaction{
		From:            walletTx.From,
		To:              walletTx.To,
		Value:           value,
		Data:            walletTx.Data,
		TransactionHash: walletTx.Hash,
		Timestamp:       timestamp,
		Signature:       signature,
		Status:          blockchain.TXN_VERIFICATION_SUCCESS, // Default status
	}
}

// Helper function to convert blockchain.Transaction to wallet.Transaction for signing
func ConvertForSigning(bcTx *blockchain.Transaction) wallet.Transaction {
	if bcTx == nil {
		return wallet.Transaction{}
	}
	amount := big.NewInt(int64(bcTx.Value))
	timestamp := time.Unix(bcTx.Timestamp, 0)
	signature := fmt.Sprintf("%x", bcTx.Signature) // Convert []byte to hex string

	return wallet.Transaction{
		From:      bcTx.From,
		To:        bcTx.To,
		Amount:    amount,
		Data:      bcTx.Data,
		Hash:      bcTx.TransactionHash,
		Timestamp: timestamp,
		Signature: signature,
	}
}

// setupTestEnvironment creates a test environment with temporary directories and initialized managers
func setupTestEnvironment(t *testing.T) (string, *ChromemManager, *wallet.WalletImpl, *AgentManager) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "agent_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}

	// Create ChromeDB config
	chromemConfig := &config.ChromemConfig{
		Path: filepath.Join(tempDir, "chromem_test"),
	}

	// Initialize ChromeManager
	chromemManager, err := NewChromemManager(chromemConfig)
	if err != nil {
		t.Fatalf("Failed to create ChromeManager: %v", err)
	}

	// Create test wallet using the actual wallet implementation
	walletImpl, err := NewWallet()
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}

	// Create AgentManager
	agentManager := NewAgentManager(agent.NewAgentChromemStore(chromemManager), nil, walletImpl, nil)

	return tempDir, chromemManager, walletImpl, agentManager
}

// ConvertBlockchainToAgent converts blockchain.BlockchainStruct to agent.BlockchainStruct
func ConvertBlockchainToAgent(bc *blockchain.BlockchainStruct) *agent.BlockchainStruct {
	// Convert blocks
	agentBlocks := make([]*agent.Block, len(bc.Blocks))
	for i, block := range bc.Blocks {
		// Convert transactions
		agentTxns := make([]*agent.Transaction, len(block.Transactions))
		for j, txn := range block.Transactions {
			agentTxns[j] = &agent.Transaction{
				TransactionHash: txn.TransactionHash,
				From:            txn.From,
				To:              txn.To,
				Type:            txn.Type,
				Data:            txn.Data,
				Fee:             int64(txn.Fee),
				Timestamp:       txn.Timestamp,
			}
		}

		agentBlocks[i] = &agent.Block{
			Hash:         block.Hash(), // Call the Hash() method
			BlockHash:    block.BlockHash,
			PrevHash:     block.PrevHash,
			BlockNumber:  int64(block.BlockNumber), // Convert uint64 to int64
			Timestamp:    block.Timestamp,
			Nonce:        int64(block.Nonce), // Convert int to int64
			Data:         []byte{},           // agent.Block has Data field, but blockchain.Block doesn't
			Transactions: agentTxns,
		}
	}

	// Convert transaction pool
	agentTxnPool := make([]*agent.Transaction, len(bc.TransactionPool))
	for i, txn := range bc.TransactionPool {
		agentTxnPool[i] = &agent.Transaction{
			TransactionHash: txn.TransactionHash,
			From:            txn.From,
			To:              txn.To,
			Type:            txn.Type,
			Data:            txn.Data,
			Fee:             int64(txn.Fee),
			Timestamp:       txn.Timestamp,
		}
	}

	return &agent.BlockchainStruct{
		Blocks:          agentBlocks,
		TransactionPool: agentTxnPool,
		PoAuDEnabled:    bc.PoAuDEnabled,
		ChainAddress:    bc.ChainAddress,
		Reflections:     bc.Reflections,
	}
}

// MockP2PConsensusManager implements p2p.ConsensusManager interface for testing
type MockP2PConsensusManager struct {
	miningLocked   bool
	updateRequired bool
}

func NewMockP2PConsensusManager() *MockP2PConsensusManager {
	return &MockP2PConsensusManager{
		miningLocked:   false,
		updateRequired: false,
	}
}

// Implement p2p.ConsensusManager interface
func (m *MockP2PConsensusManager) StartConsensus(ctx context.Context) error {
	return nil
}

func (m *MockP2PConsensusManager) StopConsensus() error {
	return nil
}

func (m *MockP2PConsensusManager) ProposeValue(value interface{}) error {
	return nil
}

func (m *MockP2PConsensusManager) VoteOnProposal(proposalID string, vote bool) error {
	return nil
}

func (m *MockP2PConsensusManager) GetConsensusState() p2p.ConsensusState {
	return p2p.ConsensusState{} // Return empty state
}

func (m *MockP2PConsensusManager) AddParticipant(peerID string) error {
	return nil
}

func (m *MockP2PConsensusManager) RemoveParticipant(peerID string) error {
	return nil
}

func (m *MockP2PConsensusManager) GetParticipants() []string {
	return []string{}
}

func (m *MockP2PConsensusManager) OnConsensusReached(handler p2p.ConsensusHandler) error {
	return nil
}

func (m *MockP2PConsensusManager) OnProposalReceived(handler p2p.ProposalHandler) error {
	return nil
}

// Mining state management - these are the important ones for ProofOfWorkMining
func (m *MockP2PConsensusManager) GetMiningLockState() bool {
	return m.miningLocked
}

func (m *MockP2PConsensusManager) GetUpdateRequired() bool {
	return m.updateRequired
}

// Helper methods for testing
func (m *MockP2PConsensusManager) SetMiningLocked(locked bool) {
	m.miningLocked = locked
}

func (m *MockP2PConsensusManager) SetUpdateRequired(required bool) {
	m.updateRequired = required
}
