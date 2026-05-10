import React, { useState, useMemo } from 'react';
import styles from './SideNavigation.module.css';
import { useNavigation } from '../hooks/useNavigation';
import { useRole } from '../contexts/RoleContext';
import IframeModal from './IframeModal';

// ─── 4-Chain Navigation Structure ──────────────────────────────────────────
//
// KNIRVCHAIN        — Decisions, Capabilities, Rules, Resolutions, Properties
// Transaction Chain — Fast tx, Payments, DVE registration, Rollup source
// Validation Chain  — Immutable signoff, Policy commits, Evidence, Audit
// Oracle            — NRN settlement, Wallets, Rollup finalization, Governance

const NAV_ITEMS = [
  // ── Dashboard ──────────────────────────────────────────────────────────
  { id: 'dashboard', label: 'Dashboard', icon: '🏠' },

  // ── Quick Access ────────────────────────────────────────────────────────
  { id: 'controller-status', label: 'Controller Status', icon: '🔌' },
  { id: 'qr-connect', label: 'QR Connect', icon: '📱' },
  { id: 'my-endpoints', label: 'My API Endpoints', icon: '🔗' },

  // ── KNIRVCHAIN — Decision & Resolution Layer ────────────────────────────
  { id: 'knirvchain', label: 'KNIRVCHAIN', icon: '🧠', children: [
    { id: 'graph-explorer', label: 'Decisions (Agent Traces)', icon: '🔗' },
    { id: 'capabilities', label: 'Capabilities', icon: '🔌' },
    { id: 'chain-explorer', label: 'Rules', icon: '⛓️' },
    { id: 'chain-explorer-new', label: 'Resolutions', icon: '✅' },
    { id: 'nft-property-explorer', label: 'Properties', icon: '🏷️' },
    { id: 'skills', label: 'Skill Registry', icon: '🛠️' },
  ]},

  // ── Transaction Chain — Fast Execution Layer ────────────────────────────
  { id: 'txchain', label: 'Transaction Chain', icon: '⚡', children: [
    { id: 'network-monitor', label: 'Transaction Monitor', icon: '🌐' },
    { id: 'payment-gateway', label: 'Payment Verification', icon: '💳' },
    { id: 'marketplace', label: 'DVE Registration', icon: '🛒' },
    { id: 'oracle-explorer', label: 'Rollup Source', icon: '🔮' },
    { id: 'settlement', label: 'Settlement', icon: '📝' },
  ]},

  // ── Validation Chain — Immutable Audit Layer ────────────────────────────
  { id: 'valchain', label: 'Validation Chain', icon: '🛡️', children: [
    { id: 'graphchain-dashboard', label: 'Evidence Anchoring', icon: '📊' },
    { id: 'graphchain-skills', label: 'Policy Commits', icon: '⚡' },
    { id: 'graphchain-errors', label: 'Audit Ledger', icon: '🚨' },
    { id: 'error-explorer', label: 'Signoff Explorer', icon: '🔍' },
  ]},

  // ── Oracle — Economics & Governance ─────────────────────────────────────
  { id: 'oracle', label: 'Oracle', icon: '🏦', children: [
    { id: 'models-dex', label: 'NRN Settlement', icon: '💱' },
    { id: 'my-wallets', label: 'Wallet Authority', icon: '💰' },
    { id: 'bootnode-dao', label: 'Governance', icon: '🗳️' },
    { id: 'network-inference-dao', label: 'Cross-Chain Control', icon: '📜' },
    { id: 'models', label: 'Economics', icon: '🤖' },
  ]},

  // ── Vault — Personal Assets ─────────────────────────────────────────────
  { id: 'vault', label: 'Vault', icon: '🔒', children: [
    { id: 'my-models', label: 'My Models', icon: '🤖' },
    { id: 'my-skills', label: 'My Skills', icon: '⚡' },
    { id: 'my-capabilities', label: 'My Capabilities', icon: '🔌' },
    { id: 'my-properties', label: 'My Properties', icon: '🏷️' },
  ]},

  // ── Tools & Settings ────────────────────────────────────────────────────
  { id: 'codex-builder', label: 'Codex Builder', icon: '🛠️' },
  { id: 'settings', label: 'Settings', icon: '⚙️' },
  { id: 'network-admin', label: 'Network Admin', icon: '👑' },
  { id: 'auth-test', label: 'Auth Testing', icon: '🔐' },

  // ── Infrastructure (Monitor sub-items) ──────────────────────────────────
  { id: 'monitor', label: 'Infrastructure', icon: '📊', children: [
    { id: 'peers', label: 'Peers', icon: '👥' },
    { id: 'operator-registry', label: 'Operator Registry', icon: '🧾' },
    { id: 'tunnel-registry', label: 'Tunnel Registry', icon: '🛰️' },
  ]},
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

    if (['my-models','my-wallets','my-skills','my-capabilities','my-properties','nft-property-explorer'].includes(id)) {
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
