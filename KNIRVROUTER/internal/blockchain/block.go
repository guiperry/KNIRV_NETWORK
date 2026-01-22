package blockchain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log"
	"strings"
	"time"

	constants "KNIRVROUTER/internal/constants"
	"KNIRVROUTER/internal/types"
)

// BlockHeader represents the minimal data needed to verify a block's position in the chain
type BlockHeader struct {
	BlockNumber uint64 `json:"block_number"`
	Hash        string `json:"hash"`
	PrevHash    string `json:"prev_hash"`
	Timestamp   int64  `json:"timestamp"`
}

type Block struct {
	Number       uint64               `json:"block_number"`
	PreviousHash string               `json:"prevHash"`
	Time         int64                `json:"timestamp"`
	Nonce        int                  `json:"nonce"`
	Txs          []*types.Transaction `json:"transactions"`
}

// Interface methods
func (b Block) BlockNumber() uint64 {
	return b.Number
}

func (b Block) PrevHash() string {
	return b.PreviousHash
}

func (b Block) Timestamp() int64 {
	return b.Time
}

func (b Block) Transactions() []*types.Transaction {
	return b.Txs
}

func (b Block) HashString() string {
	return b.Hash()
}

func (b Block) PrevHashString() string {
	return b.PrevHash()
}

func NewBlock(prevHash string, nonce int, blockNumber uint64) *Block {
	return &Block{
		PreviousHash: prevHash,
		Time:         time.Now().UnixNano(),
		Nonce:        nonce,
		Txs:          []*types.Transaction{},
		Number:       blockNumber,
	}
}

// ToJSON marshals the block to a JSON string
func (b Block) ToJSON() (string, error) {
	nb, err := json.Marshal(b)
	if err != nil {
		return "", err
	}
	return string(nb), nil
}

// ToJson is kept for backward compatibility
func (b Block) ToJson() string {
	json, err := b.ToJSON()
	if err != nil {
		return err.Error()
	}
	return json
}

// BlockFromJSON unmarshals a JSON string to a Block
func BlockFromJSON(jsonStr string) (*Block, error) {
	var block Block
	err := json.Unmarshal([]byte(jsonStr), &block)
	if err != nil {
		return nil, err
	}
	return &block, nil
}

//	func (b Block) MarshalJSON() ([]byte, error) {
//		log.Printf("Block Marshaling BlockNumber: %v, PrevHash: %v, Timestamp: %v, Nonce: %v, Transactions: %v", b.BlockNumber, b.PrevHash, b.Timestamp, b.Nonce, b.Transactions)
//
// type Alias Block
//
//	aux := struct {
//		Alias
//			Transactions []*types.Transaction `json:"transactions"`
//		}{
//		Alias:        (Alias)(b),
//		Transactions: b.Transactions,
//		}
//		return json.Marshal(aux)
//	}
//
// Hash calculates the SHA256 hash of the block's JSON representation
func (b Block) Hash() string {
	bs, _ := json.Marshal(b)
	sum := sha256.Sum256(bs)
	hexRep := hex.EncodeToString(sum[:32])
	return constants.HEX_PREFIX + hexRep
}

// VerifyHash checks if the block's hash meets the mining difficulty
func (b Block) VerifyHash() bool {
	difficulty := constants.MINING_DIFFICULTY
	hash := b.Hash()
	prefix := strings.Repeat("0", difficulty)
	return strings.HasPrefix(hash[len(constants.HEX_PREFIX):], prefix)
}

// VerifyBlock performs full block validation including:
// - Hash verification
// - Transaction validation
// - Block number sequencing
func (b Block) VerifyBlock() bool {
	if !b.VerifyHash() {
		return false
	}

	// TODO: Add transaction validation logic
	// TODO: Add block number sequencing checks

	return true
}
func (b *Block) Mine(difficulty int) error {
	startTime := time.Now()
	hashesAttempted := 0

	log.Printf("Starting mining for block %d with difficulty %d", b.Number, difficulty)

	for {
		b.Time = time.Now().UnixNano()
		guessHash := b.Hash()
		desiredHash := strings.Repeat("0", difficulty)
		ourSolutionHash := guessHash[2 : 2+difficulty]

		hashesAttempted++

		if hashesAttempted%1000 == 0 {
			elapsed := time.Since(startTime)
			hashRate := float64(hashesAttempted) / elapsed.Seconds()
			log.Printf("Mining block %d: %d hashes attempted, %.2f hashes/sec, current hash: %s",
				b.Number, hashesAttempted, hashRate, guessHash)
		}

		if ourSolutionHash == desiredHash {
			elapsed := time.Since(startTime)
			hashRate := float64(hashesAttempted) / elapsed.Seconds()
			log.Printf("BLOCK MINED SUCCESSFULLY! Block %d, Nonce: %d, Hash: %s, Time: %.2fs, Hashes: %d, Rate: %.2f hashes/sec",
				b.Number, b.Nonce, guessHash, elapsed.Seconds(), hashesAttempted, hashRate)
			return nil
		}

		b.Nonce++
	}
}

func (b *Block) AddTransactionToTheBlock(txn *types.Transaction) error {
	if txn.Status == constants.TXN_VERIFICATION_SUCCESS {
		txn.Status = constants.SUCCESS
	} else {
		txn.Status = constants.FAILED
	}

	b.Txs = append(b.Txs, txn)
	return nil
}
