import React, { useState, useEffect } from 'react';
import { useNavigation } from '../hooks/useNavigation';
import { useRole } from '../contexts/RoleContext';
import PageLayout from '../components/PageLayout';
import PageHeader from '../components/PageHeader';
import GlassyCard from '../components/GlassyCard';
import styles from './network-monitor.module.css';

export default function NetworkMonitor() {
  const { activePage } = useNavigation('network-monitor');
  const { role, getUserInfo } = useRole();
  const [searchQuery, setSearchQuery] = useState('');
  const [networkData, setNetworkData] = useState(null);
  const [services, setServices] = useState({});
  const [metrics, setMetrics] = useState(null);
  const [logs, setLogs] = useState([]);
  const [config, setConfig] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [selectedService, setSelectedService] = useState(null);
  const [showServiceModal, setShowServiceModal] = useState(false);
  const [showLogsModal, setShowLogsModal] = useState(false);
  const [showMetricsModal, setShowMetricsModal] = useState(false);

  // Root admin check
  if (role !== 'Root') {
    return (
      <PageLayout activePage={activePage} pageTitle="Network Monitor" onSearch={() => {}}>
        <div style={{
          display: 'flex',
          justifyContent: 'center',
          alignItems: 'center',
          height: '60vh',
          flexDirection: 'column'
        }}>
          <i className="fas fa-lock" style={{ fontSize: '64px', color: '#dc3545', marginBottom: '20px' }}></i>
          <h2 style={{ color: '#dc3545' }}>Access Denied</h2>
          <p style={{ color: '#ccc' }}>Network Monitor requires Root administrator privileges.</p>
          <p style={{ color: '#888', marginTop: '10px' }}>Your current role: {role}</p>
        </div>
      </PageLayout>
    );
  }

  const handleSearch = (query) => {
    setSearchQuery(query);
  };

  // Fetch all data
  const fetchAllData = async () => {
    try {
      setLoading(true);
      setError(null);

      const headers = {
        'x-knirv-role': role
      };

      // Fetch network status
      const statusRes = await fetch('/api/network-monitor/status', { headers });
      const statusData = await statusRes.json();
      if (statusData.success) {
        setNetworkData(statusData.data);
      }

      // Fetch services
      const servicesRes = await fetch('/api/network-monitor/services', { headers });
      const servicesData = await servicesRes.json();
      if (servicesData.success) {
        setServices(servicesData.data);
      }

      // Fetch metrics
      const metricsRes = await fetch('/api/network-monitor/metrics', { headers });
      const metricsData = await metricsRes.json();
      if (metricsData.success) {
        setMetrics(metricsData.data);
      }

      // Fetch logs
      const logsRes = await fetch('/api/network-monitor/logs?limit=20', { headers });
      const logsData = await logsRes.json();
      if (logsData.success) {
        setLogs(logsData.data);
      }

      // Fetch config
      const configRes = await fetch('/api/network-monitor/config', { headers });
      const configData = await configRes.json();
      if (configData.success) {
        setConfig(configData.data);
      }

    } catch (err) {
      console.error('Error fetching network monitor data:', err);
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchAllData();

    // Auto-refresh every 5 seconds
    const interval = setInterval(fetchAllData, 5000);
    return () => clearInterval(interval);
  }, [role]);

  const getStatusColor = (status) => {
    switch (status) {
      case 'up':
      case 'healthy':
        return '#28a745';
      case 'down':
      case 'critical':
        return '#dc3545';
      case 'degraded':
        return '#ffc107';
      default:
        return '#6c757d';
    }
  };

  const formatBytes = (bytes) => {
    if (!bytes) return '0 B';
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(1024));
    return `${(bytes / Math.pow(1024, i)).toFixed(2)} ${sizes[i]}`;
  };

  const formatDuration = (duration) => {
    if (!duration) return 'N/A';
    return duration.toString().replace('ns', 'ms');
  };

  const ServiceDetailModal = () => {
    if (!selectedService) return null;

    return (
      <div className={styles.modal} onClick={() => setShowServiceModal(false)}>
        <div className={styles.modalContent} onClick={(e) => e.stopPropagation()}>
          <div className={styles.modalHeader}>
            <h3><i className="fas fa-server" style={{ marginRight: '10px' }}></i>{selectedService.name}</h3>
            <button onClick={() => setShowServiceModal(false)} className={styles.closeButton}>×</button>
          </div>
          <div className={styles.modalBody}>
            <div className={styles.detailGrid}>
              <div className={styles.detailItem}>
                <strong>Status:</strong>
                <span style={{ color: getStatusColor(selectedService.status) }}>
                  {selectedService.status?.toUpperCase()}
                </span>
              </div>
              <div className={styles.detailItem}>
                <strong>URL:</strong>
                <span>{selectedService.url}</span>
              </div>
              <div className={styles.detailItem}>
                <strong>Response Time:</strong>
                <span>{formatDuration(selectedService.response_time)}</span>
              </div>
              <div className={styles.detailItem}>
                <strong>Last Check:</strong>
                <span>{selectedService.last_check ? new Date(selectedService.last_check).toLocaleString() : 'N/A'}</span>
              </div>
              {selectedService.error && (
                <div className={styles.detailItem} style={{ gridColumn: '1 / -1' }}>
                  <strong>Error:</strong>
                  <span style={{ color: '#dc3545' }}>{selectedService.error}</span>
                </div>
              )}
              {selectedService.critical && (
                <div className={styles.detailItem}>
                  <strong>Critical:</strong>
                  <span style={{ color: '#dc3545' }}>Yes</span>
                </div>
              )}
            </div>
          </div>
        </div>
      </div>
    );
  };

  const MetricsModal = () => {
    if (!metrics?.system) return null;

    const sys = metrics.system;

    return (
      <div className={styles.modal} onClick={() => setShowMetricsModal(false)}>
        <div className={styles.modalContent} onClick={(e) => e.stopPropagation()}>
          <div className={styles.modalHeader}>
            <h3><i className="fas fa-chart-bar" style={{ marginRight: '10px' }}></i>System Metrics</h3>
            <button onClick={() => setShowMetricsModal(false)} className={styles.closeButton}>×</button>
          </div>
          <div className={styles.modalBody}>
            <div className={styles.metricsSection}>
              <h4><i className="fas fa-microchip" style={{ marginRight: '8px' }}></i>CPU</h4>
              <div className={styles.detailGrid}>
                <div className={styles.detailItem}>
                  <strong>Usage:</strong>
                  <span>{sys.cpu?.usage_percent?.toFixed(1)}%</span>
                </div>
                <div className={styles.detailItem}>
                  <strong>Load Average:</strong>
                  <span>{sys.cpu?.load_average?.join(', ') || 'N/A'}</span>
                </div>
              </div>
            </div>

            <div className={styles.metricsSection}>
              <h4><i className="fas fa-memory" style={{ marginRight: '8px' }}></i>Memory</h4>
              <div className={styles.detailGrid}>
                <div className={styles.detailItem}>
                  <strong>Total:</strong>
                  <span>{formatBytes(sys.memory?.total_bytes)}</span>
                </div>
                <div className={styles.detailItem}>
                  <strong>Used:</strong>
                  <span>{formatBytes(sys.memory?.used_bytes)}</span>
                </div>
                <div className={styles.detailItem}>
                  <strong>Usage:</strong>
                  <span>{sys.memory?.usage_percent?.toFixed(1)}%</span>
                </div>
              </div>
            </div>

            <div className={styles.metricsSection}>
              <h4><i className="fas fa-hdd" style={{ marginRight: '8px' }}></i>Disk</h4>
              <div className={styles.detailGrid}>
                <div className={styles.detailItem}>
                  <strong>Total:</strong>
                  <span>{formatBytes(sys.disk?.total_bytes)}</span>
                </div>
                <div className={styles.detailItem}>
                  <strong>Used:</strong>
                  <span>{formatBytes(sys.disk?.used_bytes)}</span>
                </div>
                <div className={styles.detailItem}>
                  <strong>Usage:</strong>
                  <span>{sys.disk?.usage_percent?.toFixed(1)}%</span>
                </div>
              </div>
            </div>

            {sys.ipfs && (
              <div className={styles.metricsSection}>
                <h4><i className="fas fa-cube" style={{ marginRight: '8px' }}></i>IPFS</h4>
                <div className={styles.detailGrid}>
                  <div className={styles.detailItem}>
                    <strong>Storage:</strong>
                    <span>{formatBytes(sys.ipfs.repo_size)} / {formatBytes(sys.ipfs.repo_size_max)}</span>
                  </div>
                  <div className={styles.detailItem}>
                    <strong>Usage:</strong>
                    <span>{sys.ipfs.storage_percent?.toFixed(1)}%</span>
                  </div>
                  <div className={styles.detailItem}>
                    <strong>Peers:</strong>
                    <span>{sys.ipfs.peer_count}</span>
                  </div>
                </div>
              </div>
            )}
          </div>
        </div>
      </div>
    );
  };

  const LogsModal = () => {
    return (
      <div className={styles.modal} onClick={() => setShowLogsModal(false)}>
        <div className={styles.modalContent} onClick={(e) => e.stopPropagation()} style={{ maxWidth: '900px' }}>
          <div className={styles.modalHeader}>
            <h3><i className="fas fa-file-alt" style={{ marginRight: '10px' }}></i>System Logs</h3>
            <button onClick={() => setShowLogsModal(false)} className={styles.closeButton}>×</button>
          </div>
          <div className={styles.modalBody}>
            <div className={styles.logsContainer}>
              {logs.length === 0 ? (
                <p style={{ color: '#888', textAlign: 'center', padding: '20px' }}>No logs available</p>
              ) : (
                logs.map((log, index) => (
                  <div key={index} className={styles.logEntry}>
                    <span className={styles.logTime}>
                      {log['@timestamp'] ? new Date(log['@timestamp']).toLocaleTimeString() : 'N/A'}
                    </span>
                    <span className={`${styles.logLevel} ${styles[log.level]}`}>
                      {log.level?.toUpperCase()}
                    </span>
                    <span className={styles.logService}>{log.service}</span>
                    <span className={styles.logMessage}>{log.message}</span>
                  </div>
                ))
              )}
            </div>
          </div>
        </div>
      </div>
    );
  };

  if (loading && !networkData) {
    return (
      <PageLayout activePage={activePage} pageTitle="Network Monitor" onSearch={handleSearch}>
        <div style={{ textAlign: 'center', padding: '40px' }}>
          <i className="fas fa-spinner fa-spin" style={{ fontSize: '48px', color: '#007bff' }}></i>
          <p style={{ color: '#ccc', marginTop: '20px' }}>Loading network monitor data...</p>
        </div>
      </PageLayout>
    );
  }

  return (
    <PageLayout activePage={activePage} pageTitle="Network Monitor" onSearch={handleSearch}>
      <PageHeader
        title="Network Monitor"
        subtitle="Real-time monitoring and management of KNIRV network infrastructure (Root Admin Only)"
        titleColor="#dc3545"
      />

      {error && (
        <div style={{
          background: 'rgba(220, 53, 69, 0.1)',
          border: '1px solid #dc3545',
          borderRadius: '8px',
          padding: '15px',
          marginBottom: '20px',
          color: '#dc3545'
        }}>
          <i className="fas fa-exclamation-triangle" style={{ marginRight: '10px' }}></i>
          {error}
        </div>
      )}

      <div className={styles.gridContainer}>
        {/* Network Status Overview */}
        <GlassyCard className={styles.statusCard}>
          <h3 className={styles.cardTitle}>
            <i className="fas fa-network-wired" style={{ marginRight: '10px', color: '#007bff' }}></i>
            Network Status
          </h3>
          {networkData ? (
            <div className={styles.statusGrid}>
              <div className={styles.statusItem}>
                <div className={styles.statusLabel}>Network</div>
                <div className={styles.statusValue}>{networkData.name || 'Unknown'}</div>
              </div>
              <div className={styles.statusItem}>
                <div className={styles.statusLabel}>Overall Status</div>
                <div className={styles.statusValue} style={{ color: getStatusColor(networkData.overall_status) }}>
                  {networkData.overall_status?.toUpperCase()}
                </div>
              </div>
              <div className={styles.statusItem}>
                <div className={styles.statusLabel}>Services</div>
                <div className={styles.statusValue}>
                  {networkData.services_up} / {networkData.services_total}
                  <span style={{ color: '#28a745', marginLeft: '5px' }}>UP</span>
                </div>
              </div>
              <div className={styles.statusItem}>
                <div className={styles.statusLabel}>Last Update</div>
                <div className={styles.statusValue}>
                  {networkData.last_update ? new Date(networkData.last_update).toLocaleTimeString() : 'N/A'}
                </div>
              </div>
            </div>
          ) : (
            <p style={{ color: '#888' }}>Loading...</p>
          )}
        </GlassyCard>

        {/* Services List */}
        <GlassyCard className={styles.servicesCard}>
          <h3 className={styles.cardTitle}>
            <i className="fas fa-server" style={{ marginRight: '10px', color: '#007bff' }}></i>
            Services
          </h3>
          <div className={styles.servicesList}>
            {Object.entries(services).length === 0 ? (
              <p style={{ color: '#888', textAlign: 'center' }}>No services found</p>
            ) : (
              Object.entries(services).map(([name, service]) => (
                <div
                  key={name}
                  className={styles.serviceItem}
                  onClick={() => {
                    setSelectedService(service);
                    setShowServiceModal(true);
                  }}
                >
                  <div className={styles.serviceName}>
                    <i className="fas fa-circle" style={{
                      color: getStatusColor(service.status),
                      fontSize: '10px',
                      marginRight: '8px'
                    }}></i>
                    {name}
                  </div>
                  <div className={styles.serviceStatus} style={{ color: getStatusColor(service.status) }}>
                    {service.status?.toUpperCase()}
                  </div>
                  <div className={styles.serviceResponseTime}>
                    {formatDuration(service.response_time)}
                  </div>
                </div>
              ))
            )}
          </div>
        </GlassyCard>

        {/* Quick Metrics */}
        <GlassyCard className={styles.metricsCard}>
          <h3 className={styles.cardTitle}>
            <i className="fas fa-chart-line" style={{ marginRight: '10px', color: '#007bff' }}></i>
            System Metrics
          </h3>
          {metrics?.system ? (
            <div className={styles.quickMetrics}>
              <div className={styles.metricItem}>
                <i className="fas fa-microchip"></i>
                <div>
                  <div className={styles.metricLabel}>CPU</div>
                  <div className={styles.metricValue}>{metrics.system.cpu?.usage_percent?.toFixed(1)}%</div>
                </div>
              </div>
              <div className={styles.metricItem}>
                <i className="fas fa-memory"></i>
                <div>
                  <div className={styles.metricLabel}>Memory</div>
                  <div className={styles.metricValue}>{metrics.system.memory?.usage_percent?.toFixed(1)}%</div>
                </div>
              </div>
              <div className={styles.metricItem}>
                <i className="fas fa-hdd"></i>
                <div>
                  <div className={styles.metricLabel}>Disk</div>
                  <div className={styles.metricValue}>{metrics.system.disk?.usage_percent?.toFixed(1)}%</div>
                </div>
              </div>
              <div className={styles.metricItem}>
                <i className="fas fa-cube"></i>
                <div>
                  <div className={styles.metricLabel}>IPFS</div>
                  <div className={styles.metricValue}>{metrics.system.ipfs?.storage_percent?.toFixed(1)}%</div>
                </div>
              </div>
            </div>
          ) : (
            <p style={{ color: '#888' }}>Loading metrics...</p>
          )}
          <button
            onClick={() => setShowMetricsModal(true)}
            className={styles.actionButton}
            style={{ marginTop: '15px' }}
          >
            <i className="fas fa-chart-bar" style={{ marginRight: '8px' }}></i>
            View Detailed Metrics
          </button>
        </GlassyCard>

        {/* Recent Logs */}
        <GlassyCard className={styles.logsCard}>
          <h3 className={styles.cardTitle}>
            <i className="fas fa-file-alt" style={{ marginRight: '10px', color: '#007bff' }}></i>
            Recent Logs
          </h3>
          <div className={styles.logsPreview}>
            {logs.length === 0 ? (
              <p style={{ color: '#888', textAlign: 'center' }}>No recent logs</p>
            ) : (
              logs.slice(0, 5).map((log, index) => (
                <div key={index} className={styles.logPreviewItem}>
                  <span className={`${styles.logLevel} ${styles[log.level]}`}>
                    {log.level?.toUpperCase()}
                  </span>
                  <span className={styles.logMessage}>{log.message}</span>
                </div>
              ))
            )}
          </div>
          <button
            onClick={() => setShowLogsModal(true)}
            className={styles.actionButton}
            style={{ marginTop: '15px' }}
          >
            <i className="fas fa-list" style={{ marginRight: '8px' }}></i>
            View All Logs
          </button>
        </GlassyCard>

        {/* Actions */}
        <GlassyCard className={styles.actionsCard}>
          <h3 className={styles.cardTitle}>
            <i className="fas fa-tools" style={{ marginRight: '10px', color: '#007bff' }}></i>
            Quick Actions
          </h3>
          <div className={styles.actionsGrid}>
            <button className={styles.actionButton} onClick={fetchAllData}>
              <i className="fas fa-sync-alt" style={{ marginRight: '8px' }}></i>
              Refresh Data
            </button>
            <button className={styles.actionButton}>
              <i className="fas fa-download" style={{ marginRight: '8px' }}></i>
              Export Report
            </button>
            <button className={styles.actionButton}>
              <i className="fas fa-bell" style={{ marginRight: '8px' }}></i>
              Configure Alerts
            </button>
            <button className={styles.actionButton}>
              <i className="fas fa-cog" style={{ marginRight: '8px' }}></i>
              Settings
            </button>
          </div>
        </GlassyCard>
      </div>

      {/* Modals */}
      {showServiceModal && <ServiceDetailModal />}
      {showMetricsModal && <MetricsModal />}
      {showLogsModal && <LogsModal />}
    </PageLayout>
  );
}
