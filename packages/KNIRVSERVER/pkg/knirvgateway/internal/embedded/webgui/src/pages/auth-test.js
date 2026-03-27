import React, { useState, useEffect } from 'react';
import { useNavigation } from '../hooks/useNavigation';
import PageLayout from '../components/PageLayout';
import PageHeader from '../components/PageHeader';
import GlassyCard from '../components/GlassyCard';
import { useRole } from '../contexts/RoleContext';
import styles from './auth-test.module.css';

export default function AuthTest() {
  const { activePage, handleNavigation } = useNavigation('auth-test');
  const { role, network, isAuthenticated, canAccess, getUserInfo } = useRole();
  const [authHistory, setAuthHistory] = useState([]);

  useEffect(() => {
    // Log authentication state changes
    const timestamp = new Date().toLocaleTimeString();
    const newEntry = {
      timestamp,
      role,
      network,
      isAuthenticated,
      userAgent: navigator.userAgent.substring(0, 50) + '...'
    };
    
    setAuthHistory(prev => [newEntry, ...prev.slice(0, 9)]); // Keep last 10 entries
  }, [role, network, isAuthenticated]);

  const rolePermissions = {
    Root: [
      'dashboard', 'inventory', 'vault', 'blockchain', 'dex', 'daos',
      'nft-vault', 'nft-capability-manager', 'add-capability',
      'settlement', 'network-admin', 'peers', 'explorer', 'capabilities'
    ],
    Bootnode: [
      'dashboard', 'inventory', 'vault', 'blockchain', 'dex', 'daos',
      'nft-vault', 'nft-capability-manager', 'add-capability',
      'settlement', 'peers', 'explorer', 'capabilities'
    ],
    Dev: [
      'dashboard', 'inventory', 'vault', 'blockchain', 'dex',
      'nft-vault', 'nft-capability-manager', 'add-capability',
      'explorer', 'capabilities'
    ],
    General: [
      'dashboard', 'inventory', 'dex', 'nft-capability-manager', 'capabilities'
    ]
  };

  const allPages = [
    'dashboard', 'inventory', 'vault', 'blockchain', 'dex', 'daos',
    'nft-vault', 'nft-capability-manager', 'add-capability',
    'settlement', 'network-admin', 'peers', 'explorer', 'capabilities'
  ];

  const testPageAccess = (page) => {
    try {
      handleNavigation(page);
    } catch (error) {
      console.error(`Failed to navigate to ${page}:`, error);
    }
  };

  const userInfo = getUserInfo();
  const allowedPages = rolePermissions[role] || [];
  const deniedPages = allPages.filter(page => !allowedPages.includes(page));

  return (
    <PageLayout
      activePage={activePage}
      pageTitle="Authentication Testing"
    >
      <PageHeader
        title="Authentication & Role Testing"
        subtitle="Test different user roles and authentication scenarios"
      />

      {/* Current Authentication Status */}
      <GlassyCard className={styles.statusCard}>
        <h3>Current Authentication Status</h3>
        <div className={styles.statusGrid}>
          <div className={styles.statusItem}>
            <span className={styles.statusLabel}>Role:</span>
            <span className={`${styles.statusValue} ${styles[role?.toLowerCase()]}`}>
              {userInfo.displayName}
            </span>
          </div>
          <div className={styles.statusItem}>
            <span className={styles.statusLabel}>Network:</span>
            <span className={styles.statusValue}>{userInfo.networkDisplay}</span>
          </div>
          <div className={styles.statusItem}>
            <span className={styles.statusLabel}>Authenticated:</span>
            <span className={`${styles.statusValue} ${isAuthenticated ? styles.authenticated : styles.notAuthenticated}`}>
              {isAuthenticated ? 'Yes' : 'No'}
            </span>
          </div>
          <div className={styles.statusItem}>
            <span className={styles.statusLabel}>Demo Mode:</span>
            <span className={styles.statusValue}>
              {localStorage.getItem('knirv_demo_mode') === 'true' ? 'Enabled' : 'Disabled'}
            </span>
          </div>
        </div>
      </GlassyCard>

      {/* Page Access Testing */}
      <GlassyCard className={styles.accessCard}>
        <h3>Page Access Testing</h3>
        <p>Test which pages your current role can access. Green = Allowed, Red = Denied</p>
        
        <div className={styles.accessSection}>
          <h4>Allowed Pages ({allowedPages.length})</h4>
          <div className={styles.pageGrid}>
            {allowedPages.map(page => (
              <button
                key={page}
                className={`${styles.pageButton} ${styles.allowed}`}
                onClick={() => testPageAccess(page)}
                title={`Navigate to ${page}`}
              >
                {page}
              </button>
            ))}
          </div>
        </div>

        <div className={styles.accessSection}>
          <h4>Denied Pages ({deniedPages.length})</h4>
          <div className={styles.pageGrid}>
            {deniedPages.map(page => (
              <button
                key={page}
                className={`${styles.pageButton} ${styles.denied}`}
                onClick={() => testPageAccess(page)}
                title={`Try to navigate to ${page} (should be blocked)`}
              >
                {page}
              </button>
            ))}
          </div>
        </div>
      </GlassyCard>

      {/* Role Comparison */}
      <GlassyCard className={styles.comparisonCard}>
        <h3>Role Permission Comparison</h3>
        <div className={styles.roleComparison}>
          {Object.entries(rolePermissions).map(([roleName, pages]) => (
            <div key={roleName} className={styles.roleColumn}>
              <h4 className={`${styles.roleHeader} ${styles[roleName.toLowerCase()]}`}>
                {roleName} ({pages.length} pages)
              </h4>
              <div className={styles.permissionList}>
                {pages.map(page => (
                  <div
                    key={page}
                    className={`${styles.permissionItem} ${role === roleName ? styles.current : ''}`}
                  >
                    {page}
                  </div>
                ))}
              </div>
            </div>
          ))}
        </div>
      </GlassyCard>

      {/* Authentication History */}
      <GlassyCard className={styles.historyCard}>
        <h3>Authentication History</h3>
        <div className={styles.historyList}>
          {authHistory.map((entry, index) => (
            <div key={index} className={styles.historyEntry}>
              <div className={styles.historyTime}>{entry.timestamp}</div>
              <div className={styles.historyDetails}>
                <span className={`${styles.historyRole} ${styles[entry.role?.toLowerCase()]}`}>
                  {entry.role}
                </span>
                <span className={styles.historyNetwork}>{entry.network}</span>
                <span className={`${styles.historyAuth} ${entry.isAuthenticated ? styles.authenticated : styles.notAuthenticated}`}>
                  {entry.isAuthenticated ? 'Auth' : 'No Auth'}
                </span>
              </div>
            </div>
          ))}
        </div>
      </GlassyCard>

      {/* Instructions */}
      <GlassyCard className={styles.instructionsCard}>
        <h3>Testing Instructions</h3>
        <div className={styles.instructions}>
          <div className={styles.instruction}>
            <strong>1. Role Switching:</strong> Use the role switcher in the top-right corner to test different roles
          </div>
          <div className={styles.instruction}>
            <strong>2. Page Access:</strong> Click on page buttons above to test navigation permissions
          </div>
          <div className={styles.instruction}>
            <strong>3. Authentication:</strong> Use "Clear Auth & Show Login" to test the login screen
          </div>
          <div className={styles.instruction}>
            <strong>4. Network Testing:</strong> Switch between different networks to test connectivity
          </div>
        </div>
      </GlassyCard>
    </PageLayout>
  );
}
