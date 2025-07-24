import React from 'react';
import styles from './PageHeader.module.css';

const PageHeader = ({ title, subtitle }) => {
  return (
    <div className={styles.pageHeader}>
      <h2 className={styles.pageHeaderTitle}>{title}</h2>
      {subtitle && <p className={styles.pageHeaderSubtitle}>{subtitle}</p>}
    </div>
  );
};

export default PageHeader;