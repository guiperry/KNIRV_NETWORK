package checkpoint

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"KNIRVCHAIN/internal/blockchain"

	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark-crypto/kzg"
	"github.com/consensys/gnark/backend/plonk"
	"github.com/consensys/gnark/constraint"
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/frontend/cs/scs"
	"github.com/consensys/gnark/test/unsafekzg"
)

const (
	TransitionProofSchema = "knirv.transition-proof.v1"
	PLONKProofSystem      = "plonk"
	PLONKCircuitVersion   = "knirv.accum-plonk.v1"
)

// TransitionProof is the stable phase-2 wire object. Proof contains a
// serialized gnark PLONK proof; PublicInputs contains PLONKPublicInputs JSON.
type TransitionProof struct {
	SchemaVersion string   `json:"schema_version"`
	ChainID       string   `json:"chain_id"`
	StartHeight   uint64   `json:"start_height"`
	EndHeight     uint64   `json:"end_height"`
	PreRoot       [32]byte `json:"pre_root"`
	PostRoot      [32]byte `json:"post_root"`
	ProofSystem   string   `json:"proof_system"`
	Proof         []byte   `json:"proof"`
	PublicInputs  []byte   `json:"public_inputs"`
}

// Prover is intentionally proof-system agnostic so hashchain-v0 remains a
// degraded fallback when PLONK proving is unavailable.
type Prover interface {
	Prove(batch []*blockchain.Block) (*TransitionProof, error)
}

// PLONKConfig fixes circuit capacity. A chain registers the resulting
// verification-key envelope and must retain these values until key rotation.
type PLONKConfig struct {
	ChainID         string `json:"chain_id"`
	MaxBlocks       int    `json:"max_blocks"`
	MaxTxPerBlock   int    `json:"max_tx_per_block"`
	AuthorTreeDepth int    `json:"author_tree_depth"`
}

func (c PLONKConfig) validate() error {
	if c.ChainID == "" {
		return fmt.Errorf("chain id required")
	}
	if c.MaxBlocks <= 0 || c.MaxBlocks > 255 {
		return fmt.Errorf("max_blocks must be in [1,255]")
	}
	if c.MaxTxPerBlock <= 0 || c.MaxTxPerBlock > 255 {
		return fmt.Errorf("max_tx_per_block must be in [1,255]")
	}
	if c.AuthorTreeDepth < 0 || c.AuthorTreeDepth > 16 {
		return fmt.Errorf("author_tree_depth must be in [0,16]")
	}
	return nil
}

// PLONKPublicInputs is canonical JSON carried beside the proof. Roots and the
// batch digest are also public circuit variables, so changing this object makes
// verification fail.
type PLONKPublicInputs struct {
	CircuitVersion string   `json:"circuit_version"`
	ChainID        string   `json:"chain_id"`
	StartHeight    uint64   `json:"start_height"`
	EndHeight      uint64   `json:"end_height"`
	BlockCount     uint8    `json:"block_count"`
	PreRoot        [32]byte `json:"pre_root"`
	PostRoot       [32]byte `json:"post_root"`
	BatchHash      [32]byte `json:"batch_hash"`
	AuthorSetRoot  [32]byte `json:"author_set_root"`
}

// VerificationKeyEnvelope makes the otherwise opaque gnark key
// self-describing. The Oracle stores this byte-for-byte in ChainRegistration.
type VerificationKeyEnvelope struct {
	CircuitVersion string      `json:"circuit_version"`
	Curve          string      `json:"curve"`
	Config         PLONKConfig `json:"config"`
	AuthorSetRoot  [32]byte    `json:"author_set_root"`
	Key            []byte      `json:"key"`
}

// PLONKProver owns the circuit-specific preprocessing key. Setup is deliberately
// separate from block production; Prove is safe to run asynchronously after a
// provisional checkpoint has been posted.
type PLONKProver struct {
	config  PLONKConfig
	authors *authorMerkleTree
	ccs     constraint.ConstraintSystem
	pk      plonk.ProvingKey
	vk      plonk.VerifyingKey
	preRoot [32]byte
}

// NewPLONKProver performs circuit-specific PLONK preprocessing against an
// externally supplied universal KZG SRS. Production callers must use an MPC
// ceremony SRS and its lagrange form.
func NewPLONKProver(config PLONKConfig, authors []string, canonical, lagrange kzg.SRS) (*PLONKProver, error) {
	if err := config.validate(); err != nil {
		return nil, err
	}
	tree, err := newAuthorMerkleTree(authors, config.AuthorTreeDepth)
	if err != nil {
		return nil, err
	}
	ccs, err := compileTransitionCircuit(config)
	if err != nil {
		return nil, err
	}
	pk, vk, err := plonk.Setup(ccs, canonical, lagrange)
	if err != nil {
		return nil, fmt.Errorf("plonk setup: %w", err)
	}
	return &PLONKProver{config: config, authors: tree, ccs: ccs, pk: pk, vk: vk}, nil
}

// NewUnsafePLONKProver is for local development, tests, and benchmarks only.
// Its deterministically cached SRS is not the production ceremony SRS.
func NewUnsafePLONKProver(config PLONKConfig, authors []string) (*PLONKProver, error) {
	if err := config.validate(); err != nil {
		return nil, err
	}
	tree, err := newAuthorMerkleTree(authors, config.AuthorTreeDepth)
	if err != nil {
		return nil, err
	}
	ccs, err := compileTransitionCircuit(config)
	if err != nil {
		return nil, err
	}
	canonical, lagrange, err := unsafekzg.NewSRS(ccs, unsafekzg.WithFSCache())
	if err != nil {
		return nil, fmt.Errorf("development KZG setup: %w", err)
	}
	pk, vk, err := plonk.Setup(ccs, canonical, lagrange)
	if err != nil {
		return nil, fmt.Errorf("plonk setup: %w", err)
	}
	return &PLONKProver{config: config, authors: tree, ccs: ccs, pk: pk, vk: vk}, nil
}

func compileTransitionCircuit(config PLONKConfig) (constraint.ConstraintSystem, error) {
	template := newTransitionCircuit(config)
	ccs, err := frontend.Compile(ecc.BN254.ScalarField(), scs.NewBuilder, template)
	if err != nil {
		return nil, fmt.Errorf("compile transition circuit: %w", err)
	}
	return ccs, nil
}

// SetPreRoot sets the accumulator immediately preceding the next batch. It is
// useful when a caller only has the Prover interface; ProveTransition is less
// stateful and preferred by schedulers processing multiple chains.
func (p *PLONKProver) SetPreRoot(root [32]byte) { p.preRoot = root }

func (p *PLONKProver) Prove(batch []*blockchain.Block) (*TransitionProof, error) {
	return p.ProveTransition(p.preRoot, batch)
}

func (p *PLONKProver) ProveTransition(preRoot [32]byte, batch []*blockchain.Block) (*TransitionProof, error) {
	assignment, public, err := p.buildAssignment(preRoot, batch)
	if err != nil {
		return nil, err
	}
	fullWitness, err := frontend.NewWitness(assignment, ecc.BN254.ScalarField())
	if err != nil {
		return nil, fmt.Errorf("build PLONK witness: %w", err)
	}
	proof, err := plonk.Prove(p.ccs, p.pk, fullWitness)
	if err != nil {
		return nil, fmt.Errorf("generate PLONK proof: %w", err)
	}
	var proofBytes bytes.Buffer
	if _, err := proof.WriteTo(&proofBytes); err != nil {
		return nil, fmt.Errorf("serialize PLONK proof: %w", err)
	}
	publicBytes, err := json.Marshal(public)
	if err != nil {
		return nil, err
	}
	return &TransitionProof{
		SchemaVersion: TransitionProofSchema,
		ChainID:       p.config.ChainID,
		StartHeight:   public.StartHeight,
		EndHeight:     public.EndHeight,
		PreRoot:       public.PreRoot,
		PostRoot:      public.PostRoot,
		ProofSystem:   PLONKProofSystem,
		Proof:         proofBytes.Bytes(),
		PublicInputs:  publicBytes,
	}, nil
}

func (p *PLONKProver) VerificationKey() ([]byte, error) {
	var raw bytes.Buffer
	if _, err := p.vk.WriteTo(&raw); err != nil {
		return nil, fmt.Errorf("serialize PLONK verification key: %w", err)
	}
	envelope := VerificationKeyEnvelope{
		CircuitVersion: PLONKCircuitVersion,
		Curve:          ecc.BN254.String(),
		Config:         p.config,
		AuthorSetRoot:  p.authors.root(),
		Key:            raw.Bytes(),
	}
	return json.Marshal(envelope)
}

func (p *PLONKProver) VerifyLocally(proof *TransitionProof) error {
	vk, err := p.VerificationKey()
	if err != nil {
		return err
	}
	return VerifyPLONKTransition(proof, vk)
}

// VerifyPLONKTransition is kept on the chain for self-tests and benchmark
// tuning. The backend carries an independent verify-only copy.
func VerifyPLONKTransition(tp *TransitionProof, keyEnvelope []byte) error {
	if tp == nil || tp.ProofSystem != PLONKProofSystem {
		return fmt.Errorf("plonk transition proof required")
	}
	var env VerificationKeyEnvelope
	if err := json.Unmarshal(keyEnvelope, &env); err != nil {
		return fmt.Errorf("decode verification key envelope: %w", err)
	}
	if env.CircuitVersion != PLONKCircuitVersion || env.Curve != ecc.BN254.String() {
		return fmt.Errorf("unsupported circuit %q on curve %q", env.CircuitVersion, env.Curve)
	}
	var public PLONKPublicInputs
	if err := json.Unmarshal(tp.PublicInputs, &public); err != nil {
		return fmt.Errorf("decode PLONK public inputs: %w", err)
	}
	if err := bindTransitionPublicInputs(tp, env, public); err != nil {
		return err
	}
	assignment := publicOnlyAssignment(env.Config, public)
	publicWitness, err := frontend.NewWitness(assignment, ecc.BN254.ScalarField(), frontend.PublicOnly())
	if err != nil {
		return fmt.Errorf("build public witness: %w", err)
	}
	proof := plonk.NewProof(ecc.BN254)
	if _, err := proof.ReadFrom(bytes.NewReader(tp.Proof)); err != nil {
		return fmt.Errorf("decode PLONK proof: %w", err)
	}
	vk := plonk.NewVerifyingKey(ecc.BN254)
	if _, err := vk.ReadFrom(bytes.NewReader(env.Key)); err != nil && err != io.EOF {
		return fmt.Errorf("decode PLONK verification key: %w", err)
	}
	if err := plonk.Verify(proof, vk, publicWitness); err != nil {
		return fmt.Errorf("verify PLONK proof: %w", err)
	}
	return nil
}

func bindTransitionPublicInputs(tp *TransitionProof, env VerificationKeyEnvelope, public PLONKPublicInputs) error {
	if public.CircuitVersion != PLONKCircuitVersion || public.ChainID != env.Config.ChainID || tp.ChainID != env.Config.ChainID {
		return fmt.Errorf("proof chain/circuit does not match verification key")
	}
	if public.AuthorSetRoot != env.AuthorSetRoot {
		return fmt.Errorf("proof author set does not match verification key")
	}
	if public.PreRoot != tp.PreRoot || public.PostRoot != tp.PostRoot || public.StartHeight != tp.StartHeight || public.EndHeight != tp.EndHeight {
		return fmt.Errorf("transition proof fields do not match public witness")
	}
	if public.BlockCount == 0 || int(public.BlockCount) > env.Config.MaxBlocks {
		return fmt.Errorf("invalid public block count %d", public.BlockCount)
	}
	if public.EndHeight < public.StartHeight || public.EndHeight-public.StartHeight+1 != uint64(public.BlockCount) {
		return fmt.Errorf("public height range does not match block count")
	}
	return nil
}

type authorMerkleTree struct {
	depth  int
	leaves [][32]byte
	levels [][][32]byte
	index  map[string]int
}

func newAuthorMerkleTree(authors []string, depth int) (*authorMerkleTree, error) {
	capacity := 1 << depth
	if depth == 0 {
		capacity = 1
	}
	unique := make(map[string]struct{})
	for _, author := range authors {
		n := normalizeAuthor(author)
		if n != "" {
			unique[n] = struct{}{}
		}
	}
	ordered := make([]string, 0, len(unique))
	for author := range unique {
		ordered = append(ordered, author)
	}
	sort.Strings(ordered)
	if len(ordered) == 0 {
		return nil, fmt.Errorf("at least one PoAu-D author is required")
	}
	if len(ordered) > capacity {
		return nil, fmt.Errorf("%d authors exceed tree capacity %d", len(ordered), capacity)
	}
	t := &authorMerkleTree{depth: depth, leaves: make([][32]byte, capacity), index: make(map[string]int)}
	emptyDigest := [32]byte{}
	for i := range t.leaves {
		t.leaves[i] = merkleLeafFromDigest(emptyDigest)
	}
	for i, author := range ordered {
		digest := sha256.Sum256([]byte(author))
		t.leaves[i] = merkleLeafFromDigest(digest)
		t.index[author] = i
	}
	t.levels = append(t.levels, append([][32]byte(nil), t.leaves...))
	for len(t.levels[len(t.levels)-1]) > 1 {
		level := t.levels[len(t.levels)-1]
		next := make([][32]byte, len(level)/2)
		for i := range next {
			next[i] = merkleParent(level[2*i], level[2*i+1])
		}
		t.levels = append(t.levels, next)
	}
	return t, nil
}

func (t *authorMerkleTree) root() [32]byte { return t.levels[len(t.levels)-1][0] }

func (t *authorMerkleTree) proof(author string) ([32]byte, [][32]byte, []uint8, error) {
	n := normalizeAuthor(author)
	idx, ok := t.index[n]
	if !ok {
		return [32]byte{}, nil, nil, fmt.Errorf("block proposer %q is not a registered PoAu-D author", author)
	}
	digest := sha256.Sum256([]byte(n))
	siblings := make([][32]byte, t.depth)
	directions := make([]uint8, t.depth)
	pos := idx
	for level := 0; level < t.depth; level++ {
		siblings[level] = t.levels[level][pos^1]
		directions[level] = uint8(pos & 1)
		pos /= 2
	}
	return digest, siblings, directions, nil
}

func normalizeAuthor(author string) string { return strings.ToLower(strings.TrimSpace(author)) }

func merkleLeafFromDigest(digest [32]byte) [32]byte {
	preimage := make([]byte, 1+32)
	copy(preimage[1:], digest[:])
	return sha256.Sum256(preimage)
}

func merkleParent(left, right [32]byte) [32]byte {
	preimage := make([]byte, 1+64)
	preimage[0] = 0x01
	copy(preimage[1:33], left[:])
	copy(preimage[33:], right[:])
	return sha256.Sum256(preimage)
}
