/**
 * HRM WASM Core Model for KNIRVGRAPH Skill Discovery
 * 
 * Uses the existing HRMBridge to analyze and categorize newly trained LoRA adapters
 * Provides skill discovery, naming, and categorization through core model self-training
 */

import pino from 'pino';
import { HRMBridge, HRMConfig, HRMCognitiveInput, HRMCognitiveOutput } from '../../sensory-shell/HRMBridge';
import { LoRAAdapterSkill } from '../lora/LoRAAdapterEngine';
import { TrainingDataset } from './LoRAAdapterTrainingPipeline';

const logger = pino({ name: 'hrm-core-model' });

export interface SkillDiscoveryResult {
  skillId: string;
  discoveredName: string;
  category: string;
  subcategory: string;
  description: string;
  capabilities: string[];
  complexity: number;
  confidence: number;
  tags: string[];
  relatedSkills: string[];
}

export interface SkillAnalysisContext {
  errorPatterns: string[];
  solutionPatterns: string[];
  codeComplexity: number;
  domainContext: string;
  performanceMetrics: {
    accuracy: number;
    efficiency: number;
    reliability: number;
  };
}

export interface CoreModelConfig {
  hrmConfig: HRMConfig;
  analysisTimeout: number;
  maxConcurrentAnalysis: number;
  confidenceThreshold: number;
  categoryMappings: Record<string, string[]>;
}

export class HRMCoreModel {
  private hrmBridge: HRMBridge;
  private config: CoreModelConfig;
  private isInitialized: boolean = false;
  private analysisQueue: Map<string, SkillAnalysisContext> = new Map();
  private discoveryResults: Map<string, SkillDiscoveryResult> = new Map();
  private categoryMappings: Map<string, string[]> = new Map();

  constructor(config: Partial<CoreModelConfig> = {}) {
    this.config = {
      hrmConfig: {
        l_module_count: 8,
        h_module_count: 4,
        enable_adaptation: true,
        processing_timeout: 30000,
      },
      analysisTimeout: 60000,
      maxConcurrentAnalysis: 5,
      confidenceThreshold: 0.7,
      categoryMappings: {
        'debugging': ['error_resolution', 'bug_fixing', 'troubleshooting'],
        'optimization': ['performance', 'efficiency', 'resource_management'],
        'refactoring': ['code_improvement', 'structure_enhancement', 'maintainability'],
        'testing': ['validation', 'verification', 'quality_assurance'],
        'security': ['vulnerability_fixing', 'access_control', 'encryption'],
        'integration': ['api_connection', 'service_linking', 'data_flow'],
        'ui_ux': ['interface_design', 'user_experience', 'accessibility'],
        'data_processing': ['transformation', 'analysis', 'storage'],
      },
      ...config
    };

    this.hrmBridge = new HRMBridge(this.config.hrmConfig);
    this.initializeCategoryMappings();
  }

  /**
   * Initialize the HRM Core Model
   */
  async initialize(): Promise<void> {
    logger.info('Initializing HRM Core Model for skill discovery...');

    try {
      // Initialize HRM WASM bridge
      await this.hrmBridge.initialize();

      this.isInitialized = true;
      logger.info('HRM Core Model initialized successfully');

    } catch (error) {
      logger.error({ error }, 'Failed to initialize HRM Core Model');
      throw error;
    }
  }

  /**
   * Analyze and discover skill from LoRA adapter
   */
  async discoverSkill(
    loraAdapter: LoRAAdapterSkill,
    trainingDataset: TrainingDataset
  ): Promise<SkillDiscoveryResult> {
    if (!this.isInitialized) {
      throw new Error('HRM Core Model not initialized');
    }

    logger.info({ skillId: loraAdapter.skillId }, 'Starting skill discovery analysis...');

    try {
      // Create analysis context from training dataset
      const analysisContext = this.createAnalysisContext(trainingDataset);
      
      // Queue analysis
      this.analysisQueue.set(loraAdapter.skillId, analysisContext);

      // Perform cognitive analysis using HRM
      const cognitiveInput = this.createCognitiveInput(loraAdapter, analysisContext);
      const cognitiveOutput = await this.hrmBridge.processCognitiveInput(cognitiveInput);

      // Extract skill discovery results
      const discoveryResult = this.extractDiscoveryResult(
        loraAdapter,
        analysisContext,
        cognitiveOutput
      );

      // Store results
      this.discoveryResults.set(loraAdapter.skillId, discoveryResult);

      logger.info({
        skillId: loraAdapter.skillId,
        discoveredName: discoveryResult.discoveredName,
        category: discoveryResult.category,
        confidence: discoveryResult.confidence
      }, 'Skill discovery completed');

      return discoveryResult;

    } catch (error) {
      logger.error({ skillId: loraAdapter.skillId, error }, 'Skill discovery failed');
      throw error;
    } finally {
      // Clean up analysis queue
      this.analysisQueue.delete(loraAdapter.skillId);
    }
  }

  /**
   * Create analysis context from training dataset
   */
  private createAnalysisContext(dataset: TrainingDataset): SkillAnalysisContext {
    const errorPatterns = dataset.errorNodes?.map(node =>
      `${node.errorType}: ${node.errorMessage}`
    ) || [];

    const solutionPatterns = dataset.validatedSolutions?.map(solution =>
      `${solution.approach}: ${solution.solutionCode.substring(0, 200)}...`
    ) || [];

    let avgComplexity = 0.5; // Default complexity
    if (dataset.trainingPairs && dataset.trainingPairs.length > 0) {
      const totalComplexity = dataset.trainingPairs.reduce(
        (sum, pair) => {
          const embedding = pair.solutionContext?.codeEmbedding || [0.5];
          return sum + embedding.reduce((a, b) => a + b, 0);
        },
        0
      );
      avgComplexity = totalComplexity / dataset.trainingPairs.length;
    }

    const domainContext = this.extractDomainContext(dataset);

    return {
      errorPatterns,
      solutionPatterns,
      codeComplexity: isNaN(avgComplexity) ? 0.5 : avgComplexity,
      domainContext,
      performanceMetrics: {
        accuracy: dataset.datasetMetrics?.averageValidationScore || 0.5,
        efficiency: dataset.datasetMetrics?.qualityScore || 0.5,
        reliability: dataset.datasetMetrics?.diversityScore || 0.5
      }
    };
  }

  /**
   * Extract domain context from dataset
   */
  private extractDomainContext(dataset: TrainingDataset): string {
    const errorTypes = [...new Set(dataset.errorNodes.map(node => node.errorType))];
    const tags = [...new Set(dataset.errorNodes.flatMap(node => node.tags))];
    
    return `Error types: ${errorTypes.join(', ')}. Tags: ${tags.join(', ')}`;
  }

  /**
   * Create cognitive input for HRM analysis
   */
  private createCognitiveInput(
    loraAdapter: LoRAAdapterSkill,
    context: SkillAnalysisContext
  ): HRMCognitiveInput {
    // Convert LoRA adapter data to sensory input
    const sensoryData = this.convertLoRAToSensoryData(loraAdapter);

    // Create contextual description
    const contextDescription = `
      Skill Analysis Request:
      - LoRA Adapter: ${loraAdapter.skillName}
      - Rank: ${loraAdapter.rank}, Alpha: ${loraAdapter.alpha}
      - Error Patterns: ${context.errorPatterns.slice(0, 3).join('; ')}
      - Solution Patterns: ${context.solutionPatterns.slice(0, 2).join('; ')}
      - Domain: ${context.domainContext}
      - Performance: Accuracy=${context.performanceMetrics.accuracy}, Efficiency=${context.performanceMetrics.efficiency}
      
      Task: Analyze this LoRA adapter and provide skill categorization, naming, and capability assessment.
    `;

    return {
      sensory_data: sensoryData,
      context: contextDescription,
      task_type: 'skill_discovery_analysis'
    };
  }

  /**
   * Convert LoRA adapter to sensory data
   */
  private convertLoRAToSensoryData(loraAdapter: LoRAAdapterSkill): number[] {
    const sensoryData = new Array(256).fill(0);

    // Encode basic metadata
    sensoryData[0] = loraAdapter.rank / 64; // Normalized rank
    sensoryData[1] = loraAdapter.alpha / 32; // Normalized alpha
    sensoryData[2] = loraAdapter.version / 10; // Normalized version

    // Sample weights for pattern recognition
    const weightsASample = Array.from(loraAdapter.weightsA.slice(0, 50));
    const weightsBSample = Array.from(loraAdapter.weightsB.slice(0, 50));

    // Encode weight patterns
    for (let i = 0; i < Math.min(50, weightsASample.length); i++) {
      sensoryData[10 + i] = Math.tanh(weightsASample[i]); // Normalized weights
    }

    for (let i = 0; i < Math.min(50, weightsBSample.length); i++) {
      sensoryData[70 + i] = Math.tanh(weightsBSample[i]); // Normalized weights
    }

    // Encode statistical features
    const weightsAStats = this.calculateWeightStatistics(loraAdapter.weightsA);
    const weightsBStats = this.calculateWeightStatistics(loraAdapter.weightsB);

    sensoryData[130] = weightsAStats.mean;
    sensoryData[131] = weightsAStats.std;
    sensoryData[132] = weightsAStats.skewness;
    sensoryData[133] = weightsBStats.mean;
    sensoryData[134] = weightsBStats.std;
    sensoryData[135] = weightsBStats.skewness;

    return sensoryData;
  }

  /**
   * Calculate weight statistics
   */
  private calculateWeightStatistics(weights: Float32Array): {
    mean: number;
    std: number;
    skewness: number;
  } {
    const array = Array.from(weights);
    const mean = array.reduce((sum, val) => sum + val, 0) / array.length;
    
    const variance = array.reduce((sum, val) => sum + Math.pow(val - mean, 2), 0) / array.length;
    const std = Math.sqrt(variance);
    
    const skewness = array.reduce((sum, val) => sum + Math.pow((val - mean) / std, 3), 0) / array.length;

    return { mean, std, skewness };
  }

  /**
   * Extract discovery result from cognitive output
   */
  private extractDiscoveryResult(
    loraAdapter: LoRAAdapterSkill,
    context: SkillAnalysisContext,
    cognitiveOutput: HRMCognitiveOutput
  ): SkillDiscoveryResult {
    // Parse cognitive reasoning result
    const reasoning = cognitiveOutput.reasoning_result;
    
    // Extract skill name and category using pattern matching and HRM activations
    const discoveredName = this.extractSkillName(reasoning, context);
    const category = this.extractCategory(reasoning, context, cognitiveOutput);
    const subcategory = this.extractSubcategory(category, reasoning);
    const description = this.generateDescription(discoveredName, category, context);
    const capabilities = this.extractCapabilities(reasoning, context);
    const complexity = this.calculateComplexity(context, cognitiveOutput);
    const tags = this.generateTags(category, capabilities, context);
    const relatedSkills = this.findRelatedSkills(category, capabilities);

    return {
      skillId: loraAdapter.skillId,
      discoveredName,
      category,
      subcategory,
      description,
      capabilities,
      complexity,
      confidence: cognitiveOutput.confidence,
      tags,
      relatedSkills
    };
  }

  /**
   * Extract skill name from reasoning
   */
  private extractSkillName(reasoning: string, context: SkillAnalysisContext): string {
    // Extract primary error type and solution approach
    const primaryErrorType = context.errorPatterns[0]?.split(':')[0] || 'General';
    const primaryApproach = context.solutionPatterns[0]?.split(':')[0] || 'Problem Solving';

    // Generate meaningful name
    const cleanErrorType = primaryErrorType.replace(/([A-Z])/g, ' $1').trim();
    const cleanApproach = primaryApproach.replace(/([A-Z])/g, ' $1').trim();

    return `${cleanApproach} ${cleanErrorType} Specialist`;
  }

  /**
   * Extract category from reasoning and activations
   */
  private extractCategory(
    reasoning: string,
    context: SkillAnalysisContext,
    cognitiveOutput: HRMCognitiveOutput
  ): string {
    // Analyze H-module activations for high-level categorization
    const hActivations = cognitiveOutput.h_module_activations;
    const maxActivationIndex = hActivations.indexOf(Math.max(...hActivations));

    // Map activation patterns to categories
    const categories = Object.keys(this.config.categoryMappings);
    const categoryIndex = maxActivationIndex % categories.length;
    
    return categories[categoryIndex];
  }

  /**
   * Extract subcategory
   */
  private extractSubcategory(category: string, reasoning: string): string {
    const subcategories = this.categoryMappings.get(category) || ['general'];
    
    // Simple keyword matching for subcategory
    for (const subcategory of subcategories) {
      if (reasoning.toLowerCase().includes(subcategory.replace('_', ' '))) {
        return subcategory;
      }
    }
    
    return subcategories[0];
  }

  /**
   * Generate description
   */
  private generateDescription(
    skillName: string,
    category: string,
    context: SkillAnalysisContext
  ): string {
    const errorCount = context.errorPatterns.length;
    const avgAccuracy = context.performanceMetrics.accuracy;
    
    return `${skillName} is a ${category} skill trained on ${errorCount} error patterns with ${(avgAccuracy * 100).toFixed(1)}% validation accuracy. Specializes in resolving complex issues through systematic analysis and solution application.`;
  }

  /**
   * Extract capabilities
   */
  private extractCapabilities(reasoning: string, context: SkillAnalysisContext): string[] {
    const capabilities: string[] = [];
    
    // Extract from error patterns
    const errorTypes = [...new Set(context.errorPatterns.map(p => p.split(':')[0]))];
    capabilities.push(...errorTypes.map(type => `${type.toLowerCase()}_resolution`));
    
    // Extract from solution patterns
    if (context.solutionPatterns.some(p => p.includes('refactor'))) {
      capabilities.push('code_refactoring');
    }
    if (context.solutionPatterns.some(p => p.includes('optimize'))) {
      capabilities.push('performance_optimization');
    }
    if (context.solutionPatterns.some(p => p.includes('test'))) {
      capabilities.push('testing_enhancement');
    }
    
    return [...new Set(capabilities)].slice(0, 5); // Limit to 5 capabilities
  }

  /**
   * Calculate complexity score
   */
  private calculateComplexity(
    context: SkillAnalysisContext,
    cognitiveOutput: HRMCognitiveOutput
  ): number {
    const codeComplexity = Math.min(1, Math.abs(context.codeComplexity || 0) / 10);
    const processingComplexity = Math.min(1, (cognitiveOutput.processing_time || 0) / 1000);
    const patternComplexity = Math.min(1, (context.errorPatterns?.length || 0) / 20);

    const complexity = (codeComplexity + processingComplexity + patternComplexity) / 3;
    return isNaN(complexity) ? 0.5 : complexity; // Default to 0.5 if calculation fails
  }

  /**
   * Generate tags
   */
  private generateTags(
    category: string,
    capabilities: string[],
    context: SkillAnalysisContext
  ): string[] {
    const tags = [category];
    tags.push(...capabilities.slice(0, 3));
    
    // Add performance-based tags
    if (context.performanceMetrics.accuracy > 0.9) tags.push('high_accuracy');
    if (context.performanceMetrics.efficiency > 0.8) tags.push('efficient');
    if (context.performanceMetrics.reliability > 0.85) tags.push('reliable');
    
    return [...new Set(tags)];
  }

  /**
   * Find related skills
   */
  private findRelatedSkills(category: string, _capabilities: string[]): string[] {
    // This would typically query existing skills database
    // For now, return placeholder related skills
    const relatedSkills: string[] = [];
    
    // Add category-based related skills
    const categorySkills = this.categoryMappings.get(category) || [];
    relatedSkills.push(...categorySkills.slice(0, 2));
    
    return relatedSkills;
  }

  /**
   * Initialize category mappings
   */
  private initializeCategoryMappings(): void {
    for (const [category, subcategories] of Object.entries(this.config.categoryMappings)) {
      this.categoryMappings.set(category, subcategories);
    }
  }

  /**
   * Get discovery result
   */
  getDiscoveryResult(skillId: string): SkillDiscoveryResult | undefined {
    return this.discoveryResults.get(skillId);
  }

  /**
   * Get all discovery results
   */
  getAllDiscoveryResults(): SkillDiscoveryResult[] {
    return Array.from(this.discoveryResults.values());
  }

  /**
   * Check if ready
   */
  isReady(): boolean {
    return this.isInitialized;
  }
}
