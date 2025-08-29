package test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

// getProjectRoot returns the project root directory
func getProjectRoot(t *testing.T) string {
	_, currentFile, _, ok := runtime.Caller(0)
	require.True(t, ok, "Could not determine current file path")
	
	// Go up from test directory to project root
	return filepath.Dir(filepath.Dir(currentFile))
}
