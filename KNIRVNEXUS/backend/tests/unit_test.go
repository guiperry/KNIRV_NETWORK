package tests

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/knirv/nexus-backend/internal/models"
	"github.com/knirv/nexus-backend/internal/services/dvemanager"
	"github.com/knirv/nexus-backend/internal/services/validation"
)

// TestNodeFilter tests the NodeFilter functionality
func TestNodeFilter(t *testing.T) {
	filter := &dvemanager.NodeFilter{
		Status:        "online",
		TEEType:       "sgx",
		MinStake:      1000000,
		MinReputation: 80,
		Capabilities:  []string{"skillnode", "base_llm"},
	}

	tests := []struct {
		name     string
		node     *models.DVENode
		expected bool
	}{
		{
			name: "Matching node",
			node: &models.DVENode{
				Status:          "online",
				TEEType:         "sgx",
				StakeAmount:     2000000,
				ReputationScore: 90,
				Capabilities:    []string{"skillnode", "base_llm", "custom"},
			},
			expected: true,
		},
		{
			name: "Wrong status",
			node: &models.DVENode{
				Status:          "offline",
				TEEType:         "sgx",
				StakeAmount:     2000000,
				ReputationScore: 90,
				Capabilities:    []string{"skillnode", "base_llm"},
			},
			expected: false,
		},
		{
			name: "Insufficient stake",
			node: &models.DVENode{
				Status:          "online",
				TEEType:         "sgx",
				StakeAmount:     500000,
				ReputationScore: 90,
				Capabilities:    []string{"skillnode", "base_llm"},
			},
			expected: false,
		},
		{
			name: "Missing capability",
			node: &models.DVENode{
				Status:          "online",
				TEEType:         "sgx",
				StakeAmount:     2000000,
				ReputationScore: 90,
				Capabilities:    []string{"skillnode"},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filter.Matches(tt.node)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestTaskFilter tests the TaskFilter functionality
func TestTaskFilter(t *testing.T) {
	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)
	tomorrow := now.Add(24 * time.Hour)

	filter := &validation.TaskFilter{
		Status:        "pending",
		Type:          "skillnode",
		Priority:      5,
		RequestedBy:   "test-user",
		CreatedAfter:  &yesterday,
		CreatedBefore: &tomorrow,
	}

	tests := []struct {
		name     string
		task     *models.ValidationTask
		expected bool
	}{
		{
			name: "Matching task",
			task: &models.ValidationTask{
				Status:      "pending",
				Type:        "skillnode",
				Priority:    5,
				RequestedBy: "test-user",
				CreatedAt:   now,
			},
			expected: true,
		},
		{
			name: "Wrong status",
			task: &models.ValidationTask{
				Status:      "completed",
				Type:        "skillnode",
				Priority:    5,
				RequestedBy: "test-user",
				CreatedAt:   now,
			},
			expected: false,
		},
		{
			name: "Wrong priority",
			task: &models.ValidationTask{
				Status:      "pending",
				Type:        "skillnode",
				Priority:    3,
				RequestedBy: "test-user",
				CreatedAt:   now,
			},
			expected: false,
		},
		{
			name: "Created too early",
			task: &models.ValidationTask{
				Status:      "pending",
				Type:        "skillnode",
				Priority:    5,
				RequestedBy: "test-user",
				CreatedAt:   yesterday.Add(-1 * time.Hour),
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filter.Matches(tt.task)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestUserPermissions tests user permission functionality
func TestUserPermissions(t *testing.T) {
	tests := []struct {
		role        models.UserRole
		permissions models.UserPermissions
	}{
		{
			role: models.RoleAdmin,
			permissions: models.UserPermissions{
				CanManageNodes:      true,
				CanCreateTasks:      true,
				CanViewSystemHealth: true,
				CanManageUsers:      true,
				CanAccessTEEData:    true,
				CanGenerateReports:  true,
				CanShareReports:     true,
				CanScheduleReports:  true,
				CanViewAuditLogs:    true,
				CanManageAlerts:     true,
			},
		},
		{
			role: models.RoleViewer,
			permissions: models.UserPermissions{
				CanManageNodes:      false,
				CanCreateTasks:      false,
				CanViewSystemHealth: true,
				CanManageUsers:      false,
				CanAccessTEEData:    false,
				CanGenerateReports:  true,
				CanShareReports:     false,
				CanScheduleReports:  false,
				CanViewAuditLogs:    false,
				CanManageAlerts:     false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(string(tt.role), func(t *testing.T) {
			permissions := models.GetPermissionsForRole(tt.role)
			assert.Equal(t, tt.permissions, permissions)
		})
	}
}

// TestValidationResult tests validation result creation
func TestValidationResult(t *testing.T) {
	testResults := []models.TestResult{
		{
			TestCaseID:    "test-1",
			Status:        "passed",
			Score:         1.0,
			ExecutionTime: 100 * time.Millisecond,
		},
		{
			TestCaseID:    "test-2",
			Status:        "failed",
			Score:         0.0,
			ExecutionTime: 50 * time.Millisecond,
		},
	}

	result := &models.ValidationResult{
		ID:              "result-123",
		TaskID:          "task-456",
		ValidatorNodeID: "node-789",
		Status:          "success",
		Score:           0.5,
		TestResults:     testResults,
		ExecutionTime:   150 * time.Millisecond,
		CreatedAt:       time.Now(),
	}

	assert.NotEmpty(t, result.ID)
	assert.Equal(t, "task-456", result.TaskID)
	assert.Equal(t, "success", result.Status)
	assert.Equal(t, 0.5, result.Score)
	assert.Len(t, result.TestResults, 2)
	assert.Equal(t, 150*time.Millisecond, result.ExecutionTime)
}

// TestSystemHealth tests system health calculation
func TestSystemHealth(t *testing.T) {
	health := &models.SystemHealth{
		ID:                  "health-123",
		OverallStatus:       "healthy",
		ActiveNodes:         8,
		TotalNodes:          10,
		PendingTasks:        5,
		CompletedTasks:      100,
		FailedTasks:         2,
		AverageResponseTime: 150.5,
		NetworkLatency:      25.3,
		TEEHealthScore:      0.95,
		Timestamp:           time.Now(),
	}

	assert.Equal(t, "healthy", health.OverallStatus)
	assert.Equal(t, 8, health.ActiveNodes)
	assert.Equal(t, 10, health.TotalNodes)
	assert.Equal(t, 0.95, health.TEEHealthScore)

	// Test health status calculation based on node ratio
	ratio := float64(health.ActiveNodes) / float64(health.TotalNodes)
	assert.Equal(t, 0.8, ratio)
}

// TestTEEAttestation tests TEE attestation functionality
func TestTEEAttestation(t *testing.T) {
	attestation := &models.TEEAttestation{
		ID:           "attestation-123",
		NodeID:       "node-456",
		TEEType:      "sgx",
		Status:       "valid",
		Quote:        "base64-encoded-quote",
		Signature:    "base64-encoded-signature",
		CertChain:    "base64-encoded-cert-chain",
		Measurements: []string{"measurement1", "measurement2"},
		CreatedAt:    time.Now(),
		ExpiresAt:    time.Now().Add(24 * time.Hour),
	}

	assert.NotEmpty(t, attestation.ID)
	assert.Equal(t, "sgx", attestation.TEEType)
	assert.Equal(t, "valid", attestation.Status)
	assert.Len(t, attestation.Measurements, 2)
	assert.True(t, attestation.ExpiresAt.After(attestation.CreatedAt))
}

// TestP2PMessage tests P2P message structure
func TestP2PMessage(t *testing.T) {
	message := &models.P2PMessage{
		ID:    "msg-123",
		Type:  "validation_request",
		From:  "peer-456",
		To:    "peer-789",
		Topic: "dve-validation",
		Payload: map[string]interface{}{
			"task_id": "task-123",
			"data":    "test-data",
		},
		Timestamp: time.Now(),
		Signature: "signature-hash",
	}

	assert.NotEmpty(t, message.ID)
	assert.Equal(t, "validation_request", message.Type)
	assert.Contains(t, message.Payload, "task_id")
	assert.Equal(t, "task-123", message.Payload["task_id"])
}

// TestReportGeneration tests report generation functionality
func TestReportGeneration(t *testing.T) {
	report := &models.ReportRecord{
		ID:          "report-123",
		Title:       "System Health Report",
		Type:        "system_health",
		Format:      "pdf",
		GeneratedBy: "user-456",
		FilePath:    "/app/reports/system_health_20240101.pdf",
		FileSize:    1024000,
		SharedWith:  []string{"user-789", "user-101"},
		Status:      "completed",
		CreatedAt:   time.Now(),
	}

	assert.NotEmpty(t, report.ID)
	assert.Equal(t, "system_health", report.Type)
	assert.Equal(t, "pdf", report.Format)
	assert.Equal(t, "completed", report.Status)
	assert.Len(t, report.SharedWith, 2)
	assert.Greater(t, report.FileSize, int64(0))
}

// TestNetworkTopology tests network topology functionality
func TestNetworkTopology(t *testing.T) {
	peers := []models.PeerInfo{
		{
			ID:       "peer-1",
			Address:  "/ip4/192.168.1.100/tcp/4001",
			Role:     "dve-validator",
			Status:   "connected",
			Latency:  25,
			LastSeen: time.Now(),
		},
		{
			ID:       "peer-2",
			Address:  "/ip4/192.168.1.101/tcp/4001",
			Role:     "dve-manager",
			Status:   "connected",
			Latency:  30,
			LastSeen: time.Now(),
		},
	}

	topology := &models.NetworkTopology{
		ID:             "topology-123",
		TotalPeers:     2,
		ConnectedPeers: 2,
		Peers:          peers,
		Connections:    []models.ConnectionInfo{},
		Timestamp:      time.Now(),
	}

	assert.NotEmpty(t, topology.ID)
	assert.Equal(t, 2, topology.TotalPeers)
	assert.Equal(t, 2, topology.ConnectedPeers)
	assert.Len(t, topology.Peers, 2)
	assert.Equal(t, "dve-validator", topology.Peers[0].Role)
}

// TestAlert tests alert functionality
func TestAlert(t *testing.T) {
	alert := &models.Alert{
		ID:       "alert-123",
		Type:     "error",
		Severity: "critical",
		Title:    "Node Offline",
		Message:  "DVE node node-456 has gone offline",
		Source:   "dve-manager",
		NodeID:   "node-456",
		Metadata: map[string]interface{}{
			"last_heartbeat": time.Now().Add(-5 * time.Minute),
			"node_location":  "us-east-1",
		},
		Status:    "active",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	assert.NotEmpty(t, alert.ID)
	assert.Equal(t, "critical", alert.Severity)
	assert.Equal(t, "active", alert.Status)
	assert.Contains(t, alert.Metadata, "last_heartbeat")
	assert.Equal(t, "node-456", alert.NodeID)
}
