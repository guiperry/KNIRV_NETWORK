

---

**Source**: KNIRVROOT/docs/pending_implementation_plans/Failover_2_PWS_Implementation_Plan.md

# KNIRVCHAIN Root Node Failover Implementation Plan

## Overview

This document outlines a comprehensive implementation plan for a resilient failover mechanism in the KNIRVCHAIN network. The mechanism enables automatic recovery when the root node becomes unavailable, with the bootnode having the highest stake taking over as the new root node.

## Core Concepts

1. **Root Node Liveness Monitoring**: Bootnodes continuously monitor the health of the current root node
2. **Failure Detection**: If the root node is unresponsive for 15 minutes, a failover process is triggered
3. **Leader Election**: Bootnodes communicate to determine which has the highest stake and is eligible to become the new root
4. **Network Pause**: The network temporarily pauses operations during the transition
5. **Secure Password Sharing**: The root node's root key password is securely shared across bootnodes in a distributed manner
6. **Password Reconstruction**: During failover, the elected bootnode reconstructs the password from shared fragments
7. **Promotion and Restart**: The elected bootnode shuts down its bootnode operations, updates its configuration to Root, and restarts as the new root node using the reconstructed password
8. **Network Resume**: The new root node broadcasts a resume message to the network
9. **Chain Continuity**: The new root node uses its existing database to maintain chain continuity

## Phase 1: Secure Password Sharing System

### 1.1 Password Sharding Implementation

**File: `password_shard_manager.go`**

```go
package main

import (
    "crypto/aes"
    "crypto/cipher"
    "crypto/rand"
    "crypto/sha256"
    "encoding/base64"
    "encoding/json"
    "errors"
    "fmt"
    "io"
    "log"
    "sync"
    "time"

    "github.com/hashicorp/vault/shamir"
)

// PasswordShardManager handles the secure distribution and reconstruction
// of the agent_CREATOR_KEY_PASSWORD across bootnodes
type PasswordShardManager struct {
    mu                sync.RWMutex
    ownShard          []byte            // This bootnode's shard
    shardID           string            // Unique ID for this shard
    shardEncryptionKey []byte           // Key used to encrypt/decrypt this shard
    allShardIDs       map[string]string // Map of shardID -> bootnode nodeID
    threshold         int               // Minimum shards needed for reconstruction
    totalShards       int               // Total number of shards created
    lastUpdated       time.Time         // When the shard was last updated
}

// NewPasswordShardManager creates a new password shard manager
func NewPasswordShardManager(nodeID string, encryptionSeed string) *PasswordShardManager {
    // Generate a deterministic encryption key from the node's ID and seed
    keyMaterial := nodeID + encryptionSeed
    key := sha256.Sum256([]byte(keyMaterial))
    
    return &PasswordShardManager{
        shardEncryptionKey: key[:],
        allShardIDs:        make(map[string]string),
        threshold:          3, // Default: require 3 shards to reconstruct
        totalShards:        5, // Default: split into 5 shards
    }
}

// CreateAndDistributeShards splits the root password into shards and distributes them
func (psm *PasswordShardManager) CreateAndDistributeShards(rootPassword string, bootnodePeers []string) error {
    psm.mu.Lock()
    defer psm.mu.Unlock()
    
    // Validate inputs
    if len(bootnodePeers) < psm.threshold {
        return fmt.Errorf("not enough bootnode devs (%d) to meet threshold (%d)", 
            len(bootnodePeers), psm.threshold)
    }
    
    // Adjust totalShards if needed
    if len(bootnodePeers) < psm.totalShards {
        psm.totalShards = len(bootnodePeers)
    }
    
    // Create shards using Shamir's Secret Sharing
    passwordBytes := []byte(rootPassword)
    shards, err := shamir.Split(passwordBytes, psm.totalShards, psm.threshold)
    if err != nil {
        return fmt.Errorf("failed to create password shards: %w", err)
    }
    
    // Distribute shards to bootnodes (implementation depends on your P2P system)
    for i, dev := range bootnodePeers[:psm.totalShards] {
        shardID := fmt.Sprintf("shard-%d-%s", i, time.Now().Format("20060102-150405"))
        
        // Encrypt the shard before sending
        encryptedShard, err := psm.encryptShard(shards[i])
        if err != nil {
            return fmt.Errorf("failed to encrypt shard %s: %w", shardID, err)
        }
        
        // Store the mapping of shardID to nodeID
        psm.allShardIDs[shardID] = dev
        
        // Send the encrypted shard to the bootnode
        // This would use your P2P messaging system
        err = sendShardToPeer(dev, shardID, encryptedShard)
        if err != nil {
            return fmt.Errorf("failed to send shard %s to dev %s: %w", shardID, dev, err)
        }
        
        log.Printf("Sent password shard %s to bootnode %s", shardID, dev)
    }
    
    psm.lastUpdated = time.Now()
    return nil
}

// ReceiveShard stores a received shard
func (psm *PasswordShardManager) ReceiveShard(shardID string, encryptedShard []byte, senderPeerID string) error {
    psm.mu.Lock()
    defer psm.mu.Unlock()
    
    // Decrypt the shard
    decryptedShard, err := psm.decryptShard(encryptedShard)
    if err != nil {
        return fmt.Errorf("failed to decrypt received shard: %w", err)
    }
    
    // Store the shard
    psm.ownShard = decryptedShard
    psm.shardID = shardID
    psm.allShardIDs[shardID] = senderPeerID
    psm.lastUpdated = time.Now()
    
    log.Printf("Received and stored password shard %s from dev %s", shardID, senderPeerID)
    return nil
}

// RequestShards requests shards from other bootnodes during failover
func (psm *PasswordShardManager) RequestShards(bootnodePeers []string) error {
    psm.mu.RLock()
    ownShardID := psm.shardID
    psm.mu.RUnlock()
    
    // Track received shards
    receivedShards := make(map[string][]byte)
    if psm.ownShard != nil {
        receivedShards[ownShardID] = psm.ownShard
    }
    
    // Request shards from other bootnodes
    // This would use your P2P messaging system
    for _, dev := range bootnodePeers {
        shards, err := requestShardsFromPeer(dev)
        if err != nil {
            log.Printf("Failed to request shards from dev %s: %v", dev, err)
            continue
        }
        
        for shardID, encryptedShard := range shards {
            decryptedShard, err := psm.decryptShard(encryptedShard)
            if err != nil {
                log.Printf("Failed to decrypt shard %s from dev %s: %v", shardID, dev, err)
                continue
            }
            
            receivedShards[shardID] = decryptedShard
        }
        
        // If we have enough shards, we can stop requesting
        if len(receivedShards) >= psm.threshold {
            break
        }
    }
    
    // Check if we have enough shards
    if len(receivedShards) < psm.threshold {
        return fmt.Errorf("failed to collect enough shards: got %d, need %d", 
            len(receivedShards), psm.threshold)
    }
    
    return nil
}

// ReconstructPassword combines shards to reconstruct the original password
func (psm *PasswordShardManager) ReconstructPassword(collectedShards map[string][]byte) (string, error) {
    if len(collectedShards) < psm.threshold {
        return "", fmt.Errorf("not enough shards to reconstruct password: got %d, need %d", 
            len(collectedShards), psm.threshold)
    }
    
    // Extract just the shard values for reconstruction
    shardValues := make([][]byte, 0, len(collectedShards))
    for _, shard := range collectedShards {
        shardValues = append(shardValues, shard)
    }
    
    // Combine shards using Shamir's Secret Sharing
    passwordBytes, err := shamir.Combine(shardValues)
    if err != nil {
        return "", fmt.Errorf("failed to reconstruct password from shards: %w", err)
    }
    
    return string(passwordBytes), nil
}

// encryptShard encrypts a shard before sending it
func (psm *PasswordShardManager) encryptShard(shard []byte) ([]byte, error) {
    block, err := aes.NewCipher(psm.shardEncryptionKey)
    if err != nil {
        return nil, err
    }
    
    // Create a new GCM cipher
    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return nil, err
    }
    
    // Create a nonce
    nonce := make([]byte, gcm.NonceSize())
    if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
        return nil, err
    }
    
    // Encrypt the shard
    ciphertext := gcm.Seal(nonce, nonce, shard, nil)
    return ciphertext, nil
}

// decryptShard decrypts a received shard
func (psm *PasswordShardManager) decryptShard(encryptedShard []byte) ([]byte, error) {
    block, err := aes.NewCipher(psm.shardEncryptionKey)
    if err != nil {
        return nil, err
    }
    
    // Create a new GCM cipher
    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return nil, err
    }
    
    // Extract the nonce
    if len(encryptedShard) < gcm.NonceSize() {
        return nil, errors.New("encrypted shard too short")
    }
    nonce, ciphertext := encryptedShard[:gcm.NonceSize()], encryptedShard[gcm.NonceSize():]
    
    // Decrypt the shard
    return gcm.Open(nil, nonce, ciphertext, nil)
}

// Helper functions for P2P communication
// These would be implemented based on your P2P system

func sendShardToPeer(nodeID string, shardID string, encryptedShard []byte) error {
    // Implementation depends on your P2P messaging system
    // This is a placeholder
    return nil
}

func requestShardsFromPeer(nodeID string) (map[string][]byte, error) {
    // Implementation depends on your P2P messaging system
    // This is a placeholder
    return nil, nil
}
```

### 1.2 P2P Message Types for Password Sharing

**File: `p2p_consensus_manager.go`**

Add message types for password shard distribution:

```go
// Add to existing message types
const (
    // ... existing message types ...
    MessageTypePasswordShardDistribute = "password_shard_distribute"
    MessageTypePasswordShardRequest    = "password_shard_request"
    MessageTypePasswordShardResponse   = "password_shard_response"
)

// Password shard message structures
type PasswordShardDistributeMessage struct {
    ShardID        string `json:"shard_id"`
    EncryptedShard []byte `json:"encrypted_shard"`
    SenderPeerID   string `json:"sender_dev_id"`
    Timestamp      int64  `json:"timestamp"`
    Signature      []byte `json:"signature"` // Signed by the root node
}

type PasswordShardRequestMessage struct {
    RequesterPeerID string `json:"requester_dev_id"`
    Timestamp       int64  `json:"timestamp"`
    Signature       []byte `json:"signature"` // Signed by the requester
}

type PasswordShardResponseMessage struct {
    ShardID        string `json:"shard_id"`
    EncryptedShard []byte `json:"encrypted_shard"`
    ResponderPeerID string `json:"responder_dev_id"`
    Timestamp      int64  `json:"timestamp"`
    Signature      []byte `json:"signature"` // Signed by the responder
}
```

### 1.3 Integration with P2P Consensus Manager

**File: `p2p_consensus_manager.go`**

Add password shard handling to the P2P consensus manager:

```go
// Add to P2PConsensusManager struct
type P2PConsensusManager struct {
    // ... existing fields ...
    passwordShardMgr *PasswordShardManager
    // ... other fields ...
}

// Initialize PasswordShardManager in NewP2PConsensusManager
func NewP2PConsensusManager(blockchain *BlockchainStruct, db *LevelDB, discoveryMgr *DiscoveryManager, nodeRole config.Role) (*P2PConsensusManager, error) {
    // ... existing initialization ...
    
    // Initialize password shard manager if this is a bootnode or root
    if nodeRole == config.RoleBootnode || nodeRole == config.Root {
        // Use a combination of node-specific and network-wide values for the encryption seed
        encryptionSeed := fmt.Sprintf("%s-%s", blockchain.ChainID, discoveryMgr.GetPeerID())
        pcm.passwordShardMgr = NewPasswordShardManager(discoveryMgr.GetPeerID(), encryptionSeed)
    }
    
    // ... rest of initialization ...
    return pcm, nil
}

// Add handlers for password shard messages
func (pcm *P2PConsensusManager) handlePasswordShardDistribute(msg *pubsub.Message) {
    // Skip messages from ourselves
    if msg.ReceivedFrom == pcm.host.ID() {
        return
    }
    
    var shardMsg PasswordShardDistributeMessage
    if err := json.Unmarshal(msg.Data, &shardMsg); err != nil {
        log.Printf("[%s][%s] Error decoding password shard distribute message: %v", 
            pcm.nodeRole.String(), pcm.blockchain.ChainID, err)
        return
    }
    
    // Verify the signature (implementation depends on your crypto system)
    if !verifySignature(shardMsg.SenderPeerID, msg.Data, shardMsg.Signature) {
        log.Printf("[%s][%s] Invalid signature on password shard distribute message", 
            pcm.nodeRole.String(), pcm.blockchain.ChainID)
        return
    }
    
    // Process the shard
    if pcm.passwordShardMgr != nil {
        err := pcm.passwordShardMgr.ReceiveShard(
            shardMsg.ShardID, 
            shardMsg.EncryptedShard, 
            shardMsg.SenderPeerID,
        )
        if err != nil {
            log.Printf("[%s][%s] Error processing password shard: %v", 
                pcm.nodeRole.String(), pcm.blockchain.ChainID, err)
        }
    }
}

func (pcm *P2PConsensusManager) handlePasswordShardRequest(msg *pubsub.Message) {
    // Skip messages from ourselves
    if msg.ReceivedFrom == pcm.host.ID() {
        return
    }
    
    var requestMsg PasswordShardRequestMessage
    if err := json.Unmarshal(msg.Data, &requestMsg); err != nil {
        log.Printf("[%s][%s] Error decoding password shard request message: %v", 
            pcm.nodeRole.String(), pcm.blockchain.ChainID, err)
        return
    }
    
    // Verify the signature
    if !verifySignature(requestMsg.RequesterPeerID, msg.Data, requestMsg.Signature) {
        log.Printf("[%s][%s] Invalid signature on password shard request message", 
            pcm.nodeRole.String(), pcm.blockchain.ChainID)
        return
    }
    
    // Only respond if we have a shard and we're a bootnode
    if pcm.passwordShardMgr != nil && pcm.passwordShardMgr.ownShard != nil && pcm.nodeRole == config.RoleBootnode {
        // Create response message
        responseMsg := PasswordShardResponseMessage{
            ShardID:         pcm.passwordShardMgr.shardID,
            EncryptedShard:  pcm.passwordShardMgr.ownShard,
            ResponderPeerID: pcm.host.ID().String(),
            Timestamp:       time.Now().Unix(),
        }
        
        // Sign the response
        responseBytes, _ := json.Marshal(responseMsg)
        responseMsg.Signature = signMessage(responseBytes) // Implementation depends on your crypto system
        
        // Send the response
        responseBytes, _ = json.Marshal(responseMsg)
        topic := fmt.Sprintf("%s/password_shard_response/%s", pcm.blockchain.ChainID, requestMsg.RequesterPeerID)
        if err := pcm.pubsub.Publish(topic, responseBytes); err != nil {
            log.Printf("[%s][%s] Error publishing password shard response: %v", 
                pcm.nodeRole.String(), pcm.blockchain.ChainID, err)
        }
    }
}

// Add to the message handling switch in the main message handler
func (pcm *P2PConsensusManager) handleMessage(msg *pubsub.Message) {
    // ... existing message type handling ...
    
    switch msgType {
    // ... existing cases ...
    case MessageTypePasswordShardDistribute:
        pcm.handlePasswordShardDistribute(msg)
    case MessageTypePasswordShardRequest:
        pcm.handlePasswordShardRequest(msg)
    // ... other cases ...
    }
}
```

## Phase 2: Root Node Password Distribution

### 2.1 Update Root Node Initialization

**File: `main.go`**

Modify the root node initialization to distribute password shards:

```go
// Add to main.go or wherever the root node is initialized
func initializeRootNode(cfg *config.Config, p2pConsensusMgr *P2PConsensusManager, discoveryMgr *DiscoveryManager) error {
    // ... existing initialization ...
    
    // Get the root key password from environment or config
    rootKeyPassword := os.Getenv("agent_CREATOR_KEY_PASSWORD")
    if rootKeyPassword == "" {
        // Fallback to a config value or prompt the user
        rootKeyPassword = cfg.RootKeyPassword
        if rootKeyPassword == "" {
            return errors.New("agent_CREATOR_KEY_PASSWORD not set")
        }
    }
    
    // Distribute password shards to bootnodes
    if p2pConsensusMgr != nil && p2pConsensusMgr.passwordShardMgr != nil {
        // Get list of bootnode devs from discovery manager
        bootnodePeers, err := discoveryMgr.GetBootnodePeers()
        if err != nil {
            return fmt.Errorf("failed to get bootnode devs: %w", err)
        }
        
        if len(bootnodePeers) == 0 {
            log.Println("Warning: No bootnodes found for password shard distribution")
            return nil
        }
        
        // Create and distribute shards
        err = p2pConsensusMgr.passwordShardMgr.CreateAndDistributeShards(rootKeyPassword, bootnodePeers)
        if err != nil {
            return fmt.Errorf("failed to distribute password shards: %w", err)
        }
        
        log.Printf("Successfully distributed root key password shards to %d bootnodes", len(bootnodePeers))
    }
    
    return nil
}
```

## Phase 3: Failover Manager Enhancement

### 3.1 Update Failover Manager for Password Reconstruction

**File: `failover_manager.go`**

Enhance the failover manager to collect and reconstruct the password during failover:

```go
// Add to FailoverManager struct
type FailoverManager struct {
    // ... existing fields ...
    passwordShardMgr    *PasswordShardManager
    reconstructedPassword string
    // ... other fields ...
}

// Update NewFailoverManager to accept PasswordShardManager
func NewFailoverManager(rootAPIURL string, cfg *config.Config, configPath string, wm *WalletManager, wallet *Wallet, ps *pubsub.PubSub, passwordShardMgr *PasswordShardManager, mainCancelFn context.CancelFunc) *FailoverManager {
    // ... existing initialization ...
    
    return &FailoverManager{
        // ... existing fields ...
        passwordShardMgr: passwordShardMgr,
        // ... other fields ...
    }
}

// Update initiateFailoverProcedure to collect password shards
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
    
    // Collect password shards from other bootnodes
    if fm.passwordShardMgr != nil {
        log.Println("[FailoverManager] Collecting password shards from other bootnodes...")
        
        // Get list of bootnode devs
        bootnodePeers, err := fm.discoveryManager.GetBootnodePeers()
        if err != nil {
            log.Printf("[FailoverManager] Failed to get bootnode devs: %v", err)
            // Continue with failover even if we can't get the password
        } else {
            // Request shards from other bootnodes
            err = fm.passwordShardMgr.RequestShards(bootnodePeers)
            if err != nil {
                log.Printf("[FailoverManager] Failed to collect password shards: %v", err)
                // Continue with failover even if we can't get the password
            } else {
                // Get all collected shards
                collectedShards := make(map[string][]byte)
                // ... populate collectedShards from passwordShardMgr ...
                
                // Reconstruct the password
                password, err := fm.passwordShardMgr.ReconstructPassword(collectedShards)
                if err != nil {
                    log.Printf("[FailoverManager] Failed to reconstruct password: %v", err)
                    // Continue with failover even if we can't reconstruct the password
                } else {
                    fm.reconstructedPassword = password
                    log.Println("[FailoverManager] Successfully reconstructed root key password")
                }
            }
        }
    }

    log.Println("[FailoverManager] This node IS ELECTED. Proceeding with promotion to Root role.")
    // Signal the main application to shut down its current role and then promote.
    // The actual promotion and restart will be handled in main.go after shutdown.
    fm.mainContextCancelFn() // This will trigger the shutdown sequence in main.go
    // The main.go shutdown sequence should then call a function like promoteToRootAndRestart
}
```

### 3.2 Update Promotion Function to Use Reconstructed Password

**File: `main.go`**

Modify the promotion function to use the reconstructed password:

```go
func promoteToRootAndRestart(currentConfigPath string, currentBootnodeCfg *config.Config, wm *WalletManager, reconstructedPassword string) error {
    log.Println("PROMOTING to Root (Root) Node and restarting...")

    // ... existing code to prepare new root configuration ...

    // Set the reconstructed password in the environment for the new process
    if reconstructedPassword != "" {
        log.Println("Using reconstructed root key password for new root node")
        os.Setenv("agent_CREATOR_KEY_PASSWORD", reconstructedPassword)
    } else {
        log.Println("Warning: No reconstructed password available. The new root node may require manual password entry.")
    }

    // ... existing code to save config and restart ...

    // 6. Relaunch the application
    cmd := exec.Command(os.Args[0], os.Args[1:]...)
    cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
    
    // Pass environment variables including the reconstructed password
    cmd.Env = os.Environ()
    
    if err := cmd.Start(); err != nil {
        return fmt.Errorf("failed to relaunch application as root: %w", err)
    }
    os.Exit(0) // Exit old process
    return nil // Should not be reached
}

// Update waitForShutdownSignal to pass the reconstructed password
func waitForShutdownSignal(cancel context.CancelFunc, wg *sync.WaitGroup, isFailoverCandidate bool, configPath string, currentCfg *config.Config, walletMgr *WalletManager) {
    // ... existing code ...
    
    // After all nodes are shut down, check if this was a failover candidate that needs to promote
    if isFailoverCandidate && fmGlobalForFailover != nil && fmGlobalForFailover.failoverInProgress && fmGlobalForFailover.amIElectedToBecomeRoot() {
        log.Println("Shutdown complete. Now promoting to Root role...")
        
        // Get the reconstructed password if available
        reconstructedPassword := ""
        if fmGlobalForFailover.reconstructedPassword != "" {
            reconstructedPassword = fmGlobalForFailover.reconstructedPassword
        }
        
        promoteToRootAndRestart(configPath, currentCfg, walletMgr, reconstructedPassword)
        // promoteToRootAndRestart will os.Exit, so code below won't run for the promoted node
    }
    
    log.Println("Exiting.")
}
```

## Phase 4: Core Monitoring Infrastructure

### 4.1 Add Configuration Support

**File: `config/config.go`**

```go
// Add to Config struct
type Config struct {
    // ... existing fields ...
    IsBootnode            bool   `json:"is_bootnode"`
    CurrentRootNodeAPIURL string `json:"current_root_node_api_url,omitempty"` // URL for bootnodes to monitor
    RootKeyPassword    string `json:"root_key_password,omitempty"`      // Only used temporarily during setup
    // ... other fields ...
}

// Update in DefaultBootnodeConfig()
// cfg.CurrentRootNodeAPIURL = "http://default-root-node-address:port" // Set a default or leave for installer
```

### 4.2 Create Failover Manager

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
    passwordShardMgr    *PasswordShardManager // For password shard management
    reconstructedPassword string          // Reconstructed password during failover
    discoveryManager    *DiscoveryManager // For getting bootnode devs
    stopChan            chan struct{}
    failoverInProgress  bool
    mainContextCancelFn context.CancelFunc // To signal the main application to shut down its current role
}

// NewFailoverManager creates a new FailoverManager.
func NewFailoverManager(rootAPIURL string, cfg *config.Config, configPath string, wm *WalletManager, wallet *Wallet, ps *pubsub.PubSub, passwordShardMgr *PasswordShardManager, discoveryMgr *DiscoveryManager, mainCancelFn context.CancelFunc) *FailoverManager {
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
        passwordShardMgr:    passwordShardMgr,
        discoveryManager:    discoveryMgr,
        stopChan:            make(chan struct{}),
        mainContextCancelFn: mainCancelFn,
    }
}

// ... rest of the FailoverManager implementation ...
```

## Phase 5: Network Control Protocol

### 5.1 Define Network Control Messages

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

## Phase 6: HTTP Server Integration

### 6.1 Update BlockchainServer

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

## Phase 7: Testing and Validation

### 7.1 Unit Tests

Develop unit tests for each component:

1. PasswordShardManager
   - Test shard creation and distribution
   - Test password reconstruction
   - Test encryption/decryption of shards

2. Failover Manager
   - Test root node status monitoring
   - Test election mechanism
   - Test password collection and reconstruction
   - Test promotion logic

3. Network Control Protocol
   - Test pause/resume message handling
   - Test transaction pool clearing

### 7.2 Integration Tests

Develop integration tests to validate the complete failover flow:

1. Root Node Failure Simulation
   - Simulate root node going offline
   - Verify bootnode detection and election
   - Verify password shard collection and reconstruction
   - Verify promotion and restart with reconstructed password

2. Network Behavior During Failover
   - Verify network pause is respected
   - Verify transaction pool handling
   - Verify chain continuity

3. Client Experience
   - Verify client reconnection to new root
   - Verify transaction processing after failover

### 7.3 Security Testing

Evaluate the security of the password sharing mechanism:

1. Shard Security
   - Test encryption strength of shards
   - Verify shards cannot be used individually
   - Test threshold requirements

2. Message Authentication
   - Verify signature validation prevents forgery
   - Test resistance to replay attacks

3. Password Reconstruction
   - Verify password can only be reconstructed with sufficient shards
   - Test handling of invalid or corrupted shards

## Implementation Considerations

1. **Security**: The password sharing system uses Shamir's Secret Sharing algorithm to ensure the password can only be reconstructed when a threshold number of shards are combined.

2. **Encryption**: Each shard is encrypted before transmission to protect against eavesdropping.

3. **Authentication**: All password shard messages are signed to prevent forgery and ensure they come from authorized nodes.

4. **Threshold Configuration**: The system is configured to require a threshold number of shards (default: 3 out of 5) to reconstruct the password, balancing security with availability.

5. **Chain Continuity**: The new root node uses its existing database to maintain chain continuity, and the reconstructed password allows it to properly authenticate as a root node.

6. **Fallback Mechanism**: If password reconstruction fails, the system logs a warning but continues with the failover process, potentially requiring manual intervention for password entry.

## Future Enhancements

1. **Periodic Shard Rotation**: Implement automatic rotation of password shards on a regular schedule to enhance security.

2. **Hierarchical Sharding**: Create a hierarchical sharding system where different parts of the password are sharded separately with different thresholds.

3. **Hardware Security Module Integration**: Store shards in hardware security modules for enhanced protection.

4. **Quorum-Based Approval**: Require a quorum of bootnodes to approve the password reconstruction process before it proceeds.

5. **Audit Logging**: Implement comprehensive audit logging for all password shard operations to track access and usage.

---

<div class="footer-links">


© 2025 KNIRV Network
</div>
