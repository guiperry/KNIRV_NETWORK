package pricing

import (
	"testing"
)

func TestPricingEngine_SkillMintCost(t *testing.T) {
	pe := NewPricingEngine()

	tests := []struct {
		name     string
		size     int
		priority bool
		rank     int
		wantMin  uint64
		wantMax  uint64
	}{
		{
			name:     "basic skill mint",
			size:     1024,
			priority: false,
			rank:     0,
			wantMin:  10,
			wantMax:  20,
		},
		{
			name:     "skill mint with rank",
			size:     1024,
			priority: false,
			rank:     5,
			wantMin:  15,
			wantMax:  25,
		},
		{
			name:     "priority skill mint",
			size:     1024,
			priority: true,
			rank:     0,
			wantMin:  40,
			wantMax:  60,
		},
		{
			name:     "large content skill mint",
			size:     10240,
			priority: false,
			rank:     0,
			wantMin:  20,
			wantMax:  30,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pe.CalculateSkillMintCost(tt.size, tt.priority, tt.rank)
			if got < tt.wantMin || got > tt.wantMax {
				t.Errorf("CalculateSkillMintCost() = %v, want between %v and %v", got, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestPricingEngine_CapabilityMintCost(t *testing.T) {
	pe := NewPricingEngine()

	tests := []struct {
		name       string
		size       int
		priority   bool
		complexity int
		wantMin    uint64
		wantMax    uint64
	}{
		{
			name:       "basic capability mint",
			size:       1024,
			priority:   false,
			complexity: 1,
			wantMin:    15,
			wantMax:    25,
		},
		{
			name:       "high complexity capability",
			size:       1024,
			priority:   false,
			complexity: 3,
			wantMin:    25,
			wantMax:    35,
		},
		{
			name:       "priority capability",
			size:       1024,
			priority:   true,
			complexity: 1,
			wantMin:    60,
			wantMax:    100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pe.CalculateCapabilityMintCost(tt.size, tt.priority, tt.complexity)
			if got < tt.wantMin || got > tt.wantMax {
				t.Errorf("CalculateCapabilityMintCost() = %v, want between %v and %v", got, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestPricingEngine_PropertyMakeCost(t *testing.T) {
	pe := NewPricingEngine()

	tests := []struct {
		name         string
		size         int
		priority     bool
		inferenceNFT bool
		wantMin      uint64
		wantMax      uint64
	}{
		{
			name:         "basic property make",
			size:         1024,
			priority:     false,
			inferenceNFT: false,
			wantMin:      20,
			wantMax:      30,
		},
		{
			name:         "with inference NFT",
			size:         1024,
			priority:     false,
			inferenceNFT: true,
			wantMin:      35,
			wantMax:      45,
		},
		{
			name:         "priority property",
			size:         1024,
			priority:     true,
			inferenceNFT: false,
			wantMin:      80,
			wantMax:      120,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pe.CalculatePropertyMakeCost(tt.size, tt.priority, tt.inferenceNFT)
			if got < tt.wantMin || got > tt.wantMax {
				t.Errorf("CalculatePropertyMakeCost() = %v, want between %v and %v", got, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestPricingEngine_NodeCost(t *testing.T) {
	pe := NewPricingEngine()

	tests := []struct {
		name      string
		nodeType  NodeType
		size      int
		priority  bool
		params    map[string]interface{}
		wantError bool
	}{
		{
			name:      "skill node",
			nodeType:  NodeTypeSkill,
			size:      1024,
			priority:  false,
			params:    map[string]interface{}{"rank": 5},
			wantError: false,
		},
		{
			name:      "capability node",
			nodeType:  NodeTypeCapability,
			size:      1024,
			priority:  false,
			params:    map[string]interface{}{"complexity": 2},
			wantError: false,
		},
		{
			name:      "property node",
			nodeType:  NodeTypeProperty,
			size:      1024,
			priority:  false,
			params:    map[string]interface{}{"inference_nft": true},
			wantError: false,
		},
		{
			name:      "unknown node type",
			nodeType:  NodeType("unknown"),
			size:      1024,
			priority:  false,
			params:    nil,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := pe.CalculateNodeCost(tt.nodeType, tt.size, tt.priority, tt.params)
			if (err != nil) != tt.wantError {
				t.Errorf("CalculateNodeCost() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if !tt.wantError && got == 0 {
				t.Errorf("CalculateNodeCost() returned 0 for valid node type")
			}
		})
	}
}

func TestPricingEngine_QueryCost(t *testing.T) {
	pe := NewPricingEngine()

	tests := []struct {
		name              string
		resultLimit       int
		includeEmbeddings bool
		wantMin           uint64
		wantMax           uint64
	}{
		{
			name:              "basic query",
			resultLimit:       10,
			includeEmbeddings: false,
			wantMin:           25,
			wantMax:           30,
		},
		{
			name:              "query with embeddings",
			resultLimit:       10,
			includeEmbeddings: true,
			wantMin:           28,
			wantMax:           33,
		},
		{
			name:              "large result set",
			resultLimit:       100,
			includeEmbeddings: false,
			wantMin:           205,
			wantMax:           215,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pe.CalculateQueryCost(tt.resultLimit, tt.includeEmbeddings)
			if got < tt.wantMin || got > tt.wantMax {
				t.Errorf("CalculateQueryCost() = %v, want between %v and %v", got, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestPricingEngine_EstimateWithBreakdown(t *testing.T) {
	pe := NewPricingEngine()

	estimate := pe.EstimateWithBreakdown("skill_mint", map[string]interface{}{
		"size":     2048,
		"rank":     5,
		"priority": true,
	})

	if estimate.Operation != "skill_mint" {
		t.Errorf("Expected operation 'skill_mint', got '%s'", estimate.Operation)
	}
	if estimate.BaseCost == 0 {
		t.Error("Expected non-zero base cost")
	}
	if estimate.Total == 0 {
		t.Error("Expected non-zero total cost")
	}
	if estimate.PriorityCost == 0 {
		t.Error("Expected priority cost for priority=true")
	}
}
