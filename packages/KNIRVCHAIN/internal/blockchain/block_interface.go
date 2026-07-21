package blockchain

import (
	"encoding/hex"
	"time"

	"KNIRVCHAIN/internal/utils"
)

// NewBlockInterface is the canonical projection used by interface-based API
// consumers. MerkleRoot and AccumRoot are always populated from committed
// block state (or recomputed for a pre-upgrade block).
func NewBlockInterface(block *Block, previousAccum [32]byte) *BlockInterface {
	if block == nil {
		return nil
	}
	txRoot := block.Header.TxRoot
	if txRoot == ([32]byte{}) {
		txRoot = TxMerkleRoot(block.Transactions)
	}
	accumRoot := block.Header.AccumRoot
	if accumRoot == ([32]byte{}) {
		accumRoot = BlockStateRoot(block, previousAccum)
	}
	txs := make([]TransactionInterface, 0, len(block.Transactions))
	for _, tx := range block.Transactions {
		if tx == nil {
			continue
		}
		txs = append(txs, TransactionInterface{ID: tx.TransactionHash, From: tx.From, To: tx.To, Amount: tx.Value, Fee: tx.Fee, Timestamp: time.Unix(tx.Timestamp, 0).UTC(), Signature: hex.EncodeToString(tx.Signature), Data: append([]byte(nil), tx.Data...)})
	}
	return &BlockInterface{Hash: hex.EncodeToString(block.BlockHash), PreviousHash: hex.EncodeToString(block.PrevHash), Height: block.BlockNumber, Timestamp: time.Unix(block.Timestamp, 0).UTC(), Transactions: txs, Nonce: uint64(block.Nonce), Difficulty: uint64(utils.MINING_DIFFICULTY), MerkleRoot: hex.EncodeToString(txRoot[:]), AccumRoot: hex.EncodeToString(accumRoot[:])}
}
