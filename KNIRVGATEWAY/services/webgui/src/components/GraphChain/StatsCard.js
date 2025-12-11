import React from 'react';
import styles from './StatsCard.module.css';

/**
 * StatsCard Component
 * Displays a statistic card with an icon, value, title, and optional trend
 *
 * @param {Object} props
 * @param {string} props.title - The title of the statistic
 * @param {string|number} props.value - The value to display
 * @param {string} props.icon - Font Awesome icon class (e.g., "fa-cube")
 * @param {number} [props.trend] - Optional trend percentage (positive or negative)
 * @param {string} [props.color] - Color theme: 'blue', 'green', 'purple', 'orange', 'red'
 */
export default function StatsCard({ title, value, icon, trend, color = 'blue' }) {
  const colorClasses = {
    blue: styles.blue,
    green: styles.green,
    purple: styles.purple,
    orange: styles.orange,
    red: styles.red,
  };

  const trendColor = trend && trend > 0 ? styles.trendPositive : styles.trendNegative;

  return (
    <div className={`${styles.statsCard} ${colorClasses[color]}`}>
      <div className={styles.header}>
        <i className={`fas ${icon} ${styles.icon}`}></i>
        {trend !== undefined && (
          <div className={`${styles.trend} ${trendColor}`}>
            {trend > 0 ? '+' : ''}{trend.toFixed(1)}%
          </div>
        )}
      </div>
      <div className={styles.content}>
        <div className={styles.value}>{value}</div>
        <div className={styles.title}>{title}</div>
      </div>
    </div>
  );
}
