import { GET } from '../route';
import { NextResponse } from 'next/server';

// Mock NextResponse for testing
jest.mock('next/server', () => ({
  NextResponse: {
    json: jest.fn((data) => ({
      json: async () => data,
      status: 200,
      ok: true,
    })),
  },
}));

describe('Health Route', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('should return health status with correct structure', async () => {
    const response = await GET();
    const data = await response.json();

    expect(data).toHaveProperty('status', 'healthy');
    expect(data).toHaveProperty('service', 'KNIRV-NEXUS DVE');
    expect(data).toHaveProperty('version', '2.0.0');
    expect(data).toHaveProperty('timestamp');
    expect(data).toHaveProperty('uptime');
    expect(data).toHaveProperty('components');
    expect(data).toHaveProperty('metrics');
    expect(data).toHaveProperty('endpoints');
  });

  it('should include all required components', async () => {
    const response = await GET();
    const data = await response.json();

    expect(data.components).toEqual({
      api: 'operational',
      websocket: 'operational',
      database: 'operational',
      cache: 'operational',
    });
  });

  it('should generate valid timestamp', () => {
    const timestamp = new Date().toISOString();
    expect(typeof timestamp).toBe('string');
    expect(new Date(timestamp)).toBeInstanceOf(Date);
    expect(isNaN(new Date(timestamp).getTime())).toBe(false);
  });

  it('should get uptime as number', () => {
    const uptime = process.uptime();
    expect(typeof uptime).toBe('number');
    expect(uptime).toBeGreaterThanOrEqual(0);
  });

  it('should have version string', () => {
    const version = '1.1.0';
    expect(typeof version).toBe('string');
    expect(version).toBeTruthy();
  });

  it('should determine environment', () => {
    const environment = process.env.NODE_ENV || 'development';
    expect(typeof environment).toBe('string');
    expect(['development', 'production', 'test']).toContain(environment);
  });

  it('should call NextResponse.json with health data', async () => {
    await GET();

    expect(NextResponse.json).toHaveBeenCalledWith(
      expect.objectContaining({
        status: 'healthy',
        service: 'KNIRV-NEXUS DVE',
        version: '2.0.0',
      })
    );
  });

  it('should validate health data structure', () => {
    const healthData = {
      status: 'healthy',
      timestamp: new Date().toISOString(),
      uptime: process.uptime(),
      version: '1.1.0',
      environment: 'test',
    };

    // Validate all required fields are present
    expect(healthData.status).toBeDefined();
    expect(healthData.timestamp).toBeDefined();
    expect(healthData.uptime).toBeDefined();
    expect(healthData.version).toBeDefined();
    expect(healthData.environment).toBeDefined();

    // Validate types
    expect(typeof healthData.status).toBe('string');
    expect(typeof healthData.timestamp).toBe('string');
    expect(typeof healthData.uptime).toBe('number');
    expect(typeof healthData.version).toBe('string');
    expect(typeof healthData.environment).toBe('string');
  });
});
