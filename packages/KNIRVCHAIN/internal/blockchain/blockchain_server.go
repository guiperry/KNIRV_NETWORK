package blockchain

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"KNIRVCHAIN/internal/errors"
	pb "KNIRVCHAIN/internal/protocol/proto"
	"KNIRVCHAIN/internal/types"
	"KNIRVCHAIN/internal/uri"
	"KNIRVCHAIN/internal/utils"

	"github.com/gorilla/mux"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/protobuf/proto"
)

type BlockchainServer struct {
	port               uint64
	socketPath         string
	BlockchainPtr      *BlockchainStruct
	server             *http.Server
	db                 *LevelDB
	discoveryManager   DiscoveryService
	p2pPort            int
	testMode           bool                 // Flag indicating if running in test mode
	consensusManager   *P2PConsensusManager // Reference to consensus manager for network pause checking
	validationProofMu  sync.Mutex
	checkpointStatusMu sync.RWMutex
	checkpointStatus   func() interface{}
}

func (bcs *BlockchainServer) SetCheckpointStatusProvider(provider func() interface{}) {
	bcs.checkpointStatusMu.Lock()
	defer bcs.checkpointStatusMu.Unlock()
	bcs.checkpointStatus = provider
}

func (bcs *BlockchainServer) handleCheckpointStatus(w http.ResponseWriter, _ *http.Request) {
	bcs.checkpointStatusMu.RLock()
	provider := bcs.checkpointStatus
	bcs.checkpointStatusMu.RUnlock()
	w.Header().Set("Content-Type", "application/json")
	if provider == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"enabled": false, "error": "checkpoint runtime not configured"})
		return
	}
	_ = json.NewEncoder(w).Encode(provider())
}

// SetListenAddr overrides the HTTP server address after Prepare.
func (bcs *BlockchainServer) SetListenAddr(addr string) {
	if bcs.server == nil {
		bcs.server = &http.Server{}
	}
	bcs.server.Addr = addr
}

// ListenAddr returns the current HTTP server address.
func (bcs *BlockchainServer) ListenAddr() string {
	if bcs.server == nil {
		return ""
	}
	return bcs.server.Addr
}

// SetDiscoveryManager attaches a discovery manager after construction.
func (bcs *BlockchainServer) SetDiscoveryManager(discoveryMgr DiscoveryService) {
	bcs.discoveryManager = discoveryMgr
}

// isNetworkPaused checks if the network is currently paused for maintenance or failover
func (bcs *BlockchainServer) isNetworkPaused() bool {
	if bcs.consensusManager != nil {
		return bcs.consensusManager.IsNetworkPaused()
	}
	return false
}

// checkNetworkPauseAndRejectIfPaused checks if network is paused and returns HTTP error if so
func (bcs *BlockchainServer) checkNetworkPauseAndRejectIfPaused(w http.ResponseWriter, endpoint string) bool {
	if bcs.isNetworkPaused() {
		log.Printf("[FAILOVER] Network is paused - rejecting %s transaction", endpoint)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":    "Service temporarily unavailable",
			"message":  "Network is currently under maintenance or failover is in progress. Please try again in a few minutes.",
			"paused":   true,
			"endpoint": endpoint,
		})
		return true
	}
	return false
}

var pendingRegistrations = struct {
	sync.RWMutex
	m map[string]*pendingRegistration // Key: pendingTransactionHash
}{m: make(map[string]*pendingRegistration)}

type pendingRegistration struct {
	Transaction *Transaction
	CreatedAt   time.Time
}

// MCPPrepareCapabilityRegistrationRequest represents the request to prepare a capability registration
type MCPPrepareCapabilityRegistrationRequest struct {
	FromAddress    string                 `json:"from_address"`
	CapabilityType string                 `json:"capability_type"`
	Descriptor     map[string]interface{} `json:"descriptor"`
	DesiredName    string                 `json:"desired_name"`
	Description    string                 `json:"description"`
	Fee            uint64                 `json:"fee"`
}

// UnsignedTransactionDetails represents the details of an unsigned transaction
type UnsignedTransactionDetails struct {
	From      string      `json:"from"`
	To        string      `json:"to,omitempty"`
	Value     uint64      `json:"value"`
	Data      interface{} `json:"data"` // Contains the MCPRegisterCapabilityData with nested CapabilityDescriptor
	Timestamp int64       `json:"timestamp"`
	Fee       uint64      `json:"fee"`
	Type      string      `json:"type"`
}

// MCPPrepareCapabilityRegistrationResponse represents the response to a prepare capability registration request
type MCPPrepareCapabilityRegistrationResponse struct {
	CapabilityID                   string                     `json:"capability_id"`
	UnsignedTransactionPayloadHash string                     `json:"unsigned_transaction_payload_hash"`
	TransactionDetailsForSigning   UnsignedTransactionDetails `json:"transaction_details_for_signing"`
	Message                        string                     `json:"message"`
	EstimatedGasFee                uint64                     `json:"estimated_gas_fee,omitempty"`
	PendingTransactionHash         string                     `json:"pending_transaction_hash,omitempty"` // Hash of the transaction the client needs to sign
	FullDescriptor                 interface{}                `json:"full_descriptor,omitempty"`          // Complete descriptor with the server-generated ID
}

const pendingRegistrationTTL = 5 * time.Minute // TTL for pending registrations

// cleanupExpiredRegistrations periodically removes expired pending registrations
// This function should be called as a goroutine
var cleanupRunning sync.Once

func cleanupExpiredRegistrations() {
	// Ensure this only runs once
	cleanupRunning.Do(func() {
		log.Println("[INFO] Starting pending registration cleanup goroutine")
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			now := time.Now()
			expiredHashes := []string{}

			// Find expired registrations
			pendingRegistrations.RLock()
			for hash, reg := range pendingRegistrations.m {
				if now.Sub(reg.CreatedAt) > pendingRegistrationTTL {
					expiredHashes = append(expiredHashes, hash)
				}
			}
			pendingRegistrations.RUnlock()

			// Remove expired registrations
			if len(expiredHashes) > 0 {
				pendingRegistrations.Lock()
				for _, hash := range expiredHashes {
					delete(pendingRegistrations.m, hash)
				}
				pendingRegistrations.Unlock()
				log.Printf("[INFO] Cleaned up %d expired pending registrations", len(expiredHashes))
			}
		}
	})
}

// Stop gracefully shuts down the blockchain server
func (bcs *BlockchainServer) Stop(ctx context.Context) error {
	var err error

	// Shutdown HTTP server if running
	if bcs.server != nil {
		log.Println("Shutting down HTTP server...")

		// Create a done channel to track HTTP server shutdown completion
		httpShutdownDone := make(chan struct{})

		go func() {
			if shutdownErr := bcs.server.Shutdown(ctx); shutdownErr != nil {
				err = fmt.Errorf("HTTP server shutdown error: %w", shutdownErr)
				log.Printf("Error during HTTP server shutdown: %v", shutdownErr)
			} else {
				log.Println("HTTP server shutdown completed successfully")
			}
			close(httpShutdownDone)
		}()

		// Wait for shutdown to complete or context to be canceled
		select {
		case <-httpShutdownDone:
			// Shutdown completed normally
		case <-ctx.Done():
			log.Println("WARNING: HTTP server shutdown context canceled before completion")
		}
	} else {
		log.Println("HTTP server was not initialized")
	}

	// Stop discovery manager if exists
	if bcs.discoveryManager != nil {
		log.Println("Attempting to stop discovery manager...")

		// Create a done channel for discovery manager shutdown
		discoveryDone := make(chan struct{})

		go func() {
			// Call Close() directly
			bcs.discoveryManager.Close()
			close(discoveryDone)
		}()

		// Wait for discovery manager to close or context to be canceled
		select {
		case <-discoveryDone:
			log.Println("Discovery manager stopped successfully.")
		case <-ctx.Done():
			log.Println("WARNING: Discovery manager shutdown context canceled before completion")
		case <-time.After(5 * time.Second):
			log.Println("WARNING: Discovery manager shutdown timed out after 5 seconds")
		}
	} else {
		log.Println("Discovery manager was not initialized.")
	}

	// Log final status
	if err != nil {
		log.Printf("Graceful shutdown completed with errors: %v", err)
	} else {
		log.Println("Graceful shutdown completed successfully.")
	}

	// Return only the potential error from the HTTP server shutdown
	return err
}

// Define a struct for the request body
type URIRequest struct {
	DesiredID string `json:"desired_id"` // The ID the requestor wants to use
}

// Add this struct for the faucet request body
type FaucetRequest struct {
	Address string `json:"address"`
	Amount  uint64 `json:"amount"`
}

// ServerInfo represents information about the server
type ServerInfo struct {
	HTTPPort    uint64   `json:"http_port"`
	P2PPort     int      `json:"p2p_port"`
	ChainID     string   `json:"chain_id"`
	PeerID      string   `json:"dev_id,omitempty"`
	Multiaddrs  []string `json:"multiaddrs,omitempty"`
	Version     string   `json:"version"`
	Connections int      `json:"connections,omitempty"`
}

// corsMiddleware adds CORS headers to the response.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		// Allow all origins for simplicity, or specify your frontend's origin in production.
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE, PATCH") // Added PATCH
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

		// Handle preflight requests (OPTIONS method)
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Call the next handler in the chain
		next.ServeHTTP(w, r)
	})
}

// Modify NewBlockchainServer to accept socketPath and DiscoveryService and p2pPort
func NewBlockchainServer(port uint64, socketPath string, blockchain *BlockchainStruct, db *LevelDB, discoveryMgr DiscoveryService, p2pPort int) *BlockchainServer {
	// Set the server port globally so it can be accessed from other parts of the code
	utils.SetServerPort(port) // Keep this if needed elsewhere

	return &BlockchainServer{
		port:             port,
		socketPath:       socketPath,
		BlockchainPtr:    blockchain,
		db:               db,
		discoveryManager: discoveryMgr,   // Store the passed-in manager
		p2pPort:          p2pPort,        // Store the correct P2P port
		server:           &http.Server{}, // Initialize empty server to prevent nil dereference
	}
}

// Prepare initializes the server, finds the port, and updates ChainID if port-dependent.
// It does NOT start listening. Call StartListenAndServe after this.
func (bcs *BlockchainServer) Prepare() (uint64, error) {
	// Ensure discovery manager exists (was passed in)
	if bcs.discoveryManager == nil {
		// This can be nil if called early in main.go before discoveryMgr is ready.
		// DiscoveryManager will be assigned later.
		// log.Println("BlockchainServer.Prepare: DiscoveryManager is not yet assigned.")
	}
	if bcs.BlockchainPtr == nil {
		return 0, fmt.Errorf("blockchain pointer not provided for BlockchainServer prepare")
	}
	// Create a new HTTP server instance using Gorilla Mux for path parameters
	mux := mux.NewRouter()
	mux.HandleFunc("/chain", bcs.handleGetChain)
	mux.HandleFunc("/reflections", bcs.handleGetReflections)
	mux.HandleFunc("/add_reflection", bcs.handleAddReflection)
	mux.HandleFunc("/block", bcs.handleReceiveBlock)
	mux.HandleFunc("/transaction", bcs.HandleReceiveTransaction)
	mux.HandleFunc("/api/v1/validation-proofs/mint", bcs.handleValidationProofMint)
	mux.HandleFunc("/api/v1/event-bundles/mint", bcs.handleEventBundleMint)
	mux.HandleFunc("/api/v1/event-bundles/{event_id}", bcs.handleEventBundleGet).Methods(http.MethodGet)
	mux.HandleFunc("/txn_pool", bcs.handleGetTransactionPool)
	mux.HandleFunc("/proof/tx/", bcs.handleTxAccumProof)
	mux.HandleFunc("/checkpoint/status", bcs.handleCheckpointStatus).Methods(http.MethodGet)
	mux.HandleFunc("/ping", bcs.handlePing)
	mux.HandleFunc("/health", bcs.handleHealth)
	mux.HandleFunc("/uriGenerator", bcs.handleURIGenerator)
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/info", bcs.handleServerInfo)
	mux.HandleFunc("/devs", bcs.handleGetPeers)
	mux.HandleFunc("/p2p/connect", bcs.handleP2PConnect) // Register the new P2P connect handler
	mux.HandleFunc("/wallet/info", bcs.handleWalletInfo) // New handler

	// Add the test-only faucet handler
	mux.HandleFunc("/test/faucet", bcs.handleTestFaucet)

	// Add MCP API endpoints
	mux.HandleFunc("/mcp/capability/prepare_registration", bcs.handleMCPPrepareCapabilityRegistration) // New client-side signing flow
	mux.HandleFunc("/mcp/capability/register", bcs.handleMCPRegisterCapability)                        // Legacy endpoint (deprecated)
	mux.HandleFunc("/mcp/capability/register/initiate", bcs.handleMCPRegisterCapabilityInitiate)       // Legacy two-step registration (deprecated)
	mux.HandleFunc("/mcp/capability/register/finalize", bcs.handleMCPRegisterCapabilityFinalize)       // Legacy two-step registration (deprecated)
	mux.HandleFunc("/mcp/capability/invoke", bcs.handleMCPInvokeCapability)
	mux.HandleFunc("/mcp/capability/update", bcs.handleMCPUpdateCapability)
	mux.HandleFunc("/mcp/capability/get", bcs.handleMCPGetCapability)
	mux.HandleFunc("/mcp/capability/list", bcs.handleMCPListCapabilities)

	// Add Context Retrieval API endpoints (Phase 4)
	mux.HandleFunc("/mcp/context/", bcs.handleMCPGetContextRecordByID)                       // Handles /mcp/context/{id}
	mux.HandleFunc("/mcp/capability/contexts/", bcs.handleMCPListContextRecordsByCapability) // Handles /mcp/capability/contexts/{capability_id}
	mux.HandleFunc("/mcp/contexts", bcs.handleMCPListAllContextRecords)                      // Handles /mcp/contexts (list all context records)

	// Add NFT endpoints
	mux.HandleFunc("/nft/list", bcs.handleNFTList)
	mux.HandleFunc("/nft/upload", bcs.handleNFTUpload)
	mux.HandleFunc("/nft/attach-capability", bcs.handleNFTCapabilityAttachment)

	// /objects and /assets serve NFT data — /nft/list is the canonical
	// source; these aliases exist so the gateway proxy paths
	// /api/objects and /api/assets resolve correctly.
	mux.HandleFunc("/objects", bcs.handleNFTList)
	mux.HandleFunc("/assets", bcs.handleNFTList)
	mux.HandleFunc("/nft/capability-history/{badgeId}", bcs.handleNFTCapabilityHistory)

	// Add Resource Capability Management API endpoints (Phase 3)
	mux.HandleFunc("/agent/resource-capability/add", bcs.handleAddResourceCapability)
	mux.HandleFunc("/agent/resource-capability/link", bcs.handleLinkResourceCapability)
	mux.HandleFunc("/agent/resource-capability/group", bcs.handleCreateResourceCapabilityGroup)
	mux.HandleFunc("/agent/resource-capability/list", bcs.handleListResourceCapabilities)
	mux.HandleFunc("/agent/resource-capability/invoke", bcs.handleInvokeResourceCapability)
	mux.HandleFunc("/agent/resource-capability/history", bcs.handleGetResourceCapabilityHistory)

	// Add Agent Facts API endpoints (Phase 4)
	mux.HandleFunc("/agent/agent-facts/", bcs.HandleGetAgentFacts)         // Handles /agent/agent-facts/{id}
	mux.HandleFunc("/agent/capabilities/", bcs.HandleGetAgentCapabilities) // Handles /agent/capabilities/{agentId}
	mux.HandleFunc("/agent/capability/invoke", bcs.HandleInvokeAgentCapability)

	// Add internal API endpoints for Node.js services
	mux.HandleFunc("/internal/dht/findResource", bcs.handleInternalDHTFindResource)
	mux.HandleFunc("/internal/dht/announceResource", bcs.handleInternalDHTAnnounceResource)
	mux.HandleFunc("/internal/db/getCapability", bcs.handleInternalDBGetCapability)
	mux.HandleFunc("/internal/db/idExists", bcs.handleInternalDBIDExists)

	// P2P callback endpoints — called by KNIRVGATEWAY to forward gossiped blocks/txs and sync requests
	mux.HandleFunc("/internal/p2p/received-block", bcs.handleP2PReceivedBlock)
	mux.HandleFunc("/internal/p2p/received-tx", bcs.handleP2PReceivedTx)
	mux.HandleFunc("/internal/chain/sync", bcs.handleChainSync)

	// Add PoAu-D consensus API endpoints
	mux.HandleFunc("/poaud/enable", bcs.EnablePoAuD)
	mux.HandleFunc("/poaud/disable", bcs.DisablePoAuD)
	mux.HandleFunc("/poaud/status", bcs.GetPoAuDStatus)
	mux.HandleFunc("/poaud/network-authors/add", bcs.AddNetworkAuthor)
	mux.HandleFunc("/poaud/network-authors/remove", bcs.RemoveNetworkAuthor)
	mux.HandleFunc("/poaud/network-authors", bcs.GetNetworkAuthors)

	// NANDA-ANS service removed as per refactor plan

	actualPort := bcs.port
	if bcs.socketPath == "" {
		for !utils.IsPortAvailable(actualPort) {
			log.Printf("Port %d is in use, trying next port for chain %s", actualPort, bcs.BlockchainPtr.ChainID)
			actualPort++
		}

		if actualPort != bcs.port {
			log.Printf("Chain %s: HTTP server will use port %d instead of configured port %d", bcs.BlockchainPtr.ChainID, actualPort, bcs.port)
			bcs.port = actualPort
			// SetServerPort(actualPort) // Avoid global SetServerPort if possible, rely on returned port.
		}
	}

	// Update Blockchain's ChainID if it's port-dependent and port changed.
	// This applies to root/bootnodes whose ChainID might be KNIRVCHAIN-ROOT-<port>.
	// Peer ChainIDs are typically fixed from installation and should not change here.
	// The check for "KNIRVCHAIN-ROOT" prefix handles this distinction.
	if strings.HasPrefix(bcs.BlockchainPtr.ChainID, "KNIRVCHAIN-ROOT") {
		newChainID := fmt.Sprintf("KNIRVCHAIN-ROOT%d", actualPort)
		if bcs.BlockchainPtr.ChainID != newChainID {
			log.Printf("Updating blockchain ChainID from %s to %s due to HTTP port change to %d", bcs.BlockchainPtr.ChainID, newChainID, actualPort)
			bcs.BlockchainPtr.ChainID = newChainID
		}
	}

	bcs.server = &http.Server{
		// Listen on localhost only if NGINX (or our Go reverse proxy) is the public entry point
		// We'll need a way to know if the reverse proxy is active.
		// For now, let's assume if a specific config flag is set, or we can pass it.
		// Let's modify this to be configurable. For now, we'll keep the previous logic
		// and main.go will decide the listen address for the BlockchainServer.
		// The Addr field will be set by main.go before calling StartListenAndServe.
		// Addr:    ":" + strconv.Itoa(int(actualPort)), // This will be set by the caller
		Handler: corsMiddleware(mux), // Wrap the mux with CORS middleware
	}
	if bcs.socketPath != "" {
		log.Printf("BlockchainServer for chain %s prepared for socket %s", bcs.BlockchainPtr.ChainID, bcs.socketPath)
	} else {
		log.Printf("BlockchainServer for chain %s prepared for port %d", bcs.BlockchainPtr.ChainID, actualPort)
	}
	return actualPort, nil
}

// handleInternalDHTFindResource handles internal requests to find a resource on the DHT
func (bcs *BlockchainServer) handleInternalDHTFindResource(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if bcs.discoveryManager == nil {
		http.Error(w, "Discovery service not available", http.StatusInternalServerError)
		return
	}

	id := r.URL.Query().Get("id")
	resourceTypeStr := r.URL.Query().Get("type") // e.g., "chain", "capability"

	if id == "" || resourceTypeStr == "" {
		http.Error(w, "Missing 'id' or 'type' query parameter", http.StatusBadRequest)
		return
	}

	resourceType := types.DiscoveryResourceType(resourceTypeStr)
	if resourceType == "" {
		http.Error(w, "Invalid resource type", http.StatusBadRequest)
		return
	}

	// Use a context with timeout for the DHT query
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	providers, err := bcs.discoveryManager.FindResource(ctx, id, resourceType)
	if err != nil {
		// Log the error but return 404 if no providers found
		if strings.Contains(err.Error(), "no providers found") || strings.Contains(err.Error(), "failed to find any dev in table") {
			http.Error(w, fmt.Sprintf("Resource '%s' of type '%s' not found on DHT", id, resourceTypeStr), http.StatusNotFound)
		} else {
			log.Printf("Error finding resource '%s' on DHT: %v", id, err)
			http.Error(w, "Internal server error during DHT lookup", http.StatusInternalServerError)
		}
		return
	}

	// Return the provider information (e.g., multiaddrs)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(providers) // Assuming providers is a slice of peer.AddrInfo
}

// handleInternalDHTAnnounceResource handles internal requests to announce a resource on the DHT
func (bcs *BlockchainServer) handleInternalDHTAnnounceResource(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if bcs.discoveryManager == nil {
		http.Error(w, "Discovery service not available", http.StatusInternalServerError)
		return
	}

	// Parse the request body
	var requestBody struct {
		ID           string `json:"id"`
		Type         string `json:"type"`
		Multiaddress string `json:"multiaddress,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	if requestBody.ID == "" || requestBody.Type == "" {
		http.Error(w, "Missing 'id' or 'type' in request body", http.StatusBadRequest)
		return
	}

	resourceType := types.DiscoveryResourceType(requestBody.Type)
	if resourceType == "" {
		http.Error(w, "Invalid resource type", http.StatusBadRequest)
		return
	}

	// Use a context with timeout for the DHT announcement
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// Announce the resource on the DHT
	err := bcs.discoveryManager.AnnounceMintedResource(ctx, requestBody.ID, resourceType)
	if err != nil {
		log.Printf("Error announcing resource '%s' on DHT: %v", requestBody.ID, err)
		http.Error(w, fmt.Sprintf("Failed to announce resource: %v", err), http.StatusInternalServerError)
		return
	}

	// Return success
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": fmt.Sprintf("Resource '%s' of type '%s' announced on DHT", requestBody.ID, requestBody.Type),
	})
}

// handleInternalDBGetCapability handles internal requests to get a capability from the database
func (bcs *BlockchainServer) handleInternalDBGetCapability(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "Missing 'id' query parameter", http.StatusBadRequest)
		return
	}

	// Get the capability from the database
	capability, err := bcs.db.GetCapabilityByID(id)
	if err != nil {
		http.Error(w, fmt.Sprintf("Capability '%s' not found in DB: %v", id, err), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(capability) // Assuming capability is a serializable struct/map
}

// handleInternalDBIDExists handles internal requests to check if an ID exists in the blockchain (DB)
func (bcs *BlockchainServer) handleInternalDBIDExists(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "Missing 'id' query parameter", http.StatusBadRequest)
		return
	}

	// Check if the ID exists in blocks or the transaction pool
	// Note: This only checks the local node's view of the blockchain.
	// For true global uniqueness, a DHT check is also necessary.
	bcs.BlockchainPtr.Lock()
	existsInBlocks := bcs.BlockchainPtr.CheckIfIDExistsInBlocks(id)
	existsInPool := bcs.BlockchainPtr.CheckIfIDExistsInTransactionPool(id)
	bcs.BlockchainPtr.Unlock()

	exists := existsInBlocks || existsInPool

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"exists": exists})
}

// StartListenAndServe starts the HTTP server. Call after Prepare().
func (bcs *BlockchainServer) StartListenAndServe() error {
	if bcs.server == nil {
		return fmt.Errorf("server not prepared, call Prepare() first for chain %s", bcs.BlockchainPtr.ChainID)
	}

	// Set up server timeouts to ensure it can shut down gracefully
	bcs.server.ReadTimeout = 10 * time.Second
	bcs.server.WriteTimeout = 30 * time.Second
	bcs.server.IdleTimeout = 120 * time.Second

	var listener net.Listener
	var err error

	if bcs.socketPath != "" {
		if err := os.RemoveAll(bcs.socketPath); err != nil {
			return fmt.Errorf("failed to remove existing socket: %w", err)
		}
		if err := os.MkdirAll(filepath.Dir(bcs.socketPath), 0755); err != nil && !os.IsExist(err) {
			return fmt.Errorf("failed to create socket directory: %w", err)
		}
		listener, err = net.Listen("unix", bcs.socketPath)
		if err != nil {
			return fmt.Errorf("failed to listen on socket: %w", err)
		}
		if err := os.Chmod(bcs.socketPath, 0666); err != nil {
			return fmt.Errorf("failed to set socket permissions: %w", err)
		}
		log.Printf("Starting HTTP server listener for chain %s on socket: %s", bcs.BlockchainPtr.ChainID, bcs.socketPath)
	} else {
		// If Addr wasn't set during Prepare (e.g. if we change logic), set it now.
		if bcs.server.Addr == "" {
			bcs.server.Addr = ":" + strconv.Itoa(int(bcs.port)) // Default if not overridden
		}
		log.Printf("Starting HTTP server listener for chain %s on port: %d", bcs.BlockchainPtr.ChainID, bcs.port)
		listener, err = net.Listen("tcp", bcs.server.Addr)
		if err != nil {
			return fmt.Errorf("failed to listen: %w", err)
		}
	}

	// Start the server
	err = bcs.server.Serve(listener)

	// Check for errors other than server closed
	if err != nil && err != http.ErrServerClosed {
		log.Printf("HTTP server ListenAndServe error for chain %s: %v", bcs.BlockchainPtr.ChainID, err)
		return fmt.Errorf("failed to start HTTP server listener for chain %s: %w", bcs.BlockchainPtr.ChainID, err)
	}

	log.Printf("HTTP server listener for chain %s stopped.", bcs.BlockchainPtr.ChainID)
	return nil
}

func (bcs *BlockchainServer) handlePing(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	log.Printf("Ping request from %s", r.RemoteAddr)

	// Simple ping response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
		"port":   strconv.Itoa(int(bcs.port)),
	})
}

func (bcs *BlockchainServer) handleGetChain(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Create a simplified blockchain structure for serialization
	type SerializableBlockchain struct {
		Blocks          []*Block        `json:"blocks"`
		TransactionPool []*Transaction  `json:"transaction_pool"`
		ChainAddress    string          `json:"chain_address"`
		Reflections     map[string]bool `json:"reflections"`
		ChainID         string          `json:"chain_id"`
	}

	serializableBC := SerializableBlockchain{
		Blocks:          bcs.BlockchainPtr.Blocks,
		TransactionPool: bcs.BlockchainPtr.TransactionPool,
		ChainAddress:    bcs.BlockchainPtr.ChainAddress,
		Reflections:     bcs.BlockchainPtr.Reflections,
		ChainID:         bcs.BlockchainPtr.ChainID,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(serializableBC); err != nil {
		log.Printf("failed to encode blockchain response: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func (bcs *BlockchainServer) handleGetReflections(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	reflections := bcs.BlockchainPtr.Reflections
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(reflections); err != nil {
		log.Printf("failed to encode reflections response: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func (bcs *BlockchainServer) handleAddReflection(w http.ResponseWriter, r *http.Request) {
	// Check if network is paused and reject transaction if so
	if bcs.checkNetworkPauseAndRejectIfPaused(w, "add_reflection") {
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var newReflection string
	if err := json.NewDecoder(r.Body).Decode(&newReflection); err != nil {
		log.Printf("failed to decode reflection request: %v", err)
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	bcs.BlockchainPtr.Reflections[newReflection] = true
	fmt.Fprintf(w, "Reflection %s added successfully\n", newReflection)
}

func (bcs *BlockchainServer) handleReceiveBlock(w http.ResponseWriter, r *http.Request) {
	// Check if network is paused and reject transaction if so
	if bcs.checkNetworkPauseAndRejectIfPaused(w, "block") {
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var data map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		err = fmt.Errorf("failed to decode block request: %w", err)
		log.Println(err)
		http.Error(w, "Bad request: invalid JSON", http.StatusBadRequest)
		return
	}

	// Extract the block from the request
	blockData, ok := data["block"]
	if !ok {
		err := fmt.Errorf("missing 'block' field in request")
		log.Println(err)
		http.Error(w, "Bad request: missing 'block' field", http.StatusBadRequest)
		return
	}

	// Convert the block data to JSON
	blockJSON, err := json.Marshal(blockData)
	if err != nil {
		err = fmt.Errorf("failed to marshal block data: %w", err)
		log.Println(err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Unmarshal the JSON into a Block struct
	var block Block
	if err := json.Unmarshal(blockJSON, &block); err != nil {
		http.Error(w, "Bad request: invalid block format", http.StatusBadRequest)
		log.Println("Error unmarshaling block:", err)
		return
	}

	// Verify the block
	if !block.VerifyBlock() {
		err := fmt.Errorf("%w: %v", errors.ErrInvalidBlock, block)
		log.Println(err)
		http.Error(w, "Bad request: invalid block", http.StatusBadRequest)
		return
	}

	// Add the block to the blockchain using the stored db connection
	bcs.BlockchainPtr.AddBlock(&block)

	// Respond with success
	w.WriteHeader(http.StatusCreated)
	fmt.Fprintf(w, "Block added successfully")
}

func (bcs *BlockchainServer) HandleReceiveTransaction(w http.ResponseWriter, r *http.Request) {
	// Check if network is paused and reject transaction if so
	if bcs.checkNetworkPauseAndRejectIfPaused(w, "transaction") {
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Validate content type
	if r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "Content-Type must be application/json", http.StatusUnsupportedMediaType)
		return
	}

	var tx Transaction
	if err := json.NewDecoder(r.Body).Decode(&tx); err != nil {
		http.Error(w, fmt.Sprintf("Invalid transaction payload: %v", err), http.StatusBadRequest)
		return
	}

	// KNIRVCHAIN is sovereign: it burns NRN to mint NFT bundles (see
	// handleEventBundleMint) but is no longer the general-purpose NRN wallet
	// for plain transfers — that temporary pool now lives on KNIRVSERVER's
	// embedded Transaction Chain (pkg/embedded/transactionchain, POST
	// /transfer). Plain transfers use the empty Type (see Transaction.VerifyTxn);
	// every other type (event-bundle mint, MCP capability, LLM rooting, etc.)
	// is unaffected.
	if tx.Type == "" {
		http.Error(w, "plain NRN transfers are no longer accepted on KNIRVCHAIN; submit them to the Transaction Chain's /transfer endpoint instead", http.StatusGone)
		return
	}

	// Verify signature
	if tx.Signature == nil || tx.PublicKey == "" {
		http.Error(w, "Transaction is missing signature or public key", http.StatusBadRequest)
		return
	}

	isVerified, err := tx.VerifySignature()
	if err != nil {
		http.Error(w, fmt.Sprintf("Error verifying signature: %v", err), http.StatusInternalServerError)
		return
	}
	if !isVerified { // Use the captured boolean
		http.Error(w, "Invalid transaction signature", http.StatusUnauthorized)
		return
	}

	// Dispatch based on transaction type
	switch tx.Type {
	case TransactionTypeMCPRegisterCapability:
		var dataObj map[string]interface{}
		if err := json.Unmarshal(tx.Data, &dataObj); err == nil {
			// Use the enhanced formatted description for better logging
			desc := createDataDescriptionFormatted(dataObj, "MCP_REGISTER_CAPABILITY")
			log.Printf("Processing MCP_REGISTER_CAPABILITY transaction: %s - %s", tx.TransactionHash, desc)
		} else {
			log.Printf("Processing MCP_REGISTER_CAPABILITY transaction: %s", tx.TransactionHash)
		}
		// ProcessMCPRegisterCapability now takes the full transaction
		// and will handle unmarshalling tx.Data (JSON) internally.
		if err := bcs.BlockchainPtr.mcpProcessor.ProcessMCPRegisterCapability(&tx); err != nil {
			http.Error(w, fmt.Sprintf("Failed to process capability registration: %v", err), http.StatusInternalServerError)
			return
		}
		log.Printf("Successfully validated MCP_REGISTER_CAPABILITY transaction: %s", tx.TransactionHash)

	case TransactionTypeMCPInvokeCapability:
		var dataObj map[string]interface{}
		if err := json.Unmarshal(tx.Data, &dataObj); err == nil {
			// Use the enhanced formatted description for better logging
			desc := createDataDescriptionFormatted(dataObj, "MCP_INVOKE_CAPABILITY")
			log.Printf("Processing MCP_INVOKE_CAPABILITY transaction: %s - %s", tx.TransactionHash, desc)
		} else {
			log.Printf("Processing MCP_INVOKE_CAPABILITY transaction: %s", tx.TransactionHash)
		}

		// Create ContextRecord from transaction data
		hash := sha256.Sum256(tx.Data)
		contextRecord := &types.ContextRecord{
			ID:              tx.TransactionHash,
			CapabilityID:    tx.To,
			InteractionType: "Invoke",
			Initiator:       tx.From,
			Timestamp:       time.Now().Unix(),
			InputHash:       hex.EncodeToString(hash[:]),
			Details: map[string]interface{}{
				"fee": 0,
			},
		}

		if err := bcs.BlockchainPtr.mcpProcessor.ProcessMCPInvokeCapability(&tx, *contextRecord); err != nil {
			http.Error(w, fmt.Sprintf("Failed to process capability invocation: %v", err), http.StatusInternalServerError)
			return
		}
		log.Printf("Successfully processed MCP_INVOKE_CAPABILITY for Tx: %s", tx.TransactionHash)

	case TransactionTypeRISKSNAPSHOTCommit, TransactionTypeMODELActivate, TransactionTypePOOLCREATE,
		TransactionTypeSTAKEDEPOSIT, TransactionTypeSTAKEEXITREQUEST, TransactionTypeSUBMISSIONCOMMIT,
		TransactionTypePRICINGDECISIONCOMMIT, TransactionTypeRESERVECOMMIT, TransactionTypeSETTLEMENTCOMMIT,
		TransactionTypeLOSSALLOCATIONCOMMIT:
		commitment, err := ValidateSyndicateCommitment(tx.Type, tx.Data)
		if err != nil {
			http.Error(w, fmt.Sprintf("Invalid syndicate commitment: %v", err), http.StatusBadRequest)
			return
		}
		// The validated, signed transaction is then admitted to the normal pool
		// below, where block consensus and checkpoint finality remain the source
		// of authority. Dispatch must never imply financial finality.
		log.Printf("Validated %s commitment entity=%s hash=%s", tx.Type, commitment.EntityID, commitment.CommitmentHash)

	//case TransactionTypeStandard:
	//log.Printf("Processing STANDARD_TRANSFER transaction: %s", tx.TransactionHash)
	//if err := bcs.BlockchainPtr.ProcessStandardTransfer(&tx); err != nil {
	//	http.Error(w, fmt.Sprintf("Failed to process standard transfer: %v", err), http.StatusInternalServerError)
	//	return
	//	}
	//log.Printf("Successfully processed STANDARD_TRANSFER for Tx: %s", tx.TransactionHash)

	default:
		http.Error(w, fmt.Sprintf("Unsupported transaction type: %s", tx.Type), http.StatusBadRequest)
		return
	}

	// Add to transaction pool
	if err := bcs.BlockchainPtr.AddTransactionToTransactionPool(&tx); err != nil {
		http.Error(w, fmt.Sprintf("Failed to add transaction to pool: %v", err), http.StatusInternalServerError)
		return
	}

	// Respond with success
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated) // Use 201 for successful creation/processing of a transaction
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"tx_hash": tx.TransactionHash,
	})
}

func (bcs *BlockchainServer) handleGetTransactionPool(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	bcs.BlockchainPtr.Lock()
	defer bcs.BlockchainPtr.Unlock()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(bcs.BlockchainPtr.TransactionPool); err != nil {
		log.Printf("failed to encode transaction pool response: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func (bcs *BlockchainServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	log.Printf("Health check request from %s", r.RemoteAddr)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// Include DHT status if available
	status := "ok"
	if bcs.discoveryManager == nil {
		status = "ok-no-dht"
	}

	json.NewEncoder(w).Encode(map[string]string{
		"status": status,
		"port":   strconv.Itoa(int(bcs.port)),
	})
}

func (bcs *BlockchainServer) handleURIGenerator(w http.ResponseWriter, r *http.Request) {
	// uriCache tracks recently minted URIs during test runs
	var uriCache *sync.Map
	if bcs.testMode {
		uriCache = new(sync.Map)
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed use POST", http.StatusMethodNotAllowed)
		return
	}

	var reqBody URIRequest

	// First check if body is nil
	if r.Body == nil {
		log.Println("URI Generator: Request body is nil, generating UUID.")
		reqBody.DesiredID = "" // Treat as empty request
	} else {
		// Then try to decode the body
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			// Handle cases where body is empty or malformed
			if err == io.EOF {
				// Body is empty, proceed with UUID generation
				log.Println("URI Generator: Request body is empty, generating UUID.")
				reqBody.DesiredID = "" // Ensure it's empty
			} else {
				http.Error(w, "Invalid request body", http.StatusBadRequest)
				return
			}
		}
		defer r.Body.Close()
	}

	chosenID := ""
	isAvailable := true // Assume available initially

	// 2. Check availability if DesiredID is provided
	if reqBody.DesiredID != "" {
		// Basic validation for DesiredID (add more rules as needed)
		if len(reqBody.DesiredID) < 3 || len(reqBody.DesiredID) > 64 || strings.ContainsAny(reqBody.DesiredID, "/.?& ") {
			http.Error(w, "Invalid DesiredID format (length 3-64, no special chars)", http.StatusBadRequest)
			return
		}

		// --- Check 1: Blockchain History ---
		bcs.BlockchainPtr.Lock() // Lock for reading blocks
		existsInBlocks := bcs.BlockchainPtr.CheckIfIDExistsInBlocks(reqBody.DesiredID)
		existsInPool := bcs.BlockchainPtr.CheckIfIDExistsInTransactionPool(reqBody.DesiredID)
		bcs.BlockchainPtr.Unlock() // Unlock after reading

		if existsInBlocks {
			isAvailable = false
			log.Printf("URI Generator: Desired ID '%s' found in existing blocks.", reqBody.DesiredID)
		} else if existsInPool {
			isAvailable = false
			log.Printf("URI Generator: Desired ID '%s' found in pending transactions.", reqBody.DesiredID)
		} else {
			// Check in-memory test cache if enabled
			if uriCache != nil {
				if _, exists := uriCache.Load(reqBody.DesiredID); exists {
					isAvailable = false
					log.Printf("URI Generator: Desired ID '%s' found in test cache.", reqBody.DesiredID)
				}
			}

			// Only proceed with further checks if still available
			if isAvailable {
				// --- Check 2: Local Cache of Recently Registered IDs ---
				if bcs.discoveryManager != nil {
					// Check if ID was recently registered by this node
					if bcs.discoveryManager.IsRecentlyRegistered(reqBody.DesiredID) {
						isAvailable = false
						log.Printf("URI Generator: Desired ID '%s' was recently registered by this node.", reqBody.DesiredID)
					}
				}

				// Only proceed with DHT check if still available
				if isAvailable && bcs.discoveryManager != nil {
					log.Printf("URI Generator: Checking DHT availability for desired ID: %s", reqBody.DesiredID)
					providers, err := bcs.discoveryManager.FindResource(r.Context(), reqBody.DesiredID, DiscoveryResourceTypeChain)

					// --- Refined Availability Logic ---
					if err == nil {
						// No error during lookup
						if len(providers) > 0 {
							// Found providers, ID is taken
							isAvailable = false
							log.Printf("URI Generator: Desired ID '%s' is already taken by %d provider(s).", reqBody.DesiredID, len(providers))
						} else {
							// No error and no providers found -> ID is available
							isAvailable = true
							log.Printf("URI Generator: No providers found for '%s'. ID is available.", reqBody.DesiredID)
						}
					} else {
						// An error occurred during lookup
						if strings.Contains(err.Error(), "no providers found") || strings.Contains(err.Error(), "failed to find any dev in table") {
							// Specific error indicating availability
							isAvailable = true
							log.Printf("URI Generator: DHT lookup confirmed no providers for '%s'. ID is available. (Error: %v)", reqBody.DesiredID, err)
						} else {
							// Any other error (timeout, network issue, etc.)
							log.Printf("URI Generator: Error checking resource availability for '%s': %v", reqBody.DesiredID, err)
							// Treat unexpected errors as internal server errors, don't proceed with minting
							http.Error(w, "Internal server error during availability check", http.StatusInternalServerError)
							return // Stop processing
						}
					}
					// --- End Refined Availability Logic ---
				} else if bcs.discoveryManager == nil {
					log.Printf("URI Generator: Discovery manager not available, skipping DHT check for '%s'. Assuming available based on local chain.", reqBody.DesiredID)
					// isAvailable remains true if not found in blocks
				} // <<< Closes the 'else' for discoveryManager == nil
			} // <<< Closes the 'if isAvailable' check
		} // <<< Closes the 'else' for existsInBlocks
	} // <<< Closes the 'if' for reqBody.DesiredID != ""
	// --- End of Availability Checks ---

	// 3. Decide on the ID and handle unavailability
	if reqBody.DesiredID != "" {
		if isAvailable {
			chosenID = reqBody.DesiredID
			log.Printf("URI Generator: Desired ID '%s' is available.", chosenID)

			// Add to test cache if enabled
			if uriCache != nil {
				uriCache.Store(chosenID, true)
			}
		} else {
			// ID is taken, return conflict error
			http.Error(w, fmt.Sprintf("Desired ID '%s' is not available", reqBody.DesiredID), http.StatusConflict)
			return
		}
	} else {
		// No desired ID provided, generate a UUID
		chosenID = uri.CalculateChainID() // CalculateChainID uses uuid.New()
		log.Printf("URI Generator: No desired ID, generated UUID: %s", chosenID)
	}

	// Get or generate the chain ID
	chainID := bcs.BlockchainPtr.ChainID
	if chainID == "" {
		// If chain ID is not set, generate one
		chainID = uri.CalculateChainID()
		bcs.BlockchainPtr.ChainID = chainID
	}

	// Create chain metadata
	metadata := uri.ChainMetadata{
		ChainID: chosenID,
	}

	// Generate the URI using the new format
	uri := uri.GenerateChainURI(metadata)

	// If we have a discovery manager, announce this resource
	if bcs.discoveryManager != nil {
		// Announce the newly minted ID on the DHT
		go func(idToAnnounce string) { // Announce in background
			// Create a context with timeout for the announcement
			announceCtx, announceCancel := context.WithTimeout(context.Background(), 30*time.Second) // 30-second timeout
			defer announceCancel()

			// Announce the resource using the chosen ID
			err := bcs.discoveryManager.AnnounceMintedResource(announceCtx, idToAnnounce, DiscoveryResourceTypeChain)
			if err != nil {
				// Log the error, but the URI generation itself succeeded
				log.Printf("[%s] WARNING: Failed to announce newly minted resource %s on DHT: %v", bcs.BlockchainPtr.ChainID, idToAnnounce, err)
			}
		}(chosenID)
	}

	// Create a transaction to mint the URI onto the blockchain
	uriTxn := NewTransaction(
		utils.BLOCKCHAIN_ADDRESS, // From
		"",                       // To (empty for URI mint)
		0,                        // Value
		[]byte(uri),              // Data contains the URI
	)
	uriTxn.Type = "protocol_uri_mint"
	uriTxn.Status = TXN_VERIFICATION_SUCCESS // Bypass verification

	// Add to transaction pool
	bcs.BlockchainPtr.addVerifiedTxnToPoolAndSignal(uriTxn)

	// Log the generated URI and chosen ID
	log.Printf("Generated URI: %s (Using ID: %s)", uri, chosenID)
	log.Printf("URI Mint Transaction Hash: %s", uriTxn.TransactionHash)

	// Return the URI and hash in JSON format
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated) // Use 201 Created as we are creating a resource (the URI/Txn)
	json.NewEncoder(w).Encode(map[string]string{
		"uri":      uri,
		"txn_hash": uriTxn.TransactionHash,
	})
}

// Handle server info request
// This endpoint provides information about the server, including DHT status if available
// It returns a JSON response with the server's HTTP port, P2P port, chain ID, and DHT information

func (bcs *BlockchainServer) handleServerInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	info := ServerInfo{
		HTTPPort: bcs.port,
		P2PPort:  bcs.p2pPort,
		ChainID:  bcs.BlockchainPtr.ChainID,
		Version:  "1.0.0", // Replace with actual version
	}

	// Add DHT-specific information if available
	if bcs.discoveryManager != nil {
		info.PeerID = bcs.discoveryManager.GetPeerID()
		info.Multiaddrs = bcs.discoveryManager.GetSelfMultiaddrs()
		// Count connections (this is a placeholder, implement actual connection counting)
		info.Connections = len(bcs.BlockchainPtr.Reflections)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(info); err != nil {
		log.Printf("Failed to encode server info: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (bcs *BlockchainServer) handleGetPeers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// If we don't have a discovery manager, return an empty list
	if bcs.discoveryManager == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]string{})
		return
	}

	// Get chainID from query parameter or use our own
	chainID := r.URL.Query().Get("chainID")
	if chainID == "" {
		chainID = bcs.BlockchainPtr.ChainID
	}

	devs, err := bcs.discoveryManager.FindResource(context.Background(), chainID, DiscoveryResourceTypeChain)
	if err != nil {
		log.Printf("Failed to find devs: %v", err)
		// Return an empty list if we can't find devs
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]string{})
		return
	}

	// Convert dev info to multiaddrs
	var devAddrs []string
	for _, dev := range devs {
		for _, addr := range dev.Addrs {
			devAddrs = append(devAddrs, fmt.Sprintf("%s/p2p/%s", addr.String(), dev.ID.String()))
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(devAddrs); err != nil {
		log.Printf("Failed to encode devs: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// handleP2PConnect handles requests to connect to a dev.
// Expects a 'dev' query parameter with the full multiaddress of the target dev.
func (bcs *BlockchainServer) handleP2PConnect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		// Should be POST as it changes state (initiates connection)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if bcs.discoveryManager == nil {
		log.Println("Error in handleP2PConnect: discoveryManager is nil")
		http.Error(w, "Discovery service not available", http.StatusInternalServerError)
		return
	}

	devAddrStr := r.URL.Query().Get("dev")
	if devAddrStr == "" {
		http.Error(w, "Missing 'dev' query parameter (full multiaddress expected)", http.StatusBadRequest)
		return
	}

	log.Printf("[%s] Received /p2p/connect request for dev: %s", bcs.BlockchainPtr.ChainID, devAddrStr)

	devMA, err := multiaddr.NewMultiaddr(devAddrStr)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid dev multiaddress '%s': %v", devAddrStr, err), http.StatusBadRequest)
		return
	}

	devInfo, err := peer.AddrInfoFromP2pAddr(devMA)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create AddrInfo from multiaddress '%s': %v", devAddrStr, err), http.StatusBadRequest)
		return
	}

	// GetPeerID returns a string, so we need to convert devInfo.ID to string for comparison
	if devInfo.ID.String() == bcs.discoveryManager.GetPeerID() {
		http.Error(w, "Cannot connect to self", http.StatusBadRequest)
		return
	}

	connectCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second) // 15-second timeout
	defer cancel()

	if err := bcs.discoveryManager.ConnectToPeer(*devInfo, connectCtx); err != nil {
		http.Error(w, fmt.Sprintf("Failed to connect to dev %s: %v", devInfo.ID.String(), err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Successfully initiated connection to dev %s", devInfo.ID.String())
}

// handleWalletInfo provides information about the node's primary wallet.
func (bcs *BlockchainServer) handleWalletInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if bcs.BlockchainPtr == nil {
		http.Error(w, "Blockchain not initialized", http.StatusInternalServerError)
		return
	}

	// Use WalletAddress from BlockchainStruct, which is typically set from MinersAddress
	nodeWalletAddress := bcs.BlockchainPtr.WalletAddress
	if nodeWalletAddress == "" {
		// Fallback or error if no address is configured for the node
		log.Println("Node's wallet address (MinersAddress) is not configured.")
		http.Error(w, "Node wallet address not configured", http.StatusInternalServerError)
		return
	}

	balance, err := bcs.db.GetAccountBalance(nodeWalletAddress)
	if err != nil {
		log.Printf("Failed to get balance for node wallet %s: %v", nodeWalletAddress, err)
		http.Error(w, "Failed to retrieve wallet balance", http.StatusInternalServerError)
		return
	}

	walletInfo := map[string]interface{}{
		"address": nodeWalletAddress,
		"balance": balance,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(walletInfo)
}

func (bcs *BlockchainServer) handleTestFaucet(w http.ResponseWriter, r *http.Request) {
	if !strings.EqualFold(strings.TrimSpace(os.Getenv("KNIRV_ENABLE_DEMO")), "true") {
		http.NotFound(w, r)
		return
	}
	// IMPORTANT: Add checks here to ensure this endpoint is ONLY active during testing
	// e.g., check an environment variable, build tag, or a specific config flag.
	// if !IsTestEnvironment() {
	//     http.Error(w, "Endpoint not available", http.StatusForbidden)
	// 	   return
	// }

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req FaucetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Address == "" || req.Amount == 0 {
		http.Error(w, "Address and amount are required", http.StatusBadRequest)
		return
	}

	log.Printf("[TEST FAUCET] Request to fund %s with %d", req.Address, req.Amount)

	// Create the funding transaction from the blockchain faucet address
	faucetTxn := NewTransaction(utils.BLOCKCHAIN_ADDRESS, req.Address, req.Amount, []byte("test faucet funding"))
	faucetTxn.Type = "demo_faucet"
	// Mark as successful immediately - bypasses normal signing/verification for faucet
	faucetTxn.Status = TXN_VERIFICATION_SUCCESS // Mark as verified to enter pool

	// Add directly to the transaction pool (bypassing normal AddTransactionToTransactionPool checks)
	// Use the internal append method with locking
	bcs.BlockchainPtr.addVerifiedTxnToPoolAndSignal(faucetTxn)

	// Optionally broadcast this faucet transaction if needed for multi-node tests
	// bcs.BlockchainPtr.BroadcastTransaction(faucetTxn)

	log.Printf("[TEST FAUCET] Added funding transaction %s to pool for %s", faucetTxn.TransactionHash, req.Address)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(faucetTxn) // Respond with the created transaction
}

// --- MCP API Handlers ---

// handleMCPRegisterCapabilityInitiate - Step 1 of 2-step registration
func (bcs *BlockchainServer) handleMCPRegisterCapabilityInitiate(w http.ResponseWriter, r *http.Request) {
	// Check if network is paused and reject transaction if so
	if bcs.checkNetworkPauseAndRejectIfPaused(w, "mcp_capability_initiate") {
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check content type to determine if we're receiving protobuf or JSON
	contentType := r.Header.Get("Content-Type")
	isProtobuf := strings.Contains(contentType, "application/protobuf") || strings.Contains(contentType, "application/x-protobuf")

	var capabilityDescriptor interface{}
	var capabilityName string
	var capabilityTimestamp int64
	var requestFrom string
	var requestFee uint64
	var requestCapabilityType string
	var requestDesiredName string
	var err error

	if isProtobuf {
		// Handle protobuf request
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to read request body: %v", err), http.StatusBadRequest)
			return
		}

		// Unmarshal into a protobuf message
		var requestProto pb.MCPRegisterCapabilityDataProto
		if err := proto.Unmarshal(bodyBytes, &requestProto); err != nil {
			http.Error(w, fmt.Sprintf("Failed to unmarshal protobuf request: %v", err), http.StatusBadRequest)
			return
		}

		// Extract data from the protobuf message
		// This would need additional fields in the proto definition for from, fee, etc.
		// For now, we'll assume these are passed as HTTP headers or query parameters
		requestFrom = r.URL.Query().Get("from")
		// Public key is not used in this function
		feeStr := r.URL.Query().Get("fee")
		requestFee, _ = strconv.ParseUint(feeStr, 3, 64)
		requestDesiredName = r.URL.Query().Get("desiredName")

		// Extract capability descriptor from protobuf
		if requestProto.CapabilityDescriptor == nil {
			http.Error(w, "Missing capability descriptor in protobuf request", http.StatusBadRequest)
			return
		}

		// Convert protobuf to Go struct
		capabilityDescriptor, err = ConvertProtoToCapability(requestProto.CapabilityDescriptor)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to convert protobuf capability: %v", err), http.StatusBadRequest)
			return
		}

		// Extract capability type and name based on the descriptor type
		switch desc := capabilityDescriptor.(type) {
		case types.ResourceDescriptor:
			capabilityName = desc.Name
			capabilityTimestamp = desc.Timestamp
			requestCapabilityType = string(desc.CapabilityType)
		case types.ToolDescriptor:
			capabilityName = desc.Name
			capabilityTimestamp = desc.Timestamp
			requestCapabilityType = string(desc.CapabilityType)
		case types.PromptDescriptor:
			capabilityName = desc.Name
			capabilityTimestamp = desc.Timestamp
			requestCapabilityType = string(desc.CapabilityType)
		case types.MemoryServiceDescriptor:
			capabilityName = desc.Name
			capabilityTimestamp = desc.Timestamp
			requestCapabilityType = string(desc.CapabilityType)
		default:
			http.Error(w, "Unsupported capability type in protobuf request", http.StatusBadRequest)
			return
		}
	} else {
		// Handle JSON request (legacy format)
		var requestData struct {
			From           string          `json:"from"`
			PublicKey      string          `json:"publicKey"`
			Fee            uint64          `json:"fee"`
			CapabilityType string          `json:"capabilityType"`
			Descriptor     json.RawMessage `json:"descriptor"`
			DesiredName    string          `json:"desiredName,omitempty"`
		}

		if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
			http.Error(w, fmt.Sprintf("Failed to parse request body: %v", err), http.StatusBadRequest)
			return
		}

		requestFrom = requestData.From
		// Public key is not used in this function
		requestFee = requestData.Fee
		requestCapabilityType = requestData.CapabilityType
		requestDesiredName = requestData.DesiredName

		// Validate the capability type
		switch requestData.CapabilityType {
		case string(types.CapabilityTypeResource):
			var descriptor types.ResourceDescriptor
			if err := json.Unmarshal(requestData.Descriptor, &descriptor); err != nil {
				http.Error(w, fmt.Sprintf("Failed to parse ResourceDescriptor: %v", err), http.StatusBadRequest)
				return
			}
			capabilityName = descriptor.Name
			capabilityTimestamp = descriptor.Timestamp
			capabilityDescriptor = descriptor
		case string(types.CapabilityTypeTool):
			var descriptor types.ToolDescriptor
			if err := json.Unmarshal(requestData.Descriptor, &descriptor); err != nil {
				http.Error(w, fmt.Sprintf("Failed to parse ToolDescriptor: %v", err), http.StatusBadRequest)
				return
			}
			capabilityName = descriptor.Name
			capabilityTimestamp = descriptor.Timestamp
			capabilityDescriptor = descriptor
		case string(types.CapabilityTypePrompt):
			var descriptor types.PromptDescriptor
			if err := json.Unmarshal(requestData.Descriptor, &descriptor); err != nil {
				http.Error(w, fmt.Sprintf("Failed to parse PromptDescriptor: %v", err), http.StatusBadRequest)
				return
			}
			capabilityName = descriptor.Name
			capabilityTimestamp = descriptor.Timestamp
			capabilityDescriptor = descriptor
		case string(types.CapabilityTypeMemoryService):
			var descriptor types.MemoryServiceDescriptor
			if err := json.Unmarshal(requestData.Descriptor, &descriptor); err != nil {
				http.Error(w, fmt.Sprintf("Failed to parse MemoryServiceDescriptor: %v", err), http.StatusBadRequest)
				return
			}
			capabilityName = descriptor.Name
			capabilityTimestamp = descriptor.Timestamp
			capabilityDescriptor = descriptor
		default:
			http.Error(w, fmt.Sprintf("Invalid capability type: %s", requestData.CapabilityType), http.StatusBadRequest)
			return
		}
	}

	// Ensure timestamp is set if not provided
	if capabilityTimestamp == 0 {
		capabilityTimestamp = time.Now().Unix()
	}

	nameForIDGeneration := requestDesiredName
	if nameForIDGeneration == "" {
		if capabilityName == "" {
			http.Error(w, "Either 'desiredName' in request or 'name' in descriptor must be provided for ID generation.", http.StatusBadRequest)
			return
		}
		nameForIDGeneration = capabilityName
	}

	// Validate capability type
	capType := types.CapabilityType(requestCapabilityType)
	if !types.IsValidCapabilityType(types.CapabilityType(capType)) {
		http.Error(w, fmt.Sprintf("Invalid capability type: %s", requestCapabilityType), http.StatusBadRequest)
		return
	}

	generatedCapabilityID := uri.GenerateCapabilityID(nameForIDGeneration, capType)

	// Check if this server-generated ID already exists
	if _, errDb := bcs.db.GetCapabilityByID(generatedCapabilityID); errDb == nil {
		http.Error(w, fmt.Sprintf("Capability ID '%s' (generated from name '%s' and type '%s') already exists.", generatedCapabilityID, nameForIDGeneration, requestCapabilityType), http.StatusConflict)
		return
	}

	// Update the descriptor with the server-generated ID.
	// This requires modifying the specific descriptor type.
	var finalDescriptorForHashing interface{}
	switch desc := capabilityDescriptor.(type) {
	case types.ResourceDescriptor:
		desc.ID = generatedCapabilityID
		// Ensure timestamp is set
		if desc.Timestamp == 0 {
			desc.Timestamp = capabilityTimestamp
		}
		finalDescriptorForHashing = desc
	case types.ToolDescriptor:
		desc.ID = generatedCapabilityID
		// Ensure timestamp is set
		if desc.Timestamp == 0 {
			desc.Timestamp = capabilityTimestamp
		}
		finalDescriptorForHashing = desc
	case types.PromptDescriptor:
		desc.ID = generatedCapabilityID
		// Ensure timestamp is set
		if desc.Timestamp == 0 {
			desc.Timestamp = capabilityTimestamp
		}
		finalDescriptorForHashing = desc
	case types.MemoryServiceDescriptor:
		desc.ID = generatedCapabilityID
		// Ensure timestamp is set
		if desc.Timestamp == 0 {
			desc.Timestamp = capabilityTimestamp
		}
		finalDescriptorForHashing = desc
	default:
		http.Error(w, "Internal server error: could not set generated ID on descriptor.", http.StatusInternalServerError)
		return
	}

	// Convert to protobuf for transaction data
	capabilityProto, err := ConvertToCapabilityDescriptorContainerProto(finalDescriptorForHashing)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to convert capability to protobuf: %v", err), http.StatusInternalServerError)
		return
	}

	// Create register capability data protobuf
	capabilityProtoTyped, ok := capabilityProto.(*pb.CapabilityDescriptorContainerProto)
	if !ok {
		http.Error(w, "Failed to convert capability proto to expected type", http.StatusInternalServerError)
		return
	}
	registerData := &pb.MCPRegisterCapabilityDataProto{
		CapabilityDescriptor: capabilityProtoTyped,
	}

	// Marshal the protobuf message for transaction data
	txnData, err := proto.Marshal(registerData)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to marshal protobuf transaction data: %v", err), http.StatusInternalServerError)
		return
	}

	// Create a new MCP transaction (unsigned at this stage)
	pendingTxn, err := NewMCPTransaction(
		requestFrom,
		"", // No recipient for capability registration
		0,  // No value transfer
		txnData,
		TransactionTypeMCPRegisterCapability,
		requestFee,
		capabilityTimestamp, // Pass the timestamp from the descriptor
	)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create transaction: %v", err), http.StatusInternalServerError)
		return
	}

	// Store this pending transaction with creation timestamp
	pendingRegistrations.Lock()
	pendingRegistrations.m[pendingTxn.TransactionHash] = &pendingRegistration{
		Transaction: pendingTxn,
		CreatedAt:   time.Now(),
	}
	pendingRegistrations.Unlock()

	// Start cleanup goroutine if not already running
	go cleanupExpiredRegistrations()

	log.Printf("[INFO] Initiated capability registration. Pending hash: %s, Generated ID: %s", pendingTxn.TransactionHash, generatedCapabilityID)

	// Determine response format based on Accept header
	acceptHeader := r.Header.Get("Accept")
	if strings.Contains(acceptHeader, "application/protobuf") || strings.Contains(acceptHeader, "application/x-protobuf") {
		// Return protobuf response
		// This would need a response protobuf message definition
		// For now, we'll return JSON
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":                 "pending_signature",
			"pendingTransactionHash": pendingTxn.TransactionHash,
			"canonicalCapabilityID":  generatedCapabilityID,
			"fullDescriptor":         finalDescriptorForHashing,
		})
	} else {
		// Return JSON response
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":                    "pending_signature",
			"pendingTransactionHash":    pendingTxn.TransactionHash,
			"canonicalCapabilityID":     generatedCapabilityID,
			"fullDescriptor":            finalDescriptorForHashing,
			"transactionDataForSigning": base64.StdEncoding.EncodeToString(txnData), // Add this
		})
	}
}

// handleMCPRegisterCapabilityFinalize - Step 2 of 2-step registration
func (bcs *BlockchainServer) handleMCPRegisterCapabilityFinalize(w http.ResponseWriter, r *http.Request) {
	// Check if network is paused and reject transaction if so
	if bcs.checkNetworkPauseAndRejectIfPaused(w, "mcp_capability_finalize") {
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse the request body
	var requestData struct {
		PendingTransactionHash string `json:"pendingTransactionHash"`
		PublicKey              string `json:"publicKey"`
		Signature              string `json:"signature"` // Changed to string to handle base64 encoding
	}

	if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse request body: %v", err), http.StatusBadRequest)
		return
	}

	// Retrieve the pending transaction first
	pendingRegistrations.RLock()
	pendingReg, exists := pendingRegistrations.m[requestData.PendingTransactionHash]
	pendingRegistrations.RUnlock()

	if !exists {
		http.Error(w, fmt.Sprintf("Pending transaction with hash %s not found or expired", requestData.PendingTransactionHash), http.StatusNotFound)
		return
	}

	// Check if the pending registration has expired
	if time.Since(pendingReg.CreatedAt) > pendingRegistrationTTL {
		// Remove expired registration
		pendingRegistrations.Lock()
		delete(pendingRegistrations.m, requestData.PendingTransactionHash)
		pendingRegistrations.Unlock()

		http.Error(w, fmt.Sprintf("Pending transaction with hash %s has expired", requestData.PendingTransactionHash), http.StatusGone)
		return
	}

	// Decode the base64-encoded signature
	signature, err := base64.StdEncoding.DecodeString(requestData.Signature)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to decode signature: %v", err), http.StatusBadRequest)
		return
	}

	// Get the transaction from the pending registration
	pendingTxn := pendingReg.Transaction

	// Set the public key and signature
	pendingTxn.PublicKey = requestData.PublicKey
	pendingTxn.Signature = signature // Use the decoded signature

	// Log transaction details before verification
	log.Printf("[DEBUG] Verifying signature for transaction: %+v", pendingTxn)
	log.Printf("[DEBUG] Signature (hex): %x", pendingTxn.Signature)
	log.Printf("[DEBUG] Public Key: %s", pendingTxn.PublicKey)

	isVerified, err := pendingTxn.VerifySignature()
	if err != nil {
		log.Printf("[ERROR] Error during signature verification for transaction hash: %s: %v", pendingTxn.TransactionHash, err)
		http.Error(w, "Error during signature verification", http.StatusInternalServerError)
		return
	}
	if !isVerified {
		log.Printf("[ERROR] Signature verification failed for transaction hash: %s", pendingTxn.TransactionHash)
		http.Error(w, "Invalid signature", http.StatusBadRequest)
		return
	}

	log.Printf("[DEBUG] Signature verification successful for transaction hash: %s", pendingTxn.TransactionHash)

	// Remove from pending registrations
	pendingRegistrations.Lock()
	delete(pendingRegistrations.m, requestData.PendingTransactionHash)
	pendingRegistrations.Unlock()

	log.Printf("[INFO] Finalized capability registration for transaction hash: %s", pendingTxn.TransactionHash)

	// Add the transaction to the pool with error checking
	err = bcs.BlockchainPtr.AddTransactionToTransactionPool(pendingTxn)
	if err != nil {
		log.Printf("[ERROR] Failed to add MCP Register transaction to pool: %v\nTransaction: %+v\nSignature: %x",
			err, pendingTxn, pendingTxn.Signature)
		http.Error(w, fmt.Sprintf(`{"error":"Failed to process transaction","details":"%v"}`, err),
			http.StatusUnprocessableEntity)
		return
	}

	// Log successful addition
	var dataObj map[string]interface{}
	if err := json.Unmarshal(pendingTxn.Data, &dataObj); err == nil {
		var txType string
		switch pendingTxn.Type {
		case TransactionTypeMCPRegisterCapability:
			txType = "MCP_REGISTER_CAPABILITY"
		case TransactionTypeMCPInvokeCapability:
			txType = "MCP_INVOKE_CAPABILITY"
		default:
			txType = pendingTxn.Type
		}
		// Use the enhanced formatted description for better logging
		desc := createDataDescriptionFormatted(dataObj, txType)
		log.Printf("[INFO] Successfully added transaction %s to pool (%s): %s", pendingTxn.TransactionHash, txType, desc)
	} else {
		log.Printf("[INFO] Successfully added transaction %s to pool", pendingTxn.TransactionHash)
	}

	// Extract capability ID and type from the transaction data for DHT announcement
	var capabilityID string
	var capabilityType string

	// First try to unmarshal as protobuf
	var registerProto pb.MCPRegisterCapabilityDataProto
	if err := proto.Unmarshal(pendingTxn.Data, &registerProto); err == nil {
		// Successfully parsed as protobuf
		if registerProto.CapabilityDescriptor != nil {
			// Convert protobuf to Go struct for extraction
			capabilityDescriptor, err := ConvertProtoToCapability(registerProto.CapabilityDescriptor)
			if err == nil {
				// Extract ID and type from the capability descriptor
				capabilityID, capabilityType, err = capabilityIdentity(capabilityDescriptor)
				if err != nil {
					log.Printf("[WARN] Failed to identify capability for DHT announcement: %v", err)
				}
			}
		}
	} else {
		// If not protobuf, try JSON format (legacy)
		var txnDataMap map[string]interface{}
		if err := json.Unmarshal(pendingTxn.Data, &txnDataMap); err != nil {
			log.Printf("[ERROR] Failed to unmarshal transaction data for DHT announcement: %v", err)
			// Continue anyway since the transaction is already in the pool
		} else {
			// Extract capability ID and type from the JSON data
			if descriptor, ok := txnDataMap["capabilityDescriptor"].(map[string]interface{}); ok {
				if id, ok := descriptor["id"].(string); ok {
					capabilityID = id
				}
				if capType, ok := descriptor["capabilityType"].(string); ok {
					capabilityType = capType
				}
			}
		}
	}

	if capabilityID == "" || capabilityType == "" {
		log.Printf("[ERROR] Failed to extract capability ID or type from transaction data for DHT announcement")
		// Continue anyway since the transaction is already in the pool
	}

	// Generate and Announce URI
	capabilityURI := uri.GenerateMCPCapabilityURI(capabilityID, types.CapabilityType(capabilityType), "", nil)

	if bcs.discoveryManager != nil {
		go func(idToAnnounce string, capTypeStr string) { // Announce in background
			announceCtx, announceCancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer announceCancel()
			// Announce using the lowercase string of the capability type for DHT key consistency
			err := bcs.discoveryManager.AnnounceMCPCapability(announceCtx, idToAnnounce, strings.ToLower(capTypeStr))
			if err != nil {
				log.Printf("[ERROR] Failed to announce registered capability %s (URI: %s) on DHT: %v", idToAnnounce, capabilityURI, err)
			} else {
				log.Printf("[INFO] Successfully announced MCP capability %s (type: %s, URI: %s) on DHT", idToAnnounce, strings.ToLower(capTypeStr), capabilityURI)
			}
		}(capabilityID, capabilityType)
	}

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":           "success",
		"transaction_hash": pendingTxn.TransactionHash,
		"capability_uri":   capabilityURI,
	})
}

// handleMCPPrepareCapabilityRegistration prepares the data needed for a capability registration transaction
// This is part of the new client-side signing flow
func (bcs *BlockchainServer) handleMCPPrepareCapabilityRegistration(w http.ResponseWriter, r *http.Request) {
	// Check if network is paused and reject transaction if so
	if bcs.checkNetworkPauseAndRejectIfPaused(w, "mcp_capability_prepare") {
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse the request body
	var request MCPPrepareCapabilityRegistrationRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse request body: %v", err), http.StatusBadRequest)
		return
	}

	// Validate the input
	if request.FromAddress == "" {
		http.Error(w, "Missing from_address", http.StatusBadRequest)
		return
	}
	if request.CapabilityType == "" {
		http.Error(w, "Missing capability_type", http.StatusBadRequest)
		return
	}
	if request.Descriptor == nil {
		http.Error(w, "Missing descriptor", http.StatusBadRequest)
		return
	}
	if request.Fee == 0 {
		http.Error(w, "Fee must be greater than 0", http.StatusBadRequest)
		return
	}

	// Validate capability type
	capType := types.CapabilityType(request.CapabilityType)
	if !types.IsValidCapabilityType(types.CapabilityType(capType)) {
		http.Error(w, fmt.Sprintf("Invalid capability type: %s", request.CapabilityType), http.StatusBadRequest)
		return
	}

	// Validate descriptor structure based on capability type
	if request.Descriptor == nil {
		http.Error(w, "Descriptor cannot be nil", http.StatusBadRequest)
		return
	}

	// Generate a unique capability ID
	// This could be based on a hash of the descriptor or a UUID
	var capabilityID string
	if request.DesiredName != "" {
		capabilityID = uri.GenerateCapabilityID(request.DesiredName, capType)
	} else {
		// Fallback to using capability type as the name if no desired name is provided
		capabilityID = uri.GenerateCapabilityID(string(capType), capType)
	}

	// Check if capability ID already exists
	if bcs.db != nil {
		exists, err := bcs.db.CapabilityExists(capabilityID)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to check if capability exists: %v", err), http.StatusInternalServerError)
			return
		}
		if exists {
			http.Error(w, fmt.Sprintf("Capability ID %s already exists", capabilityID), http.StatusConflict)
			return
		}
	}

	// Create the MCPRegisterCapabilityDataProto
	// Create appropriate descriptor based on capability type
	// Create base descriptor with default values
	baseDesc := types.BaseDescriptor{
		ID:             capabilityID,
		CapabilityType: types.CapabilityType(request.CapabilityType),
		Name:           request.DesiredName,
		Owner:          request.FromAddress,
		Description:    request.Description,
		Timestamp:      time.Now().Unix(),
	}

	// Extract Version from descriptor or default
	if verStr, ok := request.Descriptor["version"].(string); ok && verStr != "" {
		baseDesc.Version = verStr
	} else {
		baseDesc.Version = "1.0.0" // Default version
	}

	// Extract GasFeeNRN from the descriptor map if present
	if gasFee, ok := request.Descriptor["gas_fee_nrn"].(float64); ok {
		baseDesc.GasFeeNRN = uint64(gasFee)
	} else {
		baseDesc.GasFeeNRN = 0 // Default if not provided
	}

	// Extract CustomMetadata from the descriptor map if present
	if customMetaVal, ok := request.Descriptor["custom_metadata"].(map[string]interface{}); ok {
		baseDesc.CustomMetadata = customMetaVal
	}

	var descriptor interface{}
	switch request.CapabilityType {
	case string(types.CapabilityTypeResource):
		descriptor = &types.ResourceDescriptor{
			BaseDescriptor: baseDesc,
			ResourceType:   types.DiscoveryResourceType(request.Descriptor["resource_type"].(string)),
			ContentHash:    request.Descriptor["content_hash"].(string),
		}
	case string(types.CapabilityTypeTool):
		descriptor = &types.ToolDescriptor{
			BaseDescriptor:   baseDesc,
			InputSchemaJSON:  request.Descriptor["input_schema"].(string),
			OutputSchemaJSON: request.Descriptor["output_schema"].(string),
		}
	default:
		http.Error(w, fmt.Sprintf("unsupported capability type: %s", request.CapabilityType), http.StatusBadRequest)
		return
	}

	// Convert descriptor to map[string]interface{} for MCPRegisterCapabilityData
	descriptorBytes, err := json.Marshal(descriptor)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to marshal descriptor: %v", err), http.StatusInternalServerError)
		return
	}
	var descriptorMap map[string]interface{}
	if err := json.Unmarshal(descriptorBytes, &descriptorMap); err != nil {
		http.Error(w, fmt.Sprintf("Failed to unmarshal descriptor: %v", err), http.StatusInternalServerError)
		return
	}

	mcpData := &types.MCPRegisterCapabilityData{
		CapabilityID:         capabilityID,
		Name:                 request.DesiredName,
		Owner:                request.FromAddress,
		Version:              baseDesc.Version,
		Description:          request.Description,
		CapabilityType:       request.CapabilityType,
		GasFeeNRN:            baseDesc.GasFeeNRN,
		CapabilityDescriptor: descriptorMap,
		Descriptor:           descriptorMap,
	}

	// Log the descriptor before marshalling
	log.Printf("[DEBUG] Server: Descriptor being prepared for transaction: %+v", descriptor)
	if rd, ok := descriptor.(*types.ResourceDescriptor); ok {
		log.Printf("[DEBUG] Server: ResourceDescriptor's BaseDescriptor.Version before marshal: '%s'", rd.BaseDescriptor.Version)
		log.Printf("[DEBUG] Server: ResourceDescriptor's BaseDescriptor.GasFeeNRN before marshal: %d", rd.BaseDescriptor.GasFeeNRN)
	} else if td, ok := descriptor.(*types.ToolDescriptor); ok {
		log.Printf("[DEBUG] Server: ToolDescriptor's BaseDescriptor.Version before marshal: '%s'", td.BaseDescriptor.Version)
	} // Add other types if necessary

	// Marshal the MCPRegisterCapabilityData to JSON
	mcpDataBytes, err := json.Marshal(mcpData)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to marshal capability data: %v", err), http.StatusInternalServerError)
		return
	}
	log.Printf("[DEBUG] Server: mcpDataBytes (string) for client: %s", string(mcpDataBytes))

	// Create the unsigned transaction details
	timestamp := time.Now().Unix()
	unsignedTxDetails := UnsignedTransactionDetails{
		From:      request.FromAddress,
		To:        "", // No recipient for capability registration
		Value:     0,  // No value transfer for capability registration
		Data:      mcpDataBytes,
		Timestamp: timestamp,
		Fee:       request.Fee,
		Type:      TransactionTypeMCPRegisterCapability,
	}

	// Create a temporary Transaction object to generate the hash
	tempTx := &Transaction{
		From:      unsignedTxDetails.From,
		To:        unsignedTxDetails.To,
		Value:     unsignedTxDetails.Value,
		Data:      unsignedTxDetails.Data.([]byte),
		Timestamp: unsignedTxDetails.Timestamp,
		Fee:       unsignedTxDetails.Fee,
		Type:      unsignedTxDetails.Type,
	}

	// Generate the transaction hash
	txProto, err := tempTx.ToProtoForHashing()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to convert transaction to proto for hashing: %v", err), http.StatusInternalServerError)
		return
	}
	canonicalBytes, err := GetCanonicalBytesForHashingTransaction(txProto)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get canonical bytes for hashing: %v", err), http.StatusInternalServerError)
		return
	}
	hash := sha256.Sum256(canonicalBytes)
	unsignedTxPayloadHash := hex.EncodeToString(hash[:])

	// Set the transaction hash
	tempTx.TransactionHash = "0x" + unsignedTxPayloadHash

	// Store this pending transaction with creation timestamp
	pendingRegistrations.Lock()
	pendingRegistrations.m[tempTx.TransactionHash] = &pendingRegistration{
		Transaction: tempTx,
		CreatedAt:   time.Now(),
	}
	pendingRegistrations.Unlock()

	// Start cleanup goroutine if not already running
	go cleanupExpiredRegistrations()

	log.Printf("[INFO] Prepared capability registration. Pending hash: %s, Generated ID: %s", tempTx.TransactionHash, capabilityID)

	// Prepare the response
	response := MCPPrepareCapabilityRegistrationResponse{
		CapabilityID:                   capabilityID,
		UnsignedTransactionPayloadHash: unsignedTxPayloadHash,
		TransactionDetailsForSigning:   unsignedTxDetails,
		Message:                        "Transaction prepared successfully. Sign the transaction data and submit to /transaction endpoint.",
		EstimatedGasFee:                request.Fee,            // For now, just return the provided fee
		PendingTransactionHash:         tempTx.TransactionHash, // Add the pending transaction hash
		FullDescriptor:                 descriptor,             // Add the full descriptor
	}

	// Return the response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
		return
	}
}

// handleMCPRegisterCapability handles the registration of a new MCP capability (legacy endpoint)
// This endpoint is deprecated and will be removed in a future version.
// Use /mcp/capability/prepare_registration and /transaction instead.
func (bcs *BlockchainServer) handleMCPRegisterCapability(w http.ResponseWriter, r *http.Request) {
	// Check if network is paused and reject transaction if so
	if bcs.checkNetworkPauseAndRejectIfPaused(w, "mcp_capability_register") {
		return
	}

	// Log deprecation warning
	log.Printf("[WARNING] Deprecated endpoint /mcp/capability/register used. Please migrate to the two-step registration process.")
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse the request body
	var requestData struct {
		From           string          `json:"from"`
		PublicKey      string          `json:"publicKey"`
		Signature      []byte          `json:"signature"`
		Fee            uint64          `json:"fee"`
		CapabilityType string          `json:"capabilityType"`
		Descriptor     json.RawMessage `json:"descriptor"`
	}

	if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse request body: %v", err), http.StatusBadRequest)
		return
	}

	// Validate the capability type
	var capabilityDescriptor interface{}
	var capabilityID string // To store the ID for URI generation and DHT announcement
	var err error

	switch requestData.CapabilityType {
	case string(types.CapabilityTypeResource):
		var descriptor types.ResourceDescriptor
		if err := json.Unmarshal(requestData.Descriptor, &descriptor); err != nil {
			http.Error(w, fmt.Sprintf("Failed to parse ResourceDescriptor: %v", err), http.StatusBadRequest)
			return
		}
		capabilityID = descriptor.ID // Get ID from the descriptor
		//capabilityTimestamp = descriptor.Timestamp
		capabilityDescriptor = descriptor
	case string(types.CapabilityTypeTool):
		var descriptor types.ToolDescriptor
		if err := json.Unmarshal(requestData.Descriptor, &descriptor); err != nil {
			http.Error(w, fmt.Sprintf("Failed to parse ToolDescriptor: %v", err), http.StatusBadRequest)
			return
		}
		capabilityID = descriptor.ID
		//capabilityTimestamp = descriptor.Timestamp
		capabilityDescriptor = descriptor
	case string(types.CapabilityTypePrompt):
		var descriptor types.PromptDescriptor
		if err := json.Unmarshal(requestData.Descriptor, &descriptor); err != nil {
			http.Error(w, fmt.Sprintf("Failed to parse PromptDescriptor: %v", err), http.StatusBadRequest)
			return
		}
		capabilityID = descriptor.ID
		//capabilityTimestamp = descriptor.Timestamp
		capabilityDescriptor = descriptor
	case string(types.CapabilityTypeMemoryService):
		var descriptor types.MemoryServiceDescriptor
		if err := json.Unmarshal(requestData.Descriptor, &descriptor); err != nil {
			http.Error(w, fmt.Sprintf("Failed to parse MemoryServiceDescriptor: %v", err), http.StatusBadRequest)
			return
		}
		capabilityID = descriptor.ID
		//capabilityTimestamp = descriptor.Timestamp
		capabilityDescriptor = descriptor
	default:
		http.Error(w, fmt.Sprintf("Invalid capability type: %s", requestData.CapabilityType), http.StatusBadRequest)
		return
	}

	if capabilityID == "" {
		http.Error(w, "Capability descriptor is missing a valid 'id' field", http.StatusBadRequest)
		return
	}

	// Create transaction data that matches what the client signed
	txnData, err := json.Marshal(map[string]interface{}{
		"capabilityDescriptor": capabilityDescriptor,
	})
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to marshal transaction data: %v", err), http.StatusInternalServerError)
		return
	}

	// Create a new MCP transaction
	transaction, err := NewMCPTransaction(
		requestData.From,
		"", // No recipient for capability registration
		0,  // No value transfer
		txnData,
		TransactionTypeMCPRegisterCapability,
		requestData.Fee,
	)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create transaction: %v", err), http.StatusInternalServerError)
		return
	}

	// Set the public key and signature
	transaction.PublicKey = requestData.PublicKey
	transaction.Signature = requestData.Signature

	// Add the transaction to the pool with error checking
	err = bcs.BlockchainPtr.AddTransactionToTransactionPool(transaction)
	if err != nil {
		log.Printf("[ERROR] Failed to add MCP Register transaction to pool: %v\nTransaction: %+v\nSignature: %x",
			err, transaction, transaction.Signature)
		http.Error(w, fmt.Sprintf(`{"error":"Failed to process transaction","details":"%v"}`, err),
			http.StatusUnprocessableEntity)
		return
	}

	// Log successful addition
	log.Printf("[INFO] Successfully added transaction %s to pool", transaction.TransactionHash)

	// --- Generate and Announce URI ---
	// Use capabilityID and its specific MCP type (e.g., "RESOURCE", "TOOL")
	capabilityURI := uri.GenerateMCPCapabilityURI(capabilityID, types.CapabilityType(requestData.CapabilityType), "", nil)

	if bcs.discoveryManager != nil {
		go func(idToAnnounce string, capTypeStr string) { // Announce in background
			announceCtx, announceCancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer announceCancel()
			// Announce using the lowercase string of the capability type for DHT key consistency
			err := bcs.discoveryManager.AnnounceMCPCapability(announceCtx, idToAnnounce, strings.ToLower(capTypeStr))
			if err != nil {
				log.Printf("[ERROR] Failed to announce registered capability %s (URI: %s) on DHT: %v", idToAnnounce, capabilityURI, err)
			} else {
				log.Printf("[INFO] Successfully announced MCP capability %s (type: %s, URI: %s) on DHT", idToAnnounce, strings.ToLower(capTypeStr), capabilityURI)
			}
		}(capabilityID, requestData.CapabilityType)
	}

	// Return the transaction hash

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":           "success",
		"transaction_hash": transaction.TransactionHash,
		"capability_uri":   capabilityURI,
	})
}

// handleMCPInvokeCapability handles the invocation of an MCP capability
func (bcs *BlockchainServer) handleMCPInvokeCapability(w http.ResponseWriter, r *http.Request) {
	// Check if network is paused and reject transaction if so
	if bcs.checkNetworkPauseAndRejectIfPaused(w, "mcp_capability_invoke") {
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse the request body
	var requestData struct {
		ContextRecord  json.RawMessage `json:"contextRecord"`
		From           string          `json:"from"`
		PublicKey      string          `json:"publicKey"`
		Signature      []byte          `json:"signature"`
		Fee            uint64          `json:"fee"`
		CapabilityType string          `json:"capabilityType"`
	}

	if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse request body: %v", err), http.StatusBadRequest)
		return
	}

	// Parse the context record
	var contextRecord types.ContextRecord
	if err := json.Unmarshal(requestData.ContextRecord, &contextRecord); err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse ContextRecord: %v", err), http.StatusBadRequest)
		return
	}

	// Resolve CapabilityID if it's a URI
	resolvedCapabilityID, err := resolveIdentifier(contextRecord.CapabilityID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to resolve capability identifier '%s': %v", contextRecord.CapabilityID, err), http.StatusBadRequest)
		return
	}
	if resolvedCapabilityID != contextRecord.CapabilityID {
		log.Printf("Resolved capability URI '%s' to ID '%s' for invocation", contextRecord.CapabilityID, resolvedCapabilityID)
		contextRecord.CapabilityID = resolvedCapabilityID // Update contextRecord with the raw ID
	}

	// Create a transaction for the capability invocation using MCPInvokeCapabilityData
	invokeData := types.MCPInvokeCapabilityData{
		ContextRecord: contextRecord,
	}
	txnData, err := json.Marshal(invokeData)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to marshal transaction data: %v", err), http.StatusInternalServerError)
		return
	}

	// Create a new MCP transaction
	transaction, err := NewMCPTransaction(
		requestData.From,
		"", // No recipient for capability invocation
		0,  // No value transfer
		txnData,
		TransactionTypeMCPInvokeCapability,
		requestData.Fee,
	)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create transaction: %v", err), http.StatusInternalServerError)
		return
	}

	// Set the public key and signature
	transaction.PublicKey = requestData.PublicKey
	transaction.Signature = requestData.Signature

	// Add the transaction to the pool
	err = bcs.BlockchainPtr.AddTransactionToTransactionPool(transaction)
	if err != nil {
		log.Printf("[ERROR] Failed to add MCP Invoke transaction to pool: %v\nTransaction: %+v\nSignature: %x",
			err, transaction, transaction.Signature)
		http.Error(w, fmt.Sprintf(`{"error":"Failed to process transaction","details":"%v"}`, err), http.StatusUnprocessableEntity)
		return
	}

	// Return the transaction hash
	// Generate MCP capability URI
	capabilityURI := uri.GenerateMCPCapabilityURI(
		transaction.TransactionHash,  // Use transaction hash as capability ID
		types.CapabilityTypeResource, // Capability type
		"",                           // No path
		map[string]string{
			"type": requestData.CapabilityType,
		},
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated) // Set HTTP status to 201 Created on success
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":           "success",
		"transaction_hash": transaction.TransactionHash,
		"capability_uri":   capabilityURI,
	})
}

// handleMCPUpdateCapability handles the update of an MCP capability
func (bcs *BlockchainServer) handleMCPUpdateCapability(w http.ResponseWriter, r *http.Request) {
	// Check if network is paused and reject transaction if so
	if bcs.checkNetworkPauseAndRejectIfPaused(w, "mcp_capability_update") {
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse the request body
	var requestData struct {
		CapabilityID   string          `json:"capabilityID"`
		CapabilityType string          `json:"capabilityType"`
		Descriptor     json.RawMessage `json:"descriptor"`
		From           string          `json:"from"`
		PublicKey      string          `json:"publicKey"`
		Signature      []byte          `json:"signature"`
		Fee            uint64          `json:"fee"`
	}

	if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse request body: %v", err), http.StatusBadRequest)
		return
	}

	// Resolve CapabilityID if it's a URI
	resolvedCapabilityID, err := resolveIdentifier(requestData.CapabilityID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to resolve capability identifier '%s': %v", requestData.CapabilityID, err), http.StatusBadRequest)
		return
	}
	if resolvedCapabilityID != requestData.CapabilityID {
		log.Printf("Resolved capability URI '%s' to ID '%s' for update", requestData.CapabilityID, resolvedCapabilityID)
		requestData.CapabilityID = resolvedCapabilityID // Update requestData with the raw ID
	}

	// Validate the capability type
	var capabilityDescriptor interface{}

	switch requestData.CapabilityType {
	case string(types.CapabilityTypeResource):
		var descriptor types.ResourceDescriptor
		if err := json.Unmarshal(requestData.Descriptor, &descriptor); err != nil {
			http.Error(w, fmt.Sprintf("Failed to parse ResourceDescriptor: %v", err), http.StatusBadRequest)
			return
		}
		capabilityDescriptor = descriptor
	case string(types.CapabilityTypeTool):
		var descriptor types.ToolDescriptor
		if err := json.Unmarshal(requestData.Descriptor, &descriptor); err != nil {
			http.Error(w, fmt.Sprintf("Failed to parse ToolDescriptor: %v", err), http.StatusBadRequest)
			return
		}
		capabilityDescriptor = descriptor
	case string(types.CapabilityTypePrompt):
		var descriptor types.PromptDescriptor
		if err := json.Unmarshal(requestData.Descriptor, &descriptor); err != nil {
			http.Error(w, fmt.Sprintf("Failed to parse PromptDescriptor: %v", err), http.StatusBadRequest)
			return
		}
		capabilityDescriptor = descriptor
	case string(types.CapabilityTypeMemoryService):
		var descriptor types.MemoryServiceDescriptor
		if err := json.Unmarshal(requestData.Descriptor, &descriptor); err != nil {
			http.Error(w, fmt.Sprintf("Failed to parse MemoryServiceDescriptor: %v", err), http.StatusBadRequest)
			return
		}
		capabilityDescriptor = descriptor
	default:
		http.Error(w, fmt.Sprintf("Invalid capability type: %s", requestData.CapabilityType), http.StatusBadRequest)
		return
	}

	// Create transaction data that matches what the client signed
	txnData, err := json.Marshal(map[string]interface{}{
		"capabilityID":         requestData.CapabilityID,
		"capabilityType":       requestData.CapabilityType,
		"capabilityDescriptor": capabilityDescriptor,
	})
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to marshal transaction data: %v", err), http.StatusInternalServerError)
		return
	}

	// Create a new MCP transaction
	transaction, err := NewMCPTransaction(
		requestData.From,
		"", // No recipient for capability update
		0,  // No value transfer
		txnData,
		TransactionTypeMCPUpdateCapability,
		requestData.Fee,
	)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create transaction: %v", err), http.StatusInternalServerError)
		return
	}

	// Set the public key and signature
	transaction.PublicKey = requestData.PublicKey
	transaction.Signature = requestData.Signature

	// Add the transaction to the pool
	bcs.BlockchainPtr.AddTransactionToTransactionPool(transaction)

	// Return the transaction hash
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":           "success",
		"transaction_hash": transaction.TransactionHash,
	})
}

// handleMCPGetCapability handles the retrieval of an MCP capability
func (bcs *BlockchainServer) handleMCPGetCapability(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get the capability ID from the query parameters
	uriParam := r.URL.Query().Get("uri") // New: allow full URI
	idParam := r.URL.Query().Get("id")   // Existing: allow raw ID

	var capabilityIDToLookup string

	if uriParam != "" {
		// If 'uri' is provided, parse it to get the ID
		parsedID, resourceType, path, params, err := uri.ParseResourceURI(uriParam)
		if err != nil {
			http.Error(w, fmt.Sprintf("Invalid capability URI '%s': %v", uriParam, err), http.StatusBadRequest)
			return
		}
		// You could optionally validate the resourceType or use path/params here
		_ = resourceType
		_ = path
		_ = params
		capabilityIDToLookup = parsedID
		log.Printf("Parsed capability ID '%s' from URI '%s'", capabilityIDToLookup, uriParam)
	} else if idParam != "" {
		// If 'id' is provided, use it directly
		capabilityIDToLookup = idParam
	} else {
		http.Error(w, "Missing capability identifier (expected 'uri' or 'id' query parameter)", http.StatusBadRequest)
		return
	}

	// Get the capability from the database
	capability, err := bcs.db.GetCapabilityByID(capabilityIDToLookup)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get capability: %v", err), http.StatusNotFound)
		return
	}

	// Return the capability
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":     "success",
		"capability": capability,
	})
}

// handleMCPListCapabilities handles the listing of MCP capabilities
func (bcs *BlockchainServer) handleMCPListCapabilities(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get the capability type from the query parameters (optional)
	capabilityTypeFilter := r.URL.Query().Get("type")

	// Get the owner from the query parameters (optional)
	ownerFilter := r.URL.Query().Get("owner")

	var capabilities []interface{}

	if capabilityTypeFilter != "" && ownerFilter != "" {
		// If both filters are provided, we need to get by type, then filter by owner (or vice-versa)
		// This could be optimized with a more specific DB query if performance becomes an issue.
		capsByType, errType := bcs.db.GetCapabilitiesByType(capabilityTypeFilter)
		if errType != nil {
			http.Error(w, fmt.Sprintf("Failed to get capabilities by type: %v", errType), http.StatusInternalServerError)
			return
		}
		for _, capInterface := range capsByType {
			// Need to assert the type to access the Owner field from BaseDescriptor
			// This is a bit verbose; a helper function might be good if this pattern repeats.
			var owner string
			// Convert proto to capability interface
			capability, err := ConvertProtoToCapability(capInterface)
			if err != nil {
				continue // Skip this capability if conversion fails
			}
			owner, err = capabilityOwner(capability)
			if err != nil {
				continue
			}
			if owner == ownerFilter {
				capabilities = append(capabilities, capInterface)
			}
		}
	} else if capabilityTypeFilter != "" {
		protoCapabilities, err := bcs.db.GetCapabilitiesByType(capabilityTypeFilter)
		if err == nil {
			// Convert slice to []interface{}
			interfaceSlice := make([]interface{}, len(protoCapabilities))
			for i, proto := range protoCapabilities {
				interfaceSlice[i] = proto
			}
			capabilities, err = ConvertProtoCapabilitiesToInterfaces(interfaceSlice)
			if err != nil {
				log.Printf("Failed to convert capabilities: %v", err)
			}
		}
	} else if ownerFilter != "" {
		protoCapabilities, err := bcs.db.GetCapabilitiesByOwner(ownerFilter)
		if err == nil {
			// Convert slice to []interface{}
			interfaceSlice := make([]interface{}, len(protoCapabilities))
			for i, proto := range protoCapabilities {
				interfaceSlice[i] = proto
			}
			capabilities, err = ConvertProtoCapabilitiesToInterfaces(interfaceSlice)
			if err != nil {
				log.Printf("Failed to convert capabilities: %v", err)
			}
		}
	} else {
		// No filters, get all (this might be too broad for production, consider pagination or default limits)
		// For now, let's assume we want all if no filters.
		// A more robust solution would be to iterate through all capability keys or have a dedicated "GetAllCapabilities" in DB.
		// This is a simplified approach for now:
		http.Error(w, "Listing all capabilities without filters is not yet fully optimized. Please provide at least one filter (type or owner).", http.StatusNotImplemented)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"capabilities": capabilities, // Return the actual fetched capabilities
	})
}

// handleMCPGetContextRecordByID handles the retrieval of a context record by its ID
// Route: GET /mcp/context/{id}
func (bcs *BlockchainServer) handleMCPGetContextRecordByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract the context record ID from the URL path
	path := r.URL.Path
	if !strings.HasPrefix(path, "/mcp/context/") {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	// Get the context record ID from the path (everything after /mcp/context/)
	contextID := strings.TrimPrefix(path, "/mcp/context/")
	if contextID == "" {
		http.Error(w, "Missing context record ID", http.StatusBadRequest)
		return
	}

	// Log the request
	log.Printf("[INFO] Getting context record with ID: %s", contextID)

	// Get the context record from the database
	contextRecord, err := bcs.db.GetContextRecord(contextID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get context record: %v", err), http.StatusNotFound)
		return
	}

	// Return the context record
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(contextRecord); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
		return
	}
}

// handleMCPListContextRecordsByCapability handles the listing of context records for a capability
// Route: GET /mcp/capability/contexts/{capability_id}
func (bcs *BlockchainServer) handleMCPListContextRecordsByCapability(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract the capability ID from the URL path
	path := r.URL.Path
	if !strings.HasPrefix(path, "/mcp/capability/contexts/") {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	// Get the capability ID from the path (everything after /mcp/capability/contexts/)
	capabilityID := strings.TrimPrefix(path, "/mcp/capability/contexts/")
	if capabilityID == "" {
		http.Error(w, "Missing capability ID", http.StatusBadRequest)
		return
	}

	// Log the request
	log.Printf("[INFO] Listing context records for capability with ID: %s", capabilityID)

	// Get the context records for the capability from the database
	contextRecords, err := bcs.db.GetContextRecordsForCapability(capabilityID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get context records: %v", err), http.StatusInternalServerError)
		return
	}

	// Return the context records
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(contextRecords); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
		return
	}
}

// handleMCPListAllContextRecords handles the listing of all context records
// Route: GET /mcp/contexts
func (bcs *BlockchainServer) handleMCPListAllContextRecords(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Log the request
	log.Printf("[INFO] Listing all context records")

	// Get all context records from the database
	contextRecords, err := bcs.db.GetAllContextRecords()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get context records: %v", err), http.StatusInternalServerError)
		return
	}

	// Return the context records
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(contextRecords); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
		return
	}
}

// resolveIdentifier checks if the input string is a URI and parses it to get the ID,
// otherwise returns the input string as is (assuming it's a raw ID).
func resolveIdentifier(identifierOrURI string) (string, error) {
	if strings.HasPrefix(identifierOrURI, "knirv://") {
		id, _, _, _, err := uri.ParseResourceURI(identifierOrURI) // We only need the ID part here
		if err != nil {
			return "", fmt.Errorf("invalid URI format '%s': %w", identifierOrURI, err)
		}
		return id, nil
	}
	return identifierOrURI, nil // Assume it's a raw ID
}

// Add NFT-related handlers to the BlockchainServer struct
func (bcs *BlockchainServer) handleNFTList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	address := bcs.BlockchainPtr.GetWallet().GetAddress()
	nfts, err := bcs.BlockchainPtr.nftManager.GetNFTsByOwner(address)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get NFTs: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"nfts": nfts,
	})
}

func (bcs *BlockchainServer) handleNFTUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse multipart form
	err := r.ParseMultipartForm(10 << 20) // 10 MB max
	if err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	description := r.FormValue("description")
	if name == "" || description == "" {
		http.Error(w, "Name and description are required", http.StatusBadRequest)
		return
	}

	file, handler, err := r.FormFile("image")
	if err != nil {
		http.Error(w, "Failed to get image file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Validate file type
	if !strings.HasSuffix(strings.ToLower(handler.Filename), ".jpg") &&
		!strings.HasSuffix(strings.ToLower(handler.Filename), ".jpeg") &&
		!strings.HasSuffix(strings.ToLower(handler.Filename), ".png") {
		http.Error(w, "Only JPG and PNG files are allowed", http.StatusBadRequest)
		return
	}

	// Create uploads directory if it doesn't exist
	uploadDir := "uploads"
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		http.Error(w, "Failed to create upload directory", http.StatusInternalServerError)
		return
	}

	// Generate unique filename
	filename := fmt.Sprintf("%d_%s", time.Now().UnixNano(), handler.Filename)
	filepath := filepath.Join(uploadDir, filename)

	// Save file
	dst, err := os.Create(filepath)
	if err != nil {
		http.Error(w, "Failed to save file", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		http.Error(w, "Failed to save file", http.StatusInternalServerError)
		return
	}

	// Create NFT
	address := bcs.BlockchainPtr.GetWallet().GetAddress()
	nft, err := bcs.BlockchainPtr.nftManager.CreateNFT(name, description, filepath, address)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create NFT: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(nft)
}

func (bcs *BlockchainServer) handleNFTCapabilityAttachment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		NFTID        string                 `json:"nftId"`
		CapabilityID string                 `json:"capabilityId"`
		Params       map[string]interface{} `json:"params,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.NFTID == "" || req.CapabilityID == "" {
		http.Error(w, "NFT ID and Capability ID are required", http.StatusBadRequest)
		return
	}

	address := bcs.BlockchainPtr.GetWallet().GetAddress()
	attachment, err := bcs.BlockchainPtr.nftManager.AttachCapability(req.NFTID, req.CapabilityID, address, req.Params)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to attach capability: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(attachment)
}

func (bcs *BlockchainServer) handleNFTCapabilityHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	vars := mux.Vars(r)
	nftID := vars["badgeId"]
	if nftID == "" {
		http.Error(w, "NFT ID is required", http.StatusBadRequest)
		return
	}

	history, err := bcs.BlockchainPtr.nftManager.GetCapabilityAttachmentHistory(nftID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get capability history: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"history": history,
	})
}

// ===== RESOURCE CAPABILITY MANAGEMENT API HANDLERS (Phase 3) =====

// handleAddResourceCapability handles adding a resource capability to an agent
func (bcs *BlockchainServer) handleAddResourceCapability(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		AgentID      string                 `json:"agent_id"`
		Name         string                 `json:"name"`
		Description  string                 `json:"description"`
		ResourceType string                 `json:"resource_type"`
		Metadata     map[string]interface{} `json:"metadata"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.AgentID == "" || req.Name == "" || req.ResourceType == "" {
		http.Error(w, "Agent ID, name, and resource type are required", http.StatusBadRequest)
		return
	}

	// Get the agent manager
	agentManager := bcs.BlockchainPtr.agentManager
	if agentManager == nil {
		http.Error(w, "Agent manager not available", http.StatusInternalServerError)
		return
	}

	// Add the resource capability
	capability, err := agentManager.AddResourceCapabilityToAgent(
		req.AgentID,
		req.Name,
		req.Description,
		req.ResourceType,
		req.Metadata,
	)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to add resource capability: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":        "success",
		"capability_id": capability.ID,
		"capability":    capability,
		"message":       "Resource capability added successfully",
	})
}

// handleLinkResourceCapability handles linking a resource capability to a functional capability
func (bcs *BlockchainServer) handleLinkResourceCapability(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		AgentID                string                 `json:"agent_id"`
		ResourceCapabilityID   string                 `json:"resource_capability_id"`
		FunctionalCapabilityID string                 `json:"functional_capability_id"`
		Parameters             map[string]interface{} `json:"parameters,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.AgentID == "" || req.ResourceCapabilityID == "" || req.FunctionalCapabilityID == "" {
		http.Error(w, "Agent ID, resource capability ID, and functional capability ID are required", http.StatusBadRequest)
		return
	}

	// Get the agent manager
	agentManager := bcs.BlockchainPtr.agentManager
	if agentManager == nil {
		http.Error(w, "Agent manager not available", http.StatusInternalServerError)
		return
	}

	// Link the resource capability
	err := agentManager.LinkResourceToCapability(
		req.AgentID,
		req.ResourceCapabilityID,
		req.FunctionalCapabilityID,
		req.Parameters,
	)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to link resource capability: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": "Resource capability linked successfully",
	})
}

// handleCreateResourceCapabilityGroup handles creating a resource capability group
func (bcs *BlockchainServer) handleCreateResourceCapabilityGroup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		AgentID       string   `json:"agent_id"`
		GroupName     string   `json:"group_name"`
		Description   string   `json:"description"`
		CapabilityIDs []string `json:"capability_ids"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.AgentID == "" || req.GroupName == "" || len(req.CapabilityIDs) == 0 {
		http.Error(w, "Agent ID, group name, and capability IDs are required", http.StatusBadRequest)
		return
	}

	// Get the agent manager
	agentManager := bcs.BlockchainPtr.agentManager
	if agentManager == nil {
		http.Error(w, "Agent manager not available", http.StatusInternalServerError)
		return
	}

	// Create the resource capability group
	err := agentManager.CreateResourceCapabilityGroup(
		req.AgentID,
		req.GroupName,
		req.Description,
		req.CapabilityIDs,
	)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create resource capability group: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":     "success",
		"group_name": req.GroupName,
		"message":    "Resource capability group created successfully",
	})
}

// handleListResourceCapabilities handles listing resource capabilities for an agent
func (bcs *BlockchainServer) handleListResourceCapabilities(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	agentID := r.URL.Query().Get("agent_id")
	if agentID == "" {
		http.Error(w, "Agent ID is required", http.StatusBadRequest)
		return
	}

	// Get the agent manager
	agentManager := bcs.BlockchainPtr.agentManager
	if agentManager == nil {
		http.Error(w, "Agent manager not available", http.StatusInternalServerError)
		return
	}

	// Get resource capabilities
	capabilities, err := agentManager.GetResourceCapabilities(agentID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get resource capabilities: %v", err), http.StatusInternalServerError)
		return
	}

	// Get resource capability groups
	groups, err := agentManager.GetResourceCapabilityGroups(agentID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get resource capability groups: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":       "success",
		"agent_id":     agentID,
		"capabilities": capabilities,
		"groups":       groups,
	})
}

// handleInvokeResourceCapability handles invoking a resource capability
func (bcs *BlockchainServer) handleInvokeResourceCapability(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		AgentID      string                 `json:"agent_id"`
		CapabilityID string                 `json:"capability_id"`
		Parameters   map[string]interface{} `json:"parameters,omitempty"`
		Initiator    string                 `json:"initiator"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.AgentID == "" || req.CapabilityID == "" || req.Initiator == "" {
		http.Error(w, "Agent ID, capability ID, and initiator are required", http.StatusBadRequest)
		return
	}

	// Get the agent manager
	agentManager := bcs.BlockchainPtr.agentManager
	if agentManager == nil {
		http.Error(w, "Agent manager not available", http.StatusInternalServerError)
		return
	}

	// Invoke the resource capability
	result, err := agentManager.InvokeResourceCapability(
		req.AgentID,
		req.CapabilityID,
		req.Parameters,
		req.Initiator,
	)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to invoke resource capability: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"result": result,
	})
}

// handleGetResourceCapabilityHistory handles getting the invocation history for a resource capability
func (bcs *BlockchainServer) handleGetResourceCapabilityHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	agentID := r.URL.Query().Get("agent_id")
	capabilityID := r.URL.Query().Get("capability_id")

	if agentID == "" || capabilityID == "" {
		http.Error(w, "Agent ID and capability ID are required", http.StatusBadRequest)
		return
	}

	// Get the agent manager
	agentManager := bcs.BlockchainPtr.agentManager
	if agentManager == nil {
		http.Error(w, "Agent manager not available", http.StatusInternalServerError)
		return
	}

	// Get the invocation history
	history, err := agentManager.GetResourceCapabilityInvocationHistory(agentID, capabilityID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get resource capability history: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":        "success",
		"agent_id":      agentID,
		"capability_id": capabilityID,
		"history":       history,
	})
}

// HandleGetAgentFacts handles retrieving AgentFacts metadata for an agent
func (bcs *BlockchainServer) HandleGetAgentFacts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract agent ID from URL path
	path := strings.TrimPrefix(r.URL.Path, "/agent/agent-facts/")
	agentID := strings.TrimSuffix(path, "/")

	if agentID == "" {
		http.Error(w, "Agent ID is required", http.StatusBadRequest)
		return
	}

	// Get the agent manager
	agentManager := bcs.BlockchainPtr.agentManager
	if agentManager == nil {
		http.Error(w, "Agent manager not available", http.StatusInternalServerError)
		return
	}

	// Get the Agent
	agent, err := agentManager.GetAgent(agentID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "Agent not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to get agent: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Return AgentFacts metadata
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(agent.Metadata)
}

// HandleGetAgentCapabilities handles retrieving capabilities for an agent
func (bcs *BlockchainServer) HandleGetAgentCapabilities(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract agent ID from URL path
	path := strings.TrimPrefix(r.URL.Path, "/agent/capabilities/")
	agentID := strings.TrimSuffix(path, "/")

	if agentID == "" {
		http.Error(w, "Agent ID is required", http.StatusBadRequest)
		return
	}

	// Get the agent manager
	agentManager := bcs.BlockchainPtr.agentManager
	if agentManager == nil {
		http.Error(w, "Agent manager not available", http.StatusInternalServerError)
		return
	}

	// Get the Agent
	agent, err := agentManager.GetAgent(agentID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "Agent not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to get agent: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Return agent capabilities
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"agent_id":     agentID,
		"capabilities": agent.Capabilities,
	})
}

// HandleInvokeAgentCapability handles invoking a specific capability of an agent
func (bcs *BlockchainServer) HandleInvokeAgentCapability(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse JSON request
	var request struct {
		AgentID      string                 `json:"agent_id"`
		CapabilityID string                 `json:"capability_id"`
		Input        map[string]interface{} `json:"input"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Failed to parse request: "+err.Error(), http.StatusBadRequest)
		return
	}

	if request.AgentID == "" || request.CapabilityID == "" {
		http.Error(w, "Agent ID and Capability ID are required", http.StatusBadRequest)
		return
	}

	// Get the agent manager
	agentManager := bcs.BlockchainPtr.agentManager
	if agentManager == nil {
		http.Error(w, "Agent manager not available", http.StatusInternalServerError)
		return
	}

	// Get the Agent
	agent, err := agentManager.GetAgent(request.AgentID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "Agent not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to get agent: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Find the target capability
	var targetCapability *Capability
	for i := range agent.Capabilities {
		if agent.Capabilities[i].ID == request.CapabilityID {
			targetCapability = &agent.Capabilities[i]
			break
		}
	}

	if targetCapability == nil {
		http.Error(w, "Capability not found", http.StatusNotFound)
		return
	}

	// Create a context record for the invocation
	contextRecord := &types.ContextRecord{
		ID:              fmt.Sprintf("ctx:%s", generateUniqueID()),
		CapabilityID:    request.CapabilityID, // Used in JSON response
		InteractionType: "TOOL_INVOCATION",    // Used in JSON response
		Initiator:       "api_user",           // Used in JSON response
		Timestamp:       time.Now().Unix(),    // Used in JSON response
		Status:          "completed",
		Details: map[string]interface{}{
			"agent_id":        request.AgentID,
			"capability_name": targetCapability.Name,
			"capability_type": targetCapability.CapabilityType,
			"input":           request.Input,
			"output": map[string]interface{}{
				"capability_id":   request.CapabilityID,
				"capability_name": targetCapability.Name,
				"invoked_at":      time.Now().Unix(),
				"result":          "Agent capability invocation completed successfully",
				"metadata":        targetCapability.Metadata,
			},
		},
	}

	// Return the result
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"context_record_id": contextRecord.ID,
		"capability_id":     contextRecord.CapabilityID,
		"interaction_type":  contextRecord.InteractionType,
		"initiator":         contextRecord.Initiator,
		"timestamp":         contextRecord.Timestamp,
		"status":            contextRecord.Status,
		"details":           contextRecord.Details,
		"output":            contextRecord.Details["output"],
	})
}

// ===== PoAu-D CONSENSUS API HANDLERS =====

// EnablePoAuD enables the PoAu-D consensus mechanism
func (bcs *BlockchainServer) EnablePoAuD(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	bcs.BlockchainPtr.Lock()
	defer bcs.BlockchainPtr.Unlock()

	bcs.BlockchainPtr.PoAuDEnabled = true

	// Save the setting to LevelDB
	if bcs.BlockchainPtr.db != nil {
		if err := bcs.BlockchainPtr.db.PutPoAuDEnabled(true); err != nil {
			http.Error(w, fmt.Sprintf("Failed to save PoAu-D setting: %v", err), http.StatusInternalServerError)
			return
		}
	}

	// Restart mining to use the new consensus mechanism
	bcs.BlockchainPtr.StopMiningGracefully()
	bcs.BlockchainPtr.StartMining()

	w.Header().Add("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"enabled": true,
		"message": "PoAu-D consensus mechanism enabled",
	})
}

// DisablePoAuD disables the PoAu-D consensus mechanism and falls back to PoW
func (bcs *BlockchainServer) DisablePoAuD(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	bcs.BlockchainPtr.Lock()
	defer bcs.BlockchainPtr.Unlock()

	bcs.BlockchainPtr.PoAuDEnabled = false

	// Save the setting to LevelDB
	if bcs.BlockchainPtr.db != nil {
		if err := bcs.BlockchainPtr.db.PutPoAuDEnabled(false); err != nil {
			http.Error(w, fmt.Sprintf("Failed to save PoAu-D setting: %v", err), http.StatusInternalServerError)
			return
		}
	}

	// Restart mining to use PoW
	bcs.BlockchainPtr.StopMiningGracefully()
	bcs.BlockchainPtr.StartMining()

	w.Header().Add("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"enabled": false,
		"message": "PoAu-D consensus mechanism disabled, using PoW",
	})
}

// GetPoAuDStatus returns the current status of PoAu-D
func (bcs *BlockchainServer) GetPoAuDStatus(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	bcs.BlockchainPtr.Lock()
	defer bcs.BlockchainPtr.Unlock()

	status := map[string]interface{}{
		"enabled": bcs.BlockchainPtr.PoAuDEnabled,
	}

	if bcs.BlockchainPtr.PoAuDEnabled {
		// Add additional status information if PoAu-D is enabled
		status["network_authors_count"] = len(bcs.BlockchainPtr.NetworkAuthors)

		// If this node has a transaction pool manager
		if bcs.BlockchainPtr.TransactionPoolManager != nil {
			poolStats := bcs.BlockchainPtr.TransactionPoolManager.GetPoolStats()
			for k, v := range poolStats {
				status[k] = v
			}
		}

		// Add delegation statistics
		status["delegation_stats"] = GetDelegationStats(bcs.BlockchainPtr.TransactionPoolManager)
	}

	w.Header().Add("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// AddNetworkAuthor adds an address to the Network Authors set
func (bcs *BlockchainServer) AddNetworkAuthor(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var requestBody struct {
		Address string `json:"address"`
	}

	if err := json.NewDecoder(req.Body).Decode(&requestBody); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if requestBody.Address == "" {
		http.Error(w, "Address is required", http.StatusBadRequest)
		return
	}

	if err := bcs.BlockchainPtr.AddNetworkAuthor(requestBody.Address); err != nil {
		http.Error(w, fmt.Sprintf("Failed to add network author: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Added %s as Network Author", requestBody.Address),
		"address": requestBody.Address,
	})
}

// RemoveNetworkAuthor removes an address from the Network Authors set
func (bcs *BlockchainServer) RemoveNetworkAuthor(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var requestBody struct {
		Address string `json:"address"`
	}

	if err := json.NewDecoder(req.Body).Decode(&requestBody); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if requestBody.Address == "" {
		http.Error(w, "Address is required", http.StatusBadRequest)
		return
	}

	if err := bcs.BlockchainPtr.RemoveNetworkAuthor(requestBody.Address); err != nil {
		http.Error(w, fmt.Sprintf("Failed to remove network author: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Removed %s from Network Authors", requestBody.Address),
		"address": requestBody.Address,
	})
}

// GetNetworkAuthors returns the list of current Network Authors
func (bcs *BlockchainServer) GetNetworkAuthors(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	authors := bcs.BlockchainPtr.GetNetworkAuthors()

	// Convert map to slice for easier consumption
	authorsList := make([]string, 0, len(authors))
	for addr := range authors {
		authorsList = append(authorsList, addr)
	}

	w.Header().Add("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"network_authors": authorsList,
		"count":           len(authorsList),
	})
}

// handleTxAccumProof serves the first hop of a two-hop light-client proof. A
// caller may pass ?target_height=H to bind the proof to a known checkpoint;
// without it, the proof targets the current chain tip.
func (bcs *BlockchainServer) handleTxAccumProof(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	txHash := strings.TrimPrefix(req.URL.Path, "/proof/tx/")
	if txHash == "" {
		http.Error(w, "transaction hash required", http.StatusBadRequest)
		return
	}
	var targetHeight uint64
	if raw := req.URL.Query().Get("target_height"); raw != "" {
		parsed, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			http.Error(w, "invalid target_height", http.StatusBadRequest)
			return
		}
		targetHeight = parsed
	}

	bcs.BlockchainPtr.Lock()
	chainID := bcs.BlockchainPtr.ChainID
	blocks := make([]*Block, len(bcs.BlockchainPtr.Blocks))
	for i, block := range bcs.BlockchainPtr.Blocks {
		if block != nil {
			blocks[i] = block.DeepCopy()
		}
	}
	bcs.BlockchainPtr.Unlock()

	proof, err := GenerateTxAccumProof(chainID, txHash, targetHeight, blocks)
	if err != nil {
		status := http.StatusBadRequest
		if strings.Contains(err.Error(), "not found") {
			status = http.StatusNotFound
		}
		http.Error(w, err.Error(), status)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(proof)
}

// handleP2PReceivedBlock processes a gossiped block forwarded by KNIRVGATEWAY.
func (bcs *BlockchainServer) handleP2PReceivedBlock(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	data, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "read body: "+err.Error(), http.StatusBadRequest)
		return
	}
	if bcs.consensusManager != nil {
		bcs.consensusManager.HandleReceivedBlockData(data)
	}
	w.WriteHeader(http.StatusOK)
}

// handleP2PReceivedTx processes a gossiped transaction forwarded by KNIRVGATEWAY.
func (bcs *BlockchainServer) handleP2PReceivedTx(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	data, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "read body: "+err.Error(), http.StatusBadRequest)
		return
	}
	if bcs.consensusManager != nil {
		bcs.consensusManager.HandleReceivedTransactionData(data)
	}
	w.WriteHeader(http.StatusOK)
}

// handleChainSync responds to chain sync requests proxied by KNIRVGATEWAY.
func (bcs *BlockchainServer) handleChainSync(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	data, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "read body: "+err.Error(), http.StatusBadRequest)
		return
	}
	if bcs.consensusManager == nil {
		http.Error(w, "consensus manager not available", http.StatusServiceUnavailable)
		return
	}
	resp, err := bcs.consensusManager.HandleSyncRequest(data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(resp)
}
