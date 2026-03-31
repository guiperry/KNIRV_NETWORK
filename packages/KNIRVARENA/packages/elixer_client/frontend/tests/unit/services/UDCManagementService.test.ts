/**
 * UDCManagementService Unit Tests
 * Comprehensive test suite for UDC management functionality
 */

import { udcManagementService, UDCManagementService, UDC } from '../../../src/services/UDCManagementService';

// Mock fetch globally
global.fetch = jest.fn();

// Mock crypto.subtle for signature generation
Object.defineProperty(global, 'crypto', {
  value: {
    subtle: {
      digest: jest.fn().mockResolvedValue(new ArrayBuffer(32))
    }
  }
});

describe('UDCManagementService', () => {
  let service: UDCManagementService;
  const mockFetch = global.fetch as jest.MockedFunction<typeof fetch>;

  beforeEach(() => {
    service = new UDCManagementService({ baseUrl: 'http://localhost:3001' });
    mockFetch.mockClear();
    jest.clearAllTimers();
    jest.useFakeTimers();
  });

  afterEach(() => {
    jest.useRealTimers();
    mockFetch.mockRestore();
  });

  describe('Initialization', () => {
    it('should initialize with default settings', () => {
      expect(service).toBeDefined();
      expect(service.createUDC).toBeDefined();
      expect(service.getAllUDCs).toBeDefined();
    });
  });

  describe('UDC Creation', () => {
    it('should create a new UDC successfully', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({ success: true })
      } as Response);

      const request = {
        agentId: 'agent-123',
        type: 'basic' as const,
        authorityLevel: 'read' as const,
        validityPeriod: 30,
        scope: 'Test scope',
        permissions: ['read', 'list'],
        constraints: {
          maxExecutions: 1000,
          allowedHours: [9, 10, 11, 12, 13, 14, 15, 16, 17]
        },
        metadata: {
          description: 'Test UDC',
          tags: ['test', 'basic']
        }
      };

      const udc = await service.createUDC(request);
      
      expect(udc).toHaveProperty('id');
      expect(udc).toHaveProperty('agentId', 'agent-123');
      expect(udc).toHaveProperty('type', 'basic');
      expect(udc).toHaveProperty('authorityLevel', 'read');
      expect(udc).toHaveProperty('status', 'active');
      expect(udc).toHaveProperty('issuedDate');
      expect(udc).toHaveProperty('expiresDate');
      expect(udc).toHaveProperty('signature');
      expect(udc).toHaveProperty('issuer', 'KNIRV-CONTROLLER');
      expect(udc).toHaveProperty('subject', 'agent-123');
      expect(udc.permissions).toEqual(['read', 'list']);
    });

    it('should set correct expiration date based on validity period', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({ success: true })
      } as Response);

      const request = {
        agentId: 'agent-123',
        type: 'basic' as const,
        authorityLevel: 'read' as const,
        validityPeriod: 7, // 7 days
        scope: 'Test scope',
        permissions: ['read']
      };

      const udc = await service.createUDC(request);
      
      const expectedExpiry = new Date(udc.issuedDate.getTime() + 7 * 24 * 60 * 60 * 1000);
      expect(udc.expiresDate.getTime()).toBeCloseTo(expectedExpiry.getTime(), -1000); // Within 1 second
    });

    it('should handle UDC creation failure', async () => {
      // Create a service instance with networking enabled for this test
      const networkService = new UDCManagementService({ enableNetworking: true });
      mockFetch.mockRejectedValueOnce(new Error('Network error'));

      const request = {
        agentId: 'agent-123',
        type: 'basic' as const,
        authorityLevel: 'read' as const,
        validityPeriod: 30,
        scope: 'Test scope',
        permissions: ['read']
      };

      await expect(networkService.createUDC(request)).rejects.toThrow('UDC creation failed');
    });
  });

  describe('UDC Renewal', () => {
    let testUDC: UDC;

    beforeEach(async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        json: async () => ({ success: true })
      } as Response);

      testUDC = await service.createUDC({
        agentId: 'agent-123',
        type: 'basic' as const,
        authorityLevel: 'read' as const,
        validityPeriod: 30,
        scope: 'Test scope',
        permissions: ['read']
      });
    });

    it('should renew UDC successfully', async () => {
      const originalExpiry = testUDC.expiresDate;
      const renewedUDC = await service.renewUDC(testUDC.id, 30);
      
      expect(renewedUDC.expiresDate.getTime()).toBeGreaterThan(originalExpiry.getTime());
      expect(renewedUDC.renewalDate).toBeDefined();
      expect(renewedUDC.status).toBe('active');
    });

    it('should not renew revoked UDC', async () => {
      await service.revokeUDC(testUDC.id, 'Test revocation');
      
      await expect(service.renewUDC(testUDC.id, 30))
        .rejects.toThrow('Cannot renew revoked UDC');
    });

    it('should handle renewal of non-existent UDC', async () => {
      await expect(service.renewUDC('non-existent-id', 30))
        .rejects.toThrow('UDC non-existent-id not found');
    });
  });

  describe('UDC Validation', () => {
    let testUDC: UDC;

    beforeEach(async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        json: async () => ({ success: true })
      } as Response);

      testUDC = await service.createUDC({
        agentId: 'agent-123',
        type: 'basic' as const,
        authorityLevel: 'read' as const,
        validityPeriod: 30,
        scope: 'Test scope',
        permissions: ['read']
      });
    });

    it('should validate active UDC successfully', async () => {
      const validation = await service.validateUDC(testUDC.id);
      
      expect(validation.isValid).toBe(true);
      expect(validation.securityChecks.signature).toBe(true);
      expect(validation.securityChecks.expiry).toBe(true);
      expect(validation.securityChecks.permissions).toBe(true);
      expect(validation.securityChecks.constraints).toBe(true);
      expect(validation.remainingTime).toBeGreaterThan(0);
    });

    it('should invalidate non-existent UDC', async () => {
      const validation = await service.validateUDC('non-existent-id');
      
      expect(validation.isValid).toBe(false);
      expect(validation.reason).toBe('UDC not found');
      expect(validation.securityChecks.signature).toBe(false);
      expect(validation.securityChecks.expiry).toBe(false);
      expect(validation.securityChecks.permissions).toBe(false);
      expect(validation.securityChecks.constraints).toBe(false);
    });

    it('should invalidate expired UDC', async () => {
      // Create an expired UDC by manipulating the expiry date
      testUDC.expiresDate = new Date(Date.now() - 24 * 60 * 60 * 1000); // 1 day ago
      
      const validation = await service.validateUDC(testUDC.id);
      
      expect(validation.isValid).toBe(false);
      expect(validation.reason).toBe('UDC expired');
    });

    it('should invalidate revoked UDC', async () => {
      await service.revokeUDC(testUDC.id, 'Test revocation');
      
      const validation = await service.validateUDC(testUDC.id);
      
      expect(validation.isValid).toBe(false);
      expect(validation.reason).toBe('UDC status: revoked');
    });

    it('should include usage quota information', async () => {
      const validation = await service.validateUDC(testUDC.id);
      
      expect(validation.usageQuota).toBeDefined();
      expect(validation.usageQuota!.used).toBe(0);
      expect(validation.usageQuota!.total).toBe(1000);
      expect(validation.usageQuota!.remaining).toBe(1000);
    });
  });

  describe('UDC Revocation', () => {
    let testUDC: UDC;

    beforeEach(async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        json: async () => ({ success: true })
      } as Response);

      testUDC = await service.createUDC({
        agentId: 'agent-123',
        type: 'basic' as const,
        authorityLevel: 'read' as const,
        validityPeriod: 30,
        scope: 'Test scope',
        permissions: ['read']
      });
    });

    it('should revoke UDC successfully', async () => {
      await service.revokeUDC(testUDC.id, 'Security breach');
      
      const udc = service.getUDC(testUDC.id);
      expect(udc!.status).toBe('revoked');
      expect(udc!.metadata.security.securityFlags).toContainEqual(
        expect.stringContaining('revoked:Security breach')
      );
    });

    it('should handle revocation of non-existent UDC', async () => {
      await expect(service.revokeUDC('non-existent-id', 'Test'))
        .rejects.toThrow('UDC non-existent-id not found');
    });
  });

  describe('Usage Recording', () => {
    let testUDC: UDC;

    beforeEach(async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        json: async () => ({ success: true })
      } as Response);

      testUDC = await service.createUDC({
        agentId: 'agent-123',
        type: 'basic' as const,
        authorityLevel: 'read' as const,
        validityPeriod: 30,
        scope: 'Test scope',
        permissions: ['read']
      });
    });

    it('should record successful usage', async () => {
      await service.recordUsage(testUDC.id, 'read_data', 'success', 'Data read successfully');
      
      const udc = service.getUDC(testUDC.id);
      expect(udc!.metadata.usage.executionCount).toBe(1);
      expect(udc!.metadata.usage.lastUsed).toBeDefined();
      expect(udc!.metadata.usage.usageHistory).toHaveLength(1);
      
      const usage = udc!.metadata.usage.usageHistory[0];
      expect(usage.action).toBe('read_data');
      expect(usage.result).toBe('success');
      expect(usage.details).toBe('Data read successfully');
    });

    it('should record failed usage without incrementing execution count', async () => {
      await service.recordUsage(testUDC.id, 'read_data', 'failure', 'Access denied');
      
      const udc = service.getUDC(testUDC.id);
      expect(udc!.metadata.usage.executionCount).toBe(0);
      expect(udc!.metadata.usage.usageHistory).toHaveLength(1);
      
      const usage = udc!.metadata.usage.usageHistory[0];
      expect(usage.result).toBe('failure');
    });

    it('should expire UDC when quota exceeded', async () => {
      // Set low quota for testing
      testUDC.metadata.constraints.maxExecutions = 2;
      
      await service.recordUsage(testUDC.id, 'action1', 'success');
      await service.recordUsage(testUDC.id, 'action2', 'success');
      
      const udc = service.getUDC(testUDC.id);
      expect(udc!.status).toBe('expired');
    });

    it('should handle usage recording for non-existent UDC gracefully', async () => {
      // Should not throw
      await expect(service.recordUsage('non-existent-id', 'action', 'success'))
        .resolves.not.toThrow();
    });
  });

  describe('UDC Retrieval', () => {
    beforeEach(async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        json: async () => ({ success: true })
      } as Response);

      // Create test UDCs
      await service.createUDC({
        agentId: 'agent-1',
        type: 'basic' as const,
        authorityLevel: 'read' as const,
        validityPeriod: 30,
        scope: 'Test scope 1',
        permissions: ['read']
      });

      await service.createUDC({
        agentId: 'agent-2',
        type: 'advanced' as const,
        authorityLevel: 'write' as const,
        validityPeriod: 60,
        scope: 'Test scope 2',
        permissions: ['read', 'write']
      });
    });

    it('should get all UDCs', () => {
      const udcs = service.getAllUDCs();
      expect(udcs).toHaveLength(2);
    });

    it('should get UDCs by agent ID', () => {
      const agent1UDCs = service.getUDCsByAgent('agent-1');
      const agent2UDCs = service.getUDCsByAgent('agent-2');
      
      expect(agent1UDCs).toHaveLength(1);
      expect(agent2UDCs).toHaveLength(1);
      expect(agent1UDCs[0].agentId).toBe('agent-1');
      expect(agent2UDCs[0].agentId).toBe('agent-2');
    });

    it('should get UDCs by status', () => {
      const activeUDCs = service.getUDCsByStatus('active');
      expect(activeUDCs).toHaveLength(2);
      
      const expiredUDCs = service.getUDCsByStatus('expired');
      expect(expiredUDCs).toHaveLength(0);
    });

    it('should get expiring UDCs', () => {
      const expiringUDCs = service.getExpiringUDCs(35); // 35 days ahead
      expect(expiringUDCs).toHaveLength(1); // Only the 30-day UDC
      
      const allExpiringUDCs = service.getExpiringUDCs(65); // 65 days ahead
      expect(allExpiringUDCs).toHaveLength(2); // Both UDCs
    });
  });

  describe('Signature Management', () => {
    it('should generate unique signatures for different UDCs', async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        json: async () => ({ success: true })
      } as Response);

      const udc1 = await service.createUDC({
        agentId: 'agent-1',
        type: 'basic' as const,
        authorityLevel: 'read' as const,
        validityPeriod: 30,
        scope: 'Test scope 1',
        permissions: ['read']
      });

      const udc2 = await service.createUDC({
        agentId: 'agent-2',
        type: 'basic' as const,
        authorityLevel: 'read' as const,
        validityPeriod: 30,
        scope: 'Test scope 2',
        permissions: ['read']
      });

      expect(udc1.signature).not.toBe(udc2.signature);
      expect(udc1.signature).toBeTruthy();
      expect(udc2.signature).toBeTruthy();
    });
  });

  describe('Singleton Instance', () => {
    it('should provide a singleton instance', () => {
      expect(udcManagementService).toBeDefined();
      expect(udcManagementService).toBeInstanceOf(UDCManagementService);
    });
  });

  describe('Error Handling', () => {
    it('should handle network errors gracefully', async () => {
      // Create a service instance with networking enabled for this test
      const networkService = new UDCManagementService({ enableNetworking: true });
      mockFetch.mockRejectedValue(new Error('Network error'));

      const request = {
        agentId: 'agent-123',
        type: 'basic' as const,
        authorityLevel: 'read' as const,
        validityPeriod: 30,
        scope: 'Test scope',
        permissions: ['read']
      };

      await expect(networkService.createUDC(request)).rejects.toThrow();
    });
  });

  describe('ID Generation', () => {
    it('should generate unique UDC IDs', async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        json: async () => ({ success: true })
      } as Response);

      const request = {
        agentId: 'agent-123',
        type: 'basic' as const,
        authorityLevel: 'read' as const,
        validityPeriod: 30,
        scope: 'Test scope',
        permissions: ['read']
      };

      const udc1 = await service.createUDC(request);
      const udc2 = await service.createUDC(request);
      
      expect(udc1.id).not.toBe(udc2.id);
    });
  });
});
