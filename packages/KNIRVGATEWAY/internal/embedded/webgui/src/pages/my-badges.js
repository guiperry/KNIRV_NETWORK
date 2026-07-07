import React, { useEffect, useState } from 'react';
import { useNavigation } from '../hooks/useNavigation';
import api from '../utils/api';
import PageLayout from '../components/PageLayout';
import PageHeader from '../components/PageHeader';
import GlassyCard from '../components/GlassyCard';
import styles from './my-badges.module.css';

export default function MyBadges() {
  const { activePage } = useNavigation('my-badges');
  const [searchQuery, setSearchQuery] = useState('');
  const [templates, setTemplates] = useState([]);
  const [selected, setSelected] = useState(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState('');

  const handleSearch = (q) => setSearchQuery(q);

  useEffect(() => {
    fetchTemplates();
  }, []);

  const fetchTemplates = async () => {
    setIsLoading(true);
    try {
      const res = await api.get('/api/badge/templates?active=true');
      setTemplates(res.data.templates || []);
      setError('');
    } catch (e) {
      console.error('Failed to fetch badge templates', e);
      setError('Failed to load badge templates. Ensure the KNIRVSERVER backend is running.');
    } finally {
      setIsLoading(false);
    }
  };

  const filtered = templates.filter((t) =>
    [t.name, t.badge_type, t.description].join(' ').toLowerCase().includes(searchQuery.toLowerCase())
  );

  return (
    <PageLayout activePage={activePage} pageTitle="Badges" onSearch={handleSearch}>
      <PageHeader title="Badges" subtitle="Browse available badge templates and mint them to your DVE workspaces" titleColor="#FBBF24" />

      <div className={styles.toolbar}>
        <button className={styles.refreshBtn} onClick={fetchTemplates}>
          ↻ Refresh
        </button>
        <span className={styles.count}>{templates.length} templates</span>
      </div>

      <div className={styles.grid}>
        {isLoading ? (
          <div className={styles.loading}>Loading badge templates...</div>
        ) : error ? (
          <GlassyCard darker className={styles.emptyState}>{error}</GlassyCard>
        ) : filtered.length === 0 ? (
          <GlassyCard className={styles.emptyState}>
            <div className={styles.emptyIcon}>🏅</div>
            <h3>No badge templates found</h3>
            <p>Admins can create badge templates in the Badge Lab (KNIRVSERVER Dashboard).</p>
          </GlassyCard>
        ) : (
          filtered.map((tmpl) => (
            <GlassyCard key={tmpl.id} className={styles.card} onClick={() => setSelected(tmpl)}>
              {/* SVG Preview */}
              {tmpl.svg_design && (
                <div className={styles.svgPreview}
                  dangerouslySetInnerHTML={{ __html: tmpl.svg_design }}
                />
              )}
              <div className={styles.cardHeader}>
                <div className={styles.cardTitle}>{tmpl.name}</div>
                <div className={styles.badgeType}>{tmpl.badge_type || 'capability'}</div>
              </div>
              {tmpl.description && (
                <p className={styles.description}>{tmpl.description}</p>
              )}
              <div className={styles.tags}>
                {tmpl.value_signals?.slice(0, 3).map((v) => (
                  <span key={v} className={styles.tagValue}>{v}</span>
                ))}
                {tmpl.value_signals?.length > 3 && (
                  <span className={styles.tagMore}>+{tmpl.value_signals.length - 3}</span>
                )}
                {tmpl.ontology_signals?.slice(0, 2).map((o) => (
                  <span key={o} className={styles.tagOntology}>{o}</span>
                ))}
              </div>
              <div className={styles.meta}>
                <div className={styles.metaRow}>
                  <span className={styles.label}>Created:</span>
                  <span className={styles.value}>{new Date(tmpl.created_at).toLocaleDateString()}</span>
                </div>
              </div>
            </GlassyCard>
          ))
        )}
      </div>

      {selected && (
        <div className={styles.modal} onClick={() => setSelected(null)}>
          <div className={styles.modalContent} onClick={(e) => e.stopPropagation()}>
            <div className={styles.modalHeader}>
              <h3>{selected.name}</h3>
              <button className={styles.closeBtn} onClick={() => setSelected(null)}>×</button>
            </div>
            <div className={styles.modalBody}>
              {selected.svg_design && (
                <div className={styles.modalSvg}
                  dangerouslySetInnerHTML={{ __html: selected.svg_design }}
                />
              )}
              <div className={styles.detailRow}><strong>Type:</strong> {selected.badge_type || 'capability'}</div>
              <div className={styles.detailRow}><strong>Description:</strong> {selected.description || '—'}</div>
              <div className={styles.detailRow}>
                <strong>Value Signals:</strong>
                <div className={styles.tagRow}>
                  {selected.value_signals?.map((v) => <span key={v} className={styles.tagValue}>{v}</span>)}
                  {(!selected.value_signals || selected.value_signals.length === 0) && <span className={styles.na}>None</span>}
                </div>
              </div>
              <div className={styles.detailRow}>
                <strong>Ontology:</strong>
                <div className={styles.tagRow}>
                  {selected.ontology_signals?.map((o) => <span key={o} className={styles.tagOntology}>{o}</span>)}
                  {(!selected.ontology_signals || selected.ontology_signals.length === 0) && <span className={styles.na}>None</span>}
                </div>
              </div>
              <div className={styles.detailRow}><strong>AI Text Tag:</strong> {selected.ai_text_tag || '—'}</div>
              <div className={styles.detailRow}><strong>Alignment Threshold:</strong> {selected.alignment_threshold || '0.75'}</div>
              <div className={styles.detailRow}><strong>Auth:</strong> {selected.auth_credentials?.api_key_provider ? `${selected.auth_credentials.api_key_provider} · ${selected.auth_credentials.api_key_scope}` : 'None'}</div>
              <div className={styles.detailRow}><strong>Created:</strong> {new Date(selected.created_at).toLocaleString()}</div>
            </div>
          </div>
        </div>
      )}
    </PageLayout>
  );
}
