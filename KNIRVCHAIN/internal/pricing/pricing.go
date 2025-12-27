package pricing

import (
	"github.com/knirvchain/internal/blockchain"
)

const (
	BaseStorage   uint64 = 10
	BaseRetrieval uint64 = 5
	BaseEmbedding uint64 = 3

	SizeMultiplier     uint64 = 1
	ResultMultiplier   uint64 = 2
	PriorityMultiplier uint64 = 5
)

type PricingEngine struct{}

func NewPricingEngine() *PricingEngine {
	return &PricingEngine{}
}

func (p *PricingEngine) CalculateStorageCost(
	contentSizeBytes int,
	category blockchain.MemoryCategory,
	priority bool,
) uint64 {
	base := BaseStorage
	sizeKB := contentSizeBytes / 1024
	sizeCost := uint64(sizeKB) * SizeMultiplier

	categoryPremium := p.getCategoryPremium(category)

	total := base + sizeCost + categoryPremium

	if priority {
		total *= PriorityMultiplier
	}

	return total
}

func (p *PricingEngine) getCategoryPremium(category blockchain.MemoryCategory) uint64 {
	premiums := map[blockchain.MemoryCategory]uint64{
		blockchain.CategoryError:   2,
		blockchain.CategoryContext: 5,
		blockchain.CategoryIdea:    8,
		blockchain.CategoryTask:    3,
		blockchain.CategoryGeneral: 5,
	}

	if premium, ok := premiums[category]; ok {
		return premium
	}
	return 5
}

func (p *PricingEngine) CalculateRetrievalCost(
	resultLimit int,
	includeEmbeddings bool,
) uint64 {
	base := BaseRetrieval
	resultCost := uint64(resultLimit) * ResultMultiplier

	embeddingCost := uint64(0)
	if includeEmbeddings {
		embeddingCost = BaseEmbedding
	}

	return base + resultCost + embeddingCost
}

func (p *PricingEngine) CalculateSyncCost(pendingBlocks int) uint64 {
	return uint64(pendingBlocks) * 3
}

type CostEstimate struct {
	Operation string `json:"operation"`
	BaseCost  uint64 `json:"base_cost"`
	SizeCost  uint64 `json:"size_cost,omitempty"`
	Premium   uint64 `json:"premium,omitempty"`
	Total     uint64 `json:"total"`
}

func (p *PricingEngine) EstimateWithBreakdown(
	operation string,
	params map[string]interface{},
) CostEstimate {
	estimate := CostEstimate{
		Operation: operation,
	}

	switch operation {
	case "store":
		size := params["size"].(int)
		category := params["category"].(blockchain.MemoryCategory)

		estimate.BaseCost = BaseStorage
		estimate.SizeCost = uint64(size/1024) * SizeMultiplier
		estimate.Premium = p.getCategoryPremium(category)
		estimate.Total = estimate.BaseCost + estimate.SizeCost + estimate.Premium

	case "retrieve":
		limit := params["limit"].(int)
		estimate.BaseCost = BaseRetrieval
		estimate.SizeCost = uint64(limit) * ResultMultiplier
		estimate.Total = estimate.BaseCost + estimate.SizeCost
	}

	return estimate
}