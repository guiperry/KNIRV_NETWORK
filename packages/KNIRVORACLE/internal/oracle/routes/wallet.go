package routes

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/knirvcorp/knirvoracle/internal/oracle/crypto"
	"go.uber.org/zap"
)

func (r *OracleRoutes) handleInstallWallet(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var walletReq struct {
		OwnerID string `json:"owner_id"`
	}

	if err := json.NewDecoder(req.Body).Decode(&walletReq); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	kp, err := crypto.GenerateKeyPair()
	if err != nil {
		r.logger.Error("Failed to generate key pair", zap.Error(err))
		http.Error(w, fmt.Sprintf("Failed to generate wallet: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"private_key":    kp.PrivateKeyHex(),
		"public_key":     kp.PublicKeyHex(),
		"wallet_address": kp.Address.String(),
	}

	if walletReq.OwnerID != "" {
		response["dve_id"] = walletReq.OwnerID
	}

	respondJSON(w, http.StatusOK, response)
}