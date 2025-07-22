package services

import (
	"context"
	"crypto-wallet-backend/internal/config"
	"crypto-wallet-backend/internal/models"
	"crypto-wallet-backend/pkg/logger"
	"crypto-wallet-backend/pkg/wasm"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AIAgentService struct {
	db       *gorm.DB
	config   *config.Config
	logger   logger.Logger
	runtime  *wasm.Runtime
}

func NewAIAgentService(db *gorm.DB, cfg *config.Config, logger logger.Logger) *AIAgentService {
	runtime := wasm.NewRuntime(cfg)
	
	return &AIAgentService{
		db:      db,
		config:  cfg,
		logger:  logger,
		runtime: runtime,
	}
}

type CreateAgentRequest struct {
	Name          string   `json:"name" binding:"required"`
	Description   string   `json:"description"`
	Category      string   `json:"category" binding:"required"`
	Code          []byte   `json:"code" binding:"required"`
	Permissions   []string `json:"permissions" binding:"required"`
	Configuration string   `json:"configuration"`
	RiskLevel     string   `json:"risk_level"`
}

type ExecuteAgentRequest struct {
	Input map[string]interface{} `json:"input"`
}

func (s *AIAgentService) CreateAgent(ctx context.Context, userID uuid.UUID, req *CreateAgentRequest) (*models.AIAgent, error) {
	// Check user's agent limit
	var count int64
	if err := s.db.Model(&models.AIAgent{}).Where("user_id = ?", userID).Count(&count).Error; err != nil {
		return nil, fmt.Errorf("failed to count user agents: %w", err)
	}

	if count >= int64(s.config.MaxAgentsPerUser) {
		return nil, fmt.Errorf("maximum number of agents reached")
	}

	// Validate permissions
	if err := s.validatePermissions(req.Permissions); err != nil {
		return nil, fmt.Errorf("invalid permissions: %w", err)
	}

	// Validate and compile WASM code
	codeHash, err := s.runtime.ValidateCode(req.Code)
	if err != nil {
		return nil, fmt.Errorf("invalid WASM code: %w", err)
	}

	// Encrypt the code
	encryptedCode, err := s.encryptCode(req.Code)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt code: %w", err)
	}

	agent := &models.AIAgent{
		UserID:        userID,
		Name:          req.Name,
		Description:   req.Description,
		Category:      req.Category,
		Version:       "1.0.0",
		Status:        "inactive",
		CodeHash:      codeHash,
		EncryptedCode: encryptedCode,
		Permissions:   req.Permissions,
		Configuration: req.Configuration,
		RiskLevel:     req.RiskLevel,
	}

	if err := s.db.Create(agent).Error; err != nil {
		return nil, fmt.Errorf("failed to create agent: %w", err)
	}

	s.logger.Info("Created AI agent", "agent_id", agent.ID, "user_id", userID)
	return agent, nil
}

func (s *AIAgentService) ExecuteAgent(ctx context.Context, agentID uuid.UUID, req *ExecuteAgentRequest) (*models.AgentExecution, error) {
	// Get agent
	var agent models.AIAgent
	if err := s.db.First(&agent, "id = ?", agentID).Error; err != nil {
		return nil, fmt.Errorf("agent not found: %w", err)
	}

	if agent.Status != "active" {
		return nil, fmt.Errorf("agent is not active")
	}

	// Create execution record
	execution := &models.AgentExecution{
		AgentID:   agentID,
		Status:    "running",
		StartTime: time.Now(),
	}

	inputJSON, _ := json.Marshal(req.Input)
	execution.Input = string(inputJSON)

	if err := s.db.Create(execution).Error; err != nil {
		return nil, fmt.Errorf("failed to create execution record: %w", err)
	}

	// Execute in goroutine
	go s.executeAgentAsync(ctx, &agent, execution, req.Input)

	return execution, nil
}

func (s *AIAgentService) executeAgentAsync(ctx context.Context, agent *models.AIAgent, execution *models.AgentExecution, input map[string]interface{}) {
	startTime := time.Now()
	
	// Decrypt code
	code, err := s.decryptCode(agent.EncryptedCode)
	if err != nil {
		s.updateExecutionError(execution, "Failed to decrypt code", startTime)
		return
	}

	// Execute WASM code with resource limits
	result, err := s.runtime.Execute(ctx, code, input, &wasm.ExecutionLimits{
		MemoryLimit:     s.config.AgentMemoryLimit,
		CPULimit:        s.config.AgentCPULimit,
		NetworkTimeout:  time.Duration(s.config.AgentNetworkTimeout) * time.Second,
		Permissions:     agent.Permissions,
	})

	duration := time.Since(startTime)
	
	if err != nil {
		s.updateExecutionError(execution, err.Error(), startTime)
		return
	}

	// Update execution record
	endTime := time.Now()
	outputJSON, _ := json.Marshal(result.Output)
	
	updates := map[string]interface{}{
		"status":       "completed",
		"end_time":     &endTime,
		"duration":     duration.Milliseconds(),
		"memory_used":  result.MemoryUsed,
		"cpu_used":     result.CPUUsed,
		"output":       string(outputJSON),
	}

	s.db.Model(execution).Updates(updates)
	s.logger.Info("Agent execution completed", "execution_id", execution.ID, "duration", duration)
}

func (s *AIAgentService) updateExecutionError(execution *models.AgentExecution, errorMsg string, startTime time.Time) {
	endTime := time.Now()
	duration := endTime.Sub(startTime)
	
	updates := map[string]interface{}{
		"status":        "failed",
		"end_time":      &endTime,
		"duration":      duration.Milliseconds(),
		"error_message": errorMsg,
	}

	s.db.Model(execution).Updates(updates)
	s.logger.Error("Agent execution failed", "execution_id", execution.ID, "error", errorMsg)
}

func (s *AIAgentService) GetUserAgents(ctx context.Context, userID uuid.UUID) ([]models.AIAgent, error) {
	var agents []models.AIAgent
	if err := s.db.Where("user_id = ?", userID).Find(&agents).Error; err != nil {
		return nil, fmt.Errorf("failed to get user agents: %w", err)
	}
	return agents, nil
}

func (s *AIAgentService) GetMarketplaceAgents(ctx context.Context, category string, limit, offset int) ([]models.AIAgent, error) {
	query := s.db.Where("is_public = ?", true)
	
	if category != "" {
		query = query.Where("category = ?", category)
	}

	var agents []models.AIAgent
	if err := query.Limit(limit).Offset(offset).Find(&agents).Error; err != nil {
		return nil, fmt.Errorf("failed to get marketplace agents: %w", err)
	}
	
	return agents, nil
}

func (s *AIAgentService) ActivateAgent(ctx context.Context, agentID uuid.UUID) error {
	return s.db.Model(&models.AIAgent{}).Where("id = ?", agentID).Update("status", "active").Error
}

func (s *AIAgentService) DeactivateAgent(ctx context.Context, agentID uuid.UUID) error {
	return s.db.Model(&models.AIAgent{}).Where("id = ?", agentID).Update("status", "inactive").Error
}

func (s *AIAgentService) validatePermissions(permissions []string) error {
	validPermissions := make(map[string]bool)
	for _, perm := range models.DefaultPermissions {
		validPermissions[perm.Name] = true
	}

	for _, perm := range permissions {
		if !validPermissions[perm] {
			return fmt.Errorf("invalid permission: %s", perm)
		}
	}

	return nil
}

func (s *AIAgentService) encryptCode(code []byte) ([]byte, error) {
	// Implement encryption logic here
	// For now, return as-is (in production, use proper encryption)
	return code, nil
}

func (s *AIAgentService) decryptCode(encryptedCode []byte) ([]byte, error) {
	// Implement decryption logic here
	// For now, return as-is (in production, use proper decryption)
	return encryptedCode, nil
}