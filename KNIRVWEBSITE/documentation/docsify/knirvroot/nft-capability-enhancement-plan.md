

---

**Source**: KNIRVROOT/altgui/docs/nft-capability-enhancement-plan.md

# NFT Capability Enhancement Implementation Plan

## Overview

This document outlines the implementation plan for adding a new page to the KNIRVCHAIN Next.js application that will allow client mode users to add registered capabilities to their already minted NFT image objects. This feature will enhance the functionality of NFTs by allowing users to attach capabilities that can extend their utility beyond simple ownership.

## Current System Analysis

Based on the analysis of the existing codebase, we have identified the following key components:

1. **NFT Management**:
   - The application already has an NFT Vault page (`/nft-vault`) that allows users to view their NFTs and attach capabilities.
   - The backend provides APIs for NFT management (`/nft/list`, `/nft/upload`, `/nft/attach-capability`).

2. **Capability Management**:
   - The application has a Capabilities page (`/capabilities`) that lists available capabilities.
   - The Add Capability page (`/add-capability`) allows users to register new capabilities.
   - The backend provides APIs for capability management (`/mcp/capability/list`, `/mcp/capability/prepare_registration`, `/mcp/capability/invoke`).

3. **Navigation and Layout**:
   - The application uses a custom navigation hook (`useNavigation`) for page navigation.
   - The `PageLayout` component provides a consistent layout with a sidebar and content area.
   - The `SideNavigation` component displays the navigation menu.

4. **Backend Communication**:
   - The application uses Axios for API communication through a custom `api` utility.
   - The `BackendContext` provides information about the backend status.
   - The `BlockchainContext` provides blockchain-related functionality.

## Implementation Plan

### 1. Create New Page: NFT Capability Manager

#### File: `/altgui/src/pages/nft-capability-manager.js`

This page will provide a dedicated interface for managing capabilities on NFTs, with a focus on client mode users. It will include:

- A list of user's minted NFTs
- A detailed view of the selected NFT
- A list of available capabilities that can be attached to the NFT
- A form for attaching capabilities to the selected NFT
- A history of capability attachments

#### Implementation Steps:

1. Create the basic page structure using the `PageLayout` component
2. Implement NFT listing and selection functionality
3. Implement capability listing and selection functionality
4. Implement capability attachment functionality
5. Implement capability attachment history

### 2. Create Styling for the New Page

#### File: `/altgui/src/pages/nft-capability-manager.module.css`

This CSS module will provide styling for the new page, following the existing design patterns of the application.

### 3. Update Navigation

#### File: `/altgui/src/components/SideNavigation.js`

Add a new navigation item for the NFT Capability Manager page.

### 4. Create New Components

#### Component: NFTSelector

**File**: `/altgui/src/components/NFTSelector.js`  
**Purpose**: Allow users to select an NFT from their collection.

#### Component: CapabilitySelector

**File**: `/altgui/src/components/CapabilitySelector.js`  
**Purpose**: Allow users to select capabilities to attach to an NFT.

#### Component: CapabilityAttachmentForm

**File**: `/altgui/src/components/CapabilityAttachmentForm.js`  
**Purpose**: Provide a form for attaching capabilities to NFTs with additional parameters.

#### Component: CapabilityAttachmentHistory

**File**: `/altgui/src/components/CapabilityAttachmentHistory.js`  
**Purpose**: Display the history of capability attachments for an NFT.

### 5. API Integration

#### Existing APIs to Use:

- `/nft/list` - Get list of user's NFTs
- `/nft/attach-capability` - Attach a capability to an NFT
- `/mcp/capability/list` - Get list of available capabilities

#### New API Endpoints Needed:

- `/nft/capability-history` - Get history of capability attachments for an NFT
- `/nft/detach-capability` - Remove a capability from an NFT (if applicable)

### 6. Detailed Component Implementation

#### NFT Capability Manager Page

```jsx
import { useState, useEffect } from 'react';
import { useNavigation } from '../hooks/useNavigation';
import { useBackend } from '../contexts/BackendContext';
import api from '../utils/api';
import PageLayout from '../components/PageLayout';
import PageHeader from '../components/PageHeader';
import GlassyCard from '../components/GlassyCard';
import NFTSelector from '../components/NFTSelector';
import CapabilitySelector from '../components/CapabilitySelector';
import CapabilityAttachmentForm from '../components/CapabilityAttachmentForm';
import CapabilityAttachmentHistory from '../components/CapabilityAttachmentHistory';
import styles from './nft-capability-manager.module.css';

export default function NFTCapabilityManager() {
  const { activePage } = useNavigation('nft-capability-manager');
  const { isRunning } = useBackend();
  const [nfts, setNfts] = useState([]);
  const [selectedNft, setSelectedNft] = useState(null);
  const [capabilities, setCapabilities] = useState([]);
  const [selectedCapability, setSelectedCapability] = useState(null);
  const [attachmentHistory, setAttachmentHistory] = useState([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');
  const [searchQuery, setSearchQuery] = useState('');

  useEffect(() => {
    if (isRunning) {
      fetchNFTs();
      fetchCapabilities();
    }
  }, [isRunning]);

  useEffect(() => {
    if (selectedNft && isRunning) {
      fetchAttachmentHistory(selectedNft.id);
    }
  }, [selectedNft, isRunning]);

  const fetchNFTs = async () => {
    setIsLoading(true);
    try {
      const response = await api.get('/nft/list');
      setNfts(response.data.nfts || []);
      setError('');
    } catch (error) {
      setError('Failed to fetch NFTs');
      console.error('Error fetching NFTs:', error);
    } finally {
      setIsLoading(false);
    }
  };

  const fetchCapabilities = async () => {
    try {
      const response = await api.get('/mcp/capability/list');
      setCapabilities(response.data.capabilities || []);
    } catch (error) {
      console.error('Error fetching capabilities:', error);
    }
  };

  const fetchAttachmentHistory = async (nftId) => {
    try {
      // This endpoint would need to be implemented on the backend
      const response = await api.get(`/nft/capability-history/${nftId}`);
      setAttachmentHistory(response.data.history || []);
    } catch (error) {
      console.error('Error fetching attachment history:', error);
    }
  };

  const handleNftSelect = (nft) => {
    setSelectedNft(nft);
    setSelectedCapability(null);
  };

  const handleCapabilitySelect = (capability) => {
    setSelectedCapability(capability);
  };

  const handleAttachCapability = async (params) => {
    if (!selectedNft || !selectedCapability) return;

    try {
      await api.post('/nft/attach-capability', {
        nft_id: selectedNft.id,
        capability_id: selectedCapability.id,
        params: params
      });
      
      setSuccess('Capability attached successfully!');
      fetchNFTs();
      fetchAttachmentHistory(selectedNft.id);
      setSelectedCapability(null);
    } catch (error) {
      setError('Failed to attach capability: ' + (error.response?.data?.message || error.message));
    }
  };

  const handleSearch = (query) => {
    setSearchQuery(query);
  };

  if (!isRunning) {
    return (
      <PageLayout activePage={activePage} pageTitle="NFT Capability Manager" onSearch={handleSearch}>
        <div className={styles.notRunning}>Backend is not running. Please start the KNIRVCHAIN node.</div>
      </PageLayout>
    );
  }

  return (
    <PageLayout activePage={activePage} pageTitle="NFT Capability Manager" onSearch={handleSearch}>
      <PageHeader 
        title="NFT Capability Manager" 
        subtitle="Enhance your NFTs with powerful capabilities"
      />

      {error && <GlassyCard darker className={styles.error}>{error}</GlassyCard>}
      {success && <GlassyCard darker className={styles.success}>{success}</GlassyCard>}

      <div className={styles.container}>
        <div className={styles.leftPanel}>
          <GlassyCard darker className={styles.nftSelectorCard}>
            <h3 className={styles.cardTitle}>Your NFTs</h3>
            <NFTSelector 
              nfts={nfts} 
              selectedNft={selectedNft} 
              onSelect={handleNftSelect} 
              searchQuery={searchQuery}
            />
          </GlassyCard>
        </div>

        <div className={styles.rightPanel}>
          {selectedNft ? (
            <>
              <GlassyCard darker className={styles.nftDetailsCard}>
                <h3 className={styles.cardTitle}>Selected NFT</h3>
                <div className={styles.nftDetails}>
                  <div className={styles.nftImage}>
                    <img src={selectedNft.image_url} alt={selectedNft.name} />
                  </div>
                  <div className={styles.nftInfo}>
                    <h2>{selectedNft.name}</h2>
                    <p>{selectedNft.description}</p>
                    <div className={styles.nftMetadata}>
                      <div><strong>ID:</strong> {selectedNft.id}</div>
                      <div><strong>Created:</strong> {new Date(selectedNft.created_at).toLocaleString()}</div>
                    </div>
                  </div>
                </div>
              </GlassyCard>

              <GlassyCard darker className={styles.capabilitiesCard}>
                <h3 className={styles.cardTitle}>Available Capabilities</h3>
                <CapabilitySelector 
                  capabilities={capabilities} 
                  selectedCapability={selectedCapability} 
                  onSelect={handleCapabilitySelect}
                  attachedCapabilities={selectedNft.capabilities || []}
                />
              </GlassyCard>

              {selectedCapability && (
                <GlassyCard darker className={styles.attachmentFormCard}>
                  <h3 className={styles.cardTitle}>Attach Capability</h3>
                  <CapabilityAttachmentForm 
                    nft={selectedNft}
                    capability={selectedCapability}
                    onAttach={handleAttachCapability}
                  />
                </GlassyCard>
              )}

              <GlassyCard darker className={styles.historyCard}>
                <h3 className={styles.cardTitle}>Capability History</h3>
                <CapabilityAttachmentHistory history={attachmentHistory} />
              </GlassyCard>
            </>
          ) : (
            <GlassyCard darker className={styles.selectPrompt}>
              <div className={styles.promptContent}>
                <div className={styles.promptIcon}>🖼️</div>
                <h3>Select an NFT</h3>
                <p>Choose an NFT from your collection to view and manage its capabilities.</p>
              </div>
            </GlassyCard>
          )}
        </div>
      </div>
    </PageLayout>
  );
}
```

### 7. Component Implementations

#### NFTSelector Component

```jsx
import React from 'react';
import Image from 'next/image';
import styles from './NFTSelector.module.css';

const NFTSelector = ({ nfts, selectedNft, onSelect, searchQuery }) => {
  const filteredNfts = nfts.filter(nft => 
    nft.name.toLowerCase().includes(searchQuery.toLowerCase())
  );

  return (
    <div className={styles.nftSelector}>
      {filteredNfts.length === 0 ? (
        <div className={styles.emptyState}>No NFTs found</div>
      ) : (
        <div className={styles.nftList}>
          {filteredNfts.map(nft => (
            <div 
              key={nft.id} 
              className={`${styles.nftItem} ${selectedNft?.id === nft.id ? styles.selected : ''}`}
              onClick={() => onSelect(nft)}
            >
              <div className={styles.nftThumbnail}>
                <Image
                  src={nft.image_url}
                  alt={nft.name}
                  width={60}
                  height={60}
                />
              </div>
              <div className={styles.nftInfo}>
                <h4>{nft.name}</h4>
                <div className={styles.capabilityCount}>
                  {nft.capabilities?.length || 0} capabilities
                </div>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
};

export default NFTSelector;
```

#### CapabilitySelector Component

```jsx
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
        <div className={styles.emptyState}>No capabilities found</div>
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
                <div className={styles.capabilityName}>
                  {capability.name}
                  {isAttached && <span className={styles.attachedBadge}>Attached</span>}
                </div>
                <div className={styles.capabilityType}>{capability.type}</div>
                <div className={styles.capabilityDescription}>{capability.description}</div>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
};

export default CapabilitySelector;
```

#### CapabilityAttachmentForm Component

```jsx
import React, { useState, useEffect } from 'react';
import styles from './CapabilityAttachmentForm.module.css';

const CapabilityAttachmentForm = ({ nft, capability, onAttach }) => {
  const [params, setParams] = useState({});
  const [paramFields, setParamFields] = useState([]);

  useEffect(() => {
    // Extract parameter fields from capability descriptor
    if (capability && capability.descriptor) {
      const fields = Object.keys(capability.descriptor).map(key => ({
        name: key,
        type: typeof capability.descriptor[key],
        value: '',
        required: key.endsWith('_required') || false
      }));
      
      setParamFields(fields);
      
      // Initialize params object
      const initialParams = {};
      fields.forEach(field => {
        initialParams[field.name] = '';
      });
      setParams(initialParams);
    }
  }, [capability]);

  const handleInputChange = (e) => {
    const { name, value } = e.target;
    setParams({
      ...params,
      [name]: value
    });
  };

  const handleSubmit = (e) => {
    e.preventDefault();
    onAttach(params);
  };

  return (
    <div className={styles.attachmentForm}>
      <div className={styles.formHeader}>
        <h4>Attach "{capability.name}" to "{nft.name}"</h4>
      </div>
      
      <form onSubmit={handleSubmit}>
        {paramFields.length > 0 ? (
          <div className={styles.paramFields}>
            {paramFields.map(field => (
              <div key={field.name} className={styles.formGroup}>
                <label htmlFor={field.name}>
                  {field.name.replace(/_/g, ' ')}
                  {field.required && <span className={styles.required}>*</span>}
                </label>
                
                {field.type === 'boolean' ? (
                  <select
                    id={field.name}
                    name={field.name}
                    value={params[field.name]}
                    onChange={handleInputChange}
                    required={field.required}
                  >
                    <option value="">Select</option>
                    <option value="true">True</option>
                    <option value="false">False</option>
                  </select>
                ) : field.type === 'number' ? (
                  <input
                    id={field.name}
                    name={field.name}
                    type="number"
                    value={params[field.name]}
                    onChange={handleInputChange}
                    required={field.required}
                  />
                ) : (
                  <input
                    id={field.name}
                    name={field.name}
                    type="text"
                    value={params[field.name]}
                    onChange={handleInputChange}
                    required={field.required}
                  />
                )}
              </div>
            ))}
          </div>
        ) : (
          <div className={styles.noParams}>
            This capability does not require any parameters.
          </div>
        )}
        
        <div className={styles.formActions}>
          <button type="submit" className={styles.attachButton}>
            Attach Capability
          </button>
        </div>
      </form>
    </div>
  );
};

export default CapabilityAttachmentForm;
```

#### CapabilityAttachmentHistory Component

```jsx
import React from 'react';
import styles from './CapabilityAttachmentHistory.module.css';

const CapabilityAttachmentHistory = ({ history }) => {
  if (!history || history.length === 0) {
    return (
      <div className={styles.emptyHistory}>
        No capability attachment history available.
      </div>
    );
  }

  return (
    <div className={styles.historyContainer}>
      <div className={styles.historyList}>
        {history.map((entry, index) => (
          <div key={index} className={styles.historyItem}>
            <div className={styles.historyIcon}>
              {entry.action === 'attach' ? '➕' : '➖'}
            </div>
            <div className={styles.historyContent}>
              <div className={styles.historyAction}>
                {entry.action === 'attach' ? 'Attached' : 'Detached'} capability
              </div>
              <div className={styles.historyCapability}>
                {entry.capability_name} ({entry.capability_type})
              </div>
              <div className={styles.historyTimestamp}>
                {new Date(entry.timestamp).toLocaleString()}
              </div>
              {entry.params && Object.keys(entry.params).length > 0 && (
                <div className={styles.historyParams}>
                  <div className={styles.paramsTitle}>Parameters:</div>
                  <div className={styles.paramsList}>
                    {Object.entries(entry.params).map(([key, value]) => (
                      <div key={key} className={styles.paramItem}>
                        <span className={styles.paramKey}>{key}:</span>
                        <span className={styles.paramValue}>{value}</span>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
};

export default CapabilityAttachmentHistory;
```

### 8. CSS Module Implementation

#### NFT Capability Manager Page CSS

```css
.container {
  display: flex;
  gap: 20px;
  margin-top: 20px;
}

.leftPanel {
  flex: 1;
  max-width: 350px;
}

.rightPanel {
  flex: 2;
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.nftSelectorCard,
.nftDetailsCard,
.capabilitiesCard,
.attachmentFormCard,
.historyCard {
  padding: 20px;
  margin-bottom: 20px;
}

.cardTitle {
  margin-top: 0;
  margin-bottom: 15px;
  font-size: 1.2rem;
  color: var(--text-primary);
}

.nftDetails {
  display: flex;
  gap: 20px;
}

.nftImage {
  width: 200px;
  height: 200px;
  position: relative;
  border-radius: 10px;
  overflow: hidden;
}

.nftImage img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.nftInfo {
  flex: 1;
}

.nftInfo h2 {
  margin-top: 0;
  margin-bottom: 10px;
  color: var(--text-primary);
}

.nftMetadata {
  margin-top: 15px;
  display: flex;
  flex-direction: column;
  gap: 5px;
}

.selectPrompt {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 300px;
}

.promptContent {
  text-align: center;
}

.promptIcon {
  font-size: 3rem;
  margin-bottom: 15px;
}

.error {
  background-color: rgba(255, 0, 0, 0.1);
  border-left: 4px solid #ff0000;
  padding: 15px;
  margin-bottom: 20px;
  color: #ff0000;
}

.success {
  background-color: rgba(0, 255, 0, 0.1);
  border-left: 4px solid #00ff00;
  padding: 15px;
  margin-bottom: 20px;
  color: #00ff00;
}

.notRunning {
  padding: 20px;
  text-align: center;
  background-color: rgba(255, 0, 0, 0.1);
  border-radius: 10px;
  margin-top: 20px;
}
```

### 9. Backend API Requirements

For this implementation to work, the backend needs to provide the following API endpoints:

1. **GET /nft/list**
   - Returns a list of NFTs owned by the user
   - Already implemented

2. **GET /mcp/capability/list**
   - Returns a list of available capabilities
   - Already implemented

3. **POST /nft/attach-capability**
   - Attaches a capability to an NFT
   - Already implemented
   - May need to be extended to support additional parameters

4. **GET /nft/capability-history/:nftId**
   - Returns the history of capability attachments for an NFT
   - Needs to be implemented

5. **POST /nft/detach-capability** (Optional)
   - Detaches a capability from an NFT
   - Needs to be implemented if detachment is supported

### 10. Testing Plan

1. **Unit Tests**
   - Test each component in isolation
   - Test form validation
   - Test API integration with mocked responses

2. **Integration Tests**
   - Test the complete flow of selecting an NFT and attaching a capability
   - Test error handling and edge cases

3. **User Acceptance Testing**
   - Test with real users to ensure the interface is intuitive
   - Gather feedback for improvements

### 11. Deployment Plan

1. **Development**
   - Implement the new page and components
   - Implement or extend the required backend APIs
   - Test in a development environment

2. **Staging**
   - Deploy to a staging environment
   - Perform integration testing
   - Verify compatibility with the existing system

3. **Production**
   - Deploy to production
   - Monitor for any issues
   - Gather user feedback

## Conclusion

This implementation plan provides a comprehensive approach to adding a new page that allows client mode users to add registered capabilities to their minted NFT image objects. By following this plan, the development team can efficiently implement this feature while maintaining consistency with the existing application architecture and design patterns.

The new NFT Capability Manager page will enhance the functionality of the KNIRVCHAIN application by providing a dedicated interface for managing NFT capabilities, improving the user experience, and expanding the utility of NFTs within the system.

---

<div class="footer-links">


© 2025 KNIRV Network
</div>
