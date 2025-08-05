

---

**Source**: KNIRVROOT/altgui/docs/ui-standardization-plan.md

# UI Standardization Plan for AltGUI

## Overview

This document outlines the implementation plan to standardize the UI across all pages in the AltGUI application. The inventory.js page will serve as the source of truth for styling and navigation.

## Current Issues

Based on the analysis of the codebase, the following inconsistencies were identified:

1. **Navigation Menu Inconsistencies**:
   - Different pages have different navigation items
   - Some pages include "Add Capability" while others don't
   - Some pages include "NFT Vault" while others don't
   - The order of navigation items varies between pages

2. **Styling Inconsistencies**:
   - Each page has its own CSS module with duplicated styles
   - Inconsistent use of CSS variables
   - Inconsistent layout and spacing
   - Inconsistent implementation of glassy containers and other UI elements

3. **Component Usage Inconsistencies**:
   - Some pages use the SearchBar component, others implement search functionality directly
   - Inconsistent implementation of headers, cards, and tables

## Implementation Plan

### Phase 1. Create Shared Navigation Component

#### Step 1: Create a Standardized Navigation Component

Create a reusable `SideNavigation` component that will be used across all pages:

```jsx
// src/components/SideNavigation.js
import React from 'react';
import styles from './SideNavigation.module.css';
import { useNavigation } from '../hooks/useNavigation';

const SideNavigation = ({ activePage }) => {
  const { handleNavigation } = useNavigation(activePage);

  return (
    <div className={styles.sidebar}>
      <h2 className={styles.dashboardTitle}>Blockchain Dashboard</h2>

      <div
        onClick={() => handleNavigation('inventory')}
        className={`${styles.navItem} ${styles.glassyContainer} ${activePage === 'inventory' ? styles.active : styles.inactive}`}
      >
        <span className={styles.navIcon}>📦</span>
        <span>Inventory</span>
      </div>

      <div
        onClick={() => handleNavigation('vault')}
        className={`${styles.navItem} ${styles.glassyContainer} ${activePage === 'vault' ? styles.active : styles.inactive}`}
      >
        <span className={styles.navIcon}>🔒</span>
        <span>Vault</span>
      </div>

      <div
        onClick={() => handleNavigation('blockchain')}
        className={`${styles.navItem} ${styles.glassyContainer} ${activePage === 'blockchain' ? styles.active : styles.inactive}`}
      >
        <span className={styles.navIcon}>⛓️</span>
        <span>Blockchain</span>
      </div>

      <div
        onClick={() => handleNavigation('dex')}
        className={`${styles.navItem} ${styles.glassyContainer} ${activePage === 'dex' ? styles.active : styles.inactive}`}
      >
        <span className={styles.navIcon}>💱</span>
        <span>DEX</span>
      </div>

      <div
        onClick={() => handleNavigation('daos')}
        className={`${styles.navItem} ${styles.glassyContainer} ${activePage === 'daos' ? styles.active : styles.inactive}`}
      >
        <span className={styles.navIcon}>🏛️</span>
        <span>DAOs</span>
      </div>

      <div
        onClick={() => handleNavigation('add-capability')}
        className={`${styles.navItem} ${styles.glassyContainer} ${activePage === 'add-capability' ? styles.active : styles.inactive}`}
      >
        <span className={styles.navIcon}>➕</span>
        <span>Add Capability</span>
      </div>

      <div
        onClick={() => handleNavigation('devs')}
        className={`${styles.navItem} ${styles.glassyContainer} ${activePage === 'devs' ? styles.active : styles.inactive}`}
      >
        <span className={styles.navIcon}>👥</span>
        <span>Peers List</span>
      </div>

      <div
        onClick={() => handleNavigation('settlement')}
        className={`${styles.navItem} ${styles.glassyContainer} ${activePage === 'settlement' ? styles.active : styles.inactive}`}
      >
        <span className={styles.navIcon}>📝</span>
        <span>Settlement</span>
      </div>
    </div>
  );
};

export default SideNavigation;
```

#### Step 2: Create CSS Module for the Navigation Component

```css
/* src/components/SideNavigation.module.css */
.sidebar {
  width: 250px;
  background-color: var(--transparent-black-6);
  padding: 20px;
  display: flex;
  flex-direction: column;
}

.dashboardTitle {
  color: var(--bright-blue);
  margin-bottom: 30px;
}

.navItem {
  padding: 10px 15px;
  margin-bottom: 10px;
  display: flex;
  align-items: center;
  cursor: pointer;
}

.active {
  background-color: var(--bright-blue) !important;
  color: white !important;
}

.inactive {
  background-color: var(--transparent-white-05);
}

.navIcon {
  margin-right: 10px;
}

.glassyContainer {
  background-color: var(--transparent-white-05);
  backdrop-filter: blur(8px);
  border-radius: 15px;
  border: 1px solid var(--transparent-white-1);
  box-shadow: 0 8px 32px 0 rgba(0, 0, 0, 0.1);
}
```

### Phase 2. Create Shared Page Layout Component

#### Step 1: Create a Standardized Page Layout Component

```jsx
// src/components/PageLayout.js
import React from 'react';
import styles from './PageLayout.module.css';
import SideNavigation from './SideNavigation';
import SearchBar from './SearchBar';

const PageLayout = ({ 
  children, 
  activePage, 
  pageTitle, 
  onSearch = () => {} 
}) => {
  return (
    <div className={styles.dashboardContainer}>
      <SideNavigation activePage={activePage} />
      
      <div className={styles.mainContent}>
        {/* Top Navigation */}
        <div className={`${styles.topNav} ${styles.glassyContainer}`}>
          <h3 className={styles.pageTitle}>{pageTitle}</h3>
          <div className={styles.userControls}>
            <SearchBar onSearch={onSearch} />
            <span className={styles.controlIcon}>🔔</span>
            <span className={styles.controlIcon}>⚙️</span>
            <span className={styles.controlIcon}>👤</span>
          </div>
        </div>
        
        {children}
      </div>
    </div>
  );
};

export default PageLayout;
```

#### Step 2: Create CSS Module for the Page Layout Component

```css
/* src/components/PageLayout.module.css */
.dashboardContainer {
  background-color: var(--dark-blue);
  color: white;
  min-height: 100vh;
  display: flex;
  position: relative;
}

.mainContent {
  flex: 1;
  padding: 20px;
}

.topNav {
  padding: 15px 20px;
  margin-bottom: 20px;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.pageTitle {
  color: var(--bright-blue);
  margin: 0;
}

.userControls {
  display: flex;
  gap: 15px;
}

.controlIcon {
  cursor: pointer;
}

.glassyContainer {
  background-color: var(--transparent-white-05);
  backdrop-filter: blur(8px);
  border-radius: 15px;
  border: 1px solid var(--transparent-white-1);
  box-shadow: 0 8px 32px 0 rgba(0, 0, 0, 0.1);
}
```

### Phase 3. Create Shared UI Components

#### Step 1: Create Common UI Components

Create standardized components for common UI elements:

1. **PageHeader Component**:
```jsx
// src/components/PageHeader.js
import React from 'react';
import styles from './PageHeader.module.css';

const PageHeader = ({ title, subtitle }) => {
  return (
    <div className={styles.pageHeader}>
      <h2 className={styles.pageHeaderTitle}>{title}</h2>
      <p className={styles.pageHeaderSubtitle}>{subtitle}</p>
    </div>
  );
};

export default PageHeader;
```

2. **GlassyCard Component**:
```jsx
// src/components/GlassyCard.js
import React from 'react';
import styles from './GlassyCard.module.css';

const GlassyCard = ({ 
  children, 
  title, 
  darker = false, 
  className = '' 
}) => {
  return (
    <div className={`${styles.glassyCard} ${darker ? styles.darkerContainer : ''} ${className}`}>
      {title && <h3 className={styles.cardTitle}>{title}</h3>}
      {children}
    </div>
  );
};

export default GlassyCard;
```

3. **DataTable Component**:
```jsx
// src/components/DataTable.js
import React from 'react';
import styles from './DataTable.module.css';

const DataTable = ({ 
  headers, 
  data, 
  renderRow 
}) => {
  return (
    <div className={styles.tableWrapper}>
      <table className={styles.dataTable}>
        <thead>
          <tr className={styles.tableRow}>
            {headers.map((header, index) => (
              <th 
                key={index} 
                className={`${styles.tableHeader} ${header.align === 'right' ? styles.textRight : header.align === 'center' ? styles.textCenter : ''}`}
              >
                {header.label}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {data.map((item, index) => renderRow(item, index))}
        </tbody>
      </table>
    </div>
  );
};

export default DataTable;
```

### Phase 4. Create Shared CSS Variables and Utilities

#### Step 1: Consolidate CSS Variables

Ensure all CSS variables are defined in a single location:

```css
/* src/styles/variables.css */
:root {
  /* Colors */
  --dark-blue: #000c2e;
  --bright-blue: #007bff;
  --white: #fff;
  --success-text: #28a745;
  --danger-text: #dc3545;

  /* Transparent colors */
  --transparent-white-05: rgba(255, 255, 255, 0.05);
  --transparent-white-08: rgba(255, 255, 255, 0.08);
  --transparent-white-1: rgba(255, 255, 255, 0.1);
  --transparent-white-2: rgba(255, 255, 255, 0.2);
  --transparent-white-7: rgba(255, 255, 255, 0.7);
  --transparent-black-2: rgba(0, 0, 0, 0.2);
  --transparent-black-3: rgba(0, 0, 0, 0.3);
  --transparent-black-5: rgba(0, 0, 0, 0.5);
  --transparent-black-6: rgba(0, 0, 0, 0.6);

  /* Component-specific colors */
  --danger-color: rgba(220, 53, 69, 0.7);
  --success-color: rgba(25, 135, 84, 0.2);
  --primary-color-light: rgba(13, 110, 253, 0.2);
  --primary-text: #0d6efd;

  /* Cryptocurrency colors */
  --bitcoin-color: #f2a900;
  --ethereum-color: #627eea;
  --usdc-color: #2775ca;
  --tether-color: #26a17b;
}
```

#### Step 2: Create Utility Classes

```css
/* src/styles/utilities.css */
/* Text alignment */
.text-left {
  text-align: left;
}

.text-center {
  text-align: center;
}

.text-right {
  text-align: right;
}

/* Flex utilities */
.d-flex {
  display: flex;
}

.flex-column {
  flex-direction: column;
}

.justify-content-between {
  justify-content: space-between;
}

.align-items-center {
  align-items: center;
}

/* Spacing utilities */
.mb-1 {
  margin-bottom: 0.25rem;
}

.mb-2 {
  margin-bottom: 0.5rem;
}

.mb-3 {
  margin-bottom: 1rem;
}

.mb-4 {
  margin-bottom: 1.5rem;
}

.mb-5 {
  margin-bottom: 3rem;
}

/* Status colors */
.positive-change {
  color: var(--success-color);
}

.negative-change {
  color: var(--danger-color);
}
```

### Phase 5. Refactor Pages to Use Shared Components

#### Step 1: Refactor Inventory Page

```jsx
// src/pages/inventory.js
import React, { useState } from 'react';
import { useNavigation } from '../hooks/useNavigation';
import PageLayout from '../components/PageLayout';
import PageHeader from '../components/PageHeader';
import GlassyCard from '../components/GlassyCard';
import DataTable from '../components/DataTable';
import OnboardingFlow from '../components/OnboardingFlowUpdated';
import styles from './inventory.module.css';

export default function Inventory() {
  const { activePage } = useNavigation('inventory');
  const [showOnboarding, setShowOnboarding] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');

  const handleOnboardingComplete = () => {
    setShowOnboarding(false);
    // You could add additional logic here, like refreshing the inventory data
  };

  const handleSearch = (query) => {
    setSearchQuery(query);
  };

  // Example data (replace with your actual data)
  const assetsData = [
    { name: 'Bitcoin', type: 'Cryptocurrency', balance: '0.45 BTC', value: '$20,534.40', change: '+2.3%' },
    { name: 'Ethereum', type: 'Cryptocurrency', balance: '2.5 ETH', value: '$8,112.50', change: '-1.1%' },
    { name: 'USDC', type: 'Stablecoin', balance: '1,500 USDC', value: '$1,500.00', change: '+0.01%' },
    { name: 'Tether', type: 'Stablecoin', balance: '2,000 USDT', value: '$2,000.00', change: '-0.02%' },
  ];

  // Filter the data based on the search query
  const filteredAssetsData = assetsData.filter((asset) =>
    asset.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
    asset.type.toLowerCase().includes(searchQuery.toLowerCase())
  );

  // Table headers
  const tableHeaders = [
    { label: 'Asset' },
    { label: 'Type' },
    { label: 'Balance', align: 'right' },
    { label: 'Value (USD)', align: 'right' },
    { label: '24h Change', align: 'right' },
    { label: 'Actions', align: 'center' },
  ];

  // Render table row
  const renderTableRow = (asset, index) => (
    <tr className={styles.tableRow} key={index}>
      <td className={styles.tableCell}>
        <div className={styles.assetNameContainer}>
          <div className={`${styles.assetIcon} ${styles[asset.name.toLowerCase()]}`}>
            {asset.name.charAt(0)}
          </div>
          <div>{asset.name}</div>
        </div>
      </td>
      <td className={styles.tableCell}>{asset.type}</td>
      <td className={`${styles.tableCell} ${styles.textRight}`}>{asset.balance}</td>
      <td className={`${styles.tableCell} ${styles.textRight}`}>{asset.value}</td>
      <td className={`${styles.tableCell} ${styles.textRight} ${asset.change.startsWith('+') ? styles.positiveChange : styles.negativeChange}`}>{asset.change}</td>
      <td className={`${styles.tableCell} ${styles.textCenter}`}>
        <div className={styles.actionButtons}>
          <button className={`${styles.actionButton} ${styles.primary}`}>Send</button>
          <button className={`${styles.actionButton} ${styles.secondary}`}>Receive</button>
        </div>
      </td>
    </tr>
  );

  return (
    <PageLayout 
      activePage={activePage} 
      pageTitle="Inventory Management" 
      onSearch={handleSearch}
    >
      {showOnboarding && <OnboardingFlow onComplete={handleOnboardingComplete} />}
      
      <PageHeader 
        title="Digital Asset Inventory" 
        subtitle="Manage your blockchain assets" 
      />

      {/* Search and Filter */}
      <GlassyCard darker className={styles.searchFilterContainer}>
        <div className={styles.filterControls}>
          <select className={`${styles.filterSelect} ${styles.glassyContainer}`}>
            <option value="all">All Assets</option>
            <option value="crypto">Cryptocurrencies</option>
            <option value="nft">NFTs</option>
            <option value="tokens">Tokens</option>
          </select>

          <button
            onClick={() => setShowOnboarding(true)}
            className={styles.addAssetButton}
          >
            Add Asset
          </button>
        </div>
      </GlassyCard>

      {/* Asset Table */}
      <GlassyCard title="Your Assets" darker className={styles.assetTableContainer}>
        <DataTable 
          headers={tableHeaders} 
          data={filteredAssetsData} 
          renderRow={renderTableRow} 
        />
      </GlassyCard>

      {/* Portfolio Summary */}
      <GlassyCard title="Portfolio Summary" darker className={styles.portfolioSummary}>
        <div className={styles.summaryCards}>
          <GlassyCard darker className={styles.summaryCard}>
            <h4 className={styles.cardTitle}>Total Value</h4>
            <h2 className={styles.cardValue}>$32,146.90</h2>
            <span className={styles.positiveChange}>+1.8% (24h)</span>
          </GlassyCard>

          <GlassyCard darker className={styles.summaryCard}>
            <h4 className={styles.cardTitle}>Asset Count</h4>
            <h2 className={styles.cardValue}>4</h2>
            <span>Across 3 blockchains</span>
          </GlassyCard>

          <GlassyCard darker className={styles.summaryCard}>
            <h4 className={styles.cardTitle}>Allocation</h4>
            <div className={styles.allocationList}>
              <div className={styles.allocationItem}>
                <span>Bitcoin</span>
                <span>63.9%</span>
              </div>
              <div className={styles.allocationItem}>
                <span>Ethereum</span>
                <span>25.2%</span>
              </div>
              <div className={styles.allocationItem}>
                <span>Stablecoins</span>
                <span>10.9%</span>
              </div>
            </div>
          </GlassyCard>
        </div>
      </GlassyCard>
    </PageLayout>
  );
}
```

#### Step 2: Refactor DEX Page

```jsx
// src/pages/dex.js
import React, { useState } from 'react';
import { useNavigation } from '../hooks/useNavigation';
import PageLayout from '../components/PageLayout';
import PageHeader from '../components/PageHeader';
import GlassyCard from '../components/GlassyCard';
import DataTable from '../components/DataTable';
import styles from './dex.module.css';

export default function DEX() {
  const { activePage } = useNavigation('dex');
  const [searchQuery, setSearchQuery] = useState('');

  const handleSearch = (query) => {
    setSearchQuery(query);
  };

  // Example trading pairs data
  const tradingPairsData = [
    { pair: 'BTC/USDT', price: '$45,632.80', change: '+2.3%', volume: '$1.2B', high: '$46,120.50', low: '$44,890.20' },
    { pair: 'ETH/USDT', price: '$3,245.75', change: '-1.1%', volume: '$820M', high: '$3,310.25', low: '$3,180.90' },
    { pair: 'SOL/USDT', price: '$120.45', change: '+5.7%', volume: '$450M', high: '$122.80', low: '$115.20' },
    { pair: 'ADA/USDT', price: '$0.58', change: '+0.8%', volume: '$210M', high: '$0.59', low: '$0.57' },
  ];

  // Filter the data based on the search query
  const filteredTradingPairsData = tradingPairsData.filter((pair) =>
    pair.pair.toLowerCase().includes(searchQuery.toLowerCase())
  );

  // Table headers
  const tableHeaders = [
    { label: 'Trading Pair' },
    { label: 'Price', align: 'right' },
    { label: '24h Change', align: 'right' },
    { label: '24h Volume', align: 'right' },
    { label: '24h High', align: 'right' },
    { label: '24h Low', align: 'right' },
    { label: 'Actions', align: 'center' },
  ];

  // Render table row
  const renderTableRow = (pair, index) => (
    <tr className={styles.tableRow} key={index}>
      <td className={styles.tableCell}>
        <div className={styles.pairNameContainer}>
          <div className={styles.pairIcon}>
            {pair.pair.split('/')[0].charAt(0)}
          </div>
          <div>{pair.pair}</div>
        </div>
      </td>
      <td className={`${styles.tableCell} ${styles.textRight}`}>{pair.price}</td>
      <td className={`${styles.tableCell} ${styles.textRight} ${pair.change.startsWith('+') ? styles.positiveChange : styles.negativeChange}`}>{pair.change}</td>
      <td className={`${styles.tableCell} ${styles.textRight}`}>{pair.volume}</td>
      <td className={`${styles.tableCell} ${styles.textRight}`}>{pair.high}</td>
      <td className={`${styles.tableCell} ${styles.textRight}`}>{pair.low}</td>
      <td className={`${styles.tableCell} ${styles.textCenter}`}>
        <div className={styles.actionButtons}>
          <button className={`${styles.actionButton} ${styles.primary}`}>Trade</button>
          <button className={`${styles.actionButton} ${styles.secondary}`}>Details</button>
        </div>
      </td>
    </tr>
  );

  return (
    <PageLayout 
      activePage={activePage} 
      pageTitle="Decentralized Exchange" 
      onSearch={handleSearch}
    >
      <PageHeader 
        title="Decentralized Exchange (DEX)" 
        subtitle="Trade digital assets directly from your wallet" 
      />

      {/* Market Overview */}
      <GlassyCard title="Market Overview" darker className={styles.marketOverview}>
        <div className={styles.marketStats}>
          <div className={styles.statItem}>
            <h4 className={styles.statLabel}>24h Volume</h4>
            <h3 className={styles.statValue}>$2.68B</h3>
            <span className={styles.positiveChange}>+12.5%</span>
          </div>
          <div className={styles.statItem}>
            <h4 className={styles.statLabel}>Active Pairs</h4>
            <h3 className={styles.statValue}>24</h3>
            <span>+2 new</span>
          </div>
          <div className={styles.statItem}>
            <h4 className={styles.statLabel}>Liquidity</h4>
            <h3 className={styles.statValue}>$890M</h3>
            <span className={styles.positiveChange}>+5.2%</span>
          </div>
          <div className={styles.statItem}>
            <h4 className={styles.statLabel}>Transactions</h4>
            <h3 className={styles.statValue}>15,432</h3>
            <span className={styles.positiveChange}>+8.7%</span>
          </div>
        </div>
      </GlassyCard>

      {/* Trading Pairs */}
      <GlassyCard title="Trading Pairs" darker className={styles.tradingPairsContainer}>
        <div className={styles.filterControls}>
          <select className={`${styles.filterSelect} ${styles.glassyContainer}`}>
            <option value="all">All Pairs</option>
            <option value="btc">BTC Pairs</option>
            <option value="eth">ETH Pairs</option>
            <option value="usdt">USDT Pairs</option>
          </select>

          <button className={styles.addPairButton}>
            Add Liquidity
          </button>
        </div>

        <DataTable 
          headers={tableHeaders} 
          data={filteredTradingPairsData} 
          renderRow={renderTableRow} 
        />
      </GlassyCard>
    </PageLayout>
  );
}
```

#### Step 3: Refactor Other Pages

Apply the same pattern as above to all other pages in the application.


### Phase 6. Testing Strategy

1. **Component Testing**:
   - Test each shared component individually
   - Ensure components render correctly with different props

2. **Integration Testing**:
   - Test navigation between pages
   - Test search functionality across pages
   - Test responsive design

3. **Visual Testing**:
   - Compare pages to ensure consistent styling
   - Verify that all pages match the inventory.js design

### Phase 7. Maintenance Plan

1. **Documentation**:
   - Create a UI component library documentation
   - Document all shared components and their usage

2. **Style Guide**:
   - Create a style guide for future development
   - Define standards for adding new components or pages

3. **Code Reviews**:
   - Implement strict code reviews for UI changes
   - Ensure all new pages follow the established patterns

## Conclusion

This implementation plan provides a comprehensive approach to standardizing the UI across all pages in the AltGUI application. By creating shared components, consistent styling, and a unified navigation system, we can ensure a cohesive user experience throughout the application.

The inventory.js page will serve as the source of truth for styling and navigation, and all other pages will be refactored to match this standard. This approach will not only improve the user experience but also make the codebase more maintainable and easier to extend in the future.

---

<div class="footer-links">


© 2025 KNIRV Network
</div>
