package routes

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/knirvcorp/knirvoracle/internal/oracle/types"
)

// checkInternalServiceToken authorizes an internal, service-to-service
// caller (as opposed to a signed end user) the same way KNIRVCHAIN's
// eventbundle.go/validation_proof.go gate their internal routes: a shared
// KNIRV_INTERNAL_AUTH_TOKEN, compared in constant time, sent via
// X-KNIRV-Internal-Token. Fails closed (503) if the token isn't configured
// at all, rather than failing open. Writes the HTTP error itself and
// returns false when unauthorized/unconfigured, so callers can just
// `if !r.checkInternalServiceToken(w, req) { return }`.
func (r *OracleRoutes) checkInternalServiceToken(w http.ResponseWriter, req *http.Request) bool {
	expectedToken := strings.TrimSpace(os.Getenv("KNIRV_INTERNAL_AUTH_TOKEN"))
	providedToken := strings.TrimSpace(req.Header.Get("X-KNIRV-Internal-Token"))
	if expectedToken == "" {
		http.Error(w, "internal skill-economics endpoints are not configured", http.StatusServiceUnavailable)
		return false
	}
	if subtle.ConstantTimeCompare([]byte(expectedToken), []byte(providedToken)) != 1 {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return false
	}
	return true
}

// handleSkillBurn burns NRN as the cost of canonicalizing a skill, on
// behalf of an internal service caller (KNIRVGRAPH's DRQ skill-minting
// pipeline) rather than a signed end user — see
// token.NRN.BurnForSkillInvocation.
func (r *OracleRoutes) handleSkillBurn(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !r.checkInternalServiceToken(w, req) {
		return
	}

	var burnReq struct {
		From    string `json:"from"`
		SkillID string `json:"skill_id"`
		Amount  string `json:"amount"`
	}
	if err := json.NewDecoder(req.Body).Decode(&burnReq); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(burnReq.SkillID) == "" {
		http.Error(w, "skill_id is required", http.StatusBadRequest)
		return
	}
	fromAddr, err := types.AddressFromString(burnReq.From)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid from address: %v", err), http.StatusBadRequest)
		return
	}
	amount, ok := new(big.Int).SetString(burnReq.Amount, 10)
	if !ok || amount.Sign() <= 0 {
		http.Error(w, "Invalid amount", http.StatusBadRequest)
		return
	}

	receipt, err := r.oracle.GetNRNToken().BurnForSkillInvocation(fromAddr, burnReq.SkillID, amount)
	if err != nil {
		http.Error(w, fmt.Sprintf("Burn failed: %v", err), http.StatusBadRequest)
		return
	}
	respondJSON(w, http.StatusOK, receipt)
}

// handleSkillBounty pays a bounty reward in NRN to a contributing agent, on
// behalf of an internal service caller. KNIRVORACLE is the only component
// allowed to disburse NRN (see routes.go's Stripe-disbursement comment), so
// this reuses the same Mint primitive that backs fiat-triggered
// disbursement and the signed public mint route — just gated by the
// internal service token instead of a signed envelope, since a bounty
// payout is a network decision (DRQ consensus), not a specific user's
// authorized action.
func (r *OracleRoutes) handleSkillBounty(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !r.checkInternalServiceToken(w, req) {
		return
	}

	var bountyReq struct {
		To      string `json:"to"`
		SkillID string `json:"skill_id"`
		Amount  string `json:"amount"`
		Reason  string `json:"reason,omitempty"`
	}
	if err := json.NewDecoder(req.Body).Decode(&bountyReq); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(bountyReq.SkillID) == "" {
		http.Error(w, "skill_id is required", http.StatusBadRequest)
		return
	}
	toAddr, err := types.AddressFromString(bountyReq.To)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid to address: %v", err), http.StatusBadRequest)
		return
	}
	amount, ok := new(big.Int).SetString(bountyReq.Amount, 10)
	if !ok || amount.Sign() <= 0 {
		http.Error(w, "Invalid amount", http.StatusBadRequest)
		return
	}

	receipt, err := r.oracle.GetNRNToken().Mint(toAddr, amount)
	if err != nil {
		http.Error(w, fmt.Sprintf("Bounty payout failed: %v", err), http.StatusBadRequest)
		return
	}
	respondJSON(w, http.StatusOK, receipt)
}

// handleSkillOwnership registers (POST) or retrieves (GET) a durable
// perpetual invocation-fee entitlement for a canonically-minted skill.
func (r *OracleRoutes) handleSkillOwnership(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodPost:
		if !r.checkInternalServiceToken(w, req) {
			return
		}
		var ownershipReq struct {
			SkillID       string  `json:"skill_id"`
			AgentID       string  `json:"agent_id"`
			InvocationFee float64 `json:"invocation_fee"`
			Perpetual     bool    `json:"perpetual"`
		}
		if err := json.NewDecoder(req.Body).Decode(&ownershipReq); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(ownershipReq.SkillID) == "" || strings.TrimSpace(ownershipReq.AgentID) == "" {
			http.Error(w, "skill_id and agent_id are required", http.StatusBadRequest)
			return
		}
		if ownershipReq.InvocationFee < 0 {
			http.Error(w, "invocation_fee must not be negative", http.StatusBadRequest)
			return
		}
		record := &types.SkillOwnershipRecord{
			SkillID:       ownershipReq.SkillID,
			AgentID:       ownershipReq.AgentID,
			InvocationFee: ownershipReq.InvocationFee,
			Perpetual:     ownershipReq.Perpetual,
			RegisteredAt:  time.Now().UTC(),
		}
		if err := r.oracle.RegisterSkillOwnership(record); err != nil {
			http.Error(w, fmt.Sprintf("Failed to register ownership: %v", err), http.StatusInternalServerError)
			return
		}
		respondJSON(w, http.StatusOK, record)

	case http.MethodGet:
		skillID := strings.TrimPrefix(req.URL.Path, "/oracle/v3/skills/ownership/")
		if strings.TrimSpace(skillID) == "" {
			http.Error(w, "skill id is required", http.StatusBadRequest)
			return
		}
		record, exists := r.oracle.GetSkillOwnership(skillID)
		if !exists {
			http.Error(w, "skill ownership not found", http.StatusNotFound)
			return
		}
		respondJSON(w, http.StatusOK, record)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleSkillVerify performs KNIRVORACLE's real skill-node verification
// ahead of canonical minting: structural/policy validation of the skill
// itself, plus — when the caller already has a KNIRVCHAIN EventBundleNFT
// mint receipt for it (KNIRVGRAPH's DRQ pipeline mints on KNIRVCHAIN before
// asking Oracle to verify, see SkillMintingProtocol.MintSkillFromCluster) —
// schema- and identity-checking that receipt.
//
// This intentionally does not re-fetch/re-hash the bundle from KNIRVCHAIN
// itself (unlike KNIRVSERVER's native_verifier.go, which does have a direct
// KNIRVCHAIN client for CLI commit proofs): KNIRVORACLE has no existing
// KNIRVCHAIN client dependency, and adding one is out of scope here. What
// this endpoint checks is real, not a stub — it just doesn't yet do live
// cross-chain re-verification.
func (r *OracleRoutes) handleSkillVerify(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !r.checkInternalServiceToken(w, req) {
		return
	}

	var verifyReq struct {
		SkillID        string   `json:"skill_id"`
		Creator        string   `json:"creator"`
		Description    string   `json:"description"`
		ResolvesErrors []string `json:"resolves_errors"`
		CodePackageURI string   `json:"code_package_uri"`
		BundleReceipt  []byte   `json:"bundle_receipt,omitempty"`
	}
	if err := json.NewDecoder(req.Body).Decode(&verifyReq); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(verifyReq.SkillID) == "" {
		respondJSON(w, http.StatusOK, map[string]any{"verified": false, "reason": "skill_id is required"})
		return
	}
	if strings.TrimSpace(verifyReq.Creator) == "" {
		respondJSON(w, http.StatusOK, map[string]any{"verified": false, "reason": "creator is required"})
		return
	}
	if strings.TrimSpace(verifyReq.Description) == "" {
		respondJSON(w, http.StatusOK, map[string]any{"verified": false, "reason": "description is required"})
		return
	}
	if len(verifyReq.ResolvesErrors) == 0 {
		respondJSON(w, http.StatusOK, map[string]any{"verified": false, "reason": "skill must resolve at least one error"})
		return
	}

	if len(verifyReq.BundleReceipt) > 0 {
		var receipt struct {
			EventID    string `json:"event_id"`
			BundleHash string `json:"bundle_hash"`
		}
		if err := json.Unmarshal(verifyReq.BundleReceipt, &receipt); err != nil {
			respondJSON(w, http.StatusOK, map[string]any{"verified": false, "reason": "malformed bundle receipt"})
			return
		}
		if receipt.EventID != verifyReq.SkillID {
			respondJSON(w, http.StatusOK, map[string]any{"verified": false, "reason": "bundle receipt event id does not match skill id"})
			return
		}
		if strings.TrimSpace(receipt.BundleHash) == "" {
			respondJSON(w, http.StatusOK, map[string]any{"verified": false, "reason": "bundle receipt is missing its hash"})
			return
		}
	}

	respondJSON(w, http.StatusOK, map[string]any{"verified": true, "reason": ""})
}
