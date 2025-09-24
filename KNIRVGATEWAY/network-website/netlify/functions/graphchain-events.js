// Netlify Function for GraphChain-specific SSE events
// Handles real-time GraphChain data streaming

const isTestnet = process.env.TESTNET_MODE === 'true' || process.env.NODE_ENV === 'testnet';

// GraphChain service configuration
const graphChainService = {
  name: "knirvgraph",
  url: process.env.KNIRVGRAPH_URL || (isTestnet ? "http://localhost:8080" : "https://graph.knirv.network"),
  healthPath: "/height",
  isHealthy: true,
  lastCheck: new Date()
};

// Event cache for SSE
let eventCache = [];
let lastHeight = 0;
let connectionCount = 0;

exports.handler = async (event, context) => {
  const { path, httpMethod, headers, queryStringParameters } = event;
  
  // Handle CORS
  if (httpMethod === 'OPTIONS') {
    return {
      statusCode: 200,
      headers: {
        'Access-Control-Allow-Origin': '*',
        'Access-Control-Allow-Headers': 'Content-Type, Authorization, Cache-Control',
        'Access-Control-Allow-Methods': 'GET, POST, OPTIONS'
      }
    };
  }

  try {
    // Route handling
    if (path === '/api/graphchain/events' || path.endsWith('/graphchain-events')) {
      return await handleGraphChainSSE(queryStringParameters);
    } else if (path.startsWith('/api/graphchain/')) {
      return await proxyToGraphChain(event);
    } else {
      return {
        statusCode: 404,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ error: 'Endpoint not found' })
      };
    }
  } catch (error) {
    console.error('GraphChain Events Error:', error);
    return {
      statusCode: 500,
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ error: error.message })
    };
  }
};

async function handleGraphChainSSE(queryParams = {}) {
  connectionCount++;
  console.log(`GraphChain SSE connection established. Active connections: ${connectionCount}`);
  
  // Generate initial events
  const events = await generateSSEEvents(queryParams);
  
  return {
    statusCode: 200,
    headers: {
      'Content-Type': 'text/event-stream',
      'Cache-Control': 'no-cache',
      'Connection': 'keep-alive',
      'Access-Control-Allow-Origin': '*',
      'Access-Control-Allow-Headers': 'Cache-Control'
    },
    body: events
  };
}

async function generateSSEEvents(queryParams = {}) {
  const events = [];
  
  try {
    // Connection established event
    events.push(formatSSEEvent('connected', {
      type: 'connected',
      timestamp: Date.now(),
      message: 'GraphChain SSE connection established',
      service: graphChainService.name,
      testnet: isTestnet
    }));

    // Get current GraphChain data
    const currentData = await fetchGraphChainData();
    
    if (currentData) {
      // Height update event
      if (currentData.height !== undefined) {
        lastHeight = currentData.height;
        events.push(formatSSEEvent('height_update', {
          type: 'height_update',
          height: currentData.height,
          timestamp: Date.now()
        }));
      }

      // Skills data event
      if (currentData.skills && currentData.skills.length > 0) {
        events.push(formatSSEEvent('skills_data', {
          type: 'skills_data',
          skills: currentData.skills.slice(0, 5), // Send recent 5
          total: currentData.skills.length,
          timestamp: Date.now()
        }));
      }

      // Errors data event
      if (currentData.errors && currentData.errors.length > 0) {
        events.push(formatSSEEvent('errors_data', {
          type: 'errors_data',
          errors: currentData.errors.slice(0, 5), // Send recent 5
          total: currentData.errors.length,
          timestamp: Date.now()
        }));
      }

      // Stats update event
      events.push(formatSSEEvent('stats_update', {
        type: 'stats_update',
        stats: {
          height: currentData.height || 0,
          totalSkillNodes: currentData.skills ? currentData.skills.length : 0,
          totalErrorNodes: currentData.errors ? currentData.errors.length : 0,
          avgResolutionTime: calculateAvgResolutionTime(currentData.skills || [])
        },
        timestamp: Date.now()
      }));
    }

    // Heartbeat event
    events.push(formatSSEEvent('heartbeat', {
      type: 'heartbeat',
      timestamp: Date.now(),
      connections: connectionCount
    }));

  } catch (error) {
    console.error('Error generating SSE events:', error);
    events.push(formatSSEEvent('error', {
      type: 'error',
      message: 'Failed to fetch GraphChain data',
      timestamp: Date.now()
    }));
  }

  return events.join('\n') + '\n\n';
}

function formatSSEEvent(eventType, data) {
  return `event: ${eventType}\ndata: ${JSON.stringify(data)}\n`;
}

async function fetchGraphChainData() {
  try {
    const baseUrl = graphChainService.url;
    
    // Fetch height
    const heightResponse = await fetch(`${baseUrl}/height`);
    const heightData = heightResponse.ok ? await heightResponse.json() : null;
    
    // Fetch skills
    const skillsResponse = await fetch(`${baseUrl}/nrv/skills`);
    const skillsData = skillsResponse.ok ? await skillsResponse.json() : [];
    
    // Fetch errors
    const errorsResponse = await fetch(`${baseUrl}/nrv/errors`);
    const errorsData = errorsResponse.ok ? await errorsResponse.json() : [];
    
    return {
      height: heightData?.height || 0,
      skills: skillsData || [],
      errors: errorsData || []
    };
    
  } catch (error) {
    console.error('Failed to fetch GraphChain data:', error);
    return null;
  }
}

function calculateAvgResolutionTime(skills) {
  const skillsWithPerformance = skills.filter(skill => skill.performance);
  if (skillsWithPerformance.length === 0) return 0;
  
  const totalTime = skillsWithPerformance.reduce(
    (sum, skill) => sum + (skill.performance.avg_resolution_time || 0), 0
  );
  
  return totalTime / skillsWithPerformance.length;
}

async function proxyToGraphChain(event) {
  const { path, httpMethod, headers, body } = event;
  
  // Remove /api/graphchain prefix for proxying
  const targetPath = path.replace('/api/graphchain', '');
  const targetUrl = `${graphChainService.url}${targetPath}`;
  
  try {
    const response = await fetch(targetUrl, {
      method: httpMethod,
      headers: {
        'Content-Type': headers['content-type'] || 'application/json',
        'Authorization': headers.authorization || ''
      },
      body: httpMethod !== 'GET' ? body : undefined
    });
    
    const responseBody = await response.text();
    
    return {
      statusCode: response.status,
      headers: { 
        'Content-Type': response.headers.get('content-type') || 'application/json',
        'Access-Control-Allow-Origin': '*'
      },
      body: responseBody
    };
  } catch (error) {
    console.error('GraphChain proxy error:', error);
    return {
      statusCode: 502,
      headers: { 
        'Content-Type': 'application/json',
        'Access-Control-Allow-Origin': '*'
      },
      body: JSON.stringify({ 
        error: 'Bad Gateway', 
        details: error.message,
        service: 'knirvgraph'
      })
    };
  }
}

// Cleanup function (called when connection closes)
function handleConnectionClose() {
  connectionCount = Math.max(0, connectionCount - 1);
  console.log(`GraphChain SSE connection closed. Active connections: ${connectionCount}`);
}

// Health check for GraphChain service
async function checkGraphChainHealth() {
  try {
    const response = await fetch(`${graphChainService.url}${graphChainService.healthPath}`);
    graphChainService.isHealthy = response.ok;
    graphChainService.lastCheck = new Date();
    return response.ok;
  } catch (error) {
    console.error('GraphChain health check failed:', error);
    graphChainService.isHealthy = false;
    graphChainService.lastCheck = new Date();
    return false;
  }
}

// Periodic health check (if running in a persistent environment)
if (typeof setInterval !== 'undefined') {
  setInterval(checkGraphChainHealth, 30000); // Check every 30 seconds
}
