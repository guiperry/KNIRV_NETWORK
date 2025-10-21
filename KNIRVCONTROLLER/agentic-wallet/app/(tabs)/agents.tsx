import React, { useState } from 'react';
import { View, Text, StyleSheet, ScrollView, TouchableOpacity } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { LinearGradient } from 'expo-linear-gradient';
import { Plus, Search, Filter, Bot, Play, Pause, Settings } from 'lucide-react-native';
import GlassCard from '@/components/GlassCard';
import AIAgentCard from '@/components/AIAgentCard';

export default function AgentsScreen() {
  const [activeTab, setActiveTab] = useState<'active' | 'marketplace' | 'installed'>('active');

  const activeAgents = [
    {
      name: 'DeFi Yield Optimizer',
      description: 'Automatically finds and executes the best yield farming opportunities across multiple protocols.',
      performance: 15.8,
      risk: 'Medium' as const,
      status: 'Active' as const,
      category: 'DeFi',
    },
    {
      name: 'Arbitrage Hunter',
      description: 'Scans multiple exchanges for price discrepancies and executes profitable arbitrage trades.',
      performance: 22.3,
      risk: 'Low' as const,
      status: 'Active' as const,
      category: 'Trading',
    },
    {
      name: 'NFT Trend Analyzer',
      description: 'Monitors NFT collections for trending patterns and investment opportunities.',
      performance: -3.2,
      risk: 'High' as const,
      status: 'Active' as const,
      category: 'NFT',
    },
  ];

  const marketplaceAgents = [
    {
      name: 'Grid Trading Bot',
      description: 'Executes buy and sell orders at predetermined intervals to profit from market volatility.',
      performance: 18.6,
      risk: 'Medium' as const,
      status: 'Installing' as const,
      category: 'Trading',
    },
    {
      name: 'Social Sentiment Trader',
      description: 'Analyzes social media sentiment and news to predict market movements.',
      performance: 12.4,
      risk: 'High' as const,
      status: 'Inactive' as const,
      category: 'Analytics',
    },
    {
      name: 'Dollar Cost Averaging',
      description: 'Automatically invests fixed amounts at regular intervals to reduce volatility impact.',
      performance: 8.9,
      risk: 'Low' as const,
      status: 'Inactive' as const,
      category: 'Investment',
    },
  ];

  const tabs = [
    { key: 'active', label: 'Active', count: 3 },
    { key: 'marketplace', label: 'Marketplace', count: 12 },
    { key: 'installed', label: 'Installed', count: 5 },
  ];

  const getAgentsToShow = () => {
    switch (activeTab) {
      case 'active':
        return activeAgents;
      case 'marketplace':
        return marketplaceAgents;
      case 'installed':
        return [...activeAgents, ...marketplaceAgents.slice(0, 2)];
      default:
        return activeAgents;
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
            <Text style={styles.title}>AI Agents</Text>
            <View style={styles.headerActions}>
              <TouchableOpacity style={styles.headerButton}>
                <Search size={20} color="#666666" />
              </TouchableOpacity>
              <TouchableOpacity style={styles.headerButton}>
                <Filter size={20} color="#666666" />
              </TouchableOpacity>
              <TouchableOpacity style={[styles.headerButton, styles.primaryButton]}>
                <Plus size={20} color="#000000" />
              </TouchableOpacity>
            </View>
          </View>

          {/* Performance Overview */}
          <GlassCard style={styles.performanceCard}>
            <View style={styles.performanceHeader}>
              <View>
                <Text style={styles.performanceLabel}>Total AI Performance</Text>
                <Text style={styles.performanceValue}>+18.4%</Text>
                <Text style={styles.performancePeriod}>Last 30 days</Text>
              </View>
              <View style={styles.performanceIcon}>
                <Bot size={32} color="#7B68EE" />
              </View>
            </View>
            
            <View style={styles.performanceStats}>
              <View style={styles.statItem}>
                <Text style={styles.statValue}>$2,847</Text>
                <Text style={styles.statLabel}>Profit Generated</Text>
              </View>
              <View style={styles.statItem}>
                <Text style={styles.statValue}>127</Text>
                <Text style={styles.statLabel}>Trades Executed</Text>
              </View>
              <View style={styles.statItem}>
                <Text style={styles.statValue}>94.2%</Text>
                <Text style={styles.statLabel}>Success Rate</Text>
              </View>
            </View>
          </GlassCard>

          {/* Agent Controls */}
          <View style={styles.agentControls}>
            <TouchableOpacity style={styles.controlButton}>
              <Play size={16} color="#00FF88" />
              <Text style={styles.controlButtonText}>Start All</Text>
            </TouchableOpacity>
            <TouchableOpacity style={styles.controlButton}>
              <Pause size={16} color="#FFD700" />
              <Text style={styles.controlButtonText}>Pause All</Text>
            </TouchableOpacity>
            <TouchableOpacity style={styles.controlButton}>
              <Settings size={16} color="#00D2FF" />
              <Text style={styles.controlButtonText}>Settings</Text>
            </TouchableOpacity>
          </View>

          {/* Tabs */}
          <View style={styles.tabContainer}>
            {tabs.map((tab) => (
              <TouchableOpacity
                key={tab.key}
                style={[
                  styles.tab,
                  activeTab === tab.key && styles.activeTab,
                ]}
                onPress={() => setActiveTab(tab.key as 'active' | 'marketplace' | 'installed')}
              >
                <Text
                  style={[
                    styles.tabText,
                    activeTab === tab.key && styles.activeTabText,
                  ]}
                >
                  {tab.label}
                </Text>
                <View style={[
                  styles.tabBadge,
                  activeTab === tab.key && styles.activeTabBadge,
                ]}>
                  <Text style={[
                    styles.tabBadgeText,
                    activeTab === tab.key && styles.activeTabBadgeText,
                  ]}>
                    {tab.count}
                  </Text>
                </View>
              </TouchableOpacity>
            ))}
          </View>

          {/* Agents List */}
          <View style={styles.agentsList}>
            {getAgentsToShow().map((agent, index) => (
              <AIAgentCard
                key={index}
                name={agent.name}
                description={agent.description}
                performance={agent.performance}
                risk={agent.risk}
                status={agent.status}
                category={agent.category}
              />
            ))}
          </View>

          {/* Quick Deploy */}
          {activeTab === 'marketplace' && (
            <GlassCard style={styles.quickDeployCard} variant="accent">
              <View style={styles.quickDeployHeader}>
                <Text style={styles.quickDeployTitle}>Quick Deploy Strategy</Text>
                <Text style={styles.quickDeploySubtitle}>
                  Deploy pre-configured strategies instantly
                </Text>
              </View>
              <View style={styles.quickDeployButtons}>
                <TouchableOpacity style={styles.quickDeployButton}>
                  <Text style={styles.quickDeployButtonText}>Conservative Portfolio</Text>
                </TouchableOpacity>
                <TouchableOpacity style={styles.quickDeployButton}>
                  <Text style={styles.quickDeployButtonText}>Aggressive Growth</Text>
                </TouchableOpacity>
              </View>
            </GlassCard>
          )}
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
  title: {
    fontSize: 28,
    fontFamily: 'Inter-Bold',
    color: '#FFFFFF',
  },
  headerActions: {
    flexDirection: 'row',
    gap: 12,
  },
  headerButton: {
    width: 40,
    height: 40,
    borderRadius: 20,
    backgroundColor: 'rgba(255, 255, 255, 0.1)',
    justifyContent: 'center',
    alignItems: 'center',
  },
  primaryButton: {
    backgroundColor: '#00D2FF',
  },
  performanceCard: {
    marginBottom: 20,
  },
  performanceHeader: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: 20,
  },
  performanceLabel: {
    fontSize: 14,
    fontFamily: 'Inter-Regular',
    color: '#999999',
    marginBottom: 4,
  },
  performanceValue: {
    fontSize: 28,
    fontFamily: 'Inter-Bold',
    color: '#00FF88',
    marginBottom: 2,
  },
  performancePeriod: {
    fontSize: 12,
    fontFamily: 'Inter-Regular',
    color: '#999999',
  },
  performanceIcon: {
    width: 60,
    height: 60,
    borderRadius: 30,
    backgroundColor: 'rgba(123, 104, 238, 0.2)',
    justifyContent: 'center',
    alignItems: 'center',
  },
  performanceStats: {
    flexDirection: 'row',
    justifyContent: 'space-between',
  },
  statItem: {
    alignItems: 'center',
  },
  statValue: {
    fontSize: 16,
    fontFamily: 'Inter-SemiBold',
    color: '#FFFFFF',
    marginBottom: 4,
  },
  statLabel: {
    fontSize: 12,
    fontFamily: 'Inter-Regular',
    color: '#999999',
    textAlign: 'center',
  },
  agentControls: {
    flexDirection: 'row',
    gap: 12,
    marginBottom: 20,
  },
  controlButton: {
    flex: 1,
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    paddingVertical: 12,
    borderRadius: 12,
    backgroundColor: 'rgba(255, 255, 255, 0.05)',
    borderWidth: 1,
    borderColor: 'rgba(255, 255, 255, 0.1)',
    gap: 8,
  },
  controlButtonText: {
    fontSize: 12,
    fontFamily: 'Inter-SemiBold',
    color: '#FFFFFF',
  },
  tabContainer: {
    flexDirection: 'row',
    marginBottom: 20,
    backgroundColor: 'rgba(255, 255, 255, 0.05)',
    borderRadius: 12,
    padding: 4,
  },
  tab: {
    flex: 1,
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    paddingVertical: 8,
    paddingHorizontal: 12,
    borderRadius: 8,
    gap: 6,
  },
  activeTab: {
    backgroundColor: '#7B68EE',
  },
  tabText: {
    fontSize: 12,
    fontFamily: 'Inter-Medium',
    color: '#999999',
  },
  activeTabText: {
    color: '#FFFFFF',
  },
  tabBadge: {
    minWidth: 18,
    height: 18,
    borderRadius: 9,
    backgroundColor: 'rgba(255, 255, 255, 0.1)',
    justifyContent: 'center',
    alignItems: 'center',
  },
  activeTabBadge: {
    backgroundColor: 'rgba(255, 255, 255, 0.2)',
  },
  tabBadgeText: {
    fontSize: 10,
    fontFamily: 'Inter-SemiBold',
    color: '#999999',
  },
  activeTabBadgeText: {
    color: '#FFFFFF',
  },
  agentsList: {
    marginBottom: 24,
  },
  quickDeployCard: {
    marginBottom: 40,
  },
  quickDeployHeader: {
    marginBottom: 16,
  },
  quickDeployTitle: {
    fontSize: 16,
    fontFamily: 'Inter-SemiBold',
    color: '#FFFFFF',
    marginBottom: 4,
  },
  quickDeploySubtitle: {
    fontSize: 14,
    fontFamily: 'Inter-Regular',
    color: '#999999',
  },
  quickDeployButtons: {
    flexDirection: 'row',
    gap: 12,
  },
  quickDeployButton: {
    flex: 1,
    paddingVertical: 12,
    paddingHorizontal: 16,
    borderRadius: 8,
    backgroundColor: 'rgba(255, 215, 0, 0.1)',
    borderWidth: 1,
    borderColor: 'rgba(255, 215, 0, 0.3)',
    alignItems: 'center',
  },
  quickDeployButtonText: {
    fontSize: 12,
    fontFamily: 'Inter-SemiBold',
    color: '#FFD700',
  },
});