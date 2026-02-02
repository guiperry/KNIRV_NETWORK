/**
 * UDC Management Service
 * Handles Universal Delegation Certificates (UDC) creation, renewal, and validation
 */

export interface UDC {
  id: string;
  agentId: string;
  type: 'basic' | 'advanced' | 'enterprise' | 'custom';
  authorityLevel: 'read' | 'write' | 'execute' | 'admin' | 'full';
  status: 'pending' | 'active' | 'expired' | 'revoked' | 'suspended';
  issuedDate: Date;
  expiresDate: Date;
  renewalDate?: Date;
  scope: string;
  permissions: string[];
  metadata: UDCMetadata;
  signature: string;
  issuer: string;
  subject: string;
}

export interface UDCMetadata {
  version: string;
  description: string;
  tags: string[];
  constraints: UDCConstraints;
  usage: UDCUsage;
  security: UDCSecurityInfo;
}

export interface UDCConstraints {
  maxExecutions?: number;
  timeWindow?: number; // in milliseconds
  allowedHours?: number[]; // hours of day when UDC is valid
  allowedDays?: number[]; // days of week when UDC is valid
  ipWhitelist?: string[];
  geofencing?: {
    latitude: number;
    longitude: number;
    radius: number; // in meters
  };
}

export interface UDCUsage {
  executionCount: number;
  lastUsed?: Date;
  usageHistory: UDCUsageRecord[];
  quotaRemaining?: number;
}

export interface UDCUsageRecord {
  timestamp: Date;
  action: string;
  result: 'success' | 'failure' | 'denied';
  details?: string;
}

export interface UDCSecurityInfo {
  encryptionLevel: 'basic' | 'standard' | 'high' | 'quantum';
  requiresMFA: boolean;
  allowedDevices?: string[];
  securityFlags: string[];
}

export interface UDCRequest {
  agentId: string;
  type: UDC['type'];
  authorityLevel: UDC['authorityLevel'];
  validityPeriod: number; // in days
  scope: string;
  permissions: string[];
  constraints?: Partial<UDCConstraints>;
  metadata?: Partial<UDCMetadata>;
}

export interface UDCValidationResult {
  isValid: boolean;
  reason?: string;
  remainingTime?: number;
  usageQuota?: {
    used: number;
    remaining: number;
    total: number;
  };
  securityChecks: {
    signature: boolean;
    expiry: boolean;
    permissions: boolean;
    constraints: boolean;
  };
}

export interface UDCManagementConfig {
  baseUrl?: string;
  enableNetworking?: boolean;
  enableMonitoring?: boolean;
}

export class UDCManagementService {
  private udcs: Map<string, UDC> = new Map();
  private baseUrl: string;
  private signingKey: string;
  private isInitialized: boolean = false;
  private config: UDCManagementConfig;

  constructor(config: UDCManagementConfig = {}) {
    this.config = {
      baseUrl: 'http://localhost:3001',
      enableNetworking: process.env.NODE_ENV !== 'test',
      enableMonitoring: process.env.NODE_ENV !== 'test',
      ...config
    };
    this.baseUrl = this.config.baseUrl!;
    this.signingKey = this.generateSigningKey();
    this.initializeService();
  }

  private async initializeService(): Promise<void> {
    try {
      // Load existing UDCs from backend only if networking is enabled
      if (this.config.enableNetworking) {
        await this.loadUDCs();
      }

      // Start renewal monitoring only if monitoring is enabled
      if (this.config.enableMonitoring) {
        this.startRenewalMonitoring();
      }

      this.isInitialized = true;
      console.log('UDC Management Service initialized');
    } catch (error) {
      console.error('Failed to initialize UDC Management Service:', error);
      // Continue with empty state if backend is unavailable
      this.isInitialized = true;
    }
  }

  /**
   * Create a new UDC
   */
  async createUDC(request: UDCRequest): Promise<UDC> {
    try {
      const udc: UDC = {
        id: this.generateUDCId(),
        agentId: request.agentId,
        type: request.type,
        authorityLevel: request.authorityLevel,
        status: 'pending',
        issuedDate: new Date(),
        expiresDate: new Date(Date.now() + request.validityPeriod * 24 * 60 * 60 * 1000),
        scope: request.scope,
        permissions: request.permissions,
        metadata: {
          version: '1.0',
          description: request.metadata?.description || `UDC for agent ${request.agentId}`,
          tags: request.metadata?.tags || [],
          constraints: {
            maxExecutions: request.constraints?.maxExecutions || 1000,
            timeWindow: request.constraints?.timeWindow || 24 * 60 * 60 * 1000,
            allowedHours: request.constraints?.allowedHours || Array.from({length: 24}, (_, i) => i),
            allowedDays: request.constraints?.allowedDays || [0, 1, 2, 3, 4, 5, 6],
            ipWhitelist: request.constraints?.ipWhitelist || [],
            geofencing: request.constraints?.geofencing
          },
          usage: {
            executionCount: 0,
            usageHistory: []
          },
          security: {
            encryptionLevel: 'standard',
            requiresMFA: request.authorityLevel === 'admin' || request.authorityLevel === 'full',
            securityFlags: []
          }
        },
        signature: '',
        issuer: 'KNIRV-CONTROLLER',
        subject: request.agentId
      };

      // Generate signature
      udc.signature = await this.generateSignature(udc);

      // Store UDC
      this.udcs.set(udc.id, udc);

      // Send to backend for persistence
      await this.saveUDCToBackend(udc);

      // Activate UDC after creation
      udc.status = 'active';
      this.udcs.set(udc.id, udc);

      console.log(`UDC created successfully: ${udc.id}`);
      return udc;
    } catch (error) {
      console.error('Failed to create UDC:', error);
      throw new Error(`UDC creation failed: ${error instanceof Error ? error.message : 'Unknown error'}`);
    }
  }

  /**
   * Renew an existing UDC
   */
  async renewUDC(udcId: string, extensionDays: number): Promise<UDC> {
    const udc = this.udcs.get(udcId);
    if (!udc) {
      throw new Error(`UDC ${udcId} not found`);
    }

    if (udc.status === 'revoked') {
      throw new Error('Cannot renew revoked UDC');
    }

    try {
      // Extend expiration date
      const newExpiryDate = new Date(udc.expiresDate.getTime() + extensionDays * 24 * 60 * 60 * 1000);
      
      const renewedUDC: UDC = {
        ...udc,
        expiresDate: newExpiryDate,
        renewalDate: new Date(),
        status: 'active'
      };

      // Regenerate signature
      renewedUDC.signature = await this.generateSignature(renewedUDC);

      // Update stored UDC
      this.udcs.set(udcId, renewedUDC);

      // Send to backend
      await this.saveUDCToBackend(renewedUDC);

      console.log(`UDC renewed successfully: ${udcId}`);
      return renewedUDC;
    } catch (error) {
      console.error('Failed to renew UDC:', error);
      throw new Error(`UDC renewal failed: ${error instanceof Error ? error.message : 'Unknown error'}`);
    }
  }

  /**
   * Validate a UDC
   */
  async validateUDC(udcId: string, action?: string): Promise<UDCValidationResult> {
    const udc = this.udcs.get(udcId);
    if (!udc) {
      return {
        isValid: false,
        reason: 'UDC not found',
        securityChecks: {
          signature: false,
          expiry: false,
          permissions: false,
          constraints: false
        }
      };
    }

    const now = new Date();
    const securityChecks = {
      signature: await this.verifySignature(udc),
      expiry: udc.expiresDate > now,
      permissions: true, // Would check specific permissions for the action
      constraints: this.checkConstraints(udc, action)
    };

    const isValid = udc.status === 'active' && 
                   Object.values(securityChecks).every(check => check);

    const result: UDCValidationResult = {
      isValid,
      securityChecks,
      remainingTime: udc.expiresDate.getTime() - now.getTime()
    };

    if (!isValid) {
      if (udc.status !== 'active') result.reason = `UDC status: ${udc.status}`;
      else if (!securityChecks.expiry) result.reason = 'UDC expired';
      else if (!securityChecks.signature) result.reason = 'Invalid signature';
      else if (!securityChecks.constraints) result.reason = 'Constraint violation';
    }

    // Add usage quota information
    if (udc.metadata.constraints.maxExecutions) {
      result.usageQuota = {
        used: udc.metadata.usage.executionCount,
        total: udc.metadata.constraints.maxExecutions,
        remaining: udc.metadata.constraints.maxExecutions - udc.metadata.usage.executionCount
      };
    }

    return result;
  }

  /**
   * Revoke a UDC
   */
  async revokeUDC(udcId: string, reason: string): Promise<void> {
    const udc = this.udcs.get(udcId);
    if (!udc) {
      throw new Error(`UDC ${udcId} not found`);
    }

    udc.status = 'revoked';
    udc.metadata.security.securityFlags.push(`revoked:${reason}:${new Date().toISOString()}`);

    this.udcs.set(udcId, udc);
    await this.saveUDCToBackend(udc);

    console.log(`UDC revoked: ${udcId} - ${reason}`);
  }

  /**
   * Record UDC usage
   */
  async recordUsage(udcId: string, action: string, result: 'success' | 'failure' | 'denied', details?: string): Promise<void> {
    const udc = this.udcs.get(udcId);
    if (!udc) {
      return;
    }

    const usageRecord: UDCUsageRecord = {
      timestamp: new Date(),
      action,
      result,
      details
    };

    udc.metadata.usage.usageHistory.push(usageRecord);
    udc.metadata.usage.lastUsed = usageRecord.timestamp;
    
    if (result === 'success') {
      udc.metadata.usage.executionCount++;
    }

    // Check if quota exceeded
    if (udc.metadata.constraints.maxExecutions && 
        udc.metadata.usage.executionCount >= udc.metadata.constraints.maxExecutions) {
      udc.status = 'expired';
    }

    this.udcs.set(udcId, udc);
    await this.saveUDCToBackend(udc);
  }

  /**
   * Get all UDCs
   */
  getAllUDCs(): UDC[] {
    return Array.from(this.udcs.values());
  }

  /**
   * Get UDCs by agent ID
   */
  getUDCsByAgent(agentId: string): UDC[] {
    return Array.from(this.udcs.values()).filter(udc => udc.agentId === agentId);
  }

  /**
   * Get UDCs by status
   */
  getUDCsByStatus(status: UDC['status']): UDC[] {
    return Array.from(this.udcs.values()).filter(udc => udc.status === status);
  }

  /**
   * Get UDC by ID
   */
  getUDC(udcId: string): UDC | undefined {
    return this.udcs.get(udcId);
  }

  /**
   * Get UDCs expiring soon
   */
  getExpiringUDCs(daysAhead: number = 7): UDC[] {
    const threshold = new Date(Date.now() + daysAhead * 24 * 60 * 60 * 1000);
    return Array.from(this.udcs.values()).filter(udc => 
      udc.status === 'active' && udc.expiresDate <= threshold
    );
  }

  private async loadUDCs(): Promise<void> {
    try {
      const response = await fetch(`${this.baseUrl}/api/udc/list`);
      if (response.ok) {
        const udcs = await response.json();
        for (const udc of udcs) {
          this.udcs.set(udc.id, {
            ...udc,
            issuedDate: new Date(udc.issuedDate),
            expiresDate: new Date(udc.expiresDate),
            renewalDate: udc.renewalDate ? new Date(udc.renewalDate) : undefined
          });
        }
      }
    } catch (error) {
      console.error('Failed to load UDCs from backend:', error);
    }
  }

  private async saveUDCToBackend(udc: UDC): Promise<void> {
    // Only save to backend if networking is enabled
    if (!this.config.enableNetworking) {
      return;
    }

    try {
      await fetch(`${this.baseUrl}/api/udc/save`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(udc)
      });
    } catch (error) {
      console.error('Failed to save UDC to backend:', error);
      // Re-throw the error so it can be handled by the calling method
      throw error;
    }
  }

  private startRenewalMonitoring(): void {
    // Check for expiring UDCs every hour
    setInterval(() => {
      this.checkExpiringUDCs();
    }, 60 * 60 * 1000);
  }

  private checkExpiringUDCs(): void {
    const expiringUDCs = this.getExpiringUDCs(7); // 7 days ahead
    
    for (const udc of expiringUDCs) {
      console.log(`UDC ${udc.id} expires in ${Math.ceil((udc.expiresDate.getTime() - Date.now()) / (24 * 60 * 60 * 1000))} days`);
      
      // Auto-renew if configured
      if (udc.metadata.security.securityFlags.includes('auto-renew')) {
        this.renewUDC(udc.id, 30); // Extend by 30 days
      }
    }
  }

  private checkConstraints(udc: UDC, _action?: string): boolean {
    const now = new Date();
    const constraints = udc.metadata.constraints;

    // Check time window
    if (constraints.allowedHours && !constraints.allowedHours.includes(now.getHours())) {
      return false;
    }

    // Check allowed days
    if (constraints.allowedDays && !constraints.allowedDays.includes(now.getDay())) {
      return false;
    }

    // Check execution quota
    if (constraints.maxExecutions && udc.metadata.usage.executionCount >= constraints.maxExecutions) {
      return false;
    }

    // Additional constraint checks would go here
    return true;
  }

  private async generateSignature(udc: UDC): Promise<string> {
    // Simple signature generation - would use proper cryptographic signing in production
    const data = `${udc.id}:${udc.agentId}:${udc.expiresDate.toISOString()}:${udc.authorityLevel}`;
    const dataToHash = data + this.signingKey;

    try {
      // Use Node.js crypto directly for better compatibility
      const { createHash } = await import('crypto');
      const hash = createHash('sha256');
      hash.update(dataToHash, 'utf8');
      return hash.digest('hex');
    } catch (error) {
      console.error('Error generating signature:', error);
      // Fallback to a simple hash if crypto fails
      const { createHash: createHashFallback } = await import('crypto');
      const simpleHash = createHashFallback('md5').update(dataToHash, 'utf8').digest('hex');
      return simpleHash;
    }
  }

  private async verifySignature(udc: UDC): Promise<boolean> {
    const expectedSignature = await this.generateSignature(udc);
    return udc.signature === expectedSignature;
  }

  private generateSigningKey(): string {
    return `knirv_udc_key_${Date.now()}_${Math.random().toString(36).substr(2, 16)}`;
  }

  private generateUDCId(): string {
    return `udc_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
  }
}

// Export singleton instance
export const udcManagementService = new UDCManagementService();
