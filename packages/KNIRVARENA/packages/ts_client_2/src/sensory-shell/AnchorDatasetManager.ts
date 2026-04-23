import { EventEmitter } from './EventEmitter';
import type { AnchorDatasetEntry, AnchorExample } from './AdalineBridge';

export interface AnchorDatasetTemplate {
  id: string;
  name: string;
  template: string;
  description: string;
  category: AnchorCategory;
  contextFields: string[];
  examples: AnchorExample[];
  constraints: TemplateConstraint[];
  metadata?: Record<string, unknown>;
}

export interface TemplateConstraint {
  field: string;
  type: 'required' | 'optional' | 'derived';
  validation?: (value: unknown) => boolean;
  defaultValue?: unknown;
}

export type AnchorCategory =
  | 'error_resolution'
  | 'game_mechanics'
  | 'combat'
  | 'exploration'
  | 'dialogue'
  | 'crafting'
  | 'navigation'
  | 'social_interaction'
  | 'resource_management'
  | 'custom';

export interface ErrorContextForAnchor {
  errorNodeId: string;
  errorType: string;
  errorMessage: string;
  historicalFailures: HistoricalFailure[];
  adversarialDriftData?: AdversarialDriftEntry[];
  context: Record<string, unknown>;
}

export interface HistoricalFailure {
  timestamp: number;
  scenario: string;
  attemptedAction: string;
  failureReason: string;
  severity: number;
  recoveryAction?: string;
}

export interface AdversarialDriftEntry {
  timestamp: number;
  driftType: string;
  detectedPattern: string;
  counterMeasure: string;
  effectiveness: number;
}

export interface PopulatedAnchorDataset {
  entry: AnchorDatasetEntry;
  populatedTemplate: string;
  contextUsed: Record<string, unknown>;
  examplesMatched: number;
  confidence: number;
}

export interface AnchorMatchResult {
  templates: AnchorDatasetTemplate[];
  scores: number[];
  bestMatch: AnchorDatasetTemplate | null;
  allContext: Record<string, unknown>;
}

export interface AnchorDatasetMetrics {
  totalTemplates: number;
  templatesByCategory: Map<AnchorCategory, number>;
  totalExamples: number;
  averageConfidence: number;
  lastUpdate: number;
  matchCount: Map<string, number>;
}

export interface AnchorDatasetConfig {
  maxTemplates: number;
  maxExamplesPerTemplate: number;
  similarityThreshold: number;
  enableAutoLearning: boolean;
  enableContextDerivation: boolean;
  cachePopulatedDatasets: boolean;
  cacheTTL: number;
}

const DEFAULT_ANCHOR_DATASET_CONFIG: AnchorDatasetConfig = {
  maxTemplates: 100,
  maxExamplesPerTemplate: 50,
  similarityThreshold: 0.7,
  enableAutoLearning: true,
  enableContextDerivation: true,
  cachePopulatedDatasets: true,
  cacheTTL: 300000,
};

export class AnchorDatasetManager extends EventEmitter {
  private templates: Map<string, AnchorDatasetTemplate> = new Map();
  private populatedCache: Map<string, PopulatedAnchorDataset> = new Map();
  private config: AnchorDatasetConfig;
  private metrics: AnchorDatasetMetrics;
  private categoryIndex: Map<AnchorCategory, Set<string>> = new Map();

  constructor(config: Partial<AnchorDatasetConfig> = {}) {
    super();
    this.config = { ...DEFAULT_ANCHOR_DATASET_CONFIG, ...config };

    this.metrics = {
      totalTemplates: 0,
      templatesByCategory: new Map(),
      totalExamples: 0,
      averageConfidence: 0,
      lastUpdate: Date.now(),
      matchCount: new Map(),
    };

    this.initializeDefaultTemplates();
  }

  private initializeDefaultTemplates(): void {
    const defaultTemplates: AnchorDatasetTemplate[] = [
      {
        id: 'anchor-error-res-001',
        name: 'Error Resolution Pattern',
        template: 'When encountering {{error_type}}, analyze the situation using {{analysis_method}} and apply the {{recovery_strategy}}. Previous similar errors ({{error_count}}) suggest focusing on {{focus_area}}.',
        description: 'Standard pattern for resolving errors encountered during gameplay',
        category: 'error_resolution',
        contextFields: ['error_type', 'analysis_method', 'recovery_strategy', 'focus_area'],
        examples: [],
        constraints: [
          { field: 'error_type', type: 'required' },
          { field: 'analysis_method', type: 'derived', defaultValue: 'root_cause_analysis' },
          { field: 'recovery_strategy', type: 'required' },
          { field: 'focus_area', type: 'optional' },
        ],
      },
      {
        id: 'anchor-combat-001',
        name: 'Combat Strategy Pattern',
        template: 'In combat with {{enemy_type}}, prioritize {{priority}} while maintaining {{defensive_position}}. Based on {{combat_history_count}} previous encounters, expect {{expected_behavior}}. Counter with {{counter_strategy}}.',
        description: 'Pattern for handling combat scenarios',
        category: 'combat',
        contextFields: ['enemy_type', 'priority', 'defensive_position', 'expected_behavior', 'counter_strategy'],
        examples: [],
        constraints: [
          { field: 'enemy_type', type: 'required' },
          { field: 'priority', type: 'required' },
          { field: 'defensive_position', type: 'optional', defaultValue: 'balanced' },
          { field: 'expected_behavior', type: 'derived' },
          { field: 'counter_strategy', type: 'required' },
        ],
      },
      {
        id: 'anchor-exploration-001',
        name: 'Exploration Pattern',
        template: 'When exploring {{location_type}}, survey for {{survey_priority}} while keeping {{awareness_focus}}. Past explorations ({{exploration_count}}) indicate {{historical_insight}}. Optimal path: {{optimal_path}}.',
        description: 'Pattern for efficient exploration',
        category: 'exploration',
        contextFields: ['location_type', 'survey_priority', 'awareness_focus', 'historical_insight', 'optimal_path'],
        examples: [],
        constraints: [
          { field: 'location_type', type: 'required' },
          { field: 'survey_priority', type: 'required' },
          { field: 'awareness_focus', type: 'optional', defaultValue: 'surroundings' },
          { field: 'historical_insight', type: 'derived' },
          { field: 'optimal_path', type: 'optional' },
        ],
      },
      {
        id: 'anchor-dialogue-001',
        name: 'NPC Dialogue Pattern',
        template: 'When engaging {{npc_name}}, use {{dialogue_style}} approach. Based on relationship level ({{relationship_level}}), expect {{npc_response_pattern}}. Key topics to address: {{key_topics}}.',
        description: 'Pattern for NPC interactions',
        category: 'dialogue',
        contextFields: ['npc_name', 'dialogue_style', 'relationship_level', 'npc_response_pattern', 'key_topics'],
        examples: [],
        constraints: [
          { field: 'npc_name', type: 'required' },
          { field: 'dialogue_style', type: 'required' },
          { field: 'relationship_level', type: 'derived' },
          { field: 'npc_response_pattern', type: 'derived' },
          { field: 'key_topics', type: 'optional' },
        ],
      },
      {
        id: 'anchor-crafting-001',
        name: 'Crafting Pattern',
        template: 'For crafting {{item_type}}, gather {{required_materials}} using {{gather_method}}. Based on {{crafting_history_count}} previous crafts, apply {{technique_adjustment}}. Expected quality: {{expected_quality}}.',
        description: 'Pattern for crafting items',
        category: 'crafting',
        contextFields: ['item_type', 'required_materials', 'gather_method', 'technique_adjustment', 'expected_quality'],
        examples: [],
        constraints: [
          { field: 'item_type', type: 'required' },
          { field: 'required_materials', type: 'required' },
          { field: 'gather_method', type: 'optional', defaultValue: 'standard' },
          { field: 'technique_adjustment', type: 'derived' },
          { field: 'expected_quality', type: 'derived' },
        ],
      },
    ];

    for (const template of defaultTemplates) {
      this.addTemplate(template);
    }
  }

  public addTemplate(template: AnchorDatasetTemplate): void {
    if (this.templates.size >= this.config.maxTemplates) {
      this.emit('templateLimitReached', { count: this.templates.size });
      return;
    }

    this.templates.set(template.id, template);

    if (!this.categoryIndex.has(template.category)) {
      this.categoryIndex.set(template.category, new Set());
    }
    this.categoryIndex.get(template.category)!.add(template.id);

    this.updateMetrics();

    this.emit('templateAdded', { template });
  }

  public removeTemplate(templateId: string): boolean {
    const template = this.templates.get(templateId);
    if (!template) {
      return false;
    }

    this.templates.delete(templateId);

    const categorySet = this.categoryIndex.get(template.category);
    if (categorySet) {
      categorySet.delete(templateId);
    }

    this.updateMetrics();

    this.emit('templateRemoved', { templateId });
    return true;
  }

  public getTemplate(templateId: string): AnchorDatasetTemplate | undefined {
    return this.templates.get(templateId);
  }

  public getTemplatesByCategory(category: AnchorCategory): AnchorDatasetTemplate[] {
    const templateIds = this.categoryIndex.get(category);
    if (!templateIds) {
      return [];
    }

    return Array.from(templateIds)
      .map((id) => this.templates.get(id))
      .filter((t): t is AnchorDatasetTemplate => t !== undefined);
  }

  public populateTemplate(
    templateId: string,
    errorContext: ErrorContextForAnchor,
    additionalContext?: Record<string, unknown>
  ): PopulatedAnchorDataset | null {
    const template = this.templates.get(templateId);
    if (!template) {
      return null;
    }

    const context = { ...errorContext.context, ...additionalContext };
    const populatedValues: Record<string, unknown> = {};

    for (const constraint of template.constraints) {
      if (constraint.type === 'required' && !context[constraint.field]) {
        console.warn(`Required field ${constraint.field} not found in context`);
        return null;
      }

      if (constraint.type === 'derived') {
        populatedValues[constraint.field] = this.deriveFieldValue(
          constraint.field,
          context,
          errorContext
        );
      } else {
        populatedValues[constraint.field] =
          context[constraint.field] ?? constraint.defaultValue ?? '';
      }
    }

    let populatedTemplate = template.template;

    for (const [key, value] of Object.entries(populatedValues)) {
      const placeholder = `{{${key}}}`;
      populatedTemplate = populatedTemplate.replace(
        new RegExp(placeholder.replace(/[{}]/g, '\\$&'), 'g'),
        String(value)
      );
    }

    const matchedExamples = this.findMatchingExamples(template, context);

    const entry: AnchorDatasetEntry = {
      template: populatedTemplate,
      context: populatedValues as Record<string, unknown>,
      examples: matchedExamples,
      metadata: {
        templateId,
        templateName: template.name,
        category: template.category,
        populatedAt: Date.now(),
      },
    };

    const populatedDataset: PopulatedAnchorDataset = {
      entry,
      populatedTemplate,
      contextUsed: context,
      examplesMatched: matchedExamples.length,
      confidence: this.calculatePopulatedConfidence(template, matchedExamples, context),
    };

    if (this.config.cachePopulatedDatasets) {
      this.cachePopulatedDataset(templateId, populatedDataset);
    }

    this.recordMatch(templateId);

    this.emit('templatePopulated', populatedDataset);

    return populatedDataset;
  }

  private deriveFieldValue(
    field: string,
    context: Record<string, unknown>,
    errorContext: ErrorContextForAnchor
  ): unknown {
    switch (field) {
      case 'analysis_method':
        if (errorContext.adversarialDriftData && errorContext.adversarialDriftData.length > 0) {
          return 'adversarial_analysis';
        }
        return 'standard_analysis';

      case 'expected_behavior':
        const enemyType = context.enemy_type as string;
        if (enemyType.includes('boss')) return 'aggressive_attacks';
        if (enemyType.includes('archer')) return 'ranged_attacks';
        return 'balanced_assault';

      case 'historical_insight':
        const historyCount = (context.exploration_count as number) || 0;
        if (historyCount > 10) return 'well_mapped_area';
        if (historyCount > 5) return 'partially_explored';
        return 'uncharted_territory';

      case 'relationship_level':
        const npcName = context.npc_name as string;
        return Math.min(100, Math.max(0, npcName.length * 5 + 50));

      case 'npc_response_pattern':
        const relationshipLevel = context.relationship_level as number;
        if (relationshipLevel >= 75) return 'friendly_and_helpful';
        if (relationshipLevel >= 50) return 'neutral_cooperative';
        if (relationshipLevel >= 25) return 'cautious_reserved';
        return 'hostile_dismissive';

      case 'technique_adjustment':
        const historyCount2 = (context.crafting_history_count as number) || 0;
        if (historyCount2 > 20) return 'advanced_optimization';
        if (historyCount2 > 10) return 'standard_refinement';
        return 'basic_fundamentals';

      case 'expected_quality':
        const techniqueAdjustment = context.technique_adjustment as string;
        if (techniqueAdjustment.includes('advanced')) return 'excellent';
        if (techniqueAdjustment.includes('standard')) return 'good';
        return 'standard';

      case 'error_count':
        return errorContext.historicalFailures?.length || 0;

      case 'focus_area':
        if (errorContext.historicalFailures.length > 0) {
          const mostCommonFailure = this.findMostCommonFailure(errorContext.historicalFailures);
          return mostCommonFailure?.failureReason || 'general_optimization';
        }
        return 'preventive_measures';

      case 'combat_history_count':
        return errorContext.historicalFailures.filter((f) =>
          f.scenario.includes('combat')
        ).length;

      case 'exploration_count':
        return errorContext.historicalFailures.filter((f) =>
          f.scenario.includes('exploration')
        ).length;

      case 'crafting_history_count':
        return errorContext.historicalFailures.filter((f) =>
          f.scenario.includes('crafting')
        ).length;

      default:
        return context[field] ?? `derived_${field}`;
    }
  }

  private findMostCommonFailure(failures: HistoricalFailure[]): HistoricalFailure | null {
    if (failures.length === 0) return null;

    const failureReasons = new Map<string, { count: number; failure: HistoricalFailure }>();

    for (const failure of failures) {
      const existing = failureReasons.get(failure.failureReason);
      if (existing) {
        existing.count++;
        if (failure.severity > existing.failure.severity) {
          existing.failure = failure;
        }
      } else {
        failureReasons.set(failure.failureReason, { count: 1, failure });
      }
    }

    let maxCount = 0;
    let mostCommon: HistoricalFailure | null = null;

    for (const { count, failure } of failureReasons.values()) {
      if (count > maxCount) {
        maxCount = count;
        mostCommon = failure;
      }
    }

    return mostCommon;
  }

  private findMatchingExamples(
    template: AnchorDatasetTemplate,
    context: Record<string, unknown>
  ): AnchorExample[] {
    const matched: AnchorExample[] = [];

    for (const example of template.examples) {
      if (matched.length >= this.config.maxExamplesPerTemplate) {
        break;
      }

      const similarity = this.calculateExampleSimilarity(example, context);
      if (similarity >= this.config.similarityThreshold) {
        matched.push(example);
      }
    }

    return matched;
  }

  private calculateExampleSimilarity(
    example: AnchorExample,
    context: Record<string, unknown>
  ): number {
    const contextText = Object.values(context).join(' ').toLowerCase();
    const exampleInput = example.input.toLowerCase();
    const exampleOutput = example.output.toLowerCase();

    const contextWords = new Set(contextText.split(/\s+/).filter((w) => w.length > 3));
    const inputWords = new Set(exampleInput.split(/\s+/).filter((w) => w.length > 3));
    const outputWords = new Set(exampleOutput.split(/\s+/).filter((w) => w.length > 3));

    let matches = 0;
    let total = 0;

    for (const word of contextWords) {
      if (inputWords.has(word) || outputWords.has(word)) {
        matches++;
      }
      total++;
    }

    if (total === 0) return 0;

    return matches / total;
  }

  private calculatePopulatedConfidence(
    template: AnchorDatasetTemplate,
    examples: AnchorExample[],
    context: Record<string, unknown>
  ): number {
    let confidence = 0.5;

    if (examples.length > 0) {
      const avgExampleConfidence =
        examples.reduce((sum, e) => sum + e.confidence, 0) / examples.length;
      confidence += avgExampleConfidence * 0.3;
    }

    const filledPlaceholders = template.template.match(/\{\{[^}]+\}\}/g) || [];
    const filledCount = filledPlaceholders.filter((p) => {
      const fieldName = p.replace(/[{}]/g, '');
      return context[fieldName] !== undefined;
    }).length;

    const fillRatio = filledCount / filledPlaceholders.length;
    confidence += fillRatio * 0.2;

    return Math.min(1, Math.max(0, confidence));
  }

  private cachePopulatedDataset(
    templateId: string,
    dataset: PopulatedAnchorDataset
  ): void {
    const cacheKey = `${templateId}_${Date.now()}`;
    const cacheEntry = {
      dataset,
      timestamp: Date.now(),
    };

    this.populatedCache.set(cacheKey, dataset);

    setTimeout(() => {
      this.populatedCache.delete(cacheKey);
    }, this.config.cacheTTL);
  }

  private recordMatch(templateId: string): void {
    const currentCount = this.metrics.matchCount.get(templateId) || 0;
    this.metrics.matchCount.set(templateId, currentCount + 1);
  }

  public findBestMatchingTemplates(
    context: Record<string, unknown>,
    category?: AnchorCategory
  ): AnchorMatchResult {
    const contextText = JSON.stringify(context).toLowerCase();
    const templatesToSearch = category
      ? this.getTemplatesByCategory(category)
      : Array.from(this.templates.values());

    const scores: number[] = [];
    const scoredTemplates: Array<{ template: AnchorDatasetTemplate; score: number }> = [];

    for (const template of templatesToSearch) {
      const score = this.calculateTemplateMatchScore(template, contextText, context);
      scores.push(score);
      scoredTemplates.push({ template, score });
    }

    scoredTemplates.sort((a, b) => b.score - a.score);

    const sortedTemplates = scoredTemplates.map((s) => s.template);
    const sortedScores = scoredTemplates.map((s) => s.score);

    return {
      templates: sortedTemplates.slice(0, 5),
      scores: sortedScores.slice(0, 5),
      bestMatch: sortedTemplates[0] || null,
      allContext: context,
    };
  }

  private calculateTemplateMatchScore(
    template: AnchorDatasetTemplate,
    contextText: string,
    context: Record<string, unknown>
  ): number {
    let score = 0;

    const templateText = `${template.name} ${template.description} ${template.template}`.toLowerCase();
    const templateWords = new Set(templateText.split(/\s+/).filter((w) => w.length > 3));
    const contextWords = new Set(contextText.split(/\s+/).filter((w) => w.length > 3));

    let wordMatches = 0;
    for (const word of templateWords) {
      if (contextWords.has(word)) {
        wordMatches++;
      }
    }

    if (templateWords.size > 0) {
      score += (wordMatches / templateWords.size) * 0.4;
    }

    const requiredFields = template.constraints
      .filter((c) => c.type === 'required')
      .map((c) => c.field);

    const matchedRequiredFields = requiredFields.filter((field) =>
      context.hasOwnProperty(field)
    );

    if (requiredFields.length > 0) {
      score += (matchedRequiredFields.length / requiredFields.length) * 0.4;
    }

    const exampleMatchCount = template.examples.filter((example) =>
      this.calculateExampleSimilarity(example, context) >= this.config.similarityThreshold
    ).length;

    if (template.examples.length > 0) {
      score += (exampleMatchCount / Math.min(template.examples.length, 10)) * 0.2;
    }

    return Math.min(1, Math.max(0, score));
  }

  public addExample(templateId: string, example: AnchorExample): boolean {
    const template = this.templates.get(templateId);
    if (!template) {
      return false;
    }

    if (template.examples.length >= this.config.maxExamplesPerTemplate) {
      this.emit('exampleLimitReached', { templateId, count: template.examples.length });
      return false;
    }

    template.examples.push(example);
    this.updateMetrics();

    this.emit('exampleAdded', { templateId, example });

    if (this.config.enableAutoLearning) {
      this.emit('learningTriggered', { type: 'example_added', templateId });
    }

    return true;
  }

  public addExampleFromInteraction(
    templateId: string,
    interaction: { input: string; output: string; feedback: number }
  ): boolean {
    const example: AnchorExample = {
      input: interaction.input,
      output: interaction.output,
      confidence: Math.max(0, Math.min(1, interaction.feedback)),
    };

    return this.addExample(templateId, example);
  }

  public clearExamples(templateId: string): boolean {
    const template = this.templates.get(templateId);
    if (!template) {
      return false;
    }

    template.examples = [];
    this.updateMetrics();

    this.emit('examplesCleared', { templateId });
    return true;
  }

  private updateMetrics(): void {
    this.metrics.totalTemplates = this.templates.size;
    this.metrics.totalExamples = 0;
    this.metrics.templatesByCategory.clear();

    for (const template of this.templates.values()) {
      this.metrics.totalExamples += template.examples.length;

      const currentCount = this.metrics.templatesByCategory.get(template.category) || 0;
      this.metrics.templatesByCategory.set(template.category, currentCount + 1);
    }

    let totalConfidence = 0;
    let exampleCount = 0;

    for (const template of this.templates.values()) {
      for (const example of template.examples) {
        totalConfidence += example.confidence;
        exampleCount++;
      }
    }

    this.metrics.averageConfidence = exampleCount > 0 ? totalConfidence / exampleCount : 0;
    this.metrics.lastUpdate = Date.now();
  }

  public getMetrics(): AnchorDatasetMetrics {
    return { ...this.metrics };
  }

  public exportTemplates(): AnchorDatasetTemplate[] {
    return Array.from(this.templates.values());
  }

  public importTemplates(templates: AnchorDatasetTemplate[]): number {
    let imported = 0;

    for (const template of templates) {
      if (!this.templates.has(template.id) && this.templates.size < this.config.maxTemplates) {
        this.addTemplate(template);
        imported++;
      }
    }

    return imported;
  }

  public clear(): void {
    this.templates.clear();
    this.populatedCache.clear();
    this.categoryIndex.clear();
    this.updateMetrics();

    this.emit('cleared');
  }

  public getTemplates(): AnchorDatasetTemplate[] {
    return Array.from(this.templates.values());
  }
}

let anchorDatasetManagerInstance: AnchorDatasetManager | null = null;

export const getAnchorDatasetManager = (
  config?: Partial<AnchorDatasetConfig>
): AnchorDatasetManager => {
  if (!anchorDatasetManagerInstance) {
    anchorDatasetManagerInstance = new AnchorDatasetManager(config);
  }
  return anchorDatasetManagerInstance;
};

export default getAnchorDatasetManager;
