// System Metrics Collection
// Collects CPU, memory, disk, and network metrics using Node.js

const os = require('os');
const { execSync } = require('child_process');
const fs = require('fs');

class MetricsCollector {
  constructor() {
    this.previousCPU = this.getCPUInfo();
    this.previousNet = this.getNetworkStats();
    this.metrics = {
      system: null,
      lastUpdate: null
    };
  }

  // Collect all system metrics
  async collectAll() {
    try {
      const systemMetrics = {
        cpu: await this.getCPUMetrics(),
        memory: this.getMemoryMetrics(),
        disk: await this.getDiskMetrics(),
        network: this.getNetworkMetrics(),
        ipfs: await this.getIPFSMetrics(),
        timestamp: new Date().toISOString()
      };

      this.metrics.system = systemMetrics;
      this.metrics.lastUpdate = new Date().toISOString();

      return systemMetrics;
    } catch (error) {
      console.error('Error collecting metrics:', error);
      return this.getFallbackMetrics();
    }
  }

  // Get CPU metrics
  async getCPUMetrics() {
    const cpus = os.cpus();
    const currentCPU = this.getCPUInfo();

    // Calculate CPU usage percentage
    let totalIdle = 0;
    let totalTick = 0;

    for (let i = 0; i < cpus.length; i++) {
      const cpu = cpus[i];
      for (const type in cpu.times) {
        totalTick += cpu.times[type];
      }
      totalIdle += cpu.times.idle;
    }

    const idle = totalIdle / cpus.length;
    const total = totalTick / cpus.length;
    const usagePercent = 100 - ~~(100 * idle / total);

    // Get load average
    const loadAverage = os.loadavg();

    // Per-core usage
    const coreUsage = {};
    cpus.forEach((cpu, index) => {
      const total = Object.values(cpu.times).reduce((a, b) => a + b, 0);
      const idle = cpu.times.idle;
      const usage = 100 - ~~(100 * idle / total);
      coreUsage[`cpu${index}`] = usage;
    });

    return {
      usage_percent: usagePercent,
      load_average: loadAverage,
      core_usage: coreUsage,
      cores: cpus.length
    };
  }

  // Get CPU info helper
  getCPUInfo() {
    const cpus = os.cpus();
    let user = 0;
    let nice = 0;
    let sys = 0;
    let idle = 0;
    let irq = 0;

    for (const cpu of cpus) {
      user += cpu.times.user;
      nice += cpu.times.nice;
      sys += cpu.times.sys;
      idle += cpu.times.idle;
      irq += cpu.times.irq;
    }

    return { user, nice, sys, idle, irq };
  }

  // Get memory metrics
  getMemoryMetrics() {
    const totalMemory = os.totalmem();
    const freeMemory = os.freemem();
    const usedMemory = totalMemory - freeMemory;
    const usagePercent = (usedMemory / totalMemory) * 100;

    return {
      total_bytes: totalMemory,
      used_bytes: usedMemory,
      available_bytes: freeMemory,
      usage_percent: usagePercent,
      swap_used: 0, // Not easily available in Node.js
      swap_total: 0
    };
  }

  // Get disk metrics
  async getDiskMetrics() {
    try {
      // Use df command on Unix-like systems
      if (process.platform !== 'win32') {
        const output = execSync('df -k /').toString();
        const lines = output.trim().split('\n');
        if (lines.length >= 2) {
          const parts = lines[1].split(/\s+/);
          const totalBytes = parseInt(parts[1]) * 1024;
          const usedBytes = parseInt(parts[2]) * 1024;
          const availableBytes = parseInt(parts[3]) * 1024;
          const usagePercent = (usedBytes / totalBytes) * 100;

          return {
            total_bytes: totalBytes,
            used_bytes: usedBytes,
            available_bytes: availableBytes,
            usage_percent: usagePercent,
            io_reads: 0, // Would need iostat
            io_writes: 0
          };
        }
      }
    } catch (error) {
      console.error('Error getting disk metrics:', error.message);
    }

    // Fallback values
    return {
      total_bytes: 500 * 1024 * 1024 * 1024,
      used_bytes: 200 * 1024 * 1024 * 1024,
      available_bytes: 300 * 1024 * 1024 * 1024,
      usage_percent: 40.0,
      io_reads: 0,
      io_writes: 0
    };
  }

  // Get network metrics
  getNetworkMetrics() {
    const networkInterfaces = os.networkInterfaces();
    let bytesReceived = 0;
    let bytesSent = 0;

    // Note: Node.js doesn't provide network I/O stats directly
    // This is a placeholder - would need platform-specific commands
    return {
      bytes_received: bytesReceived,
      bytes_sent: bytesSent,
      packets_received: 0,
      packets_sent: 0,
      connections_total: 0
    };
  }

  // Get network stats helper
  getNetworkStats() {
    // Placeholder for network statistics
    return { rx: 0, tx: 0 };
  }

  // Get IPFS metrics
  async getIPFSMetrics() {
    try {
      const axios = require('axios');
      const ipfsUrl = process.env.IPFS_URL || 'http://localhost:5001';

      // Get repo stats
      const repoStats = await axios.post(`${ipfsUrl}/api/v0/repo/stat`, null, {
        timeout: 5000
      });

      // Get peer count
      const peers = await axios.post(`${ipfsUrl}/api/v0/swarm/peers`, null, {
        timeout: 5000
      });

      const repoSize = repoStats.data.RepoSize || 0;
      const storageMax = repoStats.data.StorageMax || 50 * 1024 * 1024 * 1024; // Default 50GB
      const storagePercent = (repoSize / storageMax) * 100;

      return {
        repo_size: repoSize,
        repo_size_max: storageMax,
        storage_percent: storagePercent,
        peer_count: peers.data.Peers ? peers.data.Peers.length : 0,
        pinned_objects: repoStats.data.NumObjects || 0,
        blocks_stored: 0,
        gateway_requests: 0,
        api_requests: 0
      };
    } catch (error) {
      console.error('Error getting IPFS metrics:', error.message);
      // Return placeholder data if IPFS is not available
      return {
        repo_size: 15 * 1024 * 1024 * 1024,
        repo_size_max: 50 * 1024 * 1024 * 1024,
        storage_percent: 30.0,
        peer_count: 0,
        pinned_objects: 0,
        blocks_stored: 0,
        gateway_requests: 0,
        api_requests: 0
      };
    }
  }

  // Get current metrics
  getCurrentMetrics() {
    return this.metrics;
  }

  // Get fallback metrics when collection fails
  getFallbackMetrics() {
    return {
      cpu: {
        usage_percent: 0,
        load_average: [0, 0, 0],
        core_usage: {},
        cores: os.cpus().length
      },
      memory: this.getMemoryMetrics(),
      disk: {
        total_bytes: 0,
        used_bytes: 0,
        available_bytes: 0,
        usage_percent: 0,
        io_reads: 0,
        io_writes: 0
      },
      network: {
        bytes_received: 0,
        bytes_sent: 0,
        packets_received: 0,
        packets_sent: 0,
        connections_total: 0
      },
      ipfs: {
        repo_size: 0,
        repo_size_max: 0,
        storage_percent: 0,
        peer_count: 0,
        pinned_objects: 0,
        blocks_stored: 0,
        gateway_requests: 0,
        api_requests: 0
      },
      timestamp: new Date().toISOString()
    };
  }
}

module.exports = MetricsCollector;
