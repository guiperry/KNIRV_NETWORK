package mcphardening

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewInjectionDetector(t *testing.T) {
	id := NewInjectionDetector()
	assert.NotNil(t, id)
	assert.Len(t, id.patterns, len(defaultInjectionPatterns))
}

func TestDetectSafe(t *testing.T) {
	id := NewInjectionDetector()
	tests := []string{
		"Hello, world!",
		"SELECT * FROM users",
		"Just some normal text with DROP in it",
		"",
		"12345",
		"LOWER(table)",
	}

	for _, tt := range tests {
		t.Run(tt, func(t *testing.T) {
			detected, _ := id.Detect(tt)
			assert.False(t, detected, "unexpected detection for: %s", tt)
		})
	}
}

func TestDetectSQLInjection(t *testing.T) {
	id := NewInjectionDetector()
	tests := []struct {
		input string
		label string
	}{
		{"DROP TABLE users", "drop table"},
		{"DELETE FROM sessions", "delete from"},
		{"INSERT INTO logs VALUES ...", "insert into"},
	}

	for _, tt := range tests {
		t.Run(tt.label, func(t *testing.T) {
			detected, pattern := id.Detect(tt.input)
			assert.True(t, detected, "expected detection for: %s", tt.input)
			assert.NotEmpty(t, pattern)
		})
	}
}

func TestDetectXSS(t *testing.T) {
	id := NewInjectionDetector()
	tests := []struct {
		input string
		label string
	}{
		{"<script>alert('xss')</script>", "script tag"},
		{"javascript:void(0)", "javascript protocol"},
		{"onerror=alert(1)", "onerror handler"},
		{"onload=alert(1)", "onload handler"},
	}

	for _, tt := range tests {
		t.Run(tt.label, func(t *testing.T) {
			detected, _ := id.Detect(tt.input)
			assert.True(t, detected, "expected detection for: %s", tt.input)
		})
	}
}

func TestDetectCommandInjection(t *testing.T) {
	id := NewInjectionDetector()
	tests := []struct {
		input string
		label string
	}{
		{"${cat /etc/passwd}", "dollar brace"},
		{"| bash", "pipe bash"},
		{"| sh", "pipe sh"},
		{"rm -rf /", "rm -rf"},
	}

	for _, tt := range tests {
		t.Run(tt.label, func(t *testing.T) {
			detected, _ := id.Detect(tt.input)
			assert.True(t, detected, "expected detection for: %s", tt.input)
		})
	}
}

func TestDetectBacktick(t *testing.T) {
	id := NewInjectionDetector()
	detected, _ := id.Detect("`cat /etc/passwd`")
	assert.True(t, detected, "expected detection for backtick injection")
}

func TestSanitizeSafe(t *testing.T) {
	id := NewInjectionDetector()
	result := id.Sanitize("Hello, world!")
	assert.Equal(t, "Hello, world!", result)
}

func TestSanitizeDangerous(t *testing.T) {
	id := NewInjectionDetector()
	input := "DROP TABLE users"
	result := id.Sanitize(input)
	assert.Equal(t, strings.Repeat("*", len(input)), result)
}

func TestSanitizeEmpty(t *testing.T) {
	id := NewInjectionDetector()
	result := id.Sanitize("")
	assert.Equal(t, "", result)
}

func TestScanArguments(t *testing.T) {
	id := NewInjectionDetector()
	args := map[string]interface{}{
		"name":  "Alice",
		"query": "DROP TABLE users",
		"age":   30,
	}
	flagged := id.ScanArguments(args)
	assert.Contains(t, flagged, "query")
	assert.NotContains(t, flagged, "name")
	assert.NotContains(t, flagged, "age")
}

func TestScanArgumentsSafe(t *testing.T) {
	id := NewInjectionDetector()
	args := map[string]interface{}{
		"name": "Bob",
		"age":  25,
	}
	flagged := id.ScanArguments(args)
	assert.Empty(t, flagged)
}
