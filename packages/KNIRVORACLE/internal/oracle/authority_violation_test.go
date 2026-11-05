package oracle

import (
	"context"
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/hex"
	"math/big"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/guiperry/knirv-sdk-go/authority"
	"github.com/guiperry/knirv-sdk-go/signing"
	"github.com/knirvcorp/knirvoracle/internal/oracle/consensus"
	"github.com/knirvcorp/knirvoracle/internal/oracle/economics"
	"github.com/knirvcorp/knirvoracle/internal/oracle/registry"
	"github.com/knirvcorp/knirvoracle/internal/oracle/token"
	"github.com/knirvcorp/knirvoracle/internal/oracle/types"
	"go.uber.org/zap"
)

// The SDK runs its Rust core as a digest-pinned WASM module. A locally built core
// is not published, so seed a throwaway data directory from KNIRVSDK/dist, as the
// SDK's own tests and internal/authority in SERVER do. KNIRV_SDK_DATA_DIR wins.
var seedAuthorityCore sync.Once

func useAuthorityCore(t *testing.T) {
	t.Helper()
	seedAuthorityCore.Do(func() {
		if os.Getenv("KNIRV_SDK_DATA_DIR") != "" {
			return
		}
		matches, _ := filepath.Glob(filepath.Join("..", "..", "..", "KNIRVSDK", "dist", "*", "knirv_core.wasm"))
		if len(matches) == 0 {
			return
		}
		sort.Slice(matches, func(i, j int) bool {
			a, _ := os.Stat(matches[i])
			b, _ := os.Stat(matches[j])
			return a.ModTime().After(b.ModTime())
		})
		dir, err := os.MkdirTemp("", "oracle-authority-core-")
		if err != nil {
			return
		}
		version := filepath.Base(filepath.Dir(matches[0]))
		data, err := os.ReadFile(matches[0])
		if err != nil || os.MkdirAll(filepath.Join(dir, version), 0o700) != nil {
			return
		}
		if os.WriteFile(filepath.Join(dir, version, "knirv_core.wasm"), data, 0o600) == nil {
			os.Setenv("KNIRV_SDK_DATA_DIR", dir)
		}
	})
}

const (
	avNotBefore = int64(1_750_000_000)
	avNotAfter  = int64(1_750_086_400)
	avBits      = uint32(0x207fffff)
)

var avNow = time.Unix(1_750_000_100, 0)

type authorityHarness struct {
	o         *Oracle
	chainID   string
	operator  string
	longTerm  signing.RemoteSigner
	session   signing.RemoteSigner
	cert      authority.SessionCert
	bond      uint64
	chainKey  *ecdsa.PrivateKey
	chainBoot types.ChainRegistration
}

func bitsFor(tier authority.Tier) uint32 {
	if tier >= authority.TierT2 {
		return authority.TierTargetBits(tier)
	}
	return avBits
}

func avKey(n byte) []byte { k := make([]byte, 32); k[31] = n; return k }

// newAuthorityHarness registers `authority/<operator>` on an Oracle with a real
// economics engine, so a bond can really be burned.
func newAuthorityHarness(t *testing.T, bond uint64) *authorityHarness {
	t.Helper()
	useAuthorityCore(t)
	o := newPhase4Oracle(t)
	// The harness exercises authority records and violations, not the production
	// minimum-bond default; run it in observe mode so a zero bond is allowed.
	if err := o.SetAuthorityMinBond(0); err != nil {
		t.Fatal(err)
	}
	nrn, err := token.NewNRN("NRN", "NRN", big.NewInt(10_000_000_000), big.NewInt(20_000_000_000), strings.Repeat("02", 32))
	if err != nil {
		t.Fatal(err)
	}
	o.economicsEngine = economics.NewEconomicsEngine(nrn, zap.NewNop())
	owner := nrn.Owner()
	if _, err := o.economicsEngine.GetStakingManager().Stake(owner, big.NewInt(2_000_000_000)); err != nil {
		t.Fatal(err)
	}
	longTerm, err := signing.NewLocalSigner(avKey(1))
	if err != nil {
		t.Fatal(err)
	}
	session, err := signing.NewLocalSigner(avKey(2))
	if err != nil {
		t.Fatal(err)
	}
	key, err := ethcrypto.ToECDSA(avKey(1))
	if err != nil {
		t.Fatal(err)
	}
	h := &authorityHarness{o: o, operator: longTerm.Address(), longTerm: longTerm, session: session, bond: bond, chainKey: key}
	h.chainID = registry.AuthorityChainPrefix + h.operator
	cert := authority.SessionCert{SessionPub: session.PublicKey(), Operator: h.operator, Epoch: 7,
		NotBefore: avNotBefore, NotAfter: avNotAfter, Scope: h.operator, Entropy: "csprng"}
	h.cert, err = authority.SignSessionCert(context.Background(), cert, longTerm)
	if err != nil {
		t.Fatal(err)
	}
	reg := types.ChainRegistration{ChainID: h.chainID, QuorumNumer: 2, QuorumDenom: 3, ProofWindow: 1800,
		Authors: []types.RegisteredAuthor{{Address: h.operator, PubKey: ethcrypto.CompressPubkey(&key.PublicKey), Weight: 1}}}
	if bond > 0 {
		reg.Bond, reg.BondOwner = bond, owner.String()
	}
	if err := types.SignRegistration(&reg, key); err != nil {
		t.Fatal(err)
	}
	if err := o.RegisterChain(&reg); err != nil {
		t.Fatalf("register authority chain: %v", err)
	}
	return h
}

func (h *authorityHarness) record(t *testing.T, tier authority.Tier, scope, action string, prev [32]byte) *authority.Record {
	t.Helper()
	r := &authority.Record{Tier: tier, Domain: "evidence.commit", Scope: scope, Operator: h.operator, Epoch: 7,
		CanonicalActionHash: sha256.Sum256([]byte(action)), PrevRecordHash: prev,
		TargetBits: bitsFor(tier), WitnessKind: authority.WitnessSoftware, SessionCert: h.cert}
	if err := r.Seal(context.Background(), h.session, uint32(avNotBefore)); err != nil {
		t.Fatal(err)
	}
	return r
}

func TestAuthorityRecordInScopeVerifiesWithoutViolation(t *testing.T) {
	h := newAuthorityHarness(t, 1_000_000_000)
	rec := h.record(t, authority.TierT1, h.operator, "a", sha256.Sum256([]byte("genesis")))
	f, err := h.o.ReportAuthorityRecord(h.chainID, rec, nil, avNow)
	if err != nil || !f.Verified || f.Violation != nil || f.Err != nil {
		t.Fatalf("finding=%+v err=%v", f, err)
	}
	if len(h.o.consensusEngine.PendingEvidence()) != 0 || len(h.o.AuthorityViolations(h.chainID)) != 0 {
		t.Fatal("a valid record produced evidence")
	}
}

// Done-when: a scope-violating record from a boot node yields slashing evidence.
func TestScopeViolatingBootRecordYieldsSlashingEvidence(t *testing.T) {
	h := newAuthorityHarness(t, 1_000_000_000)
	foreign := "knirv1someoneelse"
	rec := h.record(t, authority.TierT1, foreign, "x", sha256.Sum256([]byte("genesis")))
	f, err := h.o.ReportAuthorityRecord(h.chainID, rec, nil, avNow)
	if err != nil {
		t.Fatal(err)
	}
	if f.Verified || f.Violation == nil || f.Violation.Kind != registry.ViolationScope {
		t.Fatalf("want a scope violation, got %+v", f)
	}
	if f.Violation.Amount != 100_000_000 {
		t.Fatalf("slashed %d, want 10%% of the bond", f.Violation.Amount)
	}
	reg, _ := h.o.GetChainRegistration(h.chainID)
	if reg.BondRemaining != 900_000_000 || reg.BondSlashed != 100_000_000 {
		t.Fatalf("bond view remaining=%d slashed=%d", reg.BondRemaining, reg.BondSlashed)
	}
	ev := h.o.consensusEngine.PendingEvidence()
	if len(ev) != 1 {
		t.Fatalf("pending evidence = %d", len(ev))
	}
	e, ok := ev[0].(consensus.AuthorityViolationEvidence)
	if !ok || e.ValidateBasic() != nil || e.Kind != "scope_violation" || e.Operator != h.operator || e.Amount != 100_000_000 {
		t.Fatalf("bad evidence %#v", ev[0])
	}
	if burned := h.o.economicsEngine.GetBurnTracker().GetBurnsByReason(economics.BurnReasonSlashing); burned.Uint64() != 100_000_000 {
		t.Fatalf("burned %s", burned)
	}
	// Reporting the same record again must not slash twice.
	if _, err := h.o.ReportAuthorityRecord(h.chainID, rec, nil, avNow); err != nil {
		t.Fatal(err)
	}
	if got := len(h.o.AuthorityViolations(h.chainID)); got != 1 {
		t.Fatalf("violations = %d after a repeat report", got)
	}
	if reg2, _ := h.o.GetChainRegistration(h.chainID); reg2.BondSlashed != 100_000_000 {
		t.Fatal("repeat report slashed again")
	}
}

func TestBootRecordAtT2IsAViolation(t *testing.T) {
	h := newAuthorityHarness(t, 1_000_000_000)
	rec := h.record(t, authority.TierT2, h.operator, "revoke", sha256.Sum256([]byte("genesis")))
	f, err := h.o.ReportAuthorityRecord(h.chainID, rec, nil, avNow)
	if err != nil || f.Violation == nil || f.Violation.Kind != registry.ViolationScope {
		t.Fatalf("finding=%+v err=%v", f, err)
	}
}

func TestEquivocationYieldsEvidence(t *testing.T) {
	h := newAuthorityHarness(t, 1_000_000_000)
	prev := sha256.Sum256([]byte("genesis"))
	a := h.record(t, authority.TierT1, h.operator, "a", prev)
	b := h.record(t, authority.TierT1, h.operator, "b", prev)
	if f, err := h.o.ReportAuthorityRecord(h.chainID, a, nil, avNow); err != nil || f.Violation != nil {
		t.Fatalf("first record: %+v %v", f, err)
	}
	// The same record again is not equivocation.
	if f, _ := h.o.ReportAuthorityRecord(h.chainID, a, nil, avNow); f.Violation != nil {
		t.Fatal("re-reporting one record counted as a fork")
	}
	f, err := h.o.ReportAuthorityRecord(h.chainID, b, nil, avNow)
	if err != nil || f.Violation == nil || f.Violation.Kind != registry.ViolationEquivocation {
		t.Fatalf("finding=%+v err=%v", f, err)
	}
	if f.Violation.OtherHash != hex.EncodeToString(a.Hash[:]) {
		t.Fatalf("other hash %s", f.Violation.OtherHash)
	}
	if len(h.o.consensusEngine.PendingEvidence()) != 1 {
		t.Fatal("no evidence for equivocation")
	}
}

// A record that does not verify for a reason a third party could cause must not
// cost the operator its bond.
func TestForgedOrExpiredRecordIsNotSlashed(t *testing.T) {
	h := newAuthorityHarness(t, 1_000_000_000)
	prev := sha256.Sum256([]byte("genesis"))

	forged := h.record(t, authority.TierT1, "knirv1someoneelse", "x", prev)
	forged.Signature = make([]byte, 64) // not the session key's signature
	f, err := h.o.ReportAuthorityRecord(h.chainID, forged, nil, avNow)
	if err != nil || f.Violation != nil || f.Err == nil {
		t.Fatalf("forged: %+v %v", f, err)
	}
	ok := h.record(t, authority.TierT1, h.operator, "y", prev)
	f, err = h.o.ReportAuthorityRecord(h.chainID, ok, nil, time.Unix(avNotAfter+10, 0))
	if err != nil || f.Violation != nil || f.Err == nil {
		t.Fatalf("expired: %+v %v", f, err)
	}
	if reg, _ := h.o.GetChainRegistration(h.chainID); reg.BondSlashed != 0 {
		t.Fatal("bond slashed without attributable misuse")
	}
	if len(h.o.consensusEngine.PendingEvidence()) != 0 {
		t.Fatal("evidence queued without attributable misuse")
	}
}

func TestRecordForOtherOperatorOrNonAuthorityChainIsRefused(t *testing.T) {
	h := newAuthorityHarness(t, 0)
	rec := h.record(t, authority.TierT1, h.operator, "a", [32]byte{})
	if _, err := h.o.ReportAuthorityRecord("knirvchain", rec, nil, avNow); err == nil {
		t.Fatal("non-authority chain accepted")
	}
	if _, err := h.o.ReportAuthorityRecord(registry.AuthorityChainPrefix+"knirv1other", rec, nil, avNow); err == nil {
		t.Fatal("record for another operator accepted")
	}
}

func TestViolationHoldsApprovalOfTheChainsCheckpoints(t *testing.T) {
	h := newAuthorityHarness(t, 1_000_000_000)
	for _, v := range []string{"v1", "v2", "v3"} {
		registerPhase4Verifier(t, h.o, v)
	}
	rootBytes, _ := hex.DecodeString(testBatchRoot)
	var root [32]byte
	copy(root[:], rootBytes)
	cp := &types.Checkpoint{SchemaVersion: "knirv.checkpoint.v1", ChainID: h.chainID, StartHeight: 1, EndHeight: 10, Root: root, Proposer: h.operator}
	if err := types.SignCheckpoint(cp, h.chainKey); err != nil {
		t.Fatal(err)
	}
	rec, err := h.o.SubmitCheckpoint(cp)
	if err != nil {
		t.Fatalf("SubmitCheckpoint: %v", err)
	}
	bad := h.record(t, authority.TierT1, "knirv1someoneelse", "x", sha256.Sum256([]byte("genesis")))
	if f, err := h.o.ReportAuthorityRecord(h.chainID, bad, nil, avNow); err != nil || f.Violation == nil {
		t.Fatalf("setup violation: %+v %v", f, err)
	}
	_, err = h.o.SubmitFinality(&types.FinalityRecord{CheckpointLeaf: rec.MMRPosition, ProofSystem: "hashchain-v0",
		Attestations: []types.VerifierAttestation{phase4Attestation(t, h.o, "v1", rec), phase4Attestation(t, h.o, "v2", rec)}})
	if err == nil || !strings.Contains(err.Error(), "authority violation") {
		t.Fatalf("approval should be held, got %v", err)
	}
}

func TestOperatorDefaultsAreConservative(t *testing.T) {
	h := newAuthorityHarness(t, 0)
	rec := h.record(t, authority.TierT1, h.operator, "a", [32]byte{1})
	if _, err := h.o.ReportAuthorityRecord(h.chainID, rec, nil, avNow); err != nil {
		t.Fatal(err)
	}
	e, ok := h.o.checkpoint.registry.Operator(h.operator)
	if !ok || e.Kind != "boot" || len(e.Scopes) != 1 || e.Scopes[0] != h.operator {
		t.Fatalf("entry %+v", e)
	}
	// A key that does not hash to the operator is never bound.
	other, _ := signing.NewLocalSigner(avKey(9))
	if _, err := h.o.resolveOperatorLocked("knirv1x", &types.ChainRegistration{}, other.PublicKey()); err == nil {
		t.Fatal("mismatched key was bound")
	}
}

func TestAuthorityChainBondRequirement(t *testing.T) {
	r := registry.New()
	if err := r.SetMinAuthorityBond(500); err != nil {
		t.Fatal(err)
	}
	mk := func(id string, bond uint64) *types.ChainRegistration {
		return &types.ChainRegistration{ChainID: id, QuorumNumer: 1, QuorumDenom: 1, Bond: bond,
			Authors: []types.RegisteredAuthor{{Address: "a", Weight: 1}}}
	}
	if err := r.Register(mk("authority/knirv1abc", 499)); err == nil {
		t.Fatal("under-bonded authority chain registered")
	}
	if err := r.Register(mk("authority/knirv1abc", 500)); err != nil {
		t.Fatalf("bonded: %v", err)
	}
	// Rotation (already known) and ordinary chains are not affected.
	if err := r.Register(mk("authority/knirv1abc", 0)); err != nil {
		t.Fatalf("rotation: %v", err)
	}
	if err := r.Register(mk("knirvchain", 0)); err != nil {
		t.Fatalf("ordinary chain: %v", err)
	}
	// Default requirement is DefaultAuthorityMinBond (10,000 NRN).
	if got := registry.New().MinAuthorityBond(); got != registry.DefaultAuthorityMinBond {
		t.Fatalf("default min bond = %d, want %d", got, registry.DefaultAuthorityMinBond)
	}
	if err := registry.New().Register(mk("authority/knirv1new", 0)); err == nil {
		t.Fatal("default must require a bond")
	}
	if err := registry.New().Register(mk("authority/knirv1new", registry.DefaultAuthorityMinBond)); err != nil {
		t.Fatalf("bonded at default: %v", err)
	}
}

func TestAuthorityStatePersists(t *testing.T) {
	dir := t.TempDir()
	r, err := registry.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	pub := ethcrypto.CompressPubkey(&mustKey(t).PublicKey)
	if err := r.SetOperator(registry.OperatorEntry{Address: "op", Kind: "root", LongTermPub: pub, Scopes: []string{"op"}}); err != nil {
		t.Fatal(err)
	}
	_, _, _ = r.ObserveAuthorityRecord("op", "p", "h1")
	_, _ = r.AddViolation(registry.Violation{Key: "k", Kind: registry.ViolationScope, Operator: "op", ChainID: "authority/op", RecordHash: "h1", Reason: "r"})
	_ = r.SetMinAuthorityBond(7)

	r2, err := registry.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if e, ok := r2.Operator("op"); !ok || e.Kind != "root" {
		t.Fatal("operator lost")
	}
	if other, conflict, _ := r2.ObserveAuthorityRecord("op", "p", "h2"); !conflict || other != "h1" {
		t.Fatal("seen record lost")
	}
	if len(r2.Violations("authority/op")) != 1 || r2.MinAuthorityBond() != 7 {
		t.Fatal("violations or min bond lost")
	}
}

func mustKey(t *testing.T) *ecdsa.PrivateKey {
	t.Helper()
	k, err := ethcrypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	return k
}
