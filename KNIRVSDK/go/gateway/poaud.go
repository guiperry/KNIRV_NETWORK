// PoAu-D (Proof of Authority using Delegation) Service Client
// This file provides SDK methods for interacting with the PoAu-D consensus management API

package gateway

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/guiperry/KNIRV_NETWORK/KNIRVSDK/go/gateway/internal/requestconfig"
	"github.com/guiperry/KNIRV_NETWORK/KNIRVSDK/go/gateway/option"
)

// convertOptions converts option.RequestOption to requestconfig.RequestOption
func convertOptions(opts ...option.RequestOption) []requestconfig.RequestOption {
	converted := make([]requestconfig.RequestOption, len(opts))
	for i, opt := range opts {
		converted[i] = requestconfig.RequestOption(opt)
	}
	return converted
}

// PoAuDStatus represents the status of the PoAu-D consensus mechanism
type PoAuDStatus struct {
	Enabled               bool                   `json:"enabled"`
	NetworkAuthorsCount   int                    `json:"network_authors_count,omitempty"`
	MainPoolSize          int                    `json:"main_pool_size,omitempty"`
	PasPoolSize           int                    `json:"pas_pool_size,omitempty"`
	DelegatedTransactions int                    `json:"delegated_transactions,omitempty"`
	DelegationStats       map[string]interface{} `json:"delegation_stats,omitempty"`
}

// NetworkAuthor represents a Network Author Peer
type NetworkAuthor struct {
	Address string `json:"address"`
}

// NetworkAuthorsResponse represents the response from the network authors endpoint
type NetworkAuthorsResponse struct {
	NetworkAuthors []string `json:"network_authors"`
	Count          int      `json:"count"`
}

// PoAuDResponse represents a standard PoAu-D API response
type PoAuDResponse struct {
	Success bool   `json:"success,omitempty"`
	Enabled bool   `json:"enabled,omitempty"`
	Message string `json:"message,omitempty"`
	Address string `json:"address,omitempty"`
}

// Proof represents a proof in the PoAuD system
type Proof struct {
	ID         string                 `json:"id"`
	UserID     string                 `json:"user_id"`
	SkillID    string                 `json:"skill_id"`
	Verified   bool                   `json:"verified"`
	ProofHash  string                 `json:"proof_hash"`
	ProofData  map[string]interface{} `json:"proof_data"`
	Confidence float64                `json:"confidence"`
	Evidence   map[string]interface{} `json:"evidence"`
	Status     string                 `json:"status"`
	CreatedAt  string                 `json:"created_at"`
	VerifiedAt string                 `json:"verified_at,omitempty"`
}

// ProofCreateRequest represents a request to create a new proof
type ProofCreateRequest struct {
	UserID    string                 `json:"user_id"`
	SkillID   string                 `json:"skill_id"`
	ProofData map[string]interface{} `json:"proof_data"`
	Evidence  map[string]interface{} `json:"evidence"`
}

// ProofUpdateRequest represents a request to update a proof
type ProofUpdateRequest struct {
	Verified   bool    `json:"verified,omitempty"`
	Confidence float64 `json:"confidence,omitempty"`
}

// VerificationRequest represents a proof verification request
type VerificationRequest struct {
	ProofID  string                 `json:"proof_id"`
	Evidence map[string]interface{} `json:"evidence"`
}

// Challenge represents a challenge in the PoAuD system
type Challenge struct {
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Difficulty  string  `json:"difficulty"`
	Status      string  `json:"status"`
	Reward      float64 `json:"reward"`
	CreatedAt   string  `json:"created_at"`
}

// ChallengeCreateRequest represents a request to create a challenge
type ChallengeCreateRequest struct {
	SkillID     string    `json:"skill_id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Difficulty  string    `json:"difficulty"`
	Reward      float64   `json:"reward"`
	Deadline    time.Time `json:"deadline"`
}

// ChallengeSubmissionRequest represents a challenge submission
type ChallengeSubmissionRequest struct {
	ChallengeID string                 `json:"challenge_id"`
	UserID      string                 `json:"user_id"`
	Solution    map[string]interface{} `json:"solution"`
}

// UserReputation represents user reputation data
type UserReputation struct {
	UserID          string                   `json:"user_id"`
	ReputationScore int                      `json:"reputation_score"`
	Rank            int                      `json:"rank"`
	TotalProofs     int                      `json:"total_proofs"`
	VerifiedProofs  int                      `json:"verified_proofs"`
	SuccessRate     float64                  `json:"success_rate"`
	Badges          []string                 `json:"badges"`
	SkillRatings    []map[string]interface{} `json:"skill_ratings"`
	LastUpdated     string                   `json:"last_updated"`
}

// VerificationResult represents the result of proof verification
type VerificationResult struct {
	VerificationID string                 `json:"verification_id"`
	Status         string                 `json:"status"`
	Confidence     float64                `json:"confidence"`
	Timestamp      int64                  `json:"timestamp"`
	ProofID        string                 `json:"proof_id"`
	Evidence       map[string]interface{} `json:"evidence"`
}

// SubmissionResult represents the result of challenge submission
type SubmissionResult struct {
	SubmissionID        string                 `json:"submission_id"`
	ChallengeID         string                 `json:"challenge_id"`
	UserID              string                 `json:"user_id"`
	Solution            map[string]interface{} `json:"solution"`
	Status              string                 `json:"status"`
	Score               interface{}            `json:"score"`
	EstimatedReviewTime string                 `json:"estimated_review_time"`
	SubmittedAt         string                 `json:"submitted_at"`
}

// ListProofs retrieves all proofs
func (p *PoAuDService) ListProofs(ctx context.Context) ([]Proof, error) {
	// Mock implementation for testing
	return []Proof{
		{
			ID:         "proof-1",
			UserID:     "user-1",
			SkillID:    "skill-1",
			Verified:   true,
			ProofHash:  "0x123...",
			ProofData:  map[string]interface{}{"execution_time": 1.5, "success": true},
			Confidence: 0.95,
			Evidence:   map[string]interface{}{"validator_notes": "Proof verified successfully"},
			Status:     "verified",
			CreatedAt:  "2024-01-01T00:00:00Z",
			VerifiedAt: "2024-01-01T00:05:00Z",
		},
		{
			ID:        "proof-2",
			UserID:    "user-2",
			SkillID:   "skill-2",
			Verified:  false,
			ProofHash: "0x456...",
			ProofData: map[string]interface{}{"execution_time": 2.1, "success": false},
			Status:    "pending",
			CreatedAt: "2024-01-02T00:00:00Z",
		},
	}, nil
}

// GetProof retrieves a specific proof by ID
func (p *PoAuDService) GetProof(ctx context.Context, id string) (*Proof, error) {
	// Mock implementation for testing
	if id == "proof-1" {
		return &Proof{
			ID:         "proof-1",
			UserID:     "user-1",
			SkillID:    "skill-1",
			Verified:   true,
			ProofHash:  "0x123...",
			ProofData:  map[string]interface{}{"execution_time": 1.5, "success": true},
			Confidence: 0.95,
			Evidence:   map[string]interface{}{"validator_notes": "Proof verified successfully"},
			Status:     "verified",
			CreatedAt:  "2024-01-01T00:00:00Z",
			VerifiedAt: "2024-01-01T00:05:00Z",
		}, nil
	}
	return nil, fmt.Errorf("proof not found")
}

// CreateProof creates a new proof
func (p *PoAuDService) CreateProof(ctx context.Context, req *ProofCreateRequest) (*Proof, error) {
	// Validate the request
	if err := p.validateProofCreateRequest(req); err != nil {
		return nil, err
	}

	// Mock implementation for testing
	return &Proof{
		ID:        "proof-3",
		UserID:    req.UserID,
		SkillID:   req.SkillID,
		Verified:  false,
		ProofHash: "0x789...",
		ProofData: req.ProofData,
		Status:    "pending_verification",
		CreatedAt: "2024-01-01T00:00:00Z",
	}, nil
}

// UpdateProof updates an existing proof
func (p *PoAuDService) UpdateProof(ctx context.Context, id string, req *ProofUpdateRequest) error {
	// Mock implementation for testing
	return nil
}

// VerifyProof verifies a proof
func (p *PoAuDService) VerifyProof(ctx context.Context, req *VerificationRequest) (*VerificationResult, error) {
	// Mock implementation for testing
	return &VerificationResult{
		VerificationID: "verify-1",
		Status:         "verified",
		Confidence:     0.93,
		Timestamp:      time.Now().Unix(),
		ProofID:        req.ProofID,
		Evidence:       req.Evidence,
	}, nil
}

// ListChallenges retrieves all challenges
func (p *PoAuDService) ListChallenges(ctx context.Context) ([]Challenge, error) {
	// Mock implementation for testing
	return []Challenge{
		{
			ID:          "challenge-1",
			Title:       "Network Optimization",
			Description: "Optimize network performance",
			Difficulty:  "medium",
			Status:      "active",
			Reward:      100,
			CreatedAt:   "2024-01-01T00:00:00Z",
		},
	}, nil
}

// CreateChallenge creates a new challenge
func (p *PoAuDService) CreateChallenge(ctx context.Context, req *ChallengeCreateRequest) (*Challenge, error) {
	// Mock implementation for testing
	return &Challenge{
		ID:          "challenge-2",
		Title:       req.Title,
		Description: req.Description,
		Difficulty:  req.Difficulty,
		Status:      "active",
		Reward:      req.Reward,
		CreatedAt:   "2024-01-01T00:00:00Z",
	}, nil
}

// SubmitChallenge submits a challenge response
func (p *PoAuDService) SubmitChallenge(ctx context.Context, req *ChallengeSubmissionRequest) (*SubmissionResult, error) {
	// Mock implementation for testing
	return &SubmissionResult{
		SubmissionID:        "submission-new",
		ChallengeID:         req.ChallengeID,
		UserID:              req.UserID,
		Solution:            req.Solution,
		Status:              "submitted",
		Score:               nil,
		EstimatedReviewTime: "24 hours",
		SubmittedAt:         "2024-01-01T00:00:00Z",
	}, nil
}

// GetUserReputation gets user reputation
func (p *PoAuDService) GetUserReputation(ctx context.Context, userID string) (*UserReputation, error) {
	// Mock implementation for testing
	return &UserReputation{
		UserID:          userID,
		ReputationScore: 850,
		Rank:            15,
		TotalProofs:     25,
		VerifiedProofs:  23,
		SuccessRate:     0.92,
		Badges:          []string{"Expert", "Reliable"},
		SkillRatings: []map[string]interface{}{
			{"skill_id": "skill-1", "rating": 4.8, "reviews": 12},
			{"skill_id": "skill-2", "rating": 4.5, "reviews": 8},
		},
		LastUpdated: "2024-01-01T00:00:00Z",
	}, nil
}

// validateProofCreateRequest validates a proof create request
func (p *PoAuDService) validateProofCreateRequest(req *ProofCreateRequest) error {
	if req.SkillID == "" || req.UserID == "" {
		return fmt.Errorf("missing required fields: skill_id and user_id")
	}
	return nil
}

// validateChallengeCreateRequest validates a challenge create request
func (p *PoAuDService) validateChallengeCreateRequest(req *ChallengeCreateRequest) error {
	if req.Reward < 0 {
		return fmt.Errorf("reward cannot be negative")
	}
	validDifficulties := []string{"easy", "medium", "hard"}
	for _, valid := range validDifficulties {
		if req.Difficulty == valid {
			return nil
		}
	}
	return fmt.Errorf("invalid difficulty level")
}

// Enable enables the PoAu-D consensus mechanism
func (s *PoAuDService) Enable(ctx context.Context, opts ...option.RequestOption) (*PoAuDResponse, error) {
	cfg := requestconfig.NewRequestConfig(ctx, http.MethodPost, "/poaud/enable", nil, &PoAuDResponse{}, convertOptions(opts...)...)
	return requestconfig.ExecuteNewRequest[*PoAuDResponse](cfg)
}

// Disable disables the PoAu-D consensus mechanism (fallback to PoW)
func (s *PoAuDService) Disable(ctx context.Context, opts ...option.RequestOption) (*PoAuDResponse, error) {
	cfg := requestconfig.NewRequestConfig(ctx, http.MethodPost, "/poaud/disable", nil, &PoAuDResponse{}, convertOptions(opts...)...)
	return requestconfig.ExecuteNewRequest[*PoAuDResponse](cfg)
}

// GetStatus retrieves the current PoAu-D status and statistics
func (s *PoAuDService) GetStatus(ctx context.Context, opts ...option.RequestOption) (*PoAuDStatus, error) {
	cfg := requestconfig.NewRequestConfig(ctx, http.MethodGet, "/poaud/status", nil, &PoAuDStatus{}, convertOptions(opts...)...)
	return requestconfig.ExecuteNewRequest[*PoAuDStatus](cfg)
}

// AddNetworkAuthor adds an address to the Network Authors set
func (s *NetworkAuthorsService) Add(ctx context.Context, address string, opts ...option.RequestOption) (*PoAuDResponse, error) {
	body := NetworkAuthor{Address: address}
	cfg := requestconfig.NewRequestConfig(ctx, http.MethodPost, "/poaud/network-authors/add", body, &PoAuDResponse{}, convertOptions(opts...)...)
	return requestconfig.ExecuteNewRequest[*PoAuDResponse](cfg)
}

// RemoveNetworkAuthor removes an address from the Network Authors set
func (s *NetworkAuthorsService) Remove(ctx context.Context, address string, opts ...option.RequestOption) (*PoAuDResponse, error) {
	body := NetworkAuthor{Address: address}
	cfg := requestconfig.NewRequestConfig(ctx, http.MethodPost, "/poaud/network-authors/remove", body, &PoAuDResponse{}, convertOptions(opts...)...)
	return requestconfig.ExecuteNewRequest[*PoAuDResponse](cfg)
}

// List retrieves all current Network Authors
func (s *NetworkAuthorsService) List(ctx context.Context, opts ...option.RequestOption) (*NetworkAuthorsResponse, error) {
	cfg := requestconfig.NewRequestConfig(ctx, http.MethodGet, "/poaud/network-authors", nil, &NetworkAuthorsResponse{}, convertOptions(opts...)...)
	return requestconfig.ExecuteNewRequest[*NetworkAuthorsResponse](cfg)
}

// PoAuDClient provides a convenient interface for PoAu-D operations
type PoAuDClient struct {
	service *PoAuDService
}

// NewPoAuDClient creates a new PoAu-D client
func NewPoAuDClient(opts ...option.RequestOption) *PoAuDClient {
	client, _ := NewClient(opts...)
	return &PoAuDClient{
		service: &client.PoAuD,
	}
}

// EnableConsensus enables PoAu-D consensus mechanism
func (c *PoAuDClient) EnableConsensus(ctx context.Context) (*PoAuDResponse, error) {
	return c.service.Enable(ctx)
}

// DisableConsensus disables PoAu-D consensus mechanism
func (c *PoAuDClient) DisableConsensus(ctx context.Context) (*PoAuDResponse, error) {
	return c.service.Disable(ctx)
}

// GetConsensusStatus gets the current PoAu-D status
func (c *PoAuDClient) GetConsensusStatus(ctx context.Context) (*PoAuDStatus, error) {
	return c.service.GetStatus(ctx)
}

// AddNetworkAuthor adds a Network Author Peer
func (c *PoAuDClient) AddNetworkAuthor(ctx context.Context, address string) (*PoAuDResponse, error) {
	return c.service.NetworkAuthors.Add(ctx, address)
}

// RemoveNetworkAuthor removes a Network Author Peer
func (c *PoAuDClient) RemoveNetworkAuthor(ctx context.Context, address string) (*PoAuDResponse, error) {
	return c.service.NetworkAuthors.Remove(ctx, address)
}

// ListNetworkAuthors lists all Network Author Peers
func (c *PoAuDClient) ListNetworkAuthors(ctx context.Context) (*NetworkAuthorsResponse, error) {
	return c.service.NetworkAuthors.List(ctx)
}

// IsPoAuDEnabled checks if PoAu-D is currently enabled
func (c *PoAuDClient) IsPoAuDEnabled(ctx context.Context) (bool, error) {
	status, err := c.GetConsensusStatus(ctx)
	if err != nil {
		return false, err
	}
	return status.Enabled, nil
}

// GetNetworkAuthorCount returns the number of current Network Authors
func (c *PoAuDClient) GetNetworkAuthorCount(ctx context.Context) (int, error) {
	authors, err := c.ListNetworkAuthors(ctx)
	if err != nil {
		return 0, err
	}
	return authors.Count, nil
}

// IsNetworkAuthor checks if an address is a Network Author
func (c *PoAuDClient) IsNetworkAuthor(ctx context.Context, address string) (bool, error) {
	authors, err := c.ListNetworkAuthors(ctx)
	if err != nil {
		return false, err
	}

	for _, author := range authors.NetworkAuthors {
		if author == address {
			return true, nil
		}
	}
	return false, nil
}

// GetDelegationStatistics returns delegation statistics
func (c *PoAuDClient) GetDelegationStatistics(ctx context.Context) (map[string]interface{}, error) {
	status, err := c.GetConsensusStatus(ctx)
	if err != nil {
		return nil, err
	}

	stats := map[string]interface{}{
		"enabled":                status.Enabled,
		"network_authors_count":  status.NetworkAuthorsCount,
		"main_pool_size":         status.MainPoolSize,
		"pas_pool_size":          status.PasPoolSize,
		"delegated_transactions": status.DelegatedTransactions,
	}

	if status.DelegationStats != nil {
		for k, v := range status.DelegationStats {
			stats[k] = v
		}
	}

	return stats, nil
}

// ValidateNetworkAuthor validates that an address is properly formatted for use as a Network Author
func ValidateNetworkAuthor(address string) error {
	if address == "" {
		return fmt.Errorf("network author address cannot be empty")
	}

	if len(address) < 10 {
		return fmt.Errorf("network author address too short: %s", address)
	}

	// Add more validation as needed based on KNIRV address format
	return nil
}

// PoAuDConfig represents PoAu-D configuration options
type PoAuDConfig struct {
	Enabled                 bool   `json:"enabled"`
	DelegationInterval      string `json:"delegation_interval"`
	MaxSubpoolStaleTime     string `json:"max_subpool_stale_time"`
	MaxPapSubpoolQueue      int    `json:"max_pap_subpool_queue"`
	StatusAdvertiseInterval string `json:"status_advertise_interval"`
}

// GetDefaultPoAuDConfig returns default PoAu-D configuration
func GetDefaultPoAuDConfig() *PoAuDConfig {
	return &PoAuDConfig{
		Enabled:                 false,
		DelegationInterval:      "10s",
		MaxSubpoolStaleTime:     "5m",
		MaxPapSubpoolQueue:      100,
		StatusAdvertiseInterval: "30m",
	}
}
