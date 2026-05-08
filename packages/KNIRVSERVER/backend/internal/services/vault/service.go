package vault

import (
	"time"

	"backend_server/internal/storage/mdstorage"
)

// ErrorNode represents a documented failure state in the network
type ErrorNode struct {
	ID          string                 `json:"id"`
	Description string                 `json:"description"`
	Context     map[string]interface{} `json:"context"`
	Timestamp   time.Time              `json:"timestamp"`
	BadgeID     string                 `json:"badge_id,omitempty"` // active badge at failure time (Gap 4)
}

// SolutionNode represents executable logic to resolve an ErrorNode
type SolutionNode struct {
	ID        string                 `json:"id"`
	ErrorID   string                 `json:"error_id"` // Reference to the ErrorNode it solves
	Language  string                 `json:"language"` // e.g., "python", "javascript", "go"
	Code      string                 `json:"code"`     // The executable logic
	Metadata  map[string]interface{} `json:"metadata"`
	Timestamp time.Time              `json:"timestamp"`
}

// VaultService manages the lifecycle of Error and Solution nodes
type VaultService struct {
	storage *mdstorage.MarkdownStorageDriver
}

// NewVaultService creates a new instance of the VaultService
func NewVaultService(storage *mdstorage.MarkdownStorageDriver) *VaultService {
	return &VaultService{
		storage: storage,
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
		Content: []byte(errNode.Description), // Content could be more detailed
	}
	return s.storage.SaveDocument(doc)
}

// RegisterSolution saves a new SolutionNode to the encrypted Markdown vault
func (s *VaultService) RegisterSolution(solNode *SolutionNode) error {
	// We store the code in the encrypted content block of the Markdown file
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

	return sol, nil
}
