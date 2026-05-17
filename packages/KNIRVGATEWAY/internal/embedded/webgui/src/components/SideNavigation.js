import React, { useState, useMemo } from 'react';
import styles from './SideNavigation.module.css';
import { useNavigation } from '../hooks/useNavigation';
import { useRole } from '../contexts/RoleContext';
import IframeModal from './IframeModal';

const NAV_ITEMS = [
  // Dashboard
  { id: 'dashboard', label: 'Dashboard', icon: '🏠' },
  // Quick Access
  { id: 'controller-status', label: 'KNIRVCONTROLLER Status', icon: '🔌' },
  { id: 'qr-connect', label: 'QR Connect', icon: '📱' },
  { id: 'my-endpoints', label: 'My API Endpoints', icon: '🔗' },
  { id: 'payment-gateway', label: 'Payment Gateway', icon: '💳' },
  // Monitor section
  { id: 'monitor', label: 'Monitor', icon: '📊', children: [
    { id: 'network-monitor', label: 'Network Monitor', icon: '🌐' },
    { id: 'graph-explorer', label: 'Graph Explorer', icon: '🔗' },
    { id: 'chain-explorer', label: 'Chain Explorer', icon: '⛓️' },
    { id: 'oracle-explorer', label: 'Oracle Explorer', icon: '🔮' },
    { id: 'peers', label: 'Peers', icon: '👥' },
    { id: 'operator-registry', label: 'Operator Registry', icon: '🧾' },
    { id: 'tunnel-registry', label: 'Tunnel Registry', icon: '🛰️' },
    { id: 'error-explorer', label: 'Error Explorer', icon: '🚨' },
  ]},
  // Chain Explorers
  { id: 'transaction-explorer', label: 'Transactions', icon: '⚡' },
  { id: 'validation-explorer', label: 'Validations', icon: '🛡️' },
  // Models section
  { id: 'models', label: 'Models', icon: '🤖', children: [
    { id: 'models', label: 'Models Overview', icon: '🤖' },
    { id: 'codex-builder', label: 'Codex Builder', icon: '🛠️' },
    { id: 'models-dex', label: 'Models DEX', icon: '💱' },
  ]},
  // Governance section
  { id: 'governance', label: 'Governance', icon: '🏛️', children: [
    { id: 'bootnode-dao', label: 'Bootnode DAO', icon: '🗳️' },
    { id: 'network-inference-dao', label: 'Network Inference DAO', icon: '📜' },
  ]},
  // Marketplace section
  { id: 'marketplace', label: 'Marketplace', icon: '🛒', children: [
    { id: 'skills', label: 'Skills', icon: '⚡' },
    { id: 'capabilities', label: 'Capabilities', icon: '🔌' },
    { id: 'properties', label: 'Properties', icon: '🏷️' },
    { id: 'settlement', label: 'Settlement', icon: '📝' },
  ]},
  // Vault section
  { id: 'vault', label: 'Vault', icon: '🔒', children: [
    { id: 'my-models', label: 'My Models', icon: '🤖' },
    { id: 'my-wallets', label: 'My Wallets', icon: '💰' },
    { id: 'my-skills', label: 'My Skills', icon: '⚡' },
    { id: 'my-capabilities', label: 'My Capabilities', icon: '🔌' },
    { id: 'my-properties', label: 'My Properties', icon: '🏷️' },
    { id: 'my-badges', label: 'Badges', icon: '🏅' },
    { id: 'nft-property-explorer', label: 'NFT Property Explorer', icon: '🎨' },
  ]},
  // Settings
  { id: 'settings', label: 'Settings', icon: '⚙️' },
  // Network Admin (Root role only)
  { id: 'network-admin', label: 'Network Admin', icon: '👑' },
  // Auth Testing (Root role only)
  { id: 'auth-test', label: 'Auth Testing', icon: '🔐' },
];

const SideNavigation = ({ activePage }) => {
  const { handleNavigation } = useNavigation(activePage);
  const { canAccess, role } = useRole();

  const [manualSections, setManualSections] = useState({});

  const expandedSections = useMemo(() => {
    const expanded = { ...manualSections };
    for (const item of NAV_ITEMS) {
      if (item.children) {
        for (const child of item.children) {
          if (child.id === activePage) {
            expanded[item.id] = true;
            break;
          }
        }
      }
    }
    return expanded;
  }, [activePage, manualSections]);

  const toggleSection = (sectionId) => {
    setManualSections(prev => ({
      ...prev,
      [sectionId]: !expandedSections[sectionId]
    }));
  };

  const [modal, setModal] = useState({ open: false, title: '', src: '' });
  const openModal = (title, src) => setModal({ open: true, title, src });
  const closeModal = () => setModal({ open: false, title: '', src: '' });

  const handleItemClick = async (id, hasChildren) => {
    if (hasChildren) return toggleSection(id);

    if (id === 'payment-gateway' || id === 'operator-registry' || id === 'tunnel-registry') {
      return handleNavigation(id);
    }

    if (id === 'my-endpoints') {
      try {
        const r = await fetch('/session/controller', { credentials: 'include' });
        const j = await r.json();
        if (j && j.controllerUrl) {
          const base = (typeof window !== 'undefined' && window.__GATEWAY_BASE__) || '';
          const url = base ? `${base.replace(/\/+$/, '')}/controller` : '/controller';
          window.open(url, '_blank', 'noopener');
          return;
        }
        alert('No controller connected. Use QR Connect to link your controller.');
        return handleNavigation('qr-connect');
      } catch (_) {
        alert('Unable to check controller session.');
        return;
      }
    }

    if (['my-models','my-wallets','my-skills','my-capabilities','my-properties','my-badges','nft-property-explorer'].includes(id)) {
      return handleNavigation(id);
    }

    return handleNavigation(id);
  };

  const renderNavItem = (item) => {
    if (!canAccess(item.id)) return null;

    const hasChildren = item.children && item.children.length > 0;
    const isExpanded = expandedSections[item.id];
    const isActive = activePage === item.id;

    return (
      <div key={item.id}>
        <div
          onClick={() => handleItemClick(item.id, hasChildren)}
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
                  onClick={() => handleItemClick(child.id, false)}
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
      <h2 className={styles.dashboardTitle}>KNIRV Network Oracle</h2>
      <div className={styles.roleIndicator}>
        Role: <span className={styles.roleBadge}>{role}</span>
      </div>

      {NAV_ITEMS.map(item => renderNavItem(item))}

      {modal.open && (
        <IframeModal title={modal.title} src={modal.src} onClose={closeModal} />
      )}
    </div>
  );
};

export default SideNavigation;
