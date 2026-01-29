/**
 * Test for KNIRVBASEService browser detection and initialization
 */

import { describe, it, expect, beforeEach, afterEach, jest } from '@jest/globals';

// Mock the browser environment
const originalWindow = global.window;
const originalFetch = global.fetch;

// Create a properly typed mock response helper
interface MockResponseData {
  [key: string]: unknown;
}

const createMockResponse = (data: MockResponseData, ok: boolean = true, status: number = 200) => ({
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
      global.window = {} as Window & typeof globalThis;
      global.fetch = jest.fn() as unknown as typeof fetch;

      const { KNIRVBASEService } = await import('@services/KNIRVBASEService');
      const service = new KNIRVBASEService();

      // Should use browser-compatible implementation
      expect(service).toBeDefined();
    });

    it('should detect Node.js environment correctly', async () => {
      // Mock Node.js environment
      (global.window as unknown) = undefined;
      (global.fetch as unknown) = undefined;

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
      } as unknown as Window & typeof globalThis;
      global.fetch = jest.fn() as unknown as typeof fetch;
    });

    it('should initialize browser database with correct options', async () => {
      const mockFetch = global.fetch as jest.Mock<typeof fetch>;
      
      // Mock successful initialization
      mockFetch.mockResolvedValueOnce(createMockResponse({ success: true }) as unknown as Response);

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
      const mockFetch = global.fetch as jest.Mock<typeof fetch>;
      
      // Mock initialization error
      mockFetch.mockResolvedValueOnce(createMockResponse(
        { error: 'Failed to initialize database' },
        false,
        500
      ) as unknown as Response);

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
      } as unknown as Window & typeof globalThis;
      global.fetch = jest.fn() as unknown as typeof fetch;
    });

    it('should create and use collections in browser mode', async () => {
      const mockFetch = global.fetch as jest.Mock<typeof fetch>;
      
      // Mock successful initialization
      mockFetch.mockResolvedValueOnce(createMockResponse({ success: true }) as unknown as Response);

      // Mock successful document insertion
      mockFetch.mockResolvedValueOnce(createMockResponse({
        success: true,
        document: { id: 'test-doc', name: 'Test Document' }
      }) as unknown as Response);

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