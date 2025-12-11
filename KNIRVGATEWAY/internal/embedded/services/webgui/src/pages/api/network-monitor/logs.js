import { getMonitorInstance } from '../../../lib/network-monitor';

export default async function handler(req, res) {
  // Only allow Root admin access
  const userRole = req.headers['x-knirv-role'] || 'General';
  if (userRole !== 'Root') {
    return res.status(403).json({
      success: false,
      error: 'Forbidden: Root admin access required'
    });
  }

  if (req.method !== 'GET') {
    return res.status(405).json({
      success: false,
      error: 'Method not allowed'
    });
  }

  try {
    const monitor = getMonitorInstance();
    const { limit = 100 } = req.query;

    const logs = monitor.getLogs(parseInt(limit));

    return res.status(200).json({
      success: true,
      data: logs,
      timestamp: new Date().toISOString()
    });
  } catch (error) {
    console.error('Error fetching logs:', error.message);

    return res.status(500).json({
      success: false,
      error: error.message,
      timestamp: new Date().toISOString()
    });
  }
}
