package knirvoracle

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"go.uber.org/zap"
)

// fakePromoter records the AppRole swap calls made by the 8.6 promotion path.
type fakePromoter struct {
	mu       sync.Mutex
	promote  int
	swap     int
	lastRole string
	lastTok  string
	err      error
}

func (f *fakePromoter) PromoteChildToActive(ctx context.Context, socket, roleID, oldToken string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.promote++
	f.lastRole = roleID
	f.lastTok = oldToken
	return f.err
}

func (f *fakePromoter) SwapChildToActive(ctx context.Context, socket, roleID, oldToken string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.swap++
	f.lastRole = roleID
	f.lastTok = oldToken
	return f.err
}

func newTestManager(t *testing.T) *Manager {
	t.Helper()
	return NewManager(nil, zap.NewNop())
}

// TestOracleActivatorActivate verifies the 8.6 promotion swap calls
// launcher.PromoteChildToActive exactly once with the role + old token, and
// reports the active mode.
func TestOracleActivatorActivate(t *testing.T) {
	fp := &fakePromoter{}
	a := NewOracleActivator(fp, "/tmp/bao.sock", zap.NewNop())
	mode, err := a.Activate(context.Background(), PromotionTransition{RoleID: "oracle-active", OldToken: "old-standby-token"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mode != ModeActive {
		t.Fatalf("mode = %q, want active", mode)
	}
	if fp.promote != 1 || fp.swap != 0 {
		t.Fatalf("promote=%d swap=%d, want promote=1 swap=0", fp.promote, fp.swap)
	}
	if fp.lastRole != "oracle-active" || fp.lastTok != "old-standby-token" {
		t.Fatalf("promote called with role=%q token=%q", fp.lastRole, fp.lastTok)
	}
}

// TestOracleActivatorDeactivate verifies demotion swaps to standby.
func TestOracleActivatorDeactivate(t *testing.T) {
	fp := &fakePromoter{}
	a := NewOracleActivator(fp, "/tmp/bao.sock", zap.NewNop())
	mode, err := a.Deactivate(context.Background(), PromotionTransition{RoleID: "oracle-standby", OldToken: "old-active-token"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mode != ModeStandby {
		t.Fatalf("mode = %q, want standby", mode)
	}
	if fp.swap != 1 || fp.promote != 0 {
		t.Fatalf("swap=%d promote=%d, want swap=1 promote=0", fp.swap, fp.promote)
	}
}

// TestOracleActivatorPromoteError verifies a swap failure is surfaced to the
// caller (the failover service) and leaves the phase unchanged.
func TestOracleActivatorPromoteError(t *testing.T) {
	fp := &fakePromoter{err: context.DeadlineExceeded}
	a := NewOracleActivator(fp, "/tmp/bao.sock", zap.NewNop())
	if _, err := a.Activate(context.Background(), PromotionTransition{}); err == nil {
		t.Fatalf("expected error from failed swap")
	}
}

// TestManagerPromoteWiresPromoterAndRestart verifies the Manager's Promote path
// (the real production call) invokes the AppRole swap then restarts the Oracle in
// active mode.
func TestManagerPromoteWiresPromoterAndRestart(t *testing.T) {
	fp := &fakePromoter{}
	m := newTestManager(t)
	m.activator = NewOracleActivator(fp, "/tmp/bao.sock", zap.NewNop())
	var gotMode string
	m.restart = func(ctx context.Context, mode string) error { gotMode = mode; return nil }

	if err := m.Promote(context.Background(), PromotionTransition{RoleID: "oracle-active", OldToken: "tok"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotMode != ModeActive {
		t.Fatalf("restart mode = %q, want active", gotMode)
	}
	if fp.promote != 1 {
		t.Fatalf("promoter called %d times, want 1", fp.promote)
	}
	if fp.lastRole != "oracle-active" {
		t.Fatalf("promoter role = %q", fp.lastRole)
	}
}

// TestManagerDemoteWiresPromoterAndRestart verifies the Manager's Demote path.
func TestManagerDemoteWiresPromoterAndRestart(t *testing.T) {
	fp := &fakePromoter{}
	m := newTestManager(t)
	m.activator = NewOracleActivator(fp, "/tmp/bao.sock", zap.NewNop())
	var gotMode string
	m.restart = func(ctx context.Context, mode string) error { gotMode = mode; return nil }

	if err := m.Demote(context.Background(), PromotionTransition{RoleID: "oracle-standby", OldToken: "tok"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotMode != ModeStandby {
		t.Fatalf("restart mode = %q, want standby", gotMode)
	}
	if fp.swap != 1 {
		t.Fatalf("promoter swap called %d times, want 1", fp.swap)
	}
}

// TestManagerPromoteSwapFailureNoRestart verifies that when the AppRole swap
// fails, the Oracle is NOT restarted (the failover phase is left unchanged).
func TestManagerPromoteSwapFailureNoRestart(t *testing.T) {
	fp := &fakePromoter{err: context.DeadlineExceeded}
	m := newTestManager(t)
	m.activator = NewOracleActivator(fp, "/tmp/bao.sock", zap.NewNop())
	called := false
	m.restart = func(ctx context.Context, mode string) error { called = true; return nil }

	if err := m.Promote(context.Background(), PromotionTransition{}); err == nil {
		t.Fatalf("expected error from failed swap")
	}
	if called {
		t.Fatalf("restart must not run when the swap fails")
	}
}

// fakeVault is a stateful AppRolePromoter that simulates the OpenBao AppRole swap
// (the real launcher.PromoteChildToActive talks to OpenBao over the plugin socket).
// It records the standby token it revoked and the active secret-id it issued, and
// enforces single-active (split-brain) protection: a second promote while already
// active is refused, because the Oracle AppRole is single-issue.
type fakeVault struct {
	mu      sync.Mutex
	promote int
	swap    int
	lastRole string
	lastTok  string
	err      error

	active               bool
	revokedStandbyToken  string
	issuedActiveID       string
	revokedActiveToken   string
	issuedStandbyID      string
}

func (v *fakeVault) PromoteChildToActive(ctx context.Context, socket, roleID, oldToken string) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.promote++
	v.lastRole = roleID
	v.lastTok = oldToken
	if v.err != nil {
		return v.err
	}
	if v.active {
		return fmt.Errorf("oracle-active AppRole already issued for %q (split-brain prevented)", roleID)
	}
	v.revokedStandbyToken = oldToken
	v.issuedActiveID = "active-secret-" + roleID
	v.active = true
	return nil
}

func (v *fakeVault) SwapChildToActive(ctx context.Context, socket, roleID, oldToken string) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.swap++
	v.lastRole = roleID
	v.lastTok = oldToken
	if v.err != nil {
		return v.err
	}
	v.revokedActiveToken = oldToken
	v.issuedStandbyID = "standby-secret-" + roleID
	v.active = false
	return nil
}

// TestManagerPromoteTransition drives the real 8.6 promotion transition with a fake
// vault and asserts the standby token is revoked and a fresh active secret-id is
// issued before the Oracle is restarted in active mode.
func TestManagerPromoteTransition(t *testing.T) {
	v := &fakeVault{}
	m := newTestManager(t)
	m.activator = NewOracleActivator(v, "/tmp/bao.sock", zap.NewNop())
	var gotMode string
	m.restart = func(ctx context.Context, mode string) error { gotMode = mode; return nil }

	if err := m.Promote(context.Background(), PromotionTransition{RoleID: "oracle-standby", OldToken: "standby-tok-1"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v.revokedStandbyToken != "standby-tok-1" {
		t.Fatalf("standby token not revoked: got %q", v.revokedStandbyToken)
	}
	if v.issuedActiveID == "" {
		t.Fatalf("expected a fresh oracle-active secret-id to be issued")
	}
	if !v.active {
		t.Fatalf("vault should be in active state after promote")
	}
	if gotMode != ModeActive {
		t.Fatalf("restart mode = %q, want active", gotMode)
	}
}

// TestManagerDemoteTransition drives the real 8.6 demotion transition (RECLAIMED by
// the original root) and asserts the active token is revoked and the standby
// secret-id issued before restart in standby mode.
func TestManagerDemoteTransition(t *testing.T) {
	v := &fakeVault{}
	m := newTestManager(t)
	m.activator = NewOracleActivator(v, "/tmp/bao.sock", zap.NewNop())
	var gotMode string
	m.restart = func(ctx context.Context, mode string) error { gotMode = mode; return nil }

	if err := m.Promote(context.Background(), PromotionTransition{RoleID: "oracle-standby", OldToken: "standby-tok-1"}); err != nil {
		t.Fatalf("promote: %v", err)
	}
	if err := m.Demote(context.Background(), PromotionTransition{RoleID: "oracle-active", OldToken: "active-tok-1"}); err != nil {
		t.Fatalf("demote: %v", err)
	}
	if v.revokedActiveToken != "active-tok-1" {
		t.Fatalf("active token not revoked on demote: got %q", v.revokedActiveToken)
	}
	if v.issuedStandbyID == "" {
		t.Fatalf("expected a fresh oracle-standby secret-id to be issued")
	}
	if v.active {
		t.Fatalf("vault should be in standby state after demote")
	}
	if gotMode != ModeStandby {
		t.Fatalf("restart mode = %q, want standby", gotMode)
	}
}

// TestManagerPromoteSwapFailureDegraded verifies a failed AppRole swap surfaces as a
// Degraded error and does NOT restart the Oracle (failover phase unchanged).
func TestManagerPromoteSwapFailureDegraded(t *testing.T) {
	v := &fakeVault{err: errors.New("bao vault unreachable")}
	m := newTestManager(t)
	m.activator = NewOracleActivator(v, "/tmp/bao.sock", zap.NewNop())
	called := false
	m.restart = func(ctx context.Context, mode string) error { called = true; return nil }

	if err := m.Promote(context.Background(), PromotionTransition{RoleID: "oracle-standby", OldToken: "tok"}); err == nil {
		t.Fatalf("expected Degraded error from failed swap")
	}
	if called {
		t.Fatalf("restart must not run when the AppRole swap fails")
	}
}

// TestManagerPromoteSplitBrainImpossible verifies two actives are impossible: a
// second promote while already active is refused (the AppRole is single-issue), so
// the failover cannot produce two active Oracle identities.
func TestManagerPromoteSplitBrainImpossible(t *testing.T) {
	v := &fakeVault{}
	m := newTestManager(t)
	m.activator = NewOracleActivator(v, "/tmp/bao.sock", zap.NewNop())
	restarts := 0
	var lastMode string
	m.restart = func(ctx context.Context, mode string) error { restarts++; lastMode = mode; return nil }

	if err := m.Promote(context.Background(), PromotionTransition{RoleID: "oracle-standby", OldToken: "tok-1"}); err != nil {
		t.Fatalf("first promote: %v", err)
	}
	if err := m.Promote(context.Background(), PromotionTransition{RoleID: "oracle-standby", OldToken: "tok-2"}); err == nil {
		t.Fatalf("second promote with no demote must fail (split-brain prevented)")
	}
	if restarts != 1 {
		t.Fatalf("expected exactly one restart (active); got %d", restarts)
	}
	if lastMode != ModeActive {
		t.Fatalf("restart mode = %q, want active", lastMode)
	}
}

// TestManagerPromoteNilPromoterFails is the negative guard: a Manager constructed
// via NewManager (not the production NewOracleManager) has a nil promoter, and
// Promote must fail loudly rather than silently leaving the Oracle in the wrong
// signing identity. This fails if anyone reverts to NewOracleActivator(nil).
func TestManagerPromoteNilPromoterFails(t *testing.T) {
	m := newTestManager(t) // NewManager defaults to a nil promoter
	if m.Promoter() != nil {
		t.Fatalf("expected nil promoter on a default NewManager; the production path must inject one")
	}
	if err := m.Promote(context.Background(), PromotionTransition{RoleID: "oracle-standby", OldToken: "tok"}); err == nil {
		t.Fatalf("expected Promote to fail with a nil promoter")
	}
}
