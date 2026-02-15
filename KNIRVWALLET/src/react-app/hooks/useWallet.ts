import { useState, useEffect, useCallback } from 'react';
// @ts-ignore - Importing from local workspace without type definitions might cause TS issues
import { KnirvWallet } from 'knirvwallet-module/wallet';
// @ts-ignore
import { Account } from 'knirvwallet-module/wallet/account';

export type WalletStatus = 'initializing' | 'no_wallet' | 'locked' | 'unlocked';

const WALLET_STORAGE_KEY = 'knirv_wallet_vault';

export const useWallet = () => {
  const [wallet, setWallet] = useState<any | null>(null);
  const [status, setStatus] = useState<WalletStatus>('initializing');
  const [accounts, setAccounts] = useState<Account[]>([]);
  const [currentAccount, setCurrentAccount] = useState<Account | null>(null);
  const [error, setError] = useState<string | null>(null);

  // Initialize wallet state from storage
  useEffect(() => {
    const checkStorage = () => {
      const storedVault = localStorage.getItem(WALLET_STORAGE_KEY);
      if (storedVault) {
        setStatus('locked');
      } else {
        setStatus('no_wallet');
      }
    };
    
    checkStorage();
  }, []);

  // Update accounts when wallet changes
  useEffect(() => {
    if (wallet) {
      setAccounts(wallet.accounts || []);
      setCurrentAccount(wallet.currentAccount || null);
    }
  }, [wallet]);

  const createWallet = useCallback(async (password: string): Promise<string> => {
    try {
      setStatus('initializing');
      // Generate a new mnemonic (12 words)
      const mnemonic = KnirvWallet.generateMnemonic(12);
      
      // Create wallet instance
      const newWallet = await KnirvWallet.createByMnemonic(mnemonic);
      
      // Encrypt and save to storage
      const serialized = await newWallet.serialize(password);
      localStorage.setItem(WALLET_STORAGE_KEY, serialized);
      
      setWallet(newWallet);
      setStatus('unlocked');
      return mnemonic;
    } catch (err: any) {
      console.error('Failed to create wallet:', err);
      setError(err.message || 'Failed to create wallet');
      setStatus('no_wallet');
      throw err;
    }
  }, []);

  const importWallet = useCallback(async (mnemonic: string, password: string) => {
    try {
      setStatus('initializing');
      const newWallet = await KnirvWallet.createByMnemonic(mnemonic);
      
      const serialized = await newWallet.serialize(password);
      localStorage.setItem(WALLET_STORAGE_KEY, serialized);
      
      setWallet(newWallet);
      setStatus('unlocked');
    } catch (err: any) {
      console.error('Failed to import wallet:', err);
      setError(err.message || 'Failed to import wallet');
      setStatus('no_wallet');
      throw err;
    }
  }, []);

  const unlockWallet = useCallback(async (password: string) => {
    try {
      setStatus('initializing');
      const storedVault = localStorage.getItem(WALLET_STORAGE_KEY);
      
      if (!storedVault) {
        throw new Error('No wallet found');
      }

      const unlockedWallet = await KnirvWallet.deserialize(storedVault, password);
      setWallet(unlockedWallet);
      setStatus('unlocked');
      setError(null);
    } catch (err: any) {
      console.error('Failed to unlock wallet:', err);
      setError('Incorrect password');
      setStatus('locked');
      throw err;
    }
  }, []);

  const lockWallet = useCallback(() => {
    setWallet(null);
    setAccounts([]);
    setCurrentAccount(null);
    setStatus('locked');
  }, []);

  const clearWallet = useCallback(() => {
    localStorage.removeItem(WALLET_STORAGE_KEY);
    setWallet(null);
    setAccounts([]);
    setCurrentAccount(null);
    setStatus('no_wallet');
  }, []);

  // Wrapper for transaction signing
  const signTransaction = useCallback(async (tx: any) => {
    if (!wallet || status !== 'unlocked') {
      throw new Error('Wallet is locked or not initialized');
    }
    // This is a placeholder for the actual signing logic which depends on the provider structure
    // In a real implementation, we would construct a provider and call wallet.sign
    return wallet.signTransaction(tx);
  }, [wallet, status]);

  return {
    wallet,
    status,
    accounts,
    currentAccount,
    error,
    createWallet,
    importWallet,
    unlockWallet,
    lockWallet,
    clearWallet,
    signTransaction
  };
};
