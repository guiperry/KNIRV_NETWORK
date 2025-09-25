import React, { useState, useEffect } from 'react';
import { useNavigation } from '../hooks/useNavigation';
import PageLayout from '../components/PageLayout';
import PageHeader from '../components/PageHeader';
import GlassyCard from '../components/GlassyCard';
import DataTable from '../components/DataTable';
import api from '../utils/api';
import { useBackend } from '../contexts/BackendContext';
import styles from './my-skills.module.css';

export default function MySkills() {
  const { activePage } = useNavigation('my-skills');
  const { isRunning } = useBackend();
  const [searchQuery, setSearchQuery] = useState('');
  const [showPublishModal, setShowPublishModal] = useState(false);
  const [skills, setSkills] = useState([]);
  const [selectedSkill, setSelectedSkill] = useState(null);
  const [selectedCategory, setSelectedCategory] = useState('all');

  const handleSearch = (query) => {
    setSearchQuery(query);
  };

  // Load skills from Controller API when backend is running
  useEffect(() => {
    if (!isRunning) return;
    (async () => {
      try {
        const resp = await api.get('/api/skills?owner=me');
        const list = (resp.data && resp.data.skills) || [];
        // Normalize minimal fields for the table view
        const normalized = list.map((s, idx) => ({
          id: s.id || String(idx + 1),
          name: s.name || s.metadata?.name || 'LoRA Adapter',
          category: s.metadata?.baseModel ? 'Computer Vision' : 'Language',
          level: 'N/A',
          status: 'Active',
          usage: 0,
          rating: 0,
          description: s.metadata ? JSON.stringify(s.metadata) : 'LoRA adapter',
          version: s.metadata?.version || '1.0.0',
          publishedAt: null,
          lastUpdated: null,
          earnings: '0 NRN',
          downloads: 0,
        }));
        setSkills(normalized);
      } catch (e) {
        console.error('Failed to load skills from /api/skills', e);
        setSkills([]);
      }
    })();
  }, [isRunning]);

  // Filter skills based on search query and category
  const filteredSkills = skills.filter((skill) => {
    const matchesSearch = skill.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
                         skill.description.toLowerCase().includes(searchQuery.toLowerCase()) ||
                         skill.category.toLowerCase().includes(searchQuery.toLowerCase());
    const matchesCategory = selectedCategory === 'all' || skill.category.toLowerCase() === selectedCategory.toLowerCase();
    return matchesSearch && matchesCategory;
  });

  const categories = ['all', 'Development', 'Computer Vision', 'Language', 'Analytics', 'Audio', 'Security'];

  if (!isRunning) {
    return (
      <PageLayout activePage={activePage} pageTitle="My Skills">
        <GlassyCard darker className={styles.emptyState}>
          Backend is not running. Please start the Controller API.
        </GlassyCard>
      </PageLayout>
    );
  }

  // Table headers
  const tableHeaders = [
    { label: 'Skill Name' },
    { label: 'Category' },
    { label: 'Level' },
    { label: 'Status', align: 'center' },
    { label: 'Usage', align: 'right' },
    { label: 'Rating', align: 'center' },
    { label: 'Earnings', align: 'right' },
    { label: 'Actions', align: 'center' },
  ];

  // Render table row
  const renderTableRow = (skill, index) => (
    <tr className={styles.tableRow} key={skill.id}>
      <td className={styles.tableCell}>
        <div className={styles.skillInfo}>
          <div className={styles.skillName}>{skill.name}</div>
          <div className={styles.skillDescription}>{skill.description}</div>
        </div>
      </td>
      <td className={styles.tableCell}>
        <span className={styles.category}>{skill.category}</span>
      </td>
      <td className={styles.tableCell}>
        <span className={`${styles.level} ${styles[skill.level.toLowerCase()]}`}>
          {skill.level}
        </span>
      </td>
      <td className={`${styles.tableCell} ${styles.textCenter}`}>
        <span className={`${styles.status} ${styles[skill.status.toLowerCase()]}`}>
          {skill.status}
        </span>
      </td>
      <td className={`${styles.tableCell} ${styles.textRight}`}>
        <span className={styles.usage}>{skill.usage.toLocaleString()}</span>
      </td>
      <td className={`${styles.tableCell} ${styles.textCenter}`}>
        <div className={styles.rating}>
          <span className={styles.ratingValue}>{skill.rating}</span>
          <div className={styles.stars}>
            {[...Array(5)].map((_, i) => (
              <i 
                key={i} 
                className={`fas fa-star ${i < Math.floor(skill.rating) ? styles.starFilled : styles.starEmpty}`}
              ></i>
            ))}
          </div>
        </div>
      </td>
      <td className={`${styles.tableCell} ${styles.textRight}`}>
        <span className={styles.earnings}>{skill.earnings}</span>
      </td>
      <td className={`${styles.tableCell} ${styles.textCenter}`}>
        <div className={styles.actionButtons}>
          <button 
            onClick={() => setSelectedSkill(skill)}
            className={styles.viewButton}
            title="View Details"
          >
            <i className="fas fa-eye"></i>
          </button>
          <button 
            className={styles.editButton}
            title="Edit Skill"
          >
            <i className="fas fa-edit"></i>
          </button>
          <button 
            className={styles.publishButton}
            title="Publish/Manage"
          >
            <i className="fas fa-upload"></i>
          </button>
        </div>
      </td>
    </tr>
  );

  const handlePublishSkill = () => {
    setShowPublishModal(true);
  };

  const totalEarnings = skills.reduce((acc, skill) => {
    const earnings = parseFloat(skill.earnings.replace(' NRN', ''));
    return acc + earnings;
  }, 0);

  const totalDownloads = skills.reduce((acc, skill) => acc + skill.downloads, 0);

  return (
    <PageLayout activePage={activePage} pageTitle="My Skills" onSearch={handleSearch}>
      <PageHeader 
        title="My Skills"
        subtitle="Manage and monetize your AI skills on the KNIRV network"
        titleColor="#007bff"
      />

      {/* Category Filter */}
      <GlassyCard className={styles.filterCard}>
        <div className={styles.filterHeader}>
          <h3 className={styles.filterTitle}>
            <i className="fas fa-filter" style={{ marginRight: '10px', color: '#007bff' }}></i>
            Filter by Category
          </h3>
        </div>
        <div className={styles.categoryFilters}>
          {categories.map((category) => (
            <button
              key={category}
              onClick={() => setSelectedCategory(category)}
              className={`${styles.categoryButton} ${
                selectedCategory === category ? styles.active : ''
              }`}
            >
              {category === 'all' ? 'All Categories' : category}
            </button>
          ))}
        </div>
      </GlassyCard>

      {/* Statistics Overview */}
      <div className={styles.statsContainer}>
        <GlassyCard className={styles.statCard}>
          <div className={styles.statIcon}>⚡</div>
          <div className={styles.statValue}>{skills.length}</div>
          <div className={styles.statLabel}>Total Skills</div>
        </GlassyCard>
        <GlassyCard className={styles.statCard}>
          <div className={styles.statIcon}>📈</div>
          <div className={styles.statValue}>{skills.filter(s => s.status === 'Published').length}</div>
          <div className={styles.statLabel}>Published</div>
        </GlassyCard>
        <GlassyCard className={styles.statCard}>
          <div className={styles.statIcon}>💰</div>
          <div className={styles.statValue}>{totalEarnings.toFixed(0)} NRN</div>
          <div className={styles.statLabel}>Total Earnings</div>
        </GlassyCard>
        <GlassyCard className={styles.statCard}>
          <div className={styles.statIcon}>📥</div>
          <div className={styles.statValue}>{totalDownloads.toLocaleString()}</div>
          <div className={styles.statLabel}>Total Downloads</div>
        </GlassyCard>
      </div>

      {/* Skills Table */}
      <GlassyCard className={styles.tableCard}>
        <div className={styles.tableHeader}>
          <h3 className={styles.tableTitle}>
            <i className="fas fa-cogs" style={{ marginRight: '10px', color: '#007bff' }}></i>
            My Skills ({filteredSkills.length})
          </h3>
          <button 
            onClick={handlePublishSkill}
            className={styles.publishButton}
          >
            <i className="fas fa-plus" style={{ marginRight: '8px' }}></i>
            Publish New Skill
          </button>
        </div>
        
        {filteredSkills.length > 0 ? (
          <DataTable
            headers={tableHeaders}
            data={filteredSkills}
            renderRow={renderTableRow}
            className={styles.skillsTable}
          />
        ) : (
          <div className={styles.emptyState}>
            <i className="fas fa-cogs" style={{ fontSize: '3rem', color: '#666', marginBottom: '20px' }}></i>
            <h3>No skills found</h3>
            <p>Create and publish your first skill to start earning on the KNIRV network.</p>
            <button onClick={handlePublishSkill} className={styles.publishButton}>
              <i className="fas fa-plus" style={{ marginRight: '8px' }}></i>
              Publish Your First Skill
            </button>
          </div>
        )}
      </GlassyCard>

      {/* Skill Details Modal */}
      {selectedSkill && (
        <div className={styles.modal}>
          <div className={styles.modalContent}>
            <div className={styles.modalHeader}>
              <h3>{selectedSkill.name}</h3>
              <button onClick={() => setSelectedSkill(null)} className={styles.closeButton}>×</button>
            </div>
            <div className={styles.modalBody}>
              <div className={styles.skillDetails}>
                <div className={styles.detailSection}>
                  <h4>Skill Information</h4>
                  <div className={styles.detailGrid}>
                    <div className={styles.detailItem}>
                      <span className={styles.detailLabel}>Category:</span>
                      <span className={styles.detailValue}>{selectedSkill.category}</span>
                    </div>
                    <div className={styles.detailItem}>
                      <span className={styles.detailLabel}>Level:</span>
                      <span className={styles.detailValue}>{selectedSkill.level}</span>
                    </div>
                    <div className={styles.detailItem}>
                      <span className={styles.detailLabel}>Version:</span>
                      <span className={styles.detailValue}>v{selectedSkill.version}</span>
                    </div>
                    <div className={styles.detailItem}>
                      <span className={styles.detailLabel}>Status:</span>
                      <span className={`${styles.detailValue} ${styles[selectedSkill.status.toLowerCase()]}`}>
                        {selectedSkill.status}
                      </span>
                    </div>
                  </div>
                </div>

                <div className={styles.detailSection}>
                  <h4>Performance & Earnings</h4>
                  <div className={styles.metricsGrid}>
                    <div className={styles.metricCard}>
                      <div className={styles.metricValue}>{selectedSkill.usage.toLocaleString()}</div>
                      <div className={styles.metricLabel}>Total Usage</div>
                    </div>
                    <div className={styles.metricCard}>
                      <div className={styles.metricValue}>{selectedSkill.rating}</div>
                      <div className={styles.metricLabel}>Rating</div>
                    </div>
                    <div className={styles.metricCard}>
                      <div className={styles.metricValue}>{selectedSkill.downloads.toLocaleString()}</div>
                      <div className={styles.metricLabel}>Downloads</div>
                    </div>
                    <div className={styles.metricCard}>
                      <div className={styles.metricValue}>{selectedSkill.earnings}</div>
                      <div className={styles.metricLabel}>Earnings</div>
                    </div>
                  </div>
                </div>

                <div className={styles.detailSection}>
                  <h4>Description</h4>
                  <p className={styles.skillDescription}>{selectedSkill.description}</p>
                </div>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Publish Modal */}
      {showPublishModal && (
        <div className={styles.modal}>
          <div className={styles.modalContent}>
            <div className={styles.modalHeader}>
              <h3>Publish New Skill</h3>
              <button onClick={() => setShowPublishModal(false)} className={styles.closeButton}>×</button>
            </div>
            <div className={styles.modalBody}>
              <p>Skill publishing functionality would be implemented here, integrating with the KNIRV skill registry system.</p>
              <div className={styles.modalActions}>
                <button onClick={() => setShowPublishModal(false)} className={styles.cancelButton}>
                  Cancel
                </button>
                <button className={styles.publishButton}>
                  Publish Skill
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
    </PageLayout>
  );
}
