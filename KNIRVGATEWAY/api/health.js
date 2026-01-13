/**
 * Vercel Function: /health endpoint
 * 
 * Provides health status for the serverless oracle instance.
 */

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
    const status = {
      status: 'healthy',
      mode: 'vercel-serverless',
      timestamp: Date.now(),
      chainId: process.env.KNIRV_CHAIN_ID || 'testnet',
      version: '1.0.0',
      instance: 'vercel-function',
      capabilities: ['provision', 'health-check'],
      uptime: process.uptime(),
      environment: {
        nodeVersion: process.version,
        platform: process.platform,
        arch: process.arch
      }
    };

    return res.status(200).json(status);
  } catch (error) {
    console.error('Health check error:', error);
    return res.status(500).json({
      status: 'unhealthy',
      error: error.message,
      timestamp: Date.now()
    });
  }
}
