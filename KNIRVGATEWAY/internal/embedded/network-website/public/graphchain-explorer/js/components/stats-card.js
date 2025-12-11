/**
 * StatsCard Component
 * Displays statistical information in a card format
 */

class StatsCard {
  constructor(config) {
    this.title = config.title;
    this.value = config.value;
    this.icon = config.icon;
    this.color = config.color || 'blue';
    this.trend = config.trend;
    this.element = null;
    this.animationDuration = 1000; // 1 second
  }

  /**
   * Render the stats card
   */
  render() {
    this.element = document.createElement('div');
    this.element.className = `stats-card stats-card-${this.color}`;
    this.element.innerHTML = this.getTemplate();
    this.attachEvents();
    return this.element;
  }

  /**
   * Get the HTML template for the stats card
   */
  getTemplate() {
    return `
      <div class="stats-icon">
        <em class="icon ${this.getIconClass()}"></em>
      </div>
      <div class="stats-content">
        <div class="stats-value" data-value="${this.value}">0</div>
        <div class="stats-label">${this.escapeHtml(this.title)}</div>
        ${this.trend !== undefined ? this.getTrendIndicator() : ''}
      </div>
    `;
  }

  /**
   * Get icon class based on icon type
   */
  getIconClass() {
    const iconMap = {
      'network': 'ni-network',
      'cpu': 'ni-cpu',
      'alert-triangle': 'ni-alert-triangle',
      'clock': 'ni-clock',
      'activity': 'ni-activity',
      'trending-up': 'ni-trending-up',
      'trending-down': 'ni-trending-down',
      'users': 'ni-users',
      'server': 'ni-server',
      'database': 'ni-database'
    };

    return iconMap[this.icon] || this.icon || 'ni-activity';
  }

  /**
   * Get trend indicator HTML
   */
  getTrendIndicator() {
    if (this.trend === undefined || this.trend === null) return '';

    const isPositive = this.trend > 0;
    const isNegative = this.trend < 0;
    const trendClass = isPositive ? 'trend-up' : isNegative ? 'trend-down' : 'trend-neutral';
    const trendIcon = isPositive ? 'ni-arrow-up' : isNegative ? 'ni-arrow-down' : 'ni-minus';
    const trendColor = isPositive ? 'text-success' : isNegative ? 'text-danger' : 'text-muted';

    return `
      <div class="stats-trend ${trendClass} ${trendColor}">
        <em class="icon ${trendIcon}"></em>
        <span>${Math.abs(this.trend).toFixed(1)}%</span>
      </div>
    `;
  }

  /**
   * Attach event listeners
   */
  attachEvents() {
    // Add hover effects
    this.element.addEventListener('mouseenter', () => {
      this.element.style.transform = 'translateY(-2px)';
    });

    this.element.addEventListener('mouseleave', () => {
      this.element.style.transform = 'translateY(0)';
    });
  }

  /**
   * Update the stats value with animation
   */
  updateValue(newValue, animate = true) {
    const oldValue = this.value;
    this.value = newValue;

    const valueElement = this.element.querySelector('.stats-value');
    if (!valueElement) return;

    if (animate && oldValue !== newValue) {
      this.animateValue(valueElement, oldValue, newValue);
    } else {
      valueElement.textContent = this.formatValue(newValue);
      valueElement.setAttribute('data-value', newValue);
    }
  }

  /**
   * Animate value change
   */
  animateValue(element, startValue, endValue) {
    const startTime = performance.now();
    const isNumeric = typeof endValue === 'number';
    
    if (!isNumeric) {
      element.textContent = this.formatValue(endValue);
      element.setAttribute('data-value', endValue);
      return;
    }

    const animate = (currentTime) => {
      const elapsed = currentTime - startTime;
      const progress = Math.min(elapsed / this.animationDuration, 1);
      
      // Use easing function for smooth animation
      const easedProgress = this.easeOutCubic(progress);
      const currentValue = startValue + (endValue - startValue) * easedProgress;
      
      element.textContent = this.formatValue(Math.round(currentValue));
      
      if (progress < 1) {
        requestAnimationFrame(animate);
      } else {
        element.textContent = this.formatValue(endValue);
        element.setAttribute('data-value', endValue);
      }
    };

    requestAnimationFrame(animate);
  }

  /**
   * Easing function for smooth animation
   */
  easeOutCubic(t) {
    return 1 - Math.pow(1 - t, 3);
  }

  /**
   * Format value for display
   */
  formatValue(value) {
    if (typeof value === 'number') {
      // Format large numbers with commas
      if (value >= 1000000) {
        return (value / 1000000).toFixed(1) + 'M';
      } else if (value >= 1000) {
        return (value / 1000).toFixed(1) + 'K';
      } else {
        return value.toLocaleString();
      }
    }
    
    return String(value);
  }

  /**
   * Update trend indicator
   */
  updateTrend(newTrend) {
    this.trend = newTrend;
    
    const existingTrend = this.element.querySelector('.stats-trend');
    if (existingTrend) {
      existingTrend.remove();
    }

    if (newTrend !== undefined) {
      const statsContent = this.element.querySelector('.stats-content');
      if (statsContent) {
        statsContent.insertAdjacentHTML('beforeend', this.getTrendIndicator());
      }
    }
  }

  /**
   * Update card color
   */
  updateColor(newColor) {
    if (this.element) {
      this.element.classList.remove(`stats-card-${this.color}`);
      this.color = newColor;
      this.element.classList.add(`stats-card-${this.color}`);
    }
  }

  /**
   * Update card title
   */
  updateTitle(newTitle) {
    this.title = newTitle;
    const labelElement = this.element.querySelector('.stats-label');
    if (labelElement) {
      labelElement.textContent = newTitle;
    }
  }

  /**
   * Update card icon
   */
  updateIcon(newIcon) {
    this.icon = newIcon;
    const iconElement = this.element.querySelector('.stats-icon em');
    if (iconElement) {
      iconElement.className = `icon ${this.getIconClass()}`;
    }
  }

  /**
   * Show loading state
   */
  showLoading() {
    if (this.element) {
      this.element.classList.add('loading');
      const valueElement = this.element.querySelector('.stats-value');
      if (valueElement) {
        valueElement.innerHTML = '<div class="loading-spinner"></div>';
      }
    }
  }

  /**
   * Hide loading state
   */
  hideLoading() {
    if (this.element) {
      this.element.classList.remove('loading');
      const valueElement = this.element.querySelector('.stats-value');
      if (valueElement) {
        valueElement.textContent = this.formatValue(this.value);
      }
    }
  }

  /**
   * Animate card entrance
   */
  animateIn(delay = 0) {
    if (!this.element) return;

    this.element.style.opacity = '0';
    this.element.style.transform = 'translateY(20px) scale(0.95)';
    
    setTimeout(() => {
      this.element.style.transition = 'opacity 0.4s ease, transform 0.4s ease';
      this.element.style.opacity = '1';
      this.element.style.transform = 'translateY(0) scale(1)';
      
      // Animate the value after the card appears
      setTimeout(() => {
        this.animateValue(
          this.element.querySelector('.stats-value'),
          0,
          this.value
        );
      }, 200);
    }, delay);
  }

  /**
   * Pulse animation for updates
   */
  pulseUpdate() {
    if (!this.element) return;

    this.element.style.transform = 'scale(1.05)';
    setTimeout(() => {
      this.element.style.transform = 'scale(1)';
    }, 150);
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
   * Get current value
   */
  getValue() {
    return this.value;
  }

  /**
   * Get element
   */
  getElement() {
    return this.element;
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
   * Static method to create multiple stats cards
   */
  static createMultiple(configs, container) {
    const cards = [];
    
    configs.forEach((config, index) => {
      const card = new StatsCard(config);
      const element = card.render();
      container.appendChild(element);
      card.animateIn(index * 100); // Stagger animations
      cards.push(card);
    });
    
    return cards;
  }

  /**
   * Static method to update multiple cards
   */
  static updateMultiple(cards, newValues) {
    cards.forEach((card, index) => {
      if (newValues[index] !== undefined) {
        card.updateValue(newValues[index]);
        card.pulseUpdate();
      }
    });
  }
}

// Export for module usage
if (typeof module !== 'undefined' && module.exports) {
  module.exports = StatsCard;
}
