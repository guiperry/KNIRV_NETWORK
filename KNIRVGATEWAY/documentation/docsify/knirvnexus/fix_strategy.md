

---

**Source**: KNIRVNEXUS/docs/fix_strategy.md

# Agent System Fix Strategy

This document outlines the strategy to fix several issues in the Agentic Engine platform related to agent terminals, WASM discovery, and agent configuration persistence.

## 1. Terminal Width Adjustment

**Issue:** The frontend agent terminals are 42 columns wide, causing text to bleed into the right edge. They need to be 37 columns wide instead.

**Fix Strategy:**
1. Modify the `AgentTerminal.jsx` component to adjust the terminal width:
   - Update the terminal initialization in `AgentTerminal.jsx` to change the `cols` parameter from 60 to 37
   - Ensure the terminal container has appropriate padding to prevent text bleeding
   - Adjust the CSS styling to accommodate the new width

**Implementation Details:**
```javascript
// In AgentTerminal.jsx, line ~83
const term = new Terminal({
  rows: 12,
  cols: 37, // Changed from 60 to 37
  // other settings remain the same
});
```

## 1.b. Agent Process Output Visibility

**Issue:** Need to see all processes stderr and stdout messages pertaining to an agent.

**Fix Strategy:**
1. Enhance the backend process management to capture all stdout and stderr output:
   - Modify the terminal implementation in `agentify/terminal.go` to capture and relay all process output
   - Implement a dedicated log relay service that collects all agent process outputs
   - Store logs in a structured format with timestamps and process identifiers

2. Improve the frontend terminal to display these logs:
   - Add a toggle in the terminal UI to switch between interactive mode and log view mode
   - Implement log streaming via WebSockets for real-time updates
   - Add filtering options to focus on specific types of logs (stdout/stderr)

**Implementation Details:**
- Create a new log relay service that captures all process output
- Modify the WebSocket implementation to include message type indicators (stdout/stderr)
- Add a log buffer to ensure no messages are lost during connection interruptions

## 2. WASM Discovery Function Issue

**Issue:** The wasm.meta file is confusing the WASM Discovery Function, causing it to load two different instances of the agent on the frontend.

**Fix Strategy:**
1. Modify the agent discovery implementation in `agent/core/agent_discovery.go`:
   - Update the WASM discovery logic to properly handle meta files
   - Implement deduplication logic to prevent duplicate agent instances
   - Add a unique identifier check to ensure each agent is only loaded once

2. Update the frontend agent loading logic:
   - Add a check to prevent loading duplicate agents with the same underlying WASM module
   - Implement a more robust agent identification system

**Implementation Details:**
```go
// In agent_discovery.go
// Add a check to prevent duplicate WASM agents
func (d *AgentDiscoveryImpl) DiscoverFromWASM(ctx context.Context, wasmDir string) ([]*UnifiedAgent, error) {
    // Existing code...
    
    // Add a map to track unique agents by their actual binary content
    uniqueAgents := make(map[string]*UnifiedAgent)
    
    // When processing WASM files, check if we've already seen this agent
    // Use a content hash or other unique identifier
    
    // Only add unique agents to the result list
    
    return agents, nil
}
```

## 3. Meta Files Storage Location

**Issue:** Need to save meta files in the /plugins/data subdirectory.

**Fix Strategy:**
1. Create a dedicated directory structure for meta files:
   - Create the `/plugins/data` directory if it doesn't exist
   - Update all code that generates or reads meta files to use this location

2. Modify the metadata creation process in the agent builder:
   - Update the `createPluginMetadata` method in `agent_builder.go` to use the new location
   - Ensure the metadata includes all agent configuration settings

3. Update metadata reading functions to check the new location:
   - Modify `GetPluginMetadata` to look in the new directory first
   - Add backward compatibility to still find metadata in old locations if needed

**Implementation Details:**
```go
// Add a constant for the meta files directory
const MetaFilesDirectory = "/plugins/data"

// Ensure the directory exists
func ensureMetaDirectoryExists() error {
    return os.MkdirAll(MetaFilesDirectory, 0755)
}

// Update createPluginMetadata to use the new directory
func (b *AgentBuilder) createPluginMetadata(agentID, pluginPath string) error {
    // Get plugin file info
    pluginInfo, err := os.Stat(pluginPath)
    if err != nil {
        return fmt.Errorf("failed to stat plugin file: %v", err)
    }

    // Get agent config for additional metadata
    config, err := b.GetAgent(agentID)
    if err != nil {
        return fmt.Errorf("failed to get agent config: %v", err)
    }

    // Create metadata structure with all configuration settings
    metadata := map[string]interface{}{
        "agent_id":        agentID,
        "agent_name":      config.Name,
        "version":         "1.0", // Default version
        "created_at":      time.Now().Format(time.RFC3339),
        "file_size":       pluginInfo.Size(),
        "file_mode":       pluginInfo.Mode().String(),
        "builder_version": "1.0",
        "build_timestamp": time.Now().Unix(),
        // Add all configuration settings
        "capabilities":    config.ExtraParams["capabilities"],
        "target_types":    config.ExtraParams["target_types"],
        "api_keys":        config.ExtraParams["api_keys"],
        "status":          "idle", // Ensure new agents start with idle status
    }

    // Add version from config if available
    if version, ok := config.ExtraParams["version"].(string); ok && version != "" {
        metadata["version"] = version
    }

    // Ensure the meta directory exists
    ensureMetaDirectoryExists()

    // Create metadata file path in the new location
    metadataPath := filepath.Join(MetaFilesDirectory, filepath.Base(pluginPath) + ".meta")

    // Marshal metadata to JSON
    metadataJSON, err := json.MarshalIndent(metadata, "", "  ")
    if err != nil {
        return fmt.Errorf("failed to marshal metadata: %v", err)
    }

    // Write metadata file
    if err := os.WriteFile(metadataPath, metadataJSON, 0644); err != nil {
        return fmt.Errorf("failed to write metadata file: %v", err)
    }

    return nil
}
```

## 4. Agent Deployment Button State

**Issue:** When an agent is first created, the "Stop" button is incorrectly enabled as if the agent has already been deployed. It should say "Deploy" instead.

**Fix Strategy:**
1. Fix the initial agent status in the agent creation process:
   - Ensure all newly created agents have `status: "idle"` instead of any other status
   - Update the `AgentManager.jsx` component to correctly handle the initial state

2. Improve the button state logic:
   - Modify the conditional rendering in `AgentManager.jsx` to correctly show "Deploy" for new agents
   - Add a clear status check to determine which button to show

**Implementation Details:**
```javascript
// In AgentManager.jsx
// Update the button rendering logic
{agent.status === 'idle' || agent.status === 'inactive' ? (
  <button
    onClick={(event) => {
      event.stopPropagation();
      handleDeployClick(agent);
    }}
    className="flex-1 bg-gradient-to-r from-green-500/20 to-emerald-500/30 hover:from-green-500/30 hover:to-emerald-500/40 border border-green-500/30 text-white py-2 px-4 rounded-lg transition-all duration-200 flex items-center justify-center space-x-2"
  >
    <Play className="w-4 h-4" />
    <span>Deploy</span>
  </button>
) : (
  <button
    onClick={(event) => {
      event.stopPropagation();
      handleStopAgent(agent);
    }}
    className="flex-1 bg-gradient-to-r from-red-500/20 to-orange-500/20 hover:from-red-500/30 hover:to-orange-500/30 border border-red-500/30 text-white py-2 px-4 rounded-lg transition-all duration-200 flex items-center justify-center space-x-2"
  >
    <Square className="w-4 h-4" />
    <span>Stop</span>
  </button>
)}
```

## 5. Agent Configuration Persistence

**Issue:** The capability, target systems, and API Key settings configured during creation do not persist through the compile process, requiring reset before deployment.

**Fix Strategy:**
1. Ensure configuration persistence in the agent creation process:
   - Modify the agent creation flow to properly store all configuration settings
   - Update the agent compilation process to preserve these settings
   - Store configuration in both the agent object and the metadata file

2. Implement proper configuration transfer between creation and deployment:
   - Ensure the `AgentDeploymentModal` correctly loads the agent's original configuration
   - Add a configuration validation step before deployment to verify all settings are present
   - Implement a fallback mechanism to load settings from metadata if not in agent object

3. Add configuration persistence to the backend:
   - Update the agent storage mechanism to properly store and retrieve all configuration settings
   - Implement a configuration versioning system to track changes
   - Ensure agent templates include configuration settings

4. Modify the agent builder to include user configurations in compiled plugins:
   - Update the `buildPluginFromTemplates` method to include user-configured settings
   - Ensure template processing preserves all configuration parameters
   - Add configuration settings to the template data used during compilation

**Implementation Details:**
```javascript
// In AgentCreationModal.jsx
// Ensure all configuration is properly stored in the agent object
const configObj = {
  collection: formData.collection,
  image_url: formData.imageURL,
  capabilities: formData.capabilities, // Store the full capabilities array
  target_types: formData.targetTypes, // Store the full target types array
  api_keys: formData.apiKeys, // Store all API keys
  status: 'idle',
  plugin_info: selectedAgent.plugin_info,
  build_target: selectedAgent.type || 'plugin',
  source_agent_id: selectedAgent.id
};
```

```javascript
// In AgentDeploymentModal.jsx
// Ensure we load the agent's original configuration
useEffect(() => {
  if (isOpen && agent) {
    // Load the agent's capabilities and target types from its configuration
    if (agent.capabilities) {
      // Use the agent's stored capabilities
      setSelectedCapabilities(agent.capabilities);
    } else {
      // Try to load from metadata as fallback
      loadCapabilitiesFromMetadata(agent.id);
    }
    
    if (agent.targetTypes) {
      // Use the agent's stored target types
      setSelectedTargetTypes(agent.targetTypes);
    } else {
      // Try to load from metadata as fallback
      loadTargetTypesFromMetadata(agent.id);
    }
    
    // Load API keys if available
    if (agent.apiKeys) {
      setApiKeys(agent.apiKeys);
    } else {
      // Try to load from metadata as fallback
      loadApiKeysFromMetadata(agent.id);
    }
  }
}, [isOpen, agent]);

// Function to load capabilities from metadata
const loadCapabilitiesFromMetadata = async (agentId) => {
  try {
    const metadata = await fetchAgentMetadata(agentId);
    if (metadata && metadata.capabilities) {
      setSelectedCapabilities(metadata.capabilities);
    }
  } catch (error) {
    console.error('Failed to load capabilities from metadata:', error);
  }
};
```

```go
// In agent_builder.go
// Update createTemplateData to include user configuration
func (b *AgentBuilder) createTemplateData(config AgentConfig) map[string]interface{} {
    // Existing code...
    
    // Add user configuration to template data
    templateData := map[string]interface{}{
        "AgentId":          config.AgentID,
        "AgentName":        config.Name,
        "AgentDescription": config.Description,
        "AgentVersion":     agentVersion,
        "AgentType":        config.AgentType,
        "Model":            config.Model,
        "Instruction":      config.Instruction,
        // Add user configuration
        "Capabilities":     config.ExtraParams["capabilities"],
        "TargetTypes":      config.ExtraParams["target_types"],
        "ApiKeys":          config.ExtraParams["api_keys"],
        // Other template data...
    }
    
    return templateData
}
```

```go
// In agent_builder.go
// Update buildPluginFromTemplates to include configuration in compiled plugin
func (b *AgentBuilder) buildPluginFromTemplates(config AgentConfig) (string, error) {
    // Create template data with all user configuration
    templateData := b.createTemplateData(config)
    
    // Create a temporary directory for building the plugin
    tempDir, err := os.MkdirTemp("", "agent_build_"+config.AgentID)
    if err != nil {
        return "", fmt.Errorf("failed to create temp directory: %v", err)
    }
    defer os.RemoveAll(tempDir)
    
    // Process all template files
    if err := b.processTemplateFiles(templateData, tempDir); err != nil {
        return "", fmt.Errorf("failed to process template files: %v", err)
    }
    
    // Create a config.json file in the build directory with all user settings
    configFilePath := filepath.Join(tempDir, "config.json")
    configData := map[string]interface{}{
        "agent_id":      config.AgentID,
        "name":          config.Name,
        "description":   config.Description,
        "capabilities":  config.ExtraParams["capabilities"],
        "target_types":  config.ExtraParams["target_types"],
        "api_keys":      config.ExtraParams["api_keys"],
        "status":        "idle",
        "build_target":  config.BuildTarget,
    }
    
    configJSON, err := json.MarshalIndent(configData, "", "  ")
    if err != nil {
        return "", fmt.Errorf("failed to marshal config data: %v", err)
    }
    
    if err := os.WriteFile(configFilePath, configJSON, 0644); err != nil {
        return "", fmt.Errorf("failed to write config file: %v", err)
    }
    
    // Build the plugin using the compilePluginWithConfig method
    pluginPath, err := b.compilePluginWithConfig(config, tempDir)
    if err != nil {
        return "", fmt.Errorf("failed to compile plugin: %v", err)
    }
    
    return pluginPath, nil
}
```

## Implementation Plan

1. **Terminal Width Adjustment**
   - Priority: High
   - Estimated time: 1 hour
   - Files to modify:
     - `/gui/src/components/AgentTerminal.jsx`

2. **Process Output Visibility**
   - Priority: Medium
   - Estimated time: 4-6 hours
   - Files to modify:
     - `/agentify/terminal.go`
     - `/gui/src/components/AgentTerminal.jsx`
     - Create new log relay service

3. **WASM Discovery Function Fix**
   - Priority: High
   - Estimated time: 2-3 hours
   - Files to modify:
     - `/agent/core/agent_discovery.go`
     - `/scripts/test_wasm_discovery.js` (for testing)

4. **Meta Files Storage Location**
   - Priority: Medium
   - Estimated time: 2-3 hours
   - Files to modify:
     - Create `/plugins/data` directory
     - Update `/agent/agent_builder.go` (createPluginMetadata method)
     - Update `/agent/agent_builder.go` (GetPluginMetadata method)
     - Add migration script for existing meta files

5. **Agent Deployment Button State**
   - Priority: High
   - Estimated time: 1 hour
   - Files to modify:
     - `/gui/src/components/AgentManager.jsx`
     - `/gui/src/components/modals/AgentCreationModal.jsx` (ensure status is 'idle')

6. **Agent Configuration Persistence**
   - Priority: High
   - Estimated time: 4-5 hours
   - Files to modify:
     - `/gui/src/components/modals/AgentCreationModal.jsx`
     - `/gui/src/components/modals/AgentDeploymentModal.jsx`
     - `/agent/agent_builder.go` (createTemplateData method)
     - `/agent/agent_builder.go` (buildPluginFromTemplates method)
     - Add API endpoint to fetch agent metadata

## Testing Strategy

1. **Terminal Width Testing**
   - Verify terminal displays correctly with 37 columns
   - Test with various content types to ensure no text bleeding
   - Test on different screen sizes and resolutions
   - Verify terminal width setting is applied to both new and existing terminals

2. **Process Output Testing**
   - Verify all stdout and stderr messages are captured
   - Test with high-volume output to ensure no messages are lost
   - Verify real-time updates in the UI
   - Test with multiple concurrent agent processes

3. **WASM Discovery Testing**
   - Verify only one instance of each agent is loaded
   - Test with multiple WASM files and meta files
   - Verify correct agent identification
   - Test with agents that have similar names but different content

4. **Meta Files Storage Testing**
   - Verify meta files are correctly saved in the new location
   - Test migration of existing meta files
   - Verify all operations correctly use the new location
   - Test backward compatibility with meta files in old locations
   - Verify metadata contains all required configuration settings

5. **Deployment Button Testing**
   - Verify new agents show "Deploy" button
   - Test state transitions between deploy and stop
   - Verify correct button state after page refresh
   - Test with agents created through different methods (templates, WASM, plugins)

6. **Configuration Persistence Testing**
   - Verify all settings persist through the compile process
   - Test with various configuration combinations
   - Verify settings are correctly loaded in the deployment modal
   - Test fallback to metadata when agent object doesn't have settings
   - Verify template-based agents correctly incorporate user settings
   - Test that API keys and other sensitive settings are properly preserved
   - Verify configuration is preserved when agent is recompiled or updated

## Conclusion

This fix strategy addresses all the identified issues in a systematic way. By implementing these changes, we will improve the user experience, fix the agent terminal display, ensure proper WASM agent discovery, and maintain configuration persistence throughout the agent lifecycle.

The agent configuration persistence issue is particularly important as it involves multiple components working together: the frontend agent creation and deployment modals, the backend agent builder, the template system, and the metadata storage. Our solution ensures that user-configured settings are preserved at every step of the process:

1. During agent creation, all settings are stored in the agent object
2. During compilation, settings are included in both the template data and a dedicated config.json file
3. After compilation, settings are stored in the metadata file in the new /plugins/data directory
4. During deployment, settings are loaded from the agent object with fallback to metadata if needed

The most critical issues (terminal width, deployment button state, and configuration persistence) should be addressed first, followed by the meta files storage location and process output visibility enhancements.

By implementing these fixes, we'll create a more seamless and reliable experience for users, eliminating the need to reconfigure agents between creation and deployment, and ensuring that all agent settings are properly preserved throughout the agent lifecycle.

---

<div class="footer-links">


© 2025 KNIRV Network
</div>
