package crypto

import "KNIRVCHAIN/internal/blockchain"

// NewChainProof builds the committed chain reference attached to a PoAu-D
// proof. Both roots are populated even when reading a pre-upgrade block.
func NewChainProof(block *blockchain.Block, previousAccum [32]byte, witnesses []*Witness) *ChainProof {
	if block == nil {
		return nil
	}
	merkleRoot := block.Header.TxRoot
	if merkleRoot == ([32]byte{}) {
		merkleRoot = blockchain.TxMerkleRoot(block.Transactions)
	}
	accumRoot := block.Header.AccumRoot
	if accumRoot == ([32]byte{}) {
		accumRoot = blockchain.BlockStateRoot(block, previousAccum)
	}
	return &ChainProof{PreviousHash: append([]byte(nil), block.PrevHash...), MerkleRoot: append([]byte(nil), merkleRoot[:]...), AccumRoot: append([]byte(nil), accumRoot[:]...), BlockHeight: block.BlockNumber, Witnesses: append([]*Witness(nil), witnesses...)}
}

func AttachChainProof(proof *PoAuDProof, block *blockchain.Block, previousAccum [32]byte, witnesses []*Witness) {
	if proof != nil {
		proof.ChainProof = NewChainProof(block, previousAccum, witnesses)
	}
}
