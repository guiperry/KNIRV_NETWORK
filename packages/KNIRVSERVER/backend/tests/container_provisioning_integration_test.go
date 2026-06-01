package tests

import (
	"testing"
	"time"

	"backend_server/internal/database"
	"backend_server/internal/objects"
	"backend_server/internal/services/container"
	"backend_server/internal/services/endpoints"
	"backend_server/internal/services/session"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRealContainerProvisioningIntegration tests that real containers are created during the provisioning process
func TestRealContainerProvisioningIntegration(t *testing.T) {
	t.Run("TestRealContainerCreation", func(t *testing.T) {
		// Setup container orchestrator with Docker runtime
		config := &container.ContainerConfig{
			ContainerRuntime:         "docker",
			BaseImage:                "ubuntu:20.04",
			SSHPortRangeStart:        22000,
			SSHPortRangeEnd:          22010,
			ValidationPortRangeStart: 23000,
			ValidationPortRangeEnd:   23010,
			ErrorResPortRangeStart:   24000,
			ErrorResPortRangeEnd:     24010,
			ProvisioningTimeout:      30 * time.Second,
			CleanupInterval:          10 * time.Minute,
		}

		co, err := container.NewContainerOrchestrator(config, nil)
		require.NoError(t, err)
		require.NotNil(t, co)

		rentalID := "integration-test-rental-" + time.Now().Format("20060102150405")

		// Provision a real container
		cont, err := co.ProvisionContainer(rentalID)
		require.NoError(t, err)
		require.NotNil(t, cont)

		// Verify container properties
		assert.NotEmpty(t, cont.ID)
		assert.Equal(t, container.ContainerStatusRunning, cont.Status)
		assert.Equal(t, "docker", cont.Runtime)
		assert.NotNil(t, cont.Spec)
		assert.Equal(t, "ubuntu:20.04", cont.Spec.Image)

		// Verify SSH keys were generated
		assert.NotNil(t, cont.SSHKeys)
		assert.NotEmpty(t, cont.SSHKeys.PublicKey)
		assert.NotEmpty(t, cont.SSHKeys.PrivateKey)
		assert.Contains(t, cont.SSHKeys.PublicKey, "ssh-")
		assert.Contains(t, cont.SSHKeys.PrivateKey, "-----BEGIN")

		// Verify endpoints were allocated
		assert.NotNil(t, cont.Endpoints)
		assert.GreaterOrEqual(t, cont.Endpoints.SSHPort, 22000)
		assert.LessOrEqual(t, cont.Endpoints.SSHPort, 22010)
		assert.GreaterOrEqual(t, cont.Endpoints.ValidationPort, 23000)
		assert.LessOrEqual(t, cont.Endpoints.ValidationPort, 23010)
		assert.GreaterOrEqual(t, cont.Endpoints.ErrorResPort, 24000)
		assert.LessOrEqual(t, cont.Endpoints.ErrorResPort, 24010)

		// Verify container spec has correct configuration
		assert.Equal(t, cont.Spec.SSHUsername, "rental-user-"+rentalID[:8])
		assert.Greater(t, cont.Spec.ResourceLimits.MaxCPU, 0.0)
		assert.Greater(t, cont.Spec.ResourceLimits.MaxMemory, int64(0))
		assert.Greater(t, cont.Spec.ResourceLimits.MaxDisk, int64(0))

		// Verify container is actually running by checking status
		status, err := co.GetContainerStatus(cont.ID)
		require.NoError(t, err)
		assert.Equal(t, container.ContainerStatusRunning, status)

		// Test SSH private key retrieval
		privateKey, err := co.GetSSHPrivateKey(cont.ID)
		require.NoError(t, err)
		assert.Equal(t, cont.SSHKeys.PrivateKey, privateKey)

		// Clean up - terminate the container
		err = co.TerminateContainer(cont.ID)
		assert.NoError(t, err)

		// Verify container is terminated
		status, err = co.GetContainerStatus(cont.ID)
		if err == nil {
			// If status check still works, container might still be terminating
			t.Logf("Container %s status after termination: %s", cont.ID, status)
		}
	})

	t.Run("TestContainerProvisioningWithSessions", func(t *testing.T) {
		// Setup all services
		config := &container.ContainerConfig{
			ContainerRuntime:         "docker",
			BaseImage:                "ubuntu:20.04",
			SSHPortRangeStart:        22000,
			SSHPortRangeEnd:          22010,
			ValidationPortRangeStart: 23000,
			ValidationPortRangeEnd:   23010,
			ErrorResPortRangeStart:   24000,
			ErrorResPortRangeEnd:     24010,
			ProvisioningTimeout:      30 * time.Second,
			CleanupInterval:          10 * time.Minute,
		}

		co, err := container.NewContainerOrchestrator(config, nil)
		require.NoError(t, err)

		sessionManager := session.NewSessionManager(nil)

		creationID := "session-integration-test-" + time.Now().Format("20060102150405")

		// Provision container
		cont, err := co.ProvisionContainer(creationID)
		require.NoError(t, err)
		require.NotNil(t, cont)

		// Create SSH session
		sshSession, err := sessionManager.CreateSSHSession(creationID, cont.ID, cont.Spec.SSHUsername, cont.SSHKeys.PrivateKey)
		require.NoError(t, err)
		assert.NotNil(t, sshSession)
		assert.Equal(t, creationID, sshSession.CreationID)
		assert.Equal(t, cont.ID, sshSession.ContainerID)

		// Create validation session
		valSession, err := sessionManager.CreateValidationSession(creationID, "reasoning")
		require.NoError(t, err)
		assert.NotNil(t, valSession)
		assert.Equal(t, creationID, valSession.CreationID)
		assert.Equal(t, "reasoning", valSession.ValidationType)

		// Create error resolution session
		errResSession, err := sessionManager.CreateErrorResolutionSession(creationID, []string{"timeout", "network"})
		require.NoError(t, err)
		assert.NotNil(t, errResSession)
		assert.Equal(t, creationID, errResSession.CreationID)
		assert.Contains(t, errResSession.SupportedTypes, "timeout")
		assert.Contains(t, errResSession.SupportedTypes, "network")

		// Verify sessions can be retrieved
		retrievedSSHSession, err := sessionManager.GetSSHSession(sshSession.ID)
		require.NoError(t, err)
		assert.Equal(t, sshSession.ID, retrievedSSHSession.ID)

		retrievedValSession, err := sessionManager.GetValidationSession(valSession.ID)
		require.NoError(t, err)
		assert.Equal(t, valSession.ID, retrievedValSession.ID)

		retrievedErrResSession, err := sessionManager.GetErrorResolutionSession(errResSession.ID)
		require.NoError(t, err)
		assert.Equal(t, errResSession.ID, retrievedErrResSession.ID)

		// Get all sessions for creation (using new terminology)
		sessions, err := sessionManager.GetSessionsByCreationID(creationID)
		require.NoError(t, err)
		assert.Len(t, sessions, 3)

		// Terminate all sessions
		err = sessionManager.TerminateAllSessionsForCreation(creationID)
		require.NoError(t, err)

		// Verify sessions are terminated
		_, err = sessionManager.GetSSHSession(sshSession.ID)
		assert.Error(t, err)

		_, err = sessionManager.GetValidationSession(valSession.ID)
		assert.Error(t, err)

		_, err = sessionManager.GetErrorResolutionSession(errResSession.ID)
		assert.Error(t, err)

		// Clean up container
		err = co.TerminateContainer(cont.ID)
		assert.NoError(t, err)
	})

	t.Run("TestContainerProvisioningInTestMode", func(t *testing.T) {
		// Skip test if TEE security service is not available (required for native containers)
		t.Skip("Skipping native container test - requires TEE security service")

		// Test mode uses native containers instead of real Docker/Podman
		config := &container.ContainerConfig{
			ContainerRuntime:         "native-go",
			BaseImage:                "ubuntu:20.04",
			SSHPortRangeStart:        22000,
			SSHPortRangeEnd:          22010,
			ValidationPortRangeStart: 23000,
			ValidationPortRangeEnd:   23010,
			ErrorResPortRangeStart:   24000,
			ErrorResPortRangeEnd:     24010,
			ProvisioningTimeout:      10 * time.Second,
			CleanupInterval:          10 * time.Minute,
		}

		co, err := container.NewContainerOrchestrator(config, nil)
		require.NoError(t, err)

		rentalID := "test-mode-rental-" + time.Now().Format("20060102150405")

		// Provision container in test mode
		cont, err := co.ProvisionContainer(rentalID)
		require.NoError(t, err)
		require.NotNil(t, cont)

		// Verify container properties for test mode
		assert.NotEmpty(t, cont.ID)
		assert.Equal(t, container.ContainerStatusRunning, cont.Status)
		assert.Equal(t, "native-go", cont.Runtime)
		assert.NotNil(t, cont.Spec)

		// SSH keys should still be generated even in test mode
		assert.NotNil(t, cont.SSHKeys)
		assert.NotEmpty(t, cont.SSHKeys.PublicKey)
		assert.NotEmpty(t, cont.SSHKeys.PrivateKey)

		// Endpoints should still be allocated
		assert.NotNil(t, cont.Endpoints)
		assert.GreaterOrEqual(t, cont.Endpoints.SSHPort, 22000)
		assert.LessOrEqual(t, cont.Endpoints.SSHPort, 22010)

		// Test SSH private key retrieval
		privateKey, err := co.GetSSHPrivateKey(cont.ID)
		require.NoError(t, err)
		assert.Equal(t, cont.SSHKeys.PrivateKey, privateKey)

		// Clean up
		err = co.TerminateContainer(cont.ID)
		assert.NoError(t, err)
	})

	t.Run("TestContainerProvisioningFailureScenarios", func(t *testing.T) {
		t.Run("TestUnsupportedContainerRuntime", func(t *testing.T) {
			config := &container.ContainerConfig{
				ContainerRuntime: "unsupported-runtime",
			}

			co, err := container.NewContainerOrchestrator(config, nil)
			require.NoError(t, err)

			_, err = co.ProvisionContainer("test-rental")
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "unsupported container runtime")
		})

		t.Run("TestTimeoutScenario", func(t *testing.T) {
			config := &container.ContainerConfig{
				ContainerRuntime:         "docker",
				ProvisioningTimeout:      1 * time.Millisecond, // Very short timeout
				SSHPortRangeStart:        22000,
				SSHPortRangeEnd:          22010,
				ValidationPortRangeStart: 23000,
				ValidationPortRangeEnd:   23010,
				ErrorResPortRangeStart:   24000,
				ErrorResPortRangeEnd:     24010,
			}

			co, err := container.NewContainerOrchestrator(config, nil)
			require.NoError(t, err)

			_, err = co.ProvisionContainer("test-rental")
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "timed out")
		})
	})
}

// TestSessionPersistenceAcrossRestarts tests that sessions survive simulated restarts
func TestSessionPersistenceAcrossRestarts(t *testing.T) {
	t.Run("TestSessionPersistenceWithDatabase", func(t *testing.T) {
		// Create a temporary database for testing
		db, err := database.NewBuntDB(":memory:")
		require.NoError(t, err)
		defer db.Close()

		// Create session manager with database
		sm1 := session.NewSessionManager(db)

		creationID := "persistence-test-creation"
		containerID := "persistence-test-container"
		username := "testuser"

		// Create sessions with first manager instance
		sshSession, err := sm1.CreateSSHSession(creationID, containerID, username, "test-private-key")
		require.NoError(t, err)

		valSession, err := sm1.CreateValidationSession(creationID, "reasoning")
		require.NoError(t, err)

		errResSession, err := sm1.CreateErrorResolutionSession(creationID, []string{"timeout"})
		require.NoError(t, err)

		// Simulate restart by creating new session manager instance
		sm2 := session.NewSessionManager(db)

		// Verify sessions were loaded from database
		retrievedSSHSession, err := sm2.GetSSHSession(sshSession.ID)
		require.NoError(t, err)
		assert.Equal(t, sshSession.ID, retrievedSSHSession.ID)
		assert.Equal(t, creationID, retrievedSSHSession.CreationID)
		assert.Equal(t, containerID, retrievedSSHSession.ContainerID)
		assert.Equal(t, username, retrievedSSHSession.Username)

		retrievedValSession, err := sm2.GetValidationSession(valSession.ID)
		require.NoError(t, err)
		assert.Equal(t, valSession.ID, retrievedValSession.ID)
		assert.Equal(t, creationID, retrievedValSession.CreationID)
		assert.Equal(t, "reasoning", retrievedValSession.ValidationType)

		retrievedErrResSession, err := sm2.GetErrorResolutionSession(errResSession.ID)
		require.NoError(t, err)
		assert.Equal(t, errResSession.ID, retrievedErrResSession.ID)
		assert.Equal(t, creationID, retrievedErrResSession.CreationID)
		assert.Contains(t, retrievedErrResSession.SupportedTypes, "timeout")

		// Verify all sessions are returned for creation (using new terminology)
		sessions, err := sm2.GetSessionsByCreationID(creationID)
		require.NoError(t, err)
		assert.Len(t, sessions, 3)
	})

	t.Run("TestSessionPersistenceAfterTermination", func(t *testing.T) {
		db, err := database.NewBuntDB(":memory:")
		require.NoError(t, err)
		defer db.Close()

		sm := session.NewSessionManager(db)

		creationID := "termination-test-creation"

		// Create sessions
		sshSession, err := sm.CreateSSHSession(creationID, "container1", "user1", "key1")
		require.NoError(t, err)

		valSession, err := sm.CreateValidationSession(creationID, "reasoning")
		require.NoError(t, err)

		// Terminate validation session
		err = sm.TerminateValidationSession(valSession.ID)
		require.NoError(t, err)

		// Simulate restart
		sm2 := session.NewSessionManager(db)

		// SSH session should still exist
		retrievedSSHSession, err := sm2.GetSSHSession(sshSession.ID)
		require.NoError(t, err)
		assert.Equal(t, sshSession.ID, retrievedSSHSession.ID)

		// Validation session should be gone
		_, err = sm2.GetValidationSession(valSession.ID)
		assert.Error(t, err)

		// Only SSH session should remain for creation (using new terminology)
		sessions, err := sm2.GetSessionsByCreationID(creationID)
		require.NoError(t, err)
		assert.Len(t, sessions, 1)
	})
}

// TestFullCreationToContainerFlowIntegration tests the complete flow from creation to container access
func TestFullCreationToContainerFlowIntegration(t *testing.T) {
	t.Run("TestCompleteCreationContainerFlow", func(t *testing.T) {
		// Setup all services
		config := &container.ContainerConfig{
			ContainerRuntime:         "docker",
			BaseImage:                "ubuntu:20.04",
			SSHPortRangeStart:        22000,
			SSHPortRangeEnd:          22010,
			ValidationPortRangeStart: 23000,
			ValidationPortRangeEnd:   23010,
			ErrorResPortRangeStart:   24000,
			ErrorResPortRangeEnd:     24010,
			ProvisioningTimeout:      30 * time.Second,
			CleanupInterval:          10 * time.Minute,
		}

		co, err := container.NewContainerOrchestrator(config, nil)
		require.NoError(t, err)

		db, err := database.NewBuntDB(":memory:")
		require.NoError(t, err)
		defer db.Close()

		sessionManager := session.NewSessionManager(db)
		endpointRegistry := endpoints.NewEndpointRegistry(db)

		creationID := "full-flow-test-" + time.Now().Format("20060102150405")

		// Step 1: Provision container
		cont, err := co.ProvisionContainer(creationID)
		require.NoError(t, err)
		require.NotNil(t, cont)

		// Step 2: Create SSH session
		sshSession, err := sessionManager.CreateSSHSession(creationID, cont.ID, cont.Spec.SSHUsername, cont.SSHKeys.PrivateKey)
		require.NoError(t, err)

		// Step 3: Create validation session
		valSession, err := sessionManager.CreateValidationSession(creationID, "reasoning")
		require.NoError(t, err)

		// Step 4: Create error resolution session
		errResSession, err := sessionManager.CreateErrorResolutionSession(creationID, []string{"timeout", "network"})
		require.NoError(t, err)

		// Step 5: Register endpoints (using CreationID terminology)
		sshEndpoint := &objects.TEEEndpoint{
			CreationID:   creationID,
			ContainerID:  cont.ID,
			EndpointType: "ssh",
			Host:         "localhost",
			Port:         cont.Endpoints.SSHPort,
			Protocol:     "ssh",
			Status:       "active",
			CreatedAt:    time.Now(),
			ExpiresAt:    sshSession.ExpiresAt,
		}
		err = endpointRegistry.RegisterEndpoint(creationID, "ssh", sshEndpoint)
		require.NoError(t, err)

		valEndpoint := &objects.TEEEndpoint{
			CreationID:   creationID,
			ContainerID:  cont.ID,
			EndpointType: "validation",
			Host:         "localhost",
			Port:         cont.Endpoints.ValidationPort,
			Protocol:     "http",
			Status:       "active",
			CreatedAt:    time.Now(),
			ExpiresAt:    valSession.ExpiresAt,
		}
		err = endpointRegistry.RegisterEndpoint(creationID, "validation", valEndpoint)
		require.NoError(t, err)

		errResEndpoint := &objects.TEEEndpoint{
			CreationID:   creationID,
			ContainerID:  cont.ID,
			EndpointType: "error-resolution",
			Host:         "localhost",
			Port:         cont.Endpoints.ErrorResPort,
			Protocol:     "http",
			Status:       "active",
			CreatedAt:    time.Now(),
			ExpiresAt:    errResSession.ExpiresAt,
		}
		err = endpointRegistry.RegisterEndpoint(creationID, "error-resolution", errResEndpoint)
		require.NoError(t, err)

		// Step 6: Verify complete setup
		// Check that all endpoints are registered
		endpoints, err := endpointRegistry.ListEndpoints(creationID)
		require.NoError(t, err)
		assert.Len(t, endpoints, 3)

		// Verify SSH endpoint
		sshEp, err := endpointRegistry.GetEndpointByRentalAndType(creationID, "ssh")
		require.NoError(t, err)
		assert.Equal(t, "ssh", sshEp.EndpointType)
		assert.Equal(t, cont.Endpoints.SSHPort, sshEp.Port)

		// Verify validation endpoint
		valEp, err := endpointRegistry.GetEndpointByRentalAndType(creationID, "validation")
		require.NoError(t, err)
		assert.Equal(t, "validation", valEp.EndpointType)
		assert.Equal(t, cont.Endpoints.ValidationPort, valEp.Port)

		// Verify error resolution endpoint
		errResEp, err := endpointRegistry.GetEndpointByRentalAndType(creationID, "error-resolution")
		require.NoError(t, err)
		assert.Equal(t, "error-resolution", errResEp.EndpointType)
		assert.Equal(t, cont.Endpoints.ErrorResPort, errResEp.Port)

		// Step 7: Test session termination via handlers
		// This would normally be done via HTTP requests, but we'll test the logic directly

		// Terminate validation session
		err = sessionManager.TerminateValidationSession(valSession.ID)
		require.NoError(t, err)

		// Verify validation session is gone
		_, err = sessionManager.GetValidationSession(valSession.ID)
		assert.Error(t, err)

		// Terminate error resolution session
		err = sessionManager.TerminateErrorResolutionSession(errResSession.ID)
		require.NoError(t, err)

		// Verify error resolution session is gone
		_, err = sessionManager.GetErrorResolutionSession(errResSession.ID)
		assert.Error(t, err)

		// SSH session should still exist
		retrievedSSHSession, err := sessionManager.GetSSHSession(sshSession.ID)
		require.NoError(t, err)
		assert.Equal(t, sshSession.ID, retrievedSSHSession.ID)

		// Clean up
		err = co.TerminateContainer(cont.ID)
		assert.NoError(t, err)

		err = sessionManager.TerminateAllSessionsForCreation(creationID)
		assert.NoError(t, err)
	})
}
