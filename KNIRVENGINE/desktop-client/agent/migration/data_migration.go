package migration

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"Agentic_Engine/agent"
	"Agentic_Engine/agent/core"
)

// DataMigrator handles migration from old agent system to new unified system
type DataMigrator struct {
	oldStorage *agent.UnifiedAgentStorage
	newService *core.AgentCoreService
}

// NewDataMigrator creates a new data migrator
func NewDataMigrator(oldStoragePath string, newService *core.AgentCoreService) (*DataMigrator, error) {
	oldStorage, err := agent.NewUnifiedAgentStorage(oldStoragePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create old storage: %v", err)
	}

	return &DataMigrator{
		oldStorage: oldStorage,
		newService: newService,
	}, nil
}

// MigrateAllAgents migrates all agents from old system to new system
func (m *DataMigrator) MigrateAllAgents(ctx context.Context) error {
	log.Println("Starting agent data migration...")

	// Get all agents from old storage
	oldAgents, err := m.oldStorage.ListAgents(ctx)
	if err != nil {
		return fmt.Errorf("failed to list old agents: %v", err)
	}

	log.Printf("Found %d agents to migrate", len(oldAgents))

	var migrated, failed int
	for _, oldAgent := range oldAgents {
		if err := m.migrateAgent(ctx, oldAgent); err != nil {
			log.Printf("Failed to migrate agent %s: %v", oldAgent.ID, err)
			failed++
		} else {
			log.Printf("Successfully migrated agent %s", oldAgent.ID)
			migrated++
		}
	}

	log.Printf("Migration completed: %d migrated, %d failed", migrated, failed)
	return nil
}

// migrateAgent migrates a single agent from old format to new format
func (m *DataMigrator) migrateAgent(ctx context.Context, oldAgent *agent.UnifiedAgent) error {
	// Convert old agent to new unified agent
	newAgent := m.convertToNewAgent(oldAgent)

	// Create the agent in the new system
	return (*m.newService).CreateAgent(ctx, newAgent)
}

// convertToNewAgent converts old UnifiedAgent to new core.UnifiedAgent
func (m *DataMigrator) convertToNewAgent(oldAgent *agent.UnifiedAgent) *core.UnifiedAgent {
	newAgent := &core.UnifiedAgent{
		ID:           oldAgent.ID,
		Name:         oldAgent.Name,
		Type:         oldAgent.Type,
		Version:      "1.0.0", // Default version if not present
		Description:  "",      // Will be extracted from config
		CreatedAt:    oldAgent.CreatedAt,
		UpdatedAt:    oldAgent.UpdatedAt,
		Config:       oldAgent.AgentConfig,
		BuildTarget:  "plugin", // Default build target
		PluginPath:   "",
		Status:       oldAgent.Status,
		Collection:   oldAgent.Collection,
		ImageURL:     oldAgent.ImageURL,
		Capabilities: oldAgent.Capabilities,
		TargetTypes:  oldAgent.TargetTypes,
		Tags:         []string{}, // Initialize empty tags
		OwnerID:      oldAgent.OwnerID,
		APIKeys:      oldAgent.APIKeys,
		Permissions:  make(map[string]bool),
		DefaultTerminalConfig: &core.TerminalConfig{
			DefaultRows:    24,
			DefaultCols:    80,
			FontSize:       14,
			FontFamily:     "Menlo, Monaco, 'Courier New', monospace",
			Theme:          "dark",
			ScrollbackSize: 5000,
			AutoOpen:       false,
		},
	}

	// Extract additional information from plugin info
	if oldAgent.PluginInfo != nil {
		newAgent.PluginPath = oldAgent.PluginInfo.PluginPath
		newAgent.BuildTarget = oldAgent.PluginInfo.BuildTarget
		if oldAgent.PluginInfo.PluginVersion != "" {
			newAgent.Version = oldAgent.PluginInfo.PluginVersion
		}
	}

	// Extract description from config if available
	if desc, ok := oldAgent.AgentConfig["description"].(string); ok && desc != "" {
		newAgent.Description = desc
	} else {
		newAgent.Description = fmt.Sprintf("Migrated agent: %s", oldAgent.Name)
	}

	// Extract version from config if available
	if version, ok := oldAgent.AgentConfig["version"].(string); ok && version != "" {
		newAgent.Version = version
	}

	// Extract terminal config from agent config if available
	if terminalConfig, ok := oldAgent.AgentConfig["terminal_config"].(map[string]interface{}); ok {
		newAgent.DefaultTerminalConfig = m.convertTerminalConfig(terminalConfig)
	}

	// Set default permissions
	newAgent.Permissions = map[string]bool{
		"read":   true,
		"write":  true,
		"delete": false,
		"admin":  false,
	}

	return newAgent
}

// convertTerminalConfig converts old terminal config to new format
func (m *DataMigrator) convertTerminalConfig(oldConfig map[string]interface{}) *core.TerminalConfig {
	config := &core.TerminalConfig{
		DefaultRows:    24,
		DefaultCols:    80,
		FontSize:       14,
		FontFamily:     "Menlo, Monaco, 'Courier New', monospace",
		Theme:          "dark",
		ScrollbackSize: 5000,
		AutoOpen:       false,
		CustomOptions:  make(map[string]string),
	}

	// Convert known fields
	if rows, ok := oldConfig["rows"].(float64); ok {
		config.DefaultRows = int(rows)
	}
	if cols, ok := oldConfig["cols"].(float64); ok {
		config.DefaultCols = int(cols)
	}
	if fontSize, ok := oldConfig["font_size"].(float64); ok {
		config.FontSize = int(fontSize)
	}
	if fontFamily, ok := oldConfig["font_family"].(string); ok {
		config.FontFamily = fontFamily
	}
	if theme, ok := oldConfig["theme"].(string); ok {
		config.Theme = theme
	}
	if scrollback, ok := oldConfig["scrollback"].(float64); ok {
		config.ScrollbackSize = int(scrollback)
	}
	if autoOpen, ok := oldConfig["auto_open"].(bool); ok {
		config.AutoOpen = autoOpen
	}
	if customCSS, ok := oldConfig["custom_css"].(string); ok {
		config.CustomCSS = customCSS
	}

	return config
}

// MigrationReport represents the result of a migration operation
type MigrationReport struct {
	TotalAgents    int                    `json:"total_agents"`
	MigratedAgents int                    `json:"migrated_agents"`
	FailedAgents   int                    `json:"failed_agents"`
	Errors         []MigrationError       `json:"errors"`
	StartTime      time.Time              `json:"start_time"`
	EndTime        time.Time              `json:"end_time"`
	Duration       time.Duration          `json:"duration"`
	AgentDetails   []MigratedAgentDetails `json:"agent_details"`
}

// MigrationError represents an error during migration
type MigrationError struct {
	AgentID string `json:"agent_id"`
	Error   string `json:"error"`
}

// MigratedAgentDetails represents details of a migrated agent
type MigratedAgentDetails struct {
	AgentID     string `json:"agent_id"`
	AgentName   string `json:"agent_name"`
	OldType     string `json:"old_type"`
	NewType     string `json:"new_type"`
	BuildTarget string `json:"build_target"`
	Status      string `json:"status"`
}

// MigrateWithReport migrates all agents and returns a detailed report
func (m *DataMigrator) MigrateWithReport(ctx context.Context) (*MigrationReport, error) {
	report := &MigrationReport{
		StartTime:    time.Now(),
		Errors:       make([]MigrationError, 0),
		AgentDetails: make([]MigratedAgentDetails, 0),
	}

	// Get all agents from old storage
	oldAgents, err := m.oldStorage.ListAgents(ctx)
	if err != nil {
		return report, fmt.Errorf("failed to list old agents: %v", err)
	}

	report.TotalAgents = len(oldAgents)

	for _, oldAgent := range oldAgents {
		if err := m.migrateAgent(ctx, oldAgent); err != nil {
			report.FailedAgents++
			report.Errors = append(report.Errors, MigrationError{
				AgentID: oldAgent.ID,
				Error:   err.Error(),
			})
		} else {
			report.MigratedAgents++
			report.AgentDetails = append(report.AgentDetails, MigratedAgentDetails{
				AgentID:     oldAgent.ID,
				AgentName:   oldAgent.Name,
				OldType:     oldAgent.Type,
				NewType:     oldAgent.Type,
				BuildTarget: m.getBuildTarget(oldAgent),
				Status:      oldAgent.Status,
			})
		}
	}

	report.EndTime = time.Now()
	report.Duration = report.EndTime.Sub(report.StartTime)

	return report, nil
}

// getBuildTarget determines the build target for an old agent
func (m *DataMigrator) getBuildTarget(oldAgent *agent.UnifiedAgent) string {
	if oldAgent.PluginInfo != nil && oldAgent.PluginInfo.BuildTarget != "" {
		return oldAgent.PluginInfo.BuildTarget
	}
	return "plugin" // Default to plugin
}

// SaveMigrationReport saves the migration report to a file
func (m *DataMigrator) SaveMigrationReport(report *MigrationReport, filePath string) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal report: %v", err)
	}

	// In a real implementation, this would write to a file
	// For now, we'll just log it
	log.Printf("Migration report: %s", string(data))

	return nil
}

// ValidateMigration validates that the migration was successful
func (m *DataMigrator) ValidateMigration(ctx context.Context) error {
	// Get count from old storage
	oldAgents, err := m.oldStorage.ListAgents(ctx)
	if err != nil {
		return fmt.Errorf("failed to get old agents count: %v", err)
	}
	oldCount := len(oldAgents)

	// Get count from new storage
	newAgents, err := (*m.newService).ListAgents(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to get new agents count: %v", err)
	}
	newCount := len(newAgents)

	if oldCount != newCount {
		return fmt.Errorf("migration validation failed: old count %d != new count %d", oldCount, newCount)
	}

	log.Printf("Migration validation successful: %d agents migrated", newCount)
	return nil
}
