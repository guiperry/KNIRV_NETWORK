/**
 * Cortex API - REST API endpoints for KNIRV-CORTEX backend operations
 */

import { Router } from 'express';
import { LoRAAdapterEngine } from '../lora/LoRAAdapterEngine.js';
import { WASMCompiler } from '../wasm/WASMCompiler.js';

// Utility function to safely get error message
function getErrorMessage(error: unknown): string {
  return error instanceof Error ? error.message : String(error);
}
import { ProtobufHandler } from '../protobuf/ProtobufHandler.js';
import pino from 'pino';

const logger = pino({ name: 'cortex-api' });

export class CortexAPI {
  private router: Router;

  constructor(
    private loraEngine: LoRAAdapterEngine,
    private wasmCompiler: WASMCompiler,
    private protobufHandler: ProtobufHandler
  ) {
    this.router = Router();
    this.setupRoutes();
  }

  private setupRoutes() {
    // System status
    this.router.get('/status', (req, res) => {
      res.json({
        status: 'operational',
        components: {
          loraEngine: this.loraEngine.isReady(),
          wasmCompiler: this.wasmCompiler.isReady(),
          protobufHandler: this.protobufHandler.isReady()
        },
        capabilities: [
          'lora-adapter-compilation',
          'skill-invocation',
          'wasm-compilation',
          'protobuf-serialization'
        ],
        version: '1.0.0'
      });
    });

    // LoRA adapter management
    this.router.get('/adapters', (req, res) => {
      try {
        const adapters = this.loraEngine.getAvailableAdapters();
        res.json({
          success: true,
          adapters: adapters.map(adapter => ({
            skillId: adapter.skillId,
            skillName: adapter.skillName,
            description: adapter.description,
            version: adapter.version,
            rank: adapter.rank,
            alpha: adapter.alpha,
            baseModelCompatibility: adapter.baseModelCompatibility
          }))
        });
      } catch (error) {
        logger.error({ error }, 'Failed to get adapters');
        res.status(500).json({ success: false, error: getErrorMessage(error) });
      }
    });

    this.router.get('/adapters/:skillId', (req, res) => {
      try {
        const adapter = this.loraEngine.getAdapter(req.params.skillId);
        if (!adapter) {
          return res.status(404).json({
            success: false,
            error: 'Adapter not found'
          });
        }

        res.json({
          success: true,
          adapter: {
            skillId: adapter.skillId,
            skillName: adapter.skillName,
            description: adapter.description,
            version: adapter.version,
            rank: adapter.rank,
            alpha: adapter.alpha,
            baseModelCompatibility: adapter.baseModelCompatibility,
            metadata: adapter.additionalMetadata
          }
        });
      } catch (error) {
        logger.error({ error }, 'Failed to get adapter');
        res.status(500).json({ success: false, error: getErrorMessage(error) });
      }
    });

    this.router.delete('/adapters/:skillId', (req, res) => {
      try {
        const removed = this.loraEngine.removeAdapter(req.params.skillId);
        if (!removed) {
          return res.status(404).json({
            success: false,
            error: 'Adapter not found'
          });
        }

        res.json({
          success: true,
          message: 'Adapter removed successfully'
        });
      } catch (error) {
        logger.error({ error }, 'Failed to remove adapter');
        res.status(500).json({ success: false, error: getErrorMessage(error) });
      }
    });

    // Skill compilation from KNIRVGRAPH
    this.router.post('/skills/compile', async (req, res) => {
      try {
        const { skillData, metadata } = req.body;
        
        if (!skillData || !metadata) {
          return res.status(400).json({
            success: false,
            error: 'Missing skillData or metadata'
          });
        }

        const adapter = await this.loraEngine.compileAdapter(skillData, metadata);
        
        res.json({
          success: true,
          adapter: {
            skillId: adapter.skillId,
            skillName: adapter.skillName,
            description: adapter.description,
            version: adapter.version,
            rank: adapter.rank,
            alpha: adapter.alpha
          },
          message: 'Skill compiled successfully'
        });
      } catch (error) {
        logger.error({ error }, 'Skill compilation failed');
        res.status(500).json({ success: false, error: getErrorMessage(error) });
      }
    });

    // Skill invocation (the revolutionary /invoke endpoint)
    this.router.post('/skills/invoke', async (req, res) => {
      try {
        const { skillId, parameters = {} } = req.body;
        
        if (!skillId) {
          return res.status(400).json({
            success: false,
            error: 'Missing skillId'
          });
        }

        const response = await this.loraEngine.invokeAdapter(skillId, parameters);
        
        if (response.status === 'SUCCESS') {
          // Serialize the response using protobuf
          const serializedResponse = await this.protobufHandler.createSkillInvocationResponse(
            response.invocationId,
            response.status,
            response.skill
          );

          res.set('Content-Type', 'application/octet-stream');
          res.send(Buffer.from(serializedResponse));
        } else {
          res.status(response.status === 'NOT_FOUND' ? 404 : 500).json({
            success: false,
            invocationId: response.invocationId,
            status: response.status,
            error: response.errorMessage
          });
        }
      } catch (error) {
        logger.error({ error }, 'Skill invocation failed');
        res.status(500).json({ success: false, error: getErrorMessage(error) });
      }
    });

    // WASM compilation endpoints
    this.router.post('/wasm/compile-agent-core', async (req, res) => {
      try {
        const { options = {} } = req.body;
        const wasmModule = await this.wasmCompiler.compileAgentCore(options);
        
        res.json({
          success: true,
          module: {
            size: wasmModule.metadata.size,
            compilationTime: wasmModule.metadata.compilationTime,
            features: wasmModule.metadata.features,
            target: wasmModule.metadata.target
          },
          wasmBytes: Array.from(wasmModule.wasmBytes),
          jsBindings: wasmModule.jsBindings,
          typeDefinitions: wasmModule.typeDefinitions
        });
      } catch (error) {
        logger.error({ error }, 'Agent-core WASM compilation failed');
        res.status(500).json({ success: false, error: getErrorMessage(error) });
      }
    });

    this.router.post('/wasm/build-existing', async (req, res) => {
      try {
        const wasmModule = await this.wasmCompiler.buildExistingProject();
        
        res.json({
          success: true,
          module: {
            size: wasmModule.metadata.size,
            compilationTime: wasmModule.metadata.compilationTime,
            features: wasmModule.metadata.features,
            target: wasmModule.metadata.target
          },
          message: 'Existing WASM project built successfully'
        });
      } catch (error) {
        logger.error({ error }, 'Existing WASM build failed');
        res.status(500).json({ success: false, error: getErrorMessage(error) });
      }
    });

    // Protobuf utilities
    this.router.get('/protobuf/schemas', (req, res) => {
      try {
        const schemas = this.protobufHandler.getAvailableSchemas();
        res.json({
          success: true,
          schemas
        });
      } catch (error) {
        logger.error({ error }, 'Failed to get protobuf schemas');
        res.status(500).json({ success: false, error: getErrorMessage(error) });
      }
    });

    // Development and testing endpoints
    this.router.post('/dev/test-lora-pipeline', async (req, res) => {
      try {
        // Create test skill data
        const testSkillData = {
          solutions: [
            {
              errorId: 'test-error-1',
              solution: 'function testSolution() { return "Hello World"; }',
              confidence: 0.9
            }
          ],
          errors: [
            {
              errorId: 'test-error-1',
              description: 'Need a function that returns Hello World',
              context: 'Testing LoRA adapter compilation'
            }
          ]
        };

        const testMetadata = {
          skillName: 'Test Hello World Skill',
          description: 'A test skill for LoRA adapter compilation',
          baseModel: 'CodeT5-base',
          rank: 4,
          alpha: 8.0
        };

        const adapter = await this.loraEngine.compileAdapter(testSkillData, testMetadata);
        
        res.json({
          success: true,
          message: 'LoRA pipeline test completed successfully',
          testAdapter: {
            skillId: adapter.skillId,
            skillName: adapter.skillName,
            weightsASize: adapter.weightsA.length,
            weightsBSize: adapter.weightsB.length
          }
        });
      } catch (error) {
        logger.error({ error }, 'LoRA pipeline test failed');
        res.status(500).json({ success: false, error: getErrorMessage(error) });
      }
    });
  }

  getRouter(): Router {
    return this.router;
  }
}

export default CortexAPI;
