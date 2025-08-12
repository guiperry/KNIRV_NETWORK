/**
 * KNIRV PoAuD (Proof of Audit) Service
 */

import {
  ClientOptions,
  RequestConfig,
  Proof,
  Challenge,
  Reputation,
  PaginationOptions,
  ListResponse,
  KNIRVValidationError,
} from './types';

export class ProofsService {
  constructor(private config: Required<ClientOptions>, private request: (config: RequestConfig) => Promise<any>) {}

  async list(options: PaginationOptions & { 
    skill_id?: string; 
    user_id?: string; 
    status?: string 
  } = {}): Promise<ListResponse<Proof>> {
    const response = await this.request({
      method: 'GET',
      url: '/poaud/proofs',
      params: options,
    });
    return response.data;
  }

  async get(id: string): Promise<Proof> {
    if (!id) {
      throw new KNIRVValidationError('Proof ID is required', 'id');
    }

    const response = await this.request({
      method: 'GET',
      url: `/poaud/proofs/${id}`,
    });
    return response.data;
  }

  async create(data: { skill_id: string; user_id: string; proof_data: any }): Promise<Proof> {
    if (!data.skill_id) {
      throw new KNIRVValidationError('Skill ID is required', 'skill_id');
    }
    if (!data.user_id) {
      throw new KNIRVValidationError('User ID is required', 'user_id');
    }

    const response = await this.request({
      method: 'POST',
      url: '/poaud/proofs',
      data,
    });
    return response.data;
  }

  async update(id: string, data: Partial<Proof>): Promise<Proof> {
    if (!id) {
      throw new KNIRVValidationError('Proof ID is required', 'id');
    }

    const response = await this.request({
      method: 'PUT',
      url: `/poaud/proofs/${id}`,
      data,
    });
    return response.data;
  }
}

export class ChallengesService {
  constructor(private config: Required<ClientOptions>, private request: (config: RequestConfig) => Promise<any>) {}

  async list(options: PaginationOptions & { 
    difficulty?: string; 
    status?: string 
  } = {}): Promise<ListResponse<Challenge>> {
    const response = await this.request({
      method: 'GET',
      url: '/poaud/challenges',
      params: options,
    });
    return response.data;
  }

  async get(id: string): Promise<Challenge> {
    if (!id) {
      throw new KNIRVValidationError('Challenge ID is required', 'id');
    }

    const response = await this.request({
      method: 'GET',
      url: `/poaud/challenges/${id}`,
    });
    return response.data;
  }

  async create(data: { 
    title: string; 
    description: string; 
    difficulty: 'easy' | 'medium' | 'hard'; 
    reward: number 
  }): Promise<Challenge> {
    if (!data.title) {
      throw new KNIRVValidationError('Title is required', 'title');
    }
    if (!data.description) {
      throw new KNIRVValidationError('Description is required', 'description');
    }

    const response = await this.request({
      method: 'POST',
      url: '/poaud/challenges',
      data,
    });
    return response.data;
  }

  async submit(id: string, data: { solution: any; user_id: string }): Promise<any> {
    if (!id) {
      throw new KNIRVValidationError('Challenge ID is required', 'id');
    }
    if (!data.user_id) {
      throw new KNIRVValidationError('User ID is required', 'user_id');
    }

    const response = await this.request({
      method: 'POST',
      url: `/poaud/challenges/${id}/submit`,
      data,
    });
    return response.data;
  }

  async getSubmissions(id: string): Promise<any[]> {
    if (!id) {
      throw new KNIRVValidationError('Challenge ID is required', 'id');
    }

    const response = await this.request({
      method: 'GET',
      url: `/poaud/challenges/${id}/submissions`,
    });
    return response.data;
  }
}

export class ReputationService {
  constructor(private config: Required<ClientOptions>, private request: (config: RequestConfig) => Promise<any>) {}

  async getUserReputation(userId: string): Promise<Reputation> {
    if (!userId) {
      throw new KNIRVValidationError('User ID is required', 'userId');
    }

    const response = await this.request({
      method: 'GET',
      url: `/poaud/reputation/users/${userId}`,
    });
    return response.data;
  }

  async getLeaderboard(options: { limit?: number; offset?: number } = {}): Promise<Reputation[]> {
    const response = await this.request({
      method: 'GET',
      url: '/poaud/reputation/leaderboard',
      params: options,
    });
    return response.data;
  }

  async updateReputation(data: { user_id: string; score_delta: number; reason: string }): Promise<Reputation> {
    if (!data.user_id) {
      throw new KNIRVValidationError('User ID is required', 'user_id');
    }

    const response = await this.request({
      method: 'POST',
      url: '/poaud/reputation/update',
      data,
    });
    return response.data;
  }
}

export class PoAuDService {
  public readonly proofs: ProofsService;
  public readonly challenges: ChallengesService;
  public readonly reputation: ReputationService;
  public readonly client?: any; // For test compatibility

  constructor(config: Required<ClientOptions>, client?: any) {
    this.client = client;
    // Create a request function that can be used by sub-services
    const request = async (requestConfig: RequestConfig) => {
      const url = `${config.baseURL}${requestConfig.url}`;
      
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

    this.proofs = new ProofsService(config, request);
    this.challenges = new ChallengesService(config, request);
    this.reputation = new ReputationService(config, request);
  }

  async verifyProof(proof_hash: string): Promise<{ verified: boolean; details?: any }> {
    if (!proof_hash) {
      throw new KNIRVValidationError('Proof hash is required', 'proof_hash');
    }

    // Placeholder implementation
    return {
      verified: true,
      details: { proof_hash },
    };
  }
}
