import React, { useState } from 'react';
import styles from './SideNavigation.module.css';
import { useNavigation } from '../hooks/useNavigation';
import { useRole } from '../contexts/RoleContext';

const SideNavigation = ({ activePage }) => {
  const { handleNavigation } = useNavigation(activePage);
  const { canAccess, role } = useRole();
  const [expandedSections, setExpandedSections] = useState({});

  const toggleSection = (sectionId) => {
    setExpandedSections(prev => ({
      ...prev,
      [sectionId]: !prev[sectionId]
    }));
  };
  
  // New navigation structure
  const navItems = [
    // Dashboard
    {
      id: 'dashboard',
      label: 'Dashboard',
      icon: '🏠',
    },
    // Monitor section
    {
      id: 'monitor',
      label: 'Monitor',
      icon: '📊',
      children: [
        {
          id: 'network-monitor',
          label: 'Network Monitor',
          icon: '🌐',
        },
        {
          id: 'local-analytics',
          label: 'Local Analytics',
          icon: '📈',
        },
        {
          id: 'graph-explorer',
          label: 'Graph Explorer',
          icon: '🔗',
        },
        {
          id: 'chain-explorer',
          label: 'Chain Explorer',
          icon: '⛓️',
        },
        {
          id: 'oracle-explorer',
          label: 'Oracle Explorer',
          icon: '🔮',
        },
        {
          id: 'error-explorer',
          label: 'Error Explorer',
          icon: '🚨',
        }
      ]
    },
    // Models section
    {
      id: 'models',
      label: 'Models',
      icon: '🤖',
      children: [
        {
          id: 'codex-builder',
          label: 'Codex Builder',
          icon: '🛠️',
        },
        {
          id: 'models-dex',
          label: 'Models DEX',
          icon: '💱',
        },
        {
          id: 'knirvinference-dao',
          label: 'KNIRVINFERENCE DAO',
          icon: '🏛️',
        }
      ]
    },
    // Marketplace section
    {
      id: 'marketplace',
      label: 'Marketplace',
      icon: '🛒',
      children: [
        {
          id: 'skills',
          label: 'Skills',
          icon: '⚡',
        },
        {
          id: 'capabilities',
          label: 'Capabilities',
          icon: '🔌',
        },
        {
          id: 'properties',
          label: 'Properties',
          icon: '🏷️',
        },
        {
          id: 'settlement',
          label: 'Settlement',
          icon: '📝',
        }
      ]
    },
    // Vault section
    {
      id: 'vault',
      label: 'Vault',
      icon: '🔒',
      children: [
        {
          id: 'my-models',
          label: 'My Models',
          icon: '🤖',
        },
        {
          id: 'my-wallets',
          label: 'My Wallets',
          icon: '💰',
        },
        {
          id: 'my-skills',
          label: 'My Skills',
          icon: '⚡',
        },
        {
          id: 'my-capabilities',
          label: 'My Capabilities',
          icon: '🔌',
        },
        {
          id: 'my-properties',
          label: 'My Properties',
          icon: '🏷️',
        }
      ]
    },
    // Settings
    {
      id: 'settings',
      label: 'Settings',
      icon: '⚙️',
    },
    // Network Admin (Root role only)
    {
      id: 'network-admin',
      label: 'Network Admin',
      icon: '👑',
    },
    // Auth Testing (Root role only)
    {
      id: 'auth-test',
      label: 'Auth Testing',
      icon: '🔐',
    }
  ];

  const renderNavItem = (item) => {
    if (!canAccess(item.id)) return null;

    const hasChildren = item.children && item.children.length > 0;
    const isExpanded = expandedSections[item.id];
    const isActive = activePage === item.id;

    return (
      <div key={item.id}>
        <div
          onClick={() => hasChildren ? toggleSection(item.id) : handleNavigation(item.id)}
          className={`${styles.navItem} ${styles.glassyContainer} ${
            isActive ? styles.active : styles.inactive
          }`}
        >
          <span className={styles.navIcon}>{item.icon}</span>
          <span>{item.label}</span>
          {hasChildren && (
            <span className={styles.expandIcon}>
              {isExpanded ? '▼' : '▶'}
            </span>
          )}
        </div>

        {hasChildren && isExpanded && (
          <div className={styles.subNavItems}>
            {item.children.map(child => (
              canAccess(child.id) && (
                <div
                  key={child.id}
                  onClick={() => handleNavigation(child.id)}
                  className={`${styles.subNavItem} ${styles.glassyContainer} ${
                    activePage === child.id ? styles.active : styles.inactive
                  }`}
                >
                  <span className={styles.navIcon}>{child.icon}</span>
                  <span>{child.label}</span>
                </div>
              )
            ))}
          </div>
        )}
      </div>
    );
  };

  return (
    <div className={styles.sidebar}>
      <h2 className={styles.dashboardTitle}>KNIRV Network</h2>
      <div className={styles.roleIndicator}>
        Role: <span className={styles.roleBadge}>{role}</span>
      </div>

      {navItems.map(item => renderNavItem(item))}
    </div>
  );
};

export default SideNavigation;