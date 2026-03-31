package pricing

import (
	"fmt"
	"math"
)

type NodeType string

const (
	NodeTypeSkill      NodeType = "skill"
	NodeTypeCapability NodeType = "capability"
	NodeTypeProperty   NodeType = "property"
)

const (
	BaseSkillMint      uint64 = 10
	BaseCapabilityMint uint64 = 15
	BasePropertyMake   uint64 = 20

	BaseQuery      uint64 = 5
	BaseValidation uint64 = 2

	SizeMultiplier     uint64 = 1
	PriorityMultiplier uint64 = 5
)

type PricingEngine struct{}

func NewPricingEngine() *PricingEngine {
	return &PricingEngine{}
}

func (p *PricingEngine) CalculateSkillMintCost(
	contentSizeBytes int,
	priority bool,
	rank int,
) uint64 {
	base := BaseSkillMint
	sizeKB := contentSizeBytes / 1024
	sizeCost := uint64(sizeKB) * SizeMultiplier

	rankPremium := uint64(0)
	if rank > 0 {
		rankPremium = uint64(math.Min(float64(rank), 10))
	}

	total := base + sizeCost + rankPremium

	if priority {
		total *= PriorityMultiplier
	}

	return total
}

func (p *PricingEngine) CalculateCapabilityMintCost(
	contentSizeBytes int,
	priority bool,
	complexity int,
) uint64 {
	base := BaseCapabilityMint
	sizeKB := contentSizeBytes / 1024
	sizeCost := uint64(sizeKB) * SizeMultiplier

	complexityPremium := uint64(0)
	switch complexity {
	case 1:
		complexityPremium = 2
	case 2:
		complexityPremium = 5
	case 3:
		complexityPremium = 10
	}

	total := base + sizeCost + complexityPremium

	if priority {
		total *= PriorityMultiplier
	}

	return total
}

func (p *PricingEngine) CalculatePropertyMakeCost(
	contentSizeBytes int,
	priority bool,
	inferenceNFT bool,
) uint64 {
	base := BasePropertyMake
	sizeKB := contentSizeBytes / 1024
	sizeCost := uint64(sizeKB) * SizeMultiplier

	inferencePremium := uint64(0)
	if inferenceNFT {
		inferencePremium = 15
	}

	total := base + sizeCost + inferencePremium

	if priority {
		total *= PriorityMultiplier
	}

	return total
}

func (p *PricingEngine) CalculateNodeCost(
	nodeType NodeType,
	contentSizeBytes int,
	priority bool,
	params map[string]interface{},
) (uint64, error) {
	switch nodeType {
	case NodeTypeSkill:
		rank := 0
		if r, ok := params["rank"].(int); ok {
			rank = r
		}
		return p.CalculateSkillMintCost(contentSizeBytes, priority, rank), nil

	case NodeTypeCapability:
		complexity := 1
		if c, ok := params["complexity"].(int); ok {
			complexity = c
		}
		return p.CalculateCapabilityMintCost(contentSizeBytes, priority, complexity), nil

	case NodeTypeProperty:
		inferenceNFT := false
		if i, ok := params["inference_nft"].(bool); ok {
			inferenceNFT = i
		}
		return p.CalculatePropertyMakeCost(contentSizeBytes, priority, inferenceNFT), nil

	default:
		return 0, fmt.Errorf("unknown node type: %s", nodeType)
	}
}

func (p *PricingEngine) CalculateQueryCost(
	resultLimit int,
	includeEmbeddings bool,
) uint64 {
	base := BaseQuery
	resultCost := uint64(resultLimit) * 2

	embeddingCost := uint64(0)
	if includeEmbeddings {
		embeddingCost = 3
	}

	return base + resultCost + embeddingCost
}

func (p *PricingEngine) CalculateValidationCost(
	numValidations int,
) uint64 {
	return uint64(numValidations) * BaseValidation
}

func (p *PricingEngine) CalculateSyncCost(pendingBlocks int) uint64 {
	return uint64(pendingBlocks) * 3
}

type CostEstimate struct {
	Operation    string `json:"operation"`
	NodeType     string `json:"node_type,omitempty"`
	BaseCost     uint64 `json:"base_cost"`
	SizeCost     uint64 `json:"size_cost,omitempty"`
	Premium      uint64 `json:"premium,omitempty"`
	PriorityCost uint64 `json:"priority_cost,omitempty"`
	Total        uint64 `json:"total"`
}

func (p *PricingEngine) EstimateWithBreakdown(
	operation string,
	params map[string]interface{},
) CostEstimate {
	estimate := CostEstimate{
		Operation: operation,
	}

	switch operation {
	case "skill_mint":
		size := params["size"].(int)
		rank := 0
		if r, ok := params["rank"].(int); ok {
			rank = r
		}
		priority := false
		if pr, ok := params["priority"].(bool); ok {
			priority = pr
		}

		estimate.NodeType = string(NodeTypeSkill)
		estimate.BaseCost = BaseSkillMint
		estimate.SizeCost = uint64(size/1024) * SizeMultiplier
		if rank > 0 {
			estimate.Premium = uint64(math.Min(float64(rank), 10))
		}
		if priority {
			estimate.PriorityCost = (estimate.BaseCost + estimate.SizeCost + estimate.Premium) * (PriorityMultiplier - 1)
		}
		estimate.Total = estimate.BaseCost + estimate.SizeCost + estimate.Premium + estimate.PriorityCost

	case "capability_mint":
		size := params["size"].(int)
		complexity := 1
		if c, ok := params["complexity"].(int); ok {
			complexity = c
		}
		priority := false
		if pr, ok := params["priority"].(bool); ok {
			priority = pr
		}

		estimate.NodeType = string(NodeTypeCapability)
		estimate.BaseCost = BaseCapabilityMint
		estimate.SizeCost = uint64(size/1024) * SizeMultiplier

		switch complexity {
		case 2:
			estimate.Premium = 5
		case 3:
			estimate.Premium = 10
		default:
			estimate.Premium = 2
		}

		if priority {
			estimate.PriorityCost = (estimate.BaseCost + estimate.SizeCost + estimate.Premium) * (PriorityMultiplier - 1)
		}
		estimate.Total = estimate.BaseCost + estimate.SizeCost + estimate.Premium + estimate.PriorityCost

	case "property_make":
		size := params["size"].(int)
		inferenceNFT := false
		if i, ok := params["inference_nft"].(bool); ok {
			inferenceNFT = i
		}
		priority := false
		if pr, ok := params["priority"].(bool); ok {
			priority = pr
		}

		estimate.NodeType = string(NodeTypeProperty)
		estimate.BaseCost = BasePropertyMake
		estimate.SizeCost = uint64(size/1024) * SizeMultiplier
		if inferenceNFT {
			estimate.Premium = 15
		}

		if priority {
			estimate.PriorityCost = (estimate.BaseCost + estimate.SizeCost + estimate.Premium) * (PriorityMultiplier - 1)
		}
		estimate.Total = estimate.BaseCost + estimate.SizeCost + estimate.Premium + estimate.PriorityCost

	case "query":
		limit := params["limit"].(int)
		estimate.BaseCost = BaseQuery
		estimate.SizeCost = uint64(limit) * 2
		if includeEmbeddings, ok := params["include_embeddings"].(bool); ok && includeEmbeddings {
			estimate.Premium = 3
		}
		estimate.Total = estimate.BaseCost + estimate.SizeCost + estimate.Premium
	}

	return estimate
}
