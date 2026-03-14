package types

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"
)

type Block struct {
	Header       BlockHeader    `json:"header"`
	Transactions []*Transaction `json:"transactions"`
	Hash         string         `json:"hash"`
}

type BlockHeader struct {
	Height       uint64    `json:"height"`
	PreviousHash string    `json:"previous_hash"`
	PrevHash     string    `json:"prev_hash"` // Alias for backward compatibility
	Timestamp    time.Time `json:"timestamp"`
	MerkleRoot   string    `json:"merkle_root"`
	ValidatorSet []string  `json:"validator_set"`
	Proposer     string    `json:"proposer"`
	StateRoot    string    `json:"state_root"`
}

func (b *Block) CalculateHash() string {
	headerBytes, _ := json.Marshal(b.Header)
	txBytes, _ := json.Marshal(b.Transactions)
	combined := append(headerBytes, txBytes...)
	hash := sha256.Sum256(combined)
	return hex.EncodeToString(hash[:])
}

func (b *Block) Serialize() ([]byte, error) {
	return json.Marshal(b)
}

func NewBlock(height uint64, prevHash string, txs []*Transaction, proposer string) *Block {
	header := BlockHeader{
		Height:       height,
		PreviousHash: prevHash,
		PrevHash:     prevHash, // Set both for compatibility
		Timestamp:    time.Now(),
		MerkleRoot:   calculateMerkleRoot(txs),
		Proposer:     proposer,
	}

	block := &Block{
		Header:       header,
		Transactions: txs,
	}

	block.Hash = block.CalculateHash()
	return block
}

func calculateMerkleRoot(txs []*Transaction) string {
	if len(txs) == 0 {
		return ""
	}

	var hashes []string
	for _, tx := range txs {
		hashes = append(hashes, tx.Hash())
	}

	// Simple merkle root calculation (in production, use proper merkle tree)
	combined := ""
	for _, hash := range hashes {
		combined += hash
	}

	rootHash := sha256.Sum256([]byte(combined))
	return hex.EncodeToString(rootHash[:])
}
