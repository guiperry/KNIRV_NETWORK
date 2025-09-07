/**
 * Abstraxion Wallet Provider Component
 * React provider for Abstraxion wallet integration
 */

import React from 'react';
import { AbstraxionProvider } from '@burnt-labs/abstraxion';

interface AbstraxionWalletProviderProps {
  children: React.ReactNode;
}

export const AbstraxionWalletProvider: React.FC<AbstraxionWalletProviderProps> = ({ children }) => {
  // Abstraxion configuration
  const abstraxionConfig = {
    rpcUrl: "https://rpc.xion-testnet-1.burnt.com:443",
    restUrl: "https://api.xion-testnet-1.burnt.com",
    walletUrl: "https://wallet.burnt.com",
    indexerUrl: "https://indexer.burnt.com",
  };

  return (
    <AbstraxionProvider config={abstraxionConfig}>
      {children}
    </AbstraxionProvider>
  );
};
