package models

import (
	"time"
)

// Workflow represents an orchestration workflow
type Workflow struct {
	ID           string    `json:"id" db:"id" validate:"required,uuid"`
	AgentID      string    `json:"agent_id" db:"agent_id" validate:"required,uuid"`
	TargetID     string    `json:"target_id" db:"target_id" validate:"required,uuid"`
	CapabilityID string    `json:"capability_id" db:"capability_id" validate:"required,uuid"`
	Status       string    `json:"status" db:"status" validate:"required,oneof=pending running completed failed cancelled"`
	StartTime    time.Time `json:"start_time" db:"start_time" validate:"required"`
	EndTime      time.Time `json:"end_time,omitempty" db:"end_time"`
	Result       string    `json:"result,omitempty" db:"result" validate:"max=1000"`
	// Owner information
	OwnerID int64 `json:"owner_id" db:"owner_id" validate:"required,gte=1"`
}

// WorkflowRepository defines the interface for workflow persistence
type WorkflowRepository interface {
	CreateWorkflow(workflow *Workflow) error
	GetWorkflowByID(id string) (*Workflow, error)
	UpdateWorkflow(workflow *Workflow) error
	DeleteWorkflow(id string) error
	GetWorkflowsByOwner(ownerID int64) ([]*Workflow, error)
	GetWorkflowsByStatus(status string) ([]*Workflow, error)
}

// WorkflowStats represents statistics about workflows
type WorkflowStats struct {
	TotalCount  int     `json:"total_count"`
	TodayCount  int     `json:"today_count"`
	SuccessRate float64 `json:"success_rate"`
	AvgDuration float64 `json:"avg_duration_ms"`
	TargetCount int     `json:"target_count"`
}

// TopCapability represents a frequently used capability
type TopCapability struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}
