import React, { createContext, useState, useEffect, useContext, useCallback } from 'react';
import api from '../utils/api';
import { useBackend } from './BackendContext';

const BlockchainContext = createContext();

export const BlockchainProvider = ({ children }) => {
  const { isRunning } = useBackend();
  const [blocks, setBlocks] = useState([]);
  const [transactions, setTransactions] = useState([]);
  const [walletInfo, setWalletInfo] = useState({});
  const [walletConnected, setWalletConnected] = useState(false);
  const [devs, setPeers] = useState([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState(null); // New error state

  // Check whether a wallet is connected via QR code session
  const checkWalletConnection = useCallback(async () => {
    try {
      const response = await fetch('/session/controller', {
        credentials: 'include',
      });
      if (response.ok) {
        const data = await response.json();
        const connected = !!(data && data.controllerUrl);
        setWalletConnected(connected);
        return connected;
      }
    } catch (_) {
      // Session endpoint unavailable — no wallet connected
    }
    setWalletConnected(false);
    return false;
  }, []);

  // Individual fetch functions (memoized)
  const fetchChain = useCallback(async () => {
    try {
      const response = await api.get('/chain');
      if (response.data && response.data.blocks) {
        setBlocks(response.data.blocks);
      } else {
        setBlocks([]); // Ensure it's always an array
      }
    } catch (err) {
      console.error('Error fetching blockchain:', err);
      setBlocks([]);
    }
  }, []);

  const fetchTransactionPool = useCallback(async () => {
    try {
      const response = await api.get('/txn_pool');
      setTransactions(response.data || []);
    } catch (err) {
      console.error('Error fetching transaction pool:', err);
      setTransactions([]);
    }
  }, []);

  const fetchWalletInfo = useCallback(async () => {
    try {
      const response = await api.get('/wallet/info');
      setWalletInfo(response.data || {}); // Ensure it's always an object
    } catch (err) {
      console.error('Error fetching wallet info:', err);
      setWalletInfo({});
    }
  }, []);

  const fetchPeers = useCallback(async () => {
    try {
      const response = await api.get('/devs');
      setPeers(response.data || []);
    } catch (err) {
      console.error('Error fetching devs:', err);
      setPeers([]);
    }
  }, []);

  // Main data fetching function — fetches blockchain overview data only.
  // Wallet info is NOT fetched here; call refreshWalletInfo() explicitly
  // after the user connects via QR code.
  const fetchBlockchainData = useCallback(async () => {
    setIsLoading(true);
    setError(null); // Clear previous errors
    try {
      await Promise.all([
        fetchChain(),
        fetchTransactionPool(),
        fetchPeers(),
        checkWalletConnection(),
      ]);
    } catch (_err) {
      // Silently handle — individual fetch functions already set empty state
    } finally {
      setIsLoading(false);
    }
  }, [fetchChain, fetchTransactionPool, fetchPeers, checkWalletConnection]);

  // Refresh wallet info after QR code connection — call this from
  // the QR connect page or wallet page after the user connects.
  const refreshWalletInfo = useCallback(async () => {
    await Promise.all([
      fetchWalletInfo(),
      checkWalletConnection(),
    ]);
  }, [fetchWalletInfo, checkWalletConnection]);

  useEffect(() => {
    if (isRunning) {
      fetchBlockchainData(); // Initial fetch
      const interval = setInterval(fetchBlockchainData, 10000); // Polling
      return () => clearInterval(interval);
    } else {
      // Clear data and reset states if backend is not running
      setBlocks([]);
      setTransactions([]);
      setWalletInfo({});
      setPeers([]);
      setIsLoading(false);
      setError(null);
      setWalletConnected(false);
    }
  }, [isRunning, fetchBlockchainData]);

  const createTransaction = useCallback(async (recipient, amount) => {
    try {
      const response = await api.post('/transaction', {
        recipient,
        amount: parseInt(amount),
      });
      // Re-fetch relevant data after successful transaction
      await Promise.all([fetchTransactionPool(), fetchWalletInfo()]);
      return response.data;
    } catch (error) {
      console.error('Error creating transaction:', error);
      throw error;
    }
  }, [fetchTransactionPool, fetchWalletInfo]);

  return (
    <BlockchainContext.Provider
      value={{
        blocks,
        transactions,
        walletInfo,
        walletConnected,
        devs,
        isLoading,
        error, // Expose error state
        createTransaction,
        refreshData: fetchBlockchainData,
        refreshWalletInfo, // Call after QR code connection
        checkWalletConnection,
      }}
    >
      {children}
    </BlockchainContext.Provider>
  );
};

export const useBlockchain = () => {
  const context = useContext(BlockchainContext);
  if (context === undefined) {
    throw new Error('useBlockchain must be used within a BlockchainProvider');
  }
  return context;
};
