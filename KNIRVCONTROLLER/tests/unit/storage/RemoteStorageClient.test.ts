/**
 * Integration test for KNIRVBASE browser-backend communication
 */

import { describe, it, expect, beforeEach, afterEach, jest } from '@jest/globals';

// Mock fetch for browser environment
(global.fetch as unknown) = jest.fn() as unknown as typeof fetch;

import { RemoteStorageClient } from '@core/storage/RemoteStorageClient';

// Helper function to create mock responses
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

describe('RemoteStorageClient Integration', () => {
  let client: RemoteStorageClient;
  const sessionId = 'test-session-123';

  beforeEach(() => {
    client = new RemoteStorageClient({ 
      sessionId,
      baseUrl: 'http://localhost:3000' 
    });
    jest.clearAllMocks();
  });

  afterEach(() => {
    jest.resetAllMocks();
  });

  describe('Database Initialization', () => {
    it('should initialize database session', async () => {
      const mockResponse = {
        success: true,
        message: 'KNIRVBASE database initialized successfully',
        dataDir: '/User/Library/Application Support/KNIRV/KNIRVCONTROLLER/data'
      };

      (global.fetch as jest.Mock<typeof fetch>).mockResolvedValueOnce(createMockResponse(mockResponse) as unknown as Response);

      const result = await client.initialize();

      expect(global.fetch).toHaveBeenCalledWith(
        'http://localhost:3000/api/knirvbase/initialize',
        {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
          },
          body: JSON.stringify({
            sessionId,
            dataDir: undefined,
            distributedEnabled: false,
          }),
        }
      );

      expect(result).toEqual(mockResponse);
    });

    it('should handle initialization errors', async () => {
      (global.fetch as jest.Mock<typeof fetch>).mockResolvedValueOnce(
        createMockResponse({ error: 'Initialization failed' }, false, 500) as unknown as Response
      );

      await expect(client.initialize()).rejects.toThrow('Initialization failed');
    });
  });

  describe('Document Operations', () => {
    beforeEach(() => {
      // Mock successful initialization
      (global.fetch as jest.Mock<typeof fetch>).mockResolvedValueOnce(
        createMockResponse({ success: true }) as unknown as Response
      );
    });

    it('should insert a document', async () => {
      const mockDoc = { id: 'doc1', name: 'Test Document', data: 'test' };
      const mockResponse = { success: true, document: mockDoc };

      (global.fetch as jest.Mock<typeof fetch>).mockResolvedValueOnce(
        createMockResponse(mockResponse) as unknown as Response
      );

      const result = await client.insert('test-collection', mockDoc);

      expect(global.fetch).toHaveBeenCalledWith(
        'http://localhost:3000/api/knirvbase/insert',
        {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
          },
          body: JSON.stringify({
            sessionId,
            collection: 'test-collection',
            document: mockDoc,
          }),
        }
      );

      expect(result).toEqual(mockDoc);
    });

    it('should find a document by ID', async () => {
      const mockDoc = { id: 'doc1', name: 'Test Document', data: 'test' };
      const mockResponse = { success: true, document: mockDoc };

      (global.fetch as jest.Mock<typeof fetch>).mockResolvedValueOnce(
        createMockResponse(mockResponse) as unknown as Response
      );

      const result = await client.find('test-collection', 'doc1');

      expect(global.fetch).toHaveBeenCalledWith(
        'http://localhost:3000/api/knirvbase/test-session-123/test-collection/doc1'
      );

      expect(result).toEqual(mockDoc);
    });

    it('should return null for non-existent document', async () => {
      (global.fetch as jest.Mock<typeof fetch>).mockResolvedValueOnce(
        createMockResponse({ success: false, error: 'Document not found' }) as unknown as Response
      );

      const result = await client.find('test-collection', 'nonexistent');

      expect(result).toBeNull();
    });

    it('should find all documents in collection', async () => {
      const mockDocs = [
        { id: 'doc1', name: 'Document 1' },
        { id: 'doc2', name: 'Document 2' },
      ];
      const mockResponse = { success: true, documents: mockDocs };

      (global.fetch as jest.Mock<typeof fetch>).mockResolvedValueOnce(
        createMockResponse(mockResponse) as unknown as Response
      );

      const result = await client.findAll('test-collection');

      expect(global.fetch).toHaveBeenCalledWith(
        'http://localhost:3000/api/knirvbase/test-session-123/test-collection'
      );

      expect(result).toEqual(mockDocs);
    });

    it('should update a document', async () => {
      const updateData = { name: 'Updated Document' };
      const mockResponse = { success: true, updatedCount: 1 };

      (global.fetch as jest.Mock<typeof fetch>).mockResolvedValueOnce(
        createMockResponse(mockResponse) as unknown as Response
      );

      const result = await client.update('test-collection', 'doc1', updateData);

      expect(global.fetch).toHaveBeenCalledWith(
        'http://localhost:3000/api/knirvbase/test-session-123/test-collection/doc1',
        {
          method: 'PUT',
          headers: {
            'Content-Type': 'application/json',
          },
          body: JSON.stringify({
            update: updateData,
          }),
        }
      );

      expect(result).toBe(1);
    });

    it('should delete a document', async () => {
      const mockResponse = { success: true, deletedCount: 1 };

      (global.fetch as jest.Mock<typeof fetch>).mockResolvedValueOnce(
        createMockResponse(mockResponse) as unknown as Response
      );

      const result = await client.delete('test-collection', 'doc1');

      expect(global.fetch).toHaveBeenCalledWith(
        'http://localhost:3000/api/knirvbase/test-session-123/test-collection/doc1',
        {
          method: 'DELETE',
          headers: {
            'Content-Type': 'application/json',
          },
        }
      );

      expect(result).toBe(1);
    });
  });

  describe('Database Information', () => {
    it('should get database info', async () => {
      const mockResponse = {
        success: true,
        sessionId,
        dataDir: '/User/Library/Application Support/KNIRV/KNIRVCONTROLLER/data',
        initialized: true,
      };

      (global.fetch as jest.Mock<typeof fetch>).mockResolvedValueOnce(
        createMockResponse(mockResponse) as unknown as Response
      );

      const result = await client.getInfo();

      expect(global.fetch).toHaveBeenCalledWith(
        'http://localhost:3000/api/knirvbase/test-session-123/info'
      );

      expect(result).toEqual(mockResponse);
    });

    it('should get app data path', async () => {
      const mockResponse = {
        success: true,
        appDataDir: '/User/Library/Application Support/KNIRV/KNIRVCONTROLLER/data',
        platform: 'darwin',
        homedir: '/User/testuser',
      };

      (global.fetch as jest.Mock<typeof fetch>).mockResolvedValueOnce(
        createMockResponse(mockResponse) as unknown as Response
      );

      const result = await client.getAppDataPath();

      expect(global.fetch).toHaveBeenCalledWith(
        'http://localhost:3000/api/knirvbase/appdata'
      );

      expect(result).toEqual(mockResponse);
    });
  });

  describe('Error Handling', () => {
    it('should handle network errors', async () => {
      (global.fetch as jest.Mock<typeof fetch>).mockRejectedValueOnce(new Error('Network error') as unknown);

      await expect(client.find('test-collection', 'doc1')).rejects.toThrow('Network error');
    });

    it('should handle HTTP errors', async () => {
      (global.fetch as jest.Mock<typeof fetch>).mockResolvedValueOnce(
        createMockResponse({ error: 'Database error' }, false, 500) as unknown as Response
      );

      await expect(client.find('test-collection', 'doc1')).rejects.toThrow('Database error');
    });

    it('should handle malformed JSON responses', async () => {
      (global.fetch as jest.Mock<typeof fetch>).mockResolvedValueOnce({
        ok: true,
        json: async () => {
          throw new Error('Invalid JSON');
        },
      } as unknown as Response);

      await expect(client.find('test-collection', 'doc1')).rejects.toThrow('Invalid JSON');
    });
  });
});