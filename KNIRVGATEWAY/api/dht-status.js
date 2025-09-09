/**
 * Vercel Function: /dht/status endpoint
 * 
 * Provides DHT status information for the serverless gateway instance.
 * In serverless mode, this proxies to the persistent gateway.
 */

import axios from 'axios';

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
    // In serverless mode, we proxy to the persistent gateway
    const RENDER_GATEWAY_URL = process.env.RENDER_GATEWAY_URL;
    const INTERNAL_API_KEY = process.env.INTERNAL_API_KEY;

    if (!RENDER_GATEWAY_URL) {
      return res.status(200).json({
        status: 'serverless',
        mode: 'vercel-serverless',
        message: 'DHT status not available in serverless mode',
        timestamp: Date.now(),
        persistent_gateway: 'not_configured'
      });
    }

    // Try to get status from persistent gateway
    try {
      const response = await axios.get(`${RENDER_GATEWAY_URL}/dht/status`, {
        headers: INTERNAL_API_KEY ? {
          'Authorization': `Bearer ${INTERNAL_API_KEY}`
        } : {},
        timeout: 5000
      });

      return res.status(200).json({
        ...response.data,
        proxy_mode: 'vercel-serverless',
        proxy_timestamp: Date.now()
      });
    } catch (proxyError) {
      console.error('Failed to proxy DHT status:', proxyError.message);
      
      return res.status(503).json({
        status: 'unavailable',
        mode: 'vercel-serverless',
        error: 'Persistent gateway unavailable',
        details: proxyError.message,
        timestamp: Date.now()
      });
    }
  } catch (error) {
    console.error('DHT status error:', error);
    return res.status(500).json({
      status: 'error',
      error: error.message,
      timestamp: Date.now()
    });
  }
}
