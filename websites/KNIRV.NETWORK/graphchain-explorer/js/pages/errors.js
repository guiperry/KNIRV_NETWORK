/**
 * Errors Page Controller
 * Manages the ErrorNodes page functionality
 */

class ErrorsController {
  constructor() {
    this.api = window.graphChainAPI;
    this.sse = window.graphChainSSE;
    this.errors = [];
    this.filteredErrors = [];
    this.errorCards = [];
    this.loading = true;
    this.error = null;
    this.currentPage = 1;
    this.pageSize = 20;
    this.hasMore = false;
    this.searchQuery = '';
    this.activeFilters = { all: true };
  }

  /**
   * Initialize the errors page
   */
  async init() {
    console.log('Initializing Errors Controller...');
    
    try {
      this.setupEventListeners();
      this.setupSSE();
      await this.loadErrors();
      this.render();
      
      // Register with global app
      if (window.graphChainApp) {
        window.graphChainApp.registerComponent('errors', this);
      }
      
      // Make controller globally available
      window.errorsController = this;
      
      console.log('Errors Controller initialized successfully');
    } catch (error) {
      console.error('Failed to initialize errors page:', error);
      this.handleError(error);
    }
  }

  /**
   * Setup event listeners
   */
  setupEventListeners() {
    // Search input
    const searchInput = document.getElementById('errors-search');
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
        this.loadMoreErrors();
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

    // New error created
    this.sse.on('error_created', (error) => {
      this.addNewError(error);
    });

    // Error resolved
    this.sse.on('error_resolved', (error) => {
      this.updateError(error);
    });

    // Errors data updates
    this.sse.on('errors_data', (data) => {
      if (data.errors) {
        this.updateErrorsData(data.errors);
      }
    });
  }

  /**
   * Load errors from API
   */
  async loadErrors() {
    console.log('Loading errors...');
    this.loading = true;
    this.showLoadingState();
    
    try {
      this.errors = await this.api.getErrors();
      this.filteredErrors = [...this.errors];
      this.error = null;
      
      // Update total count
      this.updateTotalCount(this.errors.length);
      
      console.log(`Loaded ${this.errors.length} errors`);
      
    } catch (error) {
      console.error('Failed to load errors:', error);
      this.error = error;
      throw error;
    } finally {
      this.loading = false;
    }
  }

  /**
   * Load more errors (pagination)
   */
  async loadMoreErrors() {
    // For now, just show more from existing data
    // In a real implementation, this would fetch more from the API
    this.currentPage++;
    this.renderErrors();
  }

  /**
   * Render the errors page
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
    
    this.showErrorsContent();
    this.renderErrors();
  }

  /**
   * Show loading state
   */
  showLoadingState() {
    const loadingState = document.getElementById('loading-state');
    const errorState = document.getElementById('error-state');
    const errorsContent = document.getElementById('errors-content');
    
    if (loadingState) loadingState.style.display = 'block';
    if (errorState) errorState.style.display = 'none';
    if (errorsContent) errorsContent.style.display = 'none';
  }

  /**
   * Show error state
   */
  showErrorState(message) {
    const loadingState = document.getElementById('loading-state');
    const errorState = document.getElementById('error-state');
    const errorsContent = document.getElementById('errors-content');
    const errorMessage = document.getElementById('error-message');
    
    if (loadingState) loadingState.style.display = 'none';
    if (errorState) errorState.style.display = 'block';
    if (errorsContent) errorsContent.style.display = 'none';
    if (errorMessage) errorMessage.textContent = message;
  }

  /**
   * Show errors content
   */
  showErrorsContent() {
    const loadingState = document.getElementById('loading-state');
    const errorState = document.getElementById('error-state');
    const errorsContent = document.getElementById('errors-content');
    
    if (loadingState) loadingState.style.display = 'none';
    if (errorState) errorState.style.display = 'none';
    if (errorsContent) errorsContent.style.display = 'block';
  }

  /**
   * Render errors grid
   */
  renderErrors() {
    const container = document.getElementById('errors-container');
    const emptyState = document.getElementById('empty-state');
    const loadMoreSection = document.getElementById('load-more-section');
    
    if (!container) return;

    // Clear existing cards
    this.errorCards.forEach(card => card.destroy());
    this.errorCards = [];
    container.innerHTML = '';

    // Check if we have errors to show
    if (this.filteredErrors.length === 0) {
      if (emptyState) emptyState.style.display = 'block';
      if (loadMoreSection) loadMoreSection.style.display = 'none';
      return;
    }

    if (emptyState) emptyState.style.display = 'none';

    // Calculate how many errors to show
    const errorsToShow = this.filteredErrors.slice(0, this.currentPage * this.pageSize);
    
    // Create error cards
    this.errorCards = ErrorCard.createMultiple(errorsToShow, container);

    // Show/hide load more button
    this.hasMore = errorsToShow.length < this.filteredErrors.length;
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
    this.filteredErrors = this.errors.filter(error => {
      // Search filter
      if (this.searchQuery && !this.matchesSearch(error, this.searchQuery)) {
        return false;
      }
      
      // Category filters
      if (this.activeFilters.resolved && error.resolution_status !== 'resolved') {
        return false;
      }
      
      if (this.activeFilters.pending && error.resolution_status !== 'pending') {
        return false;
      }
      
      if (this.activeFilters.critical && error.severity < 4) {
        return false;
      }
      
      return true;
    });
    
    // Reset pagination
    this.currentPage = 1;
    
    // Re-render
    this.renderErrors();
  }

  /**
   * Check if error matches search query
   */
  matchesSearch(error, query) {
    const searchText = query.toLowerCase();
    const errorType = error.error_type.toLowerCase();
    const description = error.description.toLowerCase();
    
    return errorType.includes(searchText) || description.includes(searchText);
  }

  /**
   * Add new error to the list
   */
  addNewError(error) {
    this.errors.unshift(error);
    this.updateTotalCount(this.errors.length);
    this.applyFilters();
  }

  /**
   * Update existing error
   */
  updateError(updatedError) {
    const index = this.errors.findIndex(error => error.id === updatedError.id);
    if (index !== -1) {
      this.errors[index] = updatedError;
      this.applyFilters();
    }
  }

  /**
   * Update errors data from SSE
   */
  updateErrorsData(newErrors) {
    this.errors = newErrors;
    this.updateTotalCount(this.errors.length);
    this.applyFilters();
  }

  /**
   * Update total count display
   */
  updateTotalCount(count) {
    const totalElement = document.getElementById('total-errors');
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
    // Clean up error cards
    this.errorCards.forEach(card => card.destroy());
    this.errorCards = [];
    
    // Remove SSE listeners
    if (this.sse) {
      this.sse.off('error_created');
      this.sse.off('error_resolved');
      this.sse.off('errors_data');
    }
  }

  /**
   * Get current state
   */
  getState() {
    return {
      loading: this.loading,
      error: this.error,
      errors: this.errors,
      filteredErrors: this.filteredErrors,
      searchQuery: this.searchQuery,
      activeFilters: this.activeFilters
    };
  }
}

// Initialize errors page when DOM is ready
let errorsController = null;

function initErrors() {
  if (errorsController) {
    errorsController.destroy();
  }
  
  errorsController = new ErrorsController();
  errorsController.init();
}

// Auto-initialize if we're on the errors page
if (document.readyState === 'loading') {
  document.addEventListener('DOMContentLoaded', initErrors);
} else {
  initErrors();
}

// Export for module usage
if (typeof module !== 'undefined' && module.exports) {
  module.exports = ErrorsController;
}
