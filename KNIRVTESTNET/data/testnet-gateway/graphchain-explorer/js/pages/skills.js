/**
 * Skills Page Controller
 * Manages the SkillNodes page functionality
 */

class SkillsController {
  constructor() {
    this.api = window.graphChainAPI;
    this.sse = window.graphChainSSE;
    this.skills = [];
    this.filteredSkills = [];
    this.skillCards = [];
    this.loading = true;
    this.error = null;
    this.currentPage = 1;
    this.pageSize = 20;
    this.hasMore = false;
    this.searchQuery = '';
    this.activeFilters = { all: true };
  }

  /**
   * Initialize the skills page
   */
  async init() {
    console.log('Initializing Skills Controller...');
    
    try {
      this.setupEventListeners();
      this.setupSSE();
      await this.loadSkills();
      this.render();
      
      // Register with global app
      if (window.graphChainApp) {
        window.graphChainApp.registerComponent('skills', this);
      }
      
      // Make controller globally available
      window.skillsController = this;
      
      console.log('Skills Controller initialized successfully');
    } catch (error) {
      console.error('Failed to initialize skills page:', error);
      this.handleError(error);
    }
  }

  /**
   * Setup event listeners
   */
  setupEventListeners() {
    // Search input
    const searchInput = document.getElementById('skills-search');
    if (searchInput) {
      searchInput.addEventListener('input', this.debounce((e) => {
        this.searchQuery = e.target.value;
        this.applyFilters();
      }, 300));
    }

    // Filter buttons
    const filterButtons = document.querySelectorAll('.filter-button');
    filterButtons.forEach(button => {
      button.addEventListener('click', (e) => {
        this.handleFilterClick(e.target);
      });
    });

    // Load more button
    const loadMoreButton = document.getElementById('load-more-button');
    if (loadMoreButton) {
      loadMoreButton.addEventListener('click', () => {
        this.loadMoreSkills();
      });
    }

    // Global search
    const globalSearch = document.getElementById('global-search');
    if (globalSearch) {
      globalSearch.addEventListener('keypress', (e) => {
        if (e.key === 'Enter') {
          this.searchQuery = globalSearch.value;
          this.applyFilters();
        }
      });
    }
  }

  /**
   * Setup SSE event listeners
   */
  setupSSE() {
    if (!this.sse) {
      console.warn('SSE client not available');
      return;
    }

    // New skill added
    this.sse.on('skill_added', (skill) => {
      this.addNewSkill(skill);
    });

    // Skills data updates
    this.sse.on('skills_data', (data) => {
      if (data.skills) {
        this.updateSkillsData(data.skills);
      }
    });
  }

  /**
   * Load skills from API
   */
  async loadSkills() {
    console.log('Loading skills...');
    this.loading = true;
    this.showLoadingState();
    
    try {
      this.skills = await this.api.getSkills();
      this.filteredSkills = [...this.skills];
      this.error = null;
      
      // Update total count
      this.updateTotalCount(this.skills.length);
      
      console.log(`Loaded ${this.skills.length} skills`);
      
    } catch (error) {
      console.error('Failed to load skills:', error);
      this.error = error;
      throw error;
    } finally {
      this.loading = false;
    }
  }

  /**
   * Load more skills (pagination)
   */
  async loadMoreSkills() {
    // For now, just show more from existing data
    // In a real implementation, this would fetch more from the API
    this.currentPage++;
    this.renderSkills();
  }

  /**
   * Render the skills page
   */
  render() {
    if (this.loading) {
      this.showLoadingState();
      return;
    }
    
    if (this.error) {
      this.showErrorState(this.error.message);
      return;
    }
    
    this.showSkillsContent();
    this.renderSkills();
  }

  /**
   * Show loading state
   */
  showLoadingState() {
    const loadingState = document.getElementById('loading-state');
    const errorState = document.getElementById('error-state');
    const skillsContent = document.getElementById('skills-content');
    
    if (loadingState) loadingState.style.display = 'block';
    if (errorState) errorState.style.display = 'none';
    if (skillsContent) skillsContent.style.display = 'none';
  }

  /**
   * Show error state
   */
  showErrorState(message) {
    const loadingState = document.getElementById('loading-state');
    const errorState = document.getElementById('error-state');
    const skillsContent = document.getElementById('skills-content');
    const errorMessage = document.getElementById('error-message');
    
    if (loadingState) loadingState.style.display = 'none';
    if (errorState) errorState.style.display = 'block';
    if (skillsContent) skillsContent.style.display = 'none';
    if (errorMessage) errorMessage.textContent = message;
  }

  /**
   * Show skills content
   */
  showSkillsContent() {
    const loadingState = document.getElementById('loading-state');
    const errorState = document.getElementById('error-state');
    const skillsContent = document.getElementById('skills-content');
    
    if (loadingState) loadingState.style.display = 'none';
    if (errorState) errorState.style.display = 'none';
    if (skillsContent) skillsContent.style.display = 'block';
  }

  /**
   * Render skills grid
   */
  renderSkills() {
    const container = document.getElementById('skills-container');
    const emptyState = document.getElementById('empty-state');
    const loadMoreSection = document.getElementById('load-more-section');
    
    if (!container) return;

    // Clear existing cards
    this.skillCards.forEach(card => card.destroy());
    this.skillCards = [];
    container.innerHTML = '';

    // Check if we have skills to show
    if (this.filteredSkills.length === 0) {
      if (emptyState) emptyState.style.display = 'block';
      if (loadMoreSection) loadMoreSection.style.display = 'none';
      return;
    }

    if (emptyState) emptyState.style.display = 'none';

    // Calculate how many skills to show
    const skillsToShow = this.filteredSkills.slice(0, this.currentPage * this.pageSize);
    
    // Create skill cards
    this.skillCards = SkillCard.createMultiple(skillsToShow, container);

    // Highlight search terms
    if (this.searchQuery) {
      this.skillCards.forEach(card => {
        card.highlightSearch(this.searchQuery);
      });
    }

    // Show/hide load more button
    this.hasMore = skillsToShow.length < this.filteredSkills.length;
    if (loadMoreSection) {
      loadMoreSection.style.display = this.hasMore ? 'block' : 'none';
    }
  }

  /**
   * Handle filter button click
   */
  handleFilterClick(button) {
    // Remove active class from all buttons
    document.querySelectorAll('.filter-button').forEach(btn => {
      btn.classList.remove('active');
    });
    
    // Add active class to clicked button
    button.classList.add('active');
    
    // Update active filters
    const filter = button.dataset.filter;
    this.activeFilters = { [filter]: true };
    
    // Apply filters
    this.applyFilters();
  }

  /**
   * Apply search and filters
   */
  applyFilters() {
    this.filteredSkills = this.skills.filter(skill => {
      // Search filter
      if (this.searchQuery && !this.matchesSearch(skill, this.searchQuery)) {
        return false;
      }
      
      // Category filters
      if (this.activeFilters.validated && !skill.validation?.is_validated) {
        return false;
      }
      
      if (this.activeFilters.pending && skill.validation?.is_validated) {
        return false;
      }
      
      if (this.activeFilters['high-performance'] && 
          (!skill.performance || skill.performance.success_rate < 0.8)) {
        return false;
      }
      
      return true;
    });
    
    // Reset pagination
    this.currentPage = 1;
    
    // Re-render
    this.renderSkills();
  }

  /**
   * Check if skill matches search query
   */
  matchesSearch(skill, query) {
    const searchText = query.toLowerCase();
    const skillType = skill.skill_type.toLowerCase();
    const capabilities = skill.capabilities.join(' ').toLowerCase();
    
    return skillType.includes(searchText) || capabilities.includes(searchText);
  }

  /**
   * Add new skill to the list
   */
  addNewSkill(skill) {
    this.skills.unshift(skill);
    this.updateTotalCount(this.skills.length);
    this.applyFilters();
  }

  /**
   * Update skills data from SSE
   */
  updateSkillsData(newSkills) {
    this.skills = newSkills;
    this.updateTotalCount(this.skills.length);
    this.applyFilters();
  }

  /**
   * Update total count display
   */
  updateTotalCount(count) {
    const totalElement = document.getElementById('total-skills');
    if (totalElement) {
      totalElement.textContent = count.toLocaleString();
    }
  }

  /**
   * Handle errors
   */
  handleError(error) {
    this.error = error;
    this.loading = false;
    this.showErrorState(error.message || 'An unexpected error occurred');
  }

  /**
   * Debounce utility
   */
  debounce(func, wait) {
    let timeout;
    return function executedFunction(...args) {
      const later = () => {
        clearTimeout(timeout);
        func(...args);
      };
      clearTimeout(timeout);
      timeout = setTimeout(later, wait);
    };
  }

  /**
   * Destroy the controller
   */
  destroy() {
    // Clean up skill cards
    this.skillCards.forEach(card => card.destroy());
    this.skillCards = [];
    
    // Remove SSE listeners
    if (this.sse) {
      this.sse.off('skill_added');
      this.sse.off('skills_data');
    }
  }

  /**
   * Get current state
   */
  getState() {
    return {
      loading: this.loading,
      error: this.error,
      skills: this.skills,
      filteredSkills: this.filteredSkills,
      searchQuery: this.searchQuery,
      activeFilters: this.activeFilters
    };
  }
}

// Initialize skills page when DOM is ready
let skillsController = null;

function initSkills() {
  if (skillsController) {
    skillsController.destroy();
  }
  
  skillsController = new SkillsController();
  skillsController.init();
}

// Auto-initialize if we're on the skills page
if (document.readyState === 'loading') {
  document.addEventListener('DOMContentLoaded', initSkills);
} else {
  initSkills();
}

// Export for module usage
if (typeof module !== 'undefined' && module.exports) {
  module.exports = SkillsController;
}
