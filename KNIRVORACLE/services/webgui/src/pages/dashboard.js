import React, { useState, useEffect } from 'react';
import { useNavigation } from '../hooks/useNavigation';
import PageLayout from '../components/PageLayout';
import PageHeader from '../components/PageHeader';
import GlassyCard from '../components/GlassyCard';
import styles from './dashboard.module.css';

export default function Dashboard() {
  const { activePage } = useNavigation('dashboard');
  const [searchQuery, setSearchQuery] = useState('');
  const [knirvControllerConnected, setKnirvControllerConnected] = useState(false);
  const [userApiEndpoints, setUserApiEndpoints] = useState([]);
  const brightBlue = '#007bff';

  const handleSearch = (query) => {
    setSearchQuery(query);
  };

  // Check KNIRVCONTROLLER connection status
  useEffect(() => {
    const checkKnirvControllerConnection = async () => {
      try {
        // This would be replaced with actual connection check
        const isConnected = localStorage.getItem('knirvControllerConnected') === 'true';
        setKnirvControllerConnected(isConnected);
      } catch (error) {
        console.error('Error checking KNIRVCONTROLLER connection:', error);
        setKnirvControllerConnected(false);
      }
    };

    // Load user API endpoints from TunnelRegistry
    const loadUserApiEndpoints = () => {
      try {
        const endpoints = JSON.parse(localStorage.getItem('userApiEndpoints') || '[]');
        setUserApiEndpoints(endpoints);
      } catch (error) {
        console.error('Error loading user API endpoints:', error);
        setUserApiEndpoints([]);
      }
    };

    checkKnirvControllerConnection();
    loadUserApiEndpoints();
  }, []);

  const generateQRCode = () => {
    // Generate QR code for KNIRVCONTROLLER connection
    alert('QR Code generation for KNIRVCONTROLLER connection would be implemented here');
  };

  return (
    <PageLayout activePage={activePage} pageTitle="Dashboard" onSearch={handleSearch}>
      <PageHeader
        title="KNIRV Network Dashboard"
        subtitle="Monitor your network status and manage your endpoints"
        titleColor={brightBlue}
      />

      <div className={styles.gridContainer}>
        {/* KNIRVCONTROLLER Connection Status */}
        <GlassyCard className={styles.dashboardCard}>
          <h3 className={styles.cardTitle}>
            <i className="fas fa-mobile-alt" style={{ marginRight: '10px', color: brightBlue }}></i>
            KNIRVCONTROLLER Connection
          </h3>
          <div style={{ display: 'flex', alignItems: 'center', marginBottom: '15px' }}>
            <div
              style={{
                width: '12px',
                height: '12px',
                borderRadius: '50%',
                backgroundColor: knirvControllerConnected ? '#28a745' : '#dc3545',
                marginRight: '10px'
              }}
            ></div>
            <span style={{ color: knirvControllerConnected ? '#28a745' : '#dc3545' }}>
              {knirvControllerConnected ? 'Connected' : 'Disconnected'}
            </span>
          </div>
          {!knirvControllerConnected && (
            <button
              onClick={generateQRCode}
              style={{
                backgroundColor: brightBlue,
                color: 'white',
                border: 'none',
                padding: '10px 15px',
                borderRadius: '5px',
                cursor: 'pointer'
              }}
            >
              <i className="fas fa-qrcode" style={{ marginRight: '8px' }}></i>
              Connect via QR Code
            </button>
          )}
        </GlassyCard>

        {/* User's Personal API Endpoints */}
        <GlassyCard className={styles.dashboardCard}>
          <h3 className={styles.cardTitle}>
            <i className="fas fa-network-wired" style={{ marginRight: '10px', color: brightBlue }}></i>
            Personal API Endpoints
          </h3>
          {userApiEndpoints.length > 0 ? (
            <div>
              {userApiEndpoints.map((endpoint, index) => (
                <div key={index} style={{ marginBottom: '10px', padding: '10px', backgroundColor: 'rgba(255,255,255,0.1)', borderRadius: '5px' }}>
                  <div style={{ fontWeight: 'bold', color: brightBlue }}>{endpoint.name}</div>
                  <div style={{ fontSize: '0.9em', color: '#ccc' }}>{endpoint.url}</div>
                  <div style={{ fontSize: '0.8em', color: '#999' }}>Status: {endpoint.status || 'Active'}</div>
                </div>
              ))}
            </div>
          ) : (
            <p style={{ color: '#ccc' }}>
              No API endpoints configured. Connect your KNIRVCONTROLLER or configure through TunnelRegistry.
            </p>
          )}
        </GlassyCard>

        {/* Network Status */}
        <GlassyCard className={styles.dashboardCard}>
          <h3 className={styles.cardTitle}>
            <i className="fas fa-globe" style={{ marginRight: '10px', color: brightBlue }}></i>
            Network Status
          </h3>
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '10px' }}>
            <div>
              <div style={{ color: '#ccc', fontSize: '0.9em' }}>Network</div>
              <div style={{ color: brightBlue, fontWeight: 'bold' }}>KNIRV Testnet</div>
            </div>
            <div>
              <div style={{ color: '#ccc', fontSize: '0.9em' }}>Block Height</div>
              <div style={{ color: brightBlue, fontWeight: 'bold' }}>1,234,567</div>
            </div>
            <div>
              <div style={{ color: '#ccc', fontSize: '0.9em' }}>Peers</div>
              <div style={{ color: '#28a745', fontWeight: 'bold' }}>42</div>
            </div>
            <div>
              <div style={{ color: '#ccc', fontSize: '0.9em' }}>Sync Status</div>
              <div style={{ color: '#28a745', fontWeight: 'bold' }}>Synced</div>
            </div>
          </div>
        </GlassyCard>

        {/* Quick Actions */}
        <GlassyCard className={styles.dashboardCard}>
          <h3 className={styles.cardTitle}>
            <i className="fas fa-bolt" style={{ marginRight: '10px', color: brightBlue }}></i>
            Quick Actions
          </h3>
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '10px' }}>
            <button
              style={{
                backgroundColor: 'rgba(0, 123, 255, 0.2)',
                border: '1px solid rgba(0, 123, 255, 0.5)',
                color: brightBlue,
                padding: '10px',
                borderRadius: '5px',
                cursor: 'pointer'
              }}
            >
              <i className="fas fa-robot" style={{ display: 'block', marginBottom: '5px' }}></i>
              Models
            </button>
            <button
              style={{
                backgroundColor: 'rgba(0, 123, 255, 0.2)',
                border: '1px solid rgba(0, 123, 255, 0.5)',
                color: brightBlue,
                padding: '10px',
                borderRadius: '5px',
                cursor: 'pointer'
              }}
            >
              <i className="fas fa-chart-line" style={{ display: 'block', marginBottom: '5px' }}></i>
              Monitor
            </button>
            <button
              style={{
                backgroundColor: 'rgba(0, 123, 255, 0.2)',
                border: '1px solid rgba(0, 123, 255, 0.5)',
                color: brightBlue,
                padding: '10px',
                borderRadius: '5px',
                cursor: 'pointer'
              }}
            >
              <i className="fas fa-store" style={{ display: 'block', marginBottom: '5px' }}></i>
              Marketplace
            </button>
            <button
              style={{
                backgroundColor: 'rgba(0, 123, 255, 0.2)',
                border: '1px solid rgba(0, 123, 255, 0.5)',
                color: brightBlue,
                padding: '10px',
                borderRadius: '5px',
                cursor: 'pointer'
              }}
            >
              <i className="fas fa-vault" style={{ display: 'block', marginBottom: '5px' }}></i>
              Vault
            </button>
          </div>
        </GlassyCard>
      </div>
    </PageLayout>
  );
}