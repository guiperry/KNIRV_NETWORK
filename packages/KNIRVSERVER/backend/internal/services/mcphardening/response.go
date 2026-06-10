package mcphardening

import (
	"strings"
)

type ResponseSanitizer struct {
	blockedPatterns []string
	maxLength       int
}

func NewResponseSanitizer() *ResponseSanitizer {
	return &ResponseSanitizer{
		blockedPatterns: []string{
			"-----BEGIN RSA PRIVATE KEY-----",
			"-----BEGIN EC PRIVATE KEY-----",
			"-----BEGIN OPENSSH PRIVATE KEY-----",
			"ghp_",
			"gho_",
			"ghu_",
			"xoxb-",
			"xoxp-",
			"sk-",
		},
		maxLength: 100000,
	}
}

func (rs *ResponseSanitizer) Sanitize(content string) (string, bool) {
	if len(content) > rs.maxLength {
		content = content[:rs.maxLength]
		return content, true
	}

	original := content
	for _, pat := range rs.blockedPatterns {
		content = strings.ReplaceAll(content, pat, "[REDACTED]")
	}

	wasModified := original != content
	return content, wasModified
}

func (rs *ResponseSanitizer) SetMaxLength(n int) {
	rs.maxLength = n
}

func (rs *ResponseSanitizer) AddBlockedPattern(pattern string) {
	rs.blockedPatterns = append(rs.blockedPatterns, pattern)
}
