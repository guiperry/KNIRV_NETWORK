package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"Agentic_Engine/agentify"
	"Agentic_Engine/utils"
)

// RealInferenceService demonstrates integration with actual inference service
type RealInferenceService struct {
	isRunning bool
}

func (r *RealInferenceService) GenerateText(promptText string, instructionText string) (string, error) {
	// This would integrate with your actual inference service
	// For demo purposes, we'll simulate a response
	response := fmt.Sprintf("AI Assistant: %s", promptText)
	if instructionText != "" {
		response = fmt.Sprintf("Following instruction '%s': %s", instructionText, response)
	}

	// Simulate tool calling suggestion
	if len(promptText) > 50 {
		response += "\n\nI can help you with that. Let me use some tools: TOOL_CALL[shopify_api](action=get_products)"
	}

	return response, nil
}

func (r *RealInferenceService) GenerateTextWithCoT(promptText string) (string, error) {
	response := fmt.Sprintf("Let me think about this step by step:\n\n")
	response += fmt.Sprintf("1. Understanding the request: %s\n", promptText)
	response += "2. Analyzing the best approach...\n"
	response += "3. Considering available tools and resources...\n"
	response += "4. Formulating a comprehensive response...\n\n"
	response += "Based on my analysis, here's my response: I'll help you with your Shopify-related task."

	return response, nil
}

func (r *RealInferenceService) GenerateStructuredOutput(content string, schema string) (string, error) {
	return fmt.Sprintf(`{
		"status": "success",
		"content": "%s",
		"schema_used": "%s",
		"timestamp": "%s",
		"structured": true
	}`, content, schema, time.Now().Format(time.RFC3339)), nil
}

func (r *RealInferenceService) IsRunning() bool {
	return r.isRunning
}

func main() {
	fmt.Println("🚀 Real Plugin Demo - Agent Inferencer with Your Plugin")
	fmt.Println("======================================================")

	// Setup paths - use system-specific plugins directory
	pluginsDir, err := utils.GetPluginsDir()
	if err != nil {
		// Fallback to relative path if utils package is not available
		pluginsDir, err = filepath.Abs("../../../plugins")
		if err != nil {
			log.Fatalf("Failed to get plugins directory: %v", err)
		}
	}

	fmt.Printf("📁 Plugins directory: %s\n", pluginsDir)

	// Check if your plugin exists
	yourPlugin := filepath.Join(pluginsDir, "build-1750292372086.so")
	if _, err := os.Stat(yourPlugin); os.IsNotExist(err) {
		log.Fatalf("❌ Your plugin not found: %s", yourPlugin)
	}

	fmt.Printf("✅ Found your plugin: %s\n", yourPlugin)

	// Create Agent Inferencer Service
	fmt.Println("\n🔧 Setting up Agent Inferencer Service...")
	service := agentify.NewAgentInferencerService(pluginsDir)

	// Setup inference service
	inferenceService := &RealInferenceService{isRunning: true}

	// Start the service
	if err := service.Start(); err != nil {
		log.Fatalf("Failed to start Agent Inferencer Service: %v", err)
	}
	defer service.Stop()

	fmt.Println("✅ Agent Inferencer Service started successfully!")

	// Set the inference service
	// Note: We need to access the underlying inferencer to set the inference service
	// In a real implementation, this would be part of the service configuration
	fmt.Println("\n🧠 Connecting LLM Inference Service...")

	// For demonstration, let's work directly with the inferencer
	inferencer := agentify.NewAgentInferencer(pluginsDir)
	inferencer.SetInferenceService(inferenceService)

	// Discover available plugins
	fmt.Println("\n🔍 Discovering Available Plugins...")
	plugins, err := inferencer.ListAvailableAgents(context.Background())
	if err != nil {
		log.Printf("Error discovering plugins: %v", err)
	} else {
		fmt.Printf("Found %d plugins:\n", len(plugins))
		for i, plugin := range plugins {
			fmt.Printf("  %d. %s\n", i+1, plugin)
		}
	}

	// Rename your plugin to follow the convention if needed
	fmt.Println("\n📝 Preparing Your Plugin...")
	agentPluginPath := filepath.Join(pluginsDir, "agent_shopify_assistant_1.0.so")

	if _, err := os.Stat(agentPluginPath); os.IsNotExist(err) {
		fmt.Printf("Renaming your plugin to follow Agent Inferencer convention...\n")
		if err := os.Rename(yourPlugin, agentPluginPath); err != nil {
			log.Printf("Warning: Could not rename plugin: %v", err)
			fmt.Println("Note: Your plugin needs to be renamed to agent_shopify_assistant_1.0.so")
			fmt.Println("And it needs to implement the AgentPluginInterface")
		} else {
			fmt.Printf("✅ Plugin renamed to: %s\n", agentPluginPath)
		}
	}

	// Try to load and activate your plugin
	fmt.Println("\n🔌 Loading Your Plugin...")
	sessionID := "demo-session-" + fmt.Sprintf("%d", time.Now().Unix())
	agentID := "shopify_assistant"
	version := "1.0"

	config := map[string]interface{}{
		"agentID":     agentID,
		"version":     version,
		"debug":       true,
		"description": "Shopify Assistant Agent",
		"tools": []interface{}{
			map[string]interface{}{
				"name":        "shopify_api",
				"description": "Access Shopify API functions",
			},
		},
	}

	fmt.Printf("Attempting to activate agent: %s v%s\n", agentID, version)
	err = inferencer.ActivateAgent(context.Background(), agentID, version, sessionID, config)

	if err != nil {
		fmt.Printf("❌ Failed to load your plugin: %v\n", err)
		fmt.Println("\n💡 This is expected because your plugin is CGO-based.")
		fmt.Println("To make it work with Agent Inferencer, you need to:")
		fmt.Println("1. Create a Go plugin that implements AgentPluginInterface")
		fmt.Println("2. Use CGO to call your existing C functions")
		fmt.Println("3. Export a 'Plugin' variable of type AgentPluginInterface")

		// Demonstrate with a mock agent instead
		fmt.Println("\n🔄 Demonstrating with Mock Agent...")
		mockAgent := agentify.NewBaseAgentPlugin()
		if err := mockAgent.Initialize(config); err != nil {
			log.Fatalf("Failed to initialize mock agent: %v", err)
		}
		if err := mockAgent.Start(); err != nil {
			log.Fatalf("Failed to start mock agent: %v", err)
		}

		// Manually set the agent for demonstration
		inferencer.SetActiveAgent(sessionID, agentID, mockAgent)
		fmt.Println("✅ Mock agent activated for demonstration")
	} else {
		fmt.Println("✅ Your plugin loaded successfully!")
	}

	// Test agent functionality
	fmt.Println("\n🧪 Testing Agent Functionality...")

	// Test 1: Get agent capabilities
	fmt.Println("\n1. Agent Capabilities:")
	capabilities, err := inferencer.GetAgentCapabilities(context.Background(), sessionID)
	if err != nil {
		log.Printf("Error getting capabilities: %v", err)
	} else {
		fmt.Printf("   ✓ Supports Streaming: %t\n", capabilities.SupportsStreaming)
		fmt.Printf("   ✓ Supports Tool Calls: %t\n", capabilities.SupportsToolCalls)
		fmt.Printf("   ✓ Supports Reasoning: %t\n", capabilities.SupportsReasoning)
		fmt.Printf("   ✓ Max Context Length: %d\n", capabilities.MaxContextLength)
	}

	// Test 2: Get agent schema
	fmt.Println("\n2. Agent Schema:")
	schema, err := inferencer.GetAgentSchema(context.Background(), sessionID)
	if err != nil {
		log.Printf("Error getting schema: %v", err)
	} else {
		fmt.Printf("   ✓ Tools: %d\n", len(schema.Tools))
		fmt.Printf("   ✓ Resources: %d\n", len(schema.Resources))
		fmt.Printf("   ✓ Prompts: %d\n", len(schema.Prompts))
	}

	// Test 3: Process inference
	fmt.Println("\n3. Processing Inference:")
	request := &agentify.InferenceRequest{
		Input:     "Hello! I need help managing my Shopify store. Can you help me get a list of my products?",
		SessionID: sessionID,
		Parameters: map[string]interface{}{
			"temperature": 0.7,
			"max_tokens":  150,
		},
	}

	response, err := inferencer.ProcessInference(context.Background(), sessionID, request)
	if err != nil {
		log.Printf("Error processing inference: %v", err)
	} else {
		fmt.Printf("   ✓ Response: %s\n", response.Output)
		fmt.Printf("   ✓ Tool Calls: %d\n", len(response.ToolCalls))
		if response.Metadata != nil {
			fmt.Printf("   ✓ Has LLM Integration: %v\n", response.Metadata["has_inference_service"])
		}
	}

	// Test 4: Memory operations
	fmt.Println("\n4. Memory Operations:")
	err = inferencer.SetAgentMemory(context.Background(), sessionID, "store_url", "my-store.myshopify.com")
	if err != nil {
		log.Printf("Error setting memory: %v", err)
	} else {
		fmt.Println("   ✓ Memory set successfully")
	}

	storeURL, err := inferencer.GetAgentMemory(context.Background(), sessionID, "store_url")
	if err != nil {
		log.Printf("Error getting memory: %v", err)
	} else {
		fmt.Printf("   ✓ Retrieved store URL: %v\n", storeURL)
	}

	// Test 5: Terminal session
	fmt.Println("\n5. Terminal Session:")
	terminalID, err := inferencer.CreateTerminal(context.Background(), sessionID, 24, 80)
	if err != nil {
		log.Printf("Error creating terminal: %v", err)
	} else {
		fmt.Printf("   ✓ Terminal created: %s\n", terminalID)

		// Write to terminal
		err = inferencer.WriteToTerminal(context.Background(), sessionID, terminalID, []byte("echo 'Hello from Shopify Assistant!'\n"))
		if err != nil {
			log.Printf("Error writing to terminal: %v", err)
		} else {
			fmt.Println("   ✓ Command sent to terminal")
		}

		// Read from terminal
		time.Sleep(100 * time.Millisecond)
		output, err := inferencer.ReadFromTerminal(context.Background(), sessionID, terminalID)
		if err != nil {
			log.Printf("Error reading from terminal: %v", err)
		} else {
			fmt.Printf("   ✓ Terminal output: %s", string(output))
		}

		// Close terminal
		err = inferencer.CloseTerminal(context.Background(), sessionID, terminalID)
		if err != nil {
			log.Printf("Error closing terminal: %v", err)
		} else {
			fmt.Println("   ✓ Terminal closed")
		}
	}

	// Cleanup
	fmt.Println("\n🧹 Cleanup:")
	err = inferencer.DeactivateAgent(context.Background(), sessionID)
	if err != nil {
		log.Printf("Error deactivating agent: %v", err)
	} else {
		fmt.Println("   ✓ Agent deactivated")
	}

	fmt.Println("\n🎉 Demo completed successfully!")
	fmt.Println("\n📋 Next Steps:")
	fmt.Println("1. Create a Go plugin wrapper for your CGO plugin")
	fmt.Println("2. Implement the AgentPluginInterface in the wrapper")
	fmt.Println("3. Use CGO to call your existing C functions")
	fmt.Println("4. Compile with: go build -buildmode=plugin")
	fmt.Println("5. Test with the Agent Inferencer system")
}
