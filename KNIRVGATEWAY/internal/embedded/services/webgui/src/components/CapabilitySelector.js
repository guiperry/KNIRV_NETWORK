import React, { useState } from 'react';
import styles from './CapabilitySelector.module.css';

const CapabilitySelector = ({ capabilities, selectedCapability, onSelect, attachedCapabilities }) => {
  const [filter, setFilter] = useState('all');
  
  // Get IDs of already attached capabilities
  const attachedCapabilityIds = attachedCapabilities.map(cap => cap.id);
  
  // Filter capabilities based on filter selection
  const filteredCapabilities = capabilities.filter(cap => {
    if (filter === 'all') return true;
    if (filter === 'available') return !attachedCapabilityIds.includes(cap.id);
    if (filter === 'attached') return attachedCapabilityIds.includes(cap.id);
    return true;
  });

  return (
    <div className={styles.capabilitySelector}>
      <div className={styles.filterControls}>
        <button 
          className={`${styles.filterButton} ${filter === 'all' ? styles.active : ''}`}
          onClick={() => setFilter('all')}
        >
          All
        </button>
        <button 
          className={`${styles.filterButton} ${filter === 'available' ? styles.active : ''}`}
          onClick={() => setFilter('available')}
        >
          Available
        </button>
        <button 
          className={`${styles.filterButton} ${filter === 'attached' ? styles.active : ''}`}
          onClick={() => setFilter('attached')}
        >
          Attached
        </button>
      </div>

      {filteredCapabilities.length === 0 ? (
        <div className={styles.emptyState}>
          {filter === 'attached' 
            ? 'No capabilities attached to this NFT yet' 
            : 'No capabilities available'}
        </div>
      ) : (
        <div className={styles.capabilityList}>
          {filteredCapabilities.map(capability => {
            const isAttached = attachedCapabilityIds.includes(capability.id);
            
            return (
              <div 
                key={capability.id} 
                className={`
                  ${styles.capabilityItem} 
                  ${selectedCapability?.id === capability.id ? styles.selected : ''}
                  ${isAttached ? styles.attached : ''}
                `}
                onClick={() => onSelect(capability)}
              >
                <div className={styles.capabilityHeader}>
                  <h4 className={styles.capabilityName}>{capability.name}</h4>
                  {isAttached && <span className={styles.attachedBadge}>Attached</span>}
                </div>
                <div className={styles.capabilityType}>{capability.type}</div>
                <div className={styles.capabilityDescription}>
                  {capability.description?.substring(0, 100)}
                  {capability.description?.length > 100 ? '...' : ''}
                </div>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
};

export default CapabilitySelector;