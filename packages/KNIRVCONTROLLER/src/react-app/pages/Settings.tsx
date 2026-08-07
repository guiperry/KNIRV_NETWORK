import { useEffect, useState } from 'react';
import { Shield, Lock, Bell, Sparkles, Trash2, Power, KeyRound, Globe } from 'lucide-react';
import Layout from '@/react-app/components/Layout';
import NetworkSelectorDialog from '@/react-app/components/NetworkSelectorDialog';
import { useVault } from '@/react-app/hooks/useVault';
import { useNetworkConfig } from '@/react-app/hooks/useNetworkConfig';
import { getNetworkOption } from '@/react-app/platform/networkConfig';

const SETTINGS_KEYS = {
  autoOpenVault: 'knirv.settings.autoOpenVault',
  badgeNotifications: 'knirv.settings.badgeNotifications',
  voiceHints: 'knirv.settings.voiceHints',
} as const;

function usePersistedToggle(storageKey: string, defaultValue: boolean) {
  const [value, setValue] = useState(() => {
    if (typeof window === 'undefined') return defaultValue;
    const stored = window.localStorage.getItem(storageKey);
    return stored === null ? defaultValue : stored === 'true';
  });

  useEffect(() => {
    window.localStorage.setItem(storageKey, String(value));
  }, [storageKey, value]);

  return [value, setValue] as const;
}

interface ToggleRowProps {
  title: string;
  description: string;
  enabled: boolean;
  onToggle: () => void;
}

function ToggleRow({ title, description, enabled, onToggle }: ToggleRowProps) {
  return (
    <button
      type="button"
      onClick={onToggle}
      className="flex w-full items-center justify-between gap-4 rounded-2xl border border-white/10 bg-white/5 p-4 text-left transition-colors hover:border-blue-500/30 hover:bg-white/[0.07]"
    >
      <div>
        <div className="text-sm font-semibold text-white">{title}</div>
        <div className="mt-1 text-xs leading-5 text-slate-400">{description}</div>
      </div>
      <div className={`flex h-7 w-12 items-center rounded-full p-1 transition-colors ${enabled ? 'bg-blue-600 justify-end' : 'bg-slate-700 justify-start'}`}>
        <div className="h-5 w-5 rounded-full bg-white shadow-sm" />
      </div>
    </button>
  );
}

export default function Settings() {
  const { status, oracleAddress, lockVault, clearVault } = useVault();
  const [autoOpenVault, setAutoOpenVault] = usePersistedToggle(SETTINGS_KEYS.autoOpenVault, true);
  const [badgeNotifications, setBadgeNotifications] = usePersistedToggle(SETTINGS_KEYS.badgeNotifications, true);
  const [voiceHints, setVoiceHints] = usePersistedToggle(SETTINGS_KEYS.voiceHints, false);
  const { networkId, serverUrl, devServerUrl, selectNetwork } = useNetworkConfig();
  const [networkDialogOpen, setNetworkDialogOpen] = useState(false);
  const activeNetworkOption = getNetworkOption(networkId);

  return (
    <Layout>
      <div className="p-4 pb-24 space-y-6">
        <div className="text-center py-4">
          <h2 className="text-2xl font-bold gradient-text mb-2">
            Settings
          </h2>
          <p className="text-slate-400 text-sm font-mono uppercase tracking-wider">
            Configure your controller vault and onboarding behavior
          </p>
        </div>

        <div className="grid gap-4 md:grid-cols-2">
          <div className="glass-panel p-5 space-y-3">
            <div className="flex items-center space-x-3">
              <div className="w-10 h-10 bg-blue-500/20 rounded-xl flex items-center justify-center border border-blue-500/20">
                <Shield className="w-5 h-5 text-blue-400" />
              </div>
              <div>
                <h3 className="text-lg font-semibold text-white">Vault Status</h3>
                <p className="text-xs text-slate-500 font-mono uppercase tracking-wider">Current session state</p>
              </div>
            </div>

            <div className="rounded-2xl border border-white/10 bg-slate-950/40 p-4 space-y-3">
              <div className="flex items-center justify-between">
                <span className="text-xs uppercase tracking-widest text-slate-500 font-mono">Status</span>
                <span className="text-sm font-mono text-blue-400 uppercase">{status}</span>
              </div>
              <div className="flex items-center justify-between gap-3">
                <span className="text-xs uppercase tracking-widest text-slate-500 font-mono">Vault Address</span>
                <span className="text-xs font-mono text-slate-300 truncate text-right max-w-[220px]">
                  {oracleAddress ?? 'Unavailable'}
                </span>
              </div>
            </div>
          </div>

          <div className="glass-panel p-5 space-y-3">
            <div className="flex items-center space-x-3">
              <div className="w-10 h-10 bg-emerald-500/20 rounded-xl flex items-center justify-center border border-emerald-500/20">
                <Sparkles className="w-5 h-5 text-emerald-400" />
              </div>
              <div>
                <h3 className="text-lg font-semibold text-white">Startup Behavior</h3>
                <p className="text-xs text-slate-500 font-mono uppercase tracking-wider">PWA and onboarding controls</p>
              </div>
            </div>

            <div className="space-y-3">
              <ToggleRow
                title="Auto-open Vault"
                description="Return directly to the vault after a successful identity setup."
                enabled={autoOpenVault}
                onToggle={() => setAutoOpenVault((current) => !current)}
              />
              <ToggleRow
                title="Badge Alerts"
                description="Surface badge state changes in the controller UI."
                enabled={badgeNotifications}
                onToggle={() => setBadgeNotifications((current) => !current)}
              />
              <ToggleRow
                title="Voice Hints"
                description="Enable spoken guidance and voice command hints."
                enabled={voiceHints}
                onToggle={() => setVoiceHints((current) => !current)}
              />
            </div>
          </div>
        </div>

        <div className="glass-panel p-5 space-y-3">
          <div className="flex items-center space-x-3">
            <div className="w-10 h-10 bg-indigo-500/20 rounded-xl flex items-center justify-center border border-indigo-500/20">
              <Globe className="w-5 h-5 text-indigo-400" />
            </div>
            <div>
              <h3 className="text-lg font-semibold text-white">Network</h3>
              <p className="text-xs text-slate-500 font-mono uppercase tracking-wider">KNIRVSERVER connection</p>
            </div>
          </div>

          <div className="rounded-2xl border border-white/10 bg-slate-950/40 p-4 space-y-3">
            <div className="flex items-center justify-between">
              <span className="text-xs uppercase tracking-widest text-slate-500 font-mono">Active Network</span>
              <span className="text-sm font-mono text-blue-400 uppercase">{activeNetworkOption.name}</span>
            </div>
            <div className="flex items-center justify-between gap-3">
              <span className="text-xs uppercase tracking-widest text-slate-500 font-mono">Server URL</span>
              <span className="text-xs font-mono text-slate-300 truncate text-right max-w-[220px]">{serverUrl}</span>
            </div>
          </div>

          <button
            type="button"
            onClick={() => setNetworkDialogOpen(true)}
            className="flex w-full items-center justify-center gap-2 rounded-xl border border-white/10 bg-white/5 px-4 py-3 text-sm font-semibold text-white transition-colors hover:border-blue-500/30 hover:bg-white/10"
          >
            <Globe className="h-4 w-4 text-blue-400" />
            Change Network
          </button>
        </div>

        <div className="glass-panel p-5 space-y-4">
          <div className="flex items-center space-x-3">
            <div className="w-10 h-10 bg-amber-500/20 rounded-xl flex items-center justify-center border border-amber-500/20">
              <Lock className="w-5 h-5 text-amber-400" />
            </div>
            <div>
              <h3 className="text-lg font-semibold text-white">Vault Controls</h3>
              <p className="text-xs text-slate-500 font-mono uppercase tracking-wider">Security actions</p>
            </div>
          </div>

          <div className="grid gap-3 md:grid-cols-3">
            <button
              type="button"
              onClick={lockVault}
              className="flex items-center justify-center gap-2 rounded-xl border border-white/10 bg-white/5 px-4 py-3 text-sm font-semibold text-white transition-colors hover:border-blue-500/30 hover:bg-white/10"
            >
              <Power className="h-4 w-4 text-blue-400" />
              Lock Vault
            </button>
            <button
              type="button"
              className="flex items-center justify-center gap-2 rounded-xl border border-white/10 bg-white/5 px-4 py-3 text-sm font-semibold text-white transition-colors hover:border-blue-500/30 hover:bg-white/10"
            >
              <KeyRound className="h-4 w-4 text-blue-400" />
              Rotate Session
            </button>
            <button
              type="button"
              onClick={clearVault}
              className="flex items-center justify-center gap-2 rounded-xl border border-red-500/20 bg-red-500/10 px-4 py-3 text-sm font-semibold text-red-300 transition-colors hover:border-red-500/40 hover:bg-red-500/20"
            >
              <Trash2 className="h-4 w-4" />
              Reset Vault
            </button>
          </div>
        </div>

        <div className="glass-panel p-5 space-y-3">
          <div className="flex items-center space-x-3">
            <div className="w-10 h-10 bg-cyan-500/20 rounded-xl flex items-center justify-center border border-cyan-500/20">
              <Bell className="w-5 h-5 text-cyan-400" />
            </div>
            <div>
              <h3 className="text-lg font-semibold text-white">Connection Notes</h3>
              <p className="text-xs text-slate-500 font-mono uppercase tracking-wider">Local testing guidance</p>
            </div>
          </div>

          <div className="rounded-2xl border border-white/10 bg-slate-950/40 p-4 text-sm leading-6 text-slate-300 space-y-2">
            <p>Use <span className="font-mono text-blue-400">npm run start:mobile</span> to open the HTTPS PWA on your phone.</p>
            <p>Scan the QR handoff from the onboarding site to unlock the vault session before entering the controller.</p>
            <p>Keep the vault locked when switching devices to avoid stale session state.</p>
          </div>
        </div>
      </div>

      <NetworkSelectorDialog
        isOpen={networkDialogOpen}
        currentNetworkId={networkId}
        currentDevServerUrl={devServerUrl}
        onSelect={(id, devUrl) => {
          selectNetwork(id, devUrl);
          setNetworkDialogOpen(false);
        }}
        onClose={() => setNetworkDialogOpen(false)}
      />
    </Layout>
  );
}
