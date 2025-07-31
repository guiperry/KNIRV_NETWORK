package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// MockServer provides mock endpoints for testing the integration framework
type MockServer struct {
	port int
	name string
}

// Health endpoint
func (m *MockServer) healthHandler(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":    "healthy",
		"service":   m.name,
		"timestamp": time.Now().Unix(),
		"port":      m.port,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Generic success response
func (m *MockServer) successHandler(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"success":   true,
		"service":   m.name,
		"method":    r.Method,
		"path":      r.URL.Path,
		"timestamp": time.Now().Unix(),
	}

	// Add mock data based on the endpoint
	switch r.URL.Path {
	case "/llm/register":
		response["tx_hash"] = "mock_tx_hash_" + fmt.Sprintf("%d", time.Now().Unix())
	case "/nrv/errors":
		response["id"] = "mock_error_" + fmt.Sprintf("%d", time.Now().Unix())
		response["error_type"] = "compilation_error"
		response["description"] = "Mock error description"
		response["severity"] = 3
	case "/nrv/skills":
		response["id"] = "mock_skill_" + fmt.Sprintf("%d", time.Now().Unix())
		response["skill_type"] = "code_fixer"
		response["capabilities"] = []string{"javascript", "syntax_repair"}
	case "/nrv/vectors":
		response["id"] = "mock_vector_" + fmt.Sprintf("%d", time.Now().Unix())
		response["target_hash"] = "test_hash_123"
		response["coordinates"] = []float64{1.0, 2.0, 3.0}
	case "/api/v1/agents":
		if r.Method == "POST" {
			response["id"] = "mock_agent_" + fmt.Sprintf("%d", time.Now().Unix())
			response["name"] = "TestAgent"
			response["type"] = "go_plugin"
		} else {
			// Return array for GET requests
			w.Header().Set("Content-Type", "application/json")
			agents := []map[string]interface{}{
				{
					"id":   "mock_agent_1",
					"name": "MockAgent1",
					"type": "test_agent",
				},
			}
			json.NewEncoder(w).Encode(agents)
			return
		}
	case "/wallet/create":
		response["address"] = "mock_address_" + fmt.Sprintf("%d", time.Now().Unix())
		response["mnemonic"] = "mock mnemonic phrase for testing"
	case "/transaction":
		response["tx_hash"] = "mock_tx_" + fmt.Sprintf("%d", time.Now().Unix())
	case "/bridge/transfer":
		response["tx_hash"] = "mock_bridge_" + fmt.Sprintf("%d", time.Now().Unix())
		response["status"] = "pending"
	case "/connectivity/proof":
		response["proof_id"] = "mock_proof_" + fmt.Sprintf("%d", time.Now().Unix())
	case "/skill/invoke":
		response["tx_hash"] = "mock_skill_invoke_" + fmt.Sprintf("%d", time.Now().Unix())
	case "/blockchain/state":
		response["latest_block"] = "mock_block_123"
		response["total_transactions"] = 1000
	default:
		// Handle agent execution endpoints
		if strings.Contains(r.URL.Path, "/agents/") && strings.Contains(r.URL.Path, "/execute") {
			response["execution_id"] = "mock_execution_" + fmt.Sprintf("%d", time.Now().Unix())
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Balance endpoint
func (m *MockServer) balanceHandler(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"balance": "1000000000", // 1000 NRN
		"address": "mock_address",
		"service": m.name,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Status endpoint
func (m *MockServer) statusHandler(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"node_id":    "mock_node_" + m.name,
		"peer_count": 5,
		"status":     "active",
		"service":    m.name,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Peers endpoint
func (m *MockServer) peersHandler(w http.ResponseWriter, r *http.Request) {
	peers := []map[string]interface{}{
		{
			"id":      "peer1",
			"address": "192.168.1.100:8080",
			"status":  "active",
		},
		{
			"id":      "peer2",
			"address": "192.168.1.101:8080",
			"status":  "active",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(peers)
}

// Resolve endpoint
func (m *MockServer) resolveHandler(w http.ResponseWriter, r *http.Request) {
	vectors := []map[string]interface{}{
		{
			"target_hash": "test_hash_123",
			"coordinates": []float64{1.0, 2.0, 3.0},
			"metadata": map[string]interface{}{
				"type": "test_vector",
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(vectors)
}

// Bridge status endpoint
func (m *MockServer) bridgeStatusHandler(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":    "completed",
		"tx_hash":   r.URL.Query().Get("tx_hash"),
		"timestamp": time.Now().Unix(),
		"service":   m.name,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Setup routes for the mock server
func (m *MockServer) setupRoutes() *http.ServeMux {
	mux := http.NewServeMux()

	// Health endpoints
	mux.HandleFunc("/health", m.healthHandler)
	mux.HandleFunc("/api/v1/health", m.healthHandler)

	// Generic endpoints that return success
	mux.HandleFunc("/llm/register", m.successHandler)
	mux.HandleFunc("/nrv/errors", m.successHandler)
	mux.HandleFunc("/nrv/skills", m.successHandler)
	mux.HandleFunc("/nrv/vectors", m.successHandler)
	mux.HandleFunc("/api/v1/agents", m.successHandler)
	mux.HandleFunc("/wallet/create", m.successHandler)
	mux.HandleFunc("/transaction", m.successHandler)
	mux.HandleFunc("/bridge/transfer", m.successHandler)
	mux.HandleFunc("/connectivity/proof", m.successHandler)
	mux.HandleFunc("/skill/invoke", m.successHandler)
	mux.HandleFunc("/send_txn", m.successHandler)
	mux.HandleFunc("/faucet/fund", m.successHandler)

	// Special endpoints
	mux.HandleFunc("/wallet/", m.balanceHandler)
	mux.HandleFunc("/status", m.statusHandler)
	mux.HandleFunc("/peers", m.peersHandler)
	mux.HandleFunc("/nrv/resolve/", m.resolveHandler)
	mux.HandleFunc("/bridge/status", m.bridgeStatusHandler)

	// Catch-all for other endpoints
	mux.HandleFunc("/", m.successHandler)

	return mux
}

// Start the mock server
func (m *MockServer) Start() {
	mux := m.setupRoutes()

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", m.port),
		Handler: mux,
	}

	log.Printf("Starting mock %s server on port %d", m.name, m.port)

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Failed to start mock %s server: %v", m.name, err)
	}
}

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: go run mock_server.go <port> <service_name>")
		fmt.Println("Example: go run mock_server.go 8080 KNIRVCHAIN")
		os.Exit(1)
	}

	port, err := strconv.Atoi(os.Args[1])
	if err != nil {
		log.Fatalf("Invalid port number: %v", err)
	}

	serviceName := os.Args[2]

	server := &MockServer{
		port: port,
		name: serviceName,
	}

	server.Start()
}
