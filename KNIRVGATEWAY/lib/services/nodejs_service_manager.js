/**
 * KNIRVORACLE Node.js Service Manager
 * 
 * Manages embedded Node.js services migrated from KNIRVORACLE:
 * - Payment Gateway
 * - Tunnel Registry  
 * - Operator Registry
 * - WebGUI
 */

import { spawn } from 'child_process';
import { fileURLToPath } from 'url';
import path from 'path';
import fs from 'fs';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

class NodeJSServiceManager {
  constructor(config = {}) {
    this.config = {
      enabled: config.enabled !== false,
      services: {
        paymentGateway: {
          enabled: config.paymentGateway?.enabled !== false,
          port: config.paymentGateway?.port || 3001,
          scriptPath: config.paymentGateway?.scriptPath || 'services/payment-oracle/server.js'
        },
        tunnelRegistry: {
          enabled: config.tunnelRegistry?.enabled !== false,
          httpPort: config.tunnelRegistry?.httpPort || 3002,
          controlPort: config.tunnelRegistry?.controlPort || 3003,
          publicRelayPort: config.tunnelRegistry?.publicRelayPort || 3004,
          stunPort: config.tunnelRegistry?.stunPort || 3005,
          scriptPath: config.tunnelRegistry?.scriptPath || 'services/tunnel-registry/server.js'
        },
        operatorRegistry: {
          enabled: false, // Temporarily disabled due to port conflicts
          port: config.operatorRegistry?.port || 3006,
          scriptPath: config.operatorRegistry?.scriptPath || 'services/operator-registry/registry-service.js',
          useNpm: true,
          workingDir: 'services/operator-registry'
        },
        webgui: {
          enabled: config.webgui?.enabled !== false,
          port: config.webgui?.port || 3007,
          scriptPath: config.webgui?.scriptPath || 'services/webgui/server.js'
        }
      },
      chainId: config.chainId || process.env.KNIRV_CHAIN_ID || 'testnet',
      nodeId: config.nodeId || 'knirvoracle-node',
      ...config
    };
    
    this.processes = new Map();
    this.isStarted = false;
    this.startupInProgress = false;
  }

  /**
   * Find an available port starting from the given port
   */
  async findAvailablePort(startPort) {
    const net = await import('net');
    
    return new Promise((resolve) => {
      const server = net.createServer();
      server.listen(startPort, () => {
        const port = server.address().port;
        server.close(() => resolve(port));
      });
      server.on('error', () => {
        resolve(this.findAvailablePort(startPort + 1));
      });
    });
  }

  /**
   * Start a specific Node.js service
   */
  async startService(serviceName, serviceConfig) {
    if (this.processes.has(serviceName)) {
      console.log(`[NodeJSServiceManager] Service ${serviceName} is already running`);
      return false;
    }

    const scriptPath = path.resolve(serviceConfig.scriptPath);
    
    // Check if script exists
    if (!fs.existsSync(scriptPath)) {
      console.error(`[NodeJSServiceManager] Script not found for ${serviceName}: ${scriptPath}`);
      return false;
    }

    try {
      // Find available ports
      const ports = {};
      if (serviceConfig.port) {
        ports.port = await this.findAvailablePort(serviceConfig.port);
      }
      if (serviceConfig.httpPort) {
        ports.httpPort = await this.findAvailablePort(serviceConfig.httpPort);
      }
      if (serviceConfig.controlPort) {
        ports.controlPort = await this.findAvailablePort(serviceConfig.controlPort);
      }
      if (serviceConfig.publicRelayPort) {
        ports.publicRelayPort = await this.findAvailablePort(serviceConfig.publicRelayPort);
      }
      if (serviceConfig.stunPort) {
        ports.stunPort = await this.findAvailablePort(serviceConfig.stunPort);
      }

      // Prepare environment variables
      const currentEnv = globalThis.process?.env || {};
      const env = {
        ...currentEnv,
        KNIRV_CHAIN_ID: this.config.chainId,
        NODE_ID: this.config.nodeId,
        NODE_ENV: currentEnv.NODE_ENV || 'production'
      };

      // Add service-specific environment variables
      if (ports.port) env.PORT = ports.port;
      if (ports.httpPort) env.HTTP_API_PORT = ports.httpPort;
      if (ports.controlPort) env.CONTROL_PORT = ports.controlPort;
      if (ports.publicRelayPort) env.PUBLIC_RELAY_PORT = ports.publicRelayPort;
      if (ports.stunPort) env.STUN_PORT = ports.stunPort;

      // Additional service-specific configuration
      if (serviceName === 'tunnelRegistry') {
        env.PUBLIC_HOST = this.config.publicHost || 'localhost';
        env.RELAY_SERVER_PEER_ID = this.config.nodeId;
      }

      console.log(`[NodeJSServiceManager] Starting ${serviceName} service...`);
      console.log(`[NodeJSServiceManager] Script: ${scriptPath}`);
      console.log(`[NodeJSServiceManager] Ports:`, ports);

      // Start the process
      let process;
      if (serviceConfig.useNpm) {
        // Use npm start for services that need it
        process = spawn('npm', ['start'], {
          env,
          cwd: serviceConfig.workingDir || path.dirname(scriptPath),
          stdio: ['pipe', 'pipe', 'pipe']
        });
      } else {
        // Direct node execution
        process = spawn('node', [scriptPath], {
          env,
          cwd: path.dirname(scriptPath),
          stdio: ['pipe', 'pipe', 'pipe']
        });
      }

      // Store process info
      this.processes.set(serviceName, {
        process,
        config: serviceConfig,
        ports,
        startTime: Date.now()
      });

      // Handle process output
      process.stdout.on('data', (data) => {
        console.log(`[${serviceName}] ${data.toString().trim()}`);
      });

      process.stderr.on('data', (data) => {
        console.error(`[${serviceName}] ERROR: ${data.toString().trim()}`);
      });

      process.on('close', (code) => {
        console.log(`[NodeJSServiceManager] Service ${serviceName} exited with code ${code}`);
        this.processes.delete(serviceName);
      });

      process.on('error', (error) => {
        console.error(`[NodeJSServiceManager] Failed to start ${serviceName}:`, error);
        this.processes.delete(serviceName);
      });

      console.log(`[NodeJSServiceManager] Service ${serviceName} started with PID ${process.pid}`);
      return true;

    } catch (error) {
      console.error(`[NodeJSServiceManager] Error starting ${serviceName}:`, error);
      return false;
    }
  }

  /**
   * Stop a specific service
   */
  async stopService(serviceName) {
    const serviceInfo = this.processes.get(serviceName);
    if (!serviceInfo) {
      console.log(`[NodeJSServiceManager] Service ${serviceName} is not running`);
      return false;
    }

    try {
      console.log(`[NodeJSServiceManager] Stopping ${serviceName} service...`);
      serviceInfo.process.kill('SIGTERM');
      
      // Wait for graceful shutdown
      await new Promise((resolve) => {
        const timeout = setTimeout(() => {
          console.log(`[NodeJSServiceManager] Force killing ${serviceName} service`);
          serviceInfo.process.kill('SIGKILL');
          resolve();
        }, 5000);

        serviceInfo.process.on('close', () => {
          clearTimeout(timeout);
          resolve();
        });
      });

      this.processes.delete(serviceName);
      console.log(`[NodeJSServiceManager] Service ${serviceName} stopped`);
      return true;

    } catch (error) {
      console.error(`[NodeJSServiceManager] Error stopping ${serviceName}:`, error);
      return false;
    }
  }

  /**
   * Start all enabled services
   */
  async startAllServices() {
    if (this.startupInProgress) {
      console.log('[NodeJSServiceManager] Startup already in progress');
      return false;
    }

    if (!this.config.enabled) {
      console.log('[NodeJSServiceManager] Node.js services are disabled');
      return false;
    }

    this.startupInProgress = true;
    console.log('[NodeJSServiceManager] Starting all enabled Node.js services...');

    const results = [];

    // Start services in order
    for (const [serviceName, serviceConfig] of Object.entries(this.config.services)) {
      if (serviceConfig.enabled) {
        const result = await this.startService(serviceName, serviceConfig);
        results.push({ serviceName, success: result });
        
        // Small delay between service starts
        await new Promise(resolve => setTimeout(resolve, 1000));
      }
    }

    this.isStarted = true;
    this.startupInProgress = false;

    const successCount = results.filter(r => r.success).length;
    console.log(`[NodeJSServiceManager] Started ${successCount}/${results.length} services`);

    return successCount > 0;
  }

  /**
   * Stop all services
   */
  async stopAllServices() {
    console.log('[NodeJSServiceManager] Stopping all Node.js services...');

    const stopPromises = Array.from(this.processes.keys()).map(serviceName => 
      this.stopService(serviceName)
    );

    await Promise.all(stopPromises);
    
    this.isStarted = false;
    console.log('[NodeJSServiceManager] All Node.js services stopped');
  }

  /**
   * Get status of all services
   */
  getServicesStatus() {
    const status = {
      enabled: this.config.enabled,
      isStarted: this.isStarted,
      startupInProgress: this.startupInProgress,
      services: {}
    };

    for (const [serviceName, serviceConfig] of Object.entries(this.config.services)) {
      const processInfo = this.processes.get(serviceName);
      status.services[serviceName] = {
        enabled: serviceConfig.enabled,
        running: !!processInfo,
        pid: processInfo?.process.pid,
        ports: processInfo?.ports,
        uptime: processInfo ? Date.now() - processInfo.startTime : 0
      };
    }

    return status;
  }

  /**
   * Get service endpoints for external access
   */
  getServiceEndpoints() {
    const endpoints = {};

    for (const [serviceName, processInfo] of this.processes) {
      if (processInfo.ports) {
        endpoints[serviceName] = {};
        
        if (processInfo.ports.port) {
          endpoints[serviceName].main = `http://localhost:${processInfo.ports.port}`;
        }
        if (processInfo.ports.httpPort) {
          endpoints[serviceName].api = `http://localhost:${processInfo.ports.httpPort}`;
        }
      }
    }

    return endpoints;
  }
}

export { NodeJSServiceManager };
