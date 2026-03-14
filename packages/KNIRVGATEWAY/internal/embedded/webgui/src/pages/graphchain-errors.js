import React, { useEffect, useState } from 'react';
import { useNavigation } from '../hooks/useNavigation';
import PageLayout from '../components/PageLayout';
import PageHeader from '../components/PageHeader';
import GlassyCard from '../components/GlassyCard';
import LoadingSpinner from '../components/GraphChain/LoadingSpinner';
import { graphChainApi } from '../services/graphchain-api';
import styles from './graphchain-errors.module.css';

export default function GraphChainErrors() {
  const { activePage } = useNavigation('graphchain-errors');
  const [errors, setErrors] = useState([]);
  const [skills, setSkills] = useState([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState(null);
  const [selectedError, setSelectedError] = useState(null);
  const [relatedSkills, setRelatedSkills] = useState([]);

  useEffect(() => {
    const fetchErrorNodes = async () => {
      setIsLoading(true);
      setError(null);

      try {
        const [fetchedErrors, fetchedSkills] = await Promise.all([
          graphChainApi.getAllErrors(),
          graphChainApi.getAllSkills(),
        ]);

        // Sort errors by timestamp (most recent first)
        fetchedErrors.sort((a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime());

        setErrors(fetchedErrors);
        setSkills(fetchedSkills);
      } catch (err) {
        setError(err.message || 'Failed to fetch ErrorNodes');
      } finally {
        setIsLoading(false);
      }
    };

    fetchErrorNodes();
  }, []);

  // Fetch related skills when an error is selected
  useEffect(() => {
    const fetchRelatedSkills = async () => {
      if (selectedError) {
        try {
          const related = await graphChainApi.getSkillsForError(selectedError.error_type);
          setRelatedSkills(related);
        } catch (err) {
          console.error('Failed to fetch related skills:', err);
          setRelatedSkills([]);
        }
      } else {
        setRelatedSkills([]);
      }
    };

    fetchRelatedSkills();
  }, [selectedError]);

  const formatTime = (timestamp) => {
    return new Date(timestamp).toLocaleString();
  };

  const getSeverityColor = (severity) => {
    if (severity >= 4) return styles.severityCritical;
    if (severity >= 3) return styles.severityHigh;
    if (severity >= 2) return styles.severityMedium;
    return styles.severityLow;
  };

  const getSeverityLabel = (severity) => {
    if (severity >= 4) return 'CRITICAL';
    if (severity >= 3) return 'HIGH';
    if (severity >= 2) return 'MEDIUM';
    return 'LOW';
  };

  const getStatusIcon = (status) => {
    switch (status) {
      case 'resolved':
        return 'fa-check-circle';
      case 'failed':
        return 'fa-times-circle';
      default:
        return 'fa-clock';
    }
  };

  const getStatusColor = (status) => {
    switch (status) {
      case 'resolved':
        return styles.statusResolved;
      case 'failed':
        return styles.statusFailed;
      default:
        return styles.statusPending;
    }
  };

  const handleErrorClick = (errorNode) => {
    setSelectedError(selectedError?.id === errorNode.id ? null : errorNode);
  };

  return (
    <PageLayout activePage={activePage} pageTitle="ErrorNodes">
      <PageHeader
        title="ErrorNodes"
        subtitle="Explore ErrorNodes and their resolution paths"
        titleColor="#fb923c"
      />

      <div className={styles.errorsContainer}>
        {/* Stats */}
        <GlassyCard className={styles.statsCard}>
          <div className={styles.statsGrid}>
            <div className={styles.statItem}>
              <div className={`${styles.statValue} ${styles.statOrange}`}>{errors.length}</div>
              <div className={styles.statLabel}>Total ErrorNodes</div>
            </div>
            <div className={styles.statItem}>
              <div className={`${styles.statValue} ${styles.statRed}`}>
                {errors.filter(e => e.severity >= 3).length}
              </div>
              <div className={styles.statLabel}>High Severity</div>
            </div>
            <div className={styles.statItem}>
              <div className={`${styles.statValue} ${styles.statGreen}`}>
                {errors.filter(e => e.resolution_status === 'resolved').length}
              </div>
              <div className={styles.statLabel}>Resolved</div>
            </div>
            <div className={styles.statItem}>
              <div className={`${styles.statValue} ${styles.statBlue}`}>{skills.length}</div>
              <div className={styles.statLabel}>Available Skills</div>
            </div>
          </div>
        </GlassyCard>

        {/* ErrorNodes List */}
        {isLoading ? (
          <div className={styles.loadingContainer}>
            <LoadingSpinner size="large" text="Loading ErrorNodes..." />
          </div>
        ) : error ? (
          <GlassyCard className={styles.errorCard}>
            <div className={styles.errorIcon}>
              <i className="fas fa-exclamation-triangle"></i>
            </div>
            <div className={styles.errorTitle}>Error Loading ErrorNodes</div>
            <div className={styles.errorMessage}>{error}</div>
          </GlassyCard>
        ) : errors.length === 0 ? (
          <GlassyCard className={styles.emptyCard}>
            <i className="fas fa-exclamation-triangle"></i>
            <h3>No ErrorNodes Found</h3>
            <p>There are no ErrorNodes in the system.</p>
          </GlassyCard>
        ) : (
          <div className={styles.errorsList}>
            {errors.map((errorNode) => (
              <GlassyCard
                key={errorNode.id}
                className={`${styles.errorCard} ${selectedError?.id === errorNode.id ? styles.errorCardSelected : ''}`}
                onClick={() => handleErrorClick(errorNode)}
              >
                {/* Header */}
                <div className={styles.errorHeader}>
                  <div className={styles.errorHeaderLeft}>
                    <div className={styles.errorIcon}>
                      <i className="fas fa-exclamation-triangle"></i>
                    </div>
                    <div>
                      <div className={styles.errorType}>{errorNode.error_type}</div>
                      <div className={styles.errorTimestamp}>
                        <i className="fas fa-clock"></i>
                        <span>{formatTime(errorNode.timestamp)}</span>
                      </div>
                    </div>
                  </div>
                  <div className={styles.errorHeaderRight}>
                    <div className={`${styles.severityBadge} ${getSeverityColor(errorNode.severity)}`}>
                      {getSeverityLabel(errorNode.severity)}
                    </div>
                    {errorNode.resolution_status && (
                      <div className={`${styles.statusBadge} ${getStatusColor(errorNode.resolution_status)}`}>
                        <i className={`fas ${getStatusIcon(errorNode.resolution_status)}`}></i>
                        <span>{errorNode.resolution_status}</span>
                      </div>
                    )}
                  </div>
                </div>

                {/* Description */}
                <div className={styles.errorDescription}>
                  <div className={styles.sectionTitle}>Description</div>
                  <div className={styles.descriptionContent}>{errorNode.description}</div>
                </div>

                {/* Details */}
                <div className={styles.errorDetails}>
                  <div className={styles.detailItem}>
                    <div className={styles.detailLabel}>
                      <i className="fas fa-hashtag"></i>
                      <span>Error ID</span>
                    </div>
                    <div className={styles.detailValue}>{errorNode.id}</div>
                  </div>
                  <div className={styles.detailItem}>
                    <div className={styles.detailLabel}>
                      <i className="fas fa-exclamation-triangle"></i>
                      <span>Error Type</span>
                    </div>
                    <div className={styles.detailValue}>{errorNode.error_type}</div>
                  </div>
                  <div className={styles.detailItem}>
                    <div className={styles.detailLabel}>
                      <span>Severity Level</span>
                    </div>
                    <div className={styles.detailValue}>{errorNode.severity}/5</div>
                  </div>
                  {errorNode.resolved_by && errorNode.resolved_by.length > 0 && (
                    <div className={styles.detailItem}>
                      <div className={styles.detailLabel}>
                        <i className="fas fa-brain"></i>
                        <span>Resolved By</span>
                      </div>
                      <div className={styles.detailValue}>{errorNode.resolved_by.join(', ')}</div>
                    </div>
                  )}
                </div>

                {/* Context */}
                {errorNode.context && Object.keys(errorNode.context).length > 0 && (
                  <div className={styles.errorContext}>
                    <div className={styles.sectionTitle}>Error Context</div>
                    <pre className={styles.contextContent}>
                      {JSON.stringify(errorNode.context, null, 2)}
                    </pre>
                  </div>
                )}

                {/* Related Skills */}
                {selectedError?.id === errorNode.id && relatedSkills.length > 0 && (
                  <div className={styles.relatedSkills}>
                    <div className={styles.sectionTitle}>
                      Related SkillNodes ({relatedSkills.length})
                    </div>
                    <div className={styles.skillsGrid}>
                      {relatedSkills.slice(0, 4).map((skill) => (
                        <div key={skill.id} className={styles.skillCard}>
                          <div className={styles.skillHeader}>
                            <i className="fas fa-brain"></i>
                            <span className={styles.skillType}>{skill.skill_type}</span>
                          </div>
                          <div className={styles.skillCapabilities}>
                            {skill.capabilities.slice(0, 2).join(', ')}
                            {skill.capabilities.length > 2 && ` +${skill.capabilities.length - 2} more`}
                          </div>
                          {skill.performance && (
                            <div className={styles.skillPerformance}>
                              {(skill.performance.success_rate * 100).toFixed(0)}% success rate
                            </div>
                          )}
                        </div>
                      ))}
                    </div>
                    {relatedSkills.length > 4 && (
                      <div className={styles.moreSkills}>
                        +{relatedSkills.length - 4} more skills available
                      </div>
                    )}
                  </div>
                )}

                {/* Footer */}
                <div className={styles.errorFooter}>
                  <div className={styles.footerText}>
                    Click to {selectedError?.id === errorNode.id ? 'hide' : 'view'} related skills
                  </div>
                  <div className={styles.footerLink}>
                    <span>View Details</span>
                    <i className="fas fa-arrow-right"></i>
                  </div>
                </div>
              </GlassyCard>
            ))}
          </div>
        )}
      </div>
    </PageLayout>
  );
}
