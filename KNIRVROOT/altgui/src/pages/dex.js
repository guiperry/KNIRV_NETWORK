import React, { useState } from 'react';
import { useNavigation } from '../hooks/useNavigation';
import PageLayout from '../components/PageLayout';
import PageHeader from '../components/PageHeader';
import GlassyCard from '../components/GlassyCard';
import DataTable from '../components/DataTable';
import styles from './dex.module.css';

export default function DEX() {
  const { activePage } = useNavigation('dex');
  const [searchQuery, setSearchQuery] = useState('');

  const handleSearch = (query) => {
    setSearchQuery(query);
  };

  // Example trading pairs data
  const tradingPairsData = [
    { pair: 'BTC/USDT', price: '$45,632.80', change: '+2.3%', volume: '$1.2B', high: '$46,120.50', low: '$44,890.20' },
    { pair: 'ETH/USDT', price: '$3,245.75', change: '-1.1%', volume: '$820M', high: '$3,310.25', low: '$3,180.90' },
    { pair: 'SOL/USDT', price: '$120.45', change: '+5.7%', volume: '$450M', high: '$122.80', low: '$115.20' },
    { pair: 'ADA/USDT', price: '$0.58', change: '+0.8%', volume: '$210M', high: '$0.59', low: '$0.57' },
  ];

  // Filter the data based on the search query
  const filteredTradingPairsData = tradingPairsData.filter((pair) =>
    pair.pair.toLowerCase().includes(searchQuery.toLowerCase())
  );

  // Table headers
  const tableHeaders = [
    { label: 'Trading Pair' },
    { label: 'Price', align: 'right' },
    { label: '24h Change', align: 'right' },
    { label: '24h Volume', align: 'right' },
    { label: '24h High', align: 'right' },
    { label: '24h Low', align: 'right' },
    { label: 'Actions', align: 'center' },
  ];

  // Render table row
  const renderTableRow = (pair, index) => (
    <tr className={styles.tableRow} key={index}>
      <td className={styles.tableCell}>
        <div className={styles.pairNameContainer}>
          <div className={styles.pairIcon}>
            {pair.pair.split('/')[0].charAt(0)}
          </div>
          <div>{pair.pair}</div>
        </div>
      </td>
      <td className={`${styles.tableCell} ${styles.textRight}`}>{pair.price}</td>
      <td className={`${styles.tableCell} ${styles.textRight} ${pair.change.startsWith('+') ? styles.positiveChange : styles.negativeChange}`}>{pair.change}</td>
      <td className={`${styles.tableCell} ${styles.textRight}`}>{pair.volume}</td>
      <td className={`${styles.tableCell} ${styles.textRight}`}>{pair.high}</td>
      <td className={`${styles.tableCell} ${styles.textRight}`}>{pair.low}</td>
      <td className={`${styles.tableCell} ${styles.textCenter}`}>
        <div className={styles.actionButtons}>
          <button className={`${styles.actionButton} ${styles.primary}`}>Trade</button>
          <button className={`${styles.actionButton} ${styles.secondary}`}>Details</button>
        </div>
      </td>
    </tr>
  );

  return (
    <PageLayout 
      activePage={activePage} 
      pageTitle="Decentralized Exchange" 
      onSearch={handleSearch}
    >
      <PageHeader 
        title="Decentralized Exchange (DEX)" 
        subtitle="Trade digital assets directly from your wallet" 
      />

      {/* Market Overview */}
      <GlassyCard title="Market Overview" darker className={styles.marketOverview}>
        <div className={styles.marketStats}>
          <div className={styles.statItem}>
            <h4 className={styles.statLabel}>24h Volume</h4>
            <h3 className={styles.statValue}>$2.68B</h3>
            <span className={styles.positiveChange}>+12.5%</span>
          </div>
          <div className={styles.statItem}>
            <h4 className={styles.statLabel}>Active Pairs</h4>
            <h3 className={styles.statValue}>24</h3>
            <span>+2 new</span>
          </div>
          <div className={styles.statItem}>
            <h4 className={styles.statLabel}>Liquidity</h4>
            <h3 className={styles.statValue}>$890M</h3>
            <span className={styles.positiveChange}>+5.2%</span>
          </div>
          <div className={styles.statItem}>
            <h4 className={styles.statLabel}>Transactions</h4>
            <h3 className={styles.statValue}>15,432</h3>
            <span className={styles.positiveChange}>+8.7%</span>
          </div>
        </div>
      </GlassyCard>

      {/* Trading Pairs */}
      <GlassyCard title="Trading Pairs" darker className={styles.tradingPairsContainer}>
        <div className={styles.filterControls}>
          <select className={`${styles.filterSelect} ${styles.glassyContainer}`}>
            <option value="all">All Pairs</option>
            <option value="btc">BTC Pairs</option>
            <option value="eth">ETH Pairs</option>
            <option value="usdt">USDT Pairs</option>
          </select>

          <button className={styles.addPairButton}>
            Add Liquidity
          </button>
        </div>

        <DataTable 
          headers={tableHeaders} 
          data={filteredTradingPairsData} 
          renderRow={renderTableRow} 
        />
      </GlassyCard>
    </PageLayout>
  );
}