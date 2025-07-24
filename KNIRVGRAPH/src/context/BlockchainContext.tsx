import React, { createContext, useContext, useEffect, useState } from 'react';
import { blockchainApi, Block, Account, Transaction } from '../services/api';

interface BlockchainContextType {
  currentHeight: number;
  isLoading: boolean;
  error: string | null;
  refreshData: () => void;
  getBlock: (height: number) => Promise<Block | null>;
  getAccount: (address: string) => Promise<Account | null>;
  submitTransaction: (tx: Partial<Transaction>) => Promise<boolean>;
}

const BlockchainContext = createContext<BlockchainContextType | undefined>(undefined);

export const useBlockchain = () => {
  const context = useContext(BlockchainContext);
  if (context === undefined) {
    throw new Error('useBlockchain must be used within a BlockchainProvider');
  }
  return context;
};

interface BlockchainProviderProps {
  children: React.ReactNode;
}

export const BlockchainProvider: React.FC<BlockchainProviderProps> = ({ children }) => {
  const [currentHeight, setCurrentHeight] = useState(0);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchCurrentHeight = async () => {
    try {
      setError(null);
      const height = await blockchainApi.getCurrentHeight();
      setCurrentHeight(height);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch blockchain height');
    } finally {
      setIsLoading(false);
    }
  };

  const getBlock = async (height: number): Promise<Block | null> => {
    try {
      return await blockchainApi.getBlock(height);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch block');
      return null;
    }
  };

  const getAccount = async (address: string): Promise<Account | null> => {
    try {
      return await blockchainApi.getAccount(address);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch account');
      return null;
    }
  };

  const submitTransaction = async (tx: Partial<Transaction>): Promise<boolean> => {
    try {
      await blockchainApi.submitTransaction(tx as Transaction);
      return true;
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to submit transaction');
      return false;
    }
  };

  const refreshData = () => {
    setIsLoading(true);
    fetchCurrentHeight();
  };

  useEffect(() => {
    fetchCurrentHeight();
    
    // Auto-refresh every 5 seconds
    const interval = setInterval(fetchCurrentHeight, 5000);
    return () => clearInterval(interval);
  }, []);

  const value: BlockchainContextType = {
    currentHeight,
    isLoading,
    error,
    refreshData,
    getBlock,
    getAccount,
    submitTransaction,
  };

  return (
    <BlockchainContext.Provider value={value}>
      {children}
    </BlockchainContext.Provider>
  );
};