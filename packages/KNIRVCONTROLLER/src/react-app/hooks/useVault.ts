import { useState, useEffect, useCallback } from 'react';
import type { MessageEnvelope } from '@knirv/sdk/signing';
import { nativeStorage } from '@/react-app/platform/nativeStorage';
import { impactMedium, notificationError } from '@/react-app/platform/haptics';
import { DVEVault } from '@/shared/types';
import { getVaultSession, updateVaultSession, type VaultStatus } from './vaultSession';

const VAULT_STORAGE_KEY = 'knirv_vault';
export const useVault = () => {
  const vaultSession = getVaultSession();
  const [vault, setVault] = useState<any | null>(() => vaultSession.vault);
  const [status, setStatus] = useState<VaultStatus>(() => vaultSession.status);
  const [accounts, setAccounts] = useState<any[]>(() => vaultSession.accounts);
  const [currentAccount, setCurrentAccount] = useState<any | null>(() => vaultSession.currentAccount);
  const [oracleAddress, setOracleAddress] = useState<string | null>(null);
  const [dveVaults, setDVEVaults] = useState<DVEVault[]>(() => vaultSession.dveVaults);
  const [error, setError] = useState<string | null>(() => vaultSession.error);

  useEffect(() => {
    const checkStorage = async () => {
      const currentSession = getVaultSession();
      if (currentSession.status !== 'initializing') {
        setVault(currentSession.vault);
        setStatus(currentSession.status);
        setAccounts(currentSession.accounts);
        setCurrentAccount(currentSession.currentAccount);
        setDVEVaults(currentSession.dveVaults);
        setError(currentSession.error);
        return;
      }
      const storedVault = await nativeStorage.get(VAULT_STORAGE_KEY);
      if (storedVault) {
        setStatus('locked');
      } else {
        setStatus('no_vault');
      }
    };
    checkStorage();
  }, []);

  useEffect(() => {
    if (vault) {
      setAccounts(vault.accounts || []);
      setCurrentAccount(vault.currentAccount || null);
    }
  }, [vault]);

  useEffect(() => {
    let cancelled = false;
    if (!currentAccount || typeof currentAccount.getAddress !== 'function') {
      setOracleAddress(null);
      return;
    }
    Promise.resolve(currentAccount.getAddress('knirv'))
      .then((address) => { if (!cancelled) setOracleAddress(address); })
      .catch(() => { if (!cancelled) setOracleAddress(null); });
    return () => { cancelled = true; };
  }, [currentAccount]);

  const createVault = useCallback(async (password: string): Promise<string> => {
    try {
      setStatus('initializing');
      // @ts-expect-error - knirvwallet-module lacks type declarations; resolved by Vite alias
      const { KnirvWallet } = await import('knirvwallet-module/wallet');
      const mnemonic = KnirvWallet.generateMnemonic(12);
      const newVault = await KnirvWallet.createByMnemonic(mnemonic);
      const serialized = await newVault.serialize(password);
      await nativeStorage.set(VAULT_STORAGE_KEY, serialized);
      setVault(newVault);
      updateVaultSession({
        vault: newVault, status: 'unlocked', accounts: newVault.accounts || [],
        currentAccount: newVault.currentAccount || null, dveVaults: [], error: null,
      });
      setStatus('unlocked');
      impactMedium();
      return mnemonic;
    } catch (err: any) {
      console.error('Failed to create vault:', err);
      notificationError();
      setError(err.message || 'Failed to create vault');
      updateVaultSession({ error: err.message || 'Failed to create vault' });
      setStatus('no_vault');
      throw err;
    }
  }, []);

  const importVault = useCallback(async (mnemonic: string, password: string) => {
    try {
      setStatus('initializing');
      // @ts-expect-error - knirvwallet-module lacks type declarations; resolved by Vite alias
      const { KnirvWallet } = await import('knirvwallet-module/wallet');
      const newVault = await KnirvWallet.createByMnemonic(mnemonic);
      const serialized = await newVault.serialize(password);
      await nativeStorage.set(VAULT_STORAGE_KEY, serialized);
      setVault(newVault);
      updateVaultSession({
        vault: newVault, status: 'unlocked', accounts: newVault.accounts || [],
        currentAccount: newVault.currentAccount || null, dveVaults: [], error: null,
      });
      setStatus('unlocked');
    } catch (err: any) {
      console.error('Failed to import vault:', err);
      notificationError();
      setError(err.message || 'Failed to import vault');
      updateVaultSession({ error: err.message || 'Failed to import vault' });
      setStatus('no_vault');
      throw err;
    }
  }, []);

  const unlockVault = useCallback(async (password: string) => {
    try {
      setStatus('initializing');
      const storedVault = await nativeStorage.get(VAULT_STORAGE_KEY);
      if (!storedVault) { throw new Error('No vault found'); }
      // @ts-expect-error - knirvwallet-module lacks type declarations; resolved by Vite alias
      const { KnirvWallet } = await import('knirvwallet-module/wallet');
      const unlockedVault = await KnirvWallet.deserialize(storedVault, password);
      setVault(unlockedVault);
      updateVaultSession({
        vault: unlockedVault, status: 'unlocked', accounts: unlockedVault.accounts || [],
        currentAccount: unlockedVault.currentAccount || null, dveVaults: [], error: null,
      });
      setStatus('unlocked');
      setError(null);
    } catch (err: any) {
      console.error('Failed to unlock vault:', err);
      notificationError();
      setError('Incorrect password');
      updateVaultSession({ error: 'Incorrect password' });
      setStatus('locked');
      throw err;
    }
  }, []);

  const lockVault = useCallback(() => {
    setVault(null); setAccounts([]); setCurrentAccount(null); setDVEVaults([]);
    setStatus('locked');
    updateVaultSession({ vault: null, status: 'locked', accounts: [], currentAccount: null, dveVaults: [] });
  }, []);

  const clearVault = useCallback(async () => {
    await nativeStorage.remove(VAULT_STORAGE_KEY);
    setVault(null); setAccounts([]); setCurrentAccount(null); setDVEVaults([]);
    setStatus('no_vault');
    updateVaultSession({ vault: null, status: 'no_vault', accounts: [], currentAccount: null, dveVaults: [], error: null });
  }, []);

  const deriveDVEVault = useCallback(async (dveID: string, dveName: string, teeType: DVEVault['teeType'], index: number): Promise<DVEVault> => {
    if (!vault || status !== 'unlocked') {
      throw new Error('Vault is locked or not initialized');
    }
    if (!Number.isSafeInteger(index) || index < 0) {
      throw new Error('DVE derivation index must be a non-negative integer');
    }
    const keyring = vault.defaultHDWalletKeyring;
    if (!keyring || typeof keyring.getPublicKey !== 'function') {
      throw new Error('DVE service keys require the controller HD wallet');
    }
    const { publicKeyToAddress } = await import('knirvwallet-module');
    const publicKey = await keyring.getPublicKey(index);
    const vaultAddress = await publicKeyToAddress(publicKey, 'knirv');
    const dveVault: DVEVault = {
      dveID, dveName,
      vaultAddress,
      derivationIndex: index, teeType, status: 'offline',
      stakeAmount: 0, reputationScore: 0, capabilities: [], attachedPolicies: [],
      badgeNFTIDs: [], supervisorAgentID: '',
      dveURI: `knirv://dve/${dveID}/${teeType}`,
    };
    setDVEVaults(prev => [...prev, dveVault]);
    return dveVault;
  }, [vault, status]);

  const updateDVEVaultStatus = useCallback((dveID: string, updates: Partial<DVEVault>) => {
    setDVEVaults(prev => {
      const next = prev.map(w => w.dveID === dveID ? { ...w, ...updates } : w);
      updateVaultSession({ dveVaults: next });
      return next;
    });
  }, []);

  const removeDVEVault = useCallback((dveID: string) => {
    setDVEVaults(prev => {
      const next = prev.filter(w => w.dveID !== dveID);
      updateVaultSession({ dveVaults: next });
      return next;
    });
  }, []);

  const signTransaction = useCallback(async (tx: any) => {
    if (!vault || status !== 'unlocked') {
      throw new Error('Vault is locked or not initialized');
    }
    return vault.signTransaction(tx);
  }, [vault, status]);

  const signMessage = useCallback(async (message: string | MessageEnvelope): Promise<string> => {
    if (!vault || status !== 'unlocked') {
      throw new Error('Vault is locked or not initialized');
    }
    return vault.signOracleMessage(message);
  }, [vault, status]);

  return {
    vault, status, accounts, currentAccount, oracleAddress, dveVaults, error,
    createVault, importVault, unlockVault, lockVault, clearVault,
    deriveDVEVault, updateDVEVaultStatus, removeDVEVault, signTransaction, signMessage,
  };
};
