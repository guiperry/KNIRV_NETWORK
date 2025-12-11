import React, { useState, useEffect } from 'react';
import { useNavigation } from '../hooks/useNavigation';
import PageLayout from '../components/PageLayout';
import PageHeader from '../components/PageHeader';
import GlassyCard from '../components/GlassyCard';
import DataTable from '../components/DataTable';
import styles from './error-explorer.module.css';

export default function ErrorExplorer() {
  const { activePage } = useNavigation('error-explorer');
  const [searchQuery, setSearchQuery] = useState('');
  const [errors, setErrors] = useState([]);
  const [selectedError, setSelectedError] = useState(null);
  const [filterSeverity, setFilterSeverity] = useState('all');
  const [filterStatus, setFilterStatus] = useState('all');

  const handleSearch = (query) => {
    setSearchQuery(query);
  };

  // Mock error data
  useEffect(() => {
    const mockErrors = [
      {
        id: '1',
        nodeId: 'node-abc123',
        errorCode: 'ERR_CONNECTION_TIMEOUT',
        severity: 'High',
        status: 'Active',
        message: 'Connection timeout to skill execution service',
        timestamp: '2024-01-22 14:30:25',
        affectedServices: ['SkillExecution', 'ModelInference'],
        stackTrace: 'Error: Connection timeout\n  at SkillExecutor.connect()\n  at ModelService.invoke()',
        resolution: null,
        occurrences: 15
      },
      {
        id: '2',
        nodeId: 'node-def456',
        errorCode: 'ERR_MEMORY_LIMIT',
        severity: 'Critical',
        status: 'Investigating',
        message: 'Memory usage exceeded 95% threshold',
        timestamp: '2024-01-22 14:25:10',
        affectedServices: ['ModelTraining', 'DataProcessing'],
        stackTrace: 'OutOfMemoryError: Java heap space\n  at ModelTrainer.loadDataset()\n  at TrainingService.start()',
        resolution: 'Scaling up node resources',
        occurrences: 3
      },
      {
        id: '3',
        nodeId: 'node-ghi789',
        errorCode: 'ERR_SKILL_NOT_FOUND',
        severity: 'Medium',
        status: 'Resolved',
        message: 'Requested skill ID not found in registry',
        timestamp: '2024-01-22 13:45:18',
        affectedServices: ['SkillRegistry'],
        stackTrace: 'SkillNotFoundError: Skill ID skill-xyz not found\n  at SkillRegistry.getSkill()',
        resolution: 'Skill re-registered successfully',
        occurrences: 8
      },
      {
        id: '4',
        nodeId: 'node-jkl012',
        errorCode: 'ERR_AUTH_FAILED',
        severity: 'Low',
        status: 'Active',
        message: 'Authentication failed for API request',
        timestamp: '2024-01-22 13:20:45',
        affectedServices: ['APIGateway'],
        stackTrace: 'AuthenticationError: Invalid token\n  at AuthService.validateToken()',
        resolution: null,
        occurrences: 42
      }
    ];
    setErrors(mockErrors);
  }, []);

  // Filter errors based on search query, severity, and status
  const filteredErrors = errors.filter((error) => {
    const matchesSearch = error.message.toLowerCase().includes(searchQuery.toLowerCase()) ||
                         error.errorCode.toLowerCase().includes(searchQuery.toLowerCase()) ||
                         error.nodeId.toLowerCase().includes(searchQuery.toLowerCase());
    const matchesSeverity = filterSeverity === 'all' || error.severity.toLowerCase() === filterSeverity.toLowerCase();
    const matchesStatus = filterStatus === 'all' || error.status.toLowerCase() === filterStatus.toLowerCase();
    return matchesSearch && matchesSeverity && matchesStatus;
  });

  const severityLevels = ['all', 'Critical', 'High', 'Medium', 'Low'];
  const statusOptions = ['all', 'Active', 'Investigating', 'Resolved'];

  // Table headers
  const tableHeaders = [
    { label: 'Error Code' },
    { label: 'Node ID' },
    { label: 'Severity', align: 'center' },
    { label: 'Status', align: 'center' },
    { label: 'Occurrences', align: 'right' },
    { label: 'Timestamp' },
    { label: 'Actions', align: 'center' },
  ];

  // Render table row
  const renderTableRow = (error, index) => (
    <tr className={styles.tableRow} key={error.id}>
      <td className={styles.tableCell}>
        <div className={styles.errorInfo}>
          <div className={styles.errorCode}>{error.errorCode}</div>
          <div className={styles.errorMessage}>{error.message}</div>
        </div>
      </td>
      <td className={styles.tableCell}>
        <code className={styles.nodeId}>{error.nodeId}</code>
      </td>
      <td className={`${styles.tableCell} ${styles.textCenter}`}>
        <span className={`${styles.severity} ${styles[error.severity.toLowerCase()]}`}>
          {error.severity}
        </span>
      </td>
      <td className={`${styles.tableCell} ${styles.textCenter}`}>
        <span className={`${styles.status} ${styles[error.status.toLowerCase()]}`}>
          {error.status}
        </span>
      </td>
      <td className={`${styles.tableCell} ${styles.textRight}`}>
        <span className={styles.occurrences}>{error.occurrences}</span>
      </td>
      <td className={styles.tableCell}>
        <span className={styles.timestamp}>{error.timestamp}</span>
      </td>
      <td className={`${styles.tableCell} ${styles.textCenter}`}>
        <div className={styles.actionButtons}>
          <button 
            onClick={() => setSelectedError(error)}
            className={styles.viewButton}
            title="View Details"
          >
            <i className="fas fa-eye"></i>
          </button>
          <button 
            className={styles.resolveButton}
            title="Mark as Resolved"
          >
            <i className="fas fa-check"></i>
          </button>
        </div>
      </td>
    </tr>
  );

  const getErrorStats = () => {
    const total = errors.length;
    const active = errors.filter(e => e.status === 'Active').length;
    const critical = errors.filter(e => e.severity === 'Critical').length;
    const totalOccurrences = errors.reduce((sum, e) => sum + e.occurrences, 0);
    
    return { total, active, critical, totalOccurrences };
  };

  const stats = getErrorStats();

  return (
    <PageLayout activePage={activePage} pageTitle="Error Explorer" onSearch={handleSearch}>
      <PageHeader 
        title="ErrorNode Explorer"
        subtitle="Monitor and analyze network errors and issues"
        titleColor="#dc3545"
      />

      {/* Filter Controls */}
      <GlassyCard className={styles.filterCard}>
        <div className={styles.filterHeader}>
          <h3 className={styles.filterTitle}>
            <i className="fas fa-filter" style={{ marginRight: '10px', color: '#dc3545' }}></i>
            Filter Errors
          </h3>
        </div>
        <div className={styles.filterControls}>
          <div className={styles.filterGroup}>
            <label>Severity:</label>
            <select 
              value={filterSeverity} 
              onChange={(e) => setFilterSeverity(e.target.value)}
              className={styles.filterSelect}
            >
              {severityLevels.map(level => (
                <option key={level} value={level}>
                  {level === 'all' ? 'All Severities' : level}
                </option>
              ))}
            </select>
          </div>
          <div className={styles.filterGroup}>
            <label>Status:</label>
            <select 
              value={filterStatus} 
              onChange={(e) => setFilterStatus(e.target.value)}
              className={styles.filterSelect}
            >
              {statusOptions.map(status => (
                <option key={status} value={status}>
                  {status === 'all' ? 'All Statuses' : status}
                </option>
              ))}
            </select>
          </div>
        </div>
      </GlassyCard>

      {/* Statistics Overview */}
      <div className={styles.statsContainer}>
        <GlassyCard className={styles.statCard}>
          <div className={styles.statIcon}>🚨</div>
          <div className={styles.statValue}>{stats.total}</div>
          <div className={styles.statLabel}>Total Errors</div>
        </GlassyCard>
        <GlassyCard className={styles.statCard}>
          <div className={styles.statIcon}>🔴</div>
          <div className={styles.statValue}>{stats.active}</div>
          <div className={styles.statLabel}>Active Errors</div>
        </GlassyCard>
        <GlassyCard className={styles.statCard}>
          <div className={styles.statIcon}>⚠️</div>
          <div className={styles.statValue}>{stats.critical}</div>
          <div className={styles.statLabel}>Critical Errors</div>
        </GlassyCard>
        <GlassyCard className={styles.statCard}>
          <div className={styles.statIcon}>📊</div>
          <div className={styles.statValue}>{stats.totalOccurrences}</div>
          <div className={styles.statLabel}>Total Occurrences</div>
        </GlassyCard>
      </div>

      {/* Errors Table */}
      <GlassyCard className={styles.tableCard}>
        <div className={styles.tableHeader}>
          <h3 className={styles.tableTitle}>
            <i className="fas fa-exclamation-triangle" style={{ marginRight: '10px', color: '#dc3545' }}></i>
            Network Errors ({filteredErrors.length})
          </h3>
        </div>
        
        {filteredErrors.length > 0 ? (
          <DataTable
            headers={tableHeaders}
            data={filteredErrors}
            renderRow={renderTableRow}
            className={styles.errorsTable}
          />
        ) : (
          <div className={styles.emptyState}>
            <i className="fas fa-check-circle" style={{ fontSize: '3rem', color: '#28a745', marginBottom: '20px' }}></i>
            <h3>No errors found</h3>
            <p>All systems are running smoothly or no errors match your current filters.</p>
          </div>
        )}
      </GlassyCard>

      {/* Error Details Modal */}
      {selectedError && (
        <div className={styles.modal}>
          <div className={styles.modalContent}>
            <div className={styles.modalHeader}>
              <h3>{selectedError.errorCode}</h3>
              <button onClick={() => setSelectedError(null)} className={styles.closeButton}>×</button>
            </div>
            <div className={styles.modalBody}>
              <div className={styles.errorDetails}>
                <div className={styles.detailSection}>
                  <h4>Error Information</h4>
                  <div className={styles.detailGrid}>
                    <div className={styles.detailItem}>
                      <span className={styles.detailLabel}>Node ID:</span>
                      <code className={styles.detailValue}>{selectedError.nodeId}</code>
                    </div>
                    <div className={styles.detailItem}>
                      <span className={styles.detailLabel}>Severity:</span>
                      <span className={`${styles.detailValue} ${styles[selectedError.severity.toLowerCase()]}`}>
                        {selectedError.severity}
                      </span>
                    </div>
                    <div className={styles.detailItem}>
                      <span className={styles.detailLabel}>Status:</span>
                      <span className={`${styles.detailValue} ${styles[selectedError.status.toLowerCase()]}`}>
                        {selectedError.status}
                      </span>
                    </div>
                    <div className={styles.detailItem}>
                      <span className={styles.detailLabel}>Occurrences:</span>
                      <span className={styles.detailValue}>{selectedError.occurrences}</span>
                    </div>
                  </div>
                </div>

                <div className={styles.detailSection}>
                  <h4>Error Message</h4>
                  <div className={styles.messageBox}>
                    {selectedError.message}
                  </div>
                </div>

                <div className={styles.detailSection}>
                  <h4>Affected Services</h4>
                  <div className={styles.servicesList}>
                    {selectedError.affectedServices.map((service, index) => (
                      <span key={index} className={styles.serviceTag}>{service}</span>
                    ))}
                  </div>
                </div>

                <div className={styles.detailSection}>
                  <h4>Stack Trace</h4>
                  <pre className={styles.stackTrace}>{selectedError.stackTrace}</pre>
                </div>

                {selectedError.resolution && (
                  <div className={styles.detailSection}>
                    <h4>Resolution</h4>
                    <div className={styles.resolutionBox}>
                      {selectedError.resolution}
                    </div>
                  </div>
                )}
              </div>
            </div>
          </div>
        </div>
      )}
    </PageLayout>
  );
}
