import React, { useEffect, useState, useCallback, useMemo } from 'react';
import { useNavigation } from '../hooks/useNavigation';
import PageLayout from '../components/PageLayout';
import PageHeader from '../components/PageHeader';
import GlassyCard from '../components/GlassyCard';
import StatsCard from '../components/GraphChain/StatsCard';
import LoadingSpinner from '../components/GraphChain/LoadingSpinner';
import { graphChainApi } from '../services/graphchain-api';
import styles from './graph-explorer.module.css';

// ===== TABS CONFIGURATION =====
const TABS = [
  { id: 'overview',       label: 'Overview',           icon: 'fa-chart-pie' },
  { id: 'errors',         label: 'ErrorNodes',          icon: 'fa-exclamation-triangle' },
  { id: 'vectors',        label: 'Vectors',             icon: 'fa-arrow-right' },
  { id: 'visualization',  label: 'Graph Visualization', icon: 'fa-project-diagram' },
];

// ===== HELPERS =====

function formatTime(timestamp) {
  return new Date(timestamp).toLocaleString();
}

function getSeverityColor(severity) {
  if (severity >= 4) return styles.severityCritical;
  if (severity >= 3) return styles.severityHigh;
  if (severity >= 2) return styles.severityMedium;
  return styles.severityLow;
}

function getSeverityLabel(severity) {
  if (severity >= 4) return 'CRITICAL';
  if (severity >= 3) return 'HIGH';
  if (severity >= 2) return 'MEDIUM';
  return 'LOW';
}

function getConfidenceLabel(confidence) {
  if (confidence >= 0.8) return 'HIGH';
  if (confidence >= 0.5) return 'MEDIUM';
  return 'LOW';
}

function getConfidenceClass(confidence) {
  if (confidence >= 0.8) return styles.confidenceHigh;
  if (confidence >= 0.5) return styles.confidenceMedium;
  return styles.confidenceLow;
}

function getStatusIcon(status) {
  switch (status) {
    case 'resolved': return 'fa-check-circle';
    case 'failed':   return 'fa-times-circle';
    default:         return 'fa-clock';
  }
}

function getStatusColor(status) {
  switch (status) {
    case 'resolved': return styles.statusResolved;
    case 'failed':   return styles.statusFailed;
    default:         return styles.statusPending;
  }
}

// ===== OVERVIEW TAB =====
function OverviewTab({ stats, onTabSwitch }) {
  const statusItems = [
    { label: 'GraphChain Status', value: 'Online', indicator: true, color: '#4ade80' },
    { label: 'Total SkillNodes',  value: stats.skills ?? 0 },
    { label: 'Total ErrorNodes',  value: stats.errors ?? 0 },
    { label: 'Total Vectors',     value: stats.vectors ?? 0 },
    { label: 'Graph Height',      value: stats.height ?? 0 },
  ];

  const quickActions = [
    { icon: '🔗', title: 'ErrorNodes',    desc: 'Explore error nodes and resolution paths',      tab: 'errors' },
    { icon: '🧭', title: 'Vectors',       desc: 'Browse NRV vectors and confidence scores',       tab: 'vectors' },
    { icon: '🌐', title: 'Graph View',    desc: 'Interactive node relationship visualization',    tab: 'visualization' },
    { icon: '🧠', title: 'SkillNodes',    desc: 'View all skill execution nodes',                 tab: 'skills' },
  ];

  return (
    <div className={styles.tabInner}>
      {/* Stats Cards */}
      <div className={styles.statsGrid}>
        <StatsCard title="Graph Height"   value={stats.height ?? '—'}  icon="fa-layer-group" color="blue" />
        <StatsCard title="SkillNodes"     value={stats.skills ?? '—'}  icon="fa-brain"       color="green" />
        <StatsCard title="ErrorNodes"     value={stats.errors ?? '—'}  icon="fa-exclamation-triangle" color="orange" />
        <StatsCard title="Vectors"        value={stats.vectors ?? '—'} icon="fa-arrow-right" color="purple" />
      </div>

      {/* Quick Actions */}
      <div className={styles.sectionHeading}>
        <i className="fas fa-bolt"></i> Quick Actions
      </div>
      <div className={styles.quickActions}>
        {quickActions.map((action) => (
          <GlassyCard
            key={action.tab}
            className={styles.actionCard}
            onClick={() => onTabSwitch(action.tab)}
          >
            <div className={styles.actionIcon}>{action.icon}</div>
            <h4>{action.title}</h4>
            <p>{action.desc}</p>
            <button
              className={styles.actionButton}
              onClick={(e) => { e.stopPropagation(); onTabSwitch(action.tab); }}
            >
              View {action.title}
            </button>
          </GlassyCard>
        ))}
      </div>

      {/* Network Status */}
      <GlassyCard className={styles.statusCard}>
        <div className={styles.statusHeader}>
          <h3 className={styles.statusTitle}>
            <i className="fas fa-heartbeat"></i>
            Network Status
          </h3>
        </div>
        <div className={styles.statusGrid}>
          {statusItems.map((item) => (
            <div key={item.label} className={styles.statusItem}>
              <div className={styles.statusLabel}>{item.label}</div>
              <div className={styles.statusValue}>
                {item.indicator && (
                  <span className={styles.statusIndicator} style={{ backgroundColor: item.color }} />
                )}
                {item.value}
              </div>
            </div>
          ))}
        </div>
      </GlassyCard>
    </div>
  );
}

// ===== ERROR NODES TAB =====
function ErrorNodesTab() {
  const [errors, setErrors] = useState([]);
  const [skills, setSkills] = useState([]);
  const [isLoading, setIsLoading] = useState(true);
  const [fetchError, setFetchError] = useState(null);
  const [selectedError, setSelectedError] = useState(null);
  const [relatedSkills, setRelatedSkills] = useState([]);

  useEffect(() => {
    const fetchData = async () => {
      setIsLoading(true);
      setFetchError(null);
      try {
        const [fetchedErrors, fetchedSkills] = await Promise.all([
          graphChainApi.getAllErrors(),
          graphChainApi.getAllSkills(),
        ]);
        fetchedErrors.sort((a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime());
        setErrors(fetchedErrors);
        setSkills(fetchedSkills);
      } catch (err) {
        setFetchError(err.message || 'Failed to fetch ErrorNodes');
      } finally {
        setIsLoading(false);
      }
    };
    fetchData();
  }, []);

  useEffect(() => {
    let cancelled = false;
    const fetchRelated = async () => {
      if (selectedError) {
        try {
          const related = await graphChainApi.getSkillsForError(selectedError.error_type);
          if (!cancelled) setRelatedSkills(related);
        } catch (err) {
          if (!cancelled) setRelatedSkills([]);
        }
      } else {
        setRelatedSkills([]);
      }
    };
    fetchRelated();
    return () => { cancelled = true; };
  }, [selectedError]);

  const handleErrorClick = useCallback((errorNode) => {
    setSelectedError((prev) => (prev?.id === errorNode.id ? null : errorNode));
  }, []);

  const highSeverity = errors.filter((e) => e.severity >= 3).length;
  const resolved    = errors.filter((e) => e.resolution_status === 'resolved').length;

  return (
    <div className={styles.tabInner}>
      <div className={styles.errorsContainer}>
        {/* Stats */}
        <GlassyCard className={styles.statsCard}>
          <div className={styles.statsGridCompact}>
            <div className={styles.statItem}>
              <div className={`${styles.statValue} ${styles.statOrange}`}>{errors.length}</div>
              <div className={styles.statLabel}>Total ErrorNodes</div>
            </div>
            <div className={styles.statItem}>
              <div className={`${styles.statValue} ${styles.statRed}`}>{highSeverity}</div>
              <div className={styles.statLabel}>High Severity</div>
            </div>
            <div className={styles.statItem}>
              <div className={`${styles.statValue} ${styles.statGreen}`}>{resolved}</div>
              <div className={styles.statLabel}>Resolved</div>
            </div>
            <div className={styles.statItem}>
              <div className={`${styles.statValue} ${styles.statBlue}`}>{skills.length}</div>
              <div className={styles.statLabel}>Available Skills</div>
            </div>
          </div>
        </GlassyCard>

        {/* Error List */}
        {isLoading ? (
          <div className={styles.loadingContainer}>
            <LoadingSpinner size="large" text="Loading ErrorNodes..." />
          </div>
        ) : fetchError ? (
          <GlassyCard className={styles.emptyCard} darker>
            <i className="fas fa-exclamation-triangle" style={{ color: '#f87171' }} />
            <h3>Error Loading ErrorNodes</h3>
            <p>{fetchError}</p>
          </GlassyCard>
        ) : errors.length === 0 ? (
          <GlassyCard className={styles.emptyCard}>
            <i className="fas fa-exclamation-triangle" />
            <h3>No ErrorNodes Found</h3>
            <p>There are no ErrorNodes in the system.</p>
          </GlassyCard>
        ) : (
          <div className={styles.errorsList}>
            {errors.map((errorNode) => (
              <GlassyCard
                key={errorNode.id}
                className={`${styles.errorCard} ${selectedError?.id === errorNode.id ? styles.errorCardSelected : ''}`}
                onClick={() => handleErrorClick(errorNode)}
              >
                {/* Header */}
                <div className={styles.errorHeader}>
                  <div className={styles.errorHeaderLeft}>
                    <div className={styles.errorIcon}>
                      <i className="fas fa-exclamation-triangle" />
                    </div>
                    <div>
                      <div className={styles.errorType}>{errorNode.error_type}</div>
                      <div className={styles.errorTimestamp}>
                        <i className="fas fa-clock" />
                        <span>{formatTime(errorNode.timestamp)}</span>
                      </div>
                    </div>
                  </div>
                  <div className={styles.errorHeaderRight}>
                    <div className={`${styles.severityBadge} ${getSeverityColor(errorNode.severity)}`}>
                      {getSeverityLabel(errorNode.severity)}
                    </div>
                    {errorNode.resolution_status && (
                      <div className={`${styles.statusBadge} ${getStatusColor(errorNode.resolution_status)}`}>
                        <i className={`fas ${getStatusIcon(errorNode.resolution_status)}`} />
                        <span>{errorNode.resolution_status}</span>
                      </div>
                    )}
                  </div>
                </div>

                {/* Description */}
                <div className={styles.errorDescription}>
                  <div className={styles.sectionTitle}>Description</div>
                  <div className={styles.descriptionContent}>{errorNode.description}</div>
                </div>

                {/* Details */}
                <div className={styles.errorDetails}>
                  <div className={styles.detailItem}>
                    <div className={styles.detailLabel}>
                      <i className="fas fa-hashtag" />
                      <span>Error ID</span>
                    </div>
                    <div className={styles.detailValue}>{errorNode.id}</div>
                  </div>
                  <div className={styles.detailItem}>
                    <div className={styles.detailLabel}>
                      <i className="fas fa-exclamation-triangle" />
                      <span>Error Type</span>
                    </div>
                    <div className={styles.detailValue}>{errorNode.error_type}</div>
                  </div>
                  <div className={styles.detailItem}>
                    <div className={styles.detailLabel}>
                      <i className="fas fa-tachometer-alt" />
                      <span>Severity Level</span>
                    </div>
                    <div className={styles.detailValue}>{errorNode.severity}/5</div>
                  </div>
                  {errorNode.resolved_by && errorNode.resolved_by.length > 0 && (
                    <div className={styles.detailItem}>
                      <div className={styles.detailLabel}>
                        <i className="fas fa-brain" />
                        <span>Resolved By</span>
                      </div>
                      <div className={styles.detailValue}>{errorNode.resolved_by.join(', ')}</div>
                    </div>
                  )}
                </div>

                {/* Context */}
                {errorNode.context && Object.keys(errorNode.context).length > 0 && (
                  <div className={styles.errorContext}>
                    <div className={styles.sectionTitle}>Error Context</div>
                    <pre className={styles.contextContent}>
                      {JSON.stringify(errorNode.context, null, 2)}
                    </pre>
                  </div>
                )}

                {/* Related Skills */}
                {selectedError?.id === errorNode.id && relatedSkills.length > 0 && (
                  <div className={styles.relatedSkills}>
                    <div className={styles.sectionTitle}>
                      Related SkillNodes ({relatedSkills.length})
                    </div>
                    <div className={styles.skillsGrid}>
                      {relatedSkills.slice(0, 4).map((skill) => (
                        <div key={skill.id} className={styles.skillCard}>
                          <div className={styles.skillHeader}>
                            <i className="fas fa-brain" />
                            <span className={styles.skillType}>{skill.skill_type}</span>
                          </div>
                          <div className={styles.skillCapabilities}>
                            {skill.capabilities.slice(0, 2).join(', ')}
                            {skill.capabilities.length > 2 && ` +${skill.capabilities.length - 2} more`}
                          </div>
                          {skill.performance && (
                            <div className={styles.skillPerformance}>
                              {(skill.performance.success_rate * 100).toFixed(0)}% success rate
                            </div>
                          )}
                        </div>
                      ))}
                    </div>
                    {relatedSkills.length > 4 && (
                      <div className={styles.moreSkills}>
                        +{relatedSkills.length - 4} more skills available
                      </div>
                    )}
                  </div>
                )}

                {/* Footer */}
                <div className={styles.errorFooter}>
                  <div className={styles.footerText}>
                    Click to {selectedError?.id === errorNode.id ? 'hide' : 'view'} related skills
                  </div>
                  <div className={styles.footerLink}>
                    <span>View Details</span>
                    <i className="fas fa-arrow-right" />
                  </div>
                </div>
              </GlassyCard>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}

// ===== VECTORS TAB =====
function VectorsTab() {
  const [vectors, setVectors] = useState([]);
  const [isLoading, setIsLoading] = useState(true);
  const [fetchError, setFetchError] = useState(null);
  const [expandedVector, setExpandedVector] = useState(null);

  useEffect(() => {
    const fetchData = async () => {
      setIsLoading(true);
      setFetchError(null);
      try {
        const fetched = await graphChainApi.getAllVectors();
        fetched.sort((a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime());
        setVectors(fetched);
      } catch (err) {
        setFetchError(err.message || 'Failed to fetch vectors');
      } finally {
        setIsLoading(false);
      }
    };
    fetchData();
  }, []);

  const toggleVector = useCallback((vector) => {
    setExpandedVector((prev) => (prev?.id === vector.id ? null : vector));
  }, []);

  const avgConfidence = useMemo(() => {
    if (vectors.length === 0) return 0;
    return vectors.reduce((sum, v) => sum + v.confidence, 0) / vectors.length;
  }, [vectors]);

  return (
    <div className={styles.tabInner}>
      <div className={styles.vectorsContainer}>
        {/* Stats */}
        <GlassyCard className={styles.statsCard}>
          <div className={styles.statsGridCompact}>
            <div className={styles.statItem}>
              <div className={`${styles.statValue} ${styles.statBlue}`}>{vectors.length}</div>
              <div className={styles.statLabel}>Total Vectors</div>
            </div>
            <div className={styles.statItem}>
              <div className={`${styles.statValue} ${styles.statGreen}`}>
                {vectors.filter((v) => v.confidence >= 0.8).length}
              </div>
              <div className={styles.statLabel}>High Confidence</div>
            </div>
            <div className={styles.statItem}>
              <div className={`${styles.statValue} ${styles.statOrange}`}>
                {vectors.filter((v) => v.confidence >= 0.5 && v.confidence < 0.8).length}
              </div>
              <div className={styles.statLabel}>Medium Confidence</div>
            </div>
            <div className={styles.statItem}>
              <div className={`${styles.statValue} ${styles.statPurple}`}>
                {(avgConfidence * 100).toFixed(1)}%
              </div>
              <div className={styles.statLabel}>Avg Confidence</div>
            </div>
          </div>
        </GlassyCard>

        {/* Vector List */}
        {isLoading ? (
          <div className={styles.loadingContainer}>
            <LoadingSpinner size="large" text="Loading NRV Vectors..." />
          </div>
        ) : fetchError ? (
          <GlassyCard className={styles.emptyCard} darker>
            <i className="fas fa-exclamation-triangle" style={{ color: '#f87171' }} />
            <h3>Error Loading Vectors</h3>
            <p>{fetchError}</p>
          </GlassyCard>
        ) : vectors.length === 0 ? (
          <GlassyCard className={styles.emptyCard}>
            <i className="fas fa-arrow-right" />
            <h3>No Vectors Found</h3>
            <p>There are no NRV vectors in the system.</p>
          </GlassyCard>
        ) : (
          <div className={styles.vectorsList}>
            {vectors.map((vector) => (
              <GlassyCard
                key={vector.id}
                className={styles.vectorCard}
                darker={expandedVector?.id === vector.id}
                onClick={() => toggleVector(vector)}
              >
                {/* Header */}
                <div className={styles.vectorHeader}>
                  <div className={styles.vectorHeaderLeft}>
                    <div className={styles.vectorIcon}>
                      <i className="fas fa-arrow-right" />
                    </div>
                    <div>
                      <div className={styles.vectorId}>{vector.id}</div>
                      <div className={styles.vectorTimestamp}>
                        <i className="fas fa-clock" />
                        <span>{formatTime(vector.timestamp)}</span>
                      </div>
                    </div>
                  </div>
                  <div className={styles.vectorHeaderRight}>
                    <div className={`${styles.confidenceBadge} ${getConfidenceClass(vector.confidence)}`}>
                      {getConfidenceLabel(vector.confidence)}
                    </div>
                  </div>
                </div>

                {/* Details (always visible) */}
                <div className={styles.vectorDetails}>
                  <div className={styles.vectorDetailItem}>
                    <div className={styles.vectorDetailLabel}>
                      <i className="fas fa-globe" />
                      <span>Source Peer</span>
                    </div>
                    <div className={styles.vectorDetailValue}>{vector.source_peer}</div>
                  </div>
                  <div className={styles.vectorDetailItem}>
                    <div className={styles.vectorDetailLabel}>
                      <i className="fas fa-link" />
                      <span>Target Hash</span>
                    </div>
                    <div className={styles.vectorDetailValue}>{vector.target_hash}</div>
                  </div>
                  <div className={styles.vectorDetailItem}>
                    <div className={styles.vectorDetailLabel}>
                      <i className="fas fa-percentage" />
                      <span>Confidence</span>
                    </div>
                    <div className={styles.vectorDetailValue}>
                      {(vector.confidence * 100).toFixed(1)}%
                    </div>
                  </div>
                  <div className={styles.vectorDetailItem}>
                    <div className={styles.vectorDetailLabel}>
                      <i className="fas fa-fingerprint" />
                      <span>Vector ID</span>
                    </div>
                    <div className={styles.vectorDetailValue}>{vector.id}</div>
                  </div>
                </div>

                {/* Coordinates (expandable) */}
                {expandedVector?.id === vector.id && vector.coordinates && vector.coordinates.length > 0 && (
                  <div className={styles.coordinatesContainer}>
                    <div className={styles.coordinatesHeader}>
                      <i className="fas fa-vector-square" />
                      NRV Coordinates ({vector.coordinates.length} dimensions)
                    </div>
                    <div className={styles.coordinatesBar}>
                      {vector.coordinates.map((coord, idx) => (
                        <React.Fragment key={idx}>
                          <span
                            className={`${styles.coordinateDot} ${Math.abs(coord) > 0.5 ? styles.coordinateDotHigh : ''}`}
                            title={`dim ${idx}: ${coord.toFixed(4)}`}
                          />
                          <span className={styles.coordinateValue}>{coord.toFixed(2)}</span>
                        </React.Fragment>
                      ))}
                    </div>
                  </div>
                )}

                {/* Footer */}
                <div className={styles.coordinatesHeader} style={{ marginTop: '0.75rem', borderTop: 'none', paddingTop: 0 }}>
                  <i className="fas fa-chevron-down" style={{ transform: expandedVector?.id === vector.id ? 'rotate(180deg)' : 'none', transition: 'transform 0.2s' }} />
                  <span>Click to {expandedVector?.id === vector.id ? 'hide' : 'show'} coordinates</span>
                </div>
              </GlassyCard>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}

// ===== GRAPH VISUALIZATION TAB =====
function GraphVisualizationTab() {
  const [skills, setSkills] = useState([]);
  const [errors, setErrors] = useState([]);
  const [selectedNode, setSelectedNode] = useState(null);
  const [connectedNodes, setConnectedNodes] = useState([]);
  const [isLoading, setIsLoading] = useState(true);
  const [fetchError, setFetchError] = useState(null);
  const [filter, setFilter] = useState('all');

  useEffect(() => {
    const fetchData = async () => {
      setIsLoading(true);
      setFetchError(null);
      try {
        const [fetchedSkills, fetchedErrors] = await Promise.all([
          graphChainApi.getAllSkills(),
          graphChainApi.getAllErrors(),
        ]);
        setSkills(fetchedSkills);
        setErrors(fetchedErrors);
      } catch (err) {
        setFetchError(err.message || 'Failed to fetch graph data');
      } finally {
        setIsLoading(false);
      }
    };
    fetchData();
  }, []);

  // When a node is selected, fetch its connections via skill-error matching
  useEffect(() => {
    if (!selectedNode) {
      setConnectedNodes([]);
      return;
    }
    const fetchConnections = async () => {
      try {
        if (selectedNode.node_type === 'skill') {
          // For skill nodes, find errors this skill can resolve — use the first
          // capability as error-type heuristic, then get skills-for-error reverse
          const capabilities = selectedNode.capabilities || [];
          const matchingErrors = errors.filter(
            (e) => capabilities.some((cap) => e.error_type.toLowerCase().includes(cap.toLowerCase())) ||
                           e.error_type.toLowerCase().includes(selectedNode.skill_type.toLowerCase())
          );
          setConnectedNodes(matchingErrors.slice(0, 6));
        } else {
          // For error nodes, show skills that can resolve this error type
          const related = await graphChainApi.getSkillsForError(selectedNode.error_type);
          setConnectedNodes(related.slice(0, 6));
        }
      } catch (err) {
        console.error('Failed to fetch connections:', err);
        setConnectedNodes([]);
      }
    };
    fetchConnections();
  }, [selectedNode, skills, errors]);

  const filteredSkills = useMemo(() => {
    if (filter === 'errors') return [];
    return skills;
  }, [skills, filter]);

  const filteredErrors = useMemo(() => {
    if (filter === 'skills') return [];
    return errors;
  }, [errors, filter]);

  const handleNodeClick = useCallback((node, nodeType) => {
    setSelectedNode((prev) =>
      prev?.id === node.id && prev?.node_type === nodeType ? null : { ...node, node_type: nodeType }
    );
  }, []);

  const totalNodes = filteredSkills.length + filteredErrors.length;
  const totalConnections = useMemo(() => {
    // Estimate connections: each skill that has matching error types
    let count = 0;
    for (const skill of filteredSkills) {
      const caps = skill.capabilities || [];
      count += filteredErrors.filter((e) =>
        caps.some((cap) => e.error_type.toLowerCase().includes(cap.toLowerCase()))
      ).length;
    }
    return count;
  }, [filteredSkills, filteredErrors]);

  return (
    <div className={styles.tabInner}>
      <div className={styles.graphContainer}>
        {/* Filter Controls */}
        <div className={styles.graphControls}>
          <span className={styles.graphFilterLabel}>
            <i className="fas fa-filter" style={{ marginRight: 6 }} />
            Filter:
          </span>
          {['all', 'skills', 'errors'].map((f) => (
            <button
              key={f}
              className={`${styles.filterButton} ${filter === f ? styles.filterButtonActive : ''}`}
              onClick={() => { setFilter(f); setSelectedNode(null); }}
            >
              {f === 'all' ? 'All' : f === 'skills' ? 'Skills Only' : 'Errors Only'}
            </button>
          ))}
        </div>

        {/* Stats */}
        <div className={styles.graphStatsRow}>
          <GlassyCard className={styles.graphStatCard}>
            <div className={styles.graphStatValue}>{totalNodes}</div>
            <div className={styles.graphStatLabel}>Total Nodes</div>
          </GlassyCard>
          <GlassyCard className={styles.graphStatCard}>
            <div className={styles.graphStatValue}>{totalConnections}</div>
            <div className={styles.graphStatLabel}>Connections</div>
          </GlassyCard>
          <GlassyCard className={styles.graphStatCard}>
            <div className={styles.graphStatValue}>{filteredSkills.length}</div>
            <div className={styles.graphStatLabel}>SkillNodes</div>
          </GlassyCard>
          <GlassyCard className={styles.graphStatCard}>
            <div className={styles.graphStatValue}>{filteredErrors.length}</div>
            <div className={styles.graphStatLabel}>ErrorNodes</div>
          </GlassyCard>
        </div>

        {/* Node Relationship View */}
        {isLoading ? (
          <div className={styles.loadingContainer}>
            <LoadingSpinner size="large" text="Loading graph data..." />
          </div>
        ) : fetchError ? (
          <GlassyCard className={styles.emptyCard} darker>
            <i className="fas fa-exclamation-triangle" style={{ color: '#f87171' }} />
            <h3>Error Loading Graph Data</h3>
            <p>{fetchError}</p>
          </GlassyCard>
        ) : totalNodes === 0 ? (
          <GlassyCard className={styles.emptyCard}>
            <i className="fas fa-project-diagram" />
            <h3>No Nodes Found</h3>
            <p>There are no nodes matching the current filter.</p>
          </GlassyCard>
        ) : (
          <div className={styles.nodeRelationships}>
            {/* Skill Nodes */}
            {filteredSkills.length > 0 && (
              <div className={styles.nodeGroup}>
                <div className={`${styles.nodeGroupTitle} ${styles.skillsGroupTitle}`}>
                  <i className="fas fa-brain" />
                  SkillNodes ({filteredSkills.length})
                </div>
                <div className={styles.nodeChips}>
                  {filteredSkills.slice(0, 20).map((skill) => (
                    <div
                      key={skill.id}
                      className={`${styles.nodeChip} ${styles.skillNodeChip}`}
                      onClick={() => handleNodeClick(skill, 'skill')}
                      title={skill.skill_type}
                    >
                      <i className={`fas fa-brain ${styles.nodeChipIcon}`} />
                      <span className={styles.nodeChipLabel}>
                        {skill.skill_type.length > 20
                          ? skill.skill_type.slice(0, 20) + '…'
                          : skill.skill_type}
                      </span>
                    </div>
                  ))}
                  {filteredSkills.length > 20 && (
                    <div className={styles.nodeChip} style={{ opacity: 0.6 }}>
                      +{filteredSkills.length - 20} more
                    </div>
                  )}
                </div>
              </div>
            )}

            {/* Error Nodes */}
            {filteredErrors.length > 0 && (
              <div className={styles.nodeGroup}>
                <div className={`${styles.nodeGroupTitle} ${styles.errorsGroupTitle}`}>
                  <i className="fas fa-exclamation-triangle" />
                  ErrorNodes ({filteredErrors.length})
                </div>
                <div className={styles.nodeChips}>
                  {filteredErrors.slice(0, 20).map((error) => (
                    <div
                      key={error.id}
                      className={`${styles.nodeChip} ${styles.errorNodeChip}`}
                      onClick={() => handleNodeClick(error, 'error')}
                      title={error.error_type}
                    >
                      <i className={`fas fa-exclamation-triangle ${styles.nodeChipIcon}`} />
                      <span className={styles.nodeChipLabel}>
                        {error.error_type.length > 20
                          ? error.error_type.slice(0, 20) + '…'
                          : error.error_type}
                      </span>
                    </div>
                  ))}
                  {filteredErrors.length > 20 && (
                    <div className={styles.nodeChip} style={{ opacity: 0.6 }}>
                      +{filteredErrors.length - 20} more
                    </div>
                  )}
                </div>
              </div>
            )}

            {/* Connection Panel */}
            {selectedNode && (
              <div className={styles.connectionPanel}>
                <div className={styles.connectionPanelTitle}>
                  <i className="fas fa-link" />
                  {selectedNode.node_type === 'skill' ? 'SkillNode' : 'ErrorNode'} Connections
                </div>
                <div className={styles.connectionPanelRow}>
                  <span className={styles.connectionLabel}>Selected Node</span>
                  <span className={styles.connectionValue}>
                    {selectedNode.node_type === 'skill'
                      ? selectedNode.skill_type
                      : selectedNode.error_type}
                  </span>
                </div>
                <div className={styles.connectionPanelRow}>
                  <span className={styles.connectionLabel}>Node ID</span>
                  <span className={styles.connectionValue}>{selectedNode.id}</span>
                </div>
                <div className={styles.connectionPanelRow}>
                  <span className={styles.connectionLabel}>Connections Found</span>
                  <span className={styles.connectionValue}>{connectedNodes.length}</span>
                </div>
                {connectedNodes.length > 0 && (
                  <div style={{ marginTop: '0.75rem' }}>
                    <div className={styles.sectionTitle}>
                      Related {selectedNode.node_type === 'skill' ? 'ErrorNodes' : 'SkillNodes'}
                    </div>
                    <div className={styles.nodeChips}>
                      {connectedNodes.map((node) => (
                        <div
                          key={node.id}
                          className={`${styles.nodeChip} ${
                            selectedNode.node_type === 'skill' ? styles.errorNodeChip : styles.skillNodeChip
                          }`}
                          onClick={() =>
                            handleNodeClick(
                              node,
                              selectedNode.node_type === 'skill' ? 'error' : 'skill'
                            )
                          }
                        >
                          <i
                            className={`fas ${
                              selectedNode.node_type === 'skill'
                                ? 'fa-exclamation-triangle'
                                : 'fa-brain'
                            } ${styles.nodeChipIcon}`}
                          />
                          <span className={styles.nodeChipLabel}>
                            {node.skill_type || node.error_type}
                          </span>
                        </div>
                      ))}
                    </div>
                  </div>
                )}
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
}

// ===== MAIN COMPONENT =====
export default function GraphExplorer() {
  const { activePage } = useNavigation('graph-explorer');
  const [activeTab, setActiveTab] = useState('overview');
  const [overviewStats, setOverviewStats] = useState({ height: 0, skills: 0, errors: 0, vectors: 0 });

  // Fetch overview stats once
  useEffect(() => {
    let cancelled = false;
    const fetchStats = async () => {
      try {
        const [height, skills, errors, vectors] = await Promise.all([
          graphChainApi.getHeight().catch(() => 0),
          graphChainApi.getAllSkills().catch(() => []),
          graphChainApi.getAllErrors().catch(() => []),
          graphChainApi.getAllVectors().catch(() => []),
        ]);
        if (!cancelled) {
          setOverviewStats({
            height: typeof height === 'number' ? height : (height?.height ?? 0),
            skills: Array.isArray(skills) ? skills.length : 0,
            errors: Array.isArray(errors) ? errors.length : 0,
            vectors: Array.isArray(vectors) ? vectors.length : 0,
          });
        }
      } catch (err) {
        console.error('Failed to fetch overview stats:', err);
      }
    };
    fetchStats();
    return () => { cancelled = true; };
  }, []);

  const handleSearch = useCallback((query) => {
    console.log('Graph Explorer search:', query);
    // Could implement search across graph nodes here
  }, []);

  const renderTab = () => {
    switch (activeTab) {
      case 'overview':
        return <OverviewTab stats={overviewStats} onTabSwitch={setActiveTab} />;
      case 'errors':
        return <ErrorNodesTab />;
      case 'vectors':
        return <VectorsTab />;
      case 'visualization':
        return <GraphVisualizationTab />;
      default:
        return <OverviewTab stats={overviewStats} onTabSwitch={setActiveTab} />;
    }
  };

  return (
    <PageLayout activePage={activePage} pageTitle="Graph Explorer" onSearch={handleSearch}>
      <PageHeader
        title="KNIRV Graph Explorer"
        subtitle="Explore the GraphChain network topology, error nodes, vectors, and node relationships"
        titleColor="#60a5fa"
      />

      {/* Tab Navigation */}
      <div className={styles.tabNav}>
        {TABS.map((tab) => (
          <button
            key={tab.id}
            className={`${styles.tabButton} ${activeTab === tab.id ? styles.tabButtonActive : ''}`}
            onClick={() => setActiveTab(tab.id)}
          >
            <i className={`fas ${tab.icon}`} />
            <span>{tab.label}</span>
          </button>
        ))}
      </div>

      {/* Tab Content */}
      <div className={styles.tabContent}>
        {renderTab()}
      </div>
    </PageLayout>
  );
}
