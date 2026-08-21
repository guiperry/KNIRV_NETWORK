import React, { useState, useEffect } from 'react';
import { useNavigation } from '../hooks/useNavigation';
import api from '../utils/api';
import PageLayout from '../components/PageLayout';
import PageHeader from '../components/PageHeader';
import GlassyCard from '../components/GlassyCard';
import IframeModal from '../components/IframeModal';
import styles from './arena.module.css';

const TOURNAMENT_STATUS_COLORS = {
  active: '#4caf50',
  upcoming: '#ffc107',
  resolved: '#9e9e9e',
};

export default function Arena() {
  const { activePage } = useNavigation('arena');
  const [searchQuery, setSearchQuery] = useState('');
  const [tournaments, setTournaments] = useState([]);
  const [datasets, setDatasets] = useState([]);
  const [activeTab, setActiveTab] = useState('tournaments');
  const [isLoading, setIsLoading] = useState(true);
  const [modal, setModal] = useState({ open: false, title: '', src: '' });

  const handleSearch = (q) => setSearchQuery(q);

  const fetchArenaData = async () => {
    setIsLoading(true);
    try {
      const [tRes, dRes] = await Promise.allSettled([
        api.get('/api/v1/arena/tournaments'),
        api.get('/api/v1/arena/datasets'),
      ]);
      if (tRes.status === 'fulfilled') {
        const raw = tRes.value.data;
        setTournaments(Array.isArray(raw?.data || raw) ? (raw?.data || raw) : mockTournaments());
      } else {
        setTournaments(mockTournaments());
      }
      if (dRes.status === 'fulfilled') {
        const raw = dRes.value.data;
        setDatasets(Array.isArray(raw?.data || raw) ? (raw?.data || raw) : mockDatasets());
      } else {
        setDatasets(mockDatasets());
      }
    } catch {
      setTournaments(mockTournaments());
      setDatasets(mockDatasets());
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    fetchArenaData();
  }, []);

  const openTournament = (id) => {
    setModal({ open: true, title: 'KNIRVARENA — Tournament', src: `/arena/?arena=${encodeURIComponent(id)}` });
  };

  const openDataset = (id) => {
    setModal({ open: true, title: 'KNIRVARENA — Dataset', src: `/arena/datasets/${id}` });
  };

  const closeModal = () => setModal({ open: false, title: '', src: '' });

  const filteredTournaments = tournaments.filter((t) =>
    (t.name || '').toLowerCase().includes(searchQuery.toLowerCase())
  );
  const filteredDatasets = datasets.filter((d) =>
    (d.name || '').toLowerCase().includes(searchQuery.toLowerCase())
  );

  return (
    <PageLayout activePage={activePage} pageTitle="KNIRVARENA" onSearch={handleSearch}>
      <PageHeader
        title="KNIRVARENA"
        subtitle="Participate in tournaments, contribute datasets, and earn HERO Model compute rewards"
        titleColor="#ff6d00"
      />

      {/* Tab bar */}
      <div className={styles.tabBar}>
        <button
          className={`${styles.tab} ${activeTab === 'tournaments' ? styles.activeTab : ''}`}
          onClick={() => setActiveTab('tournaments')}
        >
          🏆 Tournaments
        </button>
        <button
          className={`${styles.tab} ${activeTab === 'datasets' ? styles.activeTab : ''}`}
          onClick={() => setActiveTab('datasets')}
        >
          📦 Pretrained Datasets
        </button>
      </div>

      {isLoading && (
        <div className={styles.centerState}>
          <div className={styles.spinner}></div>
          <p>Loading Arena data...</p>
        </div>
      )}

      {/* Tournaments tab */}
      {!isLoading && activeTab === 'tournaments' && (
        <div className={styles.grid}>
          {filteredTournaments.length === 0 && (
            <GlassyCard className={styles.emptyCard}>
              <div className={styles.emptyIcon}>🏟️</div>
              <h3>No tournaments found</h3>
              <p>Check back soon — the next tournament will be announced shortly.</p>
            </GlassyCard>
          )}
          {filteredTournaments.map((t) => (
            <GlassyCard key={t.id} className={styles.card} onClick={() => openTournament(t.id)}>
              <div className={styles.cardHeader}>
                <div className={styles.cardTitle}>{t.name || `Tournament #${t.id}`}</div>
                <span
                  className={styles.statusBadge}
                  style={{ color: TOURNAMENT_STATUS_COLORS[t.status] || '#9e9e9e' }}
                >
                  {t.status}
                </span>
              </div>
              <p className={styles.cardDesc}>{t.description || 'Compete for HERO Model compute rewards.'}</p>
              <div className={styles.cardMeta}>
                <span>🏅 Reward: {t.reward || '—'} NRN</span>
                <span>👥 Participants: {t.participants ?? '—'}</span>
              </div>
              <button className={styles.enterBtn} onClick={(e) => { e.stopPropagation(); openTournament(t.id); }}>
                {t.status === 'resolved' ? 'View Results' : 'Enter Tournament'} →
              </button>
            </GlassyCard>
          ))}
        </div>
      )}

      {/* Datasets tab */}
      {!isLoading && activeTab === 'datasets' && (
        <div className={styles.grid}>
          {filteredDatasets.length === 0 && (
            <GlassyCard className={styles.emptyCard}>
              <div className={styles.emptyIcon}>📦</div>
              <h3>No datasets available</h3>
              <p>Resolved or pretrained datasets will appear here.</p>
            </GlassyCard>
          )}
          {filteredDatasets.map((d) => (
            <GlassyCard key={d.id} className={styles.card} onClick={() => openDataset(d.id)}>
              <div className={styles.cardHeader}>
                <div className={styles.cardTitle}>{d.name || `Dataset #${d.id}`}</div>
                <span className={styles.statusBadge} style={{ color: '#00bcd4' }}>
                  {d.status || 'resolved'}
                </span>
              </div>
              <p className={styles.cardDesc}>{d.description || 'Pretrained dataset contributed by KNIRVARENA architects.'}</p>
              <div className={styles.cardMeta}>
                <span>📊 Entries: {d.entries ?? '—'}</span>
                <span>⭐ Score: {d.score ?? '—'}</span>
              </div>
              <button className={styles.enterBtn} onClick={(e) => { e.stopPropagation(); openDataset(d.id); }}>
                View Dataset →
              </button>
            </GlassyCard>
          ))}
        </div>
      )}

      {modal.open && (
        <IframeModal title={modal.title} src={modal.src} onClose={closeModal} height="90vh" />
      )}
    </PageLayout>
  );
}

function mockTournaments() {
  return [
    { id: '1', name: 'KNIRVARENA Open #1', status: 'active', description: 'Compete with your DVEs to resolve error nodes. Top contributors earn HERO Model compute rewards.', reward: '5,000', participants: 42 },
    { id: '2', name: 'Inference Challenge Alpha', status: 'upcoming', description: 'Submit TRL-compatible datasets for active error nodes. Rewards distributed based on dataset contribution scores.', reward: '2,500', participants: 0 },
    { id: '3', name: 'DVE Validation Sprint', status: 'resolved', description: 'Closed tournament — view final standings and reward distribution.', reward: '3,000', participants: 87 },
  ];
}

function mockDatasets() {
  return [
    { id: '1', name: 'ErrorNode Resolution Set v1', status: 'pretrained', description: 'Human Architect-crafted dataset for financial error nodes. Read by HERO Model during resolution attempts.', entries: 1240, score: 94.5 },
    { id: '2', name: 'Network Inference Corpus', status: 'resolved', description: 'Curated inference dataset from KNIRVARENA Open #1. Includes skill.md files and resolution chains.', entries: 860, score: 89.2 },
  ];
}
