import { useEffect, useState } from 'react';
import { useBackend } from '../contexts/BackendContext';
import { useNavigation } from '../hooks/useNavigation';
import api from '../utils/api';
import PageLayout from '../components/PageLayout';
import PageHeader from '../components/PageHeader';
import GlassyCard from '../components/GlassyCard';
import styles from './my-capabilities.module.css';

export default function MyCapabilities() {
  const { activePage } = useNavigation('my-capabilities');
  const { isRunning } = useBackend();
  const [capabilities, setCapabilities] = useState([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState('');
  const [selectedCapability, setSelectedCapability] = useState(null);
  const [searchQuery, setSearchQuery] = useState('');

  useEffect(() => {
    if (isRunning) fetchCapabilities();
  }, [isRunning]);

  const fetchCapabilities = async () => {
    setIsLoading(true);
    try {
      // User-scoped capabilities via Controller API
      const response = await api.get('/api/capabilities?owner=me');
      const all = response.data.capabilities || [];
      setCapabilities(all);
      setError('');
    } catch (e) {
      setError('Failed to fetch capabilities');
      console.error('Error fetching capabilities:', e);
    } finally {
      setIsLoading(false);
    }
  };

  const handleSearch = (q) => setSearchQuery(q);

  const filtered = capabilities.filter((cap) =>
    (cap.name + ' ' + cap.type + ' ' + (cap.owner || '')).toLowerCase().includes(searchQuery.toLowerCase())
  );

  if (!isRunning) {
    return (
      <PageLayout activePage={activePage} pageTitle="My Capabilities">
        <GlassyCard darker className={styles.errorCard}>
          Backend is not running. Please start the KNIRVORACLE node.
        </GlassyCard>
      </PageLayout>
    );
  }

  return (
    <PageLayout activePage={activePage} pageTitle="My Capabilities" onSearch={handleSearch}>
      <PageHeader title="My Capabilities" subtitle="View and manage your minted capabilities" />

      {error && (
        <GlassyCard darker className={styles.errorCard}>{error}</GlassyCard>
      )}

      <div className={styles.container}>
        <GlassyCard darker className={styles.list}>
          <div className={styles.headerRow}>
            <h3>Owned Capabilities</h3>
            <button onClick={fetchCapabilities} className={`${styles.button} ${styles.secondary}`} disabled={isLoading}>
              {isLoading ? 'Refreshing...' : 'Refresh'}
            </button>
          </div>

          {isLoading ? (
            <div className={styles.loading}>Loading...</div>
          ) : filtered.length === 0 ? (
            <div className={styles.empty}>No capabilities found</div>
          ) : (
            <div className={styles.grid}>
              {filtered.map((cap) => (
                <GlassyCard
                  key={cap.id}
                  darker
                  className={`${styles.item} ${selectedCapability?.id === cap.id ? styles.selected : ''}`}
                  onClick={() => setSelectedCapability(cap)}
                >
                  <div className={styles.name}>{cap.name}</div>
                  <div className={styles.type}>{cap.type}</div>
                  <div className={styles.id}>ID: {cap.id.substring(0, 10)}...</div>
                </GlassyCard>
              ))}
            </div>
          )}
        </GlassyCard>

        {selectedCapability && (
          <GlassyCard darker className={styles.details}>
            <h3>Capability Details</h3>
            <div className={styles.detailRow}><strong>Name:</strong> {selectedCapability.name}</div>
            <div className={styles.detailRow}><strong>Type:</strong> {selectedCapability.type}</div>
            <div className={styles.detailRow}><strong>ID:</strong> {selectedCapability.id}</div>
            {selectedCapability.owner && (
              <div className={styles.detailRow}><strong>Owner:</strong> {selectedCapability.owner}</div>
            )}
            <GlassyCard className={styles.subsection}>
              <h4 className={styles.subTitle}>Descriptor</h4>
              <div className={styles.descriptor}>
                {Object.entries(selectedCapability.descriptor || {}).map(([k, v]) => (
                  <div key={k} className={styles.descriptorItem}><strong>{k}:</strong> {JSON.stringify(v)}</div>
                ))}
              </div>
            </GlassyCard>
          </GlassyCard>
        )}
      </div>
    </PageLayout>
  );
}