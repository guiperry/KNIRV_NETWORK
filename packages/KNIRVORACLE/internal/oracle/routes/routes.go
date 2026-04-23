package routes

import (
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/knirvcorp/knirvoracle/internal/oracle"
	"github.com/knirvcorp/knirvoracle/internal/oracle/crosschain"
	"github.com/knirvcorp/knirvoracle/internal/oracle/governance"
	"github.com/knirvcorp/knirvoracle/internal/oracle/types"
	"go.uber.org/zap"
)

// OracleRoutes provides HTTP handlers for oracle endpoints
type OracleRoutes struct {
	oracle *oracle.Oracle
	logger *zap.Logger
}

// NewOracleRoutes creates a new oracle routes handler
func NewOracleRoutes(oracleInstance *oracle.Oracle, logger *zap.Logger) *OracleRoutes {
	return &OracleRoutes{
		oracle: oracleInstance,
		logger: logger,
	}
}

// RegisterRoutes registers all oracle routes with an HTTP mux
// Compatible with standard http.ServeMux or any router
func (r *OracleRoutes) RegisterRoutes(mux *http.ServeMux) {
	// Token endpoints
	mux.HandleFunc("/oracle/v3/token/info", r.handleTokenInfo)
	mux.HandleFunc("/oracle/v3/token/balance/", r.handleTokenBalance)
	mux.HandleFunc("/oracle/v3/token/mint", r.handleTokenMint)
	mux.HandleFunc("/oracle/v3/token/transfer", r.handleTokenTransfer)
	mux.HandleFunc("/oracle/v3/token/burn", r.handleTokenBurn)

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

	// Installation endpoints
	mux.HandleFunc("/oracle/v3/install/wallet", r.handleInstallWallet)
	mux.HandleFunc("/oracle/v3/install/dve_uri", r.handleInstallDVEURI)

	// Health and status
	mux.HandleFunc("/oracle/v3/health", r.handleHealth)
	mux.HandleFunc("/oracle/v3/status", r.handleStatus)
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

func (r *OracleRoutes) handleTokenMint(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var mintReq struct {
		To     string `json:"to"`
		Amount string `json:"amount"`
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
	if !ok {
		http.Error(w, "Invalid amount", http.StatusBadRequest)
		return
	}

	receipt, err := r.oracle.GetNRNToken().Mint(toAddr, amount)
	if err != nil {
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

	var transferReq struct {
		FromPrivateKey string `json:"from_private_key"`
		To             string `json:"to"`
		Amount         string `json:"amount"`
	}

	if err := json.NewDecoder(req.Body).Decode(&transferReq); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
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

	receipt, err := r.oracle.GetNRNToken().Transfer(transferReq.FromPrivateKey, toAddr, amount)
	if err != nil {
		http.Error(w, fmt.Sprintf("Transfer failed: %v", err), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, receipt)
}

func (r *OracleRoutes) handleTokenBurn(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var burnReq struct {
		PrivateKey string `json:"private_key"`
		Amount     string `json:"amount"`
		Reason     string `json:"reason"`
	}

	if err := json.NewDecoder(req.Body).Decode(&burnReq); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	amount, ok := new(big.Int).SetString(burnReq.Amount, 10)
	if !ok {
		http.Error(w, "Invalid amount", http.StatusBadRequest)
		return
	}

	receipt, err := r.oracle.GetNRNToken().Burn(burnReq.PrivateKey, amount, burnReq.Reason)
	if err != nil {
		http.Error(w, fmt.Sprintf("Burn failed: %v", err), http.StatusInternalServerError)
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
		var propReq governance.ProposalRequest
		if err := json.NewDecoder(req.Body).Decode(&propReq); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}

		// Parse deposit from query param
		depositStr := req.URL.Query().Get("deposit")
		deposit, ok := new(big.Int).SetString(depositStr, 10)
		if !ok {
			deposit = big.NewInt(100000000) // Default deposit
		}

		proposal, err := r.oracle.GetGovernanceSystem().CreateProposal(&propReq, deposit)
		if err != nil {
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
		PrivateKey string `json:"private_key"`
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

	vote, err := r.oracle.GetGovernanceSystem().CastVote(govVoteReq, voteReq.PrivateKey)
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
		Status:      types.RollupStatusSubmitted,
		SubmittedAt: time.Now().UTC(),
		Metadata:    rollupReq.Metadata,
	}

	if err := r.oracle.SubmitRollup(record); err != nil {
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
		record, err := r.oracle.FinalizeRollup(id, time.Now().UTC())
		if err != nil {
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
			Reason string `json:"reason"`
		}
		if err := json.NewDecoder(req.Body).Decode(&disputeReq); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}
		record, err := r.oracle.DisputeRollup(id, disputeReq.Reason, time.Now().UTC())
		if err != nil {
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
