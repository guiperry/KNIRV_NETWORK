import { useState, useEffect, useCallback } from 'react';
// @ts-ignore - Importing from local workspace without type definitions might cause TS issues
import { KnirvWallet } from 'knirvwallet-module/wallet';
// @ts-ignore
import { Account } from 'knirvwallet-module/wallet/account';
import { DVEWallet } from '@/shared/types';

export type VaultStatus = 'initializing' | 'no_vault' | 'locked' | 'unlocked';

const VAULT_STORAGE_KEY = 'knirv_vault';
const DVE_DERIVATION_BASE_PATH = "m/44'/60'/1'/0/";

export const useVault = () => {
  const [vault, setVault] = useState<any | null>(null);
  const [status, setStatus] = useState<VaultStatus>('initializing');
  const [accounts, setAccounts] = useState<Account[]>([]);
  const [currentAccount, setCurrentAccount] = useState<Account | null>(null);
  const [dveWallets, setDVEWallets] = useState<DVEWallet[]>([]);
  const [error, setError] = useState<string | null>(null);

  // Initialize vault state from storage
  useEffect(() => {
    const checkStorage = () => {
      const storedVault = localStorage.getItem(VAULT_STORAGE_KEY);
      if (storedVault) {
        setStatus('locked');
      } else {
        setStatus('no_vault');
      }
    };
    checkStorage();
  }, []);

  // Update accounts when vault changes
  useEffect(() => {
    if (vault) {
      setAccounts(vault.accounts || []);
      setCurrentAccount(vault.currentAccount || null);
    }
  }, [vault]);

  const createVault = useCallback(async (password: string): Promise<string> => {
    try {
      setStatus('initializing');
      const mnemonic = KnirvWallet.generateMnemonic(12);
      const newVault = await KnirvWallet.createByMnemonic(mnemonic);
      const serialized = await newVault.serialize(password);
      localStorage.setItem(VAULT_STORAGE_KEY, serialized);
      setVault(newVault);
      setStatus('unlocked');
      return mnemonic;
    } catch (err: any) {
      console.error('Failed to create vault:', err);
      setError(err.message || 'Failed to create vault');
      setStatus('no_vault');
      throw err;
    }
  }, []);

  const importVault = useCallback(async (mnemonic: string, password: string) => {
    try {
      setStatus('initializing');
      const newVault = await KnirvWallet.createByMnemonic(mnemonic);
      const serialized = await newVault.serialize(password);
      localStorage.setItem(VAULT_STORAGE_KEY, serialized);
      setVault(newVault);
      setStatus('unlocked');
    } catch (err: any) {
      console.error('Failed to import vault:', err);
      setError(err.message || 'Failed to import vault');
      setStatus('no_vault');
      throw err;
    }
  }, []);

  const unlockVault = useCallback(async (password: string) => {
    try {
      setStatus('initializing');
      const storedVault = localStorage.getItem(VAULT_STORAGE_KEY);
      if (!storedVault) {
        throw new Error('No vault found');
      }
      const unlockedVault = await KnirvWallet.deserialize(storedVault, password);
      setVault(unlockedVault);
      setStatus('unlocked');
      setError(null);
    } catch (err: any) {
      console.error('Failed to unlock vault:', err);
      setError('Incorrect password');
      setStatus('locked');
      throw err;
    }
  }, []);

  const lockVault = useCallback(() => {
    setVault(null);
    setAccounts([]);
    setCurrentAccount(null);
    setDVEWallets([]);
    setStatus('locked');
  }, []);

  const clearVault = useCallback(() => {
    localStorage.removeItem(VAULT_STORAGE_KEY);
    setVault(null);
    setAccounts([]);
    setCurrentAccount(null);
    setDVEWallets([]);
    setStatus('no_vault');
  }, []);

  const deriveDVEWallet = useCallback(async (dveID: string, dveName: string, teeType: DVEWallet['teeType'], index: number): Promise<DVEWallet> => {
    if (!vault || status !== 'unlocked') {
      throw new Error('Vault is locked or not initialized');
    }
    const derivationPath = `${DVE_DERIVATION_BASE_PATH}${index}`;
    const dveWallet: DVEWallet = {
      dveID,
      dveName,
      walletAddress: `${derivationPath}_${dveID.substring(0, 8)}`,
      derivationIndex: index,
      teeType,
      status: 'offline',
      stakeAmount: 0,
      reputationScore: 0,
      capabilities: [],
      attachedPolicies: [],
      badgeNFTIDs: [],
      supervisorAgentID: '',
      dveURI: `knirv://dve/${dveID}/${teeType}`,
    };
    setDVEWallets(prev => [...prev, dveWallet]);
    return dveWallet;
  }, [vault, status]);

  const updateDVEWalletStatus = useCallback((dveID: string, updates: Partial<DVEWallet>) => {
    setDVEWallets(prev => prev.map(w => w.dveID === dveID ? { ...w, ...updates } : w));
  }, []);

  const removeDVEWallet = useCallback((dveID: string) => {
    setDVEWallets(prev => prev.filter(w => w.dveID !== dveID));
  }, []);

  const signTransaction = useCallback(async (tx: any) => {
    if (!vault || status !== 'unlocked') {
      throw new Error('Vault is locked or not initialized');
    }
    return vault.signTransaction(tx);
  }, [vault, status]);

  return {
    vault,
    status,
    accounts,
    currentAccount,
    dveWallets,
    error,
    createVault,
    importVault,
    unlockVault,
    lockVault,
    clearVault,
    deriveDVEWallet,
    updateDVEWalletStatus,
    removeDVEWallet,
    signTransaction,
  };
};
