package routes

import (
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"strings"
	"time"

	knirvsigning "github.com/KNIRV/KNIRV_NETWORK/KNIRVSDK/go/signing"
	"github.com/knirvcorp/knirvoracle/internal/oracle/token"
	"github.com/knirvcorp/knirvoracle/internal/oracle/types"
	"go.uber.org/zap"
)

type walletRegisterResponse struct {
	Address           string             `json:"address"`
	DVEID             string             `json:"dve_id,omitempty"`
	InitialFunding    string             `json:"initial_funding,omitempty"`
	Balance           string             `json:"balance,omitempty"`
	FundingReceipt    *token.MintReceipt `json:"funding_receipt,omitempty"`
	WalletServerOwner string             `json:"wallet_server_owner,omitempty"`
}

type faucetRequest struct {
	Address string `json:"address"`
	Amount  uint64 `json:"amount"`
}

func (r *OracleRoutes) handleInstallWallet(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	http.Error(w, "Oracle wallet generation is disabled; create and custody wallets in KNIRVCONTROLLER, then call /oracle/v3/wallet/register", http.StatusGone)
}

func (r *OracleRoutes) handleGenerateWallet(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet && req.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	http.Error(w, "Oracle wallet generation is disabled; create and custody wallets in KNIRVCONTROLLER", http.StatusGone)
}

func (r *OracleRoutes) handleRegisterWallet(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var walletReq struct {
		Address   string `json:"address"`
		OwnerID   string `json:"owner_id"`
		Signature string `json:"signature"`
	}

	if err := json.NewDecoder(req.Body).Decode(&walletReq); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}
	if walletReq.Address == "" {
		http.Error(w, "Address is required", http.StatusBadRequest)
		return
	}

	address, err := types.AddressFromString(walletReq.Address)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid address: %v", err), http.StatusBadRequest)
		return
	}

	if err := verifyWalletRegistrationSignature(address, walletReq.OwnerID, walletReq.Signature, r.oracle.ChainID()); err != nil {
		http.Error(w, fmt.Sprintf("Invalid signature: %v", err), http.StatusBadRequest)
		return
	}

	response, err := r.createWalletRegistrationResponse(address, walletReq.OwnerID)
	if err != nil {
		r.logger.Error("Failed to register wallet", zap.Error(err))
		http.Error(w, fmt.Sprintf("Failed to register wallet: %v", err), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusCreated, response)
}

func (r *OracleRoutes) handleWalletStatus(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":           "active",
		"enabled":          r.oracle.WalletServerEnabled(),
		"owner":            "KNIRVORACLE",
		"initial_funding":  r.oracle.GetWalletInitialFunding().String(),
		"treasury_address": r.oracle.GetNRNToken().Owner().String(),
	})
}

func (r *OracleRoutes) handleLegacyWalletBalance(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	addressStr := req.URL.Path[len("/balance/"):]
	address, err := types.AddressFromString(addressStr)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid address: %v", err), http.StatusBadRequest)
		return
	}

	balance := r.oracle.GetNRNToken().GetBalance(address)
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"address": address.String(),
		"balance": balance.String(),
		"owner":   "KNIRVORACLE",
	})
}

func (r *OracleRoutes) handleCreateTransaction(w http.ResponseWriter, req *http.Request) {
	r.handleSendSignedTransaction(w, req)
}

func (r *OracleRoutes) handleSendSignedTransaction(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	http.Error(w, "private-key transaction submission is disabled; use /oracle/v3/token/transfer/signed with KNIRVCONTROLLER", http.StatusGone)
}

func (r *OracleRoutes) handleTestFaucet(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	demo := strings.TrimSpace(os.Getenv("KNIRV_ENABLE_DEMO"))
	if demo != "1" && !strings.EqualFold(demo, "true") {
		http.Error(w, "test faucet is disabled; set KNIRV_ENABLE_DEMO=true explicitly", http.StatusNotFound)
		return
	}

	var fundReq faucetRequest
	if err := json.NewDecoder(req.Body).Decode(&fundReq); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if fundReq.Address == "" || fundReq.Amount == 0 {
		http.Error(w, "Address and amount are required", http.StatusBadRequest)
		return
	}

	address, err := types.AddressFromString(fundReq.Address)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid address: %v", err), http.StatusBadRequest)
		return
	}

	receipt, err := r.oracle.FundAddress(address, new(big.Int).SetUint64(fundReq.Amount), "oracle wallet faucet")
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fund wallet: %v", err), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusCreated, receipt)
}

func (r *OracleRoutes) createWalletRegistrationResponse(address types.Address, ownerID string) (*walletRegisterResponse, error) {
	response := &walletRegisterResponse{
		Address:           address.String(),
		Balance:           r.oracle.GetNRNToken().GetBalance(address).String(),
		WalletServerOwner: "KNIRVORACLE",
	}
	if ownerID != "" {
		response.DVEID = ownerID
	}

	if !r.oracle.WalletServerEnabled() {
		return response, nil
	}

	if !r.oracle.GetNRNToken().RegisterAddress(address) {
		return response, nil
	}

	initialFunding := r.oracle.GetWalletInitialFunding()
	if initialFunding.Sign() <= 0 {
		return response, nil
	}

	receipt, err := r.oracle.FundAddress(address, initialFunding, "initial registered wallet funding")
	if err != nil {
		return nil, err
	}

	response.InitialFunding = initialFunding.String()
	response.Balance = receipt.NewBalance
	response.FundingReceipt = receipt
	return response, nil
}

func verifyWalletRegistrationSignature(address types.Address, ownerID, signatureHex, chainID string) error {
	if signatureHex == "" {
		return fmt.Errorf("signature is required")
	}

	var signed knirvsigning.SignedMessage
	if err := json.Unmarshal([]byte(signatureHex), &signed); err != nil {
		return fmt.Errorf("decode canonical signed envelope: %w", err)
	}
	message := []byte(walletRegistrationMessage(address, ownerID))
	if err := knirvsigning.VerifyMessagePayload(signed, "knirv.oracle", "wallet-registration", chainID, message, time.Now()); err != nil {
		return err
	}
	if signed.Address != address.String() {
		return fmt.Errorf("signature does not match address")
	}
	return nil
}

func walletRegistrationMessage(address types.Address, ownerID string) string {
	return fmt.Sprintf("register:%s:%s", address.String(), ownerID)
}
