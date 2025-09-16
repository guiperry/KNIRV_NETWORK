/**
 * Tests for QRPaymentService
 */

import { qrPaymentService, QRPaymentRequest } from '../../../src/services/QRPaymentService';
import { walletIntegrationService } from '../../../src/services/WalletIntegrationService';

// Mock the wallet integration service
jest.mock('../../../src/services/WalletIntegrationService');

describe('QRPaymentService', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    // Reset service state
    qrPaymentService['pendingPayments'].clear();
    qrPaymentService['paymentHistory'] = [];
  });

  describe('parseQRCode', () => {
    it('should parse valid JSON payment request', () => {
      const paymentRequest: QRPaymentRequest = {
        type: 'payment',
        amount: '100.00',
        recipient: 'knirv1recipient...',
        memo: 'Test payment'
      };

      const qrData = JSON.stringify(paymentRequest);
      const result = qrPaymentService.parseQRCode(qrData);

      expect(result.success).toBe(true);
      expect(result.data).toEqual(paymentRequest);
      expect(result.rawData).toBe(qrData);
    });

    it('should parse skill invocation request', () => {
      const skillRequest: QRPaymentRequest = {
        type: 'skill_invocation',
        skillId: 'analysis-skill',
        skillName: 'Data Analysis',
        nrnCost: '50.00',
        metadata: {
          parameters: { data: 'test' },
          timeout: 30000
        }
      };

      const qrData = JSON.stringify(skillRequest);
      const result = qrPaymentService.parseQRCode(qrData);

      expect(result.success).toBe(true);
      expect(result.data).toEqual(skillRequest);
    });

    it('should parse wallet connect request', () => {
      const connectRequest: QRPaymentRequest = {
        type: 'wallet_connect',
        sessionId: 'session-123'
      };

      const qrData = JSON.stringify(connectRequest);
      const result = qrPaymentService.parseQRCode(qrData);

      expect(result.success).toBe(true);
      expect(result.data).toEqual(connectRequest);
    });

    it('should parse agent deployment request', () => {
      const deployRequest: QRPaymentRequest = {
        type: 'agent_deploy',
        agentId: 'agent-123',
        nrnCost: '75.00'
      };

      const qrData = JSON.stringify(deployRequest);
      const result = qrPaymentService.parseQRCode(qrData);

      expect(result.success).toBe(true);
      expect(result.data).toEqual(deployRequest);
    });

    it('should parse payment URI format', () => {
      const uri = 'knirv:pay?amount=100&recipient=knirv1test...&memo=test';
      const result = qrPaymentService.parseQRCode(uri);

      expect(result.success).toBe(true);
      expect(result.data?.type).toBe('payment');
      expect(result.data?.amount).toBe('100');
      expect(result.data?.recipient).toBe('knirv1test...');
      expect(result.data?.memo).toBe('test');
    });

    it('should parse skill URI format', () => {
      const uri = 'knirv:skill?skill=analysis&nrn=50&skillName=Analysis';
      const result = qrPaymentService.parseQRCode(uri);

      expect(result.success).toBe(true);
      expect(result.data?.type).toBe('skill_invocation');
      expect(result.data?.skillId).toBe('analysis');
      expect(result.data?.nrnCost).toBe('50');
      expect(result.data?.skillName).toBe('Analysis');
    });

    it('should reject expired payment request', () => {
      const expiredRequest: QRPaymentRequest = {
        type: 'payment',
        amount: '100.00',
        recipient: 'knirv1test...',
        expires: Date.now() - 60000 // Expired 1 minute ago
      };

      const qrData = JSON.stringify(expiredRequest);
      const result = qrPaymentService.parseQRCode(qrData);

      expect(result.success).toBe(false);
      expect(result.error).toBe('Payment request has expired');
    });

    it('should reject invalid JSON', () => {
      const result = qrPaymentService.parseQRCode('invalid json');

      expect(result.success).toBe(false);
      expect(result.error).toContain('Invalid payment URI format');
    });

    it('should reject invalid payment request format', () => {
      const invalidRequest = {
        type: 'invalid_type',
        amount: '100.00'
      };

      const qrData = JSON.stringify(invalidRequest);
      const result = qrPaymentService.parseQRCode(qrData);

      expect(result.success).toBe(false);
      expect(result.error).toContain('Invalid payment URI format');
    });

    it('should reject payment request missing required fields', () => {
      const incompleteRequest = {
        type: 'payment'
        // Missing amount and recipient
      };

      const qrData = JSON.stringify(incompleteRequest);
      const result = qrPaymentService.parseQRCode(qrData);

      expect(result.success).toBe(false);
      expect(result.error).toContain('Invalid payment URI format');
    });

    it('should reject skill request missing required fields', () => {
      const incompleteRequest = {
        type: 'skill_invocation'
        // Missing skillId and nrnCost
      };

      const qrData = JSON.stringify(incompleteRequest);
      const result = qrPaymentService.parseQRCode(qrData);

      expect(result.success).toBe(false);
      expect(result.error).toContain('Invalid payment URI format');
    });
  });

  describe('processPayment', () => {
    beforeEach(() => {
      // Mock wallet service methods
      (walletIntegrationService.getCurrentAccount as jest.Mock).mockReturnValue({
        id: 'test-account',
        address: 'knirv1test...',
        name: 'Test Account',
        balance: '1000.00',
        nrnBalance: '500.00',
        isConnected: true
      });

      (walletIntegrationService.createTransaction as jest.Mock).mockResolvedValue('tx-123');
      (walletIntegrationService.invokeSkill as jest.Mock).mockResolvedValue('skill-tx-456');
      (walletIntegrationService.connectWallet as jest.Mock).mockResolvedValue({
        id: 'connected-account',
        address: 'knirv1connected...',
        name: 'Connected Account',
        balance: '2000.00',
        nrnBalance: '1000.00',
        isConnected: true
      });
    });

    it('should process regular payment successfully', async () => {
      const paymentRequest: QRPaymentRequest = {
        type: 'payment',
        amount: '100.00',
        recipient: 'knirv1recipient...',
        memo: 'Test payment'
      };

      const result = await qrPaymentService.processPayment(paymentRequest);

      expect(result.success).toBe(true);
      expect(result.transactionId).toBe('tx-123');
      expect(result.receipt?.type).toBe('payment');
      expect(result.receipt?.amount).toBe('100.00');
      expect(result.receipt?.recipient).toBe('knirv1recipient...');
      expect(result.receipt?.status).toBe('pending');
      expect(result.processingTime).toBeGreaterThan(0);

      expect(walletIntegrationService.createTransaction).toHaveBeenCalledWith({
        from: 'knirv1test...',
        to: 'knirv1recipient...',
        amount: '100.00',
        memo: 'Test payment',
        nrnAmount: undefined
      });
    });

    it('should process skill invocation successfully', async () => {
      const skillRequest: QRPaymentRequest = {
        type: 'skill_invocation',
        skillId: 'analysis-skill',
        skillName: 'Data Analysis',
        nrnCost: '50.00',
        metadata: {
          parameters: { data: 'test' },
          expectedOutput: { result: 'analysis' },
          timeout: 30000
        }
      };

      const result = await qrPaymentService.processPayment(skillRequest);

      expect(result.success).toBe(true);
      expect(result.transactionId).toBe('skill-tx-456');
      expect(result.receipt?.type).toBe('skill_invocation');
      expect(result.receipt?.nrnAmount).toBe('50.00');
      expect(result.receipt?.recipient).toBe('skill_network');

      expect(walletIntegrationService.invokeSkill).toHaveBeenCalledWith({
        skillId: 'analysis-skill',
        skillName: 'Data Analysis',
        nrnCost: '50.00',
        parameters: { data: 'test' },
        expectedOutput: { result: 'analysis' },
        timeout: 30000
      });
    });

    it('should process wallet connection successfully', async () => {
      const connectRequest: QRPaymentRequest = {
        type: 'wallet_connect',
        sessionId: 'session-123'
      };

      const result = await qrPaymentService.processPayment(connectRequest);

      expect(result.success).toBe(true);
      expect(result.receipt?.type).toBe('wallet_connect');
      expect(result.receipt?.recipient).toBe('knirv1connected...');
      expect(result.receipt?.status).toBe('confirmed');

      expect(walletIntegrationService.connectWallet).toHaveBeenCalled();
    });

    it('should process agent deployment successfully', async () => {
      const deployRequest: QRPaymentRequest = {
        type: 'agent_deploy',
        agentId: 'agent-123',
        nrnCost: '75.00'
      };

      const result = await qrPaymentService.processPayment(deployRequest);

      expect(result.success).toBe(true);
      expect(result.receipt?.type).toBe('agent_deploy');
      expect(result.receipt?.nrnAmount).toBe('75.00');
      expect(result.receipt?.recipient).toBe('agent_network');

      expect(walletIntegrationService.createTransaction).toHaveBeenCalledWith({
        from: 'knirv1test...',
        to: 'agent_network',
        amount: '0',
        nrnAmount: '75.00',
        memo: 'Agent deployment: agent-123'
      });
    });

    it('should handle unsupported payment type', async () => {
      const invalidRequest = {
        type: 'unsupported_type' as never
      };

      const result = await qrPaymentService.processPayment(invalidRequest);

      expect(result.success).toBe(false);
      expect(result.error).toContain('Unsupported payment type');
    });

    it('should handle payment processing failure', async () => {
      (walletIntegrationService.createTransaction as jest.Mock).mockRejectedValue(
        new Error('Insufficient funds')
      );

      const paymentRequest: QRPaymentRequest = {
        type: 'payment',
        amount: '100.00',
        recipient: 'knirv1recipient...'
      };

      const result = await qrPaymentService.processPayment(paymentRequest);

      expect(result.success).toBe(false);
      expect(result.error).toBe('Insufficient funds');
    });

    it('should validate required fields for payment', async () => {
      const incompleteRequest: QRPaymentRequest = {
        type: 'payment',
        amount: '100.00'
        // Missing recipient
      };

      const result = await qrPaymentService.processPayment(incompleteRequest);

      expect(result.success).toBe(false);
      expect(result.error).toBe('Amount and recipient are required for regular payments');
    });

    it('should validate required fields for skill invocation', async () => {
      const incompleteRequest: QRPaymentRequest = {
        type: 'skill_invocation',
        skillId: 'test-skill'
        // Missing nrnCost
      };

      const result = await qrPaymentService.processPayment(incompleteRequest);

      expect(result.success).toBe(false);
      expect(result.error).toBe('Skill ID and NRN cost are required for skill invocation');
    });

    it('should validate required fields for agent deployment', async () => {
      const incompleteRequest: QRPaymentRequest = {
        type: 'agent_deploy',
        agentId: 'agent-123'
        // Missing nrnCost
      };

      const result = await qrPaymentService.processPayment(incompleteRequest);

      expect(result.success).toBe(false);
      expect(result.error).toBe('Agent ID and NRN cost are required for agent deployment');
    });

    it('should add payment to history', async () => {
      const paymentRequest: QRPaymentRequest = {
        type: 'payment',
        amount: '100.00',
        recipient: 'knirv1recipient...'
      };

      await qrPaymentService.processPayment(paymentRequest);

      const history = qrPaymentService.getPaymentHistory();
      expect(history).toHaveLength(1);
      expect(history[0].type).toBe('payment');
      expect(history[0].amount).toBe('100.00');
    });
  });

  describe('checkPaymentStatus', () => {
    it('should check payment status successfully', async () => {
      // First create a payment
      const paymentRequest: QRPaymentRequest = {
        type: 'payment',
        amount: '100.00',
        recipient: 'knirv1recipient...'
      };

      const result = await qrPaymentService.processPayment(paymentRequest);
      const transactionId = result.transactionId!;

      // Mock wallet service status check
      (walletIntegrationService.checkTransactionStatus as jest.Mock).mockResolvedValue({
        id: transactionId,
        status: 'confirmed',
        blockHeight: 12345,
        gasUsed: 21000
      });

      const receipt = await qrPaymentService.checkPaymentStatus(transactionId);

      expect(receipt).toBeDefined();
      expect(receipt?.status).toBe('confirmed');
      expect(receipt?.blockHeight).toBe(12345);
      expect(receipt?.gasUsed).toBe(21000);
    });

    it('should return null for non-existent transaction', async () => {
      const receipt = await qrPaymentService.checkPaymentStatus('non-existent');
      expect(receipt).toBeNull();
    });

    it('should handle status check failure gracefully', async () => {
      // Create a payment first
      const paymentRequest: QRPaymentRequest = {
        type: 'payment',
        amount: '100.00',
        recipient: 'knirv1recipient...'
      };

      const result = await qrPaymentService.processPayment(paymentRequest);
      const transactionId = result.transactionId!;

      // Mock wallet service failure
      (walletIntegrationService.checkTransactionStatus as jest.Mock).mockRejectedValue(
        new Error('Network error')
      );

      const receipt = await qrPaymentService.checkPaymentStatus(transactionId);

      expect(receipt).toBeDefined();
      expect(receipt?.status).toBe('pending'); // Should return cached status
    });
  });

  describe('generatePaymentQR', () => {
    it('should generate QR code with expiration', () => {
      const request: QRPaymentRequest = {
        type: 'payment',
        amount: '100.00',
        recipient: 'knirv1recipient...'
      };

      const qrCode = qrPaymentService.generatePaymentQR(request);
      const parsed = JSON.parse(qrCode);

      expect(parsed.type).toBe('payment');
      expect(parsed.amount).toBe('100.00');
      expect(parsed.recipient).toBe('knirv1recipient...');
      expect(parsed.expires).toBeGreaterThan(Date.now());
      expect(parsed.sessionId).toBeDefined();
    });

    it('should preserve existing expiration and session ID', () => {
      const request: QRPaymentRequest = {
        type: 'payment',
        amount: '100.00',
        recipient: 'knirv1recipient...',
        expires: Date.now() + 300000, // 5 minutes
        sessionId: 'existing-session'
      };

      const qrCode = qrPaymentService.generatePaymentQR(request);
      const parsed = JSON.parse(qrCode);

      expect(parsed.expires).toBe(request.expires);
      expect(parsed.sessionId).toBe('existing-session');
    });
  });

  describe('getters', () => {
    it('should return payment history', async () => {
      const paymentRequest: QRPaymentRequest = {
        type: 'payment',
        amount: '100.00',
        recipient: 'knirv1recipient...'
      };

      await qrPaymentService.processPayment(paymentRequest);

      const history = qrPaymentService.getPaymentHistory();
      expect(history).toHaveLength(1);
      expect(history[0].type).toBe('payment');
    });

    it('should return pending payments', () => {
      // Pending payments are internal state, so we test indirectly
      const pending = qrPaymentService.getPendingPayments();
      expect(Array.isArray(pending)).toBe(true);
    });
  });
});
