// agent-tunnel-registry/server.js
const express = require('express');
const http = require('http');
const config = require('./config');
const apiRouter = require('./api'); // Main API router
const controlListener = require('./tunneling/controlListener');
const publicRelayListener = require('./tunneling/publicRelayListener');
const tunnelManager = require('./tunneling/tunnelManager');
const StunServer = require('./stun/stunServer');

// Initialize the connection registry for tracking active connections
// This will be used by the control listener, status page, and pruning function
global.connectionRegistry = {};

const app = express();
app.use(express.json()); // Middleware to parse JSON bodies

// Mount the API router
app.use('/api', apiRouter);

// GET /status
// Shows the status of the relay service, including active connections and system metrics
// Returns HTML for browsers and JSON for API clients
app.get('/status', (req, res) => {
  // Count active connections
  const now = Date.now();
  const activeConnections = Object.entries(connectionRegistry || {}).filter(([_, info]) => 
    now - info.lastSeen < 60 * 60 * 1000 // Active in the last hour
  );
  
  // Calculate uptime
  const uptimeMs = process.uptime() * 1000;
  const uptimeDays = Math.floor(uptimeMs / (1000 * 60 * 60 * 24));
  const uptimeHours = Math.floor((uptimeMs % (1000 * 60 * 60 * 24)) / (1000 * 60 * 60));
  const uptimeMinutes = Math.floor((uptimeMs % (1000 * 60 * 60)) / (1000 * 60));
  const uptimeSeconds = Math.floor((uptimeMs % (1000 * 60)) / 1000);
  const uptimeString = `${uptimeDays}d ${uptimeHours}h ${uptimeMinutes}m ${uptimeSeconds}s`;
  
  // Prepare status data
  const statusData = {
    status: 'online',
    version: '1.0.0',
    uptime: uptimeString,
    uptimeMs: uptimeMs,
    registeredConnections: Object.keys(connectionRegistry || {}).length,
    activeConnections: activeConnections.length,
    memoryUsage: process.memoryUsage(),
    httpApiPort: config.httpApiPort,
    controlListenerPort: config.controlListenerPort,
    publicRelayPort: config.publicRelayPort,
    timestamp: new Date().toISOString()
  };
  
  // Check if request is from a browser
  const acceptHeader = req.headers.accept || '';
  const isHtmlRequest = acceptHeader.includes('text/html');
  
  if (isHtmlRequest) {
    // Return HTML for browsers
    const connectionRows = Object.entries(connectionRegistry || {}).map(([connectionID, info]) => {
      const lastSeenDate = new Date(info.lastSeen);
      const isActive = now - info.lastSeen < 60 * 60 * 1000; // Active in the last hour
      const statusClass = isActive ? 'active' : 'inactive';
      
      return `
        <tr>
          <td class="connectionid">${connectionID}</td>
          <td>${info.type || 'N/A'}</td>
          <td>${info.sourceIP}</td>
          <td>${info.sourcePort}</td>
          <td class="${statusClass}">${isActive ? 'Active' : 'Inactive'}</td>
          <td>${lastSeenDate.toLocaleString()}</td>
        </tr>
      `;
    }).join('');
    
    const html = `
      <!DOCTYPE html>
      <html lang="en">
      <head>
        <meta charset="UTF-8">
        <meta name="viewport" content="width=device-width, initial-scale=1.0">
        <title>KNIRVROOT Central Relay Status</title>
        <style>
          body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
            line-height: 1.6;
            color: #333;
            max-width: 1200px;
            margin: 0 auto;
            padding: 20px;
          }
          h1 {
            color: #2c3e50;
            border-bottom: 2px solid #eee;
            padding-bottom: 10px;
          }
          .status-card {
            background: #f8f9fa;
            border-radius: 8px;
            padding: 20px;
            margin-bottom: 20px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
          }
          .status-grid {
            display: grid;
            grid-template-columns: repeat(auto-fill, minmax(250px, 1fr));
            gap: 15px;
          }
          .status-item {
            background: white;
            padding: 15px;
            border-radius: 6px;
            box-shadow: 0 1px 3px rgba(0,0,0,0.08);
          }
          .status-label {
            font-weight: bold;
            color: #6c757d;
            margin-bottom: 5px;
            font-size: 0.9rem;
          }
          .status-value {
            font-size: 1.1rem;
            color: #212529;
          }
          table {
            width: 100%;
            border-collapse: collapse;
            margin-top: 20px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
          }
          th, td {
            padding: 12px 15px;
            text-align: left;
            border-bottom: 1px solid #e9ecef;
          }
          th {
            background-color: #f8f9fa;
            font-weight: bold;
            color: #495057;
          }
          tr:hover {
            background-color: #f8f9fa;
          }
          .active {
            color: #28a745;
            font-weight: bold;
          }
          .inactive {
            color: #dc3545;
          }
          .connectionid {
            font-family: monospace;
            word-break: break-all;
          }
          .refresh-button {
            background-color: #007bff;
            color: white;
            border: none;
            padding: 8px 16px;
            border-radius: 4px;
            cursor: pointer;
            font-size: 14px;
            margin-bottom: 20px;
          }
          .refresh-button:hover {
            background-color: #0069d9;
          }
          .memory-usage {
            font-size: 0.9rem;
          }
          @media (max-width: 768px) {
            .status-grid {
              grid-template-columns: 1fr;
            }
            table {
              font-size: 0.9rem;
            }
            th, td {
              padding: 8px 10px;
            }
          }
        </style>
      </head>
      <body>
        <h1>KNIRVROOT Central Relay Status</h1>
        
        <button class="refresh-button" onclick="window.location.reload()">Refresh Status</button>
        
        <div class="status-card">
          <div class="status-grid">
            <div class="status-item">
              <div class="status-label">Status</div>
              <div class="status-value">${statusData.status}</div>
            </div>
            <div class="status-item">
              <div class="status-label">Version</div>
              <div class="status-value">${statusData.version}</div>
            </div>
            <div class="status-item">
              <div class="status-label">Uptime</div>
              <div class="status-value">${statusData.uptime}</div>
            </div>
            <div class="status-item">
              <div class="status-label">Registered Connections</div>
              <div class="status-value">${statusData.registeredConnections}</div>
            </div>
            <div class="status-item">
              <div class="status-label">Active Connections</div>
              <div class="status-value">${statusData.activeConnections}</div>
            </div>
            <div class="status-item">
              <div class="status-label">HTTP API Port</div>
              <div class="status-value">${statusData.httpApiPort}</div>
            </div>
            <div class="status-item">
              <div class="status-label">Control Listener Port</div>
              <div class="status-value">${statusData.controlListenerPort}</div>
            </div>
            <div class="status-item">
              <div class="status-label">Public Relay Port</div>
              <div class="status-value">${statusData.publicRelayPort}</div>
            </div>
            <div class="status-item">
              <div class="status-label">Last Updated</div>
              <div class="status-value">${new Date().toLocaleString()}</div>
            </div>
          </div>
          
          <div class="status-item memory-usage">
            <div class="status-label">Memory Usage</div>
            <div class="status-value">
              RSS: ${Math.round(statusData.memoryUsage.rss / 1024 / 1024)} MB | 
              Heap Total: ${Math.round(statusData.memoryUsage.heapTotal / 1024 / 1024)} MB | 
              Heap Used: ${Math.round(statusData.memoryUsage.heapUsed / 1024 / 1024)} MB
            </div>
          </div>
        </div>
        
        <h2>Active Connections</h2>
        <table>
          <thead>
            <tr>
              <th>Connection ID</th>
              <th>Type</th>
              <th>Source IP</th>
              <th>Source Port</th>
              <th>Status</th>
              <th>Last Seen</th>
            </tr>
          </thead>
          <tbody>
            ${connectionRows.length > 0 ? connectionRows : '<tr><td colspan="6" style="text-align: center;">No active connections</td></tr>'}
          </tbody>
        </table>
        
        <script>
          // Auto-refresh the page every 30 seconds
          setTimeout(() => {
            window.location.reload();
          }, 30000);
        </script>
      </body>
      </html>
    `;
    
    res.setHeader('Content-Type', 'text/html');
    res.status(200).send(html);
  } else {
    // Return JSON for API clients
    res.status(200).json({
      ...statusData,
      connections: connectionRegistry || {}
    });
  }
});

// --- Periodic Pruning of Inactive Connections ---
function pruneInactiveConnections() {
    const now = Date.now();
    // Prune connections inactive for more than 1 hour
    const maxInactiveTime = 60 * 60 * 1000; // 1 hour in milliseconds
    let prunedCount = 0;
    let skippedCount = 0;
    let activeCount = 0;

    console.log('[Relay] Running inactive connection pruning...');
    
    // First, identify connections to prune
    const connectionsToPrune = [];
    for (const connectionID in connectionRegistry) {
        const connection = connectionRegistry[connectionID];
        const timeSinceLastSeen = now - connection.lastSeen;
        
        if (timeSinceLastSeen > maxInactiveTime) {
            // Check if this is a critical connection that should be preserved
            if (connection.type === 'bootnode' || connection.type === 'root') {
                console.log(`[Relay] Skipping pruning of critical ${connection.type} connection: ${connectionID} despite inactivity (${Math.round(timeSinceLastSeen/1000/60)} minutes)`);
                skippedCount++;
                
                // Try to ping the connection to see if it's still alive
                if (connection.socket && !connection.socket.destroyed) {
                    try {
                        connection.socket.write(JSON.stringify({ 
                            action: 'PING_CHECK', 
                            timestamp: now 
                        }) + '\n');
                        console.log(`[Relay] Sent PING_CHECK to ${connectionID}`);
                    } catch (err) {
                        console.error(`[Relay] Failed to ping ${connectionID}: ${err.message}`);
                        connectionsToPrune.push(connectionID);
                    }
                } else {
                    console.log(`[Relay] Socket for ${connectionID} is already destroyed or missing`);
                    connectionsToPrune.push(connectionID);
                }
            } else {
                // Regular connection that can be pruned
                connectionsToPrune.push(connectionID);
            }
        } else {
            activeCount++;
        }
    }
    
    // Then, prune the identified connections
    for (const connectionID of connectionsToPrune) {
        const connection = connectionRegistry[connectionID];
        console.log(`[Relay] Pruning inactive ${connection.type} connection: ${connectionID} (Last seen: ${new Date(connection.lastSeen).toISOString()})`);
        
        // Close any active sockets or resources for this connection
        if (connection.socket && typeof connection.socket.destroy === 'function' && !connection.socket.destroyed) {
            connection.socket.destroy();
        }
        
        // Remove from tunnel manager if applicable
        if (tunnelManager && typeof tunnelManager.removeControlSocket === 'function') {
            tunnelManager.removeControlSocket(connectionID, connection.socket ? connection.socket.id : null);
        }
        
        delete connectionRegistry[connectionID];
        prunedCount++;
    }
    
    console.log(`[Relay] Connection status: ${activeCount} active, ${prunedCount} pruned, ${skippedCount} critical connections skipped`);
}

// Run pruning every 15 minutes (adjust interval as needed)
const pruneInterval = 15 * 60 * 1000;
setInterval(pruneInactiveConnections, pruneInterval);

const httpServer = http.createServer(app);
httpServer.listen(config.httpApiPort, () => {
    console.log(`HTTP API server listening on port ${config.httpApiPort}`);
    pruneInactiveConnections(); // Initial prune on startup
});

// Start the control listener for internal nodes
controlListener.start();

// Start the public relay listener for external clients
publicRelayListener.start();

// Start the STUN server for NAT traversal assistance
const stunServer = new StunServer(config.stunPort);
stunServer.start();

console.log(`agent Custom Tunnel & Registry Server started.`);
console.log(`  Public Hostname: ${config.serverPublicHost}`);
console.log(`  Public Relay Port for external clients: ${config.publicRelayPort}`);
console.log(`  Control Port for internal nodes: ${config.controlListenerPort}`);
console.log(`  STUN Server Port: ${config.stunPort}`);

// Graceful shutdown
const signals = { 'SIGINT': 2, 'SIGTERM': 15 };
Object.keys(signals).forEach((signal) => {
    process.on(signal, () => {
        console.log(`\nReceived ${signal}, shutting down gracefully...`);
        httpServer.close(() => {
            console.log('HTTP server closed.');
        });
        if (controlListener.server) controlListener.server.close(() => console.log('Control listener closed.'));
        if (publicRelayListener.server) publicRelayListener.server.close(() => console.log('Public relay listener closed.'));
        if (stunServer.server) stunServer.server.close(() => console.log('STUN server closed.'));
        // Add cleanup for other resources if necessary
        process.exit(128 + signals[signal]);
    });
});