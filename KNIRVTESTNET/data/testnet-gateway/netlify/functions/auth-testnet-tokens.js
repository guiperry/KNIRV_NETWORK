// KNIRV TESTNET Gateway Authentication Tokens Function
// Provides testnet authentication tokens and validation

const { loadConfig } = require('./config-loader');
const crypto = require('crypto');

// Generate testnet token
function generateTestnetToken(userId = 'testnet-user') {
  const timestamp = Date.now();
  const randomBytes = crypto.randomBytes(16).toString('hex');
  const payload = {
    user_id: userId,
    environment: 'testnet',
    issued_at: timestamp,
    expires_at: timestamp + (24 * 60 * 60 * 1000), // 24 hours
    permissions: [
      'testnet:read',
      'testnet:write',
      'services:access',
      'gateway:use',
      'blockchain:interact',
      'skills:invoke',
      'agents:manage'
    ],
    session_id: randomBytes
  };
  
  // Simple base64 encoding for testnet (not production security)
  const token = Buffer.from(JSON.stringify(payload)).toString('base64');
  
  return {
    token: `testnet_${token}`,
    payload: payload,
    type: 'Bearer'
  };
}

// Validate testnet token
function validateTestnetToken(token) {
  try {
    if (!token.startsWith('testnet_')) {
      return { valid: false, error: 'Invalid token format' };
    }
    
    const base64Token = token.replace('testnet_', '');
    const payload = JSON.parse(Buffer.from(base64Token, 'base64').toString());
    
    // Check expiration
    if (Date.now() > payload.expires_at) {
      return { valid: false, error: 'Token expired' };
    }
    
    // Check environment
    if (payload.environment !== 'testnet') {
      return { valid: false, error: 'Invalid environment' };
    }
    
    return { valid: true, payload: payload };
  } catch (error) {
    return { valid: false, error: 'Invalid token' };
  }
}

// Main handler
exports.handler = async (event, context) => {
  console.log('Auth testnet tokens requested:', event.httpMethod, event.path);
  
  // Set CORS headers
  const headers = {
    'Access-Control-Allow-Origin': '*',
    'Access-Control-Allow-Headers': 'Content-Type, Authorization',
    'Access-Control-Allow-Methods': 'GET, POST, PUT, DELETE, OPTIONS',
    'Content-Type': 'application/json'
  };

  // Handle preflight requests
  if (event.httpMethod === 'OPTIONS') {
    return {
      statusCode: 200,
      headers,
      body: ''
    };
  }

  try {
    // Load configuration
    const config = await loadConfig();
    
    if (event.httpMethod === 'GET') {
      // Generate new testnet token
      const queryParams = event.queryStringParameters || {};
      const userId = queryParams.user_id || 'testnet-user';
      
      const tokenData = generateTestnetToken(userId);
      
      const response = {
        authentication: {
          status: 'success',
          timestamp: new Date().toISOString(),
          environment: 'testnet'
        },
        token: tokenData.token,
        token_type: tokenData.type,
        expires_in: 86400, // 24 hours in seconds
        expires_at: new Date(tokenData.payload.expires_at).toISOString(),
        permissions: tokenData.payload.permissions,
        user: {
          id: tokenData.payload.user_id,
          environment: 'testnet',
          session_id: tokenData.payload.session_id
        },
        usage: {
          header: `Authorization: Bearer ${tokenData.token}`,
          example: `curl -H "Authorization: Bearer ${tokenData.token}" http://localhost:8888/gateway/health`
        },
        testnet_features: {
          simplified_auth: true,
          no_registration_required: true,
          extended_permissions: true,
          auto_renewal: false
        }
      };
      
      return {
        statusCode: 200,
        headers,
        body: JSON.stringify(response, null, 2)
      };
      
    } else if (event.httpMethod === 'POST') {
      // Validate token
      const body = event.body ? JSON.parse(event.body) : {};
      const authHeader = event.headers.authorization || event.headers.Authorization;
      
      let token = body.token;
      if (!token && authHeader) {
        token = authHeader.replace('Bearer ', '');
      }
      
      if (!token) {
        return {
          statusCode: 400,
          headers,
          body: JSON.stringify({
            authentication: {
              status: 'error',
              timestamp: new Date().toISOString(),
              error: 'No token provided'
            },
            valid: false
          })
        };
      }
      
      const validation = validateTestnetToken(token);
      
      const response = {
        authentication: {
          status: validation.valid ? 'success' : 'error',
          timestamp: new Date().toISOString(),
          environment: 'testnet'
        },
        valid: validation.valid,
        ...(validation.valid ? {
          user: {
            id: validation.payload.user_id,
            environment: validation.payload.environment,
            session_id: validation.payload.session_id
          },
          permissions: validation.payload.permissions,
          expires_at: new Date(validation.payload.expires_at).toISOString(),
          time_remaining: Math.max(0, validation.payload.expires_at - Date.now())
        } : {
          error: validation.error
        })
      };
      
      return {
        statusCode: validation.valid ? 200 : 401,
        headers,
        body: JSON.stringify(response, null, 2)
      };
    }
    
    // Method not allowed
    return {
      statusCode: 405,
      headers,
      body: JSON.stringify({
        authentication: {
          status: 'error',
          timestamp: new Date().toISOString(),
          error: 'Method not allowed'
        },
        allowed_methods: ['GET', 'POST']
      })
    };

  } catch (error) {
    console.error('Auth testnet tokens error:', error);
    
    return {
      statusCode: 500,
      headers,
      body: JSON.stringify({
        authentication: {
          status: 'error',
          timestamp: new Date().toISOString(),
          error: error.message
        },
        valid: false
      }, null, 2)
    };
  }
};
