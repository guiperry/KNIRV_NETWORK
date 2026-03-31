// Mock Services for KNIRVWALLET Testing
import { TEST_API_RESPONSES, TEST_ADDRESSES, TEST_TRANSACTIONS } from './test-data';

// Mock HTTP Client for API testing
interface RequestHistoryEntry {
  method: string;
  url: string;
  data?: unknown;
}

export class MockHttpClient {
  private responses: Map<string, unknown> = new Map();
  private requestHistory: RequestHistoryEntry[] = [];

  setMockResponse(endpoint: string, response: unknown) {
    this.responses.set(endpoint, response);
  }

  async request(method: string, url: string, data?: unknown): Promise<unknown> {
    this.requestHistory.push({ method, url, data });
    
    const response = this.responses.get(url);
    if (response) {
      return response;
    }

    // Default responses for common endpoints
    if (url.includes('/wallet/create')) {
      return TEST_API_RESPONSES.WALLET_CREATION_SUCCESS;
    }
    if (url.includes('/balance/')) {
      return TEST_API_RESPONSES.BALANCE_RESPONSE;
    }
    if (url.includes('/transaction/send')) {
      return TEST_API_RESPONSES.TRANSACTION_SUCCESS;
    }

    throw new Error(`No mock response configured for ${method} ${url}`);
  }

  getRequestHistory() {
    return this.requestHistory;
  }

  clearHistory() {
    this.requestHistory = [];
  }
}

// Mock XION Client
export class MockXionClient {
  private balances: Map<string, string> = new Map();
  private transactions: Array<Record<string, unknown>> = [];

  constructor() {
    // Set default balances
    this.balances.set(TEST_ADDRESSES.XION, '1000000');
  }

  async getBalance(address: string): Promise<string> {
    return this.balances.get(address) || '0';
  }

  async sendTransaction(tx: Record<string, unknown>): Promise<{ txHash: string; blockHeight: number }> {
    const txHash = `0x${Math.random().toString(16).substr(2, 64)}`;
    const blockHeight = Math.floor(Math.random() * 1000000);
    
    this.transactions.push({ ...tx, txHash, blockHeight });
    
    return { txHash, blockHeight };
  }

  async burnNRNForSkill(skillId: string, amount: string): Promise<{ txHash: string }> {
    const txHash = `0x${Math.random().toString(16).substr(2, 64)}`;
    
    this.transactions.push({
      type: 'burn_nrn_for_skill',
      skillId,
      amount,
      txHash
    });
    
    return { txHash };
  }

  getTransactionHistory() {
    return this.transactions;
  }

  setBalance(address: string, balance: string) {
    this.balances.set(address, balance);
  }
}

// Mock WebSocket for cross-platform sync testing
type WebSocketCallback = (...args: unknown[]) => void;

export class MockWebSocket {
  private listeners: Map<string, WebSocketCallback[]> = new Map();
  private isConnected = false;
  private messageHistory: unknown[] = [];
  private connectionUrl: string;

  constructor(url: string) {
    this.connectionUrl = url;

    // Validate URL format
    if (!url || !url.startsWith('ws://') && !url.startsWith('wss://')) {
      throw new Error(`Invalid WebSocket URL: ${url}`);
    }

    // Simulate connection after a short delay
    setTimeout(() => {
      this.isConnected = true;
      this.emit('open', { url: this.connectionUrl });
    }, 100);
  }

  on(event: string, callback: WebSocketCallback) {
    if (!this.listeners.has(event)) {
      this.listeners.set(event, []);
    }
    this.listeners.get(event)!.push(callback);
  }

  emit(event: string, data: unknown) {
    const callbacks = this.listeners.get(event) || [];
    callbacks.forEach(callback => callback(data));
  }

  send(data: unknown) {
    if (!this.isConnected) {
      throw new Error('WebSocket not connected');
    }
    
    this.messageHistory.push(data);
    
    // Simulate server response
    setTimeout(() => {
      this.emit('message', {
        type: 'ACK',
        originalMessage: data,
        timestamp: Date.now()
      });
    }, 50);
  }

  close() {
    this.isConnected = false;
    this.emit('close', {});
  }

  getMessageHistory() {
    return this.messageHistory;
  }

  isConnectedState() {
    return this.isConnected;
  }
}

// Mock Ledger Transport for hardware wallet testing
export class MockLedgerTransport {
  private isConnected = false;
  private deviceInfo = {
    name: 'Mock Ledger Device',
    version: '1.0.0',
    model: 'Nano S'
  };

  async open(): Promise<void> {
    this.isConnected = true;
  }

  async close(): Promise<void> {
    this.isConnected = false;
  }

  async exchange(apdu: Buffer): Promise<Buffer> {
    if (!this.isConnected) {
      throw new Error('Device not connected');
    }

    // Mock responses for common APDU commands
    const command = apdu.toString('hex');
    
    if (command.startsWith('e016')) { // Get app info
      return Buffer.from('000000050107426974636f696e034254439000', 'hex');
    }
    
    if (command.startsWith('e002')) { // Get public key
      return Buffer.from('4104' + '0'.repeat(128) + '9000', 'hex');
    }
    
    if (command.startsWith('e004')) { // Sign transaction
      return Buffer.from('30440220' + '0'.repeat(64) + '0220' + '0'.repeat(64) + '9000', 'hex');
    }

    throw new Error(`Unknown APDU command: ${command}`);
  }

  getDeviceInfo() {
    return this.deviceInfo;
  }
}

// Mock Crypto Provider for testing cryptographic operations
export class MockCryptoProvider {
  async generateMnemonic(wordCount: number = 12): Promise<string> {
    const words = [
      'abandon', 'ability', 'able', 'about', 'above', 'absent', 'absorb', 'abstract',
      'absurd', 'abuse', 'access', 'accident', 'account', 'accuse', 'achieve', 'acid'
    ];
    
    return Array.from({ length: wordCount }, () => 
      words[Math.floor(Math.random() * words.length)]
    ).join(' ');
  }

  async generatePrivateKey(): Promise<string> {
    const bytes = new Uint8Array(32);
    for (let i = 0; i < 32; i++) {
      bytes[i] = Math.floor(Math.random() * 256);
    }
    return Array.from(bytes, byte => byte.toString(16).padStart(2, '0')).join('');
  }

  async deriveAddress(publicKey: string, prefix: string = 'g'): Promise<string> {
    // Simple mock address derivation
    const hash = this.simpleHash(publicKey);
    return `${prefix}${hash.substring(0, 38)}`;
  }

  async signTransaction(privateKey: string, transaction: Record<string, unknown>): Promise<string> {
    const txData = JSON.stringify(transaction);
    const signature = this.simpleHash(privateKey + txData);
    return signature;
  }

  async encryptData(data: string, password: string): Promise<string> {
    // Simple mock encryption (not secure, for testing only)
    const encrypted = Buffer.from(data).toString('base64') + '.' + this.simpleHash(password);
    return encrypted;
  }

  async decryptData(encryptedData: string, password: string): Promise<string> {
    // Simple mock decryption
    const [data, hash] = encryptedData.split('.');
    if (hash !== this.simpleHash(password)) {
      throw new Error('Invalid password');
    }
    return Buffer.from(data, 'base64').toString();
  }

  private simpleHash(input: string): string {
    let hash = 0;
    for (let i = 0; i < input.length; i++) {
      const char = input.charCodeAt(i);
      hash = ((hash << 5) - hash) + char;
      hash = hash & hash; // Convert to 32-bit integer
    }
    return Math.abs(hash).toString(16).padStart(8, '0');
  }
}

// Mock Storage Provider for testing wallet persistence
export class MockStorageProvider {
  private storage: Map<string, string> = new Map();

  async setItem(key: string, value: string): Promise<void> {
    this.storage.set(key, value);
  }

  async getItem(key: string): Promise<string | null> {
    return this.storage.get(key) || null;
  }

  async removeItem(key: string): Promise<void> {
    this.storage.delete(key);
  }

  async clear(): Promise<void> {
    this.storage.clear();
  }

  async getAllKeys(): Promise<string[]> {
    return Array.from(this.storage.keys());
  }

  getStorageState() {
    return Object.fromEntries(this.storage);
  }

  // Method to populate storage with test transactions
  loadTestTransactions() {
    this.storage.set('test_transactions', JSON.stringify(TEST_TRANSACTIONS));
    return TEST_TRANSACTIONS;
  }
}

// Mock Network Provider for testing network operations
export class MockNetworkProvider {
  private networkStatus = 'online';
  private latency = 100; // ms

  async checkConnection(): Promise<boolean> {
    await this.simulateLatency();
    return this.networkStatus === 'online';
  }

  async makeRequest(url: string, options?: Record<string, unknown>): Promise<unknown> {
    await this.simulateLatency();

    // Log request details for debugging
    if (options) {
      console.debug('Mock network request:', { url, method: options.method, headers: options.headers });
    }

    if (this.networkStatus === 'offline') {
      throw new Error('Network offline');
    }

    // Simulate different response types based on URL
    if (url.includes('health')) {
      return { status: 'ok', timestamp: Date.now() };
    }
    
    if (url.includes('balance')) {
      return { balance: '1000000', denom: 'unrn' };
    }

    return { success: true, data: {} };
  }

  setNetworkStatus(status: 'online' | 'offline') {
    this.networkStatus = status;
  }

  setLatency(ms: number) {
    this.latency = ms;
  }

  private async simulateLatency(): Promise<void> {
    return new Promise(resolve => setTimeout(resolve, this.latency));
  }
}

// Factory function to create all mock services
export function createMockServices() {
  return {
    httpClient: new MockHttpClient(),
    xionClient: new MockXionClient(),
    webSocket: MockWebSocket,
    ledgerTransport: new MockLedgerTransport(),
    cryptoProvider: new MockCryptoProvider(),
    storageProvider: new MockStorageProvider(),
    networkProvider: new MockNetworkProvider()
  };
}
