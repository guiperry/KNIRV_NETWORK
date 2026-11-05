package authority

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"knirv-server/pkg/embedded/validationchain/checkpoint"
)

// fakeOracle records what KNIRVORACLE would receive.
type fakeOracle struct {
	mu          sync.Mutex
	registered  []checkpoint.ChainRegistration
	checkpoints []checkpoint.Checkpoint
	regStatus   int
	regMessage  string
	cpStatus    int
}

func (o *fakeOracle) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/oracle/v3/registry/register", func(w http.ResponseWriter, r *http.Request) {
		var reg checkpoint.ChainRegistration
		if err := json.NewDecoder(r.Body).Decode(&reg); err != nil {
			http.Error(w, "bad", http.StatusBadRequest)
			return
		}
		o.mu.Lock()
		defer o.mu.Unlock()
		o.registered = append(o.registered, reg)
		if o.regStatus != 0 {
			w.WriteHeader(o.regStatus)
			json.NewEncoder(w).Encode(map[string]string{"error": o.regMessage})
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "registered"})
	})
	mux.HandleFunc("/oracle/v3/checkpoints", func(w http.ResponseWriter, r *http.Request) {
		var cp checkpoint.Checkpoint
		if err := json.NewDecoder(r.Body).Decode(&cp); err != nil {
			http.Error(w, "bad", http.StatusBadRequest)
			return
		}
		o.mu.Lock()
		defer o.mu.Unlock()
		if o.cpStatus != 0 {
			w.WriteHeader(o.cpStatus)
			json.NewEncoder(w).Encode(map[string]string{"error": "oracle unavailable"})
			return
		}
		o.checkpoints = append(o.checkpoints, cp)
		json.NewEncoder(w).Encode(map[string]string{"status": "accepted"})
	})
	return mux
}

func (o *fakeOracle) snapshot() ([]checkpoint.ChainRegistration, []checkpoint.Checkpoint) {
	o.mu.Lock()
	defer o.mu.Unlock()
	return append([]checkpoint.ChainRegistration(nil), o.registered...), append([]checkpoint.Checkpoint(nil), o.checkpoints...)
}

// oracleHarness wires a real checkpoint.Poster to the fake Oracle. Every Poster
// fallback URL is pointed at the fake too, so a failing test cannot reach the
// real network.
func oracleHarness(t *testing.T) (*fakeOracle, *checkpoint.Poster) {
	t.Helper()
	o := &fakeOracle{}
	srv := httptest.NewServer(o.handler())
	t.Cleanup(srv.Close)
	t.Setenv("KNIRV_ORACLE_FAILOVER_URL", srv.URL)
	return o, checkpoint.NewPoster(srv.URL, srv.URL)
}

func newAnchor(t *testing.T, h *harness, poster OraclePoster, bond uint64, owner string) *Anchor {
	t.Helper()
	a, err := NewAnchor(AnchorConfig{Poster: poster, Signer: h.signer, Store: h.store, Bond: bond, BondOwner: owner, Logf: h.logs.Logf})
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func TestRegistrationPayloadShape(t *testing.T) {
	h := newHarness(t, KindRoot, 1)
	oracle, poster := oracleHarness(t)
	a := newAnchor(t, h, poster, 0, "")
	if !strings.Contains(h.logs.String(), "bond 0") {
		t.Fatalf("a zero bond must warn: %q", h.logs.String())
	}
	if err := a.EnsureRegistered(context.Background()); err != nil {
		t.Fatal(err)
	}
	regs, _ := oracle.snapshot()
	if len(regs) != 1 {
		t.Fatalf("%d registrations posted", len(regs))
	}
	reg, own := regs[0], h.signer.Address()
	if reg.ChainID != "authority/"+own {
		t.Errorf("chain id %q", reg.ChainID)
	}
	if len(reg.Authors) != 1 || reg.Authors[0].Address != own || reg.Authors[0].Weight != 1 || hex.EncodeToString(reg.Authors[0].PubKey) != hex.EncodeToString(h.signer.PublicKey()) {
		t.Errorf("authors %+v", reg.Authors)
	}
	if reg.QuorumNumer != 1 || reg.QuorumDenom != 1 || reg.ProofWindow != 0 {
		t.Errorf("quorum %d/%d window %d", reg.QuorumNumer, reg.QuorumDenom, reg.ProofWindow)
	}
	if reg.Bond != 0 || reg.BondOwner != "" || reg.LastHeight != 0 {
		t.Errorf("bond %d owner %q last height %d", reg.Bond, reg.BondOwner, reg.LastHeight)
	}
	if len(reg.RotationSigs) != 1 || reg.RotationSigs[0].Address != own || len(reg.RotationSigs[0].Signature) != 64 || len(reg.RotationSigs[0].Envelope) == 0 {
		t.Errorf("registration signature %+v", reg.RotationSigs)
	}
	// Registering again is a no-op: the state is remembered.
	if err := a.EnsureRegistered(context.Background()); err != nil {
		t.Fatal(err)
	}
	if regs, _ = oracle.snapshot(); len(regs) != 1 {
		t.Fatal("registered twice")
	}
}

func TestRegistrationCarriesOperatorSetBond(t *testing.T) {
	h := newHarness(t, KindRoot, 1)
	oracle, poster := oracleHarness(t)
	a := newAnchor(t, h, poster, 500, "knirv1bondowner")
	if strings.Contains(h.logs.String(), "bond 0") {
		t.Fatal("no zero-bond warning expected when a bond is set")
	}
	if err := a.EnsureRegistered(context.Background()); err != nil {
		t.Fatal(err)
	}
	if regs, _ := oracle.snapshot(); regs[0].Bond != 500 || regs[0].BondOwner != "knirv1bondowner" {
		t.Fatalf("%+v", regs[0])
	}
}

func TestCheckpointsCommitTheChainHeadAndStayContinuous(t *testing.T) {
	h := newHarness(t, KindRoot, 1)
	oracle, poster := oracleHarness(t)
	a := newAnchor(t, h, poster, 0, "")
	ctx := context.Background()

	if err := a.Tick(ctx); err != nil { // no records yet: registers, posts nothing
		t.Fatal(err)
	}
	if _, cps := oracle.snapshot(); len(cps) != 0 {
		t.Fatal("a checkpoint was posted for an empty chain")
	}
	mustAuthorize(t, h, act(1, "evidence.commit", "", `{"n":1}`))
	r2 := mustAuthorize(t, h, act(1, "evidence.commit", "", `{"n":2}`))
	if err := a.Tick(ctx); err != nil {
		t.Fatal(err)
	}
	_, cps := oracle.snapshot()
	if len(cps) != 1 {
		t.Fatalf("%d checkpoints", len(cps))
	}
	cp := cps[0]
	if cp.ChainID != "authority/"+h.signer.Address() || cp.StartHeight != 1 || cp.EndHeight != 2 || cp.Proposer != h.signer.Address() {
		t.Fatalf("checkpoint %+v", cp)
	}
	if cp.Root != r2.Record.Hash {
		t.Fatalf("root %x is not the chain head %x", cp.Root, r2.Record.Hash)
	}
	if cp.PrevCheckHash != ([32]byte{}) || len(cp.Signatures) != 1 || cp.Signatures[0].Address != h.signer.Address() {
		t.Fatalf("first checkpoint prev/signatures: %+v", cp)
	}
	first := cp.Digest()

	if err := a.Tick(ctx); err != nil { // no growth: nothing to post
		t.Fatal(err)
	}
	if _, cps = oracle.snapshot(); len(cps) != 1 {
		t.Fatal("posted without growth")
	}
	r3 := mustAuthorize(t, h, act(1, "evidence.commit", "", `{"n":3}`))
	if err := a.Tick(ctx); err != nil {
		t.Fatal(err)
	}
	_, cps = oracle.snapshot()
	if len(cps) != 2 || cps[1].StartHeight != 3 || cps[1].EndHeight != 3 || cps[1].PrevCheckHash != first || cps[1].Root != r3.Record.Hash {
		t.Fatalf("second checkpoint %+v (want start 3, prev = first digest)", cps[len(cps)-1])
	}
	if st := a.Status(); !st.Registered || st.LastEnd != 3 || st.LastError != "" {
		t.Fatalf("anchor status %+v", st)
	}
}

func TestAnchorContinuesAcrossRestart(t *testing.T) {
	dir := t.TempDir()
	h1 := buildHarness(t, dir, KindRoot, 1)
	oracle, poster := oracleHarness(t)
	a1 := newAnchor(t, h1, poster, 0, "")
	mustAuthorize(t, h1, act(1, "evidence.commit", "", `{"n":1}`))
	if err := a1.Tick(context.Background()); err != nil {
		t.Fatal(err)
	}
	_, cps := oracle.snapshot()
	first := cps[0].Digest()
	h1.store.Close()

	h2 := buildHarness(t, dir, KindRoot, 1)
	a2 := newAnchor(t, h2, poster, 0, "")
	if st := a2.Status(); !st.Registered || st.LastEnd != 1 {
		t.Fatalf("anchor state was not restored: %+v", st)
	}
	mustAuthorize(t, h2, act(1, "evidence.commit", "", `{"n":2}`))
	if err := a2.Tick(context.Background()); err != nil {
		t.Fatal(err)
	}
	regs, cps := oracle.snapshot()
	if len(regs) != 1 {
		t.Fatal("re-registered after restart")
	}
	if len(cps) != 2 || cps[1].StartHeight != 2 || cps[1].PrevCheckHash != first {
		t.Fatalf("checkpoint after restart %+v", cps[len(cps)-1])
	}
}

type flakyPoster struct {
	regErr, cpErr, rotErr error
	cps                   []*checkpoint.Checkpoint
}

func (f *flakyPoster) RegisterChain(context.Context, *checkpoint.ChainRegistration) (map[string]any, error) {
	return nil, f.regErr
}
func (f *flakyPoster) PostCheckpoint(_ context.Context, cp *checkpoint.Checkpoint) (map[string]any, error) {
	if f.cpErr != nil {
		return nil, f.cpErr
	}
	f.cps = append(f.cps, cp)
	return nil, nil
}

func TestAnchorFailuresAreNonFatalAndRetriedWithTheSameRange(t *testing.T) {
	h := newHarness(t, KindRoot, 1)
	p := &flakyPoster{cpErr: errors.New("oracle unreachable")}
	a := newAnchor(t, h, p, 0, "")
	mustAuthorize(t, h, act(1, "evidence.commit", "", `{"n":1}`))

	if err := a.Tick(context.Background()); err == nil {
		t.Fatal("expected the post failure to surface")
	}
	if st := a.Status(); st.LastEnd != 0 || !strings.Contains(st.LastError, "unreachable") {
		t.Fatalf("a failed post must not advance the state: %+v", st)
	}
	// The record store is unaffected: issuing keeps working while anchoring is down.
	mustAuthorize(t, h, act(1, "evidence.commit", "", `{"n":2}`))

	p.cpErr = nil
	if err := a.Tick(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(p.cps) != 1 || p.cps[0].StartHeight != 1 || p.cps[0].EndHeight != 2 {
		t.Fatalf("retry should cover the whole unposted range: %+v", p.cps)
	}
	if st := a.Status(); st.LastError != "" || st.LastEnd != 2 {
		t.Fatalf("status after recovery: %+v", st)
	}
}

func TestAlreadyRegisteredIsSuccessAndOtherRegistrationErrorsAreNot(t *testing.T) {
	h := newHarness(t, KindRoot, 1)
	a := newAnchor(t, h, &flakyPoster{regErr: errors.New("POST /x rejected (409): chain already registered")}, 0, "")
	if err := a.EnsureRegistered(context.Background()); err != nil {
		t.Fatalf("already registered must count as done: %v", err)
	}
	b := newAnchor(t, newHarness(t, KindRoot, 3), &flakyPoster{regErr: errors.New("oracle down")}, 0, "")
	if err := b.EnsureRegistered(context.Background()); err == nil {
		t.Fatal("a real registration failure must surface")
	}
	if st := b.Status(); st.Registered {
		t.Fatal("registration state set after a failure")
	}
}

func TestNewAnchorRequiresItsParts(t *testing.T) {
	if _, err := NewAnchor(AnchorConfig{}); err == nil {
		t.Fatal("an empty config was accepted")
	}
}
