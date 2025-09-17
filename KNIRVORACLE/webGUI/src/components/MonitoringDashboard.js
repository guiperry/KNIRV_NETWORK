import React, { useState, useEffect, useRef } from 'react';
import axios from 'axios';
import { useBackend } from '../contexts/BackendContext';
import GlassyCard from './GlassyCard';
import styles from './MonitoringDashboard.module.css';

const MonitoringDashboard = () => {
  const { isRunning, oracleStatus } = useBackend();
  const [networkStatus, setNetworkStatus] = useState(null);
  const [economicsMetrics, setEconomicsMetrics] = useState(null);
  const [systemMetrics, setSystemMetrics] = useState(null);
  const [tunnelStatus, setTunnelStatus] = useState(null);
  const [paymentStatus, setPaymentStatus] = useState(null);
  const [realTimeData, setRealTimeData] = useState([]);
  const [isMonitoring, setIsMonitoring] = useState(false);
  const intervalRef = useRef(null);

  useEffect(() => {
    if (isRunning) {
      fetchAllMetrics();
      startRealTimeMonitoring();
    }

    return () => {
      stopRealTimeMonitoring();
    };
  }, [isRunning]);

  const fetchAllMetrics = async () => {
    try {
      const [networkRes, economicsRes, systemRes, tunnelRes, paymentRes] = await Promise.all([
        axios.get('/api/network/status').catch(() => ({ data: null })),
        axios.get('/api/economics/metrics').catch(() => ({ data: null })),
        axios.get('/api/system/metrics').catch(() => ({ data: null })),
        axios.get('/api/tunnel/status').catch(() => ({ data: null })),
        axios.get('/api/payments/status').catch(() => ({ data: null })),
      ]);

      setNetworkStatus(networkRes.data);
      setEconomicsMetrics(economicsRes.data);
      setSystemMetrics(systemRes.data);
      setTunnelStatus(tunnelRes.data);
      setPaymentStatus(paymentRes.data);
    } catch (error) {
      console.error('Error fetching metrics:', error);
    }
  };

  const startRealTimeMonitoring = () => {
    if (intervalRef.current) return;

    setIsMonitoring(true);
    intervalRef.current = setInterval(() => {
      fetchAllMetrics();
      
      // Add real-time data point
      const timestamp = new Date();
      const dataPoint = {
        timestamp,
        services_running: oracleStatus?.running_services || 0,
        total_services: oracleStatus?.total_services || 0,
        health_percentage: oracleStatus ? 
          (oracleStatus.running_services / oracleStatus.total_services) * 100 : 0,
      };

      setRealTimeData(prev => {
        const newData = [...prev, dataPoint];
        // Keep only last 20 data points
        return newData.slice(-20);
      });
    }, 5000); // Update every 5 seconds
  };

  const stopRealTimeMonitoring = () => {
    if (intervalRef.current) {
      clearInterval(intervalRef.current);
      intervalRef.current = null;
    }
    setIsMonitoring(false);
  };

  const toggleMonitoring = () => {
    if (isMonitoring) {
      stopRealTimeMonitoring();
    } else {
      startRealTimeMonitoring();
    }
  };

  const formatUptime = (startTime) => {
    if (!startTime) return 'N/A';
    const now = new Date();
    const start = new Date(startTime);
    const diff = now - start;
    const hours = Math.floor(diff / (1000 * 60 * 60));
    const minutes = Math.floor((diff % (1000 * 60 * 60)) / (1000 * 60));
    return `${hours}h ${minutes}m`;
  };

  const getHealthColor = (percentage) => {
    if (percentage >= 80) return '#4CAF50';
    if (percentage >= 60) return '#ff9800';
    return '#f44336';
  };

  if (!isRunning) {
    return (
      <div className={styles.container}>
        <GlassyCard>
          <div className={styles.errorState}>
            <h2>Monitoring Unavailable</h2>
            <p>Oracle backend is not connected. Please ensure the backend is running.</p>
          </div>
        </GlassyCard>
      </div>
    );
  }

  return (
    <div className={styles.container}>
      <div className={styles.header}>
        <h2>Real-Time Monitoring Dashboard</h2>
        <div className={styles.controls}>
          <button
            className={`${styles.monitoringBtn} ${isMonitoring ? styles.active : ''}`}
            onClick={toggleMonitoring}
          >
            {isMonitoring ? 'Stop Monitoring' : 'Start Monitoring'}
          </button>
          <button className={styles.refreshBtn} onClick={fetchAllMetrics}>
            Refresh
          </button>
        </div>
      </div>

      {/* System Overview */}
      <div className={styles.overviewGrid}>
        <GlassyCard>
          <h3>System Health</h3>
          <div className={styles.healthMetric}>
            <div 
              className={styles.healthCircle}
              style={{ 
                background: `conic-gradient(${getHealthColor(systemMetrics?.system?.service_health || 0)} ${(systemMetrics?.system?.service_health || 0) * 3.6}deg, rgba(255,255,255,0.1) 0deg)`
              }}
            >
              <span className={styles.healthPercentage}>
                {Math.round(systemMetrics?.system?.service_health || 0)}%
              </span>
            </div>
            <div className={styles.healthDetails}>
              <div>Running: {systemMetrics?.system?.running_services || 0}</div>
              <div>Total: {systemMetrics?.system?.total_services || 0}</div>
            </div>
          </div>
        </GlassyCard>

        <GlassyCard>
          <h3>Network Status</h3>
          <div className={styles.networkInfo}>
            <div className={styles.statusItem}>
              <span>Status:</span>
              <span className={networkStatus?.service_running ? styles.online : styles.offline}>
                {networkStatus?.service_running ? 'Online' : 'Offline'}
              </span>
            </div>
            <div className={styles.statusItem}>
              <span>Network:</span>
              <span>{networkStatus?.network || 'Unknown'}</span>
            </div>
            <div className={styles.statusItem}>
              <span>Testnet:</span>
              <span>{networkStatus?.testnet_enabled ? 'Enabled' : 'Disabled'}</span>
            </div>
          </div>
        </GlassyCard>

        <GlassyCard>
          <h3>Economics</h3>
          <div className={styles.economicsInfo}>
            {economicsMetrics?.token_supply ? (
              <>
                <div className={styles.tokenMetric}>
                  <span>Total Supply:</span>
                  <span>{parseInt(economicsMetrics.token_supply.total).toLocaleString()}</span>
                </div>
                <div className={styles.tokenMetric}>
                  <span>Circulating:</span>
                  <span>{parseInt(economicsMetrics.token_supply.circulating).toLocaleString()}</span>
                </div>
                <div className={styles.tokenMetric}>
                  <span>NRN Price:</span>
                  <span>${economicsMetrics.price_data?.nrn_usd || 'N/A'}</span>
                </div>
              </>
            ) : (
              <div className={styles.noData}>Economics data unavailable</div>
            )}
          </div>
        </GlassyCard>

        <GlassyCard>
          <h3>Services Status</h3>
          <div className={styles.servicesOverview}>
            <div className={styles.serviceStatus}>
              <span>Tunnel Registry:</span>
              <span className={tunnelStatus?.service_running ? styles.online : styles.offline}>
                {tunnelStatus?.service_running ? 'Running' : 'Stopped'}
              </span>
            </div>
            <div className={styles.serviceStatus}>
              <span>Payment Gateway:</span>
              <span className={paymentStatus?.service_running ? styles.online : styles.offline}>
                {paymentStatus?.service_running ? 'Running' : 'Stopped'}
              </span>
            </div>
            <div className={styles.serviceStatus}>
              <span>Network Monitor:</span>
              <span className={networkStatus?.service_running ? styles.online : styles.offline}>
                {networkStatus?.service_running ? 'Running' : 'Stopped'}
              </span>
            </div>
          </div>
        </GlassyCard>
      </div>

      {/* Real-Time Chart */}
      {realTimeData.length > 0 && (
        <GlassyCard>
          <h3>Real-Time Service Health</h3>
          <div className={styles.chartContainer}>
            <div className={styles.chart}>
              <svg width="100%" height="200" viewBox="0 0 800 200">
                <defs>
                  <linearGradient id="healthGradient" x1="0%" y1="0%" x2="0%" y2="100%">
                    <stop offset="0%" stopColor="#4CAF50" stopOpacity="0.8"/>
                    <stop offset="100%" stopColor="#4CAF50" stopOpacity="0.1"/>
                  </linearGradient>
                </defs>
                
                {/* Grid lines */}
                {[0, 25, 50, 75, 100].map(y => (
                  <line
                    key={y}
                    x1="0"
                    y1={200 - (y * 2)}
                    x2="800"
                    y2={200 - (y * 2)}
                    stroke="rgba(255,255,255,0.1)"
                    strokeWidth="1"
                  />
                ))}
                
                {/* Health percentage line */}
                <polyline
                  fill="none"
                  stroke="#4CAF50"
                  strokeWidth="3"
                  points={realTimeData.map((point, index) => {
                    const x = (index / (realTimeData.length - 1)) * 800;
                    const y = 200 - (point.health_percentage * 2);
                    return `${x},${y}`;
                  }).join(' ')}
                />
                
                {/* Fill area */}
                <polygon
                  fill="url(#healthGradient)"
                  points={`0,200 ${realTimeData.map((point, index) => {
                    const x = (index / (realTimeData.length - 1)) * 800;
                    const y = 200 - (point.health_percentage * 2);
                    return `${x},${y}`;
                  }).join(' ')} 800,200`}
                />
              </svg>
            </div>
            <div className={styles.chartLabels}>
              <span>0%</span>
              <span>25%</span>
              <span>50%</span>
              <span>75%</span>
              <span>100%</span>
            </div>
          </div>
        </GlassyCard>
      )}

      {/* Detailed Metrics */}
      <div className={styles.detailsGrid}>
        <GlassyCard>
          <h3>Network Details</h3>
          <div className={styles.detailsList}>
            <div className={styles.detailItem}>
              <span>Chain ID:</span>
              <span>{oracleStatus?.chain_id || 'Unknown'}</span>
            </div>
            <div className={styles.detailItem}>
              <span>Role:</span>
              <span>{oracleStatus?.role || 'Unknown'}</span>
            </div>
            <div className={styles.detailItem}>
              <span>Port:</span>
              <span>{networkStatus?.port || 'N/A'}</span>
            </div>
            <div className={styles.detailItem}>
              <span>Uptime:</span>
              <span>{formatUptime(networkStatus?.uptime)}</span>
            </div>
          </div>
        </GlassyCard>

        <GlassyCard>
          <h3>Payment Gateway</h3>
          <div className={styles.detailsList}>
            <div className={styles.detailItem}>
              <span>Status:</span>
              <span className={paymentStatus?.service_running ? styles.online : styles.offline}>
                {paymentStatus?.service_running ? 'Active' : 'Inactive'}
              </span>
            </div>
            <div className={styles.detailItem}>
              <span>Stripe:</span>
              <span className={paymentStatus?.stripe_enabled ? styles.enabled : styles.disabled}>
                {paymentStatus?.stripe_enabled ? 'Enabled' : 'Disabled'}
              </span>
            </div>
            <div className={styles.detailItem}>
              <span>Coinbase:</span>
              <span className={paymentStatus?.coinbase_enabled ? styles.enabled : styles.disabled}>
                {paymentStatus?.coinbase_enabled ? 'Enabled' : 'Disabled'}
              </span>
            </div>
            <div className={styles.detailItem}>
              <span>Port:</span>
              <span>{paymentStatus?.port || 'N/A'}</span>
            </div>
          </div>
        </GlassyCard>
      </div>

      {isMonitoring && (
        <div className={styles.monitoringIndicator}>
          <div className={styles.pulse}></div>
          <span>Live Monitoring Active</span>
        </div>
      )}
    </div>
  );
};

export default MonitoringDashboard;
