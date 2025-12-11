import { useEffect, useState, useCallback } from 'react';
import { useBackend } from '../contexts/BackendContext';
import { useNavigation } from '../hooks/useNavigation';
import api from '../utils/api';
import PageLayout from '../components/PageLayout';
import PageHeader from '../components/PageHeader';
import GlassyCard from '../components/GlassyCard';
import styles from './peers.module.css';

export default function Peers() {
  const { activePage } = useNavigation('devs');
  const { isRunning } = useBackend();
  const [devs, setPeers] = useState([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState('');
  const [selectedPeer, setSelectedPeer] = useState(null);
  const [filter, setFilter] = useState('all');

  const fetchPeers = useCallback(async () => {
    setIsLoading(true);
    try {
      const response = await api.get('/mcp/dev/list');
      let filteredPeers = response.data.devs || [];
      
      if (filter === 'bootnodes') {
        filteredPeers = filteredPeers.filter(dev => dev.is_bootnode);
      } else if (filter === 'regular') {
        filteredPeers = filteredPeers.filter(dev => !dev.is_bootnode);
      }
      
      setPeers(filteredPeers);
      setError('');
    } catch (error) {
      setError('Failed to fetch devs');
      console.error('Error fetching devs:', error);
    } finally {
      setIsLoading(false);
    }
  }, [filter]);

  useEffect(() => {
    if (isRunning) {
      fetchPeers();
    }
  }, [isRunning, filter, fetchPeers]);

  const handlePeerClick = (dev) => {
    setSelectedPeer(dev);
  };

  const handleFilterChange = (newFilter) => {
    setFilter(newFilter);
    setSelectedPeer(null);
  };

  if (!isRunning) {
    return (
      <PageLayout activePage={activePage} pageTitle="Peers">
        <GlassyCard darker className={styles.errorCard}>
          Backend is not running. Please start the KNIRVORACLE node.
        </GlassyCard>
      </PageLayout>
    );
  }

  if (isLoading) {
    return (
      <PageLayout activePage={activePage} pageTitle="Peers">
        <GlassyCard darker className={styles.loading}>
          Loading devs...
        </GlassyCard>
      </PageLayout>
    );
  }

  return (
    <PageLayout
      activePage={activePage}
      pageTitle="Network Peers"
      onSearch={(query) => {
        // Implement search functionality if needed
        console.log('Search:', query);
      }}
    >
      {error && <GlassyCard darker className={styles.error}>{error}</GlassyCard>}
      
      <PageHeader
        title="Network Peers"
        subtitle={`${devs.length} devs connected`}
      />

      <GlassyCard darker className={styles.filterControls}>
        <div className={styles.filterButtons}>
          <button
            className={`${styles.button} ${filter === 'all' ? styles.primary : styles.secondary}`}
            onClick={() => handleFilterChange('all')}
          >
            All Peers
          </button>
          <button
            className={`${styles.button} ${filter === 'bootnodes' ? styles.primary : styles.secondary}`}
            onClick={() => handleFilterChange('bootnodes')}
          >
            Bootnodes
          </button>
          <button
            className={`${styles.button} ${filter === 'regular' ? styles.primary : styles.secondary}`}
            onClick={() => handleFilterChange('regular')}
          >
            Dev Nodes
          </button>
        </div>
        
        <button
          onClick={fetchPeers}
          className={`${styles.button} ${styles.secondary}`}
        >
          Refresh
        </button>
      </GlassyCard>

      <div className={styles.twoColumnLayout}>
        <GlassyCard darker className={styles.listContainer}>
          <h3 className={styles.sectionTitle}>Connected Peers ({devs.length})</h3>
          {devs.length === 0 ? (
            <div className={styles.emptyState}>No devs found</div>
          ) : (
            devs.map((dev) => (
              <GlassyCard 
                key={dev.id} 
                className={`${styles.devItem} ${selectedPeer?.id === dev.id ? styles.selected : ''}`}
                onClick={() => handlePeerClick(dev)}
              >
                <div className={styles.devType}>
                  {dev.is_bootnode ? '⛑️ Bootnode' : '🖥️ Node'}
                </div>
                <div className={styles.devName}>{dev.name || 'Unnamed Peer'}</div>
                <div className={styles.devId}>ID: {dev.id.substring(0, 8)}...</div>
                <div className={styles.devStatus}>
                  Status: <span className={dev.connected ? styles.connected : styles.disconnected}>
                    {dev.connected ? 'Connected' : 'Disconnected'}
                  </span>
                </div>
              </GlassyCard>
            ))
          )}
        </GlassyCard>
        
        {selectedPeer && (
          <GlassyCard darker className={styles.detailsContainer}>
            <h3 className={styles.sectionTitle}>Peer Details</h3>
            <div className={styles.detailItem}>
              <strong>Name:</strong> {selectedPeer.name || 'Unnamed Peer'}
            </div>
            <div className={styles.detailItem}>
              <strong>Type:</strong> {selectedPeer.is_bootnode ? 'Bootnode' : 'Dev Node'}
            </div>
            <div className={styles.detailItem}>
              <strong>ID:</strong> {selectedPeer.id}
            </div>
            <div className={styles.detailItem}>
              <strong>Status:</strong> 
              <span className={selectedPeer.connected ? styles.connected : styles.disconnected}>
                {selectedPeer.connected ? 'Connected' : 'Disconnected'}
              </span>
            </div>
            <div className={styles.detailItem}>
              <strong>Address:</strong> {selectedPeer.address}
            </div>
            <div className={styles.detailItem}>
              <strong>Last Seen:</strong> {selectedPeer.last_seen}
            </div>
            <div className={styles.detailItem}>
              <strong>Capabilities:</strong> {selectedPeer.capabilities?.join(', ') || 'None'}
            </div>
            
            <div className={styles.actionsSection}>
              {selectedPeer.connected ? (
                <button 
                  className={`${styles.button} ${styles.danger}`}
                >
                  Disconnect
                </button>
              ) : (
                <button 
                  className={`${styles.button} ${styles.success}`}
                >
                  Connect
                </button>
              )}
            </div>
          </GlassyCard>
        )}
      </div>
    </PageLayout>
  );
}