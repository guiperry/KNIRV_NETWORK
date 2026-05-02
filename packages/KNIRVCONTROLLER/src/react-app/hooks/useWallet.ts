import { useVault } from './useVault';

export const useWallet = () => {
  const vault = useVault();

  const createWallet = async (password: string) => {
    return await vault.createVault(password);
  };

  const importWallet = async (mnemonic: string, password: string) => {
    return await vault.importVault(mnemonic, password);
  };

  return {
    currentAccount: vault.currentAccount,
    status: vault.status,
    createWallet,
    importWallet,
    unlockVault: vault.unlockVault,
    lockVault: vault.lockVault,
    accounts: vault.accounts,
  };
};
