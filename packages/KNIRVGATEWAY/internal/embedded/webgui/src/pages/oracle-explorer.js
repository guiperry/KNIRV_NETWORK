import React, { useState, useEffect, useCallback } from 'react';
import { useNavigation } from '../hooks/useNavigation';
import PageLayout from '../components/PageLayout';
import PageHeader from '../components/PageHeader';
import GlassyCard from '../components/GlassyCard';
import styles from './oracle-explorer.module.css';
import { oracleApi } from '../services/api-client';

function formatTimestamp(value) {
  if (!value) return 'N/A';
  const date = typeof value === 'number' ? new Date(value < 1e12 ? value * 1000 : value) : new Date(value);
  return Number.isNaN(date.getTime()) ? 'N/A' : date.toLocaleString();
}

export default function OracleExplorer() {
  const { activePage } = useNavigation('oracle-explorer');
  const isBlockchainEnabled = true;
  const [isLoading, setIsLoading] = useState(false);
  const [isRunning, setIsRunning] = useState(false);
  const [serverInfo, setServerInfo] = useState(null);
  const [blocks, setBlocks] = useState([]);
  const [transactions, setTransactions] = useState([]);
  const [selectedBlock, setSelectedBlock] = useState(null);
  const [error, setError] = useState(null);
  const [activeTab, setActiveTab] = useState('blocks');
  const [searchQuery, setSearchQuery] = useState('');

  const refreshData = useCallback(async () => {
    try {
      setIsLoading(true);
      setError(null);

      const [healthResult, blocksResult, transactionsResult] = await Promise.allSettled([
        oracleApi.getHealth(),
        oracleApi.getBlocks(),
        oracleApi.getTransactions(),
      ]);
      const failures = [healthResult, blocksResult, transactionsResult]
        .filter((result) => result.status === 'rejected')
        .map((result) => result.reason.message);

      if (healthResult.status === 'fulfilled') setServerInfo(healthResult.value);
      if (blocksResult.status === 'fulfilled') setBlocks(blocksResult.value);
      if (transactionsResult.status === 'fulfilled') setTransactions(transactionsResult.value);
      setError(failures.length ? failures.join(' • ') : null);
      setIsRunning(healthResult.status === 'fulfilled');
    } catch (err) {
      console.error('Error fetching oracle data:', err);
      setError(err.message);
      setIsRunning(false);
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    if (isBlockchainEnabled) {
      refreshData();
    }
  }, [isBlockchainEnabled, refreshData]);

  const handleSearch = (query) => {
    setSearchQuery(query);
  };

  return (
    <PageLayout activePage={activePage} pageTitle="Oracle Explorer" onSearch={handleSearch}>
      {!isRunning && isBlockchainEnabled && !isLoading && (
        <GlassyCard darker className={styles.errorCard}>
          Backend is not running. Please start the KNIRVORACLE node.
        </GlassyCard>
      )}

      {isLoading && isBlockchainEnabled && (
        <GlassyCard darker className={styles.loading}>
          Loading oracle chain data...
        </GlassyCard>
      )}

      {error && (
        <GlassyCard darker className={styles.errorCard}>
          Failed to load oracle data: {error}
        </GlassyCard>
      )}

      <PageHeader
        title="Oracle Explorer"
        subtitle="Monitor oracle chain node status, blocks, and transactions"
      />

      <button className={`${styles.button} ${styles.secondary}`} onClick={refreshData} disabled={isLoading}>
        {isLoading ? 'Refreshing…' : 'Refresh'}
      </button>

      {serverInfo && isBlockchainEnabled && isRunning && !isLoading && (
        <GlassyCard darker className={styles.serverInfo}>
          <h3>Node Information</h3>
          <div className={styles.infoGrid}>
            <div className={styles.infoRow}>
              <span>Chain ID:</span>
              <span>{serverInfo.chain_id || serverInfo.network || 'N/A'}</span>
            </div>
            <div className={styles.infoRow}>
              <span>Status:</span>
              <span>{serverInfo.status || 'healthy'}</span>
            </div>
            <div className={styles.infoRow}>
              <span>Version:</span>
              <span>{serverInfo.version || 'N/A'}</span>
            </div>
            <div className={styles.infoRow}>
              <span>Height:</span>
              <span>{serverInfo.height || serverInfo.block_height || 'N/A'}</span>
            </div>
            <div className={styles.infoRow}>
              <span>Peers:</span>
              <span>{serverInfo.peers || serverInfo.connections || 'N/A'}</span>
            </div>
          </div>
        </GlassyCard>
      )}

      {searchQuery && isBlockchainEnabled && isRunning && !isLoading && (
        <GlassyCard darker className={styles.searchResults}>
          <h3>Search Results</h3>
          <p>Showing results for: {searchQuery}</p>
        </GlassyCard>
      )}

      {isBlockchainEnabled && isRunning && !isLoading && (
        <div className={styles.blockchainInfo}>
          <div style={{ display: 'flex', gap: '10px', marginBottom: '16px' }}>
            {['blocks', 'transactions'].map(tab => (
              <button key={tab} onClick={() => setActiveTab(tab)}
                style={{
                  padding: '8px 16px', borderRadius: '6px', border: '1px solid rgba(100,200,130,0.3)',
                  background: activeTab === tab ? 'rgba(40,120,70,0.7)' : 'rgba(30,40,70,0.5)',
                  color: activeTab === tab ? '#fff' : '#b0ffb0', cursor: 'pointer', fontSize: '14px'
                }}>
                {tab.charAt(0).toUpperCase() + tab.slice(1)}
              </button>
            ))}
          </div>

          {activeTab === 'blocks' && (
            <GlassyCard darker className={styles.blocksContainer}>
              <div className={styles.sectionHeader}>
                <h3>Latest Blocks</h3>
                <button className={`${styles.button} ${styles.secondary}`}>
                  View All
                </button>
              </div>

              {blocks.length === 0 ? (
                <div className={styles.emptyState}>No blocks found</div>
              ) : (
                <div className={styles.tableWrapper}>
                  <table className={styles.blockTable}>
                    <thead>
                      <tr className={styles.tableHeader}>
                        <th>Height</th>
                        <th>Hash</th>
                        <th>Timestamp</th>
                        <th>Transactions</th>
                      </tr>
                    </thead>
                    <tbody>
                      {blocks.map((block) => (
                        <tr
                          key={block.hash || block.id}
                          className={styles.tableRow}
                          onClick={() => setSelectedBlock(block)}
                        >
                          <td>{block.height || block.index || block.block_height}</td>
                          <td>{(block.hash || block.block_hash || '').substring(0, 10)}...</td>
                          <td>{formatTimestamp(block.timestamp || block.time)}</td>
                          <td>{block.tx_count || block.transactions?.length || 0}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </GlassyCard>
          )}

          {activeTab === 'transactions' && (
            <GlassyCard darker className={styles.blocksContainer}>
              <div className={styles.sectionHeader}>
                <h3>Recent Transactions</h3>
              </div>

              {transactions.length === 0 ? (
                <div className={styles.emptyState}>No transactions found</div>
              ) : (
                <div className={styles.transactionsList}>
                  {transactions.map((tx) => (
                    <GlassyCard key={tx.hash || tx.id} className={styles.transaction}>
                      <div className={styles.infoRow}>
                        <span>Hash:</span>
                        <span>{(tx.hash || tx.id || '').substring(0, 16)}...</span>
                      </div>
                      <div className={styles.infoRow}>
                        <span>Type:</span>
                        <span>{tx.type || tx.operation || 'Transfer'}</span>
                      </div>
                      <div className={styles.infoRow}>
                        <span>Time:</span>
                        <span>{formatTimestamp(tx.timestamp || tx.time)}</span>
                      </div>
                    </GlassyCard>
                  ))}
                </div>
              )}
            </GlassyCard>
          )}

          {selectedBlock && (
            <GlassyCard darker className={styles.blockDetails}>
              <div className={styles.sectionHeader}>
                <h3>Block Details</h3>
                <button
                  className={`${styles.button} ${styles.secondary}`}
                  onClick={() => setSelectedBlock(null)}
                >
                  ×
                </button>
              </div>

              <div className={styles.infoGrid}>
                <div className={styles.infoRow}>
                  <span>Hash:</span>
                  <span>{selectedBlock.hash || selectedBlock.block_hash || 'N/A'}</span>
                </div>
                <div className={styles.infoRow}>
                  <span>Height:</span>
                  <span>{selectedBlock.height || selectedBlock.index || selectedBlock.block_height || 'N/A'}</span>
                </div>
                <div className={styles.infoRow}>
                  <span>Timestamp:</span>
                  <span>{formatTimestamp(selectedBlock.timestamp || selectedBlock.time)}</span>
                </div>
                <div className={styles.infoRow}>
                  <span>Transactions:</span>
                  <span>{selectedBlock.tx_count || selectedBlock.transactions?.length || 0}</span>
                </div>
              </div>

              <h4>Transactions</h4>
              {(selectedBlock.transactions || []).length === 0 ? (
                <div className={styles.emptyState}>No transactions in this block</div>
              ) : (
                <div className={styles.transactionsList}>
                  {(selectedBlock.transactions || []).map((tx, index) => (
                    <GlassyCard key={index} className={styles.transaction}>
                      <div className={styles.infoRow}>
                        <span>Hash:</span>
                        <span>{(tx.hash || tx.id || '').substring(0, 16)}...</span>
                      </div>
                      <div className={styles.infoRow}>
                        <span>Type:</span>
                        <span>{tx.type || tx.operation || 'Transfer'}</span>
                      </div>
                    </GlassyCard>
                  ))}
                </div>
              )}
            </GlassyCard>
          )}
        </div>
      )}
    </PageLayout>
  );
}
