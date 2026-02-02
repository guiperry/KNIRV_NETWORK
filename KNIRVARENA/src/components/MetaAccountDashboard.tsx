import React, { useState, useEffect, useCallback } from 'react';
import { View, Text, StyleSheet, TouchableOpacity, ScrollView } from 'react-native';
import { walletIntegrationService, WalletAccount, TransactionRequest, Transaction } from '../services/WalletIntegrationService';

interface MetaAccountConfig {
  chainId: string;
  rpcEndpoint: string;
  walletAddress?: string;
}

interface MetaAccountDashboardProps {
  config: MetaAccountConfig;
  onTransactionSend?: (transaction: Transaction) => void;
  onBalanceUpdate?: (balance: string) => void;
}

export const MetaAccountDashboard: React.FC<MetaAccountDashboardProps> = ({
  config,
  onTransactionSend,
  onBalanceUpdate
}) => {
  const [account, setAccount] = useState<WalletAccount | null>(null);
  const [balance, setBalance] = useState('0.00');
  const [nrnBalance, setNrnBalance] = useState('0.00');
  const [transactions, setTransactions] = useState<Transaction[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [isConnected, setIsConnected] = useState(false);

  const loadAccountData = useCallback(async (walletAccount: WalletAccount) => {
    try {
      // Load balance
      const balanceData = await walletIntegrationService.getAccountBalance(walletAccount.id);
      setBalance(balanceData.balance);
      setNrnBalance(balanceData.nrnBalance);

      if (onBalanceUpdate) {
        onBalanceUpdate(balanceData.balance);
      }

      // Load recent transactions
      const recentTransactions = await walletIntegrationService.getTransactionHistory();
      setTransactions(recentTransactions.slice(0, 10)); // Take only the first 10
    } catch (error) {
      console.error('Failed to load account data:', error);
    }
  }, [onBalanceUpdate]);

  const connectWallet = useCallback(async () => {
    try {
      setIsLoading(true);
      const connectedAccount = await walletIntegrationService.connectWallet();
      setAccount(connectedAccount);
      setIsConnected(true);
      await loadAccountData(connectedAccount);
    } catch (error) {
      console.error('Failed to connect wallet:', error);
    } finally {
      setIsLoading(false);
    }
  }, [loadAccountData]);

  const initializeWallet = useCallback(async () => {
    try {
      setIsLoading(true);

      // Check if wallet is already connected
      const currentAccount = walletIntegrationService.getCurrentAccount();
      if (currentAccount) {
        setAccount(currentAccount);
        setIsConnected(true);
        await loadAccountData(currentAccount);
      } else {
        // Try to connect wallet
        await connectWallet();
      }
    } catch (error) {
      console.error('Failed to initialize wallet:', error);
    } finally {
      setIsLoading(false);
    }
  }, [loadAccountData, connectWallet]);

  useEffect(() => {
    initializeWallet();
  }, [initializeWallet]);

  const handleSendTransaction = async (recipient?: string, amount?: string) => {
    if (!account) {
      console.error('No wallet account connected');
      return;
    }

    try {
      setIsLoading(true);

      const transactionRequest: TransactionRequest = {
        from: account.address,
        to: recipient || 'knirv1example...', // Default recipient for demo
        amount: amount || '25.00', // Default amount for demo
        memo: 'Test transaction from MetaAccountDashboard'
      };

      const transactionId = await walletIntegrationService.createTransaction(transactionRequest);

      // Refresh account data after transaction
      await loadAccountData(account);

      // Create transaction object for callback
      const transaction: Transaction = {
        id: transactionId,
        hash: '',
        from: transactionRequest.from,
        to: transactionRequest.to,
        amount: transactionRequest.amount,
        type: 'send',
        status: 'pending',
        timestamp: new Date()
      };

      onTransactionSend?.(transaction);
      console.log('Transaction sent successfully:', transactionId);
    } catch (error) {
      console.error('Failed to send transaction:', error);
    } finally {
      setIsLoading(false);
    }
  };

  /* const handleDisconnectWallet = async () => {
    try {
      await walletIntegrationService.disconnectWallet();
      setAccount(null);
      setIsConnected(false);
      setBalance('0.00');
      setNrnBalance('0.00');
      setTransactions([]);
    } catch (error) {
      console.error('Failed to disconnect wallet:', error);
    }
  }; */

  if (!isConnected) {
    return (
      <ScrollView style={styles.container} testID="meta-account-dashboard">
        <View style={styles.header}>
          <Text style={styles.title}>Meta Account Dashboard</Text>
          <Text style={styles.chainId}>Chain: {config.chainId}</Text>
        </View>

        <View style={styles.connectionCard}>
          <Text style={styles.connectionLabel}>Wallet Not Connected</Text>
          <Text style={styles.connectionDescription}>
            Connect your wallet to view balance and send transactions
          </Text>
          <TouchableOpacity
            style={[styles.actionButton, isLoading && styles.disabledButton]}
            onPress={connectWallet}
            disabled={isLoading}
            testID="connect-wallet-button"
          >
            <Text style={styles.actionButtonText}>
              {isLoading ? 'Connecting...' : 'Connect Wallet'}
            </Text>
          </TouchableOpacity>
        </View>
      </ScrollView>
    );
  }

  return (
    <ScrollView style={styles.container} testID="meta-account-dashboard">
      <View style={styles.header}>
        <Text style={styles.title}>Meta Account Dashboard</Text>
        <Text style={styles.chainId}>Chain: {config.chainId}</Text>
        {account && (
          <Text style={styles.accountAddress}>
            {account.address.substring(0, 10)}...{account.address.substring(account.address.length - 6)}
          </Text>
        )}
      </View>

      <View style={styles.balanceCard}>
        <Text style={styles.balanceLabel}>KNIRV Balance</Text>
        <Text style={styles.balanceAmount}>{balance} KNIRV</Text>
        <Text style={styles.balanceLabel}>NRN Balance</Text>
        <Text style={styles.balanceAmount}>{nrnBalance} NRN</Text>
      </View>

      <View style={styles.actionsContainer}>
        <TouchableOpacity
          style={[styles.actionButton, isLoading && styles.disabledButton]}
          onPress={() => handleSendTransaction()}
          disabled={isLoading}
          testID="send-transaction-button"
        >
          <Text style={styles.actionButtonText}>
            {isLoading ? 'Sending...' : 'Send Transaction'}
          </Text>
        </TouchableOpacity>
      </View>

      <View style={styles.transactionsContainer}>
        <Text style={styles.sectionTitle}>Recent Transactions</Text>
        {transactions.map((tx) => (
          <View key={tx.id} style={styles.transactionItem} testID={`transaction-${tx.id}`}>
            <Text style={styles.transactionType}>{tx.type.toUpperCase()}</Text>
            <Text style={styles.transactionAmount}>{tx.amount} XION</Text>
            <Text style={styles.transactionAddress}>
              {tx.type === 'send' ? `To: ${tx.to}` : `From: ${tx.from}`}
            </Text>
          </View>
        ))}
      </View>
    </ScrollView>
  );
};

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#1a1a2e',
    padding: 16,
  },
  header: {
    marginBottom: 20,
  },
  title: {
    fontSize: 24,
    fontWeight: 'bold',
    color: '#fff',
    marginBottom: 4,
  },
  chainId: {
    fontSize: 14,
    color: '#888',
  },
  connectionCard: {
    backgroundColor: 'rgba(255, 255, 255, 0.1)',
    borderRadius: 12,
    padding: 20,
    marginBottom: 20,
    alignItems: 'center',
  },
  connectionLabel: {
    fontSize: 16,
    color: '#ccc',
    marginBottom: 8,
    fontWeight: 'bold',
  },
  connectionDescription: {
    fontSize: 14,
    color: '#888',
    textAlign: 'center',
    marginBottom: 16,
  },
  accountAddress: {
    fontSize: 12,
    color: '#666',
    fontFamily: 'monospace',
  },
  balanceCard: {
    backgroundColor: 'rgba(255, 255, 255, 0.1)',
    borderRadius: 12,
    padding: 20,
    marginBottom: 20,
    alignItems: 'center',
  },
  balanceLabel: {
    fontSize: 16,
    color: '#ccc',
    marginBottom: 8,
  },
  balanceAmount: {
    fontSize: 32,
    fontWeight: 'bold',
    color: '#fff',
  },
  actionsContainer: {
    marginBottom: 20,
  },
  actionButton: {
    backgroundColor: '#4CAF50',
    borderRadius: 8,
    padding: 16,
    alignItems: 'center',
  },
  disabledButton: {
    backgroundColor: '#666',
  },
  actionButtonText: {
    color: '#fff',
    fontSize: 16,
    fontWeight: 'bold',
  },
  transactionsContainer: {
    flex: 1,
  },
  sectionTitle: {
    fontSize: 18,
    fontWeight: 'bold',
    color: '#fff',
    marginBottom: 12,
  },
  transactionItem: {
    backgroundColor: 'rgba(255, 255, 255, 0.05)',
    borderRadius: 8,
    padding: 12,
    marginBottom: 8,
  },
  transactionType: {
    fontSize: 14,
    fontWeight: 'bold',
    color: '#4CAF50',
    marginBottom: 4,
  },
  transactionAmount: {
    fontSize: 16,
    fontWeight: 'bold',
    color: '#fff',
    marginBottom: 4,
  },
  transactionAddress: {
    fontSize: 12,
    color: '#888',
    fontFamily: 'monospace',
  },
});

export default MetaAccountDashboard;
