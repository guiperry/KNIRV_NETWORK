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

  const gateway = ((import.meta as ImportMeta & { env?: Record<string, string> }).env?.VITE_KNIRV_GATEWAY_URL || 'https://gateway.knirv.network').replace(/\/$/, '');
  // Abstraxion configuration
  const abstraxionConfig = {
    chainId: "xion-testnet-2",
    rpcUrl: `${gateway}/xion/rpc`,
    restUrl: `${gateway}/xion/rest`,
    walletUrl: "https://wallet.burnt.com",
    indexerUrl: "https://indexer.burnt.com",
  };

  return (
    <AbstraxionProvider config={abstraxionConfig}>
      {children}
    </AbstraxionProvider>
  );
};
