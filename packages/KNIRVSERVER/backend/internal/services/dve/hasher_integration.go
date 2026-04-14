package dve

import (
	"context"
	"fmt"
	"time"

	knirvbase "github.com/knirvcorp/knirvbase"
	hasherpb "github.com/knirvcorp/knirvserver/backend/internal/proto"
	"github.com/knirvcorp/knirvserver/backend/internal/services/dve"
)

// HasherIntegration manages the integration between DVE and the KNIRVHASHER gRPC service.
// It handles event-driven training triggers and data export.
type HasherIntegration struct {
	grpcClient   hasherpb.HasherServiceClient
	dveManager   *DVEManager
	guardrailMgr *guardrails.DynamicGuardrailManager
	ontologyMgr  *DVEOntologyManager
	kvbase       knirvbase.Collection
}

// NewHasherIntegration creates a new HasherIntegration instance.
func NewHasherIntegration(
	client hasherpb.HasherServiceClient,
	dveMgr *DVEManager,
	guardrailMgr *guardrails.DynamicGuardrailManager,
	ontologyMgr *DVEOntologyManager,
	kvbase knirvbase.Collection,
) *HasherIntegration {
	return &HasherIntegration{
		grpcClient:   client,
		dveManager:   dveMgr,
		guardrailMgr: guardrailMgr,
		ontologyMgr:  ontologyMgr,
		kvbase:       kvbase,
	}
}

// OnGuardrailViolation triggers training when a guardrail violation occurs.
func (hi *HasherIntegration) OnGuardrailViolation(violation *GuardrailViolation) error {
	return hi.TriggerTraining(violation.NodeID, GUARDRAIL_VIOLATION)
}

// OnValidationComplete analyzes task results and may trigger training if patterns suggest it.
func (hi *HasherIntegration) OnValidationComplete(result *TaskResult) error {
	// Analyze patterns in the result to determine if training should be triggered
	patterns := hi.analyzePatterns(result)
	if patterns.RequiresTraining {
		return hi.TriggerTraining(result.UserID, ON_DEMAND)
	}
	return nil
}

// ExportUserData exports user security data to the hasher service.
func (hi *HasherIntegration) ExportUserData(orgID, userID string) (<-chan *EncryptedChunk, error) {
	stream, err := hi.grpcClient.ExportSecurityData(context.Background(), &hasherpb.ExportRequest{
		OrgId:     orgID,
		UserId:    userID,
		DataType:  hasherpb.DataType_ALL,
		Encrypted: true,
	})
	if err != nil {
		return nil, fmt.Errorf("export security data: %w", err)
	}

	ch := make(chan *EncryptedChunk)
	go func() {
		defer close(ch)
		for {
			chunk, err := stream.Recv()
			if err != nil {
				return
			}
			ch <- &EncryptedChunk{
				Data:    chunk.Data,
				ChunkID: chunk.ChunkId,
				IsLast:  chunk.IsLast,
			}
		}
	}()

	return ch, nil
}

// TriggerTraining initiates training for a specific user.
func (hi *HasherIntegration) TriggerTraining(orgID, userID string, trigger TrainingTrigger) error {
	resp, err := hi.grpcClient.TriggerTraining(context.Background(), &hasherpb.TrainingRequest{
		OrgId:   orgID,
		UserId:  userID,
		Trigger: hasherpb.TrainingTrigger(trigger),
		Options: make(map[string]string),
	})
	if err != nil {
		return fmt.Errorf("trigger training: %w", err)
	}

	// Log training initiation
	fmt.Printf("Training initiated: ID=%s, Status=%s\n", resp.TrainingId, resp.Status)
	return nil
}

// ValidateAction checks if an action is allowed based on trained security rules.
func (hi *HasherIntegration) ValidateAction(userID, action string, ctx map[string]string) (*ActionResponse, error) {
	resp, err := hi.grpcClient.ValidateAction(context.Background(), &hasherpb.ActionRequest{
		UserId:  userID,
		Action:  action,
		Context: ctx,
	})
	if err != nil {
		return nil, fmt.Errorf("validate action: %w", err)
	}

	return &ActionResponse{
		Allowed:      resp.Allowed,
		Confidence:   resp.Confidence,
		Violations:   resp.Violations,
		AppliedRules: resp.AppliedRules,
	}, nil
}

// analyzePatterns analyzes task results to determine if training is needed.
func (hi *HasherIntegration) analyzePatterns(result *TaskResult) *PatternAnalysis {
	// Implementation would analyze success rates, error patterns, etc.
	return &PatternAnalysis{
		RequiresTraining: false, // Placeholder logic
	}
}

// EncryptedChunk represents an encrypted data chunk from the hasher service.
type EncryptedChunk struct {
	Data    []byte
	ChunkID string
	IsLast  bool
}

// ActionResponse contains the result of an action validation.
type ActionResponse struct {
	Allowed      bool
	Confidence   float32
	Violations   []string
	AppliedRules []string
}

// TrainingTrigger represents different types of training triggers.
type TrainingTrigger int

const (
	ON_DEMAND TrainingTrigger = iota
	SCHEDULED
	GUARDRAIL_VIOLATION
)

// PatternAnalysis contains analysis of behavioral patterns.
type PatternAnalysis struct {
	RequiresTraining bool
}
