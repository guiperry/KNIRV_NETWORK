// Mock fetch for testing
global.fetch = jest.fn();

// Mock localStorage
const localStorageMock = {
  getItem: jest.fn(),
  setItem: jest.fn(),
  removeItem: jest.fn(),
  clear: jest.fn(),
};
Object.defineProperty(window, 'localStorage', {
  value: localStorageMock,
});

Object.defineProperty(process.env, 'NEXT_PUBLIC_API_BASE_URL', {
  value: 'http://localhost:8082',
  writable: true,
});

// Mock the API module to control API_BASE_URL and getApiBaseUrl explicitly
jest.mock('../api', () => {
  const actual = jest.requireActual('../api');
  return {
    ...actual, // This includes actual apiRequest, getAuthHeaders
    API_BASE_URL: 'http://localhost:8082', // Override
    getApiBaseUrl: jest.fn(() => 'http://localhost:8082'), // Override
  };
});

import { API_BASE_URL, getApiBaseUrl, getAuthHeaders, apiRequest } from '../api';

describe('API Module', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    localStorageMock.getItem.mockReturnValue(null);
  });

  afterEach(() => {
    // Restore original process.env.NEXT_PUBLIC_API_BASE_URL
    Object.defineProperty(process.env, 'NEXT_PUBLIC_API_BASE_URL', {
      value: undefined,
      writable: true,
    });
  });

  describe('getApiBaseUrl', () => {
    it('should return localhost URL by default in test environment', () => {
      // In test environment with jsdom, hostname is localhost by default
      const url = getApiBaseUrl();
      expect(url).toBe('http://localhost:8082');
    });

    it('should have correct API_BASE_URL constant', () => {
      // Test the exported constant
      expect(API_BASE_URL).toBe('http://localhost:8082');
    });
  });

  describe('getAuthHeaders', () => {
    it('should return basic headers without token', () => {
      localStorageMock.getItem.mockReturnValue(null);

      const headers = getAuthHeaders();
      expect(headers).toEqual({
        'Content-Type': 'application/json',
      });
    });

    it('should include Authorization header with token', () => {
      localStorageMock.getItem.mockReturnValue('test-token');

      const headers = getAuthHeaders();
      expect(headers).toEqual({
        'Content-Type': 'application/json',
        'Authorization': 'Bearer test-token',
      });
    });

    it('should return basic headers in browser environment', () => {
      // In browser environment (jsdom), should return content-type header
      const headers = getAuthHeaders();
      expect(headers).toHaveProperty('Content-Type', 'application/json');
    });
  });

  describe('apiRequest', () => {
    it('should make successful API request', async () => {
      const mockResponse = {
        ok: true,
        status: 200,
        json: jest.fn().mockResolvedValue({ data: 'test' }),
      };
      (global.fetch as jest.Mock).mockResolvedValue(mockResponse);

      const result = await apiRequest('/test');

      expect(fetch).toHaveBeenCalledWith('/test', {
        headers: {
          'Content-Type': 'application/json',
        },
      });
      expect(result).toEqual({ data: 'test' });
    });

    it('should handle API errors by throwing', async () => {
      const mockResponse = {
        ok: false,
        status: 400,
        statusText: 'Bad Request',
        json: jest.fn().mockResolvedValue({ message: 'Bad request' }),
      };
      (global.fetch as jest.Mock).mockResolvedValue(mockResponse);

      await expect(apiRequest('/test')).rejects.toThrow('Bad request');
    });

    it('should handle network errors by throwing', async () => {
      (global.fetch as jest.Mock).mockRejectedValue(new Error('Network error'));

      await expect(apiRequest('/test')).rejects.toThrow('Network error');
    });

    it('should include auth headers when token exists', async () => {
      localStorageMock.getItem.mockReturnValue('test-token');
      const mockResponse = {
        ok: true,
        status: 200,
        json: jest.fn().mockResolvedValue({ data: 'test' }),
      };
      (global.fetch as jest.Mock).mockResolvedValue(mockResponse);

      await apiRequest('/test');

      expect(fetch).toHaveBeenCalledWith('/test', {
        headers: {
          'Content-Type': 'application/json',
          'Authorization': 'Bearer test-token',
        },
      });
    });
  });
});
