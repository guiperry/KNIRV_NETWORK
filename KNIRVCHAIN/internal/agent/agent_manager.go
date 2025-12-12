package agent

import (
	"KNIRVCHAIN/internal/types"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

// Type definitions (temporarily defined here to avoid circular imports)
type ChromemManager struct {
	mu                        sync.RWMutex
	transactionCollection     *Collection // Placeholder for collection
	contextRecordCollection   *Collection // Placeholder for collection
	badgeAttachmentCollection *Collection // Placeholder for collection
}

// Add placeholder methods for ChromemManager
func (cm *ChromemManager) Close() {
	// Placeholder for cleanup operations
}

func (cm *ChromemManager) StoreAgent(agent *Agent) error {
	return fmt.Errorf("store agent not implemented")
}

func (cm *ChromemManager) StoreRevision(revision *Revision) error {
	return fmt.Errorf("store revision not implemented")
}

func (cm *ChromemManager) GetAgent(id string) (*Agent, error) {
	return nil, fmt.Errorf("get agent not implemented")
}

func (cm *ChromemManager) GetAgentsByOwner(owner string) ([]*Agent, error) {
	return nil, fmt.Errorf("get agents by owner not implemented")
}

func (cm *ChromemManager) GetAgentsByType(agentType string) ([]*Agent, error) {
	return nil, fmt.Errorf("get agents by type not implemented")
}

func (cm *ChromemManager) StoreAgentRelationship(relationship *AgentRelationship) error {
	return fmt.Errorf("store agent relationship not implemented")
}

func (cm *ChromemManager) GetAgentRelationships(agentID string) ([]*AgentRelationship, error) {
	return nil, fmt.Errorf("get agent relationships not implemented")
}

func (cm *ChromemManager) StoreBadgeAttachment(attachment *BadgeAttachment) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.badgeAttachmentCollection == nil {
		return fmt.Errorf("badgeAttachmentCollection not initialized")
	}

	// Convert attachment to JSON
	attachmentJSON, err := json.Marshal(attachment)
	if err != nil {
		return fmt.Errorf("failed to marshal badge attachment: %v", err)
	}

	// Create content and metadata
	content := fmt.Sprintf("Badge Attachment ID: %s, Agent ID: %s, Badge ID: %s, Status: %s, Attached: %s",
		attachment.ID, attachment.AgentId, attachment.BadgeId, attachment.Status, attachment.AttachedAt.Format(time.RFC3339))

	metadata := map[string]string{
		"id":          attachment.ID,
		"agent_id":    attachment.AgentId,
		"badge_id":    attachment.BadgeId,
		"status":      attachment.Status,
		"attached_at": attachment.AttachedAt.Format(time.RFC3339),
		"data":        string(attachmentJSON),
	}

	// Store using collection
	ctx := context.Background()
	if err := cm.badgeAttachmentCollection.Add(ctx, []string{attachment.ID}, nil, []map[string]string{metadata}, []string{content}); err != nil {
		return fmt.Errorf("failed to store badge attachment: %v", err)
	}
	return nil
}

func (cm *ChromemManager) GetBadgeAttachments(agentID string) ([]*BadgeAttachment, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if cm.badgeAttachmentCollection == nil {
		return nil, fmt.Errorf("badgeAttachmentCollection not initialized")
	}

	ctx := context.Background()
	includeMap := map[string]string{"documents": "true"}
	// Query for agent ID - collection implementation may vary
	results, err := cm.badgeAttachmentCollection.Query(ctx, fmt.Sprintf("Agent ID: %s", agentID), 100, nil, includeMap)
	if err != nil {
		return nil, err
	}

	var attachments []*BadgeAttachment
	for _, res := range results {
		// Try metadata first
		if res.Metadata != nil {
			if data, ok := res.Metadata["data"]; ok && data != "" {
				var attachment BadgeAttachment
				if err := json.Unmarshal([]byte(data), &attachment); err == nil {
					// Filter by agent ID just in case
					if attachment.AgentId == agentID {
						attachments = append(attachments, &attachment)
						continue
					}
				}
			}
		}

		// Fallback to content
		if res.Content != "" {
			var attachment BadgeAttachment
			if err := json.Unmarshal([]byte(res.Content), &attachment); err == nil {
				if attachment.AgentId == agentID {
					attachments = append(attachments, &attachment)
				}
			}
		}
	}

	return attachments, nil
}

func (cm *ChromemManager) GetBadgeAttachment(attachmentID string) (*BadgeAttachment, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if cm.badgeAttachmentCollection == nil {
		return nil, fmt.Errorf("badgeAttachmentCollection not initialized")
	}

	ctx := context.Background()
	includeMap := map[string]string{"documents": "true"}
	results, err := cm.badgeAttachmentCollection.Query(ctx, "Badge Attachment", 50, nil, includeMap)
	if err != nil {
		return nil, err
	}

	for _, res := range results {
		// Try metadata[data]
		if res.Metadata != nil {
			if data, ok := res.Metadata["data"]; ok && data != "" {
				var attachment BadgeAttachment
				if err := json.Unmarshal([]byte(data), &attachment); err == nil {
					if attachment.ID == attachmentID {
						return &attachment, nil
					}
				}
			}
		}

		// Fallback to Content
		if res.Content != "" {
			var attachment BadgeAttachment
			if err := json.Unmarshal([]byte(res.Content), &attachment); err == nil {
				if attachment.ID == attachmentID {
					return &attachment, nil
				}
			}
		}
	}

	return nil, fmt.Errorf("badge attachment not found")
}

func (cm *ChromemManager) StoreBadge(badge *Badge) error {
	return fmt.Errorf("store badge not implemented")
}

func (cm *ChromemManager) GetBadge(id string) (*Badge, error) {
	return nil, fmt.Errorf("get badge not implemented")
}

func (cm *ChromemManager) GetBadges() ([]*Badge, error) {
	return nil, fmt.Errorf("get badges not implemented")
}

func (cm *ChromemManager) StoreAgentPlugin(plugin *AgentPlugin) error {
	return fmt.Errorf("store agent plugin not implemented")
}

func (cm *ChromemManager) GetAgentPlugins(agentID string) ([]*AgentPlugin, error) {
	return nil, fmt.Errorf("get agent plugins not implemented")
}

func (cm *ChromemManager) GetAgentPlugin(pluginID string) (*AgentPlugin, error) {
	return nil, fmt.Errorf("get agent plugin not implemented")
}

func (cm *ChromemManager) GetAgentRelationship(relationshipID string) (*AgentRelationship, error) {
	return nil, fmt.Errorf("get agent relationship not implemented")
}

func (cm *ChromemManager) GetBadgesByType(badgeType string) ([]*Badge, error) {
	return nil, fmt.Errorf("get badges by type not implemented")
}

func (cm *ChromemManager) GetBadgeAttachmentByAgentAndBadge(agentID, badgeID string) (*BadgeAttachment, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if cm.badgeAttachmentCollection == nil {
		return nil, fmt.Errorf("badgeAttachmentCollection not initialized")
	}

	ctx := context.Background()
	includeMap := map[string]string{"documents": "true"}
	results, err := cm.badgeAttachmentCollection.Query(ctx, "Badge Attachment", 50, nil, includeMap)
	if err != nil {
		return nil, err
	}

	for _, res := range results {
		// Match on metadata fields
		if res.Metadata != nil {
			if res.Metadata["agent_id"] == agentID && res.Metadata["badge_id"] == badgeID {
				// Check metadata data
				if data := res.Metadata["data"]; data != "" {
					var attachment BadgeAttachment
					if err := json.Unmarshal([]byte(data), &attachment); err == nil {
						return &attachment, nil
					}
				}
			}
		}

		// Fallback to content
		if res.Content != "" {
			var attachment BadgeAttachment
			if err := json.Unmarshal([]byte(res.Content), &attachment); err == nil {
				if attachment.AgentId == agentID && attachment.BadgeId == badgeID {
					return &attachment, nil
				}
			}
		}
	}

	return nil, fmt.Errorf("badge attachment not found for agent %s and badge %s", agentID, badgeID)
}

type Collection struct {
	// Placeholder for collection
}

func (c *Collection) Add(ctx context.Context, ids []string, embeddings interface{}, metadata []map[string]string, documents []string) error {
	// Placeholder implementation
	return fmt.Errorf("collection add not implemented")
}

func (c *Collection) Query(ctx context.Context, query string, limit int, where map[string]string, include map[string]string) ([]Result, error) {
	// Placeholder implementation
	return nil, fmt.Errorf("collection query not implemented")
}

type Result struct {
	ID       string
	Metadata map[string]string
	Document string
	Content  string // Add Content field for compatibility
}

type MCPProcessor struct {
	// Placeholder for MCPProcessor
}

func (mcp *MCPProcessor) getCapabilityByID(capabilityID string) (*Capability, error) {
	if capabilityID == "" {
		return nil, fmt.Errorf("capability ID cannot be empty")
	}

	// Minimal placeholder implementation: return a capability with the given ID
	// In a real implementation this would query the on-disk DB or discovery service
	capability := &Capability{
		ID:             capabilityID,
		Name:           fmt.Sprintf("capability-%s", capabilityID),
		Description:    "placeholder capability",
		CapabilityType: "unknown",
		MCPServerURL:   "",
		Schema:         make(map[string]interface{}),
		LocationHints:  []string{},
		Metadata:       make(map[string]interface{}),
		CreatedAt:      time.Now(),
		Status:         "active",
	}

	return capability, nil
}

func (mcp *MCPProcessor) GetCapabilityDescriptor(capabilityID string) (interface{}, error) {
	cap, err := mcp.getCapabilityByID(capabilityID)
	if err != nil {
		return nil, err
	}

	// Return the schema/descriptor as the capability descriptor
	if cap.Schema != nil {
		return cap.Schema, nil
	}

	// Fallback to a generic descriptor
	descriptor := map[string]interface{}{
		"id":              cap.ID,
		"name":            cap.Name,
		"capability_type": cap.CapabilityType,
		"owner":           cap.Metadata["owner"],
	}
	return descriptor, nil
}

// Wallet interface to allow compatibility with different wallet implementations
type Wallet interface {
	GetAddress() string
	GetPublicKeyHex() string
}

type WalletManager struct {
	// Placeholder for WalletManager
}

func (wm *WalletManager) LoadMasterWallet(address string, role interface{}) (*Wallet, error) {
	// Placeholder implementation
	return nil, fmt.Errorf("load master wallet not implemented")
}

// Block type definition
type Block struct {
	Hash         []byte         `json:"hash"`
	BlockHash    []byte         `json:"block_hash"`
	PrevHash     []byte         `json:"prev_hash"`
	BlockNumber  int64          `json:"block_number"`
	Timestamp    int64          `json:"timestamp"`
	Nonce        int64          `json:"nonce"`
	Data         []byte         `json:"data"`
	Transactions []*Transaction `json:"transactions"`
}

func (b *Block) VerifyBlock() bool {
	// Placeholder implementation
	return true
}

// BlockchainStruct type definition
type BlockchainStruct struct {
	Blocks          []*Block        `json:"blocks"`
	TransactionPool []*Transaction  `json:"transaction_pool"`
	PoAuDEnabled    bool            `json:"poaud_enabled"`
	ChainAddress    string          `json:"chain_address"`
	Reflections     map[string]bool `json:"reflections"`
	mcpProcessor    *MCPProcessor   `json:"-"`
	db              *LevelDB        `json:"-"`
	mu              sync.RWMutex    `json:"-"`
}

func (bc *BlockchainStruct) SetBlocks(blocks []*Block) {
	bc.Blocks = blocks
}

func (bc *BlockchainStruct) Lock() {
	bc.mu.Lock()
}

func (bc *BlockchainStruct) Unlock() {
	bc.mu.Unlock()
}

func (bc *BlockchainStruct) IsActivelyMining() bool {
	// Placeholder implementation
	return false
}

// LevelDB type definition
type LevelDB struct {
	// Placeholder for LevelDB
}

func (db *LevelDB) Get(key []byte) ([]byte, error) {
	// Placeholder implementation
	return nil, fmt.Errorf("leveldb get not implemented")
}

func (db *LevelDB) Put(key, value []byte) error {
	// Placeholder implementation
	return fmt.Errorf("leveldb put not implemented")
}

func (db *LevelDB) PutPoAuDEnabled(enabled bool) error {
	// Placeholder implementation
	return fmt.Errorf("leveldb put poaud enabled not implemented")
}

func (db *LevelDB) PutIntoDb(blockchain *BlockchainStruct, chainAddress string) error {
	// Placeholder implementation
	return fmt.Errorf("leveldb put into db not implemented")
}

// TransactionPoolManager type definition
type TransactionPoolManager struct {
	// Placeholder for TransactionPoolManager
}

func (tpm *TransactionPoolManager) AddToPASPool(tx *Transaction) {
	// Placeholder implementation
}

func (tpm *TransactionPoolManager) IsDelegated(txHash string) bool {
	// Placeholder implementation
	return false
}

func (tpm *TransactionPoolManager) MarkAsDelegated(txHash string) {
	// Placeholder implementation
}

func (tpm *TransactionPoolManager) ReclaimStaleTransactions(maxStaleTime interface{}) {
	// Placeholder implementation
}

func (tpm *TransactionPoolManager) GetPoolStats() map[string]interface{} {
	// Placeholder implementation
	return make(map[string]interface{})
}

type DiscoveryManager struct {
	host interface{} // Placeholder for host - can be type asserted to host.Host when needed
}

func (d *DiscoveryManager) FindGenericResource(address string, resourceType string) ([]interface{}, error) {
	// Placeholder implementation
	return nil, fmt.Errorf("discovery manager not implemented")
}

func (d *DiscoveryManager) AnnounceGenericResource(resourceID string, resourceType interface{}) error {
	// Placeholder implementation
	return fmt.Errorf("announce generic resource not implemented")
}

// Discovery resource type constants
const (
	DiscoveryResourceTypeChain = "chain"
)

// Transaction type definition
type Transaction struct {
	TransactionHash string `json:"transaction_hash"`
	From            string `json:"from"`
	To              string `json:"to"`
	Type            string `json:"type"`
	Data            []byte `json:"data"`
	Fee             int64  `json:"fee"`
	Timestamp       int64  `json:"timestamp"`
}

// Transaction type constants
const (
	TransactionTypeMCPInvokeCapability = "mcp_invoke_capability"
)

// Transaction methods
func (t *Transaction) Clone() *Transaction {
	// Placeholder implementation
	return &Transaction{
		TransactionHash: t.TransactionHash,
		From:            t.From,
		To:              t.To,
		Type:            t.Type,
		Data:            append([]byte{}, t.Data...),
		Fee:             t.Fee,
		Timestamp:       t.Timestamp,
	}
}

func (t *Transaction) VerifySignature() (bool, error) {
	// Placeholder implementation
	return true, nil
}

func (t *Transaction) VerifyTxn() bool {
	// Placeholder implementation
	return true
}

// Agent represents an Agent in the system (enhanced from NFT)
type Agent struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	Description     string                 `json:"description"`
	ImageURL        string                 `json:"image_url"`
	Owner           string                 `json:"owner"`
	CreatedAt       time.Time              `json:"created_at"`
	AgentType       string                 `json:"agent_type"`       // "assistant", "worker", "analyzer", etc.
	PluginPath      string                 `json:"plugin_path"`      // Path to the compiled WASM plugin file
	ServerURL       string                 `json:"server_url"`       // URL of the server instance serving this plugin
	Metadata        map[string]interface{} `json:"metadata"`         // Type-specific metadata
	Capabilities    []Capability           `json:"capabilities"`     // Capabilities associated with this Agent
	ParentAgents    []string               `json:"parent_agents"`    // IDs of parent Agents
	ChildAgents     []string               `json:"child_agents"`     // IDs of child Agents
	AccessControls  []AccessControl        `json:"access_controls"`  // Access control rules
	RevisionHistory []Revision             `json:"revision_history"` // History of changes
	PaymentConfig   *PaymentConfig         `json:"payment_config"`   // Payment configuration for Agent invocation
	AttachedBadges  []string               `json:"attached_badges"`  // IDs of attached badges (for backward compatibility)
}

// Capability represents a capability directly associated with an Agent
type Capability struct {
	ID             string                 `json:"id"`
	Name           string                 `json:"name"`
	Description    string                 `json:"description"`
	CapabilityType string                 `json:"capability_type"` // Type of capability
	MCPServerURL   string                 `json:"mcp_server_url"`  // URL of the MCP server for this capability
	Schema         map[string]interface{} `json:"schema"`          // JSON schema for the capability
	LocationHints  []string               `json:"location_hints"`  // Hints for capability location
	Metadata       map[string]interface{} `json:"metadata"`        // Additional metadata
	CreatedAt      time.Time              `json:"created_at"`
	Status         string                 `json:"status"` // "active", "inactive", "deprecated"
}

// AccessControl defines who can access an Agent and how
type AccessControl struct {
	ID          string    `json:"id"`
	AgentId     string    `json:"agent_id"`     // Agent ID
	AccessType  string    `json:"access_type"`  // "owner", "public", "specific_user", "token_holder"
	AccessValue string    `json:"access_value"` // Address, token ID, etc. depending on AccessType
	Permissions []string  `json:"permissions"`  // "view", "execute", "modify", "transfer"
	CreatedAt   time.Time `json:"created_at"`
	ExpiresAt   time.Time `json:"expires_at,omitempty"`
}

// Revision represents a change to an Agent
type Revision struct {
	ID         string                 `json:"id"`
	AgentId    string                 `json:"agent_id"` // Agent ID
	Timestamp  time.Time              `json:"timestamp"`
	ChangedBy  string                 `json:"changed_by"`
	ChangeType string                 `json:"change_type"` // "create", "update", "add_capability", "remove_capability"
	Details    map[string]interface{} `json:"details"`
}

// AgentRelationship represents a mutable relationship between two Agents
type AgentRelationship struct {
	ID            string                 `json:"id"`
	SourceAgentId string                 `json:"source_agent_id"`
	TargetAgentId string                 `json:"target_agent_id"`
	RelationType  string                 `json:"relation_type"` // "parent-child", "dependency", "extension", etc.
	CreatedAt     time.Time              `json:"created_at"`
	CreatedBy     string                 `json:"created_by"`
	UpdatedAt     time.Time              `json:"updated_at"`
	UpdatedBy     string                 `json:"updated_by"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	Parameters    map[string]interface{} `json:"parameters"` // Configurable parameters for this relationship
	Status        string                 `json:"status"`     // "active", "inactive", "pending", "revoked"
}

// PaymentConfig defines payment settings for an Agent
type PaymentConfig struct {
	ID                 string         `json:"id"`
	AgentId            string         `json:"agent_id"`
	TokenAddress       string         `json:"token_address"`        // Address of the payment token contract
	PricePerInvocation string         `json:"price_per_invocation"` // Price in smallest token unit
	MinimumPayment     string         `json:"minimum_payment"`      // Minimum payment required
	PaymentRecipient   string         `json:"payment_recipient"`    // Address to receive payments
	RevenueShare       []RevenueShare `json:"revenue_share"`        // Optional revenue sharing with other Agents
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
	Status             string         `json:"status"` // "active", "inactive"
}

// RevenueShare defines how revenue is shared with other Agents
type RevenueShare struct {
	AgentId       string  `json:"agent_id"`
	SharePercent  float64 `json:"share_percent"` // Percentage of the payment (0-100)
	RecipientAddr string  `json:"recipient_addr"`
}

// AgentPlugin represents a compiled WASM plugin for an Agent
type AgentPlugin struct {
	ID          string    `json:"id"`
	AgentId     string    `json:"agent_id"`
	FilePath    string    `json:"file_path"`    // Path to the compiled plugin file
	Hash        string    `json:"hash"`         // Hash of the plugin file for verification
	Version     string    `json:"version"`      // Version of the plugin
	CompileTime time.Time `json:"compile_time"` // When the plugin was compiled
	CompiledBy  string    `json:"compiled_by"`  // Who compiled the plugin
	ServerURLs  []string  `json:"server_urls"`  // URLs of servers hosting this plugin
	Status      string    `json:"status"`       // "active", "deprecated", "revoked"
}

// Badge represents a Badge that can be attached to Agents (for backward compatibility)
type Badge struct {
	ID               string                 `json:"id"`
	Name             string                 `json:"name"`
	Description      string                 `json:"description"`
	ImageURL         string                 `json:"image_url"`
	Owner            string                 `json:"owner"`
	CreatedAt        time.Time              `json:"created_at"`
	BadgeType        string                 `json:"badge_type"`         // Type of badge
	Metadata         map[string]interface{} `json:"metadata"`           // Badge-specific metadata
	AttachedToAgents []string               `json:"attached_to_agents"` // IDs of agents this badge is attached to
}

// BadgeAttachment represents an immutable relationship between an Agent and a Badge
type BadgeAttachment struct {
	ID         string                 `json:"id"`
	AgentId    string                 `json:"agent_id"`
	BadgeId    string                 `json:"badge_id"`
	AttachedAt time.Time              `json:"attached_at"`
	AttachedBy string                 `json:"attached_by"`
	Parameters map[string]interface{} `json:"parameters,omitempty"` // Configuration parameters for this attachment
	Status     string                 `json:"status"`               // "active", "inactive", "detached"
	Signature  string                 `json:"signature"`            // Cryptographic signature ensuring immutability
	TxHash     string                 `json:"tx_hash"`              // Blockchain transaction hash for verification
}

// ServerInstance represents a server that can host Plugin Agents
type ServerInstance struct {
	ID           string                 `json:"id"`
	URL          string                 `json:"url"`
	Name         string                 `json:"name"`
	Owner        string                 `json:"owner"`
	Capabilities map[string]interface{} `json:"capabilities"`
	CreatedAt    time.Time              `json:"created_at"`
	Status       string                 `json:"status"` // "active", "inactive", "maintenance"
}

// Legacy structures for backward compatibility
// NFT represents a Non-Fungible Token in the system (legacy, now mapped to Agent)
type NFT struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	ImageURL     string    `json:"image_url"`
	Owner        string    `json:"owner"`
	CreatedAt    time.Time `json:"created_at"`
	Capabilities []string  `json:"capabilities"` // IDs of attached capabilities
}

// NFTCapabilityAttachment represents a relationship between an NFT and a capability (legacy)
type NFTCapabilityAttachment struct {
	ID           string                 `json:"id"`
	NFTId        string                 `json:"nft_id"`
	CapabilityID string                 `json:"capability_id"`
	AttachedAt   time.Time              `json:"attached_at"`
	AttachedBy   string                 `json:"attached_by"`
	Params       map[string]interface{} `json:"params,omitempty"`
	Status       string                 `json:"status"` // "active", "revoked", etc.
}

// Database collections in ChromeDB
const (
	// Agent-centric collections
	AgentCollection             = "agents"
	AgentRelationshipCollection = "agent_relationships"
	AccessControlCollection     = "access_controls"
	RevisionCollection          = "revisions"
	ContextRecordCollection     = "context_records"
	AgentPluginCollection       = "agent_plugins"
	ServerInstanceCollection    = "server_instances"
	PaymentConfigCollection     = "agent_payment_configs"
	PaymentReceiptCollection    = "payment_receipts"
	CapabilityCollection        = "capabilities"
	BadgeCollection             = "badges"
	BadgeAttachmentCollection   = "badge_attachments"

	// Legacy collections (for backward compatibility)
	NFTCollection                     = "nft_collection"
	NFTCapabilityAttachmentCollection = "nft_capability_attachments"
	NFTCapabilityHistoryCollection    = "nft_history"
)

// AgentManager handles Agent operations
type AgentManager struct {
	chromeManager    *ChromemManager
	mcpProcessor     *MCPProcessor
	wallet           Wallet
	pluginManager    *PluginManager
	discoveryManager *DiscoveryManager
}

// PluginManager handles plugin-related operations
type PluginManager struct {
}

// NewAgentManager creates a new Agent manager
func NewAgentManager(chromemManager *ChromemManager, mcpProcessor *MCPProcessor, wallet Wallet, discoveryManager *DiscoveryManager) *AgentManager {
	return &AgentManager{
		chromeManager:    chromemManager,
		mcpProcessor:     mcpProcessor,
		wallet:           wallet,
		pluginManager:    &PluginManager{}, // Initialize with empty plugin manager for now
		discoveryManager: discoveryManager,
	}
}

// NFTManager handles NFT-related operations (legacy, wraps AgentManager)
type NFTManager struct {
	agentManager *AgentManager
}

// NewNFTManager creates a new NFT manager (legacy wrapper)
func NewNFTManager(chromemManager *ChromemManager, mcpProcessor *MCPProcessor, wallet Wallet, discoveryManager *DiscoveryManager) *NFTManager {
	return &NFTManager{
		agentManager: NewAgentManager(chromemManager, mcpProcessor, wallet, discoveryManager),
	}
}

// StoreNFT stores an NFT in ChromeDB
func (mgr *ChromemManager) StoreNFT(nft *NFT) error {
	mgr.mu.Lock()
	defer mgr.mu.Unlock()

	// Convert NFT to JSON
	nftJSON, err := json.Marshal(nft)
	if err != nil {
		return err
	}

	// Create metadata for the NFT
	metadata := map[string]string{
		"owner": nft.Owner,
		"type":  "nft",
	}

	// Store in ChromeDB using the collection
	ctx := context.Background()
	err = mgr.transactionCollection.Add(
		ctx,
		[]string{nft.ID},
		nil, // No embeddings needed
		[]map[string]string{metadata},
		[]string{string(nftJSON)},
	)
	return err
}

// GetNFT retrieves an NFT from ChromeDB
func (mgr *ChromemManager) GetNFT(id string) (*NFT, error) {
	mgr.mu.RLock()
	defer mgr.mu.RUnlock()

	// Get the document from ChromeDB
	ctx := context.Background()
	includeMap := map[string]string{
		"documents": "true",
	}

	results, err := mgr.transactionCollection.Query(
		ctx,
		id, // Query by ID
		1,  // Limit to 1 result
		nil,
		includeMap,
	)
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("NFT not found")
	}

	// Convert document back to NFT
	var nft NFT
	err = json.Unmarshal([]byte(results[0].Content), &nft)
	if err != nil {
		return nil, err
	}
	return &nft, nil
}

// GetNFTsByOwner retrieves all NFTs owned by a specific address from ChromeDB
func (mgr *ChromemManager) GetNFTsByOwner(owner string) ([]*NFT, error) {
	mgr.mu.RLock()
	defer mgr.mu.RUnlock()

	// Query by owner metadata
	where := map[string]string{
		"owner": owner,
		"type":  "nft",
	}

	// Get all NFTs matching the query
	ctx := context.Background()
	includeMap := map[string]string{
		"documents": "true",
	}

	results, err := mgr.transactionCollection.Query(
		ctx,
		"",  // Empty query to match only where clause
		100, // Limit to 100 results
		where,
		includeMap,
	)
	if err != nil {
		return nil, err
	}

	// Process the results
	nfts := make([]*NFT, 0, len(results))
	for _, doc := range results {
		var nft NFT
		if err := json.Unmarshal([]byte(doc.Content), &nft); err != nil {
			continue // Skip this one if there's an error
		}

		nfts = append(nfts, &nft)
	}

	return nfts, nil
}

// StoreNFTCapabilityAttachment stores an NFT-capability attachment in ChromeDB
func (mgr *ChromemManager) StoreNFTCapabilityAttachment(attachment *NFTCapabilityAttachment) error {
	mgr.mu.Lock()
	defer mgr.mu.Unlock()

	// Convert attachment to JSON
	attachmentJSON, err := json.Marshal(attachment)
	if err != nil {
		return err
	}

	// Create metadata for the attachment
	metadata := map[string]string{
		"attachment.NFTId": attachment.NFTId,
		"capability_id":    attachment.CapabilityID,
		"type":             "nft",
	}

	// Store in ChromeDB using the collection
	ctx := context.Background()
	err = mgr.contextRecordCollection.Add(
		ctx,
		[]string{attachment.ID},
		nil, // No embeddings needed
		[]map[string]string{metadata},
		[]string{string(attachmentJSON)},
	)
	return err
}

// GetNFTCapabilityAttachments retrieves all capability attachments for an NFT from ChromeDB
func (mgr *ChromemManager) GetNFTCapabilityAttachments(nftId string) ([]*NFTCapabilityAttachment, error) {
	mgr.mu.RLock()
	defer mgr.mu.RUnlock()

	// Query by NFT ID metadata
	where := map[string]string{
		"nftlowBadgeId": nftId,
		"type":          "nft",
	}

	// Get all attachments matching the query
	ctx := context.Background()
	includeMap := map[string]string{
		"documents": "true",
	}

	results, err := mgr.contextRecordCollection.Query(
		ctx,
		"",  // Empty query to match only where clause
		100, // Limit to 100 results
		where,
		includeMap,
	)
	if err != nil {
		return nil, err
	}

	// Process the results
	attachments := make([]*NFTCapabilityAttachment, 0, len(results))
	for _, doc := range results {
		var attachment NFTCapabilityAttachment
		if err := json.Unmarshal([]byte(doc.Content), &attachment); err != nil {
			continue // Skip this one if there's an error
		}

		attachments = append(attachments, &attachment)
	}

	return attachments, nil
}

// CreateNFT creates a new NFT (legacy wrapper)
func (nm *NFTManager) CreateNFT(name, description, imageURL, owner string) (*NFT, error) {
	// Generate a unique ID for the NFT
	id := generateUniqueID()

	nft := &NFT{
		ID:           id,
		Name:         name,
		Description:  description,
		ImageURL:     imageURL,
		Owner:        owner,
		CreatedAt:    time.Now(),
		Capabilities: []string{},
	}

	// Store the NFT in ChromeDB
	if err := nm.agentManager.chromeManager.StoreNFT(nft); err != nil {
		return nil, err
	}

	return nft, nil
}

// GetNFT retrieves an NFT by ID (legacy wrapper)
func (nm *NFTManager) GetNFT(id string) (*NFT, error) {
	return nm.agentManager.chromeManager.GetNFT(id)
}

// GetNFTsByOwner retrieves all NFTs owned by a specific address (legacy wrapper)
func (nm *NFTManager) GetNFTsByOwner(owner string) ([]*NFT, error) {
	return nm.agentManager.chromeManager.GetNFTsByOwner(owner)
}

// AttachCapability attaches a capability to an NFT (legacy wrapper)
func (nm *NFTManager) AttachCapability(nftID, capabilityID, attachedBy string, params map[string]interface{}) (*NFTCapabilityAttachment, error) {
	// Check if the NFT exists
	nft, err := nm.GetNFT(nftID)
	if err != nil {
		return nil, fmt.Errorf("nft not found: %v", err)
	}

	// Check if the capability exists
	_, err = nm.agentManager.mcpProcessor.getCapabilityByID(capabilityID)
	if err != nil {
		return nil, fmt.Errorf("capability not found: %v", err)
	}

	// Check if the user is the owner of the NFT
	if nft.Owner != attachedBy {
		return nil, fmt.Errorf("only the nft owner can attach capabilities")
	}

	// Create the attachment
	attachment := &NFTCapabilityAttachment{
		ID:           generateUniqueID(),
		NFTId:        nftID,
		CapabilityID: capabilityID,
		AttachedAt:   time.Now(),
		AttachedBy:   attachedBy,
		Params:       params,
		Status:       "active",
	}

	// Store the attachment in ChromeDB
	if err := nm.agentManager.chromeManager.StoreNFTCapabilityAttachment(attachment); err != nil {
		return nil, err
	}

	// Update the NFT's capabilities list
	nft.Capabilities = append(nft.Capabilities, capabilityID)
	if err := nm.agentManager.chromeManager.StoreNFT(nft); err != nil {
		return nil, err
	}

	return attachment, nil
}

// GetCapabilityAttachmentHistory retrieves the history of capability attachments for an NFT (legacy wrapper)
func (nm *NFTManager) GetCapabilityAttachmentHistory(nftID string) ([]*NFTCapabilityAttachment, error) {
	return nm.agentManager.chromeManager.GetNFTCapabilityAttachments(nftID)
}

// ===== AGENT MANAGER METHODS =====

// CreateAgent creates a new Agent with the specified agent type
func (am *AgentManager) CreateAgent(name, description, imageURL, owner, agentType string, metadata map[string]interface{}) (*Agent, error) {
	// Implementation
	agent := &Agent{
		ID:              generateUniqueID(),
		Name:            name,
		Description:     description,
		ImageURL:        imageURL,
		Owner:           owner,
		CreatedAt:       time.Now(),
		AgentType:       agentType,
		Metadata:        metadata,
		Capabilities:    []Capability{},
		ParentAgents:    []string{},
		ChildAgents:     []string{},
		AccessControls:  []AccessControl{},
		RevisionHistory: []Revision{},
		AttachedBadges:  []string{},
	}

	// Store in database
	err := am.chromeManager.StoreAgent(agent)
	if err != nil {
		return nil, fmt.Errorf("failed to store agent: %v", err)
	}

	// Create initial revision
	revision := &Revision{
		ID:         generateUniqueID(),
		AgentId:    agent.ID,
		Timestamp:  time.Now(),
		ChangedBy:  am.wallet.GetAddress(),
		ChangeType: "create",
		Details:    map[string]interface{}{},
	}

	err = am.chromeManager.StoreRevision(revision)
	if err != nil {
		// Log error but don't fail the agent creation
		fmt.Printf("Failed to store revision: %v\n", err)
	}

	return agent, nil
}

// CreateAIAgent creates an AI Agent with AgentFacts metadata
func (am *AgentManager) CreateAIAgent(name, description, imageURL, owner string, agentProfile map[string]interface{}) (*Agent, error) {
	// Create metadata for the agent using the AgentFacts standard
	metadata := map[string]interface{}{
		// Standard AgentFacts fields
		"agent_id":            agentProfile["agent_id"],            // W3C DID Core compliant identifier
		"agent_name":          agentProfile["agent_name"],          // URN (e.g., urn:agent:salesforce:starbucks)
		"facts_url":           agentProfile["facts_url"],           // URL to AgentFacts (e.g., https://salesforce.com/starbucks/.agent-facts)
		"private_facts_url":   agentProfile["private_facts_url"],   // Optional privacy-enhanced reference
		"adaptive_router_url": agentProfile["adaptive_router_url"], // Optional endpoint for dynamic routing services
		"ttl":                 agentProfile["ttl"],                 // Time-To-Live for cache duration
		"signature":           agentProfile["signature"],           // Cryptographic signature from registry resolver

		// Extended AgentFacts fields
		"capabilities": map[string]interface{}{
			"modalities": agentProfile["modalities"], // e.g., ["text", "structured_data"]
			"skills":     agentProfile["skills"],     // e.g., ["analysis", "synthesis", "research"]
		},
		"endpoints": map[string]interface{}{
			"static": agentProfile["static_endpoints"], // Static endpoint URLs
			"adaptive_resolver": map[string]interface{}{
				"url":      agentProfile["adaptive_router_url"],
				"policies": agentProfile["resolver_policies"], // e.g., ["capability_negotiation", "load_balancing"]
			},
		},
		"certification": map[string]interface{}{
			"level":      agentProfile["certification_level"],  // e.g., "verified", "premium"
			"issuer":     agentProfile["certification_issuer"], // Certification authority
			"issued_at":  agentProfile["certification_date"],   // Certification date
			"expires_at": agentProfile["certification_expiry"], // Expiration date
		},

		// Legacy fields for backward compatibility
		"agent_profile":     agentProfile,
		"behavioral_params": agentProfile["behavioral_params"],
		"permission_scopes": agentProfile["permission_scopes"],
		"execution_context": agentProfile["execution_context"],
		"version":           agentProfile["version"],
	}

	// Create the Agent with agent type from profile
	agentType := "assistant"
	if profileAgentType, ok := agentProfile["agent_type"].(string); ok && profileAgentType != "" {
		agentType = profileAgentType
	}

	return am.CreateAgent(name, description, imageURL, owner, agentType, metadata)
}

// ConfigureAgentCapabilities adds multiple capabilities to an Agent
func (am *AgentManager) ConfigureAgentCapabilities(agentId string, capabilityConfigs []map[string]interface{}) error {
	// Verify the Agent exists
	agent, err := am.GetAgent(agentId)
	if err != nil {
		return fmt.Errorf("agent not found: %v", err)
	}

	// Add each capability to the Agent
	for _, config := range capabilityConfigs {
		name, ok := config["name"].(string)
		if !ok {
			return fmt.Errorf("capability name is required")
		}

		description, ok := config["description"].(string)
		if !ok {
			return fmt.Errorf("capability description is required")
		}

		capabilityType, ok := config["capability_type"].(string)
		if !ok {
			return fmt.Errorf("capability type is required")
		}

		mcpServerURL, _ := config["mcp_server_url"].(string)
		schema, _ := config["schema"].(map[string]interface{})
		locationHints, _ := config["location_hints"].([]string)
		metadata, _ := config["metadata"].(map[string]interface{})

		// Check if the capability is compatible with this agent type
		if compatibleAgents, exists := metadata["compatible_agents"].([]string); exists {
			isCompatible := false
			for _, compatibleType := range compatibleAgents {
				if compatibleType == agent.AgentType || compatibleType == "*" {
					isCompatible = true
					break
				}
			}

			if !isCompatible {
				return fmt.Errorf("capability %s is not compatible with agent type %s", name, agent.AgentType)
			}
		}

		// Add agent-specific parameters to the metadata
		if metadata == nil {
			metadata = make(map[string]interface{})
		}

		metadata["agent_id"] = agentId
		metadata["agent_type"] = agent.AgentType
		metadata["configured_at"] = time.Now()
		metadata["configured_by"] = am.wallet.GetAddress()

		// Create the capability
		capability := Capability{
			Name:           name,
			Description:    description,
			CapabilityType: capabilityType,
			MCPServerURL:   mcpServerURL,
			Schema:         schema,
			LocationHints:  locationHints,
			Metadata:       metadata,
		}

		// Add the capability to the agent
		err = am.AddCapabilityToAgent(agentId, capability)
		if err != nil {
			return fmt.Errorf("failed to add capability %s: %v", name, err)
		}
	}

	return nil
}

// CreateAgentWithCapabilities creates an Agent with multiple capabilities
func (am *AgentManager) CreateAgentWithCapabilities(name, description, imageURL, owner string,
	agentProfile map[string]interface{},
	capabilityConfigs []map[string]interface{}) (*Agent, error) {
	// Create the Agent
	agent, err := am.CreateAIAgent(name, description, imageURL, owner, agentProfile)
	if err != nil {
		return nil, err
	}

	// Configure capabilities
	err = am.ConfigureAgentCapabilities(agent.ID, capabilityConfigs)
	if err != nil {
		// Log the error but don't fail the entire operation
		fmt.Printf("Warning: Failed to configure some capabilities: %v\n", err)
	}

	return agent, nil
}

// GetAgent retrieves an Agent by ID
func (am *AgentManager) GetAgent(id string) (*Agent, error) {
	return am.chromeManager.GetAgent(id)
}

// GetAgentsByOwner retrieves all Agents owned by a specific address
func (am *AgentManager) GetAgentsByOwner(owner string) ([]*Agent, error) {
	return am.chromeManager.GetAgentsByOwner(owner)
}

// GetAgentsByType retrieves Agents by agent type
func (am *AgentManager) GetAgentsByType(agentType string) ([]*Agent, error) {
	return am.chromeManager.GetAgentsByType(agentType)
}

// AddCapabilityToAgent adds a capability to an Agent
func (am *AgentManager) AddCapabilityToAgent(agentID string, capability Capability) error {
	// Get the agent
	agent, err := am.GetAgent(agentID)
	if err != nil {
		return fmt.Errorf("failed to get agent: %v", err)
	}

	// Add the capability
	capability.ID = generateUniqueID()
	capability.CreatedAt = time.Now()
	capability.Status = "active"

	agent.Capabilities = append(agent.Capabilities, capability)

	// Update the agent in the database
	err = am.chromeManager.StoreAgent(agent)
	if err != nil {
		return fmt.Errorf("failed to update agent: %v", err)
	}

	// Create revision
	revision := &Revision{
		ID:         generateUniqueID(),
		AgentId:    agentID,
		Timestamp:  time.Now(),
		ChangedBy:  am.wallet.GetAddress(),
		ChangeType: "add_capability",
		Details: map[string]interface{}{
			"capability_id":   capability.ID,
			"capability_name": capability.Name,
		},
	}

	err = am.chromeManager.StoreRevision(revision)
	if err != nil {
		// Log error but don't fail the operation
		fmt.Printf("Failed to store revision: %v\n", err)
	}

	return nil
}

// RemoveCapabilityFromAgent removes a capability from an Agent
func (am *AgentManager) RemoveCapabilityFromAgent(agentID, capabilityID string) error {
	// Get the agent
	agent, err := am.GetAgent(agentID)
	if err != nil {
		return fmt.Errorf("failed to get agent: %v", err)
	}

	// Find and remove the capability
	for i, cap := range agent.Capabilities {
		if cap.ID == capabilityID {
			agent.Capabilities = append(agent.Capabilities[:i], agent.Capabilities[i+1:]...)
			break
		}
	}

	// Update the agent in the database
	err = am.chromeManager.StoreAgent(agent)
	if err != nil {
		return fmt.Errorf("failed to update agent: %v", err)
	}

	// Create revision
	revision := &Revision{
		ID:         generateUniqueID(),
		AgentId:    agentID,
		Timestamp:  time.Now(),
		ChangedBy:  am.wallet.GetAddress(),
		ChangeType: "remove_capability",
		Details: map[string]interface{}{
			"capability_id": capabilityID,
		},
	}

	err = am.chromeManager.StoreRevision(revision)
	if err != nil {
		// Log error but don't fail the operation
		fmt.Printf("Failed to store revision: %v\n", err)
	}

	return nil
}

// UpdateAgent updates an existing Agent
func (am *AgentManager) UpdateAgent(agent *Agent) error {
	// Get the existing agent to preserve CreatedAt
	existingAgent, err := am.GetAgent(agent.ID)
	if err != nil {
		return fmt.Errorf("failed to get existing agent: %v", err)
	}

	// Preserve the original CreatedAt timestamp
	agent.CreatedAt = existingAgent.CreatedAt

	// Store the updated agent
	err = am.chromeManager.StoreAgent(agent)
	if err != nil {
		return fmt.Errorf("failed to update agent: %v", err)
	}

	// Create revision
	revision := &Revision{
		ID:         generateUniqueID(),
		AgentId:    agent.ID,
		Timestamp:  time.Now(),
		ChangedBy:  am.wallet.GetAddress(),
		ChangeType: "update",
		Details:    map[string]interface{}{},
	}

	err = am.chromeManager.StoreRevision(revision)
	if err != nil {
		// Log error but don't fail the operation
		fmt.Printf("Failed to store revision: %v\n", err)
	}

	return nil
}

// CreateAgentRelationship creates a mutable relationship between two Agents
func (am *AgentManager) CreateAgentRelationship(sourceAgentID, targetAgentID, relationType string, parameters map[string]interface{}) (*AgentRelationship, error) {
	// Verify both agents exist
	_, err := am.GetAgent(sourceAgentID)
	if err != nil {
		return nil, fmt.Errorf("source agent not found: %v", err)
	}

	_, err = am.GetAgent(targetAgentID)
	if err != nil {
		return nil, fmt.Errorf("target agent not found: %v", err)
	}

	// Create the relationship
	relationship := &AgentRelationship{
		ID:            generateUniqueID(),
		SourceAgentId: sourceAgentID,
		TargetAgentId: targetAgentID,
		RelationType:  relationType,
		CreatedAt:     time.Now(),
		CreatedBy:     am.wallet.GetAddress(),
		UpdatedAt:     time.Now(),
		UpdatedBy:     am.wallet.GetAddress(),
		Parameters:    parameters,
		Status:        "active",
	}

	// Store the relationship
	err = am.chromeManager.StoreAgentRelationship(relationship)
	if err != nil {
		return nil, fmt.Errorf("failed to store relationship: %v", err)
	}

	return relationship, nil
}

// UpdateAgentRelationship updates an existing Agent relationship
func (am *AgentManager) UpdateAgentRelationship(relationshipID string, parameters map[string]interface{}, status string) error {
	// Get the relationship
	relationship, err := am.chromeManager.GetAgentRelationship(relationshipID)
	if err != nil {
		return fmt.Errorf("failed to get relationship: %v", err)
	}

	// Update the relationship
	relationship.Parameters = parameters
	relationship.Status = status
	relationship.UpdatedAt = time.Now()
	relationship.UpdatedBy = am.wallet.GetAddress()

	// Store the updated relationship
	err = am.chromeManager.StoreAgentRelationship(relationship)
	if err != nil {
		return fmt.Errorf("failed to update relationship: %v", err)
	}

	return nil
}

// GetAgentRelationships retrieves all relationships for an Agent
func (am *AgentManager) GetAgentRelationships(agentID string) ([]*AgentRelationship, error) {
	return am.chromeManager.GetAgentRelationships(agentID)
}

// AttachBadgeToAgent creates an immutable relationship between an Agent and a Badge
func (am *AgentManager) AttachBadgeToAgent(agentID, badgeID string, parameters map[string]interface{}) (*BadgeAttachment, error) {
	// Validate inputs
	if badgeID == "" {
		return nil, fmt.Errorf("badge ID cannot be empty")
	}

	// Verify agent exists
	agent, err := am.GetAgent(agentID)
	if err != nil {
		return nil, fmt.Errorf("agent not found: %v", err)
	}

	// Create the badge attachment
	attachment := &BadgeAttachment{
		ID:         generateUniqueID(),
		AgentId:    agentID,
		BadgeId:    badgeID,
		AttachedAt: time.Now(),
		AttachedBy: am.wallet.GetAddress(),
		Parameters: parameters,
		Status:     "active",
	}

	// Generate cryptographic signature for immutability
	signature, err := am.generateBadgeAttachmentSignature(attachment)
	if err != nil {
		return nil, fmt.Errorf("failed to generate signature: %v", err)
	}
	attachment.Signature = signature

	// Record on blockchain for verification (placeholder for now)
	txHash, err := am.recordBadgeAttachmentOnBlockchain(attachment)
	if err != nil {
		// Log warning but don't fail - blockchain integration is optional
		fmt.Printf("Warning: Failed to record badge attachment on blockchain: %v\n", err)
		attachment.TxHash = "" // Set empty if blockchain recording fails
	} else {
		attachment.TxHash = txHash
	}

	// Store the attachment
	err = am.chromeManager.StoreBadgeAttachment(attachment)
	if err != nil {
		return nil, fmt.Errorf("failed to store badge attachment: %v", err)
	}

	// Update agent's attached badges list
	agent.AttachedBadges = append(agent.AttachedBadges, badgeID)
	err = am.chromeManager.StoreAgent(agent)
	if err != nil {
		return nil, fmt.Errorf("failed to update agent: %v", err)
	}

	return attachment, nil
}

// ===== BADGE MANAGEMENT METHODS =====

// CreateBadge creates a new Badge
func (am *AgentManager) CreateBadge(name, description, imageURL, owner, badgeType string, metadata map[string]interface{}) (*Badge, error) {
	// Generate a unique ID for the Badge
	id := generateUniqueID()

	badge := &Badge{
		ID:               id,
		Name:             name,
		Description:      description,
		ImageURL:         imageURL,
		Owner:            owner,
		CreatedAt:        time.Now(),
		BadgeType:        badgeType,
		Metadata:         metadata,
		AttachedToAgents: []string{},
	}

	// Store the Badge in ChromeDB
	err := am.chromeManager.StoreBadge(badge)
	if err != nil {
		return nil, fmt.Errorf("failed to store badge: %v", err)
	}

	return badge, nil
}

// GetBadge retrieves a Badge by ID
func (am *AgentManager) GetBadge(id string) (*Badge, error) {
	return am.chromeManager.GetBadge(id)
}

// GetBadgesByType retrieves Badges by badge type
func (am *AgentManager) GetBadgesByType(badgeType string) ([]*Badge, error) {
	return am.chromeManager.GetBadgesByType(badgeType)
}

// GetSkillBadges retrieves all skill badges
func (am *AgentManager) GetSkillBadges() ([]*Badge, error) {
	return am.GetBadgesByType("skill")
}

// GetCapabilityBadges retrieves all capability badges
func (am *AgentManager) GetCapabilityBadges() ([]*Badge, error) {
	return am.GetBadgesByType("capability")
}

// GetPropertyBadges retrieves all property badges
func (am *AgentManager) GetPropertyBadges() ([]*Badge, error) {
	return am.GetBadgesByType("property")
}

// GetAgentSkills retrieves all skill badges attached to a specific agent
func (am *AgentManager) GetAgentSkills(agentID string) ([]*Badge, error) {
	attachments, err := am.chromeManager.GetBadgeAttachments(agentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get badge attachments for agent: %v", err)
	}

	var skills []*Badge
	for _, attachment := range attachments {
		badge, err := am.GetBadge(attachment.BadgeId)
		if err != nil {
			continue // Skip if badge not found
		}

		if badge.BadgeType == "skill" {
			skills = append(skills, badge)
		}
	}

	return skills, nil
}

// GetAgentProperties retrieves all property badges attached to a specific agent
func (am *AgentManager) GetAgentProperties(agentID string) ([]*Badge, error) {
	attachments, err := am.chromeManager.GetBadgeAttachments(agentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get badge attachments for agent: %v", err)
	}

	var properties []*Badge
	for _, attachment := range attachments {
		badge, err := am.GetBadge(attachment.BadgeId)
		if err != nil {
			continue // Skip if badge not found
		}

		if badge.BadgeType == "property" {
			properties = append(properties, badge)
		}
	}

	return properties, nil
}

// GetAgentCapabilities retrieves all capability badges attached to a specific agent
func (am *AgentManager) GetAgentCapabilities(agentID string) ([]*Badge, error) {
	attachments, err := am.chromeManager.GetBadgeAttachments(agentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get badge attachments for agent: %v", err)
	}

	var capabilities []*Badge
	for _, attachment := range attachments {
		badge, err := am.GetBadge(attachment.BadgeId)
		if err != nil {
			continue // Skip if badge not found
		}

		if badge.BadgeType == "capability" {
			capabilities = append(capabilities, badge)
		}
	}

	return capabilities, nil
}

// CreateBadgeBundle creates a bundle of related Badges
func (am *AgentManager) CreateBadgeBundle(name, description, owner string, badgeIds []string) (string, error) {
	// Create metadata for the bundle
	metadata := map[string]interface{}{
		"bundle_type": "badge_bundle",
		"badge_ids":   badgeIds,
		"created_at":  time.Now(),
	}

	// Create the bundle as a special Badge
	bundle, err := am.CreateBadge(name, description, "", owner, "bundle", metadata)
	if err != nil {
		return "", fmt.Errorf("failed to create badge bundle: %v", err)
	}

	return bundle.ID, nil
}

// RegisterCapabilityAsBadge registers a capability as a Badge for Agents
func (am *AgentManager) RegisterCapabilityAsBadge(name, description, imageURL, owner string, capabilityType string, schema map[string]interface{}, locationHints []string) (*Badge, error) {
	// Create metadata for the capability
	metadata := map[string]interface{}{
		"capability_type": capabilityType,
		"schema":          schema,
		"location_hints":  locationHints,
	}

	// Extract gas fee from schema if present
	if gasFee, exists := schema["gas_fee_nrn"]; exists {
		metadata["gas_fee_nrn"] = gasFee
	}

	// Create the Badge with badge type "capability"
	return am.CreateBadge(name, description, imageURL, owner, "capability", metadata)
}

// RegisterSkillAsBadge registers a skill as a Badge for Agents
func (am *AgentManager) RegisterSkillAsBadge(name, description, imageURL, owner string, skillType string, parameters map[string]interface{}, requirements []string) (*Badge, error) {
	// Create metadata for the skill
	metadata := map[string]interface{}{
		"skill_type":    skillType,
		"parameters":    parameters,
		"requirements":  requirements,
		"execution_env": "knirvchain", // Skills are executed on KNIRVCHAIN
	}

	// Extract execution cost from parameters if present
	if executionCost, exists := parameters["execution_cost_nrn"]; exists {
		metadata["execution_cost_nrn"] = executionCost
	}

	// Extract skill complexity level
	if complexity, exists := parameters["complexity_level"]; exists {
		metadata["complexity_level"] = complexity
	}

	// Create the Badge with badge type "skill"
	return am.CreateBadge(name, description, imageURL, owner, "skill", metadata)
}

// RegisterPropertyAsBadge registers a property as a Badge for Agents
func (am *AgentManager) RegisterPropertyAsBadge(name, description, imageURL, owner string, propertyType string, value interface{}, constraints map[string]interface{}) (*Badge, error) {
	// Create metadata for the property
	metadata := map[string]interface{}{
		"property_type": propertyType,
		"value":         value,
		"constraints":   constraints,
		"immutable":     true, // Properties are typically immutable characteristics
	}

	// Add validation rules if present
	if validationRules, exists := constraints["validation_rules"]; exists {
		metadata["validation_rules"] = validationRules
	}

	// Add property category for organization
	if category, exists := constraints["category"]; exists {
		metadata["category"] = category
	}

	// Create the Badge with badge type "property"
	return am.CreateBadge(name, description, imageURL, owner, "property", metadata)
}

// CreateCapabilityBundle creates a bundle of related capability Badges
func (am *AgentManager) CreateCapabilityBundle(name, description, owner string, capabilityBadgeIds []string) (string, error) {
	return am.CreateBadgeBundle(name, description, owner, capabilityBadgeIds)
}

// ===== CAPABILITY INVOCATION METHODS =====

// InvokeCapabilityBadge invokes a capability Badge attached to an Agent
func (am *AgentManager) InvokeCapabilityBadge(agentId, badgeId string, input map[string]interface{}) (*types.ContextRecord, error) {
	// Get the Agent
	agent, err := am.GetAgent(agentId)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent: %v", err)
	}

	// Get the Badge
	badge, err := am.GetBadge(badgeId)
	if err != nil {
		return nil, fmt.Errorf("failed to get badge: %v", err)
	}

	// Check if the Badge is a capability
	if badge.BadgeType != "capability" {
		return nil, fmt.Errorf("badge %s is not a capability badge", badgeId)
	}

	// Check if the Badge is attached to the Agent
	attachment, err := am.chromeManager.GetBadgeAttachmentByAgentAndBadge(agentId, badgeId)
	if err != nil {
		return nil, fmt.Errorf("badge %s is not attached to agent %s: %v", badgeId, agentId, err)
	}

	// Verify the attachment is active
	if attachment.Status != "active" {
		return nil, fmt.Errorf("badge attachment is not active: %s", attachment.Status)
	}

	// Extract capability metadata
	capabilityType, ok := badge.Metadata["capability_type"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid capability_type in badge metadata")
	}

	schema, ok := badge.Metadata["schema"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid schema in badge metadata")
	}

	locationHints, ok := badge.Metadata["location_hints"].([]string)
	if !ok {
		// Try to convert from []interface{} to []string
		if locationHintsInterface, exists := badge.Metadata["location_hints"].([]interface{}); exists {
			locationHints = make([]string, len(locationHintsInterface))
			for i, hint := range locationHintsInterface {
				if hintStr, ok := hint.(string); ok {
					locationHints[i] = hintStr
				} else {
					return nil, fmt.Errorf("invalid location_hints format in badge metadata")
				}
			}
		} else {
			return nil, fmt.Errorf("missing location_hints in badge metadata")
		}
	}

	// Get gas fee from metadata
	var gasFeeNRN uint64
	if gasFee, exists := badge.Metadata["gas_fee_nrn"]; exists {
		switch v := gasFee.(type) {
		case float64:
			gasFeeNRN = uint64(v)
		case uint64:
			gasFeeNRN = v
		case int:
			gasFeeNRN = uint64(v)
		default:
			return nil, fmt.Errorf("invalid gas_fee_nrn format in badge metadata")
		}
	}

	// Merge input with any default parameters from schema
	mergedInput := make(map[string]interface{})
	if defaultParams, exists := schema["default_params"]; exists {
		if defaultMap, ok := defaultParams.(map[string]interface{}); ok {
			for k, v := range defaultMap {
				mergedInput[k] = v
			}
		}
	}
	for k, v := range input {
		mergedInput[k] = v
	}

	// Create a context record
	contextRecord := &types.ContextRecord{
		ID:              generateUniqueID(),
		CapabilityID:    badgeId,
		InteractionType: types.InteractionTypeToolInvocation,
		Initiator:       am.wallet.GetAddress(),
		Timestamp:       time.Now().Unix(),
		InputHash:       am.hashInput(mergedInput),
		Details: map[string]interface{}{
			"capability_type":   capabilityType,
			"execution_context": "client",
			"agent_id":          agent.ID,
			"agent_type":        agent.AgentType,
			"badge_id":          badge.ID,
			"attachment_id":     attachment.ID,
		},
	}

	// Execute the capability (implementation depends on capability type)
	output, err := am.executeCapability(capabilityType, schema, locationHints, mergedInput)
	if err != nil {
		contextRecord.Status = "failed"
		contextRecord.Details["error"] = err.Error()
		am.storeContextRecord(contextRecord)
		return contextRecord, err
	}

	// Update the context record with output
	contextRecord.OutputHash = am.hashOutput(output)
	contextRecord.Status = "completed"
	contextRecord.Details["output_summary"] = am.summarizeOutput(output)

	// Store the context record
	am.storeContextRecord(contextRecord)

	// Process the NRN fee
	am.processNRNFee(badge.Owner, gasFeeNRN)

	return contextRecord, nil
}

// InvokeCapabilityBadgeByAgent invokes a capability Badge by finding the first active capability of the specified type attached to the Agent
func (am *AgentManager) InvokeCapabilityBadgeByAgent(agentId, capabilityType string, input map[string]interface{}) (*types.ContextRecord, error) {
	// Get the Agent (validate it exists)
	_, err := am.GetAgent(agentId)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent: %v", err)
	}

	// Get all badge attachments for the agent
	attachments, err := am.chromeManager.GetBadgeAttachments(agentId)
	if err != nil {
		return nil, fmt.Errorf("failed to get badge attachments for agent: %v", err)
	}

	// Find the first active capability Badge of the specified type
	var targetBadge *Badge
	for _, attachment := range attachments {
		if attachment.Status != "active" {
			continue
		}

		badge, err := am.GetBadge(attachment.BadgeId)
		if err != nil {
			continue // Skip if badge not found
		}

		if badge.BadgeType != "capability" {
			continue // Skip non-capability badges
		}

		badgeCapabilityType, ok := badge.Metadata["capability_type"].(string)
		if !ok {
			continue // Skip if capability_type not found
		}

		if badgeCapabilityType == capabilityType {
			targetBadge = badge
			break
		}
	}

	if targetBadge == nil {
		return nil, fmt.Errorf("no active capability badge of type %s found for agent %s", capabilityType, agentId)
	}

	// Invoke the found capability badge
	return am.InvokeCapabilityBadge(agentId, targetBadge.ID, input)
}

// generateBadgeAttachmentSignature generates a cryptographic signature for a badge attachment
func (am *AgentManager) generateBadgeAttachmentSignature(attachment *BadgeAttachment) (string, error) {
	// Create a canonical representation of the attachment for signing
	signatureData := fmt.Sprintf("%s:%s:%s:%s:%s",
		attachment.AgentId,
		attachment.BadgeId,
		attachment.AttachedAt.Format(time.RFC3339),
		attachment.AttachedBy,
		attachment.Status,
	)

	// Add parameters to signature data if they exist
	if attachment.Parameters != nil {
		parametersJSON, err := json.Marshal(attachment.Parameters)
		if err != nil {
			return "", fmt.Errorf("failed to marshal parameters: %v", err)
		}
		signatureData += ":" + string(parametersJSON)
	}

	// Generate hash of the signature data
	hash := sha256.Sum256([]byte(signatureData))

	// For now, use a simple hex encoding of the hash
	// In a full implementation, this would use the wallet's private key to sign
	signature := fmt.Sprintf("%x", hash)

	return signature, nil
}

// ===== HELPER METHODS FOR CAPABILITY INVOCATION =====

// hashInput creates a hash of the input data
func (am *AgentManager) hashInput(input map[string]interface{}) string {
	inputJSON, err := json.Marshal(input)
	if err != nil {
		return ""
	}
	hash := sha256.Sum256(inputJSON)
	return fmt.Sprintf("%x", hash)
}

// hashOutput creates a hash of the output data
func (am *AgentManager) hashOutput(output map[string]interface{}) string {
	outputJSON, err := json.Marshal(output)
	if err != nil {
		return ""
	}
	hash := sha256.Sum256(outputJSON)
	return fmt.Sprintf("%x", hash)
}

// summarizeOutput creates a summary of the output data
func (am *AgentManager) summarizeOutput(output map[string]interface{}) string {
	if len(output) == 0 {
		return "empty output"
	}

	summary := fmt.Sprintf("output with %d fields", len(output))
	if status, exists := output["status"]; exists {
		summary += fmt.Sprintf(", status: %v", status)
	}
	if result, exists := output["result"]; exists {
		summary += fmt.Sprintf(", result type: %T", result)
	}

	return summary
}

// executeCapability executes a capability based on its type
func (am *AgentManager) executeCapability(capabilityType string, _ map[string]interface{}, _ []string, input map[string]interface{}) (map[string]interface{}, error) {
	// This is a placeholder implementation
	// In a real system, this would:
	// 1. Download the capability from locationHints
	// 2. Verify the capability package integrity
	// 3. Execute the capability in a sandboxed environment
	// 4. Return the execution results

	output := map[string]interface{}{
		"status":          "completed",
		"capability_type": capabilityType,
		"execution_time":  time.Now().Unix(),
		"input_processed": len(input),
		"result":          "capability execution placeholder result",
	}

	// Simulate some processing based on capability type
	switch capabilityType {
	case "resource":
		output["resource_accessed"] = true
		output["resource_data"] = "simulated resource data"
	case "tool":
		output["tool_executed"] = true
		output["tool_result"] = "simulated tool execution result"
	case "plugin":
		output["plugin_loaded"] = true
		output["plugin_output"] = "simulated plugin output"
	default:
		output["generic_execution"] = true
	}

	return output, nil
}

// storeContextRecord stores a context record
func (am *AgentManager) storeContextRecord(contextRecord *types.ContextRecord) error {
	// Store the context record in the database
	// This would typically use the LevelDB or ChromeDB to store the record
	// For now, this is a placeholder implementation

	recordJSON, err := json.Marshal(contextRecord)
	if err != nil {
		return fmt.Errorf("failed to marshal context record: %v", err)
	}

	// In a real implementation, this would save to the database
	fmt.Printf("Storing context record: %s\n", string(recordJSON))

	return nil
}

// processNRNFee processes the NRN fee payment
func (am *AgentManager) processNRNFee(owner string, gasFeeNRN uint64) error {
	// Process the NRN fee payment
	// This would typically:
	// 1. Deduct the fee from the initiator's account
	// 2. Transfer the fee to the capability owner
	// 3. Record the transaction on the blockchain

	if gasFeeNRN == 0 {
		return nil // No fee to process
	}

	// For now, this is a placeholder implementation
	fmt.Printf("Processing NRN fee: %d NRN to owner %s\n", gasFeeNRN, owner)

	return nil
}

// recordBadgeAttachmentOnBlockchain records a badge attachment on the blockchain
func (am *AgentManager) recordBadgeAttachmentOnBlockchain(attachment *BadgeAttachment) (string, error) {
	// Placeholder implementation - in a real system this would:
	// 1. Create a blockchain transaction
	// 2. Include the attachment data in the transaction
	// 3. Submit to the blockchain network
	// 4. Return the transaction hash

	// For now, generate a mock transaction hash
	txData := fmt.Sprintf("badge_attachment:%s:%s:%s",
		attachment.ID,
		attachment.AgentId,
		attachment.BadgeId,
	)
	hash := sha256.Sum256([]byte(txData))
	mockTxHash := fmt.Sprintf("0x%x", hash)[:42] // Mock blockchain transaction hash format

	return mockTxHash, nil
}

// VerifyBadgeAttachmentSignature verifies the cryptographic signature of a badge attachment
func (am *AgentManager) VerifyBadgeAttachmentSignature(attachment *BadgeAttachment) (bool, error) {
	// Regenerate the signature using the same method
	expectedSignature, err := am.generateBadgeAttachmentSignature(attachment)
	if err != nil {
		return false, fmt.Errorf("failed to generate expected signature: %v", err)
	}

	// Compare with stored signature
	return attachment.Signature == expectedSignature, nil
}

// VerifyBadgeAttachmentBlockchain verifies a badge attachment against blockchain records
func (am *AgentManager) VerifyBadgeAttachmentBlockchain(attachment *BadgeAttachment) (bool, error) {
	// Placeholder implementation - in a real system this would:
	// 1. Query the blockchain using the transaction hash
	// 2. Verify the transaction exists and contains the attachment data
	// 3. Return verification result

	if attachment.TxHash == "" {
		return false, fmt.Errorf("no blockchain transaction hash available")
	}

	// For now, assume verification passes if we have a transaction hash
	// In a real implementation, this would make an actual blockchain query
	return true, nil
}

// GetBadgeAttachments retrieves all badge attachments for an Agent
func (am *AgentManager) GetBadgeAttachments(agentID string) ([]*BadgeAttachment, error) {
	return am.chromeManager.GetBadgeAttachments(agentID)
}

// GetBadgeAttachment retrieves a specific badge attachment by ID
func (am *AgentManager) GetBadgeAttachment(attachmentID string) (*BadgeAttachment, error) {
	return am.chromeManager.GetBadgeAttachment(attachmentID)
}

// ValidateBadgeAttachment validates the integrity of a badge attachment
func (am *AgentManager) ValidateBadgeAttachment(attachment *BadgeAttachment) error {
	// Verify cryptographic signature
	isValidSignature, err := am.VerifyBadgeAttachmentSignature(attachment)
	if err != nil {
		return fmt.Errorf("failed to verify signature: %v", err)
	}
	if !isValidSignature {
		return fmt.Errorf("invalid cryptographic signature")
	}

	// Verify blockchain record if available
	if attachment.TxHash != "" {
		isValidBlockchain, err := am.VerifyBadgeAttachmentBlockchain(attachment)
		if err != nil {
			return fmt.Errorf("failed to verify blockchain record: %v", err)
		}
		if !isValidBlockchain {
			return fmt.Errorf("invalid blockchain verification")
		}
	}

	return nil
}

// RegisterAgentPlugin registers a compiled WASM plugin for an Agent
func (am *AgentManager) RegisterAgentPlugin(agentID, filePath, version string, compiledBy string) (*AgentPlugin, error) {
	// Verify the Agent exists
	agent, err := am.GetAgent(agentID)
	if err != nil {
		return nil, fmt.Errorf("agent not found: %v", err)
	}

	// Verify the plugin file exists and is valid
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("plugin file does not exist: %v", err)
	}

	// Calculate hash of the plugin file
	pluginData, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read plugin file: %v", err)
	}

	hash := fmt.Sprintf("%x", sha256.Sum256(pluginData))

	// Create plugin record
	plugin := &AgentPlugin{
		ID:          generateUniqueID(),
		AgentId:     agentID,
		FilePath:    filePath,
		Hash:        hash,
		Version:     version,
		CompileTime: time.Now(),
		CompiledBy:  compiledBy,
		ServerURLs:  []string{},
		Status:      "active",
	}

	// Store plugin information in the database
	err = am.chromeManager.StoreAgentPlugin(plugin)
	if err != nil {
		return nil, fmt.Errorf("failed to store plugin: %v", err)
	}

	// Update Agent with plugin reference
	if agent.PluginPath == "" {
		agent.PluginPath = filePath
		err = am.chromeManager.StoreAgent(agent)
		if err != nil {
			return nil, fmt.Errorf("failed to update agent: %v", err)
		}
	}

	return plugin, nil
}

// GetAgentPlugins retrieves all plugins for an Agent
func (am *AgentManager) GetAgentPlugins(agentID string) ([]*AgentPlugin, error) {
	return am.chromeManager.GetAgentPlugins(agentID)
}

// GetAgentPlugin retrieves a specific plugin by ID
func (am *AgentManager) GetAgentPlugin(pluginID string) (*AgentPlugin, error) {
	return am.chromeManager.GetAgentPlugin(pluginID)
}

// UpdateAgentPluginStatus updates the status of an agent plugin
func (am *AgentManager) UpdateAgentPluginStatus(pluginID, status string) error {
	plugin, err := am.GetAgentPlugin(pluginID)
	if err != nil {
		return fmt.Errorf("plugin not found: %v", err)
	}

	plugin.Status = status
	return am.chromeManager.StoreAgentPlugin(plugin)
}

// AddServerURLToPlugin adds a server URL to an agent plugin
func (am *AgentManager) AddServerURLToPlugin(pluginID, serverURL string) error {
	plugin, err := am.GetAgentPlugin(pluginID)
	if err != nil {
		return fmt.Errorf("plugin not found: %v", err)
	}

	// Check if URL already exists
	for _, url := range plugin.ServerURLs {
		if url == serverURL {
			return nil // Already exists
		}
	}

	plugin.ServerURLs = append(plugin.ServerURLs, serverURL)
	return am.chromeManager.StoreAgentPlugin(plugin)
}

// RemoveServerURLFromPlugin removes a server URL from an agent plugin
func (am *AgentManager) RemoveServerURLFromPlugin(pluginID, serverURL string) error {
	plugin, err := am.GetAgentPlugin(pluginID)
	if err != nil {
		return fmt.Errorf("plugin not found: %v", err)
	}

	// Find and remove the URL
	for i, url := range plugin.ServerURLs {
		if url == serverURL {
			plugin.ServerURLs = append(plugin.ServerURLs[:i], plugin.ServerURLs[i+1:]...)
			break
		}
	}

	return am.chromeManager.StoreAgentPlugin(plugin)
}

// VerifyAgentPlugin verifies the integrity of an agent plugin
func (am *AgentManager) VerifyAgentPlugin(pluginID string) (bool, error) {
	plugin, err := am.GetAgentPlugin(pluginID)
	if err != nil {
		return false, fmt.Errorf("plugin not found: %v", err)
	}

	// Verify the plugin file still exists
	if _, err := os.Stat(plugin.FilePath); os.IsNotExist(err) {
		return false, fmt.Errorf("plugin file no longer exists: %s", plugin.FilePath)
	}

	// Recalculate hash and compare
	pluginData, err := os.ReadFile(plugin.FilePath)
	if err != nil {
		return false, fmt.Errorf("failed to read plugin file: %v", err)
	}

	currentHash := fmt.Sprintf("%x", sha256.Sum256(pluginData))
	if currentHash != plugin.Hash {
		return false, fmt.Errorf("plugin file hash mismatch: expected %s, got %s", plugin.Hash, currentHash)
	}

	return true, nil
}

// SetDiscoveryManager sets the discovery manager for the agent manager
func (am *AgentManager) SetDiscoveryManager(discoveryManager *DiscoveryManager) {
	am.discoveryManager = discoveryManager
}

// SetDiscoveryManager sets the discovery manager for the NFT manager (legacy wrapper)
func (nm *NFTManager) SetDiscoveryManager(discoveryManager *DiscoveryManager) {
	nm.agentManager.SetDiscoveryManager(discoveryManager)
}

// ===== RESOURCE CAPABILITY METHODS (Phase 3) =====

// validateResourceCapabilityInput validates the input parameters for resource capability creation
func (am *AgentManager) validateResourceCapabilityInput(agentId, name, _ string, resourceType string, metadata map[string]interface{}) error {
	// Validate required fields
	if agentId == "" {
		return fmt.Errorf("agent ID is required")
	}
	if name == "" {
		return fmt.Errorf("resource capability name is required")
	}
	if resourceType == "" {
		return fmt.Errorf("resource type is required")
	}

	// Validate resource type
	validResourceTypes := map[string]bool{
		"FILE":           true,
		"API":            true,
		"PLUGIN":         true,
		"DATASET":        true,
		"MODEL_ARTIFACT": true,
		"SERVICE":        true,
	}

	if !validResourceTypes[resourceType] {
		return fmt.Errorf("unsupported resource type: %s", resourceType)
	}

	// Validate resource-specific metadata
	return am.validateResourceMetadata(resourceType, metadata)
}

// validateResourceMetadata validates metadata based on resource type
func (am *AgentManager) validateResourceMetadata(resourceType string, metadata map[string]interface{}) error {
	switch resourceType {
	case "FILE":
		if _, ok := metadata["file_path"]; !ok {
			return fmt.Errorf("file_path is required for FILE resource type")
		}
		if _, ok := metadata["access_method"]; !ok {
			return fmt.Errorf("access_method is required for FILE resource type")
		}
	case "API":
		if _, ok := metadata["api_endpoint"]; !ok {
			return fmt.Errorf("api_endpoint is required for API resource type")
		}
		if _, ok := metadata["access_method"]; !ok {
			return fmt.Errorf("access_method is required for API resource type")
		}
	case "DATASET":
		if _, ok := metadata["dataset_path"]; !ok {
			return fmt.Errorf("dataset_path is required for DATASET resource type")
		}
		if _, ok := metadata["data_format"]; !ok {
			return fmt.Errorf("data_format is required for DATASET resource type")
		}
	case "MODEL_ARTIFACT":
		if _, ok := metadata["model_path"]; !ok {
			return fmt.Errorf("model_path is required for MODEL_ARTIFACT resource type")
		}
		if _, ok := metadata["model_type"]; !ok {
			return fmt.Errorf("model_type is required for MODEL_ARTIFACT resource type")
		}
	case "SERVICE":
		if _, ok := metadata["service_url"]; !ok {
			return fmt.Errorf("service_url is required for SERVICE resource type")
		}
		if _, ok := metadata["service_type"]; !ok {
			return fmt.Errorf("service_type is required for SERVICE resource type")
		}
	}
	return nil
}

// AddResourceCapabilityToAgent adds a resource capability to an Agent
func (am *AgentManager) AddResourceCapabilityToAgent(agentId, name, description, resourceType string, metadata map[string]interface{}) (*Capability, error) {
	// Validate input parameters
	if err := am.validateResourceCapabilityInput(agentId, name, description, resourceType, metadata); err != nil {
		return nil, err
	}

	// Verify the Agent exists
	_, err := am.GetAgent(agentId)
	if err != nil {
		return nil, fmt.Errorf("agent not found: %v", err)
	}

	// Create metadata for the resource
	resourceMetadata := map[string]interface{}{
		"resource_type": resourceType,
		"access_method": metadata["access_method"],
		"data_format":   metadata["data_format"],
		"size_bytes":    metadata["size_bytes"],
	}

	// Merge with provided metadata
	for k, v := range metadata {
		if _, exists := resourceMetadata[k]; !exists {
			resourceMetadata[k] = v
		}
	}

	// Extract required fields from metadata with defaults
	accessURL := ""
	if url, ok := metadata["access_url"].(string); ok {
		accessURL = url
	}

	schema := make(map[string]interface{})
	if s, ok := metadata["schema"].(map[string]interface{}); ok {
		schema = s
	}

	locationHints := []string{}
	if hints, ok := metadata["location_hints"].([]string); ok {
		locationHints = hints
	}

	// Create the capability with type "resource"
	capability := Capability{
		Name:           name,
		Description:    description,
		CapabilityType: "resource",
		MCPServerURL:   accessURL,
		Schema:         schema,
		LocationHints:  locationHints,
		Metadata:       resourceMetadata,
	}

	// Add the capability to the agent
	err = am.AddCapabilityToAgent(agentId, capability)
	if err != nil {
		return nil, fmt.Errorf("failed to add resource capability: %v", err)
	}

	// Get the agent to retrieve the added capability with its generated ID
	agent, err := am.GetAgent(agentId)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent after adding capability: %v", err)
	}

	// Find the capability we just added (it should be the last one)
	if len(agent.Capabilities) == 0 {
		return nil, fmt.Errorf("no capabilities found after adding resource capability")
	}

	addedCapability := &agent.Capabilities[len(agent.Capabilities)-1]
	return addedCapability, nil
}

// LinkResourceToCapability creates a relationship between a resource capability and a functional capability
func (am *AgentManager) LinkResourceToCapability(agentId, resourceCapabilityId, functionalCapabilityId string, parameters map[string]interface{}) error {
	// Verify the Agent exists
	agent, err := am.GetAgent(agentId)
	if err != nil {
		return fmt.Errorf("agent not found: %v", err)
	}

	// Find both capabilities in the Agent
	var resourceCapability *Capability
	var functionalCapability *Capability

	for i, cap := range agent.Capabilities {
		switch cap.ID {
		case resourceCapabilityId:
			resourceCapability = &agent.Capabilities[i]
		case functionalCapabilityId:
			functionalCapability = &agent.Capabilities[i]
		}
	}

	if resourceCapability == nil {
		return fmt.Errorf("resource capability not found in agent")
	}

	if functionalCapability == nil {
		return fmt.Errorf("functional capability not found in agent")
	}

	// Verify the capability types
	if resourceCapability.CapabilityType != "resource" {
		return fmt.Errorf("capability is not a resource capability")
	}

	// Update the functional capability's metadata to reference the resource
	if functionalCapability.Metadata["required_resources"] == nil {
		functionalCapability.Metadata["required_resources"] = []string{resourceCapabilityId}
	} else {
		resources := functionalCapability.Metadata["required_resources"].([]string)
		functionalCapability.Metadata["required_resources"] = append(resources, resourceCapabilityId)
	}

	// Add resource-specific parameters to the capability
	if functionalCapability.Metadata["resource_parameters"] == nil {
		functionalCapability.Metadata["resource_parameters"] = map[string]map[string]interface{}{
			resourceCapabilityId: parameters,
		}
	} else {
		resourceParams := functionalCapability.Metadata["resource_parameters"].(map[string]map[string]interface{})
		resourceParams[resourceCapabilityId] = parameters
		functionalCapability.Metadata["resource_parameters"] = resourceParams
	}

	// Update the Agent in the database
	err = am.chromeManager.StoreAgent(agent)
	if err != nil {
		return fmt.Errorf("failed to update agent: %v", err)
	}

	// Create revision
	revision := &Revision{
		ID:         generateUniqueID(),
		AgentId:    agent.ID,
		Timestamp:  time.Now(),
		ChangedBy:  am.wallet.GetAddress(),
		ChangeType: "link_resource_to_capability",
		Details: map[string]interface{}{
			"resource_capability_id":   resourceCapabilityId,
			"functional_capability_id": functionalCapabilityId,
			"parameters":               parameters,
		},
	}

	err = am.chromeManager.StoreRevision(revision)
	if err != nil {
		// Log error but don't fail the operation
		fmt.Printf("Failed to store revision: %v\n", err)
	}

	return nil
}

// CreateResourceCapabilityGroup creates a group of related resource capabilities for an Agent
func (am *AgentManager) CreateResourceCapabilityGroup(agentId, name, description string, resourceCapabilityIds []string) error {
	// Verify the Agent exists
	agent, err := am.GetAgent(agentId)
	if err != nil {
		return fmt.Errorf("agent not found: %v", err)
	}

	// Verify all resource capabilities exist in the Agent
	for _, capId := range resourceCapabilityIds {
		found := false
		for _, cap := range agent.Capabilities {
			if cap.ID == capId && cap.CapabilityType == "resource" {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("resource capability %s not found in agent or is not a resource type", capId)
		}
	}

	// Create a capability group in the Agent's metadata
	if agent.Metadata == nil {
		agent.Metadata = make(map[string]interface{})
	}

	var capabilityGroups []map[string]interface{}
	if existingGroups, ok := agent.Metadata["capability_groups"].([]map[string]interface{}); ok {
		capabilityGroups = existingGroups
	}

	// Create the new group
	group := map[string]interface{}{
		"id":                      generateUniqueID(),
		"name":                    name,
		"description":             description,
		"type":                    "resource_group",
		"resource_capability_ids": resourceCapabilityIds,
		"created_at":              time.Now(),
		"created_by":              am.wallet.GetAddress(),
	}

	capabilityGroups = append(capabilityGroups, group)
	agent.Metadata["capability_groups"] = capabilityGroups

	// Update the Agent in the database
	err = am.chromeManager.StoreAgent(agent)
	if err != nil {
		return fmt.Errorf("failed to update agent: %v", err)
	}

	// Create revision
	revision := &Revision{
		ID:         generateUniqueID(),
		AgentId:    agent.ID,
		Timestamp:  time.Now(),
		ChangedBy:  am.wallet.GetAddress(),
		ChangeType: "create_resource_capability_group",
		Details: map[string]interface{}{
			"group_name":              name,
			"resource_capability_ids": resourceCapabilityIds,
		},
	}

	err = am.chromeManager.StoreRevision(revision)
	if err != nil {
		// Log error but don't fail the operation
		fmt.Printf("Failed to store revision: %v\n", err)
	}

	return nil
}

// GetResourceCapabilities retrieves all resource capabilities for an Agent
func (am *AgentManager) GetResourceCapabilities(agentId string) ([]*Capability, error) {
	agent, err := am.GetAgent(agentId)
	if err != nil {
		return nil, fmt.Errorf("agent not found: %v", err)
	}

	var resourceCapabilities []*Capability
	for i, cap := range agent.Capabilities {
		if cap.CapabilityType == "resource" {
			resourceCapabilities = append(resourceCapabilities, &agent.Capabilities[i])
		}
	}

	return resourceCapabilities, nil
}

// GetResourceCapabilityGroups retrieves all resource capability groups for an Agent
func (am *AgentManager) GetResourceCapabilityGroups(agentId string) ([]map[string]interface{}, error) {
	agent, err := am.GetAgent(agentId)
	if err != nil {
		return nil, fmt.Errorf("agent not found: %v", err)
	}

	if agent.Metadata == nil {
		return []map[string]interface{}{}, nil
	}

	// Handle both []interface{} and []map[string]interface{} types
	var groups []map[string]interface{}

	if groupsInterface, ok := agent.Metadata["capability_groups"].([]interface{}); ok {
		// Convert []interface{} to []map[string]interface{}
		for _, groupInterface := range groupsInterface {
			if group, ok := groupInterface.(map[string]interface{}); ok {
				groups = append(groups, group)
			}
		}
	} else if groupsMap, ok := agent.Metadata["capability_groups"].([]map[string]interface{}); ok {
		groups = groupsMap
	} else {
		return []map[string]interface{}{}, nil
	}

	// Filter for resource groups only
	var resourceGroups []map[string]interface{}
	for _, group := range groups {
		if groupType, ok := group["type"].(string); ok && groupType == "resource_group" {
			resourceGroups = append(resourceGroups, group)
		}
	}
	return resourceGroups, nil
}

// AnnounceResourceCapabilityToDHT announces a resource capability to the DHT for discovery
func (am *AgentManager) AnnounceResourceCapabilityToDHT(agentId, capabilityId string) error {
	// Get the agent and capability
	agent, err := am.GetAgent(agentId)
	if err != nil {
		return fmt.Errorf("agent not found: %v", err)
	}

	var capability *Capability
	for i, cap := range agent.Capabilities {
		if cap.ID == capabilityId && cap.CapabilityType == "resource" {
			capability = &agent.Capabilities[i]
			break
		}
	}

	if capability == nil {
		return fmt.Errorf("resource capability not found")
	}

	// Get the resource type from capability metadata
	resourceTypeStr, ok := capability.Metadata["resource_type"].(string)
	if !ok {
		return fmt.Errorf("resource capability missing resource_type metadata")
	}

	// Convert to DiscoveryResourceType
	var resourceType types.DiscoveryResourceType
	switch resourceTypeStr {
	case "FILE":
		resourceType = types.DiscoveryResourceTypeFile
	case "API":
		resourceType = types.DiscoveryResourceTypeAPI
	case "PLUGIN":
		resourceType = types.DiscoveryResourceTypePlugin
	case "DATASET":
		resourceType = types.DiscoveryResourceTypeDataset
	case "MODEL_ARTIFACT":
		resourceType = types.DiscoveryResourceTypeModelArtifact
	case "SERVICE":
		resourceType = types.DiscoveryResourceTypeService
	default:
		return fmt.Errorf("unsupported resource type: %s", resourceTypeStr)
	}

	// Create a unique resource ID combining agent ID and capability ID
	resourceId := fmt.Sprintf("%s.%s", agentId, capabilityId)

	// Announce to DHT if discovery manager is available
	if am.discoveryManager != nil {
		err = am.discoveryManager.AnnounceGenericResource(resourceId, resourceType)
		if err != nil {
			return fmt.Errorf("failed to announce resource to DHT: %v", err)
		}
	}

	return nil
}

// AnnounceAllResourceCapabilitiesToDHT announces all resource capabilities of an agent to the DHT
func (am *AgentManager) AnnounceAllResourceCapabilitiesToDHT(agentId string) error {
	resourceCapabilities, err := am.GetResourceCapabilities(agentId)
	if err != nil {
		return fmt.Errorf("failed to get resource capabilities: %v", err)
	}

	var errors []string
	for _, capability := range resourceCapabilities {
		err := am.AnnounceResourceCapabilityToDHT(agentId, capability.ID)
		if err != nil {
			errors = append(errors, fmt.Sprintf("capability %s: %v", capability.ID, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to announce some resource capabilities: %s", strings.Join(errors, "; "))
	}

	return nil
}

// FindResourceCapabilitiesInDHT searches for resource capabilities of a specific type in the DHT
func (am *AgentManager) FindResourceCapabilitiesInDHT(resourceType string) ([]string, error) {
	// Convert to DiscoveryResourceType
	var dhtResourceType types.DiscoveryResourceType
	switch resourceType {
	case "FILE":
		dhtResourceType = types.DiscoveryResourceTypeFile
	case "API":
		dhtResourceType = types.DiscoveryResourceTypeAPI
	case "PLUGIN":
		dhtResourceType = types.DiscoveryResourceTypePlugin
	case "DATASET":
		dhtResourceType = types.DiscoveryResourceTypeDataset
	case "MODEL_ARTIFACT":
		dhtResourceType = types.DiscoveryResourceTypeModelArtifact
	case "SERVICE":
		dhtResourceType = types.DiscoveryResourceTypeService
	default:
		return nil, fmt.Errorf("unsupported resource type: %s", resourceType)
	}

	if am.discoveryManager == nil {
		return nil, fmt.Errorf("discovery manager not available")
	}

	// Search for resources of this type
	// Note: This is a simplified search - in practice, you might want to search for specific resource IDs
	// For now, we'll return an empty list as the DHT search would require specific resource IDs
	// The dhtResourceType variable is prepared for future use when implementing specific resource discovery
	_ = dhtResourceType // Suppress unused variable warning
	return []string{}, nil
}

// ===== RESOURCE CAPABILITY INVOCATION METHODS (Phase 3) =====

// InvokeResourceCapability invokes a resource capability with the provided parameters
func (am *AgentManager) InvokeResourceCapability(agentId, capabilityId string, parameters map[string]interface{}, initiator string) (map[string]interface{}, error) {
	// Get the agent
	agent, err := am.GetAgent(agentId)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent: %w", err)
	}

	// Find the resource capability
	var resourceCapability *Capability
	for i, cap := range agent.Capabilities {
		if cap.ID == capabilityId && cap.CapabilityType == "resource" {
			resourceCapability = &agent.Capabilities[i]
			break
		}
	}

	if resourceCapability == nil {
		return nil, fmt.Errorf("resource capability not found: %s", capabilityId)
	}

	// Validate access permissions
	if err := am.validateResourceAccess(agentId, capabilityId, initiator); err != nil {
		return nil, fmt.Errorf("access denied: %w", err)
	}

	// Get resource metadata
	resourceType, ok := resourceCapability.Metadata["resource_type"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid resource type in capability metadata")
	}

	// Invoke based on resource type
	switch resourceType {
	case "FILE":
		return am.invokeFileResource(resourceCapability, parameters)
	case "API":
		return am.invokeAPIResource(resourceCapability, parameters)
	case "DATASET":
		return am.invokeDatasetResource(resourceCapability, parameters)
	case "MODEL_ARTIFACT":
		return am.invokeModelArtifactResource(resourceCapability, parameters)
	case "SERVICE":
		return am.invokeServiceResource(resourceCapability, parameters)
	default:
		return nil, fmt.Errorf("unsupported resource type: %s", resourceType)
	}
}

// validateResourceAccess validates if the initiator has access to the resource capability
func (am *AgentManager) validateResourceAccess(agentId, _ string, initiator string) error {
	// Get the agent
	agent, err := am.GetAgent(agentId)
	if err != nil {
		return fmt.Errorf("failed to get agent: %w", err)
	}

	// Check if the initiator is the agent owner
	if agent.Owner == initiator {
		return nil // Owner has full access
	}

	// Check access controls
	for _, ac := range agent.AccessControls {
		if ac.AccessValue == initiator {
			// Check if the access control allows resource capability access
			for _, permission := range ac.Permissions {
				if permission == "invoke_resource_capabilities" || permission == "all" {
					return nil
				}
			}
		}
	}

	return fmt.Errorf("insufficient permissions for user %s", initiator)
}

// invokeFileResource handles file resource invocation
func (am *AgentManager) invokeFileResource(capability *Capability, _ map[string]interface{}) (map[string]interface{}, error) {
	// Get file path from metadata
	filePath, ok := capability.Metadata["file_path"].(string)
	if !ok {
		return nil, fmt.Errorf("file_path not found in capability metadata")
	}

	// Get access method
	accessMethod, ok := capability.Metadata["access_method"].(string)
	if !ok {
		accessMethod = "read" // Default to read access
	}

	result := map[string]interface{}{
		"resource_type": "FILE",
		"file_path":     filePath,
		"access_method": accessMethod,
		"capability_id": capability.ID,
		"invoked_at":    time.Now().Unix(),
	}

	// Add file metadata if available
	if size, ok := capability.Metadata["size_bytes"].(float64); ok {
		result["size_bytes"] = size
	}
	if format, ok := capability.Metadata["data_format"].(string); ok {
		result["data_format"] = format
	}

	return result, nil
}

// invokeAPIResource handles API resource invocation
func (am *AgentManager) invokeAPIResource(capability *Capability, parameters map[string]interface{}) (map[string]interface{}, error) {
	// Get API endpoint from metadata
	endpoint, ok := capability.Metadata["api_endpoint"].(string)
	if !ok {
		return nil, fmt.Errorf("api_endpoint not found in capability metadata")
	}

	// Get access method
	accessMethod, ok := capability.Metadata["access_method"].(string)
	if !ok {
		accessMethod = "GET" // Default to GET method
	}

	result := map[string]interface{}{
		"resource_type": "API",
		"api_endpoint":  endpoint,
		"access_method": accessMethod,
		"capability_id": capability.ID,
		"invoked_at":    time.Now().Unix(),
		"parameters":    parameters,
	}

	// Add authentication info if available
	if authType, ok := capability.Metadata["auth_type"].(string); ok {
		result["auth_type"] = authType
	}

	return result, nil
}

// invokeDatasetResource handles dataset resource invocation
func (am *AgentManager) invokeDatasetResource(capability *Capability, parameters map[string]interface{}) (map[string]interface{}, error) {
	// Get dataset path from metadata
	datasetPath, ok := capability.Metadata["dataset_path"].(string)
	if !ok {
		return nil, fmt.Errorf("dataset_path not found in capability metadata")
	}

	result := map[string]interface{}{
		"resource_type": "DATASET",
		"dataset_path":  datasetPath,
		"capability_id": capability.ID,
		"invoked_at":    time.Now().Unix(),
		"parameters":    parameters,
	}

	// Add dataset metadata if available
	if size, ok := capability.Metadata["size_bytes"].(float64); ok {
		result["size_bytes"] = size
	}
	if format, ok := capability.Metadata["data_format"].(string); ok {
		result["data_format"] = format
	}
	if schema, ok := capability.Metadata["schema"].(map[string]interface{}); ok {
		result["schema"] = schema
	}

	return result, nil
}

// invokeModelArtifactResource handles model artifact resource invocation
func (am *AgentManager) invokeModelArtifactResource(capability *Capability, parameters map[string]interface{}) (map[string]interface{}, error) {
	// Get model path from metadata
	modelPath, ok := capability.Metadata["model_path"].(string)
	if !ok {
		return nil, fmt.Errorf("model_path not found in capability metadata")
	}

	result := map[string]interface{}{
		"resource_type": "MODEL_ARTIFACT",
		"model_path":    modelPath,
		"capability_id": capability.ID,
		"invoked_at":    time.Now().Unix(),
		"parameters":    parameters,
	}

	// Add model metadata if available
	if modelType, ok := capability.Metadata["model_type"].(string); ok {
		result["model_type"] = modelType
	}
	if version, ok := capability.Metadata["version"].(string); ok {
		result["version"] = version
	}
	if framework, ok := capability.Metadata["framework"].(string); ok {
		result["framework"] = framework
	}

	return result, nil
}

// invokeServiceResource handles service resource invocation
func (am *AgentManager) invokeServiceResource(capability *Capability, parameters map[string]interface{}) (map[string]interface{}, error) {
	// Get service URL from metadata
	serviceURL, ok := capability.Metadata["service_url"].(string)
	if !ok {
		return nil, fmt.Errorf("service_url not found in capability metadata")
	}

	result := map[string]interface{}{
		"resource_type": "SERVICE",
		"service_url":   serviceURL,
		"capability_id": capability.ID,
		"invoked_at":    time.Now().Unix(),
		"parameters":    parameters,
	}

	// Add service metadata if available
	if serviceType, ok := capability.Metadata["service_type"].(string); ok {
		result["service_type"] = serviceType
	}
	if protocol, ok := capability.Metadata["protocol"].(string); ok {
		result["protocol"] = protocol
	}

	return result, nil
}

// CreateResourceCapabilityInvocationTransaction creates an MCP transaction for resource capability invocation
func (am *AgentManager) CreateResourceCapabilityInvocationTransaction(agentId, capabilityId, initiator string, parameters map[string]interface{}) (*Transaction, error) {
	// Get the agent
	agent, err := am.GetAgent(agentId)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent: %w", err)
	}

	// Find the resource capability
	var resourceCapability *Capability
	for i, cap := range agent.Capabilities {
		if cap.ID == capabilityId && cap.CapabilityType == "resource" {
			resourceCapability = &agent.Capabilities[i]
			break
		}
	}

	if resourceCapability == nil {
		return nil, fmt.Errorf("resource capability not found: %s", capabilityId)
	}

	// Create context record for the invocation
	contextRecord := types.ContextRecord{
		ID:              fmt.Sprintf("ctx:%s", generateUniqueID()),
		CapabilityID:    capabilityId,
		InteractionType: "InvokeResource",
		Initiator:       initiator,
		Timestamp:       time.Now().Unix(),
		Details: map[string]interface{}{
			"agent_id":      agentId,
			"parameters":    parameters,
			"resource_type": resourceCapability.Metadata["resource_type"],
		},
	}

	// Create MCP invoke capability data
	invokeData := types.MCPInvokeCapabilityData{
		ContextRecord: contextRecord,
	}

	// Marshal the invoke data
	data, err := json.Marshal(invokeData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal invoke data: %w", err)
	}

	// Create the transaction
	transaction := &Transaction{
		TransactionHash: generateUniqueID(),
		From:            initiator,
		To:              agentId,
		Type:            TransactionTypeMCPInvokeCapability,
		Data:            data,
		Fee:             1, // Minimal fee for resource access
		Timestamp:       time.Now().Unix(),
	}

	return transaction, nil
}

// ProcessResourceCapabilityInvocation processes a resource capability invocation through MCP
func (am *AgentManager) ProcessResourceCapabilityInvocation(transaction *Transaction) (map[string]interface{}, error) {
	// Parse the MCP invoke data
	var invokeData types.MCPInvokeCapabilityData
	if err := json.Unmarshal(transaction.Data, &invokeData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal invoke data: %w", err)
	}

	// Extract parameters from context record details
	agentId, ok := invokeData.ContextRecord.Details["agent_id"].(string)
	if !ok {
		return nil, fmt.Errorf("agent_id not found in context record details")
	}

	parameters, ok := invokeData.ContextRecord.Details["parameters"].(map[string]interface{})
	if !ok {
		parameters = make(map[string]interface{}) // Default to empty parameters
	}

	// Invoke the resource capability
	result, err := am.InvokeResourceCapability(
		agentId,
		invokeData.ContextRecord.CapabilityID,
		parameters,
		invokeData.ContextRecord.Initiator,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to invoke resource capability: %w", err)
	}

	// Add transaction metadata to result
	result["transaction_hash"] = transaction.TransactionHash
	result["processed_at"] = time.Now().Unix()

	return result, nil
}

// GetResourceCapabilityInvocationHistory gets the invocation history for a resource capability
func (am *AgentManager) GetResourceCapabilityInvocationHistory(agentId, capabilityId string) ([]map[string]interface{}, error) {
	// This would typically query the database for historical invocations
	// For now, return an empty list as a placeholder
	history := []map[string]interface{}{}

	// In a real implementation, this would:
	// 1. Query the database for context records related to this capability
	// 2. Filter by interaction type "InvokeResource"
	// 3. Return the historical data

	return history, nil
}

// Helper function to generate a unique ID
func generateUniqueID() string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(fmt.Sprintf("%d", time.Now().UnixNano()))))[:16]
}

// GetChromeManager returns the chrome manager for testing purposes
func (am *AgentManager) GetChromeManager() *ChromemManager {
	return am.chromeManager
}
