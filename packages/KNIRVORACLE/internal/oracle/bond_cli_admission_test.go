package oracle

import (
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	knirvsigning "github.com/guiperry/knirv-sdk-go/signing"
	"github.com/knirvcorp/knirvoracle/internal/oracle/ledger"
	"github.com/knirvcorp/knirvoracle/internal/oracle/types"
)

// signAuthorityHold signs a bond-hold on the Oracle's chain id with an ECDSA key,
// mirroring the KNIRVCONTROLLER-signed hold the CLI posts for D14 enrolment.
func signAuthorityHold(t *testing.T, key *ecdsa.PrivateKey, chainID string, hold knirvsigning.BondHold) knirvsigning.SignedMessage {
	t.Helper()
	payload, _ := json.Marshal(hold)
	signed, err := knirvsigning.SignMessage(crypto.FromECDSA(key), knirvsigning.MessageEnvelope{
		SchemaVersion: knirvsigning.MessageSchemaVersion,
		Domain:        knirvsigning.RegistryDomain,
		Purpose:       knirvsigning.PurposeBondHold,
		ChainID:       chainID,
		Nonce:         "bond-hold:" + hold.HoldID,
		IssuedAtUnix:  time.Now().Unix(),
		ExpiresAtUnix: int64(hold.ExpiresAtUnix),
		Payload:       payload,
	})
	if err != nil {
		t.Fatalf("sign hold: %v", err)
	}
	return signed
}

// newAuthorityBondRegistration builds a types.ChainRegistration that mirrors exactly
// what the KNIRV_CORP CLI posts for the D14 authority/* bond: chain_id
// "authority/<addr>", a single author carrying its pubkey, quorum 1/1, the D14
// bond of 10,000 NRN, a bond owner, and proof window 256, backed by a valid D14
// enrolment hold (the escrow flow) owned by the bond owner and covering the bond.
// It is signed with the Oracle's own scheme (types.SignRegistration) so the real
// admission path can be exercised end-to-end.
func newAuthorityBondRegistration(t *testing.T, o *Oracle, key *ecdsa.PrivateKey) types.ChainRegistration {
	t.Helper()
	cpk := crypto.CompressPubkey(&key.PublicKey)
	addr := types.OracleAddress(&key.PublicKey)
	tok := o.GetNRNToken()
	// Fund the authority operator and create a valid D14 enrolment hold so the
	// bonded authority registration can consume it (the CLI D14 flow posts the hold
	// alongside Bond=10000).
	opAddr, err := types.AddressFromString(addr)
	if err != nil {
		t.Fatalf("authority address: %v", err)
	}
	if _, err := tok.Mint(opAddr, big.NewInt(2_000_000_000)); err != nil {
		t.Fatalf("mint: %v", err)
	}
	// The D14 bond is locked by the Oracle's economics engine, which requires the bond
	// owner to have an active stake.
	if _, err := o.economicsEngine.GetStakingManager().Stake(opAddr, big.NewInt(1_000_000_000)); err != nil {
		t.Fatalf("stake: %v", err)
	}
	holdID := "bond:" + addr
	expiry := time.Now().Add(time.Hour)
	hold := knirvsigning.BondHold{
		HoldID:         holdID,
		Operator:       addr,
		Amount:         "10000",
		RootRegistryID: tok.RootRegistryID(),
		ExpiresAtUnix:  uint64(expiry.Unix()),
	}
	if err := tok.Hold(holdID, hold, signAuthorityHold(t, key, tok.ChainID(), hold), time.Now()); err != nil {
		t.Fatalf("hold: %v", err)
	}
	reg := types.ChainRegistration{
		ChainID:     "authority/" + addr,
		Authors:     []types.RegisteredAuthor{{Address: addr, PubKey: cpk, Weight: 1}},
		QuorumNumer: 1,
		QuorumDenom: 1,
		LastHeight:  0,
		Bond:        10000,
		BondOwner:   addr,
		ProofWindow: 256,
		HoldID:      holdID,
	}
	if err := types.SignRegistration(&reg, key); err != nil {
		t.Fatalf("sign registration: %v", err)
	}
	return reg
}

// TestAuthorityBondRegistrationAcceptedByRealOracle proves the D14 authority
// registration (complete author set with pubkey, D14 bond + valid enrolment hold,
// controller-signed rotation_sig over the Oracle's exact digest) is accepted by the
// real Oracle.RegisterChain admission path and consumes the D14 escrow exactly
// once. This is the audit-required test against the real Oracle handler.
func TestAuthorityBondRegistrationAcceptedByRealOracle(t *testing.T) {
	o, opKey := newBondOracle(t)
	key := opKey.PrivateKey
	reg := newAuthorityBondRegistration(t, o, key)

	if err := o.RegisterChain(&reg); err != nil {
		t.Fatalf("real Oracle rejected a complete, correctly-signed authority registration: %v", err)
	}
	got, ok := o.checkpoint.registry.Get(reg.ChainID)
	if !ok {
		t.Fatal("registered chain not present in registry after RegisterChain")
	}
	if got.Bond != 10000 {
		t.Fatalf("registered bond = %d, want 10000", got.Bond)
	}
	if len(got.RotationSigs) == 0 {
		t.Fatal("registered chain has no rotation signature")
	}
	// The D14 escrow must have been consumed (hold no longer pending).
	h, herr := o.GetNRNToken().GetHold(reg.HoldID)
	if herr != nil {
		t.Fatalf("get hold: %v", herr)
	}
	if h.Status == ledger.BondStatusPending {
		t.Fatalf("D14 hold %q still pending after authority registration", reg.HoldID)
	}
}

// TestAuthorityBondRegistrationRejectsUnsigned proves the Oracle rejects an
// authority registration that carries no rotation signature — the CLI must always
// sign before posting.
func TestAuthorityBondRegistrationRejectsUnsigned(t *testing.T) {
	o, opKey := newBondOracle(t)
	key := opKey.PrivateKey
	reg := newAuthorityBondRegistration(t, o, key)
	reg.RotationSigs = nil // strip the signature the CLI would have attached
	if err := o.RegisterChain(&reg); err == nil {
		t.Fatal("expected real Oracle to reject an unsigned authority registration")
	}
}

// TestAuthorityBondRegistrationRejectsTamperedBody proves the Oracle rejects a
// registration whose body no longer matches the signed digest (e.g. a wrong bond).
// This guards the audit P1 finding: the CLI must sign the exact digest the Oracle
// recomputes from the received body.
func TestAuthorityBondRegistrationRejectsTamperedBody(t *testing.T) {
	o, opKey := newBondOracle(t)
	key := opKey.PrivateKey
	reg := newAuthorityBondRegistration(t, o, key)
	// Tamper with a body field after signing: the digest no longer matches.
	reg.Bond = 9999
	if err := o.RegisterChain(&reg); err == nil {
		t.Fatal("expected real Oracle to reject a registration whose body disagrees with the signature")
	}
}

// authorityRegWithHold builds a bonded authority/* registration whose bond owner is
// regKey's address and whose enrolment hold is owned by holdKey (holdAddr) with the
// given amount. When regKey == holdKey the hold matches the bond owner (valid); when
// they differ, or when holdAmount is too small, the registration must be rejected.
func authorityRegWithHold(t *testing.T, o *Oracle, regKey, holdKey *ecdsa.PrivateKey, holdAmount string) types.ChainRegistration {
	t.Helper()
	regAddr := types.OracleAddress(&regKey.PublicKey)
	holdAddr := types.OracleAddress(&holdKey.PublicKey)
	tok := o.GetNRNToken()
	opHold, err := types.AddressFromString(holdAddr)
	if err != nil {
		t.Fatalf("hold address: %v", err)
	}
	if _, err := tok.Mint(opHold, big.NewInt(2_000_000_000)); err != nil {
		t.Fatalf("mint: %v", err)
	}
	holdID := "bond:" + holdAddr
	expiry := time.Now().Add(time.Hour)
	hold := knirvsigning.BondHold{
		HoldID:         holdID,
		Operator:       holdAddr,
		Amount:         holdAmount,
		RootRegistryID: tok.RootRegistryID(),
		ExpiresAtUnix:  uint64(expiry.Unix()),
	}
	if err := tok.Hold(holdID, hold, signAuthorityHold(t, holdKey, tok.ChainID(), hold), time.Now()); err != nil {
		t.Fatalf("hold: %v", err)
	}
	cpk := crypto.CompressPubkey(&regKey.PublicKey)
	reg := types.ChainRegistration{
		ChainID:     "authority/" + regAddr,
		Authors:     []types.RegisteredAuthor{{Address: regAddr, PubKey: cpk, Weight: 1}},
		QuorumNumer: 1,
		QuorumDenom: 1,
		LastHeight:  0,
		Bond:        10000,
		BondOwner:   regAddr,
		ProofWindow: 256,
		HoldID:      holdID,
	}
	if err := types.SignRegistration(&reg, regKey); err != nil {
		t.Fatalf("sign: %v", err)
	}
	return reg
}

// TestAuthorityBondRegistrationOmittedHoldRejected (Oracle test): a bonded authority
// registration with hold_id omitted is rejected BEFORE any chain-bond lock or registry
// mutation — it can never bypass the D14 escrow.
func TestAuthorityBondRegistrationOmittedHoldRejected(t *testing.T) {
	o := newTestOracle(t)
	key, _ := crypto.GenerateKey()
	addr := types.OracleAddress(&key.PublicKey)
	cpk := crypto.CompressPubkey(&key.PublicKey)
	reg := types.ChainRegistration{
		ChainID:     "authority/" + addr,
		Authors:     []types.RegisteredAuthor{{Address: addr, PubKey: cpk, Weight: 1}},
		QuorumNumer: 1,
		QuorumDenom: 1,
		Bond:        10000,
		BondOwner:   addr,
		ProofWindow: 256,
	}
	if err := types.SignRegistration(&reg, key); err != nil {
		t.Fatal(err)
	}
	err := o.RegisterChain(&reg)
	if err == nil {
		t.Fatal("expected bonded authority registration without a hold to be rejected")
	}
	if _, ok := o.GetChainRegistration(reg.ChainID); ok {
		t.Fatal("chain was registered despite omitted hold id")
	}
}

// TestAuthorityBondRegistrationWrongOwnerHoldRejected: a hold owned by a different
// operator than the bond owner is rejected.
func TestAuthorityBondRegistrationWrongOwnerHoldRejected(t *testing.T) {
	o, opKey := newBondOracle(t)
	regKey, _ := crypto.GenerateKey()
	reg := authorityRegWithHold(t, o, regKey, opKey.PrivateKey, "10000")
	if err := o.RegisterChain(&reg); err == nil {
		t.Fatal("expected registration with a hold owned by the wrong operator to be rejected")
	}
	if _, ok := o.GetChainRegistration(reg.ChainID); ok {
		t.Fatal("chain was registered despite a mismatched hold owner")
	}
}

// TestAuthorityBondRegistrationWrongAmountHoldRejected: a hold whose amount does not
// cover the bond is rejected.
func TestAuthorityBondRegistrationWrongAmountHoldRejected(t *testing.T) {
	o, opKey := newBondOracle(t)
	key := opKey.PrivateKey
	reg := authorityRegWithHold(t, o, key, key, "1") // owner matches, but 1 NRN < 10000 bond
	if err := o.RegisterChain(&reg); err == nil {
		t.Fatal("expected registration with an underfunded hold to be rejected")
	}
}

// TestAuthorityBondRegistrationValidHoldConsumesOnce: a valid hold still registers
// the chain and is consumed exactly once (an idempotent retry does not re-send the
// D14 escrow).
func TestAuthorityBondRegistrationValidHoldConsumesOnce(t *testing.T) {
	o, opKey := newBondOracle(t)
	key := opKey.PrivateKey
	reg := newAuthorityBondRegistration(t, o, key)
	tok := o.GetNRNToken()
	opAddr, err := types.AddressFromString(types.OracleAddress(&key.PublicKey))
	if err != nil {
		t.Fatalf("address: %v", err)
	}

	if err := o.RegisterChain(&reg); err != nil {
		t.Fatalf("register: %v", err)
	}
	after := tok.GetBalance(opAddr)
	// Mint 2B, stake 1B, hold locks 10000. Registration RESERVES the hold (does not
	// release the escrow): the operator ends at 1B - 10000. The escrow moves to the
	// boot node address only via the root-owner-signed Release.
	if after.Cmp(big.NewInt(1_000_000_000-10_000)) != 0 {
		t.Fatalf("operator balance = %s, want 999990000 (stake 1B minus held 10000)", after)
	}
	// Idempotent retry must not double-send.
	if err := o.RegisterChain(&reg); err != nil {
		t.Fatalf("idempotent retry: %v", err)
	}
	retry := tok.GetBalance(opAddr)
	if retry.Cmp(after) != 0 {
		t.Fatalf("balance changed on retry: before %s after %s (double send?)", after, retry)
	}
	h, herr := tok.GetHold(reg.HoldID)
	if herr != nil {
		t.Fatalf("get hold: %v", herr)
	}
	if h.Status == ledger.BondStatusPending {
		t.Fatalf("D14 hold %q still pending after registration", reg.HoldID)
	}
}

// TestRouteRegisterAuthorityOmittedHoldRejected (route-level test): a signed
// authority registration with no hold id, delivered as a JSON body the way the HTTP
// route would decode it, is rejected at admission.
func TestRouteRegisterAuthorityOmittedHoldRejected(t *testing.T) {
	o := newTestOracle(t)
	key, _ := crypto.GenerateKey()
	addr := types.OracleAddress(&key.PublicKey)
	cpk := crypto.CompressPubkey(&key.PublicKey)
	reg := types.ChainRegistration{
		ChainID:     "authority/" + addr,
		Authors:     []types.RegisteredAuthor{{Address: addr, PubKey: cpk, Weight: 1}},
		QuorumNumer: 1,
		QuorumDenom: 1,
		Bond:        10000,
		BondOwner:   addr,
		ProofWindow: 256,
	}
	if err := types.SignRegistration(&reg, key); err != nil {
		t.Fatal(err)
	}
	// Simulate the HTTP route: decode the JSON body the client posted, then admit.
	body, err := json.Marshal(reg)
	if err != nil {
		t.Fatal(err)
	}
	var routed types.ChainRegistration
	if err := json.Unmarshal(body, &routed); err != nil {
		t.Fatal(err)
	}
	if err := o.RegisterChain(&routed); err == nil {
		t.Fatal("expected route to reject a bonded authority registration with no hold id")
	}
}

// TestUnbondedOrdinaryRegistrationNoHoldAllowed proves the explicit separation in
// isBondedAuthorityEnrollment: an unbonded ordinary (non-authority) registration has
// Bond == 0 and is a separate path that never requires a hold and never creates a
// bonded authority chain.
func TestUnbondedOrdinaryRegistrationNoHoldAllowed(t *testing.T) {
	o := newTestOracle(t)
	key, _ := crypto.GenerateKey()
	addr := types.OracleAddress(&key.PublicKey)
	cpk := crypto.CompressPubkey(&key.PublicKey)
	reg := types.ChainRegistration{
		ChainID:     "knirvchain-plain",
		Authors:     []types.RegisteredAuthor{{Address: addr, PubKey: cpk, Weight: 1}},
		QuorumNumer: 1,
		QuorumDenom: 1,
		Bond:        0,
		BondOwner:   addr,
		ProofWindow: 256,
	}
	if err := types.SignRegistration(&reg, key); err != nil {
		t.Fatal(err)
	}
	if err := o.RegisterChain(&reg); err != nil {
		t.Fatalf("unbonded ordinary registration must not require a hold: %v", err)
	}
	if _, ok := o.GetChainRegistration(reg.ChainID); !ok {
		t.Fatal("unbonded ordinary chain not registered")
	}
}

// TestAuthorityBondHoldCannotAuthorizeTwoChains is the round-4 TOCTOU regression:
// many concurrent authority-chain registrations that all reference the SAME D14
// enrolment hold (same owner, same hold id) must NOT all succeed. The hold is bound
// to the chain id that consumes it, so exactly one chain wins; every other
// registration is rejected and leaves no registry entry and no leftover chain-bond
// lock (the loser's LockChainBond is rolled back). Run under -race.
func TestAuthorityBondHoldCannotAuthorizeTwoChains(t *testing.T) {
	o, opKey := newBondOracle(t)
	key := opKey.PrivateKey
	addr := types.OracleAddress(&key.PublicKey)
	tok := o.GetNRNToken()
	opAddr, err := types.AddressFromString(addr)
	if err != nil {
		t.Fatalf("address: %v", err)
	}
	if _, err := tok.Mint(opAddr, big.NewInt(2_000_000_000)); err != nil {
		t.Fatalf("mint: %v", err)
	}
	if _, err := o.economicsEngine.GetStakingManager().Stake(opAddr, big.NewInt(1_000_000_000)); err != nil {
		t.Fatalf("stake: %v", err)
	}
	// ONE enrolment hold shared by every concurrent registration.
	holdID := "bond:shared"
	expiry := time.Now().Add(time.Hour)
	hold := knirvsigning.BondHold{
		HoldID:         holdID,
		Operator:       addr,
		Amount:         "10000",
		RootRegistryID: tok.RootRegistryID(),
		ExpiresAtUnix:  uint64(expiry.Unix()),
	}
	if err := tok.Hold(holdID, hold, signAuthorityHold(t, key, tok.ChainID(), hold), time.Now()); err != nil {
		t.Fatalf("hold: %v", err)
	}

	const N = 32
	var wg sync.WaitGroup
	var mu sync.Mutex
	var winners []string
	for i := 0; i < N; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			chainID := fmt.Sprintf("authority/%s-distinct-%d", addr, i)
			cpk := crypto.CompressPubkey(&key.PublicKey)
			reg := types.ChainRegistration{
				ChainID:     chainID,
				Authors:     []types.RegisteredAuthor{{Address: addr, PubKey: cpk, Weight: 1}},
				QuorumNumer: 1,
				QuorumDenom: 1,
				LastHeight:  0,
				Bond:        10000,
				BondOwner:   addr,
				ProofWindow: 256,
				HoldID:      holdID,
			}
			if err := types.SignRegistration(&reg, key); err != nil {
				mu.Lock()
				winners = append(winners, "sign-error:"+err.Error())
				mu.Unlock()
				return
			}
			if err := o.RegisterChain(&reg); err != nil {
				return
			}
			mu.Lock()
			winners = append(winners, chainID)
			mu.Unlock()
		}(i)
	}
	wg.Wait()

	if len(winners) != 1 {
		t.Fatalf("expected exactly one chain to claim the shared hold, got %d: %v", len(winners), winners)
	}
	winner := winners[0]
	// The hold is consumed and bound to the single winning chain.
	h, herr := tok.GetHold(holdID)
	if herr != nil {
		t.Fatalf("get hold: %v", herr)
	}
	if h.Status != ledger.BondStatusConsumed {
		t.Fatalf("hold status = %q, want consumed", h.Status)
	}
	if h.ConsumedChainID != winner {
		t.Fatalf("hold bound to %q, want winner %q", h.ConsumedChainID, winner)
	}
	// Losers neither registered nor left a bond lock behind.
	for i := 0; i < N; i++ {
		chainID := fmt.Sprintf("authority/%s-distinct-%d", addr, i)
		if chainID == winner {
			if _, ok := o.GetChainRegistration(chainID); !ok {
				t.Fatalf("winner %q not registered", chainID)
			}
			continue
		}
		if _, ok := o.GetChainRegistration(chainID); ok {
			t.Fatalf("loser %q was registered despite the shared hold being consumed", chainID)
		}
		if _, berr := o.economicsEngine.GetStakingManager().GetChainBond(chainID); berr == nil {
			t.Fatalf("loser %q left a leftover chain-bond lock", chainID)
		}
	}
}
