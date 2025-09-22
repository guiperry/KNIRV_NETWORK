import React, { useState, useEffect } from 'react';
import { useNavigation } from '../hooks/useNavigation';
import PageLayout from '../components/PageLayout';
import PageHeader from '../components/PageHeader';
import GlassyCard from '../components/GlassyCard';
import styles from './network-monitor.module.css';

export default function NetworkMonitor() {
  const { activePage } = useNavigation('network-monitor');
  const [searchQuery, setSearchQuery] = useState('');
  const [networkData, setNetworkData] = useState({
    blockHeight: 1234567,
    peers: 42,
    syncStatus: 'Synced',
    networkHash: '0x1234...abcd',
    lastBlockTime: '2 seconds ago'
  });
  const [showOperatorModal, setShowOperatorModal] = useState(false);
  const [showTunnelModal, setShowTunnelModal] = useState(false);

  const handleSearch = (query) => {
    setSearchQuery(query);
  };

  // Simulate real-time network data updates
  useEffect(() => {
    const interval = setInterval(() => {
      setNetworkData(prev => ({
        ...prev,
        blockHeight: prev.blockHeight + Math.floor(Math.random() * 3),
        peers: 40 + Math.floor(Math.random() * 10),
        lastBlockTime: Math.floor(Math.random() * 10) + ' seconds ago'
      }));
    }, 5000);

    return () => clearInterval(interval);
  }, []);

  return (
    <PageLayout activePage={activePage} pageTitle="Network Monitor" onSearch={handleSearch}>
      <PageHeader 
        title="Network Monitor"
        subtitle="Monitor KNIRV network status and performance"
        titleColor="#007bff"
      />

      <div className={styles.gridContainer}>
        {/* Network Status Overview */}
        <GlassyCard className={styles.statusCard}>
          <h3 className={styles.cardTitle}>
            <i className="fas fa-network-wired" style={{ marginRight: '10px', color: '#007bff' }}></i>
            Network Status
          </h3>
          <div className={styles.statusGrid}>
            <div className={styles.statusItem}>
              <div className={styles.statusLabel}>Block Height</div>
              <div className={styles.statusValue}>{networkData.blockHeight.toLocaleString()}</div>
            </div>
            <div className={styles.statusItem}>
              <div className={styles.statusLabel}>Connected Peers</div>
              <div className={styles.statusValue}>{networkData.peers}</div>
            </div>
            <div className={styles.statusItem}>
              <div className={styles.statusLabel}>Sync Status</div>
              <div className={styles.statusValue} style={{ color: '#28a745' }}>{networkData.syncStatus}</div>
            </div>
            <div className={styles.statusItem}>
              <div className={styles.statusLabel}>Last Block</div>
              <div className={styles.statusValue}>{networkData.lastBlockTime}</div>
            </div>
          </div>
        </GlassyCard>

        {/* Operator Registry */}
        <GlassyCard className={styles.registryCard}>
          <h3 className={styles.cardTitle}>
            <i className="fas fa-users-cog" style={{ marginRight: '10px', color: '#007bff' }}></i>
            Operator Registry
          </h3>
          <p style={{ color: '#ccc', marginBottom: '15px' }}>
            View and manage network operators
          </p>
          <button 
            onClick={() => setShowOperatorModal(true)}
            className={styles.actionButton}
          >
            <i className="fas fa-external-link-alt" style={{ marginRight: '8px' }}></i>
            Open Registry
          </button>
        </GlassyCard>

        {/* Tunnel Registry */}
        <GlassyCard className={styles.registryCard}>
          <h3 className={styles.cardTitle}>
            <i className="fas fa-route" style={{ marginRight: '10px', color: '#007bff' }}></i>
            Tunnel Registry
          </h3>
          <p style={{ color: '#ccc', marginBottom: '15px' }}>
            Manage network tunnels and endpoints
          </p>
          <button 
            onClick={() => setShowTunnelModal(true)}
            className={styles.actionButton}
          >
            <i className="fas fa-external-link-alt" style={{ marginRight: '8px' }}></i>
            Open Registry
          </button>
        </GlassyCard>

        {/* Network Performance */}
        <GlassyCard className={styles.performanceCard}>
          <h3 className={styles.cardTitle}>
            <i className="fas fa-chart-line" style={{ marginRight: '10px', color: '#007bff' }}></i>
            Network Performance
          </h3>
          <div className={styles.performanceMetrics}>
            <div className={styles.metric}>
              <div className={styles.metricLabel}>Throughput</div>
              <div className={styles.metricValue}>1,234 TPS</div>
            </div>
            <div className={styles.metric}>
              <div className={styles.metricLabel}>Latency</div>
              <div className={styles.metricValue}>45ms</div>
            </div>
            <div className={styles.metric}>
              <div className={styles.metricLabel}>Uptime</div>
              <div className={styles.metricValue}>99.9%</div>
            </div>
          </div>
        </GlassyCard>
      </div>

      {/* Operator Registry Modal */}
      {showOperatorModal && (
        <div className={styles.modal}>
          <div className={styles.modalContent}>
            <div className={styles.modalHeader}>
              <h3>Operator Registry</h3>
              <button onClick={() => setShowOperatorModal(false)} className={styles.closeButton}>×</button>
            </div>
            <div className={styles.modalBody}>
              <p>Operator registry functionality would be integrated here with KNIRVORACLE network-monitor API.</p>
              <div className={styles.operatorList}>
                <div className={styles.operatorItem}>
                  <div>Operator 1</div>
                  <div style={{ color: '#28a745' }}>Active</div>
                </div>
                <div className={styles.operatorItem}>
                  <div>Operator 2</div>
                  <div style={{ color: '#28a745' }}>Active</div>
                </div>
                <div className={styles.operatorItem}>
                  <div>Operator 3</div>
                  <div style={{ color: '#ffc107' }}>Pending</div>
                </div>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Tunnel Registry Modal */}
      {showTunnelModal && (
        <div className={styles.modal}>
          <div className={styles.modalContent}>
            <div className={styles.modalHeader}>
              <h3>Tunnel Registry</h3>
              <button onClick={() => setShowTunnelModal(false)} className={styles.closeButton}>×</button>
            </div>
            <div className={styles.modalBody}>
              <p>Tunnel registry functionality would be integrated here with KNIRVORACLE network-monitor API.</p>
              <div className={styles.tunnelList}>
                <div className={styles.tunnelItem}>
                  <div>tunnel-1.knirv.network</div>
                  <div style={{ color: '#28a745' }}>Connected</div>
                </div>
                <div className={styles.tunnelItem}>
                  <div>tunnel-2.knirv.network</div>
                  <div style={{ color: '#28a745' }}>Connected</div>
                </div>
                <div className={styles.tunnelItem}>
                  <div>tunnel-3.knirv.network</div>
                  <div style={{ color: '#dc3545' }}>Disconnected</div>
                </div>
              </div>
            </div>
          </div>
        </div>
      )}
    </PageLayout>
  );
}
