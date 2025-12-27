import React from 'react';
import { View, Text, StyleSheet, ViewStyle } from 'react-native';

interface CryptoCardProps {
  symbol: string;
  name: string;
  balance: string;
  change: string;
  style?: ViewStyle;
}

export const CryptoCard: React.FC<CryptoCardProps> = ({ 
  symbol, 
  name, 
  balance, 
  change, 
  style 
}) => {
  return (
    <View style={[styles.card, style]} testID="crypto-card">
      <Text style={styles.symbol}>{symbol}</Text>
      <Text style={styles.name}>{name}</Text>
      <Text style={styles.balance}>{balance}</Text>
      <Text style={styles.change}>{change}</Text>
    </View>
  );
};

const styles = StyleSheet.create({
  card: {
    backgroundColor: 'rgba(255, 255, 255, 0.1)',
    borderRadius: 8,
    padding: 12,
    margin: 4,
    borderWidth: 1,
    borderColor: 'rgba(255, 255, 255, 0.2)',
  },
  symbol: {
    fontSize: 16,
    fontWeight: 'bold',
    color: '#fff',
  },
  name: {
    fontSize: 14,
    color: '#ccc',
  },
  balance: {
    fontSize: 18,
    fontWeight: 'bold',
    color: '#fff',
    marginTop: 4,
  },
  change: {
    fontSize: 12,
    color: '#4CAF50',
  },
});

export default CryptoCard;
