import React, { useState } from 'react';
import { useNavigation } from '../hooks/useNavigation';
import PageLayout from '../components/PageLayout';
import PageHeader from '../components/PageHeader';
import GlassyCard from '../components/GlassyCard';
import styles from './vault.module.css';

export default function Vault() {
  const { activePage } = useNavigation('vault');
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedTab, setSelectedTab] = useState('models');
  const [knirvControllerConnected, setKnirvControllerConnected] = useState(false);

  const handleSearch = (query) => {
    setSearchQuery(query);
  };

  const connectKNIRVController = () => {
    // This would trigger QR code modal for KNIRVCONTROLLER connection
    setKnirvControllerConnected(true);
  };

  const openInModal = (type) => {
    // This would open the respective agent-developer-portal page in a modal
    console.log(`Opening ${type} in modal`);
  };

  // Mock data for different asset types
  const mockData = {
    models: [
      { id: '1', name: 'GPT-4 Custom', type: 'Language Model', status: 'Active', performance: '94.5%' },
      { id: '2', name: 'Vision Classifier', type: 'Computer Vision', status: 'Training', performance: '89.2%' }
    ],
    wallets: [
      { id: '1', name: 'Main Wallet', address: '0x1234...abcd', balance: '1,250 NRN', type: 'Primary' },
      { id: '2', name: 'Trading Wallet', address: '0x5678...efgh', balance: '450 NRN', type: 'Secondary' }
    ],
    skills: [
      { id: '1', name: 'Code Generation', category: 'Development', level: 'Expert', usage: '1,234 calls' },
      { id: '2', name: 'Image Analysis', category: 'Computer Vision', level: 'Advanced', usage: '567 calls' }
    ],
    capabilities: [
      { id: '1', name: 'API Integration', type: 'Service', status: 'Active', connections: '12' },
      { id: '2', name: 'Data Processing', type: 'Compute', status: 'Idle', connections: '5' }
    ],
    properties: [
      { id: '1', name: 'Training Dataset #1', type: 'Data', size: '2.4 GB', value: '150 NRN' },
      { id: '2', name: 'Model Weights #1', type: 'Model', size: '1.8 GB', value: '300 NRN' }
    ]
  };

  const tabs = [
    { id: 'models', label: 'My Models', icon: '🤖', count: mockData.models.length },
    { id: 'wallets', label: 'My Wallets', icon: '💰', count: mockData.wallets.length },
    { id: 'skills', label: 'My Skills', icon: '⚡', count: mockData.skills.length },
    { id: 'capabilities', label: 'My Capabilities', icon: '🔌', count: mockData.capabilities.length },
    { id: 'properties', label: 'My Properties', icon: '🏷️', count: mockData.properties.length }
  ];

  const renderTabContent = () => {
    const data = mockData[selectedTab] || [];

    if (!knirvControllerConnected) {
      return (
        <div className={styles.connectionPrompt}>
          <div className={styles.connectionIcon}>📱</div>
          <h3>Connect KNIRVCONTROLLER</h3>
          <p>Connect your KNIRVCONTROLLER to manage your personal assets, or access them through the web interface.</p>
          <div className={styles.connectionActions}>
            <button onClick={connectKNIRVController} className={styles.connectButton}>
              <i className="fas fa-qrcode" style={{ marginRight: '8px' }}></i>
              Connect via QR Code
            </button>
            <button onClick={() => openInModal(selectedTab)} className={styles.webAccessButton}>
              <i className="fas fa-external-link-alt" style={{ marginRight: '8px' }}></i>
              Access via Web
            </button>
          </div>
        </div>
      );
    }

    if (data.length === 0) {
      return (
        <div className={styles.emptyState}>
          <div className={styles.emptyIcon}>{tabs.find(t => t.id === selectedTab)?.icon}</div>
          <h3>No {selectedTab} found</h3>
          <p>You don't have any {selectedTab} yet. Start by creating or acquiring some.</p>
        </div>
      );
    }

    return (
      <div className={styles.assetGrid}>
        {data.map((item) => (
          <GlassyCard key={item.id} className={styles.assetCard}>
            <div className={styles.assetHeader}>
              <h4 className={styles.assetName}>{item.name}</h4>
              <div className={styles.assetActions}>
                <button className={styles.actionButton}>
                  <i className="fas fa-eye"></i>
                </button>
                <button className={styles.actionButton}>
                  <i className="fas fa-edit"></i>
                </button>
                <button className={styles.actionButton}>
                  <i className="fas fa-ellipsis-v"></i>
                </button>
              </div>
            </div>

            <div className={styles.assetDetails}>
              {selectedTab === 'models' && (
                <>
                  <div className={styles.assetDetail}>
                    <span className={styles.detailLabel}>Type:</span>
                    <span className={styles.detailValue}>{item.type}</span>
                  </div>
                  <div className={styles.assetDetail}>
                    <span className={styles.detailLabel}>Status:</span>
                    <span className={`${styles.detailValue} ${styles[item.status.toLowerCase()]}`}>
                      {item.status}
                    </span>
                  </div>
                  <div className={styles.assetDetail}>
                    <span className={styles.detailLabel}>Performance:</span>
                    <span className={styles.detailValue}>{item.performance}</span>
                  </div>
                </>
              )}

              {selectedTab === 'wallets' && (
                <>
                  <div className={styles.assetDetail}>
                    <span className={styles.detailLabel}>Address:</span>
                    <span className={styles.detailValue}>{item.address}</span>
                  </div>
                  <div className={styles.assetDetail}>
                    <span className={styles.detailLabel}>Balance:</span>
                    <span className={styles.detailValue}>{item.balance}</span>
                  </div>
                  <div className={styles.assetDetail}>
                    <span className={styles.detailLabel}>Type:</span>
                    <span className={styles.detailValue}>{item.type}</span>
                  </div>
                </>
              )}

              {selectedTab === 'skills' && (
                <>
                  <div className={styles.assetDetail}>
                    <span className={styles.detailLabel}>Category:</span>
                    <span className={styles.detailValue}>{item.category}</span>
                  </div>
                  <div className={styles.assetDetail}>
                    <span className={styles.detailLabel}>Level:</span>
                    <span className={styles.detailValue}>{item.level}</span>
                  </div>
                  <div className={styles.assetDetail}>
                    <span className={styles.detailLabel}>Usage:</span>
                    <span className={styles.detailValue}>{item.usage}</span>
                  </div>
                </>
              )}

              {selectedTab === 'capabilities' && (
                <>
                  <div className={styles.assetDetail}>
                    <span className={styles.detailLabel}>Type:</span>
                    <span className={styles.detailValue}>{item.type}</span>
                  </div>
                  <div className={styles.assetDetail}>
                    <span className={styles.detailLabel}>Status:</span>
                    <span className={`${styles.detailValue} ${styles[item.status.toLowerCase()]}`}>
                      {item.status}
                    </span>
                  </div>
                  <div className={styles.assetDetail}>
                    <span className={styles.detailLabel}>Connections:</span>
                    <span className={styles.detailValue}>{item.connections}</span>
                  </div>
                </>
              )}

              {selectedTab === 'properties' && (
                <>
                  <div className={styles.assetDetail}>
                    <span className={styles.detailLabel}>Type:</span>
                    <span className={styles.detailValue}>{item.type}</span>
                  </div>
                  <div className={styles.assetDetail}>
                    <span className={styles.detailLabel}>Size:</span>
                    <span className={styles.detailValue}>{item.size}</span>
                  </div>
                  <div className={styles.assetDetail}>
                    <span className={styles.detailLabel}>Value:</span>
                    <span className={styles.detailValue}>{item.value}</span>
                  </div>
                </>
              )}
            </div>
          </GlassyCard>
        ))}
      </div>
    );
  };

  return (
    <PageLayout activePage={activePage} pageTitle="Vault" onSearch={handleSearch}>
      <PageHeader
        title="Personal Vault"
        subtitle="Manage your models, wallets, skills, capabilities, and properties"
        titleColor="#007bff"
      />

      {/* Connection Status */}
      <GlassyCard className={styles.statusCard}>
        <div className={styles.statusHeader}>
          <h3 className={styles.statusTitle}>
            <i className="fas fa-mobile-alt" style={{ marginRight: '10px', color: '#007bff' }}></i>
            KNIRVCONTROLLER Connection
          </h3>
          <div className={`${styles.connectionStatus} ${knirvControllerConnected ? styles.connected : styles.disconnected}`}>
            {knirvControllerConnected ? 'Connected' : 'Disconnected'}
          </div>
        </div>
        <p className={styles.statusDescription}>
          {knirvControllerConnected
            ? 'Your KNIRVCONTROLLER is connected. You can manage all your assets directly.'
            : 'Connect your KNIRVCONTROLLER for full asset management capabilities, or use web access for basic operations.'
          }
        </p>
      </GlassyCard>

      {/* Tabs */}
      <div className={styles.tabsContainer}>
        {tabs.map((tab) => (
          <button
            key={tab.id}
            onClick={() => setSelectedTab(tab.id)}
            className={`${styles.tab} ${selectedTab === tab.id ? styles.activeTab : ''}`}
          >
            <span className={styles.tabIcon}>{tab.icon}</span>
            <span className={styles.tabLabel}>{tab.label}</span>
            <span className={styles.tabCount}>({tab.count})</span>
          </button>
        ))}
      </div>

      {/* Tab Content */}
      <GlassyCard className={styles.contentCard}>
        {renderTabContent()}
      </GlassyCard>
    </PageLayout>
  );
}