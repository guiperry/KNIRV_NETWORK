import React, { useState, useEffect } from 'react';
import { View, Text, StyleSheet, ScrollView, TouchableOpacity, TextInput, Alert } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { LinearGradient } from 'expo-linear-gradient';
import { Wallet, Plus, RefreshCw, Send, Zap } from 'lucide-react-native';
import { XionMetaAccount, WalletManager, MetaAccountConfig } from '../xion-meta-accounts';
import GlassCard from '../../components/GlassCard';

interface MetaAccountDashboardProps {
  config: MetaAccountConfig;
}

export const MetaAccountDashboard: React.FC<MetaAccountDashboardProps> = ({ config }) => {
  const [walletManager] = useState(new WalletManager(config));
  const [currentWallet, setCurrentWallet] = useState<XionMetaAccount | null>(null);
  const [wallets, setWallets] = useState<string[]>([]);
  const [balance, setBalance] = useState<string>('0');
  const [loading, setLoading] = useState(false);
  const [address, setAddress] = useState<string>('');

  useEffect(() => {
    loadWallets();
  }, []);

  useEffect(() => {
    if (currentWallet) {
      updateBalance();
      updateAddress();
    }
  }, [currentWallet]);

  const loadWallets = async () => {
    const walletList = await walletManager.listWallets();
    setWallets(walletList);

    if (walletList.length > 0) {
      const wallet = await walletManager.getWallet(walletList[0]);
      setCurrentWallet(wallet || null);
    }
  };

  const updateBalance = async () => {
    if (!currentWallet) return;

    try {
      const nrnBalance = await currentWallet.getNRNBalance();
      setBalance(nrnBalance);
    } catch (error) {
      console.error('Error updating balance:', error);
    }
  };

  const updateAddress = async () => {
    if (!currentWallet) return;

    try {
      const walletAddress = await currentWallet.getAddress();
      setAddress(walletAddress);
    } catch (error) {
      console.error('Error getting address:', error);
    }
  };

  const createNewWallet = async () => {
    setLoading(true);
    try {
      const walletName = `wallet_${Date.now()}`;
      const newWallet = await walletManager.createWallet(walletName);
      setCurrentWallet(newWallet);
      await loadWallets();
      Alert.alert('Success', 'New wallet created successfully!');
    } catch (error) {
      console.error('Error creating wallet:', error);
      Alert.alert('Error', 'Failed to create wallet');
    } finally {
      setLoading(false);
    }
  };

  const switchWallet = async (walletName: string) => {
    const wallet = await walletManager.getWallet(walletName);
    setCurrentWallet(wallet || null);
  };

  const requestFromFaucet = async () => {
    if (!currentWallet) return;

    setLoading(true);
    try {
      const txHash = await currentWallet.requestNRNFromFaucet('100'); // Request equivalent of $100 USDC
      console.log('Faucet request transaction:', txHash);
      Alert.alert('Success', `Faucet request submitted! TX: ${txHash.substring(0, 8)}...`);

      // Wait a bit and update balance
      setTimeout(updateBalance, 3000);
    } catch (error) {
      console.error('Error requesting from faucet:', error);
      Alert.alert('Error', 'Failed to request from faucet');
    } finally {
      setLoading(false);
    }
  };

  const enableGasless = async () => {
    if (!currentWallet) return;

    try {
      await currentWallet.enableGaslessTransactions();
      Alert.alert('Success', 'Gasless transactions enabled!');
    } catch (error) {
      console.error('Error enabling gasless transactions:', error);
      Alert.alert('Error', 'Failed to enable gasless transactions');
    }
  };

  return (
    <LinearGradient
      colors={['#0A0A0B', '#1A1A1B', '#0A0A0B']}
      style={styles.container}
    >
      <SafeAreaView style={styles.safeArea}>
        <ScrollView style={styles.scrollView} showsVerticalScrollIndicator={false}>
          {/* Header */}
          <View style={styles.header}>
            <Text style={styles.title}>XION Meta Account</Text>
            <TouchableOpacity 
              style={styles.createButton} 
              onPress={createNewWallet} 
              disabled={loading}
            >
              <Plus size={16} color="#000000" />
              <Text style={styles.createButtonText}>Create</Text>
            </TouchableOpacity>
          </View>

          {currentWallet && (
            <GlassCard style={styles.walletCard}>
              {/* Address Section */}
              <View style={styles.addressSection}>
                <Text style={styles.sectionLabel}>Address</Text>
                <Text style={styles.addressText} numberOfLines={1} ellipsizeMode="middle">
                  {address}
                </Text>
              </View>

              {/* Balance Section */}
              <View style={styles.balanceSection}>
                <Text style={styles.sectionLabel}>NRN Balance</Text>
                <View style={styles.balanceRow}>
                  <Text style={styles.balanceText}>{balance} NRN</Text>
                  <TouchableOpacity 
                    style={styles.refreshButton} 
                    onPress={updateBalance} 
                    disabled={loading}
                  >
                    <RefreshCw size={16} color="#00D2FF" />
                  </TouchableOpacity>
                </View>
              </View>

              {/* Action Buttons */}
              <View style={styles.actionButtons}>
                <TouchableOpacity 
                  style={[styles.actionButton, styles.primaryButton]} 
                  onPress={requestFromFaucet} 
                  disabled={loading}
                >
                  <Send size={16} color="#000000" />
                  <Text style={styles.primaryButtonText}>Request NRN</Text>
                </TouchableOpacity>
                <TouchableOpacity 
                  style={[styles.actionButton, styles.secondaryButton]} 
                  onPress={enableGasless}
                >
                  <Zap size={16} color="#7B68EE" />
                  <Text style={styles.secondaryButtonText}>Enable Gasless</Text>
                </TouchableOpacity>
              </View>
            </GlassCard>
          )}

          {/* Wallet List */}
          <View style={styles.walletList}>
            <Text style={styles.sectionTitle}>Available Wallets</Text>
            {wallets.map((walletName) => (
              <TouchableOpacity
                key={walletName}
                style={[
                  styles.walletItem,
                  currentWallet && walletName === wallets[0] && styles.activeWalletItem
                ]}
                onPress={() => switchWallet(walletName)}
              >
                <Wallet size={20} color="#00D2FF" />
                <Text style={styles.walletName}>{walletName}</Text>
                <Text style={styles.switchText}>Switch</Text>
              </TouchableOpacity>
            ))}
          </View>

          <TransferForm wallet={currentWallet} onTransferComplete={updateBalance} />
          <SkillInvocationForm wallet={currentWallet} onInvocationComplete={updateBalance} />
        </ScrollView>
      </SafeAreaView>
    </LinearGradient>
  );
};

interface TransferFormProps {
  wallet: XionMetaAccount | null;
  onTransferComplete: () => void;
}

const TransferForm: React.FC<TransferFormProps> = ({ wallet, onTransferComplete }) => {
  const [recipient, setRecipient] = useState('');
  const [amount, setAmount] = useState('');
  const [loading, setLoading] = useState(false);

  const handleTransfer = async () => {
    if (!wallet || !recipient || !amount) return;

    setLoading(true);
    try {
      const txHash = await wallet.transferNRN(recipient, amount);
      console.log('Transfer transaction:', txHash);
      Alert.alert('Success', `Transfer submitted! TX: ${txHash.substring(0, 8)}...`);

      setRecipient('');
      setAmount('');
      onTransferComplete();
    } catch (error) {
      console.error('Transfer error:', error);
      Alert.alert('Error', 'Transfer failed');
    } finally {
      setLoading(false);
    }
  };

  return (
    <GlassCard style={styles.formCard}>
      <Text style={styles.formTitle}>Transfer NRN</Text>
      <TextInput
        style={styles.input}
        placeholder="Recipient address"
        placeholderTextColor="#666666"
        value={recipient}
        onChangeText={setRecipient}
      />
      <TextInput
        style={styles.input}
        placeholder="Amount"
        placeholderTextColor="#666666"
        value={amount}
        onChangeText={setAmount}
        keyboardType="numeric"
      />
      <TouchableOpacity
        style={[styles.formButton, (!wallet || !recipient || !amount || loading) && styles.disabledButton]}
        onPress={handleTransfer}
        disabled={!wallet || !recipient || !amount || loading}
      >
        <Text style={styles.formButtonText}>Transfer</Text>
      </TouchableOpacity>
    </GlassCard>
  );
};

interface SkillInvocationFormProps {
  wallet: XionMetaAccount | null;
  onInvocationComplete: () => void;
}

const SkillInvocationForm: React.FC<SkillInvocationFormProps> = ({ wallet, onInvocationComplete }) => {
  const [skillId, setSkillId] = useState('');
  const [amount, setAmount] = useState('');
  const [loading, setLoading] = useState(false);

  const handleInvocation = async () => {
    if (!wallet || !skillId || !amount) return;

    setLoading(true);
    try {
      const txHash = await wallet.burnNRNForSkill(skillId, amount);
      console.log('Skill invocation transaction:', txHash);
      Alert.alert('Success', `Skill invoked! TX: ${txHash.substring(0, 8)}...`);

      setSkillId('');
      setAmount('');
      onInvocationComplete();
    } catch (error) {
      console.error('Skill invocation error:', error);
      Alert.alert('Error', 'Skill invocation failed');
    } finally {
      setLoading(false);
    }
  };

  return (
    <GlassCard style={styles.formCard}>
      <Text style={styles.formTitle}>Invoke Skill</Text>
      <TextInput
        style={styles.input}
        placeholder="Skill ID"
        placeholderTextColor="#666666"
        value={skillId}
        onChangeText={setSkillId}
      />
      <TextInput
        style={styles.input}
        placeholder="NRN Amount to burn"
        placeholderTextColor="#666666"
        value={amount}
        onChangeText={setAmount}
        keyboardType="numeric"
      />
      <TouchableOpacity
        style={[styles.formButton, (!wallet || !skillId || !amount || loading) && styles.disabledButton]}
        onPress={handleInvocation}
        disabled={!wallet || !skillId || !amount || loading}
      >
        <Text style={styles.formButtonText}>Invoke Skill</Text>
      </TouchableOpacity>
    </GlassCard>
  );
};

const styles = StyleSheet.create({
  container: {
    flex: 1,
  },
  safeArea: {
    flex: 1,
  },
  scrollView: {
    flex: 1,
    paddingHorizontal: 20,
  },
  header: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    paddingTop: 20,
    marginBottom: 24,
  },
  title: {
    fontSize: 28,
    fontFamily: 'Inter-Bold',
    color: '#FFFFFF',
  },
  createButton: {
    flexDirection: 'row',
    alignItems: 'center',
    backgroundColor: '#00D2FF',
    paddingHorizontal: 16,
    paddingVertical: 8,
    borderRadius: 12,
    gap: 6,
  },
  createButtonText: {
    fontSize: 14,
    fontFamily: 'Inter-SemiBold',
    color: '#000000',
  },
  walletCard: {
    marginBottom: 24,
  },
  addressSection: {
    marginBottom: 16,
  },
  sectionLabel: {
    fontSize: 12,
    fontFamily: 'Inter-Medium',
    color: '#999999',
    marginBottom: 4,
  },
  addressText: {
    fontSize: 14,
    fontFamily: 'Inter-Regular',
    color: '#FFFFFF',
    backgroundColor: 'rgba(255, 255, 255, 0.1)',
    padding: 8,
    borderRadius: 8,
  },
  balanceSection: {
    marginBottom: 20,
  },
  balanceRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
  },
  balanceText: {
    fontSize: 24,
    fontFamily: 'Inter-Bold',
    color: '#00FF88',
  },
  refreshButton: {
    padding: 8,
    borderRadius: 8,
    backgroundColor: 'rgba(0, 210, 255, 0.2)',
  },
  actionButtons: {
    flexDirection: 'row',
    gap: 12,
  },
  actionButton: {
    flex: 1,
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    paddingVertical: 12,
    borderRadius: 12,
    gap: 8,
  },
  primaryButton: {
    backgroundColor: '#00D2FF',
  },
  secondaryButton: {
    backgroundColor: 'transparent',
    borderWidth: 1,
    borderColor: 'rgba(255, 255, 255, 0.2)',
  },
  primaryButtonText: {
    fontSize: 14,
    fontFamily: 'Inter-SemiBold',
    color: '#000000',
  },
  secondaryButtonText: {
    fontSize: 14,
    fontFamily: 'Inter-SemiBold',
    color: '#FFFFFF',
  },
  walletList: {
    marginBottom: 24,
  },
  sectionTitle: {
    fontSize: 18,
    fontFamily: 'Inter-SemiBold',
    color: '#FFFFFF',
    marginBottom: 16,
  },
  walletItem: {
    flexDirection: 'row',
    alignItems: 'center',
    backgroundColor: 'rgba(255, 255, 255, 0.05)',
    padding: 16,
    borderRadius: 12,
    marginBottom: 8,
    gap: 12,
  },
  activeWalletItem: {
    backgroundColor: 'rgba(0, 210, 255, 0.1)',
    borderWidth: 1,
    borderColor: 'rgba(0, 210, 255, 0.3)',
  },
  walletName: {
    flex: 1,
    fontSize: 14,
    fontFamily: 'Inter-Medium',
    color: '#FFFFFF',
  },
  switchText: {
    fontSize: 12,
    fontFamily: 'Inter-Medium',
    color: '#00D2FF',
  },
  formCard: {
    marginBottom: 20,
  },
  formTitle: {
    fontSize: 16,
    fontFamily: 'Inter-SemiBold',
    color: '#FFFFFF',
    marginBottom: 16,
  },
  input: {
    backgroundColor: 'rgba(255, 255, 255, 0.1)',
    borderRadius: 8,
    padding: 12,
    fontSize: 14,
    fontFamily: 'Inter-Regular',
    color: '#FFFFFF',
    marginBottom: 12,
  },
  formButton: {
    backgroundColor: '#00D2FF',
    paddingVertical: 12,
    borderRadius: 8,
    alignItems: 'center',
  },
  disabledButton: {
    backgroundColor: 'rgba(255, 255, 255, 0.1)',
  },
  formButtonText: {
    fontSize: 14,
    fontFamily: 'Inter-SemiBold',
    color: '#000000',
  },
});
