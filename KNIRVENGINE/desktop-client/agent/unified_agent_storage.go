package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/philippgille/chromem-go"
)

// UnifiedAgent represents a complete agent record that combines both registry and database needs
type UnifiedAgent struct {
	// Core agent fields (used by both systems)
	ID        string    `json:"id" validate:"required,uuid"`
	Name      string    `json:"name" validate:"required,min=1,max=100"`
	Type      string    `json:"type" validate:"required,agent_type"`
	OwnerID   int64     `json:"owner_id" validate:"required,gte=1"`
	CreatedAt time.Time `json:"created_at" validate:"required"`
	UpdatedAt time.Time `json:"updated_at" validate:"required"`

	// Agent configuration (used by agent builder/registry)
	AgentConfig map[string]interface{} `json:"agent_config" validate:"required"`

	// Frontend/UI fields (used by frontend)
	Collection   string   `json:"collection" validate:"max=100"`
	ImageURL     string   `json:"image_url" validate:"omitempty,url"`
	Status       string   `json:"status" validate:"required,status"`
	Capabilities []string `json:"capabilities" validate:"dive,capability"`
	TargetTypes  []string `json:"target_types" validate:"dive,min=1,max=50"`

	// Plugin/Build information
	PluginInfo *PluginInfo `json:"plugin_info,omitempty"`

	// API Keys (encrypted in production)
	APIKeys map[string]string `json:"api_keys"`
}

// PluginInfo contains information about the built plugin/WASM file
type PluginInfo struct {
	PluginID      string `json:"plugin_id" validate:"required,uuid"`
	PluginVersion string `json:"plugin_version" validate:"required,max=50"`
	PluginPath    string `json:"plugin_path" validate:"required,max=500"`
	Filename      string `json:"filename" validate:"required,max=255"`
	BuildTarget   string `json:"build_target" validate:"required,build_target"` // "plugin" or "wasm"
	Size          int64  `json:"size" validate:"gte=0"`
}

// UnifiedAgentStorage provides a single storage interface for all agent operations
type UnifiedAgentStorage struct {
	db         *chromem.DB
	collection *chromem.Collection
}

// createDeterministicEmbeddingFunction creates a deterministic embedding function for agent storage
// This ensures consistent embeddings without requiring external API calls
func createDeterministicEmbeddingFunction() func(context.Context, string) ([]float32, error) {
	return func(ctx context.Context, text string) ([]float32, error) {
		// Create a 384-dimensional embedding vector (standard size)
		embedding := make([]float32, 384)

		if len(text) == 0 {
			return embedding, nil
		}

		// Generate deterministic hash-based features
		charHash := uint64(0)
		wordHash := uint64(0)
		structHash := uint64(0)
		ngramHash := uint64(0)

		// Character-level hash
		for i, char := range text {
			charHash = charHash*31 + uint64(char) + uint64(i)
		}

		// Word-level hash
		words := strings.Fields(strings.ToLower(text))
		for i, word := range words {
			for _, char := range word {
				wordHash = wordHash*37 + uint64(char) + uint64(i*3)
			}
		}

		// Structural features (length, punctuation, etc.)
		structHash = uint64(len(text))*41 + uint64(len(words))*43
		for _, char := range text {
			if char == '.' || char == ',' || char == '!' || char == '?' {
				structHash = structHash*47 + uint64(char)
			}
		}

		// N-gram hash (bigrams)
		for i := 0; i < len(text)-1; i++ {
			bigram := uint64(text[i])*256 + uint64(text[i+1])
			ngramHash = ngramHash*53 + bigram + uint64(i)
		}

		// Fill the embedding vector using different hash components
		for i := 0; i < 384; i++ {
			var value uint64
			switch i % 4 {
			case 0:
				value = charHash + uint64(i*7)
			case 1:
				value = wordHash + uint64(i*11)
			case 2:
				value = structHash + uint64(i*13)
			case 3:
				value = ngramHash + uint64(i*17)
			}

			// Convert to float in range [-1, 1]
			normalized := float64(value%10000)/5000.0 - 1.0
			embedding[i] = float32(normalized)

			// Add text-based adjustment
			if i < len(text) {
				adjustment := float64(text[i%len(text)]) / 255.0 * 0.1
				embedding[i] += float32(adjustment)
			}

			// Clamp values
			if embedding[i] > 1.0 {
				embedding[i] = 1.0
			} else if embedding[i] < -1.0 {
				embedding[i] = -1.0
			}
		}

		// Normalize to unit length
		var magnitude float32
		for _, val := range embedding {
			magnitude += val * val
		}
		magnitude = float32(math.Sqrt(float64(magnitude)))

		if magnitude > 0 {
			for i := range embedding {
				embedding[i] /= magnitude
			}
		}

		return embedding, nil
	}
}

// NewUnifiedAgentStorage creates a new unified agent storage system
func NewUnifiedAgentStorage(dbPath string) (*UnifiedAgentStorage, error) {
	// Create or open the persistent database
	db, err := chromem.NewPersistentDB(dbPath, false)
	if err != nil {
		return nil, fmt.Errorf("failed to create chromem-go database: %v", err)
	}

	// Create deterministic embedding function to avoid external API dependencies
	embeddingFunc := createDeterministicEmbeddingFunction()

	// Get or create the unified agents collection with deterministic embeddings
	collection, err := db.GetOrCreateCollection("unified_agents", nil, embeddingFunc)
	if err != nil {
		return nil, fmt.Errorf("failed to get or create unified agents collection: %v", err)
	}

	return &UnifiedAgentStorage{
		db:         db,
		collection: collection,
	}, nil
}

// CreateAgent creates a new agent in the unified storage
func (s *UnifiedAgentStorage) CreateAgent(ctx context.Context, agent *UnifiedAgent) error {
	// Generate ID if not provided
	if agent.ID == "" {
		agent.ID = uuid.New().String()
	}

	// Set timestamps
	now := time.Now()
	if agent.CreatedAt.IsZero() {
		agent.CreatedAt = now
	}
	agent.UpdatedAt = now

	// Convert agent to JSON for content
	agentJSON, err := json.Marshal(agent)
	if err != nil {
		return fmt.Errorf("failed to marshal agent: %v", err)
	}

	// Create metadata for search and filtering
	metadata := map[string]string{
		"id":           agent.ID,
		"name":         agent.Name,
		"type":         agent.Type,
		"owner_id":     fmt.Sprintf("%d", agent.OwnerID),
		"collection":   agent.Collection,
		"status":       agent.Status,
		"created_at":   agent.CreatedAt.Format(time.RFC3339),
		"updated_at":   agent.UpdatedAt.Format(time.RFC3339),
		"capabilities": strings.Join(agent.Capabilities, ","),
		"target_types": strings.Join(agent.TargetTypes, ","),
	}

	// Add plugin info to metadata if available
	if agent.PluginInfo != nil {
		metadata["plugin_path"] = agent.PluginInfo.PluginPath
		metadata["build_target"] = agent.PluginInfo.BuildTarget
		metadata["filename"] = agent.PluginInfo.Filename
	}

	// Create document for chromem-go
	doc := chromem.Document{
		ID:       agent.ID,
		Content:  string(agentJSON),
		Metadata: metadata,
	}

	// Store agent in chromem-go
	return s.collection.AddDocuments(ctx, []chromem.Document{doc}, 1)
}

// GetAgentByID retrieves an agent by ID
func (s *UnifiedAgentStorage) GetAgentByID(ctx context.Context, id string) (*UnifiedAgent, error) {
	// Get document by ID
	doc, err := s.collection.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("agent not found: %s", id)
	}

	// Parse the JSON content
	var agent UnifiedAgent
	if err := json.Unmarshal([]byte(doc.Content), &agent); err != nil {
		return nil, fmt.Errorf("failed to unmarshal agent: %v", err)
	}

	return &agent, nil
}

// GetAgentsByOwner retrieves all agents for a specific owner
func (s *UnifiedAgentStorage) GetAgentsByOwner(ctx context.Context, ownerID int64) ([]*UnifiedAgent, error) {
	// Use Query with metadata filter
	where := map[string]string{
		"owner_id": fmt.Sprintf("%d", ownerID),
	}

	// Query for agents
	queryText := "agent"
	results, err := s.collection.Query(ctx, queryText, 100, where, nil) // Get up to 100 agents
	if err != nil {
		return []*UnifiedAgent{}, nil // Return empty list if query fails
	}

	// Convert results to UnifiedAgent structs
	agents := make([]*UnifiedAgent, 0, len(results))
	for _, result := range results {
		var agent UnifiedAgent
		if err := json.Unmarshal([]byte(result.Content), &agent); err != nil {
			continue // Skip invalid agents
		}
		agents = append(agents, &agent)
	}

	return agents, nil
}

// ListAgents retrieves all agents in the storage
func (s *UnifiedAgentStorage) ListAgents(ctx context.Context) ([]*UnifiedAgent, error) {
	// Query for all agents without owner filter
	queryText := "agent"
	results, err := s.collection.Query(ctx, queryText, 1000, nil, nil) // Get up to 1000 agents
	if err != nil {
		return []*UnifiedAgent{}, nil // Return empty list if query fails
	}

	// Convert results to UnifiedAgent structs
	agents := make([]*UnifiedAgent, 0, len(results))
	for _, result := range results {
		var agent UnifiedAgent
		if err := json.Unmarshal([]byte(result.Content), &agent); err != nil {
			continue // Skip invalid agents
		}
		agents = append(agents, &agent)
	}

	return agents, nil
}

// UpdateAgent updates an existing agent
func (s *UnifiedAgentStorage) UpdateAgent(ctx context.Context, agent *UnifiedAgent) error {
	// Check if agent exists
	_, err := s.GetAgentByID(ctx, agent.ID)
	if err != nil {
		return fmt.Errorf("agent not found: %s", agent.ID)
	}

	// Update timestamp
	agent.UpdatedAt = time.Now()

	// Delete the old document
	err = s.collection.Delete(ctx, nil, nil, agent.ID)
	if err != nil {
		return fmt.Errorf("failed to delete old agent document: %w", err)
	}

	// Create the updated agent
	return s.CreateAgent(ctx, agent)
}

// DeleteAgent deletes an agent by ID
func (s *UnifiedAgentStorage) DeleteAgent(ctx context.Context, id string) error {
	return s.collection.Delete(ctx, nil, nil, id)
}

// RegisterAgentConfig stores/updates agent configuration (for agent builder compatibility)
func (s *UnifiedAgentStorage) RegisterAgentConfig(ctx context.Context, agentID string, config map[string]interface{}) error {
	// Try to get existing agent
	existingAgent, err := s.GetAgentByID(ctx, agentID)
	if err != nil {
		// Agent doesn't exist, create a new one from config
		agent := &UnifiedAgent{
			ID:          agentID,
			Name:        getStringFromConfig(config, "name", agentID),
			Type:        getStringFromConfig(config, "agent_type", "llm"),
			OwnerID:     1, // Default owner for now
			Collection:  getStringFromConfig(config, "collection", "default"),
			Status:      "idle",
			AgentConfig: config,
		}

		// Extract capabilities and target types from config
		if caps, ok := config["capabilities"].([]interface{}); ok {
			for _, cap := range caps {
				if capStr, ok := cap.(string); ok {
					agent.Capabilities = append(agent.Capabilities, capStr)
				}
			}
		}

		if targets, ok := config["target_types"].([]interface{}); ok {
			for _, target := range targets {
				if targetStr, ok := target.(string); ok {
					agent.TargetTypes = append(agent.TargetTypes, targetStr)
				}
			}
		}

		// Add plugin info if available
		if pluginPath, ok := config["plugin_path"].(string); ok {
			buildTarget := getStringFromConfig(config, "build_target", "plugin")
			agent.PluginInfo = &PluginInfo{
				PluginID:      agentID,
				PluginVersion: "1.0",
				PluginPath:    pluginPath,
				BuildTarget:   buildTarget,
			}
		}

		return s.CreateAgent(ctx, agent)
	}

	// Agent exists, update its configuration
	existingAgent.AgentConfig = config
	existingAgent.UpdatedAt = time.Now()

	// Update plugin info if provided
	if pluginPath, ok := config["plugin_path"].(string); ok {
		buildTarget := getStringFromConfig(config, "build_target", "plugin")
		if existingAgent.PluginInfo == nil {
			existingAgent.PluginInfo = &PluginInfo{}
		}
		existingAgent.PluginInfo.PluginPath = pluginPath
		existingAgent.PluginInfo.BuildTarget = buildTarget
	}

	return s.UpdateAgent(ctx, existingAgent)
}

// GetAgentConfig retrieves agent configuration (for agent registry compatibility)
func (s *UnifiedAgentStorage) GetAgentConfig(ctx context.Context, agentID string) (map[string]interface{}, error) {
	agent, err := s.GetAgentByID(ctx, agentID)
	if err != nil {
		return nil, err
	}

	return agent.AgentConfig, nil
}

// Helper function to safely get string values from config
func getStringFromConfig(config map[string]interface{}, key, defaultValue string) string {
	if value, ok := config[key].(string); ok {
		return value
	}
	return defaultValue
}

// Close closes the database connection
func (s *UnifiedAgentStorage) Close() error {
	// chromem-go doesn't have an explicit close method
	return nil
}

// AgentRegistryAdapter adapts UnifiedAgentStorage to the AgentRegistry interface
type AgentRegistryAdapter struct {
	storage *UnifiedAgentStorage
}

// NewAgentRegistryAdapter creates a new adapter
func NewAgentRegistryAdapter(storage *UnifiedAgentStorage) *AgentRegistryAdapter {
	return &AgentRegistryAdapter{storage: storage}
}

// RegisterAgent implements the AgentRegistry interface
func (a *AgentRegistryAdapter) RegisterAgent(agentID string, config map[string]interface{}) error {
	return a.storage.RegisterAgentConfig(context.Background(), agentID, config)
}

// GetAgent implements the AgentRegistry interface
func (a *AgentRegistryAdapter) GetAgent(agentID string) (map[string]interface{}, error) {
	return a.storage.GetAgentConfig(context.Background(), agentID)
}

// AgentRepositoryAdapter adapts UnifiedAgentStorage to the AgentRepository interface
type AgentRepositoryAdapter struct {
	storage *UnifiedAgentStorage
}

// NewAgentRepositoryAdapter creates a new adapter
func NewAgentRepositoryAdapter(storage *UnifiedAgentStorage) *AgentRepositoryAdapter {
	return &AgentRepositoryAdapter{storage: storage}
}

// Agent represents the API agent structure (for compatibility)
type Agent struct {
	ID        string    `json:"id"`
	OwnerID   int64     `json:"owner_id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	Config    string    `json:"config"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// GetAgentsByOwner implements the AgentRepository interface
func (a *AgentRepositoryAdapter) GetAgentsByOwner(ctx context.Context, ownerID int64) ([]*Agent, error) {
	unifiedAgents, err := a.storage.GetAgentsByOwner(ctx, ownerID)
	if err != nil {
		return nil, err
	}

	// Convert UnifiedAgent to Agent
	agents := make([]*Agent, len(unifiedAgents))
	for i, ua := range unifiedAgents {
		configJSON, _ := json.Marshal(ua.AgentConfig)
		agents[i] = &Agent{
			ID:        ua.ID,
			OwnerID:   ua.OwnerID,
			Name:      ua.Name,
			Type:      ua.Type,
			Config:    string(configJSON),
			CreatedAt: ua.CreatedAt,
			UpdatedAt: ua.UpdatedAt,
		}
	}

	return agents, nil
}

// GetAgentByID implements the AgentRepository interface
func (a *AgentRepositoryAdapter) GetAgentByID(ctx context.Context, id string) (*Agent, error) {
	ua, err := a.storage.GetAgentByID(ctx, id)
	if err != nil {
		return nil, err
	}

	configJSON, _ := json.Marshal(ua.AgentConfig)
	return &Agent{
		ID:        ua.ID,
		OwnerID:   ua.OwnerID,
		Name:      ua.Name,
		Type:      ua.Type,
		Config:    string(configJSON),
		CreatedAt: ua.CreatedAt,
		UpdatedAt: ua.UpdatedAt,
	}, nil
}

// CreateAgent implements the AgentRepository interface
func (a *AgentRepositoryAdapter) CreateAgent(ctx context.Context, agent *Agent) error {
	// Parse config JSON
	var agentConfig map[string]interface{}
	if agent.Config != "" {
		json.Unmarshal([]byte(agent.Config), &agentConfig)
	}

	// Create UnifiedAgent
	ua := &UnifiedAgent{
		ID:          agent.ID,
		Name:        agent.Name,
		Type:        agent.Type,
		OwnerID:     agent.OwnerID,
		CreatedAt:   agent.CreatedAt,
		UpdatedAt:   agent.UpdatedAt,
		AgentConfig: agentConfig,
		Collection:  "default",
		Status:      "idle",
	}

	return a.storage.CreateAgent(ctx, ua)
}

// UpdateAgent implements the AgentRepository interface
func (a *AgentRepositoryAdapter) UpdateAgent(ctx context.Context, agent *Agent) error {
	// Get existing unified agent
	ua, err := a.storage.GetAgentByID(ctx, agent.ID)
	if err != nil {
		return err
	}

	// Update fields
	ua.Name = agent.Name
	ua.Type = agent.Type
	ua.OwnerID = agent.OwnerID
	ua.UpdatedAt = agent.UpdatedAt

	// Parse and update config
	if agent.Config != "" {
		var agentConfig map[string]interface{}
		json.Unmarshal([]byte(agent.Config), &agentConfig)
		ua.AgentConfig = agentConfig
	}

	return a.storage.UpdateAgent(ctx, ua)
}

// DeleteAgent implements the AgentRepository interface
func (a *AgentRepositoryAdapter) DeleteAgent(ctx context.Context, id string) error {
	return a.storage.DeleteAgent(ctx, id)
}

// UnifiedAgentStorageAdapter adapts UnifiedAgentStorage to the agentify UnifiedAgentStorageInterface
type UnifiedAgentStorageAdapter struct {
	storage *UnifiedAgentStorage
}

// NewUnifiedAgentStorageAdapter creates a new adapter
func NewUnifiedAgentStorageAdapter(storage *UnifiedAgentStorage) *UnifiedAgentStorageAdapter {
	return &UnifiedAgentStorageAdapter{storage: storage}
}

// RegisterAgentConfig implements the UnifiedAgentStorageInterface
func (a *UnifiedAgentStorageAdapter) RegisterAgentConfig(ctx context.Context, agentID string, config map[string]interface{}) error {
	return a.storage.RegisterAgentConfig(ctx, agentID, config)
}

// GetAgentConfig implements the UnifiedAgentStorageInterface
func (a *UnifiedAgentStorageAdapter) GetAgentConfig(ctx context.Context, agentID string) (map[string]interface{}, error) {
	return a.storage.GetAgentConfig(ctx, agentID)
}

// ListAgents implements the UnifiedAgentStorageInterface
func (a *UnifiedAgentStorageAdapter) ListAgents(ctx context.Context) ([]interface{}, error) {
	agents, err := a.storage.ListAgents(ctx)
	if err != nil {
		return nil, err
	}

	// Convert UnifiedAgent to interface{} to avoid import cycles
	agentInfos := make([]interface{}, len(agents))
	for i, agent := range agents {
		buildTarget := "plugin" // default
		if agent.PluginInfo != nil {
			buildTarget = agent.PluginInfo.BuildTarget
		}

		// Create a map that matches the agentify.AgentInfo structure
		agentInfos[i] = map[string]interface{}{
			"id":           agent.ID,
			"name":         agent.Name,
			"type":         agent.Type,
			"build_target": buildTarget,
		}
	}

	return agentInfos, nil
}

// Enhanced query methods for feature parity with SimpleAgentRepository

// FindByCapability finds agents that have a specific capability
func (s *UnifiedAgentStorage) FindByCapability(capability string) ([]*UnifiedAgent, error) {
	ctx := context.Background()

	// Get all documents and filter manually (same approach as FindByOwner)
	// Start with a small query and progressively increase if needed
	var allResults []chromem.Result
	for limit := 3; limit <= 100; limit += 10 {
		results, err := s.collection.Query(
			ctx,
			"id", // Query text that should match JSON content
			limit,
			nil, // No metadata filter
			nil, // No where document filters
		)
		if err != nil {
			// If this limit fails, use previous results
			break
		}

		allResults = results

		// If we got fewer results than the limit, we have all documents
		if len(results) < limit {
			break
		}
	}

	// Manual filtering by checking capabilities in metadata
	var filteredResults []chromem.Result
	for _, result := range allResults {
		// Check if the capability is in the metadata capabilities field
		if capabilitiesStr, ok := result.Metadata["capabilities"]; ok {
			if strings.Contains(strings.ToLower(capabilitiesStr), strings.ToLower(capability)) {
				filteredResults = append(filteredResults, result)
			}
		}
	}

	// Convert filtered results to UnifiedAgent objects
	var agents []*UnifiedAgent
	for _, result := range filteredResults {
		var agent UnifiedAgent
		if err := json.Unmarshal([]byte(result.Content), &agent); err != nil {
			continue // Skip malformed agents
		}
		agents = append(agents, &agent)
	}

	return agents, nil
}

// FindByOwner finds all agents owned by a specific user
func (s *UnifiedAgentStorage) FindByOwner(ownerID int64) ([]*UnifiedAgent, error) {
	ctx := context.Background()

	// Get all documents and filter manually (chromem-go metadata filtering doesn't work reliably)
	// Start with a small query and progressively increase if needed
	var allResults []chromem.Result
	for limit := 3; limit <= 100; limit += 10 {
		results, err := s.collection.Query(
			ctx,
			"id", // Query text that should match JSON content
			limit,
			nil, // No metadata filter
			nil, // No where document filters
		)
		if err != nil {
			// If this limit fails, use previous results
			break
		}

		allResults = results

		// If we got fewer results than the limit, we have all documents
		if len(results) < limit {
			break
		}
	}

	// Manual filtering by owner_id metadata
	var filteredResults []chromem.Result
	for _, result := range allResults {
		if result.Metadata["owner_id"] == fmt.Sprintf("%d", ownerID) {
			filteredResults = append(filteredResults, result)
		}
	}

	// Convert filtered results to UnifiedAgent objects
	var agents []*UnifiedAgent
	for _, result := range filteredResults {
		var agent UnifiedAgent
		if err := json.Unmarshal([]byte(result.Content), &agent); err != nil {
			continue // Skip malformed agents
		}
		agents = append(agents, &agent)
	}

	return agents, nil
}

// FindByType finds all agents of a specific type
func (s *UnifiedAgentStorage) FindByType(agentType string) ([]*UnifiedAgent, error) {
	ctx := context.Background()

	// Query agents by type using metadata filter
	results, err := s.collection.Query(
		ctx,
		agentType, // Use type as query text for semantic search
		100,       // Reasonable limit
		map[string]string{
			"type": agentType,
		},
		nil, // No where document filters
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query agents by type: %v", err)
	}

	agents := make([]*UnifiedAgent, 0, len(results))
	for _, result := range results {
		var agent UnifiedAgent
		if err := json.Unmarshal([]byte(result.Content), &agent); err != nil {
			continue // Skip malformed agents
		}
		agents = append(agents, &agent)
	}

	return agents, nil
}

// FindByStatus finds all agents with a specific status
func (s *UnifiedAgentStorage) FindByStatus(status string) ([]*UnifiedAgent, error) {
	ctx := context.Background()

	// Get all documents and filter manually (same approach as FindByOwner)
	// Start with a small query and progressively increase if needed
	var allResults []chromem.Result
	for limit := 3; limit <= 100; limit += 10 {
		results, err := s.collection.Query(
			ctx,
			"id", // Query text that should match JSON content
			limit,
			nil, // No metadata filter
			nil, // No where document filters
		)
		if err != nil {
			// If this limit fails, use previous results
			break
		}

		allResults = results

		// If we got fewer results than the limit, we have all documents
		if len(results) < limit {
			break
		}
	}

	// Manual filtering by status metadata
	var filteredResults []chromem.Result
	for _, result := range allResults {
		if result.Metadata["status"] == status {
			filteredResults = append(filteredResults, result)
		}
	}

	// Convert filtered results to UnifiedAgent objects
	var agents []*UnifiedAgent
	for _, result := range filteredResults {
		var agent UnifiedAgent
		if err := json.Unmarshal([]byte(result.Content), &agent); err != nil {
			continue // Skip malformed agents
		}
		agents = append(agents, &agent)
	}

	return agents, nil
}

// CountAgents returns the total number of agents in storage
func (s *UnifiedAgentStorage) CountAgents() (int, error) {
	ctx := context.Background()

	// Start with a small query to see if there are any agents
	// Use a query text that will match JSON content
	results, err := s.collection.Query(
		ctx,
		"id",                // Query text that should match JSON content
		1,                   // Start with 1 to test
		map[string]string{}, // No metadata filters
		nil,                 // No where document filters
	)
	if err != nil {
		// If even 1 fails, assume no agents
		return 0, nil
	}

	// If we got 1 result, try progressively larger queries
	if len(results) == 0 {
		return 0, nil
	}

	// Try increasing limits until we get all results
	for limit := 3; limit <= 100; limit += 5 {
		newResults, err := s.collection.Query(
			ctx,
			"id",
			limit,
			map[string]string{},
			nil,
		)
		if err != nil {
			// If this limit fails, use the previous successful count
			break
		}

		results = newResults

		// If we got fewer results than the limit, we have all agents
		if len(results) < limit {
			break
		}
	}

	return len(results), nil
}
