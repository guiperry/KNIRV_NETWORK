/**
 * SkillCard Component
 * Displays SkillNode information in a card format
 */

class SkillCard {
  constructor(skill, container) {
    this.skill = skill;
    this.container = container;
    this.element = null;
  }

  /**
   * Render the skill card
   */
  render() {
    this.element = document.createElement('div');
    this.element.className = 'skill-card';
    this.element.innerHTML = this.getTemplate();
    this.attachEvents();
    return this.element;
  }

  /**
   * Get the HTML template for the skill card
   */
  getTemplate() {
    return `
      <div class="skill-card-header">
        <div class="skill-icon">
          <em class="icon ni ni-cpu"></em>
        </div>
        <div class="skill-info">
          <h3 class="skill-type">${this.escapeHtml(this.skill.skill_type)}</h3>
          <div class="skill-meta">
            <span class="skill-timestamp">
              <em class="icon ni ni-clock"></em>
              ${this.formatTime(this.skill.timestamp)}
            </span>
          </div>
        </div>
        <div class="skill-validation">
          ${this.getValidationBadge()}
        </div>
      </div>
      <div class="skill-capabilities">
        ${this.skill.capabilities.map(cap => 
          `<span class="capability-tag">${this.escapeHtml(cap)}</span>`
        ).join('')}
      </div>
      ${this.skill.performance ? this.getPerformanceSection() : ''}
    `;
  }

  /**
   * Get validation badge HTML
   */
  getValidationBadge() {
    if (!this.skill.validation) {
      return '<span class="validation-badge validation-none">Unvalidated</span>';
    }
    
    const score = Math.round(this.skill.validation.validation_score * 100);
    const status = this.skill.validation.is_validated ? 'validated' : 'pending';
    
    return `<span class="validation-badge validation-${status}">${score}%</span>`;
  }

  /**
   * Get performance section HTML
   */
  getPerformanceSection() {
    const perf = this.skill.performance;
    return `
      <div class="skill-performance">
        <div class="perf-metric">
          <span class="perf-label">Success Rate</span>
          <span class="perf-value">${(perf.success_rate * 100).toFixed(1)}%</span>
        </div>
        <div class="perf-metric">
          <span class="perf-label">Avg Time</span>
          <span class="perf-value">${perf.avg_resolution_time.toFixed(1)}s</span>
        </div>
        <div class="perf-metric">
          <span class="perf-label">Total Resolutions</span>
          <span class="perf-value">${perf.total_resolutions.toLocaleString()}</span>
        </div>
      </div>
    `;
  }

  /**
   * Attach event listeners
   */
  attachEvents() {
    this.element.addEventListener('click', () => {
      this.handleClick();
    });

    this.element.addEventListener('keydown', (e) => {
      if (e.key === 'Enter' || e.key === ' ') {
        e.preventDefault();
        this.handleClick();
      }
    });

    // Make card focusable for accessibility
    this.element.setAttribute('tabindex', '0');
    this.element.setAttribute('role', 'button');
    this.element.setAttribute('aria-label', `View details for ${this.skill.skill_type} skill`);
  }

  /**
   * Handle card click
   */
  handleClick() {
    // Navigate to skill details page
    if (window.graphChainApp && window.graphChainApp.router) {
      window.graphChainApp.router.navigate(`/skill/${encodeURIComponent(this.skill.id)}`);
    } else {
      // Fallback navigation
      window.location.href = `skill-details.html?id=${encodeURIComponent(this.skill.id)}`;
    }
  }

  /**
   * Update skill data
   */
  update(newSkill) {
    this.skill = newSkill;
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
   * Get skill card element
   */
  getElement() {
    return this.element;
  }

  /**
   * Get skill data
   */
  getSkill() {
    return this.skill;
  }

  /**
   * Check if skill matches search query
   */
  matchesSearch(query) {
    if (!query) return true;
    
    const searchText = query.toLowerCase();
    const skillType = this.skill.skill_type.toLowerCase();
    const capabilities = this.skill.capabilities.join(' ').toLowerCase();
    
    return skillType.includes(searchText) || capabilities.includes(searchText);
  }

  /**
   * Check if skill matches filters
   */
  matchesFilters(filters) {
    if (!filters) return true;

    // Filter by skill type
    if (filters.skillType && !this.skill.skill_type.toLowerCase().includes(filters.skillType.toLowerCase())) {
      return false;
    }

    // Filter by validation status
    if (filters.validated !== undefined) {
      const isValidated = this.skill.validation?.is_validated || false;
      if (filters.validated !== isValidated) {
        return false;
      }
    }

    // Filter by capability
    if (filters.capability && !this.skill.capabilities.some(cap => 
      cap.toLowerCase().includes(filters.capability.toLowerCase())
    )) {
      return false;
    }

    // Filter by performance threshold
    if (filters.minSuccessRate !== undefined && this.skill.performance) {
      if (this.skill.performance.success_rate < filters.minSuccessRate) {
        return false;
      }
    }

    return true;
  }

  /**
   * Highlight search terms in the card
   */
  highlightSearch(query) {
    if (!query || !this.element) return;

    const searchText = query.toLowerCase();
    const skillTypeElement = this.element.querySelector('.skill-type');
    const capabilityElements = this.element.querySelectorAll('.capability-tag');

    // Highlight skill type
    if (skillTypeElement) {
      const originalText = this.skill.skill_type;
      const highlightedText = this.highlightText(originalText, searchText);
      skillTypeElement.innerHTML = highlightedText;
    }

    // Highlight capabilities
    capabilityElements.forEach((element, index) => {
      const originalText = this.skill.capabilities[index];
      const highlightedText = this.highlightText(originalText, searchText);
      element.innerHTML = highlightedText;
    });
  }

  /**
   * Highlight text helper
   */
  highlightText(text, searchText) {
    if (!searchText) return this.escapeHtml(text);

    const escapedText = this.escapeHtml(text);
    const regex = new RegExp(`(${this.escapeRegex(searchText)})`, 'gi');
    return escapedText.replace(regex, '<mark>$1</mark>');
  }

  /**
   * Escape regex special characters
   */
  escapeRegex(text) {
    return text.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
  }

  /**
   * Show loading state
   */
  showLoading() {
    if (this.element) {
      this.element.classList.add('loading');
      this.element.style.opacity = '0.6';
      this.element.style.pointerEvents = 'none';
    }
  }

  /**
   * Hide loading state
   */
  hideLoading() {
    if (this.element) {
      this.element.classList.remove('loading');
      this.element.style.opacity = '';
      this.element.style.pointerEvents = '';
    }
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
   * Static method to create multiple skill cards
   */
  static createMultiple(skills, container) {
    const cards = [];
    
    skills.forEach((skill, index) => {
      const card = new SkillCard(skill, container);
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
  module.exports = SkillCard;
}
