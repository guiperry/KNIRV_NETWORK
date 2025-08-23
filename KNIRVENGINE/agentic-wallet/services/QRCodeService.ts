import { Alert } from 'react-native';
import AsyncStorage from '@react-native-async-storage/async-storage';
import CryptoJS from 'crypto-js';

export interface WalletConnectionData {
  type: 'WALLET_CONNECT';
  sessionId: string;
  walletAddress: string;
  publicKey: string;
  chainId: string;
  timestamp: number;
  signature: string;
}

export interface TransactionRequest {
  type: 'TRANSACTION_REQUEST';
  sessionId: string;
  from: string;
  to: string;
  amount: string;
  token?: string;
  memo?: string;
  gasLimit?: string;
  timestamp: number;
  requestId: string;
}

export interface SyncRequest {
  type: 'SYNC_REQUEST';
  sessionId: string;
  walletData: {
    addresses: string[];
    balances: { [address: string]: string };
    transactions: any[];
  };
  timestamp: number;
}

export type QRCodeData = WalletConnectionData | TransactionRequest | SyncRequest;

export class QRCodeService {
  private static instance: QRCodeService;
  private activeSessions: Map<string, any> = new Map();
  private encryptionKey: string = '';

  private constructor() {
    this.initializeEncryptionKey();
  }

  public static getInstance(): QRCodeService {
    if (!QRCodeService.instance) {
      QRCodeService.instance = new QRCodeService();
    }
    return QRCodeService.instance;
  }

  private async initializeEncryptionKey(): Promise<void> {
    try {
      let key = await AsyncStorage.getItem('qr_encryption_key');
      if (!key) {
        key = CryptoJS.lib.WordArray.random(256/8).toString();
        await AsyncStorage.setItem('qr_encryption_key', key);
      }
      this.encryptionKey = key;
    } catch (error) {
      console.error('Failed to initialize encryption key:', error);
      // Fallback to a session-based key
      this.encryptionKey = CryptoJS.lib.WordArray.random(256/8).toString();
    }
  }

  /**
   * Generate QR code data for wallet connection
   */
  public async generateWalletConnectionQR(
    walletAddress: string,
    publicKey: string,
    chainId: string = 'knirv-1'
  ): Promise<string> {
    const sessionId = this.generateSessionId();
    const timestamp = Date.now();
    
    const connectionData: WalletConnectionData = {
      type: 'WALLET_CONNECT',
      sessionId,
      walletAddress,
      publicKey,
      chainId,
      timestamp,
      signature: await this.signData({ sessionId, walletAddress, publicKey, chainId, timestamp })
    };

    // Store session for validation
    this.activeSessions.set(sessionId, {
      type: 'connection',
      data: connectionData,
      createdAt: timestamp,
      status: 'pending'
    });

    return this.encryptAndEncode(connectionData);
  }

  /**
   * Generate QR code data for transaction request
   */
  public async generateTransactionRequestQR(
    sessionId: string,
    from: string,
    to: string,
    amount: string,
    options: {
      token?: string;
      memo?: string;
      gasLimit?: string;
    } = {}
  ): Promise<string> {
    const requestId = this.generateRequestId();
    const timestamp = Date.now();

    const transactionRequest: TransactionRequest = {
      type: 'TRANSACTION_REQUEST',
      sessionId,
      from,
      to,
      amount,
      token: options.token,
      memo: options.memo,
      gasLimit: options.gasLimit,
      timestamp,
      requestId
    };

    // Store request for tracking
    this.activeSessions.set(requestId, {
      type: 'transaction',
      data: transactionRequest,
      createdAt: timestamp,
      status: 'pending'
    });

    return this.encryptAndEncode(transactionRequest);
  }

  /**
   * Generate QR code data for wallet sync
   */
  public async generateSyncRequestQR(
    sessionId: string,
    walletData: {
      addresses: string[];
      balances: { [address: string]: string };
      transactions: any[];
    }
  ): Promise<string> {
    const timestamp = Date.now();

    const syncRequest: SyncRequest = {
      type: 'SYNC_REQUEST',
      sessionId,
      walletData,
      timestamp
    };

    return this.encryptAndEncode(syncRequest);
  }

  /**
   * Parse and decrypt QR code data
   */
  public async parseQRCodeData(qrData: string): Promise<QRCodeData> {
    try {
      const decryptedData = this.decryptAndDecode(qrData);
      
      // Validate timestamp (reject if older than 5 minutes)
      const now = Date.now();
      const maxAge = 5 * 60 * 1000; // 5 minutes
      
      if (now - decryptedData.timestamp > maxAge) {
        throw new Error('QR code has expired');
      }

      return decryptedData;
    } catch (error) {
      throw new Error(`Failed to parse QR code: ${error.message}`);
    }
  }

  /**
   * Handle wallet connection from QR code
   */
  public async handleWalletConnection(connectionData: WalletConnectionData): Promise<boolean> {
    try {
      // Verify signature
      const isValid = await this.verifySignature(connectionData);
      if (!isValid) {
        throw new Error('Invalid signature');
      }

      // Store connection session
      this.activeSessions.set(connectionData.sessionId, {
        type: 'connection',
        data: connectionData,
        createdAt: connectionData.timestamp,
        status: 'connected'
      });

      // Store wallet connection data
      await this.storeWalletConnection(connectionData);

      return true;
    } catch (error) {
      console.error('Failed to handle wallet connection:', error);
      Alert.alert('Connection Failed', error.message);
      return false;
    }
  }

  /**
   * Handle transaction request from QR code
   */
  public async handleTransactionRequest(
    transactionRequest: TransactionRequest,
    onApprove: (request: TransactionRequest) => Promise<boolean>,
    onReject: (request: TransactionRequest) => void
  ): Promise<void> {
    try {
      Alert.alert(
        'Transaction Request',
        `Send ${transactionRequest.amount} ${transactionRequest.token || 'NRN'} to ${transactionRequest.to}?`,
        [
          {
            text: 'Reject',
            style: 'cancel',
            onPress: () => onReject(transactionRequest)
          },
          {
            text: 'Approve',
            onPress: async () => {
              const success = await onApprove(transactionRequest);
              if (success) {
                this.updateSessionStatus(transactionRequest.requestId, 'approved');
              } else {
                this.updateSessionStatus(transactionRequest.requestId, 'failed');
              }
            }
          }
        ]
      );
    } catch (error) {
      console.error('Failed to handle transaction request:', error);
      Alert.alert('Error', 'Failed to process transaction request');
    }
  }

  /**
   * Get active sessions
   */
  public getActiveSessions(): Map<string, any> {
    return this.activeSessions;
  }

  /**
   * Clear expired sessions
   */
  public clearExpiredSessions(): void {
    const now = Date.now();
    const maxAge = 30 * 60 * 1000; // 30 minutes

    for (const [sessionId, session] of this.activeSessions.entries()) {
      if (now - session.createdAt > maxAge) {
        this.activeSessions.delete(sessionId);
      }
    }
  }

  // Private helper methods
  private generateSessionId(): string {
    return `session_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
  }

  private generateRequestId(): string {
    return `request_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
  }

  private encryptAndEncode(data: QRCodeData): string {
    const jsonString = JSON.stringify(data);
    const encrypted = CryptoJS.AES.encrypt(jsonString, this.encryptionKey).toString();
    return `knirv://wallet?data=${encodeURIComponent(encrypted)}`;
  }

  private decryptAndDecode(qrData: string): QRCodeData {
    // Extract data from URL format
    const url = new URL(qrData);
    const encryptedData = url.searchParams.get('data');
    
    if (!encryptedData) {
      throw new Error('Invalid QR code format');
    }

    const decrypted = CryptoJS.AES.decrypt(decodeURIComponent(encryptedData), this.encryptionKey);
    const jsonString = decrypted.toString(CryptoJS.enc.Utf8);
    
    if (!jsonString) {
      throw new Error('Failed to decrypt QR code data');
    }

    return JSON.parse(jsonString);
  }

  private async signData(data: any): Promise<string> {
    // Simple signature implementation - in production, use proper cryptographic signing
    const dataString = JSON.stringify(data);
    return CryptoJS.HmacSHA256(dataString, this.encryptionKey).toString();
  }

  private async verifySignature(data: WalletConnectionData): Promise<boolean> {
    const { signature, ...dataWithoutSignature } = data;
    const expectedSignature = await this.signData(dataWithoutSignature);
    return signature === expectedSignature;
  }

  private async storeWalletConnection(connectionData: WalletConnectionData): Promise<void> {
    try {
      const connections = await this.getStoredConnections();
      connections[connectionData.sessionId] = {
        walletAddress: connectionData.walletAddress,
        publicKey: connectionData.publicKey,
        chainId: connectionData.chainId,
        connectedAt: connectionData.timestamp,
        lastSeen: Date.now()
      };
      
      await AsyncStorage.setItem('wallet_connections', JSON.stringify(connections));
    } catch (error) {
      console.error('Failed to store wallet connection:', error);
    }
  }

  private async getStoredConnections(): Promise<{ [sessionId: string]: any }> {
    try {
      const stored = await AsyncStorage.getItem('wallet_connections');
      return stored ? JSON.parse(stored) : {};
    } catch (error) {
      console.error('Failed to get stored connections:', error);
      return {};
    }
  }

  private updateSessionStatus(sessionId: string, status: string): void {
    const session = this.activeSessions.get(sessionId);
    if (session) {
      session.status = status;
      session.updatedAt = Date.now();
    }
  }
}
