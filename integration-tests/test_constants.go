package integration_tests

import (
	"fmt"
	"net/http"
	"time"
)

// Shared test configuration constants
const (
	// Service URLs
	KNIRVControllerURL = "http://localhost:3000" // KNIRVCONTROLLER Unified Server
	KNIRVRouterURL     = "http://localhost:8085" // KNIRVROUTER
	KNIRVGraphURL      = "http://localhost:8081" // KNIRVGRAPH
	KNIRVChainURL      = "http://localhost:8080" // KNIRVCHAIN
	KNIRVOracleURL     = "http://localhost:8086" // KNIRVORACLE
	KNIRVNexusURL      = "http://localhost:8090" // KNIRVNEXUS (updated port)
	KNIRVNEXUS_BASE_URL = "http://localhost:8090" // Alternative name for compatibility

	// Timeouts
	TestTimeout   = 60 * time.Second // Standard test timeout
	TEST_TIMEOUT  = 30 * time.Second // Alternative name for compatibility
)

// Shared data structures
type ErrorContext struct {
	AgentID      string                 `json:"agent_id"`
	BaseModelID  string                 `json:"base_model_id"`
	OS           string                 `json:"os"`
	Architecture string                 `json:"architecture"`
	Timestamp    string                 `json:"timestamp"`
	ErrorType    string                 `json:"error_type"`
	ErrorMessage string                 `json:"error_message"`
	Context      map[string]interface{} `json:"context"`
}

type SkillInvocationRequest struct {
	InvocationID string                 `json:"invocation_id"`
	AgentID      string                 `json:"agent_id"`
	SkillURI     string                 `json:"skill_uri"`
	UserID       string                 `json:"user_id"`
	SkillID      string                 `json:"skill_id"`
	Amount       string                 `json:"amount"`
	Parameters   map[string]interface{} `json:"parameters"`
}

type SkillInvocationResponse struct {
	InvocationID string `json:"invocation_id"`
	Status       string `json:"status"`
	ErrorMessage string `json:"error_message"`
	Result       string `json:"result"`
}

// Shared utility functions
func waitForService(url string, timeout time.Duration) bool {
	client := &http.Client{Timeout: 5 * time.Second}
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err == nil && resp.StatusCode == 200 {
			resp.Body.Close()
			return true
		}
		if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(1 * time.Second)
	}
	return false
}

func makeHTTPRequest(method, url string, body interface{}) (*http.Response, error) {
	client := &http.Client{Timeout: TestTimeout}
	
	var req *http.Request
	var err error
	
	if body != nil {
		// Handle different body types as needed
		req, err = http.NewRequest(method, url, nil)
	} else {
		req, err = http.NewRequest(method, url, nil)
	}
	
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	
	return client.Do(req)
}
