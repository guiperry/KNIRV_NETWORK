package database

import (
	"context"
	"fmt"
	"time"

	"Agentic_Engine/database/models"

	"github.com/google/uuid"
)

// SimpleWorkflowRepository implements a basic workflow repository
type SimpleWorkflowRepository struct {
	collection Collection
}

// NewSimpleWorkflowRepository creates a new workflow repository
func NewSimpleWorkflowRepository(collection Collection) *SimpleWorkflowRepository {
	return &SimpleWorkflowRepository{
		collection: collection,
	}
}

// CreateWorkflow adds a new workflow to the database
func (r *SimpleWorkflowRepository) CreateWorkflow(ctx context.Context, workflow *models.Workflow) error {
	// Generate ID if not provided
	if workflow.ID == "" {
		workflow.ID = uuid.New().String()
	}

	// Set start time if not provided
	if workflow.StartTime.IsZero() {
		workflow.StartTime = time.Now()
	}

	// Convert workflow to metadata
	metadata := map[string]string{
		"agent_id":      workflow.AgentID,
		"target_id":     workflow.TargetID,
		"capability_id": workflow.CapabilityID,
		"status":        workflow.Status,
		"start_time":    workflow.StartTime.Format(time.RFC3339),
		"owner_id":      fmt.Sprintf("%d", workflow.OwnerID),
	}

	// Add end time and result if present
	if !workflow.EndTime.IsZero() {
		metadata["end_time"] = workflow.EndTime.Format(time.RFC3339)
	}

	if workflow.Result != "" {
		metadata["result"] = workflow.Result
	}

	// Create document for storage
	doc := Document{
		ID:       workflow.ID,
		Content:  fmt.Sprintf("Workflow using agent %s on target %s with capability %s", workflow.AgentID, workflow.TargetID, workflow.CapabilityID),
		Metadata: metadata,
	}

	// Store workflow
	return r.collection.AddDocument(ctx, doc)
}

// GetWorkflowByID retrieves a workflow by ID
func (r *SimpleWorkflowRepository) GetWorkflowByID(ctx context.Context, id string) (*models.Workflow, error) {
	// Get document by ID
	doc, err := r.collection.GetDocumentByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Convert document to Workflow
	return documentToWorkflow(doc)
}

// UpdateWorkflow updates an existing workflow
func (r *SimpleWorkflowRepository) UpdateWorkflow(ctx context.Context, workflow *models.Workflow) error {
	// Check if workflow exists
	_, err := r.GetWorkflowByID(ctx, workflow.ID)
	if err != nil {
		return err
	}

	// Convert workflow to metadata
	metadata := map[string]string{
		"agent_id":      workflow.AgentID,
		"target_id":     workflow.TargetID,
		"capability_id": workflow.CapabilityID,
		"status":        workflow.Status,
		"start_time":    workflow.StartTime.Format(time.RFC3339),
		"owner_id":      fmt.Sprintf("%d", workflow.OwnerID),
	}

	// Add end time and result if present
	if !workflow.EndTime.IsZero() {
		metadata["end_time"] = workflow.EndTime.Format(time.RFC3339)
	}

	if workflow.Result != "" {
		metadata["result"] = workflow.Result
	}

	// Create document for storage
	doc := Document{
		ID:       workflow.ID,
		Content:  fmt.Sprintf("Workflow using agent %s on target %s with capability %s", workflow.AgentID, workflow.TargetID, workflow.CapabilityID),
		Metadata: metadata,
	}

	// Update workflow
	return r.collection.UpdateDocument(ctx, doc)
}

// DeleteWorkflow deletes a workflow
func (r *SimpleWorkflowRepository) DeleteWorkflow(ctx context.Context, id string) error {
	return r.collection.DeleteDocument(ctx, id)
}

// GetWorkflowsByOwner retrieves all workflows for a user
func (r *SimpleWorkflowRepository) GetWorkflowsByOwner(ctx context.Context, ownerID int64) ([]*models.Workflow, error) {
	// Query for workflows with matching owner ID
	filter := map[string]string{
		"owner_id": fmt.Sprintf("%d", ownerID),
	}

	docs, err := r.collection.GetDocumentsByMetadata(ctx, filter)
	if err != nil {
		return nil, err
	}

	// Convert documents to Workflows
	workflows := make([]*models.Workflow, len(docs))
	for i, doc := range docs {
		workflow, err := documentToWorkflow(doc)
		if err != nil {
			return nil, err
		}
		workflows[i] = workflow
	}

	return workflows, nil
}

// GetWorkflowsByStatus retrieves workflows by status
func (r *SimpleWorkflowRepository) GetWorkflowsByStatus(ctx context.Context, status string) ([]*models.Workflow, error) {
	// Query for workflows with matching status
	filter := map[string]string{
		"status": status,
	}

	docs, err := r.collection.GetDocumentsByMetadata(ctx, filter)
	if err != nil {
		return nil, err
	}

	// Convert documents to Workflows
	workflows := make([]*models.Workflow, len(docs))
	for i, doc := range docs {
		workflow, err := documentToWorkflow(doc)
		if err != nil {
			return nil, err
		}
		workflows[i] = workflow
	}

	return workflows, nil
}

// Helper function to convert a document to a Workflow
func documentToWorkflow(doc Document) (*models.Workflow, error) {
	startTime, err := time.Parse(time.RFC3339, doc.Metadata["start_time"])
	if err != nil {
		return nil, fmt.Errorf("invalid start_time timestamp: %w", err)
	}

	ownerID, err := parseInt64(doc.Metadata["owner_id"])
	if err != nil {
		return nil, fmt.Errorf("invalid owner_id: %w", err)
	}

	workflow := &models.Workflow{
		ID:           doc.ID,
		AgentID:      doc.Metadata["agent_id"],
		TargetID:     doc.Metadata["target_id"],
		CapabilityID: doc.Metadata["capability_id"],
		Status:       doc.Metadata["status"],
		StartTime:    startTime,
		Result:       doc.Metadata["result"],
		OwnerID:      ownerID,
	}

	// Parse end time if present
	if endTimeStr, ok := doc.Metadata["end_time"]; ok && endTimeStr != "" {
		endTime, err := time.Parse(time.RFC3339, endTimeStr)
		if err != nil {
			return nil, fmt.Errorf("invalid end_time timestamp: %w", err)
		}
		workflow.EndTime = endTime
	}

	return workflow, nil
}

// Helper function to parse int64
func parseInt64(s string) (int64, error) {
	if s == "" {
		return 0, fmt.Errorf("empty string")
	}
	var result int64
	_, err := fmt.Sscanf(s, "%d", &result)
	return result, err
}

// GetWorkflowStats calculates statistics about workflows for a user
func (r *SimpleWorkflowRepository) GetWorkflowStats(ctx context.Context, ownerID int64) (*models.WorkflowStats, error) {
	// Get all workflows for the user
	workflows, err := r.GetWorkflowsByOwner(ctx, ownerID)
	if err != nil {
		return nil, err
	}

	// Calculate statistics
	stats := &models.WorkflowStats{}
	stats.TotalCount = len(workflows)

	// Count today's workflows
	today := time.Now().Truncate(24 * time.Hour)

	// Count successful workflows and calculate average duration
	successCount := 0
	totalDuration := int64(0)
	completedCount := 0

	// Track unique targets
	uniqueTargets := make(map[string]bool)

	for _, workflow := range workflows {
		// Count today's workflows
		if workflow.StartTime.After(today) || workflow.StartTime.Equal(today) {
			stats.TodayCount++
		}

		// Track unique targets
		uniqueTargets[workflow.TargetID] = true

		// Count completed workflows
		if workflow.Status == "completed" {
			successCount++

			// Calculate duration for completed workflows
			if !workflow.EndTime.IsZero() {
				duration := workflow.EndTime.Sub(workflow.StartTime)
				totalDuration += duration.Milliseconds()
				completedCount++
			}
		}
	}

	// Calculate success rate
	if stats.TotalCount > 0 {
		stats.SuccessRate = float64(successCount) / float64(stats.TotalCount) * 100
	}

	// Calculate average duration
	if completedCount > 0 {
		stats.AvgDuration = float64(totalDuration) / float64(completedCount)
	}

	// Count unique targets
	stats.TargetCount = len(uniqueTargets)

	return stats, nil
}

// GetTopCapabilities returns the most frequently used capabilities
func (r *SimpleWorkflowRepository) GetTopCapabilities(ctx context.Context, ownerID int64, limit int) ([]models.TopCapability, error) {
	// Get all workflows for the user
	workflows, err := r.GetWorkflowsByOwner(ctx, ownerID)
	if err != nil {
		return nil, err
	}

	// Count capability usage
	capabilityCounts := make(map[string]int)
	for _, workflow := range workflows {
		capabilityCounts[workflow.CapabilityID]++
	}

	// Convert to slice for sorting
	var topCapabilities []models.TopCapability
	for id, count := range capabilityCounts {
		topCapabilities = append(topCapabilities, models.TopCapability{
			Name:  id,
			Count: count,
		})
	}

	// Sort by count (descending)
	for i := 0; i < len(topCapabilities); i++ {
		for j := i + 1; j < len(topCapabilities); j++ {
			if topCapabilities[i].Count < topCapabilities[j].Count {
				topCapabilities[i], topCapabilities[j] = topCapabilities[j], topCapabilities[i]
			}
		}
	}

	// Limit results
	if len(topCapabilities) > limit {
		topCapabilities = topCapabilities[:limit]
	}

	return topCapabilities, nil
}

// Collection is a simplified interface for document storage
// This would be implemented by your actual database layer
type Collection interface {
	AddDocument(ctx context.Context, doc Document) error
	GetDocumentByID(ctx context.Context, id string) (Document, error)
	UpdateDocument(ctx context.Context, doc Document) error
	DeleteDocument(ctx context.Context, id string) error
	GetDocumentsByMetadata(ctx context.Context, filter map[string]string) ([]Document, error)
}

// Document represents a stored document
type Document struct {
	ID       string
	Content  string
	Metadata map[string]string
}
