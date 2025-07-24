import React, { useState } from 'react';
import { useNavigation } from '../hooks/useNavigation';
import PageLayout from '../components/PageLayout';
import PageHeader from '../components/PageHeader';
import GlassyCard from '../components/GlassyCard';
import styles from './dashboard.module.css';

export default function Dashboard() {
  const { activePage } = useNavigation('dashboard');
  const [searchQuery, setSearchQuery] = useState('');
  const brightBlue = '#007bff';

  const handleSearch = (query) => {
    setSearchQuery(query);
  };

  return (
    <PageLayout activePage={activePage} pageTitle="Dashboard" onSearch={handleSearch}>
      <PageHeader 
        title="Blockchain Dashboard"
        subtitle="Welcome to the blockchain dashboard"
        titleColor={brightBlue}
      />

      <div className={styles.gridContainer}>
        <GlassyCard className={styles.dashboardCard}>
          <h3 className={styles.cardTitle}>Dashboard Card</h3>
          <p>This is a simple card component using our standardized UI.</p>
        </GlassyCard>

        <GlassyCard className={styles.dashboardCard}>
          <h3 className={styles.cardTitle}>Analytics</h3>
          <p>This card contains analytics information.</p>
        </GlassyCard>
      </div>
    </PageLayout>
  );
}