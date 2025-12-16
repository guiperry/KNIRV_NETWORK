import React, { useState } from 'react';
import Head from 'next/head';
import PageLayout from '../components/PageLayout';
import RoleProtectedRoute from '../components/RoleProtectedRoute';
import { useRole } from '../contexts/RoleContext';
import { useRouter } from 'next/router';
import styles from './network-admin.module.css';

export default function NetworkAdmin() {
  const { role } = useRole();
  const router = useRouter();
  const [activeTab, setActiveTab] = useState('payment');
  
  // Redirect if not Root role
  React.useEffect(() => {
    if (role !== 'Root') {
      router.push('/');
    }
  }, [role, router]);

  if (role !== 'Root') {
    return null; // Don't render anything while redirecting
  }

  return (
    <RoleProtectedRoute>
      <PageLayout activePage="network-admin" pageTitle="Network Administration">
        <Head>
          <title>Network Administration - KNIRVCHAIN</title>
        </Head>
        
        <div className={styles.container}>
          <div className={styles.tabContainer}>
            <div 
              className={`${styles.tab} ${activeTab === 'payment' ? styles.activeTab : ''}`}
              onClick={() => setActiveTab('payment')}
            >
              Payment Processor
            </div>
            <div 
              className={`${styles.tab} ${activeTab === 'tunnel' ? styles.activeTab : ''}`}
              onClick={() => setActiveTab('tunnel')}
            >
              Tunnel Relay Registry
            </div>
            <div 
              className={`${styles.tab} ${activeTab === 'blockchain' ? styles.activeTab : ''}`}
              onClick={() => setActiveTab('blockchain')}
            >
              Blockchain Settings
            </div>
          </div>
          
          <div className={styles.tabContent}>
            {activeTab === 'payment' && (
              <div className={styles.section}>
                <h2>Payment Processor Configuration</h2>
                {/* Payment processor settings form */}
                <form className={styles.form}>
                  <div className={styles.formGroup}>
                    <label>Enable Payment Processor</label>
                    <input type="checkbox" />
                  </div>
                  <div className={styles.formGroup}>
                    <label>Webhook Port</label>
                    <input type="number" placeholder="Port number" />
                  </div>
                  <div className={styles.formGroup}>
                    <label>Token Symbol</label>
                    <input type="text" placeholder="e.g., KNRV" />
                  </div>
                  <div className={styles.formGroup}>
                    <label>Token Decimals</label>
                    <input type="number" placeholder="e.g., 18" />
                  </div>
                  <button type="submit" className={styles.button}>Save Changes</button>
                </form>
              </div>
            )}
            
            {activeTab === 'tunnel' && (
              <div className={styles.section}>
                <h2>Tunnel Relay Registry Configuration</h2>
                {/* Tunnel relay settings form */}
                <form className={styles.form}>
                  <div className={styles.formGroup}>
                    <label>Enable Tunnel Registry</label>
                    <input type="checkbox" />
                  </div>
                  <div className={styles.formGroup}>
                    <label>HTTP Port</label>
                    <input type="number" placeholder="Port number" />
                  </div>
                  <div className={styles.formGroup}>
                    <label>Control Port</label>
                    <input type="number" placeholder="Port number" />
                  </div>
                  <div className={styles.formGroup}>
                    <label>Public Relay Port</label>
                    <input type="number" placeholder="Port number" />
                  </div>
                  <div className={styles.formGroup}>
                    <label>STUN Port</label>
                    <input type="number" placeholder="Port number" />
                  </div>
                  <div className={styles.formGroup}>
                    <label>Server Public Host</label>
                    <input type="text" placeholder="e.g., relay.KNIRVCHAIN.com" />
                  </div>
                  <button type="submit" className={styles.button}>Save Changes</button>
                </form>
              </div>
            )}
            
            {activeTab === 'blockchain' && (
              <div className={styles.section}>
                <h2>Blockchain Settings</h2>
                {/* Blockchain settings form */}
                <form className={styles.form}>
                  <div className={styles.formGroup}>
                    <label>Chain ID</label>
                    <input type="text" placeholder="e.g., KNIRVCHAIN" />
                  </div>
                  <div className={styles.formGroup}>
                    <label>Mining Difficulty</label>
                    <input type="number" placeholder="e.g., 4" />
                  </div>
                  <div className={styles.formGroup}>
                    <label>Mining Reward</label>
                    <input type="number" placeholder="e.g., 100" />
                  </div>
                  <div className={styles.formGroup}>
                    <label>Block Time (seconds)</label>
                    <input type="number" placeholder="e.g., 15" />
                  </div>
                  <button type="submit" className={styles.button}>Save Changes</button>
                </form>
              </div>
            )}
          </div>
        </div>
      </PageLayout>
    </RoleProtectedRoute>
  );
}