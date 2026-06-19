import React, { useState, useEffect } from 'react';
import { useNavigation } from '../hooks/useNavigation';
import PageLayout from '../components/PageLayout';
import PageHeader from '../components/PageHeader';
import GlassyCard from '../components/GlassyCard';
import styles from './codex-builder.module.css';

export default function CodexBuilder() {
  const { activePage } = useNavigation('codex-builder');
  const [searchQuery, setSearchQuery] = useState('');
  const [activeTab, setActiveTab] = useState('builder'); // builder, templates, projects
  const [selectedTemplate, setSelectedTemplate] = useState(null);
  const [projects, setProjects] = useState([]);

  const handleSearch = (query) => {
    setSearchQuery(query);
  };

  // Mock data
  useEffect(() => {
    const mockProjects = [
      {
        id: '1',
        name: 'Financial Analysis Model',
        type: 'Language Model',
        status: 'Training',
        progress: 75,
        created: '2024-01-15',
        lastModified: '2024-01-22',
        description: 'Advanced financial analysis and prediction model'
      },
      {
        id: '2',
        name: 'Image Classification Codex',
        type: 'Computer Vision',
        status: 'Complete',
        progress: 100,
        created: '2024-01-10',
        lastModified: '2024-01-20',
        description: 'Medical image classification and diagnosis'
      },
      {
        id: '3',
        name: 'NLP Sentiment Analyzer',
        type: 'Natural Language',
        status: 'Draft',
        progress: 25,
        created: '2024-01-20',
        lastModified: '2024-01-22',
        description: 'Multi-language sentiment analysis system'
      }
    ];
    setProjects(mockProjects);
  }, []);

  const templates = [
    {
      id: 'template-1',
      name: 'Language Model Template',
      category: 'NLP',
      description: 'Pre-configured template for building language models',
      complexity: 'Intermediate',
      estimatedTime: '2-4 hours',
      features: ['Tokenization', 'Training Pipeline', 'Evaluation Metrics']
    },
    {
      id: 'template-2',
      name: 'Computer Vision Template',
      category: 'CV',
      description: 'Template for image classification and object detection',
      complexity: 'Advanced',
      estimatedTime: '4-6 hours',
      features: ['Data Preprocessing', 'CNN Architecture', 'Transfer Learning']
    },
    {
      id: 'template-3',
      name: 'Time Series Forecasting',
      category: 'Analytics',
      description: 'Template for time series analysis and prediction',
      complexity: 'Beginner',
      estimatedTime: '1-2 hours',
      features: ['Data Cleaning', 'LSTM Networks', 'Forecasting']
    }
  ];

  const getStatusColor = (status) => {
    switch (status) {
      case 'Complete': return '#28a745';
      case 'Training': return '#ffc107';
      case 'Draft': return '#6c757d';
      default: return '#007bff';
    }
  };

  const getComplexityColor = (complexity) => {
    switch (complexity) {
      case 'Beginner': return '#28a745';
      case 'Intermediate': return '#ffc107';
      case 'Advanced': return '#dc3545';
      default: return '#007bff';
    }
  };

  return (
    <PageLayout activePage={activePage} pageTitle="DVE Codex Builder" onSearch={handleSearch}>
      <PageHeader
        title="DVE Codex Builder"
        subtitle="Pre-build custom environments with the toolchains and SDKs of your preference"
        titleColor="#7c4dff"
      />

      {/* Tab Navigation */}
      <GlassyCard className={styles.tabsCard}>
        <div className={styles.tabsContainer}>
          {[
            { id: 'builder', label: 'Model Builder', icon: '🛠️' },
            { id: 'templates', label: 'Templates', icon: '📋' },
            { id: 'projects', label: 'My Projects', icon: '📁' }
          ].map((tab) => (
            <button
              key={tab.id}
              onClick={() => setActiveTab(tab.id)}
              className={`${styles.tabButton} ${activeTab === tab.id ? styles.active : ''}`}
            >
              <span className={styles.tabIcon}>{tab.icon}</span>
              {tab.label}
            </button>
          ))}
        </div>
      </GlassyCard>

      {/* Builder Tab */}
      {activeTab === 'builder' && (
        <div className={styles.builderSection}>
          <GlassyCard className={styles.builderCard}>
            <div className={styles.builderHeader}>
              <h3 className={styles.builderTitle}>
                <i className="fas fa-cogs" style={{ marginRight: '10px', color: '#007bff' }}></i>
                Model Configuration
              </h3>
            </div>
            
            <div className={styles.builderForm}>
              <div className={styles.formSection}>
                <h4>Basic Information</h4>
                <div className={styles.formGrid}>
                  <div className={styles.formGroup}>
                    <label>Model Name</label>
                    <input type="text" placeholder="Enter model name" className={styles.formInput} />
                  </div>
                  <div className={styles.formGroup}>
                    <label>Model Type</label>
                    <select className={styles.formSelect}>
                      <option>Language Model</option>
                      <option>Computer Vision</option>
                      <option>Time Series</option>
                      <option>Reinforcement Learning</option>
                    </select>
                  </div>
                  <div className={styles.formGroup}>
                    <label>Framework</label>
                    <select className={styles.formSelect}>
                      <option>PyTorch</option>
                      <option>TensorFlow</option>
                      <option>JAX</option>
                      <option>Hugging Face</option>
                    </select>
                  </div>
                  <div className={styles.formGroup}>
                    <label>Target Platform</label>
                    <select className={styles.formSelect}>
                      <option>KNIRV Network</option>
                      <option>Local Deployment</option>
                      <option>Cloud Deployment</option>
                      <option>Edge Deployment</option>
                    </select>
                  </div>
                </div>
              </div>

              <div className={styles.formSection}>
                <h4>Architecture Configuration</h4>
                <div className={styles.architectureBuilder}>
                  <div className={styles.layersList}>
                    <div className={styles.layer}>
                      <span className={styles.layerType}>Input Layer</span>
                      <span className={styles.layerParams}>Shape: (batch_size, 512)</span>
                    </div>
                    <div className={styles.layer}>
                      <span className={styles.layerType}>Dense Layer</span>
                      <span className={styles.layerParams}>Units: 256, Activation: ReLU</span>
                    </div>
                    <div className={styles.layer}>
                      <span className={styles.layerType}>Dropout Layer</span>
                      <span className={styles.layerParams}>Rate: 0.2</span>
                    </div>
                    <div className={styles.layer}>
                      <span className={styles.layerType}>Output Layer</span>
                      <span className={styles.layerParams}>Units: 10, Activation: Softmax</span>
                    </div>
                  </div>
                  <div className={styles.layerControls}>
                    <button className={styles.addLayerButton}>
                      <i className="fas fa-plus"></i> Add Layer
                    </button>
                  </div>
                </div>
              </div>

              <div className={styles.formSection}>
                <h4>Training Configuration</h4>
                <div className={styles.formGrid}>
                  <div className={styles.formGroup}>
                    <label>Learning Rate</label>
                    <input type="number" step="0.0001" defaultValue="0.001" className={styles.formInput} />
                  </div>
                  <div className={styles.formGroup}>
                    <label>Batch Size</label>
                    <input type="number" defaultValue="32" className={styles.formInput} />
                  </div>
                  <div className={styles.formGroup}>
                    <label>Epochs</label>
                    <input type="number" defaultValue="100" className={styles.formInput} />
                  </div>
                  <div className={styles.formGroup}>
                    <label>Optimizer</label>
                    <select className={styles.formSelect}>
                      <option>Adam</option>
                      <option>SGD</option>
                      <option>RMSprop</option>
                      <option>AdaGrad</option>
                    </select>
                  </div>
                </div>
              </div>

              <div className={styles.builderActions}>
                <button className={styles.saveButton}>
                  <i className="fas fa-save" style={{ marginRight: '8px' }}></i>
                  Save Configuration
                </button>
                <button className={styles.trainButton}>
                  <i className="fas fa-play" style={{ marginRight: '8px' }}></i>
                  Start Training
                </button>
                <button className={styles.previewButton}>
                  <i className="fas fa-eye" style={{ marginRight: '8px' }}></i>
                  Preview Model
                </button>
              </div>
            </div>
          </GlassyCard>
        </div>
      )}

      {/* Templates Tab */}
      {activeTab === 'templates' && (
        <div className={styles.templatesSection}>
          <div className={styles.templatesGrid}>
            {templates.map((template) => (
              <GlassyCard key={template.id} className={styles.templateCard}>
                <div className={styles.templateHeader}>
                  <h4 className={styles.templateName}>{template.name}</h4>
                  <span className={styles.templateCategory}>{template.category}</span>
                </div>
                <p className={styles.templateDescription}>{template.description}</p>
                <div className={styles.templateMeta}>
                  <div className={styles.templateMetaItem}>
                    <span className={styles.metaLabel}>Complexity:</span>
                    <span 
                      className={styles.metaValue}
                      style={{ color: getComplexityColor(template.complexity) }}
                    >
                      {template.complexity}
                    </span>
                  </div>
                  <div className={styles.templateMetaItem}>
                    <span className={styles.metaLabel}>Time:</span>
                    <span className={styles.metaValue}>{template.estimatedTime}</span>
                  </div>
                </div>
                <div className={styles.templateFeatures}>
                  {template.features.map((feature, index) => (
                    <span key={index} className={styles.featureTag}>{feature}</span>
                  ))}
                </div>
                <div className={styles.templateActions}>
                  <button 
                    onClick={() => setSelectedTemplate(template)}
                    className={styles.useTemplateButton}
                  >
                    Use Template
                  </button>
                  <button className={styles.previewTemplateButton}>
                    Preview
                  </button>
                </div>
              </GlassyCard>
            ))}
          </div>
        </div>
      )}

      {/* Projects Tab */}
      {activeTab === 'projects' && (
        <div className={styles.projectsSection}>
          <GlassyCard className={styles.projectsCard}>
            <div className={styles.projectsHeader}>
              <h3 className={styles.projectsTitle}>
                <i className="fas fa-folder" style={{ marginRight: '10px', color: '#007bff' }}></i>
                My Projects ({projects.length})
              </h3>
              <button className={styles.newProjectButton}>
                <i className="fas fa-plus" style={{ marginRight: '8px' }}></i>
                New Project
              </button>
            </div>
            
            <div className={styles.projectsList}>
              {projects.map((project) => (
                <div key={project.id} className={styles.projectItem}>
                  <div className={styles.projectInfo}>
                    <h4 className={styles.projectName}>{project.name}</h4>
                    <p className={styles.projectDescription}>{project.description}</p>
                    <div className={styles.projectMeta}>
                      <span className={styles.projectType}>{project.type}</span>
                      <span className={styles.projectDate}>Modified: {project.lastModified}</span>
                    </div>
                  </div>
                  <div className={styles.projectStatus}>
                    <div className={styles.statusInfo}>
                      <span 
                        className={styles.statusLabel}
                        style={{ color: getStatusColor(project.status) }}
                      >
                        {project.status}
                      </span>
                      <div className={styles.progressBar}>
                        <div 
                          className={styles.progressFill}
                          style={{ width: `${project.progress}%` }}
                        ></div>
                      </div>
                      <span className={styles.progressText}>{project.progress}%</span>
                    </div>
                    <div className={styles.projectActions}>
                      <button className={styles.editButton} title="Edit">
                        <i className="fas fa-edit"></i>
                      </button>
                      <button className={styles.deployButton} title="Deploy">
                        <i className="fas fa-rocket"></i>
                      </button>
                      <button className={styles.deleteButton} title="Delete">
                        <i className="fas fa-trash"></i>
                      </button>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </GlassyCard>
        </div>
      )}
    </PageLayout>
  );
}
