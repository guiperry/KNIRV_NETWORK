package blockchain

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	//"google.golang.org/protobuf/encoding/prototext" // For pretty printing proto
	//"google.golang.org/protobuf/types/known/timestamppb"
	"KNIRVORACLE/pkg/log"
	"KNIRVORACLE/pkg/utils"
)

type BlockHeader struct {
	Height    uint64 `json:"height"`
	Timestamp int64  `json:"timestamp"`
	Version   int    `json:"version"`
}

type Block struct {
	BlockNumber uint64 `json:"block_number"`
	PrevHash    []byte `json:"prevHash"`
	Timestamp   int64  `json:"timestamp"`
	Nonce       int    `json:"nonce"`
	// BlockProto   *BlockProto    `json:"-"` // Removed as ToProto() in proto_converters.go is used for hashing
	Transactions    []*Transaction `json:"transactions"`
	BlockHash       []byte         `json:"hash"`
	ProposerAddress string         `json:"proposer_address,omitempty"`
	Header          BlockHeader    `json:"header"`
	// Track which transactions are invalid (not persisted to JSON)
	InvalidTxHashes map[string]string `json:"-"`
}

// Placeholder proto type to avoid import cycle
type BlockProto struct {
	BlockNumber     uint64
	PrevHash        []byte
	Nonce           int32
	Timestamp       *Timestamp
	Transactions    []*TransactionProto
	ProposerAddress string
}

type Timestamp struct {
	Seconds int64
	Nanos   int32
}

func (t *Timestamp) AsTime() time.Time {
	if t == nil {
		return time.Time{}
	}
	return time.Unix(t.Seconds, int64(t.Nanos))
}

type TransactionProto struct {
	// Placeholder for TransactionProto
}

// ToProto converts the block to proto format
func (b *Block) ToProto() (*BlockProto, error) {
	// Convert timestamp to proto format
	timestamp := &Timestamp{
		Seconds: b.Timestamp,
		Nanos:   0,
	}

	// Convert transactions to proto format (placeholder for now)
	var protoTransactions []*TransactionProto
	for range b.Transactions {
		protoTransactions = append(protoTransactions, &TransactionProto{})
	}

	return &BlockProto{
		BlockNumber:     b.BlockNumber,
		PrevHash:        b.PrevHash,
		Nonce:           int32(b.Nonce),
		Timestamp:       timestamp,
		Transactions:    protoTransactions,
		ProposerAddress: b.ProposerAddress,
	}, nil
}

// GetCanonicalBytesForHashing returns canonical bytes for hashing (placeholder implementation)
func GetCanonicalBytesForHashing(proto *BlockProto) ([]byte, error) {
	// Placeholder implementation - use JSON marshaling for now
	return []byte(fmt.Sprintf("block-hash-placeholder")), nil
}

func (b *Block) IsValid() bool {
	// Check if the block has a valid hash
	calculatedHash := b.Hash()
	return bytes.Equal(calculatedHash, b.BlockHash)
}

func (b *Block) VerifyBlock() bool {
	// Check if the block has a valid hash stored internally
	calculatedHash := b.Hash()
	if !bytes.Equal(calculatedHash, b.BlockHash) {
		// Log this specific failure mode for clarity
		log.LogError(fmt.Sprintf("VerifyBlock failed for block %d: Calculated hash %s does not match stored BlockHash %s",
			b.BlockNumber, hex.EncodeToString(calculatedHash), hex.EncodeToString(b.BlockHash)), nil)
		return false
	}

	// Check if the block meets the mining difficulty requirement
	desiredHashPrefix := strings.Repeat("0", utils.MINING_DIFFICULTY) // e.g., "0" if difficulty is 1

	// --- CORRECTED CHECK ---
	// Convert the whole hash to hex first
	hashHex := hex.EncodeToString(calculatedHash) // Use calculatedHash here
	// Then check if the *prefix* matches
	if !strings.HasPrefix(hashHex, desiredHashPrefix) {
		// --- END CORRECTED CHECK ---
		// Log this specific failure mode
		log.LogError(fmt.Sprintf("VerifyBlock failed for block %d: Hash %s does not meet difficulty requirement (prefix %s)",
			b.BlockNumber, hashHex, desiredHashPrefix), nil)
		return false
	}

	// Verify all transactions in the block
	for i, txn := range b.Transactions {
		// Skip mining reward transactions (which don't have signatures)
		if txn.From == utils.BLOCKCHAIN_ADDRESS {
			continue
		}
		// Verify other transactions
		if !txn.VerifyTxn() { // VerifyTxn now includes VerifySignature
			// Log which transaction failed
			log.LogError(fmt.Sprintf("VerifyBlock failed for block %d: Transaction %d (%s) failed verification",
				b.BlockNumber, i, txn.TransactionHash), nil)
			return false
		}
	}

	// If all checks pass
	return true
}

func NewBlock(prevHash []byte, nonce int, blockNumber uint64) *Block {
	block := new(Block)
	block.PrevHash = prevHash
	block.Nonce = nonce
	block.BlockNumber = blockNumber
	block.Timestamp = time.Now().Unix()
	block.InvalidTxHashes = make(map[string]string) // Initialize the map to track invalid transactions

	return block
}
func (b Block) ToJson() string {
	nb, err := json.Marshal(b)

	if err != nil {
		return err.Error()
	} else {
		return string(nb)
	}
}

func (b *Block) Hash() []byte {
	blockProto, err := b.ToProto()
	if err != nil {
		log.LogError(fmt.Sprintf("Failed to convert block %d to proto for hashing: %v", b.BlockNumber, err), nil)
		// Return a dummy hash or panic, as this indicates a programming error
		dummyHash := sha256.Sum256([]byte(fmt.Sprintf("error-hash-block-%d", b.BlockNumber)))
		return dummyHash[:]
	}

	// ---- TEMPORARY DEBUGGING ----
	log.LogInfo(fmt.Sprintf("Block.Hash: Block #%d being converted to proto. Timestamp: %d, Nonce: %d, NumTx: %d", b.BlockNumber, b.Timestamp, b.Nonce, len(b.Transactions)))
	if blockProto == nil {
		log.LogError("Block.Hash: blockProto is NIL before marshalling!", nil)
	}
	if blockProto == nil {
		log.LogError("Block.Hash: blockProto is nil, returning error hash", nil)
		dummyHash := sha256.Sum256([]byte(fmt.Sprintf("nil-proto-block-%d", b.BlockNumber)))
		return dummyHash[:]
	}

	// Debug log the blockProto fields
	log.LogInfo(fmt.Sprintf("ProtoBlock fields for block #%d:", b.BlockNumber))
	log.LogInfo(fmt.Sprintf("- BlockNumber: %d", blockProto.BlockNumber))
	log.LogInfo(fmt.Sprintf("- PrevHash: %x", blockProto.PrevHash))
	log.LogInfo(fmt.Sprintf("- Nonce: %d", blockProto.Nonce))
	if blockProto.Timestamp != nil {
		log.LogInfo(fmt.Sprintf("- Timestamp: %v", blockProto.Timestamp.AsTime()))
	} else {
		log.LogError("ProtoBlock.Timestamp is nil!", nil)
	}
	log.LogInfo(fmt.Sprintf("- Transactions count: %d", len(blockProto.Transactions)))
	log.LogInfo(fmt.Sprintf("- ProposerAddress: %s", blockProto.ProposerAddress))

	// Only attempt prototext.Format if all required fields are present
	if blockProto.Timestamp == nil {
		log.LogError("Cannot format ProtoBlock - Timestamp is nil", nil)
		dummyHash := sha256.Sum256([]byte(fmt.Sprintf("invalid-proto-block-%d", b.BlockNumber)))
		return dummyHash[:]
	}
	// ---- END TEMPORARY DEBUGGING ----

	// Use the GetCanonicalBytesForHashing function (defined in proto_converters.go)
	canonicalBytes, err := GetCanonicalBytesForHashing(blockProto)
	if err != nil {
		log.LogError(fmt.Sprintf("Failed to marshal block %d to proto for hashing: %v", b.BlockNumber, err), nil)
		dummyHash := sha256.Sum256([]byte(fmt.Sprintf("error-marshal-block-%d", b.BlockNumber)))
		return dummyHash[:]
	}

	hashed := sha256.Sum256(canonicalBytes)
	return hashed[:]
}

// HashString returns the block hash as a hex string
func (b *Block) HashString() string {
	return hex.EncodeToString(b.BlockHash)
}

// PrevHashString returns the previous block hash as a hex string
func (b *Block) PrevHashString() string {
	return hex.EncodeToString(b.PrevHash)
}
func (b *Block) AddTransactionToTheBlock(txn *Transaction) { // Implementation type is passed as object pointer.
	// Assume the caller (miner) has already set the correct final status (SUCCESS).
	// Simply append the transaction directly to the block's list.
	// No need to check or change the status here.

	b.Transactions = append(b.Transactions, txn) // setting now properties after validation and types is matched correctly to object's property being updated/modified correctly using methods of structs.
}
func (b *Block) MineBlock() {
	nonce := 0                                                  // define variable
	desiredHash := strings.Repeat("0", utils.MINING_DIFFICULTY) // set from parameter from test validation.

	for {
		guessHashBytes := b.Hash() // call proper method as it was designed on system validation workflows
		hashHex := hex.EncodeToString(guessHashBytes)

		if strings.HasPrefix(hashHex, desiredHash) { // make state implementation changes only on success
			// IMPORTANT FIX: Update the block's hash field with the valid hash
			b.BlockHash = guessHashBytes
			return // finish if hash was created based on project requirements
		}
		nonce++ // set type variable changes for nonce as they iterate during validations

		if nonce%1000000 == 0 { // Log every 10000 nonces to avoid flooding
			log.LogInfo(fmt.Sprintf("... mining attempt, nonce: %d", nonce))
		}

		b.Nonce = nonce // update the nonce in the block
	}
}

// DeepCopy creates a deep copy of the Block instance.  This is crucial to avoid
// modifying the original block when updating the blockchain during consensus.
func (b *Block) DeepCopy() *Block {
	copiedTransactions := make([]*Transaction, len(b.Transactions))
	for i, tx := range b.Transactions {
		copiedTransactions[i] = tx.Clone() // Call Clone for transactions
	}

	copiedPrevHash := make([]byte, len(b.PrevHash))
	copy(copiedPrevHash, b.PrevHash)

	copiedHash := make([]byte, len(b.BlockHash))
	copy(copiedHash, b.BlockHash)

	// Copy the InvalidTxHashes map
	copiedInvalidTxHashes := make(map[string]string)
	for hash, reason := range b.InvalidTxHashes {
		copiedInvalidTxHashes[hash] = reason
	}

	return &Block{
		BlockNumber:  b.BlockNumber,
		PrevHash:     copiedPrevHash,
		Timestamp:    b.Timestamp,
		Nonce:        b.Nonce,
		Transactions: copiedTransactions,
		// Data:         copiedData, // Removed
		BlockHash:       copiedHash,
		ProposerAddress: b.ProposerAddress,
		Header:          b.Header,
		InvalidTxHashes: copiedInvalidTxHashes,
	}
}
