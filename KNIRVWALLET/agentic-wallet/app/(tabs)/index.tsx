import React from 'react';
import { View, Text, StyleSheet, ScrollView, Dimensions } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { LinearGradient } from 'expo-linear-gradient';
import { Eye, TrendingUp, Activity, Zap } from 'lucide-react-native';
import GlassCard from '@/components/GlassCard';
import PortfolioChart from '@/components/PortfolioChart';
import CryptoCard from '@/components/CryptoCard';

const { width } = Dimensions.get('window');

export default function HomeScreen() {
  const portfolioData = [42000, 43500, 41800, 44200, 45600, 44900, 47800, 46500, 48200];
  const portfolioChange = 12.5;

  const cryptoAssets = [
    {
      symbol: 'BTC',
      name: 'Bitcoin',
      price: '$47,842.50',
      change: 3.24,
      amount: '0.2845',
      value: '$13,613.25',
      icon: 'https://images.pexels.com/photos/844124/pexels-photo-844124.jpeg?auto=compress&cs=tinysrgb&w=100&h=100&dpr=2',
    },
    {
      symbol: 'ETH',
      name: 'Ethereum',
      price: '$2,845.32',
      change: -1.86,
      amount: '4.2567',
      value: '$12,115.89',
      icon: 'https://images.pexels.com/photos/844124/pexels-photo-844124.jpeg?auto=compress&cs=tinysrgb&w=100&h=100&dpr=2',
    },
    {
      symbol: 'SOL',
      name: 'Solana',
      price: '$98.45',
      change: 8.73,
      amount: '125.42',
      value: '$12,348.29',
      icon: 'https://images.pexels.com/photos/844124/pexels-photo-844124.jpeg?auto=compress&cs=tinysrgb&w=100&h=100&dpr=2',
    },
  ];

  return (
    <LinearGradient
      colors={['#0A0A0B', '#1A1A1B', '#0A0A0B']}
      style={styles.container}
    >
      <SafeAreaView style={styles.safeArea}>
        <ScrollView style={styles.scrollView} showsVerticalScrollIndicator={false}>
          {/* Header */}
          <View style={styles.header}>
            <View>
              <Text style={styles.greeting}>Good morning</Text>
              <Text style={styles.userName}>Alex Chen</Text>
            </View>
            <View style={styles.headerActions}>
              <View style={styles.notificationBadge}>
                <Activity size={20} color="#00D2FF" />
              </View>
            </View>
          </View>

          {/* Portfolio Overview */}
          <GlassCard style={styles.portfolioCard}>
            <View style={styles.portfolioHeader}>
              <View style={styles.portfolioInfo}>
                <Text style={styles.portfolioLabel}>Total Portfolio Value</Text>
                <Text style={styles.portfolioValue}>$48,234.67</Text>
                <View style={styles.changeContainer}>
                  <TrendingUp size={16} color="#00FF88" />
                  <Text style={styles.changeText}>+{portfolioChange}% (+$5,387.23)</Text>
                  <Text style={styles.changePeriod}>Today</Text>
                </View>
              </View>
              <View style={styles.eyeIcon}>
                <Eye size={20} color="#666666" />
              </View>
            </View>
            <PortfolioChart data={portfolioData} change={portfolioChange} />
          </GlassCard>

          {/* Quick Actions */}
          <View style={styles.quickActions}>
            <Text style={styles.sectionTitle}>Quick Actions</Text>
            <View style={styles.actionGrid}>
              <GlassCard style={styles.actionCard} variant="primary">
                <View style={styles.actionIcon}>
                  <TrendingUp size={20} color="#00D2FF" />
                </View>
                <Text style={styles.actionText}>Buy Crypto</Text>
              </GlassCard>
              <GlassCard style={styles.actionCard} variant="secondary">
                <View style={styles.actionIcon}>
                  <Activity size={20} color="#7B68EE" />
                </View>
                <Text style={styles.actionText}>Trade</Text>
              </GlassCard>
              <GlassCard style={styles.actionCard} variant="accent">
                <View style={styles.actionIcon}>
                  <Zap size={20} color="#FFD700" />
                </View>
                <Text style={styles.actionText}>AI Agents</Text>
              </GlassCard>
            </View>
          </View>

          {/* Holdings */}
          <View style={styles.holdings}>
            <View style={styles.sectionHeader}>
              <Text style={styles.sectionTitle}>Your Holdings</Text>
              <Text style={styles.seeAll}>See All</Text>
            </View>
            {cryptoAssets.map((asset, index) => (
              <CryptoCard
                key={index}
                symbol={asset.symbol}
                name={asset.name}
                price={asset.price}
                change={asset.change}
                amount={asset.amount}
                value={asset.value}
                icon={asset.icon}
              />
            ))}
          </View>

          {/* Market Insights */}
          <View style={styles.marketInsights}>
            <Text style={styles.sectionTitle}>Market Insights</Text>
            <GlassCard style={styles.insightCard}>
              <View style={styles.insightHeader}>
                <Text style={styles.insightTitle}>AI Strategy Alert</Text>
                <View style={styles.insightBadge}>
                  <Text style={styles.insightBadgeText}>New</Text>
                </View>
              </View>
              <Text style={styles.insightDescription}>
                Your DeFi Yield Optimizer found a new opportunity with 15.2% APY on USDC staking.
              </Text>
              <Text style={styles.insightAction}>View Recommendation →</Text>
            </GlassCard>
          </View>
        </ScrollView>
      </SafeAreaView>
    </LinearGradient>
  );
}

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
  greeting: {
    fontSize: 14,
    fontFamily: 'Inter-Regular',
    color: '#999999',
    marginBottom: 4,
  },
  userName: {
    fontSize: 24,
    fontFamily: 'Inter-Bold',
    color: '#FFFFFF',
  },
  headerActions: {
    flexDirection: 'row',
    gap: 12,
  },
  notificationBadge: {
    width: 40,
    height: 40,
    borderRadius: 20,
    backgroundColor: 'rgba(0, 210, 255, 0.2)',
    justifyContent: 'center',
    alignItems: 'center',
  },
  portfolioCard: {
    marginBottom: 24,
  },
  portfolioHeader: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'flex-start',
    marginBottom: 8,
  },
  portfolioInfo: {
    flex: 1,
  },
  portfolioLabel: {
    fontSize: 14,
    fontFamily: 'Inter-Regular',
    color: '#999999',
    marginBottom: 8,
  },
  portfolioValue: {
    fontSize: 32,
    fontFamily: 'Inter-Bold',
    color: '#FFFFFF',
    marginBottom: 8,
  },
  changeContainer: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 6,
  },
  changeText: {
    fontSize: 14,
    fontFamily: 'Inter-SemiBold',
    color: '#00FF88',
  },
  changePeriod: {
    fontSize: 12,
    fontFamily: 'Inter-Regular',
    color: '#999999',
    marginLeft: 4,
  },
  eyeIcon: {
    padding: 8,
  },
  quickActions: {
    marginBottom: 24,
  },
  sectionTitle: {
    fontSize: 18,
    fontFamily: 'Inter-SemiBold',
    color: '#FFFFFF',
    marginBottom: 16,
  },
  actionGrid: {
    flexDirection: 'row',
    gap: 12,
  },
  actionCard: {
    flex: 1,
    alignItems: 'center',
    paddingVertical: 16,
  },
  actionIcon: {
    width: 40,
    height: 40,
    borderRadius: 20,
    justifyContent: 'center',
    alignItems: 'center',
    marginBottom: 8,
  },
  actionText: {
    fontSize: 12,
    fontFamily: 'Inter-Medium',
    color: '#FFFFFF',
    textAlign: 'center',
  },
  holdings: {
    marginBottom: 24,
  },
  sectionHeader: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: 16,
  },
  seeAll: {
    fontSize: 14,
    fontFamily: 'Inter-Medium',
    color: '#00D2FF',
  },
  marketInsights: {
    marginBottom: 40,
  },
  insightCard: {
    marginBottom: 12,
  },
  insightHeader: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: 8,
  },
  insightTitle: {
    fontSize: 16,
    fontFamily: 'Inter-SemiBold',
    color: '#FFFFFF',
  },
  insightBadge: {
    backgroundColor: '#00D2FF',
    paddingHorizontal: 8,
    paddingVertical: 2,
    borderRadius: 10,
  },
  insightBadgeText: {
    fontSize: 10,
    fontFamily: 'Inter-SemiBold',
    color: '#000000',
  },
  insightDescription: {
    fontSize: 14,
    fontFamily: 'Inter-Regular',
    color: '#CCCCCC',
    lineHeight: 20,
    marginBottom: 12,
  },
  insightAction: {
    fontSize: 14,
    fontFamily: 'Inter-SemiBold',
    color: '#00D2FF',
  },
});