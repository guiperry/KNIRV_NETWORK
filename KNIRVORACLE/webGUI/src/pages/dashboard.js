import React, { useState } from 'react';
import { useNavigation } from '../hooks/useNavigation';
import PageLayout from '../components/PageLayout';
import PageHeader from '../components/PageHeader';
import UnifiedDashboard from '../components/UnifiedDashboard';
import styles from './dashboard.module.css';

export default function Dashboard() {
  const { activePage } = useNavigation('dashboard');
  const [searchQuery, setSearchQuery] = useState('');
  const brightBlue = '#007bff';

  const handleSearch = (query) => {
    setSearchQuery(query);
  };

  return (
    <PageLayout activePage={activePage} pageTitle="Oracle Dashboard" onSearch={handleSearch}>
      <PageHeader
        title="KNIRV Oracle Dashboard"
        subtitle="Unified management interface for all Oracle operations"
        titleColor={brightBlue}
      />

      <UnifiedDashboard />
    </PageLayout>
  );
}