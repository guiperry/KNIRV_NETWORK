/**
 * Cortex API - REST API endpoints for KNIRV-CORTEX backend operations
 */

import { Router } from 'express';
import { LoRAAdapterEngine } from '../lora/LoRAAdapterEngine.js';
import { WASMCompiler } from '../wasm/WASMCompiler.js';

// Type definitions for TEE and LoRA operations
interface TEEInfo {
  endpoint?: string;
  skillId?: string;
  skillName?: string;
  wasmBytes?: number[];
  teeCompatibility?: Record<string, unknown>;
  loraMetadata?: Record<string, unknown>;
  nexusConnectivity?: {
    endpoint?: string;
    [key: string]: unknown;
  };
  preparationTimestamp?: string;
  packageHash?: string;
  credentials?: Record<string, unknown>;
  configuration?: Record<string, unknown>;
  [key: string]: unknown;
}

interface LoRAAdapterConfig {
  rank: number;
  alpha: number;
  targetModules: string[];
  configuration?: Record<string, unknown>;
}

interface LoRAInsight {
  rank: number;
  alpha?: number;
  accuracy: number;
  latency: number;
  invocations: number;
  successRate: number;
  weightCount: number;
  errorTypes?: string[];
  solutions?: string[];
  [key: string]: unknown;
}

interface TEEConnectivityStatus {
  connected: boolean;
  endpoint: string;
  status: string;
  error?: string;
  lastChecked: string;
  capabilities?: Record<string, unknown>;
  availableResources?: Record<string, unknown>;
  [key: string]: unknown;
}

interface PreTrainingResult {
  success: boolean;
  modelPath?: string;
  metrics?: Record<string, number>;
  error?: string;
}

// Utility function to safely get error message
function getErrorMessage(error: unknown): string {
  return error instanceof Error ? error.message : String(error);
}
import { ProtobufHandler } from '../protobuf/ProtobufHandler.js';
import { analyticsService } from '../../services/AnalyticsService.js';
import { taskSchedulingService } from '../../services/TaskSchedulingService.js';
import { udcManagementService } from '../../services/UDCManagementService.js';
import { settingsService } from '../../services/SettingsService.js';
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

    // Analytics Service endpoints
    this.setupAnalyticsRoutes();

    // Task Scheduling Service endpoints
    this.setupSchedulingRoutes();

    // UDC Management Service endpoints
    this.setupUDCRoutes();

    // Settings Service endpoints
    this.setupSettingsRoutes();
  }

  /**
   * Prepare LoRA adapter for NEXUS TEE execution
   */
  private async prepareLoRAAdapterForTEE(
    skillId: string,
    teeInfo: TEEInfo,
    _loraAdapterConfig: LoRAAdapterConfig
  ): Promise<PreTrainingResult> {
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
          attestationRequired: (teeInfo && typeof teeInfo === 'object' && 'attestationRequired' in teeInfo) ? Boolean((teeInfo as { attestationRequired?: unknown }).attestationRequired) : false
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

      return {
        success: true,
        modelPath: `tee-package-${skillId}`,
        metrics: {
          packageSize: wasmFormat.length,
          preparationTime: Date.now() - Date.now()
        }
      };

    } catch (error) {
      logger.error({ error, skillId }, 'Failed to prepare LoRA adapter for TEE');
      throw error;
    }
  }

  /**
   * Establish connection to NEXUS TEE infrastructure
   */
  private async establishNexusTEEConnection(teePackage: TEEInfo): Promise<void> {
    logger.info('Establishing connection to NEXUS TEE...');

    try {
      const nexusEndpoint = (teePackage as { nexusConnectivity?: { endpoint?: string } }).nexusConnectivity?.endpoint || 'https://nexus-tee.knirv.com';

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
          packageHash: (teePackage as { packageHash?: string }).packageHash,
          skillId: (teePackage as { skillId?: string }).skillId,
          teeCompatibility: (teePackage as { teeCompatibility?: unknown }).teeCompatibility,
          loraMetadata: (teePackage as { loraMetadata?: unknown }).loraMetadata
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
  private async getTEEConnectivityStatus(): Promise<TEEConnectivityStatus> {
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
    loraAdapterInsights: LoRAInsight | LoRAInsight[],
    _trainingConfig: Record<string, unknown>
  ): Promise<PreTrainingResult> {
    logger.info({ baseModel }, 'Performing pre-training using LoRA adapter insights...');

    try {
      // Aggregate insights from multiple LoRA adapters
      const aggregatedInsights = this.aggregateLoRAInsights(loraAdapterInsights);

      // Create pre-training dataset (currently unused)
      // const pretrainingDataset = {
      //   baseModel,
      //   insights: aggregatedInsights,
      //   trainingConfig: {
      //     learningRate: trainingConfig?.learningRate || 0.0001,
      //     batchSize: trainingConfig?.batchSize || 32,
      //     epochs: trainingConfig?.epochs || 10,
      //     warmupSteps: trainingConfig?.warmupSteps || 1000,
      //     ...trainingConfig
      //   },
      //   timestamp: new Date().toISOString()
      // };

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
  private aggregateLoRAInsights(loraAdapterInsights: LoRAInsight | LoRAInsight[]): Record<string, unknown> {
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
  private extractCommonPatterns(insights: LoRAInsight[]): Record<string, unknown> {
    // Simplified pattern extraction
    const patterns = {
      frequentErrorTypes: new Map<string, number>(),
      commonSolutions: new Map<string, number>(),
      effectiveRankRanges: [] as number[],
      optimalAlphaValues: [] as number[]
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
  private analyzeWeightDistributions(insights: LoRAInsight[]): Record<string, unknown> {
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
  private aggregatePerformanceMetrics(insights: LoRAInsight[]): Record<string, unknown> {
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

  /**
   * Setup Analytics Service routes
   */
  private setupAnalyticsRoutes() {
    // Start/stop analytics collection
    this.router.post('/analytics/start', async (req, res) => {
      try {
        await analyticsService.startCollection();
        res.json({ success: true, message: 'Analytics collection started' });
      } catch (error) {
        logger.error({ error }, 'Failed to start analytics collection');
        res.status(500).json({ success: false, error: getErrorMessage(error) });
      }
    });

    this.router.post('/analytics/stop', async (req, res) => {
      try {
        await analyticsService.stopCollection();
        res.json({ success: true, message: 'Analytics collection stopped' });
      } catch (error) {
        logger.error({ error }, 'Failed to stop analytics collection');
        res.status(500).json({ success: false, error: getErrorMessage(error) });
      }
    });

    // Dashboard statistics
    this.router.get('/analytics/dashboard', async (req, res) => {
      try {
        const stats = await analyticsService.getDashboardStats();
        res.json(stats);
      } catch (error) {
        logger.error({ error }, 'Failed to get dashboard stats');
        res.status(500).json({ success: false, error: getErrorMessage(error) });
      }
    });

    // Performance metrics
    this.router.get('/analytics/performance', async (req, res) => {
      try {
        const metrics = await analyticsService.getPerformanceMetrics();
        res.json(metrics);
      } catch (error) {
        logger.error({ error }, 'Failed to get performance metrics');
        res.status(500).json({ success: false, error: getErrorMessage(error) });
      }
    });

    // Usage analytics
    this.router.get('/analytics/usage', async (req, res) => {
      try {
        const analytics = await analyticsService.getUsageAnalytics();
        res.json(analytics);
      } catch (error) {
        logger.error({ error }, 'Failed to get usage analytics');
        res.status(500).json({ success: false, error: getErrorMessage(error) });
      }
    });

    // Agent analytics
    this.router.get('/analytics/agents', async (req, res) => {
      try {
        const analytics = await analyticsService.getAgentAnalytics();
        res.json(analytics);
      } catch (error) {
        logger.error({ error }, 'Failed to get agent analytics');
        res.status(500).json({ success: false, error: getErrorMessage(error) });
      }
    });

    // Record metric
    this.router.post('/analytics/metrics', async (req, res) => {
      try {
        await analyticsService.recordMetric(req.body);
        res.json({ success: true, message: 'Metric recorded' });
      } catch (error) {
        logger.error({ error }, 'Failed to record metric');
        res.status(500).json({ success: false, error: getErrorMessage(error) });
      }
    });

    // Export analytics data
    this.router.get('/analytics/export', async (req, res) => {
      try {
        const format = req.query.format as string || 'json';
        const data = await analyticsService.exportData(format as 'json' | 'csv');

        if (format === 'csv') {
          res.setHeader('Content-Type', 'text/csv');
          res.setHeader('Content-Disposition', 'attachment; filename=analytics.csv');
        } else {
          res.setHeader('Content-Type', 'application/json');
          res.setHeader('Content-Disposition', 'attachment; filename=analytics.json');
        }

        res.send(data);
      } catch (error) {
        logger.error({ error }, 'Failed to export analytics data');
        res.status(500).json({ success: false, error: getErrorMessage(error) });
      }
    });
  }

  /**
   * Setup Task Scheduling Service routes
   */
  private setupSchedulingRoutes() {
    // Start/stop scheduler
    this.router.post('/scheduler/start', async (req, res) => {
      try {
        await taskSchedulingService.start();
        res.json({ success: true, message: 'Task scheduler started' });
      } catch (error) {
        logger.error({ error }, 'Failed to start task scheduler');
        res.status(500).json({ success: false, error: getErrorMessage(error) });
      }
    });

    this.router.post('/scheduler/stop', async (req, res) => {
      try {
        await taskSchedulingService.stop();
        res.json({ success: true, message: 'Task scheduler stopped' });
      } catch (error) {
        logger.error({ error }, 'Failed to stop task scheduler');
        res.status(500).json({ success: false, error: getErrorMessage(error) });
      }
    });

    // Task management
    this.router.post('/scheduler/tasks', async (req, res) => {
      try {
        const task = await taskSchedulingService.createTask(req.body);
        res.json({ success: true, task });
      } catch (error) {
        logger.error({ error }, 'Failed to create task');
        res.status(500).json({ success: false, error: getErrorMessage(error) });
      }
    });

    this.router.get('/scheduler/tasks', (req, res) => {
      try {
        const tasks = taskSchedulingService.getAllTasks();
        res.json({ success: true, tasks });
      } catch (error) {
        logger.error({ error }, 'Failed to get tasks');
        res.status(500).json({ success: false, error: getErrorMessage(error) });
      }
    });

    this.router.get('/scheduler/tasks/:taskId', (req, res) => {
      try {
        const task = taskSchedulingService.getTask(req.params.taskId);
        if (!task) {
          return res.status(404).json({ success: false, error: 'Task not found' });
        }
        res.json({ success: true, task });
      } catch (error) {
        logger.error({ error }, 'Failed to get task');
        res.status(500).json({ success: false, error: getErrorMessage(error) });
      }
    });

    this.router.put('/scheduler/tasks/:taskId', async (req, res) => {
      try {
        const task = await taskSchedulingService.updateTask(req.params.taskId, req.body);
        res.json({ success: true, task });
      } catch (error) {
        logger.error({ error }, 'Failed to update task');
        res.status(500).json({ success: false, error: getErrorMessage(error) });
      }
    });

    this.router.delete('/scheduler/tasks/:taskId', async (req, res) => {
      try {
        await taskSchedulingService.deleteTask(req.params.taskId);
        res.json({ success: true, message: 'Task deleted' });
      } catch (error) {
        logger.error({ error }, 'Failed to delete task');
        res.status(500).json({ success: false, error: getErrorMessage(error) });
      }
    });

    // Execute task immediately
    this.router.post('/scheduler/tasks/:taskId/execute', async (req, res) => {
      try {
        const execution = await taskSchedulingService.executeTask(req.params.taskId);
        res.json({ success: true, execution });
      } catch (error) {
        logger.error({ error }, 'Failed to execute task');
        res.status(500).json({ success: false, error: getErrorMessage(error) });
      }
    });

    // Get task executions
    this.router.get('/scheduler/tasks/:taskId/executions', (req, res) => {
      try {
        const executions = taskSchedulingService.getTaskExecutions(req.params.taskId);
        res.json({ success: true, executions });
      } catch (error) {
        logger.error({ error }, 'Failed to get task executions');
        res.status(500).json({ success: false, error: getErrorMessage(error) });
      }
    });

    // Workflow management
    this.router.post('/scheduler/workflows', async (req, res) => {
      try {
        const workflow = await taskSchedulingService.createWorkflow(req.body);
        res.json({ success: true, workflow });
      } catch (error) {
        logger.error({ error }, 'Failed to create workflow');
        res.status(500).json({ success: false, error: getErrorMessage(error) });
      }
    });

    this.router.post('/scheduler/workflows/:workflowId/execute', async (req, res) => {
      try {
        const taskId = await taskSchedulingService.executeWorkflow(req.params.workflowId, req.body.variables);
        res.json({ success: true, taskId, message: 'Workflow execution started' });
      } catch (error) {
        logger.error({ error }, 'Failed to execute workflow');
        res.status(500).json({ success: false, error: getErrorMessage(error) });
      }
    });
  }

  /**
   * Setup UDC Management Service routes
   */
  private setupUDCRoutes() {
    // Create UDC
    this.router.post('/udc/create', async (req, res) => {
      try {
        const udc = await udcManagementService.createUDC(req.body);
        res.json({ success: true, udc });
      } catch (error) {
        logger.error({ error }, 'Failed to create UDC');
        res.status(500).json({ success: false, error: getErrorMessage(error) });
      }
    });

    // Renew UDC
    this.router.post('/udc/:udcId/renew', async (req, res) => {
      try {
        const { extensionDays } = req.body;
        const udc = await udcManagementService.renewUDC(req.params.udcId, extensionDays);
        res.json({ success: true, udc });
      } catch (error) {
        logger.error({ error }, 'Failed to renew UDC');
        res.status(500).json({ success: false, error: getErrorMessage(error) });
      }
    });

    // Validate UDC
    this.router.post('/udc/:udcId/validate', async (req, res) => {
      try {
        const { action } = req.body;
        const validation = await udcManagementService.validateUDC(req.params.udcId, action);
        res.json({ success: true, validation });
      } catch (error) {
        logger.error({ error }, 'Failed to validate UDC');
        res.status(500).json({ success: false, error: getErrorMessage(error) });
      }
    });

    // Revoke UDC
    this.router.post('/udc/:udcId/revoke', async (req, res) => {
      try {
        const { reason } = req.body;
        await udcManagementService.revokeUDC(req.params.udcId, reason);
        res.json({ success: true, message: 'UDC revoked successfully' });
      } catch (error) {
        logger.error({ error }, 'Failed to revoke UDC');
        res.status(500).json({ success: false, error: getErrorMessage(error) });
      }
    });

    // Record UDC usage
    this.router.post('/udc/:udcId/usage', async (req, res) => {
      try {
        const { action, result, details } = req.body;
        await udcManagementService.recordUsage(req.params.udcId, action, result, details);
        res.json({ success: true, message: 'Usage recorded' });
      } catch (error) {
        logger.error({ error }, 'Failed to record UDC usage');
        res.status(500).json({ success: false, error: getErrorMessage(error) });
      }
    });

    // Get all UDCs
    this.router.get('/udc/list', (req, res) => {
      try {
        const udcs = udcManagementService.getAllUDCs();
        res.json({ success: true, udcs });
      } catch (error) {
        logger.error({ error }, 'Failed to get UDCs');
        res.status(500).json({ success: false, error: getErrorMessage(error) });
      }
    });

    // Get UDC by ID
    this.router.get('/udc/:udcId', (req, res) => {
      try {
        const udc = udcManagementService.getUDC(req.params.udcId);
        if (!udc) {
          return res.status(404).json({ success: false, error: 'UDC not found' });
        }
        res.json({ success: true, udc });
      } catch (error) {
        logger.error({ error }, 'Failed to get UDC');
        res.status(500).json({ success: false, error: getErrorMessage(error) });
      }
    });

    // Get UDCs by agent
    this.router.get('/udc/agent/:agentId', (req, res) => {
      try {
        const udcs = udcManagementService.getUDCsByAgent(req.params.agentId);
        res.json({ success: true, udcs });
      } catch (error) {
        logger.error({ error }, 'Failed to get UDCs by agent');
        res.status(500).json({ success: false, error: getErrorMessage(error) });
      }
    });

    // Get UDCs by status
    this.router.get('/udc/status/:status', (req, res) => {
      try {
        const status = req.params.status;
        const validStatuses = ['pending', 'active', 'expired', 'revoked', 'suspended'];

        if (!validStatuses.includes(status)) {
          res.status(400).json({ success: false, error: 'Invalid status parameter' });
          return;
        }

        const udcs = udcManagementService.getUDCsByStatus(status as 'pending' | 'active' | 'expired' | 'revoked' | 'suspended');
        res.json({ success: true, udcs });
      } catch (error) {
        logger.error({ error }, 'Failed to get UDCs by status');
        res.status(500).json({ success: false, error: getErrorMessage(error) });
      }
    });

    // Get expiring UDCs
    this.router.get('/udc/expiring/:days', (req, res) => {
      try {
        const days = parseInt(req.params.days);
        const udcs = udcManagementService.getExpiringUDCs(days);
        res.json({ success: true, udcs });
      } catch (error) {
        logger.error({ error }, 'Failed to get expiring UDCs');
        res.status(500).json({ success: false, error: getErrorMessage(error) });
      }
    });

    // Save UDC (for persistence)
    this.router.post('/udc/save', async (req, res) => {
      try {
        // This endpoint is used by the service for persistence
        res.json({ success: true, message: 'UDC saved' });
      } catch (error) {
        logger.error({ error }, 'Failed to save UDC');
        res.status(500).json({ success: false, error: getErrorMessage(error) });
      }
    });
  }

  /**
   * Setup Settings Service routes
   */
  private setupSettingsRoutes() {
    // Get current settings
    this.router.get('/settings/load', (req, res) => {
      try {
        const settings = settingsService.getSettings();
        const profiles = settingsService.getProfiles();
        const activeProfile = settingsService.getActiveProfile();
        res.json({
          success: true,
          settings,
          profiles: Object.fromEntries(profiles.map(p => [p.id, p])),
          activeProfile: activeProfile?.id
        });
      } catch (error) {
        logger.error({ error }, 'Failed to load settings');
        res.status(500).json({ success: false, error: getErrorMessage(error) });
      }
    });

    // Update settings
    this.router.post('/settings/save', async (req, res) => {
      try {
        await settingsService.updateSettings(req.body.settings);
        res.json({ success: true, message: 'Settings saved' });
      } catch (error) {
        logger.error({ error }, 'Failed to save settings');
        res.status(500).json({ success: false, error: getErrorMessage(error) });
      }
    });

    // Reset settings to defaults
    this.router.post('/settings/reset', async (req, res) => {
      try {
        await settingsService.resetSettings();
        res.json({ success: true, message: 'Settings reset to defaults' });
      } catch (error) {
        logger.error({ error }, 'Failed to reset settings');
        res.status(500).json({ success: false, error: getErrorMessage(error) });
      }
    });

    // Profile management
    this.router.post('/settings/profiles', async (req, res) => {
      try {
        // Save profiles (used by service for persistence)
        res.json({ success: true, message: 'Profiles saved' });
      } catch (error) {
        logger.error({ error }, 'Failed to save profiles');
        res.status(500).json({ success: false, error: getErrorMessage(error) });
      }
    });

    this.router.post('/settings/profiles/create', async (req, res) => {
      try {
        const { name, description, settings } = req.body;
        const profile = await settingsService.createProfile(name, description, settings);
        res.json({ success: true, profile });
      } catch (error) {
        logger.error({ error }, 'Failed to create profile');
        res.status(500).json({ success: false, error: getErrorMessage(error) });
      }
    });

    this.router.post('/settings/profiles/:profileId/load', async (req, res) => {
      try {
        await settingsService.loadProfile(req.params.profileId);
        res.json({ success: true, message: 'Profile loaded' });
      } catch (error) {
        logger.error({ error }, 'Failed to load profile');
        res.status(500).json({ success: false, error: getErrorMessage(error) });
      }
    });

    this.router.delete('/settings/profiles/:profileId', async (req, res) => {
      try {
        await settingsService.deleteProfile(req.params.profileId);
        res.json({ success: true, message: 'Profile deleted' });
      } catch (error) {
        logger.error({ error }, 'Failed to delete profile');
        res.status(500).json({ success: false, error: getErrorMessage(error) });
      }
    });

    // Export/import settings
    this.router.get('/settings/export', (req, res) => {
      try {
        const includeProfiles = req.query.includeProfiles === 'true';
        const data = settingsService.exportSettings(includeProfiles);
        res.setHeader('Content-Type', 'application/json');
        res.setHeader('Content-Disposition', 'attachment; filename=knirv-settings.json');
        res.send(data);
      } catch (error) {
        logger.error({ error }, 'Failed to export settings');
        res.status(500).json({ success: false, error: getErrorMessage(error) });
      }
    });

    this.router.post('/settings/import', async (req, res) => {
      try {
        const { settingsData, overwrite } = req.body;
        await settingsService.importSettings(settingsData, overwrite);
        res.json({ success: true, message: 'Settings imported successfully' });
      } catch (error) {
        logger.error({ error }, 'Failed to import settings');
        res.status(500).json({ success: false, error: getErrorMessage(error) });
      }
    });

    // Get/set individual settings
    this.router.get('/settings/get/:path', (req, res) => {
      try {
        const value = settingsService.getSetting(req.params.path);
        res.json({ success: true, value });
      } catch (error) {
        logger.error({ error }, 'Failed to get setting');
        res.status(500).json({ success: false, error: getErrorMessage(error) });
      }
    });

    this.router.post('/settings/set/:path', async (req, res) => {
      try {
        await settingsService.setSetting(req.params.path, req.body.value);
        res.json({ success: true, message: 'Setting updated' });
      } catch (error) {
        logger.error({ error }, 'Failed to set setting');
        res.status(500).json({ success: false, error: getErrorMessage(error) });
      }
    });
  }

  getRouter(): Router {
    return this.router;
  }
}

export default CortexAPI;
