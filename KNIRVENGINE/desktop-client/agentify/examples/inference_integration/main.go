package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"KNIRVENGINE/desktop-client/agentify"
)

// ExampleInferenceService demonstrates how to integrate with the existing inference service
type ExampleInferenceService struct {
	isRunning bool
}

func (e *ExampleInferenceService) GenerateText(promptText string, instructionText string) (string, error) {
	// In a real implementation, this would call the actual inference service
	// from the inference package
	response := fmt.Sprintf("LLM Response to: %s", promptText)

	// Simulate tool calling in the response
	if instructionText != "" {
		response += fmt.Sprintf("\n\nBased on instruction: %s", instructionText)
	}

	// Example of tool calling in response
	response += "\n\nI'll help you with that. Let me use a tool: TOOL_CALL[calculator](expression=2+2)"

	return response, nil
}

func (e *ExampleInferenceService) GenerateTextWithCoT(promptText string) (string, error) {
	// Chain of Thought reasoning
	response := fmt.Sprintf("Let me think step by step about: %s\n\n", promptText)
	response += "Step 1: Understanding the problem...\n"
	response += "Step 2: Analyzing the requirements...\n"
	response += "Step 3: Formulating a solution...\n\n"
	response += "Based on my reasoning, here's my response: This is a thoughtful answer."

	return response, nil
}

func (e *ExampleInferenceService) GenerateStructuredOutput(content string, schema string) (string, error) {
	// Structured output generation
	return fmt.Sprintf(`{
		"content": "%s",
		"schema": "%s",
		"structured": true,
		"timestamp": "%s"
	}`, content, schema, time.Now().Format(time.RFC3339)), nil
}

func (e *ExampleInferenceService) IsRunning() bool {
	return e.isRunning
}

func main() {
	fmt.Println("Agent Inferencer with LLM Integration Example")
	fmt.Println("=============================================")

	// Create the agent inferencer
	inferencer := agentify.NewAgentInferencer("./plugins")

	// Create and set up the inference service
	inferenceService := &ExampleInferenceService{isRunning: true}
	inferencer.SetInferenceService(inferenceService)

	// Create a base agent plugin for demonstration
	agent := agentify.NewBaseAgentPlugin()

	// Initialize the agent with some configuration
	config := map[string]interface{}{
		"agentID": "example-agent",
		"version": "1.0",
		"tools": []interface{}{
			map[string]interface{}{
				"name":           "calculator",
				"implementation": "return {'result': eval(params['expression'])}",
			},
			map[string]interface{}{
				"name":           "file_reader",
				"implementation": "return {'content': 'File content here'}",
			},
		},
	}

	if err := agent.Initialize(config); err != nil {
		log.Fatalf("Failed to initialize agent: %v", err)
	}

	if err := agent.Start(); err != nil {
		log.Fatalf("Failed to start agent: %v", err)
	}

	// Manually add the agent to the inferencer for this example
	sessionID := "example-session"
	agentID := "example-agent"
	inferencer.SetActiveAgent(sessionID, agentID, agent)

	// Example 1: Basic inference with LLM
	fmt.Println("\n1. Basic Inference with LLM:")
	fmt.Println("-----------------------------")

	request := &agentify.InferenceRequest{
		Input:     "What is 2 + 2?",
		SessionID: sessionID,
		Parameters: map[string]interface{}{
			"temperature": 0.7,
		},
	}

	response, err := inferencer.ProcessInference(context.Background(), sessionID, request)
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		fmt.Printf("Response: %s\n", response.Output)
		fmt.Printf("Tool Calls: %d\n", len(response.ToolCalls))
		if len(response.ToolCalls) > 0 {
			for i, toolCall := range response.ToolCalls {
				fmt.Printf("  Tool %d: %s with params %v -> %v\n",
					i+1, toolCall.Name, toolCall.Input, toolCall.Output)
			}
		}
	}

	// Example 2: Chain of Thought inference
	fmt.Println("\n2. Chain of Thought Inference:")
	fmt.Println("------------------------------")

	cotRequest := &agentify.InferenceRequest{
		Input:     "How do I solve a complex math problem?",
		SessionID: sessionID,
		Parameters: map[string]interface{}{
			"use_cot":     true,
			"temperature": 0.5,
		},
	}

	cotResponse, err := inferencer.ProcessInference(context.Background(), sessionID, cotRequest)
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		fmt.Printf("CoT Response: %s\n", cotResponse.Output)
	}

	// Example 3: Agent capabilities and schema
	fmt.Println("\n3. Agent Capabilities and Schema:")
	fmt.Println("---------------------------------")

	capabilities, err := inferencer.GetAgentCapabilities(context.Background(), sessionID)
	if err != nil {
		log.Printf("Error getting capabilities: %v", err)
	} else {
		fmt.Printf("Supports Streaming: %t\n", capabilities.SupportsStreaming)
		fmt.Printf("Supports Tool Calls: %t\n", capabilities.SupportsToolCalls)
		fmt.Printf("Max Context Length: %d\n", capabilities.MaxContextLength)
	}

	schema, err := inferencer.GetAgentSchema(context.Background(), sessionID)
	if err != nil {
		log.Printf("Error getting schema: %v", err)
	} else {
		fmt.Printf("Available Tools: %d\n", len(schema.Tools))
		for _, tool := range schema.Tools {
			fmt.Printf("  - %s: %s\n", tool.Name, tool.Description)
		}
	}

	// Example 4: Terminal session
	fmt.Println("\n4. Terminal Session:")
	fmt.Println("-------------------")

	terminalID, err := inferencer.CreateTerminal(context.Background(), sessionID, 24, 80)
	if err != nil {
		log.Printf("Error creating terminal: %v", err)
	} else {
		fmt.Printf("Created terminal: %s\n", terminalID)

		// Write to terminal
		err = inferencer.WriteToTerminal(context.Background(), sessionID, terminalID, []byte("echo 'Hello from Agent Inferencer!'\n"))
		if err != nil {
			log.Printf("Error writing to terminal: %v", err)
		}

		// Read from terminal (after a short delay)
		time.Sleep(100 * time.Millisecond)
		output, err := inferencer.ReadFromTerminal(context.Background(), sessionID, terminalID)
		if err != nil {
			log.Printf("Error reading from terminal: %v", err)
		} else {
			fmt.Printf("Terminal output: %s\n", string(output))
		}

		// Close terminal
		err = inferencer.CloseTerminal(context.Background(), sessionID, terminalID)
		if err != nil {
			log.Printf("Error closing terminal: %v", err)
		} else {
			fmt.Println("Terminal closed successfully")
		}
	}

	// Example 5: Memory operations
	fmt.Println("\n5. Memory Operations:")
	fmt.Println("--------------------")

	// Set memory
	err = inferencer.SetAgentMemory(context.Background(), sessionID, "user_preference", "dark_mode")
	if err != nil {
		log.Printf("Error setting memory: %v", err)
	} else {
		fmt.Println("Memory set successfully")
	}

	// Get memory
	value, err := inferencer.GetAgentMemory(context.Background(), sessionID, "user_preference")
	if err != nil {
		log.Printf("Error getting memory: %v", err)
	} else {
		fmt.Printf("Retrieved memory value: %v\n", value)
	}

	// Clean up
	fmt.Println("\n6. Cleanup:")
	fmt.Println("-----------")

	err = inferencer.DeactivateAgent(context.Background(), sessionID)
	if err != nil {
		log.Printf("Error deactivating agent: %v", err)
	} else {
		fmt.Println("Agent deactivated successfully")
	}

	fmt.Println("\nExample completed successfully!")
}
