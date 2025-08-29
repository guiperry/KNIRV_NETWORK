#!/usr/bin/env node

// Test script to verify WASM agent discovery is working
const fetch = require('node-modules/node-fetch');

async function testAgentDiscovery() {
  try {
    console.log('🔍 Testing Agent Discovery...');
    console.log('================================');
    
    const response = await fetch('http://localhost:8081/api/v1/adk/agents');
    
    if (!response.ok) {
      throw new Error(`HTTP ${response.status}: ${response.statusText}`);
    }
    
    const data = await response.json();
    const agents = data.agents || [];
    
    console.log(`\n📊 Total agents discovered: ${agents.length}`);
    console.log('\n🤖 Agent List:');
    console.log('================');
    
    // Categorize agents
    const pluginAgents = [];
    const wasmAgents = [];
    
    agents.forEach(agentId => {
      // Check if it's likely a WASM agent based on naming patterns
      const isWasm = agentId.includes('wasm') || 
                     agentId.includes('test_unified_storage') ||
                     agentId.includes('frontend_wasm_test') ||
                     agentId.includes('mytest');
      
      if (isWasm) {
        wasmAgents.push(agentId);
      } else {
        pluginAgents.push(agentId);
      }
    });
    
    console.log(`\n🔧 Plugin Agents (${pluginAgents.length}):`);
    pluginAgents.forEach((agent, index) => {
      console.log(`  ${index + 1}. ${agent}`);
    });
    
    console.log(`\n🌐 WASM Agents (${wasmAgents.length}):`);
    wasmAgents.forEach((agent, index) => {
      console.log(`  ${index + 1}. ${agent}`);
    });
    
    console.log('\n✅ Agent discovery test completed successfully!');
    
    if (wasmAgents.length > 0) {
      console.log('\n🎉 WASM agents are now being discovered correctly!');
    } else {
      console.log('\n⚠️  No WASM agents found in discovery results.');
    }
    
  } catch (error) {
    console.error('❌ Error testing agent discovery:', error.message);
    process.exit(1);
  }
}

// Run the test
testAgentDiscovery();
