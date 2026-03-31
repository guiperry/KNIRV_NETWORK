/**
 * KNIRV-CORTEX Backend Entry Point
 * Backend-only WASM compilation pipeline for LoRA adapter processing
 */

import express, { Request, Response, NextFunction } from 'express';
import { createServer, Server } from 'http';
import { WebSocketServer, WebSocket } from 'ws';
import { IncomingMessage } from 'http';
import cors from 'cors';
import helmet from 'helmet';
import compression from 'compression';
import pino from 'pino';
import { LoRAAdapterEngine } from './lora/LoRAAdapterEngine.js';
import { WASMCompiler } from './wasm/WASMCompiler.js';
import { ProtobufHandler } from './protobuf/ProtobufHandler.js';
import { CortexAPI } from './api/CortexAPI.js';

// Utility function to safely get error message
function getErrorMessage(error: unknown): string {
  return error instanceof Error ? error.message : String(error);
}

const logger = pino({
  name: 'knirv-cortex-backend',
  level: process.env.LOG_LEVEL || 'info',
  transport: {
    target: 'pino-pretty',
    options: {
      colorize: true
    }
  }
});

class KNIRVCortexBackend {
  private app: express.Application;
  private server: Server | undefined;
  private wss: WebSocketServer | null = null;
  public loraEngine!: LoRAAdapterEngine;
  public wasmCompiler!: WASMCompiler;
  public protobufHandler!: ProtobufHandler;
  private cortexAPI!: CortexAPI;
  private port: number;

  constructor() {
    this.port = parseInt(process.env.PORT || '3004');
    this.app = express();
    this.setupMiddleware();
    // Note: Components will be initialized in the start() method
  }

  private setupMiddleware() {
    this.app.use(helmet());
    this.app.use(compression());
    this.app.use(cors({
      origin: process.env.ALLOWED_ORIGINS?.split(',') || ['http://localhost:3000', 'http://localhost:3001', 'http://localhost:3002'],
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

  public async initializeComponents() {
    logger.info('Initializing KNIRV-CORTEX backend components...');

    try {
      // Initialize WASM compiler
      this.wasmCompiler = new WASMCompiler();
      await this.wasmCompiler.initialize();
      logger.info('WASM compiler initialized');

      // Initialize protobuf handler
      this.protobufHandler = new ProtobufHandler();
      await this.protobufHandler.initialize();
      logger.info('Protobuf handler initialized');

      // Initialize LoRA adapter engine
      this.loraEngine = new LoRAAdapterEngine(this.wasmCompiler, this.protobufHandler);
      await this.loraEngine.initialize();
      logger.info('LoRA adapter engine initialized');

      // Initialize API handler
      this.cortexAPI = new CortexAPI(this.loraEngine, this.wasmCompiler, this.protobufHandler);
      logger.info('Cortex API initialized');

    } catch (error) {
      logger.error({ error }, 'Failed to initialize components');
      throw error;
    }
  }

  private setupRoutes() {
    // Health check
    this.app.get('/health', (req, res) => {
      res.json({
        status: 'healthy',
        timestamp: new Date().toISOString(),
        components: {
          wasmCompiler: this.wasmCompiler.isReady(),
          loraEngine: this.loraEngine.isReady(),
          protobufHandler: this.protobufHandler.isReady()
        }
      });
    });

    // API routes
    this.app.use('/api', this.cortexAPI.getRouter());

    // LoRA adapter endpoints
    this.app.post('/lora/compile', async (req, res) => {
      try {
        const { skillData, metadata } = req.body;
        const adapter = await this.loraEngine.compileAdapter(skillData, metadata);
        res.json({ success: true, adapter });
      } catch (error) {
        logger.error({ error }, 'LoRA compilation failed');
        res.status(500).json({ success: false, error: getErrorMessage(error) });
      }
    });

    this.app.post('/lora/invoke', async (req, res) => {
      try {
        const { adapterId, parameters } = req.body;
        const result = await this.loraEngine.invokeAdapter(adapterId, parameters);
        res.json({ success: true, result });
      } catch (error) {
        logger.error({ error }, 'LoRA invocation failed');
        res.status(500).json({ success: false, error: getErrorMessage(error) });
      }
    });

    // WASM compilation endpoints
    this.app.post('/wasm/compile', async (req, res) => {
      try {
        const { rustCode, options } = req.body;
        const wasmModule = await this.wasmCompiler.compile(rustCode, options);
        res.json({ success: true, wasmModule });
      } catch (error) {
        logger.error({ error }, 'WASM compilation failed');
        res.status(500).json({ success: false, error: getErrorMessage(error) });
      }
    });

    // Protobuf endpoints
    this.app.post('/protobuf/serialize', async (req, res) => {
      try {
        const { data, schema } = req.body;
        const serialized = await this.protobufHandler.serialize(data, schema);
        res.json({ success: true, serialized });
      } catch (error) {
        logger.error({ error }, 'Protobuf serialization failed');
        res.status(500).json({ success: false, error: getErrorMessage(error) });
      }
    });

    this.app.post('/protobuf/deserialize', async (req, res) => {
      try {
        const { data, schema } = req.body;
        const deserialized = await this.protobufHandler.deserialize(data, schema);
        res.json({ success: true, deserialized });
      } catch (error) {
        logger.error({ error }, 'Protobuf deserialization failed');
        res.status(500).json({ success: false, error: getErrorMessage(error) });
      }
    });

    // Error handling
    this.app.use((error: unknown, req: Request, res: Response, _next: NextFunction) => {
      logger.error({ error, url: req.url, method: req.method }, 'Unhandled error');
      res.status(500).json({
        success: false,
        error: 'Internal server error',
        message: process.env.NODE_ENV === 'development' ? getErrorMessage(error) : undefined
      });
    });

    // 404 handler
    this.app.use('*', (req, res) => {
      res.status(404).json({
        success: false,
        error: 'Endpoint not found',
        path: req.originalUrl
      });
    });
  }

  private setupWebSocket() {
    this.wss = new WebSocketServer({ server: this.server });

    this.wss.on('connection', (ws: WebSocket, req: IncomingMessage) => {
      logger.info({ ip: req.socket.remoteAddress }, 'WebSocket connection established');

      ws.on('message', async (message) => {
        try {
          const data = JSON.parse(message.toString());
          await this.handleWebSocketMessage(ws, data);
        } catch (error) {
          logger.error({ error }, 'WebSocket message handling failed');
          ws.send(JSON.stringify({
            type: 'error',
            message: 'Invalid message format'
          }));
        }
      });

      ws.on('close', () => {
        logger.info('WebSocket connection closed');
      });

      // Send welcome message
      ws.send(JSON.stringify({
        type: 'welcome',
        message: 'Connected to KNIRV-CORTEX backend',
        capabilities: ['lora-compilation', 'wasm-compilation', 'protobuf-processing']
      }));
    });
  }

  private async handleWebSocketMessage(ws: WebSocket, data: { type: string; skillData?: unknown; metadata?: unknown; [key: string]: unknown }) {
    switch (data.type) {
      case 'lora_compile':
        try {
          const adapter = await this.loraEngine.compileAdapter(
            data.skillData as { solutions: { errorId: string; solution: string; confidence: number; }[]; errors: { errorId: string; description: string; context: string; }[]; },
            data.metadata as { skillName: string; description: string; baseModel: string; rank?: number; alpha?: number; }
          );
          ws.send(JSON.stringify({
            type: 'lora_compile_result',
            requestId: data.requestId,
            success: true,
            adapter
          }));
        } catch (error) {
          ws.send(JSON.stringify({
            type: 'lora_compile_result',
            requestId: data.requestId,
            success: false,
            error: getErrorMessage(error)
          }));
        }
        break;

      case 'wasm_compile':
        try {
          const wasmModule = await this.wasmCompiler.compile(data.rustCode as string, data.options as Record<string, unknown>);
          ws.send(JSON.stringify({
            type: 'wasm_compile_result',
            requestId: data.requestId,
            success: true,
            wasmModule
          }));
        } catch (error) {
          ws.send(JSON.stringify({
            type: 'wasm_compile_result',
            requestId: data.requestId,
            success: false,
            error: getErrorMessage(error)
          }));
        }
        break;

      default:
        ws.send(JSON.stringify({
          type: 'error',
          message: `Unknown message type: ${data.type}`
        }));
    }
  }

  public async start() {
    try {
      // Initialize components first
      await this.initializeComponents();

      // Setup routes after components are initialized
      this.setupRoutes();

      this.server = createServer(this.app);
      this.setupWebSocket();

      this.server.listen(this.port, () => {
        logger.info({ port: this.port }, 'KNIRV-CORTEX backend server started');
        logger.info('Available endpoints:');
        logger.info('  GET  /health - Health check');
        logger.info('  POST /lora/compile - Compile LoRA adapter');
        logger.info('  POST /lora/invoke - Invoke LoRA adapter');
        logger.info('  POST /wasm/compile - Compile WASM module');
        logger.info('  POST /protobuf/serialize - Serialize protobuf');
        logger.info('  POST /protobuf/deserialize - Deserialize protobuf');
        logger.info(`  WS   ws://localhost:${this.port} - WebSocket API`);
      });

      // Graceful shutdown
      process.on('SIGTERM', () => this.shutdown());
      process.on('SIGINT', () => this.shutdown());

    } catch (error) {
      logger.error({ error }, 'Failed to start server');
      throw error;
    }
  }

  private async shutdown() {
    logger.info('Shutting down KNIRV-CORTEX backend...');

    if (this.wss) {
      this.wss.close();
    }

    if (this.server) {
      this.server.close();
    }

    // Cleanup components
    await this.loraEngine?.cleanup();
    await this.wasmCompiler?.cleanup();

    logger.info('Shutdown complete');
    process.exit(0);
  }
}

// Jest-compatible module URL resolution
const getModuleUrl = () => {
  if (typeof process !== 'undefined' && process.env.NODE_ENV === 'test') {
    return 'file://' + process.cwd() + '/src/core/index.ts';
  }
  try {
    const importMeta = eval('import.meta');
    if (importMeta && importMeta.url) {
      return importMeta.url;
    }
  } catch {
    // Fallback for CommonJS
    return 'file://' + process.cwd() + '/src/core/index.ts';
  }
  return 'file://' + process.cwd();
};

// Start the backend if this file is run directly
if (getModuleUrl() === `file://${process.argv[1]}`) {
  const backend = new KNIRVCortexBackend();
  backend.start().catch((error) => {
    logger.error({ error }, 'Failed to start KNIRV-CORTEX backend');
    process.exit(1);
  });
}

export { KNIRVCortexBackend };
