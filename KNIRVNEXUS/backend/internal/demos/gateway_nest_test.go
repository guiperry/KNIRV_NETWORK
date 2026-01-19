package demos

import (
	"testing"

	"backend_server/internal/runtime"

	"github.com/stretchr/testify/assert"
)

// TestNewGatewayNestDeployer tests basic initialization
func TestNewGatewayNestDeployer(t *testing.T) {
	t.Parallel()

	// Create a minimal container manager
	manager := runtime.NewUnifiedContainerManager(nil, nil, nil)

	// Create a new gateway nest deployer
	deployer := NewGatewayNestDeployer(manager)
	assert.NotNil(t, deployer, "Gateway nest deployer should not be nil")
	assert.Equal(t, manager, deployer.ContainerManager, "Container manager should be set")
	assert.Nil(t, deployer.Container, "Container should be nil initially")
	assert.Nil(t, deployer.embeddedGateway, "Embedded gateway should be nil initially")
	assert.NotNil(t, deployer.logger, "Logger should be initialized")
}
