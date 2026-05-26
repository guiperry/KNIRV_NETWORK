import React from 'react';
import styles from './PageLayout.module.css';
import SideNavigation from './SideNavigation';
import SearchBar from './SearchBar';
import Footer from './Footer';
import { useRole } from '../contexts/RoleContext';

const PageLayout = ({ 
  children, 
  activePage, 
  pageTitle, 
  onSearch = () => {} 
}) => {
  const { role } = useRole();
  
  return (
    <div className={styles.dashboardContainer}>
      <SideNavigation activePage={activePage} />

      <div className={styles.mainContent}>
        {/* Top Navigation */}
        <div className={`${styles.topNav} ${styles.glassyContainer}`}>
          <h3 className={styles.pageTitle}>{pageTitle}</h3>
          <div className={styles.userControls}>
            <div className={styles.roleDisplay}>
              Role: <span className={styles.roleBadge}>{role}</span>
            </div>
            <SearchBar onSearch={onSearch} />
            <span className={styles.controlIcon}>🔔</span>
            <span className={styles.controlIcon}>⚙️</span>
            <span className={styles.controlIcon}>👤</span>
          </div>
        </div>

        <div className={styles.pageContent}>
          {children}
        </div>

        {/* Footer */}
        <Footer />
      </div>
    </div>
  );
};

export default PageLayout;