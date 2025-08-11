/**
 * Comprehensive tests for the KNIRV Gateway TypeScript SDK Client
 */

import { KNIRVGatewayClient } from '../client';
import { KNIRVAPIError, KNIRVValidationError } from '../types';

// Mock fetch globally
global.fetch = jest.fn();

describe('KNIRVGatewayClient', () => {
  let client: KNIRVGatewayClient;
  const mockFetch = global.fetch as jest.MockedFunction<typeof fetch>;

  beforeEach(() => {
    jest.clearAllMocks();
    client = new KNIRVGatewayClient({
      baseUrl: 'https://test.knirv.com',
      apiKey: 'test-key'
    });
  });

  describe('Constructor and Configuration', () => {
    it('should create client with default configuration', () => {
      const defaultClient = new KNIRVGatewayClient();
      expect(defaultClient).toBeInstanceOf(KNIRVGatewayClient);
    });

    it('should create client with custom configuration', () => {
      const customClient = new KNIRVGatewayClient({
        baseUrl: 'https://custom.knirv.com',
        apiKey: 'custom-key',
        timeout: 10000,
        retries: 5
      });
      
      expect(customClient).toBeInstanceOf(KNIRVGatewayClient);
    });

    it('should read configuration from environment variables', () => {
      process.env.KNIRV_API_KEY = 'env-key';
      process.env.KNIRV_BASE_URL = 'https://env.knirv.com';
      
      const envClient = new KNIRVGatewayClient();
      expect(envClient).toBeInstanceOf(KNIRVGatewayClient);
      
      delete process.env.KNIRV_API_KEY;
      delete process.env.KNIRV_BASE_URL;
    });

    it('should validate required configuration', () => {
      expect(() => {
        new KNIRVGatewayClient({ baseUrl: '' });
      }).toThrow();
    });
  });

  describe('HTTP Methods', () => {
    it('should make GET requests correctly', async () => {
      const mockResponse = { data: 'test' };
      mockFetch.mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => mockResponse,
        headers: new Headers({ 'content-type': 'application/json' })
      } as Response);

      const result = await client.get('/test');
      
      expect(mockFetch).toHaveBeenCalledWith(
        'https://test.knirv.com/test',
        expect.objectContaining({
          method: 'GET',
          headers: expect.objectContaining({
            'Authorization': 'Bearer test-key',
            'Content-Type': 'application/json',
            'User-Agent': expect.stringContaining('knirv-gateway-sdk-ts')
          })
        })
      );
      
      expect(result).toEqual(mockResponse);
    });

    it('should make POST requests correctly', async () => {
      const requestData = { name: 'test', value: 123 };
      const mockResponse = { id: 'created-id', created: true };
      
      mockFetch.mockResolvedValueOnce({
        ok: true,
        status: 201,
        json: async () => mockResponse,
        headers: new Headers({ 'content-type': 'application/json' })
      } as Response);

      const result = await client.post('/test', requestData);
      
      expect(mockFetch).toHaveBeenCalledWith(
        'https://test.knirv.com/test',
        expect.objectContaining({
          method: 'POST',
          body: JSON.stringify(requestData),
          headers: expect.objectContaining({
            'Authorization': 'Bearer test-key',
            'Content-Type': 'application/json'
          })
        })
      );
      
      expect(result).toEqual(mockResponse);
    });

    it('should make PUT requests correctly', async () => {
      const updateData = { name: 'updated' };
      const mockResponse = { updated: true };
      
      mockFetch.mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => mockResponse,
        headers: new Headers({ 'content-type': 'application/json' })
      } as Response);

      const result = await client.put('/test/123', updateData);
      
      expect(mockFetch).toHaveBeenCalledWith(
        'https://test.knirv.com/test/123',
        expect.objectContaining({
          method: 'PUT',
          body: JSON.stringify(updateData)
        })
      );
      
      expect(result).toEqual(mockResponse);
    });

    it('should make DELETE requests correctly', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        status: 204,
        json: async () => ({}),
        headers: new Headers()
      } as Response);

      await client.delete('/test/123');
      
      expect(mockFetch).toHaveBeenCalledWith(
        'https://test.knirv.com/test/123',
        expect.objectContaining({
          method: 'DELETE'
        })
      );
    });
  });

  describe('Request Headers', () => {
    it('should include authorization header', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => ({}),
        headers: new Headers()
      } as Response);

      await client.get('/test');
      
      const [, options] = mockFetch.mock.calls[0];
      expect(options?.headers).toEqual(expect.objectContaining({
        'Authorization': 'Bearer test-key'
      }));
    });

    it('should include user agent header', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => ({}),
        headers: new Headers()
      } as Response);

      await client.get('/test');
      
      const [, options] = mockFetch.mock.calls[0];
      expect(options?.headers).toEqual(expect.objectContaining({
        'User-Agent': expect.stringContaining('knirv-gateway-sdk-ts')
      }));
    });

    it('should allow custom headers', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => ({}),
        headers: new Headers()
      } as Response);

      await client.get('/test', {
        headers: {
          'X-Custom-Header': 'custom-value'
        }
      });
      
      const [, options] = mockFetch.mock.calls[0];
      expect(options?.headers).toEqual(expect.objectContaining({
        'X-Custom-Header': 'custom-value'
      }));
    });
  });

  describe('Query Parameters', () => {
    it('should handle query parameters correctly', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => ({}),
        headers: new Headers()
      } as Response);

      await client.get('/test', {
        params: {
          page: 1,
          limit: 10,
          filter: 'active'
        }
      });
      
      const [url] = mockFetch.mock.calls[0];
      expect(url).toBe('https://test.knirv.com/test?page=1&limit=10&filter=active');
    });

    it('should handle array query parameters', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => ({}),
        headers: new Headers()
      } as Response);

      await client.get('/test', {
        params: {
          tags: ['tag1', 'tag2', 'tag3']
        }
      });
      
      const [url] = mockFetch.mock.calls[0];
      expect(url).toBe('https://test.knirv.com/test?tags=tag1&tags=tag2&tags=tag3');
    });

    it('should handle undefined and null parameters', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => ({}),
        headers: new Headers()
      } as Response);

      await client.get('/test', {
        params: {
          defined: 'value',
          undefined: undefined,
          null: null,
          empty: ''
        }
      });
      
      const [url] = mockFetch.mock.calls[0];
      expect(url).toBe('https://test.knirv.com/test?defined=value&empty=');
    });
  });

  describe('Error Handling', () => {
    it('should handle 404 errors', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 404,
        statusText: 'Not Found',
        json: async () => ({ error: 'Resource not found', code: 'NOT_FOUND' }),
        headers: new Headers()
      } as Response);

      await expect(client.get('/nonexistent')).rejects.toThrow(KNIRVAPIError);
    });

    it('should handle 400 validation errors', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 400,
        statusText: 'Bad Request',
        json: async () => ({
          error: 'Validation failed',
          code: 'VALIDATION_ERROR',
          details: { name: ['This field is required'] }
        }),
        headers: new Headers()
      } as Response);

      await expect(client.post('/test', {})).rejects.toThrow(KNIRVValidationError);
    });

    it('should handle 500 server errors', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 500,
        statusText: 'Internal Server Error',
        json: async () => ({ error: 'Internal server error' }),
        headers: new Headers()
      } as Response);

      await expect(client.get('/test')).rejects.toThrow(KNIRVAPIError);
    });

    it('should handle network errors', async () => {
      mockFetch.mockRejectedValueOnce(new Error('Network error'));

      await expect(client.get('/test')).rejects.toThrow(KNIRVAPIError);
    });

    it('should handle timeout errors', async () => {
      const timeoutClient = new KNIRVGatewayClient({
        baseUrl: 'https://test.knirv.com',
        apiKey: 'test-key',
        timeout: 100
      });

      mockFetch.mockImplementationOnce(() => 
        new Promise(resolve => setTimeout(resolve, 200))
      );

      await expect(timeoutClient.get('/slow')).rejects.toThrow();
    });
  });

  describe('Retry Logic', () => {
    it('should retry on 5xx errors', async () => {
      const retryClient = new KNIRVGatewayClient({
        baseUrl: 'https://test.knirv.com',
        apiKey: 'test-key',
        retries: 2
      });

      // First two calls fail, third succeeds
      mockFetch
        .mockResolvedValueOnce({
          ok: false,
          status: 500,
          json: async () => ({ error: 'Server error' }),
          headers: new Headers()
        } as Response)
        .mockResolvedValueOnce({
          ok: false,
          status: 500,
          json: async () => ({ error: 'Server error' }),
          headers: new Headers()
        } as Response)
        .mockResolvedValueOnce({
          ok: true,
          status: 200,
          json: async () => ({ success: true }),
          headers: new Headers()
        } as Response);

      const result = await retryClient.get('/test');
      
      expect(mockFetch).toHaveBeenCalledTimes(3);
      expect(result).toEqual({ success: true });
    });

    it('should not retry on 4xx errors', async () => {
      const retryClient = new KNIRVGatewayClient({
        baseUrl: 'https://test.knirv.com',
        apiKey: 'test-key',
        retries: 2
      });

      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 400,
        json: async () => ({ error: 'Bad request' }),
        headers: new Headers()
      } as Response);

      await expect(retryClient.get('/test')).rejects.toThrow();
      expect(mockFetch).toHaveBeenCalledTimes(1);
    });
  });

  describe('Service Initialization', () => {
    it('should initialize all services', () => {
      expect(client.economics).toBeDefined();
      expect(client.gateway).toBeDefined();
      expect(client.health).toBeDefined();
      expect(client.integration).toBeDefined();
      expect(client.poaud).toBeDefined();
    });

    it('should pass client instance to services', () => {
      expect(client.economics).toHaveProperty('client');
      expect(client.gateway).toHaveProperty('client');
      expect(client.health).toHaveProperty('client');
      expect(client.integration).toHaveProperty('client');
      expect(client.poaud).toHaveProperty('client');
    });
  });

  describe('Request Interceptors', () => {
    it('should allow request interceptors', async () => {
      const interceptor = jest.fn((config) => {
        config.headers = { ...config.headers, 'X-Intercepted': 'true' };
        return config;
      });

      client.addRequestInterceptor(interceptor);

      mockFetch.mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => ({}),
        headers: new Headers()
      } as Response);

      await client.get('/test');

      expect(interceptor).toHaveBeenCalled();
      const [, options] = mockFetch.mock.calls[0];
      expect(options?.headers).toEqual(expect.objectContaining({
        'X-Intercepted': 'true'
      }));
    });

    it('should allow response interceptors', async () => {
      const interceptor = jest.fn((response) => {
        response.intercepted = true;
        return response;
      });

      client.addResponseInterceptor(interceptor);

      mockFetch.mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => ({ data: 'test' }),
        headers: new Headers()
      } as Response);

      const result = await client.get('/test');

      expect(interceptor).toHaveBeenCalled();
      expect(result).toEqual(expect.objectContaining({
        intercepted: true
      }));
    });
  });

  describe('Request Cancellation', () => {
    it('should support request cancellation', async () => {
      const controller = new AbortController();
      
      mockFetch.mockImplementationOnce(() => 
        new Promise((_, reject) => {
          setTimeout(() => reject(new Error('AbortError')), 100);
        })
      );

      const requestPromise = client.get('/test', {
        signal: controller.signal
      });

      controller.abort();

      await expect(requestPromise).rejects.toThrow();
    });
  });
});
