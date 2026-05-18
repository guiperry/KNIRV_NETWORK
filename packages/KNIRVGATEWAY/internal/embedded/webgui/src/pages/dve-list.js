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
  offline: '#f44336',
  maintenance: '#ff9800',
  error: '#e91e63',
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
  const [nodes, setNodes] = useState([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState('');
  const [modal, setModal] = useState({ open: false, title: '', src: '' });

  const handleSearch = (q) => setSearchQuery(q);

  const fetchNodes = async () => {
    setIsLoading(true);
    setError('');
    try {
      // Only fetch DVEs that are online (activated programatically or via admin dashboard)
      const res = await api.get('/api/dve-nodes?status=online');
      const data = res.data?.data || res.data?.nodes || [];
      // Double-check: only include nodes that are genuinely online/activated
      setNodes(Array.isArray(data) ? data.filter(n => n.status === 'online') : []);
    } catch (e) {
      console.error('Failed to fetch DVE nodes:', e);
      setError('Could not load DVE nodes. Ensure KNIRVSERVER backend is running.');
      setNodes([]);
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

  const filtered = nodes.filter((n) =>
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
        subtitle={`Your actively accessible Decentralized Verifiable Execution environments — ${nodes.length} available`}
        titleColor="#7c4dff"
      />

      {/* Stats bar */}
      <div className={styles.statsBar}>
        <div className={styles.statCard}>
          <span className={styles.statValue}>{nodes.length}</span>
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
          <p>Loading DVE nodes...</p>
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

      {/* Empty state */}
      {!isLoading && !error && filtered.length === 0 && (
        <GlassyCard className={styles.centerState}>
          <div className={styles.emptyIcon}>🖥️</div>
          <h3>No DVEs found</h3>
          <p>{searchQuery ? 'No DVEs match your search.' : 'No DVE nodes available. They will appear here once registered.'}</p>
        </GlassyCard>
      )}

      {/* DVE Grid */}
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
                    background: `${STATUS_COLORS[node.status] || '#888'}20`,
                    color: STATUS_COLORS[node.status] || '#888',
                    borderColor: `${STATUS_COLORS[node.status] || '#888'}40`,
                  }}
                >
                  {node.status || 'unknown'}
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
