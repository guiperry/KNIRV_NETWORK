import React, { useState, useEffect } from 'react';
import { useBackend } from '../contexts/BackendContext';
import GlassyCard from './GlassyCard';
import DataTable from './DataTable';
import styles from './UnifiedDashboard.module.css';

const UnifiedDashboard = () => {
  const {
    isRunning,
    oracleStatus,
    services,
    fetchOracleStatus,
    fetchServices,
    startService,
    stopService,
    restartService,
  } = useBackend();

  const [selectedService, setSelectedService] = useState(null);
  const [actionLoading, setActionLoading] = useState(false);
  const [refreshInterval, setRefreshInterval] = useState(null);

  useEffect(() => {
    if (isRunning) {
      // Set up auto-refresh every 10 seconds
      const interval = setInterval(() => {
        fetchOracleStatus();
        fetchServices();
      }, 10000);
      setRefreshInterval(interval);

      return () => {
        if (interval) clearInterval(interval);
      };
    }
  }, [isRunning, fetchOracleStatus, fetchServices]);

  const handleServiceAction = async (serviceName, action) => {
    setActionLoading(true);
    try {
      switch (action) {
        case 'start':
          await startService(serviceName);
          break;
        case 'stop':
          await stopService(serviceName);
          break;
        case 'restart':
          await restartService(serviceName);
          break;
        default:
          console.warn(`Unknown action: ${action}`);
      }
    } catch (error) {
      console.error(`Failed to ${action} service ${serviceName}:`, error);
      alert(`Failed to ${action} service: ${error.message}`);
    } finally {
      setActionLoading(false);
    }
  };

  const getStatusColor = (running) => {
    return running ? '#4CAF50' : '#f44336';
  };

  const getStatusText = (running) => {
    return running ? 'Running' : 'Stopped';
  };

  const serviceColumns = [
    {
      key: 'name',
      label: 'Service Name',
      render: (value, row) => (
        <div className={styles.serviceName}>
          <span className={styles.serviceIcon} style={{ backgroundColor: getStatusColor(row.running) }}></span>
          {value}
        </div>
      ),
    },
    {
      key: 'type',
      label: 'Type',
      render: (value) => (
        <span className={`${styles.serviceType} ${styles[value]}`}>
          {value.toUpperCase()}
        </span>
      ),
    },
    {
      key: 'running',
      label: 'Status',
      render: (value) => (
        <span className={styles.status} style={{ color: getStatusColor(value) }}>
          {getStatusText(value)}
        </span>
      ),
    },
    {
      key: 'port',
      label: 'Port',
      render: (value) => value || 'N/A',
    },
    {
      key: 'pid',
      label: 'PID',
      render: (value) => value || 'N/A',
    },
    {
      key: 'actions',
      label: 'Actions',
      render: (_, row) => (
        <div className={styles.actionButtons}>
          {!row.running ? (
            <button
              className={`${styles.actionBtn} ${styles.startBtn}`}
              onClick={() => handleServiceAction(row.name, 'start')}
              disabled={actionLoading}
            >
              Start
            </button>
          ) : (
            <>
              <button
                className={`${styles.actionBtn} ${styles.stopBtn}`}
                onClick={() => handleServiceAction(row.name, 'stop')}
                disabled={actionLoading}
              >
                Stop
              </button>
              <button
                className={`${styles.actionBtn} ${styles.restartBtn}`}
                onClick={() => handleServiceAction(row.name, 'restart')}
                disabled={actionLoading}
              >
                Restart
              </button>
            </>
          )}
        </div>
      ),
    },
  ];

  if (!isRunning) {
    return (
      <div className={styles.container}>
        <GlassyCard>
          <div className={styles.errorState}>
            <h2>Oracle Backend Not Connected</h2>
            <p>Please ensure the KNIRV Oracle backend is running and try again.</p>
          </div>
        </GlassyCard>
      </div>
    );
  }

  return (
    <div className={styles.container}>
      <div className={styles.header}>
        <h1>KNIRV Oracle Unified Dashboard</h1>
        <div className={styles.refreshControls}>
          <button
            className={styles.refreshBtn}
            onClick={() => {
              fetchOracleStatus();
              fetchServices();
            }}
          >
            Refresh
          </button>
        </div>
      </div>

      {/* Oracle Status Overview */}
      {oracleStatus && (
        <div className={styles.statusGrid}>
          <GlassyCard>
            <h3>Oracle Status</h3>
            <div className={styles.statusInfo}>
              <div className={styles.statusItem}>
                <label>Chain ID:</label>
                <span>{oracleStatus.chain_id}</span>
              </div>
              <div className={styles.statusItem}>
                <label>Role:</label>
                <span className={styles.role}>{oracleStatus.role}</span>
              </div>
              <div className={styles.statusItem}>
                <label>Root Node:</label>
                <span>{oracleStatus.is_root ? 'Yes' : 'No'}</span>
              </div>
              <div className={styles.statusItem}>
                <label>Bootnode:</label>
                <span>{oracleStatus.is_bootnode ? 'Yes' : 'No'}</span>
              </div>
              <div className={styles.statusItem}>
                <label>Testnet:</label>
                <span>{oracleStatus.testnet_enabled ? 'Enabled' : 'Disabled'}</span>
              </div>
            </div>
          </GlassyCard>

          <GlassyCard>
            <h3>Service Summary</h3>
            <div className={styles.serviceSummary}>
              <div className={styles.summaryItem}>
                <span className={styles.summaryNumber}>{oracleStatus.total_services}</span>
                <span className={styles.summaryLabel}>Total Services</span>
              </div>
              <div className={styles.summaryItem}>
                <span className={styles.summaryNumber} style={{ color: '#4CAF50' }}>
                  {oracleStatus.running_services}
                </span>
                <span className={styles.summaryLabel}>Running</span>
              </div>
              <div className={styles.summaryItem}>
                <span className={styles.summaryNumber} style={{ color: '#f44336' }}>
                  {oracleStatus.total_services - oracleStatus.running_services}
                </span>
                <span className={styles.summaryLabel}>Stopped</span>
              </div>
            </div>
          </GlassyCard>
        </div>
      )}

      {/* Services Management */}
      <GlassyCard>
        <h3>Service Management</h3>
        {services.length > 0 ? (
          <DataTable
            data={services}
            columns={serviceColumns}
            className={styles.servicesTable}
          />
        ) : (
          <div className={styles.noServices}>
            <p>No services found. Please check your Oracle configuration.</p>
          </div>
        )}
      </GlassyCard>

      {/* Quick Actions */}
      <div className={styles.quickActions}>
        <GlassyCard>
          <h3>Quick Actions</h3>
          <div className={styles.actionGrid}>
            <button
              className={`${styles.quickActionBtn} ${styles.startAllBtn}`}
              onClick={() => {
                services.forEach(service => {
                  if (!service.running) {
                    handleServiceAction(service.name, 'start');
                  }
                });
              }}
              disabled={actionLoading}
            >
              Start All Services
            </button>
            <button
              className={`${styles.quickActionBtn} ${styles.stopAllBtn}`}
              onClick={() => {
                services.forEach(service => {
                  if (service.running) {
                    handleServiceAction(service.name, 'stop');
                  }
                });
              }}
              disabled={actionLoading}
            >
              Stop All Services
            </button>
            <button
              className={`${styles.quickActionBtn} ${styles.restartAllBtn}`}
              onClick={() => {
                services.forEach(service => {
                  if (service.running) {
                    handleServiceAction(service.name, 'restart');
                  }
                });
              }}
              disabled={actionLoading}
            >
              Restart All Services
            </button>
          </div>
        </GlassyCard>
      </div>

      {actionLoading && (
        <div className={styles.loadingOverlay}>
          <div className={styles.loadingSpinner}>
            <div className={styles.spinner}></div>
            <p>Processing service action...</p>
          </div>
        </div>
      )}
    </div>
  );
};

export default UnifiedDashboard;
