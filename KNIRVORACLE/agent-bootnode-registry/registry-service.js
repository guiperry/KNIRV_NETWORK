// registry-service.js
const express = require('express');
const Turn = require('node-turn'); // TURN/STUN server library
//const fetch = require('node-fetch'); // HTTP client for Node.js
//const dgram = require('dgram'); // UDP socket library
const os = require('os');

const app = express();
const httpPort = 3003; // Registry API port
const stunPort = 3478; // Standard STUN UDP port (used implicitly by clients connecting without a port)

// --- Registry Data Store (In-Memory) ---
// Replace with a persistent store (Redis, DB) for production
const chainRegistry = {};
let bootnodeList = []; // For round-robin
let currentBootnodeIndex = 0;
/*
  Example structure:
  chainRegistry = {
    "KNIRVORACLE-5000": {
      ip: "1.2.3.4",
      port: 5050, // P2P port
      lastSeen: 1678886400000,
      type: "bootnode",
      nodeID: "Qmabcdef12345..."
    }
  }
*/

// --- Middleware ---
app.use(express.json()); // Parse JSON request bodies

// If your registry service runs behind a reverse proxy (like Nginx or a load balancer),
// uncomment the line below to trust the X-Forwarded-For header for req.ip fallback.
// Be sure to configure your proxy correctly to set this header.
// app.set('trust proxy', true);

// --- Registry API Endpoints ---

// GET /stun
// Returns STUN server status and connection details
app.get('/stun', (req, res) => {
  let isUdpListening = false;
  let udpAddress = 'N/A';
  let udpListenPort = stunPort; // Default to stunPort for UDP
  let isTcpListening = false;
  let tcpAddress = 'N/A';
  let tcpListenPort = stunPort + 1; // Default TCP port based on your turnServer config

  if (turnServer && Array.isArray(turnServer.listeningAddresses)) {
    for (const addrInfo of turnServer.listeningAddresses) {
      if (addrInfo.protocol === 'UDP') {
        isUdpListening = true;
        udpAddress = addrInfo.address;
        udpListenPort = addrInfo.port; // Actual listening UDP port
      } else if (addrInfo.protocol === 'TCP') {
        isTcpListening = true;
        tcpAddress = addrInfo.address;
        tcpListenPort = addrInfo.port; // Actual listening TCP port
      }
    }
  }

  // If TCP isn't actively listening but a specific TCP port was configured in node-turn,
  // ensure tcpListenPort reflects that configuration for the connection strings.
  if (!isTcpListening && turnServer && turnServer.options && turnServer.options.tcpListeningPort) {
    tcpListenPort = turnServer.options.tcpListeningPort;
  }

  const stunInfo = {
    protocols: {
      udp: {
        enabled: isUdpListening,
        port: udpListenPort,
        address: udpAddress
      },
      tcp: {
        enabled: isTcpListening,
        port: tcpListenPort,
        address: tcpAddress
      }
    },
    connectionStrings: {
      udp: `stun:${req.hostname}:${udpListenPort}`,
      tcp: `stun:${req.hostname}:${tcpListenPort}?transport=tcp`,
      turn_udp: `turn:${req.hostname}:${udpListenPort}?transport=udp`,
      turn_tcp: `turn:${req.hostname}:${tcpListenPort}?transport=tcp`
    },
    serverSoftware: turnServer && turnServer.options && turnServer.options.software 
                      ? turnServer.options.software 
                      : 'node-turn',
    timestamp: new Date().toISOString()
  };
  res.status(200).json(stunInfo);
});
// GET / (Root route)
// Redirects to the status page
app.get('/', (req, res) => {
  res.redirect('/status');
});

// Helper function to check for common private IPv4 addresses
function isPrivateIPv4(ip) {
  if (!ip || typeof ip !== 'string') return false;
  const parts = ip.split('.').map(Number);
  if (parts.length !== 4 || parts.some(p => isNaN(p) || p < 0 || p > 255)) return false; // Basic IPv4 check

  if (parts[0] === 10) return true; // 10.0.0.0/8
  if (parts[0] === 172 && parts[1] >= 16 && parts[1] <= 31) return true; // 172.16.0.0/12
  if (parts[0] === 192 && parts[1] === 168) return true; // 192.168.0.0/16
  if (parts[0] === 127) return true; // 127.0.0.0/8 (loopback)
  return false;
}

// Refactored registration logic to be reusable
async function handleNodeRegistration(req, res) {
  const { chainID, ip: providedIp, port, type = "client" /* 'client' or 'bootnode' */, nodeID } = req.body;
  let registrationIp = providedIp; // Start with the provided IP

  // Fallback to req.ip if no IP is provided or if it's an empty string.
  // Clients should use the STUN/TURN server for their own discovery if they need their public IP.
  if (!registrationIp || typeof registrationIp !== 'string' || registrationIp.trim() === '') {
    const detectedIp = req.ip;
    const fallbackIp = detectedIp.includes('::ffff:') ? detectedIp.split(':').pop() : detectedIp;
    registrationIp = fallbackIp;
    console.log(`[Registry] No valid IP provided for ${chainID}, using request source IP: ${registrationIp}`);
  } else {
    console.log(`[Registry] IP provided for ${chainID}: ${registrationIp}`);
  }

  // Validation
  if (!chainID || !port) {
    console.warn(`[Registry] Registration attempt with missing data. chainID: ${chainID}, port: ${port}, IP used: ${registrationIp}`);
    return res.status(400).json({ error: 'Missing required fields: chainID and port are mandatory.' });
  }
  
  // Add IP validation if needed (e.g., using a library like 'ip-address')
  if (typeof chainID !== 'string' || (typeof port !== 'number' && typeof port !== 'string') || 
      parseInt(port) <= 0 || parseInt(port) > 65535 || typeof registrationIp !== 'string') {
     console.warn(`[Registry] Registration attempt with invalid data types. chainID: ${chainID}, port: ${port}, IP used: ${registrationIp}`);
     return res.status(400).json({ error: 'Invalid data types or value for chainID, port, or IP' });
  }

  // Convert port to number if it's a string
  const portNum = typeof port === 'string' ? parseInt(port) : port;

  // Log a warning if a bootnode registers with a private IP
  if (type === "bootnode" && isPrivateIPv4(registrationIp)) {
    console.warn(`[Registry] Bootnode ${chainID} (${nodeID || 'N/A'}) registered with a private IP address: ${registrationIp}. This may affect its reachability from external networks.`);
  }
  
  if (type === "bootnode" && !bootnodeList.find(bn => bn.chainID === chainID)) {
    // Ensure nodeID is stored for bootnodes if provided
    bootnodeList.push({ chainID, ip: registrationIp, port: portNum, nodeID }); 
    console.log(`[Registry] Added to bootnode list: ${chainID}`);
  } else if (type !== "bootnode" && bootnodeList.find(bn => bn.chainID === chainID)) {
    // If a node was a bootnode and re-registers as not, remove it
    bootnodeList = bootnodeList.filter(bn => bn.chainID !== chainID);
    console.log(`[Registry] Removed from bootnode list: ${chainID}`);
    // Reset currentBootnodeIndex if it's now out of bounds
    if (currentBootnodeIndex >= bootnodeList.length && bootnodeList.length > 0) {
        currentBootnodeIndex = 0;
    } else if (bootnodeList.length === 0) {
        currentBootnodeIndex = 0;
    }
  }
  
  let suggestedBootnode = null;
  if (type === "client" && bootnodeList.length > 0) {
    // Ensure index is valid before accessing
    if (currentBootnodeIndex >= bootnodeList.length) {
        currentBootnodeIndex = 0; // Wrap around or reset
    }
    suggestedBootnode = bootnodeList[currentBootnodeIndex];
    currentBootnodeIndex = (currentBootnodeIndex + 1) % bootnodeList.length;
    console.log(`[Registry] Suggested bootnode for ${chainID}: ${suggestedBootnode.chainID} (${suggestedBootnode.ip}:${suggestedBootnode.port})`);
  }

  const registrationTime = Date.now();
  chainRegistry[chainID] = {
    ip: registrationIp,
    port: portNum,
    lastSeen: registrationTime,
    type,
    nodeID
  };

  console.log(`[Registry] Registered/Updated node: ${chainID} (${type}) -> ${registrationIp}:${portNum}`);
  
  const responsePayload = {
      message: type === "bootnode" ? 'Bootnode registered successfully' : 'Client node registered successfully',
      registeredIp: registrationIp, // Confirm the IP used for registration
      details: { chainID, type, nodeID, port: portNum }
  };
  if (suggestedBootnode) responsePayload.suggestedBootnode = suggestedBootnode;
  // Use 201 Created for successful registration as it creates/updates a resource
  res.status(201).json(responsePayload);
}

// POST /register or /register/
// Registers a node using its chainID, listening port, nodeID and type.
// It prioritizes the 'ip' field from the request body.
// If 'ip' is not provided, it falls back to detecting the IP from the request source.
app.post(/^\/register\/?$/, handleNodeRegistration);

// GET /register or /register/
// This endpoint is primarily for POST. A GET request might indicate confusion or a redirect.
// Returning 405 Method Not Allowed is semantically correct.
// Alternatively, redirecting to /status might provide a smoother client experience if registration is confirmed working.
app.get(/^\/register\/?$/, (req, res) => {
  // Option 1: Redirect to status (client will see 200 OK from /status eventually)
  // res.redirect('/status');
  // Option 2: Return 405 Method Not Allowed (client will see this error, but it's accurate)
  res.status(405).json({ error: 'Method Not Allowed. Use POST to /register for node registration.', hint: 'See /status for registry information.' });
});

// GET /lookup/:chainID
// Looks up the connection details (IP, port) for a given chainID.
app.get('/lookup/:chainID', (req, res) => {
  const { chainID } = req.params;
  const nodeInfo = chainRegistry[chainID];

  if (nodeInfo) {
    console.log(`[Registry] Lookup successful for bootnode: ${chainID}`);
    res.status(200).json({
      ip: nodeInfo.ip,
      port: nodeInfo.port,
      type: nodeInfo.type,
      nodeID: nodeInfo.nodeID
    });
  } else {
    console.log(`[Registry] Lookup failed for unknown bootnode: ${chainID}`);
    res.status(404).json({ error: 'Node not found' });
  }
});

// GET /nodes (Optional Debug Endpoint)
// Lists all currently registered nodes.
app.get('/nodes', (req, res) => {
    res.status(200).json(chainRegistry); // chainRegistry already contains all fields including nodeID and type
});

app.get('/bootnodes', (req, res) => {
  console.log('[Registry] Request for bootnode list.');
  res.status(200).json(chainRegistry);
});

// GET /status
// Shows the status of the registry service, including registered nodes
// Returns HTML for browsers and JSON for API clients
app.get('/status', (req, res) => {
  // Count active nodes
  const now = Date.now();
  const activeNodes = Object.entries(chainRegistry).filter(([_, info]) => 
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
    registeredNodes: Object.keys(chainRegistry).length,
    activeNodes: activeNodes.length,
    memoryUsage: process.memoryUsage(),
    httpPort: httpPort,
    stunPort: stunPort,
    turnServerSoftware: turnServer && turnServer.options && turnServer.options.software 
                          ? turnServer.options.software 
                          : 'node-turn',
    timestamp: new Date().toISOString()
  };
  
  // Check if request is from a browser
  const acceptHeader = req.headers.accept || '';
  const isHtmlRequest = acceptHeader.includes('text/html');
  
  if (isHtmlRequest) {
    // Return HTML for browsers
    const nodeRows = Object.entries(chainRegistry).map(([chainID, info]) => {
      const lastSeenDate = new Date(info.lastSeen);
      const isActive = now - info.lastSeen < 60 * 60 * 1000; // Active in the last hour
      const statusClass = isActive ? 'active' : 'inactive';
      
      return `
        <tr>
          <td class="chainid">${chainID}</td>
          <td>${info.type || 'N/A'}</td>
          <td>${info.ip}</td>
          <td>${info.port}</td>
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
        <title>KNIRVORACLE Bootnode Registry Status</title>
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
          .chainid {
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
        <h1>KNIRVORACLE Bootnode Registry Status</h1>
        
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
              <div class="status-label">Registered Nodes</div>
              <div class="status-value">${statusData.registeredNodes}</div>
            </div>
            <div class="status-item">
              <div class="status-label">Active Nodes</div>
              <div class="status-value">${statusData.activeNodes}</div>
            </div>
            <div class="status-item">
              <div class="status-label">HTTP Port</div>
              <div class="status-value">${statusData.httpPort}</div>
            </div>
            <div class="status-item">
              <div class="status-label">STUN Port</div>
              <div class="status-value">${statusData.stunPort}</div>
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
        
        <h2>Registered Nodes</h2>
        <table>
          <thead>
            <tr>
              <th>Chain ID</th>
              <th>Type</th>
              <th>IP Address</th>
              <th>Port</th>
              <th>Status</th>
              <th>Last Seen</th>
            </tr>
          </thead>
          <tbody>
            ${nodeRows.length > 0 ? nodeRows : '<tr><td colspan="6" style="text-align: center;">No nodes registered</td></tr>'}
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
      nodes: chainRegistry
    });
  }
});

// --- Periodic Pruning of Inactive Nodes ---
function pruneInactiveNodes() {
    const now = Date.now();
    // Example: Prune nodes inactive for more than 1 hour
    const maxInactiveTime = 60 * 60 * 1000; // 1 hour in milliseconds
    let prunedCount = 0;

    console.log('[Registry] Running inactive bootnode pruning...');
    for (const chainID in chainRegistry) {
        if (now - chainRegistry[chainID].lastSeen > maxInactiveTime) {
            const nodeType = chainRegistry[chainID].type;
            console.log(`[Registry] Pruning inactive ${nodeType} node: ${chainID} (Last seen: ${new Date(chainRegistry[chainID].lastSeen).toISOString()})`);
            delete chainRegistry[chainID];
            prunedCount++;
            
            // If it was a bootnode, remove from bootnodeList
            if (nodeType === "bootnode") {
                bootnodeList = bootnodeList.filter(bn => bn.chainID !== chainID);
                console.log(`[Registry] Removed from bootnode list during pruning: ${chainID}`);
            }
        }
    }
    if (prunedCount > 0) {
        console.log(`[Registry] Pruned ${prunedCount} inactive bootnodes.`);
    } else {
        console.log('[Registry] No inactive nodes to prune.');
    }
}

// Run pruning every 15 minutes (adjust interval as needed)
const pruneInterval = 15 * 60 * 1000;
setInterval(pruneInactiveNodes, pruneInterval);

// --- TURN/STUN Server Setup (using node-turn) ---
const turnServer = new Turn({
  // For STUN only, auth is not strictly needed.
  // For TURN relay, you MUST enable authentication in production.
  // Example (commented out, enable and configure for production relay):
  // authMech: 'long-term',
  // credentials: {
  //   'user1': 'pass1', // Static credentials
  //   'agent': Turn.generateLongTermCredentials("KNIRVORACLE_secure_password", "agent.com") // Long-term
  // },
  // realm: 'agent.com',
  // UDP primary with TCP fallback
  // UDP primary with TCP fallback
  listeningPort: stunPort, // UDP port (3478)
  tcpListeningPort: stunPort + 1, // TCP fallback port (3479)
  transportProtocols: ['udp', 'tcp'], // Enable both protocols
  // If your server is behind NAT, specify its public IP(s) here
  // listeningIps: ['your_server_public_ip1', 'your_server_public_ip2_if_any'],
  // relayIps: ['your_server_public_ip1', 'your_server_public_ip2_if_any'],
  // Or let node-turn try to find it (might require specific network setup)
  // By default, it tries to use the first non-internal IPv4 address.
  software: 'KNIRVORACLE Registry TURN/STUN Server',
  maxPort: 65535, // Default
  minPort: 49152, // Default
  verbose: true, // For more detailed logging from node-turn
  // Disable relaying if you only want STUN (not recommended if you need TURN)
  // disallowRelay: true, 
});

// Add listeners for node-turn server events
turnServer.on('listening', (address, port, protocol) => {
  console.log(`[TURN/${protocol.toUpperCase()}] Server listening on ${address}:${port}`);
});

turnServer.on('error', (code, message) => {
  console.error(`[TURN] Server error ${code}: ${message}`);
});

turnServer.on('traffic', (message, protocol, address, port) => {
  // console.log(`[TURN/${protocol.toUpperCase()}] Traffic from ${address}:${port}, type: 0x${message.type.toString(16)}`);
});

turnServer.on('allocation', (allocation) => {
    console.log(`[TURN] Allocation created: ${allocation.username} from ${allocation.clientAddress}:${allocation.clientPort} -> relay ${allocation.relayAddress}:${allocation.relayPort}`);
});

turnServer.on('credentialError', (username, realm, error_code, error_message) => {
    console.warn(`[TURN] Credential error for user ${username} in realm ${realm}: ${error_message} (Code: ${error_code})`);
});

// --- Start Servers ---

// Start HTTP Registry API Server
const httpServer = app.listen(httpPort, () => {
  console.log(`[Registry] HTTP Registry service listening on TCP port ${httpPort}`);
   pruneInactiveNodes(); // Initial prune
});

// Start TURN/STUN Server
turnServer.start();

// Servers are designed to run continuously without shutdown handlers
// UDP and TCP STUN servers will remain running along with HTTP server
