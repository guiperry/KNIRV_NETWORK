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
      url: process.env.KNIRVGRAPH_URL || "http://localhost:8080",
      healthPath: "/height",
      isHealthy: true,
      lastCheck: new Date(),
      testnet: true
    },
    knirvnexus_dve: {
      name: "knirvnexus_dve",
      url: process.env.KNIRVNEXUS_DVE_URL || "http://localhost:8080",
      healthPath: "/health",
      isHealthy: true,
      lastCheck: new Date(),
      testnet: true
    },
    knirvnexus_validation: {
      name: "knirvnexus_validation",
      url: process.env.KNIRVNEXUS_VALIDATION_URL || "http://localhost:8081",
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
      healthPath: "/height",
      isHealthy: true,
      lastCheck: new Date()
    },
    knirvnexus_dve: {
      name: "knirvnexus_dve",
      url: process.env.KNIRVNEXUS_DVE_URL || "https://nexus-dve.knirv.com",
      healthPath: "/health",
      isHealthy: true,
      lastCheck: new Date()
    },
    knirvnexus_validation: {
      name: "knirvnexus_validation",
      url: process.env.KNIRVNEXUS_VALIDATION_URL || "https://nexus-validation.knirv.com",
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

// Role-based authentication configuration
const roles = {
  admin: {
    permissions: ['*:*'], // Full access to all services and operations
    nexus_access: ['dve:*', 'validation:*', 'system:*'],
    description: 'Full administrative access'
  },
  validator: {
    permissions: ['nexus:read', 'nexus:validate', 'nexus:update_assigned'],
    nexus_access: ['dve:read', 'validation:read', 'validation:execute', 'system:read'],
    description: 'Validator node operator with scoped access'
  },
  observer: {
    permissions: ['*:read'],
    nexus_access: ['dve:read', 'validation:read', 'system:read'],
    description: 'Read-only access to all services'
  }
};

// Testnet simplified authentication
if (isTestnet) {
  // Pre-populate with testnet tokens for easy testing
  validTokens.set('testnet-admin-123', {
    user: 'testnet-admin',
    role: 'admin',
    permissions: roles.admin.permissions,
    nexus_access: roles.admin.nexus_access,
    expires: new Date(Date.now() + 24 * 60 * 60 * 1000) // 24 hours
  });
  validTokens.set('testnet-validator-456', {
    user: 'testnet-validator',
    role: 'validator',
    permissions: roles.validator.permissions,
    nexus_access: roles.validator.nexus_access,
    node_id: 'validator-node-001', // Scoped to specific node
    expires: new Date(Date.now() + 24 * 60 * 60 * 1000)
  });
  validTokens.set('testnet-observer-789', {
    user: 'testnet-observer',
    role: 'observer',
    permissions: roles.observer.permissions,
    nexus_access: roles.observer.nexus_access,
    expires: new Date(Date.now() + 24 * 60 * 60 * 1000)
  });
  // Legacy tokens for backward compatibility
  validTokens.set('testnet-token-123', {
    user: 'testnet-user',
    role: 'admin',
    permissions: ['read', 'write', 'admin'],
    expires: new Date(Date.now() + 24 * 60 * 60 * 1000)
  });
  validTokens.set('dev-token-456', {
    user: 'developer',
    role: 'observer',
    permissions: ['read', 'write'],
    expires: new Date(Date.now() + 24 * 60 * 60 * 1000)
  });
  validTokens.set('guest-token-789', {
    user: 'guest',
    role: 'observer',
    permissions: ['read'],
    expires: new Date(Date.now() + 24 * 60 * 60 * 1000)
  });
}

// Role-based authentication check
function isAuthenticated(headers) {
  if (isTestnet) {
    // In testnet mode, allow requests without auth or with any of the test tokens
    const authHeader = headers.authorization || headers.Authorization;
    if (!authHeader) {
      return {
        authenticated: true,
        user: 'testnet-anonymous',
        role: 'observer',
        permissions: ['read'],
        nexus_access: roles.observer.nexus_access
      };
    }

    const token = authHeader.replace('Bearer ', '');
    const tokenData = validTokens.get(token);
    if (tokenData && tokenData.expires > new Date()) {
      return {
        authenticated: true,
        user: tokenData.user,
        role: tokenData.role || 'observer',
        permissions: tokenData.permissions,
        nexus_access: tokenData.nexus_access || roles.observer.nexus_access,
        node_id: tokenData.node_id
      };
    }

    // Even invalid tokens are accepted in testnet mode
    return {
      authenticated: true,
      user: 'testnet-fallback',
      role: 'observer',
      permissions: ['read'],
      nexus_access: roles.observer.nexus_access
    };
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
    role: tokenData?.role || 'observer',
    permissions: tokenData?.permissions || [],
    nexus_access: tokenData?.nexus_access || roles.observer.nexus_access,
    node_id: tokenData?.node_id
  };
}

// Check if user has permission for specific NEXUS operations
function hasNexusPermission(authData, service, operation) {
  if (!authData.authenticated) {
    return false;
  }

  // Admin role has full access
  if (authData.role === 'admin') {
    return true;
  }

  // Check nexus-specific permissions
  const requiredPermission = `${service}:${operation}`;
  const hasWildcard = authData.nexus_access.includes(`${service}:*`);
  const hasSpecific = authData.nexus_access.includes(requiredPermission);

  return hasWildcard || hasSpecific;
}

// Check if validator has access to specific node (for scoped access)
function hasNodeAccess(authData, nodeId) {
  if (!authData.authenticated) {
    return false;
  }

  // Admin and observer roles have access to all nodes
  if (authData.role === 'admin' || authData.role === 'observer') {
    return true;
  }

  // Validator role is scoped to specific nodes
  if (authData.role === 'validator') {
    return !authData.node_id || authData.node_id === nodeId;
  }

  return false;
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
    } else if (path === '/gateway/events/nexus-dve') {
      headers['x-sse-channel'] = 'nexus-dve';
      return await handleSSEConnection(headers);
    } else if (path === '/gateway/events/nexus-validation') {
      headers['x-sse-channel'] = 'nexus-validation';
      return await handleSSEConnection(headers);
    } else if (path === '/gateway/events/nexus-system') {
      headers['x-sse-channel'] = 'nexus-system';
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

    // NEXUS-specific routes with role-based access control
    case 'nexus/dve-nodes':
      const authData1 = isAuthenticated(headers);
      if (!hasNexusPermission(authData1, 'dve', method === 'GET' ? 'read' : 'write')) {
        return {
          statusCode: 403,
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ error: 'Insufficient permissions for DVE nodes access' })
        };
      }
      return await proxyToService('knirvnexus_dve', '/api/dve-nodes', method, headers, body);

    case 'nexus/validation-tasks':
      const authData2 = isAuthenticated(headers);
      const operation = method === 'GET' ? 'read' : (method === 'POST' ? 'execute' : 'write');
      if (!hasNexusPermission(authData2, 'validation', operation)) {
        return {
          statusCode: 403,
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ error: 'Insufficient permissions for validation tasks access' })
        };
      }
      return await proxyToService('knirvnexus_validation', '/api/validation-tasks', method, headers, body);

    case 'nexus/validation-results':
      const authData3 = isAuthenticated(headers);
      if (!hasNexusPermission(authData3, 'validation', 'read')) {
        return {
          statusCode: 403,
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ error: 'Insufficient permissions for validation results access' })
        };
      }
      return await proxyToService('knirvnexus_validation', '/api/validation-results', method, headers, body);

    case 'nexus/system/status':
      const authData4 = isAuthenticated(headers);
      if (!hasNexusPermission(authData4, 'system', 'read')) {
        return {
          statusCode: 403,
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ error: 'Insufficient permissions for system status access' })
        };
      }
      // Aggregate status from both services
      const dveStatus = await proxyToService('knirvnexus_dve', '/api/system/status', 'GET', headers);
      const validationStatus = await proxyToService('knirvnexus_validation', '/api/system/status', 'GET', headers);

      return {
        statusCode: 200,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          dve_manager: dveStatus ? JSON.parse(dveStatus.body) : { status: 'unavailable' },
          validation_core: validationStatus ? JSON.parse(validationStatus.body) : { status: 'unavailable' },
          user_role: authData4.role,
          timestamp: Date.now()
        })
      };

    case 'nexus/system/metrics':
      const authData5 = isAuthenticated(headers);
      if (!hasNexusPermission(authData5, 'system', 'read')) {
        return {
          statusCode: 403,
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ error: 'Insufficient permissions for system metrics access' })
        };
      }
      // Aggregate metrics from both services
      const dveMetrics = await proxyToService('knirvnexus_dve', '/api/system/metrics', 'GET', headers);
      const validationMetrics = await proxyToService('knirvnexus_validation', '/api/system/metrics', 'GET', headers);

      return {
        statusCode: 200,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          dve_manager: dveMetrics ? JSON.parse(dveMetrics.body) : { status: 'unavailable' },
          validation_core: validationMetrics ? JSON.parse(validationMetrics.body) : { status: 'unavailable' },
          user_role: authData5.role,
          timestamp: Date.now()
        })
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

  // Extract channel from query parameters or headers
  const channel = headers['x-sse-channel'] || 'general';

  // Generate initial connection message based on channel
  let initialData = {
    type: 'connected',
    timestamp: Date.now(),
    channel: channel,
    message: 'SSE connection established'
  };

  // Add channel-specific initial data
  if (channel === 'nexus-dve') {
    initialData.data = {
      service: 'dve-manager',
      endpoints: ['/api/dve-nodes', '/api/tasks', '/api/system/status'],
      features: ['node_monitoring', 'task_assignment', 'load_balancing']
    };
  } else if (channel === 'nexus-validation') {
    initialData.data = {
      service: 'validation-core',
      endpoints: ['/api/validation-tasks', '/api/validation-results'],
      features: ['task_validation', 'result_processing', 'tee_attestation']
    };
  } else if (channel === 'nexus-system') {
    initialData.data = {
      services: ['dve-manager', 'validation-core'],
      monitoring: ['health', 'metrics', 'performance'],
      features: ['real_time_updates', 'aggregated_status']
    };
  }

  // Start periodic updates for NEXUS channels
  if (channel.startsWith('nexus-')) {
    // In a real implementation, this would set up periodic polling
    // For now, we'll just send the initial connection message
    setTimeout(() => {
      // This would send periodic updates in a real SSE implementation
      // Since Netlify Functions are stateless, real-time updates would need
      // to be implemented using external services like Pusher or WebSockets
    }, 1000);
  }

  return {
    statusCode: 200,
    headers: {
      'Content-Type': 'text/event-stream',
      'Cache-Control': 'no-cache',
      'Connection': 'keep-alive',
      'Access-Control-Allow-Origin': '*',
      'Access-Control-Allow-Headers': 'Cache-Control, X-SSE-Channel'
    },
    body: `data: ${JSON.stringify(initialData)}\n\n`
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

// Helper function to proxy requests to specific services
async function proxyToService(serviceName, path, method, headers, body) {
  const service = services[serviceName];
  if (!service || !service.isHealthy) {
    return {
      statusCode: 503,
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ error: `Service ${serviceName} unavailable` })
    };
  }

  try {
    const url = `${service.url}${path}`;
    const response = await fetch(url, {
      method: method,
      headers: {
        'Content-Type': 'application/json',
        ...headers
      },
      body: method !== 'GET' ? body : undefined
    });

    const responseBody = await response.text();

    return {
      statusCode: response.status,
      headers: { 'Content-Type': 'application/json' },
      body: responseBody
    };
  } catch (error) {
    return {
      statusCode: 502,
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ error: 'Service proxy error', details: error.message })
    };
  }
}

function determineServiceFromPath(path) {
  if (path.startsWith('/wallets') || path.startsWith('/nrn') || path.startsWith('/skill') || path.startsWith('/llm')) {
    return 'knirvchain';
  } else if (path.startsWith('/api/graphchain/') ||
             path.startsWith('/height') ||
             path.startsWith('/nrv/') ||
             path.startsWith('/node/') ||
             path.startsWith('/edge/') ||
             path.startsWith('/graph/') ||
             path.startsWith('/search')) {
    return 'knirvgraph';
  } else if (path.startsWith('/api/dve-nodes') ||
             path.startsWith('/api/tasks') ||
             path.startsWith('/api/system/status') ||
             path.startsWith('/api/system/metrics')) {
    return 'knirvnexus_dve';
  } else if (path.startsWith('/api/validation-tasks') ||
             path.startsWith('/api/validation-results') ||
             path.startsWith('/api/validation/')) {
    return 'knirvnexus_validation';
  } else if (path.startsWith('/api/v1/agents') || path.startsWith('/api/v1/workflows') || path.startsWith('/api/v1/mcp')) {
    return 'knirvnexus_dve'; // Legacy NEXUS routes go to DVE manager
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
