// Netlify Function for API Gateway with SSE support
// Replaces WebSocket functionality with Server-Sent Events

const { createProxyMiddleware } = require('http-proxy-middleware');

// Testnet configuration
const isTestnet = process.env.TESTNET_MODE === 'true' || process.env.NODE_ENV === 'testnet';

// Service registry - testnet uses local services, production uses remote
let services = {};

if (isTestnet) {
  // Testnet service configuration - all local
  services = {
    knirvroot: {
      name: "knirvroot",
      url: process.env.KNIRVROOT_URL || "http://localhost:1317",
      healthPath: "/health",
      isHealthy: true,
      lastCheck: new Date(),
      testnet: true
    },
    knirvchain: {
      name: "knirvchain",
      url: process.env.KNIRVCHAIN_URL || "http://localhost:8080",
      healthPath: "/health",
      isHealthy: true,
      lastCheck: new Date(),
      testnet: true
    },
    knirvgraph: {
      name: "knirvgraph",
      url: process.env.KNIRVGRAPH_URL || "http://localhost:8081",
      healthPath: "/health",
      isHealthy: true,
      lastCheck: new Date(),
      testnet: true
    },
    knirvnexus: {
      name: "knirvnexus",
      url: process.env.KNIRVNEXUS_URL || "http://localhost:8082",
      healthPath: "/health",
      isHealthy: true,
      lastCheck: new Date(),
      testnet: true
    },
    knirvrouter: {
      name: "knirvrouter",
      url: process.env.KNIRVROUTER_URL || "http://localhost:5001",
      healthPath: "/health",
      isHealthy: true,
      lastCheck: new Date(),
      testnet: true
    }
  };
} else {
  // Production service configuration
  services = {
    knirvchain: {
      name: "knirvchain",
      url: process.env.KNIRVCHAIN_URL || "https://chain.knirv.com",
      healthPath: "/health",
      isHealthy: true,
      lastCheck: new Date()
    },
    knirvgraph: {
      name: "knirvgraph",
      url: process.env.KNIRVGRAPH_URL || "https://graph.knirv.com",
      healthPath: "/health",
      isHealthy: true,
      lastCheck: new Date()
    },
    knirvnexus: {
      name: "knirvnexus",
      url: process.env.KNIRVNEXUS_URL || "https://nexus.knirv.com",
      healthPath: "/health",
      isHealthy: true,
      lastCheck: new Date()
    },
    knirvroot: {
      name: "knirvroot",
      url: process.env.KNIRVROOT_URL || "http://localhost:5002",
      healthPath: "/health",
      isHealthy: true,
      lastCheck: new Date()
    }
  };
}

// Authentication configuration
const validTokens = new Map();

// Testnet simplified authentication
if (isTestnet) {
  // Pre-populate with testnet tokens for easy testing
  validTokens.set('testnet-token-123', {
    user: 'testnet-user',
    permissions: ['read', 'write', 'admin'],
    expires: new Date(Date.now() + 24 * 60 * 60 * 1000) // 24 hours
  });
  validTokens.set('dev-token-456', {
    user: 'developer',
    permissions: ['read', 'write'],
    expires: new Date(Date.now() + 24 * 60 * 60 * 1000)
  });
  validTokens.set('guest-token-789', {
    user: 'guest',
    permissions: ['read'],
    expires: new Date(Date.now() + 24 * 60 * 60 * 1000)
  });
}

// Simplified auth check for testnet
function isAuthenticated(headers) {
  if (isTestnet) {
    // In testnet mode, allow requests without auth or with any of the test tokens
    const authHeader = headers.authorization || headers.Authorization;
    if (!authHeader) {
      return { authenticated: true, user: 'testnet-anonymous', permissions: ['read'] };
    }

    const token = authHeader.replace('Bearer ', '');
    const tokenData = validTokens.get(token);
    if (tokenData && tokenData.expires > new Date()) {
      return { authenticated: true, user: tokenData.user, permissions: tokenData.permissions };
    }

    // Even invalid tokens are accepted in testnet mode
    return { authenticated: true, user: 'testnet-fallback', permissions: ['read'] };
  }

  // Production auth logic
  const authHeader = headers.authorization || headers.Authorization;
  if (!authHeader) {
    return { authenticated: false };
  }

  const token = authHeader.replace('Bearer ', '');
  const tokenData = validTokens.get(token);
  return {
    authenticated: tokenData && tokenData.expires > new Date(),
    user: tokenData?.user,
    permissions: tokenData?.permissions || []
  };
}

// Rate limiting (simple in-memory, could use external store)
const rateLimits = new Map();

exports.handler = async (event, context) => {
  const { path, httpMethod, headers, body, queryStringParameters } = event;
  
  // Handle CORS
  if (httpMethod === 'OPTIONS') {
    return {
      statusCode: 200,
      headers: {
        'Access-Control-Allow-Origin': '*',
        'Access-Control-Allow-Headers': 'Content-Type, Authorization',
        'Access-Control-Allow-Methods': 'GET, POST, PUT, DELETE, OPTIONS'
      }
    };
  }

  try {
    // Route handling
    if (path === '/gateway/events') {
      return await handleSSEConnection(headers);
    } else if (path.startsWith('/gateway/')) {
      return await handleGatewayRoutes(path, httpMethod, headers, body);
    } else if (path.startsWith('/auth/')) {
      return await handleAuthRoutes(path, httpMethod, headers, body);
    } else {
      // Service proxy
      return await handleServiceProxy(path, httpMethod, headers, body);
    }
  } catch (error) {
    return {
      statusCode: 500,
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ error: error.message })
    };
  }
};

async function handleGatewayRoutes(path, method, headers, body) {
  const route = path.replace('/gateway/', '');
  
  switch (route) {
    case 'health':
      return {
        statusCode: 200,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          status: 'healthy',
          timestamp: Date.now(),
          services: Object.keys(services).length,
          testnet: isTestnet,
          mode: isTestnet ? 'testnet' : 'production'
        })
      };
      
    case 'services':
      if (method === 'GET') {
        return {
          statusCode: 200,
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(services)
        };
      }
      break;
      
    case 'metrics':
      return {
        statusCode: 200,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          totalRequests: 0, // Could track in external storage
          services: Object.keys(services).length,
          timestamp: Date.now(),
          testnet: isTestnet
        })
      };

    case 'testnet/status':
      if (isTestnet) {
        return {
          statusCode: 200,
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            testnet: true,
            services: services,
            features: {
              simplified_auth: true,
              static_service_discovery: true,
              local_services: true,
              mock_responses: true
            },
            endpoints: {
              health: '/gateway/health',
              services: '/gateway/services',
              auth_tokens: '/auth/testnet-tokens',
              auth_validate: '/auth/validate'
            },
            timestamp: Date.now()
          })
        };
      }
      return {
        statusCode: 404,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ error: 'Testnet endpoints not available in production' })
      };

    case 'testnet/reset':
      if (isTestnet && method === 'POST') {
        // Reset testnet state
        validTokens.clear();

        // Re-populate testnet tokens
        validTokens.set('testnet-token-123', {
          user: 'testnet-user',
          permissions: ['read', 'write', 'admin'],
          expires: new Date(Date.now() + 24 * 60 * 60 * 1000)
        });
        validTokens.set('dev-token-456', {
          user: 'developer',
          permissions: ['read', 'write'],
          expires: new Date(Date.now() + 24 * 60 * 60 * 1000)
        });
        validTokens.set('guest-token-789', {
          user: 'guest',
          permissions: ['read'],
          expires: new Date(Date.now() + 24 * 60 * 60 * 1000)
        });

        return {
          statusCode: 200,
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            message: 'Testnet state reset successfully',
            tokens_restored: 3,
            timestamp: Date.now()
          })
        };
      }
      return {
        statusCode: 404,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ error: 'Reset endpoint not available' })
      };
  }

  return {
    statusCode: 404,
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ error: 'Route not found' })
  };
}

async function handleAuthRoutes(path, method, headers, body) {
  const route = path.replace('/auth/', '');

  switch (route) {
    case 'login':
      if (method === 'POST') {
        const { username, password } = JSON.parse(body || '{}');

        if (isTestnet) {
          // Testnet simplified login - accept any credentials
          const token = generateToken();
          const permissions = username === 'admin' ? ['read', 'write', 'admin'] : ['read', 'write'];
          validTokens.set(token, {
            user: username || 'testnet-user',
            permissions,
            expires: new Date(Date.now() + 24*60*60*1000)
          });

          return {
            statusCode: 200,
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
              token,
              expiresIn: '24h',
              user: username || 'testnet-user',
              permissions,
              testnet: true
            })
          };
        }

        // Production auth check
        if (username === 'admin' && password === 'password') {
          const token = generateToken();
          validTokens.set(token, {
            user: username,
            permissions: ['read', 'write', 'admin'],
            expires: new Date(Date.now() + 24*60*60*1000)
          });

          return {
            statusCode: 200,
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ token, expiresIn: '24h' })
          };
        }

        return {
          statusCode: 401,
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ error: 'Invalid credentials' })
        };
      }
      break;

    case 'validate':
      const authResult = isAuthenticated(headers);

      return {
        statusCode: 200,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          valid: authResult.authenticated,
          user: authResult.user,
          permissions: authResult.permissions,
          testnet: isTestnet
        })
      };

    case 'testnet-tokens':
      if (isTestnet) {
        return {
          statusCode: 200,
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            testnet: true,
            tokens: {
              admin: 'testnet-token-123',
              developer: 'dev-token-456',
              guest: 'guest-token-789'
            },
            note: 'These tokens are for testnet development only'
          })
        };
      }
      return {
        statusCode: 404,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ error: 'Not available in production' })
      };
  }

  return {
    statusCode: 404,
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ error: 'Auth route not found' })
  };
}

async function handleSSEConnection(headers) {
  // SSE endpoint for real-time updates
  // This replaces WebSocket functionality
  
  return {
    statusCode: 200,
    headers: {
      'Content-Type': 'text/event-stream',
      'Cache-Control': 'no-cache',
      'Connection': 'keep-alive',
      'Access-Control-Allow-Origin': '*',
      'Access-Control-Allow-Headers': 'Cache-Control'
    },
    body: `data: ${JSON.stringify({
      type: 'connected',
      timestamp: Date.now(),
      message: 'SSE connection established'
    })}\n\n`
  };
}

async function handleServiceProxy(path, method, headers, body) {
  // Determine target service based on path
  const serviceName = determineServiceFromPath(path);
  const service = services[serviceName];
  
  if (!service) {
    return {
      statusCode: 404,
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ error: 'Service not found' })
    };
  }
  
  // Check if service is healthy
  if (!service.isHealthy) {
    return {
      statusCode: 503,
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ error: 'Service unavailable' })
    };
  }
  
  // Proxy request to target service
  try {
    const targetUrl = `${service.url}${path}`;
    const response = await fetch(targetUrl, {
      method,
      headers: {
        'Content-Type': headers['content-type'] || 'application/json',
        'Authorization': headers.authorization || ''
      },
      body: method !== 'GET' ? body : undefined
    });
    
    const responseBody = await response.text();
    
    return {
      statusCode: response.status,
      headers: { 'Content-Type': response.headers.get('content-type') || 'application/json' },
      body: responseBody
    };
  } catch (error) {
    return {
      statusCode: 502,
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ error: 'Bad Gateway', details: error.message })
    };
  }
}

function determineServiceFromPath(path) {
  if (path.startsWith('/wallets') || path.startsWith('/nrn') || path.startsWith('/skill') || path.startsWith('/llm')) {
    return 'knirvchain';
  } else if (path.startsWith('/height') || path.startsWith('/node') || path.startsWith('/edge') || path.startsWith('/graph')) {
    return 'knirvgraph';
  } else if (path.startsWith('/api/v1/agents') || path.startsWith('/api/v1/workflows') || path.startsWith('/api/v1/mcp')) {
    return 'knirvnexus';
  } else if (path.startsWith('/api/registry') || path.startsWith('/api/uri') || path.startsWith('/api/economics') || path.startsWith('/economics')) {
    return 'knirvroot';
  }

  return 'knirvroot'; // Default fallback
}

function generateToken() {
  return 'token_' + Math.random().toString(36).substr(2, 9) + Date.now().toString(36);
}

function extractToken(headers) {
  const auth = headers.authorization || headers.Authorization || '';
  return auth.replace('Bearer ', '');
}

function validateToken(token) {
  if (!token) return false;
  
  const tokenInfo = validTokens.get(token);
  if (!tokenInfo) return false;
  
  if (Date.now() > tokenInfo.expiresAt) {
    validTokens.delete(token);
    return false;
  }
  
  return true;
}
