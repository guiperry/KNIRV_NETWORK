#!/usr/bin/env node

/**
 * KNIRV-CONTROLLER Unified Orchestrator
 * Manages the integration and communication between manager, CLI, and receiver components
 */

import { spawn } from 'child_process';
import { createServer } from 'http';
import { WebSocketServer } from 'ws';
import express from 'express';
import cors from 'cors';
import helmet from 'helmet';
import compression from 'compression';
import { fileURLToPath } from 'url';
import { dirname, join } from 'path';
import fs from 'fs/promises';

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);
const rootDir = join(__dirname, '..');

class KNIRVControllerOrchestrator {
  constructor() {
    this.processes = new Map();
    this.app = express();
    this.server = null;
    this.wss = null;
    this.config = {
      ports: {
        orchestrator: 3000,
        manager: 3001,
        receiver: 3002,
        cli: 3003
      },
      components: {
        manager: { enabled: true, path: 'manager' },
        receiver: { enabled: true, path: 'receiver' },
        cli: { enabled: true, path: 'cli' }
      }
    };
    this.setupExpress();
  }

  setupExpress() {
    this.app.use(helmet());
    this.app.use(compression());
    this.app.use(cors());
    this.app.use(express.json());
    this.app.use(express.static(join(rootDir, 'public')));

    // Health check endpoint
    this.app.get('/health', (req, res) => {
      const status = {
        orchestrator: 'running',
        components: {}
      };
      
      for (const [name, process] of this.processes) {
        status.components[name] = process.killed ? 'stopped' : 'running';
      }
      
      res.json(status);
    });

    // Component management endpoints
    this.app.post('/components/:name/start', async (req, res) => {
      try {
        await this.startComponent(req.params.name);
        res.json({ success: true, message: `Component ${req.params.name} started` });
      } catch (error) {
        res.status(500).json({ success: false, error: error.message });
      }
    });

    this.app.post('/components/:name/stop', async (req, res) => {
      try {
        await this.stopComponent(req.params.name);
        res.json({ success: true, message: `Component ${req.params.name} stopped` });
      } catch (error) {
        res.status(500).json({ success: false, error: error.message });
      }
    });

    // Component communication proxy
    this.app.use('/api/:component/*', (req, res) => {
      const component = req.params.component;
      const port = this.config.ports[component];
      
      if (!port) {
        return res.status(404).json({ error: 'Component not found' });
      }

      // Proxy request to component
      const targetUrl = `http://localhost:${port}${req.originalUrl.replace(`/api/${component}`, '')}`;
      // Implementation would include actual proxying logic
      res.json({ message: 'Proxy endpoint', target: targetUrl });
    });
  }

  async startComponent(name) {
    if (this.processes.has(name)) {
      throw new Error(`Component ${name} is already running`);
    }

    const component = this.config.components[name];
    if (!component || !component.enabled) {
      throw new Error(`Component ${name} is not configured or disabled`);
    }

    console.log(`Starting component: ${name}`);
    
    let process;
    const componentPath = join(rootDir, component.path);
    
    switch (name) {
      case 'manager':
        process = spawn('npm', ['run', 'dev'], {
          cwd: componentPath,
          stdio: 'pipe',
          env: { ...process.env, PORT: this.config.ports.manager }
        });
        break;
        
      case 'receiver':
        process = spawn('npm', ['run', 'dev'], {
          cwd: componentPath,
          stdio: 'pipe',
          env: { ...process.env, PORT: this.config.ports.receiver }
        });
        break;
        
      case 'cli':
        process = spawn('go', ['run', 'main.go', '--port', this.config.ports.cli], {
          cwd: componentPath,
          stdio: 'pipe'
        });
        break;
        
      default:
        throw new Error(`Unknown component: ${name}`);
    }

    process.stdout.on('data', (data) => {
      console.log(`[${name}] ${data.toString().trim()}`);
      this.broadcastMessage({ type: 'log', component: name, data: data.toString() });
    });

    process.stderr.on('data', (data) => {
      console.error(`[${name}] ERROR: ${data.toString().trim()}`);
      this.broadcastMessage({ type: 'error', component: name, data: data.toString() });
    });

    process.on('close', (code) => {
      console.log(`[${name}] Process exited with code ${code}`);
      this.processes.delete(name);
      this.broadcastMessage({ type: 'exit', component: name, code });
    });

    this.processes.set(name, process);
    return process;
  }

  async stopComponent(name) {
    const process = this.processes.get(name);
    if (!process) {
      throw new Error(`Component ${name} is not running`);
    }

    console.log(`Stopping component: ${name}`);
    process.kill('SIGTERM');
    
    // Wait for graceful shutdown
    await new Promise((resolve) => {
      const timeout = setTimeout(() => {
        process.kill('SIGKILL');
        resolve();
      }, 5000);
      
      process.on('close', () => {
        clearTimeout(timeout);
        resolve();
      });
    });

    this.processes.delete(name);
  }

  setupWebSocket() {
    this.wss = new WebSocketServer({ server: this.server });
    
    this.wss.on('connection', (ws) => {
      console.log('WebSocket client connected');
      
      ws.on('message', (message) => {
        try {
          const data = JSON.parse(message.toString());
          this.handleWebSocketMessage(ws, data);
        } catch (error) {
          ws.send(JSON.stringify({ type: 'error', message: 'Invalid JSON' }));
        }
      });

      ws.on('close', () => {
        console.log('WebSocket client disconnected');
      });

      // Send initial status
      ws.send(JSON.stringify({
        type: 'status',
        components: Array.from(this.processes.keys())
      }));
    });
  }

  handleWebSocketMessage(ws, data) {
    switch (data.type) {
      case 'start_component':
        this.startComponent(data.component).catch(error => {
          ws.send(JSON.stringify({ type: 'error', message: error.message }));
        });
        break;
        
      case 'stop_component':
        this.stopComponent(data.component).catch(error => {
          ws.send(JSON.stringify({ type: 'error', message: error.message }));
        });
        break;
        
      default:
        ws.send(JSON.stringify({ type: 'error', message: 'Unknown message type' }));
    }
  }

  broadcastMessage(message) {
    if (this.wss) {
      this.wss.clients.forEach(client => {
        if (client.readyState === 1) { // WebSocket.OPEN
          client.send(JSON.stringify(message));
        }
      });
    }
  }

  async start() {
    console.log('Starting KNIRV-CONTROLLER Orchestrator...');
    
    this.server = createServer(this.app);
    this.setupWebSocket();
    
    this.server.listen(this.config.ports.orchestrator, () => {
      console.log(`Orchestrator running on port ${this.config.ports.orchestrator}`);
    });

    // Start all enabled components
    for (const [name, component] of Object.entries(this.config.components)) {
      if (component.enabled) {
        try {
          await this.startComponent(name);
          // Wait a bit between component starts
          await new Promise(resolve => setTimeout(resolve, 2000));
        } catch (error) {
          console.error(`Failed to start component ${name}:`, error.message);
        }
      }
    }

    console.log('All components started successfully');
  }

  async stop() {
    console.log('Stopping KNIRV-CONTROLLER Orchestrator...');
    
    // Stop all components
    for (const name of this.processes.keys()) {
      try {
        await this.stopComponent(name);
      } catch (error) {
        console.error(`Failed to stop component ${name}:`, error.message);
      }
    }

    // Close server
    if (this.server) {
      this.server.close();
    }

    console.log('Orchestrator stopped');
  }
}

// Handle process signals
const orchestrator = new KNIRVControllerOrchestrator();

process.on('SIGINT', async () => {
  console.log('\nReceived SIGINT, shutting down gracefully...');
  await orchestrator.stop();
  process.exit(0);
});

process.on('SIGTERM', async () => {
  console.log('\nReceived SIGTERM, shutting down gracefully...');
  await orchestrator.stop();
  process.exit(0);
});

// Start the orchestrator
orchestrator.start().catch(error => {
  console.error('Failed to start orchestrator:', error);
  process.exit(1);
});
