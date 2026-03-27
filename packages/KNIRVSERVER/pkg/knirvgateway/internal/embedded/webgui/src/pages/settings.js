import React, { useState, useEffect } from 'react';
import { useNavigation } from '../hooks/useNavigation';
import PageLayout from '../components/PageLayout';
import PageHeader from '../components/PageHeader';
import GlassyCard from '../components/GlassyCard';
import styles from './settings.module.css';

export default function Settings() {
  const { activePage } = useNavigation('settings');
  const [searchQuery, setSearchQuery] = useState('');
  const [settings, setSettings] = useState({
    network: 'testnet',
    theme: 'dark',
    notifications: true,
    autoSync: true,
    apiEndpoint: 'https://api.knirv.network',
    refreshInterval: 30
  });

  const handleSearch = (query) => {
    setSearchQuery(query);
  };

  const handleSettingChange = (key, value) => {
    setSettings(prev => ({
      ...prev,
      [key]: value
    }));
    // Save to localStorage
    localStorage.setItem('knirvSettings', JSON.stringify({
      ...settings,
      [key]: value
    }));
  };

  // Load settings from localStorage on component mount
  useEffect(() => {
    const savedSettings = localStorage.getItem('knirvSettings');
    if (savedSettings) {
      setSettings(JSON.parse(savedSettings));
    }
  }, []);

  const resetSettings = () => {
    const defaultSettings = {
      network: 'testnet',
      theme: 'dark',
      notifications: true,
      autoSync: true,
      apiEndpoint: 'https://api.knirv.network',
      refreshInterval: 30
    };
    setSettings(defaultSettings);
    localStorage.setItem('knirvSettings', JSON.stringify(defaultSettings));
  };

  return (
    <PageLayout activePage={activePage} pageTitle="Settings" onSearch={handleSearch}>
      <PageHeader 
        title="Settings"
        subtitle="Configure your KNIRV network preferences"
        titleColor="#007bff"
      />

      <div className={styles.settingsContainer}>
        {/* Network Settings */}
        <GlassyCard className={styles.settingsCard}>
          <h3 className={styles.cardTitle}>
            <i className="fas fa-network-wired" style={{ marginRight: '10px', color: '#007bff' }}></i>
            Network Settings
          </h3>
          
          <div className={styles.settingItem}>
            <label className={styles.settingLabel}>Network</label>
            <select 
              value={settings.network}
              onChange={(e) => handleSettingChange('network', e.target.value)}
              className={styles.settingSelect}
            >
              <option value="mainnet">Mainnet</option>
              <option value="testnet">Testnet</option>
              <option value="private">Private Network</option>
            </select>
          </div>

          <div className={styles.settingItem}>
            <label className={styles.settingLabel}>API Endpoint</label>
            <input 
              type="text"
              value={settings.apiEndpoint}
              onChange={(e) => handleSettingChange('apiEndpoint', e.target.value)}
              className={styles.settingInput}
            />
          </div>

          <div className={styles.settingItem}>
            <label className={styles.settingLabel}>Auto Sync</label>
            <label className={styles.toggleSwitch}>
              <input 
                type="checkbox"
                checked={settings.autoSync}
                onChange={(e) => handleSettingChange('autoSync', e.target.checked)}
              />
              <span className={styles.slider}></span>
            </label>
          </div>
        </GlassyCard>

        {/* Interface Settings */}
        <GlassyCard className={styles.settingsCard}>
          <h3 className={styles.cardTitle}>
            <i className="fas fa-palette" style={{ marginRight: '10px', color: '#007bff' }}></i>
            Interface Settings
          </h3>
          
          <div className={styles.settingItem}>
            <label className={styles.settingLabel}>Theme</label>
            <select 
              value={settings.theme}
              onChange={(e) => handleSettingChange('theme', e.target.value)}
              className={styles.settingSelect}
            >
              <option value="dark">Dark</option>
              <option value="light">Light</option>
              <option value="auto">Auto</option>
            </select>
          </div>

          <div className={styles.settingItem}>
            <label className={styles.settingLabel}>Notifications</label>
            <label className={styles.toggleSwitch}>
              <input 
                type="checkbox"
                checked={settings.notifications}
                onChange={(e) => handleSettingChange('notifications', e.target.checked)}
              />
              <span className={styles.slider}></span>
            </label>
          </div>

          <div className={styles.settingItem}>
            <label className={styles.settingLabel}>Refresh Interval (seconds)</label>
            <input 
              type="number"
              min="10"
              max="300"
              value={settings.refreshInterval}
              onChange={(e) => handleSettingChange('refreshInterval', parseInt(e.target.value))}
              className={styles.settingInput}
            />
          </div>
        </GlassyCard>

        {/* Security Settings */}
        <GlassyCard className={styles.settingsCard}>
          <h3 className={styles.cardTitle}>
            <i className="fas fa-shield-alt" style={{ marginRight: '10px', color: '#007bff' }}></i>
            Security Settings
          </h3>
          
          <div className={styles.settingItem}>
            <label className={styles.settingLabel}>Two-Factor Authentication</label>
            <button className={styles.actionButton}>
              Configure 2FA
            </button>
          </div>

          <div className={styles.settingItem}>
            <label className={styles.settingLabel}>API Keys</label>
            <button className={styles.actionButton}>
              Manage Keys
            </button>
          </div>

          <div className={styles.settingItem}>
            <label className={styles.settingLabel}>Session Timeout</label>
            <select className={styles.settingSelect}>
              <option value="30">30 minutes</option>
              <option value="60">1 hour</option>
              <option value="240">4 hours</option>
              <option value="480">8 hours</option>
            </select>
          </div>
        </GlassyCard>

        {/* Data & Storage */}
        <GlassyCard className={styles.settingsCard}>
          <h3 className={styles.cardTitle}>
            <i className="fas fa-database" style={{ marginRight: '10px', color: '#007bff' }}></i>
            Data & Storage
          </h3>
          
          <div className={styles.storageInfo}>
            <div className={styles.storageItem}>
              <span>Local Storage Used:</span>
              <span>2.4 MB</span>
            </div>
            <div className={styles.storageItem}>
              <span>Cache Size:</span>
              <span>15.7 MB</span>
            </div>
          </div>

          <div className={styles.actionButtons}>
            <button className={styles.actionButton}>
              Clear Cache
            </button>
            <button className={styles.actionButton}>
              Export Data
            </button>
          </div>
        </GlassyCard>

        {/* Reset Settings */}
        <GlassyCard className={styles.settingsCard}>
          <h3 className={styles.cardTitle}>
            <i className="fas fa-undo" style={{ marginRight: '10px', color: '#dc3545' }}></i>
            Reset Settings
          </h3>
          
          <p className={styles.resetWarning}>
            This will reset all settings to their default values. This action cannot be undone.
          </p>
          
          <button 
            onClick={resetSettings}
            className={`${styles.actionButton} ${styles.dangerButton}`}
          >
            Reset to Defaults
          </button>
        </GlassyCard>
      </div>
    </PageLayout>
  );
}
