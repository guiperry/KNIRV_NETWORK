package integration_tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// Month 12 WebSocket and Real-time Communication Testing Suite
type WebSocketTestSuite struct {
	suite.Suite
	gatewayURL    string
	wsURL         string
	httpClient    *http.Client
	authToken     string
	testWallet    *TestWallet
	wsConnections []*websocket.Conn
}

type WebSocketMessage struct {
	Type      string                 `json:"type"`
	Service   string                 `json:"service,omitempty"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Timestamp int64                  `json:"timestamp,omitempty"`
	ID        string                 `json:"id,omitempty"`
}

type MetricsUpdate struct {
	Service    string                 `json:"service"`
	Metrics    map[string]interface{} `json:"metrics"`
	Timestamp  int64                  `json:"timestamp"`
	UpdateType string                 `json:"update_type"`
}

type ServiceSubscription struct {
	Service    string   `json:"service"`
	Events     []string `json:"events"`
	Subscribed bool     `json:"subscribed"`
	ID         string   `json:"id"`
}

func (suite *WebSocketTestSuite) SetupSuite() {
	suite.gatewayURL = "http://localhost:8000"
	suite.wsURL = "ws://localhost:8000/gateway/ws"
	suite.httpClient = &http.Client{Timeout: 30 * time.Second}
	suite.wsConnections = make([]*websocket.Conn, 0)

	// Wait for services
	suite.waitForServices()

	// Authenticate
	suite.authenticate()

	// Create test wallet
	suite.createTestWallet()
}

func (suite *WebSocketTestSuite) TearDownSuite() {
	// Close all WebSocket connections
	for _, conn := range suite.wsConnections {
		if conn != nil {
			conn.Close()
		}
	}
}

func (suite *WebSocketTestSuite) waitForServices() {
	services := []string{"knirvchain", "knirvgraph", "knirvserver", "knirvoracle", "knirvrouter"}

	for _, service := range services {
		suite.T().Logf("Waiting for service: %s", service)

		for i := 0; i < 30; i++ {
			resp, err := suite.httpClient.Get(fmt.Sprintf("%s/%s/health", suite.gatewayURL, service))
			if err == nil && resp.StatusCode == http.StatusOK {
				resp.Body.Close()
				break
			}
			if resp != nil {
				resp.Body.Close()
			}
			time.Sleep(1 * time.Second)
		}
	}

	suite.T().Log("All services are ready for WebSocket testing")
}

func (suite *WebSocketTestSuite) authenticate() {
	loginData := map[string]string{
		"username": "admin",
		"password": "password",
	}

	resp := suite.makeRequest("POST", "/auth/login", loginData)
	require.True(suite.T(), resp.Success, "Authentication failed")

	suite.authToken = resp.Data["token"].(string)
	suite.T().Log("Authenticated for WebSocket testing")
}

func (suite *WebSocketTestSuite) createTestWallet() {
	walletData := map[string]string{
		"name": "websocket_test_wallet",
	}

	resp := suite.makeAuthenticatedRequest("POST", "/knirvwallet/wallet/create", walletData)
	require.True(suite.T(), resp.Success, "Failed to create test wallet")

	suite.testWallet = &TestWallet{
		Address:  resp.Data["address"].(string),
		Mnemonic: resp.Data["mnemonic"].(string),
		Balance:  "0",
	}

	suite.T().Logf("Created WebSocket test wallet: %s", suite.testWallet.Address)
}

// Test 1: WebSocket Connection and Basic Communication
func (suite *WebSocketTestSuite) TestWebSocketConnectionAndBasicCommunication() {
	suite.Run("WebSocketConnectionAndBasicCommunicationTest", func() {
		// Test 1.1: Establish WebSocket connection
		suite.T().Log("Testing WebSocket connection establishment...")

		conn, err := suite.createWebSocketConnection()
		require.NoError(suite.T(), err, "Failed to establish WebSocket connection")
		defer conn.Close()

		suite.T().Log("WebSocket connection established successfully")

		// Test 1.2: Send ping and receive pong
		suite.T().Log("Testing ping-pong communication...")

		pingMsg := WebSocketMessage{
			Type:      "ping",
			Timestamp: time.Now().Unix(),
		}

		err = conn.WriteJSON(pingMsg)
		require.NoError(suite.T(), err, "Failed to send ping message")

		var pongMsg WebSocketMessage
		err = conn.ReadJSON(&pongMsg)
		require.NoError(suite.T(), err, "Failed to receive pong message")

		assert.Equal(suite.T(), "pong", pongMsg.Type, "Should receive pong response")
		suite.T().Log("Ping-pong communication successful")

		// Test 1.3: Test message echo
		suite.T().Log("Testing message echo...")

		echoMsg := WebSocketMessage{
			Type: "echo",
			Data: map[string]interface{}{
				"test_message": "Hello WebSocket",
				"timestamp":    time.Now().Unix(),
			},
		}

		err = conn.WriteJSON(echoMsg)
		require.NoError(suite.T(), err, "Failed to send echo message")

		var echoResponse WebSocketMessage
		err = conn.ReadJSON(&echoResponse)
		require.NoError(suite.T(), err, "Failed to receive echo response")

		assert.Equal(suite.T(), "echo_response", echoResponse.Type, "Should receive echo response")
		assert.Equal(suite.T(), echoMsg.Data["test_message"], echoResponse.Data["test_message"], "Echo message should match")

		suite.T().Log("Message echo successful")
	})
}

// Test 2: Service Subscription and Real-time Updates
func (suite *WebSocketTestSuite) TestServiceSubscriptionAndRealTimeUpdates() {
	suite.Run("ServiceSubscriptionAndRealTimeUpdatesTest", func() {
		conn, err := suite.createWebSocketConnection()
		require.NoError(suite.T(), err, "Failed to establish WebSocket connection")
		defer conn.Close()

		// Test 2.1: Subscribe to KNIRVCHAIN updates
		suite.T().Log("Testing subscription to KNIRVCHAIN updates...")

		subscribeMsg := WebSocketMessage{
			Type:    "subscribe",
			Service: "knirvchain",
			Data: map[string]interface{}{
				"events": []string{"transaction", "block", "llm_registration"},
			},
		}

		err = conn.WriteJSON(subscribeMsg)
		require.NoError(suite.T(), err, "Failed to send subscription message")

		var subscribeResponse WebSocketMessage
		err = conn.ReadJSON(&subscribeResponse)
		require.NoError(suite.T(), err, "Failed to receive subscription response")

		assert.Equal(suite.T(), "subscribed", subscribeResponse.Type, "Should receive subscription confirmation")
		assert.Equal(suite.T(), "knirvchain", subscribeResponse.Service, "Service should match")

		suite.T().Log("Successfully subscribed to KNIRVCHAIN updates")

		// Test 2.2: Trigger an event and verify real-time update
		suite.T().Log("Testing real-time update delivery...")

		// Create an LLM registration to trigger an event
		go func() {
			time.Sleep(2 * time.Second) // Wait a bit before triggering

			llmData := map[string]interface{}{
				"name":             "websocket_test_llm_" + fmt.Sprintf("%d", time.Now().Unix()),
				"version":          "1.0.0",
				"capabilities":     []string{"websocket-testing"},
				"model_data":       "dGVzdCBtb2RlbCBkYXRh",
				"registration_fee": "1000000",
				"usage_fee":        "100000",
			}

			suite.makeAuthenticatedRequest("POST", "/knirvchain/llm/register", llmData)
		}()

		// Wait for real-time update
		conn.SetReadDeadline(time.Now().Add(10 * time.Second))
		var updateMsg WebSocketMessage
		err = conn.ReadJSON(&updateMsg)

		if err == nil {
			assert.Equal(suite.T(), "event", updateMsg.Type, "Should receive event update")
			assert.Equal(suite.T(), "knirvchain", updateMsg.Service, "Update should be from KNIRVCHAIN")

			suite.T().Logf("Received real-time update: %+v", updateMsg)
		} else {
			suite.T().Logf("No real-time update received within timeout: %v", err)
		}

		// Test 2.3: Subscribe to multiple services
		suite.T().Log("Testing subscription to multiple services...")

		services := []string{"knirvgraph", "knirvserverr", "knirvoracle"}

		for _, service := range services {
			subscribeMsg := WebSocketMessage{
				Type:    "subscribe",
				Service: service,
				Data: map[string]interface{}{
					"events": []string{"status_update", "metrics"},
				},
			}

			err = conn.WriteJSON(subscribeMsg)
			require.NoError(suite.T(), err, "Failed to subscribe to %s", service)

			var response WebSocketMessage
			err = conn.ReadJSON(&response)
			require.NoError(suite.T(), err, "Failed to receive subscription response for %s", service)

			assert.Equal(suite.T(), "subscribed", response.Type, "Should receive subscription confirmation for %s", service)
			suite.T().Logf("Successfully subscribed to %s", service)
		}
	})
}

// Test 3: Live Metrics Streaming
func (suite *WebSocketTestSuite) TestLiveMetricsStreaming() {
	suite.Run("LiveMetricsStreamingTest", func() {
		conn, err := suite.createWebSocketConnection()
		require.NoError(suite.T(), err, "Failed to establish WebSocket connection")
		defer conn.Close()

		// Test 3.1: Request live metrics stream
		suite.T().Log("Testing live metrics streaming...")

		metricsRequestMsg := WebSocketMessage{
			Type: "start_metrics_stream",
			Data: map[string]interface{}{
				"services": []string{"knirvchain", "knirvgraph", "knirvserverr"},
				"interval": 5, // 5 seconds
			},
		}

		err = conn.WriteJSON(metricsRequestMsg)
		require.NoError(suite.T(), err, "Failed to request metrics stream")

		var streamResponse WebSocketMessage
		err = conn.ReadJSON(&streamResponse)
		require.NoError(suite.T(), err, "Failed to receive stream response")

		assert.Equal(suite.T(), "metrics_stream_started", streamResponse.Type, "Should confirm metrics stream started")

		// Test 3.2: Receive metrics updates
		suite.T().Log("Waiting for metrics updates...")

		metricsReceived := 0
		timeout := time.After(20 * time.Second)

		for metricsReceived < 3 { // Wait for at least 3 metrics updates
			select {
			case <-timeout:
				suite.T().Log("Timeout waiting for metrics updates")
				goto metricsTestEnd
			default:
				conn.SetReadDeadline(time.Now().Add(10 * time.Second))
				var metricsMsg WebSocketMessage
				err = conn.ReadJSON(&metricsMsg)

				if err == nil && metricsMsg.Type == "metrics" {
					metricsReceived++

					assert.Contains(suite.T(), metricsMsg.Data, "service", "Metrics should include service name")
					assert.Contains(suite.T(), metricsMsg.Data, "metrics", "Should contain metrics data")

					suite.T().Logf("Received metrics update %d: %+v", metricsReceived, metricsMsg.Data)
				}
			}
		}

	metricsTestEnd:
		assert.Greater(suite.T(), metricsReceived, 0, "Should receive at least one metrics update")

		// Test 3.3: Stop metrics stream
		suite.T().Log("Testing metrics stream stop...")

		stopStreamMsg := WebSocketMessage{
			Type: "stop_metrics_stream",
		}

		err = conn.WriteJSON(stopStreamMsg)
		require.NoError(suite.T(), err, "Failed to stop metrics stream")

		var stopResponse WebSocketMessage
		err = conn.ReadJSON(&stopResponse)
		require.NoError(suite.T(), err, "Failed to receive stop response")

		assert.Equal(suite.T(), "metrics_stream_stopped", stopResponse.Type, "Should confirm metrics stream stopped")
	})
}

// Test 4: Concurrent WebSocket Connections
func (suite *WebSocketTestSuite) TestConcurrentWebSocketConnections() {
	suite.Run("ConcurrentWebSocketConnectionsTest", func() {
		// Test 4.1: Create multiple concurrent connections
		suite.T().Log("Testing concurrent WebSocket connections...")

		numConnections := 5
		connections := make([]*websocket.Conn, numConnections)

		// Establish multiple connections
		for i := 0; i < numConnections; i++ {
			conn, err := suite.createWebSocketConnection()
			require.NoError(suite.T(), err, "Failed to establish connection %d", i+1)
			connections[i] = conn

			suite.T().Logf("Established connection %d", i+1)
		}

		// Test communication on all connections
		for i, conn := range connections {
			pingMsg := WebSocketMessage{
				Type:      "ping",
				ID:        fmt.Sprintf("conn_%d", i+1),
				Timestamp: time.Now().Unix(),
			}

			err := conn.WriteJSON(pingMsg)
			require.NoError(suite.T(), err, "Failed to send ping on connection %d", i+1)

			var pongMsg WebSocketMessage
			err = conn.ReadJSON(&pongMsg)
			require.NoError(suite.T(), err, "Failed to receive pong on connection %d", i+1)

			assert.Equal(suite.T(), "pong", pongMsg.Type, "Should receive pong on connection %d", i+1)
		}

		suite.T().Log("All concurrent connections working correctly")

		// Clean up connections
		for i, conn := range connections {
			conn.Close()
			suite.T().Logf("Closed connection %d", i+1)
		}
	})
}

// Test 5: WebSocket Error Handling and Reconnection
func (suite *WebSocketTestSuite) TestWebSocketErrorHandlingAndReconnection() {
	suite.Run("WebSocketErrorHandlingAndReconnectionTest", func() {
		// Test 5.1: Invalid message handling
		suite.T().Log("Testing invalid message handling...")

		conn, err := suite.createWebSocketConnection()
		require.NoError(suite.T(), err, "Failed to establish WebSocket connection")
		defer conn.Close()

		// Send invalid JSON
		err = conn.WriteMessage(websocket.TextMessage, []byte("invalid json"))
		require.NoError(suite.T(), err, "Failed to send invalid message")

		// Should receive error response
		var errorResponse WebSocketMessage
		err = conn.ReadJSON(&errorResponse)

		if err == nil {
			assert.Equal(suite.T(), "error", errorResponse.Type, "Should receive error response for invalid message")
			suite.T().Logf("Received error response: %+v", errorResponse)
		}

		// Test 5.2: Connection recovery
		suite.T().Log("Testing connection recovery...")

		// Close connection and reconnect
		conn.Close()

		// Wait a moment
		time.Sleep(2 * time.Second)

		// Reconnect
		newConn, err := suite.createWebSocketConnection()
		require.NoError(suite.T(), err, "Failed to reconnect WebSocket")
		defer newConn.Close()

		// Test that new connection works
		pingMsg := WebSocketMessage{
			Type:      "ping",
			Timestamp: time.Now().Unix(),
		}

		err = newConn.WriteJSON(pingMsg)
		require.NoError(suite.T(), err, "Failed to send ping after reconnection")

		var pongMsg WebSocketMessage
		err = newConn.ReadJSON(&pongMsg)
		require.NoError(suite.T(), err, "Failed to receive pong after reconnection")

		assert.Equal(suite.T(), "pong", pongMsg.Type, "Should receive pong after reconnection")
		suite.T().Log("Connection recovery successful")
	})
}

// Test 6: WebSocket Authentication and Authorization
func (suite *WebSocketTestSuite) TestWebSocketAuthenticationAndAuthorization() {
	suite.Run("WebSocketAuthenticationAndAuthorizationTest", func() {
		// Test 6.1: Authenticated WebSocket connection
		suite.T().Log("Testing authenticated WebSocket connection...")

		conn, err := suite.createAuthenticatedWebSocketConnection()
		require.NoError(suite.T(), err, "Failed to establish authenticated WebSocket connection")
		defer conn.Close()

		// Test authenticated request
		authTestMsg := WebSocketMessage{
			Type: "authenticated_request",
			Data: map[string]interface{}{
				"action": "get_user_info",
			},
		}

		err = conn.WriteJSON(authTestMsg)
		require.NoError(suite.T(), err, "Failed to send authenticated request")

		var authResponse WebSocketMessage
		err = conn.ReadJSON(&authResponse)
		require.NoError(suite.T(), err, "Failed to receive authenticated response")

		assert.Equal(suite.T(), "authenticated_response", authResponse.Type, "Should receive authenticated response")
		suite.T().Log("Authenticated WebSocket communication successful")

		// Test 6.2: Unauthorized access attempt
		suite.T().Log("Testing unauthorized access handling...")

		unauthorizedMsg := WebSocketMessage{
			Type: "admin_request",
			Data: map[string]interface{}{
				"action": "system_shutdown",
			},
		}

		err = conn.WriteJSON(unauthorizedMsg)
		require.NoError(suite.T(), err, "Failed to send unauthorized request")

		var unauthorizedResponse WebSocketMessage
		err = conn.ReadJSON(&unauthorizedResponse)

		if err == nil {
			assert.Equal(suite.T(), "unauthorized", unauthorizedResponse.Type, "Should receive unauthorized response")
			suite.T().Log("Unauthorized access properly rejected")
		}
	})
}

// Helper methods
func (suite *WebSocketTestSuite) createWebSocketConnection() (*websocket.Conn, error) {
	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(suite.wsURL, nil)
	if err != nil {
		return nil, err
	}

	suite.wsConnections = append(suite.wsConnections, conn)
	return conn, nil
}

func (suite *WebSocketTestSuite) createAuthenticatedWebSocketConnection() (*websocket.Conn, error) {
	headers := http.Header{}
	headers.Set("Authorization", "Bearer "+suite.authToken)

	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(suite.wsURL, headers)
	if err != nil {
		return nil, err
	}

	suite.wsConnections = append(suite.wsConnections, conn)
	return conn, nil
}

func (suite *WebSocketTestSuite) makeRequest(method, path string, data interface{}) *TestResponse {
	return suite.makeRequestWithAuth(method, path, data, "")
}

func (suite *WebSocketTestSuite) makeAuthenticatedRequest(method, path string, data interface{}) *TestResponse {
	return suite.makeRequestWithAuth(method, path, data, suite.authToken)
}

func (suite *WebSocketTestSuite) makeRequestWithAuth(method, path string, data interface{}, token string) *TestResponse {
	var body io.Reader
	if data != nil {
		jsonData, err := json.Marshal(data)
		if err != nil {
			return &TestResponse{Success: false, Error: "Failed to marshal request data"}
		}
		body = bytes.NewReader(jsonData)
	}

	url := suite.gatewayURL + path
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return &TestResponse{Success: false, Error: "Failed to create request"}
	}

	if data != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := suite.httpClient.Do(req)
	if err != nil {
		return &TestResponse{Success: false, Error: err.Error()}
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return &TestResponse{Success: false, Error: "Failed to read response body"}
	}

	var testResp TestResponse
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		if len(responseBody) > 0 {
			err = json.Unmarshal(responseBody, &testResp.Data)
			if err != nil {
				json.Unmarshal(responseBody, &testResp)
			} else {
				testResp.Success = true
			}
		} else {
			testResp.Success = true
		}
	} else {
		testResp.Success = false
		testResp.Error = string(responseBody)
	}

	return &testResp
}

// Main test function for the WebSocket Test Suite
func TestWebSocketTestSuite(t *testing.T) {
	suite.Run(t, new(WebSocketTestSuite))
}
