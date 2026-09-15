// agent_inferencer_service.go
package agentify

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"
)

// AgentInferencerService provides a service interface for the Agent Inferencer
type AgentInferencerService struct {
	inferencer *AgentInferencer
	httpAPI    *AgentHTTPAPI
	isRunning  bool
	mutex      sync.RWMutex
}

// NewAgentInferencerService creates a new Agent Inferencer Service
func NewAgentInferencerService(pluginsDir string) *AgentInferencerService {
	inferencer := NewAgentInferencer(pluginsDir)
	httpAPI := NewAgentHTTPAPI(inferencer)

	return &AgentInferencerService{
		inferencer: inferencer,
		httpAPI:    httpAPI,
		isRunning:  false,
	}
}

// Start starts the Agent Inferencer Service
func (s *AgentInferencerService) Start() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.isRunning {
		return nil
	}

	// Start the service
	log.Println("Starting Agent Inferencer Service...")

	s.isRunning = true
	log.Println("Agent Inferencer Service started")

	return nil
}

// Stop stops the Agent Inferencer Service
func (s *AgentInferencerService) Stop() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if !s.isRunning {
		return nil
	}

	// Stop the service
	log.Println("Stopping Agent Inferencer Service...")

	s.isRunning = false
	log.Println("Agent Inferencer Service stopped")

	return nil
}

// IsRunning checks if the service is running
func (s *AgentInferencerService) IsRunning() bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return s.isRunning
}

// GetName returns the name of the service
func (s *AgentInferencerService) GetName() string {
	return "AgentInferencerService"
}

// GetHTTPAPI returns the HTTP API for the service
func (s *AgentInferencerService) GetHTTPAPI() *AgentHTTPAPI {
	return s.httpAPI
}

// ActivateAgent activates an agent for a session
func (s *AgentInferencerService) ActivateAgent(ctx context.Context, agentID, version, sessionID string, config map[string]interface{}) error {
	s.mutex.RLock()
	if !s.isRunning {
		s.mutex.RUnlock()
		return errors.New("service not running")
	}
	s.mutex.RUnlock()

	return s.inferencer.ActivateAgent(ctx, agentID, version, sessionID, config)
}

// DeactivateAgent deactivates an agent for a session
func (s *AgentInferencerService) DeactivateAgent(ctx context.Context, sessionID string) error {
	s.mutex.RLock()
	if !s.isRunning {
		s.mutex.RUnlock()
		return errors.New("service not running")
	}
	s.mutex.RUnlock()

	return s.inferencer.DeactivateAgent(ctx, sessionID)
}

// ProcessInference processes an inference request
func (s *AgentInferencerService) ProcessInference(ctx context.Context, sessionID string, input string, history []*ConversationMessage, parameters map[string]interface{}) (*InferenceResponse, error) {
	s.mutex.RLock()
	if !s.isRunning {
		s.mutex.RUnlock()
		return nil, errors.New("service not running")
	}
	s.mutex.RUnlock()

	request := &InferenceRequest{
		Input:      input,
		History:    history,
		SessionID:  sessionID,
		Parameters: parameters,
	}

	return s.inferencer.ProcessInference(ctx, sessionID, request)
}

// ListAvailableAgents lists the available agents
func (s *AgentInferencerService) ListAvailableAgents(ctx context.Context) ([]string, error) {
	s.mutex.RLock()
	if !s.isRunning {
		s.mutex.RUnlock()
		return nil, errors.New("service not running")
	}
	s.mutex.RUnlock()

	return s.inferencer.ListAvailableAgents(ctx)
}

// GetAgentCapabilities gets the capabilities of an agent
func (s *AgentInferencerService) GetAgentCapabilities(ctx context.Context, sessionID string) (*AgentCapabilities, error) {
	s.mutex.RLock()
	if !s.isRunning {
		s.mutex.RUnlock()
		return nil, errors.New("service not running")
	}
	s.mutex.RUnlock()

	return s.inferencer.GetAgentCapabilities(ctx, sessionID)
}

// GetAgentSchema gets the schema of an agent
func (s *AgentInferencerService) GetAgentSchema(ctx context.Context, sessionID string) (*AgentSchema, error) {
	s.mutex.RLock()
	if !s.isRunning {
		s.mutex.RUnlock()
		return nil, errors.New("service not running")
	}
	s.mutex.RUnlock()

	return s.inferencer.GetAgentSchema(ctx, sessionID)
}

// GetAgentMemory gets a value from the agent's memory
func (s *AgentInferencerService) GetAgentMemory(ctx context.Context, sessionID string, key string) (interface{}, error) {
	s.mutex.RLock()
	if !s.isRunning {
		s.mutex.RUnlock()
		return nil, errors.New("service not running")
	}
	s.mutex.RUnlock()

	return s.inferencer.GetAgentMemory(ctx, sessionID, key)
}

// SetAgentMemory sets a value in the agent's memory
func (s *AgentInferencerService) SetAgentMemory(ctx context.Context, sessionID string, key string, value interface{}) error {
	s.mutex.RLock()
	if !s.isRunning {
		s.mutex.RUnlock()
		return errors.New("service not running")
	}
	s.mutex.RUnlock()

	return s.inferencer.SetAgentMemory(ctx, sessionID, key, value)
}

// GetTEEInfo gets information about the TEE for an agent
func (s *AgentInferencerService) GetTEEInfo(ctx context.Context, sessionID string) (map[string]interface{}, error) {
	s.mutex.RLock()
	if !s.isRunning {
		s.mutex.RUnlock()
		return nil, errors.New("service not running")
	}
	s.mutex.RUnlock()

	return s.inferencer.GetTEEInfo(ctx, sessionID)
}

// GenerateText generates text using an agent
func (s *AgentInferencerService) GenerateText(prompt string) (string, error) {
	s.mutex.RLock()
	if !s.isRunning {
		s.mutex.RUnlock()
		return "", errors.New("service not running")
	}
	s.mutex.RUnlock()

	// Create a session ID for this request
	sessionID := fmt.Sprintf("generate-text-%d", time.Now().UnixNano())

	// Use the default agent for text generation
	ctx := context.Background()
	if err := s.inferencer.ActivateAgent(ctx, "default", "latest", sessionID, nil); err != nil {
		return "", fmt.Errorf("failed to activate default agent: %v", err)
	}
	defer s.inferencer.DeactivateAgent(ctx, sessionID)

	// Process the inference request
	request := &InferenceRequest{
		Input:     prompt,
		SessionID: sessionID,
	}

	response, err := s.inferencer.ProcessInference(ctx, sessionID, request)
	if err != nil {
		return "", fmt.Errorf("failed to process inference: %v", err)
	}

	return response.Output, nil
}

// Generate generates text using an agent with more options
func (s *AgentInferencerService) Generate(ctx context.Context, prompt string, opts ...interface{}) (string, error) {
	s.mutex.RLock()
	if !s.isRunning {
		s.mutex.RUnlock()
		return "", errors.New("service not running")
	}
	s.mutex.RUnlock()

	// Create a session ID for this request
	sessionID := fmt.Sprintf("generate-%d", time.Now().UnixNano())

	// Extract options
	var agentID, version string
	parameters := make(map[string]interface{})

	for _, opt := range opts {
		switch v := opt.(type) {
		case map[string]interface{}:
			// Merge parameters
			for k, val := range v {
				parameters[k] = val
			}
		case string:
			// First string is agent ID, second is version
			if agentID == "" {
				agentID = v
			} else if version == "" {
				version = v
			}
		}
	}

	// Use default values if not provided
	if agentID == "" {
		agentID = "default"
	}
	if version == "" {
		version = "latest"
	}

	// Activate the agent
	if err := s.inferencer.ActivateAgent(ctx, agentID, version, sessionID, parameters); err != nil {
		return "", fmt.Errorf("failed to activate agent: %v", err)
	}
	defer s.inferencer.DeactivateAgent(ctx, sessionID)

	// Process the inference request
	request := &InferenceRequest{
		Input:      prompt,
		SessionID:  sessionID,
		Parameters: parameters,
	}

	response, err := s.inferencer.ProcessInference(ctx, sessionID, request)
	if err != nil {
		return "", fmt.Errorf("failed to process inference: %v", err)
	}

	return response.Output, nil
}
