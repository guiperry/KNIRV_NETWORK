package agentify

import (
	"os"
	"testing"
	"time"
)

func TestTerminalManager(t *testing.T) {
	// Create a new terminal manager
	manager := NewTerminalManager()
	if manager == nil {
		t.Fatal("Failed to create terminal manager")
	}

	// Create a mock agent plugin
	agent := NewBaseAgentPlugin()
	if err := agent.Initialize(map[string]interface{}{}); err != nil {
		t.Fatalf("Failed to initialize agent: %v", err)
	}

	// Test creating a terminal session
	session, err := manager.NewTerminalSession(agent, 24, 80)
	if err != nil {
		t.Fatalf("Failed to create terminal session: %v", err)
	}

	if session.ID == "" {
		t.Error("Terminal session ID should not be empty")
	}

	if session.Rows != 24 || session.Cols != 80 {
		t.Errorf("Terminal size mismatch: expected 24x80, got %dx%d", session.Rows, session.Cols)
	}

	// Test writing to terminal
	testData := []byte("echo 'Hello, World!'\n")
	err = manager.WriteToTerminal(session.ID, testData)
	if err != nil {
		t.Errorf("Failed to write to terminal: %v", err)
	}

	// Wait a bit for the command to execute
	time.Sleep(100 * time.Millisecond)

	// Test reading from terminal
	output, err := manager.ReadFromTerminal(session.ID)
	if err != nil {
		t.Errorf("Failed to read from terminal: %v", err)
	}

	if len(output) == 0 {
		t.Error("Expected some output from terminal")
	}

	// Test resizing terminal
	err = manager.ResizeTerminal(session.ID, 30, 120)
	if err != nil {
		t.Errorf("Failed to resize terminal: %v", err)
	}

	// Verify the resize
	updatedSession, err := manager.GetTerminalSession(session.ID)
	if err != nil {
		t.Errorf("Failed to get terminal session: %v", err)
	}

	if updatedSession.Rows != 30 || updatedSession.Cols != 120 {
		t.Errorf("Terminal resize failed: expected 30x120, got %dx%d", updatedSession.Rows, updatedSession.Cols)
	}

	// Test listing terminal sessions
	sessions := manager.ListTerminalSessions()
	if len(sessions) != 1 {
		t.Errorf("Expected 1 terminal session, got %d", len(sessions))
	}

	if sessions[0] != session.ID {
		t.Errorf("Session ID mismatch: expected %s, got %s", session.ID, sessions[0])
	}

	// Test closing terminal
	err = manager.CloseTerminalSession(session.ID)
	if err != nil {
		t.Errorf("Failed to close terminal session: %v", err)
	}

	// Verify the session is closed
	sessions = manager.ListTerminalSessions()
	if len(sessions) != 0 {
		t.Errorf("Expected 0 terminal sessions after closing, got %d", len(sessions))
	}

	// Test operations on non-existent session
	err = manager.WriteToTerminal("non-existent", testData)
	if err == nil {
		t.Error("Expected error when writing to non-existent terminal")
	}

	_, err = manager.ReadFromTerminal("non-existent")
	if err == nil {
		t.Error("Expected error when reading from non-existent terminal")
	}

	err = manager.ResizeTerminal("non-existent", 24, 80)
	if err == nil {
		t.Error("Expected error when resizing non-existent terminal")
	}

	err = manager.CloseTerminalSession("non-existent")
	if err == nil {
		t.Error("Expected error when closing non-existent terminal")
	}
}

func TestBaseAgentPluginTerminalMethods(t *testing.T) {
	// Create and initialize a base agent plugin
	agent := NewBaseAgentPlugin()

	// Create a temporary working directory for the TEE
	teeWorkDir, err := os.MkdirTemp("", "tee_work")
	if err != nil {
		t.Fatalf("Failed to create TEE working directory: %v", err)
	}
	defer os.RemoveAll(teeWorkDir)

	config := map[string]interface{}{
		"tee": map[string]interface{}{
			"isolationLevel": "process",
			"workingDir":     teeWorkDir,
		},
	}

	if err := agent.Initialize(config); err != nil {
		t.Fatalf("Failed to initialize agent: %v", err)
	}

	if err := agent.Start(); err != nil {
		t.Fatalf("Failed to start agent: %v", err)
	}

	// Test creating a terminal
	terminalID, err := agent.CreateTerminal(24, 80)
	if err != nil {
		t.Fatalf("Failed to create terminal: %v", err)
	}

	if terminalID == "" {
		t.Error("Terminal ID should not be empty")
	}

	// Test writing to terminal
	testData := []byte("pwd\n")
	err = agent.WriteToTerminal(terminalID, testData)
	if err != nil {
		t.Errorf("Failed to write to terminal: %v", err)
	}

	// Wait for command execution
	time.Sleep(100 * time.Millisecond)

	// Test reading from terminal
	output, err := agent.ReadFromTerminal(terminalID)
	if err != nil {
		t.Errorf("Failed to read from terminal: %v", err)
	}

	if len(output) == 0 {
		t.Error("Expected some output from terminal")
	}

	// Test resizing terminal
	err = agent.ResizeTerminal(terminalID, 30, 120)
	if err != nil {
		t.Errorf("Failed to resize terminal: %v", err)
	}

	// Test closing terminal
	err = agent.CloseTerminal(terminalID)
	if err != nil {
		t.Errorf("Failed to close terminal: %v", err)
	}

	// Test operations on closed terminal
	err = agent.WriteToTerminal(terminalID, testData)
	if err == nil {
		t.Error("Expected error when writing to closed terminal")
	}

	// Stop the agent
	if err := agent.Stop(); err != nil {
		t.Errorf("Failed to stop agent: %v", err)
	}
}
