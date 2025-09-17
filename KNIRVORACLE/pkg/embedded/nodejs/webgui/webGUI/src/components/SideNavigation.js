import React from 'react';
import styles from './SideNavigation.module.css';
import { useNavigation } from '../hooks/useNavigation';
import { useRole } from '../contexts/RoleContext';

const SideNavigation = ({ activePage }) => {
  const { handleNavigation } = useNavigation(activePage);
  const { canAccess, role } = useRole();
  
  // Navigation items with their access requirements
  const navItems = [
    {
      id: 'inventory',
      label: 'Inventory',
      icon: '📦',
    },
    {
      id: 'vault',
      label: 'Vault',
      icon: '🔒',
    },
    {
      id: 'blockchain',
      label: 'Blockchain',
      icon: '⛓️',
    },
    {
      id: 'dex',
      label: 'DEX',
      icon: '💱',
    },
    {
      id: 'daos',
      label: 'DAOs',
      icon: '🏛️',
    },
    {
      id: 'nft-vault',
      label: 'NFT Vault',
      icon: '🖼️',
    },
    {
      id: 'nft-capability-manager',
      label: 'NFT Capability Manager',
      icon: '🔌',
    },
    {
      id: 'add-capability',
      label: 'Add Capability',
      icon: '➕',
    },
    {
      id: 'devs',
      label: 'Peers List',
      icon: '👥',
    },
    {
      id: 'settlement',
      label: 'Settlement',
      icon: '📝',
    },
    // New Network Admin page for Root only
    {
      id: 'network-admin',
      label: 'Network Admin',
      icon: '⚙️',
    }
  ];

  return (
    <div className={styles.sidebar}>
      <h2 className={styles.dashboardTitle}>Blockchain Dashboard</h2>
      <div className={styles.roleIndicator}>
        Role: <span className={styles.roleBadge}>{role}</span>
      </div>
      
      {navItems.map(item => (
        // Only render navigation items the current role can access
        canAccess(item.id) && (
          <div
            key={item.id}
            onClick={() => handleNavigation(item.id)}
            className={`${styles.navItem} ${styles.glassyContainer} ${
              activePage === item.id ? styles.active : styles.inactive
            }`}
          >
            <span className={styles.navIcon}>{item.icon}</span>
            <span>{item.label}</span>
          </div>
        )
      ))}
    </div>
  );
};

export default SideNavigation;