import React, { useState, useMemo } from 'react';
import { useNavigation } from '../hooks/useNavigation';
import PageLayout from '../components/PageLayout';
import PageHeader from '../components/PageHeader';
import GlassyCard from '../components/GlassyCard';
import RoleSwitcher from '../components/RoleSwitcher';
import { useRole } from '../contexts/RoleContext';
import styles from './auth-test.module.css';

const ROLES = ['Root', 'Bootnode', 'Dev', 'General'];

// Master page list — rows in the permission matrix.
// isSection rows are non-interactive visual dividers.
// isGroup marks a collapsible nav group toggle (e.g. "Environments").
// indent marks a child page that lives inside a group.
const ALL_PAGES = [
  { id: 'dashboard', label: 'Dashboard' },

  { id: '__sec_env', label: 'Environments', isSection: true },
  { id: 'environments', label: 'Environments (group)', isGroup: true },
  { id: 'dve-list',      label: 'My DVEs',            indent: true },
  { id: 'models',        label: 'DVE Overview',        indent: true },
  { id: 'codex-builder', label: 'DVE Codex Builder',   indent: true },
  { id: 'models-dex',    label: 'DVE DEX',             indent: true },

  { id: '__sec_quick', label: 'Quick Access', isSection: true },
  { id: 'arena',             label: 'KNIRVARENA' },
  { id: 'controller-status', label: 'KNIRVCONTROLLER Status' },
  { id: 'qr-connect',        label: 'QR Connect' },
  { id: 'my-endpoints',      label: 'My API Endpoints' },
  { id: 'payment-gateway',   label: 'Payment Gateway' },

  { id: '__sec_monitor', label: 'Monitor', isSection: true },
  { id: 'monitor',            label: 'Monitor (group)',    isGroup: true },
  { id: 'network-monitor',    label: 'Network Monitor',   indent: true },
  { id: 'graph-explorer',     label: 'Graph Explorer',    indent: true },
  { id: 'chain-explorer',     label: 'Chain Explorer',    indent: true },
  { id: 'chain-explorer-new', label: 'Chain Explorer (New)', indent: true },
  { id: 'oracle-explorer',    label: 'Oracle Explorer',   indent: true },
  { id: 'peers',              label: 'Peers',             indent: true },
  { id: 'operator-registry',  label: 'Operator Registry', indent: true },
  { id: 'tunnel-registry',    label: 'Tunnel Registry',   indent: true },
  { id: 'error-explorer',     label: 'Error Explorer',    indent: true },
  { id: 'transaction-explorer', label: 'Transactions' },
  { id: 'validation-explorer',  label: 'Validations' },

  { id: '__sec_governance', label: 'Governance', isSection: true },
  { id: 'governance',            label: 'Governance (group)',      isGroup: true },
  { id: 'bootnode-dao',          label: 'Bootnode DAO',           indent: true },
  { id: 'network-inference-dao', label: 'Network Inference DAO',  indent: true },

  { id: '__sec_marketplace', label: 'Marketplace', isSection: true },
  { id: 'marketplace',  label: 'Marketplace (group)', isGroup: true },
  { id: 'skills',       label: 'Skills',              indent: true },
  { id: 'capabilities', label: 'Capabilities',        indent: true },
  { id: 'properties',   label: 'Properties',          indent: true },
  { id: 'settlement',   label: 'Settlement',          indent: true },

  { id: '__sec_graphchain', label: 'Graphchain', isSection: true },
  { id: 'graphchain',           label: 'Graphchain' },
  { id: 'graphchain-dashboard', label: 'Graphchain Dashboard' },
  { id: 'graphchain-errors',    label: 'Graphchain Errors' },
  { id: 'graphchain-skills',    label: 'Graphchain Skills' },

  { id: '__sec_my', label: 'My Items', isSection: true },
  { id: 'my-models',            label: 'My Models' },
  { id: 'my-wallets',           label: 'My Wallets' },
  { id: 'my-skills',            label: 'My Skills' },
  { id: 'my-capabilities',      label: 'My Capabilities' },
  { id: 'my-properties',        label: 'My Properties' },
  { id: 'nft-property-explorer', label: 'NFT Property Explorer' },
  { id: 'nft-capability-manager', label: 'NFT Capability Manager' },
  { id: 'add-capability',       label: 'Add Capability' },

  { id: '__sec_tools', label: 'Tools & Finance', isSection: true },
  { id: 'tools',     label: 'Tools' },
  { id: 'basic',     label: 'Basic' },
  { id: 'advanced',  label: 'Advanced' },
  { id: 'explorer',  label: 'Explorer' },
  { id: 'inventory', label: 'Inventory' },
  { id: 'blockchain', label: 'Blockchain' },
  { id: 'dex',       label: 'DEX' },
  { id: 'daos',      label: 'DAOs' },

  { id: '__sec_admin', label: 'Admin', isSection: true },
  { id: 'settings',      label: 'Settings' },
  { id: 'network-admin', label: 'Network Admin' },
  { id: 'auth-test',     label: 'Role Permissions (this page)' },
];

const INTERACTIVE_PAGES = ALL_PAGES.filter(p => !p.isSection);

export default function AuthTest() {
  const { activePage } = useNavigation('auth-test');
  const { role: currentRole, pageAccess, updatePageAccess, defaultPageAccess } = useRole();

  // Local draft: map of role → Set<pageId>
  const [draft, setDraft] = useState(() => {
    const init = {};
    for (const r of ROLES) init[r] = new Set(pageAccess[r] || []);
    return init;
  });

  const [saving, setSaving] = useState(false);
  const [dirty, setDirty] = useState(false);

  const toggle = (pageId, r) => {
    setDraft(prev => {
      const next = { ...prev, [r]: new Set(prev[r]) };
      next[r].has(pageId) ? next[r].delete(pageId) : next[r].add(pageId);
      return next;
    });
    setDirty(true);
  };

  const toggleAll = (r) => {
    const allIds = INTERACTIVE_PAGES.map(p => p.id);
    const isFull = allIds.every(id => draft[r].has(id));
    setDraft(prev => ({
      ...prev,
      [r]: isFull ? new Set() : new Set(allIds),
    }));
    setDirty(true);
  };

  const resetToDefaults = () => {
    const reset = {};
    for (const r of ROLES) reset[r] = new Set(defaultPageAccess[r] || []);
    setDraft(reset);
    setDirty(true);
  };

  const handleSave = () => {
    const newAccess = {};
    for (const r of ROLES) newAccess[r] = Array.from(draft[r]);
    updatePageAccess(newAccess);
    setSaving(true);
    setTimeout(() => window.location.reload(), 900);
  };

  // Count per role for the column header
  const counts = useMemo(() =>
    Object.fromEntries(ROLES.map(r => [r, draft[r].size])),
    [draft]
  );

  return (
    <PageLayout activePage={activePage} pageTitle="Role Permissions">
      <PageHeader
        title="Role Access Configuration"
        subtitle="Check which pages each role can see. Changes take effect after Save & Apply."
      />

      {/* Role switcher */}
      <GlassyCard title="Switch Active Role">
        <p className={styles.roleSwitcherHint}>
          Change your session role here to preview how the navigation looks after saving.
        </p>
        <RoleSwitcher />
      </GlassyCard>

      {/* Action bar */}
      <div className={styles.actionBar}>
        <button className={styles.resetBtn} onClick={resetToDefaults} disabled={saving}>
          ↺ Reset to Defaults
        </button>
        <div className={styles.actionRight}>
          {saving && <span className={styles.savingMsg}>✓ Saved — reloading…</span>}
          <button
            className={`${styles.saveBtn} ${dirty && !saving ? styles.saveBtnDirty : ''}`}
            onClick={handleSave}
            disabled={saving || !dirty}
          >
            Save &amp; Apply
          </button>
        </div>
      </div>

      {/* Permission matrix */}
      <GlassyCard>
        <div className={styles.tableWrap}>
          <table className={styles.matrix}>
            <thead>
              <tr>
                <th className={styles.pageCol}>Page / Feature</th>
                {ROLES.map(r => (
                  <th key={r} className={`${styles.roleCol} ${styles['role_' + r.toLowerCase()]}`}>
                    <div className={styles.roleColName}>
                      {r}
                      {r === currentRole && <span className={styles.youBadge}>you</span>}
                    </div>
                    <div className={styles.roleColCount}>{counts[r]} pages</div>
                    <button className={styles.toggleAllBtn} onClick={() => toggleAll(r)}>
                      {INTERACTIVE_PAGES.every(p => draft[r].has(p.id)) ? 'Clear all' : 'Select all'}
                    </button>
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {ALL_PAGES.map(page => {
                if (page.isSection) {
                  return (
                    <tr key={page.id} className={styles.sectionRow}>
                      <td colSpan={5} className={styles.sectionCell}>
                        {page.label}
                      </td>
                    </tr>
                  );
                }
                return (
                  <tr key={page.id} className={`${styles.pageRow} ${page.isGroup ? styles.groupRow : ''}`}>
                    <td className={`${styles.pageLabel} ${page.indent ? styles.indented : ''}`}>
                      {page.indent && <span className={styles.treeLine}>└</span>}
                      <span className={styles.pageName}>{page.label}</span>
                      <span className={styles.pageId}>{page.id}</span>
                    </td>
                    {ROLES.map(r => {
                      const checked = draft[r].has(page.id);
                      return (
                        <td key={r} className={`${styles.checkCell} ${checked ? styles.checkedCell : ''}`}>
                          <input
                            type="checkbox"
                            className={styles.checkbox}
                            checked={checked}
                            onChange={() => toggle(page.id, r)}
                          />
                        </td>
                      );
                    })}
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      </GlassyCard>
    </PageLayout>
  );
}
