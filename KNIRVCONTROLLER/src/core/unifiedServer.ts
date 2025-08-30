/**
 * KNIRV-CONTROLLER Unified Server
 * Serves both backend API and receiver frontend
 * Exports templates to OS application data directory on startup
 */

import express from 'express';
import { createServer, Server } from 'http';

import cors from 'cors';
import helmet from 'helmet';
import compression from 'compression';
import path from 'path';
import { fileURLToPath } from 'url';
import pino from 'pino';
import { KNIRVCortexBackend } from './index.js';
import { TemplateExporter } from './utils/templateExporter.js';

// Jest-compatible module URL resolution
const getModuleUrl = () => {
  if (typeof process !== 'undefined' && process.env.NODE_ENV === 'test') {
    return 'file://' + process.cwd() + '/src/core/unifiedServer.ts';
  }
  try {
    const importMeta = eval('import.meta');
    if (importMeta && importMeta.url) {
      return importMeta.url;
    }
  } catch {
    // Fallback for CommonJS
    return 'file://' + __filename;
  }
  return 'file://' + process.cwd();
};

const __filename = fileURLToPath(getModuleUrl());
const __dirname = path.dirname(__filename);
const rootDir = path.join(__dirname, '..');

const logger = pino({
  name: 'knirv-controller-unified',
  level: process.env.LOG_LEVEL || 'info',
  transport: {
    target: 'pino-pretty',
    options: {
      colorize: true
    }
  }
});

export class KNIRVControllerUnifiedServer {
  private app: express.Application;
  private server: Server | undefined;
  private backend: KNIRVCortexBackend;
  private templateExporter: TemplateExporter;
  private port: number;
  private receiverDistPath: string;

  constructor() {
    this.port = parseInt(process.env.PORT || '3000');
    this.app = express();
    this.backend = new KNIRVCortexBackend();
    this.templateExporter = new TemplateExporter();
    this.receiverDistPath = path.join(rootDir, 'frontend', 'dist');
    
    this.setupMiddleware();
  }

  private setupMiddleware() {
    // Security and compression
    this.app.use(helmet({
      contentSecurityPolicy: {
        directives: {
          defaultSrc: ["'self'"],
          scriptSrc: ["'self'", "'unsafe-inline'", "'unsafe-eval'"],
          styleSrc: ["'self'", "'unsafe-inline'"],
          imgSrc: ["'self'", "data:", "blob:"],
          connectSrc: ["'self'", "ws:", "wss:"],
          fontSrc: ["'self'"],
          objectSrc: ["'none'"],
          mediaSrc: ["'self'"],
          frameSrc: ["'none'"],
        },
      },
    }));
    this.app.use(compression());
    
    // CORS for API endpoints
    this.app.use('/api', cors({
      origin: process.env.ALLOWED_ORIGINS?.split(',') || [
        'http://localhost:3000', 
        'http://localhost:3001', 
        'http://localhost:3002'
      ],
      credentials: true
    }));

    this.app.use(express.json({ limit: '50mb' }));
    this.app.use(express.raw({ type: 'application/octet-stream', limit: '50mb' }));

    // Request logging
    this.app.use((req, res, next) => {
      logger.info({ method: req.method, url: req.url, ip: req.ip }, 'Request received');
      next();
    });
  }

  private async exportTemplatesOnStartup() {
    logger.info('Exporting templates to application data directory...');
    
    try {
      await this.templateExporter.exportTemplates();
      await this.templateExporter.cleanupBackups();
      
      logger.info('Template export completed successfully');
      logger.info(`Templates available at: ${this.templateExporter.getTemplatesPath()}`);
      logger.info(`App data directory: ${this.templateExporter.getAppDataPath()}`);
      
    } catch (error) {
      logger.error(`Template export failed: ${error instanceof Error ? error.message : String(error)}`);
      // Don't fail startup if template export fails
      logger.warn('Continuing startup without template export...');
    }
  }

  private setupRoutes() {
    // Health check with template info
    this.app.get('/health', (req, res) => {
      res.json({
        status: 'healthy',
        timestamp: new Date().toISOString(),
        components: {
          backend: 'running',
          frontend: 'serving',
          templates: {
            path: this.templateExporter.getTemplatesPath(),
            appData: this.templateExporter.getAppDataPath()
          }
        }
      });
    });

    // Template management endpoints
    this.app.get('/api/templates/info', (req, res) => {
      res.json({
        templatesPath: this.templateExporter.getTemplatesPath(),
        appDataPath: this.templateExporter.getAppDataPath()
      });
    });

    this.app.post('/api/templates/export', async (req, res) => {
      try {
        await this.templateExporter.exportTemplates();
        res.json({ 
          success: true, 
          message: 'Templates exported successfully',
          path: this.templateExporter.getTemplatesPath()
        });
      } catch (error) {
        logger.error(`Manual template export failed: ${error instanceof Error ? error.message : String(error)}`);
        res.status(500).json({ 
          success: false, 
          error: error instanceof Error ? error.message : 'Template export failed'
        });
      }
    });

    // Serve receiver frontend static files
    this.app.use(express.static(this.receiverDistPath, {
      index: false, // Don't serve index.html automatically
      setHeaders: (res, path) => {
        // Set appropriate headers for different file types
        if (path.endsWith('.js')) {
          res.setHeader('Content-Type', 'application/javascript');
        } else if (path.endsWith('.css')) {
          res.setHeader('Content-Type', 'text/css');
        } else if (path.endsWith('.wasm')) {
          res.setHeader('Content-Type', 'application/wasm');
        }
      }
    }));

    // Catch-all handler for frontend routes (SPA support)
    this.app.get('*', (req, res, next) => {
      // Skip API routes
      if (req.path.startsWith('/api/') || 
          req.path.startsWith('/lora/') || 
          req.path.startsWith('/wasm/') || 
          req.path.startsWith('/protobuf/') ||
          req.path === '/health') {
        return next();
      }

      // Serve index.html for all other routes (SPA)
      const indexPath = path.join(this.receiverDistPath, 'index.html');
      res.sendFile(indexPath, (err) => {
        if (err) {
          logger.error(`Failed to serve index.html: ${err instanceof Error ? err.message : String(err)}`);
          res.status(404).json({
            success: false,
            error: 'Frontend not built. Run "npm run build:frontend" first.'
          });
        }
      });
    });
  }

  private async initializeBackend() {
    logger.info('Initializing backend components...');

    try {
      // Initialize backend components without starting a separate server
      await this.backend.initializeComponents();
      logger.info('Backend components initialized');

      // Setup backend routes directly in our app
      this.setupBackendRoutes();

    } catch (error) {
      logger.error(`Failed to initialize backend: ${error instanceof Error ? error.message : String(error)}`);
      throw error;
    }
  }

  private setupBackendRoutes() {
    // Mount backend routes directly

    // LoRA adapter endpoints
    this.app.post('/lora/compile', async (req, res) => {
      try {
        const { skillData, metadata } = req.body;
        const adapter = await this.backend.loraEngine.compileAdapter(skillData, metadata);
        res.json({ success: true, adapter });
      } catch (error) {
        logger.error(`LoRA compilation failed: ${error instanceof Error ? error.message : String(error)}`);
        res.status(500).json({ success: false, error: error instanceof Error ? error.message : 'Unknown error' });
      }
    });

    this.app.post('/lora/invoke', async (req, res) => {
      try {
        const { adapterId, parameters } = req.body;
        const result = await this.backend.loraEngine.invokeAdapter(adapterId, parameters);
        res.json({ success: true, result });
      } catch (error) {
        logger.error(`LoRA invocation failed: ${error instanceof Error ? error.message : String(error)}`);
        res.status(500).json({ success: false, error: error instanceof Error ? error.message : 'Unknown error' });
      }
    });

    // WASM compilation endpoints
    this.app.post('/wasm/compile', async (req, res) => {
      try {
        const { rustCode, options: _options } = req.body;
        const wasmModule = await this.backend.wasmCompiler.compile(rustCode, _options);
        res.json({ success: true, wasmModule });
      } catch (error) {
        logger.error(`WASM compilation failed: ${error instanceof Error ? error.message : String(error)}`);
        res.status(500).json({ success: false, error: error instanceof Error ? error.message : 'Unknown error' });
      }
    });

    // Protobuf endpoints
    this.app.post('/protobuf/serialize', async (req, res) => {
      try {
        const { data, schema } = req.body;
        const serialized = await this.backend.protobufHandler.serialize(data, schema);
        res.json({ success: true, serialized });
      } catch (error) {
        logger.error(`Protobuf serialization failed: ${error instanceof Error ? error.message : String(error)}`);
        res.status(500).json({ success: false, error: error instanceof Error ? error.message : 'Unknown error' });
      }
    });

    this.app.post('/protobuf/deserialize', async (req, res) => {
      try {
        const { data, schema } = req.body;
        const deserialized = await this.backend.protobufHandler.deserialize(data, schema);
        res.json({ success: true, deserialized });
      } catch (error) {
        logger.error(`Protobuf deserialization failed: ${error instanceof Error ? error.message : String(error)}`);
        res.status(500).json({ success: false, error: error instanceof Error ? error.message : 'Unknown error' });
      }
    });
  }

  public async start() {
    try {
      logger.info('Starting KNIRV-CONTROLLER Unified Server...');
      
      // Export templates first
      await this.exportTemplatesOnStartup();
      
      // Initialize backend
      await this.initializeBackend();
      
      // Setup routes
      this.setupRoutes();
      
      // Create server
      this.server = createServer(this.app);
      
      // Start listening
      this.server.listen(this.port, () => {
        logger.info(`🚀 KNIRV-CONTROLLER Unified Server started on port ${this.port}`);
        logger.info('📱 Unified frontend available at: http://localhost:' + this.port);
        logger.info('🔧 Backend API available at: http://localhost:' + this.port + '/api');
        logger.info('📋 Templates exported to: ' + this.templateExporter.getTemplatesPath());
        logger.info('');
        logger.info('Available endpoints:');
        logger.info('  GET  / - Unified Frontend (Manager + Receiver)');
        logger.info('  GET  /health - Health check');
        logger.info('  GET  /api/templates/info - Template information');
        logger.info('  POST /api/templates/export - Manual template export');
        logger.info('  POST /api/* - Backend API (proxied)');
        logger.info('  POST /lora/* - LoRA endpoints (proxied)');
        logger.info('  POST /wasm/* - WASM endpoints (proxied)');
        logger.info('  POST /protobuf/* - Protobuf endpoints (proxied)');
      });

      // Graceful shutdown
      process.on('SIGTERM', () => this.shutdown());
      process.on('SIGINT', () => this.shutdown());

    } catch (error) {
      logger.error(`Failed to start unified server: ${error instanceof Error ? error.message : String(error)}`);
      throw error;
    }
  }

  private async shutdown() {
    logger.info('Shutting down KNIRV-CONTROLLER Unified Server...');

    if (this.server) {
      this.server.close();
    }

    // Shutdown backend
    if (this.backend) {
      // The backend should handle its own shutdown
      logger.info('Backend shutdown handled by backend instance');
    }

    logger.info('Shutdown complete');
    process.exit(0);
  }
}

// Start the unified server if this file is run directly
if (getModuleUrl() === `file://${process.argv[1]}`) {
  const server = new KNIRVControllerUnifiedServer();
  server.start().catch((error) => {
    logger.error(`Failed to start KNIRV-CONTROLLER Unified Server: ${error instanceof Error ? error.message : String(error)}`);
    process.exit(1);
  });
}
