package blockchain

import (
	"bytes"
	"crypto/sha256"
	"fmt"
)

// TxMerkleSibling is one step from a transaction leaf to its block TxRoot.
type TxMerkleSibling struct {
	Hash   [32]byte `json:"hash"`
	IsLeft bool     `json:"is_left"`
}

// AccumulatorStep extends the block accumulator by one block. Together these
// steps carry a transaction proof forward to the AccumRoot at TargetHeight.
type AccumulatorStep struct {
	Height    uint64   `json:"height"`
	TxRoot    [32]byte `json:"tx_root"`
	BlockHash []byte   `json:"block_hash"`
}

// TxAccumProof is the first hop of the light-client proof:
// transaction -> block TxRoot -> chain AccumRoot. The second hop is the Oracle
// MMR proof for the checkpoint whose Root equals AccumRoot.
type TxAccumProof struct {
	SchemaVersion  string            `json:"schema_version"`
	ChainID        string            `json:"chain_id"`
	Transaction    *Transaction      `json:"transaction"`
	TxIndex        uint64            `json:"tx_index"`
	TxSiblings     []TxMerkleSibling `json:"tx_siblings"`
	BlockHeight    uint64            `json:"block_height"`
	PreAccumRoot   [32]byte          `json:"pre_accum_root"`
	BlockHash      []byte            `json:"block_hash"`
	BlockTxRoot    [32]byte          `json:"block_tx_root"`
	BlockAccumRoot [32]byte          `json:"block_accum_root"`
	Accumulator    []AccumulatorStep `json:"accumulator"`
	TargetHeight   uint64            `json:"target_height"`
	AccumRoot      [32]byte          `json:"accum_root"`
}

// GenerateTxAccumProof builds a proof against targetHeight from a stable block
// snapshot. targetHeight must not precede the containing block.
func GenerateTxAccumProof(chainID, txHash string, targetHeight uint64, blocks []*Block) (*TxAccumProof, error) {
	if txHash == "" {
		return nil, fmt.Errorf("transaction hash required")
	}
	blockPos, txPos := -1, -1
	for i, block := range blocks {
		if block == nil {
			continue
		}
		for j, tx := range block.Transactions {
			if tx != nil && tx.TransactionHash == txHash {
				blockPos, txPos = i, j
				break
			}
		}
		if blockPos >= 0 {
			break
		}
	}
	if blockPos < 0 {
		return nil, fmt.Errorf("transaction %s not found", txHash)
	}
	if targetHeight == 0 {
		targetHeight = blocks[len(blocks)-1].BlockNumber
	}
	containing := blocks[blockPos]
	if targetHeight < containing.BlockNumber {
		return nil, fmt.Errorf("target height %d precedes transaction block %d", targetHeight, containing.BlockNumber)
	}
	targetPos := -1
	for i := blockPos; i < len(blocks); i++ {
		if blocks[i] != nil && blocks[i].BlockNumber == targetHeight {
			targetPos = i
			break
		}
	}
	if targetPos < 0 {
		return nil, fmt.Errorf("target height %d not found", targetHeight)
	}

	leaves := TxMerkleLeafHashes(containing.Transactions)
	siblings := merkleSiblings(leaves, txPos)
	var pre [32]byte
	if blockPos > 0 {
		pre = effectiveAccumRoot(blocks[:blockPos])
	}
	blockTxRoot := TxMerkleRoot(containing.Transactions)
	blockAccum := blockStateRootFromParts(pre, blockTxRoot, containing.BlockHash)
	steps := make([]AccumulatorStep, 0, targetPos-blockPos)
	accum := blockAccum
	for i := blockPos + 1; i <= targetPos; i++ {
		block := blocks[i]
		txRoot := TxMerkleRoot(block.Transactions)
		steps = append(steps, AccumulatorStep{Height: block.BlockNumber, TxRoot: txRoot, BlockHash: append([]byte(nil), block.BlockHash...)})
		accum = blockStateRootFromParts(accum, txRoot, block.BlockHash)
	}

	return &TxAccumProof{
		SchemaVersion:  "knirv.tx-accum-proof.v1",
		ChainID:        chainID,
		Transaction:    containing.Transactions[txPos].Clone(),
		TxIndex:        uint64(txPos),
		TxSiblings:     siblings,
		BlockHeight:    containing.BlockNumber,
		PreAccumRoot:   pre,
		BlockHash:      append([]byte(nil), containing.BlockHash...),
		BlockTxRoot:    blockTxRoot,
		BlockAccumRoot: blockAccum,
		Accumulator:    steps,
		TargetHeight:   targetHeight,
		AccumRoot:      accum,
	}, nil
}

// VerifyTxAccumProof verifies the complete transaction-to-AccumRoot hop without
// any chain database access.
func VerifyTxAccumProof(proof *TxAccumProof) error {
	if proof == nil || proof.SchemaVersion != "knirv.tx-accum-proof.v1" || proof.Transaction == nil {
		return fmt.Errorf("invalid transaction accumulator proof")
	}
	leaf := txMerkleLeafHash(canonicalTransactionBytes(proof.Transaction))
	root := leaf
	for _, sibling := range proof.TxSiblings {
		if sibling.IsLeft {
			root = txMerkleParentHash(sibling.Hash, root)
		} else {
			root = txMerkleParentHash(root, sibling.Hash)
		}
	}
	if root != proof.BlockTxRoot {
		return fmt.Errorf("transaction Merkle root mismatch")
	}
	accum := blockStateRootFromParts(proof.PreAccumRoot, proof.BlockTxRoot, proof.BlockHash)
	if accum != proof.BlockAccumRoot {
		return fmt.Errorf("containing block accumulator mismatch")
	}
	lastHeight := proof.BlockHeight
	for _, step := range proof.Accumulator {
		if step.Height != lastHeight+1 {
			return fmt.Errorf("non-contiguous accumulator step %d after %d", step.Height, lastHeight)
		}
		accum = blockStateRootFromParts(accum, step.TxRoot, step.BlockHash)
		lastHeight = step.Height
	}
	if lastHeight != proof.TargetHeight || accum != proof.AccumRoot {
		return fmt.Errorf("target accumulator mismatch")
	}
	return nil
}

func merkleSiblings(level [][32]byte, index int) []TxMerkleSibling {
	proof := make([]TxMerkleSibling, 0)
	pos := index
	for len(level) > 1 {
		siblingPos := pos ^ 1
		if siblingPos >= len(level) {
			siblingPos = pos
		}
		proof = append(proof, TxMerkleSibling{Hash: level[siblingPos], IsLeft: siblingPos < pos})
		next := make([][32]byte, 0, (len(level)+1)/2)
		for i := 0; i < len(level); i += 2 {
			right := level[i]
			if i+1 < len(level) {
				right = level[i+1]
			}
			next = append(next, txMerkleParentHash(level[i], right))
		}
		level = next
		pos /= 2
	}
	return proof
}

func effectiveAccumRoot(blocks []*Block) [32]byte {
	if len(blocks) == 0 {
		return [32]byte{}
	}
	stored := blocks[len(blocks)-1].Header.AccumRoot
	if stored != ([32]byte{}) {
		return stored
	}
	return RecomputeAccumRoot(blocks)
}

func blockStateRootFromParts(previous, txRoot [32]byte, blockHash []byte) [32]byte {
	preimage := make([]byte, 1, 1+64+len(blockHash))
	preimage[0] = merkleParentDomain
	preimage = append(preimage, previous[:]...)
	preimage = append(preimage, txRoot[:]...)
	preimage = append(preimage, blockHash...)
	return sha256.Sum256(preimage)
}

// EqualTransactionProofTarget is a small client helper for binding this hop to
// a checkpoint root returned by the Oracle index.
func EqualTransactionProofTarget(proof *TxAccumProof, checkpointRoot [32]byte) bool {
	return proof != nil && bytes.Equal(proof.AccumRoot[:], checkpointRoot[:])
}
