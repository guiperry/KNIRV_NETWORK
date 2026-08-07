package routes

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

func (r *OracleRoutes) handleInstallDVEURI(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var dveReq struct {
		DVEID      string `json:"dve_id"`
		DesiredURI string `json:"desired_uri"`
	}

	if err := json.NewDecoder(req.Body).Decode(&dveReq); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	if dveReq.DVEID == "" {
		http.Error(w, "dve_id is required", http.StatusBadRequest)
		return
	}

	dveURI := generateDVEURI(dveReq.DVEID, dveReq.DesiredURI)

	response := map[string]interface{}{
		"dve_id":   dveReq.DVEID,
		"full_uri": dveURI,
		"hash":     dveURI[8:20],
		"created":  time.Now().Unix(),
	}

	respondJSON(w, http.StatusOK, response)
}

func generateDVEURI(dveID, desiredID string) string {
	data := fmt.Sprintf("dve:%s:%s:%d", dveID, desiredID, time.Now().UnixNano())
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(data)))
	return "knirv://" + hash[:10] + "/" + dveID
}
