import React, { useEffect, useRef } from 'react';
import { useNavigation } from '../hooks/useNavigation';
import PageLayout from '../components/PageLayout';
import PageHeader from '../components/PageHeader';
import GlassyCard from '../components/GlassyCard';
import styles from './graph-explorer.module.css';

export default function GraphExplorer() {
  const { activePage } = useNavigation('graph-explorer');
  const iframeRef = useRef(null);

  const handleSearch = (query) => {
    if (iframeRef.current && iframeRef.current.contentWindow) {
      iframeRef.current.contentWindow.postMessage({ type: 'search', query }, '*');
    }
  };

  useEffect(() => {
    const handleMessage = (event) => {
      if (event.data && event.data.type === 'graphchain-ready') {
        console.log('GraphChain Explorer loaded successfully');
      }
    };
    window.addEventListener('message', handleMessage);
    return () => window.removeEventListener('message', handleMessage);
  }, []);

  return (
    <PageLayout activePage={activePage} pageTitle="Graph Explorer" onSearch={handleSearch}>
      <PageHeader 
        title="KNIRV Graph Explorer"
        subtitle="Explore the GraphChain network topology and node relationships"
        titleColor="#007bff"
      />

      <GlassyCard className={styles.explorerCard}>
        <div className={styles.explorerHeader}>
          <h3 className={styles.explorerTitle}>
            <i className="fas fa-project-diagram" style={{ marginRight: '10px', color: '#007bff' }}></i>
            GraphChain Explorer
          </h3>
          <div className={styles.explorerControls}>
            <a 
              href="/graphchain-explorer/" 
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
            src="/graphchain-explorer/"
            className={styles.explorerIframe}
            title="KNIRV GraphChain Explorer"
            sandbox="allow-scripts allow-same-origin allow-forms allow-popups"
            loading="lazy"
          />
        </div>
      </GlassyCard>

      {/* Quick Actions */}
      <div className={styles.quickActions}>
        <GlassyCard className={styles.actionCard}>
          <div className={styles.actionIcon}>🔗</div>
          <h4>SkillNodes</h4>
          <p>Explore skill execution nodes and their connections</p>
          <button 
            onClick={() => {
              if (iframeRef.current) {
                iframeRef.current.src = '/graphchain-explorer/skills.html';
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
          <p>Monitor error nodes and network issues</p>
          <button 
            onClick={() => {
              if (iframeRef.current) {
                iframeRef.current.src = '/graphchain-explorer/errors.html';
              }
            }}
            className={styles.actionButton}
          >
            View ErrorNodes
          </button>
        </GlassyCard>

        <GlassyCard className={styles.actionCard}>
          <div className={styles.actionIcon}>🌐</div>
          <h4>Interactive Graph</h4>
          <p>Explore the network with interactive visualization</p>
          <button 
            onClick={() => {
              if (iframeRef.current) {
                iframeRef.current.src = '/graphchain-explorer/interactive-graph.html';
              }
            }}
            className={styles.actionButton}
          >
            Open Interactive View
          </button>
        </GlassyCard>

        <GlassyCard className={styles.actionCard}>
          <div className={styles.actionIcon}>🔍</div>
          <h4>Search</h4>
          <p>Search for specific nodes, transactions, or data</p>
          <button 
            onClick={() => {
              if (iframeRef.current) {
                iframeRef.current.src = '/graphchain-explorer/search.html';
              }
            }}
            className={styles.actionButton}
          >
            Open Search
          </button>
        </GlassyCard>
      </div>

      {/* Network Status */}
      <GlassyCard className={styles.statusCard}>
        <div className={styles.statusHeader}>
          <h3 className={styles.statusTitle}>
            <i className="fas fa-heartbeat" style={{ marginRight: '10px', color: '#28a745' }}></i>
            Network Status
          </h3>
        </div>
        <div className={styles.statusGrid}>
          <div className={styles.statusItem}>
            <div className={styles.statusLabel}>GraphChain Status</div>
            <div className={styles.statusValue}>
              <span className={styles.statusIndicator} style={{ backgroundColor: '#28a745' }}></span>
              Online
            </div>
          </div>
          <div className={styles.statusItem}>
            <div className={styles.statusLabel}>Active Nodes</div>
            <div className={styles.statusValue}>1,247</div>
          </div>
          <div className={styles.statusItem}>
            <div className={styles.statusLabel}>Skill Executions</div>
            <div className={styles.statusValue}>15,432</div>
          </div>
          <div className={styles.statusItem}>
            <div className={styles.statusLabel}>Error Rate</div>
            <div className={styles.statusValue}>0.3%</div>
          </div>
        </div>
      </GlassyCard>
    </PageLayout>
  );
}
