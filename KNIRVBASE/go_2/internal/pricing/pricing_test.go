package pricing

import (
	"testing"

	"github.com/knirvchain/internal/blockchain"
)

func TestNewPricingEngine(t *testing.T) {
	engine := NewPricingEngine()
	if engine == nil {
		t.Fatal("Expected PricingEngine, got nil")
	}
}

func TestCalculateStorageCost(t *testing.T) {
	engine := NewPricingEngine()

	// Test basic storage cost
	cost := engine.CalculateStorageCost(1024, blockchain.CategoryGeneral, false)
	expected := BaseStorage + (1 * SizeMultiplier) + 5 // 10 + 1 + 5 = 16
	if cost != expected {
		t.Errorf("Expected cost %d, got %d", expected, cost)
	}

	// Test with priority
	costPriority := engine.CalculateStorageCost(1024, blockchain.CategoryGeneral, true)
	expectedPriority := expected * PriorityMultiplier // 16 * 5 = 80
	if costPriority != expectedPriority {
		t.Errorf("Expected priority cost %d, got %d", expectedPriority, costPriority)
	}

	// Test with different category
	costError := engine.CalculateStorageCost(2048, blockchain.CategoryError, false)
	expectedError := BaseStorage + (2 * SizeMultiplier) + 2 // 10 + 2 + 2 = 14
	if costError != expectedError {
		t.Errorf("Expected error category cost %d, got %d", expectedError, costError)
	}
}

func TestGetCategoryPremium(t *testing.T) {
	engine := NewPricingEngine()

	testCases := []struct {
		category blockchain.MemoryCategory
		expected uint64
	}{
		{blockchain.CategoryError, 2},
		{blockchain.CategoryContext, 5},
		{blockchain.CategoryIdea, 8},
		{blockchain.CategoryTask, 3},
		{blockchain.CategoryGeneral, 5},
	}

	for _, tc := range testCases {
		premium := engine.getCategoryPremium(tc.category)
		if premium != tc.expected {
			t.Errorf("Expected premium %d for category %s, got %d", tc.expected, tc.category, premium)
		}
	}
}

func TestCalculateRetrievalCost(t *testing.T) {
	engine := NewPricingEngine()

	// Test without embeddings
	cost := engine.CalculateRetrievalCost(10, false)
	expected := BaseRetrieval + (10 * ResultMultiplier) // 5 + 20 = 25
	if cost != expected {
		t.Errorf("Expected cost %d, got %d", expected, cost)
	}

	// Test with embeddings
	costWithEmbeddings := engine.CalculateRetrievalCost(10, true)
	expectedWithEmbeddings := expected + BaseEmbedding // 25 + 3 = 28
	if costWithEmbeddings != expectedWithEmbeddings {
		t.Errorf("Expected cost with embeddings %d, got %d", expectedWithEmbeddings, costWithEmbeddings)
	}
}

func TestCalculateSyncCost(t *testing.T) {
	engine := NewPricingEngine()

	cost := engine.CalculateSyncCost(5)
	expected := uint64(5 * 3) // 15
	if cost != expected {
		t.Errorf("Expected sync cost %d, got %d", expected, cost)
	}
}

func TestEstimateWithBreakdownStore(t *testing.T) {
	engine := NewPricingEngine()

	params := map[string]interface{}{
		"size":     2048,
		"category": blockchain.CategoryIdea,
	}

	estimate := engine.EstimateWithBreakdown("store", params)

	if estimate.Operation != "store" {
		t.Errorf("Expected operation 'store', got '%s'", estimate.Operation)
	}
	if estimate.BaseCost != BaseStorage {
		t.Errorf("Expected base cost %d, got %d", BaseStorage, estimate.BaseCost)
	}
	if estimate.SizeCost != 2*SizeMultiplier {
		t.Errorf("Expected size cost %d, got %d", 2*SizeMultiplier, estimate.SizeCost)
	}
	if estimate.Premium != 8 {
		t.Errorf("Expected premium 8, got %d", estimate.Premium)
	}
	expectedTotal := BaseStorage + 2*SizeMultiplier + 8 // 10 + 2 + 8 = 20
	if estimate.Total != expectedTotal {
		t.Errorf("Expected total %d, got %d", expectedTotal, estimate.Total)
	}
}

func TestEstimateWithBreakdownRetrieve(t *testing.T) {
	engine := NewPricingEngine()

	params := map[string]interface{}{
		"limit": 15,
	}

	estimate := engine.EstimateWithBreakdown("retrieve", params)

	if estimate.Operation != "retrieve" {
		t.Errorf("Expected operation 'retrieve', got '%s'", estimate.Operation)
	}
	if estimate.BaseCost != BaseRetrieval {
		t.Errorf("Expected base cost %d, got %d", BaseRetrieval, estimate.BaseCost)
	}
	if estimate.SizeCost != 15*ResultMultiplier {
		t.Errorf("Expected size cost %d, got %d", 15*ResultMultiplier, estimate.SizeCost)
	}
	expectedTotal := BaseRetrieval + 15*ResultMultiplier // 5 + 30 = 35
	if estimate.Total != expectedTotal {
		t.Errorf("Expected total %d, got %d", expectedTotal, estimate.Total)
	}
}