import React, { useState } from 'react';
import {
  View,
  Text,
  TouchableOpacity,
  StyleSheet,
  ActivityIndicator,
} from 'react-native';
import { LinearGradient } from 'expo-linear-gradient';
import { Ionicons } from '@expo/vector-icons';

interface Chain {
  symbol: string;
  name: string;
  network: string;
  balance?: string;
  usdValue?: string;
  icon?: string;
}

interface MultichainWalletCardProps {
  chains: Chain[];
  onChainPress: (chain: Chain) => void;
  onAddChain: () => void;
  isLoading?: boolean;
}

const CHAIN_ICONS: { [key: string]: string } = {
  BTC: '₿',
  ETH: 'Ξ',
  SOL: '◎',
  LTC: 'Ł',
  DOGE: 'Ð',
  BCH: '₿',
  DASH: 'Đ',
  ETC: 'Ξ',
  NRN: '🔥',
};

const CHAIN_COLORS: { [key: string]: readonly [string, string] } = {
  BTC: ['#F7931A', '#FFB84D'],
  ETH: ['#627EEA', '#8FA2F7'],
  SOL: ['#9945FF', '#14F195'],
  LTC: ['#BFBBBB', '#E8E8E8'],
  DOGE: ['#C2A633', '#F4D03F'],
  BCH: ['#8DC351', '#B8E986'],
  DASH: ['#008CE7', '#4DB8FF'],
  ETC: ['#328332', '#66B366'],
  NRN: ['#FF6B35', '#FF8E53'],
};

export const MultichainWalletCard: React.FC<MultichainWalletCardProps> = ({
  chains,
  onChainPress,
  onAddChain,
  isLoading = false,
}) => {
  const [expandedChain, setExpandedChain] = useState<string | null>(null);

  const formatBalance = (balance: string | undefined): string => {
    if (!balance) return '0.00';
    const num = parseFloat(balance);
    if (num < 0.01) return num.toFixed(8);
    return num.toFixed(4);
  };

  const formatUSDValue = (usdValue: string | undefined): string => {
    if (!usdValue) return '$0.00';
    const num = parseFloat(usdValue);
    return `$${num.toFixed(2)}`;
  };

  const getTotalUSDValue = (): string => {
    const total = chains.reduce((sum, chain) => {
      return sum + (parseFloat(chain.usdValue || '0'));
    }, 0);
    return `$${total.toFixed(2)}`;
  };

  const handleChainPress = (chain: Chain) => {
    if (expandedChain === chain.symbol) {
      setExpandedChain(null);
    } else {
      setExpandedChain(chain.symbol);
    }
    onChainPress(chain);
  };

  const renderChainItem = (chain: Chain) => {
    const isExpanded = expandedChain === chain.symbol;
    const colors = CHAIN_COLORS[chain.symbol] || ['#6B7280', '#9CA3AF'];

    return (
      <TouchableOpacity
        key={chain.symbol}
        style={styles.chainItem}
        onPress={() => handleChainPress(chain)}
        activeOpacity={0.8}
      >
        <LinearGradient
          colors={colors}
          start={{ x: 0, y: 0 }}
          end={{ x: 1, y: 1 }}
          style={[styles.chainGradient, isExpanded && styles.chainExpanded]}
        >
          <View style={styles.chainHeader}>
            <View style={styles.chainInfo}>
              <Text style={styles.chainIcon}>
                {CHAIN_ICONS[chain.symbol] || '●'}
              </Text>
              <View style={styles.chainDetails}>
                <Text style={styles.chainSymbol}>{chain.symbol}</Text>
                <Text style={styles.chainName}>{chain.name}</Text>
              </View>
            </View>
            <View style={styles.chainBalance}>
              <Text style={styles.balanceAmount}>
                {formatBalance(chain.balance)} {chain.symbol}
              </Text>
              <Text style={styles.balanceUSD}>
                {formatUSDValue(chain.usdValue)}
              </Text>
            </View>
            <Ionicons
              name={isExpanded ? 'chevron-up' : 'chevron-down'}
              size={20}
              color="rgba(255, 255, 255, 0.8)"
            />
          </View>

          {isExpanded && (
            <View style={styles.chainActions}>
              <TouchableOpacity style={styles.actionButton}>
                <Ionicons name="arrow-up" size={16} color="#fff" />
                <Text style={styles.actionText}>Send</Text>
              </TouchableOpacity>
              <TouchableOpacity style={styles.actionButton}>
                <Ionicons name="arrow-down" size={16} color="#fff" />
                <Text style={styles.actionText}>Receive</Text>
              </TouchableOpacity>
              <TouchableOpacity style={styles.actionButton}>
                <Ionicons name="swap-horizontal" size={16} color="#fff" />
                <Text style={styles.actionText}>Swap</Text>
              </TouchableOpacity>
            </View>
          )}
        </LinearGradient>
      </TouchableOpacity>
    );
  };

  if (isLoading) {
    return (
      <View style={styles.loadingContainer}>
        <ActivityIndicator size="large" color="#FF6B35" />
        <Text style={styles.loadingText}>Loading wallets...</Text>
      </View>
    );
  }

  return (
    <View style={styles.container}>
      <LinearGradient
        colors={['rgba(255, 107, 53, 0.1)', 'rgba(255, 107, 53, 0.05)']}
        style={styles.card}
      >
        <View style={styles.header}>
          <View>
            <Text style={styles.title}>Multi-Chain Wallet</Text>
            <Text style={styles.subtitle}>Total Value: {getTotalUSDValue()}</Text>
          </View>
          <TouchableOpacity style={styles.addButton} onPress={onAddChain}>
            <Ionicons name="add" size={24} color="#FF6B35" />
          </TouchableOpacity>
        </View>

        <View style={styles.chainsList}>
          {chains.map(renderChainItem)}
          
          {chains.length === 0 && (
            <View style={styles.emptyState}>
              <Ionicons name="wallet-outline" size={48} color="#9CA3AF" />
              <Text style={styles.emptyText}>No wallets yet</Text>
              <Text style={styles.emptySubtext}>
                Tap the + button to add your first blockchain wallet
              </Text>
            </View>
          )}
        </View>
      </LinearGradient>
    </View>
  );
};

const styles = StyleSheet.create({
  container: {
    marginVertical: 8,
  },
  card: {
    borderRadius: 16,
    padding: 20,
    marginHorizontal: 16,
    borderWidth: 1,
    borderColor: 'rgba(255, 107, 53, 0.2)',
  },
  header: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: 16,
  },
  title: {
    fontSize: 20,
    fontWeight: 'bold',
    color: '#1F2937',
  },
  subtitle: {
    fontSize: 14,
    color: '#6B7280',
    marginTop: 2,
  },
  addButton: {
    width: 40,
    height: 40,
    borderRadius: 20,
    backgroundColor: 'rgba(255, 107, 53, 0.1)',
    justifyContent: 'center',
    alignItems: 'center',
    borderWidth: 1,
    borderColor: '#FF6B35',
  },
  chainsList: {
    gap: 12,
  },
  chainItem: {
    borderRadius: 12,
    overflow: 'hidden',
  },
  chainGradient: {
    padding: 16,
  },
  chainExpanded: {
    paddingBottom: 20,
  },
  chainHeader: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
  },
  chainInfo: {
    flexDirection: 'row',
    alignItems: 'center',
    flex: 1,
  },
  chainIcon: {
    fontSize: 24,
    color: '#fff',
    marginRight: 12,
    width: 32,
    textAlign: 'center',
  },
  chainDetails: {
    flex: 1,
  },
  chainSymbol: {
    fontSize: 16,
    fontWeight: 'bold',
    color: '#fff',
  },
  chainName: {
    fontSize: 12,
    color: 'rgba(255, 255, 255, 0.8)',
    marginTop: 2,
  },
  chainBalance: {
    alignItems: 'flex-end',
    marginRight: 12,
  },
  balanceAmount: {
    fontSize: 14,
    fontWeight: '600',
    color: '#fff',
  },
  balanceUSD: {
    fontSize: 12,
    color: 'rgba(255, 255, 255, 0.8)',
    marginTop: 2,
  },
  chainActions: {
    flexDirection: 'row',
    justifyContent: 'space-around',
    marginTop: 16,
    paddingTop: 16,
    borderTopWidth: 1,
    borderTopColor: 'rgba(255, 255, 255, 0.2)',
  },
  actionButton: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingVertical: 8,
    paddingHorizontal: 16,
    borderRadius: 8,
    backgroundColor: 'rgba(255, 255, 255, 0.2)',
  },
  actionText: {
    color: '#fff',
    fontSize: 12,
    fontWeight: '600',
    marginLeft: 4,
  },
  loadingContainer: {
    padding: 40,
    alignItems: 'center',
    justifyContent: 'center',
  },
  loadingText: {
    marginTop: 12,
    fontSize: 16,
    color: '#6B7280',
  },
  emptyState: {
    alignItems: 'center',
    paddingVertical: 32,
  },
  emptyText: {
    fontSize: 18,
    fontWeight: '600',
    color: '#6B7280',
    marginTop: 12,
  },
  emptySubtext: {
    fontSize: 14,
    color: '#9CA3AF',
    textAlign: 'center',
    marginTop: 8,
    lineHeight: 20,
  },
});
