package inferencer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewAnthropicProvider(t *testing.T) {
	provider := NewAnthropicProvider("test-key", "claude-3-sonnet-20240229", map[string]string{})
	assert.NotNil(t, provider)
	// client is a private field, cannot assert directly
}

func TestGetProviderName(t *testing.T) {
	provider := NewAnthropicProvider("test-key", "claude-3-sonnet-20240229", map[string]string{})
	assert.Equal(t, "anthropic", provider.Name())
}

func TestIsAvailable(t *testing.T) {
	// AnthropicProvider doesn't have IsAvailable method
	// This test should be removed or modified
	t.Skip("AnthropicProvider doesn't have IsAvailable method")
}
