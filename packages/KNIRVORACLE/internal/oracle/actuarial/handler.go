// Package actuarial is the only path that may disburse NRN for an actuarial
// syndicate settlement (see KNIRV_CORP/acturial_syndicate.md). It never
// mints on the caller's say-so alone: a settlement is paid only once its
// SETTLEMENT_COMMIT commitment has actually been accepted onto KNIRVCHAIN,
// verified via chainverify.Client against KNIRVCHAIN's own transaction
// inclusion proof.
package actuarial

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strconv"
	"sync"

	"github.com/knirvcorp/knirvoracle/internal/oracle/chainverify"
	"github.com/knirvcorp/knirvoracle/internal/oracle/token"
	"github.com/knirvcorp/knirvoracle/internal/oracle/types"
	"go.uber.org/zap"
)

// Disburser is the narrow interface this handler needs from the oracle:
// mint-to-address, the only NRN issuance path. *oracle.Oracle satisfies
// this directly (see Oracle.FundAddress).
type Disburser interface {
	FundAddress(addr types.Address, amount *big.Int, reason string) (*token.MintReceipt, error)
}

// ChainVerifier checks that a transaction hash was actually accepted onto
// KNIRVCHAIN and returns what it committed to. *chainverify.Client
// satisfies this. Returning (nil, nil) means the transaction was never
// accepted — distinct from a transport/parse error.
type ChainVerifier interface {
	VerifyCommitment(txHash string) (*chainverify.VerifiedCommitment, error)
}

// syndicateSettlementTxType must match KNIRVCHAIN's
// TransactionTypeSETTLEMENTCommit wire value exactly (internal/blockchain/
// transaction.go in KNIRV_NETWORK/packages/KNIRVCHAIN).
const syndicateSettlementTxType = "SETTLEMENT_COMMIT"

// settlementCommitmentHash must byte-for-byte match backend_server's
// actuarial.SettlementCommitmentHash (KNIRV_CORP/packages/server/
// backend_server/internal/services/actuarial/chain_adapter.go). This is
// what stops a public caller (this endpoint is reachable through
// KNIRVGATEWAY's public "/oracle/*" proxy) from replaying a real
// settlement's chain_tx_hash against a different destination or amount:
// the commitment anchored on chain only hashes-match a request whose
// settlement_id/destination/amount are exactly what backend_server itself
// anchored.
func settlementCommitmentHash(settlementID, destination string, amount uint64) string {
	sum := sha256.Sum256([]byte(settlementID + "|" + destination + "|" + strconv.FormatUint(amount, 10)))
	return hex.EncodeToString(sum[:])
}

type SettlementPayoutRequest struct {
	SettlementID   string `json:"settlement_id"`
	PoolID         string `json:"pool_id"`
	Destination    string `json:"destination"`
	Amount         string `json:"amount"` // decimal string, smallest units
	ChainTxHash    string `json:"chain_tx_hash"`
	IdempotencyKey string `json:"idempotency_key"`
}

type SettlementPayoutResponse struct {
	ProviderID string `json:"provider_id"`
	Status     string `json:"status"`
}

// Handler serves the settlement-payout endpoint. It holds no private key of
// its own — every disbursement goes through Disburser (the oracle's own
// mint path).
type Handler struct {
	disburser Disburser
	verifier  ChainVerifier
	logger    *zap.Logger

	mu        sync.Mutex
	processed map[string]SettlementPayoutResponse
}

func NewHandler(disburser Disburser, verifier ChainVerifier, logger *zap.Logger) *Handler {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Handler{
		disburser: disburser,
		verifier:  verifier,
		logger:    logger,
		processed: make(map[string]SettlementPayoutResponse),
	}
}

// HandleSettlementPayout is the only HTTP path that may disburse NRN for an
// actuarial syndicate settlement.
func (h *Handler) HandleSettlementPayout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req SettlementPayoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.SettlementID == "" || req.Destination == "" || req.Amount == "" || req.ChainTxHash == "" || req.IdempotencyKey == "" {
		http.Error(w, "settlement_id, destination, amount, chain_tx_hash, and idempotency_key are required", http.StatusBadRequest)
		return
	}

	h.mu.Lock()
	if cached, ok := h.processed[req.IdempotencyKey]; ok {
		h.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(cached)
		return
	}
	h.mu.Unlock()

	amount, ok := new(big.Int).SetString(req.Amount, 10)
	if !ok || amount.Sign() <= 0 || !amount.IsUint64() {
		http.Error(w, "invalid amount", http.StatusBadRequest)
		return
	}
	destination, err := types.AddressFromString(req.Destination)
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid destination: %v", err), http.StatusBadRequest)
		return
	}

	commitment, err := h.verifier.VerifyCommitment(req.ChainTxHash)
	if err != nil {
		h.logger.Error("failed to verify settlement chain proof",
			zap.String("settlement_id", req.SettlementID), zap.String("chain_tx_hash", req.ChainTxHash), zap.Error(err))
		http.Error(w, "failed to verify settlement anchor", http.StatusBadGateway)
		return
	}
	if commitment == nil {
		h.logger.Warn("refusing settlement payout: chain anchor not found",
			zap.String("settlement_id", req.SettlementID), zap.String("chain_tx_hash", req.ChainTxHash))
		http.Error(w, "settlement anchor not found on chain — refusing to disburse", http.StatusUnprocessableEntity)
		return
	}
	// The chain anchor is proof of nothing on its own unless what it
	// actually committed to matches this request — otherwise a caller
	// could replay any real settlement's public tx hash against a
	// different settlement_id/amount/destination and still pass the
	// "was this hash accepted" check above.
	if commitment.TxType != syndicateSettlementTxType {
		h.logger.Warn("refusing settlement payout: chain anchor is not a settlement commitment",
			zap.String("settlement_id", req.SettlementID), zap.String("chain_tx_hash", req.ChainTxHash), zap.String("tx_type", commitment.TxType))
		http.Error(w, "chain anchor is not a settlement commitment", http.StatusUnprocessableEntity)
		return
	}
	if commitment.EntityID != req.SettlementID {
		h.logger.Warn("refusing settlement payout: chain anchor entity mismatch",
			zap.String("settlement_id", req.SettlementID), zap.String("anchor_entity_id", commitment.EntityID))
		http.Error(w, "chain anchor does not match settlement_id", http.StatusUnprocessableEntity)
		return
	}
	if commitment.Amount != amount.Uint64() {
		h.logger.Warn("refusing settlement payout: chain anchor amount mismatch",
			zap.String("settlement_id", req.SettlementID), zap.Uint64("anchor_amount", commitment.Amount), zap.String("requested_amount", req.Amount))
		http.Error(w, "chain anchor amount does not match requested amount", http.StatusUnprocessableEntity)
		return
	}
	// Entity/amount alone aren't enough: neither is bound to *who* gets
	// paid. The commitment_hash is — recomputing it from this request's own
	// destination and comparing against what's actually on chain is what
	// stops a spoofed destination.
	expectedHash := settlementCommitmentHash(req.SettlementID, req.Destination, amount.Uint64())
	if commitment.CommitmentHash != expectedHash {
		h.logger.Warn("refusing settlement payout: chain anchor commitment hash mismatch (possible destination spoofing)",
			zap.String("settlement_id", req.SettlementID), zap.String("destination", req.Destination))
		http.Error(w, "chain anchor commitment hash does not match request", http.StatusUnprocessableEntity)
		return
	}

	receipt, err := h.disburser.FundAddress(destination, amount, "actuarial-settlement:"+req.SettlementID)
	if err != nil {
		h.logger.Error("failed to disburse settlement payout",
			zap.String("settlement_id", req.SettlementID), zap.Error(err))
		http.Error(w, "disbursement failed", http.StatusInternalServerError)
		return
	}

	response := SettlementPayoutResponse{ProviderID: receipt.TransactionHash, Status: "confirmed"}
	h.mu.Lock()
	h.processed[req.IdempotencyKey] = response
	h.mu.Unlock()

	h.logger.Info("disbursed actuarial settlement payout",
		zap.String("settlement_id", req.SettlementID),
		zap.String("pool_id", req.PoolID),
		zap.String("destination", destination.String()),
		zap.String("amount", amount.String()),
		zap.String("chain_tx_hash", req.ChainTxHash),
		zap.String("transaction_hash", receipt.TransactionHash),
	)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
