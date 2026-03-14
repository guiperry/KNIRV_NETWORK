import React, { useState, useEffect } from 'react';
import { useNavigation } from '../hooks/useNavigation';
import PageLayout from '../components/PageLayout';
import PageHeader from '../components/PageHeader';
import GlassyCard from '../components/GlassyCard';
import DataTable from '../components/DataTable';
import styles from './my-wallets.module.css';

export default function MyWallets() {
  const { activePage } = useNavigation('my-wallets');
  const [searchQuery, setSearchQuery] = useState('');
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [wallets, setWallets] = useState([]);
  const [selectedWallet, setSelectedWallet] = useState(null);

  const handleSearch = (query) => {
    setSearchQuery(query);
  };

  // Mock wallets data
  useEffect(() => {
    const mockWallets = [
      {
        id: '1',
        name: 'Main Wallet',
        address: '0x1234567890abcdef1234567890abcdef12345678',
        type: 'Primary',
        balance: {
          NRN: 1250.75,
          ETH: 2.45,
          BTC: 0.125
        },
        status: 'Active',
        createdAt: '2024-01-15',
        lastActivity: '2 hours ago',
        transactions: 156,
        isDefault: true
      },
      {
        id: '2',
        name: 'Trading Wallet',
        address: '0xabcdef1234567890abcdef1234567890abcdef12',
        type: 'Secondary',
        balance: {
          NRN: 450.25,
          ETH: 1.2,
          USDC: 1500.0
        },
        status: 'Active',
        createdAt: '2024-01-10',
        lastActivity: '1 day ago',
        transactions: 89,
        isDefault: false
      },
      {
        id: '3',
        name: 'Cold Storage',
        address: '0x9876543210fedcba9876543210fedcba98765432',
        type: 'Hardware',
        balance: {
          BTC: 1.5,
          ETH: 5.0,
          NRN: 2000.0
        },
        status: 'Offline',
        createdAt: '2023-12-20',
        lastActivity: '1 week ago',
        transactions: 23,
        isDefault: false
      }
    ];
    setWallets(mockWallets);
  }, []);

  // Filter wallets based on search query
  const filteredWallets = wallets.filter((wallet) =>
    wallet.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
    wallet.address.toLowerCase().includes(searchQuery.toLowerCase()) ||
    wallet.type.toLowerCase().includes(searchQuery.toLowerCase())
  );

  // Table headers
  const tableHeaders = [
    { label: 'Wallet Name' },
    { label: 'Address' },
    { label: 'Type' },
    { label: 'Balance (NRN)', align: 'right' },
    { label: 'Status', align: 'center' },
    { label: 'Transactions', align: 'right' },
    { label: 'Actions', align: 'center' },
  ];

  // Render table row
  const renderTableRow = (wallet, index) => (
    <tr className={styles.tableRow} key={wallet.id}>
      <td className={styles.tableCell}>
        <div className={styles.walletInfo}>
          <div className={styles.walletName}>
            {wallet.name}
            {wallet.isDefault && <span className={styles.defaultBadge}>Default</span>}
          </div>
          <div className={styles.walletType}>{wallet.type} Wallet</div>
        </div>
      </td>
      <td className={styles.tableCell}>
        <div className={styles.addressContainer}>
          <code className={styles.address}>{wallet.address.slice(0, 10)}...{wallet.address.slice(-8)}</code>
          <button className={styles.copyButton} title="Copy address">
            <i className="fas fa-copy"></i>
          </button>
        </div>
      </td>
      <td className={styles.tableCell}>
        <span className={`${styles.type} ${styles[wallet.type.toLowerCase()]}`}>
          {wallet.type}
        </span>
      </td>
      <td className={`${styles.tableCell} ${styles.textRight}`}>
        <span className={styles.balance}>
          {wallet.balance.NRN ? wallet.balance.NRN.toLocaleString() : '0'} NRN
        </span>
      </td>
      <td className={`${styles.tableCell} ${styles.textCenter}`}>
        <span className={`${styles.status} ${styles[wallet.status.toLowerCase()]}`}>
          {wallet.status}
        </span>
      </td>
      <td className={`${styles.tableCell} ${styles.textRight}`}>
        <span className={styles.transactions}>{wallet.transactions}</span>
      </td>
      <td className={`${styles.tableCell} ${styles.textCenter}`}>
        <div className={styles.actionButtons}>
          <button 
            onClick={() => setSelectedWallet(wallet)}
            className={styles.viewButton}
            title="View Details"
          >
            <i className="fas fa-eye"></i>
          </button>
          <button 
            className={styles.sendButton}
            title="Send Funds"
          >
            <i className="fas fa-paper-plane"></i>
          </button>
          <button 
            className={styles.receiveButton}
            title="Receive Funds"
          >
            <i className="fas fa-download"></i>
          </button>
        </div>
      </td>
    </tr>
  );

  const handleCreateWallet = () => {
    setShowCreateModal(true);
  };

  const getTotalBalance = () => {
    return wallets.reduce((total, wallet) => {
      return total + (wallet.balance.NRN || 0);
    }, 0);
  };

  const getTotalTransactions = () => {
    return wallets.reduce((total, wallet) => total + wallet.transactions, 0);
  };

  return (
    <PageLayout activePage={activePage} pageTitle="My Wallets" onSearch={handleSearch}>
      <PageHeader 
        title="My Wallets"
        subtitle="Manage your cryptocurrency wallets and digital assets"
        titleColor="#007bff"
      />

      {/* Statistics Overview */}
      <div className={styles.statsContainer}>
        <GlassyCard className={styles.statCard}>
          <div className={styles.statIcon}>💰</div>
          <div className={styles.statValue}>{wallets.length}</div>
          <div className={styles.statLabel}>Total Wallets</div>
        </GlassyCard>
        <GlassyCard className={styles.statCard}>
          <div className={styles.statIcon}>🟢</div>
          <div className={styles.statValue}>{wallets.filter(w => w.status === 'Active').length}</div>
          <div className={styles.statLabel}>Active Wallets</div>
        </GlassyCard>
        <GlassyCard className={styles.statCard}>
          <div className={styles.statIcon}>💎</div>
          <div className={styles.statValue}>{getTotalBalance().toLocaleString()} NRN</div>
          <div className={styles.statLabel}>Total Balance</div>
        </GlassyCard>
        <GlassyCard className={styles.statCard}>
          <div className={styles.statIcon}>📊</div>
          <div className={styles.statValue}>{getTotalTransactions()}</div>
          <div className={styles.statLabel}>Total Transactions</div>
        </GlassyCard>
      </div>

      {/* Wallets Table */}
      <GlassyCard className={styles.tableCard}>
        <div className={styles.tableHeader}>
          <h3 className={styles.tableTitle}>
            <i className="fas fa-wallet" style={{ marginRight: '10px', color: '#007bff' }}></i>
            My Wallets ({filteredWallets.length})
          </h3>
          <button 
            onClick={handleCreateWallet}
            className={styles.createButton}
          >
            <i className="fas fa-plus" style={{ marginRight: '8px' }}></i>
            Create New Wallet
          </button>
        </div>
        
        {filteredWallets.length > 0 ? (
          <DataTable
            headers={tableHeaders}
            data={filteredWallets}
            renderRow={renderTableRow}
            className={styles.walletsTable}
          />
        ) : (
          <div className={styles.emptyState}>
            <i className="fas fa-wallet" style={{ fontSize: '3rem', color: '#666', marginBottom: '20px' }}></i>
            <h3>No wallets found</h3>
            <p>Create your first wallet to start managing your digital assets.</p>
            <button onClick={handleCreateWallet} className={styles.createButton}>
              <i className="fas fa-plus" style={{ marginRight: '8px' }}></i>
              Create Your First Wallet
            </button>
          </div>
        )}
      </GlassyCard>

      {/* Wallet Details Modal */}
      {selectedWallet && (
        <div className={styles.modal}>
          <div className={styles.modalContent}>
            <div className={styles.modalHeader}>
              <h3>{selectedWallet.name}</h3>
              <button onClick={() => setSelectedWallet(null)} className={styles.closeButton}>×</button>
            </div>
            <div className={styles.modalBody}>
              <div className={styles.walletDetails}>
                <div className={styles.detailSection}>
                  <h4>Wallet Information</h4>
                  <div className={styles.detailGrid}>
                    <div className={styles.detailItem}>
                      <span className={styles.detailLabel}>Address:</span>
                      <div className={styles.addressDetail}>
                        <code className={styles.fullAddress}>{selectedWallet.address}</code>
                        <button className={styles.copyButton} title="Copy address">
                          <i className="fas fa-copy"></i>
                        </button>
                      </div>
                    </div>
                    <div className={styles.detailItem}>
                      <span className={styles.detailLabel}>Type:</span>
                      <span className={styles.detailValue}>{selectedWallet.type}</span>
                    </div>
                    <div className={styles.detailItem}>
                      <span className={styles.detailLabel}>Status:</span>
                      <span className={`${styles.detailValue} ${styles[selectedWallet.status.toLowerCase()]}`}>
                        {selectedWallet.status}
                      </span>
                    </div>
                    <div className={styles.detailItem}>
                      <span className={styles.detailLabel}>Created:</span>
                      <span className={styles.detailValue}>{selectedWallet.createdAt}</span>
                    </div>
                  </div>
                </div>

                <div className={styles.detailSection}>
                  <h4>Asset Balances</h4>
                  <div className={styles.balanceGrid}>
                    {Object.entries(selectedWallet.balance).map(([asset, amount]) => (
                      <div key={asset} className={styles.balanceCard}>
                        <div className={styles.assetSymbol}>{asset}</div>
                        <div className={styles.assetAmount}>{amount.toLocaleString()}</div>
                      </div>
                    ))}
                  </div>
                </div>

                <div className={styles.detailSection}>
                  <h4>Activity Summary</h4>
                  <div className={styles.activityGrid}>
                    <div className={styles.activityCard}>
                      <div className={styles.activityValue}>{selectedWallet.transactions}</div>
                      <div className={styles.activityLabel}>Total Transactions</div>
                    </div>
                    <div className={styles.activityCard}>
                      <div className={styles.activityValue}>{selectedWallet.lastActivity}</div>
                      <div className={styles.activityLabel}>Last Activity</div>
                    </div>
                  </div>
                </div>

                <div className={styles.walletActions}>
                  <button className={styles.sendButton}>
                    <i className="fas fa-paper-plane" style={{ marginRight: '8px' }}></i>
                    Send Funds
                  </button>
                  <button className={styles.receiveButton}>
                    <i className="fas fa-download" style={{ marginRight: '8px' }}></i>
                    Receive Funds
                  </button>
                  <button className={styles.exportButton}>
                    <i className="fas fa-download" style={{ marginRight: '8px' }}></i>
                    Export Keys
                  </button>
                </div>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Create Wallet Modal */}
      {showCreateModal && (
        <div className={styles.modal}>
          <div className={styles.modalContent}>
            <div className={styles.modalHeader}>
              <h3>Create New Wallet</h3>
              <button onClick={() => setShowCreateModal(false)} className={styles.closeButton}>×</button>
            </div>
            <div className={styles.modalBody}>
              <p>Wallet creation functionality would be implemented here, integrating with the KNIRV wallet management system.</p>
              <div className={styles.modalActions}>
                <button onClick={() => setShowCreateModal(false)} className={styles.cancelButton}>
                  Cancel
                </button>
                <button className={styles.createButton}>
                  Create Wallet
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
    </PageLayout>
  );
}
