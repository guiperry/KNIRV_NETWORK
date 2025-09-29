import React from 'react';
import { View, Text, StyleSheet, ScrollView, TouchableOpacity, Switch } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { LinearGradient } from 'expo-linear-gradient';
import { User, Shield, Bell, Smartphone, Globe, CircleHelp as HelpCircle, LogOut, ChevronRight, Fingerprint, Key, TriangleAlert as AlertTriangle } from 'lucide-react-native';
import GlassCard from '@/components/GlassCard';

export default function SettingsScreen() {
  const [biometricEnabled, setBiometricEnabled] = React.useState(true);
  const [notificationsEnabled, setNotificationsEnabled] = React.useState(true);
  const [autoLockEnabled, setAutoLockEnabled] = React.useState(true);

  const securitySettings = [
    {
      icon: Fingerprint,
      title: 'Biometric Authentication',
      subtitle: 'Use fingerprint or Face ID',
      toggle: true,
      value: biometricEnabled,
      onToggle: setBiometricEnabled,
    },
    {
      icon: Key,
      title: 'Private Key Management',
      subtitle: 'Backup and recovery options',
      action: true,
    },
    {
      icon: Shield,
      title: 'Transaction Approval',
      subtitle: 'Require approval for all transactions',
      toggle: true,
      value: autoLockEnabled,
      onToggle: setAutoLockEnabled,
    },
  ];

  const generalSettings = [
    {
      icon: Bell,
      title: 'Notifications',
      subtitle: 'Price alerts and transaction updates',
      toggle: true,
      value: notificationsEnabled,
      onToggle: setNotificationsEnabled,
    },
    {
      icon: Smartphone,
      title: 'App Preferences',
      subtitle: 'Theme, language, and display options',
      action: true,
    },
    {
      icon: Globe,
      title: 'Network Settings',
      subtitle: 'Blockchain networks and RPC endpoints',
      action: true,
    },
  ];

  const supportSettings = [
    {
      icon: HelpCircle,
      title: 'Help & Support',
      subtitle: 'FAQ, documentation, and contact',
      action: true,
    },
    {
      icon: AlertTriangle,
      title: 'Report an Issue',
      subtitle: 'Bug reports and feature requests',
      action: true,
    },
  ];

  const renderSettingItem = (item: any, index: number) => (
    <TouchableOpacity key={index}>
      <GlassCard style={styles.settingCard}>
        <View style={styles.settingContent}>
          <View style={styles.settingLeft}>
            <View style={styles.settingIcon}>
              <item.icon size={20} color="#00D2FF" />
            </View>
            <View style={styles.settingText}>
              <Text style={styles.settingTitle}>{item.title}</Text>
              <Text style={styles.settingSubtitle}>{item.subtitle}</Text>
            </View>
          </View>
          <View style={styles.settingRight}>
            {item.toggle ? (
              <Switch
                value={item.value}
                onValueChange={item.onToggle}
                trackColor={{ false: '#333333', true: '#00D2FF' }}
                thumbColor={item.value ? '#FFFFFF' : '#999999'}
              />
            ) : (
              <ChevronRight size={20} color="#666666" />
            )}
          </View>
        </View>
      </GlassCard>
    </TouchableOpacity>
  );

  return (
    <LinearGradient
      colors={['#0A0A0B', '#1A1A1B', '#0A0A0B']}
      style={styles.container}
    >
      <SafeAreaView style={styles.safeArea}>
        <ScrollView style={styles.scrollView} showsVerticalScrollIndicator={false}>
          {/* Header */}
          <View style={styles.header}>
            <Text style={styles.title}>Settings</Text>
          </View>

          {/* Profile Section */}
          <GlassCard style={styles.profileCard}>
            <View style={styles.profileContent}>
              <View style={styles.avatarContainer}>
                <User size={32} color="#00D2FF" />
              </View>
              <View style={styles.profileText}>
                <Text style={styles.profileName}>Alex Chen</Text>
                <Text style={styles.profileEmail}>alex.chen@example.com</Text>
              </View>
              <TouchableOpacity style={styles.editButton}>
                <Text style={styles.editButtonText}>Edit</Text>
              </TouchableOpacity>
            </View>
          </GlassCard>

          {/* Security Settings */}
          <View style={styles.section}>
            <Text style={styles.sectionTitle}>Security</Text>
            <Text style={styles.sectionSubtitle}>
              Keep your wallet and funds secure with advanced security features
            </Text>
            {securitySettings.map(renderSettingItem)}
          </View>

          {/* General Settings */}
          <View style={styles.section}>
            <Text style={styles.sectionTitle}>General</Text>
            {generalSettings.map(renderSettingItem)}
          </View>

          {/* Support Settings */}
          <View style={styles.section}>
            <Text style={styles.sectionTitle}>Support</Text>
            {supportSettings.map(renderSettingItem)}
          </View>

          {/* Wallet Status */}
          <GlassCard style={styles.statusCard} variant="accent">
            <View style={styles.statusHeader}>
              <Text style={styles.statusTitle}>Wallet Status</Text>
              <View style={styles.statusBadge}>
                <Text style={styles.statusBadgeText}>Secured</Text>
              </View>
            </View>
            <Text style={styles.statusDescription}>
              Your wallet is properly secured with biometric authentication and encrypted storage.
            </Text>
            <View style={styles.statusMetrics}>
              <View style={styles.statusMetric}>
                <Text style={styles.statusMetricValue}>256-bit</Text>
                <Text style={styles.statusMetricLabel}>Encryption</Text>
              </View>
              <View style={styles.statusMetric}>
                <Text style={styles.statusMetricValue}>3/3</Text>
                <Text style={styles.statusMetricLabel}>Security Checks</Text>
              </View>
            </View>
          </GlassCard>

          {/* Logout Button */}
          <TouchableOpacity style={styles.logoutButton}>
            <LogOut size={20} color="#FF6363" />
            <Text style={styles.logoutText}>Sign Out</Text>
          </TouchableOpacity>

          <View style={styles.footer}>
            <Text style={styles.footerText}>Version 1.0.0</Text>
            <Text style={styles.footerText}>© 2024 CryptoWallet AI</Text>
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
    paddingTop: 20,
    marginBottom: 24,
  },
  title: {
    fontSize: 28,
    fontFamily: 'Inter-Bold',
    color: '#FFFFFF',
  },
  profileCard: {
    marginBottom: 32,
  },
  profileContent: {
    flexDirection: 'row',
    alignItems: 'center',
  },
  avatarContainer: {
    width: 60,
    height: 60,
    borderRadius: 30,
    backgroundColor: 'rgba(0, 210, 255, 0.2)',
    justifyContent: 'center',
    alignItems: 'center',
    marginRight: 16,
  },
  profileText: {
    flex: 1,
  },
  profileName: {
    fontSize: 18,
    fontFamily: 'Inter-SemiBold',
    color: '#FFFFFF',
    marginBottom: 4,
  },
  profileEmail: {
    fontSize: 14,
    fontFamily: 'Inter-Regular',
    color: '#999999',
  },
  editButton: {
    paddingHorizontal: 16,
    paddingVertical: 8,
    borderRadius: 8,
    backgroundColor: 'rgba(0, 210, 255, 0.1)',
    borderWidth: 1,
    borderColor: '#00D2FF',
  },
  editButtonText: {
    fontSize: 12,
    fontFamily: 'Inter-SemiBold',
    color: '#00D2FF',
  },
  section: {
    marginBottom: 32,
  },
  sectionTitle: {
    fontSize: 18,
    fontFamily: 'Inter-SemiBold',
    color: '#FFFFFF',
    marginBottom: 4,
  },
  sectionSubtitle: {
    fontSize: 14,
    fontFamily: 'Inter-Regular',
    color: '#999999',
    marginBottom: 16,
    lineHeight: 20,
  },
  settingCard: {
    marginBottom: 8,
  },
  settingContent: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
  },
  settingLeft: {
    flexDirection: 'row',
    alignItems: 'center',
    flex: 1,
  },
  settingIcon: {
    width: 36,
    height: 36,
    borderRadius: 18,
    backgroundColor: 'rgba(0, 210, 255, 0.1)',
    justifyContent: 'center',
    alignItems: 'center',
    marginRight: 12,
  },
  settingText: {
    flex: 1,
  },
  settingTitle: {
    fontSize: 14,
    fontFamily: 'Inter-SemiBold',
    color: '#FFFFFF',
    marginBottom: 2,
  },
  settingSubtitle: {
    fontSize: 12,
    fontFamily: 'Inter-Regular',
    color: '#999999',
  },
  settingRight: {
    marginLeft: 12,
  },
  statusCard: {
    marginBottom: 24,
  },
  statusHeader: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: 8,
  },
  statusTitle: {
    fontSize: 16,
    fontFamily: 'Inter-SemiBold',
    color: '#FFFFFF',
  },
  statusBadge: {
    backgroundColor: '#00FF88',
    paddingHorizontal: 8,
    paddingVertical: 2,
    borderRadius: 10,
  },
  statusBadgeText: {
    fontSize: 10,
    fontFamily: 'Inter-SemiBold',
    color: '#000000',
  },
  statusDescription: {
    fontSize: 14,
    fontFamily: 'Inter-Regular',
    color: '#CCCCCC',
    lineHeight: 20,
    marginBottom: 16,
  },
  statusMetrics: {
    flexDirection: 'row',
    justifyContent: 'space-between',
  },
  statusMetric: {
    alignItems: 'center',
  },
  statusMetricValue: {
    fontSize: 16,
    fontFamily: 'Inter-SemiBold',
    color: '#FFD700',
    marginBottom: 4,
  },
  statusMetricLabel: {
    fontSize: 12,
    fontFamily: 'Inter-Regular',
    color: '#999999',
  },
  logoutButton: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    paddingVertical: 16,
    borderRadius: 12,
    backgroundColor: 'rgba(255, 99, 99, 0.1)',
    borderWidth: 1,
    borderColor: 'rgba(255, 99, 99, 0.3)',
    marginBottom: 32,
    gap: 8,
  },
  logoutText: {
    fontSize: 14,
    fontFamily: 'Inter-SemiBold',
    color: '#FF6363',
  },
  footer: {
    alignItems: 'center',
    paddingBottom: 40,
  },
  footerText: {
    fontSize: 12,
    fontFamily: 'Inter-Regular',
    color: '#666666',
    marginBottom: 4,
  },
});