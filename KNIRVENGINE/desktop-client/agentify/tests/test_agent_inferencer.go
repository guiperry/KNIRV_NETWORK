package agentify_test

import (
	"context"

	"testing"
	"time"

	"Agentic_Engine/agentify"
)

func TestAgentInferencer(t *testing.T) {
	// Create a new Agent Inferencer
	inferencer := agentify.NewAgentInferencer("./plugins")

	// Create a session ID
	sessionID := "test-session"

	// Test activating an agent
	t.Run("ActivateAgent", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Activate the agent
		err := inferencer.ActivateAgent(ctx, "example", "1.0", sessionID, nil)
		if err != nil {
			t.Fatalf("Failed to activate agent: %v", err)
		}
	})

	// Test processing an inference request
	t.Run("ProcessInference", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Create an inference request
		request := &agentify.InferenceRequest{
			Input:     "Hello, world!",
			SessionID: sessionID,
		}

		// Process the inference request
		response, err := inferencer.ProcessInference(ctx, sessionID, request)
		if err != nil {
			t.Fatalf("Failed to process inference: %v", err)
		}

		// Check the response
		if response.Output == "" {
			t.Fatalf("Empty response output")
		}
	})

	// Test getting agent capabilities
	t.Run("GetAgentCapabilities", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Get the agent capabilities
		capabilities, err := inferencer.GetAgentCapabilities(ctx, sessionID)
		if err != nil {
			t.Fatalf("Failed to get agent capabilities: %v", err)
		}

		// Check the capabilities
		if capabilities == nil {
			t.Fatalf("Nil capabilities")
		}
	})

	// Test getting agent schema
	t.Run("GetAgentSchema", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Get the agent schema
		schema, err := inferencer.GetAgentSchema(ctx, sessionID)
		if err != nil {
			t.Fatalf("Failed to get agent schema: %v", err)
		}

		// Check the schema
		if schema == nil {
			t.Fatalf("Nil schema")
		}
	})

	// Test setting and getting memory
	t.Run("Memory", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Set a memory value
		err := inferencer.SetAgentMemory(ctx, sessionID, "test-key", "test-value")
		if err != nil {
			t.Fatalf("Failed to set memory: %v", err)
		}

		// Get the memory value
		value, err := inferencer.GetAgentMemory(ctx, sessionID, "test-key")
		if err != nil {
			t.Fatalf("Failed to get memory: %v", err)
		}

		// Check the value
		if value != "test-value" {
			t.Fatalf("Unexpected memory value: %v", value)
		}
	})

	// Test getting TEE info
	t.Run("GetTEEInfo", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Get the TEE info
		info, err := inferencer.GetTEEInfo(ctx, sessionID)
		if err != nil {
			t.Fatalf("Failed to get TEE info: %v", err)
		}

		// Check the info
		if info == nil {
			t.Fatalf("Nil TEE info")
		}
	})

	// Test deactivating an agent
	t.Run("DeactivateAgent", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Deactivate the agent
		err := inferencer.DeactivateAgent(ctx, sessionID)
		if err != nil {
			t.Fatalf("Failed to deactivate agent: %v", err)
		}
	})
}
