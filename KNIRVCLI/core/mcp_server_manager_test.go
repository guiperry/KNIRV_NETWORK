package core

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMCPServerManager(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"status":"ok"}`))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Create temporary directories
	tempDir, err := os.MkdirTemp("", "mcp-server-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	configDir := filepath.Join(tempDir, "config")
	require.NoError(t, os.MkdirAll(configDir, 0755))

	// Create logger
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	// Create API client
	apiClient := NewAPIClient(
		"http://localhost:8080",
		WithLogger(logger),
	)

	// Create file manager
	fileManager, err := NewFileManager(tempDir)
	require.NoError(t, err)

	// Create MCP server manager
	serverManager, err := NewMCPServerManager(apiClient, fileManager, configDir)
	require.NoError(t, err)

	// Create context
	ctx := context.Background()

	t.Run("RegisterServer", func(t *testing.T) {
		// Create server config
		config := MCPServerConfig{
			Name:        "test-server",
			Description: "Test server",
			URL:         server.URL,
			Metadata: map[string]string{
				"region": "us-west",
				"tier":   "production",
			},
		}

		// Register server
		err := serverManager.RegisterServer(ctx, config)
		require.NoError(t, err)

		// Check if server exists
		assert.True(t, serverManager.serverExists("test-server"))

		// Check config file
		configPath := serverManager.getServerConfigPath("test-server")
		_, err = os.Stat(configPath)
		assert.NoError(t, err)
	})

	t.Run("GetServer", func(t *testing.T) {
		// Get server
		config, err := serverManager.GetServer("test-server")
		require.NoError(t, err)

		// Check server config
		assert.Equal(t, "test-server", config.Name)
		assert.Equal(t, "Test server", config.Description)
		assert.Equal(t, server.URL, config.URL)
		assert.Equal(t, "us-west", config.Metadata["region"])
		assert.Equal(t, "production", config.Metadata["tier"])
		assert.False(t, config.CreatedAt.IsZero())
		assert.False(t, config.UpdatedAt.IsZero())
	})

	t.Run("UpdateServer", func(t *testing.T) {
		// Get server
		config, err := serverManager.GetServer("test-server")
		require.NoError(t, err)

		// Update server config
		config.Description = "Updated test server"
		config.Metadata["tier"] = "staging"

		// Update server
		err = serverManager.UpdateServer(ctx, *config)
		require.NoError(t, err)

		// Get updated server
		updatedConfig, err := serverManager.GetServer("test-server")
		require.NoError(t, err)

		// Check updated server config
		assert.Equal(t, "Updated test server", updatedConfig.Description)
		assert.Equal(t, "staging", updatedConfig.Metadata["tier"])
		assert.Equal(t, config.CreatedAt, updatedConfig.CreatedAt)
		assert.True(t, updatedConfig.UpdatedAt.After(config.UpdatedAt))
	})

	t.Run("ListServers", func(t *testing.T) {
		// Register another server
		config := MCPServerConfig{
			Name:        "test-server-2",
			Description: "Test server 2",
			URL:         server.URL,
		}

		err := serverManager.RegisterServer(ctx, config)
		require.NoError(t, err)

		// List servers
		servers, err := serverManager.ListServers()
		require.NoError(t, err)

		// Check servers
		assert.Len(t, servers, 2)
		assert.Contains(t, []string{servers[0].Name, servers[1].Name}, "test-server")
		assert.Contains(t, []string{servers[0].Name, servers[1].Name}, "test-server-2")
	})

	t.Run("TestServerConnection", func(t *testing.T) {
		// Test connection
		err := serverManager.TestServerConnection(ctx, "test-server")
		require.NoError(t, err)
	})

	t.Run("DeleteServer", func(t *testing.T) {
		// Delete server
		err := serverManager.DeleteServer("test-server")
		require.NoError(t, err)

		// Check if server exists
		assert.False(t, serverManager.serverExists("test-server"))

		// Try to get deleted server
		_, err = serverManager.GetServer("test-server")
		assert.Error(t, err)
	})

	t.Run("ValidationErrors", func(t *testing.T) {
		// Empty name
		config := MCPServerConfig{
			Name:        "",
			Description: "Test server",
			URL:         server.URL,
		}
		err := serverManager.RegisterServer(ctx, config)
		assert.Error(t, err)

		// Empty URL
		config = MCPServerConfig{
			Name:        "test-server-3",
			Description: "Test server",
			URL:         "",
		}
		err = serverManager.RegisterServer(ctx, config)
		assert.Error(t, err)

		// Non-existent server
		_, err = serverManager.GetServer("non-existent")
		assert.Error(t, err)

		// Delete non-existent server
		err = serverManager.DeleteServer("non-existent")
		assert.Error(t, err)
	})
}
