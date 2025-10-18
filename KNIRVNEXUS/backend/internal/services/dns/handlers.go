package dns

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"

	"backend-server/internal/web/middleware"
)

// DNS Record Management Handlers

// CreateDNSRecordRequest represents a DNS record creation request
type CreateDNSRecordRequest struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	Value string `json:"value"`
	TTL   int    `json:"ttl"`
	Zone  string `json:"zone"`
}

// UpdateDNSRecordRequest represents a DNS record update request
type UpdateDNSRecordRequest struct {
	Value string `json:"value"`
	TTL   int    `json:"ttl"`
}

// HandleListDNSRecords handles DNS record listing requests
func (ds *DynamicDNSService) HandleListDNSRecords(w http.ResponseWriter, r *http.Request) {
	// Skip auth check for development
	// authCtx := middleware.GetAuthContext(r)
	// if authCtx == nil {
	//	writeError(w, http.StatusUnauthorized, "Authentication required")
	//	return
	// }

	// Parse query parameters for filtering
	zone := r.URL.Query().Get("zone")
	recordType := r.URL.Query().Get("type")

	// This would retrieve DNS records from the service
	// For now, return a placeholder response
	records := []map[string]interface{}{
		{
			"id":    "record-1",
			"name":  "example.knirv.com",
			"type":  "A",
			"value": "192.168.1.1",
			"ttl":   300,
			"zone":  "knirv.com",
		},
	}

	// Apply filters if provided
	if zone != "" || recordType != "" {
		// Filter logic would go here
		_ = zone
		_ = recordType
	}

	writeJSON(w, http.StatusOK, records)
}

// HandleCreateDNSRecord handles DNS record creation requests
func (ds *DynamicDNSService) HandleCreateDNSRecord(w http.ResponseWriter, r *http.Request) {
	authCtx := middleware.GetAuthContext(r)
	if authCtx == nil {
		writeError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	var req CreateDNSRecordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Name == "" || req.Type == "" || req.Value == "" {
		writeError(w, http.StatusBadRequest, "Name, type, and value are required")
		return
	}

	// Set default TTL if not provided
	if req.TTL == 0 {
		req.TTL = 300
	}

	// This would create the DNS record in the service
	// For now, return a placeholder response
	record := map[string]interface{}{
		"id":    "record-new",
		"name":  req.Name,
		"type":  req.Type,
		"value": req.Value,
		"ttl":   req.TTL,
		"zone":  req.Zone,
	}

	writeJSON(w, http.StatusCreated, record)
}

// HandleGetDNSRecord handles individual DNS record retrieval requests
func (ds *DynamicDNSService) HandleGetDNSRecord(w http.ResponseWriter, r *http.Request) {
	authCtx := middleware.GetAuthContext(r)
	if authCtx == nil {
		writeError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	vars := mux.Vars(r)
	recordID := vars["id"]

	// This would retrieve the specific DNS record
	// For now, return a placeholder response
	record := map[string]interface{}{
		"id":    recordID,
		"name":  "example.knirv.com",
		"type":  "A",
		"value": "192.168.1.1",
		"ttl":   300,
		"zone":  "knirv.com",
	}

	writeJSON(w, http.StatusOK, record)
}

// HandleUpdateDNSRecord handles DNS record update requests
func (ds *DynamicDNSService) HandleUpdateDNSRecord(w http.ResponseWriter, r *http.Request) {
	authCtx := middleware.GetAuthContext(r)
	if authCtx == nil {
		writeError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	vars := mux.Vars(r)
	recordID := vars["id"]

	var req UpdateDNSRecordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Value == "" {
		writeError(w, http.StatusBadRequest, "Value is required")
		return
	}

	// This would update the DNS record in the service
	// For now, return a placeholder response
	record := map[string]interface{}{
		"id":    recordID,
		"name":  "example.knirv.com",
		"type":  "A",
		"value": req.Value,
		"ttl":   req.TTL,
		"zone":  "knirv.com",
	}

	writeJSON(w, http.StatusOK, record)
}

// HandleDeleteDNSRecord handles DNS record deletion requests
func (ds *DynamicDNSService) HandleDeleteDNSRecord(w http.ResponseWriter, r *http.Request) {
	authCtx := middleware.GetAuthContext(r)
	if authCtx == nil {
		writeError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	vars := mux.Vars(r)
	recordID := vars["id"]

	// This would delete the DNS record from the service
	// For now, just return success
	_ = recordID

	writeJSON(w, http.StatusOK, map[string]string{"message": "DNS record deleted successfully"})
}

// DNS Zone Management Handlers

// HandleListDNSZones handles DNS zone listing requests
func (ds *DynamicDNSService) HandleListDNSZones(w http.ResponseWriter, r *http.Request) {
	// Skip auth check for development
	// authCtx := middleware.GetAuthContext(r)
	// if authCtx == nil {
	//	writeError(w, http.StatusUnauthorized, "Authentication required")
	//	return
	// }

	// This would retrieve DNS zones from the service
	// For now, return a placeholder response
	zones := []map[string]interface{}{
		{
			"id":   "zone-1",
			"name": "knirv.com",
			"type": "primary",
		},
		{
			"id":   "zone-2",
			"name": "test.knirv.com",
			"type": "secondary",
		},
	}

	writeJSON(w, http.StatusOK, zones)
}

// System Status Handlers

// HandleGetDNSStatus handles DNS service status requests
func (ds *DynamicDNSService) HandleGetDNSStatus(w http.ResponseWriter, r *http.Request) {
	// This can be public for monitoring
	status := map[string]interface{}{
		"service":   "dynamic-dns",
		"status":    "running",
		"zones":     2,                      // TODO: Get actual count
		"records":   10,                     // TODO: Get actual count
		"timestamp": "2024-01-01T00:00:00Z", // TODO: Use actual timestamp
	}

	writeJSON(w, http.StatusOK, status)
}

// Helper functions

// writeJSON writes a JSON response
func writeJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

// writeError writes an error response
func writeError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
