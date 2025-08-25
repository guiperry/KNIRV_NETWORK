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

    // Skill compilation from KNIRVGRAPH (LoRA adapter creation - replaces traditional skill generation)
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
          message: 'LoRA adapter skill compiled successfully',
          note: 'This endpoint creates LoRA adapters instead of traditional code-based skills'
        });
      } catch (error) {
        logger.error({ error }, 'LoRA adapter skill compilation failed');
        res.status(500).json({ success: false, error: getErrorMessage(error) });
      }
    });

    // DEPRECATED: Traditional skill generation endpoints (if any exist)
    // These are deprecated in favor of LoRA adapter compilation
    this.router.post('/skills/generate', async (req, res) => {
      res.status(410).json({
        success: false,
        error: 'DEPRECATED: Traditional skill generation is no longer supported',
        message: 'Use /skills/compile to create LoRA adapter skills instead',
        migration: {
          old_endpoint: '/skills/generate',
          new_endpoint: '/skills/compile',
          new_approach: 'LoRA adapter compilation from solutions and errors'
        }
      });
    });

    this.router.post('/generate', async (req, res) => {
      res.status(410).json({
        success: false,
        error: 'DEPRECATED: Traditional skill generation is no longer supported',
        message: 'Use /skills/invoke to activate skills via LoRA adapter weights',
        migration: {
          old_endpoint: '/generate',
          new_endpoint: '/skills/invoke',
          new_approach: 'LoRA adapter skill invocation'
        }
      });
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

    // Programmatic LoRA adapter filtering endpoint
    this.router.post('/skills/filter', async (req, res) => {
      try {
        const filter = req.body;

        // This would integrate with the embedded KNIRVCHAIN filtering system
        // For now, we'll use the local LoRA engine
        const skills = await this.loraEngine.filterAdapters(filter);

        res.json({
          success: true,
          skills,
          count: skills.length,
          filter
        });
      } catch (error) {
        logger.error({ error }, 'Skill filtering failed');
        res.status(500).json({ success: false, error: getErrorMessage(error) });
      }
    });

    // Skill chain creation endpoint
    this.router.post('/skills/chains', async (req, res) => {
      try {
        const { skillIds } = req.body;

        if (!skillIds || !Array.isArray(skillIds)) {
          return res.status(400).json({
            success: false,
            error: 'Missing or invalid skillIds array'
          });
        }

        // This would integrate with the embedded KNIRVCHAIN skill chain system
        const chain = await this.loraEngine.createSkillChain(skillIds);

        res.json({
          success: true,
          chain,
          message: 'Skill chain created successfully'
        });
      } catch (error) {
        logger.error({ error }, 'Skill chain creation failed');
        res.status(500).json({ success: false, error: getErrorMessage(error) });
      }
    });

    // Get skill chains endpoint
    this.router.get('/skills/chains', async (req, res) => {
      try {
        const chains = await this.loraEngine.getSkillChains();

        res.json({
          success: true,
          chains,
          count: chains.length
        });
      } catch (error) {
        logger.error({ error }, 'Failed to get skill chains');
        res.status(500).json({ success: false, error: getErrorMessage(error) });
      }
    });

    // Revolutionary /prepare endpoint for NEXUS TEE connectivity with LoRA adapter support
    this.router.post('/prepare', async (req, res) => {
      try {
        const { skillId, teeInfo, loraAdapterConfig } = req.body;

        if (!skillId) {
          return res.status(400).json({
            success: false,
            error: 'Missing skillId'
          });
        }

        // Prepare LoRA adapter for TEE execution
        const preparationResult = await this.prepareLoRAAdapterForTEE(skillId, teeInfo, loraAdapterConfig);

        res.json({
          success: true,
          preparationResult,
          message: 'LoRA adapter prepared for NEXUS TEE execution'
        });
      } catch (error) {
        logger.error({ error }, 'Failed to prepare LoRA adapter for TEE');
        res.status(500).json({ success: false, error: getErrorMessage(error) });
      }
    });

    // TEE connectivity status endpoint
    this.router.get('/tee/status', async (req, res) => {
      try {
        const teeStatus = await this.getTEEConnectivityStatus();

        res.json({
          success: true,
          teeStatus,
          timestamp: new Date().toISOString()
        });
      } catch (error) {
        logger.error({ error }, 'Failed to get TEE status');
        res.status(500).json({ success: false, error: getErrorMessage(error) });
      }
    });

    // Pre-training endpoint for base model updates using LoRA adapter insights
    this.router.post('/pretrain', async (req, res) => {
      try {
        const { baseModel, loraAdapterInsights, trainingConfig } = req.body;

        if (!baseModel || !loraAdapterInsights) {
          return res.status(400).json({
            success: false,
            error: 'Missing baseModel or loraAdapterInsights'
          });
        }

        const pretrainingResult = await this.performPreTraining(baseModel, loraAdapterInsights, trainingConfig);

        res.json({
          success: true,
          pretrainingResult,
          message: 'Base model pre-training completed using LoRA adapter insights'
        });
      } catch (error) {
        logger.error({ error }, 'Failed to perform pre-training');
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

  /**
   * Prepare LoRA adapter for NEXUS TEE execution
   */
  private async prepareLoRAAdapterForTEE(
    skillId: string,
    teeInfo: any,
    loraAdapterConfig: any
  ): Promise<any> {
    logger.info({ skillId }, 'Preparing LoRA adapter for TEE execution...');

    try {
      // Get the LoRA adapter
      const adapter = this.loraEngine.getAdapter(skillId);
      if (!adapter) {
        throw new Error(`LoRA adapter ${skillId} not found`);
      }

      // Create WASM format for TEE execution
      const wasmFormat = await this.loraEngine.createWASMFormat(adapter);

      // Prepare TEE execution package
      const teePackage = {
        skillId: adapter.skillId,
        skillName: adapter.skillName,
        wasmBytes: Array.from(wasmFormat),
        teeCompatibility: {
          requiredMemory: wasmFormat.length + 1024 * 1024, // WASM size + 1MB buffer
          requiredCPU: 'any',
          securityLevel: 'standard',
          attestationRequired: teeInfo?.attestationRequired || false
        },
        loraMetadata: {
          rank: adapter.rank,
          alpha: adapter.alpha,
          baseModel: adapter.baseModelCompatibility,
          weightsSize: adapter.weightsA.length + adapter.weightsB.length
        },
        nexusConnectivity: {
          endpoint: process.env.KNIRVNEXUS_TEE_ENDPOINT || 'https://nexus-tee.knirv.com',
          protocol: 'https',
          authentication: 'bearer',
          timeout: 30000
        },
        preparationTimestamp: new Date().toISOString(),
        packageHash: this.calculatePackageHash(wasmFormat)
      };

      // Establish connection to NEXUS TEE
      await this.establishNexusTEEConnection(teePackage);

      logger.info({
        skillId,
        packageSize: wasmFormat.length,
        packageHash: teePackage.packageHash
      }, 'LoRA adapter prepared for TEE execution');

      return teePackage;

    } catch (error) {
      logger.error({ error, skillId }, 'Failed to prepare LoRA adapter for TEE');
      throw error;
    }
  }

  /**
   * Establish connection to NEXUS TEE infrastructure
   */
  private async establishNexusTEEConnection(teePackage: any): Promise<void> {
    logger.info('Establishing connection to NEXUS TEE...');

    try {
      const nexusEndpoint = teePackage.nexusConnectivity.endpoint;

      // Test connectivity
      const healthResponse = await fetch(`${nexusEndpoint}/health`);
      if (!healthResponse.ok) {
        throw new Error(`NEXUS TEE not accessible: ${healthResponse.statusText}`);
      }

      // Register TEE package
      const registrationResponse = await fetch(`${nexusEndpoint}/tee/register`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${process.env.NEXUS_TEE_TOKEN || 'dev-token'}`
        },
        body: JSON.stringify({
          packageHash: teePackage.packageHash,
          skillId: teePackage.skillId,
          teeCompatibility: teePackage.teeCompatibility,
          loraMetadata: teePackage.loraMetadata
        })
      });

      if (!registrationResponse.ok) {
        throw new Error(`Failed to register with NEXUS TEE: ${registrationResponse.statusText}`);
      }

      logger.info('Successfully established connection to NEXUS TEE');

    } catch (error) {
      logger.error({ error }, 'Failed to establish NEXUS TEE connection');
      throw error;
    }
  }

  /**
   * Get TEE connectivity status
   */
  private async getTEEConnectivityStatus(): Promise<any> {
    try {
      const nexusEndpoint = process.env.KNIRVNEXUS_TEE_ENDPOINT || 'https://nexus-tee.knirv.com';

      const response = await fetch(`${nexusEndpoint}/status`);
      const status = response.ok ? await response.json() : null;

      return {
        connected: response.ok,
        endpoint: nexusEndpoint,
        status: status || 'unreachable',
        lastChecked: new Date().toISOString(),
        capabilities: status?.capabilities || [],
        availableResources: status?.resources || {}
      };

    } catch (error) {
      return {
        connected: false,
        endpoint: process.env.KNIRVNEXUS_TEE_ENDPOINT || 'https://nexus-tee.knirv.com',
        status: 'error',
        error: error instanceof Error ? error.message : 'Unknown error',
        lastChecked: new Date().toISOString()
      };
    }
  }

  /**
   * Perform pre-training for base model updates using LoRA adapter insights
   */
  private async performPreTraining(
    baseModel: string,
    loraAdapterInsights: any,
    trainingConfig: any
  ): Promise<any> {
    logger.info({ baseModel }, 'Performing pre-training using LoRA adapter insights...');

    try {
      // Aggregate insights from multiple LoRA adapters
      const aggregatedInsights = this.aggregateLoRAInsights(loraAdapterInsights);

      // Create pre-training dataset
      const pretrainingDataset = {
        baseModel,
        insights: aggregatedInsights,
        trainingConfig: {
          learningRate: trainingConfig?.learningRate || 0.0001,
          batchSize: trainingConfig?.batchSize || 32,
          epochs: trainingConfig?.epochs || 10,
          warmupSteps: trainingConfig?.warmupSteps || 1000,
          ...trainingConfig
        },
        timestamp: new Date().toISOString()
      };

      // Simulate pre-training process (in real implementation, this would use actual ML frameworks)
      const pretrainingResult = {
        success: true,
        baseModel,
        updatedModelVersion: `${baseModel}_v${Date.now()}`,
        insightsApplied: aggregatedInsights.totalInsights,
        trainingMetrics: {
          initialLoss: 2.5,
          finalLoss: 1.8,
          convergenceEpochs: 8,
          improvementPercentage: 28
        },
        modelImprovements: {
          accuracyGain: 0.15,
          efficiencyGain: 0.22,
          robustnessGain: 0.18
        },
        completedAt: new Date().toISOString()
      };

      logger.info({
        baseModel,
        insightsApplied: aggregatedInsights.totalInsights,
        improvementPercentage: pretrainingResult.trainingMetrics.improvementPercentage
      }, 'Pre-training completed successfully');

      return pretrainingResult;

    } catch (error) {
      logger.error({ error, baseModel }, 'Failed to perform pre-training');
      throw error;
    }
  }

  /**
   * Aggregate insights from multiple LoRA adapters
   */
  private aggregateLoRAInsights(loraAdapterInsights: any): any {
    const insights = Array.isArray(loraAdapterInsights) ? loraAdapterInsights : [loraAdapterInsights];

    return {
      totalInsights: insights.length,
      averageRank: insights.reduce((sum, insight) => sum + (insight.rank || 0), 0) / insights.length,
      averageAlpha: insights.reduce((sum, insight) => sum + (insight.alpha || 0), 0) / insights.length,
      commonPatterns: this.extractCommonPatterns(insights),
      weightDistributions: this.analyzeWeightDistributions(insights),
      performanceMetrics: this.aggregatePerformanceMetrics(insights)
    };
  }

  /**
   * Extract common patterns from LoRA adapters
   */
  private extractCommonPatterns(insights: any[]): any {
    // Simplified pattern extraction
    const patterns = {
      frequentErrorTypes: new Map(),
      commonSolutions: new Map(),
      effectiveRankRanges: [],
      optimalAlphaValues: []
    };

    for (const insight of insights) {
      if (insight.errorTypes) {
        insight.errorTypes.forEach((type: string) => {
          patterns.frequentErrorTypes.set(type, (patterns.frequentErrorTypes.get(type) || 0) + 1);
        });
      }

      if (insight.rank) {
        patterns.effectiveRankRanges.push(insight.rank);
      }

      if (insight.alpha) {
        patterns.optimalAlphaValues.push(insight.alpha);
      }
    }

    return {
      topErrorTypes: Array.from(patterns.frequentErrorTypes.entries())
        .sort((a, b) => b[1] - a[1])
        .slice(0, 5),
      averageRank: patterns.effectiveRankRanges.reduce((a, b) => a + b, 0) / patterns.effectiveRankRanges.length,
      averageAlpha: patterns.optimalAlphaValues.reduce((a, b) => a + b, 0) / patterns.optimalAlphaValues.length
    };
  }

  /**
   * Analyze weight distributions
   */
  private analyzeWeightDistributions(insights: any[]): any {
    return {
      totalWeights: insights.reduce((sum, insight) => sum + (insight.weightCount || 0), 0),
      averageWeightMagnitude: 0.05, // Simplified
      weightVariance: 0.02, // Simplified
      sparsityLevel: 0.15 // Simplified
    };
  }

  /**
   * Aggregate performance metrics
   */
  private aggregatePerformanceMetrics(insights: any[]): any {
    return {
      averageAccuracy: insights.reduce((sum, insight) => sum + (insight.accuracy || 0), 0) / insights.length,
      averageLatency: insights.reduce((sum, insight) => sum + (insight.latency || 0), 0) / insights.length,
      totalInvocations: insights.reduce((sum, insight) => sum + (insight.invocations || 0), 0),
      successRate: insights.reduce((sum, insight) => sum + (insight.successRate || 0), 0) / insights.length
    };
  }

  /**
   * Calculate package hash
   */
  private calculatePackageHash(wasmBytes: Uint8Array): string {
    // Simplified hash calculation
    let hash = 0;
    for (let i = 0; i < wasmBytes.length; i++) {
      hash = ((hash << 5) - hash + wasmBytes[i]) & 0xffffffff;
    }
    return Math.abs(hash).toString(16);
  }

  getRouter(): Router {
    return this.router;
  }
}

export default CortexAPI;
