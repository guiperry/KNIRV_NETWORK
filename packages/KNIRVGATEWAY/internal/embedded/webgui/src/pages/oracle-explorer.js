import React, { useState, useEffect, useCallback } from 'react';
import { useNavigation } from '../hooks/useNavigation';
import PageLayout from '../components/PageLayout';
import PageHeader from '../components/PageHeader';
import GlassyCard from '../components/GlassyCard';
import styles from './oracle-explorer.module.css';

export default function OracleExplorer() {
  const { activePage } = useNavigation('oracle-explorer');
  const [isBlockchainEnabled, setIsBlockchainEnabled] = useState(true);
  const [searchQuery, setSearchQuery] = useState('');
  const [activeNetwork, setActiveNetwork] = useState('Bitcoin');
  const [isRunning, setIsRunning] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [serverInfo, setServerInfo] = useState(null);
  const [blocks, setBlocks] = useState([]);
  const [selectedBlock, setSelectedBlock] = useState(null);

  const networks = ['Bitcoin', 'Ethereum', 'Solana', 'Cardano'];

  const refreshData = useCallback(async () => {
    try {
      setIsLoading(true);
      // TODO: Replace with actual API calls
      const mockServerInfo = {
        chain_id: 'KNIRVORACLE-Mainnet',
        http_port: 8080,
        p2p_port: 30303,
        version: '1.0.0',
        connections: 5
      };
      setServerInfo(mockServerInfo);

      const mockBlocks = Array(5).fill().map((_, i) => ({
        index: i + 1,
        hash: `0000000000000000000${i}abcdef1234567890`,
        previous_hash: i > 0 ? `0000000000000000000${i-1}abcdef1234567890` : '0',
        timestamp: Math.floor(Date.now() / 1000) - (i * 600),
        nonce: 12345 + i,
        data: i % 2 === 0 ? [] : [
          {
            id: `tx${i}${Date.now()}`,
            from: `addr${i}`,
            to: `addr${i+1}`,
            amount: (i * 10) + 5
          }
        ]
      }));
      setBlocks(mockBlocks);
      setIsRunning(true);
    } catch (error) {
      console.error('Error fetching data:', error);
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

      {/* Status Overlays */}
      {!isRunning && isBlockchainEnabled && (
        <GlassyCard darker className={styles.errorCard}>
          Backend is not running. Please start the KNIRVORACLE node.
        </GlassyCard>
      )}

      {isLoading && isBlockchainEnabled && isRunning && (
        <GlassyCard darker className={styles.loading}>
          Loading chain data...
        </GlassyCard>
      )}

      <PageHeader 
        title={`${activeNetwork} Explorer`}
        subtitle="Search and analyze blockchain data"
      />

      {/* Network Selection */}
      <GlassyCard darker className={styles.networkSelection}>
        <div className={styles.networkButtons}>
          {networks.map(network => (
            <button 
              key={network}
              className={`${styles.button} ${activeNetwork === network ? styles.primary : styles.secondary}`}
              onClick={() => setActiveNetwork(network)}
            >
              {network}
            </button>
          ))}
        </div>
      </GlassyCard>

      {/* Server Info */}
      {serverInfo && isBlockchainEnabled && isRunning && !isLoading && (
        <GlassyCard darker className={styles.serverInfo}>
          <h3>Node Information</h3>
          <div className={styles.infoGrid}>
            <div className={styles.infoRow}>
              <span>Chain ID:</span>
              <span>{serverInfo.chain_id}</span>
            </div>
            <div className={styles.infoRow}>
              <span>HTTP Port:</span>
              <span>{serverInfo.http_port}</span>
            </div>
            <div className={styles.infoRow}>
              <span>P2P Port:</span>
              <span>{serverInfo.p2p_port}</span>
            </div>
            <div className={styles.infoRow}>
              <span>Version:</span>
              <span>{serverInfo.version}</span>
            </div>
            <div className={styles.infoRow}>
              <span>Connected Peers:</span>
              <span>{serverInfo.connections}</span>
            </div>
          </div>
        </GlassyCard>
      )}

      {/* Search Results */}
      {searchQuery && isBlockchainEnabled && isRunning && !isLoading && (
        <GlassyCard darker className={styles.searchResults}>
          <h3>Search Results</h3>
          <p>Showing results for: {searchQuery}</p>
        </GlassyCard>
      )}

      {/* Block Explorer */}
      {isBlockchainEnabled && isRunning && !isLoading && (
        <div className={styles.blockchainInfo}>
          {/* Blocks List */}
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
                        key={block.hash}
                        className={styles.tableRow}
                        onClick={() => setSelectedBlock(block)}
                      >
                        <td>{block.index || block.height}</td>
                        <td>{block.hash.substring(0, 10)}...</td>
                        <td>{new Date(block.timestamp * 1000).toLocaleString()}</td>
                        <td>{block.data?.length || 0}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </GlassyCard>

          {/* Block Details */}
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
                  <span>{selectedBlock.hash}</span>
                </div>
                <div className={styles.infoRow}>
                  <span>Previous Hash:</span>
                  <span>{selectedBlock.previous_hash || selectedBlock.prevHash}</span>
                </div>
                <div className={styles.infoRow}>
                  <span>Timestamp:</span>
                  <span>{new Date(selectedBlock.timestamp * 1000).toLocaleString()}</span>
                </div>
                <div className={styles.infoRow}>
                  <span>Nonce:</span>
                  <span>{selectedBlock.nonce}</span>
                </div>
              </div>

              <h4>Transactions</h4>
              {selectedBlock.data && selectedBlock.data.length === 0 ? (
                <div className={styles.emptyState}>No transactions in this block</div>
              ) : (
                <div className={styles.transactionsList}>
                  {(selectedBlock.data || []).map((tx, index) => (
                    <GlassyCard key={index} className={styles.transaction}>
                      <div className={styles.infoRow}>
                        <span>ID:</span>
                        <span>{tx.id}</span>
                      </div>
                      <div className={styles.infoRow}>
                        <span>From:</span>
                        <span>{tx.from}</span>
                      </div>
                      <div className={styles.infoRow}>
                        <span>To:</span>
                        <span>{tx.to}</span>
                      </div>
                      <div className={styles.infoRow}>
                        <span>Amount:</span>
                        <span>{tx.amount}</span>
                      </div>
                    </GlassyCard>
                  ))}
                </div>
              )}
            </GlassyCard>
          )}
        </div>
      )}
      
      {/* Network Stats */}
      <div className={styles.statsContainer}>
        <GlassyCard darker className={styles.statCard}>
          <h3>Current Price</h3>
          <h2>$45,632</h2>
          <span className={styles.positiveChange}>+2.3%</span>
        </GlassyCard>

        <GlassyCard darker className={styles.statCard}>
          <h3>Hash Rate</h3>
          <h2>215 EH/s</h2>
          <span className={styles.positiveChange}>+5.7%</span>
        </GlassyCard>

        <GlassyCard darker className={styles.statCard}>
          <h3>Difficulty</h3>
          <h2>37.59 T</h2>
          <span className={styles.positiveChange}>+0.8%</span>
        </GlassyCard>

        <GlassyCard darker className={styles.statCard}>
          <h3>Block Height</h3>
          <h2>789,245</h2>
          <span>Latest</span>
        </GlassyCard>
      </div>
    </PageLayout>
  );
}