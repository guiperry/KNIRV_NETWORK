import React, { useEffect, useState } from 'react';
import { useNavigation } from '../hooks/useNavigation';
import PageLayout from '../components/PageLayout';
import PageHeader from '../components/PageHeader';
import GlassyCard from '../components/GlassyCard';
import SkillNodeCard from '../components/GraphChain/SkillNodeCard';
import LoadingSpinner from '../components/GraphChain/LoadingSpinner';
import { graphChainApi } from '../services/graphchain-api';
import styles from './graphchain-skills.module.css';

export default function GraphChainSkills() {
  const { activePage } = useNavigation('graphchain-skills');
  const [skills, setSkills] = useState([]);
  const [filteredSkills, setFilteredSkills] = useState([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState(null);
  const [page, setPage] = useState(1);
  const [filter, setFilter] = useState('all');
  const [searchTerm, setSearchTerm] = useState('');

  const skillsPerPage = 10;

  useEffect(() => {
    const fetchSkills = async () => {
      setIsLoading(true);
      setError(null);

      try {
        const fetchedSkills = await graphChainApi.getAllSkills();
        setSkills(fetchedSkills);
      } catch (err) {
        setError(err.message || 'Failed to fetch SkillNodes');
      } finally {
        setIsLoading(false);
      }
    };

    fetchSkills();
  }, []);

  // Filter and search skills
  useEffect(() => {
    let filtered = skills;

    // Apply filter
    if (filter !== 'all') {
      filtered = filtered.filter(skill => {
        switch (filter) {
          case 'validated':
            return skill.validation?.is_validated === true;
          case 'unvalidated':
            return !skill.validation?.is_validated;
          case 'high-performance':
            return skill.performance && skill.performance.success_rate > 0.8;
          default:
            return true;
        }
      });
    }

    // Apply search
    if (searchTerm) {
      filtered = filtered.filter(skill =>
        skill.skill_type.toLowerCase().includes(searchTerm.toLowerCase()) ||
        skill.capabilities.some(cap => cap.toLowerCase().includes(searchTerm.toLowerCase()))
      );
    }

    setFilteredSkills(filtered);
    setPage(1); // Reset to first page when filters change
  }, [skills, filter, searchTerm]);

  const totalPages = Math.ceil(filteredSkills.length / skillsPerPage);
  const paginatedSkills = filteredSkills.slice(
    (page - 1) * skillsPerPage,
    page * skillsPerPage
  );

  const handlePageChange = (newPage) => {
    if (newPage >= 1 && newPage <= totalPages) {
      setPage(newPage);
      window.scrollTo({ top: 0, behavior: 'smooth' });
    }
  };

  const getPageNumbers = () => {
    const pages = [];
    const maxPages = 5;
    let startPage = Math.max(1, page - Math.floor(maxPages / 2));
    let endPage = Math.min(totalPages, startPage + maxPages - 1);

    if (endPage - startPage < maxPages - 1) {
      startPage = Math.max(1, endPage - maxPages + 1);
    }

    for (let i = startPage; i <= endPage; i++) {
      pages.push(i);
    }
    return pages;
  };

  return (
    <PageLayout activePage={activePage} pageTitle="SkillNodes">
      <PageHeader
        title="SkillNodes"
        subtitle="Browse all SkillNodes in the GraphChain"
        titleColor="#4ade80"
      />

      <div className={styles.skillsContainer}>
        {/* Search and Filter */}
        <GlassyCard className={styles.filterCard}>
          <div className={styles.filterRow}>
            <div className={styles.filterGroup}>
              <label className={styles.filterLabel}>Search SkillNodes</label>
              <input
                type="text"
                placeholder="Search by skill type or capabilities..."
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
                className={styles.searchInput}
              />
            </div>
            <div className={styles.filterGroup}>
              <label className={styles.filterLabel}>Filter</label>
              <select
                value={filter}
                onChange={(e) => setFilter(e.target.value)}
                className={styles.filterSelect}
              >
                <option value="all">All SkillNodes</option>
                <option value="validated">Validated Only</option>
                <option value="unvalidated">Unvalidated Only</option>
                <option value="high-performance">High Performance</option>
              </select>
            </div>
          </div>

          {/* Stats */}
          <div className={styles.statsRow}>
            <div className={styles.statItem}>
              <div className={styles.statValue}>{skills.length}</div>
              <div className={styles.statLabel}>Total SkillNodes</div>
            </div>
            <div className={styles.statItem}>
              <div className={styles.statValue}>{filteredSkills.length}</div>
              <div className={styles.statLabel}>Filtered Results</div>
            </div>
            <div className={styles.statItem}>
              <div className={styles.statValue}>{page}</div>
              <div className={styles.statLabel}>Current Page</div>
            </div>
            <div className={styles.statItem}>
              <div className={styles.statValue}>{totalPages}</div>
              <div className={styles.statLabel}>Total Pages</div>
            </div>
          </div>
        </GlassyCard>

        {/* SkillNodes List */}
        {isLoading ? (
          <div className={styles.loadingContainer}>
            <LoadingSpinner size="large" text="Loading SkillNodes..." />
          </div>
        ) : error ? (
          <GlassyCard className={styles.errorCard}>
            <div className={styles.errorIcon}>
              <i className="fas fa-exclamation-triangle"></i>
            </div>
            <div className={styles.errorTitle}>Error Loading SkillNodes</div>
            <div className={styles.errorMessage}>{error}</div>
          </GlassyCard>
        ) : paginatedSkills.length === 0 ? (
          <GlassyCard className={styles.emptyCard}>
            <i className="fas fa-brain"></i>
            <h3>No SkillNodes Found</h3>
            <p>Try adjusting your search or filter criteria.</p>
          </GlassyCard>
        ) : (
          <div className={styles.skillsList}>
            {paginatedSkills.map((skill) => (
              <SkillNodeCard key={skill.id} skill={skill} />
            ))}
          </div>
        )}

        {/* Pagination */}
        {totalPages > 1 && !isLoading && !error && (
          <div className={styles.pagination}>
            <button
              onClick={() => handlePageChange(page - 1)}
              disabled={page === 1}
              className={styles.paginationButton}
            >
              <i className="fas fa-chevron-left"></i>
              <span>Previous</span>
            </button>

            <div className={styles.pageNumbers}>
              {getPageNumbers().map((pageNum) => (
                <button
                  key={pageNum}
                  onClick={() => handlePageChange(pageNum)}
                  className={`${styles.pageNumber} ${page === pageNum ? styles.pageNumberActive : ''}`}
                >
                  {pageNum}
                </button>
              ))}
            </div>

            <button
              onClick={() => handlePageChange(page + 1)}
              disabled={page === totalPages}
              className={styles.paginationButton}
            >
              <span>Next</span>
              <i className="fas fa-chevron-right"></i>
            </button>
          </div>
        )}
      </div>
    </PageLayout>
  );
}
