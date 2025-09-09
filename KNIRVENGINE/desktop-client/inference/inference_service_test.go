package inference

import (
	"testing"

	"KNIRVENGINE/desktop-client/database"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewInferenceService(t *testing.T) {
	t.Run("creates new inference service successfully", func(t *testing.T) {
		// Create a mock database
		db := &database.SimpleDomainDB{}

		service, err := NewInferenceService(db)

		require.NoError(t, err)
		require.NotNil(t, service)
		assert.NotNil(t, service.primaryAttempts)
		assert.NotNil(t, service.fallbackAttempts)
		assert.NotNil(t, service.contextManager)
		assert.Empty(t, service.primaryAttempts)
		assert.Empty(t, service.fallbackAttempts)
		assert.False(t, service.isRunning)
		assert.Nil(t, service.delegator)
		assert.Nil(t, service.moa)
	})

	t.Run("creates service with nil database", func(t *testing.T) {
		service, err := NewInferenceService(nil)

		require.NoError(t, err)
		require.NotNil(t, service)
		assert.NotNil(t, service.contextManager)
	})
}

func TestInferenceService_IsRunning(t *testing.T) {
	db := &database.SimpleDomainDB{}
	service, err := NewInferenceService(db)
	require.NoError(t, err)

	t.Run("returns false when not started", func(t *testing.T) {
		assert.False(t, service.IsRunning())
	})

	t.Run("returns true after starting", func(t *testing.T) {
		// Mock the start process by setting isRunning directly
		service.mutex.Lock()
		service.isRunning = true
		service.mutex.Unlock()

		assert.True(t, service.IsRunning())
	})
}

func TestInferenceService_GetName(t *testing.T) {
	db := &database.SimpleDomainDB{}
	service, err := NewInferenceService(db)
	require.NoError(t, err)

	t.Run("returns service name", func(t *testing.T) {
		name := service.GetName()
		assert.Equal(t, "InferenceService(Delegator+MOA)", name)
	})
}

func TestInferenceService_ClearConversationHistory(t *testing.T) {
	db := &database.SimpleDomainDB{}
	service, err := NewInferenceService(db)
	require.NoError(t, err)

	t.Run("clears conversation history when delegator is nil", func(t *testing.T) {
		// Should not panic when delegator is nil
		assert.NotPanics(t, func() {
			service.ClearConversationHistory()
		})
	})

	t.Run("clears conversation history when delegator exists", func(t *testing.T) {
		// Create a mock delegator
		mockDelegator := &DelegatorService{}
		service.delegator = mockDelegator

		// Should not panic
		assert.NotPanics(t, func() {
			service.ClearConversationHistory()
		})
	})
}

func TestInferenceService_GetPrimaryModels(t *testing.T) {
	db := &database.SimpleDomainDB{}
	service, err := NewInferenceService(db)
	require.NoError(t, err)

	t.Run("returns empty slice when no primary models", func(t *testing.T) {
		models := service.GetPrimaryModels()
		assert.Empty(t, models)
	})

	t.Run("returns primary model names", func(t *testing.T) {
		// Add mock primary attempts
		service.primaryAttempts = []LLMAttempt{
			{Config: LLMAttemptConfig{ModelName: "gpt-4", IsPrimary: true}},
			{Config: LLMAttemptConfig{ModelName: "claude-3", IsPrimary: true}},
		}

		models := service.GetPrimaryModels()
		assert.Len(t, models, 2)
		assert.Contains(t, models, "gpt-4")
		assert.Contains(t, models, "claude-3")
	})
}

func TestInferenceService_GetFallbackModels(t *testing.T) {
	db := &database.SimpleDomainDB{}
	service, err := NewInferenceService(db)
	require.NoError(t, err)

	t.Run("returns empty slice when no fallback models", func(t *testing.T) {
		models := service.GetFallbackModels()
		assert.Empty(t, models)
	})

	t.Run("returns fallback model names", func(t *testing.T) {
		// Add mock fallback attempts
		service.fallbackAttempts = []LLMAttempt{
			{Config: LLMAttemptConfig{ModelName: "gpt-3.5-turbo", IsPrimary: false}},
			{Config: LLMAttemptConfig{ModelName: "llama-2", IsPrimary: false}},
		}

		models := service.GetFallbackModels()
		assert.Len(t, models, 2)
		assert.Contains(t, models, "gpt-3.5-turbo")
		assert.Contains(t, models, "llama-2")
	})
}

func TestInferenceService_GetProxyModel(t *testing.T) {
	db := &database.SimpleDomainDB{}
	service, err := NewInferenceService(db)
	require.NoError(t, err)

	t.Run("returns empty string when no MOA primary model", func(t *testing.T) {
		model := service.GetProxyModel()
		assert.Empty(t, model)
	})

	t.Run("returns MOA primary model name", func(t *testing.T) {
		service.moaPrimaryModelName = "gpt-4"

		model := service.GetProxyModel()
		assert.Equal(t, "gpt-4", model)
	})
}

func TestInferenceService_GetBaseModel(t *testing.T) {
	db := &database.SimpleDomainDB{}
	service, err := NewInferenceService(db)
	require.NoError(t, err)

	t.Run("returns empty string when no MOA fallback model", func(t *testing.T) {
		model := service.GetBaseModel()
		assert.Empty(t, model)
	})

	t.Run("returns MOA fallback model name", func(t *testing.T) {
		service.moaFallbackModelName = "claude-3"

		model := service.GetBaseModel()
		assert.Equal(t, "claude-3", model)
	})
}

func TestLLMAttemptConfig(t *testing.T) {
	t.Run("creates valid LLM attempt config", func(t *testing.T) {
		config := LLMAttemptConfig{
			ProviderName: "openai",
			ModelName:    "gpt-4",
			APIKeyEnvVar: "OPENAI_API_KEY",
			MaxTokens:    4096,
			IsPrimary:    true,
		}

		assert.Equal(t, "openai", config.ProviderName)
		assert.Equal(t, "gpt-4", config.ModelName)
		assert.Equal(t, "OPENAI_API_KEY", config.APIKeyEnvVar)
		assert.Equal(t, 4096, config.MaxTokens)
		assert.True(t, config.IsPrimary)
	})

	t.Run("creates fallback config", func(t *testing.T) {
		config := LLMAttemptConfig{
			ProviderName: "anthropic",
			ModelName:    "claude-3",
			APIKeyEnvVar: "ANTHROPIC_API_KEY",
			MaxTokens:    2048,
			IsPrimary:    false,
		}

		assert.Equal(t, "anthropic", config.ProviderName)
		assert.Equal(t, "claude-3", config.ModelName)
		assert.Equal(t, "ANTHROPIC_API_KEY", config.APIKeyEnvVar)
		assert.Equal(t, 2048, config.MaxTokens)
		assert.False(t, config.IsPrimary)
	})
}

func TestLLMAttempt(t *testing.T) {
	t.Run("creates valid LLM attempt", func(t *testing.T) {
		config := LLMAttemptConfig{
			ProviderName: "openai",
			ModelName:    "gpt-4",
			APIKeyEnvVar: "OPENAI_API_KEY",
			MaxTokens:    4096,
			IsPrimary:    true,
		}

		attempt := LLMAttempt{
			Instance: nil, // Would be a real LLM instance in practice
			Config:   config,
			Opts:     nil, // Would contain config options in practice
		}

		assert.Equal(t, config, attempt.Config)
		assert.Nil(t, attempt.Instance)
		assert.Nil(t, attempt.Opts)
	})
}

func TestInferenceService_ThreadSafety(t *testing.T) {
	db := &database.SimpleDomainDB{}
	service, err := NewInferenceService(db)
	require.NoError(t, err)

	t.Run("concurrent access to IsRunning is safe", func(t *testing.T) {
		// This test ensures that concurrent calls to IsRunning don't cause data races
		done := make(chan bool, 10)

		for i := 0; i < 10; i++ {
			go func() {
				service.IsRunning()
				done <- true
			}()
		}

		// Wait for all goroutines to complete
		for i := 0; i < 10; i++ {
			<-done
		}
	})

	t.Run("concurrent access to GetName is safe", func(t *testing.T) {
		done := make(chan bool, 10)

		for i := 0; i < 10; i++ {
			go func() {
				service.GetName()
				done <- true
			}()
		}

		// Wait for all goroutines to complete
		for i := 0; i < 10; i++ {
			<-done
		}
	})
}

func TestInferenceService_ContextManager(t *testing.T) {
	db := &database.SimpleDomainDB{}
	service, err := NewInferenceService(db)
	require.NoError(t, err)

	t.Run("has context manager initialized", func(t *testing.T) {
		assert.NotNil(t, service.contextManager)
	})

	t.Run("context manager has default configuration", func(t *testing.T) {
		// The context manager should be initialized with default settings
		// We can't directly access private fields, but we can verify it exists
		assert.NotNil(t, service.contextManager)
	})
}
