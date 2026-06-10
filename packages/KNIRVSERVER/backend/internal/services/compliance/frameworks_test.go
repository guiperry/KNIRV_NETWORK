package compliance

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewFrameworkManager(t *testing.T) {
	fm := NewFrameworkManager()
	assert.NotNil(t, fm)

	fw, ok := fm.GetFramework(FrameworkKNIRV)
	assert.True(t, ok)
	assert.NotNil(t, fw)
	assert.Equal(t, "KNIRV Network Security Framework", fw.Name)
	assert.Equal(t, "1.0.0", fw.Version)
}

func TestDefaultControls(t *testing.T) {
	fm := NewFrameworkManager()
	fw, _ := fm.GetFramework(FrameworkKNIRV)

	expectedIDs := map[string]string{
		"KNIRV-01": "DID Identity Binding",
		"KNIRV-02": "Policy Enforcement",
		"KNIRV-03": "Audit Logging",
		"KNIRV-04": "Revocation Checking",
		"KNIRV-05": "Resource Guardrails",
		"KNIRV-06": "Poisoning Detection",
		"KNIRV-07": "Circuit Breaking",
	}

	if len(fw.Controls) != 7 {
		t.Fatalf("expected 7 controls, got %d", len(fw.Controls))
	}

	for _, ctrl := range fw.Controls {
		expectedName, ok := expectedIDs[ctrl.ID]
		assert.True(t, ok, "unexpected control ID: %s", ctrl.ID)
		assert.Equal(t, expectedName, ctrl.Name)
		assert.NotEmpty(t, ctrl.Description)
		assert.NotEmpty(t, ctrl.Category)
	}
}

func TestRegisterFramework(t *testing.T) {
	fm := NewFrameworkManager()
	fw := &Framework{
		ID:      "custom-framework",
		Name:    "Custom Framework",
		Version: "2.0.0",
		Controls: []Control{
			{ID: "CUSTOM-01", Name: "Custom Control", Description: "A custom control", Category: "custom"},
		},
	}
	fm.RegisterFramework(fw)

	got, ok := fm.GetFramework("custom-framework")
	assert.True(t, ok)
	assert.Equal(t, "Custom Framework", got.Name)
	assert.Len(t, got.Controls, 1)
}

func TestGetFrameworkNotFound(t *testing.T) {
	fm := NewFrameworkManager()
	_, ok := fm.GetFramework("nonexistent")
	assert.False(t, ok)
}

func TestListFrameworks(t *testing.T) {
	fm := NewFrameworkManager()
	frameworks := fm.ListFrameworks()
	assert.Len(t, frameworks, 1)

	fm.RegisterFramework(&Framework{ID: "fw2", Name: "Framework 2"})
	assert.Len(t, fm.ListFrameworks(), 2)
}

func TestUpdateControlStatus(t *testing.T) {
	fm := NewFrameworkManager()
	now := time.Now()
	status := &ControlStatus{
		ControlID:   "KNIRV-01",
		FrameworkID: FrameworkKNIRV,
		NodeID:      "node-1",
		Status:      "pass",
		LastChecked: now,
		Evidence:    "DID verified",
	}
	fm.UpdateControlStatus(status)

	got, ok := fm.GetControlStatus(FrameworkKNIRV, "KNIRV-01", "node-1")
	assert.True(t, ok)
	assert.Equal(t, "pass", got.Status)
	assert.Equal(t, "DID verified", got.Evidence)
}

func TestUpdateControlStatusOverwrite(t *testing.T) {
	fm := NewFrameworkManager()
	fm.UpdateControlStatus(&ControlStatus{
		ControlID:   "KNIRV-01",
		FrameworkID: FrameworkKNIRV,
		NodeID:      "node-1",
		Status:      "fail",
	})
	fm.UpdateControlStatus(&ControlStatus{
		ControlID:   "KNIRV-01",
		FrameworkID: FrameworkKNIRV,
		NodeID:      "node-1",
		Status:      "pass",
	})

	got, ok := fm.GetControlStatus(FrameworkKNIRV, "KNIRV-01", "node-1")
	assert.True(t, ok)
	assert.Equal(t, "pass", got.Status)
}

func TestGetControlStatusNotFound(t *testing.T) {
	fm := NewFrameworkManager()
	_, ok := fm.GetControlStatus(FrameworkKNIRV, "KNIRV-99", "node-1")
	assert.False(t, ok)
}

func TestGetNodeCompliance(t *testing.T) {
	fm := NewFrameworkManager()
	fm.UpdateControlStatus(&ControlStatus{
		ControlID:   "KNIRV-01",
		FrameworkID: FrameworkKNIRV,
		NodeID:      "node-1",
		Status:      "pass",
	})
	fm.UpdateControlStatus(&ControlStatus{
		ControlID:   "KNIRV-02",
		FrameworkID: FrameworkKNIRV,
		NodeID:      "node-1",
		Status:      "fail",
	})
	fm.UpdateControlStatus(&ControlStatus{
		ControlID:   "KNIRV-01",
		FrameworkID: FrameworkKNIRV,
		NodeID:      "node-2",
		Status:      "pass",
	})

	node1 := fm.GetNodeCompliance("node-1")
	assert.Len(t, node1, 1)
	assert.Len(t, node1[FrameworkKNIRV], 2)

	node2 := fm.GetNodeCompliance("node-2")
	assert.Len(t, node2, 1)
	assert.Len(t, node2[FrameworkKNIRV], 1)

	node3 := fm.GetNodeCompliance("node-3")
	assert.Empty(t, node3)
}
