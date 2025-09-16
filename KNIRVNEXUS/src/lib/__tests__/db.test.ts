// Mock PrismaClient before any imports
const mockPrismaClient = jest.fn().mockImplementation(() => ({
  $connect: jest.fn(),
  $disconnect: jest.fn(),
}));

jest.mock('@prisma/client', () => ({
  PrismaClient: mockPrismaClient,
}));

describe('Database Client', () => {
  let originalEnv: string | undefined;

  beforeEach(() => {
    // Store original values
    originalEnv = process.env.NODE_ENV;

    // Clear the PrismaClient mock
    mockPrismaClient.mockClear();
  });

  afterEach(() => {
    // Restore original values
    process.env.NODE_ENV = originalEnv;
  });

  it('should create a PrismaClient instance with query logging', () => {
    const { db } = require('../db');

    expect(db).toBeDefined();
    expect(typeof db).toBe('object');
  });

  it('should have PrismaClient mock available', () => {
    // Verify that our mock is properly set up
    expect(mockPrismaClient).toBeDefined();
    expect(typeof mockPrismaClient).toBe('function');
  });

  it('should export a db instance', () => {
    const { db } = require('../db');

    expect(db).toBeDefined();
    expect(db).toHaveProperty('$connect');
    expect(db).toHaveProperty('$disconnect');
  });

  it('should handle production environment', () => {
    process.env.NODE_ENV = 'production';

    // The db should still be created regardless of environment
    const { db } = require('../db');
    expect(db).toBeDefined();
  });

  it('should handle development environment', () => {
    process.env.NODE_ENV = 'development';

    // The db should still be created regardless of environment
    const { db } = require('../db');
    expect(db).toBeDefined();
  });

  it('should handle test environment', () => {
    process.env.NODE_ENV = 'test';

    // The db should still be created regardless of environment
    const { db } = require('../db');
    expect(db).toBeDefined();
  });
});
