package payment_test

import (
	"math/big"
	"testing"

	"github.com/KNIRV/KNIRV_NETWORK/KNIRVGATEWAY/internal/payment"
	"go.uber.org/zap"
)

func TestEconomicRules_HasValidationReportCost(t *testing.T) {
	engine := payment.NewEconomicsEngine(zap.NewNop())
	rules := engine.GetRules()

	expected := big.NewInt(1000000000000000000) // 1 NRN
	if rules.ValidationReportCost.Cmp(expected) != 0 {
		t.Errorf("ValidationReportCost = %s, want %s",
			rules.ValidationReportCost.String(), expected.String())
	}
	if rules.ValidationReportCostStr != expected.String() {
		t.Errorf("ValidationReportCostStr = %s, want %s",
			rules.ValidationReportCostStr, expected.String())
	}

	// Also verify the other cost fields are populated correctly
	if rules.SkillInvocationCost.Cmp(expected) != 0 {
		t.Errorf("SkillInvocationCost = %s, want %s",
			rules.SkillInvocationCost.String(), expected.String())
	}
}
