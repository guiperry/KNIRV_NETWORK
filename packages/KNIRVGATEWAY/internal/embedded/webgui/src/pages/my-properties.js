import React, { useEffect, useState } from 'react';
import { useNavigation } from '../hooks/useNavigation';
import { useBackend } from '../contexts/BackendContext';
import api from '../utils/api';
import PageLayout from '../components/PageLayout';
import PageHeader from '../components/PageHeader';
import GlassyCard from '../components/GlassyCard';
import styles from './my-properties.module.css';

export default function MyProperties() {
  const { activePage } = useNavigation('my-properties');
  const { isRunning } = useBackend();
  const [searchQuery, setSearchQuery] = useState('');
  const [properties, setProperties] = useState([]);
  const [selected, setSelected] = useState(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState('');

  const handleSearch = (q) => setSearchQuery(q);

  useEffect(() => {
    if (isRunning) fetchProperties();
  }, [isRunning]);

  const fetchProperties = async () => {
    setIsLoading(true);
    try {
      const res = await api.get('/api/properties?owner=me');
      setProperties(res.data.properties || []);
      setError('');
    } catch (e) {
      console.error('Failed to fetch properties', e);
      setError('Failed to fetch properties');
    } finally {
      setIsLoading(false);
    }
  };

  const filtered = properties.filter((p) =>
    [p.name, p.type, p.size, p.value].join(' ').toLowerCase().includes(searchQuery.toLowerCase())
  );

  if (!isRunning) {
    return (
      <PageLayout activePage={activePage} pageTitle="My Properties">
        <GlassyCard darker className={styles.emptyState}>
          Backend is not running. Please start the Controller API.
        </GlassyCard>
      </PageLayout>
    );
  }

  return (
    <PageLayout activePage={activePage} pageTitle="My Properties" onSearch={handleSearch}>
      <PageHeader title="My Properties" subtitle="Manage datasets, artifacts, licenses, and on-chain property" titleColor="#007bff" />

      <div className={styles.grid}>
        {isLoading ? (
          <div className={styles.loading}>Loading...</div>
        ) : error ? (
          <GlassyCard darker className={styles.emptyState}>{error}</GlassyCard>
        ) : filtered.length === 0 ? (
          <GlassyCard className={styles.emptyState}>
            <div className={styles.emptyIcon}>🏷️</div>
            <h3>No properties found</h3>
            <p>Add datasets, artifacts, or NFTs to see them here.</p>
          </GlassyCard>
        ) : (
          filtered.map((prop) => (
            <GlassyCard key={prop.id} className={styles.card} onClick={() => setSelected(prop)}>
              <div className={styles.cardHeader}>
                <div className={styles.cardTitle}>{prop.name}</div>
                <div className={styles.metaSmall}>{prop.type} · {prop.size}</div>
              </div>
              <div className={styles.meta}>
                <div className={styles.metaRow}><span className={styles.label}>Value:</span><span className={styles.value}>{prop.value}</span></div>
                <div className={styles.metaRow}><span className={styles.label}>Owner:</span><span className={styles.value}>{prop.owner}</span></div>
                <div className={styles.metaRow}><span className={styles.label}>Created:</span><span className={styles.value}>{prop.createdAt}</span></div>
              </div>
            </GlassyCard>
          ))
        )}
      </div>

      {selected && (
        <GlassyCard darker className={styles.detailsPanel}>
          <div className={styles.panelHeader}>
            <h3>{selected.name}</h3>
            <button className={styles.closeBtn} onClick={() => setSelected(null)}>×</button>
          </div>
          <div className={styles.panelBody}>
            <div className={styles.detailRow}><strong>Type:</strong> {selected.type}</div>
            <div className={styles.detailRow}><strong>Size:</strong> {selected.size}</div>
            <div className={styles.detailRow}><strong>Value:</strong> {selected.value}</div>
            <div className={styles.detailRow}><strong>Owner:</strong> {selected.owner}</div>
            <div className={styles.detailRow}><strong>Created:</strong> {selected.createdAt}</div>
          </div>
          <div className={styles.panelActions}>
            <button className={styles.primaryBtn}>Transfer</button>
          </div>
        </GlassyCard>
      )}
    </PageLayout>
  );
}