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
import styles from './graphchain-dashboard.module.css';

export default function GraphChainDashboard() {
  const { activePage } = useNavigation('graphchain-dashboard');
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
      <PageLayout activePage={activePage} pageTitle="GraphChain Dashboard">
        <div className={styles.loadingContainer}>
          <LoadingSpinner size="large" text="Loading GraphChain data..." />
        </div>
      </PageLayout>
    );
  }

  if (error && !stats) {
    return (
      <PageLayout activePage={activePage} pageTitle="GraphChain Dashboard">
        <PageHeader
          title="GraphChain Explorer"
          subtitle="Real-time GraphChain data, SkillNodes, and ErrorNode analytics"
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
    <PageLayout activePage={activePage} pageTitle="GraphChain Dashboard">
      <div className={styles.dashboard}>
        {/* Header */}
        <div className={styles.dashboardHeader}>
          <div>
            <h1 className={styles.title}>
              <span className={styles.gradientText}>GraphChain Explorer</span>
            </h1>
            <p className={styles.subtitle}>
              Real-time GraphChain data, SkillNodes, and ErrorNode analytics
            </p>
          </div>
          <div className={styles.liveIndicator}>
            <div className={styles.liveDot}></div>
            <span>Live</span>
          </div>
        </div>

        {/* Stats Grid */}
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

        {/* Quick Actions */}
        <div className={styles.quickActions}>
          <Link href="/graphchain-skills" className={`${styles.actionCard} ${styles.actionBlue}`}>
            <i className="fas fa-brain"></i>
            <h3>Explore SkillNodes</h3>
            <p>Browse through all SkillNodes in the GraphChain</p>
            <i className="fas fa-arrow-right"></i>
          </Link>

          <Link href="/graphchain-errors" className={`${styles.actionCard} ${styles.actionOrange}`}>
            <i className="fas fa-exclamation-triangle"></i>
            <h3>View ErrorNodes</h3>
            <p>Explore ErrorNodes and their resolution paths</p>
            <i className="fas fa-arrow-right"></i>
          </Link>

          <Link href="/graphchain-visualization" className={`${styles.actionCard} ${styles.actionPurple}`}>
            <i className="fas fa-project-diagram"></i>
            <h3>Graph Visualization</h3>
            <p>Visualize SkillNode-ErrorNode relationships</p>
            <i className="fas fa-arrow-right"></i>
          </Link>
        </div>
      </div>
    </PageLayout>
  );
}
