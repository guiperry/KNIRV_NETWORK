package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"Agentic_Engine/agentify"
	"Agentic_Engine/utils"
)

// ExampleInferenceService for demonstration
type ExampleInferenceService struct {
	isRunning bool
}

func (e *ExampleInferenceService) GenerateText(promptText string, instructionText string) (string, error) {
	response := fmt.Sprintf("LLM Response to: %s", promptText)
	if instructionText != "" {
		response += fmt.Sprintf("\n\nInstruction: %s", instructionText)
	}
	return response, nil
}

func (e *ExampleInferenceService) GenerateTextWithCoT(promptText string) (string, error) {
	return fmt.Sprintf("CoT Response: Let me think step by step about: %s", promptText), nil
}

func (e *ExampleInferenceService) GenerateStructuredOutput(content string, schema string) (string, error) {
	return fmt.Sprintf(`{"content": "%s", "schema": "%s"}`, content, schema), nil
}

func (e *ExampleInferenceService) IsRunning() bool {
	return e.isRunning
}

func main() {
	fmt.Println("Agent Inferencer Plugin Loading Example")
	fmt.Println("======================================")

	// Get the system-specific plugins directory
	pluginsDir, err := utils.GetPluginsDir()
	if err != nil {
		// Fallback to relative path if utils package is not available
		pluginsDir, err = filepath.Abs("../../../plugins")
		if err != nil {
			log.Fatalf("Failed to get plugins directory path: %v", err)
		}
	}

	fmt.Printf("Looking for plugins in: %s\n", pluginsDir)

	// Check if plugins directory exists
	if _, err := os.Stat(pluginsDir); os.IsNotExist(err) {
		log.Fatalf("Plugins directory does not exist: %s", pluginsDir)
	}

	// List available plugins
	fmt.Println("\n1. Discovering Available Plugins:")
	fmt.Println("---------------------------------")

	// Create the agent inferencer
	inferencer := agentify.NewAgentInferencer(pluginsDir)

	// Discover available plugins
	plugins, err := inferencer.ListAvailableAgents(context.Background())
	if err != nil {
		log.Printf("Error discovering plugins: %v", err)
	} else {
		fmt.Printf("Found %d plugins:\n", len(plugins))
		for i, plugin := range plugins {
			fmt.Printf("  %d. %s\n", i+1, plugin)
		}
	}

	// Check for your specific plugin file
	fmt.Println("\n2. Checking for Your Plugin:")
	fmt.Println("----------------------------")

	yourPluginPath := filepath.Join(pluginsDir, "build-1750292372086.so")
	if _, err := os.Stat(yourPluginPath); err == nil {
		fmt.Printf("✓ Found your plugin: %s\n", yourPluginPath)

		// To use this plugin with the Agent Inferencer, it needs to be renamed
		// to follow the convention: agent_{agentID}_{version}.so
		newPluginName := "agent_shopify_assistant_1.0.so"
		newPluginPath := filepath.Join(pluginsDir, newPluginName)

		fmt.Printf("→ To use with Agent Inferencer, rename to: %s\n", newPluginName)

		// Ask user if they want to rename it
		fmt.Print("Would you like to rename it now? (y/n): ")
		var response string
		fmt.Scanln(&response)

		if response == "y" || response == "Y" {
			if err := os.Rename(yourPluginPath, newPluginPath); err != nil {
				log.Printf("Failed to rename plugin: %v", err)
			} else {
				fmt.Printf("✓ Plugin renamed successfully!\n")

				// Now try to load the renamed plugin
				fmt.Println("\n3. Loading Your Plugin:")
				fmt.Println("----------------------")

				// Set up inference service
				inferenceService := &ExampleInferenceService{isRunning: true}
				inferencer.SetInferenceService(inferenceService)

				// Try to activate the plugin
				sessionID := "demo-session"
				agentID := "shopify_assistant"
				version := "1.0"

				config := map[string]interface{}{
					"agentID": agentID,
					"version": version,
					"debug":   true,
				}

				fmt.Printf("Attempting to load plugin: %s version %s\n", agentID, version)

				err := inferencer.ActivateAgent(context.Background(), agentID, version, sessionID, config)
				if err != nil {
					log.Printf("Failed to activate agent: %v", err)
					fmt.Println("\nNote: Your plugin might not implement the AgentPluginInterface.")
					fmt.Println("For the plugin to work with Agent Inferencer, it needs to:")
					fmt.Println("1. Export a 'Plugin' symbol of type AgentPluginInterface")
					fmt.Println("2. Implement all required methods (Initialize, Start, Stop, ProcessInference, etc.)")
					fmt.Println("3. Be compiled with: go build -buildmode=plugin")
				} else {
					fmt.Printf("✓ Plugin loaded successfully!\n")

					// Test the plugin
					fmt.Println("\n4. Testing Plugin Functionality:")
					fmt.Println("--------------------------------")

					// Get capabilities
					capabilities, err := inferencer.GetAgentCapabilities(context.Background(), sessionID)
					if err != nil {
						log.Printf("Error getting capabilities: %v", err)
					} else {
						fmt.Printf("Agent Capabilities:\n")
						fmt.Printf("  - Supports Streaming: %t\n", capabilities.SupportsStreaming)
						fmt.Printf("  - Supports Tool Calls: %t\n", capabilities.SupportsToolCalls)
						fmt.Printf("  - Max Context Length: %d\n", capabilities.MaxContextLength)
					}

					// Test inference
					request := &agentify.InferenceRequest{
						Input:     "Hello! Can you help me with my Shopify store?",
						SessionID: sessionID,
						Parameters: map[string]interface{}{
							"temperature": 0.7,
						},
					}

					response, err := inferencer.ProcessInference(context.Background(), sessionID, request)
					if err != nil {
						log.Printf("Error processing inference: %v", err)
					} else {
						fmt.Printf("Plugin Response: %s\n", response.Output)
						if len(response.ToolCalls) > 0 {
							fmt.Printf("Tool Calls Made: %d\n", len(response.ToolCalls))
						}
					}

					// Test memory operations
					fmt.Println("\n5. Testing Memory Operations:")
					fmt.Println("-----------------------------")

					err = inferencer.SetAgentMemory(context.Background(), sessionID, "user_store", "my-shopify-store.myshopify.com")
					if err != nil {
						log.Printf("Error setting memory: %v", err)
					} else {
						fmt.Println("✓ Memory set successfully")
					}

					value, err := inferencer.GetAgentMemory(context.Background(), sessionID, "user_store")
					if err != nil {
						log.Printf("Error getting memory: %v", err)
					} else {
						fmt.Printf("Retrieved memory: %v\n", value)
					}

					// Test terminal creation
					fmt.Println("\n6. Testing Terminal Creation:")
					fmt.Println("-----------------------------")

					terminalID, err := inferencer.CreateTerminal(context.Background(), sessionID, 24, 80)
					if err != nil {
						log.Printf("Error creating terminal: %v", err)
					} else {
						fmt.Printf("✓ Terminal created: %s\n", terminalID)

						// Clean up terminal
						err = inferencer.CloseTerminal(context.Background(), sessionID, terminalID)
						if err != nil {
							log.Printf("Error closing terminal: %v", err)
						} else {
							fmt.Println("✓ Terminal closed successfully")
						}
					}

					// Deactivate the agent
					fmt.Println("\n7. Cleanup:")
					fmt.Println("-----------")

					err = inferencer.DeactivateAgent(context.Background(), sessionID)
					if err != nil {
						log.Printf("Error deactivating agent: %v", err)
					} else {
						fmt.Println("✓ Agent deactivated successfully")
					}
				}
			}
		} else {
			fmt.Println("Plugin not renamed. To use it with Agent Inferencer:")
			fmt.Printf("  mv %s %s\n", yourPluginPath, newPluginPath)
		}
	} else {
		fmt.Printf("✗ Plugin not found: %s\n", yourPluginPath)
		fmt.Println("Make sure your plugin is in the plugins directory.")
	}

	// Show how to create a compatible plugin
	fmt.Println("\n8. Creating Compatible Plugins:")
	fmt.Println("------------------------------")
	fmt.Println("To create a plugin compatible with Agent Inferencer:")
	fmt.Println("1. Implement the AgentPluginInterface")
	fmt.Println("2. Export a 'Plugin' variable of that type")
	fmt.Println("3. Compile with: go build -buildmode=plugin -o agent_name_version.so")
	fmt.Println("4. Place in the plugins directory")

	fmt.Println("\nExample plugin structure:")
	fmt.Println("```go")
	fmt.Println("package main")
	fmt.Println("")
	fmt.Println("import \"Agentic_Engine/agentify\"")
	fmt.Println("")
	fmt.Println("type MyAgent struct {")
	fmt.Println("    *agentify.BaseAgentPlugin")
	fmt.Println("}")
	fmt.Println("")
	fmt.Println("var Plugin agentify.AgentPluginInterface = &MyAgent{}")
	fmt.Println("```")

	fmt.Println("\nExample completed!")
}
