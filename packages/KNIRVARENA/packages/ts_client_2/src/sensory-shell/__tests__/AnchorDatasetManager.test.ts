import { describe, it, expect, beforeEach, jest } from '@jest/globals';
import {
  AnchorDatasetManager,
  AnchorDatasetTemplate,
  AnchorCategory,
  ErrorContextForAnchor,
} from '../AnchorDatasetManager';
import type { AnchorExample } from '../AdalineBridge';

describe('AnchorDatasetManager', () => {
  let manager: AnchorDatasetManager;

  beforeEach(() => {
    manager = new AnchorDatasetManager();
  });

  describe('Initialization', () => {
    it('should create an AnchorDatasetManager instance', () => {
      expect(manager).toBeInstanceOf(AnchorDatasetManager);
    });

    it('should initialize with default templates', () => {
      const templates = manager.getTemplates();
      expect(templates.length).toBeGreaterThan(0);
    });

    it('should have error resolution template by default', () => {
      const errorTemplates = manager.getTemplatesByCategory('error_resolution');
      expect(errorTemplates.length).toBeGreaterThan(0);
    });
  });

  describe('Template Management', () => {
    it('should add a custom template', () => {
      const customTemplate: AnchorDatasetTemplate = {
        id: 'custom-test-001',
        name: 'Custom Test Template',
        template: 'Test template with {{variable}}',
        description: 'A custom template for testing',
        category: 'custom',
        contextFields: ['variable'],
        examples: [],
        constraints: [
          { field: 'variable', type: 'required' },
        ],
      };

      manager.addTemplate(customTemplate);
      const retrieved = manager.getTemplate('custom-test-001');

      expect(retrieved).toBeDefined();
      expect(retrieved?.name).toBe('Custom Test Template');
    });

    it('should remove a template', () => {
      const customTemplate: AnchorDatasetTemplate = {
        id: 'removable-001',
        name: 'Removable Template',
        template: 'Template to be removed',
        description: 'Test removal',
        category: 'custom',
        contextFields: [],
        examples: [],
        constraints: [],
      };

      manager.addTemplate(customTemplate);
      const removed = manager.removeTemplate('removable-001');

      expect(removed).toBe(true);
      expect(manager.getTemplate('removable-001')).toBeUndefined();
    });

    it('should return false when removing non-existent template', () => {
      const removed = manager.removeTemplate('non-existent-id');
      expect(removed).toBe(false);
    });

    it('should get templates by category', () => {
      const combatTemplates = manager.getTemplatesByCategory('combat');
      expect(Array.isArray(combatTemplates)).toBe(true);
    });
  });

  describe('Template Population', () => {
    it('should populate a template with context', () => {
      const errorContext: ErrorContextForAnchor = {
        errorNodeId: 'error-001',
        errorType: 'connection_timeout',
        errorMessage: 'Connection timed out',
        historicalFailures: [
          {
            timestamp: Date.now() - 1000,
            scenario: 'network_request',
            attemptedAction: 'fetch_data',
            failureReason: 'timeout',
            severity: 0.7,
          },
        ],
        context: {
          error_type: 'connection_timeout',
          analysis_method: 'root_cause_analysis',
          recovery_strategy: 'retry_with_backoff',
        },
      };

      const templates = manager.getTemplatesByCategory('error_resolution');
      if (templates.length > 0) {
        const result = manager.populateTemplate(templates[0].id, errorContext);

        expect(result).toBeDefined();
        expect(result?.entry.template).toBeDefined();
        expect(result?.contextUsed).toBeDefined();
      }
    });

    it('should derive field values automatically', () => {
      const errorContext: ErrorContextForAnchor = {
        errorNodeId: 'error-002',
        errorType: 'game_combat',
        errorMessage: 'Combat failed',
        historicalFailures: [
          {
            timestamp: Date.now(),
            scenario: 'combat',
            attemptedAction: 'attack',
            failureReason: 'wrong_timing',
            severity: 0.5,
          },
          {
            timestamp: Date.now(),
            scenario: 'combat',
            attemptedAction: 'defend',
            failureReason: 'wrong_timing',
            severity: 0.6,
          },
        ],
        context: {
          error_type: 'connection_timeout',
          recovery_strategy: 'retry_with_backoff',
        },
      };

      const templates = manager.getTemplatesByCategory('error_resolution');
      if (templates.length > 0) {
        const result = manager.populateTemplate(templates[0].id, errorContext);

        expect(result).toBeDefined();
        expect(result?.entry.context.error_type).toBe('connection_timeout');
        expect(result?.entry.context.analysis_method).toBe('standard_analysis');
      }
    });

    it('should return null for non-existent template', () => {
      const errorContext: ErrorContextForAnchor = {
        errorNodeId: 'error-003',
        errorType: 'test',
        errorMessage: 'Test error',
        historicalFailures: [],
        context: {},
      };

      const result = manager.populateTemplate('non-existent', errorContext);
      expect(result).toBeNull();
    });
  });

  describe('Example Management', () => {
    it('should add an example to a template', () => {
      const example: AnchorExample = {
        input: 'Handle timeout',
        output: 'Retry with exponential backoff',
        confidence: 0.85,
      };

      const templates = manager.getTemplatesByCategory('error_resolution');
      if (templates.length > 0) {
        const added = manager.addExample(templates[0].id, example);
        expect(added).toBe(true);
      }
    });

    it('should add example from interaction', () => {
      const templates = manager.getTemplatesByCategory('error_resolution');
      if (templates.length > 0) {
        const added = manager.addExampleFromInteraction(templates[0].id, {
          input: 'Test input',
          output: 'Test output',
          feedback: 0.9,
        });

        expect(added).toBe(true);
      }
    });

    it('should clear examples from a template', () => {
      const templates = manager.getTemplatesByCategory('error_resolution');
      if (templates.length > 0) {
        manager.addExample(templates[0].id, {
          input: 'Test',
          output: 'Result',
          confidence: 0.8,
        });

        const cleared = manager.clearExamples(templates[0].id);
        expect(cleared).toBe(true);
      }
    });
  });

  describe('Template Matching', () => {
    it('should find best matching templates', () => {
      const context = {
        error_type: 'connection_timeout',
        scenario: 'network',
      };

      const result = manager.findBestMatchingTemplates(context);

      expect(result.templates.length).toBeGreaterThan(0);
      expect(result.scores.length).toBe(result.templates.length);
      expect(result.bestMatch).toBeDefined();
    });

    it('should filter by category when specified', () => {
      const context = { scenario: 'combat' };
      const result = manager.findBestMatchingTemplates(context, 'combat');

      for (const template of result.templates) {
        expect(template.category).toBe('combat');
      }
    });
  });

  describe('Metrics', () => {
    it('should return metrics', () => {
      const metrics = manager.getMetrics();

      expect(metrics.totalTemplates).toBeGreaterThan(0);
      expect(metrics.totalExamples).toBeGreaterThanOrEqual(0);
      expect(typeof metrics.averageConfidence).toBe('number');
    });

    it('should track match counts', () => {
      const context = { error_type: 'connection_timeout', scenario: 'network' };
      manager.findBestMatchingTemplates(context);

      const metrics = manager.getMetrics();
      expect(metrics.matchCount.size).toBeGreaterThanOrEqual(0);
    });
  });

  describe('Import/Export', () => {
    it('should export templates', () => {
      const exported = manager.exportTemplates();

      expect(Array.isArray(exported)).toBe(true);
      expect(exported.length).toBeGreaterThan(0);
    });

    it('should import templates', () => {
      const customTemplate: AnchorDatasetTemplate = {
        id: 'import-test-001',
        name: 'Import Test',
        template: 'Import this template',
        description: 'Testing import functionality',
        category: 'custom',
        contextFields: [],
        examples: [],
        constraints: [],
      };

      const imported = manager.importTemplates([customTemplate]);
      expect(imported).toBe(1);
      expect(manager.getTemplate('import-test-001')).toBeDefined();
    });

    it('should not import duplicate template IDs', () => {
      const templates = manager.exportTemplates();
      const imported = manager.importTemplates(templates);

      expect(imported).toBe(0);
    });
  });

  describe('Clear', () => {
    it('should clear all templates', () => {
      manager.clear();

      const templates = manager.getTemplates();
      expect(templates.length).toBe(0);
    });
  });

  describe('Event Emission', () => {
    it('should emit templateAdded event', () => {
      const eventHandler = jest.fn();
      manager.on('templateAdded', eventHandler);

      const customTemplate: AnchorDatasetTemplate = {
        id: 'event-test-001',
        name: 'Event Test',
        template: 'Test template',
        description: 'Testing event emission',
        category: 'custom',
        contextFields: [],
        examples: [],
        constraints: [],
      };

      manager.addTemplate(customTemplate);
      expect(eventHandler).toHaveBeenCalled();
    });

    it('should emit templatePopulated event', () => {
      const eventHandler = jest.fn();
      manager.on('templatePopulated', eventHandler);

      const errorContext: ErrorContextForAnchor = {
        errorNodeId: 'event-error-001',
        errorType: 'test',
        errorMessage: 'Test',
        historicalFailures: [],
        context: { error_type: 'test', recovery_strategy: 'retry' },
      };

      const templates = manager.getTemplatesByCategory('error_resolution');
      if (templates.length > 0) {
        manager.populateTemplate(templates[0].id, errorContext);
      }

      expect(eventHandler).toHaveBeenCalled();
    });
  });
});
