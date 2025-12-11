import React, { useState } from 'react';
import { useNavigation } from '../hooks/useNavigation';
import PageLayout from '../components/PageLayout';
import PageHeader from '../components/PageHeader';
import GlassyCard from '../components/GlassyCard';
import styles from './daos.module.css';

export default function DAOs() {
  const { activePage, handleNavigation } = useNavigation('daos');
  const [searchQuery, setSearchQuery] = useState('');

  const handleSearch = (query) => {
    setSearchQuery(query);
  };

  return (
    <PageLayout activePage={activePage} pageTitle="DAOs" onSearch={handleSearch}>
      <PageHeader 
        title="Decentralized Autonomous Organizations"
        subtitle="Welcome to the DAOs section. Explore our various DAO offerings below:"
      />

      <div className={styles.gridContainer}>
        <GlassyCard 
          className={styles.daoCard}
          onClick={() => handleNavigation('daos/realty')}
        >
          <div className={styles.iconCircle}>🏢</div>
          <h3 className={styles.cardTitle}>Realty DAO</h3>
          <p className={styles.cardText}>
            Invest in real estate properties through our decentralized governance model.
            Vote on property acquisitions and development projects.
          </p>
          <button className={`${styles.button} ${styles.primary}`}>Explore Realty DAO</button>
        </GlassyCard>

        <GlassyCard 
          className={styles.daoCard}
          onClick={() => handleNavigation('daos/minerals')}
        >
          <div className={styles.iconCircle}>💎</div>
          <h3 className={styles.cardTitle}>Minerals DAO</h3>
          <p className={styles.cardText}>
            Participate in mineral rights and mining operations through collective governance.
            Vote on exploration and extraction projects.
          </p>
          <button className={`${styles.button} ${styles.primary}`}>Explore Minerals DAO</button>
        </GlassyCard>

        <GlassyCard 
          className={styles.daoCard}
          onClick={() => handleNavigation('daos/business')}
        >
          <div className={styles.iconCircle}>💼</div>
          <h3 className={styles.cardTitle}>Business DAO</h3>
          <p className={styles.cardText}>
            Invest in startups and established businesses through decentralized governance.
            Vote on business acquisitions and strategic decisions.
          </p>
          <button className={`${styles.button} ${styles.primary}`}>Explore Business DAO</button>
        </GlassyCard>
      </div>

      <GlassyCard className={styles.infoCard}>
        <h3 className={styles.sectionTitle}>What are DAOs?</h3>
        <p className={styles.sectionText}>
          Decentralized Autonomous Organizations (DAOs) are member-owned communities without centralized leadership.
          They operate using smart contracts on blockchain technology, allowing for transparent governance and collective decision-making.
        </p>
        <p className={styles.sectionText}>
          Our DAOs enable you to participate in various investment opportunities with voting rights proportional to your stake.
          Join any of our specialized DAOs to start participating in decentralized governance today.
        </p>
      </GlassyCard>
    </PageLayout>
  );
}