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
	"KNIRVCHAIN/internal/log"
	"KNIRVCHAIN/internal/utils"
)

type BlockHeader struct {
	Height    uint64            `json:"height"`
	Timestamp int64             `json:"timestamp"`
	Version   int               `json:"version"`
	TxRoot    [sha256.Size]byte `json:"tx_root"`
	AccumRoot [sha256.Size]byte `json:"accum_root"`
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

// GetCanonicalBytesForHashing returns a deterministic, length-delimited
// encoding for legacy callers. Block.Hash uses the stronger content commitment
// in ContentBlockHash because BlockProto predates real transaction fields.
func GetCanonicalBytesForHashing(proto *BlockProto) ([]byte, error) {
	if proto == nil || proto.Timestamp == nil {
		return nil, fmt.Errorf("block proto and timestamp are required")
	}
	buf := appendBytes(nil, []byte("knirv-block-proto-v1"))
	buf = appendUint64(buf, proto.BlockNumber)
	buf = appendBytes(buf, proto.PrevHash)
	buf = appendUint64(buf, uint64(int64(proto.Nonce)))
	buf = appendUint64(buf, uint64(proto.Timestamp.Seconds))
	buf = appendUint64(buf, uint64(int64(proto.Timestamp.Nanos)))
	buf = appendUint64(buf, uint64(len(proto.Transactions)))
	buf = appendString(buf, proto.ProposerAddress)
	return buf, nil
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

	// Verify all transactions in the block. Exactly one protocol mining reward
	// is required; all other protocol records must satisfy their explicit
	// schema instead of receiving a blanket signature bypass.
	rewardCount := 0
	for i, txn := range b.Transactions {
		if txn.From == utils.BLOCKCHAIN_ADDRESS {
			if !txn.isValidProtocolTransaction() {
				return false
			}
			if txn.Type == "protocol_mining_reward" {
				rewardCount++
				if b.ProposerAddress != "" && txn.To != b.ProposerAddress {
					return false
				}
			}
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
	if b.BlockNumber > 0 && rewardCount != 1 {
		return false
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
	hashed := ContentBlockHash(b)
	return append([]byte(nil), hashed[:]...)
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
