import React, { useState, useEffect } from 'react';
import {
  Settings,
  Save,
  RefreshCw,
  Download,
  Upload,
  User,
  Shield,
  Palette,
  Zap,
  Brain,
  Wallet,
  BarChart3
} from 'lucide-react';
import { settingsService, AppSettings, SettingsProfile } from '../services/SettingsService';

interface SettingsPanelProps {
  isOpen: boolean;
  onClose: () => void;
}

const SettingsPanel: React.FC<SettingsPanelProps> = ({ isOpen, onClose }) => {
  const [settings, setSettings] = useState<AppSettings | null>(null);
  const [profiles, setProfiles] = useState<SettingsProfile[]>([]);
  const [activeProfile, setActiveProfile] = useState<SettingsProfile | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [activeTab, setActiveTab] = useState<'general' | 'cognitive' | 'wallet' | 'analytics' | 'security' | 'ui' | 'advanced' | 'profiles'>('general');
  const [hasChanges, setHasChanges] = useState(false);

  useEffect(() => {
    if (isOpen) {
      loadSettings();
    }
  }, [isOpen]);

  const loadSettings = async () => {
    setIsLoading(true);
    try {
      const currentSettings = settingsService.getSettings();
      const allProfiles = settingsService.getProfiles();
      const currentProfile = settingsService.getActiveProfile();
      
      setSettings(currentSettings);
      setProfiles(allProfiles);
      setActiveProfile(currentProfile || null);
      setHasChanges(false);
    } catch (error) {
      console.error('Failed to load settings:', error);
    } finally {
      setIsLoading(false);
    }
  };

  const handleSaveSettings = async () => {
    if (!settings) return;
    
    try {
      await settingsService.updateSettings(settings);
      setHasChanges(false);
    } catch (error) {
      console.error('Failed to save settings:', error);
    }
  };

  const handleResetSettings = async () => {
    try {
      await settingsService.resetSettings();
      await loadSettings();
    } catch (error) {
      console.error('Failed to reset settings:', error);
    }
  };

  const handleExportSettings = () => {
    const data = settingsService.exportSettings(true);
    const blob = new Blob([data], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = 'knirv-settings.json';
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  };

  const handleImportSettings = (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (!file) return;

    const reader = new FileReader();
    reader.onload = async (e) => {
      try {
        const data = e.target?.result as string;
        await settingsService.importSettings(data, false);
        await loadSettings();
      } catch (error) {
        console.error('Failed to import settings:', error);
      }
    };
    reader.readAsText(file);
  };

  const updateSetting = (path: string, value: unknown) => {
    if (!settings) return;
    
    const keys = path.split('.');
    const newSettings = { ...settings };
    let current: Record<string, unknown> = newSettings as Record<string, unknown>;
    
    for (let i = 0; i < keys.length - 1; i++) {
      const currentValue = current[keys[i]];
      current[keys[i]] = { ...(typeof currentValue === 'object' && currentValue !== null ? currentValue as Record<string, unknown> : {}) };
      current = current[keys[i]] as Record<string, unknown>;
    }
    
    current[keys[keys.length - 1]] = value;
    setSettings(newSettings);
    setHasChanges(true);
  };

  if (!isOpen || !settings) return null;

  const tabs = [
    { id: 'general' as const, label: 'General', icon: Settings },
    { id: 'cognitive' as const, label: 'Cognitive', icon: Brain },
    { id: 'wallet' as const, label: 'Wallet', icon: Wallet },
    { id: 'analytics' as const, label: 'Analytics', icon: BarChart3 },
    { id: 'security' as const, label: 'Security', icon: Shield },
    { id: 'ui' as const, label: 'Interface', icon: Palette },
    { id: 'advanced' as const, label: 'Advanced', icon: Zap },
    { id: 'profiles' as const, label: 'Profiles', icon: User }
  ] as const;

  return (
    <div className="fixed inset-0 bg-black/50 backdrop-blur-sm z-50 flex items-center justify-center p-4">
      <div className="bg-gray-900/95 border border-gray-700/50 rounded-xl w-full max-w-6xl h-[90vh] overflow-hidden">
        {/* Header */}
        <div className="flex items-center justify-between p-6 border-b border-gray-700/50">
          <div className="flex items-center space-x-3">
            <div className="w-10 h-10 bg-gray-500/20 rounded-lg flex items-center justify-center border border-gray-500/20">
              <Settings className="w-5 h-5 text-gray-400" />
            </div>
            <div>
              <h2 className="text-xl font-bold text-white">Settings</h2>
              <p className="text-sm text-gray-400">Configure KNIRV Controller preferences</p>
            </div>
          </div>
          <div className="flex items-center space-x-2">
            {hasChanges && (
              <button
                onClick={handleSaveSettings}
                className="px-4 py-2 bg-blue-500 text-white rounded-lg hover:bg-blue-600 transition-all flex items-center space-x-2"
              >
                <Save className="w-4 h-4" />
                <span>Save</span>
              </button>
            )}
            <button
              onClick={loadSettings}
              disabled={isLoading}
              className="p-2 hover:bg-gray-700/50 rounded-lg text-gray-400 hover:text-white transition-all disabled:opacity-50"
            >
              <RefreshCw className={`w-4 h-4 ${isLoading ? 'animate-spin' : ''}`} />
            </button>
            <button
              onClick={onClose}
              className="p-2 hover:bg-gray-700/50 rounded-lg text-gray-400 hover:text-white transition-all"
            >
              ×
            </button>
          </div>
        </div>

        <div className="flex h-full">
          {/* Sidebar */}
          <div className="w-64 border-r border-gray-700/50 p-4">
            <div className="space-y-1">
              {tabs.map(tab => (
                <button
                  key={tab.id}
                  onClick={() => setActiveTab(tab.id)}
                  className={`w-full flex items-center space-x-3 px-3 py-2 text-sm font-medium rounded-lg transition-all ${
                    activeTab === tab.id
                      ? 'text-blue-400 bg-blue-500/10 border border-blue-500/20'
                      : 'text-gray-400 hover:text-white hover:bg-gray-700/30'
                  }`}
                >
                  <tab.icon className="w-4 h-4" />
                  <span>{tab.label}</span>
                </button>
              ))}
            </div>

            <div className="mt-6 pt-6 border-t border-gray-700/50">
              <div className="space-y-2">
                <button
                  onClick={handleExportSettings}
                  className="w-full flex items-center space-x-2 px-3 py-2 text-sm text-gray-400 hover:text-white hover:bg-gray-700/30 rounded-lg transition-all"
                >
                  <Download className="w-4 h-4" />
                  <span>Export</span>
                </button>
                <label className="w-full flex items-center space-x-2 px-3 py-2 text-sm text-gray-400 hover:text-white hover:bg-gray-700/30 rounded-lg transition-all cursor-pointer">
                  <Upload className="w-4 h-4" />
                  <span>Import</span>
                  <input
                    type="file"
                    accept=".json"
                    onChange={handleImportSettings}
                    className="hidden"
                  />
                </label>
                <button
                  onClick={handleResetSettings}
                  className="w-full flex items-center space-x-2 px-3 py-2 text-sm text-red-400 hover:text-red-300 hover:bg-red-500/10 rounded-lg transition-all"
                >
                  <RefreshCw className="w-4 h-4" />
                  <span>Reset</span>
                </button>
              </div>
            </div>
          </div>

          {/* Content */}
          <div className="flex-1 p-6 overflow-y-auto">
            {activeTab === 'general' && (
              <div className="space-y-6">
                <h3 className="text-lg font-semibold text-white">General Settings</h3>
                
                <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                  <div>
                    <label className="block text-sm font-medium text-gray-300 mb-2">Theme</label>
                    <select
                      value={settings.general.theme}
                      onChange={(e) => updateSetting('general.theme', e.target.value)}
                      className="w-full px-3 py-2 bg-gray-800 border border-gray-600 rounded-lg text-white focus:border-blue-400 focus:outline-none"
                    >
                      <option value="dark">Dark</option>
                      <option value="light">Light</option>
                      <option value="auto">Auto</option>
                    </select>
                  </div>

                  <div>
                    <label className="block text-sm font-medium text-gray-300 mb-2">Language</label>
                    <select
                      value={settings.general.language}
                      onChange={(e) => updateSetting('general.language', e.target.value)}
                      className="w-full px-3 py-2 bg-gray-800 border border-gray-600 rounded-lg text-white focus:border-blue-400 focus:outline-none"
                    >
                      <option value="en">English</option>
                      <option value="es">Spanish</option>
                      <option value="fr">French</option>
                      <option value="de">German</option>
                    </select>
                  </div>

                  <div>
                    <label className="block text-sm font-medium text-gray-300 mb-2">Timezone</label>
                    <input
                      type="text"
                      value={settings.general.timezone}
                      onChange={(e) => updateSetting('general.timezone', e.target.value)}
                      className="w-full px-3 py-2 bg-gray-800 border border-gray-600 rounded-lg text-white focus:border-blue-400 focus:outline-none"
                    />
                  </div>

                  <div>
                    <label className="block text-sm font-medium text-gray-300 mb-2">Backup Interval (minutes)</label>
                    <input
                      type="number"
                      value={settings.general.backupInterval}
                      onChange={(e) => updateSetting('general.backupInterval', parseInt(e.target.value))}
                      className="w-full px-3 py-2 bg-gray-800 border border-gray-600 rounded-lg text-white focus:border-blue-400 focus:outline-none"
                      min="5"
                      max="1440"
                    />
                  </div>
                </div>

                <div className="space-y-4">
                  <div className="flex items-center justify-between">
                    <div>
                      <h4 className="text-white font-medium">Auto Save</h4>
                      <p className="text-sm text-gray-400">Automatically save changes</p>
                    </div>
                    <label className="relative inline-flex items-center cursor-pointer">
                      <input
                        type="checkbox"
                        checked={settings.general.autoSave}
                        onChange={(e) => updateSetting('general.autoSave', e.target.checked)}
                        className="sr-only peer"
                      />
                      <div className="w-11 h-6 bg-gray-600 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-blue-500"></div>
                    </label>
                  </div>

                  <div className="flex items-center justify-between">
                    <div>
                      <h4 className="text-white font-medium">Auto Backup</h4>
                      <p className="text-sm text-gray-400">Automatically backup settings</p>
                    </div>
                    <label className="relative inline-flex items-center cursor-pointer">
                      <input
                        type="checkbox"
                        checked={settings.general.autoBackup}
                        onChange={(e) => updateSetting('general.autoBackup', e.target.checked)}
                        className="sr-only peer"
                      />
                      <div className="w-11 h-6 bg-gray-600 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-blue-500"></div>
                    </label>
                  </div>

                  <div className="flex items-center justify-between">
                    <div>
                      <h4 className="text-white font-medium">Debug Mode</h4>
                      <p className="text-sm text-gray-400">Enable debug logging</p>
                    </div>
                    <label className="relative inline-flex items-center cursor-pointer">
                      <input
                        type="checkbox"
                        checked={settings.general.debugMode}
                        onChange={(e) => updateSetting('general.debugMode', e.target.checked)}
                        className="sr-only peer"
                      />
                      <div className="w-11 h-6 bg-gray-600 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-blue-500"></div>
                    </label>
                  </div>

                  <div className="flex items-center justify-between">
                    <div>
                      <h4 className="text-white font-medium">Telemetry</h4>
                      <p className="text-sm text-gray-400">Send anonymous usage data</p>
                    </div>
                    <label className="relative inline-flex items-center cursor-pointer">
                      <input
                        type="checkbox"
                        checked={settings.general.telemetryEnabled}
                        onChange={(e) => updateSetting('general.telemetryEnabled', e.target.checked)}
                        className="sr-only peer"
                      />
                      <div className="w-11 h-6 bg-gray-600 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-blue-500"></div>
                    </label>
                  </div>
                </div>
              </div>
            )}

            {activeTab === 'cognitive' && (
              <div className="space-y-6">
                <h3 className="text-lg font-semibold text-white">Cognitive Engine Settings</h3>
                
                <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                  <div>
                    <label className="block text-sm font-medium text-gray-300 mb-2">Default Model</label>
                    <select
                      value={settings.cognitive.defaultModel}
                      onChange={(e) => updateSetting('cognitive.defaultModel', e.target.value)}
                      className="w-full px-3 py-2 bg-gray-800 border border-gray-600 rounded-lg text-white focus:border-blue-400 focus:outline-none"
                    >
                      <option value="gpt-4">GPT-4</option>
                      <option value="gpt-3.5-turbo">GPT-3.5 Turbo</option>
                      <option value="claude-3">Claude 3</option>
                      <option value="gemini-pro">Gemini Pro</option>
                    </select>
                  </div>

                  <div>
                    <label className="block text-sm font-medium text-gray-300 mb-2">Max Tokens</label>
                    <input
                      type="number"
                      value={settings.cognitive.maxTokens}
                      onChange={(e) => updateSetting('cognitive.maxTokens', parseInt(e.target.value))}
                      className="w-full px-3 py-2 bg-gray-800 border border-gray-600 rounded-lg text-white focus:border-blue-400 focus:outline-none"
                      min="100"
                      max="8192"
                    />
                  </div>

                  <div>
                    <label className="block text-sm font-medium text-gray-300 mb-2">Temperature</label>
                    <input
                      type="range"
                      min="0"
                      max="2"
                      step="0.1"
                      value={settings.cognitive.temperature}
                      onChange={(e) => updateSetting('cognitive.temperature', parseFloat(e.target.value))}
                      className="w-full"
                    />
                    <div className="text-sm text-gray-400 mt-1">{settings.cognitive.temperature}</div>
                  </div>

                  <div>
                    <label className="block text-sm font-medium text-gray-300 mb-2">Top P</label>
                    <input
                      type="range"
                      min="0"
                      max="1"
                      step="0.1"
                      value={settings.cognitive.topP}
                      onChange={(e) => updateSetting('cognitive.topP', parseFloat(e.target.value))}
                      className="w-full"
                    />
                    <div className="text-sm text-gray-400 mt-1">{settings.cognitive.topP}</div>
                  </div>
                </div>

                <div className="space-y-4">
                  <div className="flex items-center justify-between">
                    <div>
                      <h4 className="text-white font-medium">Auto Learning</h4>
                      <p className="text-sm text-gray-400">Enable automatic learning from interactions</p>
                    </div>
                    <label className="relative inline-flex items-center cursor-pointer">
                      <input
                        type="checkbox"
                        checked={settings.cognitive.autoLearning}
                        onChange={(e) => updateSetting('cognitive.autoLearning', e.target.checked)}
                        className="sr-only peer"
                      />
                      <div className="w-11 h-6 bg-gray-600 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-blue-500"></div>
                    </label>
                  </div>

                  <div className="flex items-center justify-between">
                    <div>
                      <h4 className="text-white font-medium">Skill Caching</h4>
                      <p className="text-sm text-gray-400">Cache frequently used skills</p>
                    </div>
                    <label className="relative inline-flex items-center cursor-pointer">
                      <input
                        type="checkbox"
                        checked={settings.cognitive.skillCaching}
                        onChange={(e) => updateSetting('cognitive.skillCaching', e.target.checked)}
                        className="sr-only peer"
                      />
                      <div className="w-11 h-6 bg-gray-600 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-blue-500"></div>
                    </label>
                  </div>
                </div>
              </div>
            )}

            {activeTab === 'profiles' && (
              <div className="space-y-6">
                <div className="flex items-center justify-between">
                  <h3 className="text-lg font-semibold text-white">Settings Profiles</h3>
                  <button
                    onClick={() => {/* Create new profile */}}
                    className="px-4 py-2 bg-blue-500 text-white rounded-lg hover:bg-blue-600 transition-all"
                  >
                    Create Profile
                  </button>
                </div>

                <div className="space-y-4">
                  {profiles.map(profile => (
                    <div key={profile.id} className="bg-gray-800/50 border border-gray-700/50 rounded-lg p-4">
                      <div className="flex items-center justify-between">
                        <div>
                          <h4 className="font-semibold text-white">{profile.name}</h4>
                          <p className="text-sm text-gray-400">{profile.description}</p>
                          <p className="text-xs text-gray-500">Created: {profile.createdAt.toLocaleDateString()}</p>
                        </div>
                        <div className="flex items-center space-x-2">
                          {activeProfile?.id === profile.id && (
                            <span className="px-2 py-1 bg-blue-500/20 border border-blue-500/20 text-blue-400 rounded text-xs">
                              Active
                            </span>
                          )}
                          <button
                            onClick={() => settingsService.loadProfile(profile.id)}
                            className="px-3 py-1 bg-gray-600 text-white rounded hover:bg-gray-700 transition-all text-sm"
                          >
                            Load
                          </button>
                          {!profile.isDefault && (
                            <button
                              onClick={() => settingsService.deleteProfile(profile.id)}
                              className="px-3 py-1 bg-red-600 text-white rounded hover:bg-red-700 transition-all text-sm"
                            >
                              Delete
                            </button>
                          )}
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {/* Add other tab content as needed */}
          </div>
        </div>
      </div>
    </div>
  );
};

export default SettingsPanel;
