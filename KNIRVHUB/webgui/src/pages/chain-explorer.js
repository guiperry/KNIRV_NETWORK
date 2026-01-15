import React, { useEffect, useRef } from 'react';
import { useNavigation } from '../hooks/useNavigation';
import PageLayout from '../components/PageLayout';
import PageHeader from '../components/PageHeader';
import GlassyCard from '../components/GlassyCard';
import styles from './chain-explorer.module.css';

export default function ChainExplorer() {
  const { activePage } = useNavigation('chain-explorer');
  const iframeRef = useRef(null);

  const handleSearch = (query) => {
    // Pass search to the embedded knirvchain portal
    if (iframeRef.current && iframeRef.current.contentWindow) {
      iframeRef.current.contentWindow.postMessage({
        type: 'search',
        query: query
      }, '*');
    }
  };

  useEffect(() => {
    // Listen for messages from the embedded knirvchain portal
    const handleMessage = (event) => {
      if (event.data && event.data.type === 'knirvchain-ready') {
        console.log('KNIRVChain Portal loaded successfully');
      }
    };

    window.addEventListener('message', handleMessage);
    return () => window.removeEventListener('message', handleMessage);
  }, []);

  return (
    <PageLayout activePage={activePage} pageTitle="Chain Explorer" onSearch={handleSearch}>
      <PageHeader 
        title="KNIRV Chain Explorer"
        subtitle="Explore blockchain data, blocks, transactions, and network activity"
        titleColor="#007bff"
      />

      {/* Embedded KNIRVChain Portal */}
      <GlassyCard className={styles.explorerCard}>
        <div className={styles.explorerHeader}>
          <h3 className={styles.explorerTitle}>
            <i className="fas fa-cube" style={{ marginRight: '10px', color: '#007bff' }}></i>
            KNIRVChain Portal
          </h3>
          <div className={styles.explorerControls}>
            <a 
              href="/knirvchain-portal/" 
              target="_blank" 
              rel="noopener noreferrer"
              className={styles.openExternalButton}
              title="Open in new window"
            >
              <i className="fas fa-external-link-alt"></i>
            </a>
            <button 
              onClick={() => iframeRef.current?.contentWindow?.location.reload()}
              className={styles.refreshButton}
              title="Refresh"
            >
              <i className="fas fa-sync-alt"></i>
            </button>
          </div>
        </div>
        
        <div className={styles.iframeContainer}>
          <iframe
            ref={iframeRef}
            src="/knirvchain-portal/"
            className={styles.explorerIframe}
            title="KNIRV Chain Portal"
            sandbox="allow-scripts allow-same-origin allow-forms allow-popups"
            loading="lazy"
          />
        </div>
      </GlassyCard>

      {/* Quick Navigation */}
      <div className={styles.quickActions}>
        <GlassyCard className={styles.actionCard}>
          <div className={styles.actionIcon}>⛓️</div>
          <h4>Blocks</h4>
          <p>Browse recent blocks and their transactions</p>
          <button 
            onClick={() => {
              if (iframeRef.current) {
                iframeRef.current.src = '/knirvchain-portal/#/blocks';
              }
            }}
            className={styles.actionButton}
          >
            View Blocks
          </button>
        </GlassyCard>

        <GlassyCard className={styles.actionCard}>
          <div className={styles.actionIcon}>💱</div>
          <h4>Transactions</h4>
          <p>Monitor transaction flow and status</p>
          <button 
            onClick={() => {
              if (iframeRef.current) {
                iframeRef.current.src = '/knirvchain-portal/#/transactions';
              }
            }}
            className={styles.actionButton}
          >
            View Transactions
          </button>
        </GlassyCard>

        <GlassyCard className={styles.actionCard}>
          <div className={styles.actionIcon}>🔗</div>
          <h4>SkillNodes</h4>
          <p>Explore skill execution nodes on the chain</p>
          <button 
            onClick={() => {
              if (iframeRef.current) {
                iframeRef.current.src = '/knirvchain-portal/#/skills';
              }
            }}
            className={styles.actionButton}
          >
            View SkillNodes
          </button>
        </GlassyCard>

        <GlassyCard className={styles.actionCard}>
          <div className={styles.actionIcon}>🚨</div>
          <h4>ErrorNodes</h4>
          <p>Monitor error nodes and chain issues</p>
          <button 
            onClick={() => {
              if (iframeRef.current) {
                iframeRef.current.src = '/knirvchain-portal/#/errors';
              }
            }}
            className={styles.actionButton}
          >
            View ErrorNodes
          </button>
        </GlassyCard>

        <GlassyCard className={styles.actionCard}>
          <div className={styles.actionIcon}>📊</div>
          <h4>Graph View</h4>
          <p>Visualize chain data relationships</p>
          <button 
            onClick={() => {
              if (iframeRef.current) {
                iframeRef.current.src = '/knirvchain-portal/#/graph';
              }
            }}
            className={styles.actionButton}
          >
            Open Graph View
          </button>
        </GlassyCard>

        <GlassyCard className={styles.actionCard}>
          <div className={styles.actionIcon}>🔍</div>
          <h4>Search</h4>
          <p>Search blocks, transactions, and addresses</p>
          <button 
            onClick={() => {
              if (iframeRef.current) {
                iframeRef.current.src = '/knirvchain-portal/#/search';
              }
            }}
            className={styles.actionButton}
          >
            Open Search
          </button>
        </GlassyCard>
      </div>

      {/* Chain Statistics */}
      <GlassyCard className={styles.statsCard}>
        <div className={styles.statsHeader}>
          <h3 className={styles.statsTitle}>
            <i className="fas fa-chart-line" style={{ marginRight: '10px', color: '#28a745' }}></i>
            Chain Statistics
          </h3>
        </div>
        <div className={styles.statsGrid}>
          <div className={styles.statItem}>
            <div className={styles.statLabel}>Latest Block</div>
            <div className={styles.statValue}>12,345,678</div>
          </div>
          <div className={styles.statItem}>
            <div className={styles.statLabel}>Total Transactions</div>
            <div className={styles.statValue}>45,678,901</div>
          </div>
          <div className={styles.statItem}>
            <div className={styles.statLabel}>Network Hash Rate</div>
            <div className={styles.statValue}>1.2 TH/s</div>
          </div>
          <div className={styles.statItem}>
            <div className={styles.statLabel}>Active Validators</div>
            <div className={styles.statValue}>156</div>
          </div>
          <div className={styles.statItem}>
            <div className={styles.statLabel}>Avg Block Time</div>
            <div className={styles.statValue}>15.2s</div>
          </div>
          <div className={styles.statItem}>
            <div className={styles.statLabel}>Gas Price</div>
            <div className={styles.statValue}>20 Gwei</div>
          </div>
        </div>
      </GlassyCard>

      {/* Network Health */}
      <GlassyCard className={styles.healthCard}>
        <div className={styles.healthHeader}>
          <h3 className={styles.healthTitle}>
            <i className="fas fa-heartbeat" style={{ marginRight: '10px', color: '#28a745' }}></i>
            Network Health
          </h3>
        </div>
        <div className={styles.healthGrid}>
          <div className={styles.healthItem}>
            <div className={styles.healthLabel}>Chain Status</div>
            <div className={styles.healthValue}>
              <span className={styles.healthIndicator} style={{ backgroundColor: '#28a745' }}></span>
              Healthy
            </div>
          </div>
          <div className={styles.healthItem}>
            <div className={styles.healthLabel}>Sync Status</div>
            <div className={styles.healthValue}>
              <span className={styles.healthIndicator} style={{ backgroundColor: '#28a745' }}></span>
              Synced
            </div>
          </div>
          <div className={styles.healthItem}>
            <div className={styles.healthLabel}>Peer Count</div>
            <div className={styles.healthValue}>47</div>
          </div>
          <div className={styles.healthItem}>
            <div className={styles.healthLabel}>Uptime</div>
            <div className={styles.healthValue}>99.8%</div>
          </div>
        </div>
      </GlassyCard>
    </PageLayout>
  );
}
