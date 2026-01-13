#!/usr/bin/env node

/**
 * Test script for DHT control API endpoints
 * 
 * Usage:
 *   node scripts/test-dht-control.js <command> [url]
 * 
 * Commands:
 *   status  - Check DHT status
 *   start   - Start DHT
 *   stop    - Stop DHT
 *   restart - Restart DHT
 * 
 * Examples:
 *   node scripts/test-dht-control.js status
 *   node scripts/test-dht-control.js start https://your-render-app.onrender.com
 *   node scripts/test-dht-control.js status http://localhost:8080
 */

import axios from 'axios';

const command = process.argv[2];
const baseUrl = process.argv[3] || 'http://localhost:8080';

if (!command) {
  console.log('Usage: node scripts/test-dht-control.js <command> [url]');
  console.log('Commands: status, start, stop, restart');
  process.exit(1);
}

async function checkHealth() {
  try {
    const response = await axios.get(`${baseUrl}/health`);
    console.log('🏥 Health Status:');
    console.log(JSON.stringify(response.data, null, 2));
    return response.data;
  } catch (error) {
    console.error('❌ Health check failed:', error.message);
    return null;
  }
}

async function checkDHTStatus() {
  try {
    const response = await axios.get(`${baseUrl}/dht/status`);
    console.log('📊 DHT Status:');
    console.log(JSON.stringify(response.data, null, 2));
    return response.data;
  } catch (error) {
    console.error('❌ DHT status check failed:', error.message);
    return null;
  }
}

async function startDHT() {
  try {
    console.log('🚀 Starting DHT...');
    const response = await axios.post(`${baseUrl}/dht/start`);
    console.log('✅ DHT Start Response:');
    console.log(JSON.stringify(response.data, null, 2));
    return response.data;
  } catch (error) {
    console.error('❌ DHT start failed:', error.response?.data || error.message);
    return null;
  }
}

async function stopDHT() {
  try {
    console.log('🛑 Stopping DHT...');
    const response = await axios.post(`${baseUrl}/dht/stop`);
    console.log('✅ DHT Stop Response:');
    console.log(JSON.stringify(response.data, null, 2));
    return response.data;
  } catch (error) {
    console.error('❌ DHT stop failed:', error.response?.data || error.message);
    return null;
  }
}

async function restartDHT() {
  try {
    console.log('🔄 Restarting DHT...');
    const response = await axios.get(`${baseUrl}/dht/restart`);
    console.log('✅ DHT Restart Response:');
    console.log(JSON.stringify(response.data, null, 2));
    return response.data;
  } catch (error) {
    console.error('❌ DHT restart failed:', error.response?.data || error.message);
    return null;
  }
}

async function main() {
  console.log(`🌐 Testing DHT control API at: ${baseUrl}`);
  console.log(`📋 Command: ${command}`);
  console.log('');

  // Always check health first
  await checkHealth();
  console.log('');

  switch (command.toLowerCase()) {
    case 'status':
      await checkDHTStatus();
      break;
    
    case 'start':
      await startDHT();
      console.log('');
      console.log('Checking status after start...');
      await checkDHTStatus();
      break;
    
    case 'stop':
      await stopDHT();
      console.log('');
      console.log('Checking status after stop...');
      await checkDHTStatus();
      break;
    
    case 'restart':
      await restartDHT();
      console.log('');
      console.log('Checking status after restart...');
      await checkDHTStatus();
      break;
    
    default:
      console.error('❌ Unknown command:', command);
      console.log('Available commands: status, start, stop, restart');
      process.exit(1);
  }
}

main().catch(error => {
  console.error('❌ Script failed:', error.message);
  process.exit(1);
});
