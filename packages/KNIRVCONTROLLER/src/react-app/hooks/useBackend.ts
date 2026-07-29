import { useState, useEffect } from 'react';
import { getActiveServerUrl } from '@/react-app/platform/networkConfig';
import { useVault } from './useVault';

export interface Transaction {
  id: string;
  type: 'consumption' | 'reward' | 'transfer';
  amount: number;
  description: string;
  timestamp: string;
  agentName?: string;
  workflowName?: string;
  txHash?: string;
}

export interface VaultData {
  nrnBalance: number;
  usdValue: number;
  change24h: number;
  vaultAddress: string;
  oracleAddress: string;
}

export interface UDCData {
  id: string;
  status: 'valid' | 'expired' | 'revoked' | 'pending';
  issuedAt: string;
  expiresAt: string;
  permissions: string[];
}

export interface TransferResult {
  success: boolean;
  txHash?: string;
  error?: string;
}

const MOCK_VAULT_DATA: VaultData = {
  nrnBalance: 1247.50,
  usdValue: 312.75,
  change24h: 5.2,
  vaultAddress: '',
  oracleAddress: '',
};

const MOCK_TRANSACTIONS: Transaction[] = [
  {
    id: '1',
    type: 'consumption',
    amount: -25,
    description: 'Code Analysis Badge',
    timestamp: new Date(Date.now() - 3600000).toISOString(),
    workflowName: 'CodeT5-Alpha',
  },
  {
    id: '2',
    type: 'reward',
    amount: 50,
    description: 'Task completion bonus',
    timestamp: new Date(Date.now() - 7200000).toISOString(),
    workflowName: 'SEAL-Beta',
  },
  {
    id: '3',
    type: 'transfer',
    amount: 100,
    description: 'Vault funding',
    timestamp: new Date(Date.now() - 86400000).toISOString(),
  },
];

const MOCK_UDC: UDCData = {
  id: 'UDC-7A8B9C2D',
  status: 'valid',
  issuedAt: new Date(Date.now() - 604800000).toISOString(),
  expiresAt: new Date(Date.now() + 604800000).toISOString(),
  permissions: ['agent.deploy', 'badge.activate', 'nrn.transfer'],
};

export const useBackend = () => {
  const { oracleAddress, signMessage, currentAccount } = useVault();
  const [vaultData, setVaultData] = useState<VaultData | null>(null);
  const [transactions, setTransactions] = useState<Transaction[]>([]);
  const [udcData, setUdcData] = useState<UDCData | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const serverUrl = getActiveServerUrl();

  const apiBase = `${serverUrl}/oracle/v3`;

  const fetchBalance = async (address: string): Promise<string | null> => {
    try {
      const resp = await fetch(`${apiBase}/token/balance/${address}`);
      if (!resp.ok) return null;
      const data = await resp.json();
      return data.balance ?? null;
    } catch {
      return null;
    }
  };

  const fetchVaultData = async () => {
    setIsLoading(true);
    setError(null);
    try {
      const addr = oracleAddress;
      if (!addr) {
        setVaultData(MOCK_VAULT_DATA);
        return;
      }
      const balanceStr = await fetchBalance(addr);
      const balance = balanceStr !== null ? parseFloat(balanceStr) : 0;
      setVaultData({
        nrnBalance: balance,
        usdValue: balance * 0.25,
        change24h: 0,
        vaultAddress: addr,
        oracleAddress: oracleAddress ?? addr,
      });
    } catch (err: any) {
      setError(err.message || 'Failed to fetch vault data');
    } finally {
      setIsLoading(false);
    }
  };

  const fetchTransactions = async () => {
    try {
      const addr = oracleAddress;
      if (!addr) {
        setTransactions(MOCK_TRANSACTIONS);
        return;
      }
      const balanceResp = await fetch(`${apiBase}/token/balance/${addr}`)
        .then(r => r.ok ? r.json() : null)
        .catch(() => null);
      const balance = balanceResp?.balance ?? '0';
      const realTxs: Transaction[] = [
        {
          id: `tx-${Date.now()}-1`,
          type: 'transfer',
          amount: parseFloat(balance) || 0,
          description: `Balance for ${addr.slice(0, 8)}...`,
          timestamp: new Date().toISOString(),
          txHash: '',
        },
      ];
      setTransactions(realTxs);
    } catch (err: any) {
      console.error('Failed to fetch transactions:', err);
      setTransactions(MOCK_TRANSACTIONS);
    }
  };

  const fetchUDC = async () => {
    try {
      await new Promise(resolve => setTimeout(resolve, 300));
      setUdcData(MOCK_UDC);
    } catch (err) {
      console.error('Failed to fetch UDC');
    }
  };

  const registerVault = async (): Promise<boolean> => {
    if (!oracleAddress) return false;
    setIsLoading(true);
    setError(null);
    try {
      const resp = await fetch(`${apiBase}/wallet/register`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          address: oracleAddress,
          owner_id: currentAccount?.id || 'controller',
        }),
      });
      if (!resp.ok) {
        const errBody = await resp.text();
        throw new Error(`Registration failed: ${resp.status} ${errBody}`);
      }
      const data = await resp.json();
      if (data.balance !== undefined) {
        setVaultData(prev => prev ? ({
          ...prev,
          nrnBalance: parseFloat(data.balance),
          oracleAddress: data.address ?? oracleAddress,
        }) : null);
      }
      return true;
    } catch (err: any) {
      setError(err.message || 'Failed to register vault');
      return false;
    } finally {
      setIsLoading(false);
    }
  };

   const fetchNonce = async (address?: string): Promise<number> => {
     const addr = address || oracleAddress;
     if (!addr) return 0;
    try {
      const resp = await fetch(`${apiBase}/account/nonce/${addr}`);
      if (!resp.ok) return 0;
      const data = await resp.json();
      return data.nonce ?? 0;
    } catch {
      return 0;
    }
  };

  const sendNRN = async (toAddress: string, amount: number): Promise<TransferResult> => {
    setIsLoading(true);
    setError(null);
    try {
       const fromAddr = oracleAddress;
       if (!fromAddr) {
         return { success: false, error: 'No oracle address available. Unlock your vault first.' };
       }
      const nonce = await fetchNonce(fromAddr);
      const message = `transfer:${fromAddr}:${toAddress}:${amount}:${nonce}`;
      const signature = await signMessage(message);

      const resp = await fetch(`${apiBase}/token/transfer/signed`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          from: fromAddr,
          to: toAddress,
          amount: amount.toString(),
          nonce,
          signature,
        }),
      });

      if (!resp.ok) {
        const errBody = await resp.text();
        throw new Error(`Transfer failed: ${resp.status} ${errBody}`);
      }

      const result = await resp.json();
      const newTx: Transaction = {
        id: Math.random().toString(36).substr(2, 9),
        type: 'transfer',
        amount: -amount,
        description: `Transfer to ${toAddress.slice(0, 8)}...`,
        timestamp: new Date().toISOString(),
        txHash: result.transactionHash,
      };
      setTransactions(prev => [newTx, ...prev]);
      setVaultData(prev => prev ? ({
        ...prev,
        nrnBalance: Math.max(0, prev.nrnBalance - amount),
      }) : null);

      return { success: true, txHash: result.transactionHash };
    } catch (err: any) {
      return { success: false, error: err.message || 'Transaction failed' };
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    fetchVaultData();
    fetchTransactions();
    fetchUDC();
  }, [oracleAddress]);

  return {
    vaultData,
    transactions,
    udcData,
    isLoading,
    error,
    refresh: fetchVaultData,
    registerVault,
    fetchNonce,
    sendNRN,
  };
};
