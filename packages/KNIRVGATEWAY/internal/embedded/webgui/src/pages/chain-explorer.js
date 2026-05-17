import React, { useEffect, useState, useCallback } from 'react';
import { useNavigation } from '../hooks/useNavigation';
import PageLayout from '../components/PageLayout';
import PageHeader from '../components/PageHeader';
import GlassyCard from '../components/GlassyCard';
import StatsCard from '../components/GraphChain/StatsCard';
import SkillNodeCard from '../components/GraphChain/SkillNodeCard';
import LoadingSpinner from '../components/GraphChain/LoadingSpinner';
import { graphChainApi } from '../services/graphchain-api';
import styles from './chain-explorer.module.css';

// ─── Tab Definitions ────────────────────────────────────────────────────────
const TABS = [
  { key: 'dashboard',    label: 'Dashboard',    icon: 'fa-chart-pie' },
  { key: 'skillnodes',   label: 'SkillNodes',    icon: 'fa-brain' },
  { key: 'capabilities', label: 'Capabilities',  icon: 'fa-cogs' },
  { key: 'properties',   label: 'Properties',    icon: 'fa-list-alt' },
  { key: 'search',       label: 'Search',        icon: 'fa-search' },
  { key: 'accounts',     label: 'Accounts',      icon: 'fa-users' },
];

// ─── Helpers ─────────────────────────────────────────────────────────────────
function formatTime(ts) {
  try { return new Date(ts).toLocaleString(); } catch { return ts || '—'; }
}

function truncate(str, len = 48) {
  if (!str) return '—';
  return str.length > len ? str.slice(0, len) + '…' : str;
}

// ─── Dashboard Tab ───────────────────────────────────────────────────────────
function DashboardTab({ loading, stats, currentDensity, recentSkills, onRefresh }) {
  if (loading && !stats) {
    return (
      <div className={styles.loadingContainer}>
        <LoadingSpinner size="large" text="Loading GraphChain data..." />
      </div>
    );
  }

  return (
    <div className={styles.tabContent}>
      {/* Live Indicator + Refresh */}
      <div className={styles.toolbarRow}>
        <div className={styles.liveIndicator}>
          <div className={styles.liveDot}></div>
          <span>Live</span>
        </div>
        <button onClick={onRefresh} className={styles.refreshBtn} title="Refresh">
          <i className="fas fa-sync-alt"></i> Refresh
        </button>
      </div>

      {/* Network Stats */}
      <div className={styles.statsGrid}>
        <StatsCard
          title="Network Density"
          value={(currentDensity ?? 0).toLocaleString()}
          icon="fa-network-wired"
          trend={2.3}
          color="blue"
        />
        <StatsCard
          title="SkillNodes"
          value={(stats?.totalSkillNodes ?? 0).toLocaleString()}
          icon="fa-brain"
          trend={5.1}
          color="green"
        />
        <StatsCard
          title="ErrorNodes"
          value={(stats?.totalErrorNodes ?? 0).toLocaleString()}
          icon="fa-exclamation-triangle"
          trend={3.2}
          color="orange"
        />
        <StatsCard
          title="Avg Resolution Time"
          value={`${(stats?.avgResolutionTime ?? 0).toFixed(1)}s`}
          icon="fa-clock"
          trend={-1.2}
          color="purple"
        />
      </div>

      {/* Recent SkillNodes */}
      <GlassyCard className={styles.sectionCard}>
        <div className={styles.cardHeader}>
          <div className={styles.cardTitle}>
            <i className="fas fa-brain"></i>
            <h2>Recent SkillNodes</h2>
          </div>
        </div>
        {recentSkills.length > 0 ? (
          <div className={styles.skillsList}>
            {recentSkills.map((skill) => (
              <SkillNodeCard key={skill.id} skill={skill} />
            ))}
          </div>
        ) : (
          <div className={styles.emptyState}>
            <i className="fas fa-brain"></i>
            <p>No SkillNodes found</p>
          </div>
        )}
      </GlassyCard>

      {/* Chain Statistics */}
      <GlassyCard className={styles.sectionCard}>
        <div className={styles.cardHeader}>
          <div className={styles.cardTitle}>
            <i className="fas fa-chart-line"></i>
            <h2>Chain Statistics</h2>
          </div>
        </div>
        <div className={styles.chainStatsGrid}>
          <div className={styles.statItem}>
            <div className={styles.statLabel}>Height</div>
            <div className={styles.statValue}>{stats?.height?.toLocaleString() ?? '—'}</div>
          </div>
          <div className={styles.statItem}>
            <div className={styles.statLabel}>Total Nodes</div>
            <div className={styles.statValue}>{stats?.totalNodes?.toLocaleString() ?? '—'}</div>
          </div>
          <div className={styles.statItem}>
            <div className={styles.statLabel}>Total Edges</div>
            <div className={styles.statValue}>{stats?.totalEdges?.toLocaleString() ?? '—'}</div>
          </div>
          <div className={styles.statItem}>
            <div className={styles.statLabel}>Vectors</div>
            <div className={styles.statValue}>{stats?.totalVectors?.toLocaleString() ?? '—'}</div>
          </div>
        </div>
      </GlassyCard>

      {/* Network Health */}
      <GlassyCard className={styles.sectionCard}>
        <div className={styles.cardHeader}>
          <div className={styles.cardTitle}>
            <i className="fas fa-heartbeat"></i>
            <h2>Network Health</h2>
          </div>
        </div>
        <div className={styles.healthGrid}>
          <div className={styles.healthItem}>
            <div className={styles.healthLabel}>Chain Status</div>
            <div className={styles.healthValue}>
              <span className={`${styles.healthIndicator} ${styles.healthHealthy}`}></span>
              Healthy
            </div>
          </div>
          <div className={styles.healthItem}>
            <div className={styles.healthLabel}>Sync Status</div>
            <div className={styles.healthValue}>
              <span className={`${styles.healthIndicator} ${styles.healthHealthy}`}></span>
              Synced
            </div>
          </div>
          <div className={styles.healthItem}>
            <div className={styles.healthLabel}>Peer Count</div>
            <div className={styles.healthValue}>—</div>
          </div>
          <div className={styles.healthItem}>
            <div className={styles.healthLabel}>Uptime</div>
            <div className={styles.healthValue}>—</div>
          </div>
        </div>
      </GlassyCard>
    </div>
  );
}

// ─── SkillNodes Tab ──────────────────────────────────────────────────────────
function SkillNodesTab({ skills, loading, error }) {
  const [searchTerm, setSearchTerm] = useState('');
  const [filter, setFilter] = useState('all');
  const [page, setPage] = useState(1);
  const perPage = 10;

  const filtered = React.useMemo(() => {
    let list = skills;
    if (filter === 'validated') list = list.filter(s => s.validation?.is_validated);
    else if (filter === 'unvalidated') list = list.filter(s => !s.validation?.is_validated);
    else if (filter === 'high-performance') list = list.filter(s => s.performance?.success_rate > 0.8);
    if (searchTerm) {
      const q = searchTerm.toLowerCase();
      list = list.filter(s =>
        s.skill_type?.toLowerCase().includes(q) ||
        (s.capabilities || []).some(c => c.toLowerCase().includes(q))
      );
    }
    return list;
  }, [skills, filter, searchTerm]);

  const totalPages = Math.ceil(filtered.length / perPage);
  const paginated = filtered.slice((page - 1) * perPage, page * perPage);

  useEffect(() => { setPage(1); }, [filter, searchTerm]);

  const loadingEl = (
    <div className={styles.loadingContainer}>
      <LoadingSpinner size="large" text="Loading SkillNodes..." />
    </div>
  );

  const errorEl = (
    <GlassyCard className={styles.errorCard}>
      <div className={styles.errorIcon}><i className="fas fa-exclamation-triangle"></i></div>
      <div className={styles.errorTitle}>Error Loading SkillNodes</div>
      <div className={styles.errorMessage}>{error}</div>
    </GlassyCard>
  );

  return (
    <div className={styles.tabContent}>
      {/* Filters */}
      <GlassyCard className={styles.filterCard}>
        <div className={styles.filterRow}>
          <div className={styles.filterGroup}>
            <label className={styles.filterLabel}>Search SkillNodes</label>
            <input
              type="text"
              placeholder="Search by skill type or capabilities…"
              value={searchTerm}
              onChange={e => setSearchTerm(e.target.value)}
              className={styles.searchInput}
            />
          </div>
          <div className={styles.filterGroup}>
            <label className={styles.filterLabel}>Filter</label>
            <select value={filter} onChange={e => setFilter(e.target.value)} className={styles.filterSelect}>
              <option value="all">All SkillNodes</option>
              <option value="validated">Validated Only</option>
              <option value="unvalidated">Unvalidated Only</option>
              <option value="high-performance">High Performance</option>
            </select>
          </div>
        </div>
        <div className={styles.statsRow}>
          <div className={styles.statItem}>
            <div className={styles.statValue}>{skills.length}</div>
            <div className={styles.statLabel}>Total</div>
          </div>
          <div className={styles.statItem}>
            <div className={styles.statValue}>{filtered.length}</div>
            <div className={styles.statLabel}>Filtered</div>
          </div>
          <div className={styles.statItem}>
            <div className={styles.statValue}>{page} / {totalPages || 1}</div>
            <div className={styles.statLabel}>Page</div>
          </div>
        </div>
      </GlassyCard>

      {loading ? loadingEl : error ? errorEl : paginated.length === 0 ? (
        <GlassyCard className={styles.emptyCard}>
          <i className="fas fa-brain"></i>
          <h3>No SkillNodes Found</h3>
          <p>Try adjusting your search or filter criteria.</p>
        </GlassyCard>
      ) : (
        <>
          <div className={styles.skillsList}>
            {paginated.map(skill => <SkillNodeCard key={skill.id} skill={skill} />)}
          </div>
          {totalPages > 1 && (
            <div className={styles.pagination}>
              <button disabled={page === 1} onClick={() => setPage(p => p - 1)} className={styles.paginationButton}>
                <i className="fas fa-chevron-left"></i> Prev
              </button>
              <span className={styles.pageInfo}>{page} / {totalPages}</span>
              <button disabled={page === totalPages} onClick={() => setPage(p => p + 1)} className={styles.paginationButton}>
                Next <i className="fas fa-chevron-right"></i>
              </button>
            </div>
          )}
        </>
      )}
    </div>
  );
}

// ─── Capabilities Tab ────────────────────────────────────────────────────────
function CapabilitiesTab({ skills }) {
  const [searchTerm, setSearchTerm] = useState('');

  const capMap = React.useMemo(() => {
    const map = {};
    for (const s of skills) {
      for (const cap of (s.capabilities || [])) {
        if (!map[cap]) map[cap] = [];
        map[cap].push(s);
      }
    }
    return map;
  }, [skills]);

  const caps = React.useMemo(() => {
    let list = Object.keys(capMap).sort();
    if (searchTerm) {
      const q = searchTerm.toLowerCase();
      list = list.filter(c => c.toLowerCase().includes(q));
    }
    return list;
  }, [capMap, searchTerm]);

  if (skills.length === 0) {
    return (
      <div className={styles.tabContent}>
        <GlassyCard className={styles.emptyCard}>
          <i className="fas fa-cogs"></i>
          <h3>No Capabilities Data</h3>
          <p>Load skills data first.</p>
        </GlassyCard>
      </div>
    );
  }

  return (
    <div className={styles.tabContent}>
      <GlassyCard className={styles.filterCard}>
        <div className={styles.filterRow}>
          <div className={styles.filterGroup}>
            <label className={styles.filterLabel}>Search Capabilities</label>
            <input
              type="text"
              placeholder="Filter capabilities…"
              value={searchTerm}
              onChange={e => setSearchTerm(e.target.value)}
              className={styles.searchInput}
            />
          </div>
        </div>
        <div className={styles.statsRow}>
          <div className={styles.statItem}>
            <div className={styles.statValue}>{Object.keys(capMap).length}</div>
            <div className={styles.statLabel}>Unique Capabilities</div>
          </div>
          <div className={styles.statItem}>
            <div className={styles.statValue}>{skills.length}</div>
            <div className={styles.statLabel}>Total SkillNodes</div>
          </div>
        </div>
      </GlassyCard>

      <div className={styles.capGrid}>
        {caps.map(cap => (
          <GlassyCard key={cap} className={styles.capCard}>
            <div className={styles.capHeader}>
              <i className="fas fa-cog"></i>
              <span className={styles.capName}>{cap}</span>
              <span className={styles.capCount}>{capMap[cap].length}</span>
            </div>
            <div className={styles.capSkills}>
              {capMap[cap].slice(0, 5).map(s => (
                <span key={s.id} className={styles.capSkillBadge} title={s.id}>
                  {s.skill_type || truncate(s.id, 24)}
                </span>
              ))}
              {capMap[cap].length > 5 && (
                <span className={styles.capMore}>+{capMap[cap].length - 5} more</span>
              )}
            </div>
          </GlassyCard>
        ))}
      </div>
    </div>
  );
}

// ─── Properties Tab ──────────────────────────────────────────────────────────
function PropertiesTab({ chainProps, propsLoading, propsError }) {
  if (propsLoading) {
    return (
      <div className={styles.tabContent}>
        <div className={styles.loadingContainer}>
          <LoadingSpinner size="large" text="Loading chain properties…" />
        </div>
      </div>
    );
  }

  if (propsError) {
    return (
      <div className={styles.tabContent}>
        <GlassyCard className={styles.errorCard}>
          <div className={styles.errorIcon}><i className="fas fa-exclamation-triangle"></i></div>
          <div className={styles.errorTitle}>Properties Error</div>
          <div className={styles.errorMessage}>{propsError}</div>
        </GlassyCard>
      </div>
    );
  }

  const entries = chainProps && typeof chainProps === 'object'
    ? Object.entries(chainProps)
    : [['raw', String(chainProps)]];

  return (
    <div className={styles.tabContent}>
      <GlassyCard className={styles.sectionCard}>
        <div className={styles.cardHeader}>
          <div className={styles.cardTitle}>
            <i className="fas fa-list-alt"></i>
            <h2>Root-Chain Connection Properties</h2>
          </div>
        </div>
        <div className={styles.propTable}>
          <div className={styles.propTableHeader}>
            <span className={styles.propColKey}>Property</span>
            <span className={styles.propColVal}>Value</span>
          </div>
          {entries.map(([key, val]) => (
            <div key={key} className={styles.propRow}>
              <span className={styles.propColKey}>{key}</span>
              <span className={styles.propColVal}>
                {typeof val === 'object' ? JSON.stringify(val, null, 2) : String(val)}
              </span>
            </div>
          ))}
        </div>
      </GlassyCard>
    </div>
  );
}

// ─── Search Tab ──────────────────────────────────────────────────────────────
function SearchTab({ skills, errors }) {
  const [query, setQuery] = useState('');
  const [results, setResults] = useState({ skills: [], errors: [], skillCount: 0, errorCount: 0 });
  const [searched, setSearched] = useState(false);

  const handleSearch = useCallback(() => {
    if (!query.trim()) { setSearched(false); setResults({ skills: [], errors: [], skillCount: 0, errorCount: 0 }); return; }
    const q = query.toLowerCase();
    const matchedSkills = skills.filter(s =>
      s.skill_type?.toLowerCase().includes(q) ||
      s.id?.toLowerCase().includes(q) ||
      (s.capabilities || []).some(c => c.toLowerCase().includes(q))
    );
    const matchedErrors = errors.filter(e =>
      e.error_type?.toLowerCase().includes(q) ||
      e.id?.toLowerCase().includes(q) ||
      (e.description || '').toLowerCase().includes(q)
    );
    setResults({
      skills: matchedSkills,
      errors: matchedErrors,
      skillCount: matchedSkills.length,
      errorCount: matchedErrors.length,
    });
    setSearched(true);
  }, [query, skills, errors]);

  return (
    <div className={styles.tabContent}>
      <GlassyCard className={styles.filterCard}>
        <div className={styles.searchBarRow}>
          <input
            type="text"
            placeholder="Search SkillNodes and ErrorNodes…"
            value={query}
            onChange={e => setQuery(e.target.value)}
            onKeyDown={e => e.key === 'Enter' && handleSearch()}
            className={styles.searchInput}
          />
          <button onClick={handleSearch} className={styles.searchBtn}>
            <i className="fas fa-search"></i> Search
          </button>
        </div>
      </GlassyCard>

      {searched && (results.skillCount + results.errorCount) === 0 && (
        <GlassyCard className={styles.emptyCard}>
          <i className="fas fa-search"></i>
          <h3>No Results</h3>
          <p>No nodes match your query.</p>
        </GlassyCard>
      )}

      {searched && results.skillCount > 0 && (
        <GlassyCard className={styles.sectionCard}>
          <div className={styles.cardHeader}>
            <div className={styles.cardTitle}>
              <i className="fas fa-brain"></i>
              <h2>SkillNodes ({results.skillCount})</h2>
            </div>
          </div>
          <div className={styles.skillsList}>
            {results.skills.slice(0, 10).map(s => <SkillNodeCard key={s.id} skill={s} />)}
          </div>
        </GlassyCard>
      )}

      {searched && results.errorCount > 0 && (
        <GlassyCard className={styles.sectionCard}>
          <div className={styles.cardHeader}>
            <div className={styles.cardTitle}>
              <i className="fas fa-exclamation-triangle"></i>
              <h2>ErrorNodes ({results.errorCount})</h2>
            </div>
          </div>
          <div className={styles.errorResultsList}>
            {results.errors.slice(0, 10).map(e => (
              <div key={e.id} className={styles.errorResultItem}>
                <div className={styles.errorResultHeader}>
                  <span className={styles.errorResultType}>{e.error_type}</span>
                  <span className={styles.errorResultId}>{truncate(e.id, 36)}</span>
                  {e.severity !== undefined && (
                    <span className={`${styles.severityBadge} ${e.severity > 5 ? styles.severityHigh : styles.severityLow}`}>
                      {e.severity}
                    </span>
                  )}
                </div>
                {e.description && <div className={styles.errorResultDesc}>{e.description}</div>}
                <div className={styles.errorResultTime}>{formatTime(e.timestamp)}</div>
              </div>
            ))}
          </div>
        </GlassyCard>
      )}
    </div>
  );
}

// ─── Accounts Tab ────────────────────────────────────────────────────────────
function AccountsTab({ accounts, loading, error }) {
  if (loading) {
    return (
      <div className={styles.tabContent}>
        <div className={styles.loadingContainer}>
          <LoadingSpinner size="large" text="Loading accounts…" />
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className={styles.tabContent}>
        <GlassyCard className={styles.errorCard}>
          <div className={styles.errorIcon}><i className="fas fa-exclamation-triangle"></i></div>
          <div className={styles.errorTitle}>Accounts Error</div>
          <div className={styles.errorMessage}>{error}</div>
        </GlassyCard>
      </div>
    );
  }

  const accountList = Array.isArray(accounts) ? accounts : [];

  if (accountList.length === 0) {
    return (
      <div className={styles.tabContent}>
        <GlassyCard className={styles.emptyCard}>
          <i className="fas fa-users"></i>
          <h3>No Accounts Found</h3>
          <p>No chain accounts or peers are currently available.</p>
        </GlassyCard>
      </div>
    );
  }

  return (
    <div className={styles.tabContent}>
      <GlassyCard className={styles.sectionCard}>
        <div className={styles.cardHeader}>
          <div className={styles.cardTitle}>
            <i className="fas fa-users"></i>
            <h2>Chain Accounts / Peers ({accountList.length})</h2>
          </div>
        </div>
        <div className={styles.accountGrid}>
          {accountList.map((acct, i) => (
            <div key={acct.id || i} className={styles.accountCard}>
              <div className={styles.accountIcon}><i className="fas fa-user"></i></div>
              <div className={styles.accountInfo}>
                <div className={styles.accountName}>{acct.name || acct.id || `Peer ${i + 1}`}</div>
                <div className={styles.accountId}>{truncate(acct.id || '', 48)}</div>
                {acct.role && <div className={styles.accountRole}>{acct.role}</div>}
                {acct.status && (
                  <div className={styles.accountStatus}>
                    <span className={`${styles.statusDot} ${acct.status === 'active' ? styles.statusActive : styles.statusInactive}`}></span>
                    {acct.status}
                  </div>
                )}
              </div>
            </div>
          ))}
        </div>
      </GlassyCard>
    </div>
  );
}

// ─── Main Page Component ─────────────────────────────────────────────────────
export default function ChainExplorer() {
  const { activePage } = useNavigation('chain-explorer');
  const [activeTab, setActiveTab] = useState('dashboard');

  // Dashboard data
  const [currentDensity, setCurrentDensity] = useState(0);
  const [stats, setStats] = useState(null);
  const [recentSkills, setRecentSkills] = useState([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState(null);

  // All skills / errors
  const [allSkills, setAllSkills] = useState([]);
  const [allErrors, setAllErrors] = useState([]);

  // Properties data
  const [chainProps, setChainProps] = useState(null);
  const [propsLoading, setPropsLoading] = useState(false);
  const [propsError, setPropsError] = useState(null);

  // Accounts data
  const [accounts, setAccounts] = useState([]);
  const [acctsLoading, setAcctsLoading] = useState(false);
  const [acctsError, setAcctsError] = useState(null);

  // ── Data fetch helpers ─────────────────────────────────────────────────
  const fetchDashboard = useCallback(async () => {
    setError(null);
    try {
      // getCurrentDensity is not defined in the API; fallback to getHeight
      let density = 0;
      try {
        density = await graphChainApi.getHeight();
      } catch {
        density = 0;
      }

      const [statsData, skillsData] = await Promise.all([
        graphChainApi.getGraphChainStats(),
        graphChainApi.getRecentSkills(5),
      ]);
      setCurrentDensity(density);
      setStats(statsData);
      setRecentSkills(skillsData);
    } catch (err) {
      setError(err.message || 'Failed to fetch GraphChain data');
    } finally {
      setIsLoading(false);
    }
  }, []);

  const fetchAllSkills = useCallback(async () => {
    try {
      const data = await graphChainApi.getAllSkills();
      setAllSkills(data);
    } catch { /* skills already loaded best-effort from dashboard */ }
  }, []);

  const fetchAllErrors = useCallback(async () => {
    try {
      const data = await graphChainApi.getAllErrors();
      setAllErrors(data);
    } catch { /* non-critical */ }
  }, []);

  // ── Lazy-load data per tab ─────────────────────────────────────────────
  useEffect(() => {
    setIsLoading(true);
    fetchDashboard();
    fetchAllSkills();
  }, [fetchDashboard, fetchAllSkills]);

  // Load errors when Search tab becomes active
  useEffect(() => {
    if (activeTab === 'search' && allErrors.length === 0) {
      fetchAllErrors();
    }
  }, [activeTab, allErrors.length, fetchAllErrors]);

  // Load properties when Properties tab becomes active
  useEffect(() => {
    if (activeTab === 'properties' && chainProps === null) {
      setPropsLoading(true);
      setPropsError(null);
      (async () => {
        try {
          const [height, heads] = await Promise.all([
            graphChainApi.getHeight().catch(() => '—'),
            graphChainApi.getGraphHeads().catch(() => []),
          ]);
          setChainProps({
            height,
            heads: Array.isArray(heads) ? heads.slice(0, 5) : heads,
            chainEndpoint: '/api/chain/',
            graphEndpoint: '/api/graph/',
          });
        } catch (err) {
          setPropsError(err.message || 'Failed to load chain properties');
        } finally {
          setPropsLoading(false);
        }
      })();
    }
  }, [activeTab, chainProps]);

  // Load accounts when Accounts tab becomes active
  useEffect(() => {
    if (activeTab === 'accounts' && accounts.length === 0) {
      setAcctsLoading(true);
      setAcctsError(null);
      (async () => {
        try {
          const data = await graphChainApi.getGraphHeads().catch(() => []);
          // Format heads as peer-like accounts
          if (Array.isArray(data)) {
            setAccounts(data.map((id, i) => ({
              id,
              name: `Head ${i + 1}`,
              role: 'graph-head',
              status: 'active',
            })));
          } else {
            setAccounts([]);
          }
        } catch (err) {
          setAcctsError(err.message || 'Failed to load accounts');
        } finally {
          setAcctsLoading(false);
        }
      })();
    }
  }, [activeTab, accounts.length]);

  const handleRefresh = useCallback(async () => {
    setIsLoading(true);
    await fetchDashboard();
  }, [fetchDashboard]);

  const handleSearch = useCallback((query) => {
    setActiveTab('search');
    // The search tab will pick up the query via its own state
    // We don't have a mechanism to inject from PageLayout search bar into child
    // but the SearchTab has its own input so that's fine
  }, []);

  // ── Render current tab ─────────────────────────────────────────────────
  const renderTab = () => {
    switch (activeTab) {
      case 'dashboard':
        return (
          <DashboardTab
            loading={isLoading}
            stats={stats}
            currentDensity={currentDensity}
            recentSkills={recentSkills}
            onRefresh={handleRefresh}
          />
        );
      case 'skillnodes':
        return (
          <SkillNodesTab
            skills={allSkills}
            loading={isLoading && allSkills.length === 0}
            error={isLoading && allSkills.length === 0 ? error : null}
          />
        );
      case 'capabilities':
        return <CapabilitiesTab skills={allSkills} />;
      case 'properties':
        return (
          <PropertiesTab
            chainProps={chainProps}
            propsLoading={propsLoading}
            propsError={propsError}
          />
        );
      case 'search':
        return <SearchTab skills={allSkills} errors={allErrors} />;
      case 'accounts':
        return (
          <AccountsTab
            accounts={accounts}
            loading={acctsLoading}
            error={acctsError}
          />
        );
      default:
        return null;
    }
  };

  return (
    <PageLayout activePage={activePage} pageTitle="Chain Explorer" onSearch={handleSearch}>
      <PageHeader
        title="KNIRV Chain Explorer"
        subtitle="Explore blockchain data, SkillNodes, ErrorNodes, and network activity"
        titleColor="#007bff"
      />

      {/* Tab Navigation */}
      <div className={styles.tabBar}>
        {TABS.map(tab => (
          <button
            key={tab.key}
            className={`${styles.tabButton} ${activeTab === tab.key ? styles.tabActive : ''}`}
            onClick={() => setActiveTab(tab.key)}
          >
            <i className={`fas ${tab.icon}`}></i>
            <span>{tab.label}</span>
          </button>
        ))}
      </div>

      {/* Tab Content */}
      <div className={styles.tabPanel}>
        {renderTab()}
      </div>
    </PageLayout>
  );
}
