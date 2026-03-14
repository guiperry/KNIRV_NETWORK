/**
 * Dashboard Page Controller
 * Manages the main dashboard page functionality
 */

class DashboardController {
  constructor() {
    this.api = window.graphChainAPI;
    this.sse = window.graphChainSSE;
    this.stats = null;
    this.recentSkills = [];
    this.loading = true;
    this.error = null;
    this.skillCards = [];
    this.refreshInterval = null;
    this.autoRefreshEnabled = true;
    this.autoRefreshDelay = 30000; // 30 seconds
  }

  /**
   * Initialize the dashboard
   */
  async init() {
    console.log('Initializing Dashboard Controller...');
    
    try {
      this.setupSSE();
      await this.loadDashboardData();
      this.render();
      this.startAutoRefresh();
      
      // Register with global app
      if (window.graphChainApp) {
        window.graphChainApp.registerComponent('dashboard', this);
      }
      
      console.log('Dashboard Controller initialized successfully');
    } catch (error) {
      console.error('Failed to initialize dashboard:', error);
      this.handleError(error);
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

    // Height updates
    this.sse.on('height_changed', (height) => {
      this.updateHeight(height);
    });

    // Skills data updates
    this.sse.on('skills_data', (data) => {
      if (data.skills) {
        this.updateRecentSkills(data.skills);
      }
    });

    // Stats updates
    this.sse.on('stats_update', (data) => {
      if (data.stats) {
        this.updateStats(data.stats);
      }
    });

    // New skill added
    this.sse.on('skill_added', (skill) => {
      this.addRecentSkill(skill);
    });

    // Connection status
    this.sse.on('connection:open', () => {
      this.updateConnectionStatus(true);
    });

    this.sse.on('connection:error', () => {
      this.updateConnectionStatus(false);
    });

    this.sse.on('connection:close', () => {
      this.updateConnectionStatus(false);
    });
  }

  /**
   * Load dashboard data
   */
  async loadDashboardData() {
    console.log('Loading dashboard data...');
    this.loading = true;
    this.showLoadingState();
    
    try {
      const [height, skills, errors] = await Promise.all([
        this.api.getHeight().catch(() => 0),
        this.api.getRecentSkills(5).catch(() => []),
        this.api.getRecentErrors(5).catch(() => [])
      ]);
      
      // Calculate stats
      this.stats = {
        height: height,
        totalSkillNodes: skills.length,
        totalErrorNodes: errors.length,
        avgResolutionTime: this.calculateAvgResolutionTime(skills)
      };
      
      this.recentSkills = skills;
      this.error = null;
      
      console.log('Dashboard data loaded successfully:', this.stats);
      
    } catch (error) {
      console.error('Failed to load dashboard data:', error);
      this.error = error;
      throw error;
    } finally {
      this.loading = false;
    }
  }

  /**
   * Render the dashboard
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
    
    this.showMainContent();
    this.renderStats();
    this.renderRecentSkills();
  }

  /**
   * Show loading state
   */
  showLoadingState() {
    if (window.graphChainApp) {
      window.graphChainApp.showLoadingState();
    }
  }

  /**
   * Show error state
   */
  showErrorState(message) {
    if (window.graphChainApp) {
      window.graphChainApp.showErrorState(message);
    }
  }

  /**
   * Show main content
   */
  showMainContent() {
    if (window.graphChainApp) {
      window.graphChainApp.showMainContent();
    }
  }

  /**
   * Render statistics
   */
  renderStats() {
    if (!this.stats) return;

    // Update individual stat elements
    this.updateStatElement('stat-height', this.stats.height);
    this.updateStatElement('stat-skills', this.stats.totalSkillNodes);
    this.updateStatElement('stat-errors', this.stats.totalErrorNodes);
    this.updateStatElement('stat-resolution-time', `${this.stats.avgResolutionTime.toFixed(1)}s`);
  }

  /**
   * Update individual stat element
   */
  updateStatElement(elementId, value) {
    const element = document.getElementById(elementId);
    if (element) {
      // Animate the value change
      const currentValue = element.textContent;
      if (currentValue !== String(value)) {
        element.style.transform = 'scale(1.1)';
        setTimeout(() => {
          element.textContent = typeof value === 'number' ? value.toLocaleString() : value;
          element.style.transform = 'scale(1)';
        }, 150);
      }
    }
  }

  /**
   * Render recent skills
   */
  renderRecentSkills() {
    const container = document.getElementById('recent-skills-container');
    if (!container) return;

    // Clear existing cards
    this.skillCards.forEach(card => card.destroy());
    this.skillCards = [];
    container.innerHTML = '';

    if (this.recentSkills.length === 0) {
      container.innerHTML = `
        <div class="empty-state">
          <em class="icon ni ni-cpu" style="font-size: 3rem; color: var(--gray-500); opacity: 0.5; margin-bottom: 1rem;"></em>
          <h3>No SkillNodes Found</h3>
          <p>No SkillNodes have been created yet.</p>
        </div>
      `;
      return;
    }

    // Create skill cards
    this.skillCards = SkillCard.createMultiple(this.recentSkills, container);
  }

  /**
   * Update height display
   */
  updateHeight(newHeight) {
    if (this.stats) {
      this.stats.height = newHeight;
      this.updateStatElement('stat-height', newHeight);
    }
  }

  /**
   * Update recent skills
   */
  updateRecentSkills(skills) {
    this.recentSkills = skills.slice(0, 5);
    this.renderRecentSkills();
  }

  /**
   * Add new skill to recent skills
   */
  addRecentSkill(skill) {
    this.recentSkills.unshift(skill);
    this.recentSkills = this.recentSkills.slice(0, 5);
    this.renderRecentSkills();
    
    // Update skill count
    if (this.stats) {
      this.stats.totalSkillNodes++;
      this.updateStatElement('stat-skills', this.stats.totalSkillNodes);
    }
  }

  /**
   * Update stats
   */
  updateStats(newStats) {
    this.stats = { ...this.stats, ...newStats };
    this.renderStats();
  }

  /**
   * Update connection status indicator
   */
  updateConnectionStatus(isConnected) {
    const statusDot = document.querySelector('.status-dot');
    const statusText = document.querySelector('.status-indicator span');
    
    if (statusDot) {
      statusDot.style.backgroundColor = isConnected ? 'var(--success-green)' : 'var(--error-red)';
    }
    
    if (statusText) {
      statusText.textContent = isConnected ? 'Live Connection' : 'Connection Lost';
    }
  }

  /**
   * Calculate average resolution time
   */
  calculateAvgResolutionTime(skills) {
    const skillsWithPerformance = skills.filter(skill => skill.performance);
    if (skillsWithPerformance.length === 0) return 0;
    
    const totalTime = skillsWithPerformance.reduce(
      (sum, skill) => sum + (skill.performance.avg_resolution_time || 0), 0
    );
    
    return totalTime / skillsWithPerformance.length;
  }

  /**
   * Start auto-refresh
   */
  startAutoRefresh() {
    if (!this.autoRefreshEnabled) return;
    
    this.refreshInterval = setInterval(async () => {
      try {
        await this.loadDashboardData();
        this.render();
      } catch (error) {
        console.error('Auto-refresh failed:', error);
      }
    }, this.autoRefreshDelay);
  }

  /**
   * Stop auto-refresh
   */
  stopAutoRefresh() {
    if (this.refreshInterval) {
      clearInterval(this.refreshInterval);
      this.refreshInterval = null;
    }
  }

  /**
   * Manual refresh
   */
  async refresh() {
    try {
      await this.loadDashboardData();
      this.render();
    } catch (error) {
      console.error('Manual refresh failed:', error);
      this.handleError(error);
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
   * Destroy the controller
   */
  destroy() {
    this.stopAutoRefresh();
    
    // Clean up skill cards
    this.skillCards.forEach(card => card.destroy());
    this.skillCards = [];
    
    // Remove SSE listeners
    if (this.sse) {
      this.sse.off('height_changed');
      this.sse.off('skills_data');
      this.sse.off('stats_update');
      this.sse.off('skill_added');
      this.sse.off('connection:open');
      this.sse.off('connection:error');
      this.sse.off('connection:close');
    }
  }

  /**
   * Get current state
   */
  getState() {
    return {
      loading: this.loading,
      error: this.error,
      stats: this.stats,
      recentSkills: this.recentSkills
    };
  }
}

// Initialize dashboard when DOM is ready
let dashboardController = null;

function initDashboard() {
  if (dashboardController) {
    dashboardController.destroy();
  }
  
  dashboardController = new DashboardController();
  dashboardController.init();
}

// Auto-initialize if we're on the dashboard page
if (document.readyState === 'loading') {
  document.addEventListener('DOMContentLoaded', initDashboard);
} else {
  initDashboard();
}

// Export for module usage
if (typeof module !== 'undefined' && module.exports) {
  module.exports = DashboardController;
}
