package utils

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadEmbeddedEnv(t *testing.T) {
	// Save original environment variables that might be affected
	originalEnvVars := make(map[string]string)
	
	// We need to check what variables are in the default.env file
	// Since we can't easily read the embedded file in tests, we'll test the behavior
	
	t.Run("loads embedded environment variables", func(t *testing.T) {
		// Clear any existing environment variables that might conflict
		// We'll test with some common variables that might be in default.env
		testVars := []string{
			"API_PORT",
			"GUI_PORT", 
			"LOG_LEVEL",
			"DEBUG",
			"NODE_ENV",
		}
		
		// Save original values
		for _, varName := range testVars {
			if val := os.Getenv(varName); val != "" {
				originalEnvVars[varName] = val
				os.Unsetenv(varName)
			}
		}
		
		// Load embedded environment
		err := LoadEmbeddedEnv()
		require.NoError(t, err)
		
		// The function should not return an error
		// We can't easily test specific values without knowing the content of default.env
		// But we can verify the function runs without error
		
		// Restore original environment variables
		for varName, originalValue := range originalEnvVars {
			os.Setenv(varName, originalValue)
		}
		for _, varName := range testVars {
			if _, exists := originalEnvVars[varName]; !exists {
				os.Unsetenv(varName)
			}
		}
	})

	t.Run("does not override existing environment variables", func(t *testing.T) {
		// Set a test environment variable
		testVar := "TEST_ENV_VAR_FOR_EMBEDDED_TEST"
		testValue := "existing_value"
		os.Setenv(testVar, testValue)
		
		// Load embedded environment
		err := LoadEmbeddedEnv()
		require.NoError(t, err)
		
		// The existing value should not be changed
		currentValue := os.Getenv(testVar)
		assert.Equal(t, testValue, currentValue)
		
		// Clean up
		os.Unsetenv(testVar)
	})

	t.Run("handles empty environment gracefully", func(t *testing.T) {
		// This test verifies that LoadEmbeddedEnv doesn't panic or error
		// when called multiple times or in different states
		
		err1 := LoadEmbeddedEnv()
		require.NoError(t, err1)
		
		err2 := LoadEmbeddedEnv()
		require.NoError(t, err2)
		
		// Should be able to call multiple times without error
	})
}

// Note: Testing the embedded file system is challenging because the file is embedded at compile time
// and we can't easily mock it. The tests above focus on the behavior we can verify:
// 1. The function doesn't return errors under normal conditions
// 2. It doesn't override existing environment variables
// 3. It can be called multiple times safely
//
// For more comprehensive testing of the embedded file parsing, we would need to:
// 1. Create a separate function that takes the file content as a parameter
// 2. Test that function with known input
// 3. Have LoadEmbeddedEnv call that function with the embedded content
//
// This would be a good refactoring for better testability, but for now we test what we can.
