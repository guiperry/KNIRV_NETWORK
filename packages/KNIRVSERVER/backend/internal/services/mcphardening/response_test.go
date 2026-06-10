package mcphardening

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewResponseSanitizer(t *testing.T) {
	rs := NewResponseSanitizer()
	assert.NotNil(t, rs)
}

func TestSanitizeNoSecrets(t *testing.T) {
	rs := NewResponseSanitizer()
	content := "Hello, this is a normal response with no secrets."
	result, modified := rs.Sanitize(content)
	assert.Equal(t, content, result)
	assert.False(t, modified)
}

func TestSanitizePrivateKey(t *testing.T) {
	rs := NewResponseSanitizer()
	content := "Some text with -----BEGIN RSA PRIVATE KEY----- something -----END RSA PRIVATE KEY-----"
	result, modified := rs.Sanitize(content)
	assert.True(t, modified)
	assert.Contains(t, result, "[REDACTED]")
	assert.NotContains(t, result, "BEGIN RSA PRIVATE KEY")
}

func TestSanitizeGitHubToken(t *testing.T) {
	rs := NewResponseSanitizer()
	content := "token: ghp_abcdefghijklmnop"
	result, modified := rs.Sanitize(content)
	assert.True(t, modified)
	assert.Contains(t, result, "[REDACTED]")
}

func TestSanitizeSlackToken(t *testing.T) {
	rs := NewResponseSanitizer()
	content := "slack: xoxb-123456789"
	result, modified := rs.Sanitize(content)
	assert.True(t, modified)
	assert.Contains(t, result, "[REDACTED]")
}

func TestSanitizeOpenAIKey(t *testing.T) {
	rs := NewResponseSanitizer()
	content := "api key: sk-proj-abc123"
	result, modified := rs.Sanitize(content)
	assert.True(t, modified)
	assert.Contains(t, result, "[REDACTED]")
}

func TestSanitizeMaxLength(t *testing.T) {
	rs := NewResponseSanitizer()
	rs.SetMaxLength(10)
	content := "This is a long response that should be truncated"
	result, modified := rs.Sanitize(content)
	assert.True(t, modified)
	assert.Equal(t, 10, len(result))
	assert.Equal(t, "This is a ", result)
}

func TestSanitizeEmptyContent(t *testing.T) {
	rs := NewResponseSanitizer()
	result, modified := rs.Sanitize("")
	assert.False(t, modified)
	assert.Empty(t, result)
}

func TestAddBlockedPattern(t *testing.T) {
	rs := NewResponseSanitizer()
	rs.AddBlockedPattern("secret-token-xyz")
	content := "my secret-token-xyz is here"
	result, modified := rs.Sanitize(content)
	assert.True(t, modified)
	assert.Contains(t, result, "[REDACTED]")
	assert.NotContains(t, result, "secret-token-xyz")
}
