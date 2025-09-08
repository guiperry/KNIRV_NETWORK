package desktop

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

// DesktopClient represents the main desktop host system
type DesktopClient struct {
	// Core components
	hrmEngine     *HRMEngine
	qrLinkage     *QRLinkageService
	secureBridge  *SecureBridge
	targetSystems *TargetSystemManager
	agentPlugins  *AgentPluginManager
	teeManager    *DesktopTEEManager
	mcpServer     *MCPServer

	// Connection management
	mobileConnections map[string]*MobileConnection
	agentSessions     map[string]*AgentSession

	// Server components
	httpServer *http.Server
	router     *mux.Router
	wsUpgrader websocket.Upgrader

	// Configuration
	desktopID string
	endpoint  string
	publicKey string

	// Synchronization
	mutex   sync.RWMutex
	running bool
}

// MobileConnection represents a connection to a mobile device
type MobileConnection struct {
	DeviceID        string           `json:"device_id"`
	WalletAddress   string           `json:"wallet_address"`
	PublicKey       string           `json:"public_key"`
	Capabilities    []string         `json:"capabilities"`
	SecureChannel   *SecureChannel   `json:"-"`
	LastHeartbeat   time.Time        `json:"last_heartbeat"`
	Status          ConnectionStatus `json:"status"`
	AssignedTargets []string         `json:"assigned_targets"`
}

// AgentSession represents an active agent session
type AgentSession struct {
	SessionID       string              `json:"session_id"`
	UserID          string              `json:"user_id"`
	MobileDeviceID  string              `json:"mobile_device_id,omitempty"`
	TargetSystemID  string              `json:"target_system_id,omitempty"`
	PersonalityData *PersonalityProfile `json:"personality_data"`
	CreatedAt       time.Time           `json:"created_at"`
	LastActivity    time.Time           `json:"last_activity"`
	Status          SessionStatus       `json:"status"`
}

// ConnectionStatus represents the status of a mobile connection
type ConnectionStatus string

const (
	ConnectionStatusConnected    ConnectionStatus = "connected"
	ConnectionStatusDisconnected ConnectionStatus = "disconnected"
	ConnectionStatusReconnecting ConnectionStatus = "reconnecting"
)

// SessionStatus represents the status of an agent session
type SessionStatus string

const (
	SessionStatusActive   SessionStatus = "active"
	SessionStatusInactive SessionStatus = "inactive"
	SessionStatusExpired  SessionStatus = "expired"
)

// PersonalityProfile represents user personality data
type PersonalityProfile struct {
	UserID  string                 `json:"user_id"`
	Metrics map[string]interface{} `json:"metrics"`
}

// SecureChannel represents a secure communication channel
type SecureChannel struct {
	// Implementation details would go here
}

// NewDesktopClient creates a new desktop host instance
func NewDesktopClient(port int) (*DesktopClient, error) {
	desktopID := uuid.New().String()
	endpoint := fmt.Sprintf("http://localhost:%d", port)
	publicKey := "mock_public_key" // In production, generate real key pair

	// Initialize components
	hrmEngine := NewHRMEngine()
	qrLinkage := NewQRLinkageService(desktopID, endpoint, publicKey)

	// Initialize TEE manager first
	teeManager, err := NewDesktopTEEManager("./data", nil) // Use default config
	if err != nil {
		return nil, fmt.Errorf("failed to create TEE manager: %w", err)
	}

	secureBridge := NewSecureBridge(teeManager)
	targetSystems := NewTargetSystemManager()
	agentPlugins := NewAgentPluginManager()

	// Initialize MCP server
	mcpServer := NewMCPServer(nil, hrmEngine) // Will set desktopClient reference later

	// Setup router
	router := mux.NewRouter()

	// WebSocket upgrader
	wsUpgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all origins for development
		},
	}

	host := &DesktopClient{
		hrmEngine:         hrmEngine,
		qrLinkage:         qrLinkage,
		secureBridge:      secureBridge,
		targetSystems:     targetSystems,
		agentPlugins:      agentPlugins,
		teeManager:        teeManager,
		mcpServer:         mcpServer,
		mobileConnections: make(map[string]*MobileConnection),
		agentSessions:     make(map[string]*AgentSession),
		router:            router,
		wsUpgrader:        wsUpgrader,
		desktopID:         desktopID,
		endpoint:          endpoint,
		publicKey:         publicKey,
		running:           false,
	}

	// Set the desktop host reference in MCP server
	mcpServer.desktopClient = host

	// Setup HTTP server
	host.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: router,
	}

	// Setup routes
	host.setupRoutes()

	log.Printf("Desktop host initialized: id=%s, endpoint=%s", desktopID, endpoint)

	return host, nil
}

// Initialize initializes the desktop host and loads required modules
func (dh *DesktopClient) Initialize() error {
	log.Printf("Initializing desktop host...")

	// Load HRM WASM module
	hrmWasmPath := "KNIRVENGINE/agent-core/dist/hrm-cognitive.wasm"
	err := dh.hrmEngine.LoadHRMModule(hrmWasmPath)
	if err != nil {
		log.Printf("Warning: Failed to load HRM WASM module: %v", err)
		// Continue without HRM for now
	}

	// Load HRM weights
	weightsPath := "KNIRVENGINE/agent-core/dist/weights.safetensors"
	err = dh.hrmEngine.LoadWeights(weightsPath)
	if err != nil {
		log.Printf("Warning: Failed to load HRM weights: %v", err)
		// Continue without weights for now
	}

	// Initialize HRM modules
	err = dh.hrmEngine.InitializeModules(8, 4) // 8 L-modules, 4 H-modules
	if err != nil {
		log.Printf("Warning: Failed to initialize HRM modules: %v", err)
	}

	// Start QR linkage service
	go dh.qrLinkage.StartService()

	// Secure bridge is ready for WebSocket connections
	// No explicit start method needed

	log.Printf("Desktop host initialization completed")
	return nil
}

// Start starts the desktop host server
func (dh *DesktopClient) Start() error {
	dh.mutex.Lock()
	defer dh.mutex.Unlock()

	if dh.running {
		return fmt.Errorf("desktop host already running")
	}

	log.Printf("Starting desktop host server on %s", dh.httpServer.Addr)

	// Start MCP server
	if err := dh.mcpServer.Start(); err != nil {
		return fmt.Errorf("failed to start MCP server: %w", err)
	}

	dh.running = true

	// Start HTTP server in goroutine
	go func() {
		if err := dh.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	log.Printf("Desktop host server started successfully with MCP integration")
	return nil
}

// Stop stops the desktop host server
func (dh *DesktopClient) Stop() error {
	dh.mutex.Lock()
	defer dh.mutex.Unlock()

	if !dh.running {
		return fmt.Errorf("desktop host not running")
	}

	log.Printf("Stopping desktop host server...")

	// Stop HTTP server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := dh.httpServer.Shutdown(ctx); err != nil {
		log.Printf("Error shutting down HTTP server: %v", err)
	}

	// Stop MCP server
	if err := dh.mcpServer.Stop(); err != nil {
		log.Printf("Error stopping MCP server: %v", err)
	}

	// Close HRM engine
	if err := dh.hrmEngine.Close(); err != nil {
		log.Printf("Error closing HRM engine: %v", err)
	}

	dh.running = false
	log.Printf("Desktop host server stopped")

	return nil
}

// GetDesktopID returns the desktop host ID
func (dh *DesktopClient) GetDesktopID() string {
	return dh.desktopID
}

// GetHRMEngine returns the HRM engine instance
func (dh *DesktopClient) GetHRMEngine() *HRMEngine {
	return dh.hrmEngine
}

// GetQRLinkage returns the QR linkage service instance
func (dh *DesktopClient) GetQRLinkage() *QRLinkageService {
	return dh.qrLinkage
}

// CreateAgentSession creates a new agent session
func (dh *DesktopClient) CreateAgentSession(userID, mobileDeviceID string) (*AgentSession, error) {
	dh.mutex.Lock()
	defer dh.mutex.Unlock()

	sessionID := uuid.New().String()

	session := &AgentSession{
		SessionID:      sessionID,
		UserID:         userID,
		MobileDeviceID: mobileDeviceID,
		PersonalityData: &PersonalityProfile{
			UserID:  userID,
			Metrics: make(map[string]interface{}),
		},
		CreatedAt:    time.Now(),
		LastActivity: time.Now(),
		Status:       SessionStatusActive,
	}

	dh.agentSessions[sessionID] = session

	log.Printf("Created agent session: session=%s, user=%s, mobile=%s", sessionID, userID, mobileDeviceID)

	return session, nil
}

// HandleMobileLinkage handles mobile device linkage
func (dh *DesktopClient) HandleMobileLinkage(qrSessionID string, mobileData *MobileLinkageData) error {
	// Validate QR session
	session, exists := dh.qrLinkage.GetSession(qrSessionID)
	if !exists {
		return fmt.Errorf("invalid QR session: %s", qrSessionID)
	}

	// Create mobile connection
	mobileConn := &MobileConnection{
		DeviceID:      mobileData.DeviceID,
		WalletAddress: mobileData.WalletAddress,
		PublicKey:     mobileData.PublicKey,
		Capabilities:  mobileData.Capabilities,
		SecureChannel: &SecureChannel{}, // Mock secure channel
		LastHeartbeat: time.Now(),
		Status:        ConnectionStatusConnected,
	}

	dh.mutex.Lock()
	dh.mobileConnections[mobileData.DeviceID] = mobileConn
	dh.mutex.Unlock()

	// If this is a target assignment, assign the target
	if session.LinkageType == LinkageTypeTargetAssignment {
		mobileConn.AssignedTargets = append(mobileConn.AssignedTargets, session.TargetSystemID)
	}

	// Update session status
	err := dh.qrLinkage.UpdateSessionStatus(qrSessionID, LinkageStatusConnected, mobileData.DeviceID)
	if err != nil {
		return fmt.Errorf("failed to update session status: %w", err)
	}

	log.Printf("Mobile device linked: device=%s, session=%s", mobileData.DeviceID, qrSessionID)

	return nil
}

// setupRoutes sets up HTTP routes for the desktop host
func (dh *DesktopClient) setupRoutes() {
	// QR code generation routes
	dh.router.HandleFunc("/api/qr/target-assignment", dh.handleTargetAssignmentQR).Methods("POST")
	dh.router.HandleFunc("/api/qr/transaction-sign", dh.handleTransactionSignQR).Methods("POST")
	dh.router.HandleFunc("/api/qr/session-status/{sessionId}", dh.handleQRSessionStatus).Methods("GET")

	// Mobile connection routes
	dh.router.HandleFunc("/api/mobile/connect", dh.handleMobileConnect).Methods("POST")

	// HRM processing routes
	dh.router.HandleFunc("/api/hrm/process", dh.handleHRMProcess).Methods("POST")
	dh.router.HandleFunc("/api/hrm/info", dh.handleHRMInfo).Methods("GET")

	// Agent session routes
	dh.router.HandleFunc("/api/agent/session", dh.handleCreateAgentSession).Methods("POST")
	dh.router.HandleFunc("/api/agent/ws", dh.handleAgentWebSocket).Methods("GET")

	// MCP (Model Context Protocol) routes
	dh.router.HandleFunc("/api/mcp/ws", dh.mcpServer.HandleWebSocket).Methods("GET")

	// Health check
	dh.router.HandleFunc("/api/health", dh.handleHealth).Methods("GET")
}
