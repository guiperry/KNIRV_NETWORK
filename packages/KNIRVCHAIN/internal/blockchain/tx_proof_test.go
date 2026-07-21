package blockchain

import (
	"crypto/sha256"
	"testing"
)

func txProofBlock(height uint64, txs ...*Transaction) *Block {
	hash := sha256.Sum256([]byte{byte(height), byte(len(txs))})
	return &Block{BlockNumber: height, BlockHash: hash[:], Transactions: txs}
}

func TestTxAccumProofRoundTrip(t *testing.T) {
	target := &Transaction{From: "a", To: "b", Value: 7, TransactionHash: "target", Status: "confirmed"}
	blocks := []*Block{
		txProofBlock(1, &Transaction{TransactionHash: "first"}, target, &Transaction{TransactionHash: "odd"}),
		txProofBlock(2, &Transaction{TransactionHash: "next"}),
		txProofBlock(3),
	}
	var previous [32]byte
	for _, block := range blocks {
		PopulateBlockMerkleRoots(block, previous)
		previous = block.Header.AccumRoot
	}

	proof, err := GenerateTxAccumProof("chain-1", "target", 3, blocks)
	if err != nil {
		t.Fatalf("generate proof: %v", err)
	}
	if err := VerifyTxAccumProof(proof); err != nil {
		t.Fatalf("verify proof: %v", err)
	}
	if proof.AccumRoot != blocks[2].Header.AccumRoot {
		t.Fatal("proof does not target requested accumulator")
	}

	proof.Accumulator[0].TxRoot[0] ^= 0xff
	if err := VerifyTxAccumProof(proof); err == nil {
		t.Fatal("tampered accumulator step must fail")
	}
}

func TestTxAccumProofRejectsBadTargets(t *testing.T) {
	tx := &Transaction{TransactionHash: "tx"}
	blocks := []*Block{txProofBlock(4, tx)}
	if _, err := GenerateTxAccumProof("chain", "missing", 4, blocks); err == nil {
		t.Fatal("missing transaction should fail")
	}
	if _, err := GenerateTxAccumProof("chain", "tx", 3, blocks); err == nil {
		t.Fatal("target before containing block should fail")
	}
}
