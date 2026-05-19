import React, { useEffect, useState } from 'react';
import { useNavigation } from '../hooks/useNavigation';
import api from '../utils/api';
import PageLayout from '../components/PageLayout';
import PageHeader from '../components/PageHeader';
import GlassyCard from '../components/GlassyCard';
import IframeModal from '../components/IframeModal';
import styles from './dve-list.module.css';

const STATUS_COLORS = {
  online: '#4caf50',
};

const TEE_ICONS = {
  sgx: '🔒',
  'sev-snp': '🛡️',
  tdx: '🔐',
  software: '💻',
  'browser-dve': '🌐',
};

export default function DVEList() {
  const { activePage } = useNavigation('dve-list');
  const [searchQuery, setSearchQuery] = useState('');
  const [activeNodes, setActiveNodes] = useState([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState('');
  const [modal, setModal] = useState({ open: false, title: '', src: '' });

  const handleSearch = (q) => setSearchQuery(q);

  const fetchNodes = async () => {
    setIsLoading(true);
    setError('');
    try {
      const res = await api.get('/api/dve-nodes');
      const raw = res.data;
      const allNodes = Array.isArray(raw?.data || raw?.nodes || raw) ? (raw?.data || raw?.nodes || raw) : [];
      // Strict filter: only nodes with status === 'online' are shown
      setActiveNodes(allNodes.filter(n => n && n.status === 'online'));
    } catch (e) {
      console.error('Failed to fetch DVE nodes:', e);
      setError('Could not load DVE nodes. Ensure KNIRVSERVER backend is running.');
      setActiveNodes([]);
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    fetchNodes();
  }, []);

  const openDVEExplorer = (node) => {
    setModal({
      open: true,
      title: `DVE File Manager — ${node.name || node.id}`,
      src: `/dve/${encodeURIComponent(node.id)}`,
    });
  };

  const closeModal = () => {
    setModal({ open: false, title: '', src: '' });
  };

  const filtered = activeNodes.filter((n) =>
    [n.name, n.id, n.status, n.tee_type, n.location]
      .filter(Boolean)
      .join(' ')
      .toLowerCase()
      .includes(searchQuery.toLowerCase())
  );

  return (
    <PageLayout activePage={activePage} pageTitle="My DVEs" onSearch={handleSearch}>
      <PageHeader
        title="My DVEs"
        subtitle={`${activeNodes.length} actively accessible DVE${activeNodes.length !== 1 ? 's' : ''} — activated by admin`}
        titleColor="#7c4dff"
      />

      {/* Stats bar */}
      <div className={styles.statsBar}>
        <div className={styles.statCard}>
          <span className={styles.statValue} style={{ color: '#4caf50' }}>{activeNodes.length}</span>
          <span className={styles.statLabel}>Active DVEs</span>
        </div>
        <div className={styles.statCard}>
          <button className={styles.refreshBtn} onClick={fetchNodes} title="Refresh">
            ↻ Refresh
          </button>
        </div>
      </div>

      {/* Loading state */}
      {isLoading && (
        <div className={styles.centerState}>
          <div className={styles.spinner}></div>
          <p>Loading active DVE nodes...</p>
        </div>
      )}

      {/* Error state */}
      {error && !isLoading && (
        <GlassyCard darker className={styles.centerState}>
          <div className={styles.errorIcon}>⚠️</div>
          <p className={styles.errorText}>{error}</p>
          <button className={styles.retryBtn} onClick={fetchNodes}>Retry</button>
        </GlassyCard>
      )}

      {/* Empty state — no active DVEs */}
      {!isLoading && !error && filtered.length === 0 && (
        <GlassyCard className={styles.centerState}>
          <div className={styles.emptyIcon}>🖥️</div>
          <h3>No active DVEs</h3>
          <p>
            {searchQuery
              ? 'No active DVEs match your search.'
              : 'No DVEs are currently active. An admin needs to activate a DVE from the backend or admin dashboard before it appears here.'}
          </p>
        </GlassyCard>
      )}

      {/* DVE Grid — only active/online nodes */}
      {!isLoading && !error && filtered.length > 0 && (
        <div className={styles.grid}>
          {filtered.map((node) => (
            <GlassyCard key={node.id} className={styles.card} onClick={() => openDVEExplorer(node)}>
              <div className={styles.cardHeader}>
                <div className={styles.cardTitleRow}>
                  <span className={styles.teeIcon}>
                    {TEE_ICONS[node.tee_type] || '📦'}
                  </span>
                  <div>
                    <div className={styles.cardTitle}>{node.name || node.id}</div>
                    <div className={styles.cardId}>ID: {node.id.substring(0, 12)}...</div>
                  </div>
                </div>
                <span
                  className={styles.statusBadge}
                  style={{
                    background: '#4caf5020',
                    color: '#4caf50',
                    borderColor: '#4caf5040',
                  }}
                >
                  active
                </span>
              </div>

              <div className={styles.cardBody}>
                <div className={styles.detailRow}>
                  <span className={styles.detailLabel}>TEE</span>
                  <span className={styles.detailValue}>{node.tee_type || '—'}</span>
                </div>
                <div className={styles.detailRow}>
                  <span className={styles.detailLabel}>Score</span>
                  <span className={styles.detailValue}>{node.reputation_score ?? '—'}</span>
                </div>
                {node.location && (
                  <div className={styles.detailRow}>
                    <span className={styles.detailLabel}>Location</span>
                    <span className={styles.detailValue}>{node.location}</span>
                  </div>
                )}
                {node.ip_address && (
                  <div className={styles.detailRow}>
                    <span className={styles.detailLabel}>Endpoint</span>
                    <span className={styles.detailValueMono}>{node.ip_address}</span>
                  </div>
                )}
              </div>

              <div className={styles.cardFooter}>
                <span className={styles.launchBtn} onClick={(e) => { e.stopPropagation(); openDVEExplorer(node); }}>
                  Open DVE Explorer →
                </span>
              </div>
            </GlassyCard>
          ))}
        </div>
      )}

      {/* DVE Explorer Modal */}
      {modal.open && (
        <IframeModal title={modal.title} src={modal.src} onClose={closeModal} height="90vh" />
      )}
    </PageLayout>
  );
}
