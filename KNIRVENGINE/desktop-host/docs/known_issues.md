# Known Issues and Troubleshooting Guide

This document provides a comprehensive list of known issues that you might encounter while using the KNIRVENGINE, along with detailed troubleshooting steps to resolve them.

## Table of Contents

1. [Installation Issues](#installation-issues)
2. [Authentication Issues](#authentication-issues)
3. [Network Connectivity Issues](#network-connectivity-issues)
4. [AI Model Integration Issues](#ai-model-integration-issues)
5. [Agent Deployment Issues](#agent-deployment-issues)
6. [Terminal and WebSocket Issues](#terminal-and-websocket-issues)
7. [Performance Issues](#performance-issues)
8. [Plugin and Extension Issues](#plugin-and-extension-issues)
9. [Database Issues](#database-issues)
10. [System-Specific Issues](#system-specific-issues)

---

## Installation Issues

### Issue: Failed to build the application

**Symptoms:**
- Error messages during the build process
- Incomplete installation
- Missing executable files

**Troubleshooting Steps:**
1. Ensure you have the correct Go version (1.21+) installed:
   ```bash
   go version
   ```
2. Verify Node.js (16+) and npm are properly installed:
   ```bash
   node -v
   npm -v
   ```
3. Check for system dependencies:
   ```bash
   # On Ubuntu/Debian
   apt-get update && apt-get install -y build-essential
   
   # On macOS
   xcode-select --install
   ```
4. Clear any previous build artifacts:
   ```bash
   rm -rf dist build node_modules
   ```
5. Reinstall dependencies and rebuild:
   ```bash
   npm install
   go build -o knirv-engine
   ```

### Issue: Port conflicts during startup

**Symptoms:**
- Error messages about ports already in use
- Application fails to start

**Troubleshooting Steps:**
1. Check if the default ports (8081 for API, 8080 for GUI) are already in use:
   ```bash
   # On Linux/macOS
   lsof -i :8081
   lsof -i :8080
   
   # On Windows
   netstat -ano | findstr :8081
   netstat -ano | findstr :8080
   ```
2. Edit the `ports.config` file to use different ports
3. Run `./sync-env.sh` to update the configuration
4. Restart the application

---

## Authentication Issues

### Issue: Unable to log in

**Symptoms:**
- Login attempts fail
- "Authentication required" errors
- JWT token errors

**Troubleshooting Steps:**
1. Verify your credentials are correct
2. Check if the JWT_SECRET is properly set in your `.env` file
3. Clear browser cookies and cache
4. Ensure the database is properly initialized:
   ```bash
   # Check if the database exists
   ls -la *.db
   ```
5. If using a custom authentication provider, verify its configuration
6. Check server logs for authentication errors:
   ```bash
   grep "auth" *.log
   ```

### Issue: Session expires too quickly

**Symptoms:**
- Frequent logouts
- "Session expired" messages
- Need to re-authenticate often

**Troubleshooting Steps:**
1. Check the JWT token expiration settings in your configuration
2. Ensure your system clock is synchronized correctly
3. Verify there are no network issues causing token validation failures
4. Increase the token expiration time in the server configuration

---

## Network Connectivity Issues

### Issue: API connection failures

**Symptoms:**
- "Network connection failed" errors
- Timeout errors when making API requests
- WebSocket disconnections

**Troubleshooting Steps:**
1. Verify your internet connection is stable
2. Check if the server is running:
   ```bash
   curl http://localhost:8081/api/v1/health
   ```
3. Ensure firewall settings allow connections to the required ports
4. Check for any proxy settings that might interfere with connections
5. Verify the API URL is correctly configured in the frontend
6. Look for CORS issues in the browser console (for web interface)

### Issue: WebSocket connection failures

**Symptoms:**
- Terminal sessions disconnect
- Real-time updates not working
- "WebSocket connection failed" errors

**Troubleshooting Steps:**
1. Check if the WebSocket server is running:
   ```bash
   curl -I -N -H "Connection: Upgrade" -H "Upgrade: websocket" \
     -H "Sec-WebSocket-Version: 13" \
     -H "Sec-WebSocket-Key: dummy" http://localhost:8081/api/v1/ws
   ```
2. Verify your network allows WebSocket connections (some corporate networks block them)
3. Check for any proxy settings that might interfere with WebSocket connections
4. Restart the application to reinitialize WebSocket connections
5. Check browser console for specific WebSocket errors

---

## AI Model Integration Issues

### Issue: AI provider API key errors

**Symptoms:**
- "Invalid API key" errors
- Failed inference requests
- Models not loading

**Troubleshooting Steps:**
1. Verify your API keys are correctly set in the `.env` file:
   ```
   CEREBRAS_API_KEY=your_cerebras_api_key_here
   GEMINI_API_KEY=your_gemini_api_key_here
   DEEPSEEK_API_KEY=your_deepseek_api_key_here
   ```
2. Check if your API keys have expired or reached usage limits
3. Ensure the API keys have the necessary permissions
4. Verify the API endpoints are correctly configured
5. Check if the AI provider services are operational

### Issue: Model inference timeouts

**Symptoms:**
- Inference requests take too long
- Timeout errors during model execution
- Incomplete responses from AI models

**Troubleshooting Steps:**
1. Check your internet connection speed and stability
2. Verify the AI provider service status
3. Reduce the complexity or length of your prompts
4. Adjust the timeout settings in the configuration
5. Try using a different AI provider as a fallback
6. Check if the input exceeds token limits and requires chunking

---

## Agent Deployment Issues

### Issue: Agent fails to deploy

**Symptoms:**
- "Failed to deploy agent" errors
- Agent status remains "idle" after deployment attempt
- Error messages in the agent deployment logs

**Troubleshooting Steps:**
1. Check if the agent configuration is valid
2. Verify all required capabilities are available
3. Ensure the target system is properly configured
4. Check for any conflicts with existing deployed agents
5. Verify the agent has the necessary permissions
6. Check server logs for specific deployment errors:
   ```bash
   grep "agent deployment" *.log
   ```

### Issue: Agent becomes unresponsive

**Symptoms:**
- Agent stops responding to requests
- Terminal sessions disconnect
- Agent status shows as "active" but doesn't process tasks

**Troubleshooting Steps:**
1. Check the agent's resource usage (CPU, memory)
2. Verify the agent's connection to required services
3. Check for any error messages in the agent logs
4. Restart the agent:
   - Stop the agent from the UI
   - Wait a few seconds
   - Redeploy the agent
5. If the issue persists, try recreating the agent from its template

---

## Terminal and WebSocket Issues

### Issue: Terminal sessions disconnect

**Symptoms:**
- Terminal sessions close unexpectedly
- "Connection error" messages in terminal
- Unable to interact with agent through terminal

**Troubleshooting Steps:**
1. Check your network connection stability
2. Verify WebSocket connections are allowed on your network
3. Check for any proxy settings that might interfere
4. Restart the terminal session
5. Check browser console for specific WebSocket errors
6. Verify the terminal service is running properly:
   ```bash
   curl http://localhost:8081/api/v1/terminal/status
   ```

### Issue: Terminal input/output delays

**Symptoms:**
- Slow response times in terminal
- Characters appear with delay when typing
- Terminal output appears in chunks

**Troubleshooting Steps:**
1. Check your network latency to the server
2. Verify system resource usage (high CPU or memory usage can cause delays)
3. Close other terminal sessions that might be consuming resources
4. Restart the browser or application
5. Try using a different browser or device

---

## Performance Issues

### Issue: High CPU/memory usage

**Symptoms:**
- System becomes slow or unresponsive
- High CPU or memory usage reported by system monitor
- Application crashes due to resource exhaustion

**Troubleshooting Steps:**
1. Check which processes are consuming resources:
   ```bash
   # On Linux/macOS
   top
   
   # On Windows
   Task Manager
   ```
2. Limit the number of concurrent agents
3. Close unnecessary terminal sessions
4. Reduce the complexity of agent tasks
5. Restart the application to clear memory
6. Ensure your system meets the recommended requirements:
   - 4+ CPU cores
   - 8GB+ RAM
   - 5GB+ free disk space

### Issue: Slow UI responsiveness

**Symptoms:**
- UI elements take time to load or respond
- Animations stutter
- Delays when navigating between pages

**Troubleshooting Steps:**
1. Check browser console for JavaScript errors
2. Clear browser cache and reload
3. Disable browser extensions that might interfere
4. Reduce the number of open browser tabs
5. Try using a different browser
6. Check if your system meets the recommended requirements

---

## Plugin and Extension Issues

### Issue: Plugin fails to load

**Symptoms:**
- "Failed to load plugin" errors
- Plugin capabilities not available
- Error messages in plugin loading logs

**Troubleshooting Steps:**
1. Verify the plugin file exists and is not corrupted
2. Check if the plugin is compatible with your version of KNIRVENGINE
3. Ensure the plugin has the correct format and structure
4. Check for any dependencies required by the plugin
5. Verify plugin permissions are correctly set
6. Check server logs for specific plugin loading errors:
   ```bash
   grep "plugin" *.log
   ```

### Issue: WASM agent errors

**Symptoms:**
- "WASM execution failed" errors
- Agent capabilities not working
- Error messages in WASM execution logs

**Troubleshooting Steps:**
1. Verify the WASM module is correctly compiled
2. Check if the WASM module is compatible with your browser
3. Ensure all required imports are available
4. Check for any memory limitations affecting WASM execution
5. Verify the WASM module has the necessary permissions
6. Try using a different browser that has better WASM support

---

## Database Issues

### Issue: Database connection errors

**Symptoms:**
- "Failed to connect to database" errors
- Data not being saved or retrieved
- Application crashes with database-related errors

**Troubleshooting Steps:**
1. Check if the database file exists and is not corrupted:
   ```bash
   ls -la *.db
   ```
2. Verify database permissions allow read/write access
3. Check for disk space issues that might prevent database growth
4. Backup and recreate the database if necessary:
   ```bash
   cp your_database.db your_database.db.backup
   rm your_database.db
   # Restart the application to recreate the database
   ```
5. Check server logs for specific database errors:
   ```bash
   grep "database" *.log
   ```

### Issue: Data inconsistency

**Symptoms:**
- Missing or incorrect data
- Agents or workflows not appearing correctly
- Unexpected behavior due to data issues

**Troubleshooting Steps:**
1. Check for database integrity issues
2. Verify all migrations have been applied correctly
3. Clear browser cache to ensure you're seeing the latest data
4. Check for any concurrent access issues
5. If necessary, restore from a backup or reset the database

---

## System-Specific Issues

### Issue: Windows-specific installation problems

**Symptoms:**
- Path-related errors on Windows
- DLL loading failures
- Windows Defender blocking execution

**Troubleshooting Steps:**
1. Run the application as Administrator
2. Add exceptions in Windows Defender or antivirus software
3. Check for any path length limitations (Windows has a 260 character path limit)
4. Ensure all required DLLs are in the system PATH
5. Use forward slashes (/) instead of backslashes (\\) in configuration paths

### Issue: macOS security restrictions

**Symptoms:**
- "App cannot be opened" warnings
- Permission denied errors
- Gatekeeper blocking execution

**Troubleshooting Steps:**
1. Right-click the application and select "Open" instead of double-clicking
2. Go to System Preferences > Security & Privacy and allow the application
3. If using a downloaded binary, remove quarantine attributes:
   ```bash
   xattr -d com.apple.quarantine ./knirv-engine
   ```
4. Ensure the application has the necessary permissions (Disk, Network, etc.)
5. Check if macOS is blocking any required ports

### Issue: Linux permission problems

**Symptoms:**
- "Permission denied" errors
- Unable to access system resources
- Socket or port binding failures

**Troubleshooting Steps:**
1. Check file permissions:
   ```bash
   ls -la ./knirv-engine
   ```
2. Ensure the executable has proper permissions:
   ```bash
   chmod +x ./knirv-engine
   ```
3. Check if you need elevated privileges for certain operations:
   ```bash
   sudo ./knirv-engine
   ```
4. Verify SELinux or AppArmor settings if applicable
5. Check if your user has the necessary group memberships

---

## Using the AI Error Inference Engine

The KNIRVENGINE includes an AI-powered error analysis system that can help diagnose and fix issues automatically. When you encounter an error:

1. Look for the notification bell in the top-right corner of the interface
2. Click on the bell to open the Error Analysis Assistant
3. Select the error you want to analyze
4. Click "Analyze Error" to get AI-powered diagnostics
5. Follow the suggested steps to resolve the issue
6. Use the chat interface to ask follow-up questions about the error

This system can provide detailed insights into complex errors and often suggests automated fixes.

---

If you encounter an issue not covered in this guide, please report it through the support channels or create an issue in the project repository.