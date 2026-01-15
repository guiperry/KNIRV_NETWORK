import React, { useEffect, useState } from 'react';
import { useNavigation } from '../hooks/useNavigation';
import PageLayout from '../components/PageLayout';
import PageHeader from '../components/PageHeader';
import GlassyCard from '../components/GlassyCard';
import StatsCard from '../components/GraphChain/StatsCard';
import SkillNodeCard from '../components/GraphChain/SkillNodeCard';
import LoadingSpinner from '../components/GraphChain/LoadingSpinner';
import { graphChainApi } from '../services/graphchain-api';
import Link from 'next/link';
import styles from './chain-explorer-new.module.css';

export default function ChainExplorer() {
  const { activePage } = useNavigation('chain-explorer');
  const [currentDensity, setCurrentDensity] = useState(0);
  const [stats, setStats] = useState(null);
  const [recentSkills, setRecentSkills] = useState([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState(null);

  // Auto-refresh data every 10 seconds
  useEffect(() => {
    const fetchData = async () => {
      try {
        setError(null);
        const [density, statsData, skillsData] = await Promise.all([
          graphChainApi.getCurrentDensity(),
          graphChainApi.getGraphChainStats(),
          graphChainApi.getRecentSkills(5),
        ]);

        setCurrentDensity(density);
        setStats(statsData);
        setRecentSkills(skillsData);
      } catch (err) {
        setError(err.message || 'Failed to fetch GraphChain data');
        console.error('GraphChain Dashboard error:', err);
      } finally {
        setIsLoading(false);
      }
    };

    fetchData();
    const interval = setInterval(fetchData, 10000);
    return () => clearInterval(interval);
  }, []);

  const handleRefresh = async () => {
    setIsLoading(true);
    try {
      const [density, statsData, skillsData] = await Promise.all([
        graphChainApi.getCurrentDensity(),
        graphChainApi.getGraphChainStats(),
        graphChainApi.getRecentSkills(5),
      ]);

      setCurrentDensity(density);
      setStats(statsData);
      setRecentSkills(skillsData);
      setError(null);
    } catch (err) {
      setError(err.message || 'Failed to refresh data');
    } finally {
      setIsLoading(false);
    }
  };

  if (isLoading && !stats) {
    return (
      <PageLayout activePage={activePage} pageTitle="Chain Explorer">
        <div className={styles.loadingContainer}>
          <LoadingSpinner size="large" text="Loading GraphChain data..." />
        </div>
      </PageLayout>
    );
  }

  if (error && !stats) {
    return (
      <PageLayout activePage={activePage} pageTitle="Chain Explorer">
        <PageHeader
          title="KNIRV Chain Explorer"
          subtitle="Explore blockchain data, SkillNodes, ErrorNodes, and network activity"
          titleColor="#007bff"
        />
        <GlassyCard className={styles.errorCard}>
          <div className={styles.errorIcon}>
            <i className="fas fa-exclamation-triangle"></i>
          </div>
          <div className={styles.errorTitle}>Connection Error</div>
          <div className={styles.errorMessage}>{error}</div>
          <button onClick={handleRefresh} className={styles.retryButton}>
            <i className="fas fa-sync-alt"></i>
            <span>Retry</span>
          </button>
        </GlassyCard>
      </PageLayout>
    );
  }

  return (
    <PageLayout activePage={activePage} pageTitle="Chain Explorer">
      <PageHeader
        title="KNIRV Chain Explorer"
        subtitle="Explore blockchain data, SkillNodes, ErrorNodes, and network activity"
        titleColor="#007bff"
      />

      <div className={styles.explorer}>
        {/* Live Indicator */}
        <div className={styles.liveIndicator}>
          <div className={styles.liveDot}></div>
          <span>Live</span>
        </div>

        {/* Network Stats */}
        <div className={styles.statsGrid}>
          <StatsCard
            title="Network Density"
            value={currentDensity.toLocaleString()}
            icon="fa-network-wired"
            trend={2.3}
            color="blue"
          />
          <StatsCard
            title="SkillNodes"
            value={stats?.totalSkillNodes.toLocaleString() || '0'}
            icon="fa-brain"
            trend={5.1}
            color="green"
          />
          <StatsCard
            title="ErrorNodes"
            value={stats?.totalErrorNodes.toLocaleString() || '0'}
            icon="fa-exclamation-triangle"
            trend={3.2}
            color="orange"
          />
          <StatsCard
            title="Avg Resolution Time"
            value={`${stats?.avgResolutionTime.toFixed(1) || '0'}s`}
            icon="fa-clock"
            trend={-1.2}
            color="purple"
          />
        </div>

        {/* Recent SkillNodes */}
        <GlassyCard className={styles.recentSkillsCard}>
          <div className={styles.cardHeader}>
            <div className={styles.cardTitle}>
              <i className="fas fa-brain"></i>
              <h2>Recent SkillNodes</h2>
            </div>
            <Link href="/graphchain-skills" className={styles.viewAllLink}>
              <span>View All</span>
              <i className="fas fa-arrow-right"></i>
            </Link>
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

        {/* Quick Navigation */}
        <div className={styles.quickNav}>
          <h3 className={styles.quickNavTitle}>
            <i className="fas fa-compass"></i>
            Quick Navigation
          </h3>
          <div className={styles.quickNavGrid}>
            <Link href="/graphchain-skills" className={`${styles.navCard} ${styles.navBlue}`}>
              <i className="fas fa-brain"></i>
              <h4>SkillNodes</h4>
              <p>Explore skill execution nodes on the chain</p>
              <i className="fas fa-arrow-right"></i>
            </Link>

            <Link href="/graphchain-errors" className={`${styles.navCard} ${styles.navOrange}`}>
              <i className="fas fa-exclamation-triangle"></i>
              <h4>ErrorNodes</h4>
              <p>Monitor error nodes and chain issues</p>
              <i className="fas fa-arrow-right"></i>
            </Link>

            <Link href="/error-explorer" className={`${styles.navCard} ${styles.navRed}`}>
              <i className="fas fa-bug"></i>
              <h4>Error Explorer</h4>
              <p>Deep dive into error analysis and patterns</p>
              <i className="fas fa-arrow-right"></i>
            </Link>

            <Link href="/graph-explorer" className={`${styles.navCard} ${styles.navPurple}`}>
              <i className="fas fa-project-diagram"></i>
              <h4>Graph Explorer</h4>
              <p>Visualize chain data relationships</p>
              <i className="fas fa-arrow-right"></i>
            </Link>
          </div>
        </div>

        {/* Chain Statistics */}
        <GlassyCard className={styles.chainStatsCard}>
          <div className={styles.cardHeader}>
            <div className={styles.cardTitle}>
              <i className="fas fa-chart-line"></i>
              <h2>Chain Statistics</h2>
            </div>
          </div>
          <div className={styles.chainStatsGrid}>
            <div className={styles.statItem}>
              <div className={styles.statLabel}>Latest Block</div>
              <div className={styles.statValue}>12,345,678</div>
            </div>
            <div className={styles.statItem}>
              <div className={styles.statLabel}>Total Transactions</div>
              <div className={styles.statValue}>45,678,901</div>
            </div>
            <div className={styles.statItem}>
              <div className={styles.statLabel}>Network Hash Rate</div>
              <div className={styles.statValue}>1.2 TH/s</div>
            </div>
            <div className={styles.statItem}>
              <div className={styles.statLabel}>Active Validators</div>
              <div className={styles.statValue}>156</div>
            </div>
            <div className={styles.statItem}>
              <div className={styles.statLabel}>Avg Block Time</div>
              <div className={styles.statValue}>15.2s</div>
            </div>
            <div className={styles.statItem}>
              <div className={styles.statLabel}>Gas Price</div>
              <div className={styles.statValue}>20 Gwei</div>
            </div>
          </div>
        </GlassyCard>

        {/* Network Health */}
        <GlassyCard className={styles.healthCard}>
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
              <div className={styles.healthValue}>47</div>
            </div>
            <div className={styles.healthItem}>
              <div className={styles.healthLabel}>Uptime</div>
              <div className={styles.healthValue}>99.8%</div>
            </div>
          </div>
        </GlassyCard>
      </div>
    </PageLayout>
  );
}
