import React, { useState, useEffect } from 'react';
import OnboardingFlow from '../components/OnboardingFlowUpdated';
import { useNavigation } from '../hooks/useNavigation';
import PageLayout from '../components/PageLayout';
import PageHeader from '../components/PageHeader';
import GlassyCard from '../components/GlassyCard';
import styles from './index.module.css';

export default function Home({ onboardingCompleted }) {
  const { activePage, handleNavigation } = useNavigation('home');
  const [showOnboarding, setShowOnboarding] = useState(false);
  const [isBlockchainEnabled, setIsBlockchainEnabled] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');

  useEffect(() => {
    const hasCompletedOnboarding = localStorage.getItem('onboardingCompleted');
    if (hasCompletedOnboarding === 'true' || onboardingCompleted) {
      setIsBlockchainEnabled(true);
    }
  }, [onboardingCompleted]);

  const handleOnboardingComplete = () => {
    setShowOnboarding(false);
    setIsBlockchainEnabled(true);
    localStorage.setItem('onboardingCompleted', 'true');
  };

  const handleSearch = (query) => {
    setSearchQuery(query);
  };

  const cryptoData = [
    { name: 'Bitcoin', price: '$45,632', change: '+2.3%' },
    { name: 'Ethereum', price: '$3,245', change: '-1.1%' },
    { name: 'Solana', price: '$97.45', change: '+0.8%' },
    { name: 'Cardano', price: '$0.53', change: '+0.5%' },
  ];

  const filteredCryptoData = cryptoData.filter((crypto) =>
    crypto.name.toLowerCase().includes(searchQuery.toLowerCase())
  );

  return (
    <PageLayout
      activePage={activePage}
      pageTitle="Blockchain Dashboard"
      onSearch={handleSearch}
    >
      {showOnboarding && <OnboardingFlow onComplete={handleOnboardingComplete} />}

      <PageHeader
        title="Blockchain Overview"
        subtitle="Welcome to your blockchain dashboard"
      />

      {/* Welcome Banner - Onboarding disabled */}
      <GlassyCard darker className={styles.welcomeBanner}>
        <div className={styles.iconCircle}>🏠</div>
        <h2 className={styles.welcomeTitle}>KNIRV Network Dashboard</h2>
        <p className={styles.welcomeText}>
          Welcome to your KNIRV Network dashboard. Monitor network activity, manage your models, and access all KNIRV services.
        </p>
      </GlassyCard>

      {/* Cards Row */}
      <div className={styles.cryptoCardsContainer}>
        {filteredCryptoData.map((crypto, index) => (
          <GlassyCard key={index} className={styles.cryptoCard}>
            <h3 className={styles.cryptoName}>{crypto.name}</h3>
            <h2 className={styles.cryptoPrice}>{crypto.price}</h2>
            <span className={crypto.change.startsWith('+') ? styles.positiveChange : styles.negativeChange}>
              {crypto.change}
            </span>
          </GlassyCard>
        ))}
      </div>

      {/* Analytics Card */}
      <GlassyCard darker className={styles.analyticsCard}>
        <h3 className={styles.analyticsTitle}>Market Overview</h3>

        <div className={styles.chartPlaceholder}>
          <span>Price Chart Placeholder</span>
        </div>

        <div className={styles.statsContainer}>
          <GlassyCard className={styles.statItem}>
            <div className={styles.analyticsTitle}>Market Cap</div>
            <div className={styles.statValue}>$1.89T</div>
            <span className={styles.positiveChange}>+1.2% (24h)</span>
          </GlassyCard>

          <GlassyCard className={styles.statItem}>
            <div className={styles.analyticsTitle}>24h Volume</div>
            <div className={styles.statValue}>$78.5B</div>
            <span className={styles.negativeChange}>-3.4% (24h)</span>
          </GlassyCard>
        </div>
      </GlassyCard>
    </PageLayout>
  );
}
