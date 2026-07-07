package integration_tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// PoAu-D Integration Test Suite
// Tests the complete PoAu-D consensus mechanism integration across all components

const (
	gatewayBaseURL = "http://localhost:8000"
	rootNodeURL    = "http://localhost:8080"
	testTimeout    = 30 * time.Second
)

// Test data structures
type PoAuDStatus struct {
	Enabled               bool                   `json:"enabled"`
	NetworkAuthorsCount   int                    `json:"network_authors_count,omitempty"`
	MainPoolSize          int                    `json:"main_pool_size,omitempty"`
	PasPoolSize           int                    `json:"pas_pool_size,omitempty"`
	DelegatedTransactions int                    `json:"delegated_transactions,omitempty"`
	DelegationStats       map[string]interface{} `json:"delegation_stats,omitempty"`
}

type PoAuDResponse struct {
	Success bool   `json:"success,omitempty"`
	Enabled bool   `json:"enabled,omitempty"`
	Message string `json:"message,omitempty"`
	Address string `json:"address,omitempty"`
}

type NetworkAuthorsResponse struct {
	NetworkAuthors []string `json:"network_authors"`
	Count          int      `json:"count"`
}

type NetworkAuthor struct {
	Address string `json:"address"`
}

// Test helper functions
func makeRequest(method, url string, body interface{}) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: testTimeout}
	return client.Do(req)
}

func TestPoAuDIntegrationSuite(t *testing.T) {
	// Run tests in sequence to ensure proper state management
	t.Run("01_InitialStatusCheck", testInitialStatusCheck)
	t.Run("02_EnablePoAuD", testEnablePoAuD)
	t.Run("03_NetworkAuthorsManagement", testNetworkAuthorsManagement)
	t.Run("04_StatusAfterConfiguration", testStatusAfterConfiguration)
	t.Run("05_GatewayIntegration", testGatewayIntegration)
	t.Run("06_ErrorHandling", testErrorHandling)
	t.Run("07_DisablePoAuD", testDisablePoAuD)
	t.Run("08_FinalStatusCheck", testFinalStatusCheck)
}

func testInitialStatusCheck(t *testing.T) {
	t.Log("Testing initial PoAu-D status check...")

	// Check status via direct KNIRV-ORACLE API
	resp, err := makeRequest("GET", rootNodeURL+"/poaud/status", nil)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var status PoAuDStatus
	err = json.NewDecoder(resp.Body).Decode(&status)
	require.NoError(t, err)

	t.Logf("Initial PoAu-D status: enabled=%t", status.Enabled)

	// PoAu-D should be disabled by default
	assert.False(t, status.Enabled, "PoAu-D should be disabled by default")
}

func testEnablePoAuD(t *testing.T) {
	t.Log("Testing PoAu-D enable functionality...")

	// Enable PoAu-D via direct API
	resp, err := makeRequest("POST", rootNodeURL+"/poaud/enable", nil)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var enableResp PoAuDResponse
	err = json.NewDecoder(resp.Body).Decode(&enableResp)
	require.NoError(t, err)

	assert.True(t, enableResp.Enabled, "PoAu-D should be enabled")
	assert.Contains(t, enableResp.Message, "enabled", "Response should indicate PoAu-D was enabled")

	t.Logf("PoAu-D enabled: %s", enableResp.Message)

	// Verify status change
	time.Sleep(1 * time.Second) // Allow time for state change

	resp, err = makeRequest("GET", rootNodeURL+"/poaud/status", nil)
	require.NoError(t, err)
	defer resp.Body.Close()

	var status PoAuDStatus
	err = json.NewDecoder(resp.Body).Decode(&status)
	require.NoError(t, err)

	assert.True(t, status.Enabled, "PoAu-D status should show enabled")
}

func testNetworkAuthorsManagement(t *testing.T) {
	t.Log("Testing Network Authors management...")

	testAuthors := []string{
		"knirv1test001abc123def456",
		"knirv1test002xyz789uvw456",
		"knirv1test003mno345pqr678",
	}

	// Add network authors
	for i, author := range testAuthors {
		t.Logf("Adding network author %d: %s", i+1, author)

		resp, err := makeRequest("POST", rootNodeURL+"/poaud/network-authors/add", NetworkAuthor{Address: author})
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var addResp PoAuDResponse
		err = json.NewDecoder(resp.Body).Decode(&addResp)
		require.NoError(t, err)

		assert.True(t, addResp.Success, "Adding network author should succeed")
		assert.Equal(t, author, addResp.Address, "Response should include the added address")
	}

	// List network authors
	t.Log("Listing network authors...")

	resp, err := makeRequest("GET", rootNodeURL+"/poaud/network-authors", nil)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var authorsResp NetworkAuthorsResponse
	err = json.NewDecoder(resp.Body).Decode(&authorsResp)
	require.NoError(t, err)

	assert.Equal(t, len(testAuthors), authorsResp.Count, "Should have correct number of network authors")
	assert.Len(t, authorsResp.NetworkAuthors, len(testAuthors), "Network authors list should match expected length")

	// Verify all test authors are present
	for _, testAuthor := range testAuthors {
		found := false
		for _, author := range authorsResp.NetworkAuthors {
			if author == testAuthor {
				found = true
				break
			}
		}
		assert.True(t, found, "Test author %s should be in the list", testAuthor)
	}

	// Remove one network author
	removeAuthor := testAuthors[1]
	t.Logf("Removing network author: %s", removeAuthor)

	resp, err = makeRequest("POST", rootNodeURL+"/poaud/network-authors/remove", NetworkAuthor{Address: removeAuthor})
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var removeResp PoAuDResponse
	err = json.NewDecoder(resp.Body).Decode(&removeResp)
	require.NoError(t, err)

	assert.True(t, removeResp.Success, "Removing network author should succeed")
	assert.Equal(t, removeAuthor, removeResp.Address, "Response should include the removed address")

	// Verify removal
	resp, err = makeRequest("GET", rootNodeURL+"/poaud/network-authors", nil)
	require.NoError(t, err)
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&authorsResp)
	require.NoError(t, err)

	assert.Equal(t, len(testAuthors)-1, authorsResp.Count, "Should have one less network author")

	// Verify removed author is not present
	for _, author := range authorsResp.NetworkAuthors {
		assert.NotEqual(t, removeAuthor, author, "Removed author should not be in the list")
	}
}

func testStatusAfterConfiguration(t *testing.T) {
	t.Log("Testing PoAu-D status after configuration...")

	resp, err := makeRequest("GET", rootNodeURL+"/poaud/status", nil)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var status PoAuDStatus
	err = json.NewDecoder(resp.Body).Decode(&status)
	require.NoError(t, err)

	assert.True(t, status.Enabled, "PoAu-D should still be enabled")
	assert.Greater(t, status.NetworkAuthorsCount, 0, "Should have network authors configured")

	t.Logf("PoAu-D status after configuration: enabled=%t, authors=%d",
		status.Enabled, status.NetworkAuthorsCount)

	// Check for delegation stats if available
	if status.DelegationStats != nil {
		t.Logf("Delegation stats available: %+v", status.DelegationStats)
	}
}

func testGatewayIntegration(t *testing.T) {
	t.Log("Testing PoAu-D integration through gateway...")

	// Test gateway routing to PoAu-D endpoints
	resp, err := makeRequest("GET", gatewayBaseURL+"/poaud/status", nil)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var status PoAuDStatus
	err = json.NewDecoder(resp.Body).Decode(&status)
	require.NoError(t, err)

	assert.True(t, status.Enabled, "PoAu-D should be enabled via gateway")

	t.Logf("Gateway PoAu-D status: enabled=%t", status.Enabled)
}

func testErrorHandling(t *testing.T) {
	t.Log("Testing PoAu-D error handling...")

	// Test adding invalid network author
	resp, err := makeRequest("POST", rootNodeURL+"/poaud/network-authors/add", NetworkAuthor{Address: ""})
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "Empty address should return bad request")

	// Test removing non-existent network author
	resp, err = makeRequest("POST", rootNodeURL+"/poaud/network-authors/remove", NetworkAuthor{Address: "nonexistent"})
	require.NoError(t, err)
	defer resp.Body.Close()

	// Should succeed (idempotent operation) or return appropriate status
	assert.True(t, resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNotFound)
}

func testDisablePoAuD(t *testing.T) {
	t.Log("Testing PoAu-D disable functionality...")

	resp, err := makeRequest("POST", rootNodeURL+"/poaud/disable", nil)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var disableResp PoAuDResponse
	err = json.NewDecoder(resp.Body).Decode(&disableResp)
	require.NoError(t, err)

	assert.False(t, disableResp.Enabled, "PoAu-D should be disabled")
	assert.Contains(t, disableResp.Message, "disabled", "Response should indicate PoAu-D was disabled")

	t.Logf("PoAu-D disabled: %s", disableResp.Message)
}

func testFinalStatusCheck(t *testing.T) {
	t.Log("Testing final PoAu-D status check...")

	// Allow time for state change
	time.Sleep(1 * time.Second)

	resp, err := makeRequest("GET", rootNodeURL+"/poaud/status", nil)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var status PoAuDStatus
	err = json.NewDecoder(resp.Body).Decode(&status)
	require.NoError(t, err)

	assert.False(t, status.Enabled, "PoAu-D should be disabled")

	t.Logf("Final PoAu-D status: enabled=%t", status.Enabled)
}

// Benchmark tests for PoAu-D operations
func BenchmarkPoAuDOperations(b *testing.B) {
	b.Run("StatusCheck", benchmarkStatusCheck)
	b.Run("AddNetworkAuthor", benchmarkAddNetworkAuthor)
	b.Run("ListNetworkAuthors", benchmarkListNetworkAuthors)
}

func benchmarkStatusCheck(b *testing.B) {
	for i := 0; i < b.N; i++ {
		resp, err := makeRequest("GET", rootNodeURL+"/poaud/status", nil)
		if err != nil {
			b.Fatal(err)
		}
		resp.Body.Close()
	}
}

func benchmarkAddNetworkAuthor(b *testing.B) {
	for i := 0; i < b.N; i++ {
		author := fmt.Sprintf("knirv1bench%06d", i)
		resp, err := makeRequest("POST", rootNodeURL+"/poaud/network-authors/add", NetworkAuthor{Address: author})
		if err != nil {
			b.Fatal(err)
		}
		resp.Body.Close()
	}
}

func benchmarkListNetworkAuthors(b *testing.B) {
	for i := 0; i < b.N; i++ {
		resp, err := makeRequest("GET", rootNodeURL+"/poaud/network-authors", nil)
		if err != nil {
			b.Fatal(err)
		}
		resp.Body.Close()
	}
}
