package inference_engine

import (
	"encoding/json"
	"testing"

	"github.com/guiperry/gollm_cerebras/config"
	gollm_types "github.com/guiperry/gollm_cerebras/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAnthropicProvider(t *testing.T) {
	tests := []struct {
		name          string
		apiKey        string
		model         string
		extraHeaders  map[string]string
		expectedModel string
	}{
		{
			name:          "basic provider creation",
			apiKey:        "test-key",
			model:         "claude-3-opus-20240229",
			extraHeaders:  nil,
			expectedModel: "claude-3-opus-20240229",
		},
		{
			name:          "empty model uses default",
			apiKey:        "test-key",
			model:         "",
			extraHeaders:  nil,
			expectedModel: "claude-3-sonnet-20240229",
		},
		{
			name:  "with extra headers",
			apiKey: "test-key",
			model:  "claude-3-haiku-20240307",
			extraHeaders: map[string]string{
				"X-Custom": "value",
			},
			expectedModel: "claude-3-haiku-20240307",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := NewAnthropicProvider(tt.apiKey, tt.model, tt.extraHeaders)

			assert.NotNil(t, provider)
			anthropicProvider, ok := provider.(*AnthropicProvider)
			require.True(t, ok)

			assert.Equal(t, tt.apiKey, anthropicProvider.apiKey)
			assert.Equal(t, tt.expectedModel, anthropicProvider.model)
			assert.Equal(t, 4096, anthropicProvider.maxTokens)

			if tt.extraHeaders != nil {
				for k, v := range tt.extraHeaders {
					assert.Equal(t, v, anthropicProvider.extraHeaders[k])
				}
			}
		})
	}
}

func TestAnthropicProvider_Name(t *testing.T) {
	provider := NewAnthropicProvider("test-key", "test-model", nil)
	assert.Equal(t, "anthropic", provider.Name())
}

func TestAnthropicProvider_Endpoint(t *testing.T) {
	provider := NewAnthropicProvider("test-key", "test-model", nil)
	assert.Equal(t, "https://api.anthropic.com/v1/messages", provider.Endpoint())
}

func TestAnthropicProvider_Headers(t *testing.T) {
	tests := []struct {
		name         string
		apiKey       string
		extraHeaders map[string]string
		expected     map[string]string
	}{
		{
			name:   "basic headers",
			apiKey: "test-key",
			extraHeaders: nil,
			expected: map[string]string{
				"Content-Type":      "application/json",
				"x-api-key":         "test-key",
				"anthropic-version": "2023-06-01",
				"User-Agent":        "zookeeper/1.0",
			},
		},
		{
			name:   "with extra headers",
			apiKey: "test-key",
			extraHeaders: map[string]string{
				"X-Custom": "value",
			},
			expected: map[string]string{
				"Content-Type":      "application/json",
				"x-api-key":         "test-key",
				"anthropic-version": "2023-06-01",
				"User-Agent":        "zookeeper/1.0",
				"X-Custom":          "value",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := NewAnthropicProvider(tt.apiKey, "test-model", tt.extraHeaders)
			headers := provider.Headers()

			for k, v := range tt.expected {
				assert.Equal(t, v, headers[k])
			}
		})
	}
}

func TestAnthropicProvider_PrepareRequest(t *testing.T) {
	provider := NewAnthropicProvider("test-key", "claude-3-sonnet-20240229", nil)

	tests := []struct {
		name     string
		prompt   string
		options  map[string]interface{}
		expected AnthropicRequest
	}{
		{
			name:   "basic request",
			prompt: "Hello world",
			options: nil,
			expected: AnthropicRequest{
				Model: "claude-3-sonnet-20240229",
				Messages: []AnthropicMessage{
					{Role: "user", Content: "Hello world"},
				},
				MaxTokens: 4096,
				Stream:    false,
			},
		},
		{
			name:   "with custom options",
			prompt: "Test prompt",
			options: map[string]interface{}{
				"model":      "claude-3-opus-20240229",
				"max_tokens": 2048,
				"stream":     true,
			},
			expected: AnthropicRequest{
				Model: "claude-3-opus-20240229",
				Messages: []AnthropicMessage{
					{Role: "user", Content: "Test prompt"},
				},
				MaxTokens: 2048,
				Stream:    true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := provider.PrepareRequest(tt.prompt, tt.options)
			require.NoError(t, err)

			var req AnthropicRequest
			err = json.Unmarshal(data, &req)
			require.NoError(t, err)

			assert.Equal(t, tt.expected.Model, req.Model)
			assert.Equal(t, tt.expected.MaxTokens, req.MaxTokens)
			assert.Equal(t, tt.expected.Stream, req.Stream)
			assert.Equal(t, len(tt.expected.Messages), len(req.Messages))
			for i, msg := range tt.expected.Messages {
				assert.Equal(t, msg.Role, req.Messages[i].Role)
				assert.Equal(t, msg.Content, req.Messages[i].Content)
			}
		})
	}
}

func TestAnthropicProvider_PrepareRequestWithMessages(t *testing.T) {
	provider := NewAnthropicProvider("test-key", "claude-3-sonnet-20240229", nil)

	tests := []struct {
		name     string
		messages []gollm_types.MemoryMessage
		options  map[string]interface{}
		expected AnthropicRequest
	}{
		{
			name: "user and assistant messages",
			messages: []gollm_types.MemoryMessage{
				{Role: "user", Content: "Hello"},
				{Role: "assistant", Content: "Hi there"},
				{Role: "user", Content: "How are you?"},
			},
			options: nil,
			expected: AnthropicRequest{
				Model: "claude-3-sonnet-20240229",
				Messages: []AnthropicMessage{
					{Role: "user", Content: "Hello"},
					{Role: "assistant", Content: "Hi there"},
					{Role: "user", Content: "How are you?"},
				},
				MaxTokens: 4096,
				Stream:    false,
			},
		},
		{
			name: "with system message",
			messages: []gollm_types.MemoryMessage{
				{Role: "system", Content: "You are a helpful assistant"},
				{Role: "user", Content: "Hello"},
			},
			options: nil,
			expected: AnthropicRequest{
				Model:    "claude-3-sonnet-20240229",
				Messages: []AnthropicMessage{
					{Role: "user", Content: "Hello"},
				},
				MaxTokens: 4096,
				System:    "You are a helpful assistant",
				Stream:    false,
			},
		},
		{
			name: "case insensitive roles",
			messages: []gollm_types.MemoryMessage{
				{Role: "USER", Content: "Hello"},
				{Role: "AI", Content: "Hi"},
			},
			options: nil,
			expected: AnthropicRequest{
				Model: "claude-3-sonnet-20240229",
				Messages: []AnthropicMessage{
					{Role: "user", Content: "Hello"},
					{Role: "assistant", Content: "Hi"},
				},
				MaxTokens: 4096,
				Stream:    false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := provider.PrepareRequestWithMessages(tt.messages, tt.options)
			require.NoError(t, err)

			var req AnthropicRequest
			err = json.Unmarshal(data, &req)
			require.NoError(t, err)

			assert.Equal(t, tt.expected.Model, req.Model)
			assert.Equal(t, tt.expected.MaxTokens, req.MaxTokens)
			assert.Equal(t, tt.expected.System, req.System)
			assert.Equal(t, tt.expected.Stream, req.Stream)
			assert.Equal(t, len(tt.expected.Messages), len(req.Messages))
			for i, msg := range tt.expected.Messages {
				assert.Equal(t, msg.Role, req.Messages[i].Role)
				assert.Equal(t, msg.Content, req.Messages[i].Content)
			}
		})
	}
}

func TestAnthropicProvider_ParseResponse(t *testing.T) {
	provider := NewAnthropicProvider("test-key", "test-model", nil)

	tests := []struct {
		name        string
		response    string
		expected    string
		expectError bool
	}{
		{
			name: "valid response",
			response: `{
				"id": "msg_123",
				"type": "message",
				"role": "assistant",
				"content": [{"type": "text", "text": "Hello world"}],
				"stop_reason": "end_turn"
			}`,
			expected:    "Hello world",
			expectError: false,
		},
		{
			name:        "invalid json",
			response:    `{"invalid": json}`,
			expected:    "",
			expectError: true,
		},
		{
			name: "no text content",
			response: `{
				"id": "msg_123",
				"type": "message",
				"role": "assistant",
				"content": [],
				"stop_reason": "end_turn"
			}`,
			expected:    "",
			expectError: true,
		},
		{
			name: "non-text content",
			response: `{
				"id": "msg_123",
				"type": "message",
				"role": "assistant",
				"content": [{"type": "image", "data": "base64data"}],
				"stop_reason": "end_turn"
			}`,
			expected:    "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := provider.ParseResponse([]byte(tt.response))

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestAnthropicProvider_SetDefaultOptions(t *testing.T) {
	provider := NewAnthropicProvider("", "", nil)
	anthropicProvider := provider.(*AnthropicProvider)

	// Test with nil config
	provider.SetDefaultOptions(nil)
	assert.Equal(t, "", anthropicProvider.apiKey)
	assert.Equal(t, "claude-3-sonnet-20240229", anthropicProvider.model)

	// Test with config
	cfg := &config.Config{
		APIKeys: map[string]string{
			"anthropic": "config-api-key",
		},
		Model:     "claude-3-opus-20240229",
		MaxTokens: 8192,
	}

	provider.SetDefaultOptions(cfg)
	assert.Equal(t, "config-api-key", anthropicProvider.apiKey)
	assert.Equal(t, "claude-3-opus-20240229", anthropicProvider.model)
	assert.Equal(t, 8192, anthropicProvider.maxTokens)
}

func TestAnthropicProvider_SetOption(t *testing.T) {
	provider := NewAnthropicProvider("test-key", "claude-3-sonnet-20240229", nil)
	anthropicProvider := provider.(*AnthropicProvider)

	// Test setting model
	provider.SetOption("model", "claude-3-haiku-20240307")
	assert.Equal(t, "claude-3-haiku-20240307", anthropicProvider.model)

	// Test setting max_tokens
	provider.SetOption("max_tokens", 2048)
	assert.Equal(t, 2048, anthropicProvider.maxTokens)

	// Test invalid key (should not change anything)
	provider.SetOption("invalid_key", "value")
	assert.Equal(t, "claude-3-haiku-20240307", anthropicProvider.model)

	// Test invalid value type (should not change anything)
	provider.SetOption("model", 123)
	assert.Equal(t, "claude-3-haiku-20240307", anthropicProvider.model)
}

func TestAnthropicProvider_UnsupportedMethods(t *testing.T) {
	provider := NewAnthropicProvider("test-key", "test-model", nil)

	// Test PrepareRequestWithSchema
	_, err := provider.PrepareRequestWithSchema("prompt", nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not support JSON schema")

	// Test SupportsJSONSchema
	assert.False(t, provider.SupportsJSONSchema())

	// Test SupportsStreaming
	assert.False(t, provider.SupportsStreaming())

	// Test PrepareStreamRequest
	_, err = provider.PrepareStreamRequest("prompt", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "streaming not implemented")

	// Test ParseStreamResponse
	_, err = provider.ParseStreamResponse([]byte("chunk"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "streaming not implemented")

	// Test HandleFunctionCalls (should return input unchanged)
	input := []byte("test data")
	result, err := provider.HandleFunctionCalls(input)
	assert.NoError(t, err)
	assert.Equal(t, input, result)
}

// Benchmark tests
func BenchmarkAnthropicProvider_PrepareRequest(b *testing.B) {
	provider := NewAnthropicProvider("test-key", "claude-3-sonnet-20240229", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := provider.PrepareRequest("Test prompt", nil)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkAnthropicProvider_ParseResponse(b *testing.B) {
	provider := NewAnthropicProvider("test-key", "test-model", nil)
	response := []byte(`{
		"id": "msg_123",
		"type": "message",
		"role": "assistant",
		"content": [{"type": "text", "text": "Hello world"}],
		"stop_reason": "end_turn"
	}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := provider.ParseResponse(response)
		if err != nil {
			b.Fatal(err)
		}
	}
}