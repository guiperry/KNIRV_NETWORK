// Log Management
// Aggregates and manages logs from various services

class LogManager {
  constructor(config) {
    this.config = config;
    this.logs = [];
    this.maxEntries = config.logging.maxEntries || 10000;
  }

  // Add a log entry
  addLog(entry) {
    const logEntry = {
      '@timestamp': entry.timestamp || new Date().toISOString(),
      level: entry.level || 'info',
      message: entry.message,
      service: entry.service,
      source: entry.source || 'network-monitor',
      fields: entry.fields || {},
      tags: entry.tags || [],
      host: entry.host || 'localhost',
      network: entry.network || this.config.activeNetwork
    };

    this.logs.unshift(logEntry); // Add to beginning

    // Trim to max entries
    if (this.logs.length > this.maxEntries) {
      this.logs = this.logs.slice(0, this.maxEntries);
    }
  }

  // Get recent logs
  getRecentLogs(limit = 100) {
    return this.logs.slice(0, limit);
  }

  // Search logs
  searchLogs(query) {
    let results = this.logs;

    // Filter by services
    if (query.services && query.services.length > 0) {
      results = results.filter(log => query.services.includes(log.service));
    }

    // Filter by level
    if (query.level) {
      results = results.filter(log => log.level === query.level);
    }

    // Filter by message
    if (query.message) {
      results = results.filter(log =>
        log.message.toLowerCase().includes(query.message.toLowerCase())
      );
    }

    // Filter by time range
    if (query.startTime) {
      const startTime = new Date(query.startTime);
      results = results.filter(log => new Date(log['@timestamp']) >= startTime);
    }

    if (query.endTime) {
      const endTime = new Date(query.endTime);
      results = results.filter(log => new Date(log['@timestamp']) <= endTime);
    }

    // Apply offset and limit
    const offset = query.offset || 0;
    const limit = query.limit || 100;

    return {
      entries: results.slice(offset, offset + limit),
      total: results.length,
      query: query,
      duration: '1ms',
      timestamp: new Date().toISOString()
    };
  }

  // Clear old logs based on retention policy
  clearOldLogs() {
    const retentionMs = this.config.logging.retention;
    const cutoffTime = new Date(Date.now() - retentionMs);

    this.logs = this.logs.filter(log => {
      return new Date(log['@timestamp']) >= cutoffTime;
    });
  }

  // Generate sample logs for testing
  generateSampleLogs() {
    const services = ['knirvoracle', 'knirvchain', 'knirvgraph', 'knirvnexus', 'ipfs'];
    const levels = ['info', 'warn', 'error', 'debug'];
    const messages = [
      'Service started successfully',
      'Processing request',
      'Database connection established',
      'Cache miss, fetching from source',
      'Request completed',
      'Peer connected',
      'Block validated',
      'Transaction processed'
    ];

    for (let i = 0; i < 20; i++) {
      this.addLog({
        timestamp: new Date(Date.now() - i * 60000).toISOString(), // 1 minute intervals
        level: levels[Math.floor(Math.random() * levels.length)],
        message: messages[Math.floor(Math.random() * messages.length)],
        service: services[Math.floor(Math.random() * services.length)]
      });
    }
  }
}

module.exports = LogManager;
