import React, { useState, useEffect } from 'react';
import axios from 'axios';
import { useBackend } from '../contexts/BackendContext';
import GlassyCard from './GlassyCard';
import DataTable from './DataTable';
import styles from './OracleOperations.module.css';

const OracleOperations = () => {
  const { isRunning } = useBackend();
  const [activeTab, setActiveTab] = useState('tunnels');
  const [tunnelData, setTunnelData] = useState({ connections: [], status: null });
  const [walletData, setWalletData] = useState({ status: null, balance: null });
  const [paymentData, setPaymentData] = useState({ status: null, history: [] });
  const [pluginData, setPluginData] = useState({ plugins: [] });
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (isRunning) {
      fetchAllData();
    }
  }, [isRunning, activeTab]);

  const fetchAllData = async () => {
    setLoading(true);
    try {
      await Promise.all([
        fetchTunnelData(),
        fetchWalletData(),
        fetchPaymentData(),
        fetchPluginData(),
      ]);
    } catch (error) {
      console.error('Error fetching Oracle operations data:', error);
    } finally {
      setLoading(false);
    }
  };

  const fetchTunnelData = async () => {
    try {
      const [statusRes, connectionsRes] = await Promise.all([
        axios.get('/api/tunnel/status'),
        axios.get('/api/tunnel/connections'),
      ]);
      setTunnelData({
        status: statusRes.data,
        connections: connectionsRes.data.connections || [],
      });
    } catch (error) {
      console.error('Error fetching tunnel data:', error);
    }
  };

  const fetchWalletData = async () => {
    try {
      const [statusRes, balanceRes] = await Promise.all([
        axios.get('/api/wallet/status'),
        axios.get('/api/wallet/balance'),
      ]);
      setWalletData({
        status: statusRes.data,
        balance: balanceRes.data,
      });
    } catch (error) {
      console.error('Error fetching wallet data:', error);
    }
  };

  const fetchPaymentData = async () => {
    try {
      const [statusRes, historyRes] = await Promise.all([
        axios.get('/api/payments/status'),
        axios.get('/api/payments/history'),
      ]);
      setPaymentData({
        status: statusRes.data,
        history: historyRes.data.payments || [],
      });
    } catch (error) {
      console.error('Error fetching payment data:', error);
    }
  };

  const fetchPluginData = async () => {
    try {
      const response = await axios.get('/api/plugins');
      setPluginData({
        plugins: response.data.plugins || [],
      });
    } catch (error) {
      console.error('Error fetching plugin data:', error);
    }
  };

  const handlePluginToggle = async (pluginName, enabled) => {
    try {
      const endpoint = enabled ? 'disable' : 'enable';
      await axios.post(`/api/plugins/${pluginName}/${endpoint}`);
      await fetchPluginData(); // Refresh plugin data
    } catch (error) {
      console.error(`Error toggling plugin ${pluginName}:`, error);
      alert(`Failed to ${enabled ? 'disable' : 'enable'} plugin: ${error.message}`);
    }
  };

  const tabs = [
    { id: 'tunnels', label: 'Tunnel Management', icon: '🔗' },
    { id: 'wallet', label: 'Wallet Operations', icon: '💰' },
    { id: 'payments', label: 'Payment Gateway', icon: '💳' },
    { id: 'plugins', label: 'Plugin Management', icon: '🔌' },
  ];

  const tunnelColumns = [
    { key: 'id', label: 'Tunnel ID' },
    { key: 'peer_id', label: 'Peer ID' },
    { key: 'status', label: 'Status', render: (value) => (
      <span className={`${styles.status} ${styles[value]}`}>
        {value.toUpperCase()}
      </span>
    )},
    { key: 'created_at', label: 'Created', render: (value) => 
      new Date(value).toLocaleString()
    },
    { key: 'data_sent', label: 'Data Sent', render: (value) => 
      `${(value / 1024).toFixed(1)} KB`
    },
    { key: 'data_recv', label: 'Data Received', render: (value) => 
      `${(value / 1024).toFixed(1)} KB`
    },
  ];

  const paymentColumns = [
    { key: 'id', label: 'Payment ID' },
    { key: 'amount', label: 'Amount', render: (value, row) => 
      `${value} ${row.currency}`
    },
    { key: 'status', label: 'Status', render: (value) => (
      <span className={`${styles.status} ${styles[value]}`}>
        {value.toUpperCase()}
      </span>
    )},
    { key: 'method', label: 'Method', render: (value) => 
      value.toUpperCase()
    },
    { key: 'timestamp', label: 'Date', render: (value) => 
      new Date(value).toLocaleString()
    },
  ];

  const pluginColumns = [
    { key: 'name', label: 'Plugin Name' },
    { key: 'version', label: 'Version' },
    { key: 'description', label: 'Description' },
    { key: 'status', label: 'Status', render: (value) => (
      <span className={`${styles.status} ${styles[value]}`}>
        {value.toUpperCase()}
      </span>
    )},
    { key: 'enabled', label: 'Actions', render: (value, row) => (
      <button
        className={`${styles.toggleBtn} ${value ? styles.disable : styles.enable}`}
        onClick={() => handlePluginToggle(row.name, value)}
      >
        {value ? 'Disable' : 'Enable'}
      </button>
    )},
  ];

  if (!isRunning) {
    return (
      <div className={styles.container}>
        <GlassyCard>
          <div className={styles.errorState}>
            <h2>Oracle Operations Unavailable</h2>
            <p>Oracle backend is not connected. Please ensure the backend is running.</p>
          </div>
        </GlassyCard>
      </div>
    );
  }

  return (
    <div className={styles.container}>
      <div className={styles.header}>
        <h2>Oracle Operations Center</h2>
        <button className={styles.refreshBtn} onClick={fetchAllData}>
          Refresh All
        </button>
      </div>

      {/* Tab Navigation */}
      <div className={styles.tabNavigation}>
        {tabs.map(tab => (
          <button
            key={tab.id}
            className={`${styles.tab} ${activeTab === tab.id ? styles.active : ''}`}
            onClick={() => setActiveTab(tab.id)}
          >
            <span className={styles.tabIcon}>{tab.icon}</span>
            {tab.label}
          </button>
        ))}
      </div>

      {/* Tab Content */}
      <div className={styles.tabContent}>
        {activeTab === 'tunnels' && (
          <div className={styles.tunnelManagement}>
            <div className={styles.statusGrid}>
              <GlassyCard>
                <h3>Tunnel Service Status</h3>
                <div className={styles.statusInfo}>
                  <div className={styles.statusItem}>
                    <span>Service:</span>
                    <span className={tunnelData.status?.service_running ? styles.online : styles.offline}>
                      {tunnelData.status?.service_running ? 'Running' : 'Stopped'}
                    </span>
                  </div>
                  <div className={styles.statusItem}>
                    <span>Port:</span>
                    <span>{tunnelData.status?.port || 'N/A'}</span>
                  </div>
                  <div className={styles.statusItem}>
                    <span>Active Tunnels:</span>
                    <span>{tunnelData.connections.length}</span>
                  </div>
                </div>
              </GlassyCard>
            </div>

            <GlassyCard>
              <h3>Active Tunnel Connections</h3>
              {tunnelData.connections.length > 0 ? (
                <DataTable
                  data={tunnelData.connections}
                  columns={tunnelColumns}
                  className={styles.dataTable}
                />
              ) : (
                <div className={styles.noData}>No active tunnel connections</div>
              )}
            </GlassyCard>
          </div>
        )}

        {activeTab === 'wallet' && (
          <div className={styles.walletOperations}>
            <div className={styles.statusGrid}>
              <GlassyCard>
                <h3>Wallet Status</h3>
                <div className={styles.statusInfo}>
                  <div className={styles.statusItem}>
                    <span>Status:</span>
                    <span className={walletData.status?.connected ? styles.online : styles.offline}>
                      {walletData.status?.connected ? 'Connected' : 'Disconnected'}
                    </span>
                  </div>
                  <div className={styles.statusItem}>
                    <span>Network:</span>
                    <span>{walletData.status?.network || 'Unknown'}</span>
                  </div>
                  <div className={styles.statusItem}>
                    <span>Address:</span>
                    <span className={styles.address}>{walletData.status?.address || 'N/A'}</span>
                  </div>
                </div>
              </GlassyCard>

              <GlassyCard>
                <h3>Balance Information</h3>
                <div className={styles.balanceInfo}>
                  {walletData.balance?.balance ? (
                    Object.entries(walletData.balance.balance).map(([token, data]) => (
                      <div key={token} className={styles.balanceItem}>
                        <div className={styles.tokenName}>{token}</div>
                        <div className={styles.tokenAmount}>{data.amount}</div>
                        <div className={styles.tokenValue}>${data.usd_value}</div>
                      </div>
                    ))
                  ) : (
                    <div className={styles.noData}>Balance information unavailable</div>
                  )}
                </div>
              </GlassyCard>
            </div>
          </div>
        )}

        {activeTab === 'payments' && (
          <div className={styles.paymentGateway}>
            <div className={styles.statusGrid}>
              <GlassyCard>
                <h3>Payment Gateway Status</h3>
                <div className={styles.statusInfo}>
                  <div className={styles.statusItem}>
                    <span>Service:</span>
                    <span className={paymentData.status?.service_running ? styles.online : styles.offline}>
                      {paymentData.status?.service_running ? 'Running' : 'Stopped'}
                    </span>
                  </div>
                  <div className={styles.statusItem}>
                    <span>Stripe:</span>
                    <span className={paymentData.status?.stripe_enabled ? styles.enabled : styles.disabled}>
                      {paymentData.status?.stripe_enabled ? 'Enabled' : 'Disabled'}
                    </span>
                  </div>
                  <div className={styles.statusItem}>
                    <span>Coinbase:</span>
                    <span className={paymentData.status?.coinbase_enabled ? styles.enabled : styles.disabled}>
                      {paymentData.status?.coinbase_enabled ? 'Enabled' : 'Disabled'}
                    </span>
                  </div>
                </div>
              </GlassyCard>
            </div>

            <GlassyCard>
              <h3>Payment History</h3>
              {paymentData.history.length > 0 ? (
                <DataTable
                  data={paymentData.history}
                  columns={paymentColumns}
                  className={styles.dataTable}
                />
              ) : (
                <div className={styles.noData}>No payment history available</div>
              )}
            </GlassyCard>
          </div>
        )}

        {activeTab === 'plugins' && (
          <div className={styles.pluginManagement}>
            <GlassyCard>
              <h3>Plugin Management</h3>
              <div className={styles.pluginStats}>
                <span>Total Plugins: {pluginData.plugins.length}</span>
                <span>Enabled: {pluginData.plugins.filter(p => p.enabled).length}</span>
                <span>Disabled: {pluginData.plugins.filter(p => !p.enabled).length}</span>
              </div>
              
              {pluginData.plugins.length > 0 ? (
                <DataTable
                  data={pluginData.plugins}
                  columns={pluginColumns}
                  className={styles.dataTable}
                />
              ) : (
                <div className={styles.noData}>No plugins available</div>
              )}
            </GlassyCard>
          </div>
        )}
      </div>

      {loading && (
        <div className={styles.loadingOverlay}>
          <div className={styles.loadingSpinner}>
            <div className={styles.spinner}></div>
            <p>Loading Oracle operations data...</p>
          </div>
        </div>
      )}
    </div>
  );
};

export default OracleOperations;
