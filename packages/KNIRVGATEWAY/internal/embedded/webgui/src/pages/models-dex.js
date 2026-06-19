import React, { useState, useEffect } from 'react';
import { useNavigation } from '../hooks/useNavigation';
import PageLayout from '../components/PageLayout';
import PageHeader from '../components/PageHeader';
import GlassyCard from '../components/GlassyCard';
import DataTable from '../components/DataTable';
import OnboardingFlow from '../components/OnboardingFlowUpdated';
import styles from './models-dex.module.css';

export default function ModelsDEX() {
  const { activePage } = useNavigation('models-dex');
  const [showOnboarding, setShowOnboarding] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedCategory, setSelectedCategory] = useState('all');
  const [models, setModels] = useState([]);

  const handleOnboardingComplete = () => {
    setShowOnboarding(false);
    // Refresh model listings after onboarding
    loadModelListings();
  };

  const handleSearch = (query) => {
    setSearchQuery(query);
  };

  const loadModelListings = () => {
    const mockModels = [
      {
        id: '1',
        name: 'Compute DVE Alpha',
        type: 'Compute',
        owner: '0x1234...abcd',
        price: '150 NRN',
        performance: '94.5%',
        status: 'active',
        description: 'High-performance compute DVE optimized for ML workloads'
      },
      {
        id: '2',
        name: 'TEE-SGX Vault DVE',
        type: 'TEE-SGX',
        owner: '0x5678...efgh',
        price: '200 NRN',
        performance: '97.2%',
        status: 'active',
        description: 'Secure enclave DVE with Intel SGX attestation'
      },
      {
        id: '3',
        name: 'Inference Engine DVE',
        type: 'Inference',
        owner: '0x9abc...ijkl',
        price: '75 NRN',
        performance: '91.8%',
        status: 'pending',
        description: 'Optimized inference DVE with multi-model support'
      },
      {
        id: '4',
        name: 'Storage DVE Prime',
        type: 'Storage',
        owner: '0xdef0...mnop',
        price: '120 NRN',
        performance: '89.3%',
        status: 'sold',
        description: 'Distributed storage DVE with redundancy guarantees'
      }
    ];
    setModels(mockModels);
  };

  useEffect(() => {
    loadModelListings();
  }, []);

  // Filter models based on search query and category
  const filteredModels = models.filter((model) => {
    const matchesSearch = model.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
                         model.type.toLowerCase().includes(searchQuery.toLowerCase()) ||
                         model.description.toLowerCase().includes(searchQuery.toLowerCase());
    const matchesCategory = selectedCategory === 'all' || model.type.toLowerCase().includes(selectedCategory.toLowerCase());
    return matchesSearch && matchesCategory;
  });

  // Table headers
  const tableHeaders = [
    { label: 'DVE Name' },
    { label: 'Type' },
    { label: 'Owner' },
    { label: 'Price', align: 'right' },
    { label: 'Performance', align: 'right' },
    { label: 'Status', align: 'center' },
    { label: 'Actions', align: 'center' },
  ];

  // Render table row
  const renderTableRow = (model, index) => (
    <tr className={styles.tableRow} key={model.id}>
      <td className={styles.tableCell}>
        <div className={styles.modelInfo}>
          <div className={styles.modelName}>{model.name}</div>
          <div className={styles.modelDescription}>{model.description}</div>
        </div>
      </td>
      <td className={styles.tableCell}>
        <span className={styles.modelType}>{model.type}</span>
      </td>
      <td className={styles.tableCell}>
        <span className={styles.ownerAddress}>{model.owner}</span>
      </td>
      <td className={`${styles.tableCell} ${styles.textRight}`}>
        <span className={styles.price}>{model.price}</span>
      </td>
      <td className={`${styles.tableCell} ${styles.textRight}`}>
        <span className={styles.performance}>{model.performance}</span>
      </td>
      <td className={`${styles.tableCell} ${styles.textCenter}`}>
        <span className={`${styles.status} ${styles[model.status]}`}>
          {model.status.charAt(0).toUpperCase() + model.status.slice(1)}
        </span>
      </td>
      <td className={`${styles.tableCell} ${styles.textCenter}`}>
        <div className={styles.actionButtons}>
          {model.status === 'active' && (
            <>
              <button className={styles.buyButton}>Buy</button>
              <button className={styles.viewButton}>View</button>
            </>
          )}
          {model.status === 'pending' && (
            <button className={styles.viewButton}>View</button>
          )}
          {model.status === 'sold' && (
            <span className={styles.soldText}>Sold</span>
          )}
        </div>
      </td>
    </tr>
  );

  const categories = ['all', 'Compute', 'Storage', 'Inference', 'TEE-SGX', 'TEE-SEV', 'General'];

  return (
    <PageLayout activePage={activePage} pageTitle="DVE DEX" onSearch={handleSearch}>
      {showOnboarding && <OnboardingFlow onComplete={handleOnboardingComplete} />}

      <PageHeader
        title="DVE DEX"
        subtitle="Decentralized exchange for Distributed Validation Environment ownership rights"
        titleColor="#7c4dff"
      />

      {/* Category Filter */}
      <GlassyCard className={styles.filterCard}>
        <div className={styles.filterHeader}>
          <h3 className={styles.filterTitle}>
            <i className="fas fa-filter" style={{ marginRight: '10px', color: '#7c4dff' }}></i>
            Filter DVEs
          </h3>
        </div>
        <div className={styles.categoryFilters}>
          {categories.map((category) => (
            <button
              key={category}
              onClick={() => setSelectedCategory(category)}
              className={`${styles.categoryButton} ${
                selectedCategory === category ? styles.active : ''
              }`}
            >
              {category === 'all' ? 'All Categories' : category}
            </button>
          ))}
        </div>
      </GlassyCard>

      {/* Statistics */}
      <div className={styles.statsContainer}>
        <GlassyCard className={styles.statCard}>
          <div className={styles.statIcon}>📊</div>
          <div className={styles.statValue}>{models.length}</div>
          <div className={styles.statLabel}>Total DVEs</div>
        </GlassyCard>
        <GlassyCard className={styles.statCard}>
          <div className={styles.statIcon}>🔥</div>
          <div className={styles.statValue}>{models.filter(m => m.status === 'active').length}</div>
          <div className={styles.statLabel}>Active Listings</div>
        </GlassyCard>
        <GlassyCard className={styles.statCard}>
          <div className={styles.statIcon}>💰</div>
          <div className={styles.statValue}>1,245</div>
          <div className={styles.statLabel}>Total Volume (NRN)</div>
        </GlassyCard>
        <GlassyCard className={styles.statCard}>
          <div className={styles.statIcon}>📈</div>
          <div className={styles.statValue}>+12.5%</div>
          <div className={styles.statLabel}>24h Change</div>
        </GlassyCard>
      </div>

      {/* Models Table */}
      <GlassyCard className={styles.tableCard}>
        <div className={styles.tableHeader}>
          <h3 className={styles.tableTitle}>
            <i className="fas fa-robot" style={{ marginRight: '10px', color: '#007bff' }}></i>
            Listed Models ({filteredModels.length})
          </h3>
          <button 
            onClick={() => setShowOnboarding(true)}
            className={styles.listModelButton}
          >
            <i className="fas fa-plus" style={{ marginRight: '8px' }}></i>
            List Your Model
          </button>
        </div>
        
        {filteredModels.length > 0 ? (
          <DataTable
            headers={tableHeaders}
            data={filteredModels}
            renderRow={renderTableRow}
            className={styles.modelsTable}
          />
        ) : (
          <div className={styles.emptyState}>
            <i className="fas fa-robot" style={{ fontSize: '3rem', color: '#666', marginBottom: '20px' }}></i>
            <h3>No models found</h3>
            <p>Try adjusting your search criteria or browse different categories.</p>
          </div>
        )}
      </GlassyCard>
    </PageLayout>
  );
}
