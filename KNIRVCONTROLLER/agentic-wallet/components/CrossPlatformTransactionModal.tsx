import React, { useState, useEffect } from 'react';
import {
  View,
  Text,
  Modal,
  TouchableOpacity,
  StyleSheet,
  ScrollView,
  Alert,
  ActivityIndicator,
} from 'react-native';
import { LinearGradient } from 'expo-linear-gradient';
import { Ionicons } from '@expo/vector-icons';
import { 
  CrossPlatformTransactionService, 
  TransactionData, 
  PendingTransaction 
} from '../services/CrossPlatformTransactionService';

interface CrossPlatformTransactionModalProps {
  visible: boolean;
  onClose: () => void;
  transactionRequest?: TransactionData;
  sessionId?: string;
  mode: 'sign' | 'view' | 'initiate';
}

export const CrossPlatformTransactionModal: React.FC<CrossPlatformTransactionModalProps> = ({
  visible,
  onClose,
  transactionRequest,
  sessionId,
  mode,
}) => {
  const [isLoading, setIsLoading] = useState(false);
  const [pendingTx, setPendingTx] = useState<PendingTransaction | null>(null);
  const [estimatedFee, setEstimatedFee] = useState<{
    gasLimit: string;
    gasPrice: string;
    estimatedFee: string;
  } | null>(null);

  const txService = CrossPlatformTransactionService.getInstance();

  useEffect(() => {
    if (visible && transactionRequest && mode === 'sign') {
      estimateTransactionFee();
    }
  }, [visible, transactionRequest, mode]);

  const estimateTransactionFee = async () => {
    if (!transactionRequest) return;

    try {
      const fee = await txService.estimateTransactionFee(transactionRequest);
      setEstimatedFee(fee);
    } catch (error) {
      console.error('Failed to estimate fee:', error);
    }
  };

  const handleSignTransaction = async () => {
    if (!transactionRequest || !sessionId) return;

    setIsLoading(true);
    try {
      // First initiate the transaction
      const transactionId = await txService.initiateTransactionFromBrowser(
        sessionId,
        transactionRequest
      );

      // Then sign it (in a real app, this would be done with the user's wallet)
      const walletAddress = transactionRequest.from;
      const signedTx = await txService.signTransactionOnMobile(
        transactionId,
        walletAddress
      );

      Alert.alert(
        'Transaction Signed',
        `Transaction has been signed successfully.\nHash: ${signedTx.hash}`,
        [
          {
            text: 'Broadcast',
            onPress: () => handleBroadcastTransaction(transactionId),
          },
          {
            text: 'Close',
            onPress: onClose,
          },
        ]
      );
    } catch (error) {
      Alert.alert('Error', `Failed to sign transaction: ${error.message}`);
    } finally {
      setIsLoading(false);
    }
  };

  const handleBroadcastTransaction = async (transactionId: string) => {
    setIsLoading(true);
    try {
      const txHash = await txService.broadcastTransaction(transactionId);
      Alert.alert(
        'Transaction Broadcast',
        `Transaction has been broadcast to the network.\nHash: ${txHash}`,
        [{ text: 'OK', onPress: onClose }]
      );
    } catch (error) {
      Alert.alert('Error', `Failed to broadcast transaction: ${error.message}`);
    } finally {
      setIsLoading(false);
    }
  };

  const handleRejectTransaction = () => {
    Alert.alert(
      'Reject Transaction',
      'Are you sure you want to reject this transaction?',
      [
        { text: 'Cancel', style: 'cancel' },
        {
          text: 'Reject',
          style: 'destructive',
          onPress: onClose,
        },
      ]
    );
  };

  const formatAmount = (amount: string, token?: string): string => {
    const num = parseFloat(amount);
    if (num < 0.01) return `${num.toFixed(8)} ${token || 'NRN'}`;
    return `${num.toFixed(4)} ${token || 'NRN'}`;
  };

  const formatAddress = (address: string): string => {
    if (address.length <= 20) return address;
    return `${address.slice(0, 8)}...${address.slice(-8)}`;
  };

  const getStatusColor = (status: string): string => {
    switch (status) {
      case 'pending': return '#F59E0B';
      case 'signed': return '#3B82F6';
      case 'broadcast': return '#8B5CF6';
      case 'confirmed': return '#10B981';
      case 'failed': return '#EF4444';
      default: return '#6B7280';
    }
  };

  const getStatusIcon = (status: string): string => {
    switch (status) {
      case 'pending': return 'time-outline';
      case 'signed': return 'checkmark-circle-outline';
      case 'broadcast': return 'radio-outline';
      case 'confirmed': return 'checkmark-done-circle-outline';
      case 'failed': return 'close-circle-outline';
      default: return 'help-circle-outline';
    }
  };

  const renderTransactionDetails = () => {
    if (!transactionRequest) return null;

    return (
      <View style={styles.detailsContainer}>
        <View style={styles.detailRow}>
          <Text style={styles.detailLabel}>From</Text>
          <Text style={styles.detailValue}>{formatAddress(transactionRequest.from)}</Text>
        </View>

        <View style={styles.detailRow}>
          <Text style={styles.detailLabel}>To</Text>
          <Text style={styles.detailValue}>{formatAddress(transactionRequest.to)}</Text>
        </View>

        <View style={styles.detailRow}>
          <Text style={styles.detailLabel}>Amount</Text>
          <Text style={styles.detailValue}>
            {formatAmount(transactionRequest.amount, transactionRequest.token)}
          </Text>
        </View>

        {transactionRequest.memo && (
          <View style={styles.detailRow}>
            <Text style={styles.detailLabel}>Memo</Text>
            <Text style={styles.detailValue}>{transactionRequest.memo}</Text>
          </View>
        )}

        {estimatedFee && (
          <View style={styles.detailRow}>
            <Text style={styles.detailLabel}>Estimated Fee</Text>
            <Text style={styles.detailValue}>{estimatedFee.estimatedFee} ETH</Text>
          </View>
        )}

        {transactionRequest.gasLimit && (
          <View style={styles.detailRow}>
            <Text style={styles.detailLabel}>Gas Limit</Text>
            <Text style={styles.detailValue}>{transactionRequest.gasLimit}</Text>
          </View>
        )}
      </View>
    );
  };

  const renderPendingTransactionStatus = () => {
    if (!pendingTx) return null;

    return (
      <View style={styles.statusContainer}>
        <View style={[styles.statusBadge, { backgroundColor: getStatusColor(pendingTx.status) }]}>
          <Ionicons 
            name={getStatusIcon(pendingTx.status) as any} 
            size={16} 
            color="#fff" 
          />
          <Text style={styles.statusText}>{pendingTx.status.toUpperCase()}</Text>
        </View>

        {pendingTx.hash && (
          <View style={styles.hashContainer}>
            <Text style={styles.hashLabel}>Transaction Hash:</Text>
            <Text style={styles.hashValue}>{formatAddress(pendingTx.hash)}</Text>
          </View>
        )}

        {pendingTx.error && (
          <View style={styles.errorContainer}>
            <Ionicons name="warning-outline" size={16} color="#EF4444" />
            <Text style={styles.errorText}>{pendingTx.error}</Text>
          </View>
        )}
      </View>
    );
  };

  const renderActions = () => {
    if (mode === 'view') {
      return (
        <TouchableOpacity style={styles.closeButton} onPress={onClose}>
          <Text style={styles.closeButtonText}>Close</Text>
        </TouchableOpacity>
      );
    }

    if (mode === 'sign') {
      return (
        <View style={styles.actionsContainer}>
          <TouchableOpacity 
            style={[styles.actionButton, styles.rejectButton]} 
            onPress={handleRejectTransaction}
            disabled={isLoading}
          >
            <Text style={styles.rejectButtonText}>Reject</Text>
          </TouchableOpacity>

          <TouchableOpacity 
            style={[styles.actionButton, styles.approveButton]} 
            onPress={handleSignTransaction}
            disabled={isLoading}
          >
            {isLoading ? (
              <ActivityIndicator size="small" color="#fff" />
            ) : (
              <Text style={styles.approveButtonText}>Sign & Send</Text>
            )}
          </TouchableOpacity>
        </View>
      );
    }

    return null;
  };

  return (
    <Modal
      visible={visible}
      animationType="slide"
      presentationStyle="pageSheet"
      onRequestClose={onClose}
    >
      <View style={styles.container}>
        <LinearGradient
          colors={['#FF6B35', '#FF8E53']}
          style={styles.header}
        >
          <TouchableOpacity style={styles.closeIcon} onPress={onClose}>
            <Ionicons name="close" size={24} color="#fff" />
          </TouchableOpacity>
          <Text style={styles.headerTitle}>
            {mode === 'sign' ? 'Sign Transaction' : 
             mode === 'view' ? 'Transaction Details' : 
             'Initiate Transaction'}
          </Text>
          <View style={styles.placeholder} />
        </LinearGradient>

        <ScrollView style={styles.content} showsVerticalScrollIndicator={false}>
          <View style={styles.iconContainer}>
            <Ionicons 
              name={mode === 'sign' ? 'create-outline' : 'swap-horizontal-outline'} 
              size={48} 
              color="#FF6B35" 
            />
          </View>

          <Text style={styles.title}>
            {mode === 'sign' ? 'Transaction Approval Required' : 'Cross-Platform Transaction'}
          </Text>
          
          <Text style={styles.subtitle}>
            {mode === 'sign' 
              ? 'Review the transaction details and approve to continue'
              : 'This transaction will be synchronized across your devices'
            }
          </Text>

          {renderTransactionDetails()}
          {renderPendingTransactionStatus()}

          <View style={styles.warningContainer}>
            <Ionicons name="shield-checkmark-outline" size={20} color="#10B981" />
            <Text style={styles.warningText}>
              This transaction is secured by your private key and will be signed locally
            </Text>
          </View>
        </ScrollView>

        <View style={styles.footer}>
          {renderActions()}
        </View>
      </View>
    </Modal>
  );
};

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#fff',
  },
  header: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingTop: 60,
    paddingBottom: 20,
    paddingHorizontal: 20,
  },
  closeIcon: {
    width: 40,
    height: 40,
    borderRadius: 20,
    backgroundColor: 'rgba(255, 255, 255, 0.2)',
    justifyContent: 'center',
    alignItems: 'center',
  },
  headerTitle: {
    fontSize: 18,
    fontWeight: 'bold',
    color: '#fff',
  },
  placeholder: {
    width: 40,
  },
  content: {
    flex: 1,
    padding: 20,
  },
  iconContainer: {
    alignItems: 'center',
    marginBottom: 20,
  },
  title: {
    fontSize: 24,
    fontWeight: 'bold',
    color: '#1F2937',
    textAlign: 'center',
    marginBottom: 8,
  },
  subtitle: {
    fontSize: 16,
    color: '#6B7280',
    textAlign: 'center',
    marginBottom: 32,
    lineHeight: 24,
  },
  detailsContainer: {
    backgroundColor: '#F9FAFB',
    borderRadius: 12,
    padding: 16,
    marginBottom: 24,
  },
  detailRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    paddingVertical: 8,
    borderBottomWidth: 1,
    borderBottomColor: '#E5E7EB',
  },
  detailLabel: {
    fontSize: 14,
    color: '#6B7280',
    fontWeight: '500',
  },
  detailValue: {
    fontSize: 14,
    color: '#1F2937',
    fontWeight: '600',
    flex: 1,
    textAlign: 'right',
  },
  statusContainer: {
    marginBottom: 24,
  },
  statusBadge: {
    flexDirection: 'row',
    alignItems: 'center',
    alignSelf: 'center',
    paddingHorizontal: 12,
    paddingVertical: 6,
    borderRadius: 16,
    marginBottom: 12,
  },
  statusText: {
    color: '#fff',
    fontSize: 12,
    fontWeight: 'bold',
    marginLeft: 4,
  },
  hashContainer: {
    alignItems: 'center',
  },
  hashLabel: {
    fontSize: 12,
    color: '#6B7280',
    marginBottom: 4,
  },
  hashValue: {
    fontSize: 14,
    color: '#1F2937',
    fontFamily: 'monospace',
  },
  errorContainer: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    marginTop: 12,
  },
  errorText: {
    color: '#EF4444',
    fontSize: 14,
    marginLeft: 8,
  },
  warningContainer: {
    flexDirection: 'row',
    alignItems: 'center',
    padding: 12,
    backgroundColor: 'rgba(16, 185, 129, 0.1)',
    borderRadius: 8,
    borderWidth: 1,
    borderColor: '#10B981',
    marginBottom: 24,
  },
  warningText: {
    color: '#10B981',
    fontSize: 12,
    marginLeft: 8,
    flex: 1,
    lineHeight: 16,
  },
  footer: {
    padding: 20,
    paddingBottom: 40,
  },
  actionsContainer: {
    flexDirection: 'row',
    gap: 12,
  },
  actionButton: {
    flex: 1,
    paddingVertical: 16,
    borderRadius: 12,
    alignItems: 'center',
    justifyContent: 'center',
  },
  rejectButton: {
    backgroundColor: '#F3F4F6',
    borderWidth: 1,
    borderColor: '#D1D5DB',
  },
  rejectButtonText: {
    color: '#6B7280',
    fontSize: 16,
    fontWeight: '600',
  },
  approveButton: {
    backgroundColor: '#FF6B35',
  },
  approveButtonText: {
    color: '#fff',
    fontSize: 16,
    fontWeight: '600',
  },
  closeButton: {
    backgroundColor: '#FF6B35',
    paddingVertical: 16,
    borderRadius: 12,
    alignItems: 'center',
  },
  closeButtonText: {
    color: '#fff',
    fontSize: 16,
    fontWeight: '600',
  },
});
