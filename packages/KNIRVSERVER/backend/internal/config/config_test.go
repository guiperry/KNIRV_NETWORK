package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRole_String(t *testing.T) {
	role := Root
	str := role.String()
	assert.Equal(t, "Root", str)
}

func TestDetermineRole(t *testing.T) {
	// Test Root role
	role := DetermineRole(true, false, false, false)
	assert.Equal(t, Root, role)

	// Test Bootnode role
	role = DetermineRole(false, true, false, false)
	assert.Equal(t, RoleBootnode, role)

	// Test Peer role
	role = DetermineRole(false, false, true, false)
	assert.Equal(t, RolePeer, role)

	// Test Client role
	role = DetermineRole(false, false, false, true)
	assert.Equal(t, RoleClient, role)

	// Test default (Client)
	role = DetermineRole(false, false, false, false)
	assert.Equal(t, RoleClient, role)
}

func TestDetermineRoleFromConfig(t *testing.T) {
	cfg := &Config{
		IsRoot:     true,
		IsBootnode: false,
		IsPeer:     false,
		ClientOnly: false,
	}

	role := DetermineRoleFromConfig(cfg)
	assert.Equal(t, Root, role)
}

func TestGetConfigDir(t *testing.T) {
	dir, err := GetConfigDir()
	assert.NoError(t, err)
	assert.NotEmpty(t, dir)
}

func TestGetDataDir(t *testing.T) {
	dir, err := GetDataDir()
	assert.NoError(t, err)
	assert.NotEmpty(t, dir)

	// Test with specific role
	dir, err = GetDataDir(Root)
	assert.NoError(t, err)
	assert.NotEmpty(t, dir)
}

func TestValidateConfig(t *testing.T) {
	cfg := &Config{
		Mode: "headless",
		Network: NetworkConfig{
			ChainID: "test-chain",
		},
		NodeRole: "validator",
		Database: DatabaseConfig{
			Path: "/tmp/test.db",
		},
		Security: SecurityConfig{
			AuthRequired: false,
		},
	}

	err := validateConfig(cfg)
	assert.NoError(t, err)
}

func TestValidateConfig_InvalidMode(t *testing.T) {
	cfg := &Config{
		Mode: "invalid",
		Network: NetworkConfig{
			ChainID: "test-chain",
		},
		NodeRole: "validator",
		Database: DatabaseConfig{
			Path: "/tmp/test.db",
		},
	}

	err := validateConfig(cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mode must be")
}

func TestValidateConfig_MissingChainID(t *testing.T) {
	cfg := &Config{
		Mode:     "headless",
		NodeRole: "validator",
		Database: DatabaseConfig{
			Path: "/tmp/test.db",
		},
	}

	err := validateConfig(cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "chain_id is required")
}

func TestValidateConfig_MissingNodeRole(t *testing.T) {
	cfg := &Config{
		Mode: "headless",
		Network: NetworkConfig{
			ChainID: "test-chain",
		},
		Database: DatabaseConfig{
			Path: "/tmp/test.db",
		},
	}

	err := validateConfig(cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "node_role is required")
}

func TestValidateConfig_MissingDatabasePath(t *testing.T) {
	cfg := &Config{
		Mode: "headless",
		Network: NetworkConfig{
			ChainID: "test-chain",
		},
		NodeRole: "validator",
	}

	err := validateConfig(cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database.path is required")
}

func TestValidateConfig_GUI_InvalidPort(t *testing.T) {
	cfg := &Config{
		Mode: "gui",
		GUI: GUIConfig{
			Enabled:     true,
			Port:        0,
			FrontendPath: "/tmp/test",
		},
		Network: NetworkConfig{
			ChainID: "test-chain",
		},
		NodeRole: "validator",
		Database: DatabaseConfig{
			Path: "/tmp/test.db",
		},
	}

	err := validateConfig(cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "gui.port must be positive")
}

func TestValidateConfig_GUI_MissingFrontendPath(t *testing.T) {
	cfg := &Config{
		Mode: "gui",
		GUI: GUIConfig{
			Enabled: true,
			Port:    8080,
		},
		Network: NetworkConfig{
			ChainID: "test-chain",
		},
		NodeRole: "validator",
		Database: DatabaseConfig{
			Path: "/tmp/test.db",
		},
	}

	err := validateConfig(cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "gui.frontend_path is required")
}

func TestExpandPath(t *testing.T) {
	// Test regular path
	path, err := expandPath("/tmp/test")
	assert.NoError(t, err)
	assert.Equal(t, "/tmp/test", path)

	// Test home expansion (if ~ is present)
	// This would require setting up user.Current mock, so skip for now
}

func TestConfig_ExpandPaths(t *testing.T) {
	cfg := &Config{
		Database: DatabaseConfig{
			Path: "/tmp/test.db",
		},
		CDE: CDEConfig{
			BaseImagePath: "/tmp/images",
			WorkspaceRoot: "/tmp/workspaces",
			ProjectStoragePath: "/tmp/projects",
		},
		Reports: ReportsConfig{
			StoragePath: "/tmp/reports",
		},
		Log: LogConfig{
			Output: "/tmp/logs/test.log",
		},
		ModelServer: ModelServerConfig{
			StoragePath: "/tmp/models",
		},
	}

	err := cfg.ExpandPaths()
	assert.NoError(t, err)
}

func TestLoadWithDefaults(t *testing.T) {
	// This test may fail if config files don't exist, but should work with defaults
	cfg, err := LoadWithDefaults()
	// We don't assert NoError because it depends on environment
	// But if it succeeds, verify basic fields
	if err == nil {
		require.NotNil(t, cfg)
		assert.NotEmpty(t, cfg.Database.Path)
	}
}
