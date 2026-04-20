import React, { useState } from 'react';
import { useNavigation } from '../hooks/useNavigation';
import PageLayout from '../components/PageLayout';
import PageHeader from '../components/PageHeader';
import GlassyCard from '../components/GlassyCard';
import RoleProtectedRoute from '../components/RoleProtectedRoute';
import styles from './settlement.module.css';

export default function Settlement() {
  const { activePage } = useNavigation('settlement');
  const [searchQuery, setSearchQuery] = useState('');

  const handleSearch = (query) => {
    setSearchQuery(query);
  };
  
  return (
    <RoleProtectedRoute>
      <PageLayout activePage={activePage} pageTitle="Settlement Layer" onSearch={handleSearch}>
        <PageHeader 
          title="Transaction Settlement"
          subtitle="Monitor and manage blockchain settlements"
        />

      {/* Status Section */}
      <GlassyCard darker className={styles.statusContainer}>
        <div className={styles.statusInfo}>
          <h3>Settlement Status</h3>
          <div className={styles.statusIndicator}>
            <div className={`${styles.statusDot} ${styles.statusActive}`}></div>
            <span>All systems operational</span>
          </div>
        </div>

        <div className={styles.statusActions}>
          <button className={`${styles.button} ${styles.primary}`}>
            New Settlement
          </button>

          <button className={`${styles.button} ${styles.secondary}`}>
            View Reports
          </button>
        </div>
      </GlassyCard>
      
      {/* Stats Section */}
      <div className={styles.statsContainer}>
        <GlassyCard darker className={styles.statCard}>
          <h3>Pending</h3>
          <h2>12</h2>
          <span>Transactions</span>
        </GlassyCard>

        <GlassyCard darker className={styles.statCard}>
          <h3>Processing</h3>
          <h2>5</h2>
          <span>Transactions</span>
        </GlassyCard>

        <GlassyCard darker className={styles.statCard}>
          <h3>Completed</h3>
          <h2>1,243</h2>
          <span>Today</span>
        </GlassyCard>

        <GlassyCard darker className={styles.statCard}>
          <h3>Failed</h3>
          <h2>3</h2>
          <span>Today</span>
        </GlassyCard>
      </div>
      
      {/* Queue Section */}
      <GlassyCard darker className={styles.queueContainer}>
        <div className={styles.queueHeader}>
          <h3>Settlement Queue</h3>
          <div className={styles.queueControls}>
            <div className={styles.searchBox}>
              <span className={styles.searchIcon}>🔍</span>
              <input
                type="text"
                placeholder="Search transactions"
                className={styles.searchInput}
              />
            </div>

            <select className={styles.statusFilter}>
              <option value="all">All Status</option>
              <option value="pending">Pending</option>
              <option value="processing">Processing</option>
              <option value="completed">Completed</option>
              <option value="failed">Failed</option>
            </select>
          </div>
        </div>
        
        <div className={styles.tableWrapper}>
          <table className={styles.settlementTable}>
            <thead>
              <tr>
                <th>ID</th>
                <th>Type</th>
                <th>From</th>
                <th>To</th>
                <th className={styles.textRight}>Amount</th>
                <th className={styles.textCenter}>Status</th>
                <th className={styles.textRight}>Time</th>
                <th className={styles.textCenter}>Actions</th>
              </tr>
            </thead>
            <tbody>
              <tr>
                <td>
                  <a href="#" className={styles.transactionLink}>TX-9385</a>
                </td>
                <td>Transfer</td>
                <td>
                  <div className={styles.cryptoCell}>
                    <div className={`${styles.cryptoIcon} ${styles.bitcoinIcon}`}>₿</div>
                    <span>Bitcoin</span>
                  </div>
                </td>
                <td>
                  <div className={styles.cryptoCell}>
                    <div className={`${styles.cryptoIcon} ${styles.ethereumIcon}`}>Ξ</div>
                    <span>Ethereum</span>
                  </div>
                </td>
                <td className={styles.textRight}>0.5 BTC</td>
                <td className={styles.textCenter}>
                  <span className={`${styles.statusBadge} ${styles.statusProcessing}`}>Processing</span>
                </td>
                <td className={styles.textRight}>2 min ago</td>
                <td className={styles.textCenter}>
                  <button className={`${styles.button} ${styles.secondary}`}>Details</button>
                </td>
              </tr>
              
              <tr>
                <td>
                  <a href="#" className={styles.transactionLink}>TX-9384</a>
                </td>
                <td>Swap</td>
                <td>
                  <div className={styles.cryptoCell}>
                    <div className={`${styles.cryptoIcon} ${styles.ethereumIcon}`}>Ξ</div>
                    <span>Ethereum</span>
                  </div>
                </td>
                <td>
                  <div className={styles.cryptoCell}>
                    <div className={`${styles.cryptoIcon} ${styles.usdcIcon}`}>U</div>
                    <span>USDC</span>
                  </div>
                </td>
                <td className={styles.textRight}>2.5 ETH</td>
                <td className={styles.textCenter}>
                  <span className={`${styles.statusBadge} ${styles.statusCompleted}`}>Completed</span>
                </td>
                <td className={styles.textRight}>15 min ago</td>
                <td className={styles.textCenter}>
                  <button className={`${styles.button} ${styles.secondary}`}>Details</button>
                </td>
              </tr>
              
              <tr>
                <td>
                  <a href="#" className={styles.transactionLink}>TX-9383</a>
                </td>
                <td>Deposit</td>
                <td>
                  <div className={styles.cryptoCell}>
                    <div className={`${styles.cryptoIcon} ${styles.usdcIcon}`}>U</div>
                    <span>USDC</span>
                  </div>
                </td>
                <td>
                  <div className={styles.cryptoCell}>
                    <div className={`${styles.cryptoIcon} ${styles.ethereumIcon}`}>Ξ</div>
                    <span>Ethereum</span>
                  </div>
                </td>
                <td className={styles.textRight}>5,000 USDC</td>
                <td className={styles.textCenter}>
                  <span className={`${styles.statusBadge} ${styles.statusFailed}`}>Failed</span>
                </td>
                <td className={styles.textRight}>45 min ago</td>
                <td className={styles.textCenter}>
                  <button className={`${styles.button} ${styles.secondary}`}>Details</button>
                </td>
              </tr>
              
              <tr>
                <td>
                  <a href="#" className={styles.transactionLink}>TX-9382</a>
                </td>
                <td>Withdrawal</td>
                <td>
                  <div className={styles.cryptoCell}>
                    <div className={`${styles.cryptoIcon} ${styles.bitcoinIcon}`}>₿</div>
                    <span>Bitcoin</span>
                  </div>
                </td>
                <td>
                  <div className={styles.cryptoCell}>
                    <div className={`${styles.cryptoIcon} ${styles.bitcoinIcon}`}>₿</div>
                    <span>External Wallet</span>
                  </div>
                </td>
                <td className={styles.textRight}>0.25 BTC</td>
                <td className={styles.textCenter}>
                  <span className={`${styles.statusBadge} ${styles.statusPending}`}>Pending</span>
                </td>
                <td className={styles.textRight}>1 hour ago</td>
                <td className={styles.textCenter}>
                  <button className={`${styles.button} ${styles.secondary}`}>Details</button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </GlassyCard>
      
      {/* Analytics Section */}
      <div className={styles.statsContainer}>
        <GlassyCard darker className={styles.historyContainer}>
          <div className={styles.historyHeader}>
            <h3>Settlement Volume</h3>
            <div className={styles.historyControls}>
              <select className={styles.dateFilter}>
                <option value="day">Today</option>
                <option value="week">This Week</option>
                <option value="month">This Month</option>
                <option value="year">This Year</option>
              </select>
            </div>
          </div>

          <div className={styles.chartContainer}>
            <span className={styles.chartPlaceholder}>Settlement Volume Chart</span>
          </div>

          <GlassyCard>
            <div className={styles.infoRow}>
              <span>Today&apos;s Volume:</span>
              <span>$12.5M</span>
            </div>
            <div className={styles.infoRow}>
              <span>Weekly Volume:</span>
              <span>$87.3M</span>
            </div>
            <div className={styles.infoRow}>
              <span>Monthly Volume:</span>
              <span>$342.1M</span>
            </div>
          </GlassyCard>
        </GlassyCard>
        
        <GlassyCard darker className={styles.historyContainer}>
          <h3>Settlement Performance</h3>
          
          <div className={styles.chartContainer}>
            <span className={styles.chartPlaceholder}>Settlement Performance Chart</span>
          </div>
          
          <GlassyCard>
            <div className={styles.infoRow}>
              <span>Average Settlement Time:</span>
              <span>2.3 minutes</span>
            </div>
            <div className={styles.infoRow}>
              <span>Success Rate:</span>
              <span>99.7%</span>
            </div>
            <div className={styles.infoRow}>
              <span>Fee Efficiency:</span>
              <span>98.2%</span>
            </div>
          </GlassyCard>
        </GlassyCard>
      </div>
      
      {/* Configuration Section */}
      <GlassyCard darker className={styles.configContainer}>
        <h3>Settlement Configuration</h3>
        
        <div className={styles.configGrid}>
          <GlassyCard>
            <h4>Gas Optimization</h4>
            
            <div className={styles.infoRow}>
              <span>Current Strategy:</span>
              <span>Balanced</span>
            </div>

            <div className={styles.buttonGroup}>
              <button className={`${styles.button} ${styles.secondary}`}>
                Economy
              </button>
              
              <button className={`${styles.button} ${styles.primary}`}>
                Balanced
              </button>

              <button className={`${styles.button} ${styles.secondary}`}>
                Fast
              </button>
            </div>
          </GlassyCard>
          
          <GlassyCard>
            <h4>Batch Processing</h4>
            
            <div className={styles.infoRow}>
              <span>Status:</span>
              <span className={styles.positiveValue}>Enabled</span>
            </div>
            
            <div className={styles.infoRow}>
              <span>Batch Size:</span>
              <span>10 transactions</span>
            </div>
            
            <div className={styles.infoRow}>
              <span>Frequency:</span>
              <span>5 minutes</span>
            </div>

            <button className={`${styles.button} ${styles.primary}`}>
              Configure
            </button>
          </GlassyCard>
          
          <GlassyCard>
            <h4>Security Settings</h4>
            
            <div className={styles.infoRow}>
              <span>Multi-sig:</span>
              <span className={styles.positiveValue}>Enabled (2/3)</span>
            </div>
            
            <div className={styles.infoRow}>
              <span>Threshold:</span>
              <span>$10,000</span>
            </div>

            <div className={styles.infoRow}>
              <span>Whitelist:</span>
              <span>12 addresses</span>
            </div>
            
            <button className={`${styles.button} ${styles.primary}`}>
              Manage
            </button>
          </GlassyCard>
        </div>
      </GlassyCard>
      </PageLayout>
    </RoleProtectedRoute>
  );
}