// Test Helper Functions for KNIRVWALLET Testing
import { TEST_MNEMONICS, TEST_PRIVATE_KEYS, TEST_ADDRESSES } from './test-data';

// Validation helpers
export function isValidMnemonic(mnemonic: string): boolean {
  const words = mnemonic.trim().split(/\s+/);
  return words.length === 12 || words.length === 24;
}

export function isValidPrivateKey(privateKey: string): boolean {
  const cleanKey = privateKey.replace(/^0x/, '');
  return /^[0-9a-fA-F]{64}$/.test(cleanKey);
}

export function isValidAddress(address: string, prefix?: string): boolean {
  if (prefix) {
    return address.startsWith(prefix) && address.length > prefix.length;
  }
  return address.length > 10; // Basic length check
}

// Test data generators
export function generateTestWalletName(): string {
  return `test-wallet-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
}

export function generateTestSessionId(): string {
  return `session-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
}

export function generateTestTransactionHash(): string {
  return '0x' + Array.from({ length: 64 }, () => 
    Math.floor(Math.random() * 16).toString(16)
  ).join('');
}

// Async test helpers
export async function waitFor(condition: () => boolean, timeout: number = 5000): Promise<void> {
  const startTime = Date.now();
  
  while (!condition() && (Date.now() - startTime) < timeout) {
    await new Promise(resolve => setTimeout(resolve, 100));
  }
  
  if (!condition()) {
    throw new Error(`Condition not met within ${timeout}ms`);
  }
}

export async function waitForAsync<T>(
  asyncCondition: () => Promise<T>, 
  predicate: (result: T) => boolean,
  timeout: number = 5000
): Promise<T> {
  const startTime = Date.now();
  
  while ((Date.now() - startTime) < timeout) {
    try {
      const result = await asyncCondition();
      if (predicate(result)) {
        return result;
      }
    } catch (error) {
      // Log error for debugging but continue trying
      console.debug('Async condition check failed, retrying:', error);
    }
    await new Promise(resolve => setTimeout(resolve, 100));
  }
  
  throw new Error(`Async condition not met within ${timeout}ms`);
}

// Service availability helpers
export async function waitForService(url: string, timeout: number = 30000): Promise<boolean> {
  const startTime = Date.now();
  
  while ((Date.now() - startTime) < timeout) {
    try {
      const response = await fetch(url);
      if (response.ok) {
        return true;
      }
    } catch (error) {
      // Service not ready yet, log for debugging
      console.debug('Service health check failed, retrying:', error);
    }
    await new Promise(resolve => setTimeout(resolve, 2000));
  }
  
  return false;
}

export async function waitForServices(urls: string[], timeout: number = 30000): Promise<boolean> {
  const promises = urls.map(url => waitForService(url, timeout));
  const results = await Promise.all(promises);
  return results.every(result => result);
}

// Test environment setup helpers
interface TestEnvironmentConfig {
  mockServices?: boolean;
  skipNetworkCalls?: boolean;
  logLevel?: string;
  timeout?: number;
}

export function setupTestEnvironment(config: TestEnvironmentConfig = {}) {
  // Set test environment variables
  process.env.NODE_ENV = 'test';
  process.env.MOCK_SERVICES = config.mockServices ? 'true' : 'false';
  process.env.SKIP_NETWORK_CALLS = config.skipNetworkCalls ? 'true' : 'false';
  
  // Mock console methods if needed
  if (config.suppressLogs) {
    jest.spyOn(console, 'log').mockImplementation(() => {});
    jest.spyOn(console, 'warn').mockImplementation(() => {});
    jest.spyOn(console, 'error').mockImplementation(() => {});
  }
  
  // Setup global test timeout
  if (config.timeout) {
    jest.setTimeout(config.timeout);
  }
}

export function cleanupTestEnvironment() {
  // Restore console methods
  if (jest.isMockFunction(console.log)) {
    (console.log as jest.MockedFunction<typeof console.log>).mockRestore();
  }
  if (jest.isMockFunction(console.warn)) {
    (console.warn as jest.MockedFunction<typeof console.warn>).mockRestore();
  }
  if (jest.isMockFunction(console.error)) {
    (console.error as jest.MockedFunction<typeof console.error>).mockRestore();
  }
  
  // Clear all mocks
  jest.clearAllMocks();
}

// Wallet test helpers
export function createTestWalletConfig(type: string = 'HD') {
  const baseConfig = {
    name: generateTestWalletName(),
    type
  };
  
  switch (type) {
    case 'HD':
      return {
        ...baseConfig,
        mnemonic: TEST_MNEMONICS.VALID_12_WORD
      };
    case 'PRIVATE_KEY':
      return {
        ...baseConfig,
        privateKey: TEST_PRIVATE_KEYS.VALID_HEX
      };
    case 'WEB3_AUTH':
      return {
        ...baseConfig,
        privateKey: TEST_PRIVATE_KEYS.VALID_HEX
      };
    case 'ADDRESS':
      return {
        ...baseConfig,
        address: TEST_ADDRESSES.GNOLANG
      };
    default:
      return baseConfig;
  }
}

// Transaction test helpers
interface TransactionOverrides {
  from?: string;
  to?: string;
  amount?: string;
  token?: string;
  memo?: string;
  gasLimit?: string;
  chainId?: string;
}

export function createTestTransaction(overrides: TransactionOverrides = {}) {
  return {
    from: TEST_ADDRESSES.GNOLANG,
    to: 'g1234567890abcdef1234567890abcdef12345678',
    amount: '1000000',
    token: 'unrn',
    memo: 'Test transaction',
    ...overrides
  };
}

// Error handling helpers
export function expectAsyncError(asyncFn: () => Promise<unknown>, expectedError?: string | RegExp) {
  return expect(asyncFn()).rejects.toThrow(expectedError);
}

export function expectAsyncSuccess<T>(asyncFn: () => Promise<T>) {
  return expect(asyncFn()).resolves;
}

// Mock response helpers
interface MockResponse {
  success: boolean;
  data?: unknown;
  error?: unknown;
  timestamp: number;
}

export function createMockResponse(data: unknown, success: boolean = true): MockResponse {
  return {
    success,
    data: success ? data : undefined,
    error: success ? undefined : data,
    timestamp: Date.now()
  };
}

export function createMockTransactionResponse(txHash?: string) {
  return createMockResponse({
    txHash: txHash || generateTestTransactionHash(),
    blockHeight: Math.floor(Math.random() * 1000000),
    gasUsed: '21000'
  });
}

// Network simulation helpers
export function simulateNetworkDelay(ms: number = 100): Promise<void> {
  return new Promise(resolve => setTimeout(resolve, ms));
}

export function simulateNetworkError(message: string = 'Network error'): never {
  throw new Error(message);
}

// Crypto test helpers
export function generateRandomBytes(length: number): Uint8Array {
  const bytes = new Uint8Array(length);
  for (let i = 0; i < length; i++) {
    bytes[i] = Math.floor(Math.random() * 256);
  }
  return bytes;
}

export function bytesToHex(bytes: Uint8Array): string {
  return Array.from(bytes, byte => byte.toString(16).padStart(2, '0')).join('');
}

export function hexToBytes(hex: string): Uint8Array {
  const cleanHex = hex.replace(/^0x/, '');
  const bytes = new Uint8Array(cleanHex.length / 2);
  for (let i = 0; i < bytes.length; i++) {
    bytes[i] = parseInt(cleanHex.substr(i * 2, 2), 16);
  }
  return bytes;
}

// Test assertion helpers
interface WalletStructure {
  accounts: unknown[];
  keyrings: unknown[];
  currentAccountId: string;
}

export function assertWalletStructure(wallet: WalletStructure) {
  expect(wallet).toHaveProperty('accounts');
  expect(wallet).toHaveProperty('keyrings');
  expect(wallet).toHaveProperty('currentAccountId');
  expect(Array.isArray(wallet.accounts)).toBe(true);
  expect(Array.isArray(wallet.keyrings)).toBe(true);
}

interface TransactionStructure {
  from: string;
  to: string;
  amount: string;
}

export function assertTransactionStructure(transaction: TransactionStructure) {
  expect(transaction).toHaveProperty('from');
  expect(transaction).toHaveProperty('to');
  expect(transaction).toHaveProperty('amount');
  expect(typeof transaction.from).toBe('string');
  expect(typeof transaction.to).toBe('string');
  expect(typeof transaction.amount).toBe('string');
}

interface ApiResponseStructure {
  success: boolean;
  data?: unknown;
  error?: unknown;
}

export function assertApiResponseStructure(response: ApiResponseStructure) {
  expect(response).toHaveProperty('success');
  expect(typeof response.success).toBe('boolean');
  
  if (response.success) {
    expect(response).toHaveProperty('data');
  } else {
    expect(response).toHaveProperty('error');
  }
}

// Performance testing helpers
export async function measureExecutionTime<T>(fn: () => Promise<T>): Promise<{ result: T; duration: number }> {
  const startTime = performance.now();
  const result = await fn();
  const endTime = performance.now();
  
  return {
    result,
    duration: endTime - startTime
  };
}

export function createPerformanceBenchmark(name: string, iterations: number = 100) {
  const times: number[] = [];
  
  return {
    async run<T>(fn: () => Promise<T>): Promise<{ average: number; min: number; max: number; results: T[] }> {
      const results: T[] = [];
      
      for (let i = 0; i < iterations; i++) {
        const { result, duration } = await measureExecutionTime(fn);
        times.push(duration);
        results.push(result);
      }
      
      return {
        average: times.reduce((a, b) => a + b, 0) / times.length,
        min: Math.min(...times),
        max: Math.max(...times),
        results
      };
    }
  };
}
