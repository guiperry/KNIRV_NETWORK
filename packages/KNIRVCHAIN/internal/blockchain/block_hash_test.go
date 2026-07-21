package blockchain

import "testing"

func TestBlockHashCommitsContentAndProposer(t *testing.T) {
	base := &Block{BlockNumber: 7, PrevHash: []byte("prev"), Timestamp: 123, Nonce: 9, ProposerAddress: "0xauthor", Transactions: []*Transaction{{From: "a", To: "b", Value: 1, TransactionHash: "tx-1"}}}
	want := base.Hash()
	copyBlock := base.DeepCopy()
	if got := copyBlock.Hash(); string(got) != string(want) {
		t.Fatal("identical block hash is not deterministic")
	}
	copyBlock.ProposerAddress = "0xintruder"
	if got := copyBlock.Hash(); string(got) == string(want) {
		t.Fatal("proposer mutation did not change block hash")
	}
	copyBlock = base.DeepCopy()
	copyBlock.Transactions[0].Value++
	if got := copyBlock.Hash(); string(got) == string(want) {
		t.Fatal("transaction mutation did not change block hash")
	}
	copyBlock = base.DeepCopy()
	copyBlock.Nonce++
	if got := copyBlock.Hash(); string(got) == string(want) {
		t.Fatal("nonce mutation did not change block hash")
	}
}
