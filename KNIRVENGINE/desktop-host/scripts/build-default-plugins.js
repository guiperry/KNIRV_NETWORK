#!/usr/bin/env node

/**
 * Build Default Plugins Script
 * 
 * This script creates default agent plugins that users can choose from
 * when creating new agents in the Agentic-Engine.
 * 
 * Usage: node scripts/build-default-plugins.js [--server-url=http://localhost:8081]
 */

const fs = require('fs');
const path = require('path');

// Parse command line arguments
const args = process.argv.slice(2);
const serverUrl = args.find(arg => arg.startsWith('--server-url='))?.split('=')[1] || 'http://localhost:8081';

// Default plugin configurations
const defaultPlugins = [
  {
    agentId: 'testagent',
    agent_name: 'testagent',
    agent_description: 'Test agent for basic functionality',
    model: 'deepseek-chat',
    instruction: 'You are a test agent.',
    agent_type: 'llm'
  },
  {
    agentId: 'webagent',
    agent_name: 'webagent',
    agent_description: 'Specialized agent for web scraping and content analysis',
    model: 'deepseek-chat',
    instruction: 'You are a web analysis specialist. You excel at scraping websites, analyzing content, monitoring changes, and extracting valuable insights from web resources.',
    agent_type: 'llm'
  },
  {
    agentId: 'fileagent',
    agent_name: 'fileagent',
    agent_description: 'Expert agent for file operations and document processing',
    model: 'deepseek-chat',
    instruction: 'You are a file processing expert. You specialize in reading, writing, transforming, and analyzing various file formats including documents, spreadsheets, and data files.',
    agent_type: 'llm'
  }
];

// Build a single plugin
async function buildPlugin(pluginConfig) {
  console.log(`🔨 Building plugin: ${pluginConfig.agent_name}`);

  const requestBody = {
    template_id: 'default',
    config: pluginConfig
  };

  console.log(`📤 Request body:`, JSON.stringify(requestBody, null, 2));

  try {
    const response = await fetch(`${serverUrl}/api/v1/agents/${pluginConfig.agentId}/build`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(requestBody)
    });

    if (!response.ok) {
      const errorText = await response.text();
      throw new Error(`HTTP ${response.status}: ${errorText}`);
    }

    const result = await response.json();
    console.log(`✅ Plugin build started: ${result.message}`);
    
    // Wait a moment and check build status
    await new Promise(resolve => setTimeout(resolve, 2000));
    await checkBuildStatus(pluginConfig.agentId);
    
    return result;
  } catch (error) {
    console.error(`❌ Failed to build plugin ${pluginConfig.agent_name}:`, error.message);
    throw error;
  }
}

// Check build status
async function checkBuildStatus(agentId) {
  try {
    const response = await fetch(`${serverUrl}/api/v1/agents/${agentId}/build`);
    
    if (!response.ok) {
      console.warn(`⚠️  Could not check build status for ${agentId}`);
      return;
    }

    const status = await response.json();
    console.log(`📊 Build status for ${agentId}: ${status.status} (${status.progress}%)`);
    
    if (status.message) {
      console.log(`   Message: ${status.message}`);
    }
    
    return status;
  } catch (error) {
    console.warn(`⚠️  Error checking build status for ${agentId}:`, error.message);
  }
}

// List available plugins
async function listPlugins() {
  try {
    const response = await fetch(`${serverUrl}/api/v1/plugins`);
    
    if (!response.ok) {
      console.warn(`⚠️  Could not list plugins`);
      return [];
    }

    const result = await response.json();
    return result.plugins || [];
  } catch (error) {
    console.warn(`⚠️  Error listing plugins:`, error.message);
    return [];
  }
}

// Check if server is running
async function checkServer() {
  try {
    const response = await fetch(`${serverUrl}/api/v1/health`);
    return response.ok;
  } catch (error) {
    return false;
  }
}

// Main function
async function main() {
  console.log('🚀 Building Default Agent Plugins\n');
  console.log(`📡 Server URL: ${serverUrl}\n`);

  // Check if server is running
  console.log('🔍 Checking server status...');
  const serverRunning = await checkServer();
  
  if (!serverRunning) {
    console.error('❌ Server is not running or not accessible.');
    console.log('   Please start the Agentic-Engine server first:');
    console.log('   cd /path/to/Agentic-Engine && ./knirv-engine');
    process.exit(1);
  }
  
  console.log('✅ Server is running\n');

  // List existing plugins
  console.log('📋 Checking existing plugins...');
  const existingPlugins = await listPlugins();
  console.log(`   Found ${existingPlugins.length} existing plugins\n`);

  // Build each default plugin
  let successCount = 0;
  let failureCount = 0;

  for (const pluginConfig of defaultPlugins) {
    try {
      await buildPlugin(pluginConfig);
      successCount++;
      console.log(''); // Add spacing between plugins
    } catch (error) {
      failureCount++;
      console.log(''); // Add spacing between plugins
    }
  }

  // Summary
  console.log('📊 Build Summary:');
  console.log(`   ✅ Successful builds: ${successCount}`);
  console.log(`   ❌ Failed builds: ${failureCount}`);
  console.log(`   📦 Total plugins attempted: ${defaultPlugins.length}\n`);

  // List final plugins
  console.log('📋 Final plugin list:');
  const finalPlugins = await listPlugins();
  if (finalPlugins.length > 0) {
    finalPlugins.forEach(plugin => {
      console.log(`   • ${plugin.fileName || plugin.agentId}`);
    });
  } else {
    console.log('   No plugins found');
  }

  console.log('\n🎉 Default plugin build process completed!');
  console.log('   Users can now create agents based on these plugins.');
}

// Handle fetch for Node.js
if (typeof fetch === 'undefined') {
  // Try to use node-fetch if available, otherwise provide helpful error
  try {
    const { default: fetch } = require('node-fetch');
    global.fetch = fetch;
  } catch (error) {
    console.error('❌ This script requires fetch support.');
    console.log('   Please install node-fetch: npm install node-fetch');
    console.log('   Or use Node.js 18+ which has built-in fetch support.');
    process.exit(1);
  }
}

// Run the script
if (require.main === module) {
  main().catch(error => {
    console.error('❌ Script failed:', error.message);
    process.exit(1);
  });
}
