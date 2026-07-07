import React, { useState, useEffect } from 'react';
import { useNavigation } from '../hooks/useNavigation';
import PageLayout from '../components/PageLayout';
import PageHeader from '../components/PageHeader';
import GlassyCard from '../components/GlassyCard';
import DataTable from '../components/DataTable';
import styles from './my-models.module.css';

export default function MyModels() {
  const { activePage } = useNavigation('my-models');
  const [searchQuery, setSearchQuery] = useState('');
  const [showRegistrationModal, setShowRegistrationModal] = useState(false);
  const [models, setModels] = useState([]);
  const [selectedModel, setSelectedModel] = useState(null);

  const handleSearch = (query) => {
    setSearchQuery(query);
  };

  // Mock model data
  useEffect(() => {
    const mockModels = [
      {
        id: '1',
        name: 'GPT-4 Custom Finance',
        type: 'Language Model',
        status: 'Active',
        performance: '94.5%',
        deployedAt: '2024-01-15',
        version: '1.2.0',
        description: 'Fine-tuned GPT-4 model specialized for financial analysis and reporting',
        endpoints: ['https://api.knirv.network/models/gpt4-finance'],
        metrics: {
          totalCalls: 15420,
          successRate: 98.7,
          avgResponseTime: '245ms',
          lastUsed: '2 hours ago'
        }
      },
      {
        id: '2',
        name: 'Vision Classifier Pro',
        type: 'Computer Vision',
        status: 'Training',
        performance: '89.2%',
        deployedAt: '2024-01-20',
        version: '2.1.0',
        description: 'Advanced image classification model for medical imaging applications',
        endpoints: ['https://api.knirv.network/models/vision-classifier'],
        metrics: {
          totalCalls: 8750,
          successRate: 96.3,
          avgResponseTime: '180ms',
          lastUsed: '1 day ago'
        }
      },
      {
        id: '3',
        name: 'Sentiment Analyzer',
        type: 'NLP',
        status: 'Inactive',
        performance: '91.8%',
        deployedAt: '2024-01-10',
        version: '1.0.5',
        description: 'Multi-language sentiment analysis model for social media monitoring',
        endpoints: ['https://api.knirv.network/models/sentiment'],
        metrics: {
          totalCalls: 3240,
          successRate: 94.1,
          avgResponseTime: '120ms',
          lastUsed: '1 week ago'
        }
      }
    ];
    setModels(mockModels);
  }, []);

  // Filter models based on search query
  const filteredModels = models.filter((model) =>
    model.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
    model.type.toLowerCase().includes(searchQuery.toLowerCase()) ||
    model.description.toLowerCase().includes(searchQuery.toLowerCase())
  );

  // Table headers
  const tableHeaders = [
    { label: 'Model Name' },
    { label: 'Type' },
    { label: 'Status', align: 'center' },
    { label: 'Performance', align: 'right' },
    { label: 'Version' },
    { label: 'Last Used' },
    { label: 'Actions', align: 'center' },
  ];

  // Render table row
  const renderTableRow = (model, index) => (
    <tr className={styles.tableRow} key={model.id}>
      <td className={styles.tableCell}>
        <div className={styles.modelInfo}>
          <div className={styles.modelName}>{model.name}</div>
          <div className={styles.modelDescription}>{model.description}</div>
        </div>
      </td>
      <td className={styles.tableCell}>
        <span className={styles.modelType}>{model.type}</span>
      </td>
      <td className={`${styles.tableCell} ${styles.textCenter}`}>
        <span className={`${styles.status} ${styles[model.status.toLowerCase()]}`}>
          {model.status}
        </span>
      </td>
      <td className={`${styles.tableCell} ${styles.textRight}`}>
        <span className={styles.performance}>{model.performance}</span>
      </td>
      <td className={styles.tableCell}>
        <span className={styles.version}>v{model.version}</span>
      </td>
      <td className={styles.tableCell}>
        <span className={styles.lastUsed}>{model.metrics.lastUsed}</span>
      </td>
      <td className={`${styles.tableCell} ${styles.textCenter}`}>
        <div className={styles.actionButtons}>
          <button 
            onClick={() => setSelectedModel(model)}
            className={styles.viewButton}
            title="View Details"
          >
            <i className="fas fa-eye"></i>
          </button>
          <button 
            className={styles.editButton}
            title="Edit Model"
          >
            <i className="fas fa-edit"></i>
          </button>
          <button 
            className={styles.deployButton}
            title="Deploy/Manage"
          >
            <i className="fas fa-rocket"></i>
          </button>
        </div>
      </td>
    </tr>
  );

  const handleRegisterModel = () => {
    setShowRegistrationModal(true);
  };

  const getStatusColor = (status) => {
    switch (status.toLowerCase()) {
      case 'active': return '#28a745';
      case 'training': return '#ffc107';
      case 'inactive': return '#6c757d';
      default: return '#007bff';
    }
  };

  return (
    <PageLayout activePage={activePage} pageTitle="My Models" onSearch={handleSearch}>
      <PageHeader 
        title="My Models"
        subtitle="Manage and monitor your deployed AI models"
        titleColor="#007bff"
      />

      {/* Statistics Overview */}
      <div className={styles.statsContainer}>
        <GlassyCard className={styles.statCard}>
          <div className={styles.statIcon}>🤖</div>
          <div className={styles.statValue}>{models.length}</div>
          <div className={styles.statLabel}>Total Models</div>
        </GlassyCard>
        <GlassyCard className={styles.statCard}>
          <div className={styles.statIcon}>🟢</div>
          <div className={styles.statValue}>{models.filter(m => m.status === 'Active').length}</div>
          <div className={styles.statLabel}>Active Models</div>
        </GlassyCard>
        <GlassyCard className={styles.statCard}>
          <div className={styles.statIcon}>📊</div>
          <div className={styles.statValue}>
            {models.length > 0 ? 
              Math.round(models.reduce((acc, m) => acc + parseFloat(m.performance), 0) / models.length) + '%' 
              : '0%'
            }
          </div>
          <div className={styles.statLabel}>Avg Performance</div>
        </GlassyCard>
        <GlassyCard className={styles.statCard}>
          <div className={styles.statIcon}>📈</div>
          <div className={styles.statValue}>
            {models.reduce((acc, m) => acc + m.metrics.totalCalls, 0).toLocaleString()}
          </div>
          <div className={styles.statLabel}>Total API Calls</div>
        </GlassyCard>
      </div>

      {/* Models Table */}
      <GlassyCard className={styles.tableCard}>
        <div className={styles.tableHeader}>
          <h3 className={styles.tableTitle}>
            <i className="fas fa-robot" style={{ marginRight: '10px', color: '#007bff' }}></i>
            Registered Models ({filteredModels.length})
          </h3>
          <button 
            onClick={handleRegisterModel}
            className={styles.registerButton}
          >
            <i className="fas fa-plus" style={{ marginRight: '8px' }}></i>
            Register New Model
          </button>
        </div>
        
        {filteredModels.length > 0 ? (
          <DataTable
            headers={tableHeaders}
            data={filteredModels}
            renderRow={renderTableRow}
            className={styles.modelsTable}
          />
        ) : (
          <div className={styles.emptyState}>
            <i className="fas fa-robot" style={{ fontSize: '3rem', color: '#666', marginBottom: '20px' }}></i>
            <h3>No models found</h3>
            <p>Register your first model to get started with the KNIRV network.</p>
            <button onClick={handleRegisterModel} className={styles.registerButton}>
              <i className="fas fa-plus" style={{ marginRight: '8px' }}></i>
              Register Your First Model
            </button>
          </div>
        )}
      </GlassyCard>

      {/* Model Details Modal */}
      {selectedModel && (
        <div className={styles.modal}>
          <div className={styles.modalContent}>
            <div className={styles.modalHeader}>
              <h3>{selectedModel.name}</h3>
              <button onClick={() => setSelectedModel(null)} className={styles.closeButton}>×</button>
            </div>
            <div className={styles.modalBody}>
              <div className={styles.modelDetails}>
                <div className={styles.detailSection}>
                  <h4>Model Information</h4>
                  <div className={styles.detailGrid}>
                    <div className={styles.detailItem}>
                      <span className={styles.detailLabel}>Type:</span>
                      <span className={styles.detailValue}>{selectedModel.type}</span>
                    </div>
                    <div className={styles.detailItem}>
                      <span className={styles.detailLabel}>Version:</span>
                      <span className={styles.detailValue}>v{selectedModel.version}</span>
                    </div>
                    <div className={styles.detailItem}>
                      <span className={styles.detailLabel}>Status:</span>
                      <span className={`${styles.detailValue} ${styles[selectedModel.status.toLowerCase()]}`}>
                        {selectedModel.status}
                      </span>
                    </div>
                    <div className={styles.detailItem}>
                      <span className={styles.detailLabel}>Performance:</span>
                      <span className={styles.detailValue}>{selectedModel.performance}</span>
                    </div>
                  </div>
                </div>

                <div className={styles.detailSection}>
                  <h4>Performance Metrics</h4>
                  <div className={styles.metricsGrid}>
                    <div className={styles.metricCard}>
                      <div className={styles.metricValue}>{selectedModel.metrics.totalCalls.toLocaleString()}</div>
                      <div className={styles.metricLabel}>Total Calls</div>
                    </div>
                    <div className={styles.metricCard}>
                      <div className={styles.metricValue}>{selectedModel.metrics.successRate}%</div>
                      <div className={styles.metricLabel}>Success Rate</div>
                    </div>
                    <div className={styles.metricCard}>
                      <div className={styles.metricValue}>{selectedModel.metrics.avgResponseTime}</div>
                      <div className={styles.metricLabel}>Avg Response</div>
                    </div>
                  </div>
                </div>

                <div className={styles.detailSection}>
                  <h4>API Endpoints</h4>
                  <div className={styles.endpointsList}>
                    {selectedModel.endpoints.map((endpoint, index) => (
                      <div key={index} className={styles.endpointItem}>
                        <code className={styles.endpointUrl}>{endpoint}</code>
                        <button className={styles.copyButton} title="Copy to clipboard">
                          <i className="fas fa-copy"></i>
                        </button>
                      </div>
                    ))}
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Registration Modal */}
      {showRegistrationModal && (
        <div className={styles.modal}>
          <div className={styles.modalContent}>
            <div className={styles.modalHeader}>
              <h3>Register New Model</h3>
              <button onClick={() => setShowRegistrationModal(false)} className={styles.closeButton}>×</button>
            </div>
            <div className={styles.modalBody}>
              <p>Model registration functionality would be implemented here, integrating with the KNIRV network registration system.</p>
              <div className={styles.modalActions}>
                <button onClick={() => setShowRegistrationModal(false)} className={styles.cancelButton}>
                  Cancel
                </button>
                <button className={styles.registerButton}>
                  Register Model
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
    </PageLayout>
  );
}
