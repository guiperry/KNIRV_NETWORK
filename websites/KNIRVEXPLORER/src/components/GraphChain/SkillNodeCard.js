import React from 'react';
import Link from 'next/link';
import styles from './SkillNodeCard.module.css';

/**
 * SkillNodeCard Component
 * Displays a SkillNode with its details, capabilities, and performance metrics
 *
 * @param {Object} props
 * @param {import('../../services/graphchain-api').SkillNode} props.skill - The skill node data
 */
export default function SkillNodeCard({ skill }) {
  const formatTime = (timestamp) => {
    return new Date(timestamp).toLocaleString();
  };

  const getValidationColor = (validation) => {
    if (!validation) return styles.validationGray;
    if (validation.is_validated && validation.validation_score > 0.8) return styles.validationGreen;
    if (validation.is_validated && validation.validation_score > 0.6) return styles.validationYellow;
    return styles.validationRed;
  };

  const getValidationIcon = (validation) => {
    if (!validation) return 'fa-exclamation-circle';
    if (validation.is_validated && validation.validation_score > 0.8) return 'fa-check-circle';
    if (validation.is_validated && validation.validation_score > 0.6) return 'fa-exclamation-circle';
    return 'fa-exclamation-circle';
  };

  return (
    <Link href={`/skill/${skill.id}`} className={styles.skillCard}>
      <div className={styles.header}>
        <div className={styles.headerLeft}>
          <div className={styles.iconWrapper}>
            <i className="fas fa-brain"></i>
          </div>
          <div className={styles.headerInfo}>
            <div className={styles.skillType}>{skill.skill_type}</div>
            <div className={styles.timestamp}>
              <i className="fas fa-clock"></i>
              <span>{formatTime(skill.timestamp)}</span>
            </div>
          </div>
        </div>
        <div className={styles.headerRight}>
          <div className={`${styles.validation} ${getValidationColor(skill.validation)}`}>
            <i className={`fas ${getValidationIcon(skill.validation)}`}></i>
            <span>
              {skill.validation?.is_validated
                ? `${(skill.validation.validation_score * 100).toFixed(0)}%`
                : 'Unvalidated'
              }
            </span>
          </div>
        </div>
      </div>

      <div className={styles.body}>
        <div className={styles.section}>
          <div className={styles.sectionTitle}>Capabilities</div>
          <div className={styles.capabilities}>
            {skill.capabilities.slice(0, 3).map((capability, index) => (
              <span key={index} className={styles.capabilityBadge}>
                {capability}
              </span>
            ))}
            {skill.capabilities.length > 3 && (
              <span className={styles.moreCapabilities}>
                +{skill.capabilities.length - 3} more
              </span>
            )}
          </div>
        </div>

        {skill.performance && (
          <div className={styles.section}>
            <div className={styles.sectionTitle}>Performance</div>
            <div className={styles.performance}>
              <div className={styles.performanceItem}>
                <i className="fas fa-chart-line"></i>
                <span className={styles.performanceValue}>
                  {(skill.performance.success_rate * 100).toFixed(1)}% success
                </span>
              </div>
              <div className={styles.performanceResolutions}>
                {skill.performance.total_resolutions} resolutions
              </div>
            </div>
          </div>
        )}
      </div>

      {skill.requirements && Object.keys(skill.requirements).length > 0 && (
        <div className={styles.requirements}>
          <div className={styles.sectionTitle}>Requirements</div>
          <div className={styles.requirementsContent}>
            {Object.entries(skill.requirements).map(([key, value]) => (
              <div key={key} className={styles.requirementRow}>
                <span className={styles.requirementKey}>{key}:</span>
                <span className={styles.requirementValue}>{String(value)}</span>
              </div>
            ))}
          </div>
        </div>
      )}
    </Link>
  );
}
