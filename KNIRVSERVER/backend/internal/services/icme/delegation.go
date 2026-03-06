package icme

import "go.uber.org/zap"

type DelegationFramework struct {
	registry *IntentRegistry
	logger   *zap.Logger
}

func NewDelegationFramework(registry *IntentRegistry, logger *zap.Logger) *DelegationFramework {
	return &DelegationFramework{registry: registry, logger: logger}
}

func (d *DelegationFramework) Resolve(ctx DecisionContext) DecisionResult {
	if d.registry.ViolatesHardBoundary(ctx.AgentID, ctx.DVEID, ctx.Action) {
		d.logger.Warn("icme hard boundary violation",
			zap.String("agent_id", ctx.AgentID),
			zap.String("dve_id", ctx.DVEID),
			zap.String("action", ctx.Action),
		)
		return DecisionResult{
			Approved: false,
			Action:   "deny",
			Reason:   "hard boundary violated: " + ctx.Action,
		}
	}

	if !d.registry.IsActionAuthorized(ctx.AgentID, ctx.DVEID, ctx.Action) {
		return DecisionResult{
			Approved: false,
			Action:   "escalate_to_manager",
			Reason:   "action not in authorized list: " + ctx.Action,
		}
	}

	if ctx.CustomerTier == "VIP" {
		return DecisionResult{
			Approved: true,
			Action:   "escalate_to_specialist",
			Reason:   "VIP tier prioritizes satisfaction",
		}
	}

	if ctx.Amount > 1000 {
		return DecisionResult{
			Approved: false,
			Action:   "deny",
			Reason:   "amount exceeds threshold",
		}
	}

	return DecisionResult{
		Approved: true,
		Action:   "approve",
		Reason:   "within objective constraints",
	}
}
