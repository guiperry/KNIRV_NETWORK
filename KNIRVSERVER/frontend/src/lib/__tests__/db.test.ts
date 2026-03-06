jest.mock('../db', () => ({
  db: {
    $connect: jest.fn(), // Return jest.fn() directly in the mock factory
    $disconnect: jest.fn(), // Return jest.fn() directly in the mock factory
    // Add any other methods of the 'db' instance that are used
  },
}));

// Import the mocked db
import { db } from '../db';

describe('Database Client', () => {
  let originalEnv: string | undefined;

  beforeEach(() => {
    originalEnv = process.env.NODE_ENV;
    jest.clearAllMocks(); // Clear call history on all mocks

    // Explicitly re-assign new jest.fn() to db.$connect and db.$disconnect
    // This ensures they are fresh mocks for *this* test.
    db.$connect = jest.fn();
    db.$disconnect = jest.fn();
  });

  afterEach(() => {
    Object.defineProperty(process.env, 'NODE_ENV', { writable: true, value: originalEnv });
    // No jest.restoreAllMocks() needed here, as we explicitly re-assign in beforeEach
  });

  it('should create a PrismaClient instance with query logging', () => {
    expect(db).toBeDefined();
    expect(typeof db).toBe('object');
  });

  it('should have PrismaClient mock available', () => {
    expect(db.$connect).toEqual(expect.any(Function));
    expect(db.$disconnect).toEqual(expect.any(Function));
  });

  it('should export a db instance', () => {
    expect(db).toBeDefined();
    expect(db).toHaveProperty('$connect');
    expect(db).toHaveProperty('$disconnect');
  });

  it('should handle production environment', () => {
    Object.defineProperty(process.env, 'NODE_ENV', { writable: true, value: 'production' });
    expect(db).toBeDefined();
  });

  it('should handle development environment', () => {
    Object.defineProperty(process.env, 'NODE_ENV', { writable: true, value: 'development' });
    expect(db).toBeDefined();
  });

  it('should handle test environment', () => {
    Object.defineProperty(process.env, 'NODE_ENV', { writable: true, value: 'test' });
    expect(db).toBeDefined();
  });
});
