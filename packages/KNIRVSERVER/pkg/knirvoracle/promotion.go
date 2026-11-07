package knirvoracle

import (
	"context"
	"fmt"

	"go.uber.org/zap"
)

// AppRolePromoter swaps the Oracle's OpenBao AppRole during a failover promotion
// or demotion (spec 2a.6 / 8.6). The production implementation (LauncherPromoter)
// lives in the KNIRVSERVER module and delegates to the launcher's
// PromoteChildToActive / SwapChildToActive hooks, which talk to OpenBao over the
// plugin unix socket. This package stays import-light (it is its own module and
// must not depend on KNIRVSERVER internals), so the concrete caller is injected.
type AppRolePromoter interface {
	// PromoteChildToActive issues a fresh oracle-active secret-id, logs in, and
	// revokes the old (standby) token.
	PromoteChildToActive(ctx context.Context, socket, roleID, oldToken string) error
	// SwapChildToActive issues a fresh oracle-standby secret-id and revokes the
	// active token (used when demoting to standby).
	SwapChildToActive(ctx context.Context, socket, roleID, oldToken string) error
}

// PromotionTransition carries the identity the failover service hands to the
// promotion path.
type PromotionTransition struct {
	// RoleID is the OpenBao role being promoted (oracle-standby <-> oracle-active).
	RoleID string
	// OldToken is the secret-id token of the role being revoked.
	OldToken string
}

// OracleActivator is the 8.6 promotion transition: it swaps the OpenBao AppRole
// and reports the target Oracle mode. The actual Oracle process restart is
// performed by the Manager that owns this activator. Every step is idempotent
// and logged; a failed swap is surfaced to the caller (the failover service) as
// an error so the failover phase is left unchanged.
type OracleActivator struct {
	promoter  AppRolePromoter
	baoSocket string
	logger    *zap.Logger
}

// NewOracleActivator builds an activator. A nil promoter leaves the activator
// unconfigured; Activate/Deactivate then return a clear error (Degraded) rather
// than silently changing the Oracle's signing identity. Tests pass a fake.
func NewOracleActivator(promoter AppRolePromoter, baoSocket string, logger *zap.Logger) *OracleActivator {
	if logger == nil {
		logger, _ = zap.NewProduction()
	}
	return &OracleActivator{promoter: promoter, baoSocket: baoSocket, logger: logger}
}

// Activate swaps the Oracle to the active AppRole and returns the mode the Oracle
// must be (re)started in.
func (a *OracleActivator) Activate(ctx context.Context, t PromotionTransition) (string, error) {
	if a.promoter == nil {
		return "", fmt.Errorf("oracle activator has no AppRole promoter configured")
	}
	if err := a.promoter.PromoteChildToActive(ctx, a.baoSocket, t.RoleID, t.OldToken); err != nil {
		return "", fmt.Errorf("oracle AppRole promotion failed: %w", err)
	}
	a.logger.Info("oracle promoted to active AppRole", zap.String("role", t.RoleID))
	return ModeActive, nil
}

// Deactivate swaps the Oracle back to the standby AppRole and returns standby mode.
func (a *OracleActivator) Deactivate(ctx context.Context, t PromotionTransition) (string, error) {
	if a.promoter == nil {
		return "", fmt.Errorf("oracle activator has no AppRole promoter configured")
	}
	if err := a.promoter.SwapChildToActive(ctx, a.baoSocket, t.RoleID, t.OldToken); err != nil {
		return "", fmt.Errorf("oracle AppRole demotion failed: %w", err)
	}
	a.logger.Info("oracle demoted to standby AppRole", zap.String("role", t.RoleID))
	return ModeStandby, nil
}
