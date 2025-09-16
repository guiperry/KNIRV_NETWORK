/**
 * API Key Service
 * Manages API key generation, validation, and authentication for KNIRVCONTROLLER
 */

import { rxdbService } from './RxDBService';
import crypto from 'crypto';

export interface ApiKey {
  id: string;
  key: string;
  name: string;
  description: string;
  permissions: string[];
  createdAt: Date;
  lastUsed?: Date;
  expiresAt?: Date;
  isActive: boolean;
  usageCount: number;
  rateLimit: {
    requestsPerMinute: number;
    requestsPerHour: number;
    requestsPerDay: number;
  };
  metadata: Record<string, unknown>;
}

export interface ApiKeyUsage {
  keyId: string;
  endpoint: string;
  method: string;
  timestamp: Date;
  ipAddress: string;
  userAgent: string;
  responseStatus: number;
  responseTime: number;
}

export interface CreateApiKeyRequest {
  name: string;
  description: string;
  permissions: string[];
  expiresAt?: Date;
  rateLimit?: {
    requestsPerMinute?: number;
    requestsPerHour?: number;
    requestsPerDay?: number;
  };
}

class ApiKeyService {
  private readonly defaultRateLimit = {
    requestsPerMinute: 60,
    requestsPerHour: 1000,
    requestsPerDay: 10000
  };

  private readonly availablePermissions = [
    'read:agents',
    'write:agents',
    'read:graph',
    'write:graph',
    'read:cortex',
    'write:cortex',
    'read:wallet',
    'write:wallet',
    'read:skills',
    'write:skills',
    'read:analytics',
    'admin:all'
  ];

  /**
   * Generate a new API key
   */
  async createApiKey(request: CreateApiKeyRequest): Promise<ApiKey> {
    const keyId = this.generateId();
    const apiKey = this.generateApiKey();

    const newKey: ApiKey = {
      id: keyId,
      key: apiKey,
      name: request.name,
      description: request.description,
      permissions: this.validatePermissions(request.permissions),
      createdAt: new Date(),
      expiresAt: request.expiresAt,
      isActive: true,
      usageCount: 0,
      rateLimit: {
        ...this.defaultRateLimit,
        ...request.rateLimit
      },
      metadata: {}
    };

    await this.saveApiKey(newKey);
    return newKey;
  }

  /**
   * Validate an API key and return key information
   */
  async validateApiKey(key: string): Promise<ApiKey | null> {
    try {
      const apiKeys = await this.getApiKeys();
      const apiKey = apiKeys.find(k => k.key === key);

      if (!apiKey) {
        return null;
      }

      // Check if key is active
      if (!apiKey.isActive) {
        return null;
      }

      // Check if key has expired
      if (apiKey.expiresAt && new Date() > apiKey.expiresAt) {
        await this.deactivateApiKey(apiKey.id);
        return null;
      }

      // Update last used timestamp
      await this.updateLastUsed(apiKey.id);

      return apiKey;
    } catch (error) {
      console.error('Failed to validate API key:', error);
      return null;
    }
  }

  /**
   * Check if API key has required permission
   */
  hasPermission(apiKey: ApiKey, requiredPermission: string): boolean {
    // Admin permission grants all access
    if (apiKey.permissions.includes('admin:all')) {
      return true;
    }

    return apiKey.permissions.includes(requiredPermission);
  }

  /**
   * Check rate limits for API key
   */
  async checkRateLimit(apiKey: ApiKey): Promise<{ allowed: boolean; resetTime?: Date }> {
    try {
      const now = new Date();
      const usage = await this.getApiKeyUsage(apiKey.id);

      // Check minute limit
      const minuteAgo = new Date(now.getTime() - 60 * 1000);
      const minuteUsage = usage.filter(u => u.timestamp > minuteAgo).length;
      if (minuteUsage >= apiKey.rateLimit.requestsPerMinute) {
        return { allowed: false, resetTime: new Date(minuteAgo.getTime() + 60 * 1000) };
      }

      // Check hour limit
      const hourAgo = new Date(now.getTime() - 60 * 60 * 1000);
      const hourUsage = usage.filter(u => u.timestamp > hourAgo).length;
      if (hourUsage >= apiKey.rateLimit.requestsPerHour) {
        return { allowed: false, resetTime: new Date(hourAgo.getTime() + 60 * 60 * 1000) };
      }

      // Check day limit
      const dayAgo = new Date(now.getTime() - 24 * 60 * 60 * 1000);
      const dayUsage = usage.filter(u => u.timestamp > dayAgo).length;
      if (dayUsage >= apiKey.rateLimit.requestsPerDay) {
        return { allowed: false, resetTime: new Date(dayAgo.getTime() + 24 * 60 * 60 * 1000) };
      }

      return { allowed: true };
    } catch (error) {
      console.error('Failed to check rate limit:', error);
      return { allowed: true }; // Allow on error to prevent service disruption
    }
  }

  /**
   * Record API key usage
   */
  async recordUsage(usage: ApiKeyUsage): Promise<void> {
    try {
      const db = rxdbService.getDatabase();
      await db.settings.upsert({
        id: `api_usage_${usage.keyId}_${Date.now()}`,
        type: 'settings',
        key: 'api_usage',
        value: JSON.stringify(usage),
        timestamp: Date.now()
      });

      // Increment usage count
      await this.incrementUsageCount(usage.keyId);
    } catch (error) {
      console.error('Failed to record API usage:', error);
    }
  }

  /**
   * Get all API keys
   */
  async getApiKeys(): Promise<ApiKey[]> {
    try {
      const db = rxdbService.getDatabase();
      const settings = await db.settings.findOne({ selector: { key: 'api_keys' } }).exec();
      
      if (settings) {
        return JSON.parse(settings.value);
      }
      return [];
    } catch (error) {
      console.error('Failed to load API keys:', error);
      return [];
    }
  }

  /**
   * Get API key usage history
   */
  async getApiKeyUsage(keyId: string): Promise<ApiKeyUsage[]> {
    try {
      const db = rxdbService.getDatabase();
      const usageRecords = await db.settings.find({
        selector: { key: 'api_usage' }
      }).exec();

      const usage: ApiKeyUsage[] = [];
      for (const record of usageRecords) {
        try {
          const usageData = JSON.parse(record.value) as ApiKeyUsage;
          if (usageData.keyId === keyId) {
            usage.push(usageData);
          }
        } catch {
          // Skip invalid records
        }
      }

      return usage.sort((a, b) => b.timestamp.getTime() - a.timestamp.getTime());
    } catch (error) {
      console.error('Failed to load API usage:', error);
      return [];
    }
  }

  /**
   * Deactivate an API key
   */
  async deactivateApiKey(keyId: string): Promise<void> {
    const apiKeys = await this.getApiKeys();
    const updatedKeys = apiKeys.map(key => 
      key.id === keyId ? { ...key, isActive: false } : key
    );
    await this.saveApiKeys(updatedKeys);
  }

  /**
   * Delete an API key
   */
  async deleteApiKey(keyId: string): Promise<void> {
    const apiKeys = await this.getApiKeys();
    const updatedKeys = apiKeys.filter(key => key.id !== keyId);
    await this.saveApiKeys(updatedKeys);
  }

  /**
   * Get available permissions
   */
  getAvailablePermissions(): string[] {
    return [...this.availablePermissions];
  }

  /**
   * Private helper methods
   */
  private generateApiKey(): string {
    const prefix = 'knirv_';
    const randomBytes = crypto.randomBytes(32).toString('hex');
    return prefix + randomBytes;
  }

  private generateId(): string {
    return crypto.randomBytes(16).toString('hex');
  }

  private validatePermissions(permissions: string[]): string[] {
    return permissions.filter(p => this.availablePermissions.includes(p));
  }

  private async saveApiKey(apiKey: ApiKey): Promise<void> {
    const apiKeys = await this.getApiKeys();
    apiKeys.push(apiKey);
    await this.saveApiKeys(apiKeys);
  }

  private async saveApiKeys(apiKeys: ApiKey[]): Promise<void> {
    try {
      const db = rxdbService.getDatabase();
      await db.settings.upsert({
        id: 'api_keys',
        type: 'settings',
        key: 'api_keys',
        value: JSON.stringify(apiKeys),
        timestamp: Date.now()
      });
    } catch (error) {
      console.error('Failed to save API keys:', error);
      throw error;
    }
  }

  private async updateLastUsed(keyId: string): Promise<void> {
    const apiKeys = await this.getApiKeys();
    const updatedKeys = apiKeys.map(key => 
      key.id === keyId ? { ...key, lastUsed: new Date() } : key
    );
    await this.saveApiKeys(updatedKeys);
  }

  private async incrementUsageCount(keyId: string): Promise<void> {
    const apiKeys = await this.getApiKeys();
    const updatedKeys = apiKeys.map(key => 
      key.id === keyId ? { ...key, usageCount: key.usageCount + 1 } : key
    );
    await this.saveApiKeys(updatedKeys);
  }
}

export const apiKeyService = new ApiKeyService();
