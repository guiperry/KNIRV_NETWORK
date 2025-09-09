/**
 * Netlify Function: /provision endpoint
 * 
 * Provides dynamic peer discovery for the private DHT.
 * In serverless mode, this proxies requests to the persistent Render gateway.
 */

const axios = require('axios');
const NodeCache = require('node-cache');

// Cache for 60 seconds
const cache = new NodeCache({ stdTTL: 60, checkperiod: 10 });

exports.handler = async (event, context) => {
  // CORS headers
  const headers = {
    'Access-Control-Allow-Origin': '*',
    'Access-Control-Allow-Headers': 'Content-Type, Authorization',
    'Access-Control-Allow-Methods': 'GET, OPTIONS',
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

  // Only allow GET requests
  if (event.httpMethod !== 'GET') {
    return {
      statusCode: 405,
      headers,
      body: JSON.stringify({ error: 'Method not allowed' })
    };
  }

  try {
    // Check cache first
    const cachedPeers = cache.get('dht_peers');
    if (cachedPeers) {
      console.log('Returning cached DHT peers (Netlify)');
      return {
        statusCode: 200,
        headers,
        body: JSON.stringify(cachedPeers)
      };
    }

    // Get configuration from environment
    const RENDER_GATEWAY_INTERNAL_API = process.env.RENDER_GATEWAY_INTERNAL_API;
    const INTERNAL_API_KEY = process.env.INTERNAL_API_KEY;

    if (!RENDER_GATEWAY_INTERNAL_API || !INTERNAL_API_KEY) {
      console.error('Gateway internal API endpoint or key not configured');
      return {
        statusCode: 500,
        headers,
        body: JSON.stringify({
          error: 'Gateway internal API endpoint or key not configured',
          message: 'Please configure RENDER_GATEWAY_INTERNAL_API and INTERNAL_API_KEY environment variables'
        })
      };
    }

    // Fetch from persistent Render gateway
    console.log('Fetching peers from persistent gateway:', RENDER_GATEWAY_INTERNAL_API);
    
    const response = await axios.get(RENDER_GATEWAY_INTERNAL_API, {
      headers: {
        'Authorization': `Bearer ${INTERNAL_API_KEY}`,
        'User-Agent': 'KNIRVGATEWAY-Netlify/1.0.0'
      },
      timeout: 5000
    });

    const dhtPeers = response.data;

    // Validate response
    if (!Array.isArray(dhtPeers)) {
      throw new Error('Invalid response format from persistent gateway');
    }

    // Cache the result
    cache.set('dht_peers', dhtPeers);

    console.log(`Successfully fetched ${dhtPeers.length} peers from persistent gateway`);

    return {
      statusCode: 200,
      headers,
      body: JSON.stringify(dhtPeers)
    };

  } catch (error) {
    console.error('Error in Netlify provision function:', error);

    // Determine appropriate error response
    let statusCode = 500;
    let errorMessage = 'Failed to fetch DHT peers';
    
    if (error.code === 'ECONNREFUSED' || error.code === 'ENOTFOUND') {
      statusCode = 503;
      errorMessage = 'Persistent gateway unavailable';
    } else if (error.response && error.response.status) {
      statusCode = error.response.status;
      errorMessage = error.response.data?.error || 'Gateway error';
    }

    return {
      statusCode,
      headers,
      body: JSON.stringify({
        error: errorMessage,
        details: error.message,
        timestamp: Date.now(),
        mode: 'netlify-serverless'
      })
    };
  }
};
