package embedded

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestDefaultConfig(t *testing.T) {
	t.Parallel()

	cfg := DefaultConfig()

	assert.Equal(t, 8080, cfg.Port, "Default port should be 8080")
	assert.Equal(t, "testnet", cfg.ChainID, "Default chain ID should be testnet")
	assert.Equal(t, "oracle_nest", cfg.Mode, "Default mode should be oracle_nest")
	assert.Equal(t, "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", cfg.OracleOwnerKey, "Default oracle owner key should be test key")
	assert.False(t, cfg.AutoOpenBrowser, "Default auto-open browser should be false")
	assert.True(t, cfg.EnableOracle, "Default oracle enable should be true")
	assert.True(t, cfg.EnableModelServer, "Default model server enable should be true")
}

func TestNewOracle(t *testing.T) {
	t.Parallel()

	logger, err := zap.NewProduction()
	require.NoError(t, err)

	// Test with default config
	oracle, err := NewOracle(nil, logger)
	require.NoError(t, err)
	assert.NotNil(t, oracle, "Oracle should be created")

	// Test with custom config
	cfg := DefaultConfig()
	cfg.Port = 9090
	cfg.ChainID = "mainnet"

	oracle, err = NewOracle(cfg, logger)
	require.NoError(t, err)
	assert.NotNil(t, oracle, "Oracle should be created")
	assert.Equal(t, 9090, oracle.GetPort(), "Port should match custom config")
	assert.Equal(t, "mainnet", oracle.GetChainID(), "Chain ID should match custom config")
}

func TestOracleIsRunning(t *testing.T) {
	t.Parallel()

	logger, err := zap.NewProduction()
	require.NoError(t, err)

	oracle, err := NewOracle(nil, logger)
	require.NoError(t, err)

	assert.True(t, oracle.IsRunning(), "Oracle should be running before starting")
}

func TestOracleGetters(t *testing.T) {
	t.Parallel()

	logger, err := zap.NewProduction()
	require.NoError(t, err)

	cfg := DefaultConfig()
	cfg.Port = 9090
	cfg.ChainID = "mainnet"

	oracle, err := NewOracle(cfg, logger)
	require.NoError(t, err)

	assert.Equal(t, 9090, oracle.GetPort(), "GetPort should return config port")
	assert.Equal(t, "mainnet", oracle.GetChainID(), "GetChainID should return config chain ID")
	assert.NotNil(t, oracle.GetRuntime(), "GetRuntime should return runtime instance")
	assert.Nil(t, oracle.GetServer(), "GetServer should return nil before server starts")
}

func TestOracleConfigCustomization(t *testing.T) {
	t.Parallel()

	logger, err := zap.NewProduction()
	require.NoError(t, err)

	customConfig := &OracleConfig{
		Port:              1234,
		ChainID:           "custom-chain",
		Mode:              "custom-mode",
		OracleOwnerKey:    "custom-private-key-1234",
		AutoOpenBrowser:   true,
		EnableOracle:      false,
		EnableModelServer: false,
	}

	oracle, err := NewOracle(customConfig, logger)
	require.NoError(t, err)

	assert.Equal(t, 1234, oracle.GetPort(), "Port should be custom value")
	assert.Equal(t, "custom-chain", oracle.GetChainID(), "Chain ID should be custom value")
	assert.True(t, oracle.IsRunning(), "Oracle should be initialized as running")
}
