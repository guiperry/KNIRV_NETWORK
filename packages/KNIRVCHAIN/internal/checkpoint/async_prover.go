package checkpoint

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sync"

	"KNIRVCHAIN/internal/blockchain"
)

type HashchainPublicInputs struct {
	PreRoot   [32]byte `json:"pre_root"`
	PostRoot  [32]byte `json:"post_root"`
	BatchHash [32]byte `json:"tx_batch_hash"`
}

type HashchainWitness struct {
	Blocks []*blockchain.Block `json:"blocks"`
}

// HashchainProver is the explicit degraded fallback. It reveals the execution
// trace commitment and relies on verifier-chain re-execution; it is not ZK.
type HashchainProver struct {
	ChainID string
	PreRoot [32]byte
}

func (p *HashchainProver) Prove(batch []*blockchain.Block) (*TransitionProof, error) {
	if p == nil || p.ChainID == "" || len(batch) == 0 {
		return nil, fmt.Errorf("hashchain prover requires chain id and non-empty batch")
	}
	accum := p.PreRoot
	trace := sha256.New()
	trace.Write([]byte("knirv-hashchain-proof-v0"))
	trace.Write(accum[:])
	for i, block := range batch {
		if block == nil || (i > 0 && block.BlockNumber != batch[i-1].BlockNumber+1) {
			return nil, fmt.Errorf("invalid/non-contiguous hashchain batch")
		}
		txRoot := blockchain.TxMerkleRoot(block.Transactions)
		trace.Write(txRoot[:])
		trace.Write(block.BlockHash)
		accum = blockchain.BlockStateRoot(block, accum)
		trace.Write(accum[:])
	}
	batchHash := sha256.Sum256(trace.Sum(nil))
	public, err := json.Marshal(HashchainPublicInputs{PreRoot: p.PreRoot, PostRoot: accum, BatchHash: batchHash})
	if err != nil {
		return nil, err
	}
	witness, err := json.Marshal(HashchainWitness{Blocks: batch})
	if err != nil {
		return nil, fmt.Errorf("marshal hashchain witness: %w", err)
	}
	return &TransitionProof{
		SchemaVersion: TransitionProofSchema, ChainID: p.ChainID,
		StartHeight: batch[0].BlockNumber, EndHeight: batch[len(batch)-1].BlockNumber,
		PreRoot: p.PreRoot, PostRoot: accum, ProofSystem: "hashchain-v0",
		Proof: witness, PublicInputs: public,
	}, nil
}

// FallbackProver attempts PLONK first and degrades to hashchain-v0 without
// blocking checkpoint production when the asynchronous prover is unavailable.
type FallbackProver struct{ Primary, Fallback Prover }

func (p *FallbackProver) Prove(batch []*blockchain.Block) (*TransitionProof, error) {
	if p.Primary != nil {
		if proof, err := p.Primary.Prove(batch); err == nil {
			return proof, nil
		}
	}
	if p.Fallback == nil {
		return nil, fmt.Errorf("primary prover unavailable and no fallback configured")
	}
	return p.Fallback.Prove(batch)
}

type ProofResult struct {
	Proof *TransitionProof
	Err   error
}
type proofJob struct {
	ctx    context.Context
	batch  []*blockchain.Block
	result chan ProofResult
}

// AsyncProver decouples proving from block admission. Queue saturation is
// reported to the caller, which can immediately choose hashchain-v0.
type AsyncProver struct {
	prover Prover
	jobs   chan proofJob
	stop   chan struct{}
	once   sync.Once
}

func NewAsyncProver(prover Prover, queueDepth int) (*AsyncProver, error) {
	if prover == nil {
		return nil, fmt.Errorf("prover required")
	}
	if queueDepth <= 0 {
		queueDepth = 1
	}
	a := &AsyncProver{prover: prover, jobs: make(chan proofJob, queueDepth), stop: make(chan struct{})}
	go a.worker()
	return a, nil
}

func (a *AsyncProver) Submit(ctx context.Context, batch []*blockchain.Block) (<-chan ProofResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	cloned := make([]*blockchain.Block, len(batch))
	for i, block := range batch {
		if block != nil {
			cloned[i] = block.DeepCopy()
		}
	}
	result := make(chan ProofResult, 1)
	job := proofJob{ctx: ctx, batch: cloned, result: result}
	select {
	case a.jobs <- job:
		return result, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		return nil, fmt.Errorf("proof queue full")
	}
}

func (a *AsyncProver) Close() { a.once.Do(func() { close(a.stop) }) }
func (a *AsyncProver) worker() {
	for {
		select {
		case <-a.stop:
			return
		case job := <-a.jobs:
			select {
			case <-job.ctx.Done():
				job.result <- ProofResult{Err: job.ctx.Err()}
				close(job.result)
				continue
			default:
			}
			proof, err := a.prover.Prove(job.batch)
			job.result <- ProofResult{Proof: proof, Err: err}
			close(job.result)
		}
	}
}
