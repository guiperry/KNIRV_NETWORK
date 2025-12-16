// Network Monitor - Main Entry Point
// Pure Node.js implementation of network monitoring

const { config, getActiveNetwork, getService } = require('./config');
const ServiceMonitor = require('./service-monitor');
const MetricsCollector = require('./metrics-collector');
const LogManager = require('./log-manager');

class NetworkMonitor {
  constructor() {
    this.config = config;
    this.serviceMonitors = new Map();
    this.metricsCollector = new MetricsCollector();
    this.logManager = new LogManager(config);
    this.isRunning = false;
    this.intervals = [];

    this.initialize();
  }

  // Initialize the monitor
  initialize() {
    const network = getActiveNetwork();

    // Create service monitors
    for (const service of network.services) {
      const monitor = new ServiceMonitor(service);
      this.serviceMonitors.set(service.name, monitor);
    }

    // Generate some sample logs
    this.logManager.generateSampleLogs();

    // Log initialization
    this.logManager.addLog({
      level: 'info',
      message: `Network monitor initialized for ${network.name}`,
      service: 'network-monitor'
    });
  }

  // Start monitoring
  start() {
    if (this.isRunning) {
      return;
    }

    this.isRunning = true;
    const network = getActiveNetwork();

    // Perform initial checks
    this.checkAllServices();
    this.metricsCollector.collectAll();

    // Set up periodic service checks
    for (const [name, monitor] of this.serviceMonitors) {
      const interval = setInterval(() => {
        this.checkService(name);
      }, monitor.service.checkInterval);

      this.intervals.push(interval);
    }

    // Set up periodic metrics collection
    const metricsInterval = setInterval(() => {
      this.metricsCollector.collectAll();
    }, this.config.monitoring.checkInterval);

    this.intervals.push(metricsInterval);

    // Set up periodic log cleanup
    const logCleanupInterval = setInterval(() => {
      this.logManager.clearOldLogs();
    }, 60 * 60 * 1000); // Every hour

    this.intervals.push(logCleanupInterval);

    this.logManager.addLog({
      level: 'info',
      message: `Network monitoring started for ${network.name}`,
      service: 'network-monitor'
    });
  }

  // Stop monitoring
  stop() {
    if (!this.isRunning) {
      return;
    }

    // Clear all intervals
    for (const interval of this.intervals) {
      clearInterval(interval);
    }

    this.intervals = [];
    this.isRunning = false;

    this.logManager.addLog({
      level: 'info',
      message: 'Network monitoring stopped',
      service: 'network-monitor'
    });
  }

  // Check all services
  async checkAllServices() {
    const promises = [];

    for (const [name, monitor] of this.serviceMonitors) {
      promises.push(this.checkService(name));
    }

    await Promise.all(promises);
  }

  // Check a specific service
  async checkService(serviceName) {
    const monitor = this.serviceMonitors.get(serviceName);
    if (!monitor) {
      throw new Error(`Service '${serviceName}' not found`);
    }

    try {
      const status = await monitor.check();

      // Log status changes
      if (status.status === 'down' && status.critical) {
        this.logManager.addLog({
          level: 'error',
          message: `Critical service ${serviceName} is down: ${status.error}`,
          service: serviceName,
          fields: { status: status.status, error: status.error }
        });
      } else if (status.status === 'degraded') {
        this.logManager.addLog({
          level: 'warn',
          message: `Service ${serviceName} is degraded: ${status.error}`,
          service: serviceName,
          fields: { status: status.status, error: status.error }
        });
      }

      return status;
    } catch (error) {
      this.logManager.addLog({
        level: 'error',
        message: `Error checking service ${serviceName}: ${error.message}`,
        service: serviceName
      });
      throw error;
    }
  }

  // Get network status
  getNetworkStatus() {
    const network = getActiveNetwork();
    const services = {};
    let servicesUp = 0;
    let servicesDown = 0;

    for (const [name, monitor] of this.serviceMonitors) {
      const status = monitor.getStatus();
      services[name] = status;

      if (status.status === 'up') {
        servicesUp++;
      } else if (status.status === 'down') {
        servicesDown++;
      }
    }

    const servicesTotal = this.serviceMonitors.size;
    let overallStatus = 'healthy';

    if (servicesDown > 0) {
      // Check if any critical services are down
      const criticalDown = Array.from(this.serviceMonitors.values())
        .some(m => m.status.status === 'down' && m.status.critical);

      if (criticalDown) {
        overallStatus = 'critical';
      } else if (servicesDown > servicesUp) {
        overallStatus = 'critical';
      } else {
        overallStatus = 'degraded';
      }
    }

    return {
      name: network.name,
      overall_status: overallStatus,
      services_up: servicesUp,
      services_down: servicesDown,
      services_total: servicesTotal,
      last_update: new Date().toISOString(),
      services: services
    };
  }

  // Get service status
  getServiceStatus(serviceName) {
    const monitor = this.serviceMonitors.get(serviceName);
    if (!monitor) {
      throw new Error(`Service '${serviceName}' not found`);
    }

    return monitor.getStatus();
  }

  // Get all services
  getAllServices() {
    const services = {};

    for (const [name, monitor] of this.serviceMonitors) {
      services[name] = monitor.getStatus();
    }

    return services;
  }

  // Get metrics
  getMetrics() {
    return this.metricsCollector.getCurrentMetrics();
  }

  // Collect metrics now
  async collectMetrics() {
    return await this.metricsCollector.collectAll();
  }

  // Get logs
  getLogs(limit = 100) {
    return this.logManager.getRecentLogs(limit);
  }

  // Search logs
  searchLogs(query) {
    return this.logManager.searchLogs(query);
  }

  // Get configuration
  getConfig() {
    const network = getActiveNetwork();

    return {
      active_network: this.config.activeNetwork,
      networks: {
        [this.config.activeNetwork]: {
          name: network.name,
          description: network.description,
          services: network.services.map(s => ({
            name: s.name,
            url: s.url,
            type: s.type,
            critical: s.critical
          }))
        }
      },
      gui: {
        refresh_interval: this.config.monitoring.checkInterval
      }
    };
  }

  // Get monitor statistics
  getStats() {
    return {
      is_running: this.isRunning,
      total_services: this.serviceMonitors.size,
      total_logs: this.logManager.logs.length,
      uptime: this.isRunning ? Date.now() - this.startTime : 0
    };
  }
}

// Create singleton instance
let monitorInstance = null;

function getMonitorInstance() {
  if (!monitorInstance) {
    monitorInstance = new NetworkMonitor();
    monitorInstance.start();
  }
  return monitorInstance;
}

module.exports = {
  NetworkMonitor,
  getMonitorInstance
};
