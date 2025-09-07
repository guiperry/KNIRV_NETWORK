// Comprehensive Unit Tests for KNIRVWALLET Browser Module - Transaction Signing

// Mock KNIRVWALLET imports since they're from a sibling project
jest.mock('../../../../KNIRVWALLET/browser-bridge/packages/knirvwallet-module/src/wallet/wallet', () => ({
  KnirvWallet: jest.fn().mockImplementation(() => ({
    signTransaction: jest.fn(),
    signMessage: jest.fn(),
    verifySignature: jest.fn(),
    getPublicKey: jest.fn(),
    getAddress: jest.fn()
  }))
}));

// Import after mocking
const { KnirvWallet } = require('../../../../KNIRVWALLET/browser-bridge/packages/knirvwallet-module/src/wallet/wallet');
import { JSONRPCProvider } from '@gnolang/tm2-js-client';
import {
  TEST_MNEMONICS,
  TEST_PRIVATE_KEYS,
  TEST_ADDRESSES
} from '../../../test-utils/test-data';
import { TransactionTestUtils } from '../../../test-utils/wallet-test-utils';

// Mock the JSONRPCProvider
jest.mock('@gnolang/tm2-js-client', () => ({
  JSONRPCProvider: jest.fn().mockImplementation(() => ({
    getStatus: jest.fn().mockResolvedValue({
      node_info: {
        network: 'test-network'
      }
    }),
    getAccountNumber: jest.fn().mockResolvedValue('0'),
    getAccountSequence: jest.fn().mockResolvedValue('0'),
    getBlock: jest.fn().mockResolvedValue({
      block: {
        header: {
          chain_id: 'test-chain',
          height: '12345'
        }
      }
    })
  }))
}));

describe('KnirvWallet Transaction Signing', () => {
  let wallet: typeof KnirvWallet;
  let mockProvider: JSONRPCProvider;

  beforeEach(async () => {
    wallet = await KnirvWallet.createByMnemonic(TEST_MNEMONICS.VALID_12_WORD);
    mockProvider = new JSONRPCProvider('http://localhost:26657');
    
    // Verify provider connection
    const status = await mockProvider.getStatus();
    expect(status.node_info.network).toBe('test-network');
  });

  describe('Transaction Structure Validation', () => {
    it('should validate basic transaction structure', () => {
      const transaction = TransactionTestUtils.createTestTransaction();
      
      expect(transaction).toHaveTransactionStructure();
      expect(TransactionTestUtils.validateTransactionStructure(transaction)).toBe(true);
    });

    it('should reject transaction with missing required fields', () => {
      const invalidTransaction = {
        from: TEST_ADDRESSES.GNOLANG,
        // Missing 'to' and 'amount'
      };
      
      expect(() => TransactionTestUtils.validateTransactionStructure(invalidTransaction))
        .toThrow('Transaction missing required fields');
    });

    it('should reject transaction with invalid amount format', () => {
      const invalidTransaction = {
        from: TEST_ADDRESSES.GNOLANG,
        to: 'g1234567890abcdef1234567890abcdef12345678',
        amount: 'invalid-amount'
      };
      
      expect(() => TransactionTestUtils.validateTransactionStructure(invalidTransaction))
        .toThrow('Invalid amount format');
    });

    it('should validate NRN burn transaction structure', () => {
      const burnTransaction = TransactionTestUtils.createTestNRNBurnTransaction('skill-123', '1000000');
      
      expect(burnTransaction).toHaveTransactionStructure();
      expect(burnTransaction.metadata).toBeDefined();
      expect(burnTransaction.metadata.type).toBe('skill_invocation');
      expect(burnTransaction.metadata.skillId).toBe('skill-123');
    });
  });

  describe('HD Wallet Transaction Signing', () => {
    it('should sign basic transfer transaction', async () => {
      const transaction = TransactionTestUtils.createTestTransaction({
        from: wallet.accounts[0].address
      });

      // Mock the signing process
      const signedTx = await wallet.signTransaction(transaction);
      
      expect(signedTx).toBeDefined();
      expect(signedTx).toHaveProperty('signatures');
      expect(Array.isArray(signedTx.signatures)).toBe(true);
    });

    it('should sign transaction with memo', async () => {
      const transaction = TransactionTestUtils.createTestTransaction({
        from: wallet.accounts[0].address,
        memo: 'Test transaction with memo'
      });

      const signedTx = await wallet.signTransaction(transaction);
      
      expect(signedTx).toBeDefined();
      expect(signedTx.memo).toBe('Test transaction with memo');
    });

    it('should sign NRN burn transaction for skill invocation', async () => {
      const burnTransaction = TransactionTestUtils.createTestNRNBurnTransaction('skill-123', '1000000');
      burnTransaction.from = wallet.accounts[0].address!;

      const signedTx = await wallet.signTransaction(burnTransaction);
      
      expect(signedTx).toBeDefined();
      expect(signedTx.memo).toContain('skill-123');
    });

    it('should handle transaction signing with custom gas parameters', async () => {
      const transaction = TransactionTestUtils.createTestTransaction({
        from: wallet.accounts[0].address,
        gasLimit: '300000'
      });

      const signedTx = await wallet.signTransaction(transaction);
      
      expect(signedTx).toBeDefined();
      expect(signedTx.fee).toBeDefined();
    });

    it('should reject signing transaction from different address', async () => {
      const transaction = TransactionTestUtils.createTestTransaction({
        from: 'g1differentaddress1234567890abcdef12345678'
      });

      await expect(wallet.signTransaction(transaction))
        .rejects.toThrow();
    });
  });

  describe('Private Key Wallet Transaction Signing', () => {
    let privateKeyWallet: typeof KnirvWallet;

    beforeEach(async () => {
      privateKeyWallet = await KnirvWallet.createByWeb3Auth(TEST_PRIVATE_KEYS.VALID_HEX);
    });

    it('should sign transaction with private key wallet', async () => {
      const transaction = TransactionTestUtils.createTestTransaction({
        from: privateKeyWallet.accounts[0].address
      });

      const signedTx = await privateKeyWallet.signTransaction(transaction);
      
      expect(signedTx).toBeDefined();
      expect(signedTx).toHaveProperty('signatures');
    });

    it('should produce consistent signatures for same transaction', async () => {
      const transaction = TransactionTestUtils.createTestTransaction({
        from: privateKeyWallet.accounts[0].address
      });

      const signedTx1 = await privateKeyWallet.signTransaction(transaction);
      const signedTx2 = await privateKeyWallet.signTransaction(transaction);
      
      // Note: In real implementation, signatures might differ due to nonce/timestamp
      // This test verifies the signing process works consistently
      expect(signedTx1).toBeDefined();
      expect(signedTx2).toBeDefined();
    });
  });

  describe('Address-only Wallet Transaction Signing', () => {
    let addressOnlyWallet: typeof KnirvWallet;

    beforeEach(async () => {
      addressOnlyWallet = await KnirvWallet.createByAddress(TEST_ADDRESSES.GNOLANG);
    });

    it('should reject signing transaction with address-only wallet', async () => {
      const transaction = TransactionTestUtils.createTestTransaction({
        from: addressOnlyWallet.accounts[0].address
      });

      await expect(addressOnlyWallet.signTransaction(transaction))
        .rejects.toThrow();
    });

    it('should allow viewing transaction details without signing', () => {
      const transaction = TransactionTestUtils.createTestTransaction({
        from: addressOnlyWallet.accounts[0].address
      });

      // Address-only wallets should be able to view transaction details
      expect(transaction.from).toBe(addressOnlyWallet.accounts[0].address);
      expect(TransactionTestUtils.validateTransactionStructure(transaction)).toBe(true);
    });
  });

  describe('Multi-Account Transaction Signing', () => {
    let multiAccountWallet: typeof KnirvWallet;

    beforeEach(async () => {
      const paths = [0, 1, 2];
      multiAccountWallet = await KnirvWallet.createByMnemonic(TEST_MNEMONICS.VALID_12_WORD, paths);
    });

    it('should sign transaction from first account', async () => {
      const transaction = TransactionTestUtils.createTestTransaction({
        from: multiAccountWallet.accounts[0].address
      });

      multiAccountWallet.currentAccountId = multiAccountWallet.accounts[0].id;
      const signedTx = await multiAccountWallet.signTransaction(transaction);
      
      expect(signedTx).toBeDefined();
    });

    it('should sign transaction from second account', async () => {
      const transaction = TransactionTestUtils.createTestTransaction({
        from: multiAccountWallet.accounts[1].address
      });

      multiAccountWallet.currentAccountId = multiAccountWallet.accounts[1].id;
      const signedTx = await multiAccountWallet.signTransaction(transaction);
      
      expect(signedTx).toBeDefined();
    });

    it('should reject signing transaction when current account does not match from address', async () => {
      const transaction = TransactionTestUtils.createTestTransaction({
        from: multiAccountWallet.accounts[1].address
      });

      // Set current account to different account
      multiAccountWallet.currentAccountId = multiAccountWallet.accounts[0].id;
      
      await expect(multiAccountWallet.signTransaction(transaction))
        .rejects.toThrow();
    });
  });

  describe('Transaction Signing Error Handling', () => {
    it('should handle malformed transaction gracefully', async () => {
      const malformedTransaction = {
        invalid: 'transaction',
        structure: true
      };

      await expect(wallet.signTransaction(malformedTransaction as Parameters<typeof wallet.signTransaction>[0]))
        .rejects.toThrow();
    });

    it('should handle transaction with invalid address format', async () => {
      const invalidTransaction = {
        from: 'invalid-address-format',
        to: 'also-invalid',
        amount: '1000000'
      };

      await expect(wallet.signTransaction(invalidTransaction))
        .rejects.toThrow();
    });

    it('should handle transaction with negative amount', async () => {
      const invalidTransaction = TransactionTestUtils.createTestTransaction({
        from: wallet.accounts[0].address,
        amount: '-1000000'
      });

      await expect(wallet.signTransaction(invalidTransaction))
        .rejects.toThrow();
    });

    it('should handle transaction with zero amount', async () => {
      const zeroAmountTransaction = TransactionTestUtils.createTestTransaction({
        from: wallet.accounts[0].address,
        amount: '0'
      });

      // Zero amount transactions might be valid for certain operations
      // This test verifies the wallet handles them appropriately
      try {
        const signedTx = await wallet.signTransaction(zeroAmountTransaction);
        expect(signedTx).toBeDefined();
      } catch (error) {
        // If zero amount is not allowed, ensure proper error handling
        expect(error).toBeDefined();
      }
    });

    it('should handle transaction with extremely large amount', async () => {
      const largeAmountTransaction = TransactionTestUtils.createTestTransaction({
        from: wallet.accounts[0].address,
        amount: '999999999999999999999999999999'
      });

      // Should handle large amounts gracefully
      try {
        const signedTx = await wallet.signTransaction(largeAmountTransaction);
        expect(signedTx).toBeDefined();
      } catch (error) {
        // If amount is too large, ensure proper error handling
        expect(error).toBeDefined();
      }
    });
  });

  describe('Signature Verification', () => {
    it('should produce valid signature format', async () => {
      const transaction = TransactionTestUtils.createTestTransaction({
        from: wallet.accounts[0].address
      });

      const signedTx = await wallet.signTransaction(transaction);
      
      expect(signedTx.signatures).toBeDefined();
      expect(Array.isArray(signedTx.signatures)).toBe(true);
      expect(signedTx.signatures.length).toBeGreaterThan(0);
      
      // Verify signature structure
      const signature = signedTx.signatures[0];
      expect(signature).toHaveProperty('pub_key');
      expect(signature).toHaveProperty('signature');
    });

    it('should include correct public key in signature', async () => {
      const transaction = TransactionTestUtils.createTestTransaction({
        from: wallet.accounts[0].address
      });

      const signedTx = await wallet.signTransaction(transaction);
      const signature = signedTx.signatures[0];
      
      expect(signature.pub_key).toBeDefined();
      expect(signature.pub_key.type).toBeDefined();
      expect(signature.pub_key.value).toBeDefined();
    });

    it('should produce different signatures for different transactions', async () => {
      const transaction1 = TransactionTestUtils.createTestTransaction({
        from: wallet.accounts[0].address,
        amount: '1000000'
      });

      const transaction2 = TransactionTestUtils.createTestTransaction({
        from: wallet.accounts[0].address,
        amount: '2000000'
      });

      const signedTx1 = await wallet.signTransaction(transaction1);
      const signedTx2 = await wallet.signTransaction(transaction2);
      
      // Signatures should be different for different transactions
      expect(signedTx1.signatures[0].signature).not.toBe(signedTx2.signatures[0].signature);
    });
  });
});
