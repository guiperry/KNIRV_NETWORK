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
	"github.com/stretchr/testify/suite"
)

// KNIRVNEXUSBackendTestSuite tests the new KNIRVSERVER backend architecture
type KNIRVNEXUSBackendTestSuite struct {
	suite.Suite
	apiGatewayURL   string
	dveManagerURL   string
	validationURL   string
	frontendURL     string
	httpClient      *http.Client
	authToken       string
	testNodeID      string
	testTaskID      string
}

// DVENode represents a DVE node for testing
type DVENode struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	TEEType      string                 `json:"tee_type"`
	Status       string                 `json:"status"`
	StakeAmount  int64                  `json:"stake_amount"`
	Location     string                 `json:"location"`
	IPAddress    string                 `json:"ip_address"`
	PublicKey    string                 `json:"public_key"`
	Capabilities []string               `json:"capabilities"`
	Latitude     float64                `json:"latitude"`
	Longitude    float64                `json:"longitude"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// ValidationTask represents a validation task
type ValidationTask struct {
	ID              string                 `json:"id"`
	Type            string                 `json:"type"`
	Status          string                 `json:"status"`
	Priority        int                    `json:"priority"`
	SkillCode       string                 `json:"skill_code,omitempty"`
	TestCases       []TestCase             `json:"test_cases,omitempty"`
	RequiredTEEType string                 `json:"required_tee_type"`
	RequestedBy     string                 `json:"requested_by"`
	Parameters      map[string]interface{} `json:"parameters,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
}

// TestCase represents a test case for validation
type TestCase struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Input       map[string]interface{} `json:"input"`
	Expected    map[string]interface{} `json:"expected"`
	Weight      float64                `json:"weight"`
}

// SystemHealth represents system health metrics
type SystemHealth struct {
	ActiveNodes     int    `json:"active_nodes"`
	TotalNodes      int    `json:"total_nodes"`
	OverallStatus   string `json:"overall_status"`
	ValidationQueue int    `json:"validation_queue"`
	P2PConnections  int    `json:"p2p_connections"`
}

func (suite *KNIRVNEXUSBackendTestSuite) SetupSuite() {
	suite.apiGatewayURL = "http://localhost:8080"
	suite.dveManagerURL = "http://localhost:8081"
	suite.validationURL = "http://localhost:8082"
	suite.frontendURL = "http://localhost:3000"
	suite.httpClient = &http.Client{Timeout: 30 * time.Second}
	suite.testNodeID = fmt.Sprintf("test-node-%d", time.Now().Unix())

	// Wait for services to be ready
	suite.waitForServices()

	// Authenticate (if required)
	suite.authenticate()

	suite.T().Log("KNIRVSERVER Backend Integration Test Suite initialized")
}

func (suite *KNIRVNEXUSBackendTestSuite) waitForServices() {
	services := map[string]string{
		"API Gateway":     suite.apiGatewayURL + "/health",
		"DVE Manager":     suite.dveManagerURL + "/health",
		"Validation Core": suite.validationURL + "/health",
		"Frontend":        suite.frontendURL + "/api/health",
	}

	for name, url := range services {
		suite.T().Logf("Waiting for %s to be ready...", name)
		for i := 0; i < 30; i++ {
			resp, err := suite.httpClient.Get(url)
			if err == nil && resp.StatusCode == http.StatusOK {
				resp.Body.Close()
				suite.T().Logf("%s is ready", name)
				break
			}
			if resp != nil {
				resp.Body.Close()
			}
			time.Sleep(2 * time.Second)
		}
	}
}

func (suite *KNIRVNEXUSBackendTestSuite) authenticate() {
	// For now, we'll use a simple test token
	// In a real implementation, this would authenticate with the API Gateway
	suite.authToken = "test-token"
}

func (suite *KNIRVNEXUSBackendTestSuite) makeRequest(method, url string, data interface{}) (*http.Response, error) {
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

	req.Header.Set("Content-Type", "application/json")
	if suite.authToken != "" {
		req.Header.Set("Authorization", "Bearer "+suite.authToken)
	}

	return suite.httpClient.Do(req)
}

// Test 1: API Gateway Health and Basic Functionality
func (suite *KNIRVNEXUSBackendTestSuite) TestAPIGatewayHealth() {
	suite.Run("APIGatewayHealthCheck", func() {
		resp, err := suite.makeRequest("GET", suite.apiGatewayURL+"/health", nil)
		require.NoError(suite.T(), err)
		defer resp.Body.Close()

		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

		var health map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&health)
		require.NoError(suite.T(), err)

		assert.Equal(suite.T(), "healthy", health["status"])
		suite.T().Log("API Gateway health check passed")
	})
}

// Test 2: DVE Manager Node Registration and Management
func (suite *KNIRVNEXUSBackendTestSuite) TestDVEManagerNodeOperations() {
	suite.Run("DVENodeRegistration", func() {
		nodeData := DVENode{
			Name:         suite.testNodeID,
			TEEType:      "software",
			StakeAmount:  1000000,
			Location:     "us-east-1",
			IPAddress:    "192.168.1.100",
			PublicKey:    "test-public-key",
			Capabilities: []string{"skillnode", "base_llm"},
			Latitude:     40.7128,
			Longitude:    -74.0060,
		}

		resp, err := suite.makeRequest("POST", suite.dveManagerURL+"/api/v1/dve-nodes", nodeData)
		require.NoError(suite.T(), err)
		defer resp.Body.Close()

		assert.Equal(suite.T(), http.StatusCreated, resp.StatusCode)

		var registeredNode DVENode
		err = json.NewDecoder(resp.Body).Decode(&registeredNode)
		require.NoError(suite.T(), err)

		assert.NotEmpty(suite.T(), registeredNode.ID)
		assert.Equal(suite.T(), nodeData.Name, registeredNode.Name)
		assert.Equal(suite.T(), "online", registeredNode.Status)

		suite.testNodeID = registeredNode.ID
		suite.T().Logf("DVE Node registered successfully with ID: %s", suite.testNodeID)
	})

	suite.Run("DVENodeRetrieval", func() {
		resp, err := suite.makeRequest("GET", suite.dveManagerURL+"/api/v1/dve-nodes", nil)
		require.NoError(suite.T(), err)
		defer resp.Body.Close()

		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

		var nodes []DVENode
		err = json.NewDecoder(resp.Body).Decode(&nodes)
		require.NoError(suite.T(), err)

		assert.GreaterOrEqual(suite.T(), len(nodes), 1)
		suite.T().Logf("Retrieved %d DVE nodes", len(nodes))
	})
}

// Test 3: Validation Core Task Management
func (suite *KNIRVNEXUSBackendTestSuite) TestValidationCoreOperations() {
	suite.Run("ValidationTaskCreation", func() {
		testCases := []TestCase{
			{
				ID:          "test-case-1",
				Name:        "Basic functionality test",
				Description: "Tests basic SkillNode functionality",
				Input: map[string]interface{}{
					"prompt": "Hello, world!",
				},
				Expected: map[string]interface{}{
					"response": "Hello! How can I help you?",
				},
				Weight: 1.0,
			},
		}

		taskData := ValidationTask{
			Type:            "skillnode",
			Priority:        5,
			SkillCode:       "def hello_world():\n    return 'Hello! How can I help you?'",
			TestCases:       testCases,
			RequiredTEEType: "software",
			RequestedBy:     "test-user",
			Parameters: map[string]interface{}{
				"timeout": "60s",
			},
		}

		resp, err := suite.makeRequest("POST", suite.validationURL+"/api/v1/validation-tasks", taskData)
		require.NoError(suite.T(), err)
		defer resp.Body.Close()

		assert.Equal(suite.T(), http.StatusCreated, resp.StatusCode)

		var createdTask ValidationTask
		err = json.NewDecoder(resp.Body).Decode(&createdTask)
		require.NoError(suite.T(), err)

		assert.NotEmpty(suite.T(), createdTask.ID)
		assert.Equal(suite.T(), "pending", createdTask.Status)
		assert.Equal(suite.T(), taskData.Type, createdTask.Type)

		suite.testTaskID = createdTask.ID
		suite.T().Logf("Validation task created successfully with ID: %s", suite.testTaskID)
	})

	suite.Run("ValidationTaskRetrieval", func() {
		resp, err := suite.makeRequest("GET", suite.validationURL+"/api/v1/validation-tasks", nil)
		require.NoError(suite.T(), err)
		defer resp.Body.Close()

		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

		var tasks []ValidationTask
		err = json.NewDecoder(resp.Body).Decode(&tasks)
		require.NoError(suite.T(), err)

		assert.GreaterOrEqual(suite.T(), len(tasks), 1)
		suite.T().Logf("Retrieved %d validation tasks", len(tasks))
	})
}

// Test 4: System Health and Metrics
func (suite *KNIRVNEXUSBackendTestSuite) TestSystemHealthMetrics() {
	suite.Run("SystemHealthCheck", func() {
		resp, err := suite.makeRequest("GET", suite.dveManagerURL+"/api/v1/system/health", nil)
		require.NoError(suite.T(), err)
		defer resp.Body.Close()

		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

		var health SystemHealth
		err = json.NewDecoder(resp.Body).Decode(&health)
		require.NoError(suite.T(), err)

		assert.GreaterOrEqual(suite.T(), health.TotalNodes, 0)
		assert.GreaterOrEqual(suite.T(), health.ActiveNodes, 0)
		assert.NotEmpty(suite.T(), health.OverallStatus)

		suite.T().Logf("System health: %d/%d nodes active, status: %s", 
			health.ActiveNodes, health.TotalNodes, health.OverallStatus)
	})
}

// Test 5: Frontend Health Check
func (suite *KNIRVNEXUSBackendTestSuite) TestFrontendHealth() {
	suite.Run("FrontendHealthCheck", func() {
		resp, err := suite.makeRequest("GET", suite.frontendURL+"/api/health", nil)
		require.NoError(suite.T(), err)
		defer resp.Body.Close()

		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

		var health map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&health)
		require.NoError(suite.T(), err)

		assert.Equal(suite.T(), "healthy", health["status"])
		suite.T().Log("Frontend health check passed")
	})
}

// TestKNIRVNEXUSBackendIntegration runs the complete backend integration test suite
func TestKNIRVNEXUSBackendIntegration(t *testing.T) {
	// Skip if services are not running
	client := &http.Client{Timeout: 2 * time.Second}
	_, err := client.Get("http://localhost:8080/health")
	if err != nil {
		t.Skip("KNIRVSERVER backend services not running - skipping integration tests")
		return
	}

	suite.Run(t, new(KNIRVNEXUSBackendTestSuite))
}
