import React, { useState } from 'react';
import { useNavigation } from '../hooks/useNavigation';
import PageLayout from '../components/PageLayout';
import PageHeader from '../components/PageHeader';
import GlassyCard from '../components/GlassyCard';
import DataTable from '../components/DataTable';
import OnboardingFlow from '../components/OnboardingFlowUpdated';
import styles from './inventory.module.css';

export default function Inventory() {
  const { activePage } = useNavigation('inventory');
  const [showOnboarding, setShowOnboarding] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');

  const handleOnboardingComplete = () => {
    setShowOnboarding(false);
    // You could add additional logic here, like refreshing the inventory data
  };

  const handleSearch = (query) => {
    setSearchQuery(query);
  };

  // Example data (replace with your actual data)
  const assetsData = [
    { name: 'Bitcoin', type: 'Cryptocurrency', balance: '0.45 BTC', value: '$20,534.40', change: '+2.3%' },
    { name: 'Ethereum', type: 'Cryptocurrency', balance: '2.5 ETH', value: '$8,112.50', change: '-1.1%' },
    { name: 'USDC', type: 'Stablecoin', balance: '1,500 USDC', value: '$1,500.00', change: '+0.01%' },
    { name: 'Tether', type: 'Stablecoin', balance: '2,000 USDT', value: '$2,000.00', change: '-0.02%' },
  ];

  // Filter the data based on the search query
  const filteredAssetsData = assetsData.filter((asset) =>
    asset.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
    asset.type.toLowerCase().includes(searchQuery.toLowerCase())
  );

  // Table headers
  const tableHeaders = [
    { label: 'Asset' },
    { label: 'Type' },
    { label: 'Balance', align: 'right' },
    { label: 'Value (USD)', align: 'right' },
    { label: '24h Change', align: 'right' },
    { label: 'Actions', align: 'center' },
  ];

  // Render table row
  const renderTableRow = (asset, index) => (
    <tr className={styles.tableRow} key={index}>
      <td className={styles.tableCell}>
        <div className={styles.assetNameContainer}>
          <div className={`${styles.assetIcon} ${styles[asset.name.toLowerCase()]}`}>
            {asset.name.charAt(0)}
          </div>
          <div>{asset.name}</div>
        </div>
      </td>
      <td className={styles.tableCell}>{asset.type}</td>
      <td className={`${styles.tableCell} ${styles.textRight}`}>{asset.balance}</td>
      <td className={`${styles.tableCell} ${styles.textRight}`}>{asset.value}</td>
      <td className={`${styles.tableCell} ${styles.textRight} ${asset.change.startsWith('+') ? styles.positiveChange : styles.negativeChange}`}>{asset.change}</td>
      <td className={`${styles.tableCell} ${styles.textCenter}`}>
        <div className={styles.actionButtons}>
          <button className={`${styles.actionButton} ${styles.primary}`}>Send</button>
          <button className={`${styles.actionButton} ${styles.secondary}`}>Receive</button>
        </div>
      </td>
    </tr>
  );

  return (
    <PageLayout 
      activePage={activePage} 
      pageTitle="Inventory Management" 
      onSearch={handleSearch}
    >
      {showOnboarding && <OnboardingFlow onComplete={handleOnboardingComplete} />}
      
      <PageHeader 
        title="Digital Asset Inventory" 
        subtitle="Manage your blockchain assets" 
      />

      {/* Search and Filter */}
      <GlassyCard darker className={styles.searchFilterContainer}>
        <div className={styles.filterControls}>
          <select className={`${styles.filterSelect} ${styles.glassyContainer}`}>
            <option value="all">All Assets</option>
            <option value="crypto">Cryptocurrencies</option>
            <option value="nft">NFTs</option>
            <option value="tokens">Tokens</option>
          </select>

          <button
            onClick={() => setShowOnboarding(true)}
            className={styles.addAssetButton}
          >
            Add Asset
          </button>
        </div>
      </GlassyCard>

      {/* Asset Table */}
      <GlassyCard title="Your Assets" darker className={styles.assetTableContainer}>
        <DataTable 
          headers={tableHeaders} 
          data={filteredAssetsData} 
          renderRow={renderTableRow} 
        />
      </GlassyCard>

      {/* Portfolio Summary */}
      <GlassyCard title="Portfolio Summary" darker className={styles.portfolioSummary}>
        <div className={styles.summaryCards}>
          <GlassyCard darker className={styles.summaryCard}>
            <h4 className={styles.cardTitle}>Total Value</h4>
            <h2 className={styles.cardValue}>$32,146.90</h2>
            <span className={styles.positiveChange}>+1.8% (24h)</span>
          </GlassyCard>

          <GlassyCard darker className={styles.summaryCard}>
            <h4 className={styles.cardTitle}>Asset Count</h4>
            <h2 className={styles.cardValue}>4</h2>
            <span>Across 3 blockchains</span>
          </GlassyCard>

          <GlassyCard darker className={styles.summaryCard}>
            <h4 className={styles.cardTitle}>Allocation</h4>
            <div className={styles.allocationList}>
              <div className={styles.allocationItem}>
                <span>Bitcoin</span>
                <span>63.9%</span>
              </div>
              <div className={styles.allocationItem}>
                <span>Ethereum</span>
                <span>25.2%</span>
              </div>
              <div className={styles.allocationItem}>
                <span>Stablecoins</span>
                <span>10.9%</span>
              </div>
            </div>
          </GlassyCard>
        </div>
      </GlassyCard>
    </PageLayout>
  );
}
