package cloudflare

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// DNSManager handles CloudFlare DNS operations
type DNSManager struct {
	apiToken string
	client   *http.Client
	baseURL  string
}

// DNSRecord represents a CloudFlare DNS record
type DNSRecord struct {
	ID       string                 `json:"id,omitempty"`
	Type     string                 `json:"type"`
	Name     string                 `json:"name"`
	Content  string                 `json:"content"`
	TTL      int                    `json:"ttl"`
	Priority int                    `json:"priority,omitempty"`
	Proxied  bool                   `json:"proxied"`
	ZoneID   string                 `json:"zone_id,omitempty"`
	ZoneName string                 `json:"zone_name,omitempty"`
	Meta     map[string]interface{} `json:"meta,omitempty"`
}

// Zone represents a CloudFlare zone
type Zone struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

// CloudFlareResponse represents the standard CloudFlare API response
type CloudFlareResponse struct {
	Success  bool                     `json:"success"`
	Errors   []map[string]interface{} `json:"errors"`
	Messages []string                 `json:"messages"`
	Result   interface{}              `json:"result"`
}

// NewDNSManager creates a new CloudFlare DNS manager
func NewDNSManager(apiToken string) *DNSManager {
	return &DNSManager{
		apiToken: apiToken,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: "https://api.cloudflare.com/client/v4",
	}
}

// GetZones retrieves all zones for the account
func (dm *DNSManager) GetZones() ([]Zone, error) {
	url := fmt.Sprintf("%s/zones", dm.baseURL)
	
	resp, err := dm.makeRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	
	var zones []Zone
	if result, ok := resp.Result.([]interface{}); ok {
		for _, item := range result {
			if zoneData, ok := item.(map[string]interface{}); ok {
				zone := Zone{
					ID:     zoneData["id"].(string),
					Name:   zoneData["name"].(string),
					Status: zoneData["status"].(string),
				}
				zones = append(zones, zone)
			}
		}
	}
	
	return zones, nil
}

// GetZoneByName retrieves a zone by its name
func (dm *DNSManager) GetZoneByName(zoneName string) (*Zone, error) {
	zones, err := dm.GetZones()
	if err != nil {
		return nil, err
	}
	
	for _, zone := range zones {
		if zone.Name == zoneName {
			return &zone, nil
		}
	}
	
	return nil, fmt.Errorf("zone %s not found", zoneName)
}

// GetDNSRecords retrieves all DNS records for a zone
func (dm *DNSManager) GetDNSRecords(zoneID string) ([]DNSRecord, error) {
	url := fmt.Sprintf("%s/zones/%s/dns_records", dm.baseURL, zoneID)
	
	resp, err := dm.makeRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	
	var records []DNSRecord
	if result, ok := resp.Result.([]interface{}); ok {
		for _, item := range result {
			if recordData, ok := item.(map[string]interface{}); ok {
				record := DNSRecord{
					ID:      recordData["id"].(string),
					Type:    recordData["type"].(string),
					Name:    recordData["name"].(string),
					Content: recordData["content"].(string),
					TTL:     int(recordData["ttl"].(float64)),
					Proxied: recordData["proxied"].(bool),
					ZoneID:  recordData["zone_id"].(string),
				}
				
				if priority, exists := recordData["priority"]; exists && priority != nil {
					record.Priority = int(priority.(float64))
				}
				
				records = append(records, record)
			}
		}
	}
	
	return records, nil
}

// GetDNSRecord retrieves a specific DNS record by name and type
func (dm *DNSManager) GetDNSRecord(zoneID, recordName, recordType string) (*DNSRecord, error) {
	records, err := dm.GetDNSRecords(zoneID)
	if err != nil {
		return nil, err
	}
	
	for _, record := range records {
		if record.Name == recordName && record.Type == recordType {
			return &record, nil
		}
	}
	
	return nil, fmt.Errorf("DNS record %s (%s) not found", recordName, recordType)
}

// CreateDNSRecord creates a new DNS record
func (dm *DNSManager) CreateDNSRecord(zoneID string, record DNSRecord) (*DNSRecord, error) {
	url := fmt.Sprintf("%s/zones/%s/dns_records", dm.baseURL, zoneID)
	
	resp, err := dm.makeRequest("POST", url, record)
	if err != nil {
		return nil, err
	}
	
	if resultData, ok := resp.Result.(map[string]interface{}); ok {
		createdRecord := DNSRecord{
			ID:      resultData["id"].(string),
			Type:    resultData["type"].(string),
			Name:    resultData["name"].(string),
			Content: resultData["content"].(string),
			TTL:     int(resultData["ttl"].(float64)),
			Proxied: resultData["proxied"].(bool),
			ZoneID:  resultData["zone_id"].(string),
		}
		
		if priority, exists := resultData["priority"]; exists && priority != nil {
			createdRecord.Priority = int(priority.(float64))
		}
		
		return &createdRecord, nil
	}
	
	return nil, fmt.Errorf("failed to parse created record response")
}

// UpdateDNSRecord updates an existing DNS record
func (dm *DNSManager) UpdateDNSRecord(zoneID, recordID string, record DNSRecord) (*DNSRecord, error) {
	url := fmt.Sprintf("%s/zones/%s/dns_records/%s", dm.baseURL, zoneID, recordID)
	
	resp, err := dm.makeRequest("PUT", url, record)
	if err != nil {
		return nil, err
	}
	
	if resultData, ok := resp.Result.(map[string]interface{}); ok {
		updatedRecord := DNSRecord{
			ID:      resultData["id"].(string),
			Type:    resultData["type"].(string),
			Name:    resultData["name"].(string),
			Content: resultData["content"].(string),
			TTL:     int(resultData["ttl"].(float64)),
			Proxied: resultData["proxied"].(bool),
			ZoneID:  resultData["zone_id"].(string),
		}
		
		if priority, exists := resultData["priority"]; exists && priority != nil {
			updatedRecord.Priority = int(priority.(float64))
		}
		
		return &updatedRecord, nil
	}
	
	return nil, fmt.Errorf("failed to parse updated record response")
}

// DeleteDNSRecord deletes a DNS record
func (dm *DNSManager) DeleteDNSRecord(zoneID, recordID string) error {
	url := fmt.Sprintf("%s/zones/%s/dns_records/%s", dm.baseURL, zoneID, recordID)
	
	_, err := dm.makeRequest("DELETE", url, nil)
	return err
}

// UpdateOrCreateDNSRecord updates a DNS record if it exists, creates it if it doesn't
func (dm *DNSManager) UpdateOrCreateDNSRecord(zoneID string, record DNSRecord) (*DNSRecord, error) {
	// Try to find existing record
	existingRecord, err := dm.GetDNSRecord(zoneID, record.Name, record.Type)
	if err != nil {
		// Record doesn't exist, create it
		return dm.CreateDNSRecord(zoneID, record)
	}
	
	// Record exists, update it
	return dm.UpdateDNSRecord(zoneID, existingRecord.ID, record)
}

// UpdateDynamicIP updates a DNS record with the current public IP
func (dm *DNSManager) UpdateDynamicIP(zoneName, recordName string) error {
	// Get current public IP
	publicIP, err := dm.GetCurrentPublicIP()
	if err != nil {
		return fmt.Errorf("failed to get public IP: %w", err)
	}
	
	// Get zone
	zone, err := dm.GetZoneByName(zoneName)
	if err != nil {
		return fmt.Errorf("failed to get zone: %w", err)
	}
	
	// Update or create A record
	record := DNSRecord{
		Type:    "A",
		Name:    recordName,
		Content: publicIP,
		TTL:     300, // 5 minutes
		Proxied: false,
	}
	
	_, err = dm.UpdateOrCreateDNSRecord(zone.ID, record)
	if err != nil {
		return fmt.Errorf("failed to update DNS record: %w", err)
	}
	
	return nil
}

// GetCurrentPublicIP gets the current public IP address
func (dm *DNSManager) GetCurrentPublicIP() (string, error) {
	resp, err := http.Get("https://api.ipify.org")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	
	return string(body), nil
}

// makeRequest makes an HTTP request to the CloudFlare API
func (dm *DNSManager) makeRequest(method, url string, data interface{}) (*CloudFlareResponse, error) {
	var body io.Reader
	
	if data != nil {
		jsonData, err := json.Marshal(data)
		if err != nil {
			return nil, err
		}
		body = bytes.NewBuffer(jsonData)
	}
	
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	
	req.Header.Set("Authorization", "Bearer "+dm.apiToken)
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := dm.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	
	var cfResp CloudFlareResponse
	if err := json.Unmarshal(responseBody, &cfResp); err != nil {
		return nil, err
	}
	
	if !cfResp.Success {
		return nil, fmt.Errorf("CloudFlare API error: %v", cfResp.Errors)
	}
	
	return &cfResp, nil
}
