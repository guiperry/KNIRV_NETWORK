// utils/utils.go
package utils

import (
	block "KNIRVCHAIN_GO_Verifyer/blockchain"
	blockchain "KNIRVCHAIN_GO_Verifyer/blockchain"
)

func CompareBlocks(blocks1 []*block.Block, blocks2 []*block.Block) bool {
	if len(blocks1) != len(blocks2) {
		return false
	}
	for i := range blocks1 {
		if !CompareBlock(blocks1[i], blocks2[i]) {
			return false
		}
	}
	return true
}
func CompareBlock(block1 *block.Block, block2 *block.Block) bool {
	if block1.PreviousHash != block2.PreviousHash {
		return false
	}
	if block1.Hash() != block2.Hash() {
		return false
	}
	if block1.Nonce != block2.Nonce {
		return false
	}
	if block1.Time != block2.Time {
		return false
	}
	if len(block1.Txs) != len(block2.Txs) {
		return false
	}
	for i := range block1.Txs {
		if block1.Txs[i].From != block2.Txs[i].From || block1.Txs[i].To != block2.Txs[i].To || block1.Txs[i].Value != block2.Txs[i].Value {
			return false
		}
	}
	return true
}
func CompareBlockchain(bc1 *blockchain.BlockchainStruct, bc2 *blockchain.BlockchainStruct) bool {
	if len(bc1.Blocks) != len(bc2.Blocks) {
		return false
	}

	for i := range bc1.Blocks {
		if !CompareBlock(bc1.Blocks[i], bc2.Blocks[i]) {
			return false
		}
	}
	return true
}
