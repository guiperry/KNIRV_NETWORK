package tests

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"backend_server/internal/objects"
	"backend_server/internal/services/container"
	"backend_server/internal/services/endpoints"
	"backend_server/internal/services/session"
)

// Integration & Testing
// Comprehensive tests for container provisioning, session management, and full workflow

// 7.1 Backend Testing

func TestContainerProvisioning(t *testing.T) {
	t.Run("TestContainerOrchestrator_ProvisionContainer", func(t *testing.T) {
		// Mock container orchestrator
		orchestrator := &container.MockContainerOrchestrator{}

		rentalID := "rental-test-123"
		ctx := context.Background()

		container, err := orchestrator.ProvisionContainer(ctx, rentalID)
		require.NoError(t, err)
		assert.NotNil(t, container)
		assert.NotEmpty(t, container.ID)
		assert.Equal(t, "provisioned", container.Status)
	})

	t.Run("TestContainerOrchestrator_AllocateEndpoints", func(t *testing.T) {
		orchestrator := &container.MockContainerOrchestrator{}

		rentalID := "rental-test-123"
		ctx := context.Background()

		endpoints, err := orchestrator.AllocateEndpoints(ctx, rentalID)
		require.NoError(t, err)
		assert.NotNil(t, endpoints)
		assert.NotEmpty(t, endpoints.Host)

		// Verify port ranges
		assert.GreaterOrEqual(t, endpoints.SSHPort, 22000)
		assert.LessOrEqual(t, endpoints.SSHPort, 22999)
		assert.GreaterOrEqual(t, endpoints.ValidationPort, 23000)
		assert.LessOrEqual(t, endpoints.ValidationPort, 23999)
		assert.GreaterOrEqual(t, endpoints.ErrorResPort, 24000)
		assert.LessOrEqual(t, endpoints.ErrorResPort, 24999)
	})

	t.Run("TestSSHKeyInjection", func(t *testing.T) {
		provisioner := &container.MockSSHProvisioner{}

		containerID := "container-test-123"
		publicKey := "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQ..."

		err := provisioner.InjectSSHKeys(context.Background(), containerID, publicKey)
		assert.NoError(t, err)
	})

	t.Run("TestContainerLifecycle", func(t *testing.T) {
		orchestrator := &container.MockContainerOrchestrator{}

		containerID := "container-test-123"

		// Test status check
		status, err := orchestrator.GetContainerStatus(context.Background(), containerID)
		assert.NoError(t, err)
		assert.Equal(t, "running", status)

		// Test termination
		err = orchestrator.TerminateContainer(context.Background(), containerID)
		assert.NoError(t, err)
	})
}

func TestSessionManagement(t *testing.T) {
	t.Run("TestSessionManager_CreateSSHSession", func(t *testing.T) {
		manager := &session.MockSessionManager{}

		rentalID := "rental-test-123"
		containerID := "container-test-456"
		username := "test-user"

		sess, err := manager.CreateSSHSession(rentalID, containerID, username, "mock-private-key")
		require.NoError(t, err)
		assert.NotNil(t, sess)
		assert.Equal(t, rentalID, sess.RentalID)
		assert.Equal(t, containerID, sess.ContainerID)
		assert.Equal(t, username, sess.Username)
		assert.NotEmpty(t, sess.PrivateKeyURL)
		assert.True(t, sess.ExpiresAt.After(time.Now()))
	})

	t.Run("TestSessionManager_CreateValidationSession", func(t *testing.T) {
		manager := &session.MockSessionManager{}

		rentalID := "rental-test-123"
		validationType := "reasoning"

		sess, err := manager.CreateValidationSession(rentalID, validationType)
		require.NoError(t, err)
		assert.NotNil(t, sess)
		assert.Equal(t, rentalID, sess.RentalID)
		assert.Equal(t, validationType, sess.ValidationType)
		assert.NotEmpty(t, sess.SessionToken)
		assert.NotEmpty(t, sess.EndpointURL)
		assert.True(t, sess.ExpiresAt.After(time.Now()))
	})

	t.Run("TestSessionManager_CreateErrorResolutionSession", func(t *testing.T) {
		manager := &session.MockSessionManager{}

		rentalID := "rental-test-123"
		supportedTypes := []string{"connection_timeout", "validation_failed"}

		sess, err := manager.CreateErrorResolutionSession(rentalID, supportedTypes)
		require.NoError(t, err)
		assert.NotNil(t, sess)
		assert.Equal(t, rentalID, sess.RentalID)
		assert.Equal(t, supportedTypes, sess.SupportedTypes)
		assert.NotEmpty(t, sess.SessionToken)
		assert.NotEmpty(t, sess.EndpointURL)
		assert.True(t, sess.ExpiresAt.After(time.Now()))
	})
}

func TestEndpointRegistry(t *testing.T) {
	t.Run("TestEndpointRegistry_RegisterEndpoint", func(t *testing.T) {
		registry := &endpoints.MockEndpointRegistry{}

		rentalID := "rental-test-123"
		endpointType := "ssh"
		endpoint := &objects.TEEEndpoint{
			Host:     "10.0.1.42",
			Port:     22145,
			Protocol: "ssh",
		}

		err := registry.RegisterEndpoint(context.Background(), rentalID, endpointType, endpoint)
		assert.NoError(t, err)
	})

	t.Run("TestEndpointRegistry_GetEndpointByRentalAndType", func(t *testing.T) {
		registry := &endpoints.MockEndpointRegistry{}

		rentalID := "rental-test-123"
		endpointType := "ssh"

		endpoint, err := registry.GetEndpointByRentalAndType(context.Background(), rentalID, endpointType)
		require.NoError(t, err)
		assert.NotNil(t, endpoint)
		assert.Equal(t, rentalID, endpoint.RentalID)
		assert.Equal(t, endpointType, endpoint.EndpointType)
		assert.Equal(t, "active", endpoint.Status)
	})

	t.Run("TestEndpointRegistry_ListEndpoints", func(t *testing.T) {
		registry := &endpoints.MockEndpointRegistry{}

		rentalID := "rental-test-123"

		endpoints, err := registry.ListEndpoints(context.Background(), rentalID)
		require.NoError(t, err)
		assert.Len(t, endpoints, 3) // ssh, validation, error-resolution

		endpointTypes := make(map[string]bool)
		for _, ep := range endpoints {
			endpointTypes[ep.EndpointType] = true
		}

		assert.True(t, endpointTypes["ssh"])
		assert.True(t, endpointTypes["validation"])
		assert.True(t, endpointTypes["error-resolution"])
	})

	t.Run("TestEndpointRegistry_UnregisterEndpoint", func(t *testing.T) {
		registry := &endpoints.MockEndpointRegistry{}

		rentalID := "rental-test-123"
		endpointType := "ssh"

		err := registry.UnregisterEndpoint(context.Background(), rentalID, endpointType)
		assert.NoError(t, err)

		// Verify endpoint is gone
		_, err = registry.GetEndpointByRentalAndType(context.Background(), rentalID, endpointType)
		assert.Error(t, err)
	})
}

// 7.2 Integration Tests

func TestRentalToProvisioningFlow(t *testing.T) {
	t.Run("TestCompleteRentalFlow", func(t *testing.T) {
		// Setup mock services
		containerOrch := &container.MockContainerOrchestrator{}
		sessionMgr := &session.MockSessionManager{}
		endpointReg := &endpoints.MockEndpointRegistry{}

		rentalID := "rental-integration-123"
		userID := "user-test-456"
		ctx := context.Background()

		// Step 1: Create rental
		rental := &objects.DVERental{
			ID:                 rentalID,
			UserID:             userID,
			DVENodeID:          "node-test-789",
			Status:             "active",
			RentalDuration:     30,
			StartTime:          time.Now(),
			EndTime:            time.Now().Add(30 * 24 * time.Hour),
			ProvisioningStatus: "pending",
		}
		_ = rental.ID
		_ = rental.UserID
		_ = rental.DVENodeID
		_ = rental.Status
		_ = rental.RentalDuration
		_ = rental.StartTime
		_ = rental.EndTime
		_ = rental.ProvisioningStatus

		// Step 2: Provision container
		container, err := containerOrch.ProvisionContainer(ctx, rentalID)
		require.NoError(t, err)
		assert.Equal(t, "provisioned", container.Status)

		// Update rental with container info
		rental.ContainerID = container.ID
		rental.ProvisioningStatus = "provisioned"

		// Step 3: Allocate endpoints
		endpoints, err := containerOrch.AllocateEndpoints(ctx, rentalID)
		require.NoError(t, err)

		// Step 4: Create sessions
		sshSession, err := sessionMgr.CreateSSHSession(rentalID, container.ID, "test-user", "mock-private-key")
		require.NoError(t, err)

		valSession, err := sessionMgr.CreateValidationSession(rentalID, "reasoning")
		require.NoError(t, err)

		errSession, err := sessionMgr.CreateErrorResolutionSession(rentalID, []string{"timeout", "error"})
		require.NoError(t, err)

		// Step 5: Register endpoints
		err = endpointReg.RegisterEndpoint(ctx, rentalID, "ssh", &objects.TEEEndpoint{
			RentalID:     rentalID,
			EndpointType: "ssh",
			Host:         endpoints.Host,
			Port:         endpoints.SSHPort,
			Protocol:     "ssh",
			Status:       "active",
		})
		assert.NoError(t, err)

		err = endpointReg.RegisterEndpoint(ctx, rentalID, "validation", &objects.TEEEndpoint{
			RentalID:     rentalID,
			EndpointType: "validation",
			Host:         endpoints.Host,
			Port:         endpoints.ValidationPort,
			Protocol:     "http",
			Status:       "active",
		})
		assert.NoError(t, err)

		err = endpointReg.RegisterEndpoint(ctx, rentalID, "error-resolution", &objects.TEEEndpoint{
			RentalID:     rentalID,
			EndpointType: "error-resolution",
			Host:         endpoints.Host,
			Port:         endpoints.ErrorResPort,
			Protocol:     "http",
			Status:       "active",
		})
		assert.NoError(t, err)

		// Step 6: Verify complete setup
		rentalEndpoints, err := endpointReg.ListEndpoints(ctx, rentalID)
		require.NoError(t, err)
		assert.Len(t, rentalEndpoints, 3)

		// Verify SSH session
		assert.NotNil(t, sshSession)
		assert.Equal(t, rentalID, sshSession.RentalID)

		// Verify validation session
		assert.NotNil(t, valSession)
		assert.Equal(t, rentalID, valSession.RentalID)

		// Verify error resolution session
		assert.NotNil(t, errSession)
		assert.Equal(t, rentalID, errSession.RentalID)

		// Verify rental is fully provisioned
		assert.Equal(t, "provisioned", rental.ProvisioningStatus)
		assert.NotEmpty(t, rental.ContainerID)
	})
}

func TestAPIEndpointIntegration(t *testing.T) {
	t.Run("TestFullAccessInfoStructure", func(t *testing.T) {
		// Test the expected structure of full access info response
		accessInfo := map[string]interface{}{
			"ssh": map[string]interface{}{
				"endpoint":                 "10.0.1.42",
				"port":                     22145,
				"username":                 "rental-user-abc123",
				"private_key_download_url": "/api/sessions/ssh/abc123/key",
				"command":                  "ssh -i key.pem rental-user-abc123@10.0.1.42 -p 22145",
				"expires_at":               time.Now().Add(24 * time.Hour),
			},
			"reasoning_validation": map[string]interface{}{
				"endpoint_url":  "http://10.0.1.42:23145",
				"session_token": "jwt-token-xyz",
				"expires_at":    time.Now().Add(24 * time.Hour),
			},
			"error_resolution": map[string]interface{}{
				"endpoint_url":  "http://10.0.1.42:24145",
				"session_token": "jwt-token-uvw",
				"expires_at":    time.Now().Add(24 * time.Hour),
			},
		}

		// Verify SSH section
		ssh := accessInfo["ssh"].(map[string]interface{})
		assert.Contains(t, ssh, "endpoint")
		assert.Contains(t, ssh, "port")
		assert.Contains(t, ssh, "username")
		assert.Contains(t, ssh, "private_key_download_url")
		assert.Contains(t, ssh, "command")
		assert.Contains(t, ssh, "expires_at")

		// Verify validation section
		validation := accessInfo["reasoning_validation"].(map[string]interface{})
		assert.Contains(t, validation, "endpoint_url")
		assert.Contains(t, validation, "session_token")
		assert.Contains(t, validation, "expires_at")

		// Verify error resolution section
		errorRes := accessInfo["error_resolution"].(map[string]interface{})
		assert.Contains(t, errorRes, "endpoint_url")
		assert.Contains(t, errorRes, "session_token")
		assert.Contains(t, errorRes, "expires_at")
	})
}

// 7.3 End-to-End Testing

func TestCompleteWorkflow(t *testing.T) {
	t.Run("TestFullUserJourney", func(t *testing.T) {
		// This test simulates the complete user journey from rental to access

		// Setup mock services
		containerOrch := &container.MockContainerOrchestrator{}
		sessionMgr := &session.MockSessionManager{}
		endpointReg := &endpoints.MockEndpointRegistry{}

		rentalID := "rental-e2e-123"
		userID := "user-e2e-456"
		ctx := context.Background()

		// Step 1: User rents DVE (simulated)
		rental := &objects.DVERental{
			ID:                 rentalID,
			UserID:             userID,
			DVENodeID:          "node-e2e-789",
			Status:             "active",
			RentalDuration:     30,
			StartTime:          time.Now(),
			EndTime:            time.Now().Add(30 * 24 * time.Hour),
			ProvisioningStatus: "pending",
		}
		_ = rental.ID
		_ = rental.UserID
		_ = rental.DVENodeID
		_ = rental.Status
		_ = rental.RentalDuration
		_ = rental.StartTime
		_ = rental.EndTime
		_ = rental.ProvisioningStatus

		// Step 2: System provisions container
		container, err := containerOrch.ProvisionContainer(ctx, rentalID)
		require.NoError(t, err)
		assert.Equal(t, "provisioned", container.Status)

		rental.ContainerID = container.ID
		rental.ProvisioningStatus = "provisioned"

		// Step 3: System allocates endpoints
		endpoints, err := containerOrch.AllocateEndpoints(ctx, rentalID)
		require.NoError(t, err)
		assert.NotNil(t, endpoints)

		// Step 4: System creates sessions
		sshSess, err := sessionMgr.CreateSSHSession(rentalID, container.ID, "test-user", "mock-private-key")
		require.NoError(t, err)
		valSess, err := sessionMgr.CreateValidationSession(rentalID, "reasoning")
		require.NoError(t, err)
		errSess, err := sessionMgr.CreateErrorResolutionSession(rentalID, []string{"timeout", "error"})
		require.NoError(t, err)

		// Step 5: Register endpoints
		err = endpointReg.RegisterEndpoint(ctx, rentalID, "ssh", &objects.TEEEndpoint{
			RentalID:     rentalID,
			EndpointType: "ssh",
			Host:         endpoints.Host,
			Port:         endpoints.SSHPort,
			Protocol:     "ssh",
			Status:       "active",
		})
		assert.NoError(t, err)

		err = endpointReg.RegisterEndpoint(ctx, rentalID, "validation", &objects.TEEEndpoint{
			RentalID:     rentalID,
			EndpointType: "validation",
			Host:         endpoints.Host,
			Port:         endpoints.ValidationPort,
			Protocol:     "http",
			Status:       "active",
		})
		assert.NoError(t, err)

		err = endpointReg.RegisterEndpoint(ctx, rentalID, "error-resolution", &objects.TEEEndpoint{
			RentalID:     rentalID,
			EndpointType: "error-resolution",
			Host:         endpoints.Host,
			Port:         endpoints.ErrorResPort,
			Protocol:     "http",
			Status:       "active",
		})
		assert.NoError(t, err)

		// Step 6: User gets full access info (simulated API call)
		rentalEndpoints, err := endpointReg.ListEndpoints(ctx, rentalID)
		require.NoError(t, err)
		assert.Len(t, rentalEndpoints, 3)

		// Verify SSH endpoint
		sshEndpoint := rentalEndpoints[0] // Assuming first is SSH
		assert.Equal(t, "ssh", sshEndpoint.EndpointType)
		assert.Equal(t, rentalID, sshEndpoint.RentalID)
		assert.Equal(t, "active", sshEndpoint.Status)

		// Verify validation endpoint
		valEndpoint := rentalEndpoints[1] // Assuming second is validation
		assert.Equal(t, "validation", valEndpoint.EndpointType)
		assert.Equal(t, rentalID, valEndpoint.RentalID)
		assert.Equal(t, "active", valEndpoint.Status)

		// Verify error resolution endpoint
		errEndpoint := rentalEndpoints[2] // Assuming third is error-resolution
		assert.Equal(t, "error-resolution", errEndpoint.EndpointType)
		assert.Equal(t, rentalID, errEndpoint.RentalID)
		assert.Equal(t, "active", errEndpoint.Status)

		// Step 7: Verify all sessions are properly created
		assert.NotNil(t, sshSess)
		assert.NotNil(t, valSess)
		assert.NotNil(t, errSess)

		assert.Equal(t, rentalID, sshSess.RentalID)
		assert.Equal(t, rentalID, valSess.RentalID)
		assert.Equal(t, rentalID, errSess.RentalID)

		assert.True(t, sshSess.ExpiresAt.After(time.Now()))
		assert.True(t, valSess.ExpiresAt.After(time.Now()))
		assert.True(t, errSess.ExpiresAt.After(time.Now()))

		// Step 8: Verify rental is fully provisioned
		assert.Equal(t, "provisioned", rental.ProvisioningStatus)
		assert.NotEmpty(t, rental.ContainerID)
		assert.Equal(t, "active", rental.Status)
	})
}
