import React, { createContext, useState, useEffect, useContext, useCallback } from 'react';
   import api from '../utils/api';
   import { useBackend } from './BackendContext';

   const BlockchainContext = createContext();

   export const BlockchainProvider = ({ children }) => {
     const { isRunning } = useBackend();
     const [blocks, setBlocks] = useState([]);
     const [transactions, setTransactions] = useState([]);
     const [walletInfo, setWalletInfo] = useState({});
     const [devs, setPeers] = useState([]);
     const [isLoading, setIsLoading] = useState(true);
     const [error, setError] = useState(null); // New error state

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
         throw new Error('Failed to fetch chain data');
       }
     }, []);

     const fetchTransactionPool = useCallback(async () => {
       try {
         const response = await api.get('/txn_pool');
         setTransactions(response.data || []);
       } catch (err) {
         console.error('Error fetching transaction pool:', err);
         throw new Error('Failed to fetch transaction pool');
       }
     }, []);

     const fetchWalletInfo = useCallback(async () => {
       try {
         const response = await api.get('/wallet/info');
         setWalletInfo(response.data || {}); // Ensure it's always an object
       } catch (err) {
         console.error('Error fetching wallet info:', err);
         throw new Error('Failed to fetch wallet info');
       }
     }, []);

     const fetchPeers = useCallback(async () => {
       try {
         const response = await api.get('/devs');
         setPeers(response.data || []);
       } catch (err) {
         console.error('Error fetching devs:', err);
         throw new Error('Failed to fetch devs');
       }
     }, []);

     // Main data fetching function
     const fetchBlockchainData = useCallback(async () => {
       setIsLoading(true);
       setError(null); // Clear previous errors
       try {
         await Promise.all([
           fetchChain(),
           fetchTransactionPool(),
           fetchWalletInfo(),
           fetchPeers(),
         ]);
       } catch (err) {
         console.error('Error fetching blockchain data:', err);
         setError(err.message || 'An unexpected error occurred while fetching blockchain data.');
       } finally {
         setIsLoading(false);
       }
     }, [fetchChain, fetchTransactionPool, fetchWalletInfo, fetchPeers]); // Dependencies are stable

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
     }, [fetchTransactionPool, fetchWalletInfo]); // Added dependencies

     return (
       <BlockchainContext.Provider
         value={{
           blocks,
           transactions,
           walletInfo,
           devs,
           isLoading,
           error, // Expose error state
           createTransaction,
           refreshData: fetchBlockchainData,
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