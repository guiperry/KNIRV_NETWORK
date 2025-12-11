import React from 'react';
import styles from './LoadingSpinner.module.css';

/**
 * LoadingSpinner Component
 * Displays a loading spinner with optional size
 *
 * @param {Object} props
 * @param {('small'|'medium'|'large')} [props.size='medium'] - Size of the spinner
 * @param {string} [props.text] - Optional loading text to display
 */
export default function LoadingSpinner({ size = 'medium', text }) {
  const sizeClasses = {
    small: styles.small,
    medium: styles.medium,
    large: styles.large,
  };

  return (
    <div className={styles.spinnerContainer}>
      <div className={`${styles.spinner} ${sizeClasses[size]}`}>
        <i className="fas fa-spinner fa-spin"></i>
      </div>
      {text && <div className={styles.loadingText}>{text}</div>}
    </div>
  );
}
