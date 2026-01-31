package classifier

import (
	"context"
	"testing"

	"github.com/knirvchain/internal/blockchain"
)

func TestNewMemoryClassifier(t *testing.T) {
	classifier := NewMemoryClassifier()
	if classifier == nil {
		t.Fatal("Expected MemoryClassifier, got nil")
	}
	if len(classifier.errorPatterns) == 0 {
		t.Error("Expected error patterns to be initialized")
	}
	if len(classifier.contextKeywords) == 0 {
		t.Error("Expected context keywords to be initialized")
	}
	if len(classifier.ideaMarkers) == 0 {
		t.Error("Expected idea markers to be initialized")
	}
}

func TestClassifyError(t *testing.T) {
	classifier := NewMemoryClassifier()

	testCases := []string{
		"This is an error message",
		"Exception occurred during processing",
		"Operation failed with timeout",
		"Stack trace: panic in main",
		"Fatal error in database",
	}

	for _, content := range testCases {
		category := classifier.Classify(context.Background(), content)
		if category != blockchain.CategoryError {
			t.Errorf("Expected CategoryError for content '%s', got %s", content, category)
		}
	}
}

func TestClassifyContext(t *testing.T) {
	classifier := NewMemoryClassifier()

	testCases := []string{
		"User prefers dark mode",
		"Location is set to New York",
		"Environment configuration updated",
		"Timezone changed to UTC",
		"Language preference is English",
	}

	for _, content := range testCases {
		category := classifier.Classify(context.Background(), content)
		if category != blockchain.CategoryContext {
			t.Errorf("Expected CategoryContext for content '%s', got %s", content, category)
		}
	}
}

func TestClassifyIdea(t *testing.T) {
	classifier := NewMemoryClassifier()

	testCases := []string{
		"What if we could fly?",
		"Could we implement this feature?",
		"This is a hypothesis about AI",
		"New concept for blockchain",
		"Imagine a world without borders",
		"Innovative solution proposed",
		"Breakthrough in quantum computing",
	}

	for _, content := range testCases {
		category := classifier.Classify(context.Background(), content)
		if category != blockchain.CategoryIdea {
			t.Errorf("Expected CategoryIdea for content '%s', got %s", content, category)
		}
	}
}

func TestClassifyTask(t *testing.T) {
	classifier := NewMemoryClassifier()

	testCases := []string{
		"TODO: implement authentication",
		"Remind me to call the client",
		"This is a task for tomorrow",
		"Action item: review code",
		"I need to finish this",
		"Must complete by Friday",
		"Should update documentation",
		"Deadline approaching",
	}

	for _, content := range testCases {
		category := classifier.Classify(context.Background(), content)
		if category != blockchain.CategoryTask {
			t.Errorf("Expected CategoryTask for content '%s', got %s", content, category)
		}
	}
}

func TestClassifyGeneral(t *testing.T) {
	classifier := NewMemoryClassifier()

	content := "This is a regular message without special indicators"
	category := classifier.Classify(context.Background(), content)
	if category != blockchain.CategoryGeneral {
		t.Errorf("Expected CategoryGeneral for content '%s', got %s", content, category)
	}
}

func TestIsTask(t *testing.T) {
	classifier := NewMemoryClassifier()

	testCases := []struct {
		content  string
		expected bool
	}{
		{"todo: do something", true},
		{"remind me later", true},
		{"this is a task", true},
		{"action item here", true},
		{"need to complete", true},
		{"must finish", true},
		{"should work", true},
		{"deadline soon", true},
		{"regular message", false},
		{"just chatting", false},
	}

	for _, tc := range testCases {
		result := classifier.isTask(tc.content)
		if result != tc.expected {
			t.Errorf("isTask('%s') = %v, expected %v", tc.content, result, tc.expected)
		}
	}
}

func TestClassifyWithConfidence(t *testing.T) {
	classifier := NewMemoryClassifier()

	content := "This is a test message"
	category, confidence := classifier.ClassifyWithConfidence(context.Background(), content)

	if category != blockchain.CategoryGeneral {
		t.Errorf("Expected CategoryGeneral, got %s", category)
	}
	if confidence != 0.85 {
		t.Errorf("Expected confidence 0.85, got %f", confidence)
	}
}
