import React, { useState } from 'react';
import type { BuiltinPluginInfo } from '../../hooks/use-builtin-plugins';

interface BuiltinPluginCardProps {
  plugin: BuiltinPluginInfo;
  onToggle: (id: string, enable: boolean) => Promise<void>;
  onUpdateConfig?: (id: string, patch: Record<string, unknown>) => Promise<void>;
}

interface FintechFeatureFlags {
  enable_kyc: boolean;
  enable_aml: boolean;
  enable_sec: boolean;
  enable_basel: boolean;
  enable_scenarios: boolean;
}

const FINTECH_PLUGIN_ID = 'fintech-validator';

const defaultFintechFlags: FintechFeatureFlags = {
  enable_kyc: false,
  enable_aml: false,
  enable_sec: false,
  enable_basel: false,
  enable_scenarios: false,
};

export function BuiltinPluginCard({ plugin, onToggle, onUpdateConfig }: BuiltinPluginCardProps) {
  const [toggling, setToggling] = useState(false);
  const [fintechFlags, setFintechFlags] = useState<FintechFeatureFlags>(defaultFintechFlags);
  const [updatingFlag, setUpdatingFlag] = useState<string | null>(null);

  const handleToggle = async () => {
    setToggling(true);
    try {
      await onToggle(plugin.id, !plugin.enabled);
    } finally {
      setToggling(false);
    }
  };

  const handleFintechFlag = async (flag: keyof FintechFeatureFlags, value: boolean) => {
    if (!onUpdateConfig) return;
    setUpdatingFlag(flag);
    try {
      const newFlags = { ...fintechFlags, [flag]: value };
      setFintechFlags(newFlags);
      await onUpdateConfig(plugin.id, { [flag]: value });
    } catch {
      // Revert on failure
      setFintechFlags(prev => ({ ...prev, [flag]: !value }));
    } finally {
      setUpdatingFlag(null);
    }
  };

  const categoryColor: Record<string, string> = {
    compliance: '#4CAF50',
    security: '#F44336',
    analytics: '#2196F3',
    payments: '#FF9800',
  };

  const badgeStyle: React.CSSProperties = {
    display: 'inline-block',
    padding: '2px 8px',
    borderRadius: '12px',
    fontSize: '11px',
    fontWeight: 600,
    color: '#fff',
    backgroundColor: categoryColor[plugin.category] ?? '#9E9E9E',
    marginLeft: '8px',
  };

  const cardStyle: React.CSSProperties = {
    border: '1px solid #e0e0e0',
    borderRadius: '8px',
    padding: '16px',
    marginBottom: '12px',
    backgroundColor: plugin.running ? '#f9fff9' : '#fafafa',
    opacity: toggling ? 0.7 : 1,
    transition: 'opacity 0.2s',
  };

  const toggleButtonStyle: React.CSSProperties = {
    padding: '6px 16px',
    borderRadius: '4px',
    border: 'none',
    cursor: toggling ? 'not-allowed' : 'pointer',
    backgroundColor: plugin.enabled ? '#F44336' : '#4CAF50',
    color: '#fff',
    fontWeight: 600,
    fontSize: '13px',
  };

  const featureToggleStyle = (active: boolean, disabled: boolean): React.CSSProperties => ({
    display: 'inline-flex',
    alignItems: 'center',
    gap: '6px',
    padding: '4px 10px',
    margin: '3px',
    borderRadius: '4px',
    border: '1px solid',
    borderColor: active ? '#4CAF50' : '#ccc',
    backgroundColor: active ? '#E8F5E9' : '#fff',
    cursor: disabled ? 'not-allowed' : 'pointer',
    fontSize: '12px',
    opacity: disabled ? 0.6 : 1,
  });

  return (
    <div style={cardStyle}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
        <div>
          <span style={{ fontWeight: 700, fontSize: '15px' }}>{plugin.name}</span>
          <span style={badgeStyle}>{plugin.category}</span>
          <span style={{ marginLeft: '8px', fontSize: '11px', color: '#888' }}>v{plugin.version}</span>
        </div>
        <button
          style={toggleButtonStyle}
          onClick={handleToggle}
          disabled={toggling}
          title={plugin.enabled ? 'Disable plugin' : 'Enable plugin'}
        >
          {toggling ? 'Working...' : plugin.enabled ? 'Disable' : 'Enable'}
        </button>
      </div>

      <p style={{ margin: '8px 0 4px', fontSize: '13px', color: '#555' }}>{plugin.description}</p>

      <div style={{ fontSize: '12px', color: '#888', marginBottom: '8px' }}>
        Status:{' '}
        <span style={{ color: plugin.running ? '#4CAF50' : '#F44336', fontWeight: 600 }}>
          {plugin.running ? 'Running' : 'Stopped'}
        </span>
      </div>

      {/* Fintech-specific sub-feature toggles */}
      {plugin.id === FINTECH_PLUGIN_ID && onUpdateConfig && (
        <div style={{ marginTop: '8px' }}>
          <div style={{ fontSize: '12px', fontWeight: 600, marginBottom: '4px', color: '#555' }}>
            Compliance Features
          </div>
          {(Object.keys(defaultFintechFlags) as Array<keyof FintechFeatureFlags>).map(flag => {
            const label = flag.replace('enable_', '').toUpperCase();
            const active = fintechFlags[flag];
            const disabled = updatingFlag === flag || !plugin.running;
            return (
              <button
                key={flag}
                style={featureToggleStyle(active, disabled)}
                disabled={disabled}
                onClick={() => handleFintechFlag(flag, !active)}
              >
                <span style={{ fontSize: '14px' }}>{active ? '✓' : '○'}</span>
                {label}
              </button>
            );
          })}
        </div>
      )}
    </div>
  );
}

export default BuiltinPluginCard;
