/**
 * XION Wallet React Hook
 * Provides React integration for XION Meta Accounts and USDC-to-NRN conversions
 * Integrates with KNIRVORACLE Payment Gateway for seamless transactions
 */

import { useState, useEffect, useCallback } from 'react';
import { 
  AbstraxionWalletService, 
  XIONAccount, 
  ConversionRequest, 
  ConversionResult 
} from '../services/AbstraxionWalletService';

export interface XIONWalletState {
  account: XIONAccount | null;
  isConnected: boolean;
  isConnecting: boolean;
  error: string | null;
  paymentGatewayConfig: unknown | null;
  conversionRates: unknown | null;
}

export interface XIONWalletActions {
  connectWallet: (authMethod?: 'email' | 'social' | 'wallet' | 'passkey') => Promise<void>;
  disconnectWallet: () => Promise<void>;
  convertUSDCToNRN: (request: ConversionRequest) => Promise<ConversionResult>;
  checkConversionStatus: (transactionId: string) => Promise<ConversionResult | null>;
  getPaymentHistory: () => Promise<ConversionResult[]>;
  refreshBalance: () => Promise<void>;
  refreshRates: () => Promise<void>;
  connectMetaAccount: (authMethod: 'email' | 'social' | 'wallet' | 'passkey', identifier: string) => Promise<{ success: boolean; account?: string; error?: string }>;
}

export function useXIONWallet(): XIONWalletState & XIONWalletActions {
  // State
  const [state, setState] = useState<XIONWalletState>({
    account: null,
    isConnected: false,
    isConnecting: false,
    error: null,
    paymentGatewayConfig: null,
    conversionRates: null,
  });

  // Wallet service instance
  const [walletService] = useState(() => new AbstraxionWalletService());

  // Initialize wallet service and load configuration
  useEffect(() => {
    const initializeWallet = async () => {
      try {
        // Load payment gateway configuration
        const config = await walletService.getPaymentGatewayConfig();
        const rates = await walletService.getConversionRates();

        setState(prev => ({
          ...prev,
          paymentGatewayConfig: config,
          conversionRates: rates,
        }));

        // Check if there's an existing connection
        const currentAccount = walletService.getCurrentAccount();
        if (currentAccount) {
          setState(prev => ({
            ...prev,
            account: currentAccount,
            isConnected: true,
          }));
        }
      } catch (error) {
        console.error('Failed to initialize XION wallet:', error);
        setState(prev => ({
          ...prev,
          error: error instanceof Error ? error.message : 'Failed to initialize wallet',
        }));
      }
    };

    initializeWallet();
  }, [walletService]);

  // Connect wallet
  const connectWallet = useCallback(async (authMethod: 'email' | 'social' | 'wallet' | 'passkey' = 'email') => {
    setState(prev => ({ ...prev, isConnecting: true, error: null }));

    try {
      const account = await walletService.connectWallet(authMethod);
      
      if (account) {
        setState(prev => ({
          ...prev,
          account,
          isConnected: true,
          isConnecting: false,
        }));
      } else {
        throw new Error('Failed to connect wallet');
      }
    } catch (error) {
      console.error('Wallet connection failed:', error);
      setState(prev => ({
        ...prev,
        error: error instanceof Error ? error.message : 'Failed to connect wallet',
        isConnecting: false,
      }));
    }
  }, [walletService]);

  // Disconnect wallet
  const disconnectWallet = useCallback(async () => {
    try {
      await walletService.disconnectWallet();
      setState(prev => ({
        ...prev,
        account: null,
        isConnected: false,
        error: null,
      }));
    } catch (error) {
      console.error('Wallet disconnection failed:', error);
      setState(prev => ({
        ...prev,
        error: error instanceof Error ? error.message : 'Failed to disconnect wallet',
      }));
    }
  }, [walletService]);

  // Convert USDC to NRN
  const convertUSDCToNRN = useCallback(async (request: ConversionRequest): Promise<ConversionResult> => {
    if (!state.isConnected) {
      throw new Error('Wallet not connected');
    }

    setState(prev => ({ ...prev, error: null }));

    try {
      const result = await walletService.convertUSDCToNRN(request);
      
      // Update account balance after conversion
      const updatedAccount = walletService.getCurrentAccount();
      if (updatedAccount) {
        setState(prev => ({
          ...prev,
          account: updatedAccount,
        }));
      }

      return result;
    } catch (error) {
      console.error('USDC to NRN conversion failed:', error);
      const errorMessage = error instanceof Error ? error.message : 'Conversion failed';
      setState(prev => ({ ...prev, error: errorMessage }));
      throw error;
    }
  }, [state.isConnected, walletService]);

  // Check conversion status
  const checkConversionStatus = useCallback(async (transactionId: string): Promise<ConversionResult | null> => {
    try {
      return await walletService.checkConversionStatus(transactionId);
    } catch (error) {
      console.error('Failed to check conversion status:', error);
      setState(prev => ({
        ...prev,
        error: error instanceof Error ? error.message : 'Failed to check status',
      }));
      return null;
    }
  }, [walletService]);

  // Get payment history
  const getPaymentHistory = useCallback(async (): Promise<ConversionResult[]> => {
    if (!state.isConnected) {
      throw new Error('Wallet not connected');
    }

    try {
      return await walletService.getPaymentHistory();
    } catch (error) {
      console.error('Failed to get payment history:', error);
      setState(prev => ({
        ...prev,
        error: error instanceof Error ? error.message : 'Failed to get history',
      }));
      return [];
    }
  }, [state.isConnected, walletService]);

  // Refresh account balance
  const refreshBalance = useCallback(async () => {
    if (!state.account) return;

    try {
      const balance = await walletService.getAccountBalance(state.account.address);
      
      setState(prev => ({
        ...prev,
        account: prev.account ? {
          ...prev.account,
          usdcBalance: balance.usdc_balance,
          nrnBalance: balance.nrn_balance,
        } : null,
      }));
    } catch (error) {
      console.error('Failed to refresh balance:', error);
      setState(prev => ({
        ...prev,
        error: error instanceof Error ? error.message : 'Failed to refresh balance',
      }));
    }
  }, [state.account, walletService]);

  // Refresh conversion rates
  const refreshRates = useCallback(async () => {
    try {
      const rates = await walletService.getConversionRates();
      setState(prev => ({
        ...prev,
        conversionRates: rates,
      }));
    } catch (error) {
      console.error('Failed to refresh rates:', error);
      setState(prev => ({
        ...prev,
        error: error instanceof Error ? error.message : 'Failed to refresh rates',
      }));
    }
  }, [walletService]);

  // Connect Meta Account
  const connectMetaAccount = useCallback(async (authMethod: 'email' | 'social' | 'wallet' | 'passkey', identifier: string): Promise<{ success: boolean; account?: string; error?: string }> => {
    try {
      const result = await walletService.connectMetaAccount(authMethod, identifier);
      return {
        success: true,
        account: typeof result === 'object' && result !== null && 'account' in result ? String(result.account) : undefined
      };
    } catch (error) {
      console.error('Failed to connect meta account:', error);
      const errorMessage = error instanceof Error ? error.message : 'Failed to connect meta account';
      setState(prev => ({
        ...prev,
        error: errorMessage,
      }));
      return {
        success: false,
        error: errorMessage
      };
    }
  }, [walletService]);

  return {
    // State
    ...state,
    
    // Actions
    connectWallet,
    disconnectWallet,
    convertUSDCToNRN,
    checkConversionStatus,
    getPaymentHistory,
    refreshBalance,
    refreshRates,
    connectMetaAccount,
  };
}

export default useXIONWallet;
