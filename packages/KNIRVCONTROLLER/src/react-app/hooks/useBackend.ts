import { useState, useEffect } from 'react';
import { getGatewayCandidates } from '@/react-app/platform/networkConfig';
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

export const useBackend = () => {
  const { oracleAddress, signMessage, currentAccount } = useVault();
  const [vaultData, setVaultData] = useState<VaultData | null>(null);
  const [transactions, setTransactions] = useState<Transaction[]>([]);
  const [udcData, setUdcData] = useState<UDCData | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

	const requestGateway = async (path: string, init?: RequestInit): Promise<Response> => {
		const failures: string[] = [];
		for (const gateway of getGatewayCandidates()) {
			try {
				const response = await fetch(`${gateway}${path}`, init);
				if (response.status < 500) return response;
				failures.push(`${gateway}: HTTP ${response.status}`);
			} catch (error) {
				failures.push(`${gateway}: ${error instanceof Error ? error.message : 'unreachable'}`);
			}
		}
		throw new Error(`No KNIRV gateway is available (${failures.join('; ')})`);
	};

  const fetchBalance = async (address: string): Promise<string | null> => {
    try {
			const resp = await requestGateway(`/oracle/v3/token/balance/${address}`);
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
				setVaultData(null);
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
		// KNIRVORACLE does not yet expose an address-history endpoint. An empty
		// list is truthful; balance records must not be fabricated as transfers.
		setTransactions([]);
	};

	const fetchUDC = async () => {
		setUdcData(null);
	};

  const registerVault = async (): Promise<boolean> => {
    if (!oracleAddress) return false;
    setIsLoading(true);
    setError(null);
    try {
      const ownerID = currentAccount?.id || 'controller';
      const now = Math.floor(Date.now() / 1000);
      const signature = await signMessage({
        domain: 'knirv.oracle',
        purpose: 'wallet-registration',
        chainId: 'knirv-1',
        nonce: `${ownerID}:${now}`,
        issuedAtUnix: now,
        expiresAtUnix: now + 300,
        payload: new TextEncoder().encode(`register:${oracleAddress}:${ownerID}`),
      });
			const resp = await requestGateway('/oracle/v3/wallet/register', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          address: oracleAddress,
          owner_id: ownerID,
          signature,
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
		const resp = await requestGateway(`/oracle/v3/account/nonce/${addr}`);
		if (!resp.ok) throw new Error(`Nonce lookup failed with HTTP ${resp.status}`);
      const data = await resp.json();
      return data.nonce ?? 0;
    } catch {
		throw new Error('Unable to load the wallet nonce');
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
      const now = Math.floor(Date.now() / 1000);
      const signature = await signMessage({
        domain: 'knirv.oracle',
        purpose: 'oracle-transfer',
        chainId: 'knirv-1',
        nonce: String(nonce),
        issuedAtUnix: now,
        expiresAtUnix: now + 300,
        payload: new TextEncoder().encode(message),
      });

			const resp = await requestGateway('/oracle/v3/token/transfer/signed', {
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
