package test

import (
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"

	"KNIRVENGINE/desktop-client/agentify"
	"KNIRVENGINE/desktop-client/api"
)

func TestGaps(t *testing.T) {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: No .env file found")
	}

	// Set demo mode for testing
	os.Setenv("AGENTIC_ENGINE_DEMO_MODE", "true")

	fmt.Println("=== Testing Implementation Gaps ===")

	// Test CustomTEE
	testCustomTEE()

	// Test Enhanced Connection Security
	testEnhancedConnectionSecurity(t)

	// Test Terminal Manager
	testTerminalManager(t)

	fmt.Println("\n=== All tests completed ===")
}

func testCustomTEE() {
	fmt.Println("\n--- Testing CustomTEE Implementation ---")

	// Create a new CustomTEE
	config := agentify.TEEConfig{
		WorkingDir: "/tmp/custom-tee-test-" + uuid.New().String(),
		Env: map[string]string{
			"TEST_VAR": "test_value",
		},
		ResourceLimits: agentify.ResourceLimits{
			MemoryMB:             512,
			CPUCores:             1.0,
			DiskSpaceMB:          1024,
			MaxProcesses:         10,
			MemoryAlertThreshold: 0.8,
			CPUAlertThreshold:    0.9,
			DiskAlertThreshold:   0.7,
		},
		SecurityPolicy: agentify.SecurityPolicy{
			AllowNetworkAccess:   false,
			AllowFileSystemWrite: true,
			AllowedCommands:      []string{"ls", "echo", "cat"},
			BlockedCommands:      []string{"rm", "wget"},
			MaxExecutionTime:     30 * time.Second,
		},
	}

	tee := agentify.NewCustomTEE(config)

	// Start the TEE
	fmt.Println("Starting CustomTEE...")
	if err := tee.Start(); err != nil {
		fmt.Printf("Error starting CustomTEE: %v\n", err)
		return
	}

	// Get TEE info
	info := tee.GetInfo()
	fmt.Println("CustomTEE Info:", info)

	// Execute a command
	fmt.Println("Executing 'echo hello world' command...")
	stdout, stderr, exitCode, err := tee.Execute("echo", []string{"hello", "world"})
	if err != nil {
		fmt.Printf("Error executing command: %v\n", err)
	} else {
		fmt.Printf("Command executed successfully:\n")
		fmt.Printf("  Stdout: %s\n", stdout)
		fmt.Printf("  Stderr: %s\n", stderr)
		fmt.Printf("  Exit code: %d\n", exitCode)
	}

	// Try to execute a blocked command
	fmt.Println("Trying to execute blocked command 'wget'...")
	_, _, _, err = tee.Execute("wget", []string{"https://example.com"})
	if err != nil {
		fmt.Printf("Expected error executing blocked command: %v\n", err)
	} else {
		fmt.Println("Error: Blocked command was allowed to execute")
	}

	// Get resource usage
	fmt.Println("Getting resource usage...")
	usage, err := tee.GetResourceUsage()
	if err != nil {
		fmt.Printf("Error getting resource usage: %v\n", err)
	} else {
		fmt.Printf("Resource usage:\n")
		fmt.Printf("  Memory: %.2f MB\n", usage.MemoryMB)
		fmt.Printf("  CPU: %.2f%%\n", usage.CPUPercent)
		fmt.Printf("  Disk: %.2f MB\n", usage.DiskUsageMB)
		fmt.Printf("  Network: %.2f MB\n", usage.NetworkMB)
		fmt.Printf("  Processes: %d\n", usage.Processes)
	}

	// Check resource alerts
	fmt.Println("Checking resource alerts...")
	alerts, err := tee.CheckResourceAlerts()
	if err != nil {
		fmt.Printf("Error checking resource alerts: %v\n", err)
	} else {
		fmt.Printf("Resource alerts: %d\n", len(alerts))
		for i, alert := range alerts {
			fmt.Printf("  Alert %d: %s - %s\n", i+1, alert.Severity, alert.Message)
		}
	}

	// Stop the TEE
	fmt.Println("Stopping CustomTEE...")
	if err := tee.Stop(); err != nil {
		fmt.Printf("Error stopping CustomTEE: %v\n", err)
	}

	// Clean up
	os.RemoveAll(config.WorkingDir)
	fmt.Println("CustomTEE test completed")
}

func testEnhancedConnectionSecurity(t *testing.T) {
	fmt.Println("\n--- Testing Enhanced Connection Security ---")

	// Create a new enhanced connection security manager
	securityManager := api.NewEnhancedConnectionSecurityManager()

	// Create a security context
	userID := int64(1)
	permissionSet := "standard"
	securityLevel := api.EnhancedSecurityLevelHigh
	authMethod := "password"
	ipAddress := "192.168.1.100"
	userAgent := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"
	mfaVerified := true
	constraints := map[string]interface{}{
		"max_sessions": 5,
	}

	t.Log("Creating security context...")
	ctx, err := securityManager.CreateSecurityContext(
		userID,
		permissionSet,
		securityLevel,
		authMethod,
		ipAddress,
		userAgent,
		mfaVerified,
		constraints,
	)
	if err != nil {
		t.Logf("Error creating security context: %v", err)
		return
	}

	t.Logf("Security context created with session ID: %s", ctx.SessionID)
	t.Logf("  Security level: %s", ctx.SecurityLevel)
	t.Logf("  MFA verified: %v", ctx.MFAVerified)
	t.Logf("  Risk score: %.2f", ctx.RiskScore)
	t.Logf("  Expires at: %s", ctx.ExpiresAt.Format(time.RFC3339))

	// Validate the security context
	t.Log("Validating security context...")
	_, err = securityManager.ValidateSecurityContext(
		ctx.SessionID,
		ipAddress,
		userAgent,
	)
	if err != nil {
		t.Logf("Error validating security context: %v", err)
	} else {
		t.Log("Security context validated successfully")
	}

	// Check permissions
	t.Log("Checking permissions...")

	// Should succeed - read_files permission is in standard set
	err = securityManager.CheckPermission(
		ctx.SessionID,
		"filesystem",
		"read",
		"/home/user/documents/file.txt",
		ipAddress,
		userAgent,
	)
	if err != nil {
		t.Logf("Error checking read permission: %v", err)
	} else {
		t.Log("Read permission check passed")
	}

	// Should fail - path not in allowed paths
	err = securityManager.CheckPermission(
		ctx.SessionID,
		"filesystem",
		"read",
		"/etc/passwd",
		ipAddress,
		userAgent,
	)
	if err != nil {
		t.Logf("Expected error checking read permission for restricted path: %v", err)
	} else {
		t.Error("Error: Read permission check for restricted path should have failed")
	}

	// Get audit log
	t.Log("Getting audit log...")
	startTime := time.Now().Add(-1 * time.Hour)
	endTime := time.Now().Add(1 * time.Hour)
	eventTypes := []string{"session_created", "permission_check"}
	severity := ""

	auditLog := securityManager.GetAuditLog(
		startTime,
		endTime,
		eventTypes,
		severity,
		userID,
		ctx.SessionID,
	)

	t.Logf("Audit log entries: %d", len(auditLog))
	for i, event := range auditLog {
		t.Logf("  Event %d: %s - %s:%s - %s",
			i+1, event.EventType, event.Resource, event.Action, event.Status)
	}

	// Revoke the session
	t.Log("Revoking security context...")
	err = securityManager.RevokeSession(
		ctx.SessionID,
		"test completed",
		ipAddress,
		userAgent,
	)
	if err != nil {
		t.Logf("Error revoking security context: %v", err)
	} else {
		t.Log("Security context revoked successfully")
	}

	// Try to validate the revoked session
	t.Log("Trying to validate revoked security context...")
	_, err = securityManager.ValidateSecurityContext(
		ctx.SessionID,
		ipAddress,
		userAgent,
	)
	if err != nil {
		t.Logf("Expected error validating revoked context: %v", err)
	} else {
		t.Error("Error: Revoked context validation should have failed")
	}

	t.Log("Enhanced Connection Security test completed")
}

func testTerminalManager(t *testing.T) {
	fmt.Println("\n--- Testing Terminal Manager ---")

	// Create a new terminal manager
	terminalManager := api.NewTerminalManager()

	// Create a terminal session
	userID := int64(1)
	workingDir := "/tmp"
	env := []string{"TEST_VAR=test_value"}

	t.Log("Creating terminal session...")
	session, err := terminalManager.CreateSession(userID, workingDir, env)
	if err != nil {
		t.Logf("Error creating terminal session: %v", err)
		return
	}

	t.Logf("Terminal session created with ID: %s", session.ID)
	t.Logf("  Command: %s", session.Command)
	t.Logf("  Working directory: %s", session.WorkingDir)
	t.Logf("  Status: %s", session.Status)

	// Start the terminal session
	t.Log("Starting terminal session...")
	if err := session.Start(); err != nil {
		t.Logf("Error starting terminal session: %v", err)
		return
	}

	// Write input to the terminal
	t.Log("Writing 'echo hello world' to terminal...")
	if err := session.WriteInput("echo hello world"); err != nil {
		t.Logf("Error writing to terminal: %v", err)
	}

	// Wait for output
	time.Sleep(1 * time.Second)

	// Get terminal history
	t.Log("Getting terminal history...")
	history := session.GetHistory()
	t.Logf("Terminal history entries: %d", len(history))
	for i, entry := range history {
		t.Logf("  Entry %d: %s", i+1, entry.Command)
		t.Logf("    Output: %s", entry.Output)
		t.Logf("    Timestamp: %s", entry.Timestamp.Format(time.RFC3339))
	}

	// Get terminal info
	t.Log("Getting terminal info...")
	info := session.GetInfo()
	t.Logf("Terminal info:")
	t.Logf("  ID: %s", info.ID)
	t.Logf("  Status: %s", info.Status)
	t.Logf("  Command: %s", info.Command)
	t.Logf("  Working directory: %s", info.WorkingDir)
	t.Logf("  Client count: %d", info.ClientCount)

	// Close the terminal session
	t.Log("Closing terminal session...")
	if err := terminalManager.CloseSession(session.ID); err != nil {
		t.Logf("Error closing terminal session: %v", err)
	}

	// Try to get the closed session
	t.Log("Trying to get closed terminal session...")
	_, err = terminalManager.GetSession(session.ID)
	if err != nil {
		t.Logf("Expected error getting closed session: %v", err)
	} else {
		t.Error("Error: Closed session retrieval should have failed")
	}

	t.Log("Terminal Manager test completed")
}
