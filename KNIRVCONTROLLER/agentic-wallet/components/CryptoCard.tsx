import React from 'react';
import { View, Text, StyleSheet, TouchableOpacity, Image } from 'react-native';
import { TrendingUp, TrendingDown } from 'lucide-react-native';
import GlassCard from './GlassCard';

interface CryptoCardProps {
  symbol: string;
  name: string;
  price: string;
  change: number;
  amount: string;
  value: string;
  icon: string;
  onPress?: () => void;
}

export default function CryptoCard({
  symbol,
  name,
  price,
  change,
  amount,
  value,
  icon,
  onPress,
}: CryptoCardProps) {
  const isPositive = change >= 0;

  return (
    <TouchableOpacity onPress={onPress}>
      <GlassCard style={styles.container}>
        <View style={styles.header}>
          <View style={styles.cryptoInfo}>
            <Image source={{ uri: icon }} style={styles.icon} />
            <View style={styles.textInfo}>
              <Text style={styles.symbol}>{symbol}</Text>
              <Text style={styles.name}>{name}</Text>
            </View>
          </View>
          <View style={styles.priceInfo}>
            <Text style={styles.price}>{price}</Text>
            <View style={styles.changeContainer}>
              {isPositive ? (
                <TrendingUp size={12} color="#00FF88" />
              ) : (
                <TrendingDown size={12} color="#FF6363" />
              )}
              <Text style={[styles.change, { color: isPositive ? '#00FF88' : '#FF6363' }]}>
                {isPositive ? '+' : ''}{change.toFixed(2)}%
              </Text>
            </View>
          </View>
        </View>
        <View style={styles.holdings}>
          <Text style={styles.amount}>{amount} {symbol}</Text>
          <Text style={styles.value}>{value}</Text>
        </View>
      </GlassCard>
    </TouchableOpacity>
  );
}

const styles = StyleSheet.create({
  container: {
    marginBottom: 12,
  },
  header: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: 12,
  },
  cryptoInfo: {
    flexDirection: 'row',
    alignItems: 'center',
  },
  icon: {
    width: 36,
    height: 36,
    borderRadius: 18,
    marginRight: 12,
  },
  textInfo: {
    flex: 1,
  },
  symbol: {
    fontSize: 16,
    fontFamily: 'Inter-SemiBold',
    color: '#FFFFFF',
    marginBottom: 2,
  },
  name: {
    fontSize: 12,
    fontFamily: 'Inter-Regular',
    color: '#999999',
  },
  priceInfo: {
    alignItems: 'flex-end',
  },
  price: {
    fontSize: 16,
    fontFamily: 'Inter-SemiBold',
    color: '#FFFFFF',
    marginBottom: 2,
  },
  changeContainer: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 4,
  },
  change: {
    fontSize: 12,
    fontFamily: 'Inter-Medium',
  },
  holdings: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    paddingTop: 12,
    borderTopWidth: 1,
    borderTopColor: 'rgba(255, 255, 255, 0.1)',
  },
  amount: {
    fontSize: 14,
    fontFamily: 'Inter-Regular',
    color: '#CCCCCC',
  },
  value: {
    fontSize: 14,
    fontFamily: 'Inter-SemiBold',
    color: '#00D2FF',
  },
});