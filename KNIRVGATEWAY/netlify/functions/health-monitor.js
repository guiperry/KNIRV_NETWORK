// Netlify Function for Health Monitoring with SSE
// Replaces WebSocket health broadcasts with Server-Sent Events

const fs = require('fs');
const path = require('path');

const services = {
  knirvchain: process.env.KNIRVCHAIN_URL || "https://chain.knirv.com",
  knirvgraph: process.env.KNIRVGRAPH_URL || "https://graph.knirv.com",
  knirvnexus: process.env.KNIRVNEXUS_URL || "https://nexus.knirv.com",
  knirvroot: process.env.KNIRVROOT_URL || "https://root.knirv.com"
};

exports.handler = async (event, context) => {
  const { path, httpMethod, headers } = event;
  
  // Handle CORS
  if (httpMethod === 'OPTIONS') {
    return {
      statusCode: 200,
      headers: {
        'Access-Control-Allow-Origin': '*',
        'Access-Control-Allow-Headers': 'Content-Type, Cache-Control',
        'Access-Control-Allow-Methods': 'GET, OPTIONS'
      }
    };
  }

  if (httpMethod !== 'GET') {
    return {
      statusCode: 405,
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ error: 'Method not allowed' })
    };
  }

  // Health monitor frontend page
  if (path === '/health-monitor' || path === '/health-monitor/') {
    return await serveHealthMonitorPage();
  }

  // SSE endpoint for health monitoring
  if (path === '/health-monitor/events') {
    return await handleHealthSSE();
  }

  // Health check endpoint
  if (path === '/health-monitor/status') {
    return await handleHealthStatus();
  }

  return {
    statusCode: 404,
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ error: 'Not found' })
  };
};

async function serveHealthMonitorPage() {
  try {
    // Try to read the health monitor HTML file
    const htmlPath = path.join(process.cwd(), 'health-monitor.html');
    let htmlContent;

    try {
      htmlContent = fs.readFileSync(htmlPath, 'utf8');
    } catch (error) {
      // Fallback HTML if file not found
      htmlContent = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>KNIRV Health Monitor</title>
    <script src="https://cdn.tailwindcss.com"></script>
</head>
<body class="bg-gray-900 text-white min-h-screen">
    <div class="container mx-auto px-4 py-8">
        <h1 class="text-3xl font-bold mb-6">KNIRV Network Health Monitor</h1>
        <div id="status" class="bg-gray-800 p-6 rounded-lg">
            <p>Loading health status...</p>
        </div>
    </div>
    <script>
        fetch('/health-monitor/status')
            .then(response => response.json())
            .then(data => {
                document.getElementById('status').innerHTML = '<pre>' + JSON.stringify(data, null, 2) + '</pre>';
            })
            .catch(error => {
                document.getElementById('status').innerHTML = '<p class="text-red-500">Error loading health data: ' + error.message + '</p>';
            });
    </script>
</body>
</html>`;
    }

    return {
      statusCode: 200,
      headers: {
        'Content-Type': 'text/html',
        'Cache-Control': 'no-cache'
      },
      body: htmlContent
    };
  } catch (error) {
    return {
      statusCode: 500,
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ error: 'Failed to serve health monitor page' })
    };
  }
}

async function handleHealthSSE() {
  // Perform health checks
  const healthResults = await checkAllServices();
  
  // Create SSE response with health data
  const sseData = {
    type: 'health_update',
    timestamp: Date.now(),
    services: healthResults
  };

  return {
    statusCode: 200,
    headers: {
      'Content-Type': 'text/event-stream',
      'Cache-Control': 'no-cache',
      'Connection': 'keep-alive',
      'Access-Control-Allow-Origin': '*',
      'Access-Control-Allow-Headers': 'Cache-Control'
    },
    body: `data: ${JSON.stringify(sseData)}\n\n`
  };
}

async function handleHealthStatus() {
  const healthResults = await checkAllServices();
  
  return {
    statusCode: 200,
    headers: { 
      'Content-Type': 'application/json',
      'Access-Control-Allow-Origin': '*'
    },
    body: JSON.stringify({
      timestamp: Date.now(),
      services: healthResults,
      overall: Object.values(healthResults).every(s => s.healthy) ? 'healthy' : 'degraded'
    })
  };
}

async function checkAllServices() {
  const results = {};
  
  // Check each service in parallel
  const checks = Object.entries(services).map(async ([name, url]) => {
    const result = await checkServiceHealth(name, url);
    results[name] = result;
  });
  
  await Promise.all(checks);
  return results;
}

async function checkServiceHealth(serviceName, serviceUrl) {
  const startTime = Date.now();
  
  try {
    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), 10000); // 10s timeout
    
    const response = await fetch(`${serviceUrl}/health`, {
      method: 'GET',
      signal: controller.signal,
      headers: {
        'User-Agent': 'KNIRV-Gateway-Health-Monitor/1.0'
      }
    });
    
    clearTimeout(timeoutId);
    
    const responseTime = Date.now() - startTime;
    const healthy = response.ok;
    
    let details = {};
    try {
      const text = await response.text();
      details = text ? JSON.parse(text) : {};
    } catch (e) {
      details = { raw: await response.text() };
    }
    
    return {
      name: serviceName,
      healthy,
      status: response.status,
      responseTime,
      lastCheck: new Date().toISOString(),
      url: serviceUrl,
      details
    };
    
  } catch (error) {
    const responseTime = Date.now() - startTime;
    
    return {
      name: serviceName,
      healthy: false,
      status: 0,
      responseTime,
      lastCheck: new Date().toISOString(),
      url: serviceUrl,
      error: error.message,
      details: {
        errorType: error.name,
        timeout: error.name === 'AbortError'
      }
    };
  }
}

// Utility function to create SSE message format
function createSSEMessage(data, event = null, id = null) {
  let message = '';
  
  if (id) {
    message += `id: ${id}\n`;
  }
  
  if (event) {
    message += `event: ${event}\n`;
  }
  
  message += `data: ${JSON.stringify(data)}\n\n`;
  return message;
}

// Example usage for different event types:
function createHealthChangeEvent(serviceName, isHealthy) {
  return createSSEMessage({
    type: 'health_change',
    service: serviceName,
    healthy: isHealthy,
    timestamp: Date.now()
  }, 'health_change', `health_${serviceName}_${Date.now()}`);
}

function createMetricsEvent(metrics) {
  return createSSEMessage({
    type: 'metrics',
    data: metrics,
    timestamp: Date.now()
  }, 'metrics', `metrics_${Date.now()}`);
}

function createPingEvent() {
  return createSSEMessage({
    type: 'ping',
    timestamp: Date.now()
  }, 'ping', `ping_${Date.now()}`);
}
