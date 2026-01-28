/**
 * Test for KNIRVBASEService browser detection and initialization
 */

import { describe, it, expect, beforeEach, afterEach, jest } from '@jest/globals';

// Mock the browser environment
const originalWindow = global.window;
const originalFetch = global.fetch;

// Create a properly typed mock response helper
const createMockResponse = (data: any, ok: boolean = true, status: number = 200) => ({
  ok,
  status,
  json: async () => data,
  text: async () => JSON.stringify(data),
  blob: async () => new Blob([JSON.stringify(data)]),
});

describe('KNIRVBASEService Browser Integration', () => {
  beforeEach(() => {
    jest.resetModules();
  });

  afterEach(() => {
    global.window = originalWindow;
    global.fetch = originalFetch;
  });

  describe('Browser Environment Detection', () => {
    it('should detect browser environment correctly', async () => {
      // Mock browser environment
      global.window = {} as any;
      (global.fetch as jest.Mock) = jest.fn();

      const { KNIRVBASEService } = await import('@services/KNIRVBASEService');
      const service = new KNIRVBASEService();

      // Should use browser-compatible implementation
      expect(service).toBeDefined();
    });

    it('should detect Node.js environment correctly', async () => {
      // Mock Node.js environment
      delete (global as any).window;
      delete (global as any).fetch;

      const { KNIRVBASEService } = await import('@services/KNIRVBASEService');
      const service = new KNIRVBASEService();

      // Should use Node.js filesystem implementation
      expect(service).toBeDefined();
    });
  });

  describe('Browser Initialization', () => {
    beforeEach(() => {
      // Mock browser environment
      global.window = {
        location: {
          origin: 'http://localhost:3000'
        }
      } as any;
      (global.fetch as jest.Mock) = jest.fn();
    });

    it('should initialize browser database with correct options', async () => {
      const mockFetch = global.fetch as jest.Mock;
      
      // Mock successful initialization - use type assertion to bypass TypeScript checking
      (mockFetch.mockResolvedValueOnce as any)(createMockResponse({ success: true }));

      const { KNIRVBASEService } = await import('@services/KNIRVBASEService');
      const service = new KNIRVBASEService();

      await service.initialize({
        dataDir: './test-data',
        distributedEnabled: false
      });

      // Verify fetch was called with correct initialization parameters
      expect(mockFetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/knirvbase/initialize'),
        expect.objectContaining({
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
          },
          body: expect.stringContaining('knirvcontroller_'),
        })
      );
    });

    it('should handle initialization errors gracefully', async () => {
      const mockFetch = global.fetch as jest.Mock;
      
      // Mock initialization error - use type assertion to bypass TypeScript checking
      (mockFetch.mockResolvedValueOnce as any)(createMockResponse(
        { error: 'Failed to initialize database' },
        false,
        500
      ));

      const { KNIRVBASEService } = await import('@services/KNIRVBASEService');
      const service = new KNIRVBASEService();

      await expect(service.initialize()).rejects.toThrow('Failed to initialize database');
    });
  });

  describe('Collection Operations', () => {
    beforeEach(() => {
      // Mock browser environment
      global.window = {
        location: {
          origin: 'http://localhost:3000'
        }
      } as any;
      (global.fetch as jest.Mock) = jest.fn();
    });

    it('should create and use collections in browser mode', async () => {
      const mockFetch = global.fetch as jest.Mock;
      
      // Mock successful initialization - use type assertion to bypass TypeScript checking
      (mockFetch.mockResolvedValueOnce as any)(createMockResponse({ success: true }));

      // Mock successful document insertion - use type assertion to bypass TypeScript checking
      (mockFetch.mockResolvedValueOnce as any)(createMockResponse({
        success: true,
        document: { id: 'test-doc', name: 'Test Document' }
      }));

      const { KNIRVBASEService } = await import('@services/KNIRVBASEService');
      const service = new KNIRVBASEService();

      await service.initialize();
      
      const collection = service.getCollection('test-collection');
      expect(collection).toBeDefined();

      const result = await collection.insert({ name: 'Test Document' });
      expect(result).toEqual({ id: 'test-doc', name: 'Test Document' });
    });
  });
});