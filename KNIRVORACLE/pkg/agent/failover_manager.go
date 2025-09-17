package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"KNIRVORACLE/config"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

const (
	rootOfflineThreshold = 15 * time.Minute
	rootPingInterval     = 1 * time.Minute
	NetworkControlTopic  = "network-control"
)

// Network control message types for failover protocol
type NetworkControlMessage struct {
	Type    string `json:"type"`
	Payload any    `json:"payload"`
}

type NetworkPausePayload struct {
	InitiatorPeerID string `json:"initiator_peer_id"`
	Reason          string `json:"reason"`
	Timestamp       int64  `json:"timestamp"`
}

// FailoverManager is responsible for monitoring the root node and initiating failover.
type FailoverManager struct {
	mu                  sync.Mutex
	currentOracleAPIURL string // API URL of the root node to monitor
	httpClient          *http.Client
	lastOracleSeen      time.Time
	isOracleOnline      bool
	nodeConfig          *config.Config // This bootnode's current configuration
	nodeConfigPath      string         // Path to this bootnode's config file
	walletManager       *WalletManager // Wallet manager for master wallet creation
	wallet              *Wallet        // The bootnode's wallet
	pubsub              *pubsub.PubSub // For network control messages
	stopChan            chan struct{}
	failoverInProgress  bool
	mainContextCancelFn context.CancelFunc // To signal the main application to shut down its current role
	networkPaused       bool
	networkPauseMux     sync.RWMutex
}

// NewFailoverManager creates a new FailoverManager.
func NewFailoverManager(rootAPIURL string, cfg *config.Config, configPath string, wm *WalletManager, wallet *Wallet, ps *pubsub.PubSub, mainCancelFn context.CancelFunc) *FailoverManager {
	if rootAPIURL == "" {
		log.Println("[FailoverManager] No root API URL configured, failover monitoring disabled.")
		return nil // Or handle appropriately
	}
	return &FailoverManager{
		currentOracleAPIURL: strings.TrimSuffix(rootAPIURL, "/"),
		httpClient:          &http.Client{Timeout: 10 * time.Second},
		lastOracleSeen:      time.Now(),
		isOracleOnline:      true,
		nodeConfig:          cfg,
		nodeConfigPath:      configPath,
		walletManager:       wm,
		wallet:              wallet,
		pubsub:              ps,
		stopChan:            make(chan struct{}),
		mainContextCancelFn: mainCancelFn,
	}
}

// StartMonitoring begins the root node monitoring process.
func (fm *FailoverManager) StartMonitoring() {
	if fm == nil {
		return
	}
	log.Printf("[FailoverManager] Starting monitoring of root node at %s", fm.currentOracleAPIURL)
	go fm.monitorOracleNodeLoop()
}

// StopMonitoring stops the root node monitoring process.
func (fm *FailoverManager) StopMonitoring() {
	if fm == nil {
		return
	}
	log.Println("[FailoverManager] Stopping root node monitoring...")
	close(fm.stopChan)
}

func (fm *FailoverManager) monitorOracleNodeLoop() {
	ticker := time.NewTicker(rootPingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-fm.stopChan:
			log.Println("[FailoverManager] Monitoring loop stopped.")
			return
		case <-ticker.C:
			fm.checkOracleStatus()
		}
	}
}

func (fm *FailoverManager) checkOracleStatus() {
	fm.mu.Lock()
	if fm.failoverInProgress {
		fm.mu.Unlock()
		return
	}
	fm.mu.Unlock()

	healthURL := fm.currentOracleAPIURL + "/health" // Assuming a /health endpoint
	resp, err := fm.httpClient.Get(healthURL)
	fm.mu.Lock()

	var statusCode int
	var status string
	if resp != nil {
		statusCode = resp.StatusCode
		status = resp.Status
		resp.Body.Close() // Always close the response body
	}

	if err != nil || statusCode != http.StatusOK {
		if fm.isOracleOnline {
			log.Printf("[FailoverManager] Oracle node at %s appears to be offline. Error: %v, Status: %s", fm.currentOracleAPIURL, err, status)
			fm.isOracleOnline = false
			// lastOracleSeen remains the time it was last confirmed online
		}
		// Check if offline threshold exceeded
		if !fm.isOracleOnline && time.Since(fm.lastOracleSeen) > rootOfflineThreshold {
			log.Printf("[FailoverManager] Oracle node offline threshold exceeded (15 minutes). Initiating failover check.")
			fm.failoverInProgress = true      // Prevent multiple failover attempts
			fm.mu.Unlock()                    // Unlock before calling potentially long-running function
			go fm.initiateFailoverProcedure() // Run in a new goroutine
			return
		}
	} else {
		if !fm.isOracleOnline {
			log.Printf("[FailoverManager] Oracle node at %s is back online.", fm.currentOracleAPIURL)
		}
		fm.isOracleOnline = true
		fm.lastOracleSeen = time.Now()
	}
	if resp != nil {
		resp.Body.Close()
	}
	fm.mu.Unlock()
}

func (fm *FailoverManager) amIElectedToBecomeOracle() bool {
	// Placeholder for stake-based leader election.
	// In a real system, this would involve:
	// 1. Discovering other eligible bootnodes.
	// 2. Communicating stake information (e.g., NRN balance, registered stake).
	// 3. Running a consensus algorithm to elect the leader.
	// For this example, we'll assume this node is always elected if the root is down.
	log.Println("[FailoverManager] Placeholder: Assuming this node is elected to become the new root.")
	return true
}

func (fm *FailoverManager) initiateFailoverProcedure() {
	log.Println("[FailoverManager] Attempting to become the new root node...")
	if !fm.amIElectedToBecomeOracle() {
		log.Println("[FailoverManager] Not elected to become root. Resuming monitoring.")
		fm.mu.Lock()
		fm.failoverInProgress = false
		fm.mu.Unlock()
		return
	}

	// Broadcast NetworkPause message if pubsub is available
	if fm.pubsub != nil {
		fm.broadcastNetworkPause()
	}

	log.Println("[FailoverManager] This node IS ELECTED. Proceeding with promotion to Oracle role.")
	// Signal the main application to shut down its current role and then promote.
	// The actual promotion and restart will be handled in main.go after shutdown.
	fm.mainContextCancelFn() // This will trigger the shutdown sequence in main.go
	// The main.go shutdown sequence should then call a function like promoteToOracleAndRestart
}

func (fm *FailoverManager) broadcastNetworkPause() {
	// Join the network control topic
	controlTopic, err := fm.pubsub.Join(NetworkControlTopic)
	if err != nil {
		log.Printf("[FailoverManager] Error joining network control topic: %v", err)
		return
	}

	// Create and broadcast the pause message
	// Use a placeholder for PeerID since it's not in the config yet
	pausePayload := NetworkPausePayload{
		InitiatorPeerID: "<bootnode-peer-id>", // TODO: Get actual PeerID from current bootnode
		Reason:          "Oracle node failover in progress",
		Timestamp:       time.Now().Unix(),
	}
	payloadBytes, _ := json.Marshal(pausePayload)
	controlMsg := NetworkControlMessage{Type: "NetworkPause", Payload: payloadBytes}

	msgBytes, _ := json.Marshal(controlMsg)
	log.Printf("[FailoverManager] Broadcasting NetworkPause message...")
	if err := controlTopic.Publish(context.Background(), msgBytes); err != nil {
		log.Printf("[FailoverManager] Error publishing NetworkPause message: %v", err)
	}
}

// Global FailoverManager instance
var fmGlobalForFailover *FailoverManager

// SetGlobalFailoverManager sets the global failover manager instance
func SetGlobalFailoverManager(fm *FailoverManager) {
	fmGlobalForFailover = fm
}

// GetGlobalFailoverManager returns the global failover manager instance
func GetGlobalFailoverManager() *FailoverManager {
	return fmGlobalForFailover
}

// GetCurrentOracleAPIURL returns the current oracle API URL
func (fm *FailoverManager) GetCurrentOracleAPIURL() string {
	fm.mu.Lock()
	defer fm.mu.Unlock()
	return fm.currentOracleAPIURL
}

// IsOracleOnline returns whether the oracle is currently online
func (fm *FailoverManager) IsOracleOnline() bool {
	fm.mu.Lock()
	defer fm.mu.Unlock()
	return fm.isOracleOnline
}

// CheckOracleStatus performs a status check on the oracle
func (fm *FailoverManager) CheckOracleStatus() {
	fm.checkOracleStatus()
}

// AmIElectedToBecomeOracle checks if this node is elected to become oracle
func (fm *FailoverManager) AmIElectedToBecomeOracle() bool {
	return fm.amIElectedToBecomeOracle()
}

// HandleFailoverPromotion handles the promotion process after shutdown
func HandleFailoverPromotion(configPath string, cfg *config.Config, wm *WalletManager) error {
	if fmGlobalForFailover == nil {
		return nil // No failover manager, nothing to do
	}

	// Check if this was a failover-triggered shutdown
	fmGlobalForFailover.mu.Lock()
	wasFailover := fmGlobalForFailover.failoverInProgress
	fmGlobalForFailover.mu.Unlock()

	if wasFailover {
		log.Println("[FailoverManager] Detected failover-triggered shutdown, proceeding with promotion...")
		return promoteToOracleAndRestart(configPath, cfg, wm)
	}

	return nil
}

func promoteToOracleAndRestart(currentConfigPath string, currentBootnodeCfg *config.Config, wm *WalletManager) error {
	log.Println("PROMOTING to Oracle (Oracle) Node and restarting...")

	// 1. Validate wallet manager
	if wm == nil {
		return fmt.Errorf("critical: WalletManager is nil, cannot proceed with promotion")
	}

	// 2. Get the bootnode's master wallet address for MinersAddress
	if currentBootnodeCfg.MinersAddress == "" {
		// Fallback to MasterAddress if MinersAddress is empty
		if currentBootnodeCfg.MasterAddress != "" {
			log.Printf("Warning: Bootnode's MinersAddress is empty, using MasterAddress '%s' for new root's MinersAddress.", currentBootnodeCfg.MasterAddress)
			currentBootnodeCfg.MinersAddress = currentBootnodeCfg.MasterAddress
		} else {
			return fmt.Errorf("critical: bootnode's MinersAddress and MasterAddress are empty in its configuration. Cannot determine MinersAddress for the new root. Promotion aborted")
		}
	}
	bootnodeMinersAddress := currentBootnodeCfg.MinersAddress
	log.Printf("Bootnode's MinersAddress to be used for new root's MinersAddress: %s", bootnodeMinersAddress)
	log.Printf("New root node MasterAddress (faucet/genesis) will be: %s", currentBootnodeCfg.MasterAddress)

	// 3. Ensure master wallet exists and is accessible
	if currentBootnodeCfg.MasterAddress != "" {
		_, err := wm.LoadMasterWallet(currentBootnodeCfg.MasterAddress, config.RoleBootnode)
		if err != nil {
			log.Printf("Warning: Could not load master wallet for address %s: %v", currentBootnodeCfg.MasterAddress, err)
		} else {
			log.Printf("Successfully validated master wallet access for address: %s", currentBootnodeCfg.MasterAddress)
		}
	}

	// 4. Get the bootnode's database path
	bootnodeDbPath := currentBootnodeCfg.BlockchainDatabasePath
	if bootnodeDbPath == "" {
		// Fallback to SearchableDatabasePath if BlockchainDatabasePath is empty
		if currentBootnodeCfg.SearchableDatabasePath != "" {
			log.Printf("Warning: Bootnode's BlockchainDatabasePath is empty, using SearchableDatabasePath '%s' for new root's BlockchainDatabasePath.", currentBootnodeCfg.SearchableDatabasePath)
			bootnodeDbPath = currentBootnodeCfg.SearchableDatabasePath
		} else {
			return fmt.Errorf("critical: bootnode's BlockchainDatabasePath and SearchableDatabasePath are empty in its configuration. Cannot determine database path for the new root. Promotion aborted")
		}
	}
	log.Printf("Bootnode's database path to be used for new root: %s", bootnodeDbPath)

	// 5. Create the new root configuration based on DefaultConfig
	newOracleCfg := config.DefaultConfig()

	// 6. Override specific fields for the new root
	newOracleCfg.IsRoot = true
	newOracleCfg.IsBootnode = false
	newOracleCfg.ClientOnly = false
	newOracleCfg.InstallComplete = true

	// ChainID: Generate a new one for the promotional root node
	newOracleCfg.ChainID = fmt.Sprintf("KNIRVORACLE-PROMOTION-%d", time.Now().Unix())

	// MinersAddress: Use the bootnode's MinersAddress
	newOracleCfg.MinersAddress = bootnodeMinersAddress
	log.Printf("New root node MinersAddress will be: %s (from bootnode's MinersAddress)", newOracleCfg.MinersAddress)
	log.Printf("New root node MasterAddress (faucet/genesis) will be: %s", newOracleCfg.MasterAddress)

	// BlockchainDatabasePath: Use the bootnode's existing database path
	newOracleCfg.BlockchainDatabasePath = bootnodeDbPath
	log.Printf("New root node BlockchainDatabasePath will be: %s (from bootnode's database)", newOracleCfg.BlockchainDatabasePath)

	// 7. Save the new configuration
	if _, err := config.SaveConfig(currentConfigPath, newOracleCfg); err != nil {
		return fmt.Errorf("failed to save new root config to %s: %w", currentConfigPath, err)
	}
	log.Printf("New root configuration saved to %s. Relaunching...", currentConfigPath)

	// 8. Relaunch the application
	cmd := exec.Command(os.Args[0], os.Args[1:]...)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to relaunch application as root: %w", err)
	}
	os.Exit(0) // Exit old process
	return nil // Should not be reached
}

// IsNetworkPaused returns whether the network is currently paused
func (fm *FailoverManager) IsNetworkPaused() bool {
	if fm == nil {
		return false
	}
	fm.networkPauseMux.RLock()
	defer fm.networkPauseMux.RUnlock()
	return fm.networkPaused
}

// SetNetworkPaused sets the network pause state
func (fm *FailoverManager) SetNetworkPaused(paused bool) {
	if fm == nil {
		return
	}
	fm.networkPauseMux.Lock()
	defer fm.networkPauseMux.Unlock()

	if paused {
		log.Printf("[FailoverManager] Network operations paused for %s", fm.nodeConfig.ChainID)
	} else {
		log.Printf("[FailoverManager] Network operations resumed for %s", fm.nodeConfig.ChainID)
	}

	fm.networkPaused = paused
}
