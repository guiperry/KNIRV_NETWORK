package routes

import (
	"bytes"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	signing "github.com/KNIRV/KNIRV_NETWORK/KNIRVSDK/go/signing"
	"github.com/knirvcorp/knirvoracle/internal/oracle"
	"github.com/knirvcorp/knirvoracle/internal/oracle/crosschain"
	"github.com/knirvcorp/knirvoracle/internal/oracle/governance"
	"github.com/knirvcorp/knirvoracle/internal/oracle/mmr"
	"github.com/knirvcorp/knirvoracle/internal/oracle/types"
	"go.uber.org/zap"
)

// OracleRoutes provides HTTP handlers for oracle endpoints
type OracleRoutes struct {
	oracle         *oracle.Oracle
	logger         *zap.Logger
	mintMu         sync.Mutex
	usedMintNonces map[string]time.Time
}

// NewOracleRoutes creates a new oracle routes handler
func NewOracleRoutes(oracleInstance *oracle.Oracle, logger *zap.Logger) *OracleRoutes {
	return &OracleRoutes{
		oracle:         oracleInstance,
		logger:         logger,
		usedMintNonces: make(map[string]time.Time),
	}
}

func decodeHexBytes(value string) ([]byte, error) {
	return hex.DecodeString(strings.TrimPrefix(value, "0x"))
}

// RegisterRoutes registers all oracle routes with an HTTP mux
// Compatible with standard http.ServeMux or any router
func (r *OracleRoutes) RegisterRoutes(mux *http.ServeMux) {
	// Token endpoints
	mux.HandleFunc("/oracle/v3/token/info", r.handleTokenInfo)
	mux.HandleFunc("/oracle/v3/token/balance/", r.handleTokenBalance)
	mux.HandleFunc("/oracle/v3/token/mint", r.handleTokenMint)
	mux.HandleFunc("/oracle/v3/token/transfer", r.handleTokenTransfer)
	mux.HandleFunc("/oracle/v3/token/transfer/signed", r.handleTokenTransferSigned)
	mux.HandleFunc("/oracle/v3/token/burn", r.handleTokenBurn)
	mux.HandleFunc("/oracle/v3/token/burn/signed", r.handleTokenBurnSigned)
	mux.HandleFunc("/oracle/v3/account/nonce/", r.handleAccountNonce)

	// Governance endpoints
	mux.HandleFunc("/oracle/v3/governance/proposals", r.handleProposals)
	mux.HandleFunc("/oracle/v3/governance/proposals/", r.handleProposal)
	mux.HandleFunc("/oracle/v3/governance/vote", r.handleVote)
	mux.HandleFunc("/oracle/v3/governance/validators", r.handleValidators)
	mux.HandleFunc("/oracle/v3/governance/validators/", r.handleValidator)
	mux.HandleFunc("/oracle/v3/governance/params", r.handleNetworkParams)

	// Economics endpoints
	mux.HandleFunc("/oracle/v3/economics/metrics", r.handleEconomicMetrics)
	mux.HandleFunc("/oracle/v3/economics/fees", r.handleFees)
	mux.HandleFunc("/oracle/v3/economics/rewards", r.handleRewards)
	mux.HandleFunc("/oracle/v3/economics/staking", r.handleStaking)
	mux.HandleFunc("/oracle/v3/economics/burns", r.handleBurns)

	// Skill economics endpoints (internal-service-token gated): used by
	// KNIRVGRAPH's DRQ skill-minting pipeline, not signed end users.
	mux.HandleFunc("/oracle/v3/skills/burn", r.handleSkillBurn)
	mux.HandleFunc("/oracle/v3/skills/bounty", r.handleSkillBounty)
	mux.HandleFunc("/oracle/v3/skills/ownership", r.handleSkillOwnership)
	mux.HandleFunc("/oracle/v3/skills/ownership/", r.handleSkillOwnership)
	mux.HandleFunc("/oracle/v3/skills/verify", r.handleSkillVerify)

	// Cross-chain endpoints
	mux.HandleFunc("/oracle/v3/crosschain/transfer", r.handleCrossChainTransfer)
	mux.HandleFunc("/oracle/v3/crosschain/transfer/", r.handleGetTransfer)
	mux.HandleFunc("/oracle/v3/crosschain/bridges", r.handleBridges)

	// IBC endpoints
	mux.HandleFunc("/oracle/v3/ibc/channels", r.handleIBCChannels)
	mux.HandleFunc("/oracle/v3/ibc/connections", r.handleIBCConnections)
	mux.HandleFunc("/oracle/v3/ibc/clients", r.handleIBCClients)

	// Consensus endpoints
	mux.HandleFunc("/oracle/v3/consensus/status", r.handleConsensusStatus)
	mux.HandleFunc("/oracle/v3/consensus/validators", r.handleConsensusValidators)

	// P2P endpoints
	mux.HandleFunc("/oracle/v3/p2p/status", r.handleP2PStatus)
	mux.HandleFunc("/oracle/v3/p2p/peers", r.handleP2PPeers)
	mux.HandleFunc("/oracle/v3/p2p/pause", r.handleP2PPause)
	mux.HandleFunc("/oracle/v3/p2p/resume", r.handleP2PResume)

	// Rollup settlement endpoints
	mux.HandleFunc("/oracle/v3/rollups/submit", r.handleSubmitRollup)
	mux.HandleFunc("/oracle/v3/rollups/", r.handleRollup)

	// Checkpoint pipeline (Phase 2): registry + admission + MMR proofs.
	mux.HandleFunc("/oracle/v3/registry/register", r.handleRegistryRegister)
	mux.HandleFunc("/oracle/v3/registry/rotate", r.handleRegistryRotate)
	mux.HandleFunc("/oracle/v3/registry/verify-key", r.handleSetVerificationKey)
	mux.HandleFunc("/oracle/v3/registry/", r.handleGetRegistration)
	mux.HandleFunc("/oracle/v3/checkpoints", r.handleSubmitCheckpoint)
	mux.HandleFunc("/oracle/v3/checkpoints/", r.handleCheckpointsByChain)
	mux.HandleFunc("/oracle/v3/mmr/root", r.handleMMRRoot)
	mux.HandleFunc("/oracle/v3/mmr/proof/", r.handleMMRProof)

	// Phase 4: finality pipeline (verifier attestation → LeafFinality).
	mux.HandleFunc("/oracle/v3/finality", r.handleSubmitFinality)
	mux.HandleFunc("/oracle/v3/finality/attest", r.handleSubmitAttestation)
	mux.HandleFunc("/oracle/v3/verifiers/register", r.handleRegisterVerifier)

	// Installation endpoints
	mux.HandleFunc("/oracle/v3/install/wallet", r.handleInstallWallet)
	mux.HandleFunc("/oracle/v3/install/dve_uri", r.handleInstallDVEURI)
	mux.HandleFunc("/oracle/v3/wallet/register", r.handleRegisterWallet)
	mux.HandleFunc("/oracle/v3/wallet/status", r.handleWalletStatus)

	// Legacy wallet server compatibility endpoints now hosted by KNIRVORACLE.
	mux.HandleFunc("/generate_wallet", r.handleGenerateWallet)
	mux.HandleFunc("/balance/", r.handleLegacyWalletBalance)
	mux.HandleFunc("/transactions", r.handleCreateTransaction)
	mux.HandleFunc("/send_signed_txn", r.handleSendSignedTransaction)
	mux.HandleFunc("/test/faucet", r.handleTestFaucet)

	// Payment processor: fiat-triggered NRN disbursement, migrated from
	// KNIRVCHAIN's internal/wallet/payment_processor.go. KNIRVORACLE is the
	// only component allowed to disburse NRN, so this replaces the old
	// P-256-signed KNIRVCHAIN wallet path with the oracle's own secp256k1 mint.
	mux.HandleFunc("/oracle/v3/payment/webhooks/stripe", r.oracle.GetPaymentProcessor().HandleStripeWebhook)

	// Subscription-plan checkout (KNIRV.COM Pro plan). Public: reached by
	// onboarding.knirv.com's browser-side JS through KNIRVGATEWAY's public
	// hostnames (gateway.knirv.network / testnet-gateway.knirv.network),
	// which proxy this path verbatim. Distinct from the disbursement webhook
	// above — this creates the Stripe Checkout Session; the webhook above
	// confirms it once Stripe redirects back.
	mux.HandleFunc("/oracle/v3/payment/checkout/create", r.oracle.GetPaymentProcessor().HandleCreateCheckoutSession)

	// Health and status
	mux.HandleFunc("/oracle/v3/health", r.handleHealth)
	mux.HandleFunc("/oracle/v3/status", r.handleStatus)

	// DID resolution engine (Phase 1-A): register/resolve/deactivate did:knirv: documents.
	r.RegisterDIDRoutes(mux)
}

// ========== Token Handlers ==========

func (r *OracleRoutes) handleTokenInfo(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	info := r.oracle.GetNRNToken().Info()
	respondJSON(w, http.StatusOK, info)
}

func (r *OracleRoutes) handleTokenBalance(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract address from path (last segment)
	addressStr := req.URL.Path[len("/oracle/v3/token/balance/"):]
	address, err := types.AddressFromString(addressStr)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid address: %v", err), http.StatusBadRequest)
		return
	}

	balance := r.oracle.GetNRNToken().GetBalance(address)
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"address": addressStr,
		"balance": balance.String(),
	})
}

func (r *OracleRoutes) handleAccountNonce(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	addressStr := req.URL.Path[len("/oracle/v3/account/nonce/"):]
	address, err := types.AddressFromString(addressStr)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid address: %v", err), http.StatusBadRequest)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"address": address.String(),
		"nonce":   r.oracle.GetNRNToken().GetNonce(address),
	})
}

func (r *OracleRoutes) handleTokenMint(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var mintReq struct {
		To        string `json:"to"`
		Amount    string `json:"amount"`
		Signature string `json:"signature"`
	}

	if err := json.NewDecoder(req.Body).Decode(&mintReq); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	toAddr, err := types.AddressFromString(mintReq.To)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid to address: %v", err), http.StatusBadRequest)
		return
	}

	amount, ok := new(big.Int).SetString(mintReq.Amount, 10)
	if !ok || amount.Sign() <= 0 {
		http.Error(w, "Invalid amount", http.StatusBadRequest)
		return
	}
	registeredMinter := strings.TrimSpace(os.Getenv("KNIRV_ORACLE_MINTER_ADDRESS"))
	if registeredMinter == "" {
		http.Error(w, "Oracle mint service is not configured", http.StatusServiceUnavailable)
		return
	}
	var signed signing.SignedMessage
	if err := json.Unmarshal([]byte(mintReq.Signature), &signed); err != nil {
		http.Error(w, "Invalid canonical mint authorization", http.StatusBadRequest)
		return
	}
	envelope, err := signing.ParseMessageEnvelope(signed.Envelope)
	if err != nil || envelope.Domain != "knirv.oracle" || envelope.Purpose != "token-mint" || signed.Address != registeredMinter {
		http.Error(w, "Invalid mint authorization domain or signer", http.StatusUnauthorized)
		return
	}
	chainID := r.oracle.ChainID()
	if err := signing.VerifyMessage(signed, "knirv.oracle", "token-mint", chainID, envelope.Nonce, time.Now()); err != nil {
		http.Error(w, "Invalid or expired mint authorization", http.StatusUnauthorized)
		return
	}
	payload, _ := json.Marshal(struct {
		To     string `json:"to"`
		Amount string `json:"amount"`
	}{toAddr.String(), amount.String()})
	if !bytes.Equal(payload, envelope.Payload) {
		http.Error(w, "Mint authorization payload mismatch", http.StatusUnauthorized)
		return
	}
	r.mintMu.Lock()
	for nonce, expiry := range r.usedMintNonces {
		if time.Now().After(expiry) {
			delete(r.usedMintNonces, nonce)
		}
	}
	if _, exists := r.usedMintNonces[envelope.Nonce]; exists {
		r.mintMu.Unlock()
		http.Error(w, "Mint authorization was already used", http.StatusConflict)
		return
	}
	r.usedMintNonces[envelope.Nonce] = time.Unix(envelope.ExpiresAtUnix, 0)
	r.mintMu.Unlock()

	receipt, err := r.oracle.GetNRNToken().Mint(toAddr, amount)
	if err != nil {
		r.mintMu.Lock()
		delete(r.usedMintNonces, envelope.Nonce)
		r.mintMu.Unlock()
		http.Error(w, fmt.Sprintf("Mint failed: %v", err), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, receipt)
}

func (r *OracleRoutes) handleTokenTransfer(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	http.Error(w, "private-key transfer is disabled; use /oracle/v3/token/transfer/signed with KNIRVCONTROLLER", http.StatusGone)
}

func (r *OracleRoutes) handleTokenTransferSigned(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var transferReq struct {
		From      string `json:"from"`
		To        string `json:"to"`
		Amount    string `json:"amount"`
		Nonce     uint64 `json:"nonce"`
		Signature string `json:"signature"`
	}

	if err := json.NewDecoder(req.Body).Decode(&transferReq); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	fromAddr, err := types.AddressFromString(transferReq.From)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid from address: %v", err), http.StatusBadRequest)
		return
	}
	toAddr, err := types.AddressFromString(transferReq.To)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid to address: %v", err), http.StatusBadRequest)
		return
	}
	amount, ok := new(big.Int).SetString(transferReq.Amount, 10)
	if !ok {
		http.Error(w, "Invalid amount", http.StatusBadRequest)
		return
	}
	var signed signing.SignedMessage
	if err := json.Unmarshal([]byte(transferReq.Signature), &signed); err != nil {
		http.Error(w, fmt.Sprintf("Invalid signature: %v", err), http.StatusBadRequest)
		return
	}

	receipt, err := r.oracle.GetNRNToken().TransferSigned(fromAddr, toAddr, amount, transferReq.Nonce, signed)
	if err != nil {
		http.Error(w, fmt.Sprintf("Transfer failed: %v", err), http.StatusBadRequest)
		return
	}

	respondJSON(w, http.StatusOK, receipt)
}

func (r *OracleRoutes) handleTokenBurn(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	http.Error(w, "private-key burn is disabled; use /oracle/v3/token/burn/signed with KNIRVCONTROLLER", http.StatusGone)
}

func (r *OracleRoutes) handleTokenBurnSigned(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var burnReq struct {
		From      string `json:"from"`
		Amount    string `json:"amount"`
		Reason    string `json:"reason"`
		Nonce     uint64 `json:"nonce"`
		Signature string `json:"signature"`
	}

	if err := json.NewDecoder(req.Body).Decode(&burnReq); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	fromAddr, err := types.AddressFromString(burnReq.From)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid from address: %v", err), http.StatusBadRequest)
		return
	}
	amount, ok := new(big.Int).SetString(burnReq.Amount, 10)
	if !ok {
		http.Error(w, "Invalid amount", http.StatusBadRequest)
		return
	}
	var signed signing.SignedMessage
	if err := json.Unmarshal([]byte(burnReq.Signature), &signed); err != nil {
		http.Error(w, fmt.Sprintf("Invalid signature: %v", err), http.StatusBadRequest)
		return
	}

	receipt, err := r.oracle.GetNRNToken().BurnSigned(fromAddr, amount, burnReq.Reason, burnReq.Nonce, signed)
	if err != nil {
		http.Error(w, fmt.Sprintf("Burn failed: %v", err), http.StatusBadRequest)
		return
	}

	respondJSON(w, http.StatusOK, receipt)
}

// ========== Governance Handlers ==========

func (r *OracleRoutes) handleProposals(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:
		proposals := r.oracle.GetGovernanceSystem().ListProposals()
		respondJSON(w, http.StatusOK, proposals)

	case http.MethodPost:
		var submit struct {
			Proposal  governance.ProposalRequest `json:"proposal"`
			Deposit   string                     `json:"deposit"`
			Nonce     uint64                     `json:"nonce"`
			Signature string                     `json:"signature"`
		}
		if err := json.NewDecoder(req.Body).Decode(&submit); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}
		deposit, ok := new(big.Int).SetString(submit.Deposit, 10)
		if !ok || deposit.Sign() <= 0 {
			http.Error(w, "Invalid positive deposit", http.StatusBadRequest)
			return
		}
		if err := r.oracle.GetGovernanceSystem().ValidateProposal(&submit.Proposal, deposit); err != nil {
			http.Error(w, fmt.Sprintf("Invalid proposal: %v", err), http.StatusBadRequest)
			return
		}
		authorizationPayload, err := json.Marshal(struct {
			Proposal governance.ProposalRequest `json:"proposal"`
			Deposit  string                     `json:"deposit"`
			Nonce    uint64                     `json:"nonce"`
		}{submit.Proposal, submit.Deposit, submit.Nonce})
		if err != nil {
			http.Error(w, "Failed to encode proposal authorization", http.StatusInternalServerError)
			return
		}
		var signed signing.SignedMessage
		if err := json.Unmarshal([]byte(submit.Signature), &signed); err != nil {
			http.Error(w, fmt.Sprintf("Invalid signature: %v", err), http.StatusBadRequest)
			return
		}
		escrowHash := sha256.Sum256([]byte("knirv.oracle.governance.deposit"))
		escrow, _ := types.AddressFromBytes(escrowHash[:20])
		if err := r.oracle.GetNRNToken().LockGovernanceDeposit(submit.Proposal.Proposer, escrow, deposit, submit.Nonce, authorizationPayload, signed); err != nil {
			http.Error(w, fmt.Sprintf("Failed to lock proposal deposit: %v", err), http.StatusUnauthorized)
			return
		}

		proposal, err := r.oracle.GetGovernanceSystem().CreateProposal(&submit.Proposal, deposit)
		if err != nil {
			// Creation can still fail due to entropy or a concurrent parameter
			// update. Restore funds; the consumed nonce remains replay-safe.
			if refundErr := r.oracle.GetNRNToken().TransferBetween(escrow, submit.Proposal.Proposer, deposit); refundErr != nil {
				r.logger.Error("failed to refund governance deposit", zap.Error(refundErr), zap.String("proposer", submit.Proposal.Proposer.String()))
			}
			http.Error(w, fmt.Sprintf("Failed to create proposal: %v", err), http.StatusInternalServerError)
			return
		}

		respondJSON(w, http.StatusCreated, proposal)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (r *OracleRoutes) handleProposal(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	proposalID := req.URL.Path[len("/oracle/v3/governance/proposals/"):]
	proposal, err := r.oracle.GetGovernanceSystem().GetProposal(proposalID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Proposal not found: %v", err), http.StatusNotFound)
		return
	}

	respondJSON(w, http.StatusOK, proposal)
}

func (r *OracleRoutes) handleVote(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var voteReq struct {
		ProposalID string `json:"proposal_id"`
		Voter      string `json:"voter"`
		Option     int    `json:"option"`
		Signature  string `json:"signature"`
	}

	if err := json.NewDecoder(req.Body).Decode(&voteReq); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	voterAddr, err := types.AddressFromString(voteReq.Voter)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid voter address: %v", err), http.StatusBadRequest)
		return
	}

	govVoteReq := &governance.VoteRequest{
		ProposalID: voteReq.ProposalID,
		Voter:      voterAddr,
		Option:     governance.VoteOption(voteReq.Option),
	}

	var signed signing.SignedMessage
	if err := json.Unmarshal([]byte(voteReq.Signature), &signed); err != nil {
		http.Error(w, fmt.Sprintf("Invalid signature: %v", err), http.StatusBadRequest)
		return
	}
	vote, err := r.oracle.GetGovernanceSystem().CastVote(govVoteReq, signed)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to cast vote: %v", err), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, vote)
}

func (r *OracleRoutes) handleValidators(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	validators := r.oracle.GetGovernanceSystem().ListValidators()
	respondJSON(w, http.StatusOK, validators)
}

func (r *OracleRoutes) handleValidator(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	addressStr := req.URL.Path[len("/oracle/v3/governance/validators/"):]
	address, err := types.AddressFromString(addressStr)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid address: %v", err), http.StatusBadRequest)
		return
	}

	validator, err := r.oracle.GetGovernanceSystem().GetValidator(address)
	if err != nil {
		http.Error(w, fmt.Sprintf("Validator not found: %v", err), http.StatusNotFound)
		return
	}

	respondJSON(w, http.StatusOK, validator)
}

func (r *OracleRoutes) handleNetworkParams(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	params := r.oracle.GetGovernanceSystem().GetNetworkParameters()
	respondJSON(w, http.StatusOK, params)
}

// ========== Economics Handlers ==========

func (r *OracleRoutes) handleEconomicMetrics(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	metrics := r.oracle.GetEconomicsEngine().GetEconomicMetrics()
	respondJSON(w, http.StatusOK, metrics)
}

func (r *OracleRoutes) handleFees(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stats := r.oracle.GetEconomicsEngine().GetFeeCollector().GetFeeStats()
	respondJSON(w, http.StatusOK, stats)
}

func (r *OracleRoutes) handleRewards(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stats := r.oracle.GetEconomicsEngine().GetRewardManager().GetRewardStats()
	respondJSON(w, http.StatusOK, stats)
}

func (r *OracleRoutes) handleStaking(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stats := r.oracle.GetEconomicsEngine().GetStakingManager().GetStakingStats()
	respondJSON(w, http.StatusOK, stats)
}

func (r *OracleRoutes) handleBurns(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stats := r.oracle.GetEconomicsEngine().GetBurnTracker().GetBurnStats()
	respondJSON(w, http.StatusOK, stats)
}

// ========== Cross-Chain Handlers ==========

func (r *OracleRoutes) handleCrossChainTransfer(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var transferReq crosschain.TransferRequest
	if err := json.NewDecoder(req.Body).Decode(&transferReq); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	receipt, err := r.oracle.GetCrossChainRouter().InitiateTransfer(&transferReq)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to initiate transfer: %v", err), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, receipt)
}

func (r *OracleRoutes) handleGetTransfer(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	transferID := req.URL.Path[len("/oracle/v3/crosschain/transfer/"):]
	transfer, err := r.oracle.GetCrossChainRouter().GetTransfer(transferID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Transfer not found: %v", err), http.StatusNotFound)
		return
	}

	respondJSON(w, http.StatusOK, transfer)
}

func (r *OracleRoutes) handleBridges(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	bridges := r.oracle.GetBridgeManager().ListBridges()
	respondJSON(w, http.StatusOK, bridges)
}

// ========== IBC Handlers ==========

func (r *OracleRoutes) handleIBCChannels(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	channels := r.oracle.GetIBCHandler().ListChannels()
	respondJSON(w, http.StatusOK, channels)
}

func (r *OracleRoutes) handleIBCConnections(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	connections := r.oracle.GetIBCHandler().ListConnections()
	respondJSON(w, http.StatusOK, connections)
}

func (r *OracleRoutes) handleIBCClients(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	clients := r.oracle.GetIBCHandler().ListClients()
	respondJSON(w, http.StatusOK, clients)
}

// ========== Consensus Handlers ==========

func (r *OracleRoutes) handleConsensusStatus(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	info := r.oracle.GetConsensusEngine().GetInfo()
	respondJSON(w, http.StatusOK, info)
}

func (r *OracleRoutes) handleConsensusValidators(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	validators := r.oracle.GetConsensusEngine().GetValidators()
	respondJSON(w, http.StatusOK, validators)
}

// ========== P2P Handlers ==========

func (r *OracleRoutes) handleP2PStatus(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stats := r.oracle.GetP2PManager().GetStats()
	respondJSON(w, http.StatusOK, stats)
}

func (r *OracleRoutes) handleP2PPeers(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	peers := r.oracle.GetP2PManager().ListPeers()
	respondJSON(w, http.StatusOK, peers)
}

func (r *OracleRoutes) handleP2PPause(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.oracle.GetP2PManager().Pause(); err != nil {
		http.Error(w, fmt.Sprintf("Failed to pause P2P: %v", err), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status": "paused",
	})
}

func (r *OracleRoutes) handleP2PResume(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.oracle.GetP2PManager().Resume(); err != nil {
		http.Error(w, fmt.Sprintf("Failed to resume P2P: %v", err), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status": "resumed",
	})
}

func (r *OracleRoutes) handleSubmitRollup(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var rollupReq struct {
		ID          string                 `json:"id"`
		BatchRoot   string                 `json:"batch_root"`
		ChainID     string                 `json:"chain_id"`
		StartHeight uint64                 `json:"start_height"`
		EndHeight   uint64                 `json:"end_height"`
		BlockCount  int                    `json:"block_count"`
		TxCount     int                    `json:"tx_count"`
		Proposer    string                 `json:"proposer"`
		Signatures  []types.AuthorSig      `json:"signatures"`
		Metadata    map[string]interface{} `json:"metadata"`
	}
	if err := json.NewDecoder(req.Body).Decode(&rollupReq); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	if rollupReq.ID == "" || rollupReq.BatchRoot == "" {
		http.Error(w, "id and batch_root are required", http.StatusBadRequest)
		return
	}

	record := &types.RollupRecord{
		ID:          rollupReq.ID,
		BatchRoot:   rollupReq.BatchRoot,
		ChainID:     rollupReq.ChainID,
		StartHeight: rollupReq.StartHeight,
		EndHeight:   rollupReq.EndHeight,
		BlockCount:  rollupReq.BlockCount,
		TxCount:     rollupReq.TxCount,
		Proposer:    rollupReq.Proposer,
		Signatures:  rollupReq.Signatures,
		Status:      types.RollupStatusSubmitted,
		SubmittedAt: time.Now().UTC(),
		Metadata:    rollupReq.Metadata,
	}

	if err := r.oracle.SubmitRollup(record); err != nil {
		if strings.Contains(err.Error(), "unauthorized") {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		http.Error(w, fmt.Sprintf("failed to submit rollup: %v", err), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, record)
}

func (r *OracleRoutes) handleRollup(w http.ResponseWriter, req *http.Request) {
	path := strings.TrimPrefix(req.URL.Path, "/oracle/v3/rollups/")
	if path == "" {
		http.Error(w, "rollup id is required", http.StatusBadRequest)
		return
	}

	parts := strings.Split(strings.Trim(path, "/"), "/")
	id := parts[0]

	if len(parts) == 1 {
		if req.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		record, ok := r.oracle.GetRollup(id)
		if !ok {
			http.Error(w, "rollup not found", http.StatusNotFound)
			return
		}
		respondJSON(w, http.StatusOK, record)
		return
	}

	switch parts[1] {
	case "finalize":
		if req.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var finalizeReq struct {
			Signature types.AuthorSig `json:"signature"`
		}
		if err := json.NewDecoder(req.Body).Decode(&finalizeReq); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}
		record, err := r.oracle.FinalizeRollup(id, time.Now().UTC(), finalizeReq.Signature)
		if err != nil {
			if strings.Contains(err.Error(), "unauthorized") {
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}
			http.Error(w, fmt.Sprintf("failed to finalize rollup: %v", err), http.StatusNotFound)
			return
		}
		respondJSON(w, http.StatusOK, record)
	case "dispute":
		if req.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var disputeReq struct {
			Reason    string          `json:"reason"`
			Signature types.AuthorSig `json:"signature"`
		}
		if err := json.NewDecoder(req.Body).Decode(&disputeReq); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}
		record, err := r.oracle.DisputeRollup(id, disputeReq.Reason, time.Now().UTC(), disputeReq.Signature)
		if err != nil {
			if strings.Contains(err.Error(), "unauthorized") {
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}
			http.Error(w, fmt.Sprintf("failed to dispute rollup: %v", err), http.StatusNotFound)
			return
		}
		respondJSON(w, http.StatusOK, record)
	default:
		http.Error(w, "Not found", http.StatusNotFound)
	}
}

// ========== Health & Status Handlers ==========

func (r *OracleRoutes) handleHealth(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "healthy",
		"version": "3.0.0",
	})
}

func (r *OracleRoutes) handleStatus(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	status := r.oracle.GetStatus()
	respondJSON(w, http.StatusOK, status)
}

// ========== Helper Functions ==========

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// ========== Checkpoint pipeline (Phase 2) ==========

func (r *OracleRoutes) handleRegistryRegister(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var reg types.ChainRegistration
	if err := json.NewDecoder(req.Body).Decode(&reg); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "invalid registration"})
		return
	}
	if err := r.oracle.RegisterChain(&reg); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{"error": err.Error()})
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"status": "registered", "chain_id": reg.ChainID})
}

func (r *OracleRoutes) handleRegistryRotate(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var reg types.ChainRegistration
	if err := json.NewDecoder(req.Body).Decode(&reg); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "invalid rotation"})
		return
	}
	if err := r.oracle.RotateChain(&reg); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{"error": err.Error()})
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"status": "rotated", "chain_id": reg.ChainID})
}

func (r *OracleRoutes) handleGetRegistration(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	chainID := strings.TrimPrefix(req.URL.Path, "/oracle/v3/registry/")
	reg, ok := r.oracle.GetChainRegistration(chainID)
	if !ok {
		respondJSON(w, http.StatusNotFound, map[string]interface{}{"error": "chain not registered"})
		return
	}
	respondJSON(w, http.StatusOK, reg)
}

// handleSetVerificationKey enrolls a SNARK verification key + preferred proof
// system for a registered chain (merkle-math.md Phase 5). Accepts the key as a
// hex string; empty key with proof_system "hashchain-v0" clears back to optimistic.
func (r *OracleRoutes) handleSetVerificationKey(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		ChainID         string `json:"chain_id"`
		ProofSystem     string `json:"proof_system"`
		VerificationKey string `json:"verification_key"`
	}
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "invalid request"})
		return
	}
	if body.ChainID == "" {
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "chain_id is required"})
		return
	}
	vk, err := decodeHexBytes(body.VerificationKey)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "invalid verification_key (hex)"})
		return
	}
	if err := r.oracle.SetChainVerificationKey(body.ChainID, body.ProofSystem, vk); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{"error": err.Error()})
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":               "verification_key_set",
		"chain_id":             body.ChainID,
		"proof_system":         body.ProofSystem,
		"verification_key_len": len(vk),
	})
}

func (r *OracleRoutes) handleSubmitCheckpoint(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var cp types.Checkpoint
	if err := json.NewDecoder(req.Body).Decode(&cp); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "invalid checkpoint"})
		return
	}
	rec, err := r.oracle.SubmitCheckpoint(&cp)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{"error": err.Error()})
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":       "admitted",
		"mmr_position": rec.MMRPosition,
		"leaf_hash":    mmr.Hash(rec.LeafHash).Hex(),
		"final_by":     rec.FinalBy,
	})
}

func (r *OracleRoutes) handleCheckpointsByChain(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	chainID := strings.TrimPrefix(req.URL.Path, "/oracle/v3/checkpoints/")
	if chainID == "" {
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "missing chain id"})
		return
	}
	recs := r.oracle.GetCheckpointRecords(chainID)
	respondJSON(w, http.StatusOK, map[string]interface{}{"chain_id": chainID, "checkpoints": recs})
}

// ========== Phase 4: finality pipeline ==========

func (r *OracleRoutes) handleSubmitFinality(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var rec types.FinalityRecord
	if err := json.NewDecoder(req.Body).Decode(&rec); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}
	out, err := r.oracle.SubmitFinality(&rec)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{"error": err.Error()})
		return
	}
	respondJSON(w, http.StatusOK, out)
}

func (r *OracleRoutes) handleSubmitAttestation(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		ChainID         string                    `json:"chain_id"`
		StartHeight     uint64                    `json:"start_height"`
		TransitionProof []byte                    `json:"transition_proof"`
		ProofSystem     string                    `json:"proof_system"`
		Attestation     types.VerifierAttestation `json:"attestation"`
	}
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}
	if body.ChainID == "" {
		http.Error(w, "chain_id is required", http.StatusBadRequest)
		return
	}
	out, err := r.oracle.SubmitProofAttestation(body.ChainID, body.StartHeight, body.TransitionProof, body.ProofSystem, body.Attestation)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{"error": err.Error()})
		return
	}
	respondJSON(w, http.StatusOK, out)
}

func (r *OracleRoutes) handleRegisterVerifier(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	registrationToken := strings.TrimSpace(os.Getenv("KNIRV_VERIFIER_REGISTRATION_TOKEN"))
	suppliedToken := strings.TrimPrefix(req.Header.Get("Authorization"), "Bearer ")
	if registrationToken == "" || subtle.ConstantTimeCompare([]byte(suppliedToken), []byte(registrationToken)) != 1 {
		respondJSON(w, http.StatusForbidden, map[string]interface{}{"error": "verifier registration requires an explicitly configured operator token"})
		return
	}
	var body struct {
		VerifierID string `json:"verifier_id"`
		PublicKey  []byte `json:"public_key"`
	}
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}
	if err := r.oracle.RegisterVerifier(body.VerifierID, body.PublicKey); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{"error": err.Error()})
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"verifier_id": body.VerifierID, "registered": true})
}

func (r *OracleRoutes) handleMMRRoot(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	root := r.oracle.MMRRoot()
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"root": root.Hex(),
		"size": r.oracle.CheckpointCount(),
	})
}

func (r *OracleRoutes) handleMMRProof(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	rest := strings.TrimPrefix(req.URL.Path, "/oracle/v3/mmr/proof/")
	idx, err := strconv.ParseUint(rest, 10, 64)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "invalid leaf index"})
		return
	}
	proof, err := r.oracle.MMRProof(idx)
	if err != nil {
		respondJSON(w, http.StatusNotFound, map[string]interface{}{"error": err.Error()})
		return
	}
	respondJSON(w, http.StatusOK, proof)
}

// ExtractPathParam extracts a parameter from the URL path
func ExtractPathParam(path, prefix string) string {
	if len(path) <= len(prefix) {
		return ""
	}
	return path[len(prefix):]
}

// ParseIntParam parses an integer parameter from query string
func ParseIntParam(req *http.Request, param string, defaultValue int) int {
	str := req.URL.Query().Get(param)
	if str == "" {
		return defaultValue
	}
	val, err := strconv.Atoi(str)
	if err != nil {
		return defaultValue
	}
	return val
}
