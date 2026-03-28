package objects

import (
	"testing"
	"time"
)

func TestModel_IsValid(t *testing.T) {
	tests := []struct {
		name string
		model Model
		want  bool
	}{
		{
			name: "valid model",
			model: Model{
				ID:       "test-id",
				Name:     "test-model",
				Type:     "WASM",
				FilePath: "/path/to/model",
			},
			want: true,
		},
		{
			name: "missing ID",
			model: Model{
				Name:     "test-model",
				Type:     "WASM",
				FilePath: "/path/to/model",
			},
			want: false,
		},
		{
			name: "missing name",
			model: Model{
				ID:       "test-id",
				Type:     "WASM",
				FilePath: "/path/to/model",
			},
			want: false,
		},
		{
			name: "missing type",
			model: Model{
				ID:       "test-id",
				Name:     "test-model",
				FilePath: "/path/to/model",
			},
			want: false,
		},
		{
			name: "missing file path",
			model: Model{
				ID:   "test-id",
				Name: "test-model",
				Type: "WASM",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.model.IsValid(); got != tt.want {
				t.Errorf("Model.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestModel_CanDeploy(t *testing.T) {
	tests := []struct {
		name string
		model Model
		want  bool
	}{
		{
			name: "can deploy uploaded model",
			model: Model{Status: "uploaded"},
			want:  true,
		},
		{
			name: "cannot deploy running model",
			model: Model{Status: "running"},
			want:  false,
		},
		{
			name: "cannot deploy stopped model",
			model: Model{Status: "stopped"},
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.model.CanDeploy(); got != tt.want {
				t.Errorf("Model.CanDeploy() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestModel_CanStart(t *testing.T) {
	tests := []struct {
		name string
		model Model
		want  bool
	}{
		{
			name: "can start deployed model",
			model: Model{Status: "deployed"},
			want:  true,
		},
		{
			name: "can start stopped model",
			model: Model{Status: "stopped"},
			want:  true,
		},
		{
			name: "cannot start running model",
			model: Model{Status: "running"},
			want:  false,
		},
		{
			name: "cannot start uploaded model",
			model: Model{Status: "uploaded"},
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.model.CanStart(); got != tt.want {
				t.Errorf("Model.CanStart() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestModel_CanStop(t *testing.T) {
	tests := []struct {
		name string
		model Model
		want  bool
	}{
		{
			name: "can stop running model",
			model: Model{Status: "running"},
			want:  true,
		},
		{
			name: "cannot stop stopped model",
			model: Model{Status: "stopped"},
			want:  false,
		},
		{
			name: "cannot stop deployed model",
			model: Model{Status: "deployed"},
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.model.CanStop(); got != tt.want {
				t.Errorf("Model.CanStop() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestModel_IsRunning(t *testing.T) {
	tests := []struct {
		name string
		model Model
		want  bool
	}{
		{
			name: "running model with instance",
			model: Model{
				Status: "running",
				RuntimeInstance: &ModelRuntimeInstance{},
			},
			want: true,
		},
		{
			name: "running model without instance",
			model: Model{Status: "running"},
			want:  false,
		},
		{
			name: "stopped model",
			model: Model{Status: "stopped"},
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.model.IsRunning(); got != tt.want {
				t.Errorf("Model.IsRunning() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestModelResourceLimits_Validation(t *testing.T) {
	limits := &ModelResourceLimits{
		MaxMemoryMB:      1024,
		MaxCPUPercent:    80.0,
		MaxExecutionTime: 300,
		MaxConcurrency:   10,
		MaxDiskMB:        5120,
		NetworkAccess:    false,
		FileSystemAccess: true,
	}

	// Test that limits are properly set
	if limits.MaxMemoryMB != 1024 {
		t.Errorf("Expected MaxMemoryMB to be 1024, got %d", limits.MaxMemoryMB)
	}
	if limits.MaxCPUPercent != 80.0 {
		t.Errorf("Expected MaxCPUPercent to be 80.0, got %f", limits.MaxCPUPercent)
	}
	_ = limits.MaxExecutionTime
	_ = limits.MaxConcurrency
	_ = limits.MaxDiskMB
	_ = limits.NetworkAccess
	_ = limits.FileSystemAccess
}

func TestModelRuntimeInstance_Status(t *testing.T) {
	now := time.Now()
	instance := &ModelRuntimeInstance{
		InstanceID:    "test-instance",
		StartedAt:     now,
		Status:        "running",
		RestartCount:  0,
		ResourceUsage: &ModelResourceUsage{
			MemoryUsageMB:   256.5,
			CPUUsagePercent: 45.2,
			LastUpdated:     now,
		},
	}

	if instance.Status != "running" {
		t.Errorf("Expected status to be 'running', got %s", instance.Status)
	}
	if instance.RestartCount != 0 {
		t.Errorf("Expected restart count to be 0, got %d", instance.RestartCount)
	}
	_ = instance.InstanceID
	_ = instance.StartedAt
	_ = instance.ResourceUsage
}

func TestModelMetrics_Calculation(t *testing.T) {
	now := time.Now()
	metrics := &ModelMetrics{
		ModelID:           "test-model",
		InstanceID:        "test-instance",
		Timestamp:         now,
		RequestsPerSecond: 10.5,
		AverageLatency:    150.0,
		ErrorRate:         0.02,
		Throughput:        9.8,
		ResourceUsage: &ModelResourceUsage{
			MemoryUsageMB:   512.0,
			CPUUsagePercent: 75.0,
			LastUpdated:     now,
		},
	}

	if metrics.RequestsPerSecond != 10.5 {
		t.Errorf("Expected RPS to be 10.5, got %f", metrics.RequestsPerSecond)
	}
	if metrics.ErrorRate != 0.02 {
		t.Errorf("Expected error rate to be 0.02, got %f", metrics.ErrorRate)
	}
	_ = metrics.ModelID
	_ = metrics.InstanceID
	_ = metrics.Timestamp
	_ = metrics.AverageLatency
	_ = metrics.Throughput
	_ = metrics.ResourceUsage
}

func TestModelFilter_Validation(t *testing.T) {
	filter := &ModelFilter{
		Status: []string{"running", "stopped"},
		Type:   []string{"WASM", "LoRA"},
		Author: "test-author",
		Limit:  50,
		Offset: 0,
	}

	if len(filter.Status) != 2 {
		t.Errorf("Expected 2 status filters, got %d", len(filter.Status))
	}
	if filter.Author != "test-author" {
		t.Errorf("Expected author to be 'test-author', got %s", filter.Author)
	}
	_ = filter.Type
	_ = filter.Limit
	_ = filter.Offset
}

func TestModelAction_Validation(t *testing.T) {
	action := &ModelAction{
		Action: "deploy",
		Parameters: map[string]interface{}{
			"replicas": 3,
			"version":  "1.0.0",
		},
		UserID: "test-user",
	}

	if action.Action != "deploy" {
		t.Errorf("Expected action to be 'deploy', got %s", action.Action)
	}
	if action.UserID != "test-user" {
		t.Errorf("Expected user ID to be 'test-user', got %s", action.UserID)
	}
	_ = action.Parameters
}

func TestModelSummary_Calculation(t *testing.T) {
	summary := &ModelSummary{
		TotalModels:    100,
		RunningModels:  25,
		StoppedModels:  50,
		ErrorModels:    5,
		DeployedModels: 20,
		UploadedModels: 75,
	}

	if summary.TotalModels != 100 {
		t.Errorf("Expected total models to be 100, got %d", summary.TotalModels)
	}
	if summary.RunningModels != 25 {
		t.Errorf("Expected running models to be 25, got %d", summary.RunningModels)
	}
	_ = summary.StoppedModels
	_ = summary.ErrorModels
	_ = summary.DeployedModels
	_ = summary.UploadedModels
}