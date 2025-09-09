package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"KNIRV_Engine/agent"
)

func main() {
	// Get config directory
	configDir := filepath.Join(os.Getenv("HOME"), ".config", "KNIRV-Engine")
	pluginsDir := filepath.Join(configDir, "plugins")
	templatesDir := "../agent/templates"
	dbPath := filepath.Join(configDir, "agents.db")

	// Ensure directories exist
	os.MkdirAll(pluginsDir, 0755)

	// Create agent builder
	builder, err := agent.NewAgentBuilder(templatesDir, pluginsDir, dbPath)
	if err != nil {
		log.Fatalf("Failed to create agent builder: %v", err)
	}
	defer builder.Close()

	// Create WASM agent configuration
	config := agent.AgentConfig{
		AgentType:   "llm",
		Name:        "TestWASMAgent",
		Description: "A test WASM agent for terminal functionality",
		Model:       "gpt-4",
		Instruction: "You are a helpful test assistant.",
		BuildTarget: "wasm",
		ExtraParams: map[string]interface{}{
			"version":    "1.0",
			"collection": "Test_Collection",
		},
	}

	// Build the agent
	agentID, err := builder.BuildAgent(config)
	if err != nil {
		log.Fatalf("Failed to build WASM agent: %v", err)
	}

	fmt.Printf("Successfully built WASM agent with ID: %s\n", agentID)

	// Check if the WASM file was created
	wasmPath := filepath.Join(pluginsDir, fmt.Sprintf("agent_%s_1.0.wasm", agentID))
	if _, err := os.Stat(wasmPath); err == nil {
		fmt.Printf("WASM file created at: %s\n", wasmPath)
	} else {
		fmt.Printf("WASM file not found at: %s\n", wasmPath)
	}
}
