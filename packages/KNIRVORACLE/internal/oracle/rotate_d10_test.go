package oracle

import (
	"crypto/ecdsa"
	"testing"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/knirvcorp/knirvoracle/internal/oracle/crypto"
	"github.com/knirvcorp/knirvoracle/internal/oracle/types"
	"go.uber.org/zap"
)

func newD10TestOracle(t *testing.T) *Oracle {
	t.Helper()
	ownerKey, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("generate owner key: %v", err)
	}
	cfg := DefaultOracleConfig()
	cfg.Mode = ModeActive
	cfg.OwnerPrivateKey = ownerKey.PrivateKeyHex()
	cfg.DataDir = t.TempDir()
	cfg.DBBackend = "memory"
	instance, err := NewOracle(cfg, zap.NewNop())
	if err != nil {
		t.Fatalf("new oracle: %v", err)
	}
	return instance
}

// genAuthor returns a registered author + its secp256k1 private key.
func genD10Author(t *testing.T) (types.RegisteredAuthor, *ecdsa.PrivateKey) {
	t.Helper()
	key, err := ethcrypto.GenerateKey()
	if err != nil {
		t.Fatalf("gen key: %v", err)
	}
	cpk := ethcrypto.CompressPubkey(&key.PublicKey)
	addr := types.OracleAddress(&key.PublicKey)
	return types.RegisteredAuthor{Address: addr, PubKey: cpk, Weight: 1}, key
}

func registerD10Chain(t *testing.T, o *Oracle, chainID string, authors []types.RegisteredAuthor) {
	t.Helper()
	// Seed the in-memory registry directly (package-internal access) so the test
	// exercises the D10 guard on RotateChain without needing the economics-engine
	// bond lock or a funded owner.
	reg := &types.ChainRegistration{
		ChainID:     chainID,
		Authors:     authors,
		QuorumNumer: 2,
		QuorumDenom: 3,
		Bond:        10000,
		ProofWindow: 256,
	}
	if err := o.checkpoint.registry.Register(reg); err != nil {
		t.Fatalf("register %s: %v", chainID, err)
	}
}

// TestRotateChainRejectsAuthorityChain_D10 asserts that a quorum-signed rotation
// of an authority chain's author to a session key is hard-rejected at the Oracle
// boundary (owner decision D10) and leaves the registration unchanged.
func TestRotateChainRejectsAuthorityChain_D10(t *testing.T) {
	o := newD10TestOracle(t)
	const chainID = "authority/operator1"
	a1, k1 := genD10Author(t)
	a2, k2 := genD10Author(t)
	a3, _ := genD10Author(t)
	registerD10Chain(t, o, chainID, []types.RegisteredAuthor{a1, a2, a3})

	// A valid quorum-signed rotation from the long-term author set to a distinct
	// session key — exactly the operation D10 prohibits.
	session, _ := genD10Author(t)
	next := types.ChainRegistration{
		ChainID:     chainID,
		Authors:     []types.RegisteredAuthor{session},
		QuorumNumer: 2,
		QuorumDenom: 3,
		Bond:        10000,
		ProofWindow: 256,
	}
	if err := types.SignRegistration(&next, k1); err != nil {
		t.Fatalf("sign rotation: %v", err)
	}
	if err := types.SignRegistration(&next, k2); err != nil {
		t.Fatalf("sign rotation: %v", err)
	}

	err := o.RotateChain(&next)
	if err == nil {
		t.Fatal("RotateChain allowed an authority-chain author rotation; D10 violation")
	}
	if err.Error() != "authority chain author rotation is deprecated (D10)" {
		t.Fatalf("unexpected error: %v", err)
	}

	got, ok := o.GetChainRegistration(chainID)
	if !ok {
		t.Fatal("authority chain vanished after a rejected rotation")
	}
	if len(got.Authors) != 3 {
		t.Fatalf("rotation mutated the registration: %d authors, want 3", len(got.Authors))
	}
}

// TestRotateChainAllowsNonAuthorityChain confirms the generic rotation path is
// preserved for non-authority chains.
func TestRotateChainAllowsNonAuthorityChain(t *testing.T) {
	o := newD10TestOracle(t)
	const chainID = "foreign-chain-1"
	a1, k1 := genD10Author(t)
	a2, k2 := genD10Author(t)
	a3, _ := genD10Author(t)
	registerD10Chain(t, o, chainID, []types.RegisteredAuthor{a1, a2, a3})

	session, _ := genD10Author(t)
	next := types.ChainRegistration{
		ChainID:     chainID,
		Authors:     []types.RegisteredAuthor{session},
		QuorumNumer: 2,
		QuorumDenom: 3,
		ProofWindow: 256,
	}
	if err := types.SignRegistration(&next, k1); err != nil {
		t.Fatalf("sign rotation: %v", err)
	}
	if err := types.SignRegistration(&next, k2); err != nil {
		t.Fatalf("sign rotation: %v", err)
	}
	if err := o.RotateChain(&next); err != nil {
		t.Fatalf("RotateChain rejected a non-authority chain: %v", err)
	}
	got, ok := o.GetChainRegistration(chainID)
	if !ok {
		t.Fatal("chain vanished")
	}
	if len(got.Authors) != 1 || got.Authors[0].Address != session.Address {
		t.Fatalf("rotation not applied: %+v", got.Authors)
	}
}
