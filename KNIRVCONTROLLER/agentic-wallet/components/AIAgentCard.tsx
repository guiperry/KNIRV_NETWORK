import React from 'react';
import { View, Text, StyleSheet, TouchableOpacity } from 'react-native';
import { Bot, TrendingUp, Shield } from 'lucide-react-native';
import GlassCard from './GlassCard';

interface AIAgentCardProps {
  name: string;
  description: string;
  performance: number;
  risk: 'Low' | 'Medium' | 'High';
  status: 'Active' | 'Inactive' | 'Installing';
  category: string;
  onPress?: () => void;
}

export default function AIAgentCard({
  name,
  description,
  performance,
  risk,
  status,
  category,
  onPress,
}: AIAgentCardProps) {
  const getRiskColor = (risk: string) => {
    switch (risk) {
      case 'Low': return '#00FF88';
      case 'Medium': return '#FFD700';
      case 'High': return '#FF6363';
      default: return '#999999';
    }
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'Active': return '#00FF88';
      case 'Inactive': return '#999999';
      case 'Installing': return '#FFD700';
      default: return '#999999';
    }
  };

  return (
    <TouchableOpacity onPress={onPress}>
      <GlassCard style={styles.container} variant="secondary">
        <View style={styles.header}>
          <View style={styles.agentInfo}>
            <View style={styles.iconContainer}>
              <Bot size={24} color="#7B68EE" />
            </View>
            <View style={styles.textInfo}>
              <Text style={styles.name}>{name}</Text>
              <Text style={styles.category}>{category}</Text>
            </View>
          </View>
          <View style={[styles.statusBadge, { backgroundColor: getStatusColor(status) + '20' }]}>
            <Text style={[styles.statusText, { color: getStatusColor(status) }]}>
              {status}
            </Text>
          </View>
        </View>
        
        <Text style={styles.description}>{description}</Text>
        
        <View style={styles.metrics}>
          <View style={styles.metric}>
            <TrendingUp size={16} color="#00D2FF" />
            <Text style={styles.metricLabel}>Performance</Text>
            <Text style={[styles.metricValue, { color: performance >= 0 ? '#00FF88' : '#FF6363' }]}>
              {performance >= 0 ? '+' : ''}{performance.toFixed(1)}%
            </Text>
          </View>
          
          <View style={styles.metric}>
            <Shield size={16} color={getRiskColor(risk)} />
            <Text style={styles.metricLabel}>Risk</Text>
            <Text style={[styles.metricValue, { color: getRiskColor(risk) }]}>
              {risk}
            </Text>
          </View>
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
  agentInfo: {
    flexDirection: 'row',
    alignItems: 'center',
    flex: 1,
  },
  iconContainer: {
    width: 40,
    height: 40,
    borderRadius: 20,
    backgroundColor: 'rgba(123, 104, 238, 0.2)',
    justifyContent: 'center',
    alignItems: 'center',
    marginRight: 12,
  },
  textInfo: {
    flex: 1,
  },
  name: {
    fontSize: 16,
    fontFamily: 'Inter-SemiBold',
    color: '#FFFFFF',
    marginBottom: 2,
  },
  category: {
    fontSize: 12,
    fontFamily: 'Inter-Regular',
    color: '#999999',
  },
  statusBadge: {
    paddingHorizontal: 8,
    paddingVertical: 4,
    borderRadius: 12,
    borderWidth: 1,
    borderColor: 'transparent',
  },
  statusText: {
    fontSize: 10,
    fontFamily: 'Inter-SemiBold',
    textTransform: 'uppercase',
  },
  description: {
    fontSize: 14,
    fontFamily: 'Inter-Regular',
    color: '#CCCCCC',
    lineHeight: 20,
    marginBottom: 16,
  },
  metrics: {
    flexDirection: 'row',
    justifyContent: 'space-between',
  },
  metric: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 6,
  },
  metricLabel: {
    fontSize: 12,
    fontFamily: 'Inter-Regular',
    color: '#999999',
  },
  metricValue: {
    fontSize: 12,
    fontFamily: 'Inter-SemiBold',
  },
});
