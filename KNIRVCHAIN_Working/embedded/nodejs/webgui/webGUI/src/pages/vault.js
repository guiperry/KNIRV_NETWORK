import React, { useState } from 'react';
import { useNavigation } from '../hooks/useNavigation';
import PageLayout from '../components/PageLayout';
import PageHeader from '../components/PageHeader';
import GlassyCard from '../components/GlassyCard';
import DataTable from '../components/DataTable';
import styles from './vault.module.css';

export default function Vault() {
  const { activePage } = useNavigation('vault');
  const [searchQuery, setSearchQuery] = useState('');

  const handleSearch = (query) => {
    setSearchQuery(query);
  };

  // Example vault data
  const vaultsData = [
    { 
      name: 'Cold Storage', 
      status: 'Active',
      description: 'Offline storage for long-term holdings',
      details: [
        { label: 'Bitcoin:', value: '1.25 BTC' },
        { label: 'Ethereum:', value: '15.5 ETH' },
        { label: 'Total Value:', value: '$113,450.00' }
      ]
    },
    { 
      name: 'Hardware Wallet', 
      status: 'Connected',
      description: 'Secure hardware wallet integration',
      details: [
        { label: 'Device:', value: 'Ledger Nano X' },
        { label: 'Firmware:', value: '2.0.0' },
        { label: 'Last Sync:', value: '2 hours ago' }
      ]
    },
    { 
      name: 'Multi-Signature', 
      status: '2 of 3',
      description: 'Enhanced security with multiple signatures',
      details: [
        { label: 'Signers:', value: '3 configured' },
        { label: 'Threshold:', value: '2 signatures' },
        { label: 'Pending Txs:', value: '1' }
      ]
    }
  ];

  // Recent activity data
  const activityData = [
    { 
      date: '2023-06-15 14:32',
      action: 'Deposit',
      vault: 'Cold Storage',
      asset: 'Bitcoin',
      amount: '0.25 BTC',
      status: 'Completed'
    },
    { 
      date: '2023-06-14 09:17',
      action: 'Withdraw',
      vault: 'Hardware Wallet',
      asset: 'Ethereum',
      amount: '1.5 ETH',
      status: 'Completed'
    },
    { 
      date: '2023-06-13 18:45',
      action: 'Transfer',
      vault: 'Multi-Signature',
      asset: 'USDC',
      amount: '5,000 USDC',
      status: 'Pending (1/2)'
    },
    { 
      date: '2023-06-10 11:23',
      action: 'Backup',
      vault: 'Cold Storage',
      asset: 'All Assets',
      amount: '-',
      status: 'Completed'
    }
  ];

  // Table headers for activity
  const activityHeaders = [
    { label: 'Date' },
    { label: 'Action' },
    { label: 'Vault' },
    { label: 'Asset' },
    { label: 'Amount', align: 'right' },
    { label: 'Status', align: 'center' }
  ];

  // Render activity row
  const renderActivityRow = (item, index) => (
    <tr key={index}>
      <td>{item.date}</td>
      <td>{item.action}</td>
      <td>{item.vault}</td>
      <td>{item.asset}</td>
      <td style={{ textAlign: 'right' }}>{item.amount}</td>
      <td style={{ textAlign: 'center' }}>
        <span className={item.status.includes('Completed') ? styles.statusBadgeCompleted : styles.statusBadgePending}>
          {item.status}
        </span>
      </td>
    </tr>
  );

  return (
    <PageLayout activePage={activePage} pageTitle="Secure Vault" onSearch={handleSearch}>
      <PageHeader 
        title="Vault Management" 
        subtitle="Secure storage for your most valuable assets" 
      />

      {/* Security Status */}
      <GlassyCard darker className={styles.statusCard}>
        <div className={styles.statusContent}>
          <h3 className={styles.sectionTitle}>Security Status</h3>
          <div className={styles.statusIndicator}>
            <div className={`${styles.statusDot} ${styles.statusActive}`}></div>
            <span>All systems secure</span>
          </div>
        </div>

        <button className={`${styles.button} ${styles.primary}`}>
          Security Audit
        </button>
      </GlassyCard>

      {/* Vault Cards */}
      <div className={styles.vaultCardsContainer}>
        {vaultsData.map((vault, index) => (
          <GlassyCard key={index} darker className={styles.vaultCard}>
            <div className={styles.vaultCardHeader}>
              <h3 className={styles.vaultCardTitle}>{vault.name}</h3>
              <span className={`${styles.statusBadge} ${
                vault.status === 'Active' || vault.status === 'Connected' 
                  ? styles.statusActive 
                  : styles.statusWarning
              }`}>
                {vault.status}
              </span>
            </div>

            <p className={styles.vaultCardDescription}>{vault.description}</p>

            <div className={`${styles.vaultCardContent} ${styles.glassyContainer}`}>
              {vault.details.map((detail, i) => (
                <div key={i} className={styles.infoRow}>
                  <span>{detail.label}</span>
                  <span>{detail.value}</span>
                </div>
              ))}
            </div>

            <div className={styles.buttonGroup}>
              <button className={`${styles.button} ${styles.primary}`}>
                {vault.name === 'Multi-Signature' ? 'Sign Tx' : vault.name === 'Hardware Wallet' ? 'Sync Now' : 'Deposit'}
              </button>

              <button className={`${styles.button} ${styles.secondary}`}>
                {vault.name === 'Multi-Signature' ? 'Manage' : vault.name === 'Hardware Wallet' ? 'Disconnect' : 'Withdraw'}
              </button>
            </div>
          </GlassyCard>
        ))}
      </div>

      {/* Recent Activity */}
      <GlassyCard darker className={styles.activitySection}>
        <h3 className={styles.sectionTitle}>Recent Vault Activity</h3>
        <DataTable 
          headers={activityHeaders}
          data={activityData}
          renderRow={renderActivityRow}
        />
      </GlassyCard>
    </PageLayout>
  );
}