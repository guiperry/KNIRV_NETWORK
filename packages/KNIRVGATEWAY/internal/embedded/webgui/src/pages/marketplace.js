import React, { useState } from 'react';
import { useNavigation } from '../hooks/useNavigation';
import PageLayout from '../components/PageLayout';
import PageHeader from '../components/PageHeader';
import GlassyCard from '../components/GlassyCard';
import OnboardingFlow from '../components/OnboardingFlowUpdated';
import styles from './marketplace.module.css';

export default function Marketplace() {
  const { activePage } = useNavigation('marketplace');
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedType, setSelectedType] = useState('all');
  const [selectedCategory, setSelectedCategory] = useState('all');
  const [showAddModal, setShowAddModal] = useState(false);

  const handleSearch = (query) => {
    setSearchQuery(query);
  };

  // Mock marketplace data
  const marketplaceItems = [
    {
      id: '1',
      name: 'Advanced Code Generation',
      type: 'skill',
      price: '50 NRN',
      owner: '0x1234...abcd',
      rating: 4.8,
      description: 'High-quality code generation skill for multiple programming languages',
      category: 'Development'
    },
    {
      id: '2',
      name: 'Image Recognition API',
      type: 'capability',
      price: '75 NRN',
      owner: '0x5678...efgh',
      rating: 4.6,
      description: 'Advanced image recognition capability with 99% accuracy',
      category: 'Computer Vision'
    },
    {
      id: '3',
      name: 'Sentiment Analysis Dataset',
      type: 'property',
      price: '25 NRN',
      owner: '0x9abc...ijkl',
      rating: 4.3,
      description: 'Curated dataset for sentiment analysis training',
      category: 'Data'
    },
    {
      id: '4',
      name: 'Natural Language Processing',
      type: 'skill',
      price: '80 NRN',
      owner: '0xdef0...mnop',
      rating: 4.9,
      description: 'Comprehensive NLP skill with multilingual support',
      category: 'Language'
    }
  ];

  // Filter items based on search and filters
  const filteredItems = marketplaceItems.filter((item) => {
    const matchesSearch = item.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
                         item.description.toLowerCase().includes(searchQuery.toLowerCase());
    const matchesType = selectedType === 'all' || item.type === selectedType;
    const matchesCategory = selectedCategory === 'all' || item.category === selectedCategory;
    return matchesSearch && matchesType && matchesCategory;
  });

  const categories = ['all', 'Development', 'Computer Vision', 'Data', 'Language', 'Audio', 'Security'];
  const types = ['all', 'skill', 'capability', 'property'];

  const getTypeIcon = (type) => {
    switch (type) {
      case 'skill': return '⚡';
      case 'capability': return '🔌';
      case 'property': return '🏷️';
      default: return '📦';
    }
  };

  const getTypeColor = (type) => {
    switch (type) {
      case 'skill': return '#ffc107';
      case 'capability': return '#28a745';
      case 'property': return '#007bff';
      default: return '#6c757d';
    }
  };

  const renderStars = (rating) => {
    const stars = [];
    const fullStars = Math.floor(rating);
    const hasHalfStar = rating % 1 !== 0;

    for (let i = 0; i < fullStars; i++) {
      stars.push(<i key={i} className="fas fa-star" style={{ color: '#ffc107' }}></i>);
    }

    if (hasHalfStar) {
      stars.push(<i key="half" className="fas fa-star-half-alt" style={{ color: '#ffc107' }}></i>);
    }

    const emptyStars = 5 - Math.ceil(rating);
    for (let i = 0; i < emptyStars; i++) {
      stars.push(<i key={`empty-${i}`} className="far fa-star" style={{ color: '#6c757d' }}></i>);
    }

    return <div className={styles.rating}>{stars}</div>;
  };

  return (
    <PageLayout activePage={activePage} pageTitle="Marketplace" onSearch={handleSearch}>
      <PageHeader 
        title="KNIRV Marketplace"
        subtitle="Discover and trade skills, capabilities, and properties"
        titleColor="#007bff"
      />

      {/* Filters */}
      <GlassyCard className={styles.filtersCard}>
        <div className={styles.filtersHeader}>
          <h3 className={styles.filtersTitle}>
            <i className="fas fa-filter" style={{ marginRight: '10px', color: '#007bff' }}></i>
            Filters
          </h3>
          <button
            onClick={() => setShowAddModal(true)}
            className={styles.addButton}
          >
            <i className="fas fa-plus" style={{ marginRight: '6px' }}></i>
            Add
          </button>
        </div>
        
        <div className={styles.filtersGrid}>
          <div className={styles.filterGroup}>
            <label className={styles.filterLabel}>Type</label>
            <div className={styles.filterButtons}>
              {types.map((type) => (
                <button
                  key={type}
                  onClick={() => setSelectedType(type)}
                  className={`${styles.filterButton} ${
                    selectedType === type ? styles.active : ''
                  }`}
                >
                  {type === 'all' ? 'All Types' : type.charAt(0).toUpperCase() + type.slice(1)}
                </button>
              ))}
            </div>
          </div>

          <div className={styles.filterGroup}>
            <label className={styles.filterLabel}>Category</label>
            <div className={styles.filterButtons}>
              {categories.map((category) => (
                <button
                  key={category}
                  onClick={() => setSelectedCategory(category)}
                  className={`${styles.filterButton} ${
                    selectedCategory === category ? styles.active : ''
                  }`}
                >
                  {category === 'all' ? 'All Categories' : category}
                </button>
              ))}
            </div>
          </div>
        </div>
      </GlassyCard>

      {/* Statistics */}
      <div className={styles.statsContainer}>
        <GlassyCard className={styles.statCard}>
          <div className={styles.statIcon}>⚡</div>
          <div className={styles.statValue}>{marketplaceItems.filter(i => i.type === 'skill').length}</div>
          <div className={styles.statLabel}>Skills</div>
        </GlassyCard>
        <GlassyCard className={styles.statCard}>
          <div className={styles.statIcon}>🔌</div>
          <div className={styles.statValue}>{marketplaceItems.filter(i => i.type === 'capability').length}</div>
          <div className={styles.statLabel}>Capabilities</div>
        </GlassyCard>
        <GlassyCard className={styles.statCard}>
          <div className={styles.statIcon}>🏷️</div>
          <div className={styles.statValue}>{marketplaceItems.filter(i => i.type === 'property').length}</div>
          <div className={styles.statLabel}>Properties</div>
        </GlassyCard>
        <GlassyCard className={styles.statCard}>
          <div className={styles.statIcon}>💰</div>
          <div className={styles.statValue}>2,847</div>
          <div className={styles.statLabel}>Total Volume (NRN)</div>
        </GlassyCard>
      </div>

      {/* Marketplace Items */}
      <div className={styles.itemsGrid}>
        {filteredItems.length > 0 ? (
          filteredItems.map((item) => (
            <GlassyCard key={item.id} className={styles.itemCard}>
              <div className={styles.itemHeader}>
                <div className={styles.itemType} style={{ backgroundColor: getTypeColor(item.type) }}>
                  {getTypeIcon(item.type)} {item.type.charAt(0).toUpperCase() + item.type.slice(1)}
                </div>
                <div className={styles.itemCategory}>{item.category}</div>
              </div>
              
              <h3 className={styles.itemName}>{item.name}</h3>
              <p className={styles.itemDescription}>{item.description}</p>
              
              <div className={styles.itemMeta}>
                <div className={styles.itemOwner}>
                  <i className="fas fa-user" style={{ marginRight: '5px', color: '#ccc' }}></i>
                  {item.owner}
                </div>
                {renderStars(item.rating)}
                <span className={styles.ratingValue}>({item.rating})</span>
              </div>
              
              <div className={styles.itemFooter}>
                <div className={styles.itemPrice}>{item.price}</div>
                <div className={styles.itemActions}>
                  <button className={styles.viewButton}>View</button>
                  <button className={styles.buyButton}>Buy</button>
                </div>
              </div>
            </GlassyCard>
          ))
        ) : (
          <div className={styles.emptyState}>
            <i className="fas fa-store" style={{ fontSize: '3rem', color: '#666', marginBottom: '20px' }}></i>
            <h3>No items found</h3>
            <p>Try adjusting your search criteria or browse different categories.</p>
          </div>
        )}
      </div>

      {/* Settlement Information */}
      <GlassyCard className={styles.settlementCard}>
        <h3 className={styles.settlementTitle}>
          <i className="fas fa-handshake" style={{ marginRight: '10px', color: '#007bff' }}></i>
          Settlement & Trading
        </h3>
        <p className={styles.settlementDescription}>
          All transactions are settled automatically through smart contracts. 
          Ownership transfers are recorded on the KNIRV blockchain for transparency and security.
        </p>
        <div className={styles.settlementStats}>
          <div className={styles.settlementStat}>
            <span className={styles.settlementStatValue}>99.9%</span>
            <span className={styles.settlementStatLabel}>Success Rate</span>
          </div>
          <div className={styles.settlementStat}>
            <span className={styles.settlementStatValue}>&lt; 5s</span>
            <span className={styles.settlementStatLabel}>Avg Settlement Time</span>
          </div>
          <div className={styles.settlementStat}>
            <span className={styles.settlementStatValue}>0.1%</span>
            <span className={styles.settlementStatLabel}>Transaction Fee</span>
          </div>
        </div>
      </GlassyCard>

      {showAddModal && (
        <div className={styles.modalOverlay} onClick={() => setShowAddModal(false)}>
          <div className={styles.modalContent} onClick={(e) => e.stopPropagation()}>
            <OnboardingFlow onComplete={() => setShowAddModal(false)} />
          </div>
        </div>
      )}
    </PageLayout>
  );
}
