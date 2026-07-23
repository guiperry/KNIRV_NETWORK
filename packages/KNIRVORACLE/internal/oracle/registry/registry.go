package registry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/knirvcorp/knirvoracle/internal/oracle/types"
)

// DefaultProofWindow is the wall-clock window, in seconds, allowed for
// phase-2 proof submission when a chain does not specify its own — about 30
// minutes, over ten times the measured 175-second cold development
// setup/prove/verify cycle for the minimal PLONK circuit. This is real time,
// not an Oracle block-height count: the Oracle's own commit cadence is just a
// heartbeat (see consensus/engine.go's consensusLoop), not a security clock,
// so a chain's dispute window must not depend on how often — or whether —
// the Oracle happens to be ticking.
const DefaultProofWindow uint64 = 1800

// Registry is the Oracle's per-foreign-chain trust anchor: which chains may
// post checkpoints, who their authors are, and continuity state. It is the
// "light-client registry" from Gap G3.
type Registry struct {
	mu     sync.RWMutex
	path   string
	chains map[string]*types.ChainRegistration
}

// New returns an empty in-memory registry.
func New() *Registry {
	return &Registry{chains: make(map[string]*types.ChainRegistration)}
}

// Load reads the registry from chain_registry.json next to the Oracle data dir.
// A missing file yields an empty registry (first-run).
func Load(dataDir string) (*Registry, error) {
	r := New()
	if dataDir == "" {
		return r, nil
	}
	r.path = filepath.Join(dataDir, "chain_registry.json")
	data, err := os.ReadFile(r.path)
	if err != nil {
		if os.IsNotExist(err) {
			return r, nil
		}
		return nil, fmt.Errorf("read registry: %w", err)
	}
	var list []*types.ChainRegistration
	if err := json.Unmarshal(data, &list); err != nil {
		return nil, fmt.Errorf("decode registry: %w", err)
	}
	for _, c := range list {
		if c != nil {
			r.chains[c.ChainID] = c
		}
	}
	return r, nil
}

// Persist writes the registry to chain_registry.json atomically.
func (r *Registry) Persist() error {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.path == "" {
		return nil
	}
	list := make([]*types.ChainRegistration, 0, len(r.chains))
	for _, c := range r.chains {
		list = append(list, c)
	}
	payload, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	tmp := r.path + ".tmp"
	if err := os.WriteFile(tmp, payload, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, r.path)
}

// Register adds (or replaces) a chain's registration.
func (r *Registry) Register(c *types.ChainRegistration) error {
	if c.ChainID == "" {
		return fmt.Errorf("empty chain id")
	}
	if c.QuorumDenom == 0 {
		return fmt.Errorf("quorum denom must be > 0")
	}
	if len(c.Authors) == 0 {
		return fmt.Errorf("no registered authors")
	}
	if c.ProofWindow == 0 {
		c.ProofWindow = DefaultProofWindow
	}
	if c.Bond > 0 && c.BondRemaining == 0 && c.BondSlashed == 0 {
		c.BondRemaining = c.Bond
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.chains[c.ChainID] = c
	return nil
}

// ApplySlash updates the persisted indexed bond view after the economics
// staking manager burns part of a locked chain bond.
func (r *Registry) ApplySlash(chainID string, amount uint64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	c, ok := r.chains[chainID]
	if !ok {
		return fmt.Errorf("chain %s not registered", chainID)
	}
	if amount > c.BondRemaining {
		amount = c.BondRemaining
	}
	c.BondRemaining -= amount
	c.BondSlashed += amount
	return nil
}

// Get returns a chain's registration.
func (r *Registry) Get(chainID string) (*types.ChainRegistration, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.chains[chainID]
	return c, ok
}

// List returns all registered chains.
func (r *Registry) List() []*types.ChainRegistration {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*types.ChainRegistration, 0, len(r.chains))
	for _, c := range r.chains {
		out = append(out, c)
	}
	return out
}

// AuthorWeight returns the registered weight of an author address, and whether
// it is a recognized author for the chain.
func (r *Registry) AuthorWeight(chainID, addr string) (uint64, bool) {
	c, ok := r.Get(chainID)
	if !ok {
		return 0, false
	}
	for _, a := range c.Authors {
		if a.Address == addr {
			w := a.Weight
			if w == 0 {
				w = 1
			}
			return w, true
		}
	}
	return 0, false
}

// VerifyCheckpointQuorum validates a checkpoint's signatures against the
// registry's author set and quorum by weight, and enforces continuity
// (StartHeight == LastHeight+1 and PrevCheckHash == LastCheckHash).
func (r *Registry) VerifyCheckpointQuorum(cp *types.Checkpoint) error {
	c, ok := r.Get(cp.ChainID)
	if !ok {
		return fmt.Errorf("chain %s not registered", cp.ChainID)
	}
	if c.Bond > 0 && c.BondRemaining == 0 {
		return fmt.Errorf("chain %s bond is exhausted", cp.ChainID)
	}

	// Admission order is intentionally stable: registration/bond, height
	// continuity, PoAu-D proposer membership, author quorum, then the previous
	// checkpoint hash link.
	if cp.StartHeight != c.LastHeight+1 {
		return fmt.Errorf("non-contiguous start: got %d want %d", cp.StartHeight, c.LastHeight+1)
	}
	if _, ok := authorWeight(c, cp.Proposer); !ok {
		return fmt.Errorf("proposer %s is not a registered author", cp.Proposer)
	}
	if err := VerifySignatureQuorum(c, cp.Digest(), cp.Signatures); err != nil {
		return err
	}
	if cp.PrevCheckHash != c.LastCheckHash {
		return fmt.Errorf("prev check hash mismatch")
	}
	return nil
}

// VerifySignatureQuorum is the single weighted-quorum implementation shared
// by checkpoint admission, initial registration, and registry rotation.
func VerifySignatureQuorum(c *types.ChainRegistration, digest [32]byte, sigs []types.AuthorSig) error {
	if c == nil || len(c.Authors) == 0 {
		return fmt.Errorf("author registry required")
	}
	totalWeight := uint64(0)
	for _, a := range c.Authors {
		w := a.Weight
		if w == 0 {
			w = 1
		}
		totalWeight += w
	}
	seen := make(map[string]bool)
	validWeight := uint64(0)
	for _, sig := range sigs {
		if seen[sig.Address] {
			continue
		}
		w, ok := authorWeight(c, sig.Address)
		if !ok {
			continue
		}
		if !types.VerifyAuthorSigDigest(digest, sig) {
			continue
		}
		seen[sig.Address] = true
		validWeight += w
	}
	numer, denom := c.QuorumNumer, c.QuorumDenom
	if numer == 0 {
		numer = 2
	}
	if denom == 0 {
		denom = 3
	}
	if numer > denom || validWeight*denom < totalWeight*numer {
		return fmt.Errorf("quorum not met: %d/%d weight", validWeight, totalWeight)
	}
	return nil
}

func authorWeight(c *types.ChainRegistration, addr string) (uint64, bool) {
	for _, a := range c.Authors {
		if a.Address == addr {
			if a.Weight == 0 {
				return 1, true
			}
			return a.Weight, true
		}
	}
	return 0, false
}

// Advance updates the chain's continuity cursor after a checkpoint is admitted.
func (r *Registry) Advance(chainID string, lastHeight uint64, lastCheckHash [32]byte) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	c, ok := r.chains[chainID]
	if !ok {
		return fmt.Errorf("chain %s not registered", chainID)
	}
	c.LastHeight = lastHeight
	c.LastCheckHash = lastCheckHash
	return nil
}

// SetVerificationKey enrolls a SNARK verification key + preferred proof system
// for a registered chain (merkle-math.md Phase 5). The key lets the Oracle
// accept groth16/plonk finality records; hashchain-v0 needs no key.
func (r *Registry) SetVerificationKey(chainID, proofSystem string, vk []byte) error {
	if proofSystem == "" {
		proofSystem = "hashchain-v0"
	}
	if proofSystem != "hashchain-v0" && proofSystem != "groth16" && proofSystem != "plonk" {
		return fmt.Errorf("unsupported proof system %q", proofSystem)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	c, ok := r.chains[chainID]
	if !ok {
		return fmt.Errorf("chain %s not registered", chainID)
	}
	c.VerificationKey = vk
	c.PreferredProofSystem = proofSystem
	return nil
}
