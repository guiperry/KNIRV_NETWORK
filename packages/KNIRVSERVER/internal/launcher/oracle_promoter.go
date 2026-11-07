package launcher

import (
	"context"

	knirvoracle "github.com/KNIRV/KNIRV_NETWORK/KNIRVORACLE"
	"go.uber.org/zap"
)

// OraclePromoter is the production knirvoracle.AppRolePromoter. It is the real
// production caller of the launcher's failover/promotion transition hooks
// (PromoteChildToActive / SwapChildToActive), which talk to OpenBao over the
// plugin unix socket. The knirvoracle.Manager.Promote/Demote paths (spec 8.6) use
// this promoter so a confirmed failover swaps the Oracle's AppRole from
// oracle-standby to oracle-active (and back on RECLAIMED) before restarting the
// Oracle child.
type OraclePromoter struct{}

// PromoteChildToActive implements knirvoracle.AppRolePromoter.
func (OraclePromoter) PromoteChildToActive(ctx context.Context, socket, roleID, oldToken string) error {
	_, err := PromoteChildToActive(ctx, socket, roleID, oldToken)
	return err
}

// SwapChildToActive implements knirvoracle.AppRolePromoter.
func (OraclePromoter) SwapChildToActive(ctx context.Context, socket, roleID, oldToken string) error {
	_, err := SwapChildToActive(ctx, socket, roleID, oldToken)
	return err
}

// Compile-time guarantee that OraclePromoter satisfies the knirvoracle contract.
var _ knirvoracle.AppRolePromoter = OraclePromoter{}

// NewOracleManager is the production constructor for the Oracle promotion Manager
// (spec 8.6). It builds the knirvoracle.Manager and injects the AppRolePromoter
// passed by the caller — the production path passes OraclePromoter{}, the real
// promoter that reaches launcher.PromoteChildToActive / SwapChildToActive over the
// OpenBao plugin socket on a CONFIRMED/RECLAIMED transition. Passing the promoter
// in keeps the dependency narrow: the constructor never swaps in a fake, so the
// transition test exercises the real injected promoter. A nil promoter makes
// Promote fail loudly (see the nil-promoter guard test); always pass a real
// promoter in production.
func NewOracleManager(cfg *knirvoracle.ManagerConfig, logger *zap.Logger, promoter knirvoracle.AppRolePromoter) *knirvoracle.Manager {
	m := knirvoracle.NewManager(cfg, logger)
	m.SetActivator(knirvoracle.NewOracleActivator(promoter, cfg.BaoSocketPath, logger))
	return m
}

// OracleFailoverManager returns the production Oracle promotion Manager (spec 8.6)
// constructed in startOpenBao with the real AppRolePromoter injected. The failover
// service drives CONFIRMED/RECLAIMED transitions through it; it is nil when OpenBao
// (the AppRole backend) is disabled.
func (app *ServerApp) OracleFailoverManager() *knirvoracle.Manager {
	return app.oracleManager
}
