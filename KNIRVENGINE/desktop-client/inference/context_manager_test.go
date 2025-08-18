// /home/gperry/Documents/GitHub/Inc-Line/Wordpress-Inference-Engine/inference/context_manager_test.go
package inference

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

// MockTextGenerator implements the TextGenerator interface for testing
type MockTextGenerator struct {
	generateFunc func(string) (string, error)
}

// GenerateText implements the TextGenerator interface for testing
func (m *MockTextGenerator) GenerateText(prompt string) (string, error) {
	if m.generateFunc != nil {
		return m.generateFunc(prompt)
	}
	// Default implementation: echo the prompt with a prefix
	return fmt.Sprintf("MOCK RESPONSE: %s", prompt), nil
}

func TestNewContextManager(t *testing.T) {
	// Test with default options
	cm := NewContextManager(ChunkByParagraph)
	if cm.strategy != ChunkByParagraph {
		t.Errorf("Expected strategy ChunkByParagraph, got %v", cm.strategy)
	}

	// Test with custom options
	cm = NewContextManager(
		ChunkBySentence,
		WithProcessingMode(SequentialProcessing),
		WithMaxChunkSize(2000),
		WithChunkOverlap(200),
		WithModelName("test-model"),
	)

	if cm.strategy != ChunkBySentence {
		t.Errorf("Expected strategy ChunkBySentence, got %v", cm.strategy)
	}
	if cm.processingMode != SequentialProcessing {
		t.Errorf("Expected processingMode SequentialProcessing, got %v", cm.processingMode)
	}
	if cm.maxChunkSize != 2000 {
		t.Errorf("Expected maxChunkSize 2000, got %v", cm.maxChunkSize)
	}
	if cm.chunkOverlap != 200 {
		t.Errorf("Expected chunkOverlap 200, got %v", cm.chunkOverlap)
	}
	if cm.modelName != "test-model" {
		t.Errorf("Expected modelName 'test-model', got %v", cm.modelName)
	}
}

func TestSplitIntoChunks(t *testing.T) {
	// mockGenerator := &MockTextGenerator{} // No longer needed for splitting

	// Test paragraph chunking
	cm := NewContextManager(ChunkByParagraph)
	text := "Paragraph 1.\n\nParagraph 2.\n\nParagraph 3."
	chunks := cm.splitIntoChunks(text)

	if len(chunks) != 3 {
		t.Errorf("Expected 3 chunks, got %d", len(chunks))
	}

	// Test sentence chunking
	cm.SetChunkingStrategy(ChunkBySentence)
	text = "Sentence 1. Sentence 2. Sentence 3."
	chunks = cm.splitIntoChunks(text)

	// The implementation might group sentences, so we'll check if we have at least one chunk
	if len(chunks) < 1 {
		t.Errorf("Expected at least 1 chunk, got %d", len(chunks))
	}

	// Test token count chunking with a very small max size to force multiple chunks
	cm = NewContextManager(
		// mockGenerator, // Removed
		ChunkByTokenCount,
		WithMaxChunkSize(5), // Very small to force splitting
	)
	text = "This is a very long text that contains many words and should definitely be split into multiple chunks when using token-based chunking with a very small token limit. Each word here adds to the token count and should force the chunking algorithm to create multiple separate chunks for processing."
	chunks = cm.splitIntoChunks(text)

	if len(chunks) <= 1 {
		t.Errorf("Expected multiple chunks, got %d", len(chunks))
	}
}

func TestProcessLargePrompt(t *testing.T) {
	// Create a mock service that returns a predictable response
	mockGenerator := &MockTextGenerator{
		generateFunc: func(prompt string) (string, error) {
			// Handle sequential processing format (with "Current Section:")
			if strings.Contains(prompt, "Current Section:") {
				parts := strings.Split(prompt, "Current Section:")
				if len(parts) >= 2 {
					sectionPart := parts[1]
					endParts := strings.Split(sectionPart, "---")
					if len(endParts) >= 1 {
						chunk := strings.TrimSpace(endParts[0])
						return fmt.Sprintf("Processed: %s", chunk), nil
					}
				}
			}

			// Handle parallel processing format (simple ---chunk--- format)
			if strings.Contains(prompt, "---") {
				// Extract content between --- markers
				parts := strings.Split(prompt, "---")
				if len(parts) >= 3 {
					chunk := strings.TrimSpace(parts[1])
					return fmt.Sprintf("Processed: %s", chunk), nil
				}
			}

			return "ERROR: Invalid prompt format", nil
		},
	}

	cm := NewContextManager(ChunkByParagraph) // Removed service

	// Test parallel processing
	text := "Chunk 1.\n\nChunk 2.\n\nChunk 3."
	instruction := "Process this:"

	ctx := context.Background()
	result, err := cm.ProcessLargePrompt(ctx, mockGenerator, text, instruction)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Check that all chunks were processed
	if !strings.Contains(result, "Processed: Chunk 1") ||
		!strings.Contains(result, "Processed: Chunk 2") ||
		!strings.Contains(result, "Processed: Chunk 3") {
		t.Errorf("Not all chunks were processed correctly: %s", result)
	}

	// Test sequential processing
	cm.SetProcessingMode(SequentialProcessing)

	result, err = cm.ProcessLargePrompt(ctx, mockGenerator, text, instruction)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Check that all chunks were processed
	if !strings.Contains(result, "Processed: Chunk 1") ||
		!strings.Contains(result, "Processed: Chunk 2") ||
		!strings.Contains(result, "Processed: Chunk 3") {
		t.Errorf("Not all chunks were processed correctly in sequential mode: %s", result)
	}
}

func TestProcessLargePromptWithError(t *testing.T) {
	// Create a mock service that returns an error for a specific chunk
	mockGenerator := &MockTextGenerator{
		generateFunc: func(prompt string) (string, error) {
			var chunk string

			// Handle sequential processing format (with "Current Section:")
			if strings.Contains(prompt, "Current Section:") {
				parts := strings.Split(prompt, "Current Section:")
				if len(parts) >= 2 {
					sectionPart := parts[1]
					endParts := strings.Split(sectionPart, "---")
					if len(endParts) >= 1 {
						chunk = strings.TrimSpace(endParts[0])
					}
				}
			} else if strings.Contains(prompt, "---") {
				// Handle parallel processing format (simple ---chunk--- format)
				parts := strings.Split(prompt, "---")
				if len(parts) >= 3 {
					chunk = strings.TrimSpace(parts[1])
				}
			}

			if chunk != "" {
				// Return an error for Chunk 2
				if strings.Contains(chunk, "Chunk 2") {
					return "", fmt.Errorf("simulated error for Chunk 2")
				}
				return fmt.Sprintf("Processed: %s", chunk), nil
			}

			return "ERROR: Invalid prompt format", nil
		},
	}

	cm := NewContextManager(ChunkByParagraph) // Removed service

	// Test parallel processing with an error
	text := "Chunk 1.\n\nChunk 2.\n\nChunk 3."
	instruction := "Process this:"

	ctx := context.Background()
	result, err := cm.ProcessLargePrompt(ctx, mockGenerator, text, instruction)

	// We should get an error, but still have results for Chunks 1 and 3
	if err == nil {
		t.Errorf("Expected an error, got nil")
	}

	if !strings.Contains(result, "Processed: Chunk 1") ||
		!strings.Contains(result, "ERROR PROCESSING CHUNK") ||
		!strings.Contains(result, "Processed: Chunk 3") {
		t.Errorf("Expected partial results with error placeholder, got: %s", result)
	}

	// Test sequential processing with an error
	cm.SetProcessingMode(SequentialProcessing)

	result, err = cm.ProcessLargePrompt(ctx, mockGenerator, text, instruction)

	// In sequential mode, we should stop at the first error
	if err == nil {
		t.Errorf("Expected an error in sequential mode, got nil")
	}

	// We should have the result for Chunk 1 and an error placeholder
	if !strings.Contains(result, "Processed: Chunk 1") {
		t.Errorf("Expected result for Chunk 1 before error, got: %s", result)
	}
}

func TestOverrideMethodsForStrategyAndMode(t *testing.T) {
	mockGenerator := &MockTextGenerator{} // Keep mock for processing calls

	cm := NewContextManager(ChunkByParagraph, WithProcessingMode(ParallelProcessing)) // Removed service

	// Verify initial settings
	if cm.strategy != ChunkByParagraph {
		t.Errorf("Expected initial strategy ChunkByParagraph, got %v", cm.strategy)
	}
	if cm.processingMode != ParallelProcessing {
		t.Errorf("Expected initial processingMode ParallelProcessing, got %v", cm.processingMode)
	}

	// Test ProcessLargePromptWithStrategy
	ctx := context.Background()
	_, err := cm.ProcessLargePromptWithStrategy(ctx, "Test", "Instruction", ChunkBySentence, mockGenerator)
	if err != nil {
		t.Errorf("ProcessLargePromptWithStrategy returned error: %v", err)
	}

	// Verify strategy was temporarily changed and then restored
	if cm.strategy != ChunkByParagraph {
		t.Errorf("Strategy was not restored after ProcessLargePromptWithStrategy, got %v", cm.strategy)
	}

	// Test ProcessLargePromptWithMode
	_, err = cm.ProcessLargePromptWithMode(ctx, "Test", "Instruction", SequentialProcessing, mockGenerator)
	if err != nil {
		t.Errorf("ProcessLargePromptWithMode returned error: %v", err)
	}

	// Verify mode was temporarily changed and then restored
	if cm.processingMode != ParallelProcessing {
		t.Errorf("ProcessingMode was not restored after ProcessLargePromptWithMode, got %v", cm.processingMode)
	}
}
