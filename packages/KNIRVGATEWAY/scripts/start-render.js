#!/usr/bin/env node

/**
 * Render-specific startup script for KNIRVORACLE
 * 
 * This script provides additional error handling and diagnostics
 * specifically for Render deployment environment.
 */

import { fileURLToPath } from 'url';
import path from 'path';

// ES module compatibility
const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

console.log('[Render] Starting KNIRVORACLE for Render deployment...');
console.log('[Render] Current working directory:', process.cwd());
console.log('[Render] Script directory:', __dirname);

// Set environment variables for Render
process.env.GATEWAY_MODE = 'persistent';
process.env.NODE_ENV = process.env.NODE_ENV || 'production';

console.log('[Render] DHT is disabled by default - use POST /dht/start to enable');

// Log all environment variables for debugging
console.log('[Render] Environment variables:');
Object.keys(process.env)
  .filter(key => key.startsWith('KNIRV') || key.startsWith('GATEWAY') || key.startsWith('NODE') || key.startsWith('PORT') || key.startsWith('DHT') || key.startsWith('DISABLE'))
  .forEach(key => {
    const value = key.includes('KEY') || key.includes('TOKEN') ? 'SET' : process.env[key];
    console.log(`[Render]   ${key}: ${value}`);
  });

// Import and start the main server
try {
  console.log('[Render] Importing server module...');
  const { main } = await import('../server.js');
  
  console.log('[Render] Starting main server function...');
  await main();
  
} catch (error) {
  console.error('[Render] ❌ Failed to start server:', error);
  console.error('[Render] ❌ Error details:', error.message);
  console.error('[Render] ❌ Stack trace:', error.stack);
  
  // Exit with error code to signal deployment failure
  process.exit(1);
}
