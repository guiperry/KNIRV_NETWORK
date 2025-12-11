import React, { useState, useEffect } from 'react';
import { useNavigation } from '../hooks/useNavigation';
import PageLayout from '../components/PageLayout';
import PageHeader from '../components/PageHeader';
import GlassyCard from '../components/GlassyCard';
import styles from './tunnel-registry.module.css';

export default function TunnelRegistry() {
  const { activePage } = useNavigation('tunnel-registry');
  const [status, setStatus] = useState(null);
  const [connections, setConnections] = useState([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState(null);

  const fetchStatus = async () => {
    try {
      setIsLoading(true);
      const response = await fetch('/status');
      if (!response.ok) {
        throw new Error('Failed to fetch tunnel registry status');
      }
      const data = await response.json();
      setStatus(data);
      setConnections(data.connections || {});
      setError(null);
    } catch (err) {
      console.error('Failed to fetch status:', err);
      setError('Could not load tunnel registry status. Please try again later.');
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    fetchStatus();
    // Auto-refresh every 30 seconds
    const interval = setInterval(fetchStatus, 30000);
    return () => clearInterval(interval);
  }, []);

  const formatUptime = (uptimeMs) => {
    if (!uptimeMs) return 'N/A';
    const days = Math.floor(uptimeMs / (1000 * 60 * 60 * 24));
    const hours = Math.floor((uptimeMs % (1000 * 60 * 60 * 24)) / (1000 * 60 * 60));
    const minutes = Math.floor((uptimeMs % (1000 * 60 * 60)) / (1000 * 60));
    const seconds = Math.floor((uptimeMs % (1000 * 60)) / 1000);
    return `${days}d ${hours}h ${minutes}m ${seconds}s`;
  };

  const formatMemory = (bytes) => {
    if (!bytes) return 'N/A';
    const mb = Math.round(bytes / 1024 / 1024);
    return `${mb} MB`;
  };

  const getConnectionStatus = (lastSeen) => {
    if (!lastSeen) return { status: 'unknown', className: 'unknown' };
    const now = Date.now();
    const timeDiff = now - new Date(lastSeen).getTime();
    const isActive = timeDiff < 60 * 60 * 1000; // Active in last hour
    return {
      status: isActive ? 'Active' : 'Inactive',
      className: isActive ? 'active' : 'inactive'
    };
  };

  return (
    <PageLayout activePage={activePage} pageTitle="Tunnel Registry">
      <PageHeader
        title="KNIRVORACLE Central Relay Status"
        subtitle="Monitor the tunnel registry service, active connections, and system metrics"
        titleColor="#007bff"
      />

      <div className={styles.container}>
        {/* Refresh Button */}
        <div className={styles.controls}>
          <button
            onClick={fetchStatus}
            disabled={isLoading}
            className={styles.refreshButton}
          >
            {isLoading ? '🔄 Refreshing...' : '🔄 Refresh Status'}
          </button>
        </div>

        {/* Error Alert */}
        {error && (
          <GlassyCard className={styles.errorCard}>
            <div className={styles.errorContent}>
              <div className={styles.errorIcon}>⚠️</div>
              <div>
                <h3 className={styles.errorTitle}>Error</h3>
                <p className={styles.errorText}>{error}</p>
              </div>
            </div>
          </GlassyCard>
        )}

        {/* Status Cards */}
        {status && (
          <div className={styles.statusGrid}>
            <GlassyCard className={styles.statusCard}>
              <div className={styles.statusItem}>
                <div className={styles.statusLabel}>Status</div>
                <div className={styles.statusValue}>{status.status || 'Unknown'}</div>
              </div>
            </GlassyCard>

            <GlassyCard className={styles.statusCard}>
              <div className={styles.statusItem}>
                <div className={styles.statusLabel}>Version</div>
                <div className={styles.statusValue}>{status.version || 'Unknown'}</div>
              </div>
            </GlassyCard>

            <GlassyCard className={styles.statusCard}>
              <div className={styles.statusItem}>
                <div className={styles.statusLabel}>Uptime</div>
                <div className={styles.statusValue}>{formatUptime(status.uptimeMs)}</div>
              </div>
            </GlassyCard>

            <GlassyCard className={styles.statusCard}>
              <div className={styles.statusItem}>
                <div className={styles.statusLabel}>Registered Connections</div>
                <div className={styles.statusValue}>{status.registeredConnections || 0}</div>
              </div>
            </GlassyCard>

            <GlassyCard className={styles.statusCard}>
              <div className={styles.statusItem}>
                <div className={styles.statusLabel}>Active Connections</div>
                <div className={styles.statusValue}>{status.activeConnections || 0}</div>
              </div>
            </GlassyCard>

            <GlassyCard className={styles.statusCard}>
              <div className={styles.statusItem}>
                <div className={styles.statusLabel}>HTTP API Port</div>
                <div className={styles.statusValue}>{status.httpApiPort || 'N/A'}</div>
              </div>
            </GlassyCard>

            <GlassyCard className={styles.statusCard}>
              <div className={styles.statusItem}>
                <div className={styles.statusLabel}>Control Listener Port</div>
                <div className={styles.statusValue}>{status.controlListenerPort || 'N/A'}</div>
              </div>
            </GlassyCard>

            <GlassyCard className={styles.statusCard}>
              <div className={styles.statusItem}>
                <div className={styles.statusLabel}>Public Relay Port</div>
                <div className={styles.statusValue}>{status.publicRelayPort || 'N/A'}</div>
              </div>
            </GlassyCard>

            <GlassyCard className={styles.statusCard}>
              <div className={styles.statusItem}>
                <div className={styles.statusLabel}>Last Updated</div>
                <div className={styles.statusValue}>
                  {status.timestamp ? new Date(status.timestamp).toLocaleString() : 'N/A'}
                </div>
              </div>
            </GlassyCard>

            {/* Memory Usage */}
            {status.memoryUsage && (
              <GlassyCard className={`${styles.statusCard} ${styles.memoryCard}`}>
                <div className={styles.statusItem}>
                  <div className={styles.statusLabel}>Memory Usage</div>
                  <div className={styles.statusValue}>
                    RSS: {formatMemory(status.memoryUsage.rss)}<br />
                    Heap Total: {formatMemory(status.memoryUsage.heapTotal)}<br />
                    Heap Used: {formatMemory(status.memoryUsage.heapUsed)}
                  </div>
                </div>
              </GlassyCard>
            )}
          </div>
        )}

        {/* Active Connections Table */}
        <GlassyCard className={styles.connectionsCard}>
          <h3 className={styles.connectionsTitle}>Active Connections</h3>
          {Object.keys(connections).length > 0 ? (
            <div className={styles.tableContainer}>
              <table className={styles.connectionsTable}>
                <thead>
                  <tr>
                    <th>Connection ID</th>
                    <th>Type</th>
                    <th>Source IP</th>
                    <th>Source Port</th>
                    <th>Status</th>
                    <th>Last Seen</th>
                  </tr>
                </thead>
                <tbody>
                  {Object.entries(connections).map(([connectionID, info]) => {
                    const connStatus = getConnectionStatus(info.lastSeen);
                    return (
                      <tr key={connectionID}>
                        <td className={styles.connectionId}>{connectionID}</td>
                        <td>{info.type || 'N/A'}</td>
                        <td>{info.sourceIP || 'N/A'}</td>
                        <td>{info.sourcePort || 'N/A'}</td>
                        <td className={styles[connStatus.className]}>{connStatus.status}</td>
                        <td>{info.lastSeen ? new Date(info.lastSeen).toLocaleString() : 'N/A'}</td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          ) : (
            <div className={styles.noConnections}>
              <div className={styles.noConnectionsIcon}>📡</div>
              <p>No active connections</p>
            </div>
          )}
        </GlassyCard>
      </div>
    </PageLayout>
  );
}