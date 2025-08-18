/**
 * Gateway SSE Handler
 * Migrated from KNIRVGATEWAY/netlify/functions/gateway-sse.js
 */

const handleSSE = (req, res) => {
  // Set SSE headers
  res.writeHead(200, {
    'Content-Type': 'text/event-stream',
    'Cache-Control': 'no-cache',
    'Connection': 'keep-alive',
    'Access-Control-Allow-Origin': '*',
    'Access-Control-Allow-Headers': 'Cache-Control'
  });

  // Send initial connection message
  const sendEvent = (data) => {
    res.write(`data: ${JSON.stringify(data)}\n\n`);
  };

  sendEvent({
    type: 'connection',
    message: 'Connected to KNIRVTESTNET Gateway',
    timestamp: new Date().toISOString(),
    environment: process.env.NODE_ENV || 'testnet'
  });

  // Send periodic updates
  const heartbeatInterval = setInterval(() => {
    sendEvent({
      type: 'heartbeat',
      timestamp: new Date().toISOString(),
      uptime: process.uptime()
    });
  }, 30000);

  // Send service status updates
  const statusInterval = setInterval(async () => {
    try {
      const { loadEndpoints } = require('../../scripts/load-endpoints');
      const { endpoints } = loadEndpoints(process.env.NODE_ENV || 'testnet');
      
      sendEvent({
        type: 'service_status',
        services: Object.keys(endpoints).map(key => ({
          name: key.replace('_API', ''),
          status: 'unknown', // Would need actual health checks
          endpoint: endpoints[key]
        })),
        timestamp: new Date().toISOString()
      });
    } catch (error) {
      sendEvent({
        type: 'error',
        message: 'Failed to get service status',
        timestamp: new Date().toISOString()
      });
    }
  }, 60000);

  // Clean up on disconnect
  req.on('close', () => {
    clearInterval(heartbeatInterval);
    clearInterval(statusInterval);
  });

  req.on('end', () => {
    clearInterval(heartbeatInterval);
    clearInterval(statusInterval);
  });
};

const handleSSEPost = (req, res) => {
  // Handle SSE configuration or commands
  const { action, data } = req.body;
  
  switch (action) {
    case 'subscribe':
      res.json({
        success: true,
        message: 'Subscription request received',
        channels: data?.channels || ['default']
      });
      break;
      
    case 'unsubscribe':
      res.json({
        success: true,
        message: 'Unsubscription request received'
      });
      break;
      
    default:
      res.status(400).json({
        error: 'Invalid action',
        validActions: ['subscribe', 'unsubscribe']
      });
  }
};

module.exports = {
  handleSSE,
  handleSSEPost
};
