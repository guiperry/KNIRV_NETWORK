package migration

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"KNIRVENGINE/desktop-client/agent"
	"KNIRVENGINE/desktop-client/database"
)

// SimpleToUnifiedMigrator handles migration from SimpleAgentRepository to UnifiedAgentStorage
type SimpleToUnifiedMigrator struct {
	simpleRepo     *database.SimpleAgentRepository
	unifiedStorage *agent.UnifiedAgentStorage
	backupPath     string
}

// NewSimpleToUnifiedMigrator creates a new migrator for SimpleAgent to UnifiedAgent migration
func NewSimpleToUnifiedMigrator(simpleDBPath, unifiedDBPath, backupPath string) (*SimpleToUnifiedMigrator, error) {
	// Initialize SimpleDomainDB first
	db, err := database.NewSimpleDomainDB(simpleDBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create simple domain db: %v", err)
	}

	// Get or create the default agent collection
	collection, err := db.GetOrCreateCollection("agents")
	if err != nil {
		return nil, fmt.Errorf("failed to get/create agent collection: %v", err)
	}

	// Initialize SimpleAgentRepository
	simpleRepo := database.NewSimpleAgentRepository(collection)

	// Initialize UnifiedAgentStorage
	unifiedStorage, err := agent.NewUnifiedAgentStorage(unifiedDBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create unified agent storage: %v", err)
	}

	// Ensure backup directory exists
	if err := os.MkdirAll(backupPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create backup directory: %v", err)
	}

	return &SimpleToUnifiedMigrator{
		simpleRepo:     simpleRepo,
		unifiedStorage: unifiedStorage,
		backupPath:     backupPath,
	}, nil
}

// Close closes the migrator and ensures data persistence
func (m *SimpleToUnifiedMigrator) Close() error {
	// Close unified storage to ensure data is persisted
	if m.unifiedStorage != nil {
		return m.unifiedStorage.Close()
	}
	return nil
}

// MigrationReport represents the result of a SimpleAgent to UnifiedAgent migration
type SimpleToUnifiedMigrationReport struct {
	TotalAgents      int                          `json:"total_agents"`
	MigratedAgents   int                          `json:"migrated_agents"`
	FailedAgents     int                          `json:"failed_agents"`
	SkippedAgents    int                          `json:"skipped_agents"`
	Errors           []SimpleMigrationError       `json:"errors"`
	StartTime        time.Time                    `json:"start_time"`
	EndTime          time.Time                    `json:"end_time"`
	Duration         time.Duration                `json:"duration"`
	AgentDetails     []SimpleMigratedAgentDetails `json:"agent_details"`
	BackupLocation   string                       `json:"backup_location"`
	DataStructureMap map[string]interface{}       `json:"data_structure_mapping"`
}

// SimpleMigrationError represents an error during SimpleAgent migration
type SimpleMigrationError struct {
	AgentID string `json:"agent_id"`
	Error   string `json:"error"`
	Step    string `json:"step"`
}

// SimpleMigratedAgentDetails represents details of a migrated SimpleAgent
type SimpleMigratedAgentDetails struct {
	AgentID         string    `json:"agent_id"`
	AgentName       string    `json:"agent_name"`
	Collection      string    `json:"collection"`
	Status          string    `json:"status"`
	OwnerID         int64     `json:"owner_id"`
	CapabilitiesOld []string  `json:"capabilities_old"`
	CapabilitiesNew []string  `json:"capabilities_new"`
	MigratedAt      time.Time `json:"migrated_at"`
}

// MigrateAllAgents migrates all agents from SimpleAgentRepository to UnifiedAgentStorage
func (m *SimpleToUnifiedMigrator) MigrateAllAgents(ctx context.Context) (*SimpleToUnifiedMigrationReport, error) {
	report := &SimpleToUnifiedMigrationReport{
		StartTime:        time.Now(),
		Errors:           make([]SimpleMigrationError, 0),
		AgentDetails:     make([]SimpleMigratedAgentDetails, 0),
		BackupLocation:   m.backupPath,
		DataStructureMap: m.getDataStructureMapping(),
	}

	log.Println("Starting SimpleAgent to UnifiedAgent migration...")

	// Create backup first
	if err := m.createBackup(ctx); err != nil {
		return report, fmt.Errorf("failed to create backup: %v", err)
	}

	// Get all SimpleAgents
	// Get all agents by querying with ownerID 0 (assuming this gets all agents)
	simpleAgents, err := m.simpleRepo.GetAgentsByOwner(ctx, 0)
	if err != nil {
		return report, fmt.Errorf("failed to get simple agents: %v", err)
	}

	report.TotalAgents = len(simpleAgents)
	log.Printf("Found %d SimpleAgents to migrate", len(simpleAgents))

	// Migrate each agent
	for _, simpleAgent := range simpleAgents {
		if err := m.migrateSimpleAgent(ctx, simpleAgent, report); err != nil {
			log.Printf("Failed to migrate agent %s: %v", simpleAgent.ID, err)
			report.FailedAgents++
			report.Errors = append(report.Errors, SimpleMigrationError{
				AgentID: simpleAgent.ID,
				Error:   err.Error(),
				Step:    "migration",
			})
		} else {
			log.Printf("Successfully migrated agent %s", simpleAgent.ID)
			report.MigratedAgents++
		}
	}

	report.EndTime = time.Now()
	report.Duration = report.EndTime.Sub(report.StartTime)

	log.Printf("Migration completed: %d migrated, %d failed, %d skipped",
		report.MigratedAgents, report.FailedAgents, report.SkippedAgents)

	return report, nil
}

// migrateSimpleAgent migrates a single SimpleAgent to UnifiedAgent
func (m *SimpleToUnifiedMigrator) migrateSimpleAgent(ctx context.Context, simpleAgent *database.SimpleAgent, report *SimpleToUnifiedMigrationReport) error {
	// Check if agent already exists in unified storage
	existingAgent, err := m.unifiedStorage.GetAgentByID(ctx, simpleAgent.ID)
	if err == nil && existingAgent != nil {
		log.Printf("Agent %s already exists in unified storage, skipping", simpleAgent.ID)
		report.SkippedAgents++
		return nil
	}

	// Convert SimpleAgent to UnifiedAgent
	unifiedAgent := m.convertSimpleToUnified(simpleAgent)

	// Create the agent in unified storage
	if err := m.unifiedStorage.CreateAgent(ctx, unifiedAgent); err != nil {
		return fmt.Errorf("failed to create unified agent: %v", err)
	}

	// Record migration details
	report.AgentDetails = append(report.AgentDetails, SimpleMigratedAgentDetails{
		AgentID:         simpleAgent.ID,
		AgentName:       simpleAgent.Name,
		Collection:      simpleAgent.Collection,
		Status:          simpleAgent.Status,
		OwnerID:         simpleAgent.OwnerID,
		CapabilitiesOld: simpleAgent.Capabilities,
		CapabilitiesNew: unifiedAgent.Capabilities,
		MigratedAt:      time.Now(),
	})

	return nil
}

// convertSimpleToUnified converts a SimpleAgent to UnifiedAgent
func (m *SimpleToUnifiedMigrator) convertSimpleToUnified(simpleAgent *database.SimpleAgent) *agent.UnifiedAgent {
	now := time.Now()

	unifiedAgent := &agent.UnifiedAgent{
		ID:           simpleAgent.ID,
		Name:         simpleAgent.Name,
		Type:         "llm", // Default type for migrated agents
		OwnerID:      simpleAgent.OwnerID,
		CreatedAt:    simpleAgent.CreatedAt,
		UpdatedAt:    now,
		Collection:   simpleAgent.Collection,
		ImageURL:     simpleAgent.ImageURL,
		Status:       simpleAgent.Status,
		Capabilities: simpleAgent.Capabilities,
		TargetTypes:  []string{"general"}, // Default target type

		// Create basic agent config from SimpleAgent fields
		AgentConfig: map[string]interface{}{
			"collection":     simpleAgent.Collection,
			"image_url":      simpleAgent.ImageURL,
			"status":         simpleAgent.Status,
			"capabilities":   simpleAgent.Capabilities,
			"token_id":       simpleAgent.TokenID,
			"contract_addr":  simpleAgent.ContractAddr,
			"migrated_from":  "SimpleAgent",
			"migration_date": now.Format(time.RFC3339),
		},

		// Initialize empty plugin info (will be populated during agent building)
		PluginInfo: nil,

		// Initialize empty API keys (will be configured later)
		APIKeys: make(map[string]string),
	}

	// Add blockchain-related fields if present
	if simpleAgent.TokenID != "" || simpleAgent.ContractAddr != "" {
		unifiedAgent.TargetTypes = append(unifiedAgent.TargetTypes, "blockchain")
	}

	return unifiedAgent
}

// createBackup creates a backup of the current SimpleAgent data
func (m *SimpleToUnifiedMigrator) createBackup(ctx context.Context) error {
	log.Println("Creating backup of SimpleAgent data...")

	// Get all simple agents
	// Get all agents by querying with ownerID 0 (assuming this gets all agents)
	simpleAgents, err := m.simpleRepo.GetAgentsByOwner(ctx, 0)
	if err != nil {
		return fmt.Errorf("failed to get agents for backup: %v", err)
	}

	// Create backup file
	backupFile := filepath.Join(m.backupPath, fmt.Sprintf("simple_agents_backup_%s.json",
		time.Now().Format("2006-01-02_15-04-05")))

	// Marshal agents to JSON
	backupData, err := json.MarshalIndent(simpleAgents, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal backup data: %v", err)
	}

	// Write backup file
	if err := os.WriteFile(backupFile, backupData, 0644); err != nil {
		return fmt.Errorf("failed to write backup file: %v", err)
	}

	log.Printf("Backup created: %s", backupFile)
	return nil
}

// getDataStructureMapping returns a mapping of data structure changes
func (m *SimpleToUnifiedMigrator) getDataStructureMapping() map[string]interface{} {
	return map[string]interface{}{
		"SimpleAgent_fields": []string{
			"ID", "Name", "Collection", "ImageURL", "Status",
			"Capabilities", "TokenID", "ContractAddr", "OwnerID", "CreatedAt",
		},
		"UnifiedAgent_fields": []string{
			"ID", "Name", "Type", "OwnerID", "CreatedAt", "UpdatedAt",
			"AgentConfig", "Collection", "ImageURL", "Status", "Capabilities",
			"TargetTypes", "PluginInfo", "APIKeys",
		},
		"new_fields": []string{
			"Type", "UpdatedAt", "AgentConfig", "TargetTypes", "PluginInfo", "APIKeys",
		},
		"mapped_fields": map[string]string{
			"TokenID":      "AgentConfig.token_id",
			"ContractAddr": "AgentConfig.contract_addr",
		},
		"default_values": map[string]interface{}{
			"Type":        "llm",
			"TargetTypes": []string{"general"},
			"APIKeys":     map[string]string{},
		},
	}
}
