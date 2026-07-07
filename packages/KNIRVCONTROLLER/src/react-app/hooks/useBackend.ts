import { useState, useEffect } from 'react';

// Types from backend_spec.md
export interface Transaction {
  id: string;
  type: 'consumption' | 'reward' | 'transfer';
  amount: number;
  description: string;
  timestamp: string;
  agentName?: string;
  workflowName?: string; // Frontend specific
}

export interface VaultData {
  nrnBalance: number;
  usdValue: number;
  change24h: number;
  vaultAddress: string;
}

export interface UDCData {
  id: string;
  status: 'valid' | 'expired' | 'revoked' | 'pending';
  issuedAt: string;
  expiresAt: string;
  permissions: string[];
}

// Mock Data
const MOCK_VAULT_DATA: VaultData = {
  nrnBalance: 1247.50,
  usdValue: 312.75,
  change24h: 5.2,
  vaultAddress: '0x742d35Cc6aa34567...8B9fA2e1C4D' // This will be replaced by the real vault address
};

const MOCK_TRANSACTIONS: Transaction[] = [
  {
    id: '1',
    type: 'consumption',
    amount: -25,
    description: 'Code Analysis Badge',
    timestamp: new Date(Date.now() - 3600000).toISOString(),
    workflowName: 'CodeT5-Alpha'
  },
  {
    id: '2',
    type: 'reward',
    amount: 50,
    description: 'Task completion bonus',
    timestamp: new Date(Date.now() - 7200000).toISOString(),
    workflowName: 'SEAL-Beta'
  },
  {
    id: '3',
    type: 'transfer',
    amount: 100,
    description: 'Vault funding',
    timestamp: new Date(Date.now() - 86400000).toISOString(),
  }
];

const MOCK_UDC: UDCData = {
  id: 'UDC-7A8B9C2D',
  status: 'valid',
  issuedAt: new Date(Date.now() - 604800000).toISOString(), // 7 days ago
  expiresAt: new Date(Date.now() + 604800000).toISOString(), // 7 days from now
  permissions: ['agent.deploy', 'badge.activate', 'nrn.transfer']
};

export const useBackend = (vaultAddress: string | null) => {
  const [vaultData, setVaultData] = useState<VaultData | null>(null);
  const [transactions, setTransactions] = useState<Transaction[]>([]);
  const [udcData, setUdcData] = useState<UDCData | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchVaultData = async () => {
    setIsLoading(true);
    setError(null);
    try {
      // simulate API delay
      await new Promise(resolve => setTimeout(resolve, 800));
      
      if (vaultAddress) {
        setVaultData({ ...MOCK_VAULT_DATA, vaultAddress });
      } else {
        setVaultData(MOCK_VAULT_DATA);
      }
    } catch (err) {
      setError('Failed to fetch vault data');
    } finally {
      setIsLoading(false);
    }
  };

  const fetchTransactions = async () => {
    try {
      await new Promise(resolve => setTimeout(resolve, 600));
      setTransactions(MOCK_TRANSACTIONS);
    } catch (err) {
      console.error('Failed to fetch transactions');
    }
  };

  const fetchUDC = async () => {
    try {
      await new Promise(resolve => setTimeout(resolve, 500));
      setUdcData(MOCK_UDC);
    } catch (err) {
      console.error('Failed to fetch UDC');
    }
  };

  // Initial fetch
  useEffect(() => {
    if (vaultAddress) {
      fetchVaultData();
      fetchTransactions();
      fetchUDC();
    }
  }, [vaultAddress]);

  const sendNRN = async (toAddress: string, amount: number) => {
    setIsLoading(true);
    try {
      await new Promise(resolve => setTimeout(resolve, 1500));
      // Mock successful transaction
      const newTx: Transaction = {
        id: Math.random().toString(36).substr(2, 9),
        type: 'transfer',
        amount: -amount,
        description: `Transfer to ${toAddress.substring(0, 6)}...`,
        timestamp: new Date().toISOString()
      };
      setTransactions(prev => [newTx, ...prev]);
      setVaultData(prev => prev ? ({ ...prev, nrnBalance: prev.nrnBalance - amount }) : null);
      return { success: true, txHash: '0x' + Math.random().toString(36).substr(2, 40) };
    } catch (err) {
      return { success: false, error: 'Transaction failed' };
    } finally {
      setIsLoading(false);
    }
  };

  return {
    vaultData,
    transactions,
    udcData,
    isLoading,
    error,
    refresh: fetchVaultData,
    sendNRN
  };
};
