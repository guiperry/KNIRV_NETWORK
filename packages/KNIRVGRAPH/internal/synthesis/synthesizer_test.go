package synthesis

import (
	"testing"
)

func TestNewSynthesizer(t *testing.T) {
	s := NewSynthesizer("", "")
	if s == nil {
		t.Fatal("Expected non-nil synthesizer")
	}
	if s.endpoint != "http://localhost:11434" {
		t.Errorf("Expected default endpoint, got %s", s.endpoint)
	}
}

func TestBuildPrompt(t *testing.T) {
	s := NewSynthesizer("", "")
	prompt := s.buildPrompt("what is x", "context here", 100)
	if prompt == "" {
		t.Error("Expected non-empty prompt")
	}
}

func TestSplitAnswerReasoning(t *testing.T) {
	answer, reasoning := splitAnswerReasoning("The answer is 42. Reasoning: because math")
	if answer != "The answer is 42." {
		t.Errorf("Expected answer 'The answer is 42.', got %s", answer)
	}
	if reasoning != "because math" {
		t.Errorf("Expected reasoning 'because math', got %s", reasoning)
	}
}

func TestTruncate(t *testing.T) {
	if truncate("hello", 3) != "hel..." {
		t.Errorf("Expected 'hel...', got %s", truncate("hello", 3))
	}
	if truncate("hi", 10) != "hi" {
		t.Errorf("Expected 'hi', got %s", truncate("hi", 10))
	}
}
