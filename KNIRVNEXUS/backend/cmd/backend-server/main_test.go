package main

import (
	"testing"
	"backend_server/internal/config"
)

func TestNewServer(t *testing.T) {
	// Load test configuration
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Test creating a new server
	server, err := NewServer(cfg)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	if server == nil {
		t.Fatal("Server is nil")
	}

	// Test that server has required components
	if server.config == nil {
		t.Error("Server config is nil")
	}

	if server.db == nil {
		t.Error("Server database is nil")
	}

	if server.router == nil {
		t.Error("Server router is nil")
	}

	if server.p2pManager == nil {
		t.Error("Server P2P manager is nil")
	}
}

func TestServerSetupRoutes(t *testing.T) {
	// Load test configuration
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Create server
	server, err := NewServer(cfg)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// setupRoutes is called in NewServer, so we can check if routes are set up
	if server.router == nil {
		t.Error("Router should be initialized")
	}
}

func TestVersionVariables(t *testing.T) {
	// Test that version variables are set
	if Version == "" {
		t.Error("Version should not be empty")
	}

	if BuildTime == "" {
		t.Error("BuildTime should not be empty")
	}

	if GitCommit == "" {
		t.Error("GitCommit should not be empty")
	}
}
