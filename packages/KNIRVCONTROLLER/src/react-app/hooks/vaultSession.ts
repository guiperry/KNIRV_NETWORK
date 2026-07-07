import type { DVEVault } from '@/shared/types';

export type VaultStatus = 'initializing' | 'no_vault' | 'locked' | 'unlocked';

export interface VaultSession {
  vault: any | null;
  status: VaultStatus;
  accounts: any[];
  currentAccount: any | null;
  dveVaults: DVEVault[];
  error: string | null;
}

const createInitialSession = (): VaultSession => ({
  vault: null,
  status: 'initializing',
  accounts: [],
  currentAccount: null,
  dveVaults: [],
  error: null,
});

let sharedVaultSession: VaultSession = createInitialSession();

export const getVaultSession = () => sharedVaultSession;

export const updateVaultSession = (updates: Partial<VaultSession>) => {
  sharedVaultSession = {
    ...sharedVaultSession,
    ...updates,
  };

  return sharedVaultSession;
};

export const resetVaultSession = () => {
  sharedVaultSession = createInitialSession();
  return sharedVaultSession;
};
