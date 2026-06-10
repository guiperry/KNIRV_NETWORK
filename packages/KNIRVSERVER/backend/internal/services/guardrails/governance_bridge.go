package guardrails

type GovernanceBridge struct {
	guardrailManager *DynamicGuardrailManager
	policyEngine     *PolicyEngine
}

func NewGovernanceBridge(gm *DynamicGuardrailManager, pe *PolicyEngine) *GovernanceBridge {
	return &GovernanceBridge{
		guardrailManager: gm,
		policyEngine:     pe,
	}
}

func (gb *GovernanceBridge) ValidateNodeAction(nodeID, action string, context map[string]interface{}) (bool, string) {
	reason := ""
	if gb.policyEngine != nil {
		allowed, r := gb.policyEngine.EvaluateAction(nodeID, action, context)
		if !allowed {
			return false, r
		}
		reason = r
	}

	if gb.guardrailManager != nil {
		if metrics, ok := context["metrics"].(map[string]interface{}); ok {
			violation, hasViolation := gb.guardrailManager.ValidateResourceUsage(nodeID, metrics)
			if hasViolation {
				return false, violation.Message
			}
		}
	}

	if reason != "" {
		return true, reason
	}
	return true, "allowed"
}

func (gb *GovernanceBridge) RecordViolation(nodeID, action string, severity string, details map[string]interface{}) error {
	if gb.guardrailManager == nil {
		return nil
	}

	violation := &GuardrailViolation{
		NodeID:        nodeID,
		GuardrailType: GuardrailTypeExecution,
		ViolationType: action,
		Message:       details["message"].(string),
		Severity:      severity,
		Context:       details,
	}

	return gb.guardrailManager.RecordViolation(violation)
}
