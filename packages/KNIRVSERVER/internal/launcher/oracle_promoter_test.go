package launcher

import (
	"context"
	"sync"
	"testing"

	knirvoracle "github.com/KNIRV/KNIRV_NETWORK/KNIRVORACLE"
	"go.uber.org/zap"
)

// TestLauncherOracleManagerInjectsPromoter proves the production constructor
// (NewOracleManager) injects launcher.OraclePromoter into the Manager, so the
// failover CONFIRMED/RECLAIMED transition reaches PromoteChildToActive. This is the
// fix for the nil-promoter regression: NewManager alone leaves the promoter nil.
func TestLauncherOracleManagerInjectsPromoter(t *testing.T) {
	cfg := &knirvoracle.ManagerConfig{BaoSocketPath: "/tmp/bao.sock"}
	m := NewOracleManager(cfg, zap.NewNop(), OraclePromoter{})

	if m.Promoter() == nil {
		t.Fatalf("NewOracleManager must inject a non-nil AppRolePromoter (OraclePromoter)")
	}
}

// TestLauncherOraclePromoterImplementsContract proves the production promoter
// satisfies knirvoracle.AppRolePromoter (compile-time contract plus a swap call that
// delegates to the launcher's hooks).
func TestLauncherOraclePromoterImplementsContract(t *testing.T) {
	var p knirvoracle.AppRolePromoter = OraclePromoter{}
	if p == nil {
		t.Fatalf("OraclePromoter must be a non-nil AppRolePromoter")
	}
	// A swap must delegate to the launcher hooks without panicking on a missing
	// socket; the error returned is a transport error, not a nil-deref.
	if err := p.PromoteChildToActive(context.Background(), "/tmp/bao.sock", "oracle-standby", "tok"); err == nil {
		t.Fatalf("expected a transport error talking to a non-existent bao socket")
	}
}

// fakeAppRolePromoter is a no-IO AppRolePromoter recording the CONFIRMED/RECLAIMED
// swap calls, so the integration test can assert the production NewOracleManager
// path actually invokes the promoter.
type fakeAppRolePromoter struct {
	mu      sync.Mutex
	promote int
	swap    int
	lastRole string
}

func (f *fakeAppRolePromoter) PromoteChildToActive(ctx context.Context, socket, roleID, oldToken string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.promote++
	f.lastRole = roleID
	return nil
}

func (f *fakeAppRolePromoter) SwapChildToActive(ctx context.Context, socket, roleID, oldToken string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.swap++
	f.lastRole = roleID
	return nil
}

// TestNewOracleManagerDrivesConfirmedReclaimed drives the real 8.6 production
// constructor (NewOracleManager) through CONFIRMED (standby->active) and RECLAIMED
// (active->standby) transitions, passing the fake vault promoter INTO the
// constructor (never replacing the activator after construction), and asserts the
// injected promoter is invoked and the Oracle is restarted in the correct mode.
func TestNewOracleManagerDrivesConfirmedReclaimed(t *testing.T) {
	fp := &fakeAppRolePromoter{}
	m := NewOracleManager(&knirvoracle.ManagerConfig{BaoSocketPath: "/tmp/bao.sock"}, zap.NewNop(), fp)

	var gotMode string
	m.SetRestartFunc(func(ctx context.Context, mode string) error { gotMode = mode; return nil })

	if err := m.Promote(context.Background(), knirvoracle.PromotionTransition{RoleID: "oracle-standby", OldToken: "tok-standby"}); err != nil {
		t.Fatalf("CONFIRMED promote: %v", err)
	}
	if fp.promote != 1 {
		t.Fatalf("promoter PromoteChildToActive not invoked on CONFIRMED; promote=%d", fp.promote)
	}
	if gotMode != knirvoracle.ModeActive {
		t.Fatalf("restart mode after CONFIRMED = %q, want active", gotMode)
	}

	if err := m.Demote(context.Background(), knirvoracle.PromotionTransition{RoleID: "oracle-active", OldToken: "tok-active"}); err != nil {
		t.Fatalf("RECLAIMED demote: %v", err)
	}
	if fp.swap != 1 {
		t.Fatalf("promoter SwapChildToActive not invoked on RECLAIMED; swap=%d", fp.swap)
	}
	if gotMode != knirvoracle.ModeStandby {
		t.Fatalf("restart mode after RECLAIMED = %q, want standby", gotMode)
	}
}

// TestNewOracleManagerRejectsNilPromoterRegression is the negative guard: the
// production path passes OraclePromoter{} (a real promoter), and Promote must work;
// if the production path ever passes a nil promoter, Promote must fail loudly
// (failover would otherwise silently no-op instead of swapping the AppRole).
func TestNewOracleManagerRejectsNilPromoterRegression(t *testing.T) {
	// Production default: a real promoter is injected.
	m := NewOracleManager(&knirvoracle.ManagerConfig{BaoSocketPath: "/tmp/bao.sock"}, zap.NewNop(), OraclePromoter{})
	if m.Promoter() == nil {
		t.Fatalf("production NewOracleManager must inject a non-nil AppRolePromoter")
	}

	// A nil promoter (what the production path must never pass) must be detectable
	// and make Promote fail rather than silently changing the Oracle's identity.
	nilM := NewOracleManager(&knirvoracle.ManagerConfig{BaoSocketPath: "/tmp/bao.sock"}, zap.NewNop(), nil)
	if nilM.Promoter() != nil {
		t.Fatalf("expected nil promoter when nil is passed to the constructor")
	}
	nilM.SetRestartFunc(func(ctx context.Context, mode string) error { return nil })
	if err := nilM.Promote(context.Background(), knirvoracle.PromotionTransition{RoleID: "oracle-standby", OldToken: "tok"}); err == nil {
		t.Fatalf("expected Promote to fail with a nil promoter")
	}
}

// TestStartupConstructsOracleFailoverManagerWithPromoter is the production-call-site
// guard. It follows startOpenBao (the real backend construction path that owns the
// Oracle failover service) with an injected fake bao manager, then asserts
// app.OracleFailoverManager() has a non-nil promoter that is the real
// launcher.OraclePromoter. This protects bao_start.go:72: mutating the constructor
// argument there from OraclePromoter{} to nil makes this test FAIL, whereas the
// constructor-level TestNewOracleManagerRejectsNilPromoterRegression keeps passing
// (it builds with OraclePromoter{} directly) and therefore does not protect the
// production call site.
func TestStartupConstructsOracleFailoverManagerWithPromoter(t *testing.T) {
	app := &ServerApp{
		config: &Config{Secrets: SecretsConfig{OpenBao: OpenBaoConfig{Enabled: true}}},
		bao:    &fakeBaoManager{},
	}
	if err := app.startOpenBao(context.Background()); err != nil {
		t.Fatalf("startOpenBao returned error: %v", err)
	}

	m := app.OracleFailoverManager()
	if m == nil {
		t.Fatalf("startOpenBao must construct an OracleFailoverManager")
	}
	p := m.Promoter()
	if p == nil {
		t.Fatalf("startOpenBao must construct OracleFailoverManager with a non-nil promoter; a nil promoter would make failover silently no-op")
	}
	if _, ok := p.(OraclePromoter); !ok {
		t.Fatalf("startOpenBao must inject the real launcher.OraclePromoter; got %T", p)
	}
}
