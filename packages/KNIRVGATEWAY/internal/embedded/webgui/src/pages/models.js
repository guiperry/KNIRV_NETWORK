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
    window.location.href = '/codex-builder';
  };

  const openDVEDEX = () => {
    window.location.href = '/models-dex';
  };

  const openMyDVEs = () => {
    window.location.href = '/dve-list';
  };

  return (
    <PageLayout activePage={activePage} pageTitle="DVE Overview" onSearch={handleSearch}>
      <PageHeader
        title="Distributed Validation Environments"
        subtitle="Deploy, manage, and exchange sovereign compute environments on the KNIRV network"
        titleColor="#7c4dff"
      />

      <div className={styles.gridContainer}>
        {/* Codex Builder */}
        <GlassyCard className={styles.modelCard}>
          <div className={styles.cardIcon}>
            <i className="fas fa-hammer" style={{ fontSize: '2rem', color: '#7c4dff' }}></i>
          </div>
          <h3 className={styles.cardTitle}>DVE Codex Builder</h3>
          <p className={styles.cardDescription}>
            Pre-build custom environments with the toolchains and SDKs of your preference.
            Design and compile sovereign DVEs for any workload.
          </p>
          <button onClick={openCodexBuilder} className={styles.actionButton}>
            <i className="fas fa-external-link-alt" style={{ marginRight: '8px' }}></i>
            Open Builder
          </button>
        </GlassyCard>

        {/* DVE DEX */}
        <GlassyCard className={styles.modelCard}>
          <div className={styles.cardIcon}>
            <i className="fas fa-exchange-alt" style={{ fontSize: '2rem', color: '#28a745' }}></i>
          </div>
          <h3 className={styles.cardTitle}>DVE DEX</h3>
          <p className={styles.cardDescription}>
            Decentralized exchange for Distributed Validation Environments. Trade, buy, and sell
            DVE ownership rights. Browse listed inventory and discover curated environments.
          </p>
          <div className={styles.statsContainer}>
            <div className={styles.stat}>
              <div className={styles.statValue}>156</div>
              <div className={styles.statLabel}>Listed DVEs</div>
            </div>
            <div className={styles.stat}>
              <div className={styles.statValue}>42</div>
              <div className={styles.statLabel}>Active Traders</div>
            </div>
          </div>
          <button onClick={openDVEDEX} className={styles.actionButton}>
            <i className="fas fa-store" style={{ marginRight: '8px' }}></i>
            Browse DEX
          </button>
        </GlassyCard>

        {/* My DVEs */}
        <GlassyCard className={styles.modelCard}>
          <div className={styles.cardIcon}>
            <i className="fas fa-server" style={{ fontSize: '2rem', color: '#00bcd4' }}></i>
          </div>
          <h3 className={styles.cardTitle}>My DVEs</h3>
          <p className={styles.cardDescription}>
            Manage your active Distributed Validation Environments. Monitor status,
            launch workspaces, and control your sovereign compute nodes.
          </p>
          <button onClick={openMyDVEs} className={styles.actionButton}>
            <i className="fas fa-th-large" style={{ marginRight: '8px' }}></i>
            View My DVEs
          </button>
        </GlassyCard>

        {/* Network Statistics */}
        <GlassyCard className={styles.statsCard}>
          <h3 className={styles.cardTitle}>
            <i className="fas fa-chart-bar" style={{ marginRight: '10px', color: '#7c4dff' }}></i>
            Network Statistics
          </h3>
          <div className={styles.networkStats}>
            <div className={styles.networkStat}>
              <div className={styles.networkStatValue}>2,847</div>
              <div className={styles.networkStatLabel}>Total DVEs</div>
            </div>
            <div className={styles.networkStat}>
              <div className={styles.networkStatValue}>1,234</div>
              <div className={styles.networkStatLabel}>Active DVEs</div>
            </div>
            <div className={styles.networkStat}>
              <div className={styles.networkStatValue}>89.5%</div>
              <div className={styles.networkStatLabel}>Success Rate</div>
            </div>
            <div className={styles.networkStat}>
              <div className={styles.networkStatValue}>456</div>
              <div className={styles.networkStatLabel}>Operators</div>
            </div>
          </div>
        </GlassyCard>

        {/* Recent Activity */}
        <GlassyCard className={styles.activityCard}>
          <h3 className={styles.cardTitle}>
            <i className="fas fa-clock" style={{ marginRight: '10px', color: '#7c4dff' }}></i>
            Recent Activity
          </h3>
          <div className={styles.activityList}>
            <div className={styles.activityItem}>
              <div className={styles.activityIcon}>🖥️</div>
              <div className={styles.activityContent}>
                <div className={styles.activityTitle}>New DVE provisioned</div>
                <div className={styles.activityTime}>2 minutes ago</div>
              </div>
            </div>
            <div className={styles.activityItem}>
              <div className={styles.activityIcon}>💱</div>
              <div className={styles.activityContent}>
                <div className={styles.activityTitle}>DVE traded on DEX</div>
                <div className={styles.activityTime}>15 minutes ago</div>
              </div>
            </div>
            <div className={styles.activityItem}>
              <div className={styles.activityIcon}>🔧</div>
              <div className={styles.activityContent}>
                <div className={styles.activityTitle}>DVE Codex updated</div>
                <div className={styles.activityTime}>1 hour ago</div>
              </div>
            </div>
            <div className={styles.activityItem}>
              <div className={styles.activityIcon}>🛡️</div>
              <div className={styles.activityContent}>
                <div className={styles.activityTitle}>DVE validation completed</div>
                <div className={styles.activityTime}>3 hours ago</div>
              </div>
            </div>
          </div>
        </GlassyCard>
      </div>
    </PageLayout>
  );
}
