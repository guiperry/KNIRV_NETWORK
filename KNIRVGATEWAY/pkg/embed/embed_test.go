package embed

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
	assert.Equal(t, "gateway_nest", cfg.Mode, "Default mode should be gateway_nest")
	assert.Equal(t, "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", cfg.OracleOwnerKey, "Default oracle owner key should be test key")
	assert.False(t, cfg.AutoOpenBrowser, "Default auto-open browser should be false")
	assert.True(t, cfg.EnableOracle, "Default oracle enable should be true")
	assert.True(t, cfg.EnableModelServer, "Default model server enable should be true")
}

func TestNewGateway(t *testing.T) {
	t.Parallel()

	logger, err := zap.NewProduction()
	require.NoError(t, err)

	// Test with default config
	gateway, err := NewGateway(nil, logger)
	require.NoError(t, err)
	assert.NotNil(t, gateway, "Gateway should be created")

	// Test with custom config
	cfg := DefaultConfig()
	cfg.Port = 9090
	cfg.ChainID = "mainnet"

	gateway, err = NewGateway(cfg, logger)
	require.NoError(t, err)
	assert.NotNil(t, gateway, "Gateway should be created")
	assert.Equal(t, 9090, gateway.GetPort(), "Port should match custom config")
	assert.Equal(t, "mainnet", gateway.GetChainID(), "Chain ID should match custom config")
}

func TestGatewayIsRunning(t *testing.T) {
	t.Parallel()

	logger, err := zap.NewProduction()
	require.NoError(t, err)

	gateway, err := NewGateway(nil, logger)
	require.NoError(t, err)

	assert.True(t, gateway.IsRunning(), "Gateway should be running before starting")
}

func TestGatewayGetters(t *testing.T) {
	t.Parallel()

	logger, err := zap.NewProduction()
	require.NoError(t, err)

	cfg := DefaultConfig()
	cfg.Port = 9090
	cfg.ChainID = "mainnet"

	gateway, err := NewGateway(cfg, logger)
	require.NoError(t, err)

	assert.Equal(t, 9090, gateway.GetPort(), "GetPort should return config port")
	assert.Equal(t, "mainnet", gateway.GetChainID(), "GetChainID should return config chain ID")
	assert.NotNil(t, gateway.GetRuntime(), "GetRuntime should return runtime instance")
	assert.Nil(t, gateway.GetServer(), "GetServer should return nil before server starts")
}

func TestGatewayConfigCustomization(t *testing.T) {
	t.Parallel()

	logger, err := zap.NewProduction()
	require.NoError(t, err)

	customConfig := &GatewayConfig{
		Port:              1234,
		ChainID:           "custom-chain",
		Mode:              "custom-mode",
		OracleOwnerKey:    "custom-private-key-1234",
		AutoOpenBrowser:   true,
		EnableOracle:      false,
		EnableModelServer: false,
	}

	gateway, err := NewGateway(customConfig, logger)
	require.NoError(t, err)

	assert.Equal(t, 1234, gateway.GetPort(), "Port should be custom value")
	assert.Equal(t, "custom-chain", gateway.GetChainID(), "Chain ID should be custom value")
	assert.True(t, gateway.IsRunning(), "Gateway should be initialized as running")
}
