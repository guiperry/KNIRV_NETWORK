// Custom Jest matchers for KNIRVWALLET tests
import { expect } from '@jest/globals';
import {
  TransactionTestUtils,
  AccountTestUtils,
  KeyringTestUtils
} from './wallet-test-utils';

declare global {
  // eslint-disable-next-line @typescript-eslint/no-namespace
  namespace jest {
    interface Expect {
      toHaveTransactionStructure(): unknown;
      toBeValidAddress(prefix?: string): unknown;
      toHaveWalletStructure(): unknown;
    }
    interface Matchers<R> {
      toHaveTransactionStructure(): R;
      toBeValidAddress(prefix?: string): R;
      toHaveWalletStructure(): R;
    }
    interface InverseAsymmetricMatchers {
      toHaveTransactionStructure(): unknown;
      toBeValidAddress(prefix?: string): unknown;
      toHaveWalletStructure(): unknown;
    }
  }
}

export const customMatchers = {
  toHaveTransactionStructure(received: unknown) {
    try {
      TransactionTestUtils.validateTransactionStructure(received as Record<string, unknown>);
      return {
        message: () => `Expected transaction to have valid structure`,
        pass: true,
      };
    } catch (error) {
      return {
        message: () => `Expected transaction to have valid structure, but validation failed: ${error instanceof Error ? error.message : String(error)}`,
        pass: false,
      };
    }
  },

  toBeValidAddress(received: string, prefix?: string) {
    const isValid = typeof received === 'string' && 
                   received.length >= 38 && 
                   received.length <= 45 && 
                   (prefix ? received.startsWith(prefix) : true) &&
                   /^[a-zA-Z0-9]+$/.test(received);

    return {
      message: () => `Expected ${received} to be a valid address${prefix ? ` with prefix '${prefix}'` : ''}`,
      pass: isValid,
    };
  },

  toHaveWalletStructure(received: unknown) {
    const requiredProperties = ['accounts', 'keyrings', 'currentAccountId'];
    const receivedObj = received as Record<string, unknown>;
    const missingProperties = requiredProperties.filter(prop => !(prop in receivedObj));

    if (missingProperties.length > 0) {
      return {
        message: () => `Expected wallet to have structure with properties: ${requiredProperties.join(', ')}. Missing: ${missingProperties.join(', ')}`,
        pass: false,
      };
    }

    // Validate accounts structure
    try {
      const accounts = receivedObj.accounts as unknown[];
      accounts.forEach((account: unknown) => {
        AccountTestUtils.validateAccountStructure(account as Record<string, unknown>);
      });
    } catch (error) {
      return {
        message: () => `Wallet account validation failed: ${error instanceof Error ? error.message : String(error)}`,
        pass: false,
      };
    }

    // Validate keyrings structure
    try {
      const keyrings = receivedObj.keyrings as unknown[];
      keyrings.forEach((keyring: unknown) => {
        KeyringTestUtils.validateKeyringStructure(keyring as Record<string, unknown>);
      });
    } catch (error) {
      return {
        message: () => `Wallet keyring validation failed: ${error instanceof Error ? error.message : String(error)}`,
        pass: false,
      };
    }

    return {
      message: () => `Expected wallet to have valid structure`,
      pass: true,
    };
  },
};

// Extend Jest expect with our custom matchers
export function setupCustomMatchers() {
  expect.extend(customMatchers);
}