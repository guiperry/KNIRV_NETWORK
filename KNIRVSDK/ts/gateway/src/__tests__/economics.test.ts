/**
 * Comprehensive tests for the KNIRV Gateway Economics service
 */

import { EconomicsService } from '../economics';
import { KNIRVGatewayClient } from '../client';
import { KNIRVAPIError, KNIRVValidationError } from '../types';

// Mock the client
jest.mock('../client');

describe('EconomicsService', () => {
  let mockClient: jest.Mocked<KNIRVGatewayClient>;
  let economicsService: EconomicsService;

  beforeEach(() => {
    mockClient = {
      get: jest.fn(),
      post: jest.fn(),
      put: jest.fn(),
      delete: jest.fn(),
    } as any;

    economicsService = new EconomicsService(mockClient);
  });

  afterEach(() => {
    jest.clearAllMocks();
  });

  describe('Skills Service', () => {
    describe('listSkills', () => {
      it('should list skills successfully', async () => {
        const mockResponse = {
          skills: [
            {
              id: 'skill-1',
              name: 'Network Repair',
              description: 'Repairs network connectivity issues',
              cost: 100,
              success_rate: 0.95
            },
            {
              id: 'skill-2',
              name: 'Data Analysis',
              description: 'Analyzes data patterns',
              cost: 150,
              success_rate: 0.88
            }
          ],
          total: 2,
          page: 1,
          per_page: 10
        };

        mockClient.get.mockResolvedValue(mockResponse as any);

        const result = await economicsService.skills.list();

        expect(mockClient.get).toHaveBeenCalledWith('/economics/skills', {
          params: {}
        });
        expect(result).toEqual(mockResponse);
      });

      it('should list skills with pagination', async () => {
        const mockResponse = {
          skills: [],
          total: 25,
          page: 2,
          per_page: 5
        };

        mockClient.get.mockResolvedValue(mockResponse as any);

        await economicsService.skills.list({ page: 2, per_page: 5 });

        expect(mockClient.get).toHaveBeenCalledWith('/economics/skills', {
          params: { page: 2, per_page: 5 }
        });
      });

      it('should list skills with filters', async () => {
        const mockResponse = { skills: [], total: 0 };
        mockClient.get.mockResolvedValue(mockResponse as any);

        await economicsService.skills.list({
          category: 'network',
          min_cost: 50,
          max_cost: 200,
          verified: true
        });

        expect(mockClient.get).toHaveBeenCalledWith('/economics/skills', {
          params: {
            category: 'network',
            min_cost: 50,
            max_cost: 200,
            verified: true
          }
        });
      });
    });

    describe('getSkill', () => {
      it('should get skill by ID successfully', async () => {
        const mockSkill = {
          id: 'skill-1',
          name: 'Network Repair',
          description: 'Repairs network connectivity issues',
          cost: 100,
          success_rate: 0.95,
          usage_count: 1250,
          total_earned: 125000
        };

        mockClient.get.mockResolvedValue(mockSkill as any);

        const result = await economicsService.skills.get('skill-1');

        expect(mockClient.get).toHaveBeenCalledWith('/economics/skills/skill-1');
        expect(result).toEqual(mockSkill);
      });

      it('should throw error for invalid skill ID', async () => {
        await expect(economicsService.skills.get('')).rejects.toThrow(
          'Skill ID is required'
        );
      });

      it('should handle skill not found', async () => {
        mockClient.get.mockRejectedValue(
          new KNIRVAPIError('Skill not found', 404)
        );

        await expect(economicsService.skills.get('nonexistent')).rejects.toThrow(
          KNIRVAPIError
        );
      });
    });

    describe('createSkill', () => {
      it('should create skill successfully', async () => {
        const skillData = {
          name: 'Test Skill',
          description: 'A test skill',
          cost: 200,
          category: 'testing'
        };

        const mockResponse = {
          id: 'skill-3',
          created: true,
          ...skillData
        };

        mockClient.post.mockResolvedValue(mockResponse as any);

        const result = await economicsService.skills.create(skillData);

        expect(mockClient.post).toHaveBeenCalledWith('/economics/skills', skillData);
        expect(result).toEqual(mockResponse);
      });

      it('should validate required fields', async () => {
        await expect(economicsService.skills.create({})).rejects.toThrow(
          'Name is required'
        );

        await expect(economicsService.skills.create({ name: 'Test' })).rejects.toThrow(
          'Description is required'
        );

        await expect(economicsService.skills.create({
          name: 'Test',
          description: 'Test desc'
        })).rejects.toThrow('Cost is required');
      });

      it('should validate cost is positive', async () => {
        await expect(economicsService.skills.create({
          name: 'Test',
          description: 'Test desc',
          cost: -100
        })).rejects.toThrow('Cost must be positive');
      });

      it('should handle validation errors from API', async () => {
        mockClient.post.mockRejectedValue(
          new KNIRVValidationError('Validation failed', 400, {
            name: ['Name already exists']
          })
        );

        await expect(economicsService.skills.create({
          name: 'Existing Skill',
          description: 'Test',
          cost: 100
        })).rejects.toThrow(KNIRVValidationError);
      });
    });

    describe('updateSkill', () => {
      it('should update skill successfully', async () => {
        const updateData = {
          cost: 120,
          description: 'Updated description'
        };

        const mockResponse = { updated: true };
        mockClient.put.mockResolvedValue(mockResponse as any);

        const result = await economicsService.skills.update('skill-1', updateData);

        expect(mockClient.put).toHaveBeenCalledWith('/economics/skills/skill-1', updateData);
        expect(result).toEqual(mockResponse);
      });

      it('should validate skill ID', async () => {
        await expect(economicsService.skills.update('', {})).rejects.toThrow(
          'Skill ID is required'
        );
      });

      it('should validate update data', async () => {
        await expect(economicsService.skills.update('skill-1', {
          cost: -50
        })).rejects.toThrow('Cost must be positive');
      });
    });

    describe('deleteSkill', () => {
      it('should delete skill successfully', async () => {
        mockClient.delete.mockResolvedValue(undefined);

        await economicsService.skills.delete('skill-1');

        expect(mockClient.delete).toHaveBeenCalledWith('/economics/skills/skill-1');
      });

      it('should validate skill ID', async () => {
        await expect(economicsService.skills.delete('')).rejects.toThrow(
          'Skill ID is required'
        );
      });
    });

    describe('searchSkills', () => {
      it('should search skills successfully', async () => {
        const mockResponse = {
          skills: [
            {
              id: 'skill-1',
              name: 'Network Repair',
              description: 'Repairs network connectivity',
              cost: 100
            }
          ],
          total: 1
        };

        mockClient.get.mockResolvedValue(mockResponse as any);

        const result = await economicsService.skills.search({
          query: 'network',
          category: 'repair'
        });

        expect(mockClient.get).toHaveBeenCalledWith('/economics/skills/search', {
          params: {
            query: 'network',
            category: 'repair'
          }
        });
        expect(result).toEqual(mockResponse);
      });

      it('should handle empty search query', async () => {
        await expect(economicsService.skills.search({
          query: ''
        })).rejects.toThrow('Search query is required');
      });
    });
  });

  describe('LLM Service', () => {
    describe('listModels', () => {
      it('should list LLM models successfully', async () => {
        const mockResponse = {
          models: [
            {
              id: 'model-1',
              name: 'GPT-4',
              cost_per_token: 0.00003,
              max_tokens: 8192,
              provider: 'openai'
            },
            {
              id: 'model-2',
              name: 'Claude-3',
              cost_per_token: 0.000015,
              max_tokens: 4096,
              provider: 'anthropic'
            }
          ]
        };

        mockClient.get.mockResolvedValue(mockResponse as any);

        const result = await economicsService.llm.listModels();

        expect(mockClient.get).toHaveBeenCalledWith('/economics/llm/models');
        expect(result).toEqual(mockResponse);
      });
    });

    describe('getUsage', () => {
      it('should get LLM usage statistics', async () => {
        const mockResponse = {
          total_tokens: 1500000,
          total_cost: 45.50,
          requests: 2500,
          period: '2024-01',
          breakdown: {
            'gpt-4': { tokens: 800000, cost: 24.00 },
            'claude-3': { tokens: 700000, cost: 21.50 }
          }
        };

        mockClient.get.mockResolvedValue(mockResponse as any);

        const result = await economicsService.llm.getUsage();

        expect(mockClient.get).toHaveBeenCalledWith('/economics/llm/usage');
        expect(result).toEqual(mockResponse);
      });

      it('should get usage for specific period', async () => {
        const mockResponse = { total_tokens: 500000, total_cost: 15.00 };
        mockClient.get.mockResolvedValue(mockResponse as any);

        await economicsService.llm.getUsage({ period: '2024-01' });

        expect(mockClient.get).toHaveBeenCalledWith('/economics/llm/usage', {
          params: { period: '2024-01' }
        });
      });
    });

    describe('estimateCost', () => {
      it('should estimate LLM cost successfully', async () => {
        const mockResponse = {
          estimated_cost: 0.15,
          token_count: 5000,
          model: 'gpt-4'
        };

        mockClient.post.mockResolvedValue(mockResponse as any);

        const result = await economicsService.llm.estimateCost({
          text: 'This is a test prompt for cost estimation',
          model: 'gpt-4'
        });

        expect(mockClient.post).toHaveBeenCalledWith('/economics/llm/estimate', {
          text: 'This is a test prompt for cost estimation',
          model: 'gpt-4'
        });
        expect(result).toEqual(mockResponse);
      });

      it('should validate required fields', async () => {
        await expect(economicsService.llm.estimateCost({
          text: '',
          model: 'gpt-4'
        })).rejects.toThrow('Text is required');

        await expect(economicsService.llm.estimateCost({
          text: 'Test',
          model: ''
        })).rejects.toThrow('Model is required');
      });
    });
  });

  describe('Validation Service', () => {
    describe('validate', () => {
      it('should validate skill successfully', async () => {
        const mockResponse = {
          valid: true,
          confidence: 0.95,
          errors: [],
          warnings: ['Cost is higher than average for this category']
        };

        mockClient.post.mockResolvedValue(mockResponse as any);

        const validationData = {
          skill_id: 'skill-1',
          data: {
            cost: 100,
            name: 'Test Skill',
            category: 'testing'
          }
        };

        const result = await economicsService.validation.validate(validationData);

        expect(mockClient.post).toHaveBeenCalledWith('/economics/validation/validate', validationData);
        expect(result).toEqual(mockResponse);
      });

      it('should validate required fields', async () => {
        await expect(economicsService.validation.validate({
          skill_id: '',
          data: {}
        })).rejects.toThrow('Skill ID is required');

        await expect(economicsService.validation.validate({
          skill_id: 'skill-1',
          data: null as any
        })).rejects.toThrow('Data is required');
      });
    });

    describe('listRules', () => {
      it('should list validation rules successfully', async () => {
        const mockResponse = {
          rules: [
            {
              id: 'rule-1',
              name: 'Cost Validation',
              description: 'Validates skill cost ranges',
              active: true,
              category: 'cost'
            },
            {
              id: 'rule-2',
              name: 'Name Validation',
              description: 'Validates skill names',
              active: true,
              category: 'naming'
            }
          ]
        };

        mockClient.get.mockResolvedValue(mockResponse as any);

        const result = await economicsService.validation.listRules();

        expect(mockClient.get).toHaveBeenCalledWith('/economics/validation/rules');
        expect(result).toEqual(mockResponse);
      });

      it('should filter rules by category', async () => {
        const mockResponse = { rules: [] };
        mockClient.get.mockResolvedValue(mockResponse as any);

        await economicsService.validation.listRules({ category: 'cost' });

        expect(mockClient.get).toHaveBeenCalledWith('/economics/validation/rules', {
          params: { category: 'cost' }
        });
      });
    });
  });

  describe('Error Handling', () => {
    it('should handle network errors', async () => {
      mockClient.get.mockRejectedValue(new Error('Network error'));

      await expect(economicsService.skills.list()).rejects.toThrow('Network error');
    });

    it('should handle API errors', async () => {
      mockClient.get.mockRejectedValue(new KNIRVAPIError('API Error', 500));

      await expect(economicsService.skills.list()).rejects.toThrow(KNIRVAPIError);
    });

    it('should handle validation errors', async () => {
      mockClient.post.mockRejectedValue(
        new KNIRVValidationError('Validation failed', 400, {
          name: ['This field is required']
        })
      );

      await expect(economicsService.skills.create({
        name: '',
        description: 'Test',
        cost: 100
      })).rejects.toThrow(KNIRVValidationError);
    });
  });
});
