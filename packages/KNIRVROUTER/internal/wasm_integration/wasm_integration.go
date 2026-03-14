// WASM Integration for Revolutionary Embedded KNIRVCHAIN
package wasm_integration

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"time"

	"KNIRVROUTER/internal/wasm_loader"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// WASMIntegration manages the WASM-based KNIRVCHAIN integration
type WASMIntegration struct {
	wasmChain   *wasm_loader.WASMKNIRVChain
	assetsDir   string
	initialized bool
}

// NewWASMIntegration creates a new WASM integration instance
func NewWASMIntegration(assetsDir string) *WASMIntegration {
	return &WASMIntegration{
		assetsDir:   assetsDir,
		initialized: false,
	}
}

// Initialize initializes the WASM integration
func (wi *WASMIntegration) Initialize() error {
	log.Printf("🚀 Initializing Revolutionary WASM Integration...")

	// Load WASM KNIRVCHAIN
	wasmChain, err := wasm_loader.LoadWASMKNIRVChain(wi.assetsDir)
	if err != nil {
		return fmt.Errorf("failed to load WASM KNIRVCHAIN: %v", err)
	}

	// Initialize the WASM chain
	if err := wasmChain.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize WASM KNIRVCHAIN: %v", err)
	}

	wi.wasmChain = wasmChain
	wi.initialized = true

	// Log version and build info
	if version, err := wasmChain.GetVersion(); err == nil {
		log.Printf("📦 WASM KNIRVCHAIN Version: %s", version)
	}

	if buildInfo, err := wasmChain.GetBuildInfo(); err == nil {
		log.Printf("🔧 WASM KNIRVCHAIN Build: %s", buildInfo)
	}

	if skillCount, err := wasmChain.GetSkillCount(); err == nil {
		log.Printf("🧠 WASM KNIRVCHAIN Skills: %d registered", skillCount)
	}

	log.Printf("✅ Revolutionary WASM Integration initialized successfully")
	return nil
}

// RegisterRoutes registers HTTP routes for WASM endpoints
func (wi *WASMIntegration) RegisterRoutes(router *mux.Router) {
	// Revolutionary WASM-based /invoke endpoint
	router.HandleFunc("/wasm/invoke", wi.handleWASMInvoke).Methods("POST")

	// WASM status and info endpoints
	router.HandleFunc("/wasm/status", wi.handleWASMStatus).Methods("GET")
	router.HandleFunc("/wasm/version", wi.handleWASMVersion).Methods("GET")
	router.HandleFunc("/wasm/skills/count", wi.handleWASMSkillCount).Methods("GET")

	log.Printf("🔗 Revolutionary WASM endpoints registered")
}

// handleWASMInvoke handles the revolutionary WASM-based skill invocation
func (wi *WASMIntegration) handleWASMInvoke(w http.ResponseWriter, r *http.Request) {
	if !wi.initialized {
		http.Error(w, "WASM integration not initialized", http.StatusServiceUnavailable)
		return
	}

	var request wasm_loader.SkillInvocationRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	// Generate invocation ID if not provided
	if request.InvocationID == "" {
		request.InvocationID = uuid.New().String()
	}

	// Set timestamp if not provided
	if request.Timestamp == 0 {
		request.Timestamp = time.Now().Unix()
	}

	// Set default priority if not provided
	if request.Priority == "" {
		request.Priority = "normal"
	}

	log.Printf("🎯 Revolutionary WASM skill invocation: %s (agent: %s, URI: %s)",
		request.InvocationID, request.AgentID, request.SkillURI)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Invoke skill via WASM
	response, err := wi.wasmChain.InvokeSkill(ctx, &request)
	if err != nil {
		log.Printf("❌ WASM skill invocation failed: %v", err)
		http.Error(w, fmt.Sprintf("Skill invocation failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Set response headers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-KNIRV-Response-Format", "wasm-skill-response-v1")
	w.Header().Set("X-KNIRV-Invocation-ID", response.InvocationID)
	w.Header().Set("X-KNIRV-Status", response.Status)
	w.Header().Set("X-KNIRV-Engine", "wasm")

	// Return JSON response
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("❌ Failed to encode response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}

	log.Printf("✅ Revolutionary WASM skill invocation completed: %s (%dms)",
		response.InvocationID, response.ExecutionTime)
}

// handleWASMStatus handles WASM status requests
func (wi *WASMIntegration) handleWASMStatus(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"initialized": wi.initialized,
		"engine":      "wasm",
		"timestamp":   time.Now().Unix(),
	}

	if wi.initialized && wi.wasmChain != nil {
		status["wasm_initialized"] = wi.wasmChain.IsInitialized()

		if skillCount, err := wi.wasmChain.GetSkillCount(); err == nil {
			status["skill_count"] = skillCount
		}

		if version, err := wi.wasmChain.GetVersion(); err == nil {
			status["version"] = version
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// handleWASMVersion handles WASM version requests
func (wi *WASMIntegration) handleWASMVersion(w http.ResponseWriter, r *http.Request) {
	if !wi.initialized {
		http.Error(w, "WASM integration not initialized", http.StatusServiceUnavailable)
		return
	}

	version, err := wi.wasmChain.GetVersion()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get version: %v", err), http.StatusInternalServerError)
		return
	}

	buildInfo, err := wi.wasmChain.GetBuildInfo()
	if err != nil {
		buildInfo = "Unknown"
	}

	response := map[string]string{
		"version":    version,
		"build_info": buildInfo,
		"engine":     "wasm",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleWASMSkillCount handles WASM skill count requests
func (wi *WASMIntegration) handleWASMSkillCount(w http.ResponseWriter, r *http.Request) {
	if !wi.initialized {
		http.Error(w, "WASM integration not initialized", http.StatusServiceUnavailable)
		return
	}

	skillCount, err := wi.wasmChain.GetSkillCount()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get skill count: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"skill_count": skillCount,
		"engine":      "wasm",
		"timestamp":   time.Now().Unix(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// StartHTTPServer starts the WASM integration HTTP server
func (wi *WASMIntegration) StartHTTPServer(port string) error {
	if !wi.initialized {
		return fmt.Errorf("WASM integration not initialized")
	}

	router := mux.NewRouter()

	// Register WASM routes
	wi.RegisterRoutes(router)

	// Add CORS headers
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	})

	server := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	log.Printf("🚀 Starting Revolutionary WASM HTTP server on port %s", port)
	log.Printf("🎯 WASM /invoke endpoint: http://localhost:%s/wasm/invoke", port)
	log.Printf("📊 WASM status endpoint: http://localhost:%s/wasm/status", port)

	return server.ListenAndServe()
}

// Shutdown gracefully shuts down the WASM integration
func (wi *WASMIntegration) Shutdown() error {
	if !wi.initialized {
		return nil
	}

	log.Printf("🛑 Shutting down Revolutionary WASM Integration...")

	if wi.wasmChain != nil {
		if err := wi.wasmChain.Shutdown(); err != nil {
			log.Printf("❌ Error shutting down WASM chain: %v", err)
		}
	}

	wi.initialized = false
	log.Printf("✅ Revolutionary WASM Integration shutdown complete")
	return nil
}

// GetAssetsPath returns the path to WASM assets
func GetAssetsPath() string {
	// Default assets path relative to KNIRVROUTER
	return filepath.Join(".", "assets")
}
