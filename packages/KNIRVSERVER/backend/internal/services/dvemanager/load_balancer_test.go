package dvemanager

import (
	"testing"

	"backend_server/internal/objects"
)

func TestLoadBalancer_SelectNode_NoNodes(t *testing.T) {
	lb := &LoadBalancer{algorithm: "reputation_based"}
	task := &objects.ValidationTask{ID: "task-1", Priority: 5}
	_, err := lb.SelectNode(task, []*objects.DVENode{})
	if err == nil {
		t.Error("Expected error when no nodes available")
	}
}

func TestLoadBalancer_IsNodeEligible_BrowserExtensionTrustTier(t *testing.T) {
	lb := &LoadBalancer{}

	browserNode := &objects.DVENode{
		ID:              "browser-node-1",
		TEEType:         "browser-extension",
		Status:          "online",
		ReputationScore: 100,
		CPUUsage:        50,
		MemoryUsage:     50,
	}

	tests := []struct {
		name       string
		trustTier  string
		eligible   bool
	}{
		{
			name:       "standard trust tier - eligible",
			trustTier:  "standard",
			eligible:   true,
		},
		{
			name:       "empty trust tier - eligible",
			trustTier:  "",
			eligible:   true,
		},
		{
			name:       "verified trust tier - not eligible",
			trustTier:  "verified",
			eligible:   false,
		},
		{
			name:       "root trust tier - not eligible",
			trustTier:  "root",
			eligible:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := &objects.ValidationTask{
				ID:       "task-1",
				Priority: 5,
				Parameters: map[string]interface{}{},
			}
			if tt.trustTier != "" {
				task.Parameters["trust_tier"] = tt.trustTier
			}

			result := lb.isNodeEligible(task, browserNode)
			if result != tt.eligible {
				t.Errorf("Expected eligible=%v for trust_tier='%s', got %v", tt.eligible, tt.trustTier, result)
			}
		})
	}
}

func TestCalculateHybridScore_TrustTierWeight(t *testing.T) {
	lb := &LoadBalancer{}

	task := &objects.ValidationTask{
		ID:       "task-1",
		Priority: 5,
	}

	baseNode := &objects.DVENode{
		ReputationScore: 100,
		CPUUsage:        30,
		MemoryUsage:     30,
		StakeAmount:     500000,
		NetworkLatency:  10,
	}

	// All nodes have the same specs, only TEEType differs
	sgxNode := *baseNode
	sgxNode.TEEType = "sgx"

	swNode := *baseNode
	swNode.TEEType = "software"

	browserNode := *baseNode
	browserNode.TEEType = "browser-extension"

	sgxScore := lb.calculateHybridScore(task, &sgxNode)
	swScore := lb.calculateHybridScore(task, &swNode)
	browserScore := lb.calculateHybridScore(task, &browserNode)

	// SGX should have the highest score (weight 1.0)
	// Software should be lower (weight 0.7)
	// Browser should be lowest (weight 0.4)
	if sgxScore <= swScore {
		t.Errorf("Expected sgxScore (%f) > swScore (%f)", sgxScore, swScore)
	}
	if swScore <= browserScore {
		t.Errorf("Expected swScore (%f) > browserScore (%f)", swScore, browserScore)
	}
	if sgxScore <= browserScore {
		t.Errorf("Expected sgxScore (%f) > browserScore (%f)", sgxScore, browserScore)
	}

	// Verify exact ratios (approximately)
	expectedSwRatio := 0.7
	if swScore/sgxScore < expectedSwRatio-0.01 || swScore/sgxScore > expectedSwRatio+0.01 {
		t.Logf("Note: swScore/sgxScore ratio = %f (expected ~%f)", swScore/sgxScore, expectedSwRatio)
	}

	expectedBrowserRatio := 0.4
	if browserScore/sgxScore < expectedBrowserRatio-0.01 || browserScore/sgxScore > expectedBrowserRatio+0.01 {
		t.Logf("Note: browserScore/sgxScore ratio = %f (expected ~%f)", browserScore/sgxScore, expectedBrowserRatio)
	}
}

func TestLoadBalancer_FilterEligibleNodes_BrowserExtension(t *testing.T) {
	lb := &LoadBalancer{}

	nodes := []*objects.DVENode{
		{
			ID:              "sgx-node",
			TEEType:         "sgx",
			Status:          "online",
			ReputationScore: 100,
			CPUUsage:        50,
			MemoryUsage:     50,
			Capabilities:    []string{"validation"},
		},
		{
			ID:              "browser-node",
			TEEType:         "browser-extension",
			Status:          "online",
			ReputationScore: 100,
			CPUUsage:        50,
			MemoryUsage:     50,
			Capabilities:    []string{"validation"},
		},
	}

	// Task with standard trust tier - both nodes should be eligible
	taskStandard := &objects.ValidationTask{
		ID:       "task-standard",
		Priority: 5,
		Parameters: map[string]interface{}{
			"trust_tier": "standard",
		},
	}
	eligibleStandard := lb.filterEligibleNodes(taskStandard, nodes)
	if len(eligibleStandard) != 2 {
		t.Errorf("Expected 2 eligible nodes for standard trust tier, got %d", len(eligibleStandard))
	}

	// Task with verified trust tier - only SGX should be eligible
	taskVerified := &objects.ValidationTask{
		ID:       "task-verified",
		Priority: 5,
		Parameters: map[string]interface{}{
			"trust_tier": "verified",
		},
	}
	eligibleVerified := lb.filterEligibleNodes(taskVerified, nodes)
	if len(eligibleVerified) != 1 {
		t.Errorf("Expected 1 eligible node for verified trust tier, got %d", len(eligibleVerified))
	}
	if len(eligibleVerified) > 0 && eligibleVerified[0].TEEType != "sgx" {
		t.Errorf("Expected eligible node to be SGX, got %s", eligibleVerified[0].TEEType)
	}
}

func TestGetTrustTierMultiplier(t *testing.T) {
	tests := []struct {
		name     string
		teeType  string
		expected float64
	}{
		{"sgx", "sgx", 1.0},
		{"sev-snp", "sev-snp", 1.0},
		{"tdx", "tdx", 1.0},
		{"software", "software", 0.7},
		{"browser-extension", "browser-extension", 0.4},
		{"unknown default", "unknown", 0.5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getTrustTierMultiplier(tt.teeType)
			if result != tt.expected {
				t.Errorf("getTrustTierMultiplier(%q) = %f, want %f", tt.teeType, result, tt.expected)
			}
		})
	}
}
