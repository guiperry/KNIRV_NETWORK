import React, { useState, useEffect, useMemo } from 'react';
import { useNavigation } from '../hooks/useNavigation';
import PageLayout from '../components/PageLayout';
import PageHeader from '../components/PageHeader';
import GlassyCard from '../components/GlassyCard';
import styles from './operator-registry.module.css';

export default function OperatorRegistry() {
  const { activePage } = useNavigation('operator-registry');
  const [allAgents, setAllAgents] = useState([]);
  const [searchTerm, setSearchTerm] = useState('');
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState(null);

  useEffect(() => {
    const fetchAgents = async () => {
      try {
        setIsLoading(true);
        const response = await fetch('/operator-registry/api/operators');
        if (!response.ok) {
          throw new Error('Failed to fetch operators');
        }
        const agents = await response.json();
        setAllAgents(agents);
        setError(null);
      } catch (err) {
        console.error("Failed to fetch agents:", err);
        setError("Could not load operators. Please try again later.");
      } finally {
        setIsLoading(false);
      }
    };
    fetchAgents();
  }, []);

  const filteredAgents = useMemo(() => {
    if (!searchTerm) return allAgents;
    return allAgents.filter(operator =>
      operator.name?.toLowerCase().includes(searchTerm.toLowerCase()) ||
      operator.capability?.toLowerCase().includes(searchTerm.toLowerCase()) ||
      (operator.provider && operator.provider.toLowerCase().includes(searchTerm.toLowerCase())) ||
      (operator.description && operator.description.toLowerCase().includes(searchTerm.toLowerCase()))
    );
  }, [searchTerm, allAgents]);

  const handleSearchSubmit = (event) => {
    event.preventDefault();
    // Search is already live filtering, so submit doesn't need to do much more
  };

  return (
    <PageLayout activePage={activePage} pageTitle="Operator Registry">
      <PageHeader
        title="Operator Registry"
        subtitle="Discover and onboard verified network operators (bootnodes, routers)"
        titleColor="#007bff"
      />

      <div className={styles.container}>
        {/* Hero Section */}
        <GlassyCard className={styles.heroSection}>
          <div className={styles.heroContent}>
            <div className={styles.heroIcon}>
              🛡️
            </div>
            <h1 className={styles.heroTitle}>
              Operator Registry — Verifiable Network Operators
            </h1>
            <p className={styles.heroText}>
              Discover and onboard verified network operators (bootnodes, routers). The Operator Registry uses NANDA+ANS principles to provide signed pointers (OperatorAddr) to verifiable OperatorFacts. Operators are discoverable, auditable, and can be validated for secure network participation.
            </p>
          </div>
        </GlassyCard>

        {/* Search Form */}
        <GlassyCard className={styles.searchSection}>
          <form onSubmit={handleSearchSubmit} className={styles.searchForm}>
            <div className={styles.searchInputGroup}>
              <input
                type="text"
                placeholder="Search operators by name, capability, provider, or description..."
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
                className={styles.searchInput}
              />
              <button type="submit" className={styles.searchButton}>
                🔍 Search
              </button>
            </div>
          </form>
        </GlassyCard>

        {/* Error Alert */}
        {error && (
          <GlassyCard className={styles.errorCard}>
            <div className={styles.errorContent}>
              <div className={styles.errorIcon}>⚠️</div>
              <div>
                <h3 className={styles.errorTitle}>Error</h3>
                <p className={styles.errorText}>{error}</p>
              </div>
            </div>
          </GlassyCard>
        )}

        {/* Loading State */}
        {isLoading ? (
          <div className={styles.loadingGrid}>
            {[...Array(6)].map((_, i) => (
              <OperatorCardSkeleton key={i} />
            ))}
          </div>
        ) : filteredAgents.length > 0 ? (
          /* Operators Grid */
          <div className={styles.operatorsGrid}>
            {filteredAgents.map(operator => (
              <OperatorCard key={operator.id} operator={operator} />
            ))}
          </div>
        ) : (
          /* No Results */
          <GlassyCard className={styles.noResultsCard}>
            <div className={styles.noResultsContent}>
              <div className={styles.noResultsIcon}>🔍</div>
              <h3 className={styles.noResultsTitle}>No Operators Found</h3>
              <p className={styles.noResultsText}>
                Your search for "{searchTerm}" did not match any operators. Try a different term or explore all operators. If this is your first time, try registering an operator!
              </p>
            </div>
          </GlassyCard>
        )}
      </div>
    </PageLayout>
  );
}

function OperatorCard({ operator }) {
  return (
    <GlassyCard className={styles.operatorCard}>
      <div className={styles.operatorHeader}>
        <div className={styles.operatorIcon}>
          🤖
        </div>
        <div className={styles.operatorInfo}>
          <h3 className={styles.operatorName}>{operator.name || 'Unnamed Operator'}</h3>
          <p className={styles.operatorCapability}>{operator.capability || 'Unknown Capability'}</p>
        </div>
      </div>

      <div className={styles.operatorDetails}>
        {operator.provider && (
          <div className={styles.detailItem}>
            <span className={styles.detailLabel}>Provider:</span>
            <span className={styles.detailValue}>{operator.provider}</span>
          </div>
        )}

        {operator.description && (
          <div className={styles.detailItem}>
            <span className={styles.detailLabel}>Description:</span>
            <span className={styles.detailValue}>{operator.description}</span>
          </div>
        )}

        <div className={styles.detailItem}>
          <span className={styles.detailLabel}>ID:</span>
          <span className={styles.detailValue}>{operator.id}</span>
        </div>

        {operator.endpoint && (
          <div className={styles.detailItem}>
            <span className={styles.detailLabel}>Endpoint:</span>
            <span className={styles.detailValue}>{operator.endpoint}</span>
          </div>
        )}
      </div>

      <div className={styles.operatorActions}>
        <button className={styles.actionButton}>
          View Details
        </button>
        <button className={styles.actionButton}>
          Connect
        </button>
      </div>
    </GlassyCard>
  );
}

function OperatorCardSkeleton() {
  return (
    <GlassyCard className={styles.operatorCard}>
      <div className={styles.operatorHeader}>
        <div className={styles.skeletonIcon}></div>
        <div className={styles.operatorInfo}>
          <div className={styles.skeletonTitle}></div>
          <div className={styles.skeletonText}></div>
        </div>
      </div>
      <div className={styles.operatorDetails}>
        <div className={styles.skeletonText}></div>
        <div className={styles.skeletonText}></div>
        <div className={styles.skeletonText}></div>
      </div>
      <div className={styles.operatorActions}>
        <div className={styles.skeletonButton}></div>
        <div className={styles.skeletonButton}></div>
      </div>
    </GlassyCard>
  );
}