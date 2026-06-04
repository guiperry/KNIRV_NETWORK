/**
 * KNIRV Economics Service
 */

import {
  ClientOptions,
  RequestConfig,
  Skill,
  SkillCreateRequest,
  SkillUpdateRequest,
  SkillSearchRequest,
  LLMModel,
  LLMUsage,
  LLMCostEstimate,
  LLMCostEstimateRequest,
  ValidationRequest,
  ValidationResponse,
  ValidationRule,
  PaginationOptions,
  PaginatedResponse,
  ListResponse,
  KNIRVValidationError,
} from './types';

export class SkillsService {
  constructor(private config: Required<ClientOptions>, private request: (config: RequestConfig) => Promise<any>) {}

  async list(options: PaginationOptions & {
    category?: string;
    min_cost?: number;
    max_cost?: number;
    verified?: boolean
  } = {}): Promise<ListResponse<Skill>> {
    const response = await this.request({
      method: 'GET',
      url: '/economics/skills',
      params: options,
    });
    return response.data;
  }

  async get(id: string): Promise<Skill> {
    if (!id) {
      throw new KNIRVValidationError('Skill ID is required', 'id');
    }

    const response = await this.request({
      method: 'GET',
      url: `/economics/skills/${id}`,
    });
    return response.data;
  }

  async create(data: SkillCreateRequest): Promise<Skill> {
    if (!data.name) {
      throw new KNIRVValidationError('Name is required', 'name');
    }
    if (!data.description) {
      throw new KNIRVValidationError('Description is required', 'description');
    }
    if (data.cost === undefined) {
      throw new KNIRVValidationError('Cost is required', 'cost');
    }
    if (data.capabilities && data.capabilities.length > 0) {
      if (data.capabilities.some(cap => typeof cap !== 'string' || cap.trim() === '')) {
        throw new KNIRVValidationError('All capabilities must be non-empty strings', 'capabilities');
      }
    }
    if (data.cost !== undefined && data.cost < 0) {
      throw new KNIRVValidationError('Cost must be positive', 'cost');
    }

    const response = await this.request({
      method: 'POST',
      url: '/economics/skills',
      data,
    });
    return response.data;
  }

  async update(id: string, data: SkillUpdateRequest): Promise<Skill> {
    if (!id) {
      throw new KNIRVValidationError('Skill ID is required', 'id');
    }
    if (data.cost !== undefined && data.cost < 0) {
      throw new KNIRVValidationError('Cost must be positive', 'cost');
    }
    if (data.capabilities && data.capabilities.some(cap => typeof cap !== 'string' || cap.trim() === '')) {
      throw new KNIRVValidationError('All capabilities must be non-empty strings', 'capabilities');
    }

    const response = await this.request({
      method: 'PUT',
      url: `/economics/skills/${id}`,
      data,
    });
    return response.data;
  }

  async delete(id: string): Promise<void> {
    if (!id) {
      throw new KNIRVValidationError('Skill ID is required', 'id');
    }

    await this.request({
      method: 'DELETE',
      url: `/economics/skills/${id}`,
    });
  }

  async search(options: SkillSearchRequest): Promise<ListResponse<Skill>> {
    if (!options.query && !options.category && !options.capabilities) {
      throw new KNIRVValidationError('Search query is required');
    }

    const response = await this.request({
      method: 'GET',
      url: '/economics/skills/search',
      params: options,
    });
    return response.data;
  }
}

export class LLMService {
  constructor(private config: Required<ClientOptions>, private request: (config: RequestConfig) => Promise<any>) {}

  async listModels(): Promise<ListResponse<LLMModel>> {
    const response = await this.request({
      method: 'GET',
      url: '/economics/llm/models',
    });
    return response.data;
  }

  async getUsage(options: { period?: string } = {}): Promise<LLMUsage> {
    const requestConfig: any = {
      method: 'GET',
      url: '/economics/llm/usage',
    };

    // Only add params if options has actual properties
    if (Object.keys(options).length > 0) {
      requestConfig.params = options;
    }

    const response = await this.request(requestConfig);
    return response.data;
  }

  async estimateCost(request: LLMCostEstimateRequest): Promise<LLMCostEstimate> {
    if (!request.text || request.text.trim() === '') {
      throw new KNIRVValidationError('Text is required');
    }
    if (!request.model || request.model.trim() === '') {
      throw new KNIRVValidationError('Model is required', 'model');
    }

    const response = await this.request({
      method: 'POST',
      url: '/economics/llm/estimate',
      data: request,
    });
    return response.data;
  }
}

export class ValidationService {
  constructor(private config: Required<ClientOptions>, private request: (config: RequestConfig) => Promise<any>) {}

  async validate(request: ValidationRequest): Promise<ValidationResponse> {
    if (!request.skill_id || request.skill_id.trim() === '') {
      throw new KNIRVValidationError('Skill ID is required');
    }
    if (!request.data) {
      throw new KNIRVValidationError('Data is required', 'data');
    }

    const response = await this.request({
      method: 'POST',
      url: '/economics/validation/validate',
      data: request,
    });
    return response.data;
  }

  async listRules(options: { category?: string } = {}): Promise<ListResponse<ValidationRule>> {
    const requestConfig: any = {
      method: 'GET',
      url: '/economics/validation/rules',
    };

    // Only add params if options has actual properties
    if (Object.keys(options).length > 0) {
      requestConfig.params = options;
    }

    const response = await this.request(requestConfig);
    return response.data;
  }
}

export class EconomicsService {
  public readonly skills: SkillsService;
  public readonly llm: LLMService;
  public readonly validation: ValidationService;
  public readonly client?: any; // For test compatibility

  constructor(configOrClient: Required<ClientOptions> | any, client?: any) {
    // Check if this is a client object (has get/post/put/delete methods) or a config
    if (configOrClient && typeof configOrClient.get === 'function') {
      // This is a client object (likely a mock in tests)
      const client = configOrClient;
      this.client = client;

      // Create a request function that uses the client's methods
      const request = async (requestConfig: RequestConfig) => {
        const method = requestConfig.method?.toLowerCase() || 'get';
        const url = requestConfig.url || '';

        let result;
        switch (method) {
          case 'get':
            // Pass params if they are defined (even if empty object)
            if (requestConfig.params !== undefined) {
              result = await client.get(url, { params: requestConfig.params });
            } else {
              result = await client.get(url);
            }
            break;
          case 'post':
            // Pass params if they are defined (even if empty object)
            if (requestConfig.params !== undefined) {
              result = await client.post(url, requestConfig.data, { params: requestConfig.params });
            } else {
              result = await client.post(url, requestConfig.data);
            }
            break;
          case 'put':
            // Pass params if they are defined (even if empty object)
            if (requestConfig.params !== undefined) {
              result = await client.put(url, requestConfig.data, { params: requestConfig.params });
            } else {
              result = await client.put(url, requestConfig.data);
            }
            break;
          case 'delete':
            // Pass params if they are defined (even if empty object)
            if (requestConfig.params !== undefined) {
              result = await client.delete(url, { params: requestConfig.params });
            } else {
              result = await client.delete(url);
            }
            break;
          default:
            throw new Error(`Unsupported method: ${method}`);
        }

        // For mock clients, return the result directly since the service methods expect response.data
        // The mock client should return the data directly, not wrapped in APIResponse
        return {
          data: result,
          status: 200,
          statusText: 'OK',
          headers: {},
        };
      };

      this.skills = new SkillsService(configOrClient, request);
      this.llm = new LLMService(configOrClient, request);
      this.validation = new ValidationService(configOrClient, request);
    } else {
      // This is a config object
      const config = configOrClient;
      this.client = client;

      // Create a request function that can be used by sub-services
      const request = async (requestConfig: RequestConfig) => {
        const url = `${config.economicsURL}${requestConfig.url}`;

        const headers: Record<string, string> = {
          'Content-Type': 'application/json',
          ...requestConfig.headers,
        };

        if (config.apiKey) {
          headers['Authorization'] = `Bearer ${config.apiKey}`;
        }

        const options: RequestInit = {
          method: requestConfig.method || 'GET',
          headers,
        };

        if (requestConfig.data && requestConfig.method !== 'GET') {
          options.body = JSON.stringify(requestConfig.data);
        }

        // Add query parameters
        const urlWithParams = new URL(url);
        if (requestConfig.params) {
          Object.entries(requestConfig.params).forEach(([key, value]) => {
            if (value !== undefined && value !== null) {
              if (Array.isArray(value)) {
                value.forEach(v => urlWithParams.searchParams.append(key, String(v)));
              } else {
                urlWithParams.searchParams.append(key, String(value));
              }
            }
          });
        }

        const response = await fetch(urlWithParams.toString(), options);

        let data: any;
        const contentType = response.headers.get('content-type');

        if (contentType && contentType.includes('application/json')) {
          data = await response.json();
        } else {
          data = await response.text();
        }

        return {
          data,
          status: response.status,
          statusText: response.statusText,
          headers: Object.fromEntries(response.headers.entries()),
        };
      };

      this.skills = new SkillsService(config, request);
      this.llm = new LLMService(config, request);
      this.validation = new ValidationService(config, request);
    }
  }
}
