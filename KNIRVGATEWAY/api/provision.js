/**
 * Vercel Function: /provision endpoint
 * 
 * Provides dynamic peer discovery for the private DHT.
 * In serverless mode, this proxies requests to the persistent Render gateway.
 */

import axios from 'axios';
import NodeCache from 'node-cache';

// Cache for 60 seconds
const cache = new NodeCache({ stdTTL: 60, checkperiod: 10 });

export default async function handler(req, res) {
  // CORS headers
  res.setHeader('Access-Control-Allow-Origin', '*');
  res.setHeader('Access-Control-Allow-Headers', 'Content-Type, Authorization');
  res.setHeader('Access-Control-Allow-Methods', 'GET, OPTIONS');

  // Handle preflight requests
  if (req.method === 'OPTIONS') {
    return res.status(200).end();
  }

  // Only allow GET requests
  if (req.method !== 'GET') {
    return res.status(405).json({ error: 'Method not allowed' });
  }

  try {
    // Check cache first
    const cachedPeers = cache.get('dht_peers');
    if (cachedPeers) {
      console.log('Returning cached DHT peers (Vercel)');
      return res.status(200).json(cachedPeers);
    }

    // Get configuration from environment
    const RENDER_GATEWAY_INTERNAL_API = process.env.RENDER_GATEWAY_INTERNAL_API;
    const INTERNAL_API_KEY = process.env.INTERNAL_API_KEY;

    if (!RENDER_GATEWAY_INTERNAL_API || !INTERNAL_API_KEY) {
      console.error('Gateway internal API endpoint or key not configured');
      return res.status(500).json({
        error: 'Gateway internal API endpoint or key not configured',
        message: 'Please configure RENDER_GATEWAY_INTERNAL_API and INTERNAL_API_KEY environment variables'
      });
    }

    // Fetch from persistent Render gateway
    console.log('Fetching peers from persistent gateway:', RENDER_GATEWAY_INTERNAL_API);
    
    const response = await axios.get(RENDER_GATEWAY_INTERNAL_API, {
      headers: {
        'Authorization': `Bearer ${INTERNAL_API_KEY}`,
        'User-Agent': 'KNIRVGATEWAY-Vercel/1.0.0'
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

    return res.status(200).json(dhtPeers);

  } catch (error) {
    console.error('Error in Vercel provision function:', error);

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

    return res.status(statusCode).json({
      error: errorMessage,
      details: error.message,
      timestamp: Date.now(),
      mode: 'vercel-serverless'
    });
  }
}
