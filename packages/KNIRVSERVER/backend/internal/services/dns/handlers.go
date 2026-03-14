package dns

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"

	"backend_server/internal/web/middleware"
	"backend_server/internal/utils/cloudflare"
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
	zoneName := r.URL.Query().Get("zone")
	recordType := r.URL.Query().Get("type")

	// For development mode, return mock data
	if ds.config.CloudFlareAPIToken == "dev-token" {
		allRecords := []map[string]interface{}{
			{
				"id":       "dev-record-1",
				"name":     "example",
				"type":     "A",
				"value":    "192.0.2.1",
				"ttl":      300,
				"zone":     "knirv.com",
				"zone_id":  "dev-zone-1",
				"proxied":  false,
				"priority": 0,
			},
			{
				"id":       "dev-record-2",
				"name":     "www",
				"type":     "CNAME",
				"value":    "example.knirv.com",
				"ttl":      300,
				"zone":     "knirv.com",
				"zone_id":  "dev-zone-1",
				"proxied":  false,
				"priority": 0,
			},
		}
		writeJSON(w, http.StatusOK, allRecords)
		return
	}

	// Get zone information
	var zoneID string
	if zoneName != "" {
		zone, err := ds.dnsManager.GetZoneByName(zoneName)
		if err != nil {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("Failed to get zone: %v", err))
			return
		}
		zoneID = zone.ID
	} else {
		// If no zone specified, get all zones and their records
		zones, err := ds.dnsManager.GetZones()
		if err != nil {
			// Fallback to empty list on error
			writeJSON(w, http.StatusOK, []map[string]interface{}{})
			return
		}

		var allRecords []map[string]interface{}
		for _, zone := range zones {
			records, err := ds.dnsManager.GetDNSRecords(zone.ID)
			if err != nil {
				continue // Skip zones with errors
			}

			for _, record := range records {
				if recordType != "" && record.Type != recordType {
					continue
				}

				recordMap := map[string]interface{}{
					"id":       record.ID,
					"name":     record.Name,
					"type":     record.Type,
					"value":    record.Content,
					"ttl":      record.TTL,
					"zone":     zone.Name,
					"zone_id":  zone.ID,
					"proxied":  record.Proxied,
					"priority": record.Priority,
				}
				allRecords = append(allRecords, recordMap)
			}
		}

		writeJSON(w, http.StatusOK, allRecords)
		return
	}

	// Get DNS records for the specified zone
	records, err := ds.dnsManager.GetDNSRecords(zoneID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get DNS records: %v", err))
		return
	}

	// Convert to response format and apply filters
	var responseRecords []map[string]interface{}
	for _, record := range records {
		if recordType != "" && record.Type != recordType {
			continue
		}

		recordMap := map[string]interface{}{
			"id":       record.ID,
			"name":     record.Name,
			"type":     record.Type,
			"value":    record.Content,
			"ttl":      record.TTL,
			"zone":     zoneName,
			"zone_id":  zoneID,
			"proxied":  record.Proxied,
			"priority": record.Priority,
		}
		responseRecords = append(responseRecords, recordMap)
	}

	writeJSON(w, http.StatusOK, responseRecords)
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

	if req.Name == "" || req.Type == "" || req.Value == "" || req.Zone == "" {
		writeError(w, http.StatusBadRequest, "Name, type, value, and zone are required")
		return
	}

	// Set default TTL if not provided
	if req.TTL == 0 {
		req.TTL = 300
	}

	// Get zone information
	zone, err := ds.dnsManager.GetZoneByName(req.Zone)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("Failed to get zone: %v", err))
		return
	}

	// Create DNS record
	cfRecord := cloudflare.DNSRecord{
		Type:    req.Type,
		Name:    req.Name,
		Content: req.Value,
		TTL:     req.TTL,
		Proxied: false, // Default to not proxied
	}

	createdRecord, err := ds.dnsManager.CreateDNSRecord(zone.ID, cfRecord)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to create DNS record: %v", err))
		return
	}

	// Return created record in response format
	record := map[string]interface{}{
		"id":       createdRecord.ID,
		"name":     createdRecord.Name,
		"type":     createdRecord.Type,
		"value":    createdRecord.Content,
		"ttl":      createdRecord.TTL,
		"zone":     req.Zone,
		"zone_id":  zone.ID,
		"proxied":  createdRecord.Proxied,
		"priority": createdRecord.Priority,
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
	zoneName := r.URL.Query().Get("zone")

	if recordID == "" {
		writeError(w, http.StatusBadRequest, "Record ID is required")
		return
	}

	if zoneName == "" {
		writeError(w, http.StatusBadRequest, "Zone parameter is required")
		return
	}

	// Get zone information
	zone, err := ds.dnsManager.GetZoneByName(zoneName)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("Failed to get zone: %v", err))
		return
	}

	// Get all records for the zone and find the specific record
	records, err := ds.dnsManager.GetDNSRecords(zone.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get DNS records: %v", err))
		return
	}

	// Find the record by ID
	var foundRecord *cloudflare.DNSRecord
	for _, record := range records {
		if record.ID == recordID {
			foundRecord = &record
			break
		}
	}

	if foundRecord == nil {
		writeError(w, http.StatusNotFound, "DNS record not found")
		return
	}

	// Return record in response format
	record := map[string]interface{}{
		"id":       foundRecord.ID,
		"name":     foundRecord.Name,
		"type":     foundRecord.Type,
		"value":    foundRecord.Content,
		"ttl":      foundRecord.TTL,
		"zone":     zoneName,
		"zone_id":  zone.ID,
		"proxied":  foundRecord.Proxied,
		"priority": foundRecord.Priority,
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
	zoneName := r.URL.Query().Get("zone")

	if recordID == "" {
		writeError(w, http.StatusBadRequest, "Record ID is required")
		return
	}

	if zoneName == "" {
		writeError(w, http.StatusBadRequest, "Zone parameter is required")
		return
	}

	var req UpdateDNSRecordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Value == "" {
		writeError(w, http.StatusBadRequest, "Value is required")
		return
	}

	// Get zone information
	zone, err := ds.dnsManager.GetZoneByName(zoneName)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("Failed to get zone: %v", err))
		return
	}

	// Get current record to preserve other fields
	records, err := ds.dnsManager.GetDNSRecords(zone.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get DNS records: %v", err))
		return
	}

	var currentRecord *cloudflare.DNSRecord
	for _, record := range records {
		if record.ID == recordID {
			currentRecord = &record
			break
		}
	}

	if currentRecord == nil {
		writeError(w, http.StatusNotFound, "DNS record not found")
		return
	}

	// Update the record
	updatedRecord := *currentRecord
	updatedRecord.Content = req.Value
	if req.TTL > 0 {
		updatedRecord.TTL = req.TTL
	}

	result, err := ds.dnsManager.UpdateDNSRecord(zone.ID, recordID, updatedRecord)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to update DNS record: %v", err))
		return
	}

	// Return updated record in response format
	record := map[string]interface{}{
		"id":       result.ID,
		"name":     result.Name,
		"type":     result.Type,
		"value":    result.Content,
		"ttl":      result.TTL,
		"zone":     zoneName,
		"zone_id":  zone.ID,
		"proxied":  result.Proxied,
		"priority": result.Priority,
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
	zoneName := r.URL.Query().Get("zone")

	if recordID == "" {
		writeError(w, http.StatusBadRequest, "Record ID is required")
		return
	}

	if zoneName == "" {
		writeError(w, http.StatusBadRequest, "Zone parameter is required")
		return
	}

	// Get zone information
	zone, err := ds.dnsManager.GetZoneByName(zoneName)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("Failed to get zone: %v", err))
		return
	}

	// Delete the DNS record
	err = ds.dnsManager.DeleteDNSRecord(zone.ID, recordID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to delete DNS record: %v", err))
		return
	}

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

	// For development mode, return mock data
	if ds.config.CloudFlareAPIToken == "dev-token" {
		zones := []map[string]interface{}{
			{
				"id":     "dev-zone-1",
				"name":   "knirv.com",
				"status": "active",
				"type":   "primary",
			},
			{
				"id":     "dev-zone-2",
				"name":   "knirv.dev",
				"status": "active",
				"type":   "primary",
			},
		}
		writeJSON(w, http.StatusOK, zones)
		return
	}

	// Get zones from Cloudflare
	cfZones, err := ds.dnsManager.GetZones()
	if err != nil {
		// Fallback to mock data on error
		zones := []map[string]interface{}{}
		writeJSON(w, http.StatusOK, zones)
		return
	}

	// Convert to response format
	var zones []map[string]interface{}
	for _, zone := range cfZones {
		zoneMap := map[string]interface{}{
			"id":     zone.ID,
			"name":   zone.Name,
			"status": zone.Status,
			"type":   "primary", // Cloudflare zones are typically primary
		}
		zones = append(zones, zoneMap)
	}

	writeJSON(w, http.StatusOK, zones)
}

// System Status Handlers

// HandleGetDNSStatus handles DNS service status requests
func (ds *DynamicDNSService) HandleGetDNSStatus(w http.ResponseWriter, r *http.Request) {
	// This can be public for monitoring

	// For development mode, return mock status
	if ds.config.CloudFlareAPIToken == "dev-token" {
		status := map[string]interface{}{
			"status":        "operational",
			"zones_count":   2,
			"records_count": 10,
			"mode":          "development",
			"last_update":   ds.lastUpdate.Format("2006-01-02T15:04:05Z07:00"),
			"uptime":        "24h",
		}
		writeJSON(w, http.StatusOK, status)
		return
	}

	// Get actual zone count
	zones, err := ds.dnsManager.GetZones()
	zoneCount := 0
	recordCount := 0
	if err == nil {
		zoneCount = len(zones)

		// Get record count across all zones
		for _, zone := range zones {
			records, err := ds.dnsManager.GetDNSRecords(zone.ID)
			if err == nil {
				recordCount += len(records)
			}
		}
	}

	// Get service status
	serviceStatus := ds.GetStatus()

	status := map[string]interface{}{
		"service":      "dynamic-dns",
		"status":       "running",
		"zones":        zoneCount,
		"records":      recordCount,
		"current_ip":   serviceStatus["current_ip"],
		"last_update":  serviceStatus["last_update"],
		"update_count": serviceStatus["update_count"],
		"error_count":  serviceStatus["error_count"],
		"timestamp":    serviceStatus["last_update"], // Use last update as timestamp
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
