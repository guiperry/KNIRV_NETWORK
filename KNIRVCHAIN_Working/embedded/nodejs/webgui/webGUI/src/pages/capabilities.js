import { useEffect, useState } from 'react';
import { useBackend } from '../contexts/BackendContext';
import { useNavigation } from '../hooks/useNavigation';
import api from '../utils/api';
import PageLayout from '../components/PageLayout';
import PageHeader from '../components/PageHeader';
import GlassyCard from '../components/GlassyCard';
import styles from './capabilities.module.css';

export default function Capabilities() {
  const { activePage } = useNavigation('capabilities');
  const { isRunning } = useBackend();
  const [capabilities, setCapabilities] = useState([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState('');
  const [selectedCapability, setSelectedCapability] = useState(null);
  const [searchQuery, setSearchQuery] = useState('');

  useEffect(() => {
    if (isRunning) {
      fetchCapabilities();
    }
  }, [isRunning]);

  const fetchCapabilities = async () => {
    setIsLoading(true);
    try {
      const response = await api.get('/mcp/capability/list');
      setCapabilities(response.data.capabilities || []);
      setError('');
    } catch (error) {
      setError('Failed to fetch capabilities');
      console.error('Error fetching capabilities:', error);
    } finally {
      setIsLoading(false);
    }
  };

  const handleCapabilityClick = (capability) => {
    setSelectedCapability(capability);
  };

  const handleInvokeCapability = async (capabilityId) => {
    try {
      await api.post('/mcp/capability/invoke', {
        capability_id: capabilityId,
        params: {}
      });
      alert('Capability invoked successfully!');
    } catch (error) {
      alert('Failed to invoke capability: ' + (error.response?.data?.message || error.message));
    }
  };

  const handleSearch = (query) => {
    setSearchQuery(query);
  };

  const filteredCapabilities = capabilities.filter(cap => 
    cap.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
    cap.type.toLowerCase().includes(searchQuery.toLowerCase())
  );

  if (!isRunning) {
    return (
      <PageLayout activePage={activePage} pageTitle="Capabilities">
        <GlassyCard darker className={styles.errorCard}>
          Backend is not running. Please start the KNIRVCHAIN node.
        </GlassyCard>
      </PageLayout>
    );
  }

  return (
    <PageLayout 
      activePage={activePage} 
      pageTitle="Capabilities" 
      onSearch={handleSearch}
    >
      <PageHeader 
        title="Capabilities" 
        subtitle="Manage and invoke available capabilities"
      />

      {error && (
        <GlassyCard darker className={styles.errorCard}>
          {error}
        </GlassyCard>
      )}

      <div className={styles.capabilitiesContainer}>
        {/* Capabilities List */}
        <GlassyCard darker className={styles.listContainer}>
          <div className={styles.sectionHeader}>
            <h3>Available Capabilities</h3>
            <button
              onClick={fetchCapabilities}
              className={`${styles.button} ${styles.secondary} ${styles.refreshButton}`}
              disabled={isLoading}
            >
              {isLoading ? 'Refreshing...' : 'Refresh'}
            </button>
          </div>

          {isLoading ? (
            <div className={styles.loading}>Loading capabilities...</div>
          ) : filteredCapabilities.length === 0 ? (
            <div className={styles.emptyState}>No capabilities found</div>
          ) : (
            <div className={styles.capabilitiesGrid}>
              {filteredCapabilities.map((capability) => (
                <GlassyCard 
                  key={capability.id}
                  darker
                  className={`${styles.capabilityItem} ${selectedCapability?.id === capability.id ? styles.selected : ''}`}
                  onClick={() => handleCapabilityClick(capability)}
                >
                  <div className={styles.capabilityName}>{capability.name}</div>
                  <div className={styles.capabilityType}>{capability.type}</div>
                  <div className={styles.capabilityId}>ID: {capability.id.substring(0, 8)}...</div>
                </GlassyCard>
              ))}
            </div>
          )}
        </GlassyCard>

        {/* Capability Details */}
        {selectedCapability && (
          <GlassyCard darker className={styles.detailsContainer}>
            <h3 className={styles.sectionTitle}>Capability Details</h3>
            
            <div className={styles.detailsGrid}>
              <div className={styles.detailItem}>
                <strong>Name:</strong> {selectedCapability.name}
              </div>
              <div className={styles.detailItem}>
                <strong>Type:</strong> {selectedCapability.type}
              </div>
              <div className={styles.detailItem}>
                <strong>ID:</strong> {selectedCapability.id}
              </div>
              <div className={styles.detailItem}>
                <strong>Owner:</strong> {selectedCapability.owner}
              </div>
              <div className={styles.detailItem}>
                <strong>Description:</strong> {selectedCapability.description}
              </div>
            </div>

            <GlassyCard className={styles.propertiesSection}>
              <h4 className={styles.subsectionTitle}>Properties</h4>
              <div className={styles.descriptorContent}>
                {Object.entries(selectedCapability.descriptor || {}).map(([key, value]) => (
                  <div key={key} className={styles.descriptorItem}>
                    <strong>{key}:</strong> {JSON.stringify(value)}
                  </div>
                ))}
              </div>
            </GlassyCard>

            <div className={styles.actionsSection}>
              <button 
                onClick={() => handleInvokeCapability(selectedCapability.id)}
                className={`${styles.button} ${styles.primary}`}
              >
                Invoke Capability
              </button>
            </div>
          </GlassyCard>
        )}
      </div>
    </PageLayout>
  );
}