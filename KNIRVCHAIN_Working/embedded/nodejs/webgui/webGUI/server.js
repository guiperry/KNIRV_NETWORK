#!/usr/bin/env node

/**
 * Server wrapper for Web GUI
 * This wrapper starts the Next.js application for the Web GUI
 */

const { spawn } = require('child_process');
const path = require('path');

// Get port from environment variable or use default
const port = process.env.HTTP_API_PORT || process.env.PORT || 3007;

console.log(`[WebGUI] Starting Web GUI on port ${port}...`);

// Change to the correct directory
process.chdir(__dirname);

// Start the Next.js application
const nextProcess = spawn('npm', ['run', 'start', '--', '-p', port], {
    stdio: 'inherit',
    env: {
        ...process.env,
        PORT: port,
        NODE_ENV: process.env.NODE_ENV || 'production'
    }
});

// Handle process events
nextProcess.on('error', (error) => {
    console.error(`[WebGUI] Failed to start: ${error.message}`);
    process.exit(1);
});

nextProcess.on('close', (code) => {
    console.log(`[WebGUI] Process exited with code ${code}`);
    process.exit(code);
});

// Handle graceful shutdown
process.on('SIGTERM', () => {
    console.log('[WebGUI] Received SIGTERM, shutting down gracefully...');
    nextProcess.kill('SIGTERM');
});

process.on('SIGINT', () => {
    console.log('[WebGUI] Received SIGINT, shutting down gracefully...');
    nextProcess.kill('SIGINT');
});

console.log(`[WebGUI] Agent Notary System server wrapper started`);
