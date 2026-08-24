package utils

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetString(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		key      string
		expected string
	}{
		{
			name:     "valid string value",
			input:    map[string]interface{}{"key": "value"},
			key:      "key",
			expected: "value",
		},
		{
			name:     "non-existent key",
			input:    map[string]interface{}{"other": "value"},
			key:      "key",
			expected: "",
		},
		{
			name:     "non-string value",
			input:    map[string]interface{}{"key": 123},
			key:      "key",
			expected: "",
		},
		{
			name:     "empty map",
			input:    map[string]interface{}{},
			key:      "key",
			expected: "",
		},
		{
			name:     "nil map",
			input:    nil,
			key:      "key",
			expected: "",
		},
		{
			name:     "empty string value",
			input:    map[string]interface{}{"key": ""},
			key:      "key",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetString(tt.input, tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetInt(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		key      string
		expected int
	}{
		{
			name:     "valid int value",
			input:    map[string]interface{}{"key": 42},
			key:      "key",
			expected: 42,
		},
		{
			name:     "valid float64 value",
			input:    map[string]interface{}{"key": 42.0},
			key:      "key",
			expected: 42,
		},
		{
			name:     "float64 with decimal",
			input:    map[string]interface{}{"key": 42.7},
			key:      "key",
			expected: 42,
		},
		{
			name:     "negative int value",
			input:    map[string]interface{}{"key": -42},
			key:      "key",
			expected: -42,
		},
		{
			name:     "zero value",
			input:    map[string]interface{}{"key": 0},
			key:      "key",
			expected: 0,
		},
		{
			name:     "non-existent key",
			input:    map[string]interface{}{"other": 42},
			key:      "key",
			expected: 0,
		},
		{
			name:     "non-numeric value",
			input:    map[string]interface{}{"key": "value"},
			key:      "key",
			expected: 0,
		},
		{
			name:     "empty map",
			input:    map[string]interface{}{},
			key:      "key",
			expected: 0,
		},
		{
			name:     "nil map",
			input:    nil,
			key:      "key",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetInt(tt.input, tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetFloatPtr(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		key      string
		expected *float64
	}{
		{
			name:     "valid float64 value",
			input:    map[string]interface{}{"key": 42.5},
			key:      "key",
			expected: func() *float64 { f := 42.5; return &f }(),
		},
		{
			name:     "valid int value",
			input:    map[string]interface{}{"key": 42},
			key:      "key",
			expected: func() *float64 { f := 42.0; return &f }(),
		},
		{
			name:     "zero float value",
			input:    map[string]interface{}{"key": 0.0},
			key:      "key",
			expected: func() *float64 { f := 0.0; return &f }(),
		},
		{
			name:     "negative float value",
			input:    map[string]interface{}{"key": -42.5},
			key:      "key",
			expected: func() *float64 { f := -42.5; return &f }(),
		},
		{
			name:     "non-existent key",
			input:    map[string]interface{}{"other": 42.5},
			key:      "key",
			expected: nil,
		},
		{
			name:     "non-numeric value",
			input:    map[string]interface{}{"key": "value"},
			key:      "key",
			expected: nil,
		},
		{
			name:     "empty map",
			input:    map[string]interface{}{},
			key:      "key",
			expected: nil,
		},
		{
			name:     "nil map",
			input:    nil,
			key:      "key",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetFloatPtr(tt.input, tt.key)
			if tt.expected == nil {
				assert.Nil(t, result)
			} else {
				assert.NotNil(t, result)
				assert.Equal(t, *tt.expected, *result)
			}
		})
	}
}

func TestGetInt64Ptr(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		key      string
		expected *int64
	}{
		{
			name:     "valid int value",
			input:    map[string]interface{}{"key": 42},
			key:      "key",
			expected: func() *int64 { i := int64(42); return &i }(),
		},
		{
			name:     "valid int64 value",
			input:    map[string]interface{}{"key": int64(42)},
			key:      "key",
			expected: func() *int64 { i := int64(42); return &i }(),
		},
		{
			name:     "valid float64 value",
			input:    map[string]interface{}{"key": 42.0},
			key:      "key",
			expected: func() *int64 { i := int64(42); return &i }(),
		},
		{
			name:     "zero value",
			input:    map[string]interface{}{"key": 0},
			key:      "key",
			expected: func() *int64 { i := int64(0); return &i }(),
		},
		{
			name:     "negative value",
			input:    map[string]interface{}{"key": -42},
			key:      "key",
			expected: func() *int64 { i := int64(-42); return &i }(),
		},
		{
			name:     "non-existent key",
			input:    map[string]interface{}{"other": 42},
			key:      "key",
			expected: nil,
		},
		{
			name:     "non-numeric value",
			input:    map[string]interface{}{"key": "value"},
			key:      "key",
			expected: nil,
		},
		{
			name:     "empty map",
			input:    map[string]interface{}{},
			key:      "key",
			expected: nil,
		},
		{
			name:     "nil map",
			input:    nil,
			key:      "key",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetInt64Ptr(tt.input, tt.key)
			if tt.expected == nil {
				assert.Nil(t, result)
			} else {
				assert.NotNil(t, result)
				assert.Equal(t, *tt.expected, *result)
			}
		})
	}
}

func TestReadPortConfig(t *testing.T) {
	// Test with non-existent file (should return defaults)
	t.Run("non-existent file returns defaults", func(t *testing.T) {
		config, err := ReadPortConfig("non-existent-file.config")
		require.NoError(t, err)
		assert.Equal(t, 8081, config.APIPort)
		assert.Equal(t, 8080, config.GUIPort)
	})

	// Test with valid config file
	t.Run("valid config file", func(t *testing.T) {
		// Create temporary config file
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "ports.config")

		configContent := `# Test config file
API_PORT=9001
GUI_PORT=9002
`
		err := os.WriteFile(configPath, []byte(configContent), 0644)
		require.NoError(t, err)

		config, err := ReadPortConfig(configPath)
		require.NoError(t, err)
		assert.Equal(t, 9001, config.APIPort)
		assert.Equal(t, 9002, config.GUIPort)
	})

	// Test with partial config file
	t.Run("partial config file", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "ports.config")

		configContent := `API_PORT=9001
# GUI_PORT is missing, should use default
`
		err := os.WriteFile(configPath, []byte(configContent), 0644)
		require.NoError(t, err)

		config, err := ReadPortConfig(configPath)
		require.NoError(t, err)
		assert.Equal(t, 9001, config.APIPort)
		assert.Equal(t, 8080, config.GUIPort) // Default value
	})

	// Test with empty lines and comments
	t.Run("config with empty lines and comments", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "ports.config")

		configContent := `
# This is a comment
API_PORT=9001

# Another comment
GUI_PORT=9002

# End comment
`
		err := os.WriteFile(configPath, []byte(configContent), 0644)
		require.NoError(t, err)

		config, err := ReadPortConfig(configPath)
		require.NoError(t, err)
		assert.Equal(t, 9001, config.APIPort)
		assert.Equal(t, 9002, config.GUIPort)
	})

	// Test with invalid port values
	t.Run("invalid port values use defaults", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "ports.config")

		configContent := `API_PORT=invalid
GUI_PORT=also_invalid
`
		err := os.WriteFile(configPath, []byte(configContent), 0644)
		require.NoError(t, err)

		config, err := ReadPortConfig(configPath)
		require.NoError(t, err)
		assert.Equal(t, 8081, config.APIPort) // Default value
		assert.Equal(t, 8080, config.GUIPort) // Default value
	})

	// Test with malformed lines
	t.Run("malformed lines are ignored", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "ports.config")

		configContent := `API_PORT=9001
malformed_line_without_equals
GUI_PORT=9002
another=malformed=line=with=multiple=equals
`
		err := os.WriteFile(configPath, []byte(configContent), 0644)
		require.NoError(t, err)

		config, err := ReadPortConfig(configPath)
		require.NoError(t, err)
		assert.Equal(t, 9001, config.APIPort)
		assert.Equal(t, 9002, config.GUIPort)
	})

	// Test with empty file
	t.Run("empty file returns defaults", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "ports.config")

		err := os.WriteFile(configPath, []byte(""), 0644)
		require.NoError(t, err)

		config, err := ReadPortConfig(configPath)
		require.NoError(t, err)
		assert.Equal(t, 8081, config.APIPort)
		assert.Equal(t, 8080, config.GUIPort)
	})

	// Test with whitespace handling
	t.Run("whitespace handling", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "ports.config")

		configContent := `  API_PORT  =  9001
  GUI_PORT  =  9002  `
		err := os.WriteFile(configPath, []byte(configContent), 0644)
		require.NoError(t, err)

		config, err := ReadPortConfig(configPath)
		require.NoError(t, err)
		assert.Equal(t, 9001, config.APIPort)
		assert.Equal(t, 9002, config.GUIPort)
	})
}
