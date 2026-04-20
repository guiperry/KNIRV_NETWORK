/**
 * ErrorCard Component
 * Displays ErrorNode information in a card format
 */

class ErrorCard {
  constructor(error, container) {
    this.error = error;
    this.container = container;
    this.element = null;
    this.expanded = false;
    this.relatedSkills = [];
    this.loadingRelatedSkills = false;
  }

  /**
   * Render the error card
   */
  render() {
    this.element = document.createElement('div');
    this.element.className = 'error-card';
    this.element.innerHTML = this.getTemplate();
    this.attachEvents();
    return this.element;
  }

  /**
   * Get the HTML template for the error card
   */
  getTemplate() {
    return `
      <div class="error-card-header">
        <div class="error-icon">
          <em class="icon ni ni-alert-triangle"></em>
        </div>
        <div class="error-info">
          <h3 class="error-type">${this.escapeHtml(this.error.error_type)}</h3>
          <p class="error-description">${this.escapeHtml(this.error.description)}</p>
          <div class="error-meta">
            <span class="error-timestamp">
              <em class="icon ni ni-clock"></em>
              ${this.formatTime(this.error.timestamp)}
            </span>
          </div>
        </div>
        <div class="error-status">
          ${this.getSeverityBadge()}
          ${this.getStatusBadge()}
        </div>
      </div>
      <div class="error-expandable ${this.expanded ? 'expanded' : ''}">
        <div class="related-skills-section">
          <h4>Related SkillNodes</h4>
          <div class="related-skills-container">
            ${this.getRelatedSkillsContent()}
          </div>
        </div>
      </div>
    `;
  }

  /**
   * Get severity badge HTML
   */
  getSeverityBadge() {
    const severityLabels = ['LOW', 'LOW', 'MEDIUM', 'HIGH', 'CRITICAL'];
    const severityColors = ['blue', 'blue', 'yellow', 'orange', 'red'];
    
    const severity = Math.min(Math.max(this.error.severity || 0, 0), 4);
    const label = severityLabels[severity];
    const color = severityColors[severity];
    
    return `<span class="severity-badge severity-${color}">${label}</span>`;
  }

  /**
   * Get status badge HTML
   */
  getStatusBadge() {
    if (!this.error.resolution_status) return '';
    
    const statusIcons = {
      'resolved': 'ni-check',
      'failed': 'ni-cross',
      'pending': 'ni-clock'
    };
    
    const icon = statusIcons[this.error.resolution_status] || 'ni-clock';
    
    return `
      <span class="status-badge status-${this.error.resolution_status}">
        <em class="icon ${icon}"></em>
        ${this.error.resolution_status}
      </span>
    `;
  }

  /**
   * Get related skills content
   */
  getRelatedSkillsContent() {
    if (this.loadingRelatedSkills) {
      return '<div class="loading-spinner"></div>';
    }

    if (this.relatedSkills.length === 0) {
      return '<p class="text-muted">No related skills found</p>';
    }

    return this.relatedSkills.map(skill => `
      <div class="related-skill-item" data-skill-id="${skill.id}">
        <div class="related-skill-header">
          <span class="skill-type">${this.escapeHtml(skill.skill_type)}</span>
          <span class="skill-success-rate">${(skill.performance?.success_rate * 100 || 0).toFixed(0)}%</span>
        </div>
        <div class="skill-capabilities-mini">
          ${skill.capabilities.slice(0, 2).join(', ')}
          ${skill.capabilities.length > 2 ? ` +${skill.capabilities.length - 2} more` : ''}
        </div>
      </div>
    `).join('');
  }

  /**
   * Attach event listeners
   */
  attachEvents() {
    this.element.addEventListener('click', (e) => {
      // Don't toggle if clicking on related skills
      if (!e.target.closest('.related-skills-section')) {
        this.toggleExpanded();
      }
    });

    this.element.addEventListener('keydown', (e) => {
      if (e.key === 'Enter' || e.key === ' ') {
        e.preventDefault();
        this.toggleExpanded();
      }
    });

    // Make card focusable for accessibility
    this.element.setAttribute('tabindex', '0');
    this.element.setAttribute('role', 'button');
    this.element.setAttribute('aria-label', `View details for ${this.error.error_type} error`);
    this.element.setAttribute('aria-expanded', this.expanded.toString());
  }

  /**
   * Toggle expanded state
   */
  async toggleExpanded() {
    this.expanded = !this.expanded;
    
    if (this.expanded && this.relatedSkills.length === 0 && !this.loadingRelatedSkills) {
      await this.loadRelatedSkills();
    }
    
    this.updateExpandedState();
  }

  /**
   * Load related skills
   */
  async loadRelatedSkills() {
    if (this.loadingRelatedSkills) return;
    
    this.loadingRelatedSkills = true;
    this.updateRelatedSkillsContent();
    
    try {
      if (window.graphChainAPI) {
        this.relatedSkills = await window.graphChainAPI.getSkillsForError(this.error.error_type);
      }
    } catch (error) {
      console.error('Failed to load related skills:', error);
      this.relatedSkills = [];
    } finally {
      this.loadingRelatedSkills = false;
      this.updateRelatedSkillsContent();
      this.attachRelatedSkillEvents();
    }
  }

  /**
   * Update related skills content
   */
  updateRelatedSkillsContent() {
    const container = this.element.querySelector('.related-skills-container');
    if (container) {
      container.innerHTML = this.getRelatedSkillsContent();
    }
  }

  /**
   * Attach events to related skill items
   */
  attachRelatedSkillEvents() {
    const skillItems = this.element.querySelectorAll('.related-skill-item');
    skillItems.forEach(item => {
      item.addEventListener('click', (e) => {
        e.stopPropagation();
        const skillId = item.dataset.skillId;
        this.navigateToSkill(skillId);
      });

      item.addEventListener('keydown', (e) => {
        if (e.key === 'Enter' || e.key === ' ') {
          e.preventDefault();
          e.stopPropagation();
          const skillId = item.dataset.skillId;
          this.navigateToSkill(skillId);
        }
      });

      item.setAttribute('tabindex', '0');
      item.setAttribute('role', 'button');
    });
  }

  /**
   * Navigate to skill details
   */
  navigateToSkill(skillId) {
    if (window.graphChainApp && window.graphChainApp.router) {
      window.graphChainApp.router.navigate(`/skill/${encodeURIComponent(skillId)}`);
    } else {
      window.location.href = `skill-details.html?id=${encodeURIComponent(skillId)}`;
    }
  }

  /**
   * Update expanded state
   */
  updateExpandedState() {
    const expandable = this.element.querySelector('.error-expandable');
    if (expandable) {
      expandable.classList.toggle('expanded', this.expanded);
    }
    
    this.element.setAttribute('aria-expanded', this.expanded.toString());
  }

  /**
   * Update error data
   */
  update(newError) {
    this.error = newError;
    if (this.element) {
      this.element.innerHTML = this.getTemplate();
      this.attachEvents();
    }
  }

  /**
   * Destroy the component
   */
  destroy() {
    if (this.element && this.element.parentNode) {
      this.element.parentNode.removeChild(this.element);
    }
    this.element = null;
  }

  /**
   * Format timestamp
   */
  formatTime(timestamp) {
    const date = new Date(timestamp);
    const now = new Date();
    const diffMs = now - date;
    const diffMins = Math.floor(diffMs / 60000);
    const diffHours = Math.floor(diffMs / 3600000);
    const diffDays = Math.floor(diffMs / 86400000);

    if (diffMins < 1) {
      return 'Just now';
    } else if (diffMins < 60) {
      return `${diffMins}m ago`;
    } else if (diffHours < 24) {
      return `${diffHours}h ago`;
    } else if (diffDays < 7) {
      return `${diffDays}d ago`;
    } else {
      return date.toLocaleDateString();
    }
  }

  /**
   * Escape HTML to prevent XSS
   */
  escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
  }

  /**
   * Get error card element
   */
  getElement() {
    return this.element;
  }

  /**
   * Get error data
   */
  getError() {
    return this.error;
  }

  /**
   * Check if error matches search query
   */
  matchesSearch(query) {
    if (!query) return true;
    
    const searchText = query.toLowerCase();
    const errorType = this.error.error_type.toLowerCase();
    const description = this.error.description.toLowerCase();
    
    return errorType.includes(searchText) || description.includes(searchText);
  }

  /**
   * Check if error matches filters
   */
  matchesFilters(filters) {
    if (!filters) return true;

    // Filter by error type
    if (filters.errorType && !this.error.error_type.toLowerCase().includes(filters.errorType.toLowerCase())) {
      return false;
    }

    // Filter by severity
    if (filters.severity !== undefined && this.error.severity !== filters.severity) {
      return false;
    }

    // Filter by status
    if (filters.status && this.error.resolution_status !== filters.status) {
      return false;
    }

    return true;
  }

  /**
   * Animate card entrance
   */
  animateIn(delay = 0) {
    if (!this.element) return;

    this.element.style.opacity = '0';
    this.element.style.transform = 'translateY(20px)';
    
    setTimeout(() => {
      this.element.style.transition = 'opacity 0.3s ease, transform 0.3s ease';
      this.element.style.opacity = '1';
      this.element.style.transform = 'translateY(0)';
    }, delay);
  }

  /**
   * Static method to create multiple error cards
   */
  static createMultiple(errors, container) {
    const cards = [];
    
    errors.forEach((error, index) => {
      const card = new ErrorCard(error, container);
      const element = card.render();
      container.appendChild(element);
      card.animateIn(index * 50); // Stagger animations
      cards.push(card);
    });
    
    return cards;
  }
}

// Export for module usage
if (typeof module !== 'undefined' && module.exports) {
  module.exports = ErrorCard;
}
