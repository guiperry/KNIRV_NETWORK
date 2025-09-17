import React, { useState, useEffect } from 'react';
import { useBackend } from '../contexts/BackendContext';
import GlassyCard from './GlassyCard';
import styles from './ServiceManagement.module.css';

const ServiceManagement = () => {
  const {
    services,
    fetchServices,
    startService,
    stopService,
    restartService,
  } = useBackend();

  const [selectedServices, setSelectedServices] = useState(new Set());
  const [actionLoading, setActionLoading] = useState(false);
  const [filterType, setFilterType] = useState('all');
  const [filterStatus, setFilterStatus] = useState('all');

  useEffect(() => {
    fetchServices();
  }, [fetchServices]);

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

  const handleBulkAction = async (action) => {
    if (selectedServices.size === 0) {
      alert('Please select at least one service');
      return;
    }

    setActionLoading(true);
    const promises = Array.from(selectedServices).map(serviceName => {
      switch (action) {
        case 'start':
          return startService(serviceName);
        case 'stop':
          return stopService(serviceName);
        case 'restart':
          return restartService(serviceName);
        default:
          return Promise.resolve();
      }
    });

    try {
      await Promise.all(promises);
      setSelectedServices(new Set());
    } catch (error) {
      console.error(`Failed to ${action} selected services:`, error);
      alert(`Failed to ${action} some services: ${error.message}`);
    } finally {
      setActionLoading(false);
    }
  };

  const toggleServiceSelection = (serviceName) => {
    const newSelection = new Set(selectedServices);
    if (newSelection.has(serviceName)) {
      newSelection.delete(serviceName);
    } else {
      newSelection.add(serviceName);
    }
    setSelectedServices(newSelection);
  };

  const selectAllServices = () => {
    const filteredServices = getFilteredServices();
    setSelectedServices(new Set(filteredServices.map(s => s.name)));
  };

  const clearSelection = () => {
    setSelectedServices(new Set());
  };

  const getFilteredServices = () => {
    return services.filter(service => {
      const typeMatch = filterType === 'all' || service.type === filterType;
      const statusMatch = filterStatus === 'all' || 
        (filterStatus === 'running' && service.running) ||
        (filterStatus === 'stopped' && !service.running);
      return typeMatch && statusMatch;
    });
  };

  const getStatusColor = (running) => {
    return running ? '#4CAF50' : '#f44336';
  };

  const getStatusText = (running) => {
    return running ? 'Running' : 'Stopped';
  };

  const filteredServices = getFilteredServices();
  const runningCount = services.filter(s => s.running).length;
  const stoppedCount = services.length - runningCount;

  return (
    <div className={styles.container}>
      <div className={styles.header}>
        <h2>Service Management</h2>
        <div className={styles.stats}>
          <span className={styles.stat}>
            Total: <strong>{services.length}</strong>
          </span>
          <span className={styles.stat} style={{ color: '#4CAF50' }}>
            Running: <strong>{runningCount}</strong>
          </span>
          <span className={styles.stat} style={{ color: '#f44336' }}>
            Stopped: <strong>{stoppedCount}</strong>
          </span>
        </div>
      </div>

      {/* Filters and Bulk Actions */}
      <GlassyCard>
        <div className={styles.controls}>
          <div className={styles.filters}>
            <div className={styles.filterGroup}>
              <label>Type:</label>
              <select 
                value={filterType} 
                onChange={(e) => setFilterType(e.target.value)}
                className={styles.filterSelect}
              >
                <option value="all">All Types</option>
                <option value="nodejs">Node.js</option>
                <option value="binary">Binary</option>
              </select>
            </div>
            <div className={styles.filterGroup}>
              <label>Status:</label>
              <select 
                value={filterStatus} 
                onChange={(e) => setFilterStatus(e.target.value)}
                className={styles.filterSelect}
              >
                <option value="all">All Status</option>
                <option value="running">Running</option>
                <option value="stopped">Stopped</option>
              </select>
            </div>
          </div>

          <div className={styles.selectionControls}>
            <button 
              className={styles.selectionBtn}
              onClick={selectAllServices}
              disabled={filteredServices.length === 0}
            >
              Select All
            </button>
            <button 
              className={styles.selectionBtn}
              onClick={clearSelection}
              disabled={selectedServices.size === 0}
            >
              Clear Selection
            </button>
            <span className={styles.selectionCount}>
              {selectedServices.size} selected
            </span>
          </div>

          <div className={styles.bulkActions}>
            <button 
              className={`${styles.bulkBtn} ${styles.startBtn}`}
              onClick={() => handleBulkAction('start')}
              disabled={selectedServices.size === 0 || actionLoading}
            >
              Start Selected
            </button>
            <button 
              className={`${styles.bulkBtn} ${styles.stopBtn}`}
              onClick={() => handleBulkAction('stop')}
              disabled={selectedServices.size === 0 || actionLoading}
            >
              Stop Selected
            </button>
            <button 
              className={`${styles.bulkBtn} ${styles.restartBtn}`}
              onClick={() => handleBulkAction('restart')}
              disabled={selectedServices.size === 0 || actionLoading}
            >
              Restart Selected
            </button>
          </div>
        </div>
      </GlassyCard>

      {/* Services List */}
      <div className={styles.servicesList}>
        {filteredServices.length > 0 ? (
          filteredServices.map(service => (
            <GlassyCard key={service.name} className={styles.serviceCard}>
              <div className={styles.serviceHeader}>
                <div className={styles.serviceInfo}>
                  <input
                    type="checkbox"
                    checked={selectedServices.has(service.name)}
                    onChange={() => toggleServiceSelection(service.name)}
                    className={styles.serviceCheckbox}
                  />
                  <div className={styles.serviceDetails}>
                    <div className={styles.serviceName}>
                      <span 
                        className={styles.statusIndicator} 
                        style={{ backgroundColor: getStatusColor(service.running) }}
                      ></span>
                      <h3>{service.name}</h3>
                      <span className={`${styles.serviceType} ${styles[service.type]}`}>
                        {service.type.toUpperCase()}
                      </span>
                    </div>
                    <div className={styles.serviceMetrics}>
                      <span className={styles.metric}>
                        Status: <strong style={{ color: getStatusColor(service.running) }}>
                          {getStatusText(service.running)}
                        </strong>
                      </span>
                      {service.port && (
                        <span className={styles.metric}>
                          Port: <strong>{service.port}</strong>
                        </span>
                      )}
                      {service.pid && (
                        <span className={styles.metric}>
                          PID: <strong>{service.pid}</strong>
                        </span>
                      )}
                      {service.start_time && (
                        <span className={styles.metric}>
                          Started: <strong>{new Date(service.start_time).toLocaleString()}</strong>
                        </span>
                      )}
                    </div>
                  </div>
                </div>

                <div className={styles.serviceActions}>
                  {!service.running ? (
                    <button
                      className={`${styles.actionBtn} ${styles.startBtn}`}
                      onClick={() => handleServiceAction(service.name, 'start')}
                      disabled={actionLoading}
                    >
                      Start
                    </button>
                  ) : (
                    <>
                      <button
                        className={`${styles.actionBtn} ${styles.stopBtn}`}
                        onClick={() => handleServiceAction(service.name, 'stop')}
                        disabled={actionLoading}
                      >
                        Stop
                      </button>
                      <button
                        className={`${styles.actionBtn} ${styles.restartBtn}`}
                        onClick={() => handleServiceAction(service.name, 'restart')}
                        disabled={actionLoading}
                      >
                        Restart
                      </button>
                    </>
                  )}
                </div>
              </div>

              {service.error && (
                <div className={styles.serviceError}>
                  <strong>Error:</strong> {service.error}
                </div>
              )}
            </GlassyCard>
          ))
        ) : (
          <GlassyCard>
            <div className={styles.noServices}>
              <p>No services match the current filters.</p>
            </div>
          </GlassyCard>
        )}
      </div>

      {actionLoading && (
        <div className={styles.loadingOverlay}>
          <div className={styles.loadingSpinner}>
            <div className={styles.spinner}></div>
            <p>Processing service actions...</p>
          </div>
        </div>
      )}
    </div>
  );
};

export default ServiceManagement;
