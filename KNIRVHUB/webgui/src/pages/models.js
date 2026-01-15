import React, { useState } from 'react';
import { useNavigation } from '../hooks/useNavigation';
import PageLayout from '../components/PageLayout';
import PageHeader from '../components/PageHeader';
import GlassyCard from '../components/GlassyCard';
import styles from './models.module.css';

export default function Models() {
  const { activePage } = useNavigation('models');
  const [searchQuery, setSearchQuery] = useState('');

  const handleSearch = (query) => {
    setSearchQuery(query);
  };

  const openCodexBuilder = () => {
    // This would open the primary-website in an iframe
    window.open('/primary-website', '_blank');
  };

  const openModelsDEX = () => {
    // Navigate to models DEX (formerly inventory with onboarding)
    window.location.href = '/models-dex';
  };

  const openKNIRVINFERENCEDAO = () => {
    // Navigate to KNIRVINFERENCE DAO
    window.location.href = '/knirvinference-dao';
  };

  return (
    <PageLayout activePage={activePage} pageTitle="Models" onSearch={handleSearch}>
      <PageHeader 
        title="Models"
        subtitle="Build, deploy, and manage AI models on the KNIRV network"
        titleColor="#007bff"
      />

      <div className={styles.gridContainer}>
        {/* Codex Builder */}
        <GlassyCard className={styles.modelCard}>
          <div className={styles.cardIcon}>
            <i className="fas fa-hammer" style={{ fontSize: '2rem', color: '#007bff' }}></i>
          </div>
          <h3 className={styles.cardTitle}>Codex Builder</h3>
          <p className={styles.cardDescription}>
            Build and compile AI models using our integrated development environment. 
            Create custom models with advanced tooling and real-time compilation.
          </p>
          <button onClick={openCodexBuilder} className={styles.actionButton}>
            <i className="fas fa-external-link-alt" style={{ marginRight: '8px' }}></i>
            Open Builder
          </button>
        </GlassyCard>

        {/* Models DEX */}
        <GlassyCard className={styles.modelCard}>
          <div className={styles.cardIcon}>
            <i className="fas fa-exchange-alt" style={{ fontSize: '2rem', color: '#28a745' }}></i>
          </div>
          <h3 className={styles.cardTitle}>Models DEX</h3>
          <p className={styles.cardDescription}>
            Decentralized exchange for AI models. Trade, buy, and sell model ownership rights.
            Browse listed inventory and discover new models.
          </p>
          <div className={styles.statsContainer}>
            <div className={styles.stat}>
              <div className={styles.statValue}>156</div>
              <div className={styles.statLabel}>Listed Models</div>
            </div>
            <div className={styles.stat}>
              <div className={styles.statValue}>42</div>
              <div className={styles.statLabel}>Active Traders</div>
            </div>
          </div>
          <button onClick={openModelsDEX} className={styles.actionButton}>
            <i className="fas fa-store" style={{ marginRight: '8px' }}></i>
            Browse DEX
          </button>
        </GlassyCard>

        {/* KNIRVINFERENCE DAO */}
        <GlassyCard className={styles.modelCard}>
          <div className={styles.cardIcon}>
            <i className="fas fa-vote-yea" style={{ fontSize: '2rem', color: '#ffc107' }}></i>
          </div>
          <h3 className={styles.cardTitle}>KNIRVINFERENCE DAO</h3>
          <p className={styles.cardDescription}>
            Participate in decentralized governance for network inference models.
            Vote on bootnode elections and network model governance proposals.
          </p>
          <div className={styles.governanceStats}>
            <div className={styles.governanceItem}>
              <span className={styles.governanceLabel}>Active Proposals:</span>
              <span className={styles.governanceValue}>3</span>
            </div>
            <div className={styles.governanceItem}>
              <span className={styles.governanceLabel}>Your Voting Power:</span>
              <span className={styles.governanceValue}>1,250 NRN</span>
            </div>
          </div>
          <button onClick={openKNIRVINFERENCEDAO} className={styles.actionButton}>
            <i className="fas fa-gavel" style={{ marginRight: '8px' }}></i>
            Enter DAO
          </button>
        </GlassyCard>

        {/* Model Statistics */}
        <GlassyCard className={styles.statsCard}>
          <h3 className={styles.cardTitle}>
            <i className="fas fa-chart-bar" style={{ marginRight: '10px', color: '#007bff' }}></i>
            Network Statistics
          </h3>
          <div className={styles.networkStats}>
            <div className={styles.networkStat}>
              <div className={styles.networkStatValue}>2,847</div>
              <div className={styles.networkStatLabel}>Total Models</div>
            </div>
            <div className={styles.networkStat}>
              <div className={styles.networkStatValue}>1,234</div>
              <div className={styles.networkStatLabel}>Active Models</div>
            </div>
            <div className={styles.networkStat}>
              <div className={styles.networkStatValue}>89.5%</div>
              <div className={styles.networkStatLabel}>Success Rate</div>
            </div>
            <div className={styles.networkStat}>
              <div className={styles.networkStatValue}>456</div>
              <div className={styles.networkStatLabel}>Developers</div>
            </div>
          </div>
        </GlassyCard>

        {/* Recent Activity */}
        <GlassyCard className={styles.activityCard}>
          <h3 className={styles.cardTitle}>
            <i className="fas fa-clock" style={{ marginRight: '10px', color: '#007bff' }}></i>
            Recent Activity
          </h3>
          <div className={styles.activityList}>
            <div className={styles.activityItem}>
              <div className={styles.activityIcon}>🤖</div>
              <div className={styles.activityContent}>
                <div className={styles.activityTitle}>New model deployed</div>
                <div className={styles.activityTime}>2 minutes ago</div>
              </div>
            </div>
            <div className={styles.activityItem}>
              <div className={styles.activityIcon}>💱</div>
              <div className={styles.activityContent}>
                <div className={styles.activityTitle}>Model traded on DEX</div>
                <div className={styles.activityTime}>15 minutes ago</div>
              </div>
            </div>
            <div className={styles.activityItem}>
              <div className={styles.activityIcon}>🗳️</div>
              <div className={styles.activityContent}>
                <div className={styles.activityTitle}>New governance proposal</div>
                <div className={styles.activityTime}>1 hour ago</div>
              </div>
            </div>
            <div className={styles.activityItem}>
              <div className={styles.activityIcon}>🔧</div>
              <div className={styles.activityContent}>
                <div className={styles.activityTitle}>Model updated</div>
                <div className={styles.activityTime}>3 hours ago</div>
              </div>
            </div>
          </div>
        </GlassyCard>
      </div>
    </PageLayout>
  );
}
