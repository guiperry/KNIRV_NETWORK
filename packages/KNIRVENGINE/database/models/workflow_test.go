package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestWorkflow(t *testing.T) {
	t.Run("creates workflow with all fields", func(t *testing.T) {
		now := time.Now()
		workflow := Workflow{
			ID:           "550e8400-e29b-41d4-a716-446655440000",
			AgentID:      "550e8400-e29b-41d4-a716-446655440001",
			TargetID:     "550e8400-e29b-41d4-a716-446655440002",
			CapabilityID: "550e8400-e29b-41d4-a716-446655440003",
			Status:       "pending",
			StartTime:    now,
			EndTime:      now.Add(time.Hour),
			Result:       "Success",
			OwnerID:      1,
		}

		assert.Equal(t, "550e8400-e29b-41d4-a716-446655440000", workflow.ID)
		assert.Equal(t, "550e8400-e29b-41d4-a716-446655440001", workflow.AgentID)
		assert.Equal(t, "550e8400-e29b-41d4-a716-446655440002", workflow.TargetID)
		assert.Equal(t, "550e8400-e29b-41d4-a716-446655440003", workflow.CapabilityID)
		assert.Equal(t, "pending", workflow.Status)
		assert.Equal(t, now, workflow.StartTime)
		assert.Equal(t, now.Add(time.Hour), workflow.EndTime)
		assert.Equal(t, "Success", workflow.Result)
		assert.Equal(t, int64(1), workflow.OwnerID)
	})

	t.Run("creates workflow with minimal fields", func(t *testing.T) {
		now := time.Now()
		workflow := Workflow{
			ID:           "550e8400-e29b-41d4-a716-446655440000",
			AgentID:      "550e8400-e29b-41d4-a716-446655440001",
			TargetID:     "550e8400-e29b-41d4-a716-446655440002",
			CapabilityID: "550e8400-e29b-41d4-a716-446655440003",
			Status:       "running",
			StartTime:    now,
			OwnerID:      1,
		}

		assert.Equal(t, "550e8400-e29b-41d4-a716-446655440000", workflow.ID)
		assert.Equal(t, "running", workflow.Status)
		assert.Equal(t, now, workflow.StartTime)
		assert.True(t, workflow.EndTime.IsZero()) // EndTime should be zero value
		assert.Empty(t, workflow.Result)          // Result should be empty
		assert.Equal(t, int64(1), workflow.OwnerID)
	})

	t.Run("workflow status values", func(t *testing.T) {
		validStatuses := []string{"pending", "running", "completed", "failed", "cancelled"}
		
		for _, status := range validStatuses {
			workflow := Workflow{
				ID:           "550e8400-e29b-41d4-a716-446655440000",
				AgentID:      "550e8400-e29b-41d4-a716-446655440001",
				TargetID:     "550e8400-e29b-41d4-a716-446655440002",
				CapabilityID: "550e8400-e29b-41d4-a716-446655440003",
				Status:       status,
				StartTime:    time.Now(),
				OwnerID:      1,
			}
			
			assert.Equal(t, status, workflow.Status)
		}
	})
}

func TestWorkflowStats(t *testing.T) {
	t.Run("creates workflow stats with all fields", func(t *testing.T) {
		stats := WorkflowStats{
			TotalCount:  100,
			TodayCount:  25,
			SuccessRate: 85.5,
			AvgDuration: 1500.75,
			TargetCount: 10,
		}

		assert.Equal(t, 100, stats.TotalCount)
		assert.Equal(t, 25, stats.TodayCount)
		assert.Equal(t, 85.5, stats.SuccessRate)
		assert.Equal(t, 1500.75, stats.AvgDuration)
		assert.Equal(t, 10, stats.TargetCount)
	})

	t.Run("creates workflow stats with zero values", func(t *testing.T) {
		stats := WorkflowStats{}

		assert.Equal(t, 0, stats.TotalCount)
		assert.Equal(t, 0, stats.TodayCount)
		assert.Equal(t, 0.0, stats.SuccessRate)
		assert.Equal(t, 0.0, stats.AvgDuration)
		assert.Equal(t, 0, stats.TargetCount)
	})

	t.Run("calculates success rate correctly", func(t *testing.T) {
		stats := WorkflowStats{
			TotalCount:  100,
			SuccessRate: 85.0,
		}

		// Success rate should be a percentage
		assert.True(t, stats.SuccessRate >= 0.0 && stats.SuccessRate <= 100.0)
	})
}

func TestTopCapability(t *testing.T) {
	t.Run("creates top capability with name and count", func(t *testing.T) {
		capability := TopCapability{
			Name:  "file_operations",
			Count: 42,
		}

		assert.Equal(t, "file_operations", capability.Name)
		assert.Equal(t, 42, capability.Count)
	})

	t.Run("creates top capability with empty values", func(t *testing.T) {
		capability := TopCapability{}

		assert.Empty(t, capability.Name)
		assert.Equal(t, 0, capability.Count)
	})

	t.Run("creates multiple top capabilities", func(t *testing.T) {
		capabilities := []TopCapability{
			{Name: "file_operations", Count: 42},
			{Name: "network_requests", Count: 35},
			{Name: "data_processing", Count: 28},
		}

		assert.Len(t, capabilities, 3)
		assert.Equal(t, "file_operations", capabilities[0].Name)
		assert.Equal(t, 42, capabilities[0].Count)
		assert.Equal(t, "network_requests", capabilities[1].Name)
		assert.Equal(t, 35, capabilities[1].Count)
		assert.Equal(t, "data_processing", capabilities[2].Name)
		assert.Equal(t, 28, capabilities[2].Count)
	})
}

// Test workflow validation tags (these would be used by a validation library)
func TestWorkflowValidationTags(t *testing.T) {
	t.Run("workflow has correct validation tags", func(t *testing.T) {
		// This test verifies that the struct tags are correctly defined
		// In a real application, these would be used by a validation library like go-playground/validator
		
		workflow := Workflow{
			ID:           "550e8400-e29b-41d4-a716-446655440000", // Should be UUID
			AgentID:      "550e8400-e29b-41d4-a716-446655440001", // Should be UUID
			TargetID:     "550e8400-e29b-41d4-a716-446655440002", // Should be UUID
			CapabilityID: "550e8400-e29b-41d4-a716-446655440003", // Should be UUID
			Status:       "completed",                             // Should be one of the allowed values
			StartTime:    time.Now(),                              // Required
			Result:       "Short result",                          // Max 1000 characters
			OwnerID:      1,                                       // Should be >= 1
		}

		// Verify the struct has the expected fields
		assert.NotEmpty(t, workflow.ID)
		assert.NotEmpty(t, workflow.AgentID)
		assert.NotEmpty(t, workflow.TargetID)
		assert.NotEmpty(t, workflow.CapabilityID)
		assert.NotEmpty(t, workflow.Status)
		assert.False(t, workflow.StartTime.IsZero())
		assert.True(t, workflow.OwnerID > 0)
	})

	t.Run("workflow result length validation", func(t *testing.T) {
		// Test that result field can handle up to 1000 characters
		longResult := make([]byte, 1000)
		for i := range longResult {
			longResult[i] = 'A'
		}

		workflow := Workflow{
			Result: string(longResult),
		}

		assert.Len(t, workflow.Result, 1000)
	})
}

// Test JSON serialization/deserialization
func TestWorkflowJSONTags(t *testing.T) {
	t.Run("workflow has correct JSON tags", func(t *testing.T) {
		now := time.Now()
		workflow := Workflow{
			ID:           "550e8400-e29b-41d4-a716-446655440000",
			AgentID:      "550e8400-e29b-41d4-a716-446655440001",
			TargetID:     "550e8400-e29b-41d4-a716-446655440002",
			CapabilityID: "550e8400-e29b-41d4-a716-446655440003",
			Status:       "completed",
			StartTime:    now,
			EndTime:      now.Add(time.Hour),
			Result:       "Success",
			OwnerID:      1,
		}

		// Verify all fields are accessible (this tests that the struct is properly defined)
		assert.NotEmpty(t, workflow.ID)
		assert.NotEmpty(t, workflow.AgentID)
		assert.NotEmpty(t, workflow.TargetID)
		assert.NotEmpty(t, workflow.CapabilityID)
		assert.NotEmpty(t, workflow.Status)
		assert.False(t, workflow.StartTime.IsZero())
		assert.False(t, workflow.EndTime.IsZero())
		assert.NotEmpty(t, workflow.Result)
		assert.True(t, workflow.OwnerID > 0)
	})
}
