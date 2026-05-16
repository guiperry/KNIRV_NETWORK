package payment_test

import (
	"math/big"
	"testing"

	"github.com/KNIRV/KNIRV_NETWORK/KNIRVGATEWAY/internal/payment"
	"go.uber.org/zap"
)

func TestNewEconomicsEngine_HasValidationReportCost(t *testing.T) {
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
}

func TestProcessValidationReport(t *testing.T) {
	oneNRN := "1000000000000000000"

	tests := []struct {
		name    string
		req     *payment.ValidationReportRequest
		wantErr bool
		errMsg  string
	}{
		{
			name: "ValidPayment",
			req: &payment.ValidationReportRequest{
				WalletAddress: "0xabc123",
				DVEID:         "dve-test-1",
				SignedTx:      "0xtxhash",
				Amount:        oneNRN, // 1 NRN
			},
			wantErr: false,
		},
		{
			name: "InsufficientPayment",
			req: &payment.ValidationReportRequest{
				WalletAddress: "0xabc123",
				DVEID:         "dve-test-1",
				SignedTx:      "0xtxhash",
				Amount:        "500000000000000000", // 0.5 NRN — below cost
			},
			wantErr: true,
			errMsg:  "insufficient payment",
		},
		{
			name: "InvalidAmount",
			req: &payment.ValidationReportRequest{
				WalletAddress: "0xabc123",
				DVEID:         "dve-test-1",
				SignedTx:      "0xtxhash",
				Amount:        "not-a-number",
			},
			wantErr: true,
			errMsg:  "invalid amount",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := payment.NewEconomicsEngine(zap.NewNop())

			tx, err := engine.ProcessValidationReport(tt.req)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.errMsg != "" {
					// Use Contains check by comparing prefix
					if len(err.Error()) >= len(tt.errMsg) && err.Error()[:len(tt.errMsg)] != tt.errMsg {
						t.Errorf("error message = %q, want prefix %q", err.Error(), tt.errMsg)
					}
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tx == nil {
				t.Fatal("expected transaction, got nil")
			}

			if tx.Type != "validation_report" {
				t.Errorf("tx.Type = %q, want %q", tx.Type, "validation_report")
			}
			if tx.Status != "confirmed" {
				t.Errorf("tx.Status = %q, want %q", tx.Status, "confirmed")
			}
			if tx.To != "reward_pool" {
				t.Errorf("tx.To = %q, want %q", tx.To, "reward_pool")
			}
			if tx.From != "0xabc123" {
				t.Errorf("tx.From = %q, want %q", tx.From, "0xabc123")
			}
		})
	}
}

func TestProcessValidationReport_RecordsTransaction(t *testing.T) {
	engine := payment.NewEconomicsEngine(zap.NewNop())

	req := &payment.ValidationReportRequest{
		WalletAddress: "0xabc123",
		DVEID:         "dve-test-1",
		SignedTx:      "0xtxhash",
		Amount:        "1000000000000000000", // 1 NRN
	}

	tx, err := engine.ProcessValidationReport(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Retrieve transactions without filtering by status or limit
	transactions := engine.GetTransactions(0, "")
	if len(transactions) != 1 {
		t.Fatalf("expected 1 transaction, got %d", len(transactions))
	}

	recorded := transactions[0]
	if recorded.ID != tx.ID {
		t.Errorf("recorded tx ID = %q, want %q", recorded.ID, tx.ID)
	}
	if recorded.Type != "validation_report" {
		t.Errorf("recorded tx.Type = %q, want %q", recorded.Type, "validation_report")
	}
	if recorded.From != "0xabc123" {
		t.Errorf("recorded tx.From = %q, want %q", recorded.From, "0xabc123")
	}
	if recorded.To != "reward_pool" {
		t.Errorf("recorded tx.To = %q, want %q", recorded.To, "reward_pool")
	}
	if recorded.Status != "confirmed" {
		t.Errorf("recorded tx.Status = %q, want %q", recorded.Status, "confirmed")
	}
}

func TestProcessValidationReport_RecordsBurnEvent(t *testing.T) {
	engine := payment.NewEconomicsEngine(zap.NewNop())

	req := &payment.ValidationReportRequest{
		WalletAddress: "0xabc123",
		DVEID:         "dve-test-1",
		SignedTx:      "0xtxhash",
		Amount:        "1000000000000000000", // 1 NRN
	}

	_, err := engine.ProcessValidationReport(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Retrieve burn events without filtering
	events := engine.GetBurnHistory(0, "", "")
	if len(events) != 1 {
		t.Fatalf("expected 1 burn event, got %d", len(events))
	}

	event := events[0]
	if event.User != "0xabc123" {
		t.Errorf("burn event User = %q, want %q", event.User, "0xabc123")
	}
	if event.Purpose != "validation_report" {
		t.Errorf("burn event Purpose = %q, want %q", event.Purpose, "validation_report")
	}
	if event.SkillID != "dve-test-1" {
		t.Errorf("burn event SkillID = %q, want %q", event.SkillID, "dve-test-1")
	}
	if event.Validated {
		t.Errorf("burn event Validated = true, want false")
	}
}
