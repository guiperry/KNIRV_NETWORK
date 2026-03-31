/**
 * QR Payment Processing Service
 * Handles QR code scanning, payment processing, and transaction management
 */

import { walletIntegrationService, TransactionRequest } from './WalletIntegrationService';

export interface QRPaymentRequest {
  type: 'payment' | 'skill_invocation' | 'wallet_connect' | 'agent_deploy';
  amount?: string;
  recipient?: string;
  skillId?: string;
  skillName?: string;
  agentId?: string;
  nrnCost?: string;
  memo?: string;
  sessionId?: string;
  expires?: number;
  metadata?: Record<string, unknown>;
}

export interface QRScanResult {
  success: boolean;
  data?: QRPaymentRequest;
  error?: string;
  rawData?: string;
}

export interface PaymentProcessingResult {
  success: boolean;
  transactionId?: string;
  error?: string;
  processingTime: number;
  receipt?: PaymentReceipt;
}

export interface PaymentReceipt {
  transactionId: string;
  type: string;
  amount: string;
  nrnAmount?: string;
  recipient?: string;
  timestamp: Date;
  status: 'pending' | 'confirmed' | 'failed';
  blockHeight?: number;
  gasUsed?: number;
}

export interface SkillInvocationPayment {
  skillId: string;
  skillName: string;
  nrnCost: string;
  parameters: Record<string, unknown>;
  expectedOutput: Record<string, unknown>;
  timeout: number;
}

export class QRPaymentService {
  private pendingPayments: Map<string, QRPaymentRequest> = new Map();
  private paymentHistory: PaymentReceipt[] = [];
  private baseUrl: string;

  constructor(baseUrl: string = 'http://localhost:3001') {
    this.baseUrl = baseUrl;
  }

  /**
   * Parse QR code data
   */
  parseQRCode(qrData: string): QRScanResult {
    try {
      // Try to parse as JSON first
      let paymentRequest: QRPaymentRequest;

      try {
        const parsed = JSON.parse(qrData);
        
        // Validate required fields based on type
        if (this.isValidPaymentRequest(parsed)) {
          paymentRequest = parsed as QRPaymentRequest;
        } else {
          throw new Error('Invalid payment request format');
        }
      } catch {
        // Try to parse as URI format (e.g., knirv:pay?amount=100&recipient=...)
        paymentRequest = this.parsePaymentURI(qrData);
      }

      // Check if payment request has expired
      if (paymentRequest.expires && Date.now() > paymentRequest.expires) {
        return {
          success: false,
          error: 'Payment request has expired',
          rawData: qrData
        };
      }

      return {
        success: true,
        data: paymentRequest,
        rawData: qrData
      };
    } catch (error) {
      return {
        success: false,
        error: error instanceof Error ? error.message : 'Invalid QR code format',
        rawData: qrData
      };
    }
  }

  /**
   * Process a payment request
   */
  async processPayment(paymentRequest: QRPaymentRequest): Promise<PaymentProcessingResult> {
    const startTime = Date.now();
    const paymentId = this.generatePaymentId();

    try {
      // Store pending payment
      this.pendingPayments.set(paymentId, paymentRequest);

      let transactionId: string;
      let receipt: PaymentReceipt;

      switch (paymentRequest.type) {
        case 'payment': {
          const result = await this.processRegularPayment(paymentRequest);
          transactionId = result.transactionId;
          receipt = result.receipt;
          break;
        }
        case 'skill_invocation': {
          const skillResult = await this.processSkillInvocationPayment(paymentRequest);
          transactionId = skillResult.transactionId;
          receipt = skillResult.receipt;
          break;
        }
        case 'wallet_connect': {
          const connectResult = await this.processWalletConnection(paymentRequest);
          transactionId = connectResult.transactionId;
          receipt = connectResult.receipt;
          break;
        }
        case 'agent_deploy': {
          const deployResult = await this.processAgentDeployment(paymentRequest);
          transactionId = deployResult.transactionId;
          receipt = deployResult.receipt;
          break;
        }
        default:
          throw new Error(`Unsupported payment type: ${paymentRequest.type}`);
      }

      // Add to payment history
      this.paymentHistory.push(receipt);

      // Remove from pending
      this.pendingPayments.delete(paymentId);

      // Ensure minimum processing time for realistic simulation
      const processingTime = Math.max(1, Date.now() - startTime);

      return {
        success: true,
        transactionId,
        receipt,
        processingTime
      };
    } catch (error) {
      // Remove from pending on error
      this.pendingPayments.delete(paymentId);

      // Ensure minimum processing time for realistic simulation
      const processingTime = Math.max(1, Date.now() - startTime);

      return {
        success: false,
        error: error instanceof Error ? error.message : 'Payment processing failed',
        processingTime
      };
    }
  }

  /**
   * Process regular payment
   */
  private async processRegularPayment(request: QRPaymentRequest): Promise<{ transactionId: string; receipt: PaymentReceipt }> {
    if (!request.amount || !request.recipient) {
      throw new Error('Amount and recipient are required for regular payments');
    }

    const transactionRequest: TransactionRequest = {
      from: walletIntegrationService.getCurrentAccount()?.address || '',
      to: request.recipient,
      amount: request.amount,
      memo: request.memo,
      nrnAmount: request.nrnCost
    };

    const transactionId = await walletIntegrationService.createTransaction(transactionRequest);

    const receipt: PaymentReceipt = {
      transactionId,
      type: 'payment',
      amount: request.amount,
      nrnAmount: request.nrnCost,
      recipient: request.recipient,
      timestamp: new Date(),
      status: 'pending'
    };

    return { transactionId, receipt };
  }

  /**
   * Process skill invocation payment
   */
  private async processSkillInvocationPayment(request: QRPaymentRequest): Promise<{ transactionId: string; receipt: PaymentReceipt }> {
    if (!request.skillId || !request.nrnCost) {
      throw new Error('Skill ID and NRN cost are required for skill invocation');
    }

    const skillInvocation: SkillInvocationPayment = {
      skillId: request.skillId,
      skillName: request.skillName || request.skillId,
      nrnCost: request.nrnCost,
      parameters: request.metadata?.parameters as Record<string, unknown> || {},
      expectedOutput: request.metadata?.expectedOutput as Record<string, unknown> || {},
      timeout: (request.metadata?.timeout as number) || 30000
    };

    const transactionId = await walletIntegrationService.invokeSkill(skillInvocation);

    const receipt: PaymentReceipt = {
      transactionId,
      type: 'skill_invocation',
      amount: '0',
      nrnAmount: request.nrnCost,
      recipient: 'skill_network',
      timestamp: new Date(),
      status: 'pending'
    };

    return { transactionId, receipt };
  }

  /**
   * Process wallet connection
   */
  private async processWalletConnection(_request: QRPaymentRequest): Promise<{ transactionId: string; receipt: PaymentReceipt }> {
    // Connect to wallet using session ID
    const account = await walletIntegrationService.connectWallet();

    // Create a connection receipt
    const transactionId = `connect_${Date.now()}`;
    const receipt: PaymentReceipt = {
      transactionId,
      type: 'wallet_connect',
      amount: '0',
      recipient: account.address,
      timestamp: new Date(),
      status: 'confirmed'
    };

    return { transactionId, receipt };
  }

  /**
   * Process agent deployment payment
   */
  private async processAgentDeployment(request: QRPaymentRequest): Promise<{ transactionId: string; receipt: PaymentReceipt }> {
    if (!request.agentId || !request.nrnCost) {
      throw new Error('Agent ID and NRN cost are required for agent deployment');
    }

    // Create deployment transaction
    const transactionRequest: TransactionRequest = {
      from: walletIntegrationService.getCurrentAccount()?.address || '',
      to: 'agent_network',
      amount: '0',
      nrnAmount: request.nrnCost,
      memo: `Agent deployment: ${request.agentId}`
    };

    const transactionId = await walletIntegrationService.createTransaction(transactionRequest);

    const receipt: PaymentReceipt = {
      transactionId,
      type: 'agent_deploy',
      amount: '0',
      nrnAmount: request.nrnCost,
      recipient: 'agent_network',
      timestamp: new Date(),
      status: 'pending'
    };

    return { transactionId, receipt };
  }

  /**
   * Check payment status
   */
  async checkPaymentStatus(transactionId: string): Promise<PaymentReceipt | null> {
    // Check local history first
    const localReceipt = this.paymentHistory.find(r => r.transactionId === transactionId);
    if (!localReceipt) {
      return null;
    }

    try {
      // Update status from wallet service
      const transaction = await walletIntegrationService.checkTransactionStatus(transactionId);
      
      if (transaction) {
        localReceipt.status = transaction.status;
        localReceipt.blockHeight = transaction.blockHeight;
        localReceipt.gasUsed = transaction.gasUsed;
      }

      return localReceipt;
    } catch (error) {
      console.error('Failed to check payment status:', error);
      return localReceipt;
    }
  }

  /**
   * Get payment history
   */
  getPaymentHistory(): PaymentReceipt[] {
    return [...this.paymentHistory];
  }

  /**
   * Get pending payments
   */
  getPendingPayments(): QRPaymentRequest[] {
    return Array.from(this.pendingPayments.values());
  }

  /**
   * Generate QR code for payment request
   */
  generatePaymentQR(request: QRPaymentRequest): string {
    // Add expiration if not provided (default 10 minutes)
    if (!request.expires) {
      request.expires = Date.now() + (10 * 60 * 1000);
    }

    // Add session ID if not provided
    if (!request.sessionId) {
      request.sessionId = this.generateSessionId();
    }

    return JSON.stringify(request);
  }

  /**
   * Validate payment request format
   */
  private isValidPaymentRequest(data: unknown): boolean {
    if (!data || typeof data !== 'object') {
      return false;
    }

    interface PaymentRequestData {
      type?: string;
      amount?: unknown;
      recipient?: unknown;
      skillId?: unknown;
      nrnCost?: unknown;
    }

    // Check required type field
    const dataTyped = data as PaymentRequestData;
    if (!dataTyped.type || !['payment', 'skill_invocation', 'wallet_connect', 'agent_deploy'].includes(dataTyped.type)) {
      return false;
    }

    // Type-specific validation
    switch (dataTyped.type) {
      case 'payment':
        return !!(dataTyped.amount && dataTyped.recipient);
      case 'skill_invocation':
        return !!(dataTyped.skillId && dataTyped.nrnCost);
      case 'wallet_connect':
        return true; // No additional requirements
      case 'agent_deploy':
        return !!((dataTyped as { agentId?: unknown; nrnCost?: unknown }).agentId && (dataTyped as { agentId?: unknown; nrnCost?: unknown }).nrnCost);
      default:
        return false;
    }
  }

  /**
   * Parse payment URI format
   */
  private parsePaymentURI(uri: string): QRPaymentRequest {
    try {
      // Validate that this looks like a valid URI format
      if (!uri.includes(':') || !uri.startsWith('knirv:')) {
        throw new Error('Invalid URI format - must start with knirv:');
      }

      // Parse custom knirv: protocol manually since URL constructor doesn't handle it well
      const [protocol, rest] = uri.split(':', 2);
      if (protocol !== 'knirv' || !rest) {
        throw new Error('Invalid knirv URI format');
      }

      // Split path and query string
      const [path, queryString] = rest.includes('?') ? rest.split('?', 2) : [rest, ''];
      const params = new URLSearchParams(queryString);

      // Determine type based on path
      let type: QRPaymentRequest['type'] = 'payment';
      if (path === 'skill') {
        type = 'skill_invocation';
      } else if (path === 'agent') {
        type = 'agent_deploy';
      } else if (path === 'connect') {
        type = 'wallet_connect';
      } else if (path === 'pay') {
        type = 'payment';
      }

      const request: QRPaymentRequest = {
        type,
        amount: params.get('amount') || undefined,
        recipient: params.get('recipient') || params.get('to') || undefined,
        skillId: params.get('skill') || params.get('skillId') || undefined,
        skillName: params.get('skillName') || undefined,
        agentId: params.get('agent') || params.get('agentId') || undefined,
        nrnCost: params.get('nrn') || params.get('nrnCost') || undefined,
        memo: params.get('memo') || params.get('message') || undefined,
        sessionId: params.get('session') || undefined
      };

      // Add expiration if provided
      const expires = params.get('expires');
      if (expires) {
        request.expires = parseInt(expires, 10);
      }

      return request;
    } catch (error) {
      throw new Error(`Invalid payment URI format: ${error instanceof Error ? error.message : 'Unknown error'}`);
    }
  }

  private generatePaymentId(): string {
    return `payment_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
  }

  private generateSessionId(): string {
    return `session_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
  }
}

// Export singleton instance
export const qrPaymentService = new QRPaymentService();
