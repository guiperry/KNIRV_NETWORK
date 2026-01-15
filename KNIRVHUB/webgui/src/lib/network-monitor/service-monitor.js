// Service Health Monitoring
// Implements service health checking and status tracking

const axios = require('axios');

class ServiceMonitor {
  constructor(service) {
    this.service = service;
    this.status = {
      name: service.name,
      url: service.url,
      status: 'unknown',
      lastCheck: null,
      responseTime: null,
      error: null,
      critical: service.critical,
      uptime: 0,
      totalChecks: 0,
      successfulChecks: 0
    };
    this.startTime = Date.now();
  }

  // Perform health check
  async check() {
    const startTime = Date.now();
    this.status.lastCheck = new Date().toISOString();
    this.status.totalChecks++;

    try {
      const healthUrl = `${this.service.url}${this.service.healthEndpoint}`;

      const response = await axios.get(healthUrl, {
        timeout: this.service.timeout,
        validateStatus: (status) => status >= 200 && status < 500
      });

      const responseTime = Date.now() - startTime;
      this.status.responseTime = responseTime;

      // Check if response is healthy
      if (response.status >= 200 && response.status < 300) {
        this.status.status = 'up';
        this.status.error = null;
        this.status.successfulChecks++;
      } else {
        this.status.status = 'degraded';
        this.status.error = `HTTP ${response.status}`;
      }

      // Calculate uptime percentage
      this.status.uptime = (this.status.successfulChecks / this.status.totalChecks) * 100;

      return this.status;
    } catch (error) {
      const responseTime = Date.now() - startTime;
      this.status.responseTime = responseTime;
      this.status.status = 'down';
      this.status.error = error.message;
      this.status.uptime = (this.status.successfulChecks / this.status.totalChecks) * 100;

      return this.status;
    }
  }

  // Get current status
  getStatus() {
    return { ...this.status };
  }

  // Collect additional metrics if available
  async collectMetrics() {
    if (!this.service.metricsEndpoint) {
      return null;
    }

    try {
      const metricsUrl = `${this.service.url}${this.service.metricsEndpoint}`;
      const response = await axios.get(metricsUrl, {
        timeout: this.service.timeout
      });

      // Parse metrics based on service type
      if (this.service.type === 'ipfs') {
        return this.parseIPFSMetrics(response.data);
      } else if (this.service.type === 'cosmos') {
        return this.parseCosmosMetrics(response.data);
      }

      return response.data;
    } catch (error) {
      console.error(`Failed to collect metrics for ${this.service.name}:`, error.message);
      return null;
    }
  }

  // Parse IPFS metrics
  parseIPFSMetrics(data) {
    return {
      repoSize: data.RepoSize || 0,
      storageMax: data.StorageMax || 0,
      numObjects: data.NumObjects || 0,
      repoPath: data.RepoPath,
      version: data.Version
    };
  }

  // Parse Cosmos chain metrics
  parseCosmosMetrics(data) {
    // Cosmos chains typically expose Prometheus metrics
    // This is a simplified parser
    return {
      raw: data
    };
  }

  // Reset statistics
  resetStats() {
    this.status.totalChecks = 0;
    this.status.successfulChecks = 0;
    this.status.uptime = 0;
    this.startTime = Date.now();
  }
}

module.exports = ServiceMonitor;
