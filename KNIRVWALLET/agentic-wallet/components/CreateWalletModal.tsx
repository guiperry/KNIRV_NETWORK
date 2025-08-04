import React, { useState } from 'react';
import {
  View,
  Text,
  Modal,
  TouchableOpacity,
  StyleSheet,
  TextInput,
  ScrollView,
  Alert,
  ActivityIndicator,
} from 'react-native';
import { LinearGradient } from 'expo-linear-gradient';
import { Ionicons } from '@expo/vector-icons';

interface SupportedChain {
  symbol: string;
  name: string;
  network: string;
  derivation: string;
  isTestnet: boolean;
}

interface CreateWalletModalProps {
  visible: boolean;
  onClose: () => void;
  onCreateWallet: (walletData: {
    name: string;
    chains: string[];
    mnemonic?: string;
  }) => Promise<void>;
  supportedChains: SupportedChain[];
}

export const CreateWalletModal: React.FC<CreateWalletModalProps> = ({
  visible,
  onClose,
  onCreateWallet,
  supportedChains,
}) => {
  const [walletName, setWalletName] = useState('');
  const [selectedChains, setSelectedChains] = useState<string[]>([]);
  const [customMnemonic, setCustomMnemonic] = useState('');
  const [useCustomMnemonic, setUseCustomMnemonic] = useState(false);
  const [isCreating, setIsCreating] = useState(false);

  const resetForm = () => {
    setWalletName('');
    setSelectedChains([]);
    setCustomMnemonic('');
    setUseCustomMnemonic(false);
    setIsCreating(false);
  };

  const handleClose = () => {
    if (!isCreating) {
      resetForm();
      onClose();
    }
  };

  const toggleChainSelection = (chainSymbol: string) => {
    setSelectedChains(prev => {
      if (prev.includes(chainSymbol)) {
        return prev.filter(symbol => symbol !== chainSymbol);
      } else {
        return [...prev, chainSymbol];
      }
    });
  };

  const selectAllChains = () => {
    if (selectedChains.length === supportedChains.length) {
      setSelectedChains([]);
    } else {
      setSelectedChains(supportedChains.map(chain => chain.symbol));
    }
  };

  const handleCreateWallet = async () => {
    if (!walletName.trim()) {
      Alert.alert('Error', 'Please enter a wallet name');
      return;
    }

    if (selectedChains.length === 0) {
      Alert.alert('Error', 'Please select at least one blockchain');
      return;
    }

    if (useCustomMnemonic && !customMnemonic.trim()) {
      Alert.alert('Error', 'Please enter a mnemonic phrase or disable custom mnemonic');
      return;
    }

    setIsCreating(true);

    try {
      await onCreateWallet({
        name: walletName.trim(),
        chains: selectedChains,
        mnemonic: useCustomMnemonic ? customMnemonic.trim() : undefined,
      });
      
      resetForm();
      onClose();
    } catch (error) {
      Alert.alert('Error', 'Failed to create wallet. Please try again.');
    } finally {
      setIsCreating(false);
    }
  };

  const renderChainItem = (chain: SupportedChain) => {
    const isSelected = selectedChains.includes(chain.symbol);

    return (
      <TouchableOpacity
        key={chain.symbol}
        style={[styles.chainItem, isSelected && styles.chainItemSelected]}
        onPress={() => toggleChainSelection(chain.symbol)}
        disabled={isCreating}
      >
        <View style={styles.chainInfo}>
          <View style={[styles.chainIcon, isSelected && styles.chainIconSelected]}>
            <Text style={[styles.chainSymbol, isSelected && styles.chainSymbolSelected]}>
              {chain.symbol}
            </Text>
          </View>
          <View style={styles.chainDetails}>
            <Text style={styles.chainName}>{chain.name}</Text>
            <Text style={styles.chainNetwork}>{chain.network}</Text>
          </View>
        </View>
        <View style={[styles.checkbox, isSelected && styles.checkboxSelected]}>
          {isSelected && <Ionicons name="checkmark" size={16} color="#fff" />}
        </View>
      </TouchableOpacity>
    );
  };

  return (
    <Modal
      visible={visible}
      animationType="slide"
      presentationStyle="pageSheet"
      onRequestClose={handleClose}
    >
      <View style={styles.container}>
        <LinearGradient
          colors={['#FF6B35', '#FF8E53']}
          style={styles.header}
        >
          <TouchableOpacity
            style={styles.closeButton}
            onPress={handleClose}
            disabled={isCreating}
          >
            <Ionicons name="close" size={24} color="#fff" />
          </TouchableOpacity>
          <Text style={styles.headerTitle}>Create Multi-Chain Wallet</Text>
          <View style={styles.placeholder} />
        </LinearGradient>

        <ScrollView style={styles.content} showsVerticalScrollIndicator={false}>
          <View style={styles.section}>
            <Text style={styles.sectionTitle}>Wallet Name</Text>
            <TextInput
              style={styles.textInput}
              placeholder="Enter wallet name"
              value={walletName}
              onChangeText={setWalletName}
              editable={!isCreating}
            />
          </View>

          <View style={styles.section}>
            <View style={styles.sectionHeader}>
              <Text style={styles.sectionTitle}>Select Blockchains</Text>
              <TouchableOpacity
                style={styles.selectAllButton}
                onPress={selectAllChains}
                disabled={isCreating}
              >
                <Text style={styles.selectAllText}>
                  {selectedChains.length === supportedChains.length ? 'Deselect All' : 'Select All'}
                </Text>
              </TouchableOpacity>
            </View>
            <View style={styles.chainsList}>
              {supportedChains.map(renderChainItem)}
            </View>
          </View>

          <View style={styles.section}>
            <View style={styles.mnemonicHeader}>
              <Text style={styles.sectionTitle}>Mnemonic Phrase</Text>
              <TouchableOpacity
                style={styles.toggleButton}
                onPress={() => setUseCustomMnemonic(!useCustomMnemonic)}
                disabled={isCreating}
              >
                <Text style={styles.toggleText}>
                  {useCustomMnemonic ? 'Use Generated' : 'Use Custom'}
                </Text>
              </TouchableOpacity>
            </View>
            
            {useCustomMnemonic ? (
              <TextInput
                style={[styles.textInput, styles.mnemonicInput]}
                placeholder="Enter your 12 or 24 word mnemonic phrase"
                value={customMnemonic}
                onChangeText={setCustomMnemonic}
                multiline
                numberOfLines={3}
                editable={!isCreating}
              />
            ) : (
              <View style={styles.generatedMnemonicInfo}>
                <Ionicons name="information-circle" size={20} color="#6B7280" />
                <Text style={styles.infoText}>
                  A new 12-word mnemonic phrase will be generated for you
                </Text>
              </View>
            )}
          </View>
        </ScrollView>

        <View style={styles.footer}>
          <TouchableOpacity
            style={[styles.createButton, isCreating && styles.createButtonDisabled]}
            onPress={handleCreateWallet}
            disabled={isCreating}
          >
            {isCreating ? (
              <ActivityIndicator size="small" color="#fff" />
            ) : (
              <Text style={styles.createButtonText}>Create Wallet</Text>
            )}
          </TouchableOpacity>
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
  closeButton: {
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
  section: {
    marginBottom: 24,
  },
  sectionHeader: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: 12,
  },
  sectionTitle: {
    fontSize: 16,
    fontWeight: '600',
    color: '#1F2937',
  },
  selectAllButton: {
    paddingVertical: 4,
    paddingHorizontal: 8,
  },
  selectAllText: {
    fontSize: 14,
    color: '#FF6B35',
    fontWeight: '500',
  },
  textInput: {
    borderWidth: 1,
    borderColor: '#D1D5DB',
    borderRadius: 8,
    padding: 12,
    fontSize: 16,
    backgroundColor: '#F9FAFB',
  },
  chainsList: {
    gap: 8,
  },
  chainItem: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    padding: 12,
    borderRadius: 8,
    borderWidth: 1,
    borderColor: '#E5E7EB',
    backgroundColor: '#F9FAFB',
  },
  chainItemSelected: {
    borderColor: '#FF6B35',
    backgroundColor: 'rgba(255, 107, 53, 0.05)',
  },
  chainInfo: {
    flexDirection: 'row',
    alignItems: 'center',
    flex: 1,
  },
  chainIcon: {
    width: 40,
    height: 40,
    borderRadius: 20,
    backgroundColor: '#E5E7EB',
    justifyContent: 'center',
    alignItems: 'center',
    marginRight: 12,
  },
  chainIconSelected: {
    backgroundColor: '#FF6B35',
  },
  chainSymbol: {
    fontSize: 12,
    fontWeight: 'bold',
    color: '#6B7280',
  },
  chainSymbolSelected: {
    color: '#fff',
  },
  chainDetails: {
    flex: 1,
  },
  chainName: {
    fontSize: 14,
    fontWeight: '500',
    color: '#1F2937',
  },
  chainNetwork: {
    fontSize: 12,
    color: '#6B7280',
    marginTop: 2,
  },
  checkbox: {
    width: 20,
    height: 20,
    borderRadius: 10,
    borderWidth: 2,
    borderColor: '#D1D5DB',
    justifyContent: 'center',
    alignItems: 'center',
  },
  checkboxSelected: {
    backgroundColor: '#FF6B35',
    borderColor: '#FF6B35',
  },
  mnemonicHeader: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: 12,
  },
  toggleButton: {
    paddingVertical: 4,
    paddingHorizontal: 8,
  },
  toggleText: {
    fontSize: 14,
    color: '#FF6B35',
    fontWeight: '500',
  },
  mnemonicInput: {
    height: 80,
    textAlignVertical: 'top',
  },
  generatedMnemonicInfo: {
    flexDirection: 'row',
    alignItems: 'center',
    padding: 12,
    backgroundColor: '#F3F4F6',
    borderRadius: 8,
  },
  infoText: {
    fontSize: 14,
    color: '#6B7280',
    marginLeft: 8,
    flex: 1,
  },
  footer: {
    padding: 20,
    paddingBottom: 40,
  },
  createButton: {
    backgroundColor: '#FF6B35',
    borderRadius: 12,
    paddingVertical: 16,
    alignItems: 'center',
    justifyContent: 'center',
  },
  createButtonDisabled: {
    opacity: 0.6,
  },
  createButtonText: {
    fontSize: 16,
    fontWeight: 'bold',
    color: '#fff',
  },
});
