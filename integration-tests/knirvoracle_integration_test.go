package integration_tests

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKNIRVOracledDaemon(t *testing.T) {
	// Skip if services are not running
	suite := NewIntegrationTestSuite()

	// Check health of knirv-oracled
	suite.CheckServiceHealth(t, "knirv-oracled", suite.knirvOracledRPCURL, "/status")

	t.Run("OracledStatus", func(t *testing.T) {
		status, err := suite.GetOracledStatus(t)
		require.NoError(t, err)
		assert.NotNil(t, status)

		// Assert basic properties of the oracle daemon
		assert.NotEmpty(t, status.ChainID)
		assert.NotEmpty(t, status.NetworkID)
		assert.Equal(t, "KNIRV Network Token", status.Token.Name)
		assert.Equal(t, "NRN", status.Token.Symbol)
		assert.True(t, status.Consensus.LatestBlockHeight >= 0) // Block height can be 0 initially
		assert.True(t, status.Consensus.ValidatorCount >= 0)     // Validator count can be 0 initially
		assert.True(t, status.IBC.Enabled)

		t.Logf("KNIRV-Oracled Daemon Status: ChainID=%s, NetworkID=%s, BlockHeight=%d",
			status.ChainID, status.NetworkID, status.Consensus.LatestBlockHeight)
	})

	// Add more specific tests for token transfers, governance, etc., here later.
	// For now, basic status check is sufficient for initial validation.
}