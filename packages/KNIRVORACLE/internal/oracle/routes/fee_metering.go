package routes

import (
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strings"

	"github.com/knirvcorp/knirvoracle/internal/oracle/economics"
	"github.com/knirvcorp/knirvoracle/internal/oracle/types"
	"go.uber.org/zap"
)

// handleInternalFeeCollect is the settlement path for every metered NRN
// consumption event on the platform (the backend's nrnmeter service).
//
// It exists because the economics engine already implements exactly this
// operation — economics.FeeCollector.CollectFee debits the payer, burns the
// configured share (BurnConfig.BurnPercentage, 0.5 per
// NRN_Consumption_Report.md §4.1) and credits the remainder to the on-ledger
// reward pool, atomically, returning a FeeReceipt — but no route ever exposed
// it. Metering therefore had no server-side settlement primitive at all.
//
// This handler is deliberately thin: it authenticates the internal caller,
// validates the payload, and delegates straight to CollectFee. It performs no
// arithmetic of its own, so the burn/reward split has exactly one
// implementation (the FeeCollector's) and cannot drift from what the
// economics metrics report.
//
// Unlike /oracle/v3/token/burn/signed this route does not require a per-user
// signature: the caller is a trusted platform service, authenticated by the
// shared KNIRV_INTERNAL_AUTH_TOKEN, exactly like the other /skills/* routes.
// The platform service is monetising a user's already-authenticated action,
// not acting as the user.
func (r *OracleRoutes) handleInternalFeeCollect(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !r.checkInternalServiceToken(w, req) {
		return
	}

	var collectReq struct {
		From    string `json:"from"`
		FeeType string `json:"fee_type"`
		Amount  string `json:"amount"`
		Reason  string `json:"reason,omitempty"`
	}
	if err := json.NewDecoder(req.Body).Decode(&collectReq); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(collectReq.From) == "" {
		http.Error(w, "from is required", http.StatusBadRequest)
		return
	}
	fromAddr, err := types.AddressFromString(collectReq.From)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid from address: %v", err), http.StatusBadRequest)
		return
	}
	if fromAddr.IsZero() {
		http.Error(w, "from must not be the zero address", http.StatusBadRequest)
		return
	}

	feeType, ok := economics.MeteredFeeType(collectReq.FeeType)
	if !ok {
		http.Error(w, fmt.Sprintf("Unsupported fee_type %q", collectReq.FeeType), http.StatusBadRequest)
		return
	}

	amount, ok := new(big.Int).SetString(strings.TrimSpace(collectReq.Amount), 10)
	if !ok || amount.Sign() <= 0 {
		http.Error(w, "amount must be a positive integer in base units", http.StatusBadRequest)
		return
	}

	engine := r.oracle.GetEconomicsEngine()
	if engine == nil {
		http.Error(w, "economics engine unavailable", http.StatusServiceUnavailable)
		return
	}

	receipt, err := engine.GetFeeCollector().CollectFee(fromAddr, feeType, amount)
	if err != nil {
		// Insufficient balance is a client-side problem (402), anything else is
		// a genuine server fault. Both are surfaced rather than swallowed.
		status := http.StatusBadRequest
		if strings.Contains(err.Error(), types.ErrInsufficientBalance.Error()) {
			status = http.StatusPaymentRequired
		}
		http.Error(w, fmt.Sprintf("Fee collection failed: %v", err), status)
		return
	}

	if r.logger != nil {
		r.logger.Info("metered fee collected",
			zap.String("from", fromAddr.String()),
			zap.String("fee_type", string(feeType)),
			zap.String("amount", amount.String()),
			zap.String("burned", receipt.Burned.String()),
			zap.String("tx_hash", receipt.TxHash),
			zap.String("reason", collectReq.Reason),
		)
	}

	respondJSON(w, http.StatusOK, struct {
		From      string `json:"from"`
		FeeType   string `json:"fee_type"`
		Amount    string `json:"amount"`
		Burned    string `json:"burned"`
		Rewarded  string `json:"rewarded"`
		TxHash    string `json:"tx_hash"`
		Timestamp string `json:"timestamp"`
	}{
		From:      receipt.From.String(),
		FeeType:   string(receipt.FeeType),
		Amount:    receipt.Amount.String(),
		Burned:    receipt.Burned.String(),
		Rewarded:  receipt.Rewarded.String(),
		TxHash:    receipt.TxHash,
		Timestamp: receipt.Timestamp.UTC().Format("2006-01-02T15:04:05Z07:00"),
	})
}
