package classifier

import (
	"context"
	"regexp"
	"strings"

	"github.com/knirvchain/internal/blockchain"
)

type MemoryClassifier struct {
	errorPatterns   []*regexp.Regexp
	contextKeywords []string
	ideaMarkers     []string
}

func NewMemoryClassifier() *MemoryClassifier {
	return &MemoryClassifier{
		errorPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)(error|exception|failed|timeout)`),
			regexp.MustCompile(`(?i)(stack trace|panic|fatal)`),
		},
		contextKeywords: []string{
			"user prefers", "location", "environment", "setting",
			"timezone", "language preference",
		},
		ideaMarkers: []string{
			"what if", "could we", "hypothesis", "concept",
			"imagine", "innovative", "breakthrough",
		},
	}
}

func (c *MemoryClassifier) Classify(ctx context.Context, content string) blockchain.MemoryCategory {
	lowerContent := strings.ToLower(content)

	// Check for errors
	for _, pattern := range c.errorPatterns {
		if pattern.MatchString(lowerContent) {
			return blockchain.CategoryError
		}
	}

	// Check for context
	for _, keyword := range c.contextKeywords {
		if strings.Contains(lowerContent, keyword) {
			return blockchain.CategoryContext
		}
	}

	// Check for ideas
	for _, marker := range c.ideaMarkers {
		if strings.Contains(lowerContent, marker) {
			return blockchain.CategoryIdea
		}
	}

	// Check for task indicators
	if c.isTask(lowerContent) {
		return blockchain.CategoryTask
	}

	return blockchain.CategoryGeneral
}

func (c *MemoryClassifier) isTask(content string) bool {
	taskIndicators := []string{
		"todo", "remind me", "task", "action item",
		"need to", "must", "should", "deadline",
	}

	for _, indicator := range taskIndicators {
		if strings.Contains(content, indicator) {
			return true
		}
	}

	return false
}

func (c *MemoryClassifier) ClassifyWithConfidence(
	ctx context.Context,
	content string,
) (blockchain.MemoryCategory, float64) {
	// ML-based classification would go here
	// For now, return rule-based with fixed confidence
	category := c.Classify(ctx, content)
	return category, 0.85
}