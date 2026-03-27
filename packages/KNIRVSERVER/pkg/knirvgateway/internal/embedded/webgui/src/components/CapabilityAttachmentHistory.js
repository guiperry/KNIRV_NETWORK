import React from 'react';
import styles from './CapabilityAttachmentHistory.module.css';

const CapabilityAttachmentHistory = ({ history }) => {
  if (!history || history.length === 0) {
    return (
      <div className={styles.emptyHistory}>
        No capability attachment history available
      </div>
    );
  }

  return (
    <div className={styles.historyContainer}>
      <div className={styles.historyList}>
        {history.map((item, index) => (
          <div key={index} className={styles.historyItem}>
            <div className={styles.historyIcon}>
              {item.action === 'attach' ? '➕' : '➖'}
            </div>
            <div className={styles.historyContent}>
              <div className={styles.historyHeader}>
                <span className={styles.capabilityName}>{item.capability_name}</span>
                <span className={styles.timestamp}>
                  {new Date(item.timestamp).toLocaleString()}
                </span>
              </div>
              <div className={styles.actionType}>
                {item.action === 'attach' ? 'Attached' : 'Detached'}
              </div>
              {item.params && Object.keys(item.params).length > 0 && (
                <div className={styles.paramsSection}>
                  <div className={styles.paramsTitle}>Parameters:</div>
                  <div className={styles.paramsList}>
                    {Object.entries(item.params).map(([key, value]) => (
                      <div key={key} className={styles.paramItem}>
                        <span className={styles.paramName}>{key}:</span>
                        <span className={styles.paramValue}>
                          {typeof value === 'object' ? JSON.stringify(value) : value.toString()}
                        </span>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
};

export default CapabilityAttachmentHistory;