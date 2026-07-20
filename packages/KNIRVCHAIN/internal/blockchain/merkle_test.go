package blockchain

import (
	"encoding/hex"
	"testing"
)

func TestTxMerkleRootGolden(t *testing.T) {
	txs := merkleFixtureTransactions()
	tests := []struct {
		name string
		txs  []*Transaction
		want string
	}{
		{name: "empty", txs: nil, want: "3abba63a98970f7ec72112d3b7658c1fc64b0d025173b655e15dd2c8a147b86d"},
		{name: "one", txs: txs[:1], want: "8c53709f11f3f915467ff739441337041fb52e7c65c336c111e4c71bbfe25019"},
		{name: "odd", txs: txs, want: "ab299e9ade06abd0b575ea8387ccee632e082958e1734aa74f748eb9c2184fee"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := TxMerkleRoot(tt.txs)
			got := hex.EncodeToString(root[:])
			if got != tt.want {
				t.Fatalf("root = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestTxMerkleRootCommitsToOrderAndContent(t *testing.T) {
	txs := merkleFixtureTransactions()
	original := TxMerkleRoot(txs)
	reordered := []*Transaction{txs[1], txs[0], txs[2]}
	if TxMerkleRoot(reordered) == original {
		t.Fatal("transaction root did not commit to order")
	}
	mutated := make([]*Transaction, len(txs))
	for i, tx := range txs {
		mutated[i] = tx.Clone()
	}
	mutated[0].Fee++
	if TxMerkleRoot(mutated) == original {
		t.Fatal("transaction root did not commit to transaction contents")
	}
}

func TestBlockAccumulatorMatchesIndependentReplay(t *testing.T) {
	txs := merkleFixtureTransactions()
	blocks := []*Block{
		{BlockNumber: 1, BlockHash: bytes32(0x11), Transactions: txs[:1]},
		{BlockNumber: 2, BlockHash: bytes32(0x22), Transactions: txs[1:]},
		{BlockNumber: 3, BlockHash: bytes32(0x33), Transactions: nil},
	}
	var previous [32]byte
	for _, block := range blocks {
		PopulateBlockMerkleRoots(block, previous)
		if block.Header.TxRoot != TxMerkleRoot(block.Transactions) {
			t.Fatalf("block %d transaction root mismatch", block.BlockNumber)
		}
		previous = block.Header.AccumRoot
	}
	if got := RecomputeAccumRoot(blocks); got != previous {
		t.Fatalf("recomputed accumulator = %x, stored = %x", got, previous)
	}
	const expected = "ba20a1d1672f21a27272672c15ec4922008987c216bbef41c99a301e75b0c7c3"
	if got := hex.EncodeToString(previous[:]); got != expected {
		t.Fatalf("accumulator = %s, want %s", got, expected)
	}

	// The independent replay must not trust the persisted header fields.
	blocks[1].Header.AccumRoot[0] ^= 0xff
	if got := RecomputeAccumRoot(blocks); got != previous {
		t.Fatal("recomputation unexpectedly depended on stored header roots")
	}
}

func merkleFixtureTransactions() []*Transaction {
	return []*Transaction{
		{From: "alice", To: "bob", Value: 7, Data: []byte("alpha"), Status: "SUCCESS", Timestamp: 1700000001, TransactionHash: "0x01", PublicKey: "pub-a", Signature: []byte{1, 2, 3}, Fee: 2, Type: "transfer", Verified: true},
		{From: "bob", To: "carol", Value: 11, Data: []byte("beta"), Status: "SUCCESS", Timestamp: 1700000002, TransactionHash: "0x02", PublicKey: "pub-b", Signature: []byte{4, 5, 6}, Fee: 3, Type: "data", Verified: true},
		{From: "carol", To: "dave", Value: 13, Data: []byte("gamma"), Status: "SUCCESS", Timestamp: 1700000003, TransactionHash: "0x03", PublicKey: "pub-c", Signature: []byte{7, 8, 9}, Fee: 5, Type: "transfer", Verified: true},
	}
}

func bytes32(value byte) []byte {
	result := make([]byte, 32)
	for i := range result {
		result[i] = value
	}
	return result
}
