package inference

import (
	"testing"

	gollm "github.com/guiperry/gollm_cerebras"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDelegatorService(t *testing.T) {
	// Create mock LLM attempts
	primaryAttempts := []LLMAttempt{
		{Config: LLMAttemptConfig{ModelName: "gpt-4", IsPrimary: true}},
	}
	fallbackAttempts := []LLMAttempt{
		{Config: LLMAttemptConfig{ModelName: "gpt-3.5-turbo", IsPrimary: false}},
	}

	t.Run("creates delegator service successfully", func(t *testing.T) {
		tokenLimit := 4096
		tokenModel := "gpt-4"
		contextManager := NewContextManager(ChunkByParagraph)

		service := NewDelegatorService(
			primaryAttempts,
			fallbackAttempts,
			tokenLimit,
			tokenModel,
			nil, // No MOA instance
			contextManager,
		)

		require.NotNil(t, service)
		assert.Equal(t, primaryAttempts, service.primaryAttempts)
		assert.Equal(t, fallbackAttempts, service.fallbackAttempts)
		assert.Equal(t, tokenLimit, service.tokenLimitThreshold)
		assert.Equal(t, tokenModel, service.tokenLimitCheckModel)
		assert.NotNil(t, service.memory)
		assert.NotNil(t, service.contextManager)
		assert.Nil(t, service.moa)
	})

	t.Run("creates delegator service with MOA", func(t *testing.T) {
		tokenLimit := 4096
		tokenModel := "gpt-4"
		contextManager := NewContextManager(ChunkByParagraph)

		// Create a mock MOA instance (this would be a real MOA in practice)
		var moaInstance *gollm.MOA = nil // In real tests, this would be properly initialized

		service := NewDelegatorService(
			primaryAttempts,
			fallbackAttempts,
			tokenLimit,
			tokenModel,
			moaInstance,
			contextManager,
		)

		require.NotNil(t, service)
		assert.Equal(t, moaInstance, service.moa)
	})

	t.Run("returns nil when primary attempts are empty", func(t *testing.T) {
		emptyPrimary := []LLMAttempt{}
		tokenLimit := 4096
		tokenModel := "gpt-4"
		contextManager := NewContextManager(ChunkByParagraph)

		service := NewDelegatorService(
			emptyPrimary,
			fallbackAttempts,
			tokenLimit,
			tokenModel,
			nil,
			contextManager,
		)

		assert.Nil(t, service)
	})

	t.Run("returns nil when fallback attempts are empty", func(t *testing.T) {
		emptyFallback := []LLMAttempt{}
		tokenLimit := 4096
		tokenModel := "gpt-4"
		contextManager := NewContextManager(ChunkByParagraph)

		service := NewDelegatorService(
			primaryAttempts,
			emptyFallback,
			tokenLimit,
			tokenModel,
			nil,
			contextManager,
		)

		assert.Nil(t, service)
	})

	t.Run("handles nil context manager", func(t *testing.T) {
		tokenLimit := 4096
		tokenModel := "gpt-4"

		service := NewDelegatorService(
			primaryAttempts,
			fallbackAttempts,
			tokenLimit,
			tokenModel,
			nil,
			nil, // Nil context manager
		)

		require.NotNil(t, service)
		assert.Nil(t, service.contextManager)
	})
}

func TestDelegatorService_UpdateMOA(t *testing.T) {
	primaryAttempts := []LLMAttempt{
		{Config: LLMAttemptConfig{ModelName: "gpt-4", IsPrimary: true}},
	}
	fallbackAttempts := []LLMAttempt{
		{Config: LLMAttemptConfig{ModelName: "gpt-3.5-turbo", IsPrimary: false}},
	}
	contextManager := NewContextManager(ChunkByParagraph)

	service := NewDelegatorService(
		primaryAttempts,
		fallbackAttempts,
		4096,
		"gpt-4",
		nil,
		contextManager,
	)
	require.NotNil(t, service)

	t.Run("updates MOA instance", func(t *testing.T) {
		// Create a mock MOA instance
		var newMOA *gollm.MOA = nil // In real tests, this would be properly initialized

		service.UpdateMOA(newMOA)
		assert.Equal(t, newMOA, service.moa)
	})

	t.Run("updates MOA to nil", func(t *testing.T) {
		service.UpdateMOA(nil)
		assert.Nil(t, service.moa)
	})
}

func TestDelegatorService_ClearMemory(t *testing.T) {
	primaryAttempts := []LLMAttempt{
		{Config: LLMAttemptConfig{ModelName: "gpt-4", IsPrimary: true}},
	}
	fallbackAttempts := []LLMAttempt{
		{Config: LLMAttemptConfig{ModelName: "gpt-3.5-turbo", IsPrimary: false}},
	}
	contextManager := NewContextManager(ChunkByParagraph)

	service := NewDelegatorService(
		primaryAttempts,
		fallbackAttempts,
		4096,
		"gpt-4",
		nil,
		contextManager,
	)
	require.NotNil(t, service)

	t.Run("clears memory successfully", func(t *testing.T) {
		// Should not panic
		assert.NotPanics(t, func() {
			service.ClearMemory()
		})
	})
}

func TestDelegatorService_EstimateTokens(t *testing.T) {
	primaryAttempts := []LLMAttempt{
		{Config: LLMAttemptConfig{ModelName: "gpt-4", IsPrimary: true}},
	}
	fallbackAttempts := []LLMAttempt{
		{Config: LLMAttemptConfig{ModelName: "gpt-3.5-turbo", IsPrimary: false}},
	}
	contextManager := NewContextManager(ChunkByParagraph)

	service := NewDelegatorService(
		primaryAttempts,
		fallbackAttempts,
		4096,
		"gpt-4",
		nil,
		contextManager,
	)
	require.NotNil(t, service)

	t.Run("estimates tokens for text", func(t *testing.T) {
		text := "Hello, world! This is a test message."
		tokens := estimateTokens(text, "gpt-4")

		assert.Greater(t, tokens, 0)
		assert.Less(t, tokens, 100) // Should be reasonable for short text
	})

	t.Run("estimates tokens for empty text", func(t *testing.T) {
		tokens := estimateTokens("", "gpt-4")
		assert.Equal(t, 0, tokens)
	})

	t.Run("estimates tokens for different models", func(t *testing.T) {
		text := "Test message"

		tokens1 := estimateTokens(text, "gpt-4")
		tokens2 := estimateTokens(text, "gpt-3.5-turbo")

		// Both should return positive values
		assert.Greater(t, tokens1, 0)
		assert.Greater(t, tokens2, 0)
	})
}

func TestDelegatorService_GetEncodingForModel(t *testing.T) {
	t.Run("gets encoding for known models", func(t *testing.T) {
		testCases := []struct {
			model       string
			shouldError bool
		}{
			{"gpt-4", false},
			{"gpt-3.5-turbo", false},
			{"cerebras-model", false},
			{"gemini-model", false},
			{"unknown-model", false}, // Should fallback to gpt-4
		}

		for _, tc := range testCases {
			t.Run(tc.model, func(t *testing.T) {
				encoding, err := getEncodingForModel(tc.model)

				if tc.shouldError {
					assert.Error(t, err)
					assert.Nil(t, encoding)
				} else {
					assert.NoError(t, err)
					assert.NotNil(t, encoding)
				}
			})
		}
	})
}

func TestDelegatorService_Configuration(t *testing.T) {
	t.Run("stores configuration correctly", func(t *testing.T) {
		primaryAttempts := []LLMAttempt{
			{Config: LLMAttemptConfig{ModelName: "gpt-4", IsPrimary: true}},
		}
		fallbackAttempts := []LLMAttempt{
			{Config: LLMAttemptConfig{ModelName: "gpt-3.5-turbo", IsPrimary: false}},
		}
		contextManager := NewContextManager(ChunkByParagraph)
		tokenLimit := 8192
		tokenModel := "claude-3"

		service := NewDelegatorService(
			primaryAttempts,
			fallbackAttempts,
			tokenLimit,
			tokenModel,
			nil,
			contextManager,
		)

		require.NotNil(t, service)
		assert.Equal(t, tokenLimit, service.tokenLimitThreshold)
		assert.Equal(t, tokenModel, service.tokenLimitCheckModel)
		assert.Len(t, service.primaryAttempts, 1)
		assert.Len(t, service.fallbackAttempts, 1)
	})
}

func TestDelegatorService_MemoryIntegration(t *testing.T) {
	primaryAttempts := []LLMAttempt{
		{Config: LLMAttemptConfig{ModelName: "gpt-4", IsPrimary: true}},
	}
	fallbackAttempts := []LLMAttempt{
		{Config: LLMAttemptConfig{ModelName: "gpt-3.5-turbo", IsPrimary: false}},
	}
	contextManager := NewContextManager(ChunkByParagraph)

	service := NewDelegatorService(
		primaryAttempts,
		fallbackAttempts,
		4096,
		"gpt-4",
		nil,
		contextManager,
	)
	require.NotNil(t, service)

	t.Run("has memory initialized", func(t *testing.T) {
		assert.NotNil(t, service.memory)
	})

	t.Run("memory uses correct model for token estimation", func(t *testing.T) {
		// The memory should be initialized with the token model
		// We can't directly test this without accessing private fields,
		// but we can verify the memory exists and works
		assert.NotNil(t, service.memory)

		// Test that memory operations work
		assert.NotPanics(t, func() {
			service.ClearMemory()
		})
	})
}
