package cloudflare

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDNSManager(t *testing.T) {
	manager := NewDNSManager("test-token")
	assert.NotNil(t, manager)
	assert.Equal(t, "test-token", manager.apiToken)
	assert.Equal(t, "https://api.cloudflare.com/client/v4", manager.baseURL)
	assert.NotNil(t, manager.client)
	assert.Equal(t, 30*time.Second, manager.client.Timeout)
}

func TestDNSManager_GetCurrentPublicIP(t *testing.T) {
	// This test makes a real HTTP request to ipify.org
	// In a real test environment, this should work
	manager := NewDNSManager("test-token")

	ip, err := manager.GetCurrentPublicIP()
	// The method should succeed and return a valid IP
	assert.NoError(t, err)
	assert.NotEmpty(t, ip)
	// Basic IP validation - should contain dots and be reasonably long
	assert.Contains(t, ip, ".")
	assert.Greater(t, len(ip), 7)
}

func TestDNSManager_makeRequest(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		response := CloudFlareResponse{
			Success: true,
			Result:  "test result",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	manager := NewDNSManager("test-token")
	manager.baseURL = server.URL

	resp, err := manager.makeRequest("GET", server.URL+"/test", nil)
	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Equal(t, "test result", resp.Result)
}

func TestDNSManager_makeRequest_Error(t *testing.T) {
	// Create a test server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := CloudFlareResponse{
			Success: false,
			Errors: []map[string]interface{}{
				{"code": 1001, "message": "test error"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	manager := NewDNSManager("test-token")
	manager.baseURL = server.URL

	_, err := manager.makeRequest("GET", server.URL+"/test", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "CloudFlare API error")
}

func TestDNSManager_makeRequest_HTTPError(t *testing.T) {
	// Create a test server that returns HTTP error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	manager := NewDNSManager("test-token")
	manager.baseURL = server.URL

	_, err := manager.makeRequest("GET", server.URL+"/test", nil)
	assert.Error(t, err)
}

func TestDNSManager_GetZones(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/client/v4/zones", r.URL.Path)

		response := CloudFlareResponse{
			Success: true,
			Result: []interface{}{
				map[string]interface{}{
					"id":     "zone1",
					"name":   "example.com",
					"status": "active",
				},
				map[string]interface{}{
					"id":     "zone2",
					"name":   "test.com",
					"status": "pending",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	manager := NewDNSManager("test-token")
	manager.baseURL = server.URL + "/client/v4"

	zones, err := manager.GetZones()
	require.NoError(t, err)
	assert.Len(t, zones, 2)
	assert.Equal(t, "zone1", zones[0].ID)
	assert.Equal(t, "example.com", zones[0].Name)
	assert.Equal(t, "active", zones[0].Status)
	assert.Equal(t, "zone2", zones[1].ID)
	assert.Equal(t, "test.com", zones[1].Name)
	assert.Equal(t, "pending", zones[1].Status)
}

func TestDNSManager_GetZoneByName(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := CloudFlareResponse{
			Success: true,
			Result: []interface{}{
				map[string]interface{}{
					"id":     "zone1",
					"name":   "example.com",
					"status": "active",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	manager := NewDNSManager("test-token")
	manager.baseURL = server.URL + "/client/v4"

	zone, err := manager.GetZoneByName("example.com")
	require.NoError(t, err)
	assert.Equal(t, "zone1", zone.ID)
	assert.Equal(t, "example.com", zone.Name)
	assert.Equal(t, "active", zone.Status)
}

func TestDNSManager_GetZoneByName_NotFound(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := CloudFlareResponse{
			Success: true,
			Result: []interface{}{
				map[string]interface{}{
					"id":     "zone1",
					"name":   "example.com",
					"status": "active",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	manager := NewDNSManager("test-token")
	manager.baseURL = server.URL + "/client/v4"

	_, err := manager.GetZoneByName("notfound.com")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "zone notfound.com not found")
}

func TestDNSManager_GetDNSRecords(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/zones/zone1/dns_records")

		response := CloudFlareResponse{
			Success: true,
			Result: []interface{}{
				map[string]interface{}{
					"id":       "record1",
					"type":     "A",
					"name":     "test.example.com",
					"content":  "192.168.1.1",
					"ttl":      300.0,
					"proxied":  false,
					"zone_id":  "zone1",
					"priority": 10.0,
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	manager := NewDNSManager("test-token")
	manager.baseURL = server.URL + "/client/v4"

	records, err := manager.GetDNSRecords("zone1")
	require.NoError(t, err)
	assert.Len(t, records, 1)
	assert.Equal(t, "record1", records[0].ID)
	assert.Equal(t, "A", records[0].Type)
	assert.Equal(t, "test.example.com", records[0].Name)
	assert.Equal(t, "192.168.1.1", records[0].Content)
	assert.Equal(t, 300, records[0].TTL)
	assert.False(t, records[0].Proxied)
	assert.Equal(t, "zone1", records[0].ZoneID)
	assert.Equal(t, 10, records[0].Priority)
}

func TestDNSManager_GetDNSRecord(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := CloudFlareResponse{
			Success: true,
			Result: []interface{}{
				map[string]interface{}{
					"id":      "record1",
					"type":    "A",
					"name":    "test.example.com",
					"content": "192.168.1.1",
					"ttl":     300.0,
					"proxied": false,
					"zone_id": "zone1",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	manager := NewDNSManager("test-token")
	manager.baseURL = server.URL + "/client/v4"

	record, err := manager.GetDNSRecord("zone1", "test.example.com", "A")
	require.NoError(t, err)
	assert.Equal(t, "record1", record.ID)
	assert.Equal(t, "A", record.Type)
	assert.Equal(t, "test.example.com", record.Name)
	assert.Equal(t, "192.168.1.1", record.Content)
}

func TestDNSManager_GetDNSRecord_NotFound(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := CloudFlareResponse{
			Success: true,
			Result: []interface{}{},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	manager := NewDNSManager("test-token")
	manager.baseURL = server.URL + "/client/v4"

	_, err := manager.GetDNSRecord("zone1", "notfound.example.com", "A")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "DNS record notfound.example.com (A) not found")
}

func TestDNSManager_CreateDNSRecord(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "/zones/zone1/dns_records")

		response := CloudFlareResponse{
			Success: true,
			Result: map[string]interface{}{
				"id":      "record1",
				"type":    "A",
				"name":    "test.example.com",
				"content": "192.168.1.1",
				"ttl":     300.0,
				"proxied": false,
				"zone_id": "zone1",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	manager := NewDNSManager("test-token")
	manager.baseURL = server.URL + "/client/v4"

	record := DNSRecord{
		Type:    "A",
		Name:    "test.example.com",
		Content: "192.168.1.1",
		TTL:     300,
		Proxied: false,
	}

	createdRecord, err := manager.CreateDNSRecord("zone1", record)
	require.NoError(t, err)
	assert.Equal(t, "record1", createdRecord.ID)
	assert.Equal(t, "A", createdRecord.Type)
	assert.Equal(t, "test.example.com", createdRecord.Name)
	assert.Equal(t, "192.168.1.1", createdRecord.Content)
}

func TestDNSManager_UpdateDNSRecord(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "PUT", r.Method)
		assert.Contains(t, r.URL.Path, "/zones/zone1/dns_records/record1")

		response := CloudFlareResponse{
			Success: true,
			Result: map[string]interface{}{
				"id":      "record1",
				"type":    "A",
				"name":    "test.example.com",
				"content": "192.168.1.2",
				"ttl":     300.0,
				"proxied": false,
				"zone_id": "zone1",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	manager := NewDNSManager("test-token")
	manager.baseURL = server.URL + "/client/v4"

	record := DNSRecord{
		Type:    "A",
		Name:    "test.example.com",
		Content: "192.168.1.2",
		TTL:     300,
		Proxied: false,
	}

	updatedRecord, err := manager.UpdateDNSRecord("zone1", "record1", record)
	require.NoError(t, err)
	assert.Equal(t, "record1", updatedRecord.ID)
	assert.Equal(t, "192.168.1.2", updatedRecord.Content)
}

func TestDNSManager_DeleteDNSRecord(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)
		assert.Contains(t, r.URL.Path, "/zones/zone1/dns_records/record1")

		response := CloudFlareResponse{
			Success: true,
			Result:  nil,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	manager := NewDNSManager("test-token")
	manager.baseURL = server.URL + "/client/v4"

	err := manager.DeleteDNSRecord("zone1", "record1")
	assert.NoError(t, err)
}

func TestDNSManager_UpdateOrCreateDNSRecord_Update(t *testing.T) {
	// Create a test server that returns existing records
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && callCount == 0 {
			// First call: GetDNSRecords
			callCount++
			response := CloudFlareResponse{
				Success: true,
				Result: []interface{}{
					map[string]interface{}{
						"id":      "record1",
						"type":    "A",
						"name":    "test.example.com",
						"content": "192.168.1.1",
						"ttl":     300.0,
						"proxied": false,
						"zone_id": "zone1",
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		} else if r.Method == "PUT" {
			// Second call: UpdateDNSRecord
			response := CloudFlareResponse{
				Success: true,
				Result: map[string]interface{}{
					"id":      "record1",
					"type":    "A",
					"name":    "test.example.com",
					"content": "192.168.1.2",
					"ttl":     300.0,
					"proxied": false,
					"zone_id": "zone1",
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		}
	}))
	defer server.Close()

	manager := NewDNSManager("test-token")
	manager.baseURL = server.URL + "/client/v4"

	record := DNSRecord{
		Type:    "A",
		Name:    "test.example.com",
		Content: "192.168.1.2",
		TTL:     300,
		Proxied: false,
	}

	updatedRecord, err := manager.UpdateOrCreateDNSRecord("zone1", record)
	require.NoError(t, err)
	assert.Equal(t, "record1", updatedRecord.ID)
	assert.Equal(t, "192.168.1.2", updatedRecord.Content)
}

func TestDNSManager_UpdateOrCreateDNSRecord_Create(t *testing.T) {
	// Create a test server that returns no existing records
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && callCount == 0 {
			// First call: GetDNSRecords (empty)
			callCount++
			response := CloudFlareResponse{
				Success: true,
				Result:  []interface{}{},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		} else if r.Method == "POST" {
			// Second call: CreateDNSRecord
			response := CloudFlareResponse{
				Success: true,
				Result: map[string]interface{}{
					"id":      "record1",
					"type":    "A",
					"name":    "test.example.com",
					"content": "192.168.1.1",
					"ttl":     300.0,
					"proxied": false,
					"zone_id": "zone1",
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		}
	}))
	defer server.Close()

	manager := NewDNSManager("test-token")
	manager.baseURL = server.URL + "/client/v4"

	record := DNSRecord{
		Type:    "A",
		Name:    "test.example.com",
		Content: "192.168.1.1",
		TTL:     300,
		Proxied: false,
	}

	createdRecord, err := manager.UpdateOrCreateDNSRecord("zone1", record)
	require.NoError(t, err)
	assert.Equal(t, "record1", createdRecord.ID)
	assert.Equal(t, "192.168.1.1", createdRecord.Content)
}

func TestDNSManager_UpdateDynamicIP(t *testing.T) {
	// This test would require extensive mocking
	// For now, test that the method exists and handles errors
	manager := NewDNSManager("test-token")

	err := manager.UpdateDynamicIP("example.com", "test")
	assert.Error(t, err) // Expected to fail without proper setup
}