

---

**Source**: KNIRVROOT/docs/pending_implementation_plans/Failover_Implementation_Plan.md

# KNIRVCHAIN Root Node Failover Implementation Plan

## Overview

This document outlines a comprehensive implementation plan for a resilient failover mechanism in the KNIRVCHAIN network. The mechanism enables automatic recovery when the root node becomes unavailable, with the bootnode having the highest stake taking over as the new root node.

## Core Concepts

1. **Root Node Liveness Monitoring**: Bootnodes continuously monitor the health of the current root node
2. **Failure Detection**: If the root node is unresponsive for 15 minutes, a failover process is triggered
3. **Leader Election**: Bootnodes communicate to determine which has the highest stake and is eligible to become the new root
4. **Network Pause**: The network temporarily pauses operations during the transition
5. **Promotion and Restart**: The elected bootnode shuts down its bootnode operations, updates its configuration to Root, and restarts as the new root node
6. **Network Resume**: The new root node broadcasts a resume message to the network
7. **Chain Continuity**: The new root node uses its existing database to maintain chain continuity

## Phase 1: Core Monitoring Infrastructure

### 1.1 Add Configuration Support

**File: `config/config.go`**
```go
// Add to Config struct
type Config struct {
    // ... existing fields ...
    IsBootnode            bool   `json:"is_bootnode"`
    CurrentRootNodeAPIURL string `json:"current_root_node_api_url,omitempty"` // URL for bootnodes to monitor
    // ... other fields ...
}

// Update in DefaultBootnodeConfig()
// cfg.CurrentRootNodeAPIURL = "http://default-root-node-address:port" // Set a default or leave for installer
```

### 1.2 Create Failover Manager

**File: `failover_manager.go`**

Implement a new component responsible for monitoring the root node and initiating failover:

```go
package main

import (
    "context"
    "fmt"
    "log"
    "net/http"
    "os"
    "os/exec"
    "runtime"
    "strings"
    "sync"
    "time"

    "KNIRVCHAIN/config"
)

const (
    rootOfflineThreshold = 15 * time.Minute
    rootPingInterval     = 1 * time.Minute
)

// FailoverManager is responsible for monitoring the root node and initiating failover.
type FailoverManager struct {
    mu                  sync.Mutex
    currentRootAPIURL   string // API URL of the root node to monitor
    httpClient          *http.Client
    lastRootSeen        time.Time
    isRootOnline        bool
    nodeConfig          *config.Config    // This bootnode's current configuration
    nodeConfigPath      string            // Path to this bootnode's config file
    walletManager       *WalletManager    // Wallet manager for master wallet creation
    wallet              *Wallet           // The bootnode's wallet
    pubsub              *pubsub.PubSub    // For network control messages
    stopChan            chan struct{}
    failoverInProgress  bool
    mainContextCancelFn context.CancelFunc // To signal the main application to shut down its current role
}

// NewFailoverManager creates a new FailoverManager.
func NewFailoverManager(rootAPIURL string, cfg *config.Config, configPath string, wm *WalletManager, wallet *Wallet, ps *pubsub.PubSub, mainCancelFn context.CancelFunc) *FailoverManager {
    if rootAPIURL == "" {
        log.Println("[FailoverManager] No root API URL configured, failover monitoring disabled.")
        return nil // Or handle appropriately
    }
    return &FailoverManager{
        currentRootAPIURL:   strings.TrimSuffix(rootAPIURL, "/"),
        httpClient:          &http.Client{Timeout: 10 * time.Second},
        lastRootSeen:        time.Now(),
        isRootOnline:        true,
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
    log.Printf("[FailoverManager] Starting monitoring of root node at %s", fm.currentRootAPIURL)
    go fm.monitorRootNodeLoop()
}

// StopMonitoring stops the root node monitoring process.
func (fm *FailoverManager) StopMonitoring() {
    if fm == nil {
        return
    }
    log.Println("[FailoverManager] Stopping root node monitoring...")
    close(fm.stopChan)
}

func (fm *FailoverManager) monitorRootNodeLoop() {
    ticker := time.NewTicker(rootPingInterval)
    defer ticker.Stop()

    for {
        select {
        case <-fm.stopChan:
            log.Println("[FailoverManager] Monitoring loop stopped.")
            return
        case <-ticker.C:
            fm.checkRootStatus()
        }
    }
}

func (fm *FailoverManager) checkRootStatus() {
    fm.mu.Lock()
    if fm.failoverInProgress {
        fm.mu.Unlock()
        return
    }
    fm.mu.Unlock()

    healthURL := fm.currentRootAPIURL + "/health" // Assuming a /health endpoint
    resp, err := fm.httpClient.Get(healthURL)
    fm.mu.Lock()
    if err != nil || resp.StatusCode != http.StatusOK {
        if fm.isRootOnline {
            log.Printf("[FailoverManager] Root node at %s appears to be offline. Error: %v, Status: %s", fm.currentRootAPIURL, err, resp.Status)
            fm.isRootOnline = false
            // lastRootSeen remains the time it was last confirmed online
        }
        // Check if offline threshold exceeded
        if !fm.isRootOnline && time.Since(fm.lastRootSeen) > rootOfflineThreshold {
            log.Printf("[FailoverManager] Root node offline threshold exceeded (15 minutes). Initiating failover check.")
            fm.failoverInProgress = true // Prevent multiple failover attempts
            fm.mu.Unlock()               // Unlock before calling potentially long-running function
            go fm.initiateFailoverProcedure() // Run in a new goroutine
            return
        }
    } else {
        if !fm.isRootOnline {
            log.Printf("[FailoverManager] Root node at %s is back online.", fm.currentRootAPIURL)
        }
        fm.isRootOnline = true
        fm.lastRootSeen = time.Now()
    }
    if resp != nil {
        resp.Body.Close()
    }
    fm.mu.Unlock()
}

func (fm *FailoverManager) amIElectedToBecomeRoot() bool {
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
    if !fm.amIElectedToBecomeRoot() {
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

    log.Println("[FailoverManager] This node IS ELECTED. Proceeding with promotion to Root role.")
    // Signal the main application to shut down its current role and then promote.
    // The actual promotion and restart will be handled in main.go after shutdown.
    fm.mainContextCancelFn() // This will trigger the shutdown sequence in main.go
    // The main.go shutdown sequence should then call a function like promoteToRootAndRestart
}

func (fm *FailoverManager) broadcastNetworkPause() {
    // Join the network control topic
    controlTopic, err := fm.pubsub.Join(NetworkControlTopic)
    if err != nil {
        log.Printf("[FailoverManager] Error joining network control topic: %v", err)
        return
    }

    // Create and broadcast the pause message
    pausePayload := NetworkPausePayload{
        InitiatorPeerID: fm.nodeConfig.PeerID, // Assuming PeerID is stored in config
        Reason:          "Root node failover in progress",
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
```

## Phase 2: Network Control Protocol

### 2.1 Define Network Control Messages

**File: `p2p_consensus_manager.go`**

Add network control message types and handling:

```go
// Network control message types
type NetworkControlMessage struct {
    Type    string      `json:"type"`
    Payload interface{} `json:"payload"`
    // TODO: Add signature field for authentication
}

type NetworkPausePayload struct {
    InitiatorPeerID string `json:"initiator_dev_id"`
    Reason          string `json:"reason"`
    Timestamp       int64  `json:"timestamp"`
}

type NetworkResumePayload struct {
    NewRootPeerID     string   `json:"new_root_dev_id"`
    NewRootMultiaddrs []string `json:"new_root_multiaddrs"`
    NewRootChainID    string   `json:"new_root_chain_id"`
    Timestamp         int64    `json:"timestamp"`
}

// Add to P2PConsensusManager struct
type P2PConsensusManager struct {
    // ... existing fields ...
    networkPaused    bool
    networkControlSub *pubsub.Subscription
    // ... other fields ...
}
```

### 2.2 Implement Network Control Event Handling

**File: `p2p_consensus_manager.go`**

```go
// handleNetworkControlEvents processes incoming network control messages
func (pcm *P2PConsensusManager) handleNetworkControlEvents() {
    for {
        msg, err := pcm.networkControlSub.Next(pcm.ctx)
        if err != nil {
            if pcm.ctx.Err() != nil {
                return // Context canceled
            }
            log.Printf("[%s][%s] Error receiving network control message: %v", pcm.nodeRole.String(), pcm.blockchain.ChainID, err)
            continue
        }

        // Skip messages from ourselves
        if msg.ReceivedFrom == pcm.host.ID() {
            continue
        }

        var controlMsg NetworkControlMessage
        if err := json.Unmarshal(msg.Data, &controlMsg); err != nil {
            log.Printf("[%s][%s] Error decoding network control message: %v", pcm.nodeRole.String(), pcm.blockchain.ChainID, err)
            continue
        }

        // TODO: Add signature verification for controlMsg to ensure authenticity

        switch controlMsg.Type {
        case "NetworkPause":
            var payload NetworkPausePayload
            if err := json.Unmarshal(controlMsg.Payload.([]byte), &payload); err != nil {
                log.Printf("[%s][%s] Error unmarshalling NetworkPausePayload: %v", pcm.nodeRole.String(), pcm.blockchain.ChainID, err)
                continue
            }
            log.Printf("[%s][%s] Received NetworkPause signal. Reason: %s", pcm.nodeRole.String(), pcm.blockchain.ChainID, payload.Reason)
            pcm.mu.Lock()
            pcm.networkPaused = true
            pcm.blockchain.StopMining = true // Signal miner to stop
            pcm.mu.Unlock()

        case "NetworkResume":
            var payload NetworkResumePayload
            if err := json.Unmarshal(controlMsg.Payload.([]byte), &payload); err != nil {
                log.Printf("[%s][%s] Error unmarshalling NetworkResumePayload: %v", pcm.nodeRole.String(), pcm.blockchain.ChainID, err)
                continue
            }
            log.Printf("[%s][%s] Received NetworkResume signal from new root %s (ChainID: %s)", pcm.nodeRole.String(), pcm.blockchain.ChainID, payload.NewRootPeerID, payload.NewRootChainID)
            pcm.mu.Lock()
            pcm.networkPaused = false
            pcm.blockchain.StopMining = false // Allow mining to resume if applicable
            pcm.mu.Unlock()

            // Clear transaction pool as it might contain stale transactions
            // that were not included in the new root's chain history.
            pcm.blockchain.Lock()
            log.Printf("[%s][%s] Clearing transaction pool (%d transactions) due to network resume.", pcm.nodeRole.String(), pcm.blockchain.ChainID, len(pcm.blockchain.TransactionPool))
            pcm.blockchain.TransactionPool = []*Transaction{}
            pcm.blockchain.Unlock()

            // TODO: Update discovery/connection logic to use new root info (payload.NewRootMultiaddrs)
            log.Printf("[%s][%s] Network resumed. New root multiaddrs: %v", pcm.nodeRole.String(), pcm.blockchain.ChainID, payload.NewRootMultiaddrs)

        default:
            log.Printf("[%s][%s] Received unknown network control message type: %s", pcm.nodeRole.String(), pcm.blockchain.ChainID, controlMsg.Type)
        }
    }
}
```

### 2.3 Update Transaction and Block Processing

**File: `p2p_consensus_manager.go`**

Modify transaction and block processing to respect network pause state:

```go
// processReceivedTransaction validates and adds a transaction received from the network
func (pcm *P2PConsensusManager) processReceivedTransaction(transaction *Transaction) {
    pcm.mu.Lock()
    if pcm.networkPaused {
        pcm.mu.Unlock()
        log.Printf("[%s][%s] Network is paused, dropping transaction %s", pcm.nodeRole.String(), pcm.blockchain.ChainID, transaction.TransactionHash)
        return
    }
    pcm.mu.Unlock()

    // Existing transaction processing logic...
}

// BroadcastBlock publishes a block to the network using pubsub
func (pcm *P2PConsensusManager) BroadcastBlock(block *Block) error {
    pcm.mu.Lock()
    if pcm.networkPaused {
        pcm.mu.Unlock()
        log.Printf("[%s][%s] Network is paused, not broadcasting block #%d", pcm.nodeRole.String(), pcm.blockchain.ChainID, block.BlockNumber)
        return fmt.Errorf("network is paused")
    }
    pcm.mu.Unlock()
    
    // Existing block broadcasting logic...
}

// BroadcastTransaction publishes a transaction to the network using pubsub
func (pcm *P2PConsensusManager) BroadcastTransaction(transaction *Transaction) error {
    pcm.mu.Lock()
    if pcm.networkPaused {
        pcm.mu.Unlock()
        log.Printf("[%s][%s] Network is paused, not broadcasting transaction %s", pcm.nodeRole.String(), pcm.blockchain.ChainID, transaction.TransactionHash)
        return fmt.Errorf("network is paused")
    }
    pcm.mu.Unlock()
    
    // Existing transaction broadcasting logic...
}
```

## Phase 3: HTTP Server Integration

### 3.1 Update BlockchainServer

**File: `blockchain_server.go`**

Modify the BlockchainServer to check network pause state:

```go
type BlockchainServer struct {
    port             uint64
    blockchainPtr    *BlockchainStruct
    chainAddress     string
    server           *http.Server
    db               *LevelDB
    p2pConsensusMgr  *P2PConsensusManager // Added to check network pause state
    discoveryManager DiscoveryService
    p2pPort          int
}

// Modify NewBlockchainServer to accept P2PConsensusManager
func NewBlockchainServer(port uint64, blockchain *BlockchainStruct, db *LevelDB, p2pMgr *P2PConsensusManager, discoveryMgr DiscoveryService, p2pPort int) *BlockchainServer {
    // Set the server port globally so it can be accessed from other parts of the code
    SetServerPort(port)

    return &BlockchainServer{
        port:             port,
        blockchainPtr:    blockchain,
        db:               db,
        p2pConsensusMgr:  p2pMgr,
        discoveryManager: discoveryMgr,
        p2pPort:          p2pPort,
    }
}
```

### 3.2 Update HTTP Endpoints

**File: `blockchain_server.go`**

Add network pause checks to relevant endpoints:

```go
// handleAddBlock handles POST requests to add a new block
func (bcs *BlockchainServer) handleAddBlock(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    if bcs.p2pConsensusMgr != nil && bcs.p2pConsensusMgr.networkPaused {
        log.Printf("[%s] Network is paused, rejecting new block.", bcs.blockchainPtr.ChainID)
        http.Error(w, "Network is temporarily paused, please try again later.", http.StatusServiceUnavailable)
        return
    }

    // Existing block handling logic...
}

// handleAddTransaction handles POST requests to add a new transaction
func (bcs *BlockchainServer) handleAddTransaction(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    if bcs.p2pConsensusMgr != nil && bcs.p2pConsensusMgr.networkPaused {
        log.Printf("[%s] Network is paused, rejecting new transaction.", bcs.blockchainPtr.ChainID)
        http.Error(w, "Network is temporarily paused, please try again later.", http.StatusServiceUnavailable)
        return
    }

    // Existing transaction handling logic...
}

// Add similar checks to other state-modifying endpoints
```

## Phase 4: Promotion Mechanism

### 4.1 Implement Node Promotion

**File: `main.go`**

Add the function to promote a bootnode to a root node:

```go
func promoteToRootAndRestart(currentConfigPath string, currentBootnodeCfg *config.Config, wm *WalletManager) error {
    log.Println("PROMOTING to Root (Root) Node and restarting...")

    // 1. Get the bootnode's master wallet address for MinersAddress
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

    // 2. Get the bootnode's database path
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

    // 3. Create the new root configuration based on DefaultRootConfig
    newRootCfg := config.DefaultRootConfig()

    // 4. Override specific fields for the new root
    newRootCfg.IsRoot = true
    newRootCfg.IsBootnode = false
    newRootCfg.ClientOnly = false
    newRootCfg.InstallComplete = true

    // ChainID: Use DefaultRootConfig's port-based ChainID
    if newRootCfg.ChainID == "" {
        newRootCfg.ChainID = fmt.Sprintf("KNIRVCHAIN-ROOT%d", newRootCfg.Port)
    }

    // MinersAddress: Use the bootnode's MinersAddress
    newRootCfg.MinersAddress = bootnodeMinersAddress
    log.Printf("New root node MinersAddress will be: %s (from bootnode's MinersAddress)", newRootCfg.MinersAddress)
    log.Printf("New root node MasterAddress (faucet/genesis) will be: %s", newRootCfg.MasterAddress)

    // BlockchainDatabasePath: Use the bootnode's existing database path
    newRootCfg.BlockchainDatabasePath = bootnodeDbPath
    log.Printf("New root node BlockchainDatabasePath will be: %s (from bootnode's database)", newRootCfg.BlockchainDatabasePath)

    // 5. Save the new configuration
    if err := config.SaveConfig(currentConfigPath, newRootCfg, config.Root); err != nil {
        return fmt.Errorf("failed to save new root config to %s: %w", currentConfigPath, err)
    }
    log.Printf("New root configuration saved to %s. Relaunching...", currentConfigPath)

    // 6. Relaunch the application
    cmd := exec.Command(os.Args[0], os.Args[1:]...)
    cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
    if err := cmd.Start(); err != nil {
        return fmt.Errorf("failed to relaunch application as root: %w", err)
    }
    os.Exit(0) // Exit old process
    return nil // Should not be reached
}
```

### 4.2 Integrate with Main Application Flow

**File: `main.go`**

Update the main application flow to handle failover:

```go
// Global FailoverManager instance
var fmGlobalForFailover *FailoverManager

// In main() function
func main() {
    // ... existing initialization ...

    // Initialize FailoverManager if the node is a bootnode
    if nodeRole == config.RoleBootnode && cfg.CurrentRootNodeAPIURL != "" {
        // Pass a cancel function that will be specific to this node's context
        var psForFailover *pubsub.PubSub
        if p2pConsensusMgr != nil {
            psForFailover = p2pConsensusMgr.pubsub
        }
        fmGlobalForFailover = NewFailoverManager(cfg.CurrentRootNodeAPIURL, &cfg, loadedConfigPath, wm, wallet, psForFailover, ctx.Done)
        if fmGlobalForFailover != nil {
            fmGlobalForFailover.StartMonitoring()
            defer fmGlobalForFailover.StopMonitoring()
        }
    }

    // ... existing code ...

    // If this node is a Root (potentially a new root after failover),
    // broadcast a NetworkResume message
    if nodeRole == config.Root && p2pConsensusMgr != nil && p2pConsensusMgr.pubsub != nil {
        go func() {
            // Allow some time for the node to fully initialize before broadcasting
            time.Sleep(5 * time.Second)

            controlTopic, err := p2pConsensusMgr.pubsub.Join(NetworkControlTopic)
            if err != nil {
                log.Printf("[%s][%s] Root: Error joining network control topic for resume: %v", nodeRole.String(), cfg.ChainID, err)
                return
            }

            resumePayload := NetworkResumePayload{
                NewRootPeerID:     discoveryMgr.GetPeerID(),
                NewRootMultiaddrs: discoveryMgr.GetSelfMultiaddrs(),
                NewRootChainID:    cfg.ChainID,
                Timestamp:         time.Now().Unix(),
            }
            payloadBytes, _ := json.Marshal(resumePayload)
            controlMsg := NetworkControlMessage{Type: "NetworkResume", Payload: payloadBytes}

            msgBytes, _ := json.Marshal(controlMsg)
            log.Printf("[%s][%s] Root: Broadcasting NetworkResume message...", nodeRole.String(), cfg.ChainID)
            if err := controlTopic.Publish(context.Background(), msgBytes); err != nil {
                log.Printf("[%s][%s] Root: Error publishing NetworkResume message: %v", nodeRole.String(), cfg.ChainID, err)
            }
        }()
    }

    // ... rest of main function ...
}

// Update waitForShutdownSignal to handle promotion after shutdown
func waitForShutdownSignal(cancel context.CancelFunc, wg *sync.WaitGroup, isFailoverCandidate bool, configPath string, currentCfg *config.Config, walletMgr *WalletManager) {
    signalChan := make(chan os.Signal, 1)
    signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
    
    sig := <-signalChan
    log.Printf("Received signal: %s. Initiating shutdown...", sig)
    
    cancel() // Signal all components to shut down
    
    // Wait for all components to shut down with a timeout
    shutdownTimeout := 30 * time.Second
    shutdownComplete := make(chan struct{})
    
    go func() {
        wg.Wait()
        close(shutdownComplete)
    }()
    
    select {
    case <-shutdownComplete:
        log.Println("All components shut down successfully.")
    case <-time.After(shutdownTimeout):
        log.Println("WARN: Timeout waiting for all nodes to shut down.")
    }
    
    // After all nodes are shut down, check if this was a failover candidate that needs to promote
    if isFailoverCandidate && fmGlobalForFailover != nil && fmGlobalForFailover.failoverInProgress && fmGlobalForFailover.amIElectedToBecomeRoot() {
        log.Println("Shutdown complete. Now promoting to Root role...")
        promoteToRootAndRestart(configPath, currentCfg, walletMgr)
        // promoteToRootAndRestart will os.Exit, so code below won't run for the promoted node
    }
    
    log.Println("Exiting.")
}
```

## Phase 5: Testing and Validation

### 5.1 Unit Tests

Develop unit tests for each component:

1. FailoverManager
   - Test root node status monitoring
   - Test election mechanism
   - Test promotion logic

2. Network Control Protocol
   - Test pause/resume message handling
   - Test transaction pool clearing

3. HTTP Server Integration
   - Test endpoint behavior during network pause

### 5.2 Integration Tests

Develop integration tests to validate the complete failover flow:

1. Root Node Failure Simulation
   - Simulate root node going offline
   - Verify bootnode detection and election
   - Verify promotion and restart

2. Network Behavior During Failover
   - Verify network pause is respected
   - Verify transaction pool handling
   - Verify chain continuity

3. Client Experience
   - Verify client reconnection to new root
   - Verify transaction processing after failover

### 5.3 Performance Testing

Evaluate the performance impact of the failover mechanism:

1. Monitoring Overhead
   - Measure CPU/memory impact of continuous monitoring
   - Optimize ping frequency if needed

2. Failover Time
   - Measure total time from root failure to network resume
   - Identify and optimize bottlenecks

## Implementation Considerations

1. **Chain Continuity**: The new root node uses its existing database to maintain chain continuity. This relies on bootnodes consistently syncing the full blockchain state.

2. **Transaction Pool Clearing**: When nodes resume and connect to a new root, they clear their local transaction pools to prevent replaying transactions that might be invalid or already included in the new root's chain history.

3. **Authentication of Control Messages**: In a production system, network control messages should be authenticated (e.g., signed by the bootnode's wallet) to prevent malicious actors from disrupting the network.

4. **Leader Election Security**: The stake-based leader election mechanism must be secure against Sybil attacks and ensure only eligible bootnodes can participate.

5. **Network Re-Discovery**: Clients and devs need a reliable mechanism to discover and connect to the new root node after failover.

## Future Enhancements

1. **Advanced Leader Election**: Implement a more sophisticated consensus algorithm for leader election based on stake, uptime, and other metrics.

2. **Quorum-Based Failover**: Require a quorum of bootnodes to agree on root node failure before initiating failover.

3. **Gradual Transition**: Implement a more gradual transition process to minimize disruption to the network.

4. **Automatic Root Node Recovery**: Add mechanisms for a recovered root node to gracefully rejoin the network as a regular node.

5. **Client Notification**: Implement a mechanism to notify clients about the failover process and guide them to the new root node.

---

<div class="footer-links">


© 2025 KNIRV Network
</div>
