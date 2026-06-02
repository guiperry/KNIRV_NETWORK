import React, { useState } from 'react';
import { useNetwork } from '../contexts/NetworkContext';
import styles from './NetworkSelector.module.css';

const NetworkSelector = ({ compact = false }) => {
  const {
    currentNetwork,
    networks,
    isConnecting,
    connectionStatus,
    networkHealth,
    switchNetwork,
    refreshHealth
  } = useNetwork();

  const [isDropdownOpen, setIsDropdownOpen] = useState(false);
  const [isSwitching, setIsSwitching] = useState(false);

  const handleNetworkSwitch = async (networkId) => {
    if (networkId === currentNetwork?.id || isSwitching) return;
    
    setIsSwitching(true);
    try {
      await switchNetwork(networkId);
      setIsDropdownOpen(false);
    } catch (error) {
      console.error('Failed to switch network:', error);
      // Could show a toast notification here
    } finally {
      setIsSwitching(false);
    }
  };

  const getStatusIcon = () => {
    switch (connectionStatus) {
      case 'connected':
        return <span className={styles.statusIcon} style={{ color: '#28a745' }}>●</span>;
      case 'connecting':
        return <span className={styles.statusIcon} style={{ color: '#ffc107' }}>●</span>;
      case 'error':
        return <span className={styles.statusIcon} style={{ color: '#dc3545' }}>●</span>;
      default:
        return <span className={styles.statusIcon} style={{ color: '#6c757d' }}>●</span>;
    }
  };

  const getStatusText = () => {
    switch (connectionStatus) {
      case 'connected': return 'Connected';
      case 'connecting': return 'Connecting...';
      case 'error': return 'Connection Error';
      default: return 'Disconnected';
    }
  };

  if (!currentNetwork) {
    return (
      <div className={styles.networkSelector}>
        <div className={styles.loadingState}>
          <span className={styles.loadingSpinner}></span>
          Loading networks...
        </div>
      </div>
    );
  }

  if (compact) {
    return (
      <div className={styles.compactSelector}>
        <button
          onClick={() => setIsDropdownOpen(!isDropdownOpen)}
          className={styles.compactButton}
          disabled={isConnecting || isSwitching}
        >
          {getStatusIcon()}
          <span className={styles.networkName}>{currentNetwork.name}</span>
          <i className={`fas fa-chevron-down ${isDropdownOpen ? styles.rotated : ''}`}></i>
        </button>
        
        {isDropdownOpen && (
          <div className={styles.compactDropdown}>
            {Object.values(networks).map((network) => (
              <button
                key={network.id}
                onClick={() => handleNetworkSwitch(network.id)}
                className={`${styles.compactOption} ${
                  network.id === currentNetwork.id ? styles.active : ''
                }`}
                disabled={isSwitching}
              >
                <span 
                  className={styles.networkIndicator}
                  style={{ backgroundColor: network.color }}
                ></span>
                <span className={styles.optionName}>{network.name}</span>
                {network.id === currentNetwork.id && (
                  <i className="fas fa-check" style={{ color: '#28a745' }}></i>
                )}
              </button>
            ))}
          </div>
        )}
      </div>
    );
  }

  return (
    <div className={styles.networkSelector}>
      <div className={styles.selectorHeader}>
        <h3 className={styles.selectorTitle}>
          <i className="fas fa-network-wired" style={{ marginRight: '10px', color: '#007bff' }}></i>
          Network Selection
        </h3>
        <button
          onClick={refreshHealth}
          className={styles.refreshButton}
          disabled={isConnecting}
          title="Refresh network status"
        >
          <i className={`fas fa-sync-alt ${isConnecting ? styles.spinning : ''}`}></i>
        </button>
      </div>

      {/* Current Network Status */}
      <div className={styles.currentNetwork}>
        <div className={styles.networkInfo}>
          <div className={styles.networkHeader}>
            <span 
              className={styles.networkDot}
              style={{ backgroundColor: currentNetwork.color }}
            ></span>
            <h4 className={styles.networkTitle}>{currentNetwork.name}</h4>
            {getStatusIcon()}
          </div>
          <p className={styles.networkDescription}>{currentNetwork.description}</p>
          <div className={styles.networkMeta}>
            <span className={styles.metaItem}>
              <strong>Chain ID:</strong> {currentNetwork.chainId}
            </span>
            <span className={styles.metaItem}>
              <strong>Currency:</strong> {currentNetwork.currency}
            </span>
            <span className={styles.metaItem}>
              <strong>Status:</strong> {getStatusText()}
            </span>
          </div>
        </div>
      </div>

      {/* Network Health */}
      {networkHealth && (
        <div className={styles.healthStatus}>
          <h4 className={styles.healthTitle}>Network Health</h4>
          <div className={styles.healthGrid}>
            <div className={styles.healthItem}>
              <span className={styles.healthLabel}>API</span>
              <span className={`${styles.healthValue} ${networkHealth.api ? styles.healthy : styles.unhealthy}`}>
                {networkHealth.api ? 'Healthy' : 'Offline'}
              </span>
            </div>
            <div className={styles.healthItem}>
              <span className={styles.healthLabel}>WebSocket</span>
              <span className={`${styles.healthValue} ${networkHealth.websocket ? styles.healthy : styles.unhealthy}`}>
                {networkHealth.websocket ? 'Connected' : 'Disconnected'}
              </span>
            </div>
          </div>
          {networkHealth.lastChecked && (
            <div className={styles.lastChecked}>
              Last checked: {new Date(networkHealth.lastChecked).toLocaleTimeString()}
            </div>
          )}
        </div>
      )}

      {/* Available Networks */}
      <div className={styles.availableNetworks}>
        <h4 className={styles.networksTitle}>Available Networks</h4>
        <div className={styles.networksList}>
          {Object.values(networks).map((network) => (
            <button
              key={network.id}
              onClick={() => handleNetworkSwitch(network.id)}
              className={`${styles.networkOption} ${
                network.id === currentNetwork.id ? styles.active : ''
              }`}
              disabled={isSwitching || isConnecting}
            >
              <div className={styles.optionHeader}>
                <span 
                  className={styles.optionDot}
                  style={{ backgroundColor: network.color }}
                ></span>
                <span className={styles.optionName}>{network.name}</span>
                {network.id === currentNetwork.id && (
                  <i className="fas fa-check" style={{ color: '#28a745' }}></i>
                )}
                {isSwitching && (
                  <span className={styles.switchingSpinner}></span>
                )}
              </div>
              <p className={styles.optionDescription}>{network.description}</p>
              <div className={styles.optionMeta}>
                <span className={styles.optionChain}>{network.chainId}</span>
                <span className={styles.optionCurrency}>{network.currency}</span>
                <span className={`${styles.optionStatus} ${styles[network.status]}`}>
                  {network.status}
                </span>
              </div>
            </button>
          ))}
        </div>
      </div>

      {/* Connection Actions */}
      <div className={styles.connectionActions}>
        <button
          onClick={() => refreshHealth()}
          className={styles.testConnectionButton}
          disabled={isConnecting}
        >
          <i className="fas fa-plug" style={{ marginRight: '8px' }}></i>
          Test Connection
        </button>
        
        {currentNetwork.explorerUrl && (
          <a
            href={currentNetwork.explorerUrl}
            target="_blank"
            rel="noopener noreferrer"
            className={styles.explorerButton}
          >
            <i className="fas fa-external-link-alt" style={{ marginRight: '8px' }}></i>
            Open Explorer
          </a>
        )}
      </div>
    </div>
  );
};

export default NetworkSelector;
