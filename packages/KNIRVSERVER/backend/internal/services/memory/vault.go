package memory

import (
	"fmt"

	"backend_server/internal/storage/mdstorage"
	"go.uber.org/zap"
)

// VaultService manages the lifecycle of Error and Solution nodes
type VaultService struct {
	storage *mdstorage.MarkdownStorageDriver
	logger  *zap.Logger
}

// NewVaultService creates a new instance of the VaultService
func NewVaultService(storage *mdstorage.MarkdownStorageDriver, logger *zap.Logger) *VaultService {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &VaultService{
		storage: storage,
		logger:  logger,
	}
}

// RegisterError saves a new ErrorNode to the encrypted Markdown vault
func (s *VaultService) RegisterError(errNode *ErrorNode) error {
	doc := &mdstorage.MarkdownDocument{
		ID:        errNode.ID,
		Type:      "ERROR",
		Timestamp: errNode.Timestamp,
		Metadata: map[string]interface{}{
			"description": errNode.Description,
			"context":     errNode.Context,
		},
		Content: []byte(errNode.Description),
	}
	return s.storage.SaveDocument(doc)
}

// RegisterSolution saves a new SolutionNode to the encrypted Markdown vault
func (s *VaultService) RegisterSolution(solNode *SolutionNode) error {
	doc := &mdstorage.MarkdownDocument{
		ID:        solNode.ID,
		Type:      "SOLUTION",
		Timestamp: solNode.Timestamp,
		Metadata: map[string]interface{}{
			"error_id": solNode.ErrorID,
			"language": solNode.Language,
			"metadata": solNode.Metadata,
		},
		Content: []byte(solNode.Code),
	}
	return s.storage.SaveDocument(doc)
}

// GetSolution retrieves and decrypts a solution node
func (s *VaultService) GetSolution(id string) (*SolutionNode, error) {
	doc, err := s.storage.LoadDocument("SOLUTION", id)
	if err != nil {
		return nil, err
	}

	sol := &SolutionNode{
		ID:        doc.ID,
		Code:      string(doc.Content),
		Timestamp: doc.Timestamp,
	}

	if eid, ok := doc.Metadata["error_id"].(string); ok {
		sol.ErrorID = eid
	}
	if lang, ok := doc.Metadata["language"].(string); ok {
		sol.Language = lang
	}
	if metadata, ok := doc.Metadata["metadata"].(map[string]interface{}); ok {
		sol.Metadata = metadata
	}

	return sol, nil
}

// ExecuteSolution retrieves and executes logic from a SolutionNode
func (s *VaultService) ExecuteSolution(id string, params map[string]interface{}) (string, error) {
	sol, err := s.GetSolution(id)
	if err != nil {
		return "", fmt.Errorf("solution not found: %w", err)
	}

	s.logger.Info("executing solution", zap.String("solution_id", id), zap.String("language", sol.Language))

	result, err := s.executeCode(sol.Code, sol.Language, params)
	if err != nil {
		return "", fmt.Errorf("execution failed: %w", err)
	}

	return result, nil
}

// executeCode executes code in the specified language with parameters
func (s *VaultService) executeCode(code, language string, params map[string]interface{}) (string, error) {
	switch language {
	case "go":
		return s.executeGoCode(code, params)
	case "python":
		return s.executePythonCode(code, params)
	case "javascript", "js":
		return s.executeJavaScript(code, params)
	default:
		return "", fmt.Errorf("unsupported language: %s", language)
	}
}

// executeGoCode executes Go code (placeholder - would use gops or similar)
func (s *VaultService) executeGoCode(code string, params map[string]interface{}) (string, error) {
	s.logger.Debug("executing Go code", zap.String("code", code[:min(100, len(code))]))
	return fmt.Sprintf("Go execution result for code length: %d", len(code)), nil
}

// executePythonCode executes Python code (placeholder - would use exec or similar)
func (s *VaultService) executePythonCode(code string, params map[string]interface{}) (string, error) {
	s.logger.Debug("executing Python code", zap.String("code", code[:min(100, len(code))]))
	return fmt.Sprintf("Python execution result for code length: %d", len(code)), nil
}

// executeJavaScript executes JavaScript code (placeholder - would use otto or similar)
func (s *VaultService) executeJavaScript(code string, params map[string]interface{}) (string, error) {
	s.logger.Debug("executing JavaScript code", zap.String("code", code[:min(100, len(code))]))
	return fmt.Sprintf("JavaScript execution result for code length: %d", len(code)), nil
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
