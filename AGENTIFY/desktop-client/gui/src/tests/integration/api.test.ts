import { testADKIntegration } from '../../utils/testAPI';

// Mock fetch for Node.js environment
global.fetch = jest.fn();

describe('ADK API Integration', () => {
  beforeEach(() => {
    // Reset fetch mock before each test
    (fetch as jest.MockedFunction<typeof fetch>).mockClear();
  });

  it('should successfully test all ADK endpoints', async () => {
    // Mock successful API responses
    (fetch as jest.MockedFunction<typeof fetch>)
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({ status: 'healthy', version: '1.0.0' })
      } as Response);

    const result = await testADKIntegration();

    // Verify the result
    expect(result.success).toBe(true);
    expect(result.message).toBe('ADK integration test passed');
    expect(result.data).toEqual({ status: 'healthy', version: '1.0.0' });

    // Verify fetch was called with correct endpoint
    expect(fetch).toHaveBeenCalledWith('/api/v1/health');
  }, 10000); // 10s timeout for API calls
});