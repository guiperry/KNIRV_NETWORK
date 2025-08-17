import React, { useState, useEffect } from 'react';
import OnboardingFlow from '../components/OnboardingFlowUpdated';
import { useNavigation } from '../hooks/useNavigation';
import PageLayout from '../components/PageLayout';
import PageHeader from '../components/PageHeader';
import GlassyCard from '../components/GlassyCard';
import DataTable from '../components/DataTable';
import styles from './blockchain.module.css';

export default function Blockchain({ onboardingCompleted }) {
  const { activePage } = useNavigation('blockchain');
  const [showOnboarding, setShowOnboarding] = useState(false);
  const [isBlockchainEnabled, setIsBlockchainEnabled] = useState(false);
  const [activeNetwork, setActiveNetwork] = useState('Bitcoin');
  const [searchQuery, setSearchQuery] = useState('');

  useEffect(() => {
    const hasCompletedOnboarding = localStorage.getItem('onboardingCompleted');
    if (hasCompletedOnboarding === 'true' || onboardingCompleted) {
      setIsBlockchainEnabled(true);
    } else {
      setShowOnboarding(true);
    }
  }, [onboardingCompleted]);
  
  const handleOnboardingComplete = () => {
    setShowOnboarding(false);
    setIsBlockchainEnabled(true);
    localStorage.setItem('onboardingCompleted', 'true');
  };

  const handleSearch = (query) => {
    setSearchQuery(query);
  };

  const handleNetworkChange = (network) => {
    setActiveNetwork(network);
  };

  // Latest blocks data
  const blocksData = [
    { 
      height: '789,245',
      age: '2 minutes ago',
      transactions: '2,543',
      size: '1.2 MB',
      miner: 'F2Pool',
      reward: '6.25 BTC'
    },
    { 
      height: '789,244',
      age: '12 minutes ago',
      transactions: '1,987',
      size: '1.1 MB',
      miner: 'Antpool',
      reward: '6.25 BTC'
    },
    { 
      height: '789,243',
      age: '18 minutes ago',
      transactions: '2,156',
      size: '1.3 MB',
      miner: 'Binance Pool',
      reward: '6.25 BTC'
    },
    { 
      height: '789,242',
      age: '25 minutes ago',
      transactions: '1,876',
      size: '1.0 MB',
      miner: 'ViaBTC',
      reward: '6.25 BTC'
    }
  ];

  // Table headers for blocks
  const blocksHeaders = [
    { label: 'Height' },
    { label: 'Age' },
    { label: 'Transactions' },
    { label: 'Size' },
    { label: 'Miner', align: 'right' },
    { label: 'Reward', align: 'right' }
  ];

  // Render block row
  const renderBlockRow = (block, index) => (
    <tr key={index}>
      <td><a href="#" className={styles.blockLink}>{block.height}</a></td>
      <td>{block.age}</td>
      <td>{block.transactions}</td>
      <td>{block.size}</td>
      <td style={{ textAlign: 'right' }}>
        <a href="#" className={styles.blockLink}>{block.miner}</a>
      </td>
      <td style={{ textAlign: 'right' }}>{block.reward}</td>
    </tr>
  );

  return (
    <PageLayout 
      activePage={activePage} 
      pageTitle="Blockchain Explorer" 
      onSearch={handleSearch}
      disabled={!isBlockchainEnabled}
      disabledMessage="Complete onboarding to access blockchain features"
      onEnable={() => setShowOnboarding(true)}
    >
      {showOnboarding && <OnboardingFlow onComplete={handleOnboardingComplete} />}

      <PageHeader 
        title="Blockchain Networks" 
        subtitle="Monitor and interact with blockchain networks" 
      />

      {/* Network Selection */}
      <GlassyCard darker className={styles.networkSelection}>
        <div className={styles.networkButtons}>
          <button 
            className={`${styles.networkButton} ${activeNetwork === 'Bitcoin' ? styles.primaryButton : styles.secondaryButton}`}
            onClick={() => handleNetworkChange('Bitcoin')}
          >
            Bitcoin
          </button>

          <button 
            className={`${styles.networkButton} ${activeNetwork === 'Ethereum' ? styles.primaryButton : styles.secondaryButton}`}
            onClick={() => handleNetworkChange('Ethereum')}
          >
            Ethereum
          </button>

          <button 
            className={`${styles.networkButton} ${activeNetwork === 'Solana' ? styles.primaryButton : styles.secondaryButton}`}
            onClick={() => handleNetworkChange('Solana')}
          >
            Solana
          </button>

          <button 
            className={`${styles.networkButton} ${activeNetwork === 'Cardano' ? styles.primaryButton : styles.secondaryButton}`}
            onClick={() => handleNetworkChange('Cardano')}
          >
            Cardano
          </button>
        </div>
      </GlassyCard>

      {/* Network Stats */}
      <div className={styles.statsContainer}>
        <GlassyCard darker className={styles.statCard}>
          <h3 className={styles.statTitle}>Current Price</h3>
          <h2 className={styles.statValue}>$45,632</h2>
          <span className={styles.positiveChange}>+2.3%</span>
        </GlassyCard>

        <GlassyCard darker className={styles.statCard}>
          <h3 className={styles.statTitle}>Hash Rate</h3>
          <h2 className={styles.statValue}>215 EH/s</h2>
          <span className={styles.positiveChange}>+5.7%</span>
        </GlassyCard>

        <GlassyCard darker className={styles.statCard}>
          <h3 className={styles.statTitle}>Difficulty</h3>
          <h2 className={styles.statValue}>37.59 T</h2>
          <span className={styles.positiveChange}>+0.8%</span>
        </GlassyCard>

        <GlassyCard darker className={styles.statCard}>
          <h3 className={styles.statTitle}>Block Height</h3>
          <h2 className={styles.statValue}>789,245</h2>
          <span>Latest</span>
        </GlassyCard>
      </div>

      {/* Latest Blocks */}
      <GlassyCard darker className={styles.blocksContainer}>
        <div className={styles.sectionHeader}>
          <h3 className={styles.sectionTitle}>Latest Blocks</h3>
          <button className={styles.viewAllButton}>
            View All
          </button>
        </div>

        <DataTable 
          headers={blocksHeaders}
          data={blocksData}
          renderRow={renderBlockRow}
        />
      </GlassyCard>

      {/* Mempool & Network Activity */}
      <div className={styles.flexContainer}>
        <GlassyCard darker className={styles.flexItem}>
          <h3 className={styles.sectionTitle}>Mempool Status</h3>

          <GlassyCard className={styles.mempoolInfo}>
            <div className={styles.infoRow}>
              <span>Pending Transactions:</span>
              <span>15,432</span>
            </div>
            <div className={styles.infoRow}>
              <span>Memory Usage:</span>
              <span>125 MB</span>
            </div>
            <div className={`${styles.infoRow} ${styles.lastRow}`}>
              <span>Estimated Wait Time:</span>
              <span>~15 minutes</span>
            </div>
          </GlassyCard>

          <h4 className={styles.feeRatesTitle}>Fee Rates (sat/vB)</h4>
          <div className={styles.feeRatesContainer}>
            <GlassyCard className={styles.feeRateCard}>
              <div className={styles.feeRateType}>High</div>
              <div className={styles.feeRateValue}>25</div>
              <div className={styles.feeRateTime}>~10 min</div>
            </GlassyCard>

            <GlassyCard className={styles.feeRateCard}>
              <div className={styles.feeRateType}>Medium</div>
              <div className={styles.feeRateValue}>15</div>
              <div className={styles.feeRateTime}>~30 min</div>
            </GlassyCard>

            <GlassyCard className={styles.feeRateCard}>
              <div className={styles.feeRateType}>Low</div>
              <div className={styles.feeRateValue}>5</div>
              <div className={styles.feeRateTime}>~60 min</div>
            </GlassyCard>
          </div>
        </GlassyCard>

        <GlassyCard darker className={styles.flexItem}>
          <h3 className={styles.sectionTitle}>Network Activity</h3>

          <div className={styles.chartPlaceholder}>
            <span>Transaction Volume Chart</span>
          </div>

          <GlassyCard className={styles.networkStats}>
            <div className={styles.infoRow}>
              <span>24h Transactions:</span>
              <span>324,567</span>
            </div>
            <div className={styles.infoRow}>
              <span>24h Volume:</span>
              <span>$12.5B</span>
            </div>
            <div className={`${styles.infoRow} ${styles.lastRow}`}>
              <span>Avg. Transaction Fee:</span>
              <span>$2.34</span>
            </div>
          </GlassyCard>
        </GlassyCard>
      </div>

      {/* Block Explorer */}
      <GlassyCard darker className={styles.blockExplorer}>
        <h3 className={styles.sectionTitle}>Block Explorer</h3>

        <GlassyCard className={styles.searchContainer}>
          <input
            type="text"
            placeholder="Enter block height, transaction hash, or address"
            className={styles.explorerInput}
          />
          <button className={styles.searchButton}>
            Search
          </button>
        </GlassyCard>

        <GlassyCard className={styles.quickLinksContainer}>
          <div className={styles.quickLinksTitle}>Quick Links</div>
          <div className={styles.quickLinks}>
            <a href="#" className={styles.quickLink}>Latest Block</a>
            <span>|</span>
            <a href="#" className={styles.quickLink}>Mempool</a>
            <span>|</span>
            <a href="#" className={styles.quickLink}>Rich List</a>
            <span>|</span>
            <a href="#" className={styles.quickLink}>Mining Pools</a>
            <span>|</span>
            <a href="#" className={styles.quickLink}>Network Nodes</a>
            <span>|</span>
            <a href="#" className={styles.quickLink}>API</a>
          </div>
        </GlassyCard>
      </GlassyCard>
    </PageLayout>
  );
}