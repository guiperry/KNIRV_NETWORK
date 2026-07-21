package checkpoint

import (
	"os"
	"testing"

	"KNIRVCHAIN/internal/blockchain"
	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark/test"
)

func proofBlock(height uint64, author string, txCount int) *blockchain.Block {
	txs := make([]*blockchain.Transaction, txCount)
	for i := range txs {
		txs[i] = &blockchain.Transaction{
			From:            author,
			To:              "receiver",
			Value:           uint64(i + 1),
			Status:          "confirmed",
			Timestamp:       int64(height),
			TransactionHash: "tx",
			Verified:        true,
		}
	}
	block := &blockchain.Block{
		BlockNumber:     height,
		Transactions:    txs,
		ProposerAddress: author,
	}
	block.BlockHash = block.Hash()
	return block
}

func TestPLONKTransitionProofRoundTrip(t *testing.T) {
	if os.Getenv("KNIRV_PLONK_E2E") != "1" {
		t.Skip("set KNIRV_PLONK_E2E=1 to run resource-intensive PLONK setup/prove/verify")
	}
	author := "0xA11CE"
	prover, err := NewUnsafePLONKProver(PLONKConfig{
		ChainID:         "knirvchain-zk-test",
		MaxBlocks:       1,
		MaxTxPerBlock:   1,
		AuthorTreeDepth: 0,
	}, []string{author})
	if err != nil {
		t.Fatalf("setup prover: %v", err)
	}

	proof, err := prover.ProveTransition([32]byte{}, []*blockchain.Block{proofBlock(1, author, 1)})
	if err != nil {
		t.Fatalf("prove transition: %v", err)
	}
	if len(proof.Proof) == 0 || len(proof.PublicInputs) == 0 {
		t.Fatal("proof artifacts must not be empty")
	}
	if err := prover.VerifyLocally(proof); err != nil {
		t.Fatalf("verify transition: %v", err)
	}

	tampered := *proof
	tampered.PostRoot[0] ^= 0xff
	key, err := prover.VerificationKey()
	if err != nil {
		t.Fatalf("verification key: %v", err)
	}
	if err := VerifyPLONKTransition(&tampered, key); err == nil {
		t.Fatal("tampered public root must fail verification")
	}
}

func TestPLONKTransitionCircuitSatisfied(t *testing.T) {
	config := PLONKConfig{ChainID: "knirvchain-circuit-test", MaxBlocks: 1, MaxTxPerBlock: 1, AuthorTreeDepth: 0}
	tree, err := newAuthorMerkleTree([]string{"author"}, config.AuthorTreeDepth)
	if err != nil {
		t.Fatal(err)
	}
	p := &PLONKProver{config: config, authors: tree}
	assignment, _, err := p.buildAssignment([32]byte{}, []*blockchain.Block{proofBlock(1, "author", 1)})
	if err != nil {
		t.Fatal(err)
	}
	if err := test.IsSolved(newTransitionCircuit(config), assignment, ecc.BN254.ScalarField()); err != nil {
		t.Fatalf("transition circuit is not satisfied: %v", err)
	}
	assignment.PostRoot[0] = uint8(assignmentByte(assignment.PostRoot[0]) ^ 0xff)
	if err := test.IsSolved(newTransitionCircuit(config), assignment, ecc.BN254.ScalarField()); err == nil {
		t.Fatal("tampered post-root unexpectedly satisfies circuit")
	}
}

func TestPLONKRejectsUnauthorizedProposerAndCapacityOverflow(t *testing.T) {
	config := PLONKConfig{
		ChainID:         "knirvchain-zk-policy",
		MaxBlocks:       1,
		MaxTxPerBlock:   1,
		AuthorTreeDepth: 1,
	}
	tree, err := newAuthorMerkleTree([]string{"author-1", "author-2"}, config.AuthorTreeDepth)
	if err != nil {
		t.Fatalf("setup author tree: %v", err)
	}
	prover := &PLONKProver{config: config, authors: tree}
	if _, _, err := prover.buildAssignment([32]byte{}, []*blockchain.Block{proofBlock(1, "intruder", 1)}); err == nil {
		t.Fatal("unregistered proposer must be rejected before proving")
	}
	if _, _, err := prover.buildAssignment([32]byte{}, []*blockchain.Block{proofBlock(1, "author-1", 2)}); err == nil {
		t.Fatal("transaction capacity overflow must be rejected")
	}
}

func TestPLONKAuthorWitnessMustMatchBlockHash(t *testing.T) {
	config := PLONKConfig{ChainID: "author-binding", MaxBlocks: 1, MaxTxPerBlock: 1, AuthorTreeDepth: 1}
	tree, err := newAuthorMerkleTree([]string{"author-1", "author-2"}, 1)
	if err != nil {
		t.Fatal(err)
	}
	prover := &PLONKProver{config: config, authors: tree}
	assignment, _, err := prover.buildAssignment([32]byte{}, []*blockchain.Block{proofBlock(1, "author-1", 1)})
	if err != nil {
		t.Fatal(err)
	}
	digest, siblings, directions, err := tree.proof("author-2")
	if err != nil {
		t.Fatal(err)
	}
	setBytes(assignment.Blocks[0].AuthorDigest[:], digest[:])
	for i := range siblings {
		setBytes(assignment.Blocks[0].AuthorSiblings[i][:], siblings[i][:])
		assignment.Blocks[0].AuthorDirections[i] = directions[i]
	}
	if err := test.IsSolved(newTransitionCircuit(config), assignment, ecc.BN254.ScalarField()); err == nil {
		t.Fatal("different registered author unexpectedly authorized an unrelated block hash")
	}
}

func BenchmarkPLONKProof8Blocks8Tx(b *testing.B) {
	authors := []string{"author-1", "author-2", "author-3", "author-4"}
	prover, err := NewUnsafePLONKProver(PLONKConfig{
		ChainID:         "knirvchain-zk-benchmark",
		MaxBlocks:       8,
		MaxTxPerBlock:   8,
		AuthorTreeDepth: 2,
	}, authors)
	if err != nil {
		b.Fatal(err)
	}
	batch := make([]*blockchain.Block, 8)
	for i := range batch {
		batch[i] = proofBlock(uint64(i+1), authors[i%len(authors)], 8)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := prover.ProveTransition([32]byte{}, batch); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPLONKVerify8Blocks8Tx(b *testing.B) {
	authors := []string{"author-1", "author-2", "author-3", "author-4"}
	prover, err := NewUnsafePLONKProver(PLONKConfig{
		ChainID:         "knirvchain-zk-verify-benchmark",
		MaxBlocks:       8,
		MaxTxPerBlock:   8,
		AuthorTreeDepth: 2,
	}, authors)
	if err != nil {
		b.Fatal(err)
	}
	batch := make([]*blockchain.Block, 8)
	for i := range batch {
		batch[i] = proofBlock(uint64(i+1), authors[i%len(authors)], 8)
	}
	proof, err := prover.ProveTransition([32]byte{}, batch)
	if err != nil {
		b.Fatal(err)
	}
	key, err := prover.VerificationKey()
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := VerifyPLONKTransition(proof, key); err != nil {
			b.Fatal(err)
		}
	}
}
