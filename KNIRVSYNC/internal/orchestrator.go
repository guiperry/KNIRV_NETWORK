package sync

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

// SyncOrchestrator coordinates all synchronization operations
type SyncOrchestrator struct {
	TestnetRoot      string
	ProductionRoot   string
	SyncManager      *SyncManager
	Monitor          *SyncMonitor
	RollbackManager  *RollbackManager
	Config           *SyncConfig
	Logger           *log.Logger
}

// OrchestrationPlan defines a complete synchronization plan
type OrchestrationPlan struct {
	ID                string                 `json:"id"`
	Timestamp         time.Time              `json:"timestamp"`
	Description       string                 `json:"description"`
	PreSyncSnapshot   string                 `json:"pre_sync_snapshot"`
	SyncOperations    []SyncOperation        `json:"sync_operations"`
	ValidationChecks  []ValidationCheck      `json:"validation_checks"`
	RollbackPlan      *RollbackPlan          `json:"rollback_plan"`
	EstimatedDuration time.Duration          `json:"estimated_duration"`
	Status            string                 `json:"status"`
}

// SyncOperation represents a single synchronization operation
type SyncOperation struct {
	Type        string   `json:"type"`
	Component   string   `json:"component"`
	Patterns    []string `json:"patterns"`
	Priority    int      `json:"priority"`
	Dependencies []string `json:"dependencies"`
	Status      string   `json:"status"`
}

// ValidationCheck represents a validation check to perform
type ValidationCheck struct {
	Type        string `json:"type"`
	Component   string `json:"component"`
	Pattern     string `json:"pattern"`
	Required    bool   `json:"required"`
	Status      string `json:"status"`
}

// RollbackPlan defines rollback procedures
type RollbackPlan struct {
	TriggerConditions []string `json:"trigger_conditions"`
	Steps            []string `json:"steps"`
	SnapshotID       string   `json:"snapshot_id"`
}

// NewSyncOrchestrator creates a new synchronization orchestrator
func NewSyncOrchestrator(testnetRoot, productionRoot string) *SyncOrchestrator {
	logger := log.New(os.Stdout, "[ORCHESTRATOR] ", log.LstdFlags)
	
	syncManager := NewSyncManager(testnetRoot, productionRoot)
	monitor := NewSyncMonitor(testnetRoot, productionRoot, filepath.Join(testnetRoot, "sync", "reports"))
	rollbackManager := NewRollbackManager(testnetRoot, filepath.Join(testnetRoot, "sync", "backups"))
	
	return &SyncOrchestrator{
		TestnetRoot:     testnetRoot,
		ProductionRoot:  productionRoot,
		SyncManager:     syncManager,
		Monitor:         monitor,
		RollbackManager: rollbackManager,
		Logger:          logger,
	}
}

// LoadConfiguration loads the synchronization configuration
func (so *SyncOrchestrator) LoadConfiguration(configPath string) error {
	if err := so.SyncManager.LoadConfig(configPath); err != nil {
		return fmt.Errorf("failed to load sync configuration: %w", err)
	}
	
	so.Config = so.SyncManager.SyncConfig
	so.Logger.Printf("Configuration loaded with %d components", len(so.Config.Components))
	
	return nil
}

// CreateOrchestrationPlan creates a comprehensive synchronization plan
func (so *SyncOrchestrator) CreateOrchestrationPlan(description string) (*OrchestrationPlan, error) {
	so.Logger.Printf("Creating orchestration plan: %s", description)
	
	plan := &OrchestrationPlan{
		ID:          fmt.Sprintf("plan-%s", time.Now().Format("20060102-150405")),
		Timestamp:   time.Now(),
		Description: description,
		Status:      "created",
	}
	
	// Create sync operations for each enabled component
	for _, component := range so.Config.Components {
		if !component.Enabled {
			continue
		}
		
		// Script synchronization operation
		scriptOp := SyncOperation{
			Type:      "scripts",
			Component: component.Name,
			Patterns:  []string{},
			Priority:  1,
			Status:    "pending",
		}
		
		for _, pattern := range so.Config.ScriptPatterns {
			if pattern.Enabled {
				scriptOp.Patterns = append(scriptOp.Patterns, pattern.Name)
			}
		}
		
		plan.SyncOperations = append(plan.SyncOperations, scriptOp)
		
		// Test synchronization operation
		testOp := SyncOperation{
			Type:         "tests",
			Component:    component.Name,
			Patterns:     []string{},
			Priority:     2,
			Dependencies: []string{scriptOp.Component + "-scripts"},
			Status:       "pending",
		}
		
		for _, pattern := range so.Config.TestPatterns {
			if pattern.Enabled {
				testOp.Patterns = append(testOp.Patterns, pattern.Name)
			}
		}
		
		plan.SyncOperations = append(plan.SyncOperations, testOp)

		// API synchronization operation (global, not per component)
		apiOp := SyncOperation{
			Type:      "api",
			Component: "global",
			Patterns:  []string{},
			Priority:  3,
			Status:    "pending",
		}

		for _, pattern := range so.Config.APIPatterns {
			if pattern.Enabled {
				apiOp.Patterns = append(apiOp.Patterns, pattern.Name)
			}
		}

		if len(apiOp.Patterns) > 0 {
			plan.SyncOperations = append(plan.SyncOperations, apiOp)
		}
	}
	
	// Create validation checks
	for _, component := range so.Config.Components {
		if !component.Enabled {
			continue
		}
		
		// Script validation
		scriptCheck := ValidationCheck{
			Type:      "script_validation",
			Component: component.Name,
			Pattern:   "all",
			Required:  true,
			Status:    "pending",
		}
		plan.ValidationChecks = append(plan.ValidationChecks, scriptCheck)
		
		// Test validation
		testCheck := ValidationCheck{
			Type:      "test_validation",
			Component: component.Name,
			Pattern:   "all",
			Required:  true,
			Status:    "pending",
		}
		plan.ValidationChecks = append(plan.ValidationChecks, testCheck)
	}
	
	// Create rollback plan
	plan.RollbackPlan = &RollbackPlan{
		TriggerConditions: []string{
			"validation_failure",
			"sync_error",
			"manual_trigger",
		},
		Steps: []string{
			"stop_sync_operations",
			"restore_from_snapshot",
			"validate_rollback",
			"notify_completion",
		},
	}
	
	// Estimate duration (simplified calculation)
	plan.EstimatedDuration = time.Duration(len(plan.SyncOperations)*2) * time.Minute
	
	so.Logger.Printf("Orchestration plan created with %d operations and %d validation checks",
		len(plan.SyncOperations), len(plan.ValidationChecks))
	
	return plan, nil
}

// ExecuteOrchestrationPlan executes a complete synchronization plan
func (so *SyncOrchestrator) ExecuteOrchestrationPlan(plan *OrchestrationPlan) error {
	so.Logger.Printf("Executing orchestration plan: %s", plan.ID)
	
	plan.Status = "executing"
	
	// Step 1: Create pre-sync snapshot
	so.Logger.Println("Creating pre-sync snapshot...")
	snapshot, err := so.RollbackManager.CreateSnapshot(
		fmt.Sprintf("Pre-sync snapshot for plan %s", plan.ID),
		[]string{},
	)
	if err != nil {
		so.Logger.Printf("Warning: Failed to create pre-sync snapshot: %v", err)
	} else {
		plan.PreSyncSnapshot = snapshot.ID
		plan.RollbackPlan.SnapshotID = snapshot.ID
		so.Logger.Printf("Pre-sync snapshot created: %s", snapshot.ID)
	}
	
	// Step 2: Execute sync operations in priority order
	so.Logger.Println("Executing synchronization operations...")
	
	// Group operations by priority
	priorityGroups := make(map[int][]SyncOperation)
	for _, op := range plan.SyncOperations {
		priorityGroups[op.Priority] = append(priorityGroups[op.Priority], op)
	}
	
	// Execute operations by priority
	for priority := 1; priority <= 3; priority++ {
		operations := priorityGroups[priority]
		if len(operations) == 0 {
			continue
		}
		
		so.Logger.Printf("Executing priority %d operations (%d operations)", priority, len(operations))
		
		for i, op := range operations {
			so.Logger.Printf("Executing operation %d/%d: %s for %s", i+1, len(operations), op.Type, op.Component)
			
			if err := so.executeOperation(op); err != nil {
				so.Logger.Printf("Operation failed: %v", err)
				plan.Status = "failed"
				
				// Trigger rollback
				if err := so.triggerRollback(plan); err != nil {
					so.Logger.Printf("Rollback failed: %v", err)
				}
				
				return fmt.Errorf("operation failed: %w", err)
			}
			
			plan.SyncOperations[i].Status = "completed"
		}
	}
	
	// Step 3: Execute validation checks
	so.Logger.Println("Executing validation checks...")
	
	validationReport, err := so.Monitor.ValidateSync(so.Config)
	if err != nil {
		so.Logger.Printf("Validation failed: %v", err)
		plan.Status = "validation_failed"
		
		// Trigger rollback
		if err := so.triggerRollback(plan); err != nil {
			so.Logger.Printf("Rollback failed: %v", err)
		}
		
		return fmt.Errorf("validation failed: %w", err)
	}
	
	// Check validation results
	if validationReport.InvalidComponents > 0 {
		so.Logger.Printf("Validation found %d invalid components", validationReport.InvalidComponents)
		plan.Status = "validation_failed"
		
		// Trigger rollback
		if err := so.triggerRollback(plan); err != nil {
			so.Logger.Printf("Rollback failed: %v", err)
		}
		
		return fmt.Errorf("validation found %d invalid components", validationReport.InvalidComponents)
	}
	
	// Step 4: Mark all validation checks as completed
	for i := range plan.ValidationChecks {
		plan.ValidationChecks[i].Status = "completed"
	}
	
	plan.Status = "completed"
	so.Logger.Printf("Orchestration plan completed successfully: %s", plan.ID)
	
	return nil
}

// executeOperation executes a single synchronization operation
func (so *SyncOrchestrator) executeOperation(op SyncOperation) error {
	switch op.Type {
	case "scripts":
		_, err := so.SyncManager.SyncScriptPatterns()
		return err
	case "tests":
		_, err := so.SyncManager.SyncTestPatterns()
		return err
	case "api":
		_, err := so.SyncManager.SyncAPIPatterns()
		return err
	default:
		return fmt.Errorf("unknown operation type: %s", op.Type)
	}
}

// triggerRollback triggers the rollback procedure
func (so *SyncOrchestrator) triggerRollback(plan *OrchestrationPlan) error {
	if plan.RollbackPlan.SnapshotID == "" {
		return fmt.Errorf("no rollback snapshot available")
	}
	
	so.Logger.Printf("Triggering rollback to snapshot: %s", plan.RollbackPlan.SnapshotID)
	
	_, err := so.RollbackManager.RollbackToSnapshot(plan.RollbackPlan.SnapshotID)
	if err != nil {
		return fmt.Errorf("rollback failed: %w", err)
	}
	
	so.Logger.Println("Rollback completed successfully")
	return nil
}

// SaveOrchestrationPlan saves an orchestration plan to disk
func (so *SyncOrchestrator) SaveOrchestrationPlan(plan *OrchestrationPlan, outputPath string) error {
	data, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal plan: %w", err)
	}
	
	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write plan: %w", err)
	}
	
	so.Logger.Printf("Orchestration plan saved: %s", outputPath)
	return nil
}

// LoadOrchestrationPlan loads an orchestration plan from disk
func (so *SyncOrchestrator) LoadOrchestrationPlan(planPath string) (*OrchestrationPlan, error) {
	data, err := os.ReadFile(planPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read plan: %w", err)
	}
	
	var plan OrchestrationPlan
	if err := json.Unmarshal(data, &plan); err != nil {
		return nil, fmt.Errorf("failed to unmarshal plan: %w", err)
	}
	
	so.Logger.Printf("Orchestration plan loaded: %s", plan.ID)
	return &plan, nil
}

// GetSynchronizationStatus returns the current synchronization status
func (so *SyncOrchestrator) GetSynchronizationStatus() (*MonitoringReport, error) {
	return so.Monitor.ValidateSync(so.Config)
}

// PerformHealthCheck performs a comprehensive health check
func (so *SyncOrchestrator) PerformHealthCheck() error {
	so.Logger.Println("Performing comprehensive health check...")
	
	// Check configuration
	if so.Config == nil {
		return fmt.Errorf("configuration not loaded")
	}
	
	// Check directories
	if _, err := os.Stat(so.TestnetRoot); os.IsNotExist(err) {
		return fmt.Errorf("testnet root directory not found: %s", so.TestnetRoot)
	}
	
	if _, err := os.Stat(so.ProductionRoot); os.IsNotExist(err) {
		return fmt.Errorf("production root directory not found: %s", so.ProductionRoot)
	}
	
	// Check component availability
	for _, component := range so.Config.Components {
		if !component.Enabled {
			continue
		}
		
		componentPath := filepath.Join(so.ProductionRoot, component.ProductionPath)
		if _, err := os.Stat(componentPath); os.IsNotExist(err) {
			so.Logger.Printf("Warning: Component not found: %s", componentPath)
		}
	}
	
	so.Logger.Println("Health check completed successfully")
	return nil
}
