package checkpoint

import (
	"crypto/sha256"
	"fmt"

	"KNIRVCHAIN/internal/blockchain"

	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/std/hash/sha2"
	"github.com/consensys/gnark/std/math/uints"
)

var (
	emptyTxRoot = sha256.Sum256([]byte("knirv-tx-merkle-empty"))
	batchDomain = []byte("knirv-zk-batch-v1")
	blockDomain = []byte("knirv-block-hash-v1")
)

type circuitBlock struct {
	Enabled          frontend.Variable
	TxCount          frontend.Variable
	BlockHash        [32]frontend.Variable
	CoreDigest       [32]frontend.Variable
	AuthorDigest     [32]frontend.Variable
	TxLeaves         [][32]frontend.Variable
	AuthorSiblings   [][32]frontend.Variable
	AuthorDirections []frontend.Variable
}

type transitionCircuit struct {
	PreRoot       [32]frontend.Variable `gnark:",public"`
	PostRoot      [32]frontend.Variable `gnark:",public"`
	BatchHash     [32]frontend.Variable `gnark:",public"`
	AuthorSetRoot [32]frontend.Variable `gnark:",public"`
	BlockCount    frontend.Variable     `gnark:",public"`
	Blocks        []circuitBlock
}

func newTransitionCircuit(config PLONKConfig) *transitionCircuit {
	c := &transitionCircuit{Blocks: make([]circuitBlock, config.MaxBlocks)}
	for i := range c.Blocks {
		c.Blocks[i].TxLeaves = make([][32]frontend.Variable, config.MaxTxPerBlock)
		c.Blocks[i].AuthorSiblings = make([][32]frontend.Variable, config.AuthorTreeDepth)
		c.Blocks[i].AuthorDirections = make([]frontend.Variable, config.AuthorTreeDepth)
	}
	return c
}

func (c *transitionCircuit) Define(api frontend.API) error {
	byteAPI, err := uints.NewBytes(api)
	if err != nil {
		return err
	}
	toBytes := func(values [32]frontend.Variable) []uints.U8 {
		out := make([]uints.U8, 32)
		for i := range values {
			out[i] = byteAPI.ValueOf(values[i])
		}
		return out
	}
	assertBytes := func(expected [32]frontend.Variable, actual []uints.U8) {
		for i := 0; i < 32; i++ {
			api.AssertIsEqual(expected[i], byteAPI.Value(actual[i]))
		}
	}
	hashBytes := func(input []uints.U8) ([]uints.U8, error) {
		h, err := sha2.New(api)
		if err != nil {
			return nil, err
		}
		h.Write(input)
		return h.Sum(), nil
	}
	parent := func(left, right []uints.U8) ([]uints.U8, error) {
		input := make([]uints.U8, 0, 65)
		input = append(input, uints.NewU8(0x01))
		input = append(input, left...)
		input = append(input, right...)
		return hashBytes(input)
	}
	selectDigest := func(selector frontend.Variable, whenTrue, whenFalse []uints.U8) []uints.U8 {
		out := make([]uints.U8, 32)
		for i := range out {
			out[i] = byteAPI.ValueOf(api.Select(selector, byteAPI.Value(whenTrue[i]), byteAPI.Value(whenFalse[i])))
		}
		return out
	}
	merkleForCount := func(leaves [][]uints.U8, count int) ([]uints.U8, error) {
		if count == 0 {
			return uints.NewU8Array(emptyTxRoot[:]), nil
		}
		level := append([][]uints.U8(nil), leaves[:count]...)
		for len(level) > 1 {
			next := make([][]uints.U8, 0, (len(level)+1)/2)
			for i := 0; i < len(level); i += 2 {
				right := level[i]
				if i+1 < len(level) {
					right = level[i+1]
				}
				digest, err := parent(level[i], right)
				if err != nil {
					return nil, err
				}
				next = append(next, digest)
			}
			level = next
		}
		return level[0], nil
	}

	accumulator := toBytes(c.PreRoot)
	batchPreimage := uints.NewU8Array(batchDomain)
	enabledSum := frontend.Variable(0)

	for blockIndex := range c.Blocks {
		block := &c.Blocks[blockIndex]
		api.AssertIsBoolean(block.Enabled)
		if blockIndex+1 < len(c.Blocks) {
			// Active blocks form a prefix; no holes can hide unconstrained data.
			api.AssertIsEqual(api.Mul(c.Blocks[blockIndex+1].Enabled, api.Sub(1, block.Enabled)), 0)
		}
		enabledSum = api.Add(enabledSum, block.Enabled)

		leafBytes := make([][]uints.U8, len(block.TxLeaves))
		for i := range block.TxLeaves {
			leafBytes[i] = toBytes(block.TxLeaves[i])
		}
		selectedRoot := uints.NewU8Array(emptyTxRoot[:])
		validCount := frontend.Variable(0)
		for n := 0; n <= len(block.TxLeaves); n++ {
			candidate, err := merkleForCount(leafBytes, n)
			if err != nil {
				return err
			}
			matches := api.IsZero(api.Sub(block.TxCount, n))
			validCount = api.Add(validCount, matches)
			selectedRoot = selectDigest(matches, candidate, selectedRoot)
		}
		api.AssertIsEqual(validCount, 1)

		// Verify an in-circuit Merkle membership path for the block proposer.
		authorInput := append([]uints.U8{uints.NewU8(0x00)}, toBytes(block.AuthorDigest)...)
		authorNode, err := hashBytes(authorInput)
		if err != nil {
			return err
		}
		for level := range block.AuthorSiblings {
			api.AssertIsBoolean(block.AuthorDirections[level])
			sibling := toBytes(block.AuthorSiblings[level])
			left := selectDigest(block.AuthorDirections[level], sibling, authorNode)
			right := selectDigest(block.AuthorDirections[level], authorNode, sibling)
			authorNode, err = parent(left, right)
			if err != nil {
				return err
			}
		}
		for i := 0; i < 32; i++ {
			// Inactive padded blocks do not require an author proof.
			api.AssertIsEqual(api.Mul(block.Enabled, api.Sub(byteAPI.Value(authorNode[i]), c.AuthorSetRoot[i])), 0)
		}

		// Bind the admitted block hash to its metadata, transaction root, and
		// registered proposer. This closes the gap where an arbitrary author
		// membership witness could previously accompany an unrelated block hash.
		blockInput := uints.NewU8Array(blockDomain)
		blockInput = append(blockInput, toBytes(block.CoreDigest)...)
		blockInput = append(blockInput, selectedRoot...)
		blockInput = append(blockInput, toBytes(block.AuthorDigest)...)
		calculatedBlockHash, err := hashBytes(blockInput)
		if err != nil {
			return err
		}
		for i := 0; i < 32; i++ {
			api.AssertIsEqual(api.Mul(block.Enabled, api.Sub(byteAPI.Value(calculatedBlockHash[i]), block.BlockHash[i])), 0)
		}

		stateInput := make([]uints.U8, 0, 97)
		stateInput = append(stateInput, uints.NewU8(0x01))
		stateInput = append(stateInput, accumulator...)
		stateInput = append(stateInput, selectedRoot...)
		stateInput = append(stateInput, toBytes(block.BlockHash)...)
		nextAccumulator, err := hashBytes(stateInput)
		if err != nil {
			return err
		}
		accumulator = selectDigest(block.Enabled, nextAccumulator, accumulator)

		// Commit all padded witness slots into a fixed-size public batch digest.
		batchPreimage = append(batchPreimage, byteAPI.ValueOf(block.Enabled), byteAPI.ValueOf(block.TxCount))
		batchPreimage = append(batchPreimage, toBytes(block.BlockHash)...)
		batchPreimage = append(batchPreimage, toBytes(block.CoreDigest)...)
		batchPreimage = append(batchPreimage, toBytes(block.AuthorDigest)...)
		for i := range leafBytes {
			batchPreimage = append(batchPreimage, leafBytes[i]...)
		}
	}
	api.AssertIsEqual(c.BlockCount, enabledSum)
	assertBytes(c.PostRoot, accumulator)
	batchDigest, err := hashBytes(batchPreimage)
	if err != nil {
		return err
	}
	assertBytes(c.BatchHash, batchDigest)
	return nil
}

func (p *PLONKProver) buildAssignment(preRoot [32]byte, batch []*blockchain.Block) (*transitionCircuit, PLONKPublicInputs, error) {
	if len(batch) == 0 {
		return nil, PLONKPublicInputs{}, fmt.Errorf("cannot prove an empty block batch")
	}
	if len(batch) > p.config.MaxBlocks {
		return nil, PLONKPublicInputs{}, fmt.Errorf("batch has %d blocks; circuit capacity is %d", len(batch), p.config.MaxBlocks)
	}
	assignment := newTransitionCircuit(p.config)
	setBytes(assignment.PreRoot[:], preRoot[:])
	authorRoot := p.authors.root()
	setBytes(assignment.AuthorSetRoot[:], authorRoot[:])
	assignment.BlockCount = len(batch)

	accumulator := preRoot
	for i, block := range batch {
		if block == nil {
			return nil, PLONKPublicInputs{}, fmt.Errorf("block %d is nil", i)
		}
		if i > 0 && block.BlockNumber != batch[i-1].BlockNumber+1 {
			return nil, PLONKPublicInputs{}, fmt.Errorf("non-contiguous block heights %d and %d", batch[i-1].BlockNumber, block.BlockNumber)
		}
		if len(block.BlockHash) != 32 {
			return nil, PLONKPublicInputs{}, fmt.Errorf("block %d hash length is %d, want 32", block.BlockNumber, len(block.BlockHash))
		}
		leaves := blockchain.TxMerkleLeafHashes(block.Transactions)
		if len(leaves) > p.config.MaxTxPerBlock {
			return nil, PLONKPublicInputs{}, fmt.Errorf("block %d has %d transactions; circuit capacity is %d", block.BlockNumber, len(leaves), p.config.MaxTxPerBlock)
		}
		digest, siblings, directions, err := p.authors.proof(block.ProposerAddress)
		if err != nil {
			return nil, PLONKPublicInputs{}, err
		}
		w := &assignment.Blocks[i]
		w.Enabled = 1
		w.TxCount = len(leaves)
		setBytes(w.BlockHash[:], block.BlockHash)
		coreDigest := blockchain.BlockCoreDigest(block)
		setBytes(w.CoreDigest[:], coreDigest[:])
		setBytes(w.AuthorDigest[:], digest[:])
		for j := range leaves {
			setBytes(w.TxLeaves[j][:], leaves[j][:])
		}
		for j := range siblings {
			setBytes(w.AuthorSiblings[j][:], siblings[j][:])
			w.AuthorDirections[j] = directions[j]
		}
		accumulator = blockchain.BlockStateRoot(block, accumulator)
	}
	for i := len(batch); i < len(assignment.Blocks); i++ {
		assignment.Blocks[i].Enabled = 0
		assignment.Blocks[i].TxCount = 0
	}
	setBytes(assignment.PostRoot[:], accumulator[:])
	batchHash := assignmentBatchHash(assignment)
	setBytes(assignment.BatchHash[:], batchHash[:])

	public := PLONKPublicInputs{
		CircuitVersion: PLONKCircuitVersion,
		ChainID:        p.config.ChainID,
		StartHeight:    batch[0].BlockNumber,
		EndHeight:      batch[len(batch)-1].BlockNumber,
		BlockCount:     uint8(len(batch)),
		PreRoot:        preRoot,
		PostRoot:       accumulator,
		BatchHash:      batchHash,
		AuthorSetRoot:  p.authors.root(),
	}
	return assignment, public, nil
}

func publicOnlyAssignment(config PLONKConfig, public PLONKPublicInputs) *transitionCircuit {
	assignment := newTransitionCircuit(config)
	setBytes(assignment.PreRoot[:], public.PreRoot[:])
	setBytes(assignment.PostRoot[:], public.PostRoot[:])
	setBytes(assignment.BatchHash[:], public.BatchHash[:])
	setBytes(assignment.AuthorSetRoot[:], public.AuthorSetRoot[:])
	assignment.BlockCount = public.BlockCount
	return assignment
}

func setBytes(dst []frontend.Variable, src []byte) {
	for i := range dst {
		if i < len(src) {
			dst[i] = src[i]
		} else {
			dst[i] = 0
		}
	}
}

func assignmentByte(v frontend.Variable) byte {
	switch value := v.(type) {
	case uint8:
		return byte(value)
	case int:
		return byte(value)
	default:
		return 0
	}
}

func assignmentBatchHash(assignment *transitionCircuit) [32]byte {
	preimage := append([]byte(nil), batchDomain...)
	for i := range assignment.Blocks {
		block := &assignment.Blocks[i]
		preimage = append(preimage, assignmentByte(block.Enabled), assignmentByte(block.TxCount))
		for _, value := range block.BlockHash {
			preimage = append(preimage, assignmentByte(value))
		}
		for _, value := range block.CoreDigest {
			preimage = append(preimage, assignmentByte(value))
		}
		for _, value := range block.AuthorDigest {
			preimage = append(preimage, assignmentByte(value))
		}
		for j := range block.TxLeaves {
			for _, value := range block.TxLeaves[j] {
				preimage = append(preimage, assignmentByte(value))
			}
		}
	}
	return sha256.Sum256(preimage)
}
