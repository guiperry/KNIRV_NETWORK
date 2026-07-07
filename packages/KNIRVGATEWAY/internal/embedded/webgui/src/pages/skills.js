import React from 'react';
import PageLayout from '../components/PageLayout';
import PageHeader from '../components/PageHeader';

export default function Skills() {
  return (
    <PageLayout activePage={'skills'} pageTitle="Skills" onSearch={() => {}}>
      <PageHeader title="Skills" subtitle="Discover and manage skills in the marketplace" />
      <div style={{ padding: 16, color: '#e6efff' }}>
        <p>Coming soon. Use Marketplace for listings.</p>
      </div>
    </PageLayout>
  );
}