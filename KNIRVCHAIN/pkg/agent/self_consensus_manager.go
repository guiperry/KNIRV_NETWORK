package agent

import (
	"KNIRVCHAIN/pkg/utils"
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

type ConsensusManager struct { // Implement consensus lock functionality.
	syncState      chan bool         `json:"-"` // Exclude from JSON serialization
	LongestChain   []*Block          `json:"longest_chain"`
	UpdateRequired bool              `json:"update_required"`
	Blockchain     *BlockchainStruct `json:"-"` // Exclude from JSON serialization to avoid circular references
	MiningLocked   bool              `json:"mining_locked"`
	stop           bool              `json:"-"` // Flag to stop consensus manager
	reflectionURLs map[string]bool   `json:"-"` // Store HTTP URLs from config
	selfURL        string            `json:"-"` // This node's own HTTP URL

	mu sync.Mutex `json:"-"` // Exclude from JSON serialization
}

func NewConsensusManager(blockchain *BlockchainStruct, urls []string, selfURL string) *ConsensusManager {
	refMap := make(map[string]bool)
	for _, u := range urls {
		// Basic validation/normalization could happen here
		if u != "" {
			refMap[u] = true // Initially assume active, ping will verify
		}
	}
	return &ConsensusManager{
		syncState:      make(chan bool),
		LongestChain:   []*Block{},
		UpdateRequired: false,
		Blockchain:     blockchain,
		MiningLocked:   false,
		reflectionURLs: refMap, // Store the map
		selfURL:        selfURL,
	}
}

func (cm *ConsensusManager) getUpdateRequired() bool {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	return cm.UpdateRequired
}

func (cm *ConsensusManager) setUpdateRequired(value bool) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.UpdateRequired = value
}

func (cm *ConsensusManager) setLongestChain(blocks []*Block) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Deep copy the blocks to avoid unintended modifications
	cm.LongestChain = make([]*Block, len(blocks))
	for i, block := range blocks {
		// Create a deep copy of the block
		cm.LongestChain[i] = &Block{
			BlockNumber:  block.BlockNumber,
			Timestamp:    block.Timestamp,
			PrevHash:     append([]byte{}, block.PrevHash...),
			BlockHash:    append([]byte{}, block.BlockHash...),
			Nonce:        block.Nonce,
			Transactions: make([]*Transaction, len(block.Transactions)),
			//Data:         block.Data,
		}

		// Deep copy each transaction in the block
		for j, txn := range block.Transactions {
			cm.LongestChain[i].Transactions[j] = txn.Clone()
		}
	}
}

func (cm *ConsensusManager) RunConsensus() {
	log.Println("Starting consensus manager...")

	for {
		// Pause between consensus rounds to avoid overwhelming the network
		time.Sleep(time.Duration(utils.CONSENSUS_PAUSE_TIME) * time.Second)

		// Skip consensus if there are no reflections
		if len(cm.reflectionURLs) == 0 {
			log.Println("No reflection URLs configured. Skipping consensus.")
			continue
		}

		// Wait for active mining to complete
		for cm.Blockchain.IsActivelyMining() {
			log.Println("Consensus waiting for active mining to complete...")
			time.Sleep(1 * time.Second)
		}

		// Lock mining during consensus
		cm.lockMining()
		log.Printf("Consensus now running at node URL: %s", cm.selfURL)
		log.Println("Checking configured reflection URLs:")

		activeReflections := make(map[string]bool) // Track currently active ones
		hasActiveReflection := false

		// Ping configured URLs
		for url := range cm.reflectionURLs {
			if url == cm.selfURL {
				continue
			} // Skip self

			// Check if the reflection is actually reachable with retries
			var resp *http.Response
			var err error
			maxRetries := 2
			retryDelay := 150 * time.Millisecond

			for i := 0; i <= maxRetries; i++ {
				client := &http.Client{Timeout: 3 * time.Second}
				resp, err = client.Get(url + "/ping")
				if err == nil {
					// Verify response status code
					if resp.StatusCode != http.StatusOK {
						resp.Body.Close()
						err = fmt.Errorf("invalid status code: %d", resp.StatusCode)
						continue
					}

					// Verify response content
					var pingResponse map[string]string
					if err = json.NewDecoder(resp.Body).Decode(&pingResponse); err != nil {
						resp.Body.Close()
						err = fmt.Errorf("invalid ping response: %v", err)
						continue
					}

					if pingResponse["status"] != "ok" {
						resp.Body.Close()
						err = fmt.Errorf("invalid status in response: %s", pingResponse["status"])
						continue
					}

					resp.Body.Close()
					break
				}

				if i < maxRetries {
					log.Printf("  - Reflection %s ping attempt %d failed: %v - retrying...", url, i+1, err)
					time.Sleep(retryDelay)
				}
			}

			if err != nil {
				log.Printf("  - Reflection %s is inactive: %v", url, err)
				// Don't modify cm.reflectionURLs here, just skip for this round
				continue
			}

			// Mark the reflection as active
			log.Printf("  - Reflection %s is active.", url)
			activeReflections[url] = true // Add to map of active ones for this round
			hasActiveReflection = true
		}

		if !hasActiveReflection {
			log.Println("No active reflection chains found among configured URLs. Skipping consensus.")
			cm.unlockMining()
			continue
		}

		// Find the reflection with the longest chain using parallel requests
		type chainResult struct {
			url    string
			chain  []*Block
			length int
			err    error
		}

		resultChan := make(chan chainResult)
		timeout := time.After(30 * time.Second)
		var activeRequests int

		// Start goroutines for each active reflection
		for reflectionURL := range activeReflections {
			// We already filtered out self and inactive reflections when building activeReflections
			activeRequests++
			go func(url string) {
				blockchain, err := cm.GetBlockchainFromReflection(url)
				if err != nil {
					resultChan <- chainResult{url: url, err: err}
					return
				}
				resultChan <- chainResult{
					url:    url,
					chain:  blockchain.Blocks,
					length: len(blockchain.Blocks),
				}
			}(reflectionURL)
		}

		var longestChainReflection string
		var longestChainLength int

		// Process results as they come in
		for activeRequests > 0 {
			select {
			case result := <-resultChan:
				activeRequests--
				if result.err != nil {
					log.Printf("Failed to get blockchain from reflection %s: %v", result.url, result.err)
					continue
				}

				// Check if this reflection has a longer chain
				if result.length > longestChainLength {
					longestChainLength = result.length
					longestChainReflection = result.url
					cm.setLongestChain(result.chain) // Use the setter method for thread safety
				}

			case <-timeout:
				log.Println("Timeout waiting for reflection responses")
				activeRequests = 0 // Stop waiting for remaining responses
			}
		}

		// If we found a reflection with a longer chain, update root blockchain
		if longestChainLength > len(cm.Blockchain.Blocks) {
			log.Printf("Found longer chain (%d blocks) from reflection chain %s", longestChainLength, longestChainReflection)

			// Validate the longest chain
			if cm.validateLongestChain(cm.LongestChain) {
				log.Println("Longest chain is valid. Updating root blockchain.")

				// Update root blockchain with the longest chain
				// Pass the db instance from the blockchain struct
				cm.updateBlockchain(cm.LongestChain, cm.Blockchain.db)

				// Update the transaction pool
				cm.updateTransactionPool(cm.LongestChain)

				// Set the update required flag
				cm.setUpdateRequired(true)
			} else {
				log.Println("Longest chain is invalid. Keeping current blockchain.")
			}
		} else {
			log.Printf("Chain (%d blocks) is already the longest. No update needed.", len(cm.Blockchain.Blocks))
		}

		// Unlock mining after consensus
		cm.unlockMining()
		log.Println("Mining unlocked after consensus")
	}
}

func (cm *ConsensusManager) GetBlockchainFromReflection(reflectionURL string) (*BlockchainStruct, error) {
	log.Printf("Getting blockchain from reflection chain: %s", reflectionURL)

	// Create a new HTTP client with a longer timeout for larger blockchains
	client := &http.Client{Timeout: 30 * time.Second}

	// Send a GET request to the reflection's blockchain endpoint
	resp, err := client.Get(reflectionURL + "/chain")
	if err != nil {
		return nil, fmt.Errorf("failed to get blockchain from reflection chain: %w", err)
	}
	defer resp.Body.Close()

	// Define a struct that matches the SerializableBlockchain struct in handleGetChain
	type SerializableBlockchain struct {
		Blocks          []*Block        `json:"blocks"`
		TransactionPool []*Transaction  `json:"transaction_pool"`
		ChainAddress    string          `json:"chain_address"`
		Reflections     map[string]bool `json:"reflections"`
	}

	// Decode the response into a SerializableBlockchain
	var serializableBC SerializableBlockchain
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&serializableBC); err != nil {
		return nil, fmt.Errorf("failed to decode blockchain from reflection chain: %w", err)
	}

	// Convert SerializableBlockchain to BlockchainStruct
	blockchain := &BlockchainStruct{
		Blocks:          serializableBC.Blocks,
		TransactionPool: serializableBC.TransactionPool,
		ChainAddress:    serializableBC.ChainAddress,
		Reflections:     serializableBC.Reflections,
	}

	log.Printf("Got blockchain with %d blocks from reflection chain", len(blockchain.Blocks))

	// Merge the reflection list from the reflection's blockchain with the root chain's reflection list
	for reflectionAddr, isActive := range blockchain.Reflections {
		if isActive && reflectionAddr != fmt.Sprintf("http://127.0.0.1:%d", utils.GetServerPort()) {
			cm.Blockchain.Reflections[reflectionAddr] = true
		}
	}

	return blockchain, nil
}

// validateLongestChain checks the validity of the provided chain
func (cm *ConsensusManager) validateLongestChain(chain []*Block) bool {
	if len(chain) == 0 {
		log.Println("Chain is empty, validation failed")
		return false
	}

	log.Printf("Validating chain with %d blocks", len(chain))

	// Validate each block in the chain
	for i := 1; i < len(chain); i++ {
		// Ensure each block is linked correctly to the previous block
		if !bytes.Equal(chain[i].PrevHash, chain[i-1].Hash) {
			log.Printf("Block %d has invalid previous hash", i)
			return false
		}

		// Ensure the block has transactions
		if chain[i].Transactions == nil {
			log.Printf("Block %d has nil transactions", i)
			return false
		}

		// Verify the block's hash
		if !chain[i].VerifyBlock() {
			log.Printf("Block %d failed verification", i)
			return false
		}

		// Verify all transactions in the block
		for j, txn := range chain[i].Transactions {
			// Skip mining reward transactions
			if txn.From == utils.BLOCKCHAIN_ADDRESS {
				continue
			}

			// Verify the transaction signature
			valid, err := txn.VerifySignature()
			if err != nil {
				log.Printf("Error verifying transaction %d in block %d: %v", j, i, err)
				return false
			}
			if !valid {
				log.Printf("Transaction %d in block %d has invalid signature", j, i)
				return false
			}
		}

		// Verify the block number is sequential
		if chain[i].BlockNumber != chain[i-1].BlockNumber+1 {
			log.Printf("Block %d has invalid block number: expected %d, got %d",
				i, chain[i-1].BlockNumber+1, chain[i].BlockNumber)
			return false
		}

		// Verify the timestamp is after the previous block
		if chain[i].Timestamp <= chain[i-1].Timestamp {
			log.Printf("Block %d has invalid timestamp: must be after previous block", i)
			return false
		}
	}

	log.Printf("Chain validation successful for %d blocks", len(chain))
	return true
}

// updateTransactionPool removes confirmed transactions from the transaction pool
// Locking strategy:
// 1. First lock ConsensusManager.mu to protect ConsensusManager state
// 2. Then lock BlockchainStruct.mu to protect blockchain data
// This order must be maintained consistently to prevent deadlocks
func (cm *ConsensusManager) updateTransactionPool(validatedChain []*Block) {
	log.Println("Updating transaction pool based on validated chain")

	// Check if the blockchain is nil
	if cm.Blockchain == nil {
		log.Println("Blockchain is nil. Skipping transaction pool update.")
		return
	}

	cm.mu.Lock() // Lock ConsensusManager first
	defer cm.mu.Unlock()

	cm.Blockchain.Lock() // Then lock BlockchainStruct
	defer cm.Blockchain.Unlock()

	// Create a map to quickly check if a transaction is in the validated chain
	confirmedTxns := make(map[string]bool)
	for _, block := range validatedChain {
		for _, txn := range block.Transactions {
			confirmedTxns[txn.TransactionHash] = true
		}
	}

	log.Printf("Found %d confirmed transactions in the validated chain", len(confirmedTxns))

	// Filter out confirmed transactions from the transaction pool
	var newTxnPool []*Transaction
	for _, txn := range cm.Blockchain.TransactionPool {
		if !confirmedTxns[txn.TransactionHash] {
			// Keep transactions that are not in the validated chain
			newTxnPool = append(newTxnPool, txn)
		}
	}

	log.Printf("Removed %d confirmed transactions from the pool",
		len(cm.Blockchain.TransactionPool)-len(newTxnPool))

	// Update the transaction pool
	cm.Blockchain.TransactionPool = newTxnPool

	// Add any transactions from the validated chain that are not in root pool
	for _, block := range validatedChain {
		for _, txn := range block.Transactions {
			// Skip mining reward transactions
			if txn.From == utils.BLOCKCHAIN_ADDRESS {
				continue
			}

			// Check if the transaction is already in root pool
			found := false
			for _, poolTxn := range cm.Blockchain.TransactionPool {
				if poolTxn.TransactionHash == txn.TransactionHash {
					found = true
					break
				}
			}

			// If the transaction is not in root pool, add it
			if !found {
				// Verify the transaction before adding it
				if txn.VerifyTxn() {
					cm.Blockchain.TransactionPool = append(cm.Blockchain.TransactionPool, txn)
				}
			}
		}
	}

	log.Printf("Updated transaction pool now has %d transactions", len(cm.Blockchain.TransactionPool))
}

func (cm *ConsensusManager) lockMining() {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.MiningLocked = true
}

func (cm *ConsensusManager) unlockMining() {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.MiningLocked = false
}

func (cm *ConsensusManager) getMiningLockState() bool {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	return cm.MiningLocked
}

func (cm *ConsensusManager) updateBlockchain(validatedChain []*Block, db *LevelDB) {
	log.Println("Updating blockchain with validated chain")

	// Locking strategy:
	// 1. First lock ConsensusManager.mu to protect ConsensusManager state
	// 2. Then lock BlockchainStruct.mu to protect blockchain data
	// This order must be maintained consistently to prevent deadlocks
	cm.mu.Lock() // Lock ConsensusManager first
	defer cm.mu.Unlock()

	log.Printf("Updating blockchain with %d validated blocks", len(validatedChain))

	// Use BlockchainStruct's SetBlocks which handles deep copying internally
	cm.Blockchain.SetBlocks(validatedChain)

	// Save the updated blockchain using the passed db connection
	err := db.PutIntoDb(cm.Blockchain, cm.Blockchain.ChainAddress)
	if err != nil {
		log.Printf("Failed to save updated blockchain to database: %v", err)
	} else {
		log.Println("Successfully saved updated blockchain to database")
	}

}

// Stop signals the consensus manager to stop gracefully
func (cm *ConsensusManager) Stop() {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.stop = true
	log.Println("Consensus manager stop requested")
}

// GetStatus returns the current consensus status as a string
func (cm *ConsensusManager) GetStatus() string {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.stop {
		return "stopped"
	}
	if cm.MiningLocked {
		return "consensus-active"
	}
	if len(cm.reflectionURLs) == 0 {
		return "standalone"
	}
	return "active"
}

// GetPeerCount returns the number of active dev connections
func (cm *ConsensusManager) GetPeerCount() int {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	count := 0
	for _, active := range cm.reflectionURLs {
		if active {
			count++
		}
	}
	return count
}

// IsSyncing returns true if the node is currently syncing with devs
func (cm *ConsensusManager) IsSyncing() bool {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	return cm.UpdateRequired
}
